package tfe

import (
	"context"
	"fmt"
	"net/url"
	"time"
)

// Compile-time proof of interface implementation.
var _ QueryRuns = (*queryRuns)(nil)

// QueryRuns describes all the run related methods that the Terraform Enterprise
// API supports.
//
// NOTE WELL: This is a beta feature and is subject to change until noted otherwise in the
// release notes.
type QueryRuns interface {
	// List all the query runs of the given workspace.
	List(ctx context.Context, workspaceID string, options *QueryRunListOptions) (*QueryRunList, error)

	// Create a new query run with the given options.
	Create(ctx context.Context, options QueryRunCreateOptions) (*QueryRun, error)

	// Read a query run by its ID.
	Read(ctx context.Context, queryRunID string) (*QueryRun, error)

	// ReadWithOptions reads a query run by its ID using the options supplied
	ReadWithOptions(ctx context.Context, queryRunID string, options *QueryRunReadOptions) (*QueryRun, error)

	// Cancel a query run by its ID.
	Cancel(ctx context.Context, runID string) error

	// Force-cancel a query run by its ID.
	ForceCancel(ctx context.Context, runID string) error
}

// QueryRunCreateOptions represents the options for creating a new run.
type QueryRunCreateOptions struct {
	// Type is a public field utilized by JSON:API to
	// set the resource type via the field tag.
	// It is not a user-defined value and does not need to be set.
	// https://jsonapi.org/format/#crud-creating
	Type string `jsonapi:"primary,queries"`

	// TerraformVersion specifies the Terraform version to use in this run.
	// Only valid for plan-only runs; must be a valid Terraform version available to the organization.
	TerraformVersion *string `jsonapi:"attr,terraform-version,omitempty"`

	Source QueryRunSource `jsonapi:"attr,source"`

	// Specifies the configuration version to use for this run. If the
	// configuration version object is omitted, the run will be created using the
	// workspace's latest configuration version.
	ConfigurationVersion *ConfigurationVersion `jsonapi:"relation,configuration-version"`

	// Specifies the workspace where the run will be executed.
	Workspace *Workspace `jsonapi:"relation,workspace"`

	// Variables allows you to specify terraform input variables for
	// a particular run, prioritized over variables defined on the workspace.
	Variables []*RunVariable `jsonapi:"attr,variables,omitempty"`
}

// QueryRunStatusTimestamps holds the timestamps for individual run statuses.
type QueryRunStatusTimestamps struct {
	CanceledAt      time.Time `jsonapi:"attr,canceled-at,rfc3339"`
	ErroredAt       time.Time `jsonapi:"attr,errored-at,rfc3339"`
	ForceCanceledAt time.Time `jsonapi:"attr,force-canceled-at,rfc3339"`
	QueuingAt       time.Time `jsonapi:"attr,queuing-at,rfc3339"`
	FinishedAt      time.Time `jsonapi:"attr,finished-at,rfc3339"`
	RunningAt       time.Time `jsonapi:"attr,running-at,rfc3339"`
}

// QueryRunIncludeOpt represents the available options for include query params.
// https://developer.hashicorp.com/terraform/cloud-docs/api-docs/run#available-related-resources
type QueryRunIncludeOpt string

// QueryRunSource represents the available sources for query runs.
type QueryRunSource string

// List all available run sources.
const (
	QueryRunSourceAPI QueryRunSource = "tfe-api"
)

const (
	QueryRunCreatedBy RunIncludeOpt = "created_by"
	QueryRunConfigVer RunIncludeOpt = "configuration_version"
)

// queryRuns implements QueryRuns.
type queryRuns struct {
	client *Client
}

// QueryRunList represents a list of runs.
type QueryRunList struct {
	*Pagination
	Items []*QueryRun
}

// QueryRunListOptions represents the options for listing runs.
type QueryRunListOptions struct {
	ListOptions
	// Optional: A list of relations to include. See available resources:
	// https://developer.hashicorp.com/terraform/cloud-docs/api-docs/run#available-related-resources
	Include []QueryRunIncludeOpt `url:"include,omitempty"`
}

