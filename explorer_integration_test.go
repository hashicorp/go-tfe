package tfe

import (
	"bytes"
	"context"
	"encoding/csv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExplorer_QueryModules(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()
	organization := testExplorerOrganization(t)

	t.Run("without any filter, sort, field query params", func(t *testing.T) {
		wql, err := client.Explorer.QueryModules(ctx, organization, ExplorerQueryOptions{})
		require.NoError(t, err)
		require.Greater(t, len(wql.Items), 0)

		for _, view := range wql.Items {
			require.NotEmpty(t, view.Name)
			require.NotEmpty(t, view.Source)
			require.NotEmpty(t, view.Version)
		}
	})

	t.Run("with a sort query param", func(t *testing.T) {
		wql, err := client.Explorer.QueryModules(ctx, organization, ExplorerQueryOptions{
			Sort: "workspace_count",
		})
		require.NoError(t, err)
		require.Greater(t, len(wql.Items), 0)
	})

	t.Run("with a filter query param", func(t *testing.T) {
		wql, err := client.Explorer.QueryModules(ctx, organization, ExplorerQueryOptions{
			Filters: []*ExplorerQueryFilter{
				{
					Index:    0,
					Name:     "workspace_count",
					Operator: OpGreaterThan,
					Value:    "0",
				},
				{
					Index:    1,
					Name:     "workspaces",
					Operator: OpContains,
					Value:    "tflocal",
				},
			},
		})
		require.NoError(t, err)
		require.Greater(t, len(wql.Items), 0)

		for _, view := range wql.Items {
			require.Contains(t, view.Workspaces, "tflocal")
			require.Greater(t, view.WorkspaceCount, 0)
		}
	})
}

func TestExplorer_QueryProviders(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()
	organization := testExplorerOrganization(t)

	t.Run("without any filter, sort, field query params", func(t *testing.T) {
		pql, err := client.Explorer.QueryProviders(ctx, organization, ExplorerQueryOptions{})
		require.NoError(t, err)
		require.Greater(t, len(pql.Items), 0)

		for _, view := range pql.Items {
			require.NotEmpty(t, view.Name)
			require.NotEmpty(t, view.Version)
			require.NotEmpty(t, view.Source)
		}
	})

	t.Run("with a sort query param", func(t *testing.T) {
		wql, err := client.Explorer.QueryWorkspaces(ctx, organization, ExplorerQueryOptions{
			Sort: "module_count",
		})
		require.NoError(t, err)
		require.Greater(t, len(wql.Items), 0)

		prev := wql.Items[0]
		for _, view := range wql.Items {
			if view.ModuleCount > prev.ModuleCount {
				t.Fatalf("entry not sorted: %d > %d", view.ModuleCount, prev.ModuleCount)
			}
			prev = view
		}
	})

	t.Run("with a filter query param", func(t *testing.T) {
		wql, err := client.Explorer.QueryWorkspaces(ctx, organization, ExplorerQueryOptions{
			Filters: []*ExplorerQueryFilter{
				{
					Index:    0,
					Name:     "provider_count",
					Operator: OpGreaterThan,
					Value:    "0",
				},
				{
					Index:    1,
					Name:     "current_run_status",
					Operator: OpIs,
					Value:    "errored",
				},
			},
		})
		require.NoError(t, err)
		require.Greater(t, len(wql.Items), 0)

		for _, view := range wql.Items {
			require.Greater(t, view.ProviderCount, 0)
			require.Equal(t, view.CurrentRunStatus, RunErrored)
		}
	})
}

func TestExplorer_QueryTerraformVersions(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()
	organization := testExplorerOrganization(t)

	t.Run("without any filter, sort, field query params", func(t *testing.T) {
		wql, err := client.Explorer.QueryTerraformVersions(ctx, organization, ExplorerQueryOptions{})
		require.NoError(t, err)
		require.Greater(t, len(wql.Items), 0)

		for _, view := range wql.Items {
			require.NotEmpty(t, view.Version)
		}
	})

	t.Run("with a filter query param", func(t *testing.T) {
		wql, err := client.Explorer.QueryTerraformVersions(ctx, organization, ExplorerQueryOptions{
			Filters: []*ExplorerQueryFilter{
				{
					Index:    0,
					Name:     "version",
					Operator: OpIs,
					Value:    "0.12.0",
				},
			},
		})
		require.NoError(t, err)
		require.Equal(t, len(wql.Items), 0)
	})
}

func TestExplorer_QueryWorkspaces(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()
	organization := testExplorerOrganization(t)

	t.Run("without any filter, sort, field query params", func(t *testing.T) {
		wql, err := client.Explorer.QueryWorkspaces(ctx, organization, ExplorerQueryOptions{})
		require.NoError(t, err)
		require.Greater(t, len(wql.Items), 0)

		for _, view := range wql.Items {
			require.NotEmpty(t, view.WorkspaceName)
			require.NotEmpty(t, view.ExternalID)
			require.NotEmpty(t, view.WorkspaceCreatedAt)
		}
	})

	t.Run("with a sort query param", func(t *testing.T) {
		wql, err := client.Explorer.QueryWorkspaces(ctx, organization, ExplorerQueryOptions{
			Sort: "module_count",
		})
		require.NoError(t, err)
		require.Greater(t, len(wql.Items), 0)

		prev := wql.Items[0]
		for _, view := range wql.Items {
			if view.ModuleCount > prev.ModuleCount {
				t.Fatalf("entry not sorted: %d > %d", view.ModuleCount, prev.ModuleCount)
			}
			prev = view
		}
	})

	t.Run("with a filter query param", func(t *testing.T) {
		wql, err := client.Explorer.QueryWorkspaces(ctx, organization, ExplorerQueryOptions{
			Filters: []*ExplorerQueryFilter{
				{
					Index:    0,
					Name:     "provider_count",
					Operator: OpGreaterThan,
					Value:    "0",
				},
				{
					Index:    1,
					Name:     "current_run_status",
					Operator: OpIs,
					Value:    "errored",
				},
			},
		})
		require.NoError(t, err)
		require.Greater(t, len(wql.Items), 0)

		for _, view := range wql.Items {
			require.Greater(t, view.ProviderCount, 0)
			require.Equal(t, view.CurrentRunStatus, RunErrored)
		}
	})
}

func TestExplorer_ExportToCSV(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()
	organization := testExplorerOrganization(t)

	csvResult, err := client.Explorer.ExportToCSV(ctx, organization, ExplorerQueryOptions{
		View:   WorkspacesViewType,
		Fields: []string{"workspace_name", "current_run_status"},
		Filters: []*ExplorerQueryFilter{
			{
				Index:    0,
				Name:     "current_run_status",
				Operator: OpIs,
				Value:    "applied",
			},
		},
	})
	require.NoError(t, err)
	r := csv.NewReader(bytes.NewReader(csvResult))

	header, err := r.Read()
	require.NoError(t, err)
	assert.Equal(t, len(header), 2)
	// Fields come in the order specified in the request
	assert.Equal(t, header[0], "workspace_name")
	assert.Equal(t, header[1], "current_run_status")

	rows, err := r.ReadAll()
	require.NoError(t, err)
	assert.Greater(t, len(rows), 0)
}
