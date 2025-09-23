package tfe

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestStackDiagnosticsRead(t *testing.T) {
	skipUnlessBeta(t)

	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	t.Cleanup(orgTestCleanup)

	oauthClient, cleanup := createOAuthClient(t, client, orgTest, nil)
	t.Cleanup(cleanup)

	stack, err := client.Stacks.Create(ctx, StackCreateOptions{
		Name: "cc-test-stack",
		VCSRepo: &StackVCSRepoOptions{
			Identifier:   "ctrombley/linked-stacks-demo-network",
			OAuthTokenID: oauthClient.OAuthTokens[0].ID,
		},
		Project: &Project{
			ID: orgTest.DefaultProject.ID,
		},
	})

	require.NoError(t, err)
	require.NotNil(t, stack)

	// Trigger first stack configuration with a fetch
	_, err = client.Stacks.FetchLatestFromVcs(ctx, stack.ID)
	require.NoError(t, err)

	updatedStack := pollStackDeploymentGroups(t, ctx, client, stack.ID)
	require.NotNil(t, updatedStack.LatestStackConfiguration.ID)

	sdgl, err := client.StackDeploymentGroups.List(ctx, updatedStack.LatestStackConfiguration.ID, nil)
	require.NoError(t, err)
	require.NotNil(t, sdgl)
	require.Len(t, sdgl.Items, 2)

	t.Run("Read with valid ID", func(t *testing.T) {
		_, err := client.StackDiagnostics.Read(ctx, sdgl.Items[0].ID)
		require.NoError(t, err)
		// assert.Equal(t, sdgl.Items[0].ID, sdgRead.StackDeploymentGroup.ID)
		// assert.NotNil(t, sdgRead.Diagnostics)
	})

	t.Run("Read with invalid ID", func(t *testing.T) {
		_, err := client.StackDiagnostics.Read(ctx, "")
		require.Error(t, err)
	})
}

func TestStackDiagnosticsAcknowledge(t *testing.T) {
	skipUnlessBeta(t)

	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	t.Cleanup(orgTestCleanup)

	oauthClient, cleanup := createOAuthClient(t, client, orgTest, nil)
	t.Cleanup(cleanup)

	stack, err := client.Stacks.Create(ctx, StackCreateOptions{
		Name: "cc-test-stack",
		VCSRepo: &StackVCSRepoOptions{
			Identifier:   "ctrombley/linked-stacks-demo-network",
			OAuthTokenID: oauthClient.OAuthTokens[0].ID,
		},
		Project: &Project{
			ID: orgTest.DefaultProject.ID,
		},
	})

	require.NoError(t, err)
	require.NotNil(t, stack)

	// Trigger first stack configuration with a fetch
	_, err = client.Stacks.FetchLatestFromVcs(ctx, stack.ID)
	require.NoError(t, err)

	updatedStack := pollStackDeploymentGroups(t, ctx, client, stack.ID)
	require.NotNil(t, updatedStack.LatestStackConfiguration.ID)

	sdgl, err := client.StackDeploymentGroups.List(ctx, updatedStack.LatestStackConfiguration.ID, nil)
	require.NoError(t, err)
	require.NotNil(t, sdgl)
	require.Len(t, sdgl.Items, 2)

	t.Run("Acknowledge with valid ID", func(t *testing.T) {
		err := client.StackDiagnostics.Acknowledge(ctx, "")
		require.NoError(t, err)
	})

	t.Run("Acknowledge with invalid ID", func(t *testing.T) {
		err := client.StackDiagnostics.Acknowledge(ctx, "")
		require.Error(t, err)
	})
}
