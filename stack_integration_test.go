// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package tfe

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStackCreateAndList(t *testing.T) {
	skipUnlessBeta(t)

	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	t.Cleanup(orgTestCleanup)

	project2, err := client.Projects.Create(ctx, orgTest.Name, ProjectCreateOptions{
		Name: "test-project-2",
	})
	require.NoError(t, err)

	oauthClient, cleanup := createOAuthClient(t, client, orgTest, nil)
	t.Cleanup(cleanup)

	stack1, err := client.Stacks.Create(ctx, StackCreateOptions{
		Name: "aa-test-stack",
		VCSRepo: &StackVCSRepo{
			Identifier:   "hashicorp-guides/pet-nulls-stack",
			OAuthTokenID: oauthClient.OAuthTokens[0].ID,
		},
		Project: &Project{
			ID: orgTest.DefaultProject.ID,
		},
	})

	require.NoError(t, err)
	require.NotNil(t, stack1)

	stack2, err := client.Stacks.Create(ctx, StackCreateOptions{
		Name: "zz-test-stack",
		VCSRepo: &StackVCSRepo{
			Identifier:   "hashicorp-guides/pet-nulls-stack",
			OAuthTokenID: oauthClient.OAuthTokens[0].ID,
		},
		Project: &Project{
			ID: project2.ID,
		},
	})

	require.NoError(t, err)
	require.NotNil(t, stack2)

	t.Run("List without options", func(t *testing.T) {
		t.Parallel()

		stackList, err := client.Stacks.List(ctx, orgTest.Name, nil)
		require.NoError(t, err)

		assert.Len(t, stackList.Items, 2)
	})

	t.Run("List with project filter", func(t *testing.T) {
		t.Parallel()

		stackList, err := client.Stacks.List(ctx, orgTest.Name, &StackListOptions{
			ProjectID: project2.ID,
		})
		require.NoError(t, err)

		assert.Len(t, stackList.Items, 1)
		assert.Equal(t, stack2.ID, stackList.Items[0].ID)
	})

	t.Run("List with name filter", func(t *testing.T) {
		t.Parallel()

		stackList, err := client.Stacks.List(ctx, orgTest.Name, &StackListOptions{
			SearchByName: "zz",
		})
		require.NoError(t, err)

		assert.Len(t, stackList.Items, 1)
		assert.Equal(t, stack2.ID, stackList.Items[0].ID)
	})

	t.Run("List with sort options", func(t *testing.T) {
		t.Parallel()

		// By name ASC
		stackList, err := client.Stacks.List(ctx, orgTest.Name, &StackListOptions{
			Sort: StackSortByName,
		})
		require.NoError(t, err)

		assert.Len(t, stackList.Items, 2)
		assert.Equal(t, stack1.ID, stackList.Items[0].ID)

		// By name DESC
		stackList, err = client.Stacks.List(ctx, orgTest.Name, &StackListOptions{
			Sort: StackSortByNameDesc,
		})
		require.NoError(t, err)

		assert.Len(t, stackList.Items, 2)
		assert.Equal(t, stack2.ID, stackList.Items[0].ID)
	})

	t.Run("List with pagination", func(t *testing.T) {
		t.Parallel()

		stackList, err := client.Stacks.List(ctx, orgTest.Name, &StackListOptions{
			ListOptions: ListOptions{
				PageNumber: 1,
				PageSize:   1,
			},
		})
		require.NoError(t, err)

		assert.Len(t, stackList.Items, 1)
		assert.Equal(t, 2, stackList.Pagination.TotalPages)
		assert.Equal(t, 2, stackList.Pagination.TotalCount)
	})
}

func TestStackReadUpdateDelete(t *testing.T) {
	skipUnlessBeta(t)

	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	t.Cleanup(orgTestCleanup)

	oauthClient, cleanup := createOAuthClient(t, client, orgTest, nil)
	t.Cleanup(cleanup)

	stack, err := client.Stacks.Create(ctx, StackCreateOptions{
		Name: "test-stack",
		VCSRepo: &StackVCSRepo{
			Identifier:   "brandonc/pet-nulls-stack",
			OAuthTokenID: oauthClient.OAuthTokens[0].ID,
		},
		Project: &Project{
			ID: orgTest.DefaultProject.ID,
		},
	})

	require.NoError(t, err)
	require.NotNil(t, stack)

	stackRead, err := client.Stacks.Read(ctx, stack.ID)
	require.NoError(t, err)

	assert.Equal(t, stack, stackRead)

	stackUpdated, err := client.Stacks.Update(ctx, stack.ID, StackUpdateOptions{
		Description: String("updated description"),
	})

	require.NoError(t, err)
	require.Equal(t, "updated description", stackUpdated.Description)

	err = client.Stacks.Delete(ctx, stack.ID)
	require.NoError(t, err)

	stackReadAfterDelete, err := client.Stacks.Read(ctx, stack.ID)
	require.ErrorIs(t, err, ErrResourceNotFound)
	require.Nil(t, stackReadAfterDelete)
}
