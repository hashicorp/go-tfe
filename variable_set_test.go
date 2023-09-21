// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package tfe

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVariableSetsList(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	t.Cleanup(orgTestCleanup)

	vsTest1, vsTestCleanup1 := createVariableSet(t, client, orgTest, VariableSetCreateOptions{})
	t.Cleanup(vsTestCleanup1)
	vsTest2, vsTestCleanup2 := createVariableSet(t, client, orgTest, VariableSetCreateOptions{})
	t.Cleanup(vsTestCleanup2)

	t.Run("without list options", func(t *testing.T) {
		vsl, err := client.VariableSets.List(ctx, orgTest.Name, nil)
		require.NoError(t, err)
		require.NotEmpty(t, vsl.Items)
		assert.Contains(t, vsl.Items, vsTest1)
		assert.Contains(t, vsl.Items, vsTest2)

		t.Skip("paging not supported yet in API")
		assert.Equal(t, 1, vsl.CurrentPage)
		assert.Equal(t, 2, vsl.TotalCount)
	})

	t.Run("with list options", func(t *testing.T) {
		t.Skip("paging not supported yet in API")
		// Request a page number which is out of range. The result should
		// be successful, but return no results if the paging options are
		// properly passed along.
		vsl, err := client.VariableSets.List(ctx, orgTest.Name, &VariableSetListOptions{
			ListOptions: ListOptions{
				PageNumber: 999,
				PageSize:   100,
			},
		})
		require.NoError(t, err)
		assert.Empty(t, vsl.Items)
		assert.Equal(t, 999, vsl.CurrentPage)
		assert.Equal(t, 2, vsl.TotalCount)
	})

	t.Run("when Organization name is an invalid ID", func(t *testing.T) {
		vsl, err := client.VariableSets.List(ctx, badIdentifier, nil)
		assert.Nil(t, vsl)
		assert.EqualError(t, err, ErrInvalidOrg.Error())
	})
}

func TestVariableSetsListForWorkspace(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	t.Cleanup(orgTestCleanup)
	workspaceTest, workspaceTestCleanup := createWorkspace(t, client, orgTest)
	t.Cleanup(workspaceTestCleanup)

	vsTest1, vsTestCleanup1 := createVariableSet(t, client, orgTest, VariableSetCreateOptions{})
	t.Cleanup(vsTestCleanup1)
	vsTest2, vsTestCleanup2 := createVariableSet(t, client, orgTest, VariableSetCreateOptions{})
	t.Cleanup(vsTestCleanup2)

	applyVariableSetToWorkspace(t, client, vsTest1.ID, workspaceTest.ID)
	applyVariableSetToWorkspace(t, client, vsTest2.ID, workspaceTest.ID)

	t.Run("without list options", func(t *testing.T) {
		vsl, err := client.VariableSets.ListForWorkspace(ctx, workspaceTest.ID, nil)
		require.NoError(t, err)
		require.Len(t, vsl.Items, 2)

		ids := []string{vsTest1.ID, vsTest2.ID}
		for _, varset := range vsl.Items {
			assert.Contains(t, ids, varset.ID)
		}
	})

	t.Run("with list options", func(t *testing.T) {
		t.Skip("paging not supported yet in API")
		// Request a page number which is out of range. The result should
		// be successful, but return no results if the paging options are
		// properly passed along.
		vsl, err := client.VariableSets.ListForWorkspace(ctx, workspaceTest.ID, &VariableSetListOptions{
			ListOptions: ListOptions{
				PageNumber: 999,
				PageSize:   100,
			},
		})
		require.NoError(t, err)
		assert.Empty(t, vsl.Items)
		assert.Equal(t, 999, vsl.CurrentPage)
		assert.Equal(t, 2, vsl.TotalCount)
	})

	t.Run("when Workspace ID is an invalid ID", func(t *testing.T) {
		vsl, err := client.VariableSets.ListForWorkspace(ctx, badIdentifier, nil)
		assert.Nil(t, vsl)
		assert.EqualError(t, err, ErrInvalidWorkspaceID.Error())
	})
}

