// Copyright IBM Corp. 2018, 2025
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
	t.Parallel()
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
			assert.NotNil(t, item.Archs)
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
	t.Parallel()
	skipUnlessEnterprise(t)
	skipUnlessBeta(t)

	client := testClient(t)
	ctx := context.Background()
	amd64Sha := genSha(t)
	url := "https://www.hashicorp.com"

	t.Run("with valid options including top level url & sha and archs", func(t *testing.T) {
		opts := AdminTerraformVersionCreateOptions{
			Version:          String(genSafeRandomTerraformVersion()),
			URL:              String(url),
			Sha:              &amd64Sha,
			Deprecated:       Bool(true),
			DeprecatedReason: String("Test Reason"),
			Official:         Bool(false),
			Enabled:          Bool(false),
			Beta:             Bool(false),
			Archs: []*ToolVersionArchitecture{
				{
					URL:  url,
					Sha:  amd64Sha,
					OS:   linux,
					Arch: amd64,
				},
				{
					URL:  url,
					Sha:  genSha(t),
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
		assert.Equal(t, *opts.URL, tfv.URL)
		assert.Equal(t, *opts.Sha, tfv.Sha)
		assert.Equal(t, *opts.Official, tfv.Official)
		assert.Equal(t, *opts.Deprecated, tfv.Deprecated)
		assert.Equal(t, *opts.DeprecatedReason, *tfv.DeprecatedReason)
		assert.Equal(t, *opts.Enabled, tfv.Enabled)
		assert.Equal(t, *opts.Beta, tfv.Beta)
		for i, arch := range opts.Archs {
			assert.Equal(t, arch.URL, tfv.Archs[i].URL)
			assert.Equal(t, arch.Sha, tfv.Archs[i].Sha)
			assert.Equal(t, arch.OS, tfv.Archs[i].OS)
			assert.Equal(t, arch.Arch, tfv.Archs[i].Arch)
		}
	})

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
					URL:  url,
					Sha:  amd64Sha,
					OS:   linux,
					Arch: amd64,
				},
				{
					URL:  url,
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
		assert.Equal(t, *opts.Official, tfv.Official)
		assert.Equal(t, *opts.Deprecated, tfv.Deprecated)
		assert.Equal(t, *opts.DeprecatedReason, *tfv.DeprecatedReason)
		assert.Equal(t, *opts.Enabled, tfv.Enabled)
		assert.Equal(t, *opts.Beta, tfv.Beta)
		assert.Equal(t, len(opts.Archs), len(tfv.Archs))
		for i, arch := range opts.Archs {
			assert.Equal(t, arch.URL, tfv.Archs[i].URL)
			assert.Equal(t, arch.Sha, tfv.Archs[i].Sha)
			assert.Equal(t, arch.OS, tfv.Archs[i].OS)
			assert.Equal(t, arch.Arch, tfv.Archs[i].Arch)
		}
	})

	t.Run("with valid options including url and sha", func(t *testing.T) {
		opts := AdminTerraformVersionCreateOptions{
			Version:          String(genSafeRandomTerraformVersion()),
			URL:              &url,
			Sha:              &amd64Sha,
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
		assert.Equal(t, opts.DeprecatedReason, tfv.DeprecatedReason)
		assert.Equal(t, *opts.Enabled, tfv.Enabled)
		assert.Equal(t, *opts.Beta, tfv.Beta)
		assert.Equal(t, 1, len(tfv.Archs))
		assert.Equal(t, *opts.URL, tfv.Archs[0].URL)
		assert.Equal(t, *opts.Sha, tfv.Archs[0].Sha)
		assert.Equal(t, linux, tfv.Archs[0].OS)
		assert.Equal(t, amd64, tfv.Archs[0].Arch)
	})

	t.Run("with only required options including tool version url and sha", func(t *testing.T) {
		version := genSafeRandomTerraformVersion()
		opts := AdminTerraformVersionCreateOptions{
			Version: String(version),
			URL:     &url,
			Sha:     &amd64Sha,
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
		assert.Equal(t, 1, len(tfv.Archs))
		assert.Equal(t, *opts.URL, tfv.Archs[0].URL)
		assert.Equal(t, *opts.Sha, tfv.Archs[0].Sha)
		assert.Equal(t, linux, tfv.Archs[0].OS)
		assert.Equal(t, amd64, tfv.Archs[0].Arch)
	})

	t.Run("with only required options including archs", func(t *testing.T) {
		version := genSafeRandomTerraformVersion()
		opts := AdminTerraformVersionCreateOptions{
			Version: String(version),
			Archs: []*ToolVersionArchitecture{
				{
					URL:  url,
					Sha:  amd64Sha,
					OS:   linux,
					Arch: amd64,
				},
				{
					URL:  url,
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
	t.Parallel()
	skipUnlessEnterprise(t)

	client := testClient(t)
	ctx := context.Background()
	url := "https://www.hashicorp.com"
	amd64Sha := genSha(t)

	t.Run("reads and updates", func(t *testing.T) {
		version := genSafeRandomTerraformVersion()
		opts := AdminTerraformVersionCreateOptions{
			Version:          String(version),
			URL:              String(url),
			Sha:              &amd64Sha,
			Official:         Bool(false),
			Deprecated:       Bool(true),
			DeprecatedReason: String("Test Reason"),
			Enabled:          Bool(false),
			Beta:             Bool(false),
			Archs: []*ToolVersionArchitecture{{
				URL:  url,
				Sha:  amd64Sha,
				OS:   linux,
				Arch: amd64,
			}, {
				URL:  url,
				Sha:  genSha(t),
				OS:   linux,
				Arch: arm64,
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
		assert.Equal(t, 2, len(tfv.Archs))
		assert.Equal(t, opts.Archs[0].URL, tfv.URL)
		assert.Equal(t, opts.Archs[0].Sha, tfv.Sha)
		assert.Equal(t, *opts.Official, tfv.Official)
		assert.Equal(t, *opts.Deprecated, tfv.Deprecated)
		assert.Equal(t, *opts.DeprecatedReason, *tfv.DeprecatedReason)
		assert.Equal(t, *opts.Enabled, tfv.Enabled)
		assert.Equal(t, *opts.Beta, tfv.Beta)

		updateVersion := genSafeRandomTerraformVersion()
		updateURL := "https://app.terraform.io/"
		updateSha := genSha(t)
		updateOpts := AdminTerraformVersionUpdateOptions{
			Version:    String(updateVersion),
			URL:        String(updateURL),
			Sha:        &updateSha,
			Deprecated: Bool(false),
		}

		tfv, err = client.Admin.TerraformVersions.Update(ctx, id, updateOpts)
		require.NoError(t, err)

		assert.Equal(t, updateVersion, tfv.Version)
		assert.Equal(t, updateURL, tfv.URL)
		assert.Equal(t, updateSha, tfv.Sha)
		assert.Equal(t, *opts.Official, tfv.Official)
		assert.Equal(t, *updateOpts.Deprecated, tfv.Deprecated)
		assert.Equal(t, *opts.Enabled, tfv.Enabled)
		assert.Equal(t, *opts.Beta, tfv.Beta)
		assert.Equal(t, 1, len(tfv.Archs))
		assert.Equal(t, updateURL, tfv.Archs[0].URL)
		assert.Equal(t, updateSha, tfv.Archs[0].Sha)
		assert.Equal(t, linux, tfv.Archs[0].OS)
		assert.Equal(t, amd64, tfv.Archs[0].Arch)
	})

	t.Run("update with Archs", func(t *testing.T) {
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

		updateArchOpts := AdminTerraformVersionUpdateOptions{
			Archs: []*ToolVersionArchitecture{{
				URL:  "https://www.hashicorp.com",
				Sha:  *sha,
				OS:   linux,
				Arch: arm64,
			}},
		}

		tfv, err = client.Admin.TerraformVersions.Update(ctx, id, updateArchOpts)
		require.NoError(t, err)

		assert.Equal(t, *opts.Version, tfv.Version)
		assert.Equal(t, "", tfv.URL)
		assert.Equal(t, "", tfv.Sha)
		assert.Equal(t, *opts.Official, tfv.Official)
		assert.Equal(t, *opts.Deprecated, tfv.Deprecated)
		assert.Equal(t, *opts.Enabled, tfv.Enabled)
		assert.Equal(t, *opts.Beta, tfv.Beta)
		assert.Equal(t, 1, len(tfv.Archs))
		assert.Equal(t, updateArchOpts.Archs[0].URL, tfv.Archs[0].URL)
		assert.Equal(t, updateArchOpts.Archs[0].Sha, tfv.Archs[0].Sha)
		assert.Equal(t, updateArchOpts.Archs[0].OS, tfv.Archs[0].OS)
		assert.Equal(t, updateArchOpts.Archs[0].Arch, tfv.Archs[0].Arch)
	})

	t.Run("with non-existent terraform version", func(t *testing.T) {
		randomID := "random-id"
		_, err := client.Admin.TerraformVersions.Read(ctx, randomID)
		require.Error(t, err)
	})
}
