// Copyright IBM Corp. 2018, 2025
// SPDX-License-Identifier: MPL-2.0

package tfe

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStackCreateAndList(t *testing.T) {
	t.Parallel()
	skipUnlessBeta(t)

	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	t.Cleanup(orgTestCleanup)

	project2, err := client.Projects.Create(ctx, orgTest.Name, ProjectCreateOptions{
		Name: "test-project-2",
	})
	require.NoError(t, err)

	oauthClient, cleanup := createOAuthClient(t, client, orgTest, nil)
	t.Cleanup(cleanup)

	stack1, err := client.Stacks.Create(ctx, StackCreateOptions{
		Name: "aa-test-stack",
		VCSRepo: &StackVCSRepoOptions{
			Identifier:   "hashicorp-guides/pet-nulls-stack",
			OAuthTokenID: oauthClient.OAuthTokens[0].ID,
		},
		Project: &Project{
			ID: orgTest.DefaultProject.ID,
		},
		Migration:          Bool(true),
		SpeculativeEnabled: Bool(true),
	})

	require.NoError(t, err)
	require.NotNil(t, stack1)

	stack2, err := client.Stacks.Create(ctx, StackCreateOptions{
		Name: "zz-test-stack",
		VCSRepo: &StackVCSRepoOptions{
			Identifier:   "hashicorp-guides/pet-nulls-stack",
			OAuthTokenID: oauthClient.OAuthTokens[0].ID,
		},
		Project: &Project{
			ID: project2.ID,
		},
	})

	require.NoError(t, err)
	require.NotNil(t, stack2)

	t.Run("List without options", func(t *testing.T) {
		t.Parallel()

		stackList, err := client.Stacks.List(ctx, orgTest.Name, nil)
		require.NoError(t, err)

		assert.Len(t, stackList.Items, 2)
		assert.Equal(t, stack1.CreationSource, "migration-api")
		assert.Equal(t, stack2.CreationSource, "api")
	})

	t.Run("List with project filter", func(t *testing.T) {
		t.Parallel()

		stackList, err := client.Stacks.List(ctx, orgTest.Name, &StackListOptions{
			ProjectID: project2.ID,
		})
		require.NoError(t, err)

		assert.Len(t, stackList.Items, 1)
		assert.Equal(t, stack2.ID, stackList.Items[0].ID)
	})

	t.Run("List with name filter", func(t *testing.T) {
		t.Parallel()

		stackList, err := client.Stacks.List(ctx, orgTest.Name, &StackListOptions{
			SearchByName: "zz",
		})
		require.NoError(t, err)

		assert.Len(t, stackList.Items, 1)
		assert.Equal(t, stack2.ID, stackList.Items[0].ID)
	})

	t.Run("List with sort options", func(t *testing.T) {
		t.Parallel()

		// By name ASC
		stackList, err := client.Stacks.List(ctx, orgTest.Name, &StackListOptions{
			Sort: StackSortByName,
		})
		require.NoError(t, err)

		assert.Len(t, stackList.Items, 2)
		assert.Equal(t, stack1.ID, stackList.Items[0].ID)

		// By name DESC
		stackList, err = client.Stacks.List(ctx, orgTest.Name, &StackListOptions{
			Sort: StackSortByNameDesc,
		})
		require.NoError(t, err)

		assert.Len(t, stackList.Items, 2)
		assert.Equal(t, stack2.ID, stackList.Items[0].ID)
	})

	t.Run("List with pagination", func(t *testing.T) {
		t.Parallel()

		stackList, err := client.Stacks.List(ctx, orgTest.Name, &StackListOptions{
			ListOptions: ListOptions{
				PageNumber: 1,
				PageSize:   1,
			},
		})
		require.NoError(t, err)

		assert.Len(t, stackList.Items, 1)
		assert.Equal(t, 2, stackList.TotalPages)
		assert.Equal(t, 2, stackList.TotalCount)
	})
}

