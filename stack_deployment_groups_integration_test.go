package tfe

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStackDeploymentGroupsList(t *testing.T) {
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
		},
		Project: &Project{
			ID: orgTest.DefaultProject.ID,
		},
	})

	require.NoError(t, err)
	require.NotNil(t, stack)

	stackUpdated, err := client.Stacks.UpdateConfiguration(ctx, stack.ID)
	require.NoError(t, err)
	require.NotNil(t, stackUpdated)
	require.NotEmpty(t, stackUpdated.LatestStackConfiguration.ID)

	stackUpdated = pollStackDeployments(t, ctx, client, stackUpdated.ID)
	require.NotNil(t, stackUpdated.LatestStackConfiguration)

	t.Run("List with valid stack configuration ID", func(t *testing.T) {
		sdgl, err := client.StackDeploymentGroups.List(ctx, stackUpdated.LatestStackConfiguration.ID, nil)
		require.NoError(t, err)
		require.NotNil(t, sdgl)
		for _, item := range sdgl.Items {
			assert.NotNil(t, item.ID)
			assert.NotEmpty(t, item.Name)
			assert.NotEmpty(t, item.Status)
			assert.NotNil(t, item.CreatedAt)
			assert.NotNil(t, item.UpdatedAt)
		}
		require.Len(t, sdgl.Items, 2)
	})

	t.Run("List with invalid stack configuration ID", func(t *testing.T) {
		_, err := client.StackDeploymentGroups.List(ctx, "", nil)
		require.Error(t, err)
	})

	t.Run("List with pagination", func(t *testing.T) {
		options := &StackDeploymentGroupListOptions{
			ListOptions: ListOptions{
				PageNumber: 1,
				PageSize:   1,
			},
		}
		sdgl, err := client.StackDeploymentGroups.List(ctx, stackUpdated.LatestStackConfiguration.ID, options)
		require.NoError(t, err)
		require.NotNil(t, sdgl)
		require.Len(t, sdgl.Items, 1)
	})

	t.Run("List with filtering by group name", func(t *testing.T) {
		noOptionSdg, err := client.StackDeploymentGroups.List(ctx, stackUpdated.LatestStackConfiguration.ID, nil)
		require.NoError(t, err)
		require.NotNil(t, noOptionSdg)
		require.GreaterOrEqual(t, len(noOptionSdg.Items), 1)
		options := &StackDeploymentGroupListOptions{
			GroupName: noOptionSdg.Items[0].Name,
		}
		sdgl, err := client.StackDeploymentGroups.List(ctx, stackUpdated.LatestStackConfiguration.ID, options)
		require.NoError(t, err)
		require.NotNil(t, sdgl)
		require.Equal(t, 1, len(sdgl.Items))
		require.Equal(t, noOptionSdg.Items[0].Name, sdgl.Items[0].Name)
	})
}

func TestStackDeploymentGroupsRead(t *testing.T) {
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
		},
		Project: &Project{
			ID: orgTest.DefaultProject.ID,
		},
	})

	require.NoError(t, err)
	require.NotNil(t, stack)

	stackUpdated, err := client.Stacks.UpdateConfiguration(ctx, stack.ID)
	require.NoError(t, err)
	require.NotNil(t, stackUpdated)

	stackUpdated = pollStackDeployments(t, ctx, client, stackUpdated.ID)
	require.NotNil(t, stackUpdated.LatestStackConfiguration)

	sdgl, err := client.StackDeploymentGroups.List(ctx, stackUpdated.LatestStackConfiguration.ID, nil)
	require.NoError(t, err)
	require.NotNil(t, sdgl)
	require.Len(t, sdgl.Items, 2)

	t.Run("Read with valid ID", func(t *testing.T) {
		sdgRead, err := client.StackDeploymentGroups.Read(ctx, sdgl.Items[0].ID)
		require.NoError(t, err)
		assert.Equal(t, sdgl.Items[0].ID, sdgRead.ID)
		assert.Equal(t, sdgl.Items[0].Name, sdgRead.Name)
		assert.Equal(t, sdgl.Items[0].Status, sdgRead.Status)
	})

	t.Run("Read with invalid ID", func(t *testing.T) {
		_, err := client.StackDeploymentGroups.Read(ctx, "")
		require.Error(t, err)
	})
}

func TestStackDeploymentGroupsApproveAllPlans(t *testing.T) {
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

	stackUpdated, err := client.Stacks.UpdateConfiguration(ctx, stack.ID)
	require.NoError(t, err)
	require.NotNil(t, stackUpdated)

	stack = pollStackDeployments(t, ctx, client, stackUpdated.ID)
	require.NotNil(t, stack.LatestStackConfiguration)

	// Get the deployment group ID from the stack configuration
	deploymentGroups, err := client.StackDeploymentGroups.List(ctx, stack.LatestStackConfiguration.ID, nil)
	require.NoError(t, err)
	require.NotNil(t, deploymentGroups)
	require.NotEmpty(t, deploymentGroups.Items)

	deploymentGroupID := deploymentGroups.Items[0].ID

	t.Run("Approving all plans", func(t *testing.T) {
		err := client.StackDeploymentGroups.ApproveAllPlans(ctx, deploymentGroupID)
		require.NoError(t, err)
	})
}
