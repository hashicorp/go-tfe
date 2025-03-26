// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package tfe

import (
	"bytes"
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"
)

// Compile-time proof of interface implementation.
var _ Explorer = (*explorer)(nil)

// Explorer describes all the explorer related methods that the Terraform Enterprise API supports.
//
// TFE API docs: https://developer.hashicorp.com/terraform/cloud-docs/api-docs/explorer
type Explorer interface {
	// Query information about workspaces within an organization.
	QueryWorkspaces(ctx context.Context, organization string, options ExplorerQueryOptions) (*ExplorerWorkspaceViewList, error)
	// Query information about module version usage within an organization.
	QueryModules(ctx context.Context, organization string, options ExplorerQueryOptions) (*ExplorerModuleViewList, error)
	// Query information about provider version usage within an organization.
	QueryProviders(ctx context.Context, organization string, options ExplorerQueryOptions) (*ExplorerProviderViewList, error)
	// Query information about Terraform version usage within an organization.
	QueryTerraformVersions(ctx context.Context, organization string, options ExplorerQueryOptions) (*ExplorerTerraformVersionViewList, error)
	// Download a full, unpaged export of query results in CSV format.
	ExportToCSV(ctx context.Context, organization string, options ExplorerQueryOptions) ([]byte, error)
}

type explorer struct {
	client *Client
}

// ExplorerViewType represents the view types the Explorer API supports
// https://developer.hashicorp.com/terraform/cloud-docs/api-docs/explorer#view-types
type ExplorerViewType string

const (
	WorkspacesViewType        ExplorerViewType = "workspaces"
	ProvidersViewType         ExplorerViewType = "providers"
	ModulesViewType           ExplorerViewType = "modules"
	TerraformVersionsViewType ExplorerViewType = "tf_versions"
)

// ExplorerQueryFilterOperator represents the supported operations for filtering.
// https://developer.hashicorp.com/terraform/cloud-docs/api-docs/explorer#filter-operators
type ExplorerQueryFilterOperator string

const (
	OpIs                 ExplorerQueryFilterOperator = "is"
	OpIsNot              ExplorerQueryFilterOperator = "is_not"
	OpContains           ExplorerQueryFilterOperator = "contains"
	OpDoesNotContain     ExplorerQueryFilterOperator = "does_not_contain"
	OpIsEmpty            ExplorerQueryFilterOperator = "is_empty"
	OpIsNotEmpty         ExplorerQueryFilterOperator = "is_not_empty"
	OpGreaterThan        ExplorerQueryFilterOperator = "gt"
	OpLessThan           ExplorerQueryFilterOperator = "lt"
	OpGreaterThanOrEqual ExplorerQueryFilterOperator = "gteq"
	OpLessThanOrEqual    ExplorerQueryFilterOperator = "lteq"
	OpIsBefore           ExplorerQueryFilterOperator = "is_before"
	OpIsAfter            ExplorerQueryFilterOperator = "is_after"
)

// ExplorerQueryFilter represents a filter query parameter for the query endpoint.
type ExplorerQueryFilter struct {
	// Unique, sequential index for each filter, starting at 0 and incrementing by 1.
	Index int
	// Field name to apply the filter, valid for the queried view type.
	Name string
	// The operator use when filtering, must be supported by the field type.
	Operator ExplorerQueryFilterOperator
	// The filter value used by the filter during the query.
	Value string
}

func (eqf *ExplorerQueryFilter) toKeyValue() (string, string) {
	key := fmt.Sprintf("filter[%d][%s][%s][0]", eqf.Index, eqf.Name, eqf.Operator)
	return key, eqf.Value
}

// ExplorerQueryOptions represents the parameter options for querying the Explorer API
type ExplorerQueryOptions struct {
	ListOptions

	// Must be one of the following available views: WorkspacesViewType, ModulesViewType,
	// ProvidersViewType or TerraformVersionsViewType. Each query function will
	// set this value automatically, except ExportToCSV().
	View ExplorerViewType `url:"type"`
	// Optional snake_case field to sort data, prefix with '-' for descending, must exist in view type.
	Sort string `url:"sort,omitempty"`

	// List of fields to limit the data returned by the query.
	Fields []string `url:"-"`
	// List of filters to limit the data returned by the query.
	Filters []*ExplorerQueryFilter `url:"-"`
}

