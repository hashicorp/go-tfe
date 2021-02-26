package tfe

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAdminWorkspaces_List(t *testing.T) {
	skipIfCloud(t)

	client := testClient(t)
	ctx := context.Background()

	org, orgCleanup := createOrganization(t, client)
	defer orgCleanup()

	wTest1, wTest1Cleanup := createWorkspace(t, client, org)
	defer wTest1Cleanup()

	wTest2, wTest2Cleanup := createWorkspace(t, client, org)
	defer wTest2Cleanup()

	t.Run("without list options", func(t *testing.T) {
		wl, err := client.Admin.Workspaces.List(ctx, AdminWorkspaceListOptions{})
		require.NoError(t, err)

		assert.Equal(t, workspaceItemsContainsID(wl.Items, wTest1.ID), true)
		assert.Equal(t, workspaceItemsContainsID(wl.Items, wTest2.ID), true)
	})

	t.Run("with list options", func(t *testing.T) {
		wl, err := client.Admin.Workspaces.List(ctx, AdminWorkspaceListOptions{
			ListOptions: ListOptions{
				PageNumber: 999,
				PageSize:   100,
			},
		})
		require.NoError(t, err)
		// Out of range page number, so the items should be empty
		assert.Empty(t, wl.Items)
		assert.Equal(t, 999, wl.CurrentPage)

		wl, err = client.Admin.Workspaces.List(ctx, AdminWorkspaceListOptions{
			ListOptions: ListOptions{
				PageNumber: 1,
				PageSize:   100,
			},
		})
		require.NoError(t, err)
		assert.Equal(t, 1, wl.CurrentPage)
		assert.Equal(t, workspaceItemsContainsID(wl.Items, wTest1.ID), true)
		assert.Equal(t, workspaceItemsContainsID(wl.Items, wTest2.ID), true)
	})

	t.Run("when searching a known workspace", func(t *testing.T) {
		// Use a known workspace prefix as search attribute. The result
		// should be successful and only contain the matching workspace.
		wl, err := client.Admin.Workspaces.List(ctx, AdminWorkspaceListOptions{
			Query: String(wTest1.Name),
		})
		require.NoError(t, err)
		assert.Equal(t, workspaceItemsContainsID(wl.Items, wTest1.ID), true)
		assert.Equal(t, workspaceItemsContainsID(wl.Items, wTest2.ID), false)
		assert.Equal(t, 1, wl.CurrentPage)
		assert.Equal(t, true, wl.TotalCount == 1)
	})

	t.Run("when searching an unknown workspace", func(t *testing.T) {
		// Use a nonexisting workspace name as search attribute. The result
		// should be successful, but return no results.
		wl, err := client.Admin.Workspaces.List(ctx, AdminWorkspaceListOptions{
			Query: String("nonexisting"),
		})
		require.NoError(t, err)
		assert.Empty(t, wl.Items)
		assert.Equal(t, 1, wl.CurrentPage)
		assert.Equal(t, 0, wl.TotalCount)
	})

	t.Run("with organization included", func(t *testing.T) {
		wl, err := client.Admin.Workspaces.List(ctx, AdminWorkspaceListOptions{
			Include: String("organization"),
		})

		assert.NoError(t, err)

		assert.NotEmpty(t, wl.Items)
		assert.NotNil(t, wl.Items[0].Organization)
		assert.NotEmpty(t, wl.Items[0].Organization.Name)
	})

	t.Run("with current_run included", func(t *testing.T) {
		cvTest, cvCleanup := createUploadedConfigurationVersion(t, client, wTest1)
		defer cvCleanup()

		runOpts := RunCreateOptions{
			ConfigurationVersion: cvTest,
			Workspace:            wTest1,
		}
		run, err := client.Runs.Create(ctx, runOpts)
		assert.NoError(t, err)

		wl, err := client.Admin.Workspaces.List(ctx, AdminWorkspaceListOptions{
			Include: String("current_run"),
		})

		assert.NoError(t, err)

		assert.NotEmpty(t, wl.Items)
		assert.NotNil(t, wl.Items[0].CurrentRun)
		assert.Equal(t, wl.Items[0].CurrentRun.ID, run.ID)
	})

	t.Run("with invalid resource included", func(t *testing.T) {
		wl, err := client.Admin.Workspaces.List(ctx, AdminWorkspaceListOptions{
			Include: String("invalid-included-resource"),
		})
		assert.Nil(t, wl)
		assert.Error(t, err)
		assert.EqualError(t, err, ErrInvalidIncludeParam.Error())
	})
}

