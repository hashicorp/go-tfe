// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package tfe

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOrganizationTagsList(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	defer orgTestCleanup()

	assert.NotNil(t, orgTest)

	workspaceTest, workspaceTestCleanup := createWorkspace(t, client, orgTest)
	defer workspaceTestCleanup()

	assert.NotNil(t, workspaceTest)

	var tags []*Tag
	for i := 0; i < 10; i++ {
		tags = append(tags, &Tag{
			Name: fmt.Sprintf("tag%d", i),
		})
	}

	err := client.Workspaces.AddTags(ctx, workspaceTest.ID, WorkspaceAddTagsOptions{
		Tags: tags,
	})
	require.NoError(t, err)

	// this is a tag id we'll use in the filter param of the second test
	var testTagID string

	t.Run("with no query params", func(t *testing.T) {
		tags, err := client.OrganizationTags.List(ctx, orgTest.Name, nil)
		require.NoError(t, err)

		assert.Equal(t, 10, len(tags.Items))

		testTagID = tags.Items[0].ID

		for _, tag := range tags.Items {
			assert.NotNil(t, tag.ID)
			assert.NotNil(t, tag.Name)
			assert.GreaterOrEqual(t, tag.InstanceCount, 1)

			t.Run("ensure org relation is properly decoded", func(t *testing.T) {
				assert.NotNil(t, tag.Organization)
			})
		}
	})

	t.Run("with query params", func(t *testing.T) {
		tags, err := client.OrganizationTags.List(ctx, orgTest.Name, &OrganizationTagsListOptions{
			ListOptions: ListOptions{
				PageNumber: 1,
				PageSize:   5,
			},
			Filter: testTagID,
		})
		require.NoError(t, err)

		assert.Equal(t, 5, len(tags.Items))

		for _, tag := range tags.Items {
			// ensure tag specified in filter param was omitted from results
			assert.NotNil(t, tag.ID, testTagID)

			t.Run("ensure org relation is properly decoded", func(t *testing.T) {
				assert.NotNil(t, tag.Organization)
			})
		}
	})
}

func TestOrganizationTagsDelete(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	defer orgTestCleanup()

	assert.NotNil(t, orgTest)

	workspaceTest, workspaceTestCleanup := createWorkspace(t, client, orgTest)
	defer workspaceTestCleanup()

	assert.NotNil(t, workspaceTest)

	var tags []*Tag
	for i := 0; i < 10; i++ {
		tags = append(tags, &Tag{
			Name: fmt.Sprintf("tag%d", i),
		})
	}

	err := client.Workspaces.AddTags(ctx, workspaceTest.ID, WorkspaceAddTagsOptions{
		Tags: tags,
	})
	require.NoError(t, err)

	t.Run("delete tags by id", func(t *testing.T) {
		tags, err := client.OrganizationTags.List(ctx, orgTest.Name, nil)
		require.NoError(t, err)

		var tagIds []string
		// since we added 10 tags to the org, grab a subset
		for i := 0; i < 5; i++ {
			assert.NotNil(t, tags.Items[i].ID)
			tagIds = append(tagIds, tags.Items[i].ID)
		}

		err = client.OrganizationTags.Delete(ctx, orgTest.Name, OrganizationTagsDeleteOptions{
			IDs: tagIds,
		})
		require.NoError(t, err)

		// sanity check ensure tags were deleted from the organization
		tags, err = client.OrganizationTags.List(ctx, orgTest.Name, nil)
		require.NoError(t, err)

		assert.Equal(t, 5, len(tags.Items))
	})
}

func TestOrganizationTagsAddWorkspace(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	defer orgTestCleanup()

	assert.NotNil(t, orgTest)

	workspaceTest, workspaceTestCleanup := createWorkspace(t, client, orgTest)
	defer workspaceTestCleanup()

	assert.NotNil(t, workspaceTest)

	var tags []*Tag
	for i := 0; i < 2; i++ {
		tags = append(tags, &Tag{
			Name: fmt.Sprintf("tag%d", i),
		})
	}

	err := client.Workspaces.AddTags(ctx, workspaceTest.ID, WorkspaceAddTagsOptions{
		Tags: tags,
	})
	require.NoError(t, err)

	t.Run("add tags to new workspaces", func(t *testing.T) {
		// fetch tag ids to associate to workspace
		tags, err := client.OrganizationTags.List(ctx, orgTest.Name, nil)
		require.NoError(t, err)

		tagID := tags.Items[0].ID

		// create the workspaces we'll use to associate tags
		workspaceToAdd1, workspaceToAdd1Cleanup := createWorkspace(t, client, orgTest)
		defer workspaceToAdd1Cleanup()

		workspaceToAdd2, workspaceToAdd2Cleanup := createWorkspace(t, client, orgTest)
		defer workspaceToAdd2Cleanup()

		err = client.OrganizationTags.AddWorkspaces(ctx, tagID, AddWorkspacesToTagOptions{
			WorkspaceIDs: []string{workspaceToAdd1.ID, workspaceToAdd2.ID},
		})
		require.NoError(t, err)

		// Ensure the tag was properly associated with the workspaces
		fetched, err := client.Workspaces.ListTags(ctx, workspaceToAdd1.ID, nil)
		require.NoError(t, err)
		assert.Equal(t, fetched.Items[0].ID, tagID)

		fetched, err = client.Workspaces.ListTags(ctx, workspaceToAdd2.ID, nil)
		require.NoError(t, err)
		assert.Equal(t, fetched.Items[0].ID, tagID)
	})
}
