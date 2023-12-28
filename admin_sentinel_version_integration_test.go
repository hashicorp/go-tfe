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

func TestAdminSentinelVersions_List(t *testing.T) {
	skipUnlessEnterprise(t)

	client := testClient(t)
	ctx := context.Background()

	t.Run("without list options", func(t *testing.T) {
		sList, err := client.Admin.SentinelVersions.List(ctx, nil)
		require.NoError(t, err)

		assert.NotEmpty(t, sList.Items)
	})

	t.Run("with list options", func(t *testing.T) {
		sList, err := client.Admin.SentinelVersions.List(ctx, &AdminSentinelVersionsListOptions{
			ListOptions: ListOptions{
				PageNumber: 999,
				PageSize:   100,
			},
		})
		require.NoError(t, err)
		// Out of range page number, so the items should be empty
		assert.Empty(t, sList.Items)
		assert.Equal(t, 999, sList.CurrentPage)

		sList, err = client.Admin.SentinelVersions.List(ctx, &AdminSentinelVersionsListOptions{
			ListOptions: ListOptions{
				PageNumber: 1,
				PageSize:   100,
			},
		})
		require.NoError(t, err)
		assert.Equal(t, 1, sList.CurrentPage)
		for _, item := range sList.Items {
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
		}
	})

	t.Run("with filter query string", func(t *testing.T) {
		sList, err := client.Admin.SentinelVersions.List(ctx, &AdminSentinelVersionsListOptions{
			Filter: "0.22.1",
		})
		require.NoError(t, err)
		assert.Equal(t, 1, len(sList.Items))

		// Query for a Sentinel version that does not exist
		sList, err = client.Admin.SentinelVersions.List(ctx, &AdminSentinelVersionsListOptions{
			Filter: "1000.1000.42",
		})
		require.NoError(t, err)
		assert.Empty(t, sList.Items)
	})

	t.Run("with search version query string", func(t *testing.T) {
		searchVersion := "0.22.1"
		sList, err := client.Admin.SentinelVersions.List(ctx, &AdminSentinelVersionsListOptions{
			Search: searchVersion,
		})
		require.NoError(t, err)
		assert.NotEmpty(t, sList.Items)

		t.Run("ensure each version matches substring", func(t *testing.T) {
			for _, item := range sList.Items {
				assert.Equal(t, true, strings.Contains(item.Version, searchVersion))
			}
		})
	})
}

func TestAdminSentinelVersions_CreateDelete(t *testing.T) {
	skipUnlessEnterprise(t)

	client := testClient(t)
	ctx := context.Background()
	version := createAdminSentinelVersion()

	t.Run("with valid options", func(t *testing.T) {
		opts := AdminSentinelVersionCreateOptions{
			Version:          version,
			URL:              "https://www.hashicorp.com",
			SHA:              genSha(t),
			Deprecated:       Bool(true),
			DeprecatedReason: String("Test Reason"),
			Official:         Bool(false),
			Enabled:          Bool(false),
			Beta:             Bool(false),
		}
		sv, err := client.Admin.SentinelVersions.Create(ctx, opts)
		require.NoError(t, err)

		defer func() {
			deleteErr := client.Admin.SentinelVersions.Delete(ctx, sv.ID)
			require.NoError(t, deleteErr)
		}()

		assert.Equal(t, opts.Version, sv.Version)
		assert.Equal(t, opts.URL, sv.URL)
		assert.Equal(t, opts.SHA, sv.SHA)
		assert.Equal(t, *opts.Official, sv.Official)
		assert.Equal(t, *opts.Deprecated, sv.Deprecated)
		assert.Equal(t, *opts.DeprecatedReason, *sv.DeprecatedReason)
		assert.Equal(t, *opts.Enabled, sv.Enabled)
		assert.Equal(t, *opts.Beta, sv.Beta)
	})

	t.Run("with only required options", func(t *testing.T) {
		version := createAdminSentinelVersion()
		opts := AdminSentinelVersionCreateOptions{
			Version: version,
			URL:     "https://www.hashicorp.com",
			SHA:     genSha(t),
		}
		sv, err := client.Admin.SentinelVersions.Create(ctx, opts)
		require.NoError(t, err)

		defer func() {
			deleteErr := client.Admin.SentinelVersions.Delete(ctx, sv.ID)
			require.NoError(t, deleteErr)
		}()

		assert.Equal(t, opts.Version, sv.Version)
		assert.Equal(t, opts.URL, sv.URL)
		assert.Equal(t, opts.SHA, sv.SHA)
		assert.Equal(t, false, sv.Official)
		assert.Equal(t, false, sv.Deprecated)
		assert.Nil(t, sv.DeprecatedReason)
		assert.Equal(t, true, sv.Enabled)
		assert.Equal(t, false, sv.Beta)
	})

	t.Run("with empty options", func(t *testing.T) {
		_, err := client.Admin.SentinelVersions.Create(ctx, AdminSentinelVersionCreateOptions{})
		require.Equal(t, err, ErrRequiredSentinelVerCreateOps)
	})
}

func TestAdminSentinelVersions_ReadUpdate(t *testing.T) {
	skipUnlessEnterprise(t)

	client := testClient(t)
	ctx := context.Background()

	t.Run("reads and updates", func(t *testing.T) {
		version := createAdminSentinelVersion()
		opts := AdminSentinelVersionCreateOptions{
			Version:          version,
			URL:              "https://www.hashicorp.com",
			SHA:              genSha(t),
			Official:         Bool(false),
			Deprecated:       Bool(true),
			DeprecatedReason: String("Test Reason"),
			Enabled:          Bool(false),
			Beta:             Bool(false),
		}
		sv, err := client.Admin.SentinelVersions.Create(ctx, opts)
		require.NoError(t, err)
		id := sv.ID

		defer func() {
			deleteErr := client.Admin.SentinelVersions.Delete(ctx, id)
			require.NoError(t, deleteErr)
		}()

		sv, err = client.Admin.SentinelVersions.Read(ctx, id)
		require.NoError(t, err)

		assert.Equal(t, opts.Version, sv.Version)
		assert.Equal(t, opts.URL, sv.URL)
		assert.Equal(t, opts.SHA, sv.SHA)
		assert.Equal(t, *opts.Official, sv.Official)
		assert.Equal(t, *opts.Deprecated, sv.Deprecated)
		assert.Equal(t, *opts.DeprecatedReason, *sv.DeprecatedReason)
		assert.Equal(t, *opts.Enabled, sv.Enabled)
		assert.Equal(t, *opts.Beta, sv.Beta)

		updateVersion := createAdminSentinelVersion()
		updateURL := "https://app.terraform.io/"
		updateOpts := AdminSentinelVersionUpdateOptions{
			Version:    String(updateVersion),
			URL:        String(updateURL),
			Deprecated: Bool(false),
		}

		sv, err = client.Admin.SentinelVersions.Update(ctx, id, updateOpts)
		require.NoError(t, err)

		assert.Equal(t, updateVersion, sv.Version)
		assert.Equal(t, updateURL, sv.URL)
		assert.Equal(t, opts.SHA, sv.SHA)
		assert.Equal(t, *opts.Official, sv.Official)
		assert.Equal(t, *updateOpts.Deprecated, sv.Deprecated)
		assert.Equal(t, *opts.Enabled, sv.Enabled)
		assert.Equal(t, *opts.Beta, sv.Beta)
	})

	t.Run("with non-existent Sentinel version", func(t *testing.T) {
		randomID := "random-id"
		_, err := client.Admin.SentinelVersions.Read(ctx, randomID)
		require.Error(t, err)
	})
}