func TestVariableSetsListForProject(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	t.Cleanup(orgTestCleanup)
	projectTest, projectTestCleanup := createProject(t, client, orgTest)
	t.Cleanup(projectTestCleanup)

	vsTest1, vsTestCleanup1 := createVariableSet(t, client, orgTest, VariableSetCreateOptions{})
	t.Cleanup(vsTestCleanup1)
	vsTest2, vsTestCleanup2 := createVariableSet(t, client, orgTest, VariableSetCreateOptions{})
	t.Cleanup(vsTestCleanup2)

	applyVariableSetToProject(t, client, vsTest1.ID, projectTest.ID)
	applyVariableSetToProject(t, client, vsTest2.ID, projectTest.ID)

	t.Run("without list options", func(t *testing.T) {
		vsl, err := client.VariableSets.ListForProject(ctx, projectTest.ID, nil)
		require.NoError(t, err)
		require.Len(t, vsl.Items, 2)

		ids := []string{vsTest1.ID, vsTest2.ID}
		for _, varset := range vsl.Items {
			assert.Contains(t, ids, varset.ID)
		}
	})

	t.Run("with list options", func(t *testing.T) {
		vsl, err := client.VariableSets.ListForProject(ctx, projectTest.ID, &VariableSetListOptions{
			ListOptions: ListOptions{
				PageNumber: 999,
				PageSize:   100,
			},
		})
		require.NoError(t, err)
		assert.Empty(t, vsl.Items)
		assert.Equal(t, 999, vsl.CurrentPage)
		assert.Equal(t, 2, vsl.TotalCount)
	})

	t.Run("when Project ID is an invalid ID", func(t *testing.T) {
		vsl, err := client.VariableSets.ListForProject(ctx, badIdentifier, nil)
		assert.Nil(t, vsl)
		assert.EqualError(t, err, ErrInvalidProjectID.Error())
	})
}

func TestVariableSetsCreate(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	t.Cleanup(orgTestCleanup)

	t.Run("with valid options", func(t *testing.T) {
		options := VariableSetCreateOptions{
			Name:        String("varset"),
			Description: String("a variable set"),
			Global:      Bool(false),
		}

		vs, err := client.VariableSets.Create(ctx, orgTest.Name, &options)
		require.NoError(t, err)

		// Get refreshed view from the API
		refreshed, err := client.VariableSets.Read(ctx, vs.ID, nil)
		require.NoError(t, err)

		for _, item := range []*VariableSet{
			vs,
			refreshed,
		} {
			assert.NotEmpty(t, item.ID)
			assert.Equal(t, *options.Name, item.Name)
			assert.Equal(t, *options.Description, item.Description)
			assert.Equal(t, *options.Global, item.Global)
		}
	})

	t.Run("when options is missing name", func(t *testing.T) {
		vs, err := client.VariableSets.Create(ctx, "foo", &VariableSetCreateOptions{
			Global: Bool(true),
		})
		assert.Nil(t, vs)
		assert.EqualError(t, err, ErrRequiredName.Error())
	})

	t.Run("when options is missing global flag", func(t *testing.T) {
		vs, err := client.VariableSets.Create(ctx, "foo", &VariableSetCreateOptions{
			Name: String("foo"),
		})
		assert.Nil(t, vs)
		assert.EqualError(t, err, ErrRequiredGlobalFlag.Error())
	})
}

func TestVariableSetsWithEnforcedCreate(t *testing.T) {
	skipUnlessBeta(t)
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	t.Cleanup(orgTestCleanup)

	t.Run("with valid options", func(t *testing.T) {
		options := VariableSetCreateOptions{
			Name:        String("varset"),
			Description: String("a variable set"),
			Global:      Bool(false),
			Enforced:    Bool(false),
		}

		vs, err := client.VariableSets.Create(ctx, orgTest.Name, &options)
		require.NoError(t, err)

		// Get refreshed view from the API
		refreshed, err := client.VariableSets.Read(ctx, vs.ID, nil)
		require.NoError(t, err)

		for _, item := range []*VariableSet{
			vs,
			refreshed,
		} {
			assert.NotEmpty(t, item.ID)
			assert.Equal(t, *options.Name, item.Name)
			assert.Equal(t, *options.Description, item.Description)
			assert.Equal(t, *options.Global, item.Global)
			assert.Equal(t, *options.Enforced, item.Enforced)
		}
	})

	t.Run("when options is missing name", func(t *testing.T) {
		vs, err := client.VariableSets.Create(ctx, "foo", &VariableSetCreateOptions{
			Global: Bool(true),
		})
		assert.Nil(t, vs)
		assert.EqualError(t, err, ErrRequiredName.Error())
	})

	t.Run("when options is missing global flag", func(t *testing.T) {
		vs, err := client.VariableSets.Create(ctx, "foo", &VariableSetCreateOptions{
			Name: String("foo"),
		})
		assert.Nil(t, vs)
		assert.EqualError(t, err, ErrRequiredGlobalFlag.Error())
	})
}

