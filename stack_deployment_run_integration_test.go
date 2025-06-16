// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package tfe

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStackDeploymentRunList(t *testing.T) {
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
			Identifier:   "shwetamurali/pet-nulls-stack",
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
	deploymentGroupID := stack.LatestStackConfiguration.ID

	t.Run("List without options", func(t *testing.T) {
		t.Parallel()

		runList, err := client.StackDeploymentRuns.List(ctx, deploymentGroupID, nil)
		require.NoError(t, err)
		assert.NotNil(t, runList)
	})

	t.Run("List with pagination", func(t *testing.T) {
		t.Parallel()

		runList, err := client.StackDeploymentRuns.List(ctx, deploymentGroupID, &StackDeploymentRunListOptions{
			ListOptions: ListOptions{
				PageNumber: 1,
				PageSize:   10,
			},
		})
		require.NoError(t, err)
		assert.NotNil(t, runList)
	})
}
