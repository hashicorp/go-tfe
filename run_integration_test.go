// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package tfe

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	retryablehttp "github.com/hashicorp/go-retryablehttp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunsList(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	defer orgTestCleanup()

	wTest, _ := createWorkspace(t, client, orgTest)
	rTest1, _ := createRun(t, client, wTest)
	rTest2, _ := createRun(t, client, wTest)

	t.Run("without list options", func(t *testing.T) {
		rl, err := client.Runs.List(ctx, wTest.ID, nil)
		require.NoError(t, err)

		found := []string{}
		for _, r := range rl.Items {
			found = append(found, r.ID)
		}

		assert.Contains(t, found, rTest1.ID)
		assert.Contains(t, found, rTest2.ID)
		assert.Equal(t, 1, rl.CurrentPage)
		assert.Equal(t, 2, rl.TotalCount)
	})

	t.Run("without list options and include as nil", func(t *testing.T) {
		rl, err := client.Runs.List(ctx, wTest.ID, &RunListOptions{
			Include: []RunIncludeOpt{},
		})
		require.NoError(t, err)
		require.NotEmpty(t, rl.Items)

		found := []string{}
		for _, r := range rl.Items {
			found = append(found, r.ID)
		}

		assert.Contains(t, found, rTest1.ID)
		assert.Contains(t, found, rTest2.ID)
		assert.Equal(t, 1, rl.CurrentPage)
		assert.Equal(t, 2, rl.TotalCount)
	})

	t.Run("with list options", func(t *testing.T) {
		t.Skip("paging not supported yet in API")

		// Request a page number which is out of range. The result should
		// be successful, but return no results if the paging options are
		// properly passed along.
		rl, err := client.Runs.List(ctx, wTest.ID, &RunListOptions{
			ListOptions: ListOptions{
				PageNumber: 999,
				PageSize:   100,
			},
		})
		require.NoError(t, err)
		assert.Empty(t, rl.Items)
		assert.Equal(t, 999, rl.CurrentPage)
		assert.Equal(t, 2, rl.TotalCount)
	})

	t.Run("with workspace included", func(t *testing.T) {
		rl, err := client.Runs.List(ctx, wTest.ID, &RunListOptions{
			Include: []RunIncludeOpt{RunWorkspace},
		})
		require.NoError(t, err)

		require.NotEmpty(t, rl.Items)
		require.NotNil(t, rl.Items[0].Workspace)
		assert.NotEmpty(t, rl.Items[0].Workspace.Name)
	})

	t.Run("without a valid workspace ID", func(t *testing.T) {
		rl, err := client.Runs.List(ctx, badIdentifier, nil)
		assert.Nil(t, rl)
		assert.EqualError(t, err, ErrInvalidWorkspaceID.Error())
	})
}

func TestRunsListQueryParams(t *testing.T) {
	type testCase struct {
		options     *RunListOptions
		description string
		assertion   func(tc testCase, rl *RunList, err error)
	}

	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	defer orgTestCleanup()

	workspaceTest, _ := createWorkspace(t, client, orgTest)
	createPlannedRun(t, client, workspaceTest)
	createRun(t, client, workspaceTest)

	testCases := []testCase{
		{
			description: "with status query parameter",
			options:     &RunListOptions{Status: string(RunPending), Include: []RunIncludeOpt{RunWorkspace}},
			assertion: func(tc testCase, rl *RunList, err error) {
				require.NoError(t, err)
				assert.Equal(t, 1, len(rl.Items))
			},
		},
		{
			description: "with source query parameter",
			options:     &RunListOptions{Source: string(RunSourceAPI), Include: []RunIncludeOpt{RunWorkspace}},
			assertion: func(tc testCase, rl *RunList, err error) {
				require.NoError(t, err)
				assert.Equal(t, 2, len(rl.Items))
				assert.Equal(t, rl.Items[0].Source, RunSourceAPI)
			},
		},
		{
			description: "with operation of plan_only parameter",
			options:     &RunListOptions{Operation: string(RunOperationPlanOnly), Include: []RunIncludeOpt{RunWorkspace}},
			assertion: func(tc testCase, rl *RunList, err error) {
				require.NoError(t, err)
				assert.Equal(t, 0, len(rl.Items))
			},
		},
		{
			description: "with mismatch user & commit parameter",
			options:     &RunListOptions{User: randomString(t), Commit: randomString(t), Include: []RunIncludeOpt{RunWorkspace}},
			assertion: func(tc testCase, rl *RunList, err error) {
				require.NoError(t, err)
				assert.Equal(t, 0, len(rl.Items))
			},
		},
		{
			description: "with operation of save_plan parameter",
			options:     &RunListOptions{Operation: string(RunOperationSavePlan), Include: []RunIncludeOpt{RunWorkspace}},
			assertion: func(tc testCase, rl *RunList, err error) {
				require.NoError(t, err)
				assert.Equal(t, 0, len(rl.Items))
			},
		},
	}

	betaTestCases := []testCase{}

	for _, testCase := range testCases {
		t.Run(testCase.description, func(t *testing.T) {
			runs, err := client.Runs.List(ctx, workspaceTest.ID, testCase.options)
			testCase.assertion(testCase, runs, err)
		})
	}

	for _, testCase := range betaTestCases {
		t.Run(testCase.description, func(t *testing.T) {
			skipUnlessBeta(t)
			runs, err := client.Runs.List(ctx, workspaceTest.ID, testCase.options)
			testCase.assertion(testCase, runs, err)
		})
	}
}