func (eqo *ExplorerQueryOptions) extractFilters() map[string][]string {
	filterParams := make(map[string][]string)
	for _, filter := range eqo.Filters {
		if filter != nil {
			k, v := filter.toKeyValue()
			filterParams[k] = []string{v}
		}
	}

	// Append the fields query param, ensuring the correct view type is specified
	if len(eqo.Fields) > 0 {
		fieldsKey := fmt.Sprintf("fields[%s]", eqo.View)
		filterParams[fieldsKey] = []string{strings.Join(eqo.Fields, ",")}
	}
	return filterParams
}

// WorkspaceView represents information about a workspace in the target
// organization and any current runs associated with that workspace.
type WorkspaceView struct {
	Type                         string      `jsonapi:"primary,visibility-workspace"`
	AllChecksSucceeded           bool        `jsonapi:"attr,all-checks-succeeded"`
	ChecksErrored                int         `jsonapi:"attr,checks-errored"`
	ChecksFailed                 int         `jsonapi:"attr,checks-failed"`
	ChecksPassed                 int         `jsonapi:"attr,checks-passed"`
	ChecksUnknown                int         `jsonapi:"attr,checks-unknown"`
	CurrentRunAppliedAt          time.Time   `jsonapi:"attr,current-run-applied-at,rfc3339"`
	CurrentRunExternalID         string      `jsonapi:"attr,current-run-external-id"`
	CurrentRunStatus             RunStatus   `jsonapi:"attr,current-run-status"`
	Drifted                      bool        `jsonapi:"attr,drifted"`
	ExternalID                   string      `jsonapi:"attr,external-id"`
	ModuleCount                  int         `jsonapi:"attr,module-count"`
	Modules                      interface{} `jsonapi:"attr,modules"`
	OrganizationName             string      `jsonapi:"attr,organization-name"`
	ProjectExternalID            string      `jsonapi:"attr,project-external-id"`
	ProjectName                  string      `jsonapi:"attr,project-name"`
	ProviderCount                int         `jsonapi:"attr,provider-count"`
	Providers                    interface{} `jsonapi:"attr,providers"`
	ResourcesDrifted             int         `jsonapi:"attr,resources-drifted"`
	ResourcesUndrifted           int         `jsonapi:"attr,resources-undrifted"`
	StateVersionTerraformVersion string      `jsonapi:"attr,state-version-terraform-version"`
	VCSRepoIdentifier            *string     `jsonapi:"attr,vcs-repo-identifier"`
	WorkspaceCreatedAt           time.Time   `jsonapi:"attr,workspace-created-at,rfc3339"`
	WorkspaceName                string      `jsonapi:"attr,workspace-name"`
	WorkspaceTerraformVersion    string      `jsonapi:"attr,workspace-terraform-version"`
	WorkspaceUpdatedAt           time.Time   `jsonapi:"attr,workspace-updated-at,rfc3339"`
}

// ModuleView represents information about a Terraform module version used by
// an organization.
type ModuleView struct {
	Type           string `jsonapi:"primary,visibility-module-version"`
	Name           string `jsonapi:"attr,name"`
	Source         string `jsonapi:"attr,source"`
	Version        string `jsonapi:"attr,version"`
	WorkspaceCount int    `jsonapi:"attr,workspace-count"`
	Workspaces     string `jsonapi:"attr,workspaces"`
}

// ProviderView represents information about a Terraform provider version used
// by an organization.
type ProviderView struct {
	Type           string `jsonapi:"primary,visibility-provider-version"`
	Name           string `jsonapi:"attr,name"`
	Source         string `jsonapi:"attr,source"`
	Version        string `jsonapi:"attr,version"`
	WorkspaceCount int    `jsonapi:"attr,workspace-count"`
	Workspaces     string `jsonapi:"attr,workspaces"`
}

// TerraformVersionView represents information about a Terraform version used
// by workspaces in an organization.
type TerraformVersionView struct {
	Type           string `jsonapi:"primary,visibility-tf-version"`
	Version        string `jsonapi:"attr,version"`
	WorkspaceCount int    `jsonapi:"attr,workspace-count"`
	Workspaces     string `jsonapi:"attr,workspaces"`
}

