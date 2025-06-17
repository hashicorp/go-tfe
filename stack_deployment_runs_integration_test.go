// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package tfe

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStackDeploymentRunsList(t *testing.T) {
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
			Identifier:   "hashicorp-guides/pet-nulls-stack",
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
	deploymentGroups, err := client.StackDeploymentGroups.List(ctx, stack.LatestStackConfiguration.ID, nil)
	require.NoError(t, err)
	require.NotNil(t, deploymentGroups)
	require.NotEmpty(t, deploymentGroups.Items)
	deploymentGroupID := deploymentGroups.Items[0].ID

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

func TestStackDeploymentRunsRead(t *testing.T) {
	skipUnlessBeta(t)

	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	t.Cleanup(orgTestCleanup)

	oauthClient, cleanup := createOAuthClient(t, client, orgTest, nil)
	t.Cleanup(cleanup)

	stack, err := client.Stacks.Create(ctx, StackCreateOptions{
		Project: orgTest.DefaultProject,
		Name:    "test-stack",
		VCSRepo: &StackVCSRepoOptions{
			Identifier:   "hashicorp-guides/pet-nulls-stack",
			OAuthTokenID: oauthClient.OAuthTokens[0].ID,
			Branch:       "main",
		},
	})
	require.NoError(t, err)
	require.NotNil(t, stack)

	stackUpdated, err := client.Stacks.UpdateConfiguration(ctx, stack.ID)
	require.NoError(t, err)
	require.NotNil(t, stackUpdated)

	stack = pollStackDeployments(t, ctx, client, stackUpdated.ID)
	require.NotNil(t, stack.LatestStackConfiguration)

	stackDeploymentGroups, err := client.StackDeploymentGroups.List(ctx, stack.LatestStackConfiguration.ID, nil)
	require.NoError(t, err)
	require.NotEmpty(t, stackDeploymentGroups)

	sdg := stackDeploymentGroups.Items[0]

	stackDeploymentRuns, err := client.StackDeploymentRuns.List(ctx, sdg.ID, nil)
	require.NoError(t, err)
	require.NotEmpty(t, stackDeploymentRuns)

	sdr := stackDeploymentGroups.Items[0]

	t.Run("Read with valid ID", func(t *testing.T) {
		run, err := client.StackDeploymentRuns.Read(ctx, sdr.ID)
		assert.NoError(t, err)
		assert.NotNil(t, run)
	})

	t.Run("Read with invalid ID", func(t *testing.T) {
		_, err := client.StackDeploymentRuns.Read(ctx, "")
		assert.Error(t, err)
	})
}

func TestStackDeploymentRunsApproveAllPlans(t *testing.T) {
	skipUnlessBeta(t)

	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	t.Cleanup(orgTestCleanup)

	oauthClient, cleanup := createOAuthClient(t, client, orgTest, nil)
	t.Cleanup(cleanup)

	stack, err := client.Stacks.Create(ctx, StackCreateOptions{
		Project: orgTest.DefaultProject,
		Name:    "test-stack",
		VCSRepo: &StackVCSRepoOptions{
			Identifier:   "hashicorp-guides/pet-nulls-stack",
			OAuthTokenID: oauthClient.OAuthTokens[0].ID,
			Branch:       "main",
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
	deploymentGroups, err := client.StackDeploymentGroups.List(ctx, stack.LatestStackConfiguration.ID, nil)
	require.NoError(t, err)
	require.NotNil(t, deploymentGroups)
	require.NotEmpty(t, deploymentGroups.Items)

	deploymentGroupID := deploymentGroups.Items[0].ID

	runList, err := client.StackDeploymentRuns.List(ctx, deploymentGroupID, nil)
	require.NoError(t, err)
	assert.NotNil(t, runList)

	deploymentRunID := runList.Items[0].ID

	t.Run("Approve all plans", func(t *testing.T) {
		t.Parallel()

		err := client.StackDeploymentRuns.ApproveAllPlans(ctx, deploymentRunID)
		require.NoError(t, err)
	})
}