func TestRunsCreate(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	wTest, wTestCleanup := createWorkspace(t, client, nil)
	defer wTestCleanup()

	cvTest, _ := createUploadedConfigurationVersion(t, client, wTest)

	t.Run("without a configuration version", func(t *testing.T) {
		options := RunCreateOptions{
			Workspace: wTest,
		}

		r, err := client.Runs.Create(ctx, options)
		require.NoError(t, err)
		assert.NotNil(t, r.ID)
		assert.NotNil(t, r.CreatedAt)
		assert.NotNil(t, r.Source)
		require.NotNil(t, r.StatusTimestamps)
		assert.NotZero(t, r.StatusTimestamps.PlanQueueableAt)
	})

	t.Run("with a configuration version", func(t *testing.T) {
		options := RunCreateOptions{
			ConfigurationVersion: cvTest,
			Workspace:            wTest,
		}

		r, err := client.Runs.Create(ctx, options)
		require.NoError(t, err)
		require.NotNil(t, r.ConfigurationVersion)
		assert.Equal(t, cvTest.ID, r.ConfigurationVersion.ID)
	})

	t.Run("with allow empty apply", func(t *testing.T) {
		options := RunCreateOptions{
			Workspace:       wTest,
			AllowEmptyApply: Bool(true),
		}

		r, err := client.Runs.Create(ctx, options)
		require.NoError(t, err)
		assert.Equal(t, true, r.AllowEmptyApply)
	})

	t.Run("with save-plan", func(t *testing.T) {
		options := RunCreateOptions{
			Workspace: wTest,
			SavePlan:  Bool(true),
		}

		r, err := client.Runs.Create(ctx, options)
		require.NoError(t, err)
		assert.Equal(t, true, r.SavePlan)
	})

	t.Run("with terraform version and plan only", func(t *testing.T) {
		options := RunCreateOptions{
			Workspace:        wTest,
			TerraformVersion: String("1.0.0"),
		}
		_, err := client.Runs.Create(ctx, options)
		require.ErrorIs(t, err, ErrTerraformVersionValidForPlanOnly)

		options.PlanOnly = Bool(true)

		r, err := client.Runs.Create(ctx, options)
		require.NoError(t, err)
		assert.Equal(t, true, r.PlanOnly)
		assert.Equal(t, "1.0.0", r.TerraformVersion)
	})

	t.Run("refresh defaults to true if not set as a create option", func(t *testing.T) {
		options := RunCreateOptions{
			Workspace: wTest,
		}

		r, err := client.Runs.Create(ctx, options)
		require.NoError(t, err)
		assert.Equal(t, true, r.Refresh)
	})

	t.Run("with refresh-only requested", func(t *testing.T) {
		// TODO: remove this skip after the release of Terraform 0.15.4
		t.Skip("Skipping this test until -refresh-only is released in the Terraform CLI")

		options := RunCreateOptions{
			Workspace:   wTest,
			RefreshOnly: Bool(true),
		}

		r, err := client.Runs.Create(ctx, options)
		require.NoError(t, err)
		assert.Equal(t, true, r.RefreshOnly)
	})

	t.Run("with auto-apply requested", func(t *testing.T) {
		// ensure the worksapce auto-apply is false so it does not default to that.
		assert.Equal(t, false, wTest.AutoApply)

		options := RunCreateOptions{
			Workspace: wTest,
			AutoApply: Bool(true),
		}

		r, err := client.Runs.Create(ctx, options)
		require.NoError(t, err)
		assert.Equal(t, true, r.AutoApply)
	})

	t.Run("without auto-apply, defaulting to workspace autoapply", func(t *testing.T) {
		options := RunCreateOptions{
			Workspace: wTest,
		}

		r, err := client.Runs.Create(ctx, options)
		require.NoError(t, err)
		assert.Equal(t, wTest.AutoApply, r.AutoApply)
	})

	t.Run("without a workspace", func(t *testing.T) {
		r, err := client.Runs.Create(ctx, RunCreateOptions{})
		assert.Nil(t, r)
		assert.Equal(t, err, ErrRequiredWorkspace)
	})

	t.Run("with additional attributes", func(t *testing.T) {
		options := RunCreateOptions{
			Message:           String("yo"),
			Workspace:         wTest,
			Refresh:           Bool(false),
			ReplaceAddrs:      []string{"null_resource.example"},
			TargetAddrs:       []string{"null_resource.example"},
			InvokeActionAddrs: []string{"actions.foo.bar"},
		}

		r, err := client.Runs.Create(ctx, options)
		require.NoError(t, err)
		assert.Equal(t, *options.Message, r.Message)
		assert.Equal(t, *options.Refresh, r.Refresh)
		assert.Equal(t, options.ReplaceAddrs, r.ReplaceAddrs)
		assert.Equal(t, options.TargetAddrs, r.TargetAddrs)
		assert.Equal(t, options.InvokeActionAddrs, r.InvokeActionAddrs)
		assert.Nil(t, r.Variables)
	})

	t.Run("with variables", func(t *testing.T) {
		vars := []*RunVariable{
			{
				Key:   "test_variable",
				Value: "Hello, World!",
			},
			{
				Key:   "test_foo",
				Value: "Hello, Foo!",
			},
		}

		options := RunCreateOptions{
			Message:   String("yo"),
			Workspace: wTest,
			Variables: vars,
		}

		r, err := client.Runs.Create(ctx, options)
		require.NoError(t, err)
		assert.NotNil(t, r.Variables)
		assert.Equal(t, len(vars), len(r.Variables))

		for _, v := range r.Variables {
			if v.Key == "test_foo" {
				assert.Equal(t, v.Value, "Hello, Foo!")
			} else if v.Key == "test_variable" {
				assert.Equal(t, v.Value, "Hello, World!")
			} else {
				t.Fatalf("Unexpected variable key: %s", v.Key)
			}
		}
	})

	t.Run("with policy paths", func(t *testing.T) {
		skipUnlessBeta(t)

		opts := RunCreateOptions{
			Message:     String("creating with policy paths"),
			Workspace:   wTest,
			PolicyPaths: []string{"./path/to/dir1", "./path/to/dir2"},
		}

		r, err := client.Runs.Create(ctx, opts)
		require.NoError(t, err)
		require.NotEmpty(t, r.PolicyPaths)

		assert.Len(t, r.PolicyPaths, 2)
		assert.Contains(t, r.PolicyPaths, "./path/to/dir1")
		assert.Contains(t, r.PolicyPaths, "./path/to/dir2")
	})

	t.Run("with action invocations", func(t *testing.T) {
		skipUnlessBeta(t)

		opts := RunCreateOptions{
			Message:           String("creating with policy paths"),
			Workspace:         wTest,
			InvokeActionAddrs: []string{"actions.foo.bar"},
		}

		r, err := client.Runs.Create(ctx, opts)
		require.NoError(t, err)
		require.NotEmpty(t, r.InvokeActionAddrs)

		assert.Len(t, r.InvokeActionAddrs, 1)
		assert.Contains(t, r.InvokeActionAddrs, "actions.foo.bar")
	})
}

