//go:build integration
// +build integration

package tfe

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAdminTerraformVersions_List(t *testing.T) {
	skipIfCloud(t)

	client := testClient(t)
	ctx := context.Background()

	t.Run("without list options", func(t *testing.T) {
		tfList, err := client.Admin.TerraformVersions.List(ctx, AdminTerraformVersionsListOptions{})
		require.NoError(t, err)

		assert.NotEmpty(t, tfList.Items)
	})

	t.Run("with list options", func(t *testing.T) {
		tfList, err := client.Admin.TerraformVersions.List(ctx, AdminTerraformVersionsListOptions{
			ListOptions: ListOptions{
				PageNumber: 999,
				PageSize:   100,
			},
		})
		require.NoError(t, err)
		// Out of range page number, so the items should be empty
		assert.Empty(t, tfList.Items)
		assert.Equal(t, 999, tfList.CurrentPage)

		tfList, err = client.Admin.TerraformVersions.List(ctx, AdminTerraformVersionsListOptions{
			ListOptions: ListOptions{
				PageNumber: 1,
				PageSize:   100,
			},
		})
		require.NoError(t, err)
		assert.Equal(t, 1, tfList.CurrentPage)
		for _, item := range tfList.Items {
			assert.NotNil(t, item.ID)
			assert.NotNil(t, item.Version)
			assert.NotNil(t, item.URL)
			assert.NotNil(t, item.Sha)
			assert.NotNil(t, item.Official)
			assert.NotNil(t, item.Enabled)
			assert.NotNil(t, item.Beta)
			assert.NotNil(t, item.Usage)
			assert.NotNil(t, item.CreatedAt)
		}
	})

	t.Run("with version query string", func(t *testing.T) {
		tfList, err := client.Admin.TerraformVersions.List(ctx, AdminTerraformVersionsListOptions{
			Version: String("1.0.4"),
		})
		require.NoError(t, err)
		assert.Equal(t, 1, len(tfList.Items))

		// Query for a Terraform version that does not exist
		tfList, err = client.Admin.TerraformVersions.List(ctx, AdminTerraformVersionsListOptions{
			Version: String("1000.1000.42"),
		})
		require.NoError(t, err)
		assert.Empty(t, tfList.Items)
	})

	t.Run("with search version query string", func(t *testing.T) {
		searchVersion := "1.0"
		tfList, err := client.Admin.TerraformVersions.List(ctx, AdminTerraformVersionsListOptions{
			Search: String(searchVersion),
		})
		require.NoError(t, err)
		assert.NotEmpty(t, tfList.Items)

		t.Run("ensure each version matches substring", func(t *testing.T) {
			for _, item := range tfList.Items {
				assert.Equal(t, true, strings.Contains(item.Version, searchVersion))
			}
		})
	})
}

func TestAdminTerraformVersions_CreateDelete(t *testing.T) {
	skipIfCloud(t)

	client := testClient(t)
	ctx := context.Background()

	t.Run("with valid options", func(t *testing.T) {
		opts := AdminTerraformVersionCreateOptions{
			Version:  String("1.1.1"),
			URL:      String("https://www.hashicorp.com"),
			Sha:      String(genSha(t, "secret", "data")),
			Official: Bool(false),
			Enabled:  Bool(false),
			Beta:     Bool(false),
		}
		tfv, err := client.Admin.TerraformVersions.Create(ctx, opts)
		require.NoError(t, err)

		defer func() {
			deleteErr := client.Admin.TerraformVersions.Delete(ctx, tfv.ID)
			require.NoError(t, deleteErr)
		}()

		assert.Equal(t, *opts.Version, tfv.Version)
		assert.Equal(t, *opts.URL, tfv.URL)
		assert.Equal(t, *opts.Sha, tfv.Sha)
		assert.Equal(t, *opts.Official, tfv.Official)
		assert.Equal(t, *opts.Enabled, tfv.Enabled)
		assert.Equal(t, *opts.Beta, tfv.Beta)
	})

	t.Run("with only required options", func(t *testing.T) {
		opts := AdminTerraformVersionCreateOptions{
			Version: String("1.1.1"),
			URL:     String("https://www.hashicorp.com"),
			Sha:     String(genSha(t, "secret", "data")),
		}
		tfv, err := client.Admin.TerraformVersions.Create(ctx, opts)
		require.NoError(t, err)

		defer func() {
			deleteErr := client.Admin.TerraformVersions.Delete(ctx, tfv.ID)
			require.NoError(t, deleteErr)
		}()

		assert.Equal(t, *opts.Version, tfv.Version)
		assert.Equal(t, *opts.URL, tfv.URL)
		assert.Equal(t, *opts.Sha, tfv.Sha)
		assert.Equal(t, false, tfv.Official)
		assert.Equal(t, true, tfv.Enabled)
		assert.Equal(t, false, tfv.Beta)
	})

	t.Run("with empty options", func(t *testing.T) {
		opts := AdminTerraformVersionCreateOptions{}

		_, err := client.Admin.TerraformVersions.Create(ctx, opts)
		require.Error(t, err)
	})
}

func TestAdminTerraformVersions_ReadUpdate(t *testing.T) {
	skipIfCloud(t)

	client := testClient(t)
	ctx := context.Background()

	t.Run("reads and updates", func(t *testing.T) {
		opts := AdminTerraformVersionCreateOptions{
			Version:  String("1.1.1"),
			URL:      String("https://www.hashicorp.com"),
			Sha:      String(genSha(t, "secret", "data")),
			Official: Bool(false),
			Enabled:  Bool(false),
			Beta:     Bool(false),
		}
		tfv, err := client.Admin.TerraformVersions.Create(ctx, opts)
		require.NoError(t, err)
		id := tfv.ID

		defer func() {
			deleteErr := client.Admin.TerraformVersions.Delete(ctx, id)
			require.NoError(t, deleteErr)
		}()

		tfv, err = client.Admin.TerraformVersions.Read(ctx, id)
		require.NoError(t, err)

		assert.Equal(t, *opts.Version, tfv.Version)
		assert.Equal(t, *opts.URL, tfv.URL)
		assert.Equal(t, *opts.Sha, tfv.Sha)
		assert.Equal(t, *opts.Official, tfv.Official)
		assert.Equal(t, *opts.Enabled, tfv.Enabled)
		assert.Equal(t, *opts.Beta, tfv.Beta)

		updateVersion := "1.1.2"
		updateURL := "https://app.terraform.io/"
		updateOpts := AdminTerraformVersionUpdateOptions{
			Version: String(updateVersion),
			URL:     String(updateURL),
		}

		tfv, err = client.Admin.TerraformVersions.Update(ctx, id, updateOpts)
		require.NoError(t, err)

		assert.Equal(t, updateVersion, tfv.Version)
		assert.Equal(t, updateURL, tfv.URL)
		assert.Equal(t, *opts.Sha, tfv.Sha)
		assert.Equal(t, *opts.Official, tfv.Official)
		assert.Equal(t, *opts.Enabled, tfv.Enabled)
		assert.Equal(t, *opts.Beta, tfv.Beta)
	})

	t.Run("with non-existent terraform version", func(t *testing.T) {
		randomID := "random-id"
		_, err := client.Admin.TerraformVersions.Read(ctx, randomID)
		require.Error(t, err)
	})
}
