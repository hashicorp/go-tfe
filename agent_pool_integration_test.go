// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package tfe

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAgentPoolsList(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	t.Cleanup(orgTestCleanup)

	upgradeOrganizationSubscription(t, client, orgTest)

	agentPool, agentPoolCleanup := createAgentPool(t, client, orgTest)
	t.Cleanup(agentPoolCleanup)

	t.Run("without list options", func(t *testing.T) {
		pools, err := client.AgentPools.List(ctx, orgTest.Name, nil)
		require.NoError(t, err)
		assert.Contains(t, pools.Items, agentPool)

		assert.Equal(t, 1, pools.CurrentPage)
		assert.Equal(t, 1, pools.TotalCount)
	})

	t.Run("with Include option", func(t *testing.T) {
		_, wTestCleanup := createWorkspaceWithVCS(t, client, orgTest, WorkspaceCreateOptions{
			Name:          String("bar"),
			ExecutionMode: String("agent"),
			AgentPoolID:   String(agentPool.ID),
		})
		t.Cleanup(wTestCleanup)

		k, err := client.AgentPools.List(ctx, orgTest.Name, &AgentPoolListOptions{
			Include: []AgentPoolIncludeOpt{AgentPoolWorkspaces},
		})
		require.NoError(t, err)
		require.NotEmpty(t, k.Items)
		require.NotEmpty(t, k.Items[0].Workspaces)
		assert.NotNil(t, k.Items[0].Workspaces[0])
	})

	t.Run("with list options", func(t *testing.T) {
		// Request a page number which is out of range. The result should
		// be successful, but return no results if the paging options are
		// properly passed along.
		pools, err := client.AgentPools.List(ctx, orgTest.Name, &AgentPoolListOptions{
			ListOptions: ListOptions{
				PageNumber: 999,
				PageSize:   100,
			},
		})
		require.NoError(t, err)
		assert.Empty(t, pools.Items)
		assert.Equal(t, 999, pools.CurrentPage)
		assert.Equal(t, 1, pools.TotalCount)
	})

	t.Run("without a valid organization", func(t *testing.T) {
		pools, err := client.AgentPools.List(ctx, badIdentifier, nil)
		assert.Nil(t, pools)
		assert.EqualError(t, err, ErrInvalidOrg.Error())
	})

	t.Run("with query options", func(t *testing.T) {
		pools, err := client.AgentPools.List(ctx, orgTest.Name, &AgentPoolListOptions{
			Query: agentPool.Name,
		})
		require.NoError(t, err)
		assert.Equal(t, len(pools.Items), 1)

		pools, err = client.AgentPools.List(ctx, orgTest.Name, &AgentPoolListOptions{
			Query: agentPool.Name + "not_going_to_match",
		})
		require.NoError(t, err)
		assert.Empty(t, pools.Items)
	})

	t.Run("with allowed workspace name filter", func(t *testing.T) {
		ws1, ws1TestCleanup := createWorkspace(t, client, orgTest)
		t.Cleanup(ws1TestCleanup)

		ws2, ws2TestCleanup := createWorkspace(t, client, orgTest)
		t.Cleanup(ws2TestCleanup)

		organizationScoped := false
		ap, apCleanup := createAgentPoolWithOptions(t, client, orgTest, AgentPoolCreateOptions{
			Name:               String("a-pool"),
			OrganizationScoped: &organizationScoped,
			AllowedWorkspaces:  []*Workspace{ws1},
		})
		t.Cleanup(apCleanup)

		ap2, ap2Cleanup := createAgentPoolWithOptions(t, client, orgTest, AgentPoolCreateOptions{
			Name:               String("b-pool"),
			OrganizationScoped: &organizationScoped,
			AllowedWorkspaces:  []*Workspace{ws2},
		})
		t.Cleanup(ap2Cleanup)

		pools, err := client.AgentPools.List(ctx, orgTest.Name, &AgentPoolListOptions{
			AllowedWorkspacesName: ws1.Name,
		})
		require.NoError(t, err)
		assert.NotEmpty(t, pools.Items)
		assert.Contains(t, pools.Items, ap)
		assert.Contains(t, pools.Items, agentPool)
		assert.Equal(t, 2, pools.TotalCount)

		pools, err = client.AgentPools.List(ctx, orgTest.Name, &AgentPoolListOptions{
			AllowedWorkspacesName: ws2.Name,
		})
		require.NoError(t, err)
		assert.NotEmpty(t, pools.Items)
		assert.Contains(t, pools.Items, agentPool)
		assert.Contains(t, pools.Items, ap2)
		assert.Equal(t, 2, pools.TotalCount)
	})

	t.Run("with allowed projects name filter", func(t *testing.T) {
		proj1, proj1TestCleanup := createProject(t, client, orgTest)
		t.Cleanup(proj1TestCleanup)

		proj2, proj2TestCleanup := createProject(t, client, orgTest)
		t.Cleanup(proj2TestCleanup)

		organizationScoped := false
		ap, apCleanup := createAgentPoolWithOptions(t, client, orgTest, AgentPoolCreateOptions{
			Name:               String("a-pool"),
			OrganizationScoped: &organizationScoped,
			AllowedProjects:    []*Project{proj1},
		})
		t.Cleanup(apCleanup)

		ap2, ap2Cleanup := createAgentPoolWithOptions(t, client, orgTest, AgentPoolCreateOptions{
			Name:               String("b-pool"),
			OrganizationScoped: &organizationScoped,
			AllowedProjects:    []*Project{proj2},
		})
		t.Cleanup(ap2Cleanup)

		pools, err := client.AgentPools.List(ctx, orgTest.Name, &AgentPoolListOptions{
			AllowedProjectsName: proj1.Name,
		})
		require.NoError(t, err)
		assert.NotEmpty(t, pools.Items)
		assert.Contains(t, pools.Items, ap)
		assert.Contains(t, pools.Items, agentPool)
		assert.Equal(t, 2, pools.TotalCount)

		pools, err = client.AgentPools.List(ctx, orgTest.Name, &AgentPoolListOptions{
			AllowedProjectsName: proj2.Name,
		})
		require.NoError(t, err)
		assert.NotEmpty(t, pools.Items)
		assert.Contains(t, pools.Items, agentPool)
		assert.Contains(t, pools.Items, ap2)
		assert.Equal(t, 2, pools.TotalCount)
	})
}