func TestVariableSetsRead(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	t.Cleanup(orgTestCleanup)

	vsTest, vsTestCleanup := createVariableSet(t, client, orgTest, VariableSetCreateOptions{})
	t.Cleanup(vsTestCleanup)

	t.Run("when the variable set exists", func(t *testing.T) {
		vs, err := client.VariableSets.Read(ctx, vsTest.ID, nil)
		require.NoError(t, err)
		assert.Equal(t, vsTest, vs)
	})

	t.Run("when variable set does not exist", func(t *testing.T) {
		vs, err := client.VariableSets.Read(ctx, "nonexisting", nil)
		assert.Nil(t, vs)
		assert.Error(t, err)
	})
}

func TestVariableSetsUpdate(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	t.Cleanup(orgTestCleanup)

	vsTest, _ := createVariableSet(t, client, orgTest, VariableSetCreateOptions{
		Name:        String("OriginalName"),
		Description: String("Original Description"),
		Global:      Bool(false),
	})

	t.Run("when updating a subset of values", func(t *testing.T) {
		options := VariableSetUpdateOptions{
			Name:        String("UpdatedName"),
			Description: String("Updated Description"),
			Global:      Bool(true),
		}

		vsAfter, err := client.VariableSets.Update(ctx, vsTest.ID, &options)
		require.NoError(t, err)

		assert.Equal(t, *options.Name, vsAfter.Name)
		assert.Equal(t, *options.Description, vsAfter.Description)
		assert.Equal(t, *options.Global, vsAfter.Global)
	})

	t.Run("when options has an invalid variable set ID", func(t *testing.T) {
		vsAfter, err := client.VariableSets.Update(ctx, badIdentifier, &VariableSetUpdateOptions{
			Name:        String("UpdatedName"),
			Description: String("Updated Description"),
			Global:      Bool(true),
		})
		assert.Nil(t, vsAfter)
		assert.EqualError(t, err, ErrInvalidVariableSetID.Error())
	})
}

func TestVariableSetsUpdateWithEnforced(t *testing.T) {
	skipUnlessBeta(t)
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	t.Cleanup(orgTestCleanup)

	vsTest, _ := createVariableSet(t, client, orgTest, VariableSetCreateOptions{
		Name:        String("OriginalName"),
		Description: String("Original Description"),
		Global:      Bool(false),
		Enforced:    Bool(false),
	})

	t.Run("when updating a subset of values", func(t *testing.T) {
		options := VariableSetUpdateOptions{
			Name:        String("UpdatedName"),
			Description: String("Updated Description"),
			Global:      Bool(true),
			Enforced:    Bool(true),
		}

		vsAfter, err := client.VariableSets.Update(ctx, vsTest.ID, &options)
		require.NoError(t, err)

		assert.Equal(t, *options.Name, vsAfter.Name)
		assert.Equal(t, *options.Description, vsAfter.Description)
		assert.Equal(t, *options.Global, vsAfter.Global)
		assert.Equal(t, *options.Enforced, vsAfter.Enforced)
	})

	t.Run("when options has an invalid variable set ID", func(t *testing.T) {
		vsAfter, err := client.VariableSets.Update(ctx, badIdentifier, &VariableSetUpdateOptions{
			Name:        String("UpdatedName"),
			Description: String("Updated Description"),
			Global:      Bool(true),
			Enforced:    Bool(true),
		})
		assert.Nil(t, vsAfter)
		assert.EqualError(t, err, ErrInvalidVariableSetID.Error())
	})
}

func TestVariableSetsDelete(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	t.Cleanup(orgTestCleanup)

	// Do not defer cleanup since the next step in this test is to delete it
	vsTest, _ := createVariableSet(t, client, orgTest, VariableSetCreateOptions{})

	t.Run("with valid ID", func(t *testing.T) {
		err := client.VariableSets.Delete(ctx, vsTest.ID)
		require.NoError(t, err)

		// Try loading the variable set - it should fail.
		_, err = client.VariableSets.Read(ctx, vsTest.ID, nil)
		assert.Equal(t, ErrResourceNotFound, err)
	})

	t.Run("when ID is invalid", func(t *testing.T) {
		err := client.VariableSets.Delete(ctx, badIdentifier)
		assert.EqualError(t, err, ErrInvalidVariableSetID.Error())
	})
}

