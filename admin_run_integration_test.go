//go:build integration
// +build integration

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

func TestAdminRuns_List(t *testing.T) {
	skipIfNotCINode(t)
	skipIfCloud(t)

	client := testClient(t)
	ctx := context.Background()

	org, orgCleanup := createOrganization(t, client)
	defer orgCleanup()

	wTest, wTestCleanup := createWorkspace(t, client, org)
	defer wTestCleanup()

	rTest1, rTestCleanup1 := createRun(t, client, wTest)
	defer rTestCleanup1()
	rTest2, rTestCleanup2 := createRun(t, client, wTest)
	defer rTestCleanup2()

	t.Run("without list options", func(t *testing.T) {
		rl, err := client.Admin.Runs.List(ctx, nil)
		require.NoError(t, err)

		require.NotEmpty(t, rl.Items)
		assert.Equal(t, adminRunItemsContainsID(rl.Items, rTest1.ID), true)
		assert.Equal(t, adminRunItemsContainsID(rl.Items, rTest2.ID), true)
	})

	t.Run("with list options", func(t *testing.T) {
		rl, err := client.Admin.Runs.List(ctx, &AdminRunsListOptions{
			ListOptions: ListOptions{
				PageNumber: 999,
				PageSize:   100,
			},
		})
		require.NoError(t, err)
		// Out of range page number, so the items should be empty
		assert.Empty(t, rl.Items)
		assert.Equal(t, 999, rl.CurrentPage)

		rl, err = client.Admin.Runs.List(ctx, &AdminRunsListOptions{
			ListOptions: ListOptions{
				PageNumber: 1,
				PageSize:   100,
			},
		})
		require.NoError(t, err)
		require.NotEmpty(t, rl.Items)
		assert.Equal(t, 1, rl.CurrentPage)
		assert.Equal(t, adminRunItemsContainsID(rl.Items, rTest1.ID), true)
		assert.Equal(t, adminRunItemsContainsID(rl.Items, rTest2.ID), true)
	})

	t.Run("with workspace included", func(t *testing.T) {
		rl, err := client.Admin.Runs.List(ctx, &AdminRunsListOptions{
			Include: []AdminRunIncludeOpt{AdminRunWorkspace},
		})
		require.NoError(t, err)

		require.NotEmpty(t, rl.Items)
		require.NotNil(t, rl.Items[0].Workspace)
		assert.NotEmpty(t, rl.Items[0].Workspace.Name)
	})

	t.Run("with workspace.organization included", func(t *testing.T) {
		rl, err := client.Admin.Runs.List(ctx, &AdminRunsListOptions{
			Include: []AdminRunIncludeOpt{AdminRunWorkspaceOrg},
		})

		require.NoError(t, err)
		require.NotEmpty(t, rl.Items)

		require.NotNil(t, rl.Items[0].Workspace)
		require.NotNil(t, rl.Items[0].Workspace.Organization)
		assert.NotEmpty(t, rl.Items[0].Workspace.Organization.Name)
	})

	t.Run("with invalid Include option", func(t *testing.T) {
		_, err := client.Admin.Runs.List(ctx, &AdminRunsListOptions{
			Include: []AdminRunIncludeOpt{"workpsace"},
		})

		assert.Equal(t, err, ErrInvalidIncludeValue)
	})

	t.Run("with RunStatus.pending filter", func(t *testing.T) {
		r1, err := client.Runs.Read(ctx, rTest1.ID)
		require.NoError(t, err)
		r2, err := client.Runs.Read(ctx, rTest2.ID)
		require.NoError(t, err)

		// There should be pending Runs
		rl, err := client.Admin.Runs.List(ctx, &AdminRunsListOptions{
			RunStatus: string(RunPending),
		})
		require.NoError(t, err)
		require.NotEmpty(t, rl.Items)

		assert.Equal(t, r1.Status, RunPlanning)
		assert.Equal(t, adminRunItemsContainsID(rl.Items, r1.ID), false)
		assert.Equal(t, r2.Status, RunPending)
		assert.Equal(t, adminRunItemsContainsID(rl.Items, r2.ID), true)
	})

	t.Run("with RunStatus.applied filter", func(t *testing.T) {
		// There should be no applied Runs
		rl, err := client.Admin.Runs.List(ctx, &AdminRunsListOptions{
			RunStatus: string(RunApplied),
		})
		require.NoError(t, err)
		assert.Empty(t, rl.Items)
	})

	t.Run("with query", func(t *testing.T) {
		rl, err := client.Admin.Runs.List(ctx, &AdminRunsListOptions{
			Query: rTest1.ID,
		})
		require.NoError(t, err)

		require.NotEmpty(t, rl.Items)
		assert.Equal(t, adminRunItemsContainsID(rl.Items, rTest1.ID), true)
		assert.Equal(t, adminRunItemsContainsID(rl.Items, rTest2.ID), false)

		rl, err = client.Admin.Runs.List(ctx, &AdminRunsListOptions{
			Query: rTest2.ID,
		})
		require.NoError(t, err)

		require.NotEmpty(t, rl.Items)
		assert.Equal(t, adminRunItemsContainsID(rl.Items, rTest1.ID), false)
		assert.Equal(t, adminRunItemsContainsID(rl.Items, rTest2.ID), true)
	})
}