func TestStackReadUpdateDelete(t *testing.T) {
	t.Parallel()
	skipUnlessBeta(t)

	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	t.Cleanup(orgTestCleanup)

	oauthClient, cleanup := createOAuthClient(t, client, orgTest, nil)
	t.Cleanup(cleanup)

	initialPool, err := client.AgentPools.Create(ctx, orgTest.Name, AgentPoolCreateOptions{
		Name: String("initial-test-pool"),
	})
	require.NoError(t, err)

	stack, err := client.Stacks.Create(ctx, StackCreateOptions{
		Name: "test-stack",
		VCSRepo: &StackVCSRepoOptions{
			Identifier:   "hashicorp-guides/pet-nulls-stack",
			OAuthTokenID: oauthClient.OAuthTokens[0].ID,
			Branch:       "main",
		},
		Project: &Project{
			ID: orgTest.DefaultProject.ID,
		},
		AgentPool: initialPool,
	})

	require.NoError(t, err)
	require.NotNil(t, stack)
	require.NotEmpty(t, stack.VCSRepo.Identifier)
	require.NotEmpty(t, stack.VCSRepo.OAuthTokenID)
	require.NotEmpty(t, stack.VCSRepo.Branch)

	stackRead, err := client.Stacks.Read(ctx, stack.ID)
	require.NoError(t, err)
	require.Equal(t, stack.VCSRepo.Identifier, stackRead.VCSRepo.Identifier)
	require.Equal(t, stack.VCSRepo.OAuthTokenID, stackRead.VCSRepo.OAuthTokenID)
	require.Equal(t, stack.VCSRepo.Branch, stackRead.VCSRepo.Branch)
	require.Equal(t, stack.AgentPool.ID, stackRead.AgentPool.ID)
	assert.Equal(t, stack, stackRead)

	updatedPool, err := client.AgentPools.Create(ctx, orgTest.Name, AgentPoolCreateOptions{
		Name: String("updated-test-pool"),
	})
	require.NoError(t, err)

	stackUpdated, err := client.Stacks.Update(ctx, stack.ID, StackUpdateOptions{
		Description: String("updated description"),
		VCSRepo: &StackVCSRepoOptions{
			Identifier:   "hashicorp-guides/pet-nulls-stack",
			OAuthTokenID: oauthClient.OAuthTokens[0].ID,
			Branch:       "main",
		},
		AgentPool: updatedPool,
	})

	require.NoError(t, err)
	require.Equal(t, "updated description", stackUpdated.Description)
	require.Equal(t, updatedPool.ID, stackUpdated.AgentPool.ID)

	stackUpdatedConfig, err := client.Stacks.FetchLatestFromVcs(ctx, stack.ID)
	require.NoError(t, err)
	require.Equal(t, stack.Name, stackUpdatedConfig.Name)

	err = client.Stacks.Delete(ctx, stack.ID)
	require.NoError(t, err)

	stackReadAfterDelete, err := client.Stacks.Read(ctx, stack.ID)
	require.ErrorIs(t, err, ErrResourceNotFound)
	require.Nil(t, stackReadAfterDelete)
}

func TestStackRemoveVCSBacking(t *testing.T) {
	t.Parallel()
	skipUnlessBeta(t)

	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	t.Cleanup(orgTestCleanup)

	oauthClient, cleanup := createOAuthClient(t, client, orgTest, nil)
	t.Cleanup(cleanup)

	stack, err := client.Stacks.Create(ctx, StackCreateOptions{
		Name: "test-stack",
		VCSRepo: &StackVCSRepoOptions{
			Identifier:   "hashicorp-guides/pet-nulls-stack",
			OAuthTokenID: oauthClient.OAuthTokens[0].ID,
			Branch:       "main",
		},
		Project: &Project{
			ID: orgTest.DefaultProject.ID,
		},
	})

	require.NoError(t, err)
	require.NotNil(t, stack)
	require.NotEmpty(t, stack.VCSRepo.Identifier)
	require.NotEmpty(t, stack.VCSRepo.OAuthTokenID)
	require.NotEmpty(t, stack.VCSRepo.Branch)

	stackRead, err := client.Stacks.Read(ctx, stack.ID)
	require.NoError(t, err)
	require.Equal(t, stack.VCSRepo.Identifier, stackRead.VCSRepo.Identifier)
	require.Equal(t, stack.VCSRepo.OAuthTokenID, stackRead.VCSRepo.OAuthTokenID)
	require.Equal(t, stack.VCSRepo.Branch, stackRead.VCSRepo.Branch)

	assert.Equal(t, stack, stackRead)

	stackUpdated, err := client.Stacks.Update(ctx, stack.ID, StackUpdateOptions{
		VCSRepo: nil,
	})

	require.NoError(t, err)
	require.Nil(t, stackUpdated.VCSRepo)
}