func TestAgentPoolsCreate(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	t.Cleanup(orgTestCleanup)

	upgradeOrganizationSubscription(t, client, orgTest)

	t.Run("with valid options", func(t *testing.T) {
		options := AgentPoolCreateOptions{
			Name: String("cool-pool"),
		}

		pool, err := client.AgentPools.Create(ctx, orgTest.Name, options)
		require.NoError(t, err)

		// Get a refreshed view from the API.
		refreshed, err := client.AgentPools.Read(ctx, pool.ID)
		require.NoError(t, err)

		for _, item := range []*AgentPool{
			pool,
			refreshed,
		} {
			assert.NotEmpty(t, item.ID)
		}
	})

	t.Run("when options is missing name", func(t *testing.T) {
		k, err := client.AgentPools.Create(ctx, "foo", AgentPoolCreateOptions{})
		assert.Nil(t, k)
		assert.EqualError(t, err, ErrRequiredName.Error())
	})

	t.Run("with an invalid organization", func(t *testing.T) {
		pool, err := client.AgentPools.Create(ctx, badIdentifier, AgentPoolCreateOptions{
			Name: String("cool-pool"),
		})
		assert.Nil(t, pool)
		assert.EqualError(t, err, ErrInvalidOrg.Error())
	})

	t.Run("with allowed-workspaces options", func(t *testing.T) {
		workspaceTest, workspaceTestCleanup := createWorkspace(t, client, orgTest)
		t.Cleanup(workspaceTestCleanup)

		organizationScoped := false
		options := AgentPoolCreateOptions{
			Name:               String("a-pool"),
			OrganizationScoped: &organizationScoped,
			AllowedWorkspaces: []*Workspace{
				workspaceTest,
			},
		}

		pool, err := client.AgentPools.Create(ctx, orgTest.Name, options)
		require.NoError(t, err)

		assert.Equal(t, 1, len(pool.AllowedWorkspaces))
		assert.Equal(t, workspaceTest.ID, pool.AllowedWorkspaces[0].ID)

		// Get a refreshed view from the API.
		refreshed, err := client.AgentPools.Read(ctx, pool.ID)
		require.NoError(t, err)

		for _, item := range []*AgentPool{
			pool,
			refreshed,
		} {
			assert.NotEmpty(t, item.ID)
		}
	})

	t.Run("with allowed-projects options", func(t *testing.T) {
		projectTest, projectTestCleanup := createProject(t, client, orgTest)
		t.Cleanup(projectTestCleanup)

		organizationScoped := false
		options := AgentPoolCreateOptions{
			Name:               String("a-pool-2"),
			OrganizationScoped: &organizationScoped,
			AllowedProjects: []*Project{
				projectTest,
			},
		}

		pool, err := client.AgentPools.Create(ctx, orgTest.Name, options)
		require.NoError(t, err)

		assert.Equal(t, 1, len(pool.AllowedProjects))
		assert.Equal(t, projectTest.ID, pool.AllowedProjects[0].ID)

		// Get a refreshed view from the API.
		refreshed, err := client.AgentPools.Read(ctx, pool.ID)
		require.NoError(t, err)

		for _, item := range []*AgentPool{
			pool,
			refreshed,
		} {
			assert.NotEmpty(t, item.ID)
		}
	})

	t.Run("with excluded-workspaces options", func(t *testing.T) {
		workspaceTest, workspaceTestCleanup := createWorkspace(t, client, orgTest)
		t.Cleanup(workspaceTestCleanup)

		organizationScoped := false
		options := AgentPoolCreateOptions{
			Name:               String("a-pool-3"),
			OrganizationScoped: &organizationScoped,
			ExcludedWorkspaces: []*Workspace{
				workspaceTest,
			},
		}

		pool, err := client.AgentPools.Create(ctx, orgTest.Name, options)
		require.NoError(t, err)

		assert.Equal(t, 1, len(pool.ExcludedWorkspaces))
		assert.Equal(t, workspaceTest.ID, pool.ExcludedWorkspaces[0].ID)

		// Get a refreshed view from the API.
		refreshed, err := client.AgentPools.Read(ctx, pool.ID)
		require.NoError(t, err)

		for _, item := range []*AgentPool{
			pool,
			refreshed,
		} {
			assert.NotEmpty(t, item.ID)
		}
	})
}