func TestAdminRuns_ForceCancel(t *testing.T) {
	skipIfNotCINode(t)
	skipIfCloud(t)

	client := testClient(t)
	ctx := context.Background()

	org, orgCleanup := createOrganization(t, client)
	defer orgCleanup()

	wTest, wTestCleanup := createWorkspace(t, client, org)
	defer wTestCleanup()

	// We need to create 2 runs here.
	// The first run will automatically be planned
	// so that one cannot be cancelled.
	rTest1, rCleanup1 := createRun(t, client, wTest)
	defer rCleanup1()
	// The second one will be pending until the first one is
	// confirmed or discarded, so we can cancel that one.
	rTest2, rCleanup2 := createRun(t, client, wTest)
	defer rCleanup2()

	assert.Equal(t, true, rTest1.Actions.IsCancelable)
	assert.Equal(t, true, rTest1.Permissions.CanForceCancel)

	assert.Equal(t, true, rTest2.Actions.IsCancelable)
	assert.Equal(t, true, rTest2.Permissions.CanForceCancel)

	t.Run("when the run does not exist", func(t *testing.T) {
		err := client.Admin.Runs.ForceCancel(ctx, "nonexisting", AdminRunForceCancelOptions{})
		assert.Equal(t, err, ErrResourceNotFound)
	})

	t.Run("with invalid run ID", func(t *testing.T) {
		err := client.Admin.Runs.ForceCancel(ctx, badIdentifier, AdminRunForceCancelOptions{})
		assert.EqualError(t, err, ErrInvalidRunID.Error())
	})

	t.Run("with can force cancel", func(t *testing.T) {
		rTestPlanning, err := client.Runs.Read(ctx, rTest1.ID)
		require.NoError(t, err)
		assert.Equal(t, RunPlanning, rTestPlanning.Status)

		require.NotNil(t, rTestPlanning.Actions)
		require.NotNil(t, rTestPlanning.Permissions)
		assert.Equal(t, true, rTestPlanning.Actions.IsCancelable)
		assert.Equal(t, true, rTestPlanning.Permissions.CanForceCancel)

		rTestPending, err := client.Runs.Read(ctx, rTest2.ID)
		require.NoError(t, err)
		assert.Equal(t, RunPending, rTestPending.Status)

		require.NotNil(t, rTestPlanning.Actions)
		require.NotNil(t, rTestPlanning.Permissions)
		assert.Equal(t, true, rTestPending.Actions.IsCancelable)
		assert.Equal(t, true, rTestPending.Permissions.CanForceCancel)

		comment1 := "Misclick"
		err = client.Admin.Runs.ForceCancel(ctx, rTestPending.ID, AdminRunForceCancelOptions{
			Comment: String(comment1),
		})
		require.NoError(t, err)

		rTestPendingResult, err := client.Runs.Read(ctx, rTestPending.ID)
		require.NoError(t, err)
		assert.Equal(t, RunCanceled, rTestPendingResult.Status)

		comment2 := "Another misclick"
		err = client.Admin.Runs.ForceCancel(ctx, rTestPlanning.ID, AdminRunForceCancelOptions{
			Comment: String(comment2),
		})
		require.NoError(t, err)

		rTestPlanningResult, err := client.Runs.Read(ctx, rTestPlanning.ID)
		require.NoError(t, err)
		assert.Equal(t, RunCanceled, rTestPlanningResult.Status)
	})
}