func TestVariableSetsApplyToAndRemoveFromWorkspaces(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	t.Cleanup(orgTestCleanup)

	vsTest, vsTestCleanup := createVariableSet(t, client, orgTest, VariableSetCreateOptions{})
	t.Cleanup(vsTestCleanup)

	wTest1, wTest1Cleanup := createWorkspace(t, client, orgTest)
	defer wTest1Cleanup()
	wTest2, wTest2Cleanup := createWorkspace(t, client, orgTest)
	defer wTest2Cleanup()

	t.Run("with first workspace added", func(t *testing.T) {
		options := VariableSetApplyToWorkspacesOptions{
			Workspaces: []*Workspace{wTest1},
		}

		err := client.VariableSets.ApplyToWorkspaces(ctx, vsTest.ID, &options)
		require.NoError(t, err)

		vsAfter, err := client.VariableSets.Read(ctx, vsTest.ID, nil)
		require.NoError(t, err)

		// Variable set should be applied to [wTest1]
		assert.Equal(t, 1, len(vsAfter.Workspaces))
		assert.Equal(t, wTest1.ID, vsAfter.Workspaces[0].ID)
	})

	t.Run("with second workspace added", func(t *testing.T) {
		options := VariableSetApplyToWorkspacesOptions{
			Workspaces: []*Workspace{wTest2},
		}

		err := client.VariableSets.ApplyToWorkspaces(ctx, vsTest.ID, &options)
		require.NoError(t, err)

		vsAfter, err := client.VariableSets.Read(ctx, vsTest.ID, nil)
		require.NoError(t, err)

		// Variable set should be applied to [wTest1, wTest2]
		assert.Equal(t, 2, len(vsAfter.Workspaces))
		wsIDs := []string{vsAfter.Workspaces[0].ID, vsAfter.Workspaces[1].ID}

		assert.Contains(t, wsIDs, wTest1.ID)
		assert.Contains(t, wsIDs, wTest2.ID)
	})

	t.Run("with first workspace removed", func(t *testing.T) {
		options := VariableSetRemoveFromWorkspacesOptions{
			Workspaces: []*Workspace{wTest1},
		}

		err := client.VariableSets.RemoveFromWorkspaces(ctx, vsTest.ID, &options)
		require.NoError(t, err)

		vsAfter, err := client.VariableSets.Read(ctx, vsTest.ID, nil)
		require.NoError(t, err)

		// Variable set should be applied to [wTest2]
		assert.Equal(t, 1, len(vsAfter.Workspaces))
		assert.Equal(t, wTest2.ID, vsAfter.Workspaces[0].ID)
	})

	t.Run("when variable set ID is invalid", func(t *testing.T) {
		applyOptions := VariableSetApplyToWorkspacesOptions{
			Workspaces: []*Workspace{wTest1},
		}

		err := client.VariableSets.ApplyToWorkspaces(ctx, badIdentifier, &applyOptions)
		assert.EqualError(t, err, ErrInvalidVariableSetID.Error())

		removeOptions := VariableSetRemoveFromWorkspacesOptions{
			Workspaces: []*Workspace{wTest1},
		}
		err = client.VariableSets.RemoveFromWorkspaces(ctx, badIdentifier, &removeOptions)
		assert.EqualError(t, err, ErrInvalidVariableSetID.Error())
	})

	t.Run("when workspace ID is invalid", func(t *testing.T) {
		badWorkspace := &Workspace{
			ID: badIdentifier,
		}

		applyOptions := VariableSetApplyToWorkspacesOptions{
			Workspaces: []*Workspace{badWorkspace},
		}

		err := client.VariableSets.ApplyToWorkspaces(ctx, vsTest.ID, &applyOptions)
		assert.EqualError(t, err, ErrRequiredWorkspaceID.Error())

		removeOptions := VariableSetRemoveFromWorkspacesOptions{
			Workspaces: []*Workspace{badWorkspace},
		}

		err = client.VariableSets.RemoveFromWorkspaces(ctx, vsTest.ID, &removeOptions)
		assert.EqualError(t, err, ErrRequiredWorkspaceID.Error())
	})
}

