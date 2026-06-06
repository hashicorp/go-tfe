// Copyright IBM Corp. 2018, 2026
// SPDX-License-Identifier: MPL-2.0

package tfe

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/url"
)

// Compile-time proof of interface implementation.
var _ Explorer = (*explorer)(nil)

// Explorer describes the data-querying methods of the HCP Terraform Explorer
// API. Queries are scoped to an organization and run across its workspaces.
//
// **Note:** The set of queryable view types, their fields, and the operators
// each field supports are defined by the backend, not by this client. The
// exported view-type and operator constants below are conveniences only; the
// corresponding option fields accept any string, so values the backend adds
// later work without upgrading go-tfe.
//
// TFE API Docs: https://developer.hashicorp.com/terraform/cloud-docs/api-docs/explorer
type Explorer interface {
	// Query executes an Explorer query and returns one page of records.
	Query(ctx context.Context, organization string, options ExplorerQueryOptions) (*ExplorerQueryResult, error)

	// ExportCSV executes an Explorer query and returns the result as CSV bytes.
	// The CSV header row uses snake_case field names (e.g. "workspace_name"),
	// unlike the JSON query response, which keys attributes in kebab-case.
	ExportCSV(ctx context.Context, organization string, options ExplorerQueryOptions) ([]byte, error)
}

// explorer implements Explorer.
type explorer struct {
	client *Client
}

// ExplorerViewType identifies which view to query. The exported constants are
// the views available at the time of writing; because this is a string type,
// callers may pass view names the backend adds later without a client upgrade.
type ExplorerViewType string

const (
	ExplorerViewWorkspaces        ExplorerViewType = "workspaces"
	ExplorerViewProviders         ExplorerViewType = "providers"
	ExplorerViewModules           ExplorerViewType = "modules"
	ExplorerViewTerraformVersions ExplorerViewType = "tf_versions"
)

// ExplorerOperator is a filter operator. As with ExplorerViewType, the
// constants are conveniences — the field accepts any operator the backend
// supports, so this list does not have to be kept exhaustively in sync.
type ExplorerOperator string

const (
	// String and shared operators.
	ExplorerOpIs             ExplorerOperator = "is"
	ExplorerOpIsNot          ExplorerOperator = "is_not"
	ExplorerOpContains       ExplorerOperator = "contains"
	ExplorerOpDoesNotContain ExplorerOperator = "does_not_contain"
	ExplorerOpIsEmpty        ExplorerOperator = "is_empty"
	ExplorerOpIsNotEmpty     ExplorerOperator = "is_not_empty"

	// Numeric operators.
	ExplorerOpGreaterThan        ExplorerOperator = "gt"
	ExplorerOpLessThan           ExplorerOperator = "lt"
	ExplorerOpGreaterThanOrEqual ExplorerOperator = "gteq"
	ExplorerOpLessThanOrEqual    ExplorerOperator = "lteq"

	// Datetime operators.
	ExplorerOpIsBefore ExplorerOperator = "is_before"
	ExplorerOpIsAfter  ExplorerOperator = "is_after"
)

// ExplorerFilter is a single filter applied to a query. Field names are passed
// through verbatim and validated server-side, so no field whitelist is baked
// into this client.
type ExplorerFilter struct {
	// Required: the field to filter on, e.g. "workspace_name".
	Field string

	// Required: the operator to apply.
	Operator ExplorerOperator

	// One or more values for the operator.
	Values []string
}

// ExplorerQueryOptions are the options for an Explorer query. Type, Sort, and
// Fields are encoded by go-querystring; Filters are encoded manually because
// their query keys are dynamic (filter[i][field][operator][j]).
type ExplorerQueryOptions struct {
	ListOptions

	// Required: the view type to query.
	Type ExplorerViewType `url:"type"`

	// Optional: a field to sort by; prefix with "-" for descending order.
	Sort string `url:"sort,omitempty"`

	// Optional: restrict the response to the named fields.
	Fields []string `url:"fields,comma,omitempty"`

	// Optional: filters combined with a logical AND.
	Filters []ExplorerFilter `url:"-"`
}