func TestAgentPoolsRead(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	t.Cleanup(orgTestCleanup)

	upgradeOrganizationSubscription(t, client, orgTest)

	pool, poolCleanup := createAgentPool(t, client, orgTest)
	t.Cleanup(poolCleanup)

	t.Run("when the agent pool exists", func(t *testing.T) {
		k, err := client.AgentPools.Read(ctx, pool.ID)
		require.NoError(t, err)
		assert.Equal(t, pool, k)
	})

	t.Run("when the agent pool does not exist", func(t *testing.T) {
		k, err := client.AgentPools.Read(ctx, "nonexisting")
		assert.Nil(t, k)
		assert.Equal(t, err, ErrResourceNotFound)
	})

	t.Run("without a valid agent pool ID", func(t *testing.T) {
		k, err := client.AgentPools.Read(ctx, badIdentifier)
		assert.Nil(t, k)
		assert.EqualError(t, err, ErrInvalidAgentPoolID.Error())
	})

	t.Run("with Include option", func(t *testing.T) {
		_, wTestCleanup := createWorkspaceWithVCS(t, client, orgTest, WorkspaceCreateOptions{
			Name:          String("foo"),
			ExecutionMode: String("agent"),
			AgentPoolID:   String(pool.ID),
		})
		t.Cleanup(wTestCleanup)

		k, err := client.AgentPools.ReadWithOptions(ctx, pool.ID, &AgentPoolReadOptions{
			Include: []AgentPoolIncludeOpt{AgentPoolWorkspaces},
		})
		require.NoError(t, err)
		assert.NotEmpty(t, k.Workspaces[0])
	})
}

func TestAgentPoolsReadCreatedAt(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	defer orgTestCleanup()

	upgradeOrganizationSubscription(t, client, orgTest)

	pool, poolCleanup := createAgentPool(t, client, orgTest)
	defer poolCleanup()

	k, err := client.AgentPools.Read(ctx, pool.ID)
	assert.NotEmpty(t, k.CreatedAt)
	require.NoError(t, err)
}