func TestVariableSetsApplyToAndRemoveFromProjects(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	t.Cleanup(orgTestCleanup)

	vsTest, vsTestCleanup := createVariableSet(t, client, orgTest, VariableSetCreateOptions{})
	t.Cleanup(vsTestCleanup)

	prjTest1, prjTest1Cleanup := createProject(t, client, orgTest)
	defer prjTest1Cleanup()
	prjTest2, prjTest2Cleanup := createProject(t, client, orgTest)
	defer prjTest2Cleanup()
	t.Run("with first project added", func(t *testing.T) {
		options := VariableSetApplyToProjectsOptions{
			Projects: []*Project{prjTest1},
		}

		err := client.VariableSets.ApplyToProjects(ctx, vsTest.ID, options)
		require.NoError(t, err)

		vsAfter, err := client.VariableSets.Read(ctx, vsTest.ID, nil)
		require.NoError(t, err)

		// Variable set should be applied to [prjTest1]
		assert.Equal(t, 1, len(vsAfter.Projects))
		assert.Equal(t, prjTest1.ID, vsAfter.Projects[0].ID)
	})

	t.Run("with second project added", func(t *testing.T) {
		options := VariableSetApplyToProjectsOptions{
			Projects: []*Project{prjTest2},
		}

		err := client.VariableSets.ApplyToProjects(ctx, vsTest.ID, options)
		require.NoError(t, err)

		vsAfter, err := client.VariableSets.Read(ctx, vsTest.ID, nil)
		require.NoError(t, err)

		// Variable set should be applied to [prjTest1, prjTest2]
		assert.Equal(t, 2, len(vsAfter.Projects))
		prjIDs := []string{vsAfter.Projects[0].ID, vsAfter.Projects[1].ID}

		assert.Contains(t, prjIDs, prjTest1.ID)
		assert.Contains(t, prjIDs, prjTest2.ID)
	})

	t.Run("with first project removed", func(t *testing.T) {
		options := VariableSetRemoveFromProjectsOptions{
			Projects: []*Project{prjTest1},
		}

		err := client.VariableSets.RemoveFromProjects(ctx, vsTest.ID, options)
		require.NoError(t, err)

		vsAfter, err := client.VariableSets.Read(ctx, vsTest.ID, nil)
		require.NoError(t, err)

		// Variable set should be applied to [wTest2]
		assert.Equal(t, 1, len(vsAfter.Projects))
		assert.Equal(t, prjTest2.ID, vsAfter.Projects[0].ID)
	})

	t.Run("when variable set ID is invalid", func(t *testing.T) {
		applyOptions := VariableSetApplyToProjectsOptions{
			Projects: []*Project{prjTest1},
		}

		err := client.VariableSets.ApplyToProjects(ctx, badIdentifier, applyOptions)
		assert.EqualError(t, err, ErrInvalidVariableSetID.Error())

		removeOptions := VariableSetRemoveFromProjectsOptions{
			Projects: []*Project{prjTest1},
		}
		err = client.VariableSets.RemoveFromProjects(ctx, badIdentifier, removeOptions)
		assert.EqualError(t, err, ErrInvalidVariableSetID.Error())
	})

	t.Run("when project ID is invalid", func(t *testing.T) {
		badProject := &Project{
			ID: badIdentifier,
		}

		applyOptions := VariableSetApplyToProjectsOptions{
			Projects: []*Project{badProject},
		}

		err := client.VariableSets.ApplyToProjects(ctx, vsTest.ID, applyOptions)
		assert.EqualError(t, err, ErrRequiredProjectID.Error())

		removeOptions := VariableSetRemoveFromProjectsOptions{
			Projects: []*Project{badProject},
		}

		err = client.VariableSets.RemoveFromProjects(ctx, vsTest.ID, removeOptions)
		assert.EqualError(t, err, ErrRequiredProjectID.Error())
	})
}

func TestVariableSetsUpdateWorkspaces(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	t.Cleanup(orgTestCleanup)

	vsTest, vsTestCleanup := createVariableSet(t, client, orgTest, VariableSetCreateOptions{})
	t.Cleanup(vsTestCleanup)

	wTest, wTestCleanup := createWorkspace(t, client, orgTest)
	t.Cleanup(wTestCleanup)

	t.Run("with valid workspaces", func(t *testing.T) {
		options := VariableSetUpdateWorkspacesOptions{
			Workspaces: []*Workspace{wTest},
		}

		vsAfter, err := client.VariableSets.UpdateWorkspaces(ctx, vsTest.ID, &options)
		require.NoError(t, err)

		assert.Equal(t, len(options.Workspaces), len(vsAfter.Workspaces))
		assert.Equal(t, options.Workspaces[0].ID, vsAfter.Workspaces[0].ID)

		options = VariableSetUpdateWorkspacesOptions{
			Workspaces: []*Workspace{},
		}

		vsAfter, err = client.VariableSets.UpdateWorkspaces(ctx, vsTest.ID, &options)
		require.NoError(t, err)

		assert.Equal(t, len(options.Workspaces), len(vsAfter.Workspaces))
	})
}
