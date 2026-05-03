package tfe

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRegistryComponentUpdate_Beta(t *testing.T) {
	t.Parallel()
	skipUnlessBeta(t)
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	defer orgTestCleanup()

	t.Run("with valid options", func(t *testing.T) {
		// Create project tags
		_, projectTestCleanup := createProjectWithOptions(t, client, orgTest, ProjectCreateOptions{
			Name: "project-with-tags",
			TagBindings: []*TagBinding{
				{Key: "env", Value: "production"},
			},
		})
		t.Cleanup(projectTestCleanup)

		rcBefore, rcTestCleanup := createRegistryComponent(t, client, orgTest)
		t.Cleanup(rcTestCleanup)

		// Update the component with tag_bindings
		updateOptions := &RegistryComponentUpdateOptions{
			TagBindings: []*TagBinding{
				{Key: "env", Value: "production"},
			},
		}

		rcAfter, err := client.RegistryComponents.Update(ctx, rcBefore.ID, updateOptions)
		require.NoError(t, err)

		// Verify the provider has the new tag bindings
		bindings, err := client.RegistryComponents.ListTagBindings(ctx, rcAfter.ID)
		require.NoError(t, err)

		require.Len(t, bindings, 1)
		assert.Equal(t, "env", bindings[0].Key)
		assert.Equal(t, "production", bindings[0].Value)

		// Delete the tag_bindings
		deleteOptions := &RegistryComponentUpdateOptions{
			TagBindings: []*TagBinding{},
		}

		rcAfterDelete, err := client.RegistryComponents.Update(ctx, rcBefore.ID, deleteOptions)
		require.NoError(t, err)

		// Verify the component has no tag bindings
		bindingsAfterDelete, err := client.RegistryComponents.ListTagBindings(ctx, rcAfterDelete.ID)
		require.NoError(t, err)
		require.Empty(t, bindingsAfterDelete)
	})
}