func TestRunsRead_CostEstimate(t *testing.T) {
	skipIfEnterprise(t)

	client := testClient(t)
	ctx := context.Background()

	rTest, rTestCleanup := createCostEstimatedRun(t, client, nil)
	defer rTestCleanup()

	t.Run("when the run exists", func(t *testing.T) {
		r, err := client.Runs.Read(ctx, rTest.ID)
		require.NoError(t, err)
		assert.Equal(t, rTest, r)
	})

	t.Run("when the run does not exist", func(t *testing.T) {
		r, err := client.Runs.Read(ctx, "nonexisting")
		assert.Nil(t, r)
		assert.Equal(t, err, ErrResourceNotFound)
	})

	t.Run("with invalid run ID", func(t *testing.T) {
		r, err := client.Runs.Read(ctx, badIdentifier)
		assert.Nil(t, r)
		assert.EqualError(t, err, ErrInvalidRunID.Error())
	})
}

func TestRunsReadWithOptions(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	rTest, rTestCleanup := createRun(t, client, nil)
	defer rTestCleanup()

	t.Run("when the run exists", func(t *testing.T) {
		curOpts := &RunReadOptions{
			Include: []RunIncludeOpt{RunCreatedBy},
		}

		r, err := client.Runs.ReadWithOptions(ctx, rTest.ID, curOpts)
		require.NoError(t, err)

		require.NotEmpty(t, r.CreatedBy)
		assert.NotEmpty(t, r.CreatedBy.Username)
	})
}

