// Copyright IBM Corp. 2018, 2025
// SPDX-License-Identifier: MPL-2.0

package tfe

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStackDeploymentGroupSummaryList(t *testing.T) {
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

	// Trigger first stack configuration with a fetch
	_, err = client.Stacks.FetchLatestFromVcs(ctx, stack.ID)
	require.NoError(t, err)

	updatedStack := pollStackDeploymentGroups(t, ctx, client, stack.ID)
	require.NotNil(t, updatedStack.LatestStackConfiguration.ID)

	// Trigger second stack configuration with a fetch
	_, err = client.Stacks.FetchLatestFromVcs(ctx, stack2.ID)
	require.NoError(t, err)

	updatedStack2 := pollStackDeploymentGroups(t, ctx, client, stack2.ID)
	require.NotNil(t, updatedStack2.LatestStackConfiguration.ID)

	t.Run("Successful multiple deployment group summary list", func(t *testing.T) {
		stackConfigSummaryList, err := client.StackDeploymentGroupSummaries.List(ctx, updatedStack2.LatestStackConfiguration.ID, nil)
		require.NoError(t, err)

		assert.Len(t, stackConfigSummaryList.Items, 2)
	})

	t.Run("Unsuccessful list", func(t *testing.T) {
		_, err := client.StackDeploymentGroupSummaries.List(ctx, "", nil)
		require.Error(t, err)
	})
}
