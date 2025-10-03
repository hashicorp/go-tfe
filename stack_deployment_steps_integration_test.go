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

	stackUpdated, err := client.Stacks.FetchLatestFromVcs(ctx, stack.ID)
	require.NoError(t, err)
	require.NotNil(t, stackUpdated)

	stackUpdated = pollStackDeploymentGroups(t, ctx, client, stackUpdated.ID)
	require.NotNil(t, stackUpdated.LatestStackConfiguration)

	stackDeploymentGroups, err := client.StackDeploymentGroups.List(ctx, stackUpdated.LatestStackConfiguration.ID, nil)
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
		assert.Equal(t, sdr.ID, step.StackDeploymentRun.ID)
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
		assert.Equal(t, sdr.ID, step.StackDeploymentRun.ID)
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

	stackUpdated, err := client.Stacks.FetchLatestFromVcs(ctx, stack.ID)
	require.NoError(t, err)
	require.NotNil(t, stackUpdated)

	stackUpdated = pollStackDeploymentGroups(t, ctx, client, stackUpdated.ID)
	require.NotNil(t, stackUpdated.LatestStackConfiguration)

	stackDeploymentGroups, err := client.StackDeploymentGroups.List(ctx, stackUpdated.LatestStackConfiguration.ID, nil)
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

func TestStackDeploymentStepsAdvance(t *testing.T) {
	skipUnlessBeta(t)

	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	t.Cleanup(orgTestCleanup)

	oauthClient, cleanup := createOAuthClient(t, client, orgTest, nil)
	t.Cleanup(cleanup)

	stack, err := client.Stacks.Create(ctx, StackCreateOptions{
		Project: orgTest.DefaultProject,
		Name:    "testing-stack",
		VCSRepo: &StackVCSRepoOptions{
			Identifier:   "hashicorp-guides/pet-nulls-stack",
			OAuthTokenID: oauthClient.OAuthTokens[0].ID,
			Branch:       "main",
		},
	})
	require.NoError(t, err)
	require.NotNil(t, stack)

	stackUpdated, err := client.Stacks.FetchLatestFromVcs(ctx, stack.ID)
	require.NoError(t, err)
	require.NotNil(t, stackUpdated)

	stackUpdated = pollStackDeploymentGroups(t, ctx, client, stackUpdated.ID)
	require.NotNil(t, stackUpdated.LatestStackConfiguration)

	stackDeploymentGroups, err := client.StackDeploymentGroups.List(ctx, stackUpdated.LatestStackConfiguration.ID, nil)
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
	step = pollStackDeploymentStepStatus(t, ctx, client, step.ID, "pending_operator")
	require.NotNil(t, step)

	t.Run("Advance with valid ID", func(t *testing.T) {
		err := client.StackDeploymentSteps.Advance(ctx, step.ID)
		assert.NoError(t, err)

		// Verify that the step status has changed to "completed"
		sds, err := client.StackDeploymentSteps.Read(ctx, step.ID)
		assert.NoError(t, err)
		assert.Equal(t, "completed", sds.Status)
	})

	t.Run("Advance with invalid ID", func(t *testing.T) {
		err := client.StackDeploymentSteps.Advance(ctx, "")
		require.Error(t, err)
	})
}

func pollStackDeploymentStepStatus(t *testing.T, ctx context.Context, client *Client, stackDeploymentStepID, status string) (deploymentStep *StackDeploymentStep) {
	// pollStackDeploymentStepStatus will poll the given stack deployment step until its status changes or the deadline is reached.
	ctx, cancel := context.WithDeadline(ctx, time.Now().Add(5*time.Minute))
	defer cancel()

	deadline, _ := ctx.Deadline()
	t.Logf("Polling stack deployment step %q for change in status with deadline of %s", stackDeploymentStepID, deadline)

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	var err error
	for finished := false; !finished; {
		t.Log("...")
		select {
		case <-ctx.Done():
			t.Fatalf("Stack deployment step %s did not have status %q at deadline", stackDeploymentStepID, status)
		case <-ticker.C:
			deploymentStep, err = client.StackDeploymentSteps.Read(ctx, stackDeploymentStepID)
			if err != nil {
				t.Fatalf("Failed to read stack deployment step %s: %s", stackDeploymentStepID, err)
			}

			t.Logf("Stack deployment step %s had status %q", deploymentStep.ID, deploymentStep.Status)
			if deploymentStep.Status == status {
				finished = true
			}
		}
	}

	return
}

func TestStackDeploymentStepsDiagnostics(t *testing.T) {
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
			Identifier:   "ctrombley/linked-stacks-demo-network",
			OAuthTokenID: oauthClient.OAuthTokens[0].ID,
			Branch:       "main",
		},
	})
	require.NoError(t, err)
	require.NotNil(t, stack)

	stackUpdated, err := client.Stacks.FetchLatestFromVcs(ctx, stack.ID)
	require.NoError(t, err)
	require.NotNil(t, stackUpdated)

	stackUpdated = pollStackDeploymentGroups(t, ctx, client, stackUpdated.ID)
	require.NotNil(t, stackUpdated.LatestStackConfiguration)

	stackDeploymentGroups, err := client.StackDeploymentGroups.List(ctx, stackUpdated.LatestStackConfiguration.ID, nil)
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
	step = pollStackDeploymentStepStatus(t, ctx, client, step.ID, "pending_operator")
	require.NotNil(t, step)

	t.Run("Diagnostics with valid ID", func(t *testing.T) {
		sds, err := client.StackDeploymentSteps.Diagnostics(ctx, step.ID)
		assert.NoError(t, err)
		assert.NotEmpty(t, sds)
	})

	t.Run("Diagnostics with invalid ID", func(t *testing.T) {
		_, err := client.StackDeploymentSteps.Diagnostics(ctx, step.ID)
		require.Error(t, err)
	})
}
