package tfe

import (
	"context"
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
}

func TestAdminTerraformVersions_CreateDelete(t *testing.T) {
	skipIfCloud(t)

	client := testClient(t)
	ctx := context.Background()

	t.Run("with valid options", func(t *testing.T) {
		opts := AdminTerraformVersionCreateOptions{
			Type:     String("terraform-version"),
			Version:  String("1.1.1"),
			URL:      String("https://www.hashicorp.com"),
			Sha:      String(genSha("secret", "data")),
			Official: Bool(false),
			Enabled:  Bool(false),
			Beta:     Bool(false),
		}
		tfv, err := client.Admin.TerraformVersions.Create(ctx, opts)
		require.NoError(t, err)

		defer func() {
			err = client.Admin.TerraformVersions.Delete(ctx, tfv.ID)
			require.NoError(t, err)
		}()

		assert.Equal(t, *opts.Version, tfv.Version)
		assert.Equal(t, *opts.URL, tfv.URL)
		assert.Equal(t, *opts.Sha, tfv.Sha)
		assert.Equal(t, *opts.Official, tfv.Official)
		assert.Equal(t, *opts.Enabled, tfv.Enabled)
		assert.Equal(t, *opts.Beta, tfv.Beta)
	})

	t.Run("with invalid options", func(t *testing.T) {
		opts := AdminTerraformVersionCreateOptions{
			Type: String("not-terraform"),
		}

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
			Type:     String("terraform-version"),
			Version:  String("1.1.1"),
			URL:      String("https://www.hashicorp.com"),
			Sha:      String(genSha("secret", "data")),
			Official: Bool(false),
			Enabled:  Bool(false),
			Beta:     Bool(false),
		}
		tfv, err := client.Admin.TerraformVersions.Create(ctx, opts)
		require.NoError(t, err)
		id := tfv.ID

		defer func() {
			err = client.Admin.TerraformVersions.Delete(ctx, id)
			require.NoError(t, err)
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
			Type:    String("terraform-version"),
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

	t.Run("with non existant terraform version", func(t *testing.T) {
		randomID := "random-id"
		_, err := client.Admin.TerraformVersions.Read(ctx, randomID)
		require.Error(t, err)
	})
}