func TestRunsReadWithPolicyPaths(t *testing.T) {
	skipUnlessBeta(t)

	client := testClient(t)
	ctx := context.Background()

	wTest, wTestCleanup := createWorkspace(t, client, nil)
	t.Cleanup(wTestCleanup)

	_, cvCleanup := createUploadedConfigurationVersion(t, client, wTest)
	t.Cleanup(cvCleanup)

	r, err := client.Runs.Create(ctx, RunCreateOptions{
		Workspace:   wTest,
		PolicyPaths: []string{"./foo"},
	})
	require.NoError(t, err)

	r, err = client.Runs.Read(ctx, r.ID)
	require.NoError(t, err)

	require.NotEmpty(t, r.PolicyPaths)
	assert.Contains(t, r.PolicyPaths, "./foo")
}

func TestRunsConfirmedBy(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	t.Run("with apply", func(t *testing.T) {
		rTest, rTestCleanup := createRunApply(t, client, nil)
		t.Cleanup(rTestCleanup)

		r, err := client.Runs.Read(ctx, rTest.ID)
		require.NoError(t, err)

		assert.NotNil(t, r.ConfirmedBy)
		assert.NotZero(t, r.ConfirmedBy.ID)
	})

	t.Run("without apply", func(t *testing.T) {
		rTest, rTestCleanup := createPlannedRun(t, client, nil)
		t.Cleanup(rTestCleanup)

		r, err := client.Runs.Read(ctx, rTest.ID)
		require.NoError(t, err)
		assert.Equal(t, rTest, r)

		assert.Nil(t, r.ConfirmedBy)
	})
}

func TestRunsCanceledAt(t *testing.T) {
	client := testClient(t)

	ctx := context.Background()

	wTest, wTestCleanup := createWorkspace(t, client, nil)
	t.Cleanup(wTestCleanup)

	// We need to create 2 runs here. The first run will automatically
	// be planned so that one cannot be cancelled. The second one will
	// be pending until the first one is confirmed or discarded, so we
	// can cancel that one.
	createRun(t, client, wTest)
	rTest, _ := createRun(t, client, wTest)

	t.Run("when the run is not canceled", func(t *testing.T) {
		r, err := client.Runs.Read(ctx, rTest.ID)
		require.NoError(t, err)

		assert.Empty(t, r.CanceledAt)
	})

	t.Run("when the run is canceled", func(t *testing.T) {
		err := client.Runs.Cancel(ctx, rTest.ID, RunCancelOptions{})
		require.NoError(t, err)

		for i := 1; ; i++ {
			// Refresh the view of the run
			rTest, err = client.Runs.Read(ctx, rTest.ID)
			require.NoError(t, err)

			// Check if the timestamp is present.
			if !rTest.ForceCancelAvailableAt.IsZero() {
				break
			}

			if i > 30 {
				t.Fatal("Timeout waiting for run to be canceled")
			}

			time.Sleep(time.Second)
		}

		r, err := client.Runs.Read(ctx, rTest.ID)
		require.NoError(t, err)

		assert.NotEmpty(t, r.CanceledAt)
	})
}

