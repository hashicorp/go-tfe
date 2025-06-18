// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package tfe

import (
	"context"
	"testing"

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

	// Trigger a stack configuration by updating configuration
	_, err = client.Stacks.UpdateConfiguration(ctx, stack.ID)
	NoError(t, err)

	// List stack configurations
	list, err := client.StackConfigurations.List(ctx, stack.ID, nil)
	NoError(t, err)
	NotNil(t, list)
	GreaterOrEqual(t, len(list.Items), 1)

	for _, cfg := range list.Items {
		NotEmpty(t, cfg.ID)
		// Optionally, check other fields that relate the configuration to the stack, if available
	}
}
