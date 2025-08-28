// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package tfe

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStackConfigurationSummaryList(t *testing.T) {
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
			Identifier:   "hwatkins05-hashicorp/pet-nulls-stack",
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
			Identifier:   "hwatkins05-hashicorp/pet-nulls-stack",
			OAuthTokenID: oauthClient.OAuthTokens[0].ID,
		},
		Project: &Project{
			ID: orgTest.DefaultProject.ID,
		},
	})
	require.NoError(t, err)
	require.NotNil(t, stack2)

	// Trigger first stack configuration by updating configuration
	_, err = client.Stacks.UpdateConfiguration(ctx, stack2.ID)
	require.NoError(t, err)

	// Wait a bit and trigger second stack configuration
	time.Sleep(2 * time.Second)
	_, err = client.Stacks.UpdateConfiguration(ctx, stack2.ID)
	require.NoError(t, err)

	t.Run("Successful empty list", func(t *testing.T) {
		stackConfigSummaryList, err := client.StackConfigurationSummaries.List(ctx, stack.ID)
		require.NoError(t, err)

		assert.Len(t, stackConfigSummaryList.Items, 0)
	})

	t.Run("Successful multiple config summary list", func(t *testing.T) {
		stackConfigSummaryList, err := client.StackConfigurationSummaries.List(ctx, stack2.ID)
		require.NoError(t, err)

		assert.Len(t, stackConfigSummaryList.Items, 2)
	})

	t.Run("Unsuccessful list", func(t *testing.T) {
		_, err := client.StackConfigurationSummaries.List(ctx, "")
		require.Error(t, err)
	})
}