func TestRunsRunEvents(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	wTest, wTestCleanup := createWorkspace(t, client, nil)
	defer wTestCleanup()

	_, cvCleanup := createUploadedConfigurationVersion(t, client, wTest)
	t.Cleanup(cvCleanup)

	options := RunCreateOptions{
		Workspace: wTest,
	}

	r, err := client.Runs.Create(ctx, options)
	require.NoError(t, err)

	assert.NotEmpty(t, r.RunEvents)
}

func TestRunsTriggerReason(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	wTest, wTestCleanup := createWorkspace(t, client, nil)
	defer wTestCleanup()

	_, cvCleanup := createUploadedConfigurationVersion(t, client, wTest)
	t.Cleanup(cvCleanup)

	options := RunCreateOptions{
		Workspace: wTest,
	}

	r, err := client.Runs.Create(ctx, options)
	require.NoError(t, err)

	assert.NotNil(t, r.TriggerReason)
}

func TestRunsApply(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	defer orgTestCleanup()
	wTest, _ := createWorkspace(t, client, orgTest)

	rTest, _ := createPlannedRun(t, client, wTest)

	t.Run("when the run exists", func(t *testing.T) {
		err := client.Runs.Apply(ctx, rTest.ID, RunApplyOptions{
			Comment: String("Hello, Earl"),
		})
		require.NoError(t, err)

		r, err := client.Runs.Read(ctx, rTest.ID)
		require.NoError(t, err)

		assert.Len(t, r.Comments, 1)

		c, err := client.Comments.Read(ctx, r.Comments[0].ID)
		require.NoError(t, err)
		assert.Equal(t, "Hello, Earl", c.Body)
	})

	t.Run("when the run does not exist", func(t *testing.T) {
		err := client.Runs.Apply(ctx, "nonexisting", RunApplyOptions{})
		assert.Equal(t, err, ErrResourceNotFound)
	})

	t.Run("with invalid run ID", func(t *testing.T) {
		err := client.Runs.Apply(ctx, badIdentifier, RunApplyOptions{})
		assert.EqualError(t, err, ErrInvalidRunID.Error())
	})
}

func TestRunsCancel(t *testing.T) {
	client := testClient(t)

	ctx := context.Background()

	wTest, wTestCleanup := createWorkspace(t, client, nil)
	t.Cleanup(wTestCleanup)

	// We need to create 2 runs here. The first run will automatically
	// be planned so that one cannot be cancelled. The second one will
	// be pending until the first one is confirmed or discarded, so we
	// can cancel that one.
	createRun(t, client, wTest)
	rTest, _ := createRun(t, client, wTest)

	t.Run("when the run exists", func(t *testing.T) {
		err := client.Runs.Cancel(ctx, rTest.ID, RunCancelOptions{})
		require.NoError(t, err)
	})

	t.Run("when the run does not exist", func(t *testing.T) {
		err := client.Runs.Cancel(ctx, "nonexisting", RunCancelOptions{})
		assert.Equal(t, err, ErrResourceNotFound)
	})

	t.Run("with invalid run ID", func(t *testing.T) {
		err := client.Runs.Cancel(ctx, badIdentifier, RunCancelOptions{})
		assert.EqualError(t, err, ErrInvalidRunID.Error())
	})
}

