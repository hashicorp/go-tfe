package tfe

import (
	"context"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStackStateListReadDescription(t *testing.T) {
	skipUnlessBeta(t)

	client := testClient(t)
	ctx := context.Background()
	orgTest, orgTestCleanup := createOrganization(t, client)
	t.Cleanup(orgTestCleanup)
	oauthClient, cleanup := createOAuthClient(t, client, orgTest, nil)
	t.Cleanup(cleanup)
	stack, err := client.Stacks.Create(ctx, StackCreateOptions{
		Name: "aa-test-stack",
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
	stack2, err := client.Stacks.Create(ctx, StackCreateOptions{
		Name: "bb-test-stack",
		VCSRepo: &StackVCSRepoOptions{
			Identifier:   "hashicorp-guides/pet-nulls-stack",
			OAuthTokenID: oauthClient.OAuthTokens[0].ID,
		},
		Project: &Project{
			ID: orgTest.DefaultProject.ID,
		},
	})
	require.NoError(t, err)
	require.NotNil(t, stack2)

	stackUpdated, err := client.Stacks.FetchLatestFromVcs(ctx, stack.ID)
	require.NoError(t, err)
	require.NotNil(t, stackUpdated)

	stackUpdated = pollStackDeploymentGroups(t, ctx, client, stackUpdated.ID)
	require.NotNil(t, stackUpdated.LatestStackConfiguration)

	// Get the deployment group ID from the stack configuration
	deploymentGroups, err := client.StackDeploymentGroups.List(ctx, stackUpdated.LatestStackConfiguration.ID, nil)
	require.NoError(t, err)
	require.NotNil(t, deploymentGroups)
	require.NotEmpty(t, deploymentGroups.Items)

	for _, dg := range deploymentGroups.Items {
		err = client.StackDeploymentGroups.ApproveAllPlans(ctx, dg.ID)
		require.NoError(t, err)
	}

	pollStackDeploymentGroupStatus(t, ctx, client, stackUpdated.LatestStackConfiguration.ID, "succeeded")

	t.Run("List with valid ID", func(t *testing.T) {
		states, err := client.StackStates.List(ctx, stackUpdated.ID, nil)
		require.NoError(t, err)
		require.NotNil(t, states)
		require.NotEmpty(t, states.Items)
	})

	t.Run("List with invalid ID", func(t *testing.T) {
		_, err := client.StackStates.List(ctx, "invalid-id", nil)
		require.Error(t, err)
	})

	states, err := client.StackStates.List(ctx, stackUpdated.ID, nil)
	require.NoError(t, err)
	require.NotNil(t, states)
	require.NotEmpty(t, states.Items)

	state := states.Items[0]

	t.Run("Read with valid ID", func(t *testing.T) {
		state, err := client.StackStates.Read(ctx, state.ID)
		require.NoError(t, err)
		require.NotNil(t, state)

		assert.NotEmpty(t, state.ID)

		// Assert attribute presence
		assert.NotZero(t, state.Generation)
		assert.NotEmpty(t, state.Status)
		assert.NotEmpty(t, state.Deployment)
		assert.NotNil(t, state.Components)
		assert.True(t, state.IsCurrent)
		assert.NotZero(t, state.ResourceInstanceCount)

		// Assert relationship presence
		assert.NotNil(t, state.Stack)
		assert.NotEmpty(t, state.Stack.ID)
		assert.NotNil(t, state.StackDeploymentRun)
		assert.NotEmpty(t, state.StackDeploymentRun)

		// Assert link presence
		assert.NotEmpty(t, state.Links)
		// Description link
		description, ok := state.Links["description"].(string)
		require.True(t, ok)
		assert.NotEmpty(t, description)
	})

	t.Run("Read with invalid ID", func(t *testing.T) {
		_, err := client.StackStates.Read(ctx, "invalid-id")
		require.Error(t, err)
	})

	t.Run("Description with valid ID", func(t *testing.T) {
		rawBytes, err := client.StackStates.Description(ctx, state.ID)
		require.NoError(t, err)
		defer rawBytes.Close()

		b, err := io.ReadAll(rawBytes)
		require.NoError(t, err)
		require.NotEmpty(t, string(b))
	})

	t.Run("Description with invalid ID", func(t *testing.T) {
		_, err := client.StackStates.Description(ctx, "invalid-id")
		require.Error(t, err)
	})
}