func TestAgentPoolsUpdate(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	t.Cleanup(orgTestCleanup)

	upgradeOrganizationSubscription(t, client, orgTest)

	t.Run("with valid options", func(t *testing.T) {
		kBefore, kTestCleanup := createAgentPool(t, client, orgTest)
		t.Cleanup(kTestCleanup)

		kAfter, err := client.AgentPools.Update(ctx, kBefore.ID, AgentPoolUpdateOptions{
			Name: String(randomString(t)),
		})
		require.NoError(t, err)

		assert.Equal(t, kBefore.ID, kAfter.ID)
		assert.NotEqual(t, kBefore.Name, kAfter.Name)
	})

	t.Run("when updating only the name", func(t *testing.T) {
		workspaceTest, workspaceTestCleanup := createWorkspace(t, client, orgTest)
		t.Cleanup(workspaceTestCleanup)

		projectTest, projectTestCleanup := createProject(t, client, orgTest)
		t.Cleanup(projectTestCleanup)

		excludedWorkspaceTest, excludedWorkspaceTestCleanup := createWorkspace(t, client, orgTest)
		t.Cleanup(excludedWorkspaceTestCleanup)

		organizationScoped := false
		options := AgentPoolCreateOptions{
			Name:               String("a-pool"),
			OrganizationScoped: &organizationScoped,
			AllowedWorkspaces: []*Workspace{
				workspaceTest,
			},
			AllowedProjects: []*Project{
				projectTest,
			},
			ExcludedWorkspaces: []*Workspace{
				excludedWorkspaceTest,
			},
		}
		kBefore, err := client.AgentPools.Create(ctx, orgTest.Name, options)
		require.NoError(t, err)

		kAfter, err := client.AgentPools.Update(ctx, kBefore.ID, AgentPoolUpdateOptions{
			Name: String("updated-key-name"),
		})
		require.NoError(t, err)

		assert.Equal(t, kBefore.ID, kAfter.ID)
		assert.Equal(t, "updated-key-name", kAfter.Name)
		assert.Equal(t, 1, len(kAfter.AllowedWorkspaces))
		assert.Equal(t, workspaceTest.ID, kAfter.AllowedWorkspaces[0].ID)
		assert.Equal(t, 1, len(kAfter.AllowedProjects))
		assert.Equal(t, projectTest.ID, kAfter.AllowedProjects[0].ID)
		assert.Equal(t, 1, len(kAfter.ExcludedWorkspaces))
		assert.Equal(t, excludedWorkspaceTest.ID, kAfter.ExcludedWorkspaces[0].ID)
	})

	t.Run("without a valid agent pool ID", func(t *testing.T) {
		w, err := client.AgentPools.Update(ctx, badIdentifier, AgentPoolUpdateOptions{})
		assert.Nil(t, w)
		assert.EqualError(t, err, ErrInvalidAgentPoolID.Error())
	})

	t.Run("when updating organization scope", func(t *testing.T) {
		kBefore, kTestCleanup := createAgentPool(t, client, orgTest)
		t.Cleanup(kTestCleanup)

		organizationScoped := false
		kAfter, err := client.AgentPools.Update(ctx, kBefore.ID, AgentPoolUpdateOptions{
			Name:               String(kBefore.Name),
			OrganizationScoped: &organizationScoped,
		})
		require.NoError(t, err)

		assert.NotEqual(t, kBefore.OrganizationScoped, kAfter.OrganizationScoped)
		assert.Equal(t, organizationScoped, kAfter.OrganizationScoped)
	})

	t.Run("when updating allowed-workspaces", func(t *testing.T) {
		kBefore, kTestCleanup := createAgentPool(t, client, orgTest)
		t.Cleanup(kTestCleanup)

		workspaceTest, workspaceTestCleanup := createWorkspace(t, client, orgTest)
		t.Cleanup(workspaceTestCleanup)

		kAfter, err := client.AgentPools.Update(ctx, kBefore.ID, AgentPoolUpdateOptions{
			AllowedWorkspaces: []*Workspace{
				workspaceTest,
			},
		})
		require.NoError(t, err)

		assert.Equal(t, kBefore.Name, kAfter.Name)
		assert.NotEqual(t, kBefore.AllowedWorkspaces, kAfter.AllowedWorkspaces)
		assert.Equal(t, 1, len(kAfter.AllowedWorkspaces))
		assert.Equal(t, workspaceTest.ID, kAfter.AllowedWorkspaces[0].ID)
	})

	t.Run("when updating allowed-projects", func(t *testing.T) {
		kBefore, kTestCleanup := createAgentPool(t, client, orgTest)
		t.Cleanup(kTestCleanup)

		projectTest, projectTestCleanup := createProject(t, client, orgTest)
		t.Cleanup(projectTestCleanup)

		kAfter, err := client.AgentPools.Update(ctx, kBefore.ID, AgentPoolUpdateOptions{
			AllowedProjects: []*Project{
				projectTest,
			},
		})
		require.NoError(t, err)

		assert.Equal(t, kBefore.Name, kAfter.Name)
		assert.NotEqual(t, kBefore.AllowedProjects, kAfter.AllowedProjects)
		assert.Equal(t, 1, len(kAfter.AllowedProjects))
		assert.Equal(t, projectTest.ID, kAfter.AllowedProjects[0].ID)
	})

	t.Run("when updating excluded-workspaces", func(t *testing.T) {
		kBefore, kTestCleanup := createAgentPool(t, client, orgTest)
		t.Cleanup(kTestCleanup)

		workspaceTest, workspaceTestCleanup := createWorkspace(t, client, orgTest)
		t.Cleanup(workspaceTestCleanup)

		kAfter, err := client.AgentPools.Update(ctx, kBefore.ID, AgentPoolUpdateOptions{
			ExcludedWorkspaces: []*Workspace{
				workspaceTest,
			},
		})
		require.NoError(t, err)

		assert.Equal(t, kBefore.Name, kAfter.Name)
		assert.NotEqual(t, kBefore.ExcludedWorkspaces, kAfter.ExcludedWorkspaces)
		assert.Equal(t, 1, len(kAfter.ExcludedWorkspaces))
		assert.Equal(t, workspaceTest.ID, kAfter.ExcludedWorkspaces[0].ID)
	})
}