func TestRunsForceCancel(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	wTest, wTestCleanup := createWorkspace(t, client, nil)
	defer wTestCleanup()

	// We need to create 2 runs here. The first run will automatically
	// be planned so that one cannot be cancelled. The second one will
	// be pending until the first one is confirmed or discarded, so we
	// can cancel that one.
	createRun(t, client, wTest)
	rTest, _ := createRun(t, client, wTest)

	t.Run("run is not force-cancelable", func(t *testing.T) {
		assert.False(t, rTest.Actions.IsForceCancelable)
	})

	t.Run("user is allowed to force-cancel", func(t *testing.T) {
		assert.True(t, rTest.Permissions.CanForceCancel)
	})

	t.Run("after a normal cancel", func(t *testing.T) {
		// Request the normal cancel
		err := client.Runs.Cancel(ctx, rTest.ID, RunCancelOptions{})
		require.NoError(t, err)

		for i := 1; ; i++ {
			// Refresh the view of the run
			rTest, err = client.Runs.Read(ctx, rTest.ID)
			require.NoError(t, err)

			// Check if the timestamp is present.
			if !rTest.ForceCancelAvailableAt.IsZero() {
				break
			}

			if i > 30 {
				t.Fatal("Timeout waiting for run to be canceled")
			}

			time.Sleep(time.Second)
		}

		t.Run("force-cancel-available-at timestamp is present", func(t *testing.T) {
			assert.True(t, rTest.ForceCancelAvailableAt.After(time.Now()))
		})

		// This test case is minimal because a force-cancel is not needed in
		// any normal circumstance. Only if Terraform encounters unexpected
		// errors or behaves abnormally should this functionality be required.
		// Force-cancel only becomes available if a normal cancel is performed
		// first, and the desired canceled state is not reached within a pre-
		// determined amount of time (see
		// https://developer.hashicorp.com/terraform/cloud-docs/api-docs/run#forcefully-cancel-a-run).
	})

	t.Run("when the run does not exist", func(t *testing.T) {
		err := client.Runs.ForceCancel(ctx, "nonexisting", RunForceCancelOptions{})
		assert.Equal(t, err, ErrResourceNotFound)
	})

	t.Run("with invalid run ID", func(t *testing.T) {
		err := client.Runs.ForceCancel(ctx, badIdentifier, RunForceCancelOptions{})
		assert.EqualError(t, err, ErrInvalidRunID.Error())
	})
}

func TestRunsForceExecute(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	wTest, wTestCleanup := createWorkspace(t, client, nil)
	defer wTestCleanup()

	// We need to create 2 runs here:
	// - The first run will automatically be planned so that the second
	//   run can't be executed.
	// - The second run will be pending until the first run is confirmed or
	//   discarded, so we will force execute this run.
	rToCancel, _ := createPlannedRun(t, client, wTest)
	rTest, _ := createRunWaitForStatus(t, client, wTest, RunPending)

	t.Run("a successful force-execute", func(t *testing.T) {
		// Verify the user has permission to force-execute the run
		assert.True(t, rTest.Permissions.CanForceExecute)

		err := client.Runs.ForceExecute(ctx, rTest.ID)
		require.NoError(t, err)

		timeout := 2 * time.Minute
		ctxPollRunForceExecute, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()

		// Verify the second run has a status that is an applyable status
		rTest = pollRunStatus(t,
			client,
			ctxPollRunForceExecute,
			rTest,
			applyableStatuses(rTest))
		if rTest.Status == RunErrored {
			fatalDumpRunLog(t, client, ctx, rTest)
		}

		// Refresh the view of the first run
		rToCancel, err = client.Runs.Read(ctx, rToCancel.ID)
		require.NoError(t, err)

		// Verify the first run was discarded
		assert.Equal(t, RunDiscarded, rToCancel.Status)
	})

	t.Run("when the run does not exist", func(t *testing.T) {
		err := client.Runs.ForceExecute(ctx, "nonexisting")
		assert.Equal(t, err, ErrResourceNotFound)
	})

	t.Run("with invalid run ID", func(t *testing.T) {
		err := client.Runs.ForceExecute(ctx, badIdentifier)
		assert.EqualError(t, err, ErrInvalidRunID.Error())
	})
}

func TestRunsDiscard(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	wTest, wTestCleanup := createWorkspace(t, client, nil)
	defer wTestCleanup()

	rTest, _ := createPlannedRun(t, client, wTest)

	t.Run("when the run exists", func(t *testing.T) {
		err := client.Runs.Discard(ctx, rTest.ID, RunDiscardOptions{})
		require.NoError(t, err)
	})

	t.Run("when the run does not exist", func(t *testing.T) {
		err := client.Runs.Discard(ctx, "nonexisting", RunDiscardOptions{})
		assert.Equal(t, err, ErrResourceNotFound)
	})

	t.Run("with invalid run ID", func(t *testing.T) {
		err := client.Runs.Discard(ctx, badIdentifier, RunDiscardOptions{})
		assert.EqualError(t, err, ErrInvalidRunID.Error())
	})
}

