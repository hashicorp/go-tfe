package tfe

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestStackDeploymentsList(t *testing.T) {
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
			Identifier:   "hashicorp-guides/pet-nulls-stack",
			OAuthTokenID: oauthClient.OAuthTokens[0].ID,
		},
		Project: &Project{
			ID: orgTest.DefaultProject.ID,
		},
	})
	require.NoError(t, err)
	require.NotNil(t, stack)

	stackUpdated, err := client.Stacks.FetchLatestFromVcs(ctx, stack.ID)
	require.NoError(t, err)
	require.NotNil(t, stackUpdated)

	stackUpdated = pollStackDeploymentGroups(t, ctx, client, stackUpdated.ID)

	t.Run("List with valid options", func(t *testing.T) {
		opts := &StackDeploymentListOptions{
			ListOptions: ListOptions{
				PageNumber: 1,
				PageSize:   1,
			},
		}
		sdl, err := client.StackDeployments.List(ctx, stackUpdated.ID, opts)
		require.NoError(t, err)
		require.NotNil(t, sdl)
		require.Len(t, sdl.Items, 1)
	})

	t.Run("List with invalid options", func(t *testing.T) {
		opts := &StackDeploymentListOptions{
			ListOptions: ListOptions{
				PageNumber: -1,
				PageSize:   -1,
			},
		}

		_, err := client.StackDeployments.List(ctx, stackUpdated.ID, opts)
		require.Error(t, err)
	})
}