// ExplorerWorkspaceViewList represents a list of workspace views
type ExplorerWorkspaceViewList struct {
	*Pagination
	Items []*WorkspaceView
}

// ExplorerModuleViewList represents a list of module views
type ExplorerModuleViewList struct {
	*Pagination
	Items []*ModuleView
}

// ExplorerProviderViewList represents a list of provider views
type ExplorerProviderViewList struct {
	*Pagination
	Items []*ProviderView
}

// ExplorerTerraformVersionViewList represents a list of Terraform version views
type ExplorerTerraformVersionViewList struct {
	*Pagination
	Items []*TerraformVersionView
}

// QueryWorkspaces invokes the Explorer's Query endpoint to return information
// about workspaces and their associated runs in the specified organization.
func (e *explorer) QueryWorkspaces(ctx context.Context, organization string, options ExplorerQueryOptions) (*ExplorerWorkspaceViewList, error) {
	// Force the correct view type
	options.View = WorkspacesViewType

	req, err := e.buildExplorerQueryRequest(organization, options)
	if err != nil {
		return nil, err
	}

	eql := &ExplorerWorkspaceViewList{}
	err = req.Do(ctx, eql)
	if err != nil {
		return nil, err
	}

	return eql, nil
}

// QueryModules invokes the Explorer's Query endpoint to return information
// about module versions in use across the specified organization.
func (e *explorer) QueryModules(ctx context.Context, organization string, options ExplorerQueryOptions) (*ExplorerModuleViewList, error) {
	// Force the correct view type
	options.View = ModulesViewType

	req, err := e.buildExplorerQueryRequest(organization, options)
	if err != nil {
		return nil, err
	}

	eql := &ExplorerModuleViewList{}
	err = req.Do(ctx, eql)
	if err != nil {
		return nil, err
	}

	return eql, nil
}

// QueryProviders invokes the Explorer's Query endpoint to return information
// about provider versions in use across the specified organization.
func (e *explorer) QueryProviders(ctx context.Context, organization string, options ExplorerQueryOptions) (*ExplorerProviderViewList, error) {
	// Force the correct view type
	options.View = ProvidersViewType

	req, err := e.buildExplorerQueryRequest(organization, options)
	if err != nil {
		return nil, err
	}

	eql := &ExplorerProviderViewList{}
	err = req.Do(ctx, eql)
	if err != nil {
		return nil, err
	}

	return eql, nil
}

// QueryTerraformVersions invokes the Explorer's Query endpoint to return information
// about Terraform versions in use across the specified organization.
func (e *explorer) QueryTerraformVersions(ctx context.Context, organization string, options ExplorerQueryOptions) (*ExplorerTerraformVersionViewList, error) {
	// Force the correct view type
	options.View = TerraformVersionsViewType

	req, err := e.buildExplorerQueryRequest(organization, options)
	if err != nil {
		return nil, err
	}

	eql := &ExplorerTerraformVersionViewList{}
	err = req.Do(ctx, eql)
	if err != nil {
		return nil, err
	}

	return eql, nil
}

// ExportToCSV performs an Explorer query and exports the results to CSV format.
// https://developer.hashicorp.com/terraform/cloud-docs/api-docs/explorer#export-data-as-csv
func (e *explorer) ExportToCSV(ctx context.Context, organization string, options ExplorerQueryOptions) ([]byte, error) {
	filterParams := options.extractFilters()

	u := fmt.Sprintf("organizations/%s/explorer/export/csv", url.QueryEscape(organization))
	req, err := e.client.NewRequestWithAdditionalQueryParams("GET", u, options, filterParams)
	if err != nil {
		return nil, err
	}

	// Override accept header
	req.retryableRequest.Header.Set("Accept", "*/*")

	buf := &bytes.Buffer{}
	err = req.Do(ctx, buf)
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func (e *explorer) buildExplorerQueryRequest(organization string, options ExplorerQueryOptions) (*ClientRequest, error) {
	filterParams := options.extractFilters()

	u := fmt.Sprintf("organizations/%s/explorer", url.QueryEscape(organization))
	return e.client.NewRequestWithAdditionalQueryParams("GET", u, options, filterParams)
}