// ExplorerQueryResult is a single page of query records.
type ExplorerQueryResult struct {
	*Pagination
	Items []*ExplorerRecord
}

// ExplorerRecord is a single result row. Attributes are intentionally untyped:
// the available fields differ per view type and are defined by the backend, so
// we surface them as-is rather than hardcoding a struct per view.
type ExplorerRecord struct {
	ID         string
	Type       string
	Attributes map[string]any
}

// explorerQueryResponse mirrors the JSON:API envelope for generic decoding.
// Decoding the attributes as a map avoids the jsonapi library's lack of
// support for unmarshalling polymorphic record slices.
type explorerQueryResponse struct {
	Data []struct {
		ID         string         `json:"id"`
		Type       string         `json:"type"`
		Attributes map[string]any `json:"attributes"`
	} `json:"data"`
	Meta struct {
		Pagination *Pagination `json:"pagination"`
	} `json:"meta"`
}

// Query executes an Explorer query and returns one page of records.
func (s *explorer) Query(ctx context.Context, organization string, options ExplorerQueryOptions) (*ExplorerQueryResult, error) {
	if !validStringID(&organization) {
		return nil, ErrInvalidOrg
	}
	if err := options.valid(); err != nil {
		return nil, err
	}

	u := fmt.Sprintf("/api/v2/organizations/%s/explorer", url.PathEscape(organization))
	req, err := s.client.NewRequestWithAdditionalQueryParams("GET", u, &options, options.filterParams())
	if err != nil {
		return nil, err
	}

	// Passing an io.Writer makes Do apply checkResponseCode (so we get the
	// refined go-tfe errors) and hand us the raw body to decode generically.
	var buf bytes.Buffer
	if err := req.Do(ctx, &buf); err != nil {
		return nil, err
	}

	var raw explorerQueryResponse
	if err := json.Unmarshal(buf.Bytes(), &raw); err != nil {
		return nil, err
	}

	result := &ExplorerQueryResult{Pagination: raw.Meta.Pagination}
	for _, d := range raw.Data {
		result.Items = append(result.Items, &ExplorerRecord{
			ID:         d.ID,
			Type:       d.Type,
			Attributes: d.Attributes,
		})
	}

	return result, nil
}

// ExportCSV executes an Explorer query and returns the result as CSV bytes.
func (s *explorer) ExportCSV(ctx context.Context, organization string, options ExplorerQueryOptions) ([]byte, error) {
	if !validStringID(&organization) {
		return nil, ErrInvalidOrg
	}
	if err := options.valid(); err != nil {
		return nil, err
	}

	u := fmt.Sprintf("/api/v2/organizations/%s/explorer/export/csv", url.PathEscape(organization))
	req, err := s.client.NewRequestWithAdditionalQueryParams("GET", u, &options, options.filterParams())
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	if err := req.Do(ctx, &buf); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// filterParams renders Filters as filter[i][field][operator][j]=value pairs.
func (o *ExplorerQueryOptions) filterParams() map[string][]string {
	if len(o.Filters) == 0 {
		return nil
	}

	params := make(map[string][]string)
	for i, f := range o.Filters {
		for j, v := range f.Values {
			key := fmt.Sprintf("filter[%d][%s][%s][%d]", i, f.Field, f.Operator, j)
			params[key] = []string{v}
		}
	}

	return params
}

func (o *ExplorerQueryOptions) valid() error {
	if o.Type == "" {
		return ErrInvalidExplorerViewType
	}

	for _, f := range o.Filters {
		if f.Field == "" {
			return ErrInvalidExplorerFilterField
		}
		if f.Operator == "" {
			return ErrInvalidExplorerFilterOperator
		}
	}

	return nil
}
