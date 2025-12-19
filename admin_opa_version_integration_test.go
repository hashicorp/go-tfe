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

func TestAdminOPAVersions_List(t *testing.T) {
	t.Parallel()
	skipUnlessEnterprise(t)

	client := testClient(t)
	ctx := context.Background()

	t.Run("without list options", func(t *testing.T) {
		oList, err := client.Admin.OPAVersions.List(ctx, nil)
		require.NoError(t, err)

		assert.NotEmpty(t, oList.Items)
	})

	t.Run("with list options", func(t *testing.T) {
		oList, err := client.Admin.OPAVersions.List(ctx, &AdminOPAVersionsListOptions{
			ListOptions: ListOptions{
				PageNumber: 999,
				PageSize:   100,
			},
		})
		require.NoError(t, err)
		// Out of range page number, so the items should be empty
		assert.Empty(t, oList.Items)
		assert.Equal(t, 999, oList.CurrentPage)

		oList, err = client.Admin.OPAVersions.List(ctx, &AdminOPAVersionsListOptions{
			ListOptions: ListOptions{
				PageNumber: 1,
				PageSize:   100,
			},
		})
		require.NoError(t, err)
		assert.Equal(t, 1, oList.CurrentPage)
		for _, item := range oList.Items {
			assert.NotNil(t, item.ID)
			assert.NotEmpty(t, item.Version)
			assert.NotEmpty(t, item.URL)
			assert.NotEmpty(t, item.SHA)
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
			assert.NotEmpty(t, item.Archs)
		}
	})

	t.Run("with filter query string", func(t *testing.T) {
		oList, err := client.Admin.OPAVersions.List(ctx, &AdminOPAVersionsListOptions{
			Filter: "0.59.0",
		})
		require.NoError(t, err)
		assert.Equal(t, 1, len(oList.Items))

		// Query for a OPA version that does not exist
		oList, err = client.Admin.OPAVersions.List(ctx, &AdminOPAVersionsListOptions{
			Filter: "1000.1000.42",
		})
		require.NoError(t, err)
		assert.Empty(t, oList.Items)
	})

	t.Run("with search version query string", func(t *testing.T) {
		searchVersion := "0.59.0"
		oList, err := client.Admin.OPAVersions.List(ctx, &AdminOPAVersionsListOptions{
			Search: searchVersion,
		})
		require.NoError(t, err)
		assert.NotEmpty(t, oList.Items)

		t.Run("ensure each version matches substring", func(t *testing.T) {
			for _, item := range oList.Items {
				assert.Equal(t, true, strings.Contains(item.Version, searchVersion))
			}
		})
	})
}

