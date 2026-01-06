// Copyright IBM Corp. 2018, 2025
// SPDX-License-Identifier: MPL-2.0

package tfe

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVariableSetsList(t *testing.T) {
	t.Parallel()
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

		assert.Equal(t, 1, vsl.CurrentPage)
		assert.Equal(t, 2, vsl.TotalCount)
	})

	t.Run("with list options", func(t *testing.T) {
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

	t.Run("with query parameter", func(t *testing.T) {
		vsl, err := client.VariableSets.List(ctx, orgTest.Name, &VariableSetListOptions{
			Query: vsTest2.Name,
		})
		require.NoError(t, err)
		assert.Len(t, vsl.Items, 1)
		assert.Equal(t, vsTest2.ID, vsl.Items[0].ID)
	})
}

func TestVariableSetsListForWorkspace(t *testing.T) {
	t.Parallel()
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

	t.Run("with query parameter", func(t *testing.T) {
		vsl, err := client.VariableSets.List(ctx, orgTest.Name, &VariableSetListOptions{
			Query: vsTest2.Name,
		})
		require.NoError(t, err)
		assert.Len(t, vsl.Items, 1)
		assert.Equal(t, vsTest2.ID, vsl.Items[0].ID)
	})
}

func TestVariableSetsListForProject(t *testing.T) {
	t.Parallel()
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

	t.Run("with query parameter", func(t *testing.T) {
		vsl, err := client.VariableSets.List(ctx, orgTest.Name, &VariableSetListOptions{
			Query: vsTest2.Name,
		})
		require.NoError(t, err)
		assert.Len(t, vsl.Items, 1)
		assert.Equal(t, vsTest2.ID, vsl.Items[0].ID)
	})
}

func TestVariableSetsCreate(t *testing.T) {
	t.Parallel()
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	t.Cleanup(orgTestCleanup)

	t.Run("with valid options", func(t *testing.T) {
		options := VariableSetCreateOptions{
			Name:        String("varset"),
			Description: String("a variable set"),
			Global:      Bool(false),
			Priority:    Bool(false),
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
			assert.Equal(t, *options.Priority, item.Priority)
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

	t.Run("when creating project-owned variable set", func(t *testing.T) {
		skipUnlessBeta(t)

		prjTest, prjTestCleanup := createProject(t, client, orgTest)
		t.Cleanup(prjTestCleanup)

		options := VariableSetCreateOptions{
			Name:        String("project-varset"),
			Description: String("a project variable set"),
			Global:      Bool(false),
			Parent: &Parent{
				Project: prjTest,
			},
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
			assert.Equal(t, options.Parent.Project.ID, item.Parent.Project.ID)
		}
	})
}

func TestVariableSetsRead(t *testing.T) {
	t.Parallel()
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

	t.Run("with parent relationship", func(t *testing.T) {
		skipUnlessBeta(t)

		vs, err := client.VariableSets.Read(ctx, vsTest.ID, nil)
		require.NoError(t, err)
		assert.Equal(t, vsTest, vs)
		assert.Equal(t, orgTest.Name, vs.Parent.Organization.Name)
	})
}

func TestVariableSetsUpdate(t *testing.T) {
	t.Parallel()
	client := testClient(t)
	ctx := context.Background()
	orgTest, orgTestCleanup := createOrganization(t, client)
	t.Cleanup(orgTestCleanup)
	vsTest, _ := createVariableSet(t, client, orgTest, VariableSetCreateOptions{
		Name:        String("OriginalName"),
		Description: String("Original Description"),
		Global:      Bool(false),
		Priority:    Bool(false),
	})
	t.Run("when updating a subset of values", func(t *testing.T) {
		options := VariableSetUpdateOptions{
			Name:        String("UpdatedName"),
			Description: String("Updated Description"),
			Global:      Bool(true),
			Priority:    Bool(true),
		}
		vsAfter, err := client.VariableSets.Update(ctx, vsTest.ID, &options)
		require.NoError(t, err)
		assert.Equal(t, *options.Name, vsAfter.Name)
		assert.Equal(t, *options.Description, vsAfter.Description)
		assert.Equal(t, *options.Global, vsAfter.Global)
		assert.Equal(t, *options.Priority, vsAfter.Priority)
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

func TestVariableSetsDelete(t *testing.T) {
	t.Parallel()
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

func TestVariableSetsApplyToAndRemoveFromStacks(t *testing.T) {
	skipUnlessBeta(t)

	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	t.Cleanup(orgTestCleanup)

	vsTest, vsTestCleanup := createVariableSet(t, client, orgTest, VariableSetCreateOptions{})
	t.Cleanup(vsTestCleanup)

	oauthClient, cleanup := createOAuthClient(t, client, orgTest, nil)
	t.Cleanup(cleanup)

	stackTest1, err := client.Stacks.Create(ctx, StackCreateOptions{
		Name: "test-stack-1",
		VCSRepo: &StackVCSRepoOptions{
			Identifier:   "hashicorp-guides/pet-nulls-stack",
			OAuthTokenID: oauthClient.OAuthTokens[0].ID,
		},
		Project: &Project{
			ID: orgTest.DefaultProject.ID,
		},
	})
	require.NoError(t, err)
	t.Cleanup(func() {
		if err := client.Stacks.Delete(ctx, stackTest1.ID); err != nil {
			t.Logf("Failed to cleanup stack %s: %v", stackTest1.ID, err)
		}
	})

	// Wait for stack to be ready by triggering configuration update
	_, err = client.Stacks.FetchLatestFromVcs(ctx, stackTest1.ID)
	require.NoError(t, err)

	stackTest2, err := client.Stacks.Create(ctx, StackCreateOptions{
		Name: "test-stack-2",
		VCSRepo: &StackVCSRepoOptions{
			Identifier:   "hashicorp-guides/pet-nulls-stack",
			OAuthTokenID: oauthClient.OAuthTokens[0].ID,
		},
		Project: &Project{
			ID: orgTest.DefaultProject.ID,
		},
	})
	require.NoError(t, err)
	t.Cleanup(func() {
		if err := client.Stacks.Delete(ctx, stackTest2.ID); err != nil {
			t.Logf("Failed to cleanup stack %s: %v", stackTest2.ID, err)
		}
	})

	// Wait for stack to be ready by triggering configuration update
	_, err = client.Stacks.FetchLatestFromVcs(ctx, stackTest2.ID)
	// Don't require this to succeed as it might not be needed

	t.Run("with first stack added", func(t *testing.T) {
		options := VariableSetApplyToStacksOptions{
			Stacks: []*Stack{{ID: stackTest1.ID}},
		}
		err = client.VariableSets.ApplyToStacks(ctx, vsTest.ID, &options)
		require.NoError(t, err)

		readOpts := &VariableSetReadOptions{Include: &[]VariableSetIncludeOpt{VariableSetStacks}}
		vsAfter, err := client.VariableSets.Read(ctx, vsTest.ID, readOpts)
		require.NoError(t, err)

		assert.Equal(t, 1, len(vsAfter.Stacks))
		assert.Equal(t, stackTest1.ID, vsAfter.Stacks[0].ID)
	})

	t.Run("with second stack added", func(t *testing.T) {
		options := VariableSetApplyToStacksOptions{
			Stacks: []*Stack{stackTest2},
		}

		err := client.VariableSets.ApplyToStacks(ctx, vsTest.ID, &options)
		require.NoError(t, err)

		readOpts := &VariableSetReadOptions{Include: &[]VariableSetIncludeOpt{VariableSetStacks}}
		vsAfter, err := client.VariableSets.Read(ctx, vsTest.ID, readOpts)
		require.NoError(t, err)

		assert.Equal(t, 2, len(vsAfter.Stacks))
		stackIDs := []string{vsAfter.Stacks[0].ID, vsAfter.Stacks[1].ID}

		assert.Contains(t, stackIDs, stackTest1.ID)
		assert.Contains(t, stackIDs, stackTest2.ID)
	})

	t.Run("with first stack removed", func(t *testing.T) {
		options := VariableSetRemoveFromStacksOptions{
			Stacks: []*Stack{stackTest1},
		}

		err := client.VariableSets.RemoveFromStacks(ctx, vsTest.ID, &options)
		require.NoError(t, err)

		readOpts := &VariableSetReadOptions{Include: &[]VariableSetIncludeOpt{VariableSetStacks}}
		vsAfter, err := client.VariableSets.Read(ctx, vsTest.ID, readOpts)
		require.NoError(t, err)

		assert.Equal(t, 1, len(vsAfter.Stacks))
		assert.Equal(t, stackTest2.ID, vsAfter.Stacks[0].ID)
	})

	t.Run("when variable set ID is invalid", func(t *testing.T) {
		applyOptions := VariableSetApplyToStacksOptions{
			Stacks: []*Stack{stackTest1},
		}
		err := client.VariableSets.ApplyToStacks(ctx, badIdentifier, &applyOptions)
		assert.EqualError(t, err, ErrInvalidVariableSetID.Error())

		removeOptions := VariableSetRemoveFromStacksOptions{
			Stacks: []*Stack{stackTest1},
		}
		err = client.VariableSets.RemoveFromStacks(ctx, badIdentifier, &removeOptions)
		assert.EqualError(t, err, ErrInvalidVariableSetID.Error())
	})

	t.Run("when stack ID is invalid", func(t *testing.T) {
		badStack := &Stack{
			ID: badIdentifier,
		}

		applyOptions := VariableSetApplyToStacksOptions{
			Stacks: []*Stack{badStack},
		}
		err := client.VariableSets.ApplyToStacks(ctx, vsTest.ID, &applyOptions)
		assert.EqualError(t, err, ErrRequiredStackID.Error())

		removeOptions := VariableSetRemoveFromStacksOptions{
			Stacks: []*Stack{badStack},
		}
		err = client.VariableSets.RemoveFromStacks(ctx, vsTest.ID, &removeOptions)
		assert.EqualError(t, err, ErrRequiredStackID.Error())
	})
}

func TestVariableSetsUpdateWorkspaces(t *testing.T) {
	t.Parallel()
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

func TestVariableSetsUpdateStacks(t *testing.T) {
	t.Parallel()
	skipUnlessBeta(t)

	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	t.Cleanup(orgTestCleanup)

	vsTest, vsTestCleanup := createVariableSet(t, client, orgTest, VariableSetCreateOptions{})
	t.Cleanup(vsTestCleanup)

	oauthClient, cleanup := createOAuthClient(t, client, orgTest, nil)
	t.Cleanup(cleanup)

	stackTest, err := client.Stacks.Create(ctx, StackCreateOptions{
		Name: "test-stack",
		VCSRepo: &StackVCSRepoOptions{
			Identifier:   "hashicorp-guides/pet-nulls-stack",
			OAuthTokenID: oauthClient.OAuthTokens[0].ID,
		},
		Project: &Project{
			ID: orgTest.DefaultProject.ID,
		},
	})
	require.NoError(t, err)
	t.Cleanup(func() {
		if err := client.Stacks.Delete(ctx, stackTest.ID); err != nil {
			t.Logf("Failed to cleanup stack %s: %v", stackTest.ID, err)
		}
	})

	// Wait for stack to be ready by triggering configuration update
	_, err = client.Stacks.FetchLatestFromVcs(ctx, stackTest.ID)
	require.NoError(t, err)

	t.Run("with valid stacks", func(t *testing.T) {
		options := VariableSetUpdateStacksOptions{
			Stacks: []*Stack{stackTest},
		}

		_, err := client.VariableSets.UpdateStacks(ctx, vsTest.ID, &options)
		require.NoError(t, err)

		readOpts := &VariableSetReadOptions{Include: &[]VariableSetIncludeOpt{VariableSetStacks}}
		vsAfter, err := client.VariableSets.Read(ctx, vsTest.ID, readOpts)
		require.NoError(t, err)

		assert.Equal(t, len(options.Stacks), len(vsAfter.Stacks))
		assert.Equal(t, options.Stacks[0].ID, vsAfter.Stacks[0].ID)

		options = VariableSetUpdateStacksOptions{
			Stacks: []*Stack{},
		}

		_, err = client.VariableSets.UpdateStacks(ctx, vsTest.ID, &options)
		require.NoError(t, err)

		readOpts = &VariableSetReadOptions{Include: &[]VariableSetIncludeOpt{VariableSetStacks}}
		vsAfter, err = client.VariableSets.Read(ctx, vsTest.ID, readOpts)
		require.NoError(t, err)

		assert.Equal(t, len(options.Stacks), len(vsAfter.Stacks))
	})
}