type QueryRunReadOptions struct {
	// Optional: A list of relations to include. See available resources:
	// https://developer.hashicorp.com/terraform/cloud-docs/api-docs/run#available-related-resources
	Include []QueryRunIncludeOpt `url:"include,omitempty"`
}

// Run represents a Terraform Enterprise run.
type QueryRun struct {
	ID               string               `jsonapi:"primary,queries"`
	CreatedAt        time.Time            `jsonapi:"attr,created-at,iso8601"`
	Source           RunSource            `jsonapi:"attr,source"`
	Status           RunStatus            `jsonapi:"attr,status"`
	StatusTimestamps *RunStatusTimestamps `jsonapi:"attr,status-timestamps"`
	TerraformVersion string               `jsonapi:"attr,terraform-version"`
	Variables        []*RunVariableAttr   `jsonapi:"attr,variables"`

	// Relations
	ConfigurationVersion *ConfigurationVersion `jsonapi:"relation,configuration-version"`
	CreatedBy            *User                 `jsonapi:"relation,created-by"`
	CanceledBy           *User                 `jsonapi:"relation,canceled-by"`
	Workspace            *Workspace            `jsonapi:"relation,workspace"`
}

func (o *QueryRunListOptions) valid() error {
	return nil
}

func (o QueryRunCreateOptions) valid() error {
	if o.Workspace == nil {
		return ErrRequiredWorkspace
	}

	return nil
}

func (r *queryRuns) List(ctx context.Context, workspaceID string, options *QueryRunListOptions) (*QueryRunList, error) {
	if workspaceID == "" {
		return nil, ErrInvalidWorkspaceID
	}
	if err := options.valid(); err != nil {
		return nil, err
	}

	u := fmt.Sprintf("workspaces/%s/queries", url.PathEscape(workspaceID))
	req, err := r.client.NewRequest("GET", u, options)
	if err != nil {
		return nil, err
	}

	var runs QueryRunList
	if err := req.Do(ctx, &runs); err != nil {
		return nil, err
	}

	return &runs, nil
}

func (r *queryRuns) Create(ctx context.Context, options QueryRunCreateOptions) (*QueryRun, error) {
	if err := options.valid(); err != nil {
		return nil, err
	}

	req, err := r.client.NewRequest("POST", "queries", &options)
	if err != nil {
		return nil, err
	}

	var run QueryRun
	if err := req.Do(ctx, &run); err != nil {
		return nil, err
	}

	return &run, nil
}

func (r *queryRuns) Read(ctx context.Context, queryRunID string) (*QueryRun, error) {
	return r.ReadWithOptions(ctx, queryRunID, &QueryRunReadOptions{})
}

func (r *queryRuns) ReadWithOptions(ctx context.Context, queryRunID string, options *QueryRunReadOptions) (*QueryRun, error) {
	if queryRunID == "" {
		return nil, ErrInvalidQueryRunID
	}

	u := fmt.Sprintf("queries/%s", url.PathEscape(queryRunID))
	req, err := r.client.NewRequest("GET", u, options)
	if err != nil {
		return nil, err
	}

	var run QueryRun
	if err := req.Do(ctx, &run); err != nil {
		return nil, err
	}

	return &run, nil
}

func (r *queryRuns) Cancel(ctx context.Context, queryRunID string) error {
	if queryRunID == "" {
		return ErrInvalidQueryRunID
	}

	u := fmt.Sprintf("queries/%s/actions/cancel", url.PathEscape(queryRunID))
	req, err := r.client.NewRequest("POST", u, nil)
	if err != nil {
		return err
	}

	return req.Do(ctx, nil)
}

func (r *queryRuns) ForceCancel(ctx context.Context, queryRunID string) error {
	if queryRunID == "" {
		return ErrInvalidQueryRunID
	}

	u := fmt.Sprintf("queries/%s/actions/force-cancel", url.PathEscape(queryRunID))
	req, err := r.client.NewRequest("POST", u, nil)
	if err != nil {
		return err
	}

	return req.Do(ctx, nil)
}