func TestAdminOPAVersions_CreateDelete(t *testing.T) {
	t.Parallel()
	skipUnlessEnterprise(t)

	client := testClient(t)
	ctx := context.Background()
	version := createAdminOPAVersion()
	url := "https://www.hashicorp.com"
	amd64Sha := *String(genSha(t))

	t.Run("with valid options including top level url & sha and archs", func(t *testing.T) {
		opts := AdminOPAVersionCreateOptions{
			Version:          version,
			Deprecated:       Bool(true),
			DeprecatedReason: String("Test Reason"),
			Official:         Bool(false),
			Enabled:          Bool(false),
			Beta:             Bool(false),
			URL:              url,
			SHA:              amd64Sha,

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
		ov, err := client.Admin.OPAVersions.Create(ctx, opts)
		require.NoError(t, err)

		defer func() {
			deleteErr := client.Admin.OPAVersions.Delete(ctx, ov.ID)
			require.NoError(t, deleteErr)
		}()

		assert.Equal(t, opts.Version, ov.Version)
		assert.Equal(t, *opts.Official, ov.Official)
		assert.Equal(t, *opts.Deprecated, ov.Deprecated)
		assert.Equal(t, *opts.DeprecatedReason, *ov.DeprecatedReason)
		assert.Equal(t, *opts.Enabled, ov.Enabled)
		assert.Equal(t, *opts.Beta, ov.Beta)
		assert.Equal(t, len(opts.Archs), len(ov.Archs))
		assert.Equal(t, opts.URL, ov.URL)
		assert.Equal(t, opts.SHA, ov.SHA)
		for i, arch := range opts.Archs {
			assert.Equal(t, arch.URL, ov.Archs[i].URL)
			assert.Equal(t, arch.Sha, ov.Archs[i].Sha)
			assert.Equal(t, arch.OS, ov.Archs[i].OS)
			assert.Equal(t, arch.Arch, ov.Archs[i].Arch)
		}
	})

	t.Run("with valid options including archs", func(t *testing.T) {
		version = createAdminOPAVersion()
		opts := AdminOPAVersionCreateOptions{
			Version:          version,
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

		ov, err := client.Admin.OPAVersions.Create(ctx, opts)
		require.NoError(t, err)
		defer func() {
			deleteErr := client.Admin.OPAVersions.Delete(ctx, ov.ID)
			require.NoError(t, deleteErr)
		}()

		assert.Equal(t, opts.Version, ov.Version)
		assert.Equal(t, *opts.Official, ov.Official)
		assert.Equal(t, *opts.Deprecated, ov.Deprecated)
		assert.Equal(t, *opts.DeprecatedReason, *ov.DeprecatedReason)
		assert.Equal(t, *opts.Enabled, ov.Enabled)
		assert.Equal(t, *opts.Beta, ov.Beta)
		assert.Equal(t, len(opts.Archs), len(ov.Archs))
		for i, arch := range opts.Archs {
			assert.Equal(t, arch.URL, ov.Archs[i].URL)
			assert.Equal(t, arch.Sha, ov.Archs[i].Sha)
			assert.Equal(t, arch.OS, ov.Archs[i].OS)
			assert.Equal(t, arch.Arch, ov.Archs[i].Arch)
		}
	})

	t.Run("with valid options including, url, and sha", func(t *testing.T) {
		opts := AdminOPAVersionCreateOptions{
			Version:          version,
			URL:              url,
			SHA:              genSha(t),
			Deprecated:       Bool(true),
			DeprecatedReason: String("Test Reason"),
			Official:         Bool(false),
			Enabled:          Bool(false),
			Beta:             Bool(false),
		}
		ov, err := client.Admin.OPAVersions.Create(ctx, opts)
		require.NoError(t, err)

		defer func() {
			deleteErr := client.Admin.OPAVersions.Delete(ctx, ov.ID)
			require.NoError(t, deleteErr)
		}()

		assert.Equal(t, opts.Version, ov.Version)
		assert.Equal(t, opts.URL, ov.URL)
		assert.Equal(t, opts.SHA, ov.SHA)
		assert.Equal(t, *opts.Official, ov.Official)
		assert.Equal(t, *opts.Deprecated, ov.Deprecated)
		assert.Equal(t, *opts.DeprecatedReason, *ov.DeprecatedReason)
		assert.Equal(t, *opts.Enabled, ov.Enabled)
		assert.Equal(t, *opts.Beta, ov.Beta)
		assert.Equal(t, 1, len(ov.Archs))
		assert.Equal(t, opts.URL, ov.Archs[0].URL)
		assert.Equal(t, opts.SHA, ov.Archs[0].Sha)
		assert.Equal(t, linux, ov.Archs[0].OS)
		assert.Equal(t, amd64, ov.Archs[0].Arch)
	})

	t.Run("with only required options including tool version url and sha", func(t *testing.T) {
		version = createAdminOPAVersion()
		opts := AdminOPAVersionCreateOptions{
			Version: version,
			URL:     "https://www.hashicorp.com",
			SHA:     genSha(t),
		}
		ov, err := client.Admin.OPAVersions.Create(ctx, opts)
		require.NoError(t, err)

		defer func() {
			deleteErr := client.Admin.OPAVersions.Delete(ctx, ov.ID)
			require.NoError(t, deleteErr)
		}()

		assert.Equal(t, opts.Version, ov.Version)
		assert.Equal(t, opts.URL, ov.URL)
		assert.Equal(t, opts.SHA, ov.SHA)
		assert.Equal(t, false, ov.Official)
		assert.Equal(t, false, ov.Deprecated)
		assert.Nil(t, ov.DeprecatedReason)
		assert.Equal(t, true, ov.Enabled)
		assert.Equal(t, false, ov.Beta)
		assert.Equal(t, 1, len(ov.Archs))
		assert.Equal(t, opts.URL, ov.Archs[0].URL)
		assert.Equal(t, opts.SHA, ov.Archs[0].Sha)
		assert.Equal(t, linux, ov.Archs[0].OS)
		assert.Equal(t, amd64, ov.Archs[0].Arch)
	})

	t.Run("with only required options including archs", func(t *testing.T) {
		version = createAdminOPAVersion()
		opts := AdminOPAVersionCreateOptions{
			Version: version,
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
		ov, err := client.Admin.OPAVersions.Create(ctx, opts)
		require.NoError(t, err)

		defer func() {
			deleteErr := client.Admin.OPAVersions.Delete(ctx, ov.ID)
			require.NoError(t, deleteErr)
		}()

		assert.Equal(t, opts.Version, ov.Version)
		assert.Equal(t, false, ov.Official)
		assert.Equal(t, false, ov.Deprecated)
		assert.Nil(t, ov.DeprecatedReason)
		assert.Equal(t, true, ov.Enabled)
		assert.Equal(t, false, ov.Beta)
		assert.Equal(t, len(opts.Archs), len(ov.Archs))
		for i, arch := range opts.Archs {
			assert.Equal(t, arch.URL, ov.Archs[i].URL)
			assert.Equal(t, arch.Sha, ov.Archs[i].Sha)
			assert.Equal(t, arch.OS, ov.Archs[i].OS)
			assert.Equal(t, arch.Arch, ov.Archs[i].Arch)
		}
	})

	t.Run("with empty options", func(t *testing.T) {
		_, err := client.Admin.OPAVersions.Create(ctx, AdminOPAVersionCreateOptions{})
		require.Equal(t, err, ErrRequiredOPAVerCreateOps)
	})
}

func TestAdminOPAVersions_ReadUpdate(t *testing.T) {
	t.Parallel()
	skipUnlessEnterprise(t)

	client := testClient(t)
	ctx := context.Background()

	t.Run("reads and updates", func(t *testing.T) {
		version := createAdminOPAVersion()
		sha := String(genSha(t))
		opts := AdminOPAVersionCreateOptions{
			Version:          version,
			URL:              "https://www.hashicorp.com",
			SHA:              genSha(t),
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
		ov, err := client.Admin.OPAVersions.Create(ctx, opts)
		require.NoError(t, err)
		id := ov.ID

		defer func() {
			deleteErr := client.Admin.OPAVersions.Delete(ctx, id)
			require.NoError(t, deleteErr)
		}()

		ov, err = client.Admin.OPAVersions.Read(ctx, id)
		require.NoError(t, err)

		assert.Equal(t, opts.Version, ov.Version)
		assert.Equal(t, opts.Archs[0].URL, ov.URL)
		assert.Equal(t, opts.Archs[0].Sha, ov.SHA)
		assert.Equal(t, *opts.Official, ov.Official)
		assert.Equal(t, *opts.Deprecated, ov.Deprecated)
		assert.Equal(t, *opts.DeprecatedReason, *ov.DeprecatedReason)
		assert.Equal(t, *opts.Enabled, ov.Enabled)
		assert.Equal(t, *opts.Beta, ov.Beta)
		assert.Equal(t, len(opts.Archs), len(ov.Archs))
		for i, arch := range opts.Archs {
			assert.Equal(t, arch.URL, ov.Archs[i].URL)
			assert.Equal(t, arch.Sha, ov.Archs[i].Sha)
			assert.Equal(t, arch.OS, ov.Archs[i].OS)
			assert.Equal(t, arch.Arch, ov.Archs[i].Arch)
		}

		updateVersion := createAdminOPAVersion()
		updateURL := "https://app.terraform.io/"
		updateOpts := AdminOPAVersionUpdateOptions{
			Version:    String(updateVersion),
			URL:        String(updateURL),
			Deprecated: Bool(false),
		}

		ov, err = client.Admin.OPAVersions.Update(ctx, id, updateOpts)
		require.NoError(t, err)

		assert.Equal(t, updateVersion, ov.Version)
		assert.Equal(t, updateURL, ov.URL)
		assert.Equal(t, opts.SHA, ov.SHA)
		assert.Equal(t, *opts.Official, ov.Official)
		assert.Equal(t, *updateOpts.Deprecated, ov.Deprecated)
		assert.Equal(t, *opts.Enabled, ov.Enabled)
		assert.Equal(t, *opts.Beta, ov.Beta)
		assert.Equal(t, len(opts.Archs), len(ov.Archs))
		assert.Equal(t, *updateOpts.URL, ov.Archs[0].URL)
		assert.Equal(t, opts.Archs[0].Sha, ov.Archs[0].Sha)
		assert.Equal(t, opts.Archs[0].OS, ov.Archs[0].OS)
		assert.Equal(t, opts.Archs[0].Arch, ov.Archs[0].Arch)
	})

	t.Run("update with Archs", func(t *testing.T) {
		version := genSafeRandomTerraformVersion()
		sha := String(genSha(t))
		opts := AdminOPAVersionCreateOptions{
			Version:          *String(version),
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
		ov, err := client.Admin.OPAVersions.Create(ctx, opts)
		require.NoError(t, err)
		id := ov.ID

		defer func() {
			deleteErr := client.Admin.OPAVersions.Delete(ctx, id)
			require.NoError(t, deleteErr)
		}()

		updateArchOpts := AdminOPAVersionUpdateOptions{
			Archs: []*ToolVersionArchitecture{{
				URL:  "https://www.hashicorp.com",
				Sha:  *sha,
				OS:   linux,
				Arch: arm64,
			}},
		}

		ov, err = client.Admin.OPAVersions.Update(ctx, id, updateArchOpts)
		require.NoError(t, err)

		assert.Equal(t, opts.Version, ov.Version)
		assert.Equal(t, "", ov.URL)
		assert.Equal(t, "", ov.SHA)
		assert.Equal(t, *opts.Official, ov.Official)
		assert.Equal(t, *opts.Deprecated, ov.Deprecated)
		assert.Equal(t, *opts.Enabled, ov.Enabled)
		assert.Equal(t, *opts.Beta, ov.Beta)
		assert.Equal(t, len(ov.Archs), 1)
		assert.Equal(t, updateArchOpts.Archs[0].URL, ov.Archs[0].URL)
		assert.Equal(t, updateArchOpts.Archs[0].Sha, ov.Archs[0].Sha)
		assert.Equal(t, updateArchOpts.Archs[0].OS, ov.Archs[0].OS)
		assert.Equal(t, updateArchOpts.Archs[0].Arch, ov.Archs[0].Arch)
	})

	t.Run("with non-existent OPA version", func(t *testing.T) {
		randomID := "random-id"
		_, err := client.Admin.OPAVersions.Read(ctx, randomID)
		require.Error(t, err)
	})
}
