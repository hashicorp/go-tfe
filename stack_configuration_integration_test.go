// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package tfe

import (
	"context"
	"testing"
	"time"

	. "github.com/stretchr/testify/require"
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
	NoError(t, err)

	// Trigger first stack configuration by updating configuration
	_, err = client.Stacks.UpdateConfiguration(ctx, stack.ID)
	NoError(t, err)

	// Wait a bit and trigger second stack configuration
	time.Sleep(2 * time.Second)
	_, err = client.Stacks.UpdateConfiguration(ctx, stack.ID)
	NoError(t, err)

	list, err := client.StackConfigurations.List(ctx, stack.ID, nil)
	NoError(t, err)
	NotNil(t, list)
	Equal(t, len(list.Items), 2)

	// Assert attributes for each configuration
	for _, cfg := range list.Items {
		NotEmpty(t, cfg.ID)
		NotEmpty(t, cfg.Status)
		GreaterOrEqual(t, cfg.SequenceNumber, 1)
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
		NoError(t, err)
		NotNil(t, listWithOptions)
		Equal(t, len(listWithOptions.Items), 2)
		NotNil(t, listWithOptions.Pagination)
	})
}
