// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package tfe

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStackDeploymentStepsList(t *testing.T) {
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

	sdr := stackDeploymentRuns.Items[0]

	t.Run("List with invalid stack deployment run ID", func(t *testing.T) {
		t.Parallel()

		_, err := client.StackDeploymentSteps.List(ctx, "", nil)
		assert.Error(t, err)
	})

	t.Run("List without options", func(t *testing.T) {
		t.Parallel()

		steps, err := client.StackDeploymentSteps.List(ctx, sdr.ID, nil)
		assert.NoError(t, err)
		assert.NotEmpty(t, steps)

		step := steps.Items[0]

		assert.NotNil(t, step)
		assert.NotNil(t, step.ID)
		assert.NotNil(t, step.Status)

		require.NotNil(t, step.StackDeploymentRun)
		assert.Equal(t, sdg.ID, step.StackDeploymentRun.ID)
	})

	t.Run("List with pagination", func(t *testing.T) {
		t.Parallel()

		steps, err := client.StackDeploymentSteps.List(ctx, sdr.ID, &StackDeploymentStepsListOptions{
			ListOptions: ListOptions{
				PageNumber: 1,
				PageSize:   10,
			},
		})
		assert.NoError(t, err)
		assert.NotEmpty(t, steps)

		step := steps.Items[0]

		assert.NotNil(t, step)
		assert.NotNil(t, step.ID)
		assert.NotNil(t, step.Status)

		require.NotNil(t, step.StackDeploymentRun)
		assert.Equal(t, sdg.ID, step.StackDeploymentRun.ID)
	})
}

func TestStackDeploymentStepsRead(t *testing.T) {
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

	sdr := stackDeploymentRuns.Items[0]

	steps, err := client.StackDeploymentSteps.List(ctx, sdr.ID, nil)
	assert.NoError(t, err)
	assert.NotEmpty(t, steps)

	step := steps.Items[0]

	t.Run("Read with valid ID", func(t *testing.T) {
		sds, err := client.StackDeploymentSteps.Read(ctx, step.ID)
		assert.NoError(t, err)
		assert.NotEmpty(t, sds.ID)
		assert.NotEmpty(t, sds.Status)
	})

	t.Run("Read with invalid ID", func(t *testing.T) {
		_, err := client.StackDeploymentSteps.Read(ctx, "")
		require.Error(t, err)
	})
}
