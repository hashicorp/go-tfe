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

		assert.NotEmpty(t, diag.ID)
		assert.NotEmpty(t, diag.Severity)
		assert.NotEmpty(t, diag.Summary)
		assert.NotEmpty(t, diag.Detail)
		assert.NotEmpty(t, diag.Diags)

		for _, d := range diag.Diags {
			assert.NotEmpty(t, d.Detail)
			assert.NotEmpty(t, d.Severity)
			assert.NotEmpty(t, d.Summary)
			assert.Empty(t, d.Origin)

			require.NotNil(t, d.Range)
			assert.NotEmpty(t, d.Range.Filename)
			assert.NotEmpty(t, d.Range.Source)

			require.NotNil(t, d.Range.Start)
			assert.NotZero(t, d.Range.Start.Line)
			assert.NotZero(t, d.Range.Start.Column)
			assert.NotZero(t, d.Range.Start.Byte)

			require.NotNil(t, d.Range.End)
			assert.NotZero(t, d.Range.End.Line)
			assert.NotZero(t, d.Range.End.Column)
			assert.NotZero(t, d.Range.End.Byte)

			require.NotNil(t, d.Snippet)
			assert.NotEmpty(t, d.Snippet.Code)
			assert.Empty(t, d.Snippet.Values)
			assert.Nil(t, d.Snippet.Context)
			assert.Zero(t, d.Snippet.HighlightStartOffset)
			assert.NotZero(t, d.Snippet.HighlightEndOffset)
		}

		assert.False(t, diag.Acknowledged)
		assert.Nil(t, diag.AcknowledgedAt)
		assert.NotZero(t, diag.CreatedAt)

		assert.Nil(t, diag.StackDeploymentStep)
		assert.NotNil(t, diag.StackConfiguration)
		assert.Nil(t, diag.AcknowledgedBy)
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
