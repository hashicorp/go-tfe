package tfe

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAdminRuns_List(t *testing.T) {
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
		rl, err := client.Admin.Runs.List(ctx, AdminRunsListOptions{})
		require.NoError(t, err)

		assert.NotEmpty(t, rl.Items)
		assert.Equal(t, adminRunItemsContainsID(rl.Items, rTest1.ID), true)
		assert.Equal(t, adminRunItemsContainsID(rl.Items, rTest2.ID), true)
	})

	t.Run("with list options", func(t *testing.T) {
		rl, err := client.Admin.Runs.List(ctx, AdminRunsListOptions{
			ListOptions: ListOptions{
				PageNumber: 999,
				PageSize:   100,
			},
		})
		require.NoError(t, err)
		// Out of range page number, so the items should be empty
		assert.Empty(t, rl.Items)
		assert.Equal(t, 999, rl.CurrentPage)

		rl, err = client.Admin.Runs.List(ctx, AdminRunsListOptions{
			ListOptions: ListOptions{
				PageNumber: 1,
				PageSize:   100,
			},
		})
		require.NoError(t, err)
		assert.NotEmpty(t, rl.Items)
		assert.Equal(t, 1, rl.CurrentPage)
		assert.Equal(t, adminRunItemsContainsID(rl.Items, rTest1.ID), true)
		assert.Equal(t, adminRunItemsContainsID(rl.Items, rTest2.ID), true)
	})

	t.Run("with workspace included", func(t *testing.T) {
		rl, err := client.Admin.Runs.List(ctx, AdminRunsListOptions{
			Include: String("workspace"),
		})

		assert.NoError(t, err)

		assert.NotEmpty(t, rl.Items)
		assert.NotNil(t, rl.Items[0].Workspace)
		assert.NotEmpty(t, rl.Items[0].Workspace.Name)
	})

	t.Run("with workspace.organization included", func(t *testing.T) {
		rl, err := client.Admin.Runs.List(ctx, AdminRunsListOptions{
			Include: String("workspace.organization"),
		})

		assert.NoError(t, err)

		assert.NotEmpty(t, rl.Items)
		assert.NotNil(t, rl.Items[0].Workspace)
		assert.NotNil(t, rl.Items[0].Workspace.Organization)
		assert.NotEmpty(t, rl.Items[0].Workspace.Organization.Name)
	})

	t.Run("with RunStatus.pending filter", func(t *testing.T) {
		r1, err := client.Runs.Read(ctx, rTest1.ID)
		assert.NoError(t, err)
		r2, err := client.Runs.Read(ctx, rTest2.ID)
		assert.NoError(t, err)

		// There should be pending Runs
		rl, err := client.Admin.Runs.List(ctx, AdminRunsListOptions{
			RunStatus: String(string(RunPending)),
		})
		assert.NoError(t, err)
		assert.NotEmpty(t, rl.Items)

		assert.Equal(t, r1.Status, RunPlanning)
		assert.Equal(t, adminRunItemsContainsID(rl.Items, r1.ID), false)
		assert.Equal(t, r2.Status, RunPending)
		assert.Equal(t, adminRunItemsContainsID(rl.Items, r2.ID), true)
	})

	t.Run("with RunStatus.applied filter", func(t *testing.T) {
		// There should be no applied Runs
		rl, err := client.Admin.Runs.List(ctx, AdminRunsListOptions{
			RunStatus: String(string(RunApplied)),
		})
		assert.NoError(t, err)
		assert.Empty(t, rl.Items)
	})

	t.Run("with query", func(t *testing.T) {
		rl, err := client.Admin.Runs.List(ctx, AdminRunsListOptions{
			Query: String(rTest1.ID),
		})
		assert.NoError(t, err)

		assert.NotEmpty(t, rl.Items)
		assert.Equal(t, adminRunItemsContainsID(rl.Items, rTest1.ID), true)
		assert.Equal(t, adminRunItemsContainsID(rl.Items, rTest2.ID), false)

		rl, err = client.Admin.Runs.List(ctx, AdminRunsListOptions{
			Query: String(rTest2.ID),
		})
		assert.NoError(t, err)

		assert.NotEmpty(t, rl.Items)
		assert.Equal(t, adminRunItemsContainsID(rl.Items, rTest1.ID), false)
		assert.Equal(t, adminRunItemsContainsID(rl.Items, rTest2.ID), true)
	})
}

func TestAdminRuns_ForceCancel(t *testing.T) {
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
		assert.Equal(t, true, rTestPlanning.Actions.IsCancelable)
		assert.Equal(t, true, rTestPlanning.Permissions.CanForceCancel)

		rTestPending, err := client.Runs.Read(ctx, rTest2.ID)
		require.NoError(t, err)
		assert.Equal(t, RunPending, rTestPending.Status)
		assert.Equal(t, true, rTestPending.Actions.IsCancelable)
		assert.Equal(t, true, rTestPending.Permissions.CanForceCancel)

		comment1 := "Misclick"
		err = client.Admin.Runs.ForceCancel(ctx, rTestPending.ID, AdminRunForceCancelOptions{
			Comment: comment1,
		})
		require.NoError(t, err)

		rTestPendingResult, err := client.Runs.Read(ctx, rTestPending.ID)
		require.NoError(t, err)
		assert.Equal(t, RunCanceled, rTestPendingResult.Status)

		comment2 := "Another misclick"
		err = client.Admin.Runs.ForceCancel(ctx, rTestPlanning.ID, AdminRunForceCancelOptions{
			Comment: comment2,
		})
		require.NoError(t, err)

		rTestPlanningResult, err := client.Runs.Read(ctx, rTestPlanning.ID)
		require.NoError(t, err)
		assert.Equal(t, RunCanceled, rTestPlanningResult.Status)
	})
}

func TestAdminRuns_AdminRunsListOptions_valid(t *testing.T) {
	skipIfCloud(t)

	t.Run("has valid status", func(t *testing.T) {
		opts := AdminRunsListOptions{
			RunStatus: String(string(RunPending)),
		}

		err := opts.valid()
		assert.NoError(t, err)
	})

	t.Run("has invalid status", func(t *testing.T) {
		opts := AdminRunsListOptions{
			RunStatus: String("random_status"),
		}

		err := opts.valid()
		assert.Error(t, err)
	})

	t.Run("has invalid status, even with a valid one", func(t *testing.T) {
		statuses := fmt.Sprintf("%s,%s", string(RunPending), "random_status")
		opts := AdminRunsListOptions{
			RunStatus: String(statuses),
		}

		err := opts.valid()
		assert.Error(t, err)
	})
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
