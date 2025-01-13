// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package tfe

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestStackSourceCreateUploadAndRead(t *testing.T) {
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
		},
	})
	require.NoError(t, err)

	ss, err := client.StackSources.CreateAndUpload(ctx, stack.ID, "test-fixtures/stack-source", &CreateStackSourceOptions{
		SelectedDeployments: []string{"simple"},
	})
	require.NoError(t, err)
	require.NotNil(t, ss)
	require.Nil(t, ss.StackConfiguration)

	ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()

	done := make(chan struct{})
	go func() {
		for {
			ss, err = client.StackSources.Read(ctx, ss.ID)
			require.NoError(t, err)

			if ss.StackConfiguration != nil {
				done <- struct{}{}
				return
			}

			time.Sleep(2 * time.Second)
		}
	}()

	select {
	case <-done:
		t.Logf("Found stack source configuration %q", ss.StackConfiguration.ID)
		return
	case <-ctx.Done():
		require.Fail(t, "timed out waiting for stack source to be processed")
	}
}