func TestAdminWorkspaces_Read(t *testing.T) {
	skipIfCloud(t)

	client := testClient(t)
	ctx := context.Background()

	t.Run("it fails to read a workspace with an invalid name", func(t *testing.T) {
		workspace, err := client.Admin.Workspaces.Read(ctx, "")
		require.Error(t, err)
		assert.EqualError(t, err, ErrInvalidOrg.Error())
		assert.Nil(t, workspace)
	})

	t.Run("it fails to read a workspace that is non existant", func(t *testing.T) {
		workspaceID := fmt.Sprintf("non-existent-%s", randomString(t))
		adminWorkspace, err := client.Admin.Workspaces.Read(ctx, workspaceID)
		require.Error(t, err)
		assert.EqualError(t, err, ErrResourceNotFound.Error())
		assert.Nil(t, adminWorkspace)
	})

	t.Run("it reads a worksapce successfully", func(t *testing.T) {
		org, orgCleanup := createOrganization(t, client)
		defer orgCleanup()

		workspace, workspaceCleanup := createWorkspace(t, client, org)
		defer workspaceCleanup()

		adminWorkspace, err := client.Admin.Workspaces.Read(ctx, workspace.ID)
		assert.NoError(t, err)
		assert.NotNilf(t, adminWorkspace, "Admin Workspace is not nil")
		assert.Equal(t, adminWorkspace.ID, workspace.ID)

		// attributes part of an AdminWorkspace response that are not null
		assert.NotNilf(t, adminWorkspace.Name, "Name is not nil")
		assert.NotNilf(t, adminWorkspace.Locked, "Locked is not nil")
	})
}

func TestAdminWorkspaces_Delete(t *testing.T) {
	skipIfCloud(t)

	client := testClient(t)
	ctx := context.Background()

	t.Run("it fails to delete an organization with an invalid id", func(t *testing.T) {
		err := client.Admin.Workspaces.Delete(ctx, "")
		require.Error(t, err)
		assert.EqualError(t, err, ErrInvalidWorkspaceValue.Error())
	})

	t.Run("it fails to delete an organization with an bad org name", func(t *testing.T) {
		workspaceID := fmt.Sprintf("non-existent-%s", randomString(t))
		err := client.Admin.Workspaces.Delete(ctx, workspaceID)
		require.Error(t, err)
		assert.EqualError(t, err, ErrResourceNotFound.Error())
	})

	t.Run("it deletes a workspace successfully", func(t *testing.T) {
		org, orgCleanup := createOrganization(t, client)
		defer orgCleanup()

		workspace, _ := createWorkspace(t, client, org)

		adminWorkspace, err := client.Admin.Workspaces.Read(ctx, workspace.ID)
		assert.NoError(t, err)
		assert.NotNilf(t, adminWorkspace, "Admin Workspace is not nil")
		assert.Equal(t, adminWorkspace.ID, workspace.ID)

		err = client.Admin.Workspaces.Delete(ctx, adminWorkspace.ID)
		assert.NoError(t, err)

		// Cannot find deleted workspace
		_, err = client.Admin.Workspaces.Read(ctx, workspace.ID)
		assert.Error(t, err)
		assert.EqualError(t, err, ErrResourceNotFound.Error())
	})
}

func workspaceItemsContainsID(items []*AdminWorkspace, id string) bool {
	hasID := false
	for _, item := range items {
		if item.ID == id {
			hasID = true
			break
		}
	}

	return hasID
}