func TestStackReadUpdateForceDelete(t *testing.T) {
	t.Parallel()
	skipUnlessBeta(t)

	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	t.Cleanup(orgTestCleanup)

	oauthClient, cleanup := createOAuthClient(t, client, orgTest, nil)
	t.Cleanup(cleanup)

	stack, err := client.Stacks.Create(ctx, StackCreateOptions{
		Name: "test-stack",
		VCSRepo: &StackVCSRepoOptions{
			Identifier:   "hashicorp-guides/pet-nulls-stack",
			OAuthTokenID: oauthClient.OAuthTokens[0].ID,
			Branch:       "main",
		},
		Project: &Project{
			ID: orgTest.DefaultProject.ID,
		},
	})

	require.NoError(t, err)
	require.NotNil(t, stack)
	require.NotEmpty(t, stack.VCSRepo.Identifier)
	require.NotEmpty(t, stack.VCSRepo.OAuthTokenID)
	require.NotEmpty(t, stack.VCSRepo.Branch)

	stackRead, err := client.Stacks.Read(ctx, stack.ID)
	require.NoError(t, err)
	require.Equal(t, stack.VCSRepo.Identifier, stackRead.VCSRepo.Identifier)
	require.Equal(t, stack.VCSRepo.OAuthTokenID, stackRead.VCSRepo.OAuthTokenID)
	require.Equal(t, stack.VCSRepo.Branch, stackRead.VCSRepo.Branch)

	assert.Equal(t, stack, stackRead)

	stackUpdated, err := client.Stacks.Update(ctx, stack.ID, StackUpdateOptions{
		Description: String("updated description"),
		VCSRepo: &StackVCSRepoOptions{
			Identifier:   "hashicorp-guides/pet-nulls-stack",
			OAuthTokenID: oauthClient.OAuthTokens[0].ID,
			Branch:       "main",
		},
	})

	require.NoError(t, err)
	require.Equal(t, "updated description", stackUpdated.Description)

	stackUpdatedConfig, err := client.Stacks.FetchLatestFromVcs(ctx, stack.ID)
	require.NoError(t, err)
	require.Equal(t, stack.Name, stackUpdatedConfig.Name)

	err = client.Stacks.ForceDelete(ctx, stack.ID)
	require.NoError(t, err)

	stackReadAfterDelete, err := client.Stacks.Read(ctx, stack.ID)
	require.ErrorIs(t, err, ErrResourceNotFound)
	require.Nil(t, stackReadAfterDelete)
}

func pollStackDeploymentGroups(t *testing.T, ctx context.Context, client *Client, stackID string) (stack *Stack) {
	t.Helper()

	// pollStackDeployments will poll the given stack until it has deployments or the deadline is reached.
	ctx, cancel := context.WithDeadline(ctx, time.Now().Add(5*time.Minute))
	defer cancel()

	deadline, _ := ctx.Deadline()
	t.Logf("Polling stack %q for deployment groups with deadline of %s", stackID, deadline)

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for finished := false; !finished; {
		t.Log("...")
		select {
		case <-ctx.Done():
			t.Fatalf("Stack %q had no deployment groups at deadline", stackID)
		case <-ticker.C:
			var err error
			stack, err = client.Stacks.Read(ctx, stackID)
			if err != nil {
				t.Fatalf("Failed to read stack %q: %s", stackID, err)
			}
			groups, err := client.StackDeploymentGroups.List(ctx, stack.LatestStackConfiguration.ID, nil)
			if err != nil {
				t.Fatalf("Failed to read deployment groups %q: %s", stackID, err)
			}

			t.Logf("Stack %q had %d deployment groups", stack.ID, groups.TotalCount)
			if groups.TotalCount > 0 {
				finished = true
			}
		}
	}

	return stack
}

