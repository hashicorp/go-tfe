package tfe

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunsList(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	defer orgTestCleanup()
	wTest, wTestCleanup := createWorkspace(t, client, orgTest)
	defer wTestCleanup()

	rTest1, rTestCleanup1 := createRun(t, client, wTest)
	defer rTestCleanup1()
	rTest2, rTestCleanup2 := createRun(t, client, wTest)
	defer rTestCleanup2()

	t.Run("without list options", func(t *testing.T) {
		rl, err := client.Runs.List(ctx, wTest.ID, RunListOptions{})
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

	t.Run("with list options", func(t *testing.T) {
		t.Skip("paging not supported yet in API")

		// Request a page number which is out of range. The result should
		// be successful, but return no results if the paging options are
		// properly passed along.
		rl, err := client.Runs.List(ctx, wTest.ID, RunListOptions{
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
		rl, err := client.Runs.List(ctx, wTest.ID, RunListOptions{
			Include: String("workspace"),
		})

		assert.NoError(t, err)

		assert.NotEmpty(t, rl.Items)
		assert.NotNil(t, rl.Items[0].Workspace)
		assert.NotEmpty(t, rl.Items[0].Workspace.Name)
	})

	t.Run("without a valid workspace ID", func(t *testing.T) {
		rl, err := client.Runs.List(ctx, badIdentifier, RunListOptions{})
		assert.Nil(t, rl)
		assert.EqualError(t, err, ErrInvalidWorkspaceID.Error())
	})
}

func TestRunsCreate(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	wTest, wTestCleanup := createWorkspace(t, client, nil)
	defer wTestCleanup()

	cvTest, cvTestCleanup := createUploadedConfigurationVersion(t, client, wTest)
	defer cvTestCleanup()

	t.Run("without a configuration version", func(t *testing.T) {
		options := RunCreateOptions{
			Workspace: wTest,
		}

		r, err := client.Runs.Create(ctx, options)
		assert.NoError(t, err)
		assert.NotNil(t, r.ID)
		assert.NotNil(t, r.CreatedAt)
		assert.NotNil(t, r.Source)
		assert.NotEmpty(t, r.StatusTimestamps)
		assert.NotNil(t, r.StatusTimestamps.StartedAt)
	})

	t.Run("with a configuration version", func(t *testing.T) {
		options := RunCreateOptions{
			ConfigurationVersion: cvTest,
			Workspace:            wTest,
		}

		r, err := client.Runs.Create(ctx, options)
		require.NoError(t, err)
		assert.Equal(t, cvTest.ID, r.ConfigurationVersion.ID)
	})

	t.Run("without a workspace", func(t *testing.T) {
		r, err := client.Runs.Create(ctx, RunCreateOptions{})
		assert.Nil(t, r)
		assert.EqualError(t, err, "workspace is required")
	})

	t.Run("with additional attributes", func(t *testing.T) {
		options := RunCreateOptions{
			Message:     String("yo"),
			Workspace:   wTest,
			TargetAddrs: []string{"null_resource.example"},
		}

		r, err := client.Runs.Create(ctx, options)
		require.NoError(t, err)
		assert.Equal(t, *options.Message, r.Message)
		assert.Equal(t, options.TargetAddrs, r.TargetAddrs)
	})
}

func TestRunsRead(t *testing.T) {
	skipIfEnterprise(t)

	client := testClient(t)
	ctx := context.Background()

	rTest, rTestCleanup := createCostEstimatedRun(t, client, nil)
	defer rTestCleanup()

	t.Run("when the run exists", func(t *testing.T) {
		r, err := client.Runs.Read(ctx, rTest.ID)
		assert.NoError(t, err)
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
			Include: "created_by",
		}

		r, err := client.Runs.ReadWithOptions(ctx, rTest.ID, curOpts)
		require.NoError(t, err)

		assert.NotEmpty(t, r.CreatedBy)
		assert.NotEmpty(t, r.CreatedBy.Username)
	})
}

func TestRunsApply(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	defer orgTestCleanup()
	wTest, wTestCleanup := createWorkspace(t, client, orgTest)
	defer wTestCleanup()

	rTest, rTestCleanup := createPlannedRun(t, client, wTest)
	defer rTestCleanup()

	t.Run("when the run exists", func(t *testing.T) {
		err := client.Runs.Apply(ctx, rTest.ID, RunApplyOptions{})
		assert.NoError(t, err)
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
	defer wTestCleanup()

	// We need to create 2 runs here. The first run will automatically
	// be planned so that one cannot be cancelled. The second one will
	// be pending until the first one is confirmed or discarded, so we
	// can cancel that one.
	_, rTestCleanup1 := createRun(t, client, wTest)
	defer rTestCleanup1()
	rTest2, rTestCleanup2 := createRun(t, client, wTest)
	defer rTestCleanup2()

	t.Run("when the run exists", func(t *testing.T) {
		err := client.Runs.Cancel(ctx, rTest2.ID, RunCancelOptions{})
		assert.NoError(t, err)
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
	_, rTestCleanup1 := createRun(t, client, wTest)
	defer rTestCleanup1()
	rTest, rTestCleanup2 := createRun(t, client, wTest)
	defer rTestCleanup2()

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
		// https://www.terraform.io/docs/enterprise/api/run.html#forcefully-cancel-a-run).
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

func TestRunsDiscard(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	wTest, wTestCleanup := createWorkspace(t, client, nil)
	defer wTestCleanup()

	rTest, rTestCleanup := createPlannedRun(t, client, wTest)
	defer rTestCleanup()

	t.Run("when the run exists", func(t *testing.T) {
		err := client.Runs.Discard(ctx, rTest.ID, RunDiscardOptions{})
		assert.NoError(t, err)
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
					"queued-at":  "2020-03-16T23:15:59+00:00",
					"errored-at": "2019-03-16T23:23:59+00:00",
				},
			},
		},
	}
	byteData, err := json.Marshal(data)
	require.NoError(t, err)

	responseBody := bytes.NewReader(byteData)
	run := &Run{}
	err = unmarshalResponse(responseBody, run)
	require.NoError(t, err)

	queuedParsedTime, err := time.Parse(time.RFC3339, "2020-03-16T23:15:59+00:00")
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
	assert.Equal(t, run.StatusTimestamps.QueuedAt, queuedParsedTime)
	assert.Equal(t, run.StatusTimestamps.ErroredAt, erroredParsedTime)
}