func TestAgentPoolsUpdateAllowedWorkspaces(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	t.Cleanup(orgTestCleanup)

	upgradeOrganizationSubscription(t, client, orgTest)

	t.Run("when updating allowed-workspaces", func(t *testing.T) {
		kBefore, kTestCleanup := createAgentPool(t, client, orgTest)
		t.Cleanup(kTestCleanup)

		workspaceTest, workspaceTestCleanup := createWorkspace(t, client, orgTest)
		t.Cleanup(workspaceTestCleanup)

		kAfter, err := client.AgentPools.UpdateAllowedWorkspaces(ctx, kBefore.ID, AgentPoolAllowedWorkspacesUpdateOptions{
			AllowedWorkspaces: []*Workspace{
				workspaceTest,
			},
		})
		require.NoError(t, err)

		assert.Equal(t, kBefore.Name, kAfter.Name)
		assert.NotEqual(t, kBefore.AllowedWorkspaces, kAfter.AllowedWorkspaces)
		assert.Equal(t, 1, len(kAfter.AllowedWorkspaces))
		assert.Equal(t, workspaceTest.ID, kAfter.AllowedWorkspaces[0].ID)
	})

	t.Run("when removing all the allowed-workspaces", func(t *testing.T) {
		workspaceTest, workspaceTestCleanup := createWorkspace(t, client, orgTest)
		t.Cleanup(workspaceTestCleanup)

		organizationScoped := false
		options := AgentPoolCreateOptions{
			Name:               String("a-pool"),
			OrganizationScoped: &organizationScoped,
			AllowedWorkspaces: []*Workspace{
				workspaceTest,
			},
		}

		kBefore, kTestCleanup := createAgentPoolWithOptions(t, client, orgTest, options)
		t.Cleanup(kTestCleanup)

		kAfter, err := client.AgentPools.UpdateAllowedWorkspaces(ctx, kBefore.ID, AgentPoolAllowedWorkspacesUpdateOptions{
			AllowedWorkspaces: []*Workspace{},
		})
		require.NoError(t, err)

		assert.Equal(t, kBefore.ID, kAfter.ID)
		assert.Equal(t, "a-pool", kAfter.Name)
		assert.Empty(t, kAfter.AllowedWorkspaces)
	})
}

