// Copyright IBM Corp. 2018, 2026
// SPDX-License-Identifier: MPL-2.0

package tfe

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// These tests run against a pre-existing organization that already contains
// workspaces, because the Explorer aggregates data across an organization and
// there is no API to seed it synchronously. Set EXPLORER_TEST_ORGANIZATION to
// the name of such an organization to enable them.
func explorerTestOrganization(t *testing.T) string {
	t.Helper()

	org := os.Getenv("EXPLORER_TEST_ORGANIZATION")
	if org == "" {
		t.Skip("Set EXPLORER_TEST_ORGANIZATION to the name of an organization with existing workspaces to run explorer tests")
	}

	return org
}

func TestExplorerQuery(t *testing.T) {
	org := explorerTestOrganization(t)
	client := testClient(t)
	ctx := context.Background()

	t.Run("query workspaces", func(t *testing.T) {
		result, err := client.Explorer.Query(ctx, org, ExplorerQueryOptions{
			Type: ExplorerViewWorkspaces,
		})
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.NotEmpty(t, result.Items)

		// Each record carries an id, a polymorphic type, and untyped attributes.
		for _, record := range result.Items {
			assert.NotEmpty(t, record.ID)
			assert.NotEmpty(t, record.Type)
			assert.NotNil(t, record.Attributes)
		}
	})

	t.Run("query with a filter", func(t *testing.T) {
		result, err := client.Explorer.Query(ctx, org, ExplorerQueryOptions{
			Type: ExplorerViewWorkspaces,
			Filters: []ExplorerFilter{
				{
					Field:    "workspace_name",
					Operator: ExplorerOpIsNotEmpty,
				},
			},
		})
		require.NoError(t, err)
		assert.NotEmpty(t, result.Items)
	})

	t.Run("query with sort and field projection", func(t *testing.T) {
		result, err := client.Explorer.Query(ctx, org, ExplorerQueryOptions{
			Type:   ExplorerViewWorkspaces,
			Sort:   "-workspace_updated_at",
			Fields: []string{"workspace_name", "workspace_updated_at"},
		})
		require.NoError(t, err)
		assert.NotEmpty(t, result.Items)
	})

	t.Run("query other view types", func(t *testing.T) {
		for _, viewType := range []ExplorerViewType{
			ExplorerViewProviders,
			ExplorerViewModules,
			ExplorerViewTerraformVersions,
		} {
			_, err := client.Explorer.Query(ctx, org, ExplorerQueryOptions{Type: viewType})
			require.NoError(t, err)
		}
	})

	t.Run("query with a numeric operator", func(t *testing.T) {
		// workspace_count is a number field on the tf_versions view; gt exercises
		// the numeric-operator encoding path against the backend.
		result, err := client.Explorer.Query(ctx, org, ExplorerQueryOptions{
			Type: ExplorerViewTerraformVersions,
			Filters: []ExplorerFilter{
				{Field: "workspace_count", Operator: ExplorerOpGreaterThan, Values: []string{"0"}},
			},
		})
		require.NoError(t, err)
		assert.NotNil(t, result)
	})

	t.Run("with invalid organization", func(t *testing.T) {
		_, err := client.Explorer.Query(ctx, badIdentifier, ExplorerQueryOptions{
			Type: ExplorerViewWorkspaces,
		})
		assert.EqualError(t, err, ErrInvalidOrg.Error())
	})

	t.Run("without a type", func(t *testing.T) {
		_, err := client.Explorer.Query(ctx, org, ExplorerQueryOptions{})
		assert.EqualError(t, err, ErrInvalidExplorerViewType.Error())
	})
}

func TestExplorerExportCSV(t *testing.T) {
	org := explorerTestOrganization(t)
	client := testClient(t)
	ctx := context.Background()

	t.Run("export workspaces as csv", func(t *testing.T) {
		data, err := client.Explorer.ExportCSV(ctx, org, ExplorerQueryOptions{
			Type: ExplorerViewWorkspaces,
		})
		require.NoError(t, err)
		assert.NotEmpty(t, data)
	})

	t.Run("with invalid organization", func(t *testing.T) {
		_, err := client.Explorer.ExportCSV(ctx, badIdentifier, ExplorerQueryOptions{
			Type: ExplorerViewWorkspaces,
		})
		assert.EqualError(t, err, ErrInvalidOrg.Error())
	})
}
