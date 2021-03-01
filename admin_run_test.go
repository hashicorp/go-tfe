package tfe

import (
	"context"
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

	rTest1, _ := createRun(t, client, wTest)
	rTest2, _ := createRun(t, client, wTest)

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