func TestRun_Unmarshal(t *testing.T) {
	data := map[string]interface{}{
		"data": map[string]interface{}{
			"type": "runs",
			"id":   "1",
			"attributes": map[string]interface{}{
				"created-at":  "2018-03-02T23:42:06.651Z",
				"has-changes": true,
				"is-destroy":  false,
				"message":     "run message",
				"actions": map[string]interface{}{
					"is-cancelable":       true,
					"is-confirmable":      true,
					"is-discardable":      true,
					"is-force-cancelable": true,
				},
				"permissions": map[string]interface{}{
					"can-apply":         true,
					"can-cancel":        true,
					"can-discard":       true,
					"can-force-cancel":  true,
					"can-force-execute": true,
				},
				"status-timestamps": map[string]string{
					"plan-queued-at": "2020-03-16T23:15:59+00:00",
					"errored-at":     "2019-03-16T23:23:59+00:00",
				},
				"variables": []map[string]string{{"key": "a-key", "value": "\"a-value\""}},
			},
		},
	}
	byteData, err := json.Marshal(data)
	require.NoError(t, err)

	responseBody := bytes.NewReader(byteData)
	run := &Run{}
	err = unmarshalResponse(responseBody, run)
	require.NoError(t, err)

	planQueuedParsedTime, err := time.Parse(time.RFC3339, "2020-03-16T23:15:59+00:00")
	require.NoError(t, err)
	erroredParsedTime, err := time.Parse(time.RFC3339, "2019-03-16T23:23:59+00:00")
	require.NoError(t, err)

	iso8601TimeFormat := "2006-01-02T15:04:05Z"
	parsedTime, err := time.Parse(iso8601TimeFormat, "2018-03-02T23:42:06.651Z")
	require.NoError(t, err)
	assert.Equal(t, run.ID, "1")
	assert.Equal(t, run.CreatedAt, parsedTime)
	assert.Equal(t, run.HasChanges, true)
	assert.Equal(t, run.IsDestroy, false)
	assert.Equal(t, run.Message, "run message")
	assert.Equal(t, run.Actions.IsConfirmable, true)
	assert.Equal(t, run.Actions.IsCancelable, true)
	assert.Equal(t, run.Actions.IsDiscardable, true)
	assert.Equal(t, run.Actions.IsForceCancelable, true)
	assert.Equal(t, run.Permissions.CanApply, true)
	assert.Equal(t, run.Permissions.CanCancel, true)
	assert.Equal(t, run.Permissions.CanDiscard, true)
	assert.Equal(t, run.Permissions.CanForceExecute, true)
	assert.Equal(t, run.Permissions.CanForceCancel, true)
	assert.Equal(t, run.StatusTimestamps.PlanQueuedAt, planQueuedParsedTime)
	assert.Equal(t, run.StatusTimestamps.ErroredAt, erroredParsedTime)

	require.NotEmpty(t, run.Variables)
	assert.Equal(t, run.Variables[0].Key, "a-key")
	assert.Equal(t, run.Variables[0].Value, "\"a-value\"")
}

func TestRunCreateOptions_Marshal(t *testing.T) {
	client := testClient(t)

	wTest, wTestCleanup := createWorkspace(t, client, nil)
	defer wTestCleanup()

	opts := RunCreateOptions{
		Workspace: wTest,
		Variables: []*RunVariable{
			{
				Key:   "test_variable",
				Value: "Hello, World!",
			},
			{
				Key:   "test_foo",
				Value: "Hello, Foo!",
			},
		},
	}

	reqBody, err := serializeRequestBody(&opts)
	require.NoError(t, err)
	req, err := retryablehttp.NewRequest("POST", "url", reqBody)
	require.NoError(t, err)
	bodyBytes, err := req.BodyBytes()
	require.NoError(t, err)

	expectedBody := fmt.Sprintf(`{"data":{"type":"runs","attributes":{"variables":[{"key":"test_variable","value":"Hello, World!"},{"key":"test_foo","value":"Hello, Foo!"}]},"relationships":{"configuration-version":{"data":null},"workspace":{"data":{"type":"workspaces","id":"%s"}}}}}
`, wTest.ID)

	assert.Equal(t, string(bodyBytes), expectedBody)
}

