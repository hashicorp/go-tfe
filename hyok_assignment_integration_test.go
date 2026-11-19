// Copyright IBM Corp. 2018, 2026
// SPDX-License-Identifier: MPL-2.0

package tfe

import (
	"context"
	"os"
	"testing"

	"github.com/hashicorp/jsonapi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testHYOKConfigurationID returns an available HYOK configuration in the
// HYOK_ORGANIZATION_NAME organization. That organization must have HYOK
// configurations by workspace enabled, and the token must belong to an
// organization owner.
func testHYOKConfigurationID(t *testing.T) string {
	t.Helper()
	id := os.Getenv("HYOK_CONFIGURATION_ID")
	if id == "" {
		t.Fatal("Export a valid HYOK_CONFIGURATION_ID (an available HYOK configuration in HYOK_ORGANIZATION_NAME) before running this test!")
	}
	return id
}

func TestProjects_DefaultHYOKConfiguration(t *testing.T) {
	skipHYOKIntegrationTests(t)

	client := testClient(t)
	ctx := context.Background()
	org := testHyokOrganization(t, client)
	hyokConfigID := testHYOKConfigurationID(t)

	p, pCleanup := createProjectWithOptions(t, client, org, ProjectCreateOptions{
		Name:                       randomStringWithoutSpecialChar(t),
		DefaultHYOKConfigurationID: String(hyokConfigID),
	})
	t.Cleanup(pCleanup)

	require.NotNil(t, p.DefaultHYOKConfiguration)
	assert.Equal(t, hyokConfigID, p.DefaultHYOKConfiguration.ID)

	t.Run("read returns the default", func(t *testing.T) {
		read, err := client.Projects.Read(ctx, p.ID)
		require.NoError(t, err)
		require.NotNil(t, read.DefaultHYOKConfiguration)
		assert.Equal(t, hyokConfigID, read.DefaultHYOKConfiguration.ID)
	})

	t.Run("update without the field keeps the default", func(t *testing.T) {
		updated, err := client.Projects.Update(ctx, p.ID, ProjectUpdateOptions{
			Description: String("unchanged hyok default"),
		})
		require.NoError(t, err)
		require.NotNil(t, updated.DefaultHYOKConfiguration)
		assert.Equal(t, hyokConfigID, updated.DefaultHYOKConfiguration.ID)
	})

	t.Run("explicit null clears the default", func(t *testing.T) {
		updated, err := client.Projects.Update(ctx, p.ID, ProjectUpdateOptions{
			DefaultHYOKConfigurationID: jsonapi.NewNullNullableAttr[string](),
		})
		require.NoError(t, err)
		assert.Nil(t, updated.DefaultHYOKConfiguration)

		read, err := client.Projects.Read(ctx, p.ID)
		require.NoError(t, err)
		assert.Nil(t, read.DefaultHYOKConfiguration)
	})

	t.Run("update sets the default", func(t *testing.T) {
		updated, err := client.Projects.Update(ctx, p.ID, ProjectUpdateOptions{
			DefaultHYOKConfigurationID: jsonapi.NewNullableAttrWithValue(hyokConfigID),
		})
		require.NoError(t, err)
		require.NotNil(t, updated.DefaultHYOKConfiguration)
		assert.Equal(t, hyokConfigID, updated.DefaultHYOKConfiguration.ID)
	})
}

func TestWorkspaces_HYOKConfiguration(t *testing.T) {
	skipHYOKIntegrationTests(t)

	client := testClient(t)
	ctx := context.Background()
	org := testHyokOrganization(t, client)
	hyokConfigID := testHYOKConfigurationID(t)

	w, err := client.Workspaces.Create(ctx, org.Name, WorkspaceCreateOptions{
		Name:                String(randomStringWithoutSpecialChar(t)),
		HYOKEnabled:         Bool(true),
		HYOKConfigurationID: String(hyokConfigID),
	})
	require.NoError(t, err)
	t.Cleanup(func() {
		if err := client.Workspaces.DeleteByID(ctx, w.ID); err != nil {
			t.Errorf("Error deleting workspace %s: %v", w.ID, err)
		}
	})

	require.NotNil(t, w.HYOKConfiguration)
	assert.Equal(t, hyokConfigID, w.HYOKConfiguration.ID)

	t.Run("read returns the override and re-encryption status", func(t *testing.T) {
		read, err := client.Workspaces.ReadByID(ctx, w.ID)
		require.NoError(t, err)
		require.NotNil(t, read.HYOKConfiguration)
		assert.Equal(t, hyokConfigID, read.HYOKConfiguration.ID)

		// Unset means no re-encryption is outstanding; otherwise it is
		// pending or failed.
		if read.HYOKReencryptionStatus.IsSpecified() {
			status, err := read.HYOKReencryptionStatus.Get()
			require.NoError(t, err)
			assert.Contains(t, []string{"pending", "failed"}, status)
		}
	})

	t.Run("explicit null clears the override", func(t *testing.T) {
		updated, err := client.Workspaces.UpdateByID(ctx, w.ID, WorkspaceUpdateOptions{
			HYOKConfigurationID: jsonapi.NewNullNullableAttr[string](),
		})
		require.NoError(t, err)
		assert.Nil(t, updated.HYOKConfiguration)

		read, err := client.Workspaces.ReadByID(ctx, w.ID)
		require.NoError(t, err)
		assert.Nil(t, read.HYOKConfiguration)
	})

	t.Run("update sets the override", func(t *testing.T) {
		updated, err := client.Workspaces.UpdateByID(ctx, w.ID, WorkspaceUpdateOptions{
			HYOKConfigurationID: jsonapi.NewNullableAttrWithValue(hyokConfigID),
		})
		require.NoError(t, err)
		require.NotNil(t, updated.HYOKConfiguration)
		assert.Equal(t, hyokConfigID, updated.HYOKConfiguration.ID)
	})
}