func TestAdminRuns_AdminRunsListOptions_valid(t *testing.T) {
	skipIfNotCINode(t)
	skipIfCloud(t)

	t.Run("has valid status", func(t *testing.T) {
		opts := AdminRunsListOptions{
			RunStatus: string(RunPending),
		}

		err := opts.valid()
		require.NoError(t, err)
	})

	t.Run("has invalid status", func(t *testing.T) {
		opts := AdminRunsListOptions{
			RunStatus: "random_status",
		}

		err := opts.valid()
		assert.Error(t, err)
	})

	t.Run("has invalid status, even with a valid one", func(t *testing.T) {
		statuses := fmt.Sprintf("%s,%s", string(RunPending), "random_status")
		opts := AdminRunsListOptions{
			RunStatus: statuses,
		}

		err := opts.valid()
		assert.Error(t, err)
	})

	t.Run("has trailing comma and trailing space", func(t *testing.T) {
		opts := AdminRunsListOptions{
			RunStatus: "pending, ",
		}

		err := opts.valid()
		require.NoError(t, err)
	})
}

func TestAdminRun_ForceCancel_Marshal(t *testing.T) {
	skipIfNotCINode(t)

	opts := AdminRunForceCancelOptions{
		Comment: String("cancel comment"),
	}

	reqBody, err := serializeRequestBody(&opts)
	require.NoError(t, err)
	req, err := retryablehttp.NewRequest("POST", "url", reqBody)
	require.NoError(t, err)
	bodyBytes, err := req.BodyBytes()
	require.NoError(t, err)

	expectedBody := `{"comment":"cancel comment"}`
	assert.Equal(t, expectedBody, string(bodyBytes))
}

func TestAdminRun_Unmarshal(t *testing.T) {
	skipIfNotCINode(t)

	data := map[string]interface{}{
		"data": map[string]interface{}{
			"type": "runs",
			"id":   "run-VCsNJXa59eUza53R",
			"attributes": map[string]interface{}{
				"created-at":  "2018-03-02T23:42:06.651Z",
				"has-changes": true,
				"status":      RunApplied,
				"status-timestamps": map[string]string{
					"plan-queued-at": "2020-03-16T23:15:59+00:00",
				},
			},
		},
	}
	byteData, err := json.Marshal(data)
	if err != nil {
		t.Fatal(err)
	}

	planQueuedParsedTime, err := time.Parse(time.RFC3339, "2020-03-16T23:15:59+00:00")
	require.NoError(t, err)

	adminRun := &AdminRun{}
	responseBody := bytes.NewReader(byteData)
	err = unmarshalResponse(responseBody, adminRun)
	require.NoError(t, err)
	assert.Equal(t, adminRun.ID, "run-VCsNJXa59eUza53R")
	assert.Equal(t, adminRun.HasChanges, true)
	assert.Equal(t, adminRun.Status, RunApplied)
	assert.Equal(t, adminRun.StatusTimestamps.PlanQueuedAt, planQueuedParsedTime)
}

func adminRunItemsContainsID(items []*AdminRun, id string) bool {
	hasID := false
	for _, item := range items {
		if item.ID == id {
			hasID = true
			break
		}
	}

	return hasID
}