func TestRunsListForOrganization(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	defer orgTestCleanup()

	apTest, _ := createAgentPool(t, client, orgTest)

	wTest, _ := createWorkspaceWithOptions(t, client, orgTest, WorkspaceCreateOptions{
		Name:          String(randomString(t)),
		ExecutionMode: String("agent"),
		AgentPoolID:   &apTest.ID,
	})
	rTest1, _ := createRun(t, client, wTest)
	rTest2, _ := createRun(t, client, wTest)

	t.Run("without list options", func(t *testing.T) {
		rl, err := client.Runs.ListForOrganization(ctx, orgTest.Name, nil)
		require.NoError(t, err)

		found := []string{}
		for _, r := range rl.Items {
			found = append(found, r.ID)
		}

		assert.Contains(t, found, rTest1.ID)
		assert.Contains(t, found, rTest2.ID)
		assert.Equal(t, 1, rl.CurrentPage)
		assert.Empty(t, rl.NextPage)
	})

	t.Run("without list options and include as nil", func(t *testing.T) {
		rl, err := client.Runs.ListForOrganization(ctx, orgTest.Name, &RunListForOrganizationOptions{
			Include: []RunIncludeOpt{},
		})
		require.NoError(t, err)
		require.NotEmpty(t, rl.Items)

		found := []string{}
		for _, r := range rl.Items {
			found = append(found, r.ID)
		}

		assert.Contains(t, found, rTest1.ID)
		assert.Contains(t, found, rTest2.ID)
		assert.Equal(t, 1, rl.CurrentPage)
		assert.Empty(t, rl.NextPage)
	})

	t.Run("with list options", func(t *testing.T) {
		// Request a page number that is out of range. The result should
		// be successful, but return no results if the paging options are
		// properly passed along.
		rl, err := client.Runs.ListForOrganization(ctx, orgTest.Name, &RunListForOrganizationOptions{
			ListOptions: ListOptions{
				PageNumber: 999,
				PageSize:   100,
			},
		})
		require.NoError(t, err)
		assert.Empty(t, rl.Items)
		assert.Equal(t, 999, rl.CurrentPage)
		assert.Empty(t, rl.NextPage)
	})

	t.Run("with workspace included", func(t *testing.T) {
		rl, err := client.Runs.ListForOrganization(ctx, orgTest.Name, &RunListForOrganizationOptions{
			Include: []RunIncludeOpt{RunWorkspace},
		})
		require.NoError(t, err)

		require.NotEmpty(t, rl.Items)
		require.NotNil(t, rl.Items[0].Workspace)
		assert.NotEmpty(t, rl.Items[0].Workspace.Name)
	})

	t.Run("without a valid organization name", func(t *testing.T) {
		rl, err := client.Runs.ListForOrganization(ctx, badIdentifier, nil)
		assert.Nil(t, rl)
		assert.EqualError(t, err, ErrInvalidOrg.Error())
	})

	t.Run("with filter by agent pool", func(t *testing.T) {
		rl, err := client.Runs.ListForOrganization(ctx, orgTest.Name, &RunListForOrganizationOptions{
			AgentPoolNames: apTest.Name,
		})
		require.NoError(t, err)

		found := make([]string, len(rl.Items))
		for i, r := range rl.Items {
			found[i] = r.ID
		}

		assert.Contains(t, found, rTest1.ID)
		assert.Contains(t, found, rTest2.ID)
		assert.Equal(t, 1, rl.CurrentPage)
		assert.Empty(t, rl.NextPage)
	})

	t.Run("with filter by workspace", func(t *testing.T) {
		rl, err := client.Runs.ListForOrganization(ctx, orgTest.Name, &RunListForOrganizationOptions{
			WorkspaceNames: wTest.Name,
			Include:        []RunIncludeOpt{RunWorkspace},
		})
		require.NoError(t, err)

		found := make([]string, len(rl.Items))
		for i, r := range rl.Items {
			found[i] = r.ID
		}

		assert.Contains(t, found, rTest1.ID)
		assert.Contains(t, found, rTest2.ID)
		require.NotNil(t, rl.Items[0].Workspace)
		assert.NotEmpty(t, rl.Items[0].Workspace.Name)
		require.NotNil(t, rl.Items[1].Workspace)
		assert.NotEmpty(t, rl.Items[1].Workspace.Name)
		assert.Equal(t, 1, rl.CurrentPage)
		assert.Empty(t, rl.NextPage)
	})
}