func TestAgentPoolsUpdateAllowedProjects(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	t.Cleanup(orgTestCleanup)

	upgradeOrganizationSubscription(t, client, orgTest)

	t.Run("when updating allowed-projects", func(t *testing.T) {
		kBefore, kTestCleanup := createAgentPool(t, client, orgTest)
		t.Cleanup(kTestCleanup)

		projectTest, projectTestCleanup := createProject(t, client, orgTest)
		t.Cleanup(projectTestCleanup)

		kAfter, err := client.AgentPools.UpdateAllowedProjects(ctx, kBefore.ID, AgentPoolAllowedProjectsUpdateOptions{
			AllowedProjects: []*Project{
				projectTest,
			},
		})
		require.NoError(t, err)

		assert.Equal(t, kBefore.Name, kAfter.Name)
		assert.NotEqual(t, kBefore.AllowedProjects, kAfter.AllowedProjects)
		assert.Equal(t, 1, len(kAfter.AllowedProjects))
		assert.Equal(t, projectTest.ID, kAfter.AllowedProjects[0].ID)
	})

	t.Run("when removing all the allowed-projects", func(t *testing.T) {
		projectTest, projectTestCleanup := createProject(t, client, orgTest)
		t.Cleanup(projectTestCleanup)

		organizationScoped := false
		options := AgentPoolCreateOptions{
			Name:               String("a-pool"),
			OrganizationScoped: &organizationScoped,
			AllowedProjects: []*Project{
				projectTest,
			},
		}

		kBefore, kTestCleanup := createAgentPoolWithOptions(t, client, orgTest, options)
		t.Cleanup(kTestCleanup)

		kAfter, err := client.AgentPools.UpdateAllowedProjects(ctx, kBefore.ID, AgentPoolAllowedProjectsUpdateOptions{
			AllowedProjects: []*Project{},
		})
		require.NoError(t, err)

		assert.Equal(t, kBefore.ID, kAfter.ID)
		assert.Equal(t, "a-pool", kAfter.Name)
		assert.Empty(t, kAfter.AllowedProjects)
	})
}

func TestAgentPoolsUpdateExcludedWorkspaces(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	t.Cleanup(orgTestCleanup)

	upgradeOrganizationSubscription(t, client, orgTest)

	t.Run("when updating excluded-workspaces", func(t *testing.T) {
		kBefore, kTestCleanup := createAgentPool(t, client, orgTest)
		t.Cleanup(kTestCleanup)

		workspaceTest, workspaceTestCleanup := createWorkspace(t, client, orgTest)
		t.Cleanup(workspaceTestCleanup)

		kAfter, err := client.AgentPools.UpdateExcludedWorkspaces(ctx, kBefore.ID, AgentPoolExcludedWorkspacesUpdateOptions{
			ExcludedWorkspaces: []*Workspace{
				workspaceTest,
			},
		})
		require.NoError(t, err)

		assert.Equal(t, kBefore.Name, kAfter.Name)
		assert.NotEqual(t, kBefore.ExcludedWorkspaces, kAfter.ExcludedWorkspaces)
		assert.Equal(t, 1, len(kAfter.ExcludedWorkspaces))
		assert.Equal(t, workspaceTest.ID, kAfter.ExcludedWorkspaces[0].ID)
	})

	t.Run("when removing all the excluded-workspaces", func(t *testing.T) {
		workspaceTest, workspaceTestCleanup := createWorkspace(t, client, orgTest)
		t.Cleanup(workspaceTestCleanup)

		organizationScoped := false
		options := AgentPoolCreateOptions{
			Name:               String("a-pool"),
			OrganizationScoped: &organizationScoped,
			ExcludedWorkspaces: []*Workspace{
				workspaceTest,
			},
		}

		kBefore, kTestCleanup := createAgentPoolWithOptions(t, client, orgTest, options)
		t.Cleanup(kTestCleanup)

		kAfter, err := client.AgentPools.UpdateExcludedWorkspaces(ctx, kBefore.ID, AgentPoolExcludedWorkspacesUpdateOptions{
			ExcludedWorkspaces: []*Workspace{},
		})
		require.NoError(t, err)

		assert.Equal(t, kBefore.ID, kAfter.ID)
		assert.Equal(t, "a-pool", kAfter.Name)
		assert.Empty(t, kAfter.ExcludedWorkspaces)
	})
}

func TestAgentPoolsDelete(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	t.Cleanup(orgTestCleanup)

	upgradeOrganizationSubscription(t, client, orgTest)

	agentPool, _ := createAgentPool(t, client, orgTest)

	t.Run("with valid options", func(t *testing.T) {
		err := client.AgentPools.Delete(ctx, agentPool.ID)
		require.NoError(t, err)

		// Try loading the agent pool - it should fail.
		_, err = client.AgentPools.Read(ctx, agentPool.ID)
		assert.Equal(t, err, ErrResourceNotFound)
	})

	t.Run("when the agent pool does not exist", func(t *testing.T) {
		err := client.AgentPools.Delete(ctx, agentPool.ID)
		assert.Equal(t, err, ErrResourceNotFound)
	})

	t.Run("when the agent pool ID is invalid", func(t *testing.T) {
		err := client.AgentPools.Delete(ctx, badIdentifier)
		assert.EqualError(t, err, ErrInvalidAgentPoolID.Error())
	})
}