func pollStackDeploymentGroupStatus(t *testing.T, ctx context.Context, client *Client, configurationID, status string) {
	// pollStackDeploymentGroupStatus will poll the given stack until its deployment groups
	// all match the given status, or the deadline is reached.
	ctx, cancel := context.WithDeadline(ctx, time.Now().Add(5*time.Minute))
	defer cancel()

	deadline, _ := ctx.Deadline()
	t.Logf("Polling configuration %q for deployments with deadline of %s", configurationID, deadline)

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for finished := false; !finished; {
		t.Log("...")
		select {
		case <-ctx.Done():
			t.Fatalf("Stack deployment groups for config %s did not have status %q at deadline", configurationID, status)
		case <-ticker.C:
			var err error
			summaries, err := client.StackDeploymentGroupSummaries.List(ctx, configurationID, nil)
			if err != nil {
				t.Fatalf("Failed to read stack deployment groups for config %s: %s", configurationID, err)
			}

			for _, group := range summaries.Items {
				t.Logf("Stack deployment group %s for config %s had status %q", group.ID, configurationID, group.Status)
				if group.Status == status {
					finished = true
				}
			}
		}
	}
}

func pollStackDeploymentRunStatus(t *testing.T, ctx context.Context, client *Client, deploymentRunID, status string) {
	// pollStackDeploymentRunStatus will poll the given stack until the targeted
	// deployment run matches the given status, or the deadline is reached.
	ctx, cancel := context.WithDeadline(ctx, time.Now().Add(5*time.Minute))
	defer cancel()

	deadline, _ := ctx.Deadline()
	t.Logf("Polling deployment run %q for status %s with deadline of %s", deploymentRunID, status, deadline)

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for finished := false; !finished; {
		t.Log("...")
		select {
		case <-ctx.Done():
			t.Fatalf("Stack deployment run %s did not have status %q at deadline", deploymentRunID, status)
		case <-ticker.C:
			var err error
			deploymentRun, err := client.StackDeploymentRuns.Read(ctx, deploymentRunID)
			if err != nil {
				t.Fatalf("Failed to read stack deployment run %s: %s", deploymentRunID, err)
			}

			t.Logf("Stack deployment run %s had status %q", deploymentRunID, deploymentRun.Status)
			if deploymentRun.Status.String() == status {
				finished = true
			}
		}
	}
}

func pollStackConfigurationStatus(t *testing.T, ctx context.Context, client *Client, stackConfigID, status string) (stackConfig *StackConfiguration) {
	// pollStackDeployments will poll the given stack until it has deployments or the deadline is reached.
	ctx, cancel := context.WithDeadline(ctx, time.Now().Add(5*time.Minute))
	defer cancel()

	deadline, _ := ctx.Deadline()
	t.Logf("Polling stack configuration %q for status %q with deadline of %s", stackConfigID, status, deadline)

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	var err error
	for finished := false; !finished; {
		t.Log("...")
		select {
		case <-ctx.Done():
			t.Fatalf("Stack configuration %q did not have status %q at deadline", stackConfigID, status)
		case <-ticker.C:
			stackConfig, err = client.StackConfigurations.Read(ctx, stackConfigID)
			if err != nil {
				t.Fatalf("Failed to read stack configuration %q: %s", stackConfigID, err)
			}

			t.Logf("Stack configuration %q had status %q", stackConfigID, stackConfig.Status)
			if stackConfig.Status.String() == status {
				finished = true
			}
		}
	}

	return
}

func pollNewStackConfiguration(t *testing.T, ctx context.Context, client *Client, stackID string) (stackConfig *StackConfiguration) {
	// pollNewStackConfiguration can be used after a new configuration is fetched
	// to ensure a non-nil configuration is returned.
	ctx, cancel := context.WithDeadline(ctx, time.Now().Add(5*time.Minute))
	defer cancel()

	deadline, _ := ctx.Deadline()
	t.Logf("Polling stack %s for new stack configuration with deadline of %s", stackID, deadline)

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for finished := false; !finished; {
		t.Log("...")
		select {
		case <-ctx.Done():
			t.Fatalf("Stack %q had no new configuration at deadline", stackID)
		case <-ticker.C:
			stackConfigs, err := client.StackConfigurations.List(ctx, stackID, nil)
			if err != nil {
				t.Fatalf("Failed to list stack configurations for stack %q: %s", stackID, err)
			}

			if len(stackConfigs.Items) > 0 {
				stackConfig = stackConfigs.Items[0]
				finished = true
			}
		}
	}

	return
}
