// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package tfe

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestStackPlanList(t *testing.T) {
	skipUnlessBeta(t)

	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	t.Cleanup(orgTestCleanup)

	_, err := client.Projects.Create(ctx, orgTest.Name, ProjectCreateOptions{
		Name: "test-project-2",
	})
	require.NoError(t, err)

	oauthClient, cleanup := createOAuthClient(t, client, orgTest, nil)
	t.Cleanup(cleanup)

	stack, err := client.Stacks.Create(ctx, StackCreateOptions{
		Name: "aa-test-stack",
		VCSRepo: &StackVCSRepo{
			Identifier:   "brandonc/pet-nulls-stack",
			OAuthTokenID: oauthClient.OAuthTokens[0].ID,
		},
		Project: &Project{
			ID: orgTest.DefaultProject.ID,
		},
	})
	require.NoError(t, err)

	stackUpdated, err := client.Stacks.UpdateConfiguration(ctx, stack.ID)
	require.NoError(t, err)

	stackUpdated = pollStackDeployments(t, ctx, client, stackUpdated.ID)
	require.NotNil(t, stackUpdated.LatestStackConfiguration)

	planList, err := client.StackPlans.ListByConfiguration(ctx, stackUpdated.LatestStackConfiguration.ID, &StackPlansListOptions{})
	require.NoError(t, err)

	require.Len(t, planList.Items, 4)

	plan, err := client.StackPlans.Read(ctx, planList.Items[0].ID)
	require.NoError(t, err)
	require.NotNil(t, plan)
}
