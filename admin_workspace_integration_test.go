// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package tfe

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// BEWARE: The admin workspaces API can view all of the workspaces created by
// EVERY test organization in EVERY concurrent test run (or other usage) for the
// current HCP Terraform instance. It's generally not safe to assume that the workspaces
// you create in a given test will be within the first page of list results, so
// you might have to get creative and/or settle for less when testing the
// behavior of these endpoints.

func TestAdminWorkspaces_ListWithFilter(t *testing.T) {
	skipUnlessEnterprise(t)

	client := testClient(t)
	ctx := context.Background()

	org, orgCleanup := createOrganization(t, client)
	defer orgCleanup()

	wTest1, wTest1Cleanup := createWorkspace(t, client, org)
	defer wTest1Cleanup()

	wTest2, wTest2Cleanup := createWorkspace(t, client, org)
	defer wTest2Cleanup()

	t.Run("when filtering workspaces on a current run status", func(t *testing.T) {
		_, appliedCleanup := createRunApply(t, client, wTest1)
		t.Cleanup(appliedCleanup)

		_, unAppliedCleanup := createRunUnapplied(t, client, wTest2)
		t.Cleanup(unAppliedCleanup)

		wl, err := client.Admin.Workspaces.List(ctx, &AdminWorkspaceListOptions{
			Filter: string(RunApplied), Include: []AdminWorkspaceIncludeOpt{AdminWorkspaceCurrentRun},
		})

		require.NoError(t, err)
		require.NotEmpty(t, wl.Items)
		assert.Equal(t, wl.Items[0].CurrentRun.Status, RunApplied)
		assert.NotContains(t, wl.Items, wTest2)
	})
}

func TestAdminWorkspaces_ListWithSort(t *testing.T) {
	skipUnlessEnterprise(t)

	client := testClient(t)
	ctx := context.Background()

	org, orgCleanup := createOrganization(t, client)
	defer orgCleanup()

	wTest1, wTest1Cleanup := createWorkspace(t, client, org)
	defer wTest1Cleanup()

	wTest2, wTest2Cleanup := createWorkspace(t, client, org)
	defer wTest2Cleanup()

	t.Run("when sorting by workspace names", func(t *testing.T) {
		wl, err := client.Admin.Workspaces.List(ctx, &AdminWorkspaceListOptions{
			Sort: "name",
		})
		require.NoError(t, err)
		require.NotEmpty(t, wl.Items)
		require.GreaterOrEqual(t, len(wl.Items), 2)
		assert.Equal(t, wl.Items[0].Name < wl.Items[1].Name, true)
	})

	t.Run("when sorting workspaces on current-run.created-at", func(t *testing.T) {
		_, unappliedCleanup1 := createRunUnapplied(t, client, wTest1)
		t.Cleanup(unappliedCleanup1)

		_, unappliedCleanup2 := createRunUnapplied(t, client, wTest2)
		t.Cleanup(unappliedCleanup2)

		wl, err := client.Admin.Workspaces.List(ctx, &AdminWorkspaceListOptions{
			Include: []AdminWorkspaceIncludeOpt{AdminWorkspaceCurrentRun},
			Sort:    "current-run.created-at",
		})

		require.NoError(t, err)
		require.NotEmpty(t, wl.Items)
		require.GreaterOrEqual(t, len(wl.Items), 2)
		assert.True(t, wl.Items[1].CurrentRun.CreatedAt.After(wl.Items[0].CurrentRun.CreatedAt))
	})
}

func TestAdminWorkspaces_List(t *testing.T) {
	skipUnlessEnterprise(t)

	client := testClient(t)
	ctx := context.Background()

	org, orgCleanup := createOrganization(t, client)
	defer orgCleanup()

	wTest1, wTest1Cleanup := createWorkspace(t, client, org)
	defer wTest1Cleanup()

	wTest2, wTest2Cleanup := createWorkspace(t, client, org)
	defer wTest2Cleanup()

	t.Run("without list options", func(t *testing.T) {
		wl, err := client.Admin.Workspaces.List(ctx, nil)
		require.NoError(t, err)

		require.GreaterOrEqual(t, len(wl.Items), 2)
	})

	t.Run("with list options", func(t *testing.T) {
		wl, err := client.Admin.Workspaces.List(ctx, &AdminWorkspaceListOptions{
			ListOptions: ListOptions{
				PageNumber: 999,
				PageSize:   100,
			},
		})
		require.NoError(t, err)
		// Out of range page number, so the items should be empty
		assert.Empty(t, wl.Items)
		assert.Equal(t, 999, wl.CurrentPage)

		wl, err = client.Admin.Workspaces.List(ctx, &AdminWorkspaceListOptions{
			ListOptions: ListOptions{
				PageNumber: 1,
				PageSize:   100,
			},
		})
		require.NoError(t, err)
		assert.Equal(t, 1, wl.CurrentPage)
		assert.Equal(t, adminWorkspaceItemsContainsID(wl.Items, wTest1.ID), true)
		assert.Equal(t, adminWorkspaceItemsContainsID(wl.Items, wTest2.ID), true)
	})

	t.Run("when searching a known workspace", func(t *testing.T) {
		// Use a known workspace prefix as search attribute. The result
		// should be successful and only contain the matching workspace.
		wl, err := client.Admin.Workspaces.List(ctx, &AdminWorkspaceListOptions{
			Query: wTest1.Name,
		})
		require.NoError(t, err)
		assert.Equal(t, adminWorkspaceItemsContainsID(wl.Items, wTest1.ID), true)
		assert.Equal(t, adminWorkspaceItemsContainsID(wl.Items, wTest2.ID), false)
		assert.Equal(t, 1, wl.CurrentPage)
		assert.Equal(t, true, wl.TotalCount == 1)
	})

	t.Run("when searching an unknown workspace", func(t *testing.T) {
		// Use a nonexisting workspace name as search attribute. The result
		// should be successful, but return no results.
		wl, err := client.Admin.Workspaces.List(ctx, &AdminWorkspaceListOptions{
			Query: "nonexisting",
		})
		require.NoError(t, err)
		assert.Empty(t, wl.Items)
		assert.Equal(t, 1, wl.CurrentPage)
		assert.Equal(t, 0, wl.TotalCount)
	})

	t.Run("with organization included", func(t *testing.T) {
		wl, err := client.Admin.Workspaces.List(ctx, &AdminWorkspaceListOptions{
			Include: []AdminWorkspaceIncludeOpt{AdminWorkspaceOrg},
		})

		require.NoError(t, err)
		require.NotEmpty(t, wl.Items)
		require.NotNil(t, wl.Items[0].Organization)
		assert.NotEmpty(t, wl.Items[0].Organization.Name)
	})

	// This sub-test should remain last because it creates a run that does not apply
	// Any subsequent runs will be queued until a timeout is triggered
	t.Run("with current_run included", func(t *testing.T) {
		cvTest, cvCleanup := createUploadedConfigurationVersion(t, client, wTest1)
		defer cvCleanup()

		runOpts := RunCreateOptions{
			ConfigurationVersion: cvTest,
			Workspace:            wTest1,
		}
		run, err := client.Runs.Create(ctx, runOpts)
		require.NoError(t, err)

		wl, err := client.Admin.Workspaces.List(ctx, &AdminWorkspaceListOptions{
			Include: []AdminWorkspaceIncludeOpt{AdminWorkspaceCurrentRun},
		})

		require.NoError(t, err)

		require.NotEmpty(t, wl.Items)
		require.NotNil(t, wl.Items[0].CurrentRun)
		assert.Equal(t, wl.Items[0].CurrentRun.ID, run.ID)
	})
}

