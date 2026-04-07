package tfe

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStackDiagnosticsRead(t *testing.T) {
	t.Parallel()

	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	t.Cleanup(orgTestCleanup)

	oauthClient, cleanup := createOAuthClient(t, client, orgTest, nil)
	t.Cleanup(cleanup)

	stack, err := client.Stacks.Create(ctx, StackCreateOptions{
		Name: "cc-test-stack",
		VCSRepo: &StackVCSRepoOptions{
			Identifier:   stackVCSRepoIdentifier(t),
			Branch:       "diags",
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

	expectedDiag := diags.Items[0]

	t.Run("Read with valid ID", func(t *testing.T) {
		diag, err := client.StackDiagnostics.Read(ctx, expectedDiag.ID)
		require.NoError(t, err)
		assert.NotNil(t, diag)

		assert.Equal(t, expectedDiag.ID, diag.ID)
		assert.Equal(t, expectedDiag.Severity, diag.Severity)
		assert.Equal(t, expectedDiag.Summary, diag.Summary)
		assert.Equal(t, expectedDiag.Detail, diag.Detail)
		assert.Equal(t, expectedDiag.CreatedAt, diag.CreatedAt)
		require.Len(t, diag.Diags, len(expectedDiag.Diags))

		for i, d := range diag.Diags {
			expectedNestedDiag := expectedDiag.Diags[i]
			assert.Equal(t, expectedNestedDiag.Severity, d.Severity)
			assert.Equal(t, expectedNestedDiag.Summary, d.Summary)
			assert.Equal(t, expectedNestedDiag.Detail, d.Detail)
			assert.Equal(t, expectedNestedDiag.Origin, d.Origin)
			assert.Equal(t, expectedNestedDiag.Range, d.Range)
			assert.Equal(t, expectedNestedDiag.Snippet, d.Snippet)
		}

		assert.NotZero(t, diag.CreatedAt)
		assert.Equal(t, expectedDiag.StackDeploymentStep, diag.StackDeploymentStep)
		require.NotNil(t, diag.StackConfiguration)
		require.NotNil(t, expectedDiag.StackConfiguration)
		assert.Equal(t, expectedDiag.StackConfiguration.ID, diag.StackConfiguration.ID)
	})

	t.Run("Read with invalid ID", func(t *testing.T) {
		_, err := client.StackDiagnostics.Read(ctx, "")
		require.Error(t, err)
	})
}
