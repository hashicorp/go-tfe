// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package tfe

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAdminTerraformVersions_List(t *testing.T) {
	skipUnlessEnterprise(t)

	client := testClient(t)
	ctx := context.Background()

	t.Run("without list options", func(t *testing.T) {
		tfList, err := client.Admin.TerraformVersions.List(ctx, nil)
		require.NoError(t, err)

		assert.NotEmpty(t, tfList.Items)
	})

	t.Run("with list options", func(t *testing.T) {
		tfList, err := client.Admin.TerraformVersions.List(ctx, &AdminTerraformVersionsListOptions{
			ListOptions: ListOptions{
				PageNumber: 999,
				PageSize:   100,
			},
		})
		require.NoError(t, err)
		// Out of range page number, so the items should be empty
		assert.Empty(t, tfList.Items)
		assert.Equal(t, 999, tfList.CurrentPage)

		tfList, err = client.Admin.TerraformVersions.List(ctx, &AdminTerraformVersionsListOptions{
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
			assert.NotNil(t, item.Deprecated)
			if item.Deprecated {
				assert.NotNil(t, item.DeprecatedReason)
			} else {
				assert.Nil(t, item.DeprecatedReason)
			}
			assert.NotNil(t, item.Enabled)
			assert.NotNil(t, item.Beta)
			assert.NotNil(t, item.Usage)
			assert.NotNil(t, item.CreatedAt)
		}
	})

	t.Run("with filter query string", func(t *testing.T) {
		tfList, err := client.Admin.TerraformVersions.List(ctx, &AdminTerraformVersionsListOptions{
			Filter: "1.0.4",
		})
		require.NoError(t, err)
		assert.Equal(t, 1, len(tfList.Items))

		// Query for a Terraform version that does not exist
		tfList, err = client.Admin.TerraformVersions.List(ctx, &AdminTerraformVersionsListOptions{
			Filter: "1000.1000.42",
		})
		require.NoError(t, err)
		assert.Empty(t, tfList.Items)
	})

	t.Run("with search version query string", func(t *testing.T) {
		searchVersion := "1.0"
		tfList, err := client.Admin.TerraformVersions.List(ctx, &AdminTerraformVersionsListOptions{
			Search: searchVersion,
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
	skipUnlessEnterprise(t)
	skipUnlessBeta(t)

	client := testClient(t)
	ctx := context.Background()

	t.Run("with valid options including archs", func(t *testing.T) {
		opts := AdminTerraformVersionCreateOptions{
			Version:          String(genSafeRandomTerraformVersion()),
			Deprecated:       Bool(true),
			DeprecatedReason: String("Test Reason"),
			Official:         Bool(false),
			Enabled:          Bool(false),
			Beta:             Bool(false),
			Archs: []*ToolVersionArchitecture{
				{
					URL:  "https://www.hashicorp.com",
					Sha:  *String(genSha(t)),
					OS:   linux,
					Arch: amd64,
				},
				{
					URL:  "https://www.hashicorp.com",
					Sha:  *String(genSha(t)),
					OS:   linux,
					Arch: arm64,
				}},
		}
		tfv, err := client.Admin.TerraformVersions.Create(ctx, opts)
		require.NoError(t, err)

		defer func() {
			deleteErr := client.Admin.TerraformVersions.Delete(ctx, tfv.ID)
			require.NoError(t, deleteErr)
		}()

		assert.Equal(t, *opts.Version, tfv.Version)
		assert.ElementsMatch(t, opts.Archs, tfv.Archs)
		assert.Equal(t, *opts.Official, tfv.Official)
		assert.Equal(t, *opts.Deprecated, tfv.Deprecated)
		assert.Equal(t, *opts.DeprecatedReason, *tfv.DeprecatedReason)
		assert.Equal(t, *opts.Enabled, tfv.Enabled)
		assert.Equal(t, *opts.Beta, tfv.Beta)
	})

	t.Run("with valid options, url, and sha", func(t *testing.T) {
		opts := AdminTerraformVersionCreateOptions{
			Version:          String(genSafeRandomTerraformVersion()),
			URL:              String("https://www.hashicorp.com"),
			Sha:              String(genSha(t)),
			Deprecated:       Bool(true),
			DeprecatedReason: String("Test Reason"),
			Official:         Bool(false),
			Enabled:          Bool(false),
			Beta:             Bool(false),
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
		assert.Equal(t, *opts.Deprecated, tfv.Deprecated)
		assert.Equal(t, *opts.DeprecatedReason, *tfv.DeprecatedReason)
		assert.Equal(t, *opts.Enabled, tfv.Enabled)
		assert.Equal(t, *opts.Beta, tfv.Beta)
	})

	t.Run("with only required options including tool version url and sha", func(t *testing.T) {
		version := genSafeRandomTerraformVersion()
		opts := AdminTerraformVersionCreateOptions{
			Version: String(version),
			URL:     String("https://www.hashicorp.com"),
			Sha:     String(genSha(t)),
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
		assert.Equal(t, false, tfv.Deprecated)
		assert.Nil(t, tfv.DeprecatedReason)
		assert.Equal(t, true, tfv.Enabled)
		assert.Equal(t, false, tfv.Beta)
	})

	t.Run("with only required options including archs", func(t *testing.T) {
		version := genSafeRandomTerraformVersion()
		opts := AdminTerraformVersionCreateOptions{
			Version: String(version),
			Archs: []*ToolVersionArchitecture{
				{
					URL:  "https://www.hashicorp.com",
					Sha:  *String(genSha(t)),
					OS:   linux,
					Arch: amd64,
				},
				{
					URL:  "https://www.hashicorp.com",
					Sha:  *String(genSha(t)),
					OS:   linux,
					Arch: arm64,
				}},
		}
		tfv, err := client.Admin.TerraformVersions.Create(ctx, opts)
		require.NoError(t, err)

		defer func() {
			deleteErr := client.Admin.TerraformVersions.Delete(ctx, tfv.ID)
			require.NoError(t, deleteErr)
		}()

		assert.Equal(t, *opts.Version, tfv.Version)
		assert.ElementsMatch(t, opts.Archs, tfv.Archs)
		assert.Equal(t, false, tfv.Official)
		assert.Equal(t, false, tfv.Deprecated)
		assert.Nil(t, tfv.DeprecatedReason)
		assert.Equal(t, true, tfv.Enabled)
		assert.Equal(t, false, tfv.Beta)
	})

	t.Run("with empty options", func(t *testing.T) {
		_, err := client.Admin.TerraformVersions.Create(ctx, AdminTerraformVersionCreateOptions{})
		require.Equal(t, err, ErrRequiredTFVerCreateOps)
	})
}

func TestAdminTerraformVersions_ReadUpdate(t *testing.T) {
	skipUnlessEnterprise(t)

	client := testClient(t)
	ctx := context.Background()

	t.Run("reads and updates", func(t *testing.T) {
		version := genSafeRandomTerraformVersion()
		sha := String(genSha(t))
		opts := AdminTerraformVersionCreateOptions{
			Version:          String(version),
			URL:              String("https://www.hashicorp.com"),
			Sha:              String(genSha(t)),
			Official:         Bool(false),
			Deprecated:       Bool(true),
			DeprecatedReason: String("Test Reason"),
			Enabled:          Bool(false),
			Beta:             Bool(false),
			Archs: []*ToolVersionArchitecture{{
				URL:  "https://www.hashicorp.com",
				Sha:  *sha,
				OS:   linux,
				Arch: amd64,
			}},
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
		assert.Equal(t, opts.Archs[0].URL, tfv.URL)
		assert.Equal(t, opts.Archs[0].Sha, tfv.Sha)
		assert.ElementsMatch(t, opts.Archs, tfv.Archs)
		assert.Equal(t, *opts.Official, tfv.Official)
		assert.Equal(t, *opts.Deprecated, tfv.Deprecated)
		assert.Equal(t, *opts.DeprecatedReason, *tfv.DeprecatedReason)
		assert.Equal(t, *opts.Enabled, tfv.Enabled)
		assert.Equal(t, *opts.Beta, tfv.Beta)

		updateVersion := genSafeRandomTerraformVersion()
		updateURL := "https://app.terraform.io/"
		updateOpts := AdminTerraformVersionUpdateOptions{
			Version:    String(updateVersion),
			URL:        String(updateURL),
			Sha:        sha,
			Deprecated: Bool(false),
		}

		tfv, err = client.Admin.TerraformVersions.Update(ctx, id, updateOpts)
		require.NoError(t, err)

		assert.Equal(t, updateVersion, tfv.Version)
		assert.Equal(t, updateURL, tfv.URL)
		assert.Equal(t, *opts.Sha, tfv.Sha)
		assert.Equal(t, updateURL, tfv.Archs[0].URL)
		assert.Equal(t, *opts.Sha, tfv.Archs[0].Sha)
		assert.Equal(t, *opts.Official, tfv.Official)
		assert.Equal(t, *updateOpts.Deprecated, tfv.Deprecated)
		assert.Equal(t, *opts.Enabled, tfv.Enabled)
		assert.Equal(t, *opts.Beta, tfv.Beta)

		updateOpts = AdminTerraformVersionUpdateOptions{
			Archs: []*ToolVersionArchitecture{
				{
					URL:  "https://www.hashicorp.com/update",
					Sha:  *sha,
					OS:   linux,
					Arch: amd64,
				},
				{
					URL:  "https://www.hashicorp.com/update/arm64",
					Sha:  *sha,
					OS:   linux,
					Arch: arm64,
				},
			},
		}

		tfv, err = client.Admin.TerraformVersions.Update(ctx, id, updateOpts)

		require.NoError(t, err)

		assert.Equal(t, "https://www.hashicorp.com/update", tfv.URL)
		assert.Equal(t, *opts.Sha, tfv.Sha)
		assert.ElementsMatch(t, updateOpts.Archs, tfv.Archs)
	})

	t.Run("with non-existent terraform version", func(t *testing.T) {
		randomID := "random-id"
		_, err := client.Admin.TerraformVersions.Read(ctx, randomID)
		require.Error(t, err)
	})
}
