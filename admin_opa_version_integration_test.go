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

func TestAdminOpaVersions_List(t *testing.T) {
	skipUnlessEnterprise(t)

	client := testClient(t)
	ctx := context.Background()

	t.Run("without list options", func(t *testing.T) {
		oList, err := client.Admin.OpaVersions.List(ctx, nil)
		require.NoError(t, err)

		assert.NotEmpty(t, oList.Items)
	})

	t.Run("with list options", func(t *testing.T) {
		oList, err := client.Admin.OpaVersions.List(ctx, &AdminOpaVersionsListOptions{
			ListOptions: ListOptions{
				PageNumber: 999,
				PageSize:   100,
			},
		})
		require.NoError(t, err)
		// Out of range page number, so the items should be empty
		assert.Empty(t, oList.Items)
		assert.Equal(t, 999, oList.CurrentPage)

		oList, err = client.Admin.OpaVersions.List(ctx, &AdminOpaVersionsListOptions{
			ListOptions: ListOptions{
				PageNumber: 1,
				PageSize:   100,
			},
		})
		require.NoError(t, err)
		assert.Equal(t, 1, oList.CurrentPage)
		for _, item := range oList.Items {
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
		oList, err := client.Admin.OpaVersions.List(ctx, &AdminOpaVersionsListOptions{
			Filter: "0.46.1",
		})
		require.NoError(t, err)
		assert.Equal(t, 1, len(oList.Items))

		// Query for a OPA version that does not exist
		oList, err = client.Admin.OpaVersions.List(ctx, &AdminOpaVersionsListOptions{
			Filter: "1000.1000.42",
		})
		require.NoError(t, err)
		assert.Empty(t, oList.Items)
	})

	t.Run("with search version query string", func(t *testing.T) {
		searchVersion := "0.46.1"
		oList, err := client.Admin.OpaVersions.List(ctx, &AdminOpaVersionsListOptions{
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

func TestAdminOpaVersions_CreateDelete(t *testing.T) {
	skipUnlessEnterprise(t)

	client := testClient(t)
	ctx := context.Background()
	version := genSafeRandomPolicyVersion()

	t.Run("with valid options", func(t *testing.T) {
		opts := AdminOpaVersionCreateOptions{
			Version:          String(version),
			URL:              String("https://www.hashicorp.com"),
			Sha:              String(genSha(t)),
			Deprecated:       Bool(true),
			DeprecatedReason: String("Test Reason"),
			Official:         Bool(false),
			Enabled:          Bool(false),
			Beta:             Bool(false),
		}
		ov, err := client.Admin.OpaVersions.Create(ctx, opts)
		require.NoError(t, err)

		defer func() {
			deleteErr := client.Admin.OpaVersions.Delete(ctx, ov.ID)
			require.NoError(t, deleteErr)
		}()

		assert.Equal(t, *opts.Version, ov.Version)
		assert.Equal(t, *opts.URL, ov.URL)
		assert.Equal(t, *opts.Sha, ov.Sha)
		assert.Equal(t, *opts.Official, ov.Official)
		assert.Equal(t, *opts.Deprecated, ov.Deprecated)
		assert.Equal(t, *opts.DeprecatedReason, *ov.DeprecatedReason)
		assert.Equal(t, *opts.Enabled, ov.Enabled)
		assert.Equal(t, *opts.Beta, ov.Beta)
	})

	t.Run("with only required options", func(t *testing.T) {
		version := genSafeRandomPolicyVersion()
		opts := AdminOpaVersionCreateOptions{
			Version: String(version),
			URL:     String("https://www.hashicorp.com"),
			Sha:     String(genSha(t)),
		}
		ov, err := client.Admin.OpaVersions.Create(ctx, opts)
		require.NoError(t, err)

		defer func() {
			deleteErr := client.Admin.OpaVersions.Delete(ctx, ov.ID)
			require.NoError(t, deleteErr)
		}()

		assert.Equal(t, *opts.Version, ov.Version)
		assert.Equal(t, *opts.URL, ov.URL)
		assert.Equal(t, *opts.Sha, ov.Sha)
		assert.Equal(t, false, ov.Official)
		assert.Equal(t, false, ov.Deprecated)
		assert.Nil(t, ov.DeprecatedReason)
		assert.Equal(t, true, ov.Enabled)
		assert.Equal(t, false, ov.Beta)
	})

	t.Run("with empty options", func(t *testing.T) {
		_, err := client.Admin.OpaVersions.Create(ctx, AdminOpaVersionCreateOptions{})
		require.Equal(t, err, ErrRequiredOpaVerCreateOps)
	})
}

func TestAdminOpaVersions_ReadUpdate(t *testing.T) {
	skipUnlessEnterprise(t)

	client := testClient(t)
	ctx := context.Background()

	t.Run("reads and updates", func(t *testing.T) {
		version := genSafeRandomPolicyVersion()
		opts := AdminOpaVersionCreateOptions{
			Version:          String(version),
			URL:              String("https://www.hashicorp.com"),
			Sha:              String(genSha(t)),
			Official:         Bool(false),
			Deprecated:       Bool(true),
			DeprecatedReason: String("Test Reason"),
			Enabled:          Bool(false),
			Beta:             Bool(false),
		}
		ov, err := client.Admin.OpaVersions.Create(ctx, opts)
		require.NoError(t, err)
		id := ov.ID

		defer func() {
			deleteErr := client.Admin.OpaVersions.Delete(ctx, id)
			require.NoError(t, deleteErr)
		}()

		ov, err = client.Admin.OpaVersions.Read(ctx, id)
		require.NoError(t, err)

		assert.Equal(t, *opts.Version, ov.Version)
		assert.Equal(t, *opts.URL, ov.URL)
		assert.Equal(t, *opts.Sha, ov.Sha)
		assert.Equal(t, *opts.Official, ov.Official)
		assert.Equal(t, *opts.Deprecated, ov.Deprecated)
		assert.Equal(t, *opts.DeprecatedReason, *ov.DeprecatedReason)
		assert.Equal(t, *opts.Enabled, ov.Enabled)
		assert.Equal(t, *opts.Beta, ov.Beta)

		updateVersion := genSafeRandomPolicyVersion()
		updateURL := "https://app.terraform.io/"
		updateOpts := AdminOpaVersionUpdateOptions{
			Version:    String(updateVersion),
			URL:        String(updateURL),
			Deprecated: Bool(false),
		}

		ov, err = client.Admin.OpaVersions.Update(ctx, id, updateOpts)
		require.NoError(t, err)

		assert.Equal(t, updateVersion, ov.Version)
		assert.Equal(t, updateURL, ov.URL)
		assert.Equal(t, *opts.Sha, ov.Sha)
		assert.Equal(t, *opts.Official, ov.Official)
		assert.Equal(t, *updateOpts.Deprecated, ov.Deprecated)
		assert.Equal(t, *opts.Enabled, ov.Enabled)
		assert.Equal(t, *opts.Beta, ov.Beta)
	})

	t.Run("with non-existent OPA version", func(t *testing.T) {
		randomID := "random-id"
		_, err := client.Admin.OpaVersions.Read(ctx, randomID)
		require.Error(t, err)
	})
}
