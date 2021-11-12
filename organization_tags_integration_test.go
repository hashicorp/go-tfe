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

	t.Run("with no query params", func(t *testing.T) {
		tags, err := client.OrganizationTags.List(ctx, orgTest.Name, OrganizationTagsListOptions{})
		require.NoError(t, err)

		assert.Equal(t, 10, len(tags.Items))

		for _, tag := range tags.Items {
			assert.NotNil(t, tag.ID)
			assert.NotNil(t, tag.Name)
			assert.NotNil(t, tag.InstanceCount)

			t.Run("ensure org relation is properly decoded", func(t *testing.T) {
				assert.NotNil(t, tag.Organization)
			})
		}
	})
}