func TestAdminWorkspaces_Read(t *testing.T) {
	skipUnlessEnterprise(t)

	client := testClient(t)
	ctx := context.Background()

	t.Run("it fails to read a workspace with an invalid name", func(t *testing.T) {
		workspace, err := client.Admin.Workspaces.Read(ctx, "")
		require.Error(t, err)
		assert.EqualError(t, err, ErrInvalidWorkspaceValue.Error())
		assert.Nil(t, workspace)
	})

	t.Run("it fails to read a workspace that is non existent", func(t *testing.T) {
		workspaceID := fmt.Sprintf("non-existent-%s", randomString(t))
		adminWorkspace, err := client.Admin.Workspaces.Read(ctx, workspaceID)
		require.Error(t, err)
		assert.EqualError(t, err, ErrResourceNotFound.Error())
		assert.Nil(t, adminWorkspace)
	})

	t.Run("it reads a workspace successfully", func(t *testing.T) {
		org, orgCleanup := createOrganization(t, client)
		defer orgCleanup()

		workspace, workspaceCleanup := createWorkspace(t, client, org)
		defer workspaceCleanup()

		adminWorkspace, err := client.Admin.Workspaces.Read(ctx, workspace.ID)
		require.NoError(t, err)
		require.NotNilf(t, adminWorkspace, "Admin Workspace is not nil")
		assert.Equal(t, adminWorkspace.ID, workspace.ID)
		assert.Equal(t, adminWorkspace.Name, workspace.Name)
		assert.Equal(t, adminWorkspace.Locked, workspace.Locked)
	})
}

func TestAdminWorkspaces_Delete(t *testing.T) {
	skipUnlessEnterprise(t)

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
		require.NoError(t, err)
		require.NotNilf(t, adminWorkspace, "Admin Workspace is not nil")
		assert.Equal(t, adminWorkspace.ID, workspace.ID)

		err = client.Admin.Workspaces.Delete(ctx, adminWorkspace.ID)
		require.NoError(t, err)

		// Cannot find deleted workspace
		_, err = client.Admin.Workspaces.Read(ctx, workspace.ID)
		assert.Error(t, err)
		assert.EqualError(t, err, ErrResourceNotFound.Error())
	})
}

func adminWorkspaceItemsContainsID(items []*AdminWorkspace, id string) bool {
	hasID := false
	for _, item := range items {
		if item.ID == id {
			hasID = true
			break
		}
	}

	return hasID
}

func TestAdminWorkspace_Unmarshal(t *testing.T) {
	data := map[string]interface{}{
		"data": map[string]interface{}{
			"type": "workspaces",
			"id":   "workspaces-VCsNJXa59eUza53R",
			"attributes": map[string]interface{}{
				"name":   "workspace-name",
				"locked": false,
				"vcs-repo": map[string]string{
					"identifier": "github",
				},
			},
		},
	}
	byteData, err := json.Marshal(data)
	if err != nil {
		t.Fatal(err)
	}

	adminWorkspace := &AdminWorkspace{}
	responseBody := bytes.NewReader(byteData)
	err = unmarshalResponse(responseBody, adminWorkspace)
	require.NoError(t, err)
	assert.Equal(t, adminWorkspace.ID, "workspaces-VCsNJXa59eUza53R")
	assert.Equal(t, adminWorkspace.Name, "workspace-name")
	assert.Equal(t, adminWorkspace.Locked, false)
	assert.Equal(t, adminWorkspace.VCSRepo.Identifier, "github")
}
