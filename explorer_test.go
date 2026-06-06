// Copyright IBM Corp. 2018, 2026
// SPDX-License-Identifier: MPL-2.0

package tfe

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExplorerQueryOptions_filterParams(t *testing.T) {
	t.Run("no filters", func(t *testing.T) {
		opts := ExplorerQueryOptions{Type: ExplorerViewWorkspaces}
		assert.Nil(t, opts.filterParams())
	})

	t.Run("single filter with single value", func(t *testing.T) {
		opts := ExplorerQueryOptions{
			Type: ExplorerViewWorkspaces,
			Filters: []ExplorerFilter{
				{Field: "workspace_name", Operator: ExplorerOpContains, Values: []string{"prod"}},
			},
		}

		assert.Equal(t, map[string][]string{
			"filter[0][workspace_name][contains][0]": {"prod"},
		}, opts.filterParams())
	})

	t.Run("multiple filters and values are indexed independently", func(t *testing.T) {
		opts := ExplorerQueryOptions{
			Type: ExplorerViewModules,
			Filters: []ExplorerFilter{
				{Field: "name", Operator: ExplorerOpContains, Values: []string{"aws", "gcp"}},
				{Field: "version", Operator: ExplorerOpIs, Values: []string{"1.1"}},
			},
		}

		assert.Equal(t, map[string][]string{
			"filter[0][name][contains][0]": {"aws"},
			"filter[0][name][contains][1]": {"gcp"},
			"filter[1][version][is][0]":    {"1.1"},
		}, opts.filterParams())
	})
}

func TestExplorerQueryOptions_valid(t *testing.T) {
	t.Run("missing type", func(t *testing.T) {
		opts := ExplorerQueryOptions{}
		assert.Equal(t, ErrInvalidExplorerViewType, opts.valid())
	})

	t.Run("filter without a field", func(t *testing.T) {
		opts := ExplorerQueryOptions{
			Type:    ExplorerViewWorkspaces,
			Filters: []ExplorerFilter{{Operator: ExplorerOpIs, Values: []string{"x"}}},
		}
		assert.Equal(t, ErrInvalidExplorerFilterField, opts.valid())
	})

	t.Run("filter without an operator", func(t *testing.T) {
		opts := ExplorerQueryOptions{
			Type:    ExplorerViewWorkspaces,
			Filters: []ExplorerFilter{{Field: "workspace_name", Values: []string{"x"}}},
		}
		assert.Equal(t, ErrInvalidExplorerFilterOperator, opts.valid())
	})

	t.Run("valid", func(t *testing.T) {
		opts := ExplorerQueryOptions{
			Type: ExplorerViewWorkspaces,
			Filters: []ExplorerFilter{
				{Field: "workspace_name", Operator: ExplorerOpContains, Values: []string{"prod"}},
			},
		}
		assert.NoError(t, opts.valid())
	})
}

// sampleResponse mirrors the documented Explorer query response. Note the
// snake_case query parameters versus the kebab-case response attributes.
// https://developer.hashicorp.com/terraform/cloud-docs/api-docs/explorer
const sampleExplorerResponse = `{
  "data": [
    {
      "id": "ws-j2sAWeRxuo1b5HYf",
      "type": "visibility-workspace",
      "attributes": {
        "workspace-name": "payments-service",
        "all-checks-succeeded": true
      }
    }
  ],
  "meta": {
    "pagination": {
      "current-page": 1,
      "total-pages": 1,
      "total-count": 2
    }
  }
}`

func TestExplorerQuery_requestEncodingAndDecoding(t *testing.T) {
	t.Parallel()

	var gotURL *url.URL
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotURL = r.URL
		w.Header().Set("Content-Type", "application/vnd.api+json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, sampleExplorerResponse)
	}))
	defer testServer.Close()

	client, err := NewClient(&Config{Address: testServer.URL, Token: "fake-token"})
	require.NoError(t, err)

	result, err := client.Explorer.Query(context.Background(), "my-org", ExplorerQueryOptions{
		Type:   ExplorerViewWorkspaces,
		Sort:   "-workspace_updated_at",
		Fields: []string{"workspace_name", "all_checks_succeeded"},
		Filters: []ExplorerFilter{
			{Field: "workspace_name", Operator: ExplorerOpContains, Values: []string{"test"}},
		},
	})
	require.NoError(t, err)

	// Request encoding: path and snake_case query parameters.
	require.NotNil(t, gotURL)
	assert.Equal(t, "/api/v2/organizations/my-org/explorer", gotURL.Path)
	q := gotURL.Query()
	assert.Equal(t, "workspaces", q.Get("type"))
	assert.Equal(t, "-workspace_updated_at", q.Get("sort"))
	assert.Equal(t, "workspace_name,all_checks_succeeded", q.Get("fields"))
	assert.Equal(t, "test", q.Get("filter[0][workspace_name][contains][0]"))

	// Response decoding: polymorphic type, kebab-case attributes, pagination.
	require.Len(t, result.Items, 1)
	record := result.Items[0]
	assert.Equal(t, "ws-j2sAWeRxuo1b5HYf", record.ID)
	assert.Equal(t, "visibility-workspace", record.Type)
	assert.Equal(t, "payments-service", record.Attributes["workspace-name"])
	assert.Equal(t, true, record.Attributes["all-checks-succeeded"])

	require.NotNil(t, result.Pagination)
	assert.Equal(t, 1, result.CurrentPage)
	assert.Equal(t, 2, result.TotalCount)
}

func TestExplorerExportCSV_returnsRawBytes(t *testing.T) {
	t.Parallel()

	const csv = "workspace-name,all-checks-succeeded\npayments-service,true\n"
	var gotPath string
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "text/csv")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, csv)
	}))
	defer testServer.Close()

	client, err := NewClient(&Config{Address: testServer.URL, Token: "fake-token"})
	require.NoError(t, err)

	data, err := client.Explorer.ExportCSV(context.Background(), "my-org", ExplorerQueryOptions{
		Type: ExplorerViewWorkspaces,
	})
	require.NoError(t, err)

	assert.Equal(t, "/api/v2/organizations/my-org/explorer/export/csv", gotPath)
	assert.Equal(t, csv, string(data))
}
