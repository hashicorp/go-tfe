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

func TestStackConfigurationList(t *testing.T) {
	skipUnlessBeta(t)

	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	t.Cleanup(orgTestCleanup)

	oauthClient, cleanup := createOAuthClient(t, client, orgTest, nil)
	t.Cleanup(cleanup)

	stack, err := client.Stacks.Create(ctx, StackCreateOptions{
		Name: "test-stack-list",
		VCSRepo: &StackVCSRepoOptions{
			Identifier:   "shwetamurali/pet-nulls-stack",
			OAuthTokenID: oauthClient.OAuthTokens[0].ID,
		},
		Project: &Project{
			ID: orgTest.DefaultProject.ID,
		},
	})
	require.NoError(t, err)

	// Trigger first stack configuration by updating configuration
	_, err = client.Stacks.UpdateConfiguration(ctx, stack.ID)
	require.NoError(t, err)

	// Wait a bit and trigger second stack configuration
	time.Sleep(2 * time.Second)
	_, err = client.Stacks.UpdateConfiguration(ctx, stack.ID)
	require.NoError(t, err)

	list, err := client.StackConfigurations.List(ctx, stack.ID, nil)
	require.NoError(t, err)
	require.NotNil(t, list)
	assert.Equal(t, len(list.Items), 2)

	// Assert attributes for each configuration
	for _, cfg := range list.Items {
		require.NotEmpty(t, cfg.ID)
		require.NotEmpty(t, cfg.Status)
		require.GreaterOrEqual(t, cfg.SequenceNumber, 1)

		if cfg.Relationships != nil {
			stackRel, ok := cfg.Relationships["stack"].(map[string]interface{})
			require.True(t, ok)
			data, ok := stackRel["data"].(map[string]interface{})
			require.True(t, ok)
			assert.NotEmpty(t, data["id"])
			assert.Equal(t, "stacks", data["type"])
		}
	}

	// Test with pagination options
	t.Run("with pagination options", func(t *testing.T) {
		options := &StackConfigurationListOptions{
			ListOptions: ListOptions{
				PageNumber: 1,
				PageSize:   10,
			},
		}

		listWithOptions, err := client.StackConfigurations.List(ctx, stack.ID, options)
		require.NoError(t, err)
		require.NotNil(t, listWithOptions)
		assert.GreaterOrEqual(t, len(listWithOptions.Items), 2)

		require.NotNil(t, listWithOptions.Pagination)
		assert.GreaterOrEqual(t, listWithOptions.Pagination.TotalCount, 2)
	})
}
