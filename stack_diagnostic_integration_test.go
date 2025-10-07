package tfe

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStackDiagnosticsReadAcknowledge(t *testing.T) {
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
			Branch:       "diagnostics",
			OAuthTokenID: oauthClient.OAuthTokens[0].ID,
		},
		Project: &Project{
			ID: orgTest.DefaultProject.ID,
		},
	})

	require.NoError(t, err)
	require.NotNil(t, stack)

	stackUpdated, err := client.Stacks.FetchLatestFromVcs(ctx, stack.ID)
	require.NoError(t, err)
	require.NotNil(t, stackUpdated)

	stackUpdated, err = client.Stacks.Read(ctx, stackUpdated.ID)
	require.NoError(t, err)
	require.NotNil(t, stackUpdated.LatestStackConfiguration)

	pollStackConfigurationStatus(t, ctx, client, stackUpdated.LatestStackConfiguration.ID, "failed")

	diags, err := client.StackConfigurations.Diagnostics(ctx, stackUpdated.LatestStackConfiguration.ID)
	assert.NoError(t, err)
	require.NotEmpty(t, diags.Items)

	diag := diags.Items[0]

	t.Run("Read with valid ID", func(t *testing.T) {
		diag, err := client.StackDiagnostics.Read(ctx, diag.ID)
		require.NoError(t, err)
		assert.NotNil(t, diag)
	})

	t.Run("Read with invalid ID", func(t *testing.T) {
		_, err := client.StackDiagnostics.Read(ctx, "")
		require.Error(t, err)
	})

	t.Run("Acknowledge with valid ID", func(t *testing.T) {
		err := client.StackDiagnostics.Acknowledge(ctx, diag.ID)
		require.NoError(t, err)

		diag, err := client.StackDiagnostics.Read(ctx, diag.ID)
		require.NoError(t, err)
		assert.NotNil(t, diag)
		assert.True(t, diag.Acknowledged)
		assert.NotNil(t, diag.AcknowledgedAt)
		assert.NotNil(t, diag.AcknowledgedBy)
	})

	t.Run("Acknowledge with invalid ID", func(t *testing.T) {
		err := client.StackDiagnostics.Acknowledge(ctx, "")
		require.Error(t, err)
	})
}
