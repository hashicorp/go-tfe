// Copyright IBM Corp. 2018, 2026
// SPDX-License-Identifier: MPL-2.0

// This file contains the final version of the go-tfe (v1) package.

// You may add critical fixes or security updates to this file, but the functionality is
// NO LONGER TESTED and SHOULD NOT BE EXTENDED except for in uncommon situations as determined by
// @hashicorp/tf-core-cloud.

// To add functionality to go-tfe, edit the OpenAPI specification in the Terraform Platform code.
// The github.com/hashicorp/go-tfe/v2 package will be generated from that specification nightly and
// will include the new functionality.

package tfe

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log"
	"math"
	"math/rand"
	"net/http"
	"net/mail"
	"net/url"
	"os"
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/google/go-querystring/query"
	cleanhttp "github.com/hashicorp/go-cleanhttp"
	retryablehttp "github.com/hashicorp/go-retryablehttp"
	slug "github.com/hashicorp/go-slug"
	version "github.com/hashicorp/go-version"
	"github.com/hashicorp/jsonapi"
	"golang.org/x/sync/errgroup"
	"golang.org/x/time/rate"
)

// Compile-time proof of interface implementation.
var _ AdminOPAVersions = (*adminOPAVersions)(nil)

// AdminOPAVersions describes all the admin OPA versions related methods that
// the Terraform Enterprise API supports.
// Note that admin OPA versions are only available in Terraform Enterprise.
//
// TFE API docs: https://developer.hashicorp.com/terraform/enterprise/api-docs/admin/opa-versions
type AdminOPAVersions interface {
	// List all the OPA versions.
	List(ctx context.Context, options *AdminOPAVersionsListOptions) (*AdminOPAVersionsList, error)

	// Read a OPA version by its ID.
	Read(ctx context.Context, id string) (*AdminOPAVersion, error)

	// Create a OPA version.
	Create(ctx context.Context, options AdminOPAVersionCreateOptions) (*AdminOPAVersion, error)

	// Update a OPA version.
	Update(ctx context.Context, id string, options AdminOPAVersionUpdateOptions) (*AdminOPAVersion, error)

	// Delete a OPA version
	Delete(ctx context.Context, id string) error
}

// adminOPAVersions implements AdminOPAVersions.
type adminOPAVersions struct {
	client *Client
}

// AdminOPAVersion represents a OPA Version
type AdminOPAVersion struct {
	ID               string                     `jsonapi:"primary,opa-versions"`
	Version          string                     `jsonapi:"attr,version"`
	URL              string                     `jsonapi:"attr,url,omitempty"`
	SHA              string                     `jsonapi:"attr,sha,omitempty"`
	Deprecated       bool                       `jsonapi:"attr,deprecated"`
	DeprecatedReason *string                    `jsonapi:"attr,deprecated-reason,omitempty"`
	Official         bool                       `jsonapi:"attr,official"`
	Enabled          bool                       `jsonapi:"attr,enabled"`
	Beta             bool                       `jsonapi:"attr,beta"`
	Usage            int                        `jsonapi:"attr,usage"`
	CreatedAt        time.Time                  `jsonapi:"attr,created-at,iso8601"`
	Archs            []*ToolVersionArchitecture `jsonapi:"attr,archs,omitempty"`
}

// AdminOPAVersionsListOptions represents the options for listing
// OPA versions.
type AdminOPAVersionsListOptions struct {
	ListOptions

	// Optional: A query string to find an exact version
	Filter string `url:"filter[version],omitempty"`

	// Optional: A search query string to find all versions that match version substring
	Search string `url:"search[version],omitempty"`
}

// AdminOPAVersionCreateOptions for creating an OPA version.
type AdminOPAVersionCreateOptions struct {
	Type             string                     `jsonapi:"primary,opa-versions"`
	Version          string                     `jsonapi:"attr,version"`       // Required
	URL              string                     `jsonapi:"attr,url,omitempty"` // Required w/ SHA unless Archs are provided
	SHA              string                     `jsonapi:"attr,sha,omitempty"` // Required w/ URL unless Archs are provided
	Official         *bool                      `jsonapi:"attr,official,omitempty"`
	Deprecated       *bool                      `jsonapi:"attr,deprecated,omitempty"`
	DeprecatedReason *string                    `jsonapi:"attr,deprecated-reason,omitempty"`
	Enabled          *bool                      `jsonapi:"attr,enabled,omitempty"`
	Beta             *bool                      `jsonapi:"attr,beta,omitempty"`
	Archs            []*ToolVersionArchitecture `jsonapi:"attr,archs,omitempty"` // Required unless URL and SHA are provided
}

// AdminOPAVersionUpdateOptions for updating OPA version.
type AdminOPAVersionUpdateOptions struct {
	Type             string                     `jsonapi:"primary,opa-versions"`
	Version          *string                    `jsonapi:"attr,version,omitempty"`
	URL              *string                    `jsonapi:"attr,url,omitempty"`
	SHA              *string                    `jsonapi:"attr,sha,omitempty"`
	Official         *bool                      `jsonapi:"attr,official,omitempty"`
	Deprecated       *bool                      `jsonapi:"attr,deprecated,omitempty"`
	DeprecatedReason *string                    `jsonapi:"attr,deprecated-reason,omitempty"`
	Enabled          *bool                      `jsonapi:"attr,enabled,omitempty"`
	Beta             *bool                      `jsonapi:"attr,beta,omitempty"`
	Archs            []*ToolVersionArchitecture `jsonapi:"attr,archs,omitempty"`
}

// AdminOPAVersionsList represents a list of OPA versions.
type AdminOPAVersionsList struct {
	*Pagination
	Items []*AdminOPAVersion
}

// List all the OPA versions.
func (a *adminOPAVersions) List(ctx context.Context, options *AdminOPAVersionsListOptions) (*AdminOPAVersionsList, error) {
	req, err := a.client.NewRequest("GET", "admin/opa-versions", options)
	if err != nil {
		return nil, err
	}

	ol := &AdminOPAVersionsList{}
	err = req.Do(ctx, ol)
	if err != nil {
		return nil, err
	}

	return ol, nil
}

// Read a OPA version by its ID.
func (a *adminOPAVersions) Read(ctx context.Context, id string) (*AdminOPAVersion, error) {
	if !validStringID(&id) {
		return nil, ErrInvalidOPAVersionID
	}

	u := fmt.Sprintf("admin/opa-versions/%s", url.PathEscape(id))
	req, err := a.client.NewRequest("GET", u, nil)
	if err != nil {
		return nil, err
	}

	ov := &AdminOPAVersion{}
	err = req.Do(ctx, ov)
	if err != nil {
		return nil, err
	}

	return ov, nil
}

// Create a new OPA version.
func (a *adminOPAVersions) Create(ctx context.Context, options AdminOPAVersionCreateOptions) (*AdminOPAVersion, error) {
	if err := options.valid(); err != nil {
		return nil, err
	}
	req, err := a.client.NewRequest("POST", "admin/opa-versions", &options)
	if err != nil {
		return nil, err
	}

	ov := &AdminOPAVersion{}
	err = req.Do(ctx, ov)
	if err != nil {
		return nil, err
	}

	return ov, nil
}

// Update an existing OPA version.
func (a *adminOPAVersions) Update(ctx context.Context, id string, options AdminOPAVersionUpdateOptions) (*AdminOPAVersion, error) {
	if !validStringID(&id) {
		return nil, ErrInvalidOPAVersionID
	}

	u := fmt.Sprintf("admin/opa-versions/%s", url.PathEscape(id))
	req, err := a.client.NewRequest("PATCH", u, &options)
	if err != nil {
		return nil, err
	}

	ov := &AdminOPAVersion{}
	err = req.Do(ctx, ov)
	if err != nil {
		return nil, err
	}

	return ov, nil
}

// Delete a OPA version.
func (a *adminOPAVersions) Delete(ctx context.Context, id string) error {
	if !validStringID(&id) {
		return ErrInvalidOPAVersionID
	}

	u := fmt.Sprintf("admin/opa-versions/%s", url.PathEscape(id))
	req, err := a.client.NewRequest("DELETE", u, nil)
	if err != nil {
		return err
	}

	return req.Do(ctx, nil)
}

func (o AdminOPAVersionCreateOptions) valid() error {
	if (reflect.DeepEqual(o, AdminOPAVersionCreateOptions{})) {
		return ErrRequiredOPAVerCreateOps
	}
	if o.Version == "" {
		return ErrRequiredVersion
	}
	if !o.validArch() {
		return ErrRequiredArchsOrURLAndSha
	}
	return nil
}

func (o AdminOPAVersionCreateOptions) validArch() bool {
	if o.Archs == nil && o.URL != "" && o.SHA != "" {
		return true
	}

	for _, a := range o.Archs {
		if !validArch(a) {
			return false
		}
	}

	return true
}

// Compile-time proof of interface implementation.
var _ AdminOrganizations = (*adminOrganizations)(nil)

// AdminOrganizations describes all of the admin organization related methods that the Terraform
// Enterprise API supports. Note that admin settings are only available in Terraform Enterprise.
//
// TFE API docs: https://developer.hashicorp.com/terraform/enterprise/api-docs/admin/organizations
type AdminOrganizations interface {
	// List all the organizations visible to the current user.
	List(ctx context.Context, options *AdminOrganizationListOptions) (*AdminOrganizationList, error)

	// Read attributes of an existing organization via admin API.
	Read(ctx context.Context, organization string) (*AdminOrganization, error)

	// Update attributes of an existing organization via admin API.
	Update(ctx context.Context, organization string, options AdminOrganizationUpdateOptions) (*AdminOrganization, error)

	// Delete an organization by its name via admin API
	Delete(ctx context.Context, organization string) error

	// ListModuleConsumers lists specific organizations in the Terraform Enterprise installation that have permission to use an organization's modules.
	ListModuleConsumers(ctx context.Context, organization string, options *AdminOrganizationListModuleConsumersOptions) (*AdminOrganizationList, error)

	// UpdateModuleConsumers specifies a list of organizations that can use modules from the sharing organization's private registry. Setting a list of module consumers will turn off global module sharing for an organization.
	UpdateModuleConsumers(ctx context.Context, organization string, consumerOrganizations []string) error
}

// adminOrganizations implements AdminOrganizations.
type adminOrganizations struct {
	client *Client
}

// AdminOrganization represents a Terraform Enterprise organization returned from the Admin API.
type AdminOrganization struct {
	Name                             string `jsonapi:"primary,organizations"`
	AccessBetaTools                  bool   `jsonapi:"attr,access-beta-tools"`
	ExternalID                       string `jsonapi:"attr,external-id"`
	GlobalModuleSharing              *bool  `jsonapi:"attr,global-module-sharing"`
	GlobalProviderSharing            *bool  `jsonapi:"attr,global-provider-sharing"`
	IsDisabled                       bool   `jsonapi:"attr,is-disabled"`
	NotificationEmail                string `jsonapi:"attr,notification-email"`
	SsoEnabled                       bool   `jsonapi:"attr,sso-enabled"`
	TerraformBuildWorkerApplyTimeout string `jsonapi:"attr,terraform-build-worker-apply-timeout"`
	TerraformBuildWorkerPlanTimeout  string `jsonapi:"attr,terraform-build-worker-plan-timeout"`
	ApplyTimeout                     string `jsonapi:"attr,apply-timeout"`
	PlanTimeout                      string `jsonapi:"attr,plan-timeout"`
	TerraformWorkerSudoEnabled       bool   `jsonapi:"attr,terraform-worker-sudo-enabled"`
	WorkspaceLimit                   *int   `jsonapi:"attr,workspace-limit"`

	// Relations
	Owners []*User `jsonapi:"relation,owners"`
}

// AdminOrganizationUpdateOptions represents the admin options for updating an organization.
// https://developer.hashicorp.com/terraform/enterprise/api-docs/admin/organizations#request-body
type AdminOrganizationUpdateOptions struct {
	AccessBetaTools                  *bool   `jsonapi:"attr,access-beta-tools,omitempty"`
	GlobalModuleSharing              *bool   `jsonapi:"attr,global-module-sharing,omitempty"`
	GlobalProviderSharing            *bool   `jsonapi:"attr,global-provider-sharing,omitempty"`
	IsDisabled                       *bool   `jsonapi:"attr,is-disabled,omitempty"`
	TerraformBuildWorkerApplyTimeout *string `jsonapi:"attr,terraform-build-worker-apply-timeout,omitempty"`
	TerraformBuildWorkerPlanTimeout  *string `jsonapi:"attr,terraform-build-worker-plan-timeout,omitempty"`
	ApplyTimeout                     *string `jsonapi:"attr,apply-timeout,omitempty"`
	PlanTimeout                      *string `jsonapi:"attr,plan-timeout,omitempty"`
	TerraformWorkerSudoEnabled       bool    `jsonapi:"attr,terraform-worker-sudo-enabled,omitempty"`
	WorkspaceLimit                   *int    `jsonapi:"attr,workspace-limit,omitempty"`
}

// AdminOrganizationList represents a list of organizations via Admin API.
type AdminOrganizationList struct {
	*Pagination
	Items []*AdminOrganization
}

// AdminOrgIncludeOpt represents the available options for include query params.
// https://developer.hashicorp.com/terraform/enterprise/api-docs/admin/organizations#available-related-resources
type AdminOrgIncludeOpt string

const AdminOrgOwners AdminOrgIncludeOpt = "owners"

// AdminOrganizationListOptions represents the options for listing organizations via Admin API.
type AdminOrganizationListOptions struct {
	ListOptions

	// Optional: A query string used to filter organizations.
	// Any organizations with a name or notification email partially matching this value will be returned.
	Query string `url:"q,omitempty"`
	// Optional: A list of relations to include. See available resources
	// https://developer.hashicorp.com/terraform/enterprise/api-docs/admin/organizations#available-related-resources
	Include []AdminOrgIncludeOpt `url:"include,omitempty"`
}

// AdminOrganizationListModuleConsumersOptions represents the options for listing organization module consumers through the Admin API
type AdminOrganizationListModuleConsumersOptions struct {
	ListOptions
}

type AdminOrganizationID struct {
	ID string `jsonapi:"primary,organizations"`
}

// List all the organizations visible to the current user.
func (s *adminOrganizations) List(ctx context.Context, options *AdminOrganizationListOptions) (*AdminOrganizationList, error) {
	if err := options.valid(); err != nil {
		return nil, err
	}
	u := "admin/organizations"
	req, err := s.client.NewRequest("GET", u, options)
	if err != nil {
		return nil, err
	}

	orgl := &AdminOrganizationList{}
	err = req.Do(ctx, orgl)
	if err != nil {
		return nil, err
	}

	return orgl, nil
}

// ListModuleConsumers lists specific organizations in the Terraform Enterprise installation that have permission to use an organization's modules.
func (s *adminOrganizations) ListModuleConsumers(ctx context.Context, organization string, options *AdminOrganizationListModuleConsumersOptions) (*AdminOrganizationList, error) {
	if !validStringID(&organization) {
		return nil, ErrInvalidOrg
	}

	u := fmt.Sprintf("admin/organizations/%s/relationships/module-consumers", url.PathEscape(organization))

	req, err := s.client.NewRequest("GET", u, nil)
	if err != nil {
		return nil, err
	}

	orgl := &AdminOrganizationList{}
	err = req.Do(ctx, orgl)
	if err != nil {
		return nil, err
	}

	return orgl, nil
}

// Read an organization by its name.
func (s *adminOrganizations) Read(ctx context.Context, organization string) (*AdminOrganization, error) {
	if !validStringID(&organization) {
		return nil, ErrInvalidOrg
	}

	u := fmt.Sprintf("admin/organizations/%s", url.PathEscape(organization))
	req, err := s.client.NewRequest("GET", u, nil)
	if err != nil {
		return nil, err
	}

	org := &AdminOrganization{}
	err = req.Do(ctx, org)
	if err != nil {
		return nil, err
	}

	return org, nil
}

// Update an organization by its name.
func (s *adminOrganizations) Update(ctx context.Context, organization string, options AdminOrganizationUpdateOptions) (*AdminOrganization, error) {
	if !validStringID(&organization) {
		return nil, ErrInvalidOrg
	}

	u := fmt.Sprintf("admin/organizations/%s", url.PathEscape(organization))
	req, err := s.client.NewRequest("PATCH", u, &options)
	if err != nil {
		return nil, err
	}

	org := &AdminOrganization{}
	err = req.Do(ctx, org)
	if err != nil {
		return nil, err
	}

	return org, nil
}

// UpdateModuleConsumers updates an organization to specify a list of organizations that can use modules from the sharing organization's private registry.
func (s *adminOrganizations) UpdateModuleConsumers(ctx context.Context, organization string, consumerOrganizationIDs []string) error {
	if !validStringID(&organization) {
		return ErrInvalidOrg
	}

	u := fmt.Sprintf("admin/organizations/%s/relationships/module-consumers", url.PathEscape(organization))

	var organizations []*AdminOrganizationID
	for _, id := range consumerOrganizationIDs {
		if !validStringID(&id) {
			return ErrInvalidOrg
		}
		organizations = append(organizations, &AdminOrganizationID{ID: id})
	}

	req, err := s.client.NewRequest("PATCH", u, organizations)
	if err != nil {
		return err
	}

	err = req.Do(ctx, nil)
	if err != nil {
		return err
	}

	return nil
}

// Delete an organization by its name.
func (s *adminOrganizations) Delete(ctx context.Context, organization string) error {
	if !validStringID(&organization) {
		return ErrInvalidOrg
	}

	u := fmt.Sprintf("admin/organizations/%s", url.PathEscape(organization))
	req, err := s.client.NewRequest("DELETE", u, nil)
	if err != nil {
		return err
	}

	return req.Do(ctx, nil)
}

func (o *AdminOrganizationListOptions) valid() error {
	return nil
}

// Copyright IBM Corp. 2018, 2026
// SPDX-License-Identifier: MPL-2.0

// Compile-time proof of interface implementation.
var _ AdminRuns = (*adminRuns)(nil)

// AdminRuns describes all the admin run related methods that the Terraform
// Enterprise  API supports.
// It contains endpoints to help site administrators manage their runs.
//
// TFE API docs: https://developer.hashicorp.com/terraform/enterprise/api-docs/admin/runs
type AdminRuns interface {
	// List all the runs of the given installation.
	List(ctx context.Context, options *AdminRunsListOptions) (*AdminRunsList, error)

	// Force-cancel a run by its ID.
	ForceCancel(ctx context.Context, runID string, options AdminRunForceCancelOptions) error
}

// AdminRun represents AdminRuns interface.
type AdminRun struct {
	ID               string               `jsonapi:"primary,runs"`
	CreatedAt        time.Time            `jsonapi:"attr,created-at,iso8601"`
	HasChanges       bool                 `jsonapi:"attr,has-changes"`
	Status           RunStatus            `jsonapi:"attr,status"`
	StatusTimestamps *RunStatusTimestamps `jsonapi:"attr,status-timestamps"`

	// Relations
	Workspace    *AdminWorkspace    `jsonapi:"relation,workspace"`
	Organization *AdminOrganization `jsonapi:"relation,workspace.organization"`
}

// AdminRunsList represents a list of runs.
type AdminRunsList struct {
	*Pagination
	Items []*AdminRun
}

// AdminRunIncludeOpt represents the available options for include query params.
// https://developer.hashicorp.com/terraform/enterprise/api-docs/admin/runs#available-related-resources
type AdminRunIncludeOpt string

const (
	AdminRunWorkspace          AdminRunIncludeOpt = "workspace"
	AdminRunWorkspaceOrg       AdminRunIncludeOpt = "workspace.organization"
	AdminRunWorkspaceOrgOwners AdminRunIncludeOpt = "workspace.organization.owners"
)

// AdminRunsListOptions represents the options for listing runs.
// https://developer.hashicorp.com/terraform/enterprise/api-docs/admin/runs#query-parameters
type AdminRunsListOptions struct {
	ListOptions

	RunStatus     string `url:"filter[status],omitempty"`
	CreatedBefore string `url:"filter[to],omitempty"`
	CreatedAfter  string `url:"filter[from],omitempty"`
	Query         string `url:"q,omitempty"`
	// Optional: A list of relations to include. See available resources
	// https://developer.hashicorp.com/terraform/enterprise/api-docs/admin/runs#available-related-resources
	Include []AdminRunIncludeOpt `url:"include,omitempty"`
}

// adminRuns implements the AdminRuns interface.
type adminRuns struct {
	client *Client
}

// List all the runs of the terraform enterprise installation.
// https://developer.hashicorp.com/terraform/enterprise/api-docs/admin/runs#list-all-runs
func (s *adminRuns) List(ctx context.Context, options *AdminRunsListOptions) (*AdminRunsList, error) {
	if err := options.valid(); err != nil {
		return nil, err
	}

	u := "admin/runs"
	req, err := s.client.NewRequest("GET", u, options)
	if err != nil {
		return nil, err
	}

	rl := &AdminRunsList{}
	err = req.Do(ctx, rl)
	if err != nil {
		return nil, err
	}

	return rl, nil
}

// AdminRunForceCancelOptions represents the options for force-canceling a run.
type AdminRunForceCancelOptions struct {
	// An optional comment explaining the reason for the force-cancel.
	// https://developer.hashicorp.com/terraform/enterprise/api-docs/admin/runs#request-body
	Comment *string `json:"comment,omitempty"`
}

// ForceCancel is used to forcefully cancel a run by its ID.
// https://developer.hashicorp.com/terraform/enterprise/api-docs/admin/runs#force-a-run-into-the-quot-cancelled-quot-state
func (s *adminRuns) ForceCancel(ctx context.Context, runID string, options AdminRunForceCancelOptions) error {
	if !validStringID(&runID) {
		return ErrInvalidRunID
	}

	u := fmt.Sprintf("admin/runs/%s/actions/force-cancel", url.PathEscape(runID))
	req, err := s.client.NewRequest("POST", u, &options)
	if err != nil {
		return err
	}

	return req.Do(ctx, nil)
}

func (o *AdminRunsListOptions) valid() error {
	if o == nil { // nothing to validate
		return nil
	}

	if err := validateAdminRunDateRanges(o.CreatedBefore, o.CreatedAfter); err != nil {
		return err
	}

	if err := validateAdminRunFilterParams(o.RunStatus); err != nil {
		return err
	}

	return nil
}

func validateAdminRunDateRanges(before, after string) error {
	if validString(&before) {
		_, err := time.Parse(time.RFC3339, before)
		if err != nil {
			return fmt.Errorf("invalid date format for CreatedBefore: '%s', must be in RFC3339 format", before)
		}
	}

	if validString(&after) {
		_, err := time.Parse(time.RFC3339, after)
		if err != nil {
			return fmt.Errorf("invalid date format for CreatedAfter: '%s', must be in RFC3339 format", after)
		}
	}

	return nil
}

func validateAdminRunFilterParams(runStatus string) error {
	// For the platform, an invalid filter value is a semantically understood query that returns an empty set, no error, no warning. But for go-tfe, an invalid value is good enough reason to error prior to a network call to the platform:
	if validString(&runStatus) {
		sanitizedRunstatus := strings.TrimSpace(runStatus)
		runStatuses := strings.Split(sanitizedRunstatus, ",")
		// iterate over our statuses, and ensure it is valid.
		for _, status := range runStatuses {
			switch status {
			case string(RunApplied),
				string(RunApplyQueued),
				string(RunApplying),
				string(RunCanceled),
				string(RunConfirmed),
				string(RunCostEstimated),
				string(RunCostEstimating),
				string(RunDiscarded),
				string(RunErrored),
				string(RunFetching),
				string(RunFetchingCompleted),
				string(RunPending),
				string(RunPlanned),
				string(RunPlannedAndFinished),
				string(RunPlannedAndSaved),
				string(RunPlanning),
				string(RunPlanQueued),
				string(RunPolicyChecked),
				string(RunPolicyChecking),
				string(RunPolicyOverride),
				string(RunPolicySoftFailed),
				string(RunPostPlanAwaitingDecision),
				string(RunPostPlanCompleted),
				string(RunPostPlanRunning),
				string(RunPreApplyRunning),
				string(RunPreApplyCompleted),
				string(RunPrePlanCompleted),
				string(RunPrePlanRunning),
				string(RunPostApplyCompleted),
				string(RunPostApplyRunning),
				string(RunQueuing),
				string(RunQueuingApply),
				"":
				// do nothing
			default:
				return fmt.Errorf(`invalid value "%s" for run status`, status)
			}
		}
	}

	return nil
}

// Compile-time proof of interface implementation.
var _ AdminSentinelVersions = (*adminSentinelVersions)(nil)

// AdminSentinelVersions describes all the admin Sentinel versions related methods that
// the Terraform Enterprise API supports.
// Note that admin Sentinel versions are only available in Terraform Enterprise.
//
// TFE API docs: https://developer.hashicorp.com/terraform/enterprise/api-docs/admin/sentinel-versions
type AdminSentinelVersions interface {
	// List all the Sentinel versions.
	List(ctx context.Context, options *AdminSentinelVersionsListOptions) (*AdminSentinelVersionsList, error)

	// Read a Sentinel version by its ID.
	Read(ctx context.Context, id string) (*AdminSentinelVersion, error)

	// Create a Sentinel version.
	Create(ctx context.Context, options AdminSentinelVersionCreateOptions) (*AdminSentinelVersion, error)

	// Update a Sentinel version.
	Update(ctx context.Context, id string, options AdminSentinelVersionUpdateOptions) (*AdminSentinelVersion, error)

	// Delete a Sentinel version
	Delete(ctx context.Context, id string) error
}

// adminSentinelVersions implements AdminSentinelVersions.
type adminSentinelVersions struct {
	client *Client
}

// AdminSentinelVersion represents a Sentinel Version
type AdminSentinelVersion struct {
	ID               string                     `jsonapi:"primary,sentinel-versions"`
	Version          string                     `jsonapi:"attr,version"`
	URL              string                     `jsonapi:"attr,url,omitempty"`
	SHA              string                     `jsonapi:"attr,sha,omitempty"`
	Deprecated       bool                       `jsonapi:"attr,deprecated"`
	DeprecatedReason *string                    `jsonapi:"attr,deprecated-reason,omitempty"`
	Official         bool                       `jsonapi:"attr,official"`
	Enabled          bool                       `jsonapi:"attr,enabled"`
	Beta             bool                       `jsonapi:"attr,beta"`
	Usage            int                        `jsonapi:"attr,usage"`
	CreatedAt        time.Time                  `jsonapi:"attr,created-at,iso8601"`
	Archs            []*ToolVersionArchitecture `jsonapi:"attr,archs,omitempty"`
}

// AdminSentinelVersionsListOptions represents the options for listing
// Sentinel versions.
type AdminSentinelVersionsListOptions struct {
	ListOptions

	// Optional: A query string to find an exact version
	Filter string `url:"filter[version],omitempty"`

	// Optional: A search query string to find all versions that match version substring
	Search string `url:"search[version],omitempty"`
}

// AdminSentinelVersionCreateOptions for creating an Sentinel version.
type AdminSentinelVersionCreateOptions struct {
	Type             string                     `jsonapi:"primary,sentinel-versions"`
	Version          string                     `jsonapi:"attr,version"`       // Required
	URL              string                     `jsonapi:"attr,url,omitempty"` // Required w/ SHA unless Archs are provided
	SHA              string                     `jsonapi:"attr,sha,omitempty"` // Required w/ URL unless Archs are provided
	Official         *bool                      `jsonapi:"attr,official,omitempty"`
	Deprecated       *bool                      `jsonapi:"attr,deprecated,omitempty"`
	DeprecatedReason *string                    `jsonapi:"attr,deprecated-reason,omitempty"`
	Enabled          *bool                      `jsonapi:"attr,enabled,omitempty"`
	Beta             *bool                      `jsonapi:"attr,beta,omitempty"`
	Archs            []*ToolVersionArchitecture `jsonapi:"attr,archs,omitempty"` // Required unless URL and SHA are provided
}

// AdminSentinelVersionUpdateOptions for updating Sentinel version.
type AdminSentinelVersionUpdateOptions struct {
	Type             string                     `jsonapi:"primary,sentinel-versions"`
	Version          *string                    `jsonapi:"attr,version,omitempty"`
	URL              *string                    `jsonapi:"attr,url,omitempty"`
	SHA              *string                    `jsonapi:"attr,sha,omitempty"`
	Official         *bool                      `jsonapi:"attr,official,omitempty"`
	Deprecated       *bool                      `jsonapi:"attr,deprecated,omitempty"`
	DeprecatedReason *string                    `jsonapi:"attr,deprecated-reason,omitempty"`
	Enabled          *bool                      `jsonapi:"attr,enabled,omitempty"`
	Beta             *bool                      `jsonapi:"attr,beta,omitempty"`
	Archs            []*ToolVersionArchitecture `jsonapi:"attr,archs,omitempty"`
}

// AdminSentinelVersionsList represents a list of Sentinel versions.
type AdminSentinelVersionsList struct {
	*Pagination
	Items []*AdminSentinelVersion
}

// List all the Sentinel versions.
func (a *adminSentinelVersions) List(ctx context.Context, options *AdminSentinelVersionsListOptions) (*AdminSentinelVersionsList, error) {
	req, err := a.client.NewRequest("GET", "admin/sentinel-versions", options)
	if err != nil {
		return nil, err
	}

	sl := &AdminSentinelVersionsList{}
	err = req.Do(ctx, sl)
	if err != nil {
		return nil, err
	}

	return sl, nil
}

// Read a Sentinel version by its ID.
func (a *adminSentinelVersions) Read(ctx context.Context, id string) (*AdminSentinelVersion, error) {
	if !validStringID(&id) {
		return nil, ErrInvalidSentinelVersionID
	}

	u := fmt.Sprintf("admin/sentinel-versions/%s", url.PathEscape(id))
	req, err := a.client.NewRequest("GET", u, nil)
	if err != nil {
		return nil, err
	}

	sv := &AdminSentinelVersion{}
	err = req.Do(ctx, sv)
	if err != nil {
		return nil, err
	}

	return sv, nil
}

// Create a new Sentinel version.
func (a *adminSentinelVersions) Create(ctx context.Context, options AdminSentinelVersionCreateOptions) (*AdminSentinelVersion, error) {
	if err := options.valid(); err != nil {
		return nil, err
	}
	req, err := a.client.NewRequest("POST", "admin/sentinel-versions", &options)
	if err != nil {
		return nil, err
	}

	sv := &AdminSentinelVersion{}
	err = req.Do(ctx, sv)
	if err != nil {
		return nil, err
	}

	return sv, nil
}

// Update an existing Sentinel version.
func (a *adminSentinelVersions) Update(ctx context.Context, id string, options AdminSentinelVersionUpdateOptions) (*AdminSentinelVersion, error) {
	if !validStringID(&id) {
		return nil, ErrInvalidSentinelVersionID
	}

	u := fmt.Sprintf("admin/sentinel-versions/%s", url.PathEscape(id))
	req, err := a.client.NewRequest("PATCH", u, &options)
	if err != nil {
		return nil, err
	}

	sv := &AdminSentinelVersion{}
	err = req.Do(ctx, sv)
	if err != nil {
		return nil, err
	}

	return sv, nil
}

// Delete a Sentinel version.
func (a *adminSentinelVersions) Delete(ctx context.Context, id string) error {
	if !validStringID(&id) {
		return ErrInvalidSentinelVersionID
	}

	u := fmt.Sprintf("admin/sentinel-versions/%s", url.PathEscape(id))
	req, err := a.client.NewRequest("DELETE", u, nil)
	if err != nil {
		return err
	}

	return req.Do(ctx, nil)
}

func (o AdminSentinelVersionCreateOptions) valid() error {
	if (reflect.DeepEqual(o, AdminSentinelVersionCreateOptions{})) {
		return ErrRequiredSentinelVerCreateOps
	}
	if o.Version == "" {
		return ErrRequiredVersion
	}
	if !o.validArch() {
		return ErrRequiredArchsOrURLAndSha
	}
	return nil
}

func (o AdminSentinelVersionCreateOptions) validArch() bool {
	if o.Archs == nil && o.URL != "" && o.SHA != "" {
		return true
	}

	for _, a := range o.Archs {
		if !validArch(a) {
			return false
		}
	}

	return true
}

// Compile-time proof of interface implementation.
var _ CostEstimationSettings = (*adminCostEstimationSettings)(nil)

// CostEstimationSettings describes all the cost estimation admin settings for the Admin Setting API.
// https://developer.hashicorp.com/terraform/enterprise/api-docs/admin/settings
type CostEstimationSettings interface {
	// Read returns the cost estimation settings.
	Read(ctx context.Context) (*AdminCostEstimationSetting, error)

	// Update updates the cost estimation settings.
	Update(ctx context.Context, options AdminCostEstimationSettingOptions) (*AdminCostEstimationSetting, error)
}

type adminCostEstimationSettings struct {
	client *Client
}

// AdminCostEstimationSetting represents the admin cost estimation settings.
type AdminCostEstimationSetting struct {
	ID                        string `jsonapi:"primary,cost-estimation-settings"`
	Enabled                   bool   `jsonapi:"attr,enabled"`
	AWSAccessKeyID            string `jsonapi:"attr,aws-access-key-id"`
	AWSAccessKey              string `jsonapi:"attr,aws-secret-key"`
	AWSEnabled                bool   `jsonapi:"attr,aws-enabled"`
	AWSInstanceProfileEnabled bool   `jsonapi:"attr,aws-instance-profile-enabled"`
	GCPCredentials            string `jsonapi:"attr,gcp-credentials"`
	GCPEnabled                bool   `jsonapi:"attr,gcp-enabled"`
	AzureEnabled              bool   `jsonapi:"attr,azure-enabled"`
	AzureClientID             string `jsonapi:"attr,azure-client-id"`
	AzureClientSecret         string `jsonapi:"attr,azure-client-secret"`
	AzureSubscriptionID       string `jsonapi:"attr,azure-subscription-id"`
	AzureTenantID             string `jsonapi:"attr,azure-tenant-id"`
}

// AdminCostEstimationSettingOptions represents the admin options for updating
// the cost estimation settings.
// https://developer.hashicorp.com/terraform/enterprise/api-docs/admin/settings#request-body-1
type AdminCostEstimationSettingOptions struct {
	Enabled             *bool   `jsonapi:"attr,enabled,omitempty"`
	AWSAccessKeyID      *string `jsonapi:"attr,aws-access-key-id,omitempty"`
	AWSAccessKey        *string `jsonapi:"attr,aws-secret-key,omitempty"`
	GCPCredentials      *string `jsonapi:"attr,gcp-credentials,omitempty"`
	AzureClientID       *string `jsonapi:"attr,azure-client-id,omitempty"`
	AzureClientSecret   *string `jsonapi:"attr,azure-client-secret,omitempty"`
	AzureSubscriptionID *string `jsonapi:"attr,azure-subscription-id,omitempty"`
	AzureTenantID       *string `jsonapi:"attr,azure-tenant-id,omitempty"`
}

// Read returns the cost estimation settings.
func (a *adminCostEstimationSettings) Read(ctx context.Context) (*AdminCostEstimationSetting, error) {
	req, err := a.client.NewRequest("GET", "admin/cost-estimation-settings", nil)
	if err != nil {
		return nil, err
	}

	ace := &AdminCostEstimationSetting{}
	err = req.Do(ctx, ace)
	if err != nil {
		return nil, err
	}

	return ace, nil
}

// Update updates the cost-estimation settings.
func (a *adminCostEstimationSettings) Update(ctx context.Context, options AdminCostEstimationSettingOptions) (*AdminCostEstimationSetting, error) {
	req, err := a.client.NewRequest("PATCH", "admin/cost-estimation-settings", &options)
	if err != nil {
		return nil, err
	}

	ace := &AdminCostEstimationSetting{}
	err = req.Do(ctx, ace)
	if err != nil {
		return nil, err
	}

	return ace, nil
}

// Compile-time proof of interface implementation.
var _ CustomizationSettings = (*adminCustomizationSettings)(nil)

// CustomizationSettings describes all the Customization admin settings.
type CustomizationSettings interface {
	// Read returns the customization settings.
	Read(ctx context.Context) (*AdminCustomizationSetting, error)

	// Update updates the customization settings.
	Update(ctx context.Context, options AdminCustomizationSettingsUpdateOptions) (*AdminCustomizationSetting, error)
}

type adminCustomizationSettings struct {
	client *Client
}

// AdminCustomizationSetting represents the Customization settings in Terraform Enterprise for the Admin Settings API.
// https://developer.hashicorp.com/terraform/enterprise/api-docs/admin/settings
type AdminCustomizationSetting struct {
	ID           string `jsonapi:"primary,customization-settings"`
	SupportEmail string `jsonapi:"attr,support-email-address"`
	LoginHelp    string `jsonapi:"attr,login-help"`
	Footer       string `jsonapi:"attr,footer"`
	Error        string `jsonapi:"attr,error"`
	NewUser      string `jsonapi:"attr,new-user"`
}

// Read returns the Customization settings.
func (a *adminCustomizationSettings) Read(ctx context.Context) (*AdminCustomizationSetting, error) {
	req, err := a.client.NewRequest("GET", "admin/customization-settings", nil)
	if err != nil {
		return nil, err
	}

	cs := &AdminCustomizationSetting{}
	err = req.Do(ctx, cs)
	if err != nil {
		return nil, err
	}

	return cs, nil
}

// AdminCustomizationSettingsUpdateOptions represents the admin options for updating
// Customization settings.
// https://developer.hashicorp.com/terraform/enterprise/api-docs/admin/settings#request-body-6
type AdminCustomizationSettingsUpdateOptions struct {
	SupportEmail *string `jsonapi:"attr,support-email-address,omitempty"`
	LoginHelp    *string `jsonapi:"attr,login-help,omitempty"`
	Footer       *string `jsonapi:"attr,footer,omitempty"`
	Error        *string `jsonapi:"attr,error,omitempty"`
	NewUser      *string `jsonapi:"attr,new-user,omitempty"`
}

// Update updates the customization settings.
func (a *adminCustomizationSettings) Update(ctx context.Context, options AdminCustomizationSettingsUpdateOptions) (*AdminCustomizationSetting, error) {
	req, err := a.client.NewRequest("PATCH", "admin/customization-settings", &options)
	if err != nil {
		return nil, err
	}

	cs := &AdminCustomizationSetting{}
	err = req.Do(ctx, cs)
	if err != nil {
		return nil, err
	}

	return cs, nil
}

// Compile-time proof of interface implementation.
var _ GeneralSettings = (*adminGeneralSettings)(nil)

// GeneralSettings describes the general admin settings for the Admin Setting API.
// https://developer.hashicorp.com/terraform/enterprise/api-docs/admin/settings
type GeneralSettings interface {
	// Read returns the general settings
	Read(ctx context.Context) (*AdminGeneralSetting, error)

	// Update updates general settings.
	Update(ctx context.Context, options AdminGeneralSettingsUpdateOptions) (*AdminGeneralSetting, error)
}

type adminGeneralSettings struct {
	client *Client
}

// AdminGeneralSetting represents a the general settings in Terraform Enterprise.
type AdminGeneralSetting struct {
	ID                               string `jsonapi:"primary,general-settings"`
	LimitUserOrganizationCreation    bool   `jsonapi:"attr,limit-user-organization-creation"`
	APIRateLimitingEnabled           bool   `jsonapi:"attr,api-rate-limiting-enabled"`
	APIRateLimit                     int    `jsonapi:"attr,api-rate-limit"`
	SendPassingStatusesEnabled       bool   `jsonapi:"attr,send-passing-statuses-for-untriggered-speculative-plans"`
	AllowSpeculativePlansOnPR        bool   `jsonapi:"attr,allow-speculative-plans-on-pull-requests-from-forks"`
	RequireTwoFactorForAdmin         bool   `jsonapi:"attr,require-two-factor-for-admins"`
	FairRunQueuingEnabled            bool   `jsonapi:"attr,fair-run-queuing-enabled"`
	LimitOrgsPerUser                 bool   `jsonapi:"attr,limit-organizations-per-user"`
	DefaultOrgsPerUserCeiling        int    `jsonapi:"attr,default-organizations-per-user-ceiling"`
	LimitWorkspacesPerOrg            bool   `jsonapi:"attr,limit-workspaces-per-organization"`
	DefaultWorkspacesPerOrgCeiling   int    `jsonapi:"attr,default-workspaces-per-organization-ceiling"`
	TerraformBuildWorkerApplyTimeout string `jsonapi:"attr,terraform-build-worker-apply-timeout"`
	TerraformBuildWorkerPlanTimeout  string `jsonapi:"attr,terraform-build-worker-plan-timeout"`
	ApplyTimeout                     string `jsonapi:"attr,apply-timeout"`
	PlanTimeout                      string `jsonapi:"attr,plan-timeout"`
	DefaultRemoteStateAccess         bool   `jsonapi:"attr,default-remote-state-access"`
}

// AdminGeneralSettingsUpdateOptions represents the admin options for updating
// general settings.
// https://developer.hashicorp.com/terraform/enterprise/api-docs/admin/settings#request-body
type AdminGeneralSettingsUpdateOptions struct {
	LimitUserOrgCreation              *bool   `jsonapi:"attr,limit-user-organization-creation,omitempty"`
	APIRateLimitingEnabled            *bool   `jsonapi:"attr,api-rate-limiting-enabled,omitempty"`
	APIRateLimit                      *int    `jsonapi:"attr,api-rate-limit,omitempty"`
	SendPassingStatusUntriggeredPlans *bool   `jsonapi:"attr,send-passing-statuses-for-untriggered-speculative-plans,omitempty"`
	AllowSpeculativePlansOnPR         *bool   `jsonapi:"attr,allow-speculative-plans-on-pull-requests-from-forks,omitempty"`
	DefaultRemoteStateAccess          *bool   `jsonapi:"attr,default-remote-state-access,omitempty"`
	ApplyTimeout                      *string `jsonapi:"attr,apply-timeout"`
	PlanTimeout                       *string `jsonapi:"attr,plan-timeout"`
}

// Read returns the general settings.
func (a *adminGeneralSettings) Read(ctx context.Context) (*AdminGeneralSetting, error) {
	req, err := a.client.NewRequest("GET", "admin/general-settings", nil)
	if err != nil {
		return nil, err
	}

	ags := &AdminGeneralSetting{}
	err = req.Do(ctx, ags)
	if err != nil {
		return nil, err
	}

	return ags, nil
}

// Update updates the general settings.
func (a *adminGeneralSettings) Update(ctx context.Context, options AdminGeneralSettingsUpdateOptions) (*AdminGeneralSetting, error) {
	req, err := a.client.NewRequest("PATCH", "admin/general-settings", &options)
	if err != nil {
		return nil, err
	}

	ags := &AdminGeneralSetting{}
	err = req.Do(ctx, ags)
	if err != nil {
		return nil, err
	}

	return ags, nil
}

// Compile-time proof of interface implementation.
var _ OIDCSettings = (*adminOIDCSettings)(nil)

// OidcSettings describes all the OIDC admin settings for the Admin Setting API.
// https://developer.hashicorp.com/terraform/enterprise/api-docs/admin/settings
type OIDCSettings interface {
	// Rotate the key used for signing OIDC tokens for workload identity
	RotateKey(ctx context.Context) error

	// Trim old version of the key used for signing OIDC tokens for workload identity
	TrimKey(ctx context.Context) error
}

type adminOIDCSettings struct {
	client *Client
}

// Rotate the key used for signing OIDC tokens for workload identity
func (a *adminOIDCSettings) RotateKey(ctx context.Context) error {
	req, err := a.client.NewRequest("POST", "admin/oidc-settings/actions/rotate-key", nil)
	if err != nil {
		return err
	}

	return req.Do(ctx, nil)
}

// Trim old version of the key used for signing OIDC tokens for workload identity
func (a *adminOIDCSettings) TrimKey(ctx context.Context) error {
	req, err := a.client.NewRequest("POST", "admin/oidc-settings/actions/trim-key", nil)
	if err != nil {
		return err
	}

	return req.Do(ctx, nil)
}

// Copyright IBM Corp. 2018, 2026
// SPDX-License-Identifier: MPL-2.0

// Compile-time proof of interface implementation.
var _ SAMLSettings = (*adminSAMLSettings)(nil)

// SAMLSettings describes all the SAML admin settings for the Admin Setting API.
// https://developer.hashicorp.com/terraform/enterprise/api-docs/admin/settings
type SAMLSettings interface {
	// Read returns the SAML settings.
	Read(ctx context.Context) (*AdminSAMLSetting, error)

	// Update updates the SAML settings.
	Update(ctx context.Context, options AdminSAMLSettingsUpdateOptions) (*AdminSAMLSetting, error)

	// RevokeIdpCert revokes the older IdP certificate when the new IdP
	// certificate is known to be functioning correctly.
	RevokeIdpCert(ctx context.Context) (*AdminSAMLSetting, error)
}

type adminSAMLSettings struct {
	client *Client
}

// SAMLProviderType represents the SAML identity provider type.
type SAMLProviderType string

// SAMLProviderType constants define the supported SAML identity provider types.
const (
	SAMLProviderTypeOkta    SAMLProviderType = "okta"
	SAMLProviderTypeEntra   SAMLProviderType = "entra"
	SAMLProviderTypeGeneric SAMLProviderType = "saml"
	SAMLProviderTypeUnknown SAMLProviderType = "unknown"
)

// AdminSAMLSetting represents the SAML settings in Terraform Enterprise.
type AdminSAMLSetting struct {
	ID                        string           `jsonapi:"primary,saml-settings"`
	Enabled                   bool             `jsonapi:"attr,enabled"`
	Debug                     bool             `jsonapi:"attr,debug"`
	AuthnRequestsSigned       bool             `jsonapi:"attr,authn-requests-signed"`
	WantAssertionsSigned      bool             `jsonapi:"attr,want-assertions-signed"`
	TeamManagementEnabled     bool             `jsonapi:"attr,team-management-enabled"`
	OldIDPCert                string           `jsonapi:"attr,old-idp-cert"`
	IDPCert                   string           `jsonapi:"attr,idp-cert"`
	SLOEndpointURL            string           `jsonapi:"attr,slo-endpoint-url"`
	SSOEndpointURL            string           `jsonapi:"attr,sso-endpoint-url"`
	AttrUsername              string           `jsonapi:"attr,attr-username"`
	AttrGroups                string           `jsonapi:"attr,attr-groups"`
	AttrSiteAdmin             string           `jsonapi:"attr,attr-site-admin"`
	SiteAdminRole             string           `jsonapi:"attr,site-admin-role"`
	SSOAPITokenSessionTimeout int              `jsonapi:"attr,sso-api-token-session-timeout"`
	ACSConsumerURL            string           `jsonapi:"attr,acs-consumer-url"`
	MetadataURL               string           `jsonapi:"attr,metadata-url"`
	Certificate               string           `jsonapi:"attr,certificate"`
	PrivateKey                string           `jsonapi:"attr,private-key"`
	SignatureSigningMethod    string           `jsonapi:"attr,signature-signing-method"`
	SignatureDigestMethod     string           `jsonapi:"attr,signature-digest-method"`
	ProviderType              SAMLProviderType `jsonapi:"attr,provider-type"`
}

// Read returns the SAML settings.
func (a *adminSAMLSettings) Read(ctx context.Context) (*AdminSAMLSetting, error) {
	req, err := a.client.NewRequest("GET", "admin/saml-settings", nil)
	if err != nil {
		return nil, err
	}

	saml := &AdminSAMLSetting{}
	err = req.Do(ctx, saml)
	if err != nil {
		return nil, err
	}

	return saml, nil
}

// AdminSAMLSettingsUpdateOptions represents the admin options for updating
// SAML settings.
// https://developer.hashicorp.com/terraform/enterprise/api-docs/admin/settings#request-body-2
type AdminSAMLSettingsUpdateOptions struct {
	Enabled                   *bool             `jsonapi:"attr,enabled,omitempty"`
	Debug                     *bool             `jsonapi:"attr,debug,omitempty"`
	IDPCert                   *string           `jsonapi:"attr,idp-cert,omitempty"`
	Certificate               *string           `jsonapi:"attr,certificate,omitempty"`
	PrivateKey                *string           `jsonapi:"attr,private-key,omitempty"`
	SLOEndpointURL            *string           `jsonapi:"attr,slo-endpoint-url,omitempty"`
	SSOEndpointURL            *string           `jsonapi:"attr,sso-endpoint-url,omitempty"`
	AttrUsername              *string           `jsonapi:"attr,attr-username,omitempty"`
	AttrGroups                *string           `jsonapi:"attr,attr-groups,omitempty"`
	AttrSiteAdmin             *string           `jsonapi:"attr,attr-site-admin,omitempty"`
	SiteAdminRole             *string           `jsonapi:"attr,site-admin-role,omitempty"`
	SSOAPITokenSessionTimeout *int              `jsonapi:"attr,sso-api-token-session-timeout,omitempty"`
	TeamManagementEnabled     *bool             `jsonapi:"attr,team-management-enabled,omitempty"`
	AuthnRequestsSigned       *bool             `jsonapi:"attr,authn-requests-signed,omitempty"`
	WantAssertionsSigned      *bool             `jsonapi:"attr,want-assertions-signed,omitempty"`
	SignatureSigningMethod    *string           `jsonapi:"attr,signature-signing-method,omitempty"`
	SignatureDigestMethod     *string           `jsonapi:"attr,signature-digest-method,omitempty"`
	ProviderType              *SAMLProviderType `jsonapi:"attr,provider-type,omitempty"`
}

// Update updates the SAML settings.
func (a *adminSAMLSettings) Update(ctx context.Context, options AdminSAMLSettingsUpdateOptions) (*AdminSAMLSetting, error) {
	if options.ProviderType != nil {
		switch *options.ProviderType {
		case SAMLProviderTypeOkta, SAMLProviderTypeEntra, SAMLProviderTypeGeneric, SAMLProviderTypeUnknown:
		default:
			return nil, ErrInvalidSAMLProviderType
		}
	}

	req, err := a.client.NewRequest("PATCH", "admin/saml-settings", &options)
	if err != nil {
		return nil, err
	}

	saml := &AdminSAMLSetting{}
	err = req.Do(ctx, saml)
	if err != nil {
		return nil, err
	}

	return saml, nil
}

// RevokeIdpCert revokes the older IdP certificate when the new IdP
// certificate is known to be functioning correctly.
func (a *adminSAMLSettings) RevokeIdpCert(ctx context.Context) (*AdminSAMLSetting, error) {
	req, err := a.client.NewRequest("POST", "admin/saml-settings/actions/revoke-old-certificate", nil)
	if err != nil {
		return nil, err
	}

	saml := &AdminSAMLSetting{}
	err = req.Do(ctx, saml)
	if err != nil {
		return nil, err
	}

	return saml, nil
}

// Copyright IBM Corp. 2018, 2026
// SPDX-License-Identifier: MPL-2.0

var _ AdminSCIMGroupMappings = (*adminSCIMGroupMappings)(nil)

// AdminSCIMGroupMappings describes all the SCIM group mapping related methods that the Terraform
// Enterprise API supports
//
// TFE API docs: https://developer.hashicorp.com/terraform/enterprise/api-docs/admin/team-scim-group-mapping
type AdminSCIMGroupMappings interface {
	// Create a SCIM group mapping.
	Create(ctx context.Context, teamID string, options *AdminSCIMGroupMappingCreateOptions) error

	// Update a SCIM group mapping.
	Update(ctx context.Context, teamID string, options *AdminSCIMGroupMappingUpdateOptions) error

	// Delete a SCIM group mapping.
	Delete(ctx context.Context, teamID string) error
}

// adminSCIMGroupMappings implements AdminSCIMGroupMappings
type adminSCIMGroupMappings struct {
	client *Client
}

// AdminSCIMGroupMappingCreateOptions represents the options for creating a SCIM group mapping
type AdminSCIMGroupMappingCreateOptions struct {
	// Type is a public field utilized by JSON:API to
	// set the resource type via the field tag.
	// It is not a user-defined value and does not need to be set.
	// https://jsonapi.org/format/#crud-creating
	Type        string `jsonapi:"primary,scim-group-mappings"`
	SCIMGroupID string `jsonapi:"attr,scim-group-id"`
}

// AdminSCIMGroupMappingUpdateOptions represents the options for updating a SCIM group mapping
type AdminSCIMGroupMappingUpdateOptions struct {
	// Type is a public field utilized by JSON:API to
	// set the resource type via the field tag.
	// It is not a user-defined value and does not need to be set.
	// https://jsonapi.org/format/#crud-creating
	Type           string `jsonapi:"primary,scim-group-mappings"`
	SCIMSyncPaused *bool  `jsonapi:"attr,scim-sync-paused"`
}

// Create a SCIM group mapping.
func (a *adminSCIMGroupMappings) Create(ctx context.Context, teamID string, options *AdminSCIMGroupMappingCreateOptions) error {
	if !validStringID(&teamID) {
		return ErrInvalidTeamID
	}
	if options == nil {
		return ErrRequiredSCIMGroupMappingCreateOps
	}
	if !validStringID(&options.SCIMGroupID) {
		return ErrInvalidSCIMGroupID
	}

	req, err := a.client.NewRequest("POST", fmt.Sprintf(AdminSCIMGroupMappingPath, url.PathEscape(teamID)), options)
	if err != nil {
		return err
	}

	return req.Do(ctx, nil)
}

// Update a SCIM group mapping.
func (a *adminSCIMGroupMappings) Update(ctx context.Context, teamID string, options *AdminSCIMGroupMappingUpdateOptions) error {
	if !validStringID(&teamID) {
		return ErrInvalidTeamID
	}

	if options == nil {
		return ErrRequiredSCIMGroupMappingUpdateOps
	}

	if options.SCIMSyncPaused == nil {
		return ErrSCIMSyncPausedNil
	}

	req, err := a.client.NewRequest("PATCH", fmt.Sprintf(AdminSCIMGroupMappingPath, url.PathEscape(teamID)), options)
	if err != nil {
		return err
	}

	return req.Do(ctx, nil)
}

// Delete a SCIM group mapping.
func (a *adminSCIMGroupMappings) Delete(ctx context.Context, teamID string) error {
	if !validStringID(&teamID) {
		return ErrInvalidTeamID
	}

	req, err := a.client.NewRequest("DELETE", fmt.Sprintf(AdminSCIMGroupMappingPath, url.PathEscape(teamID)), nil)
	if err != nil {
		return err
	}

	return req.Do(ctx, nil)
}

// Copyright IBM Corp. 2018, 2026
// SPDX-License-Identifier: MPL-2.0

var _ AdminSCIMGroups = (*adminSCIMGroups)(nil)

// AdminSCIMGroups describes all the SCIM group related methods that the Terraform
// Enterprise API supports
//
// TFE API docs: https://developer.hashicorp.com/terraform/enterprise/api-docs/admin/scim-groups
type AdminSCIMGroups interface {
	// List all SCIM groups.
	List(ctx context.Context, options *AdminSCIMGroupListOptions) (*AdminSCIMGroupList, error)
}

// adminSCIMGroups implements AdminSCIMGroups
type adminSCIMGroups struct {
	client *Client
}

// AdminSCIMGroupList represents a list of SCIM groups
type AdminSCIMGroupList struct {
	*Pagination
	Items []*AdminSCIMGroup
}

// AdminSCIMGroup represents a Terraform Enterprise SCIM group
type AdminSCIMGroup struct {
	ID   string `jsonapi:"primary,scim-groups"`
	Name string `jsonapi:"attr,name"`
}

// AdminSCIMGroupListOptions represents the options for listing SCIM groups
type AdminSCIMGroupListOptions struct {
	ListOptions
	Query string `url:"q,omitempty"`
}

func (o *AdminSCIMGroupListOptions) valid() error {
	if o == nil {
		return nil
	}
	if o.PageNumber < 0 || o.PageSize < 0 {
		return ErrInvalidPagination
	}

	return nil
}

// List all SCIM groups.
func (a *adminSCIMGroups) List(ctx context.Context, options *AdminSCIMGroupListOptions) (*AdminSCIMGroupList, error) {
	if err := options.valid(); err != nil {
		return nil, err
	}
	req, err := a.client.NewRequest("GET", AdminSCIMGroupsPath, options)
	if err != nil {
		return nil, err
	}

	scimGroups := &AdminSCIMGroupList{}
	err = req.Do(ctx, scimGroups)
	if err != nil {
		return nil, err
	}

	return scimGroups, nil
}

// Copyright IBM Corp. 2018, 2026
// SPDX-License-Identifier: MPL-2.0

var _ AdminSCIMTokens = (*adminSCIMTokens)(nil)

// AdminSCIMTokens describes all the Admin SCIM token related methods that the Terraform
// Enterprise API supports
//
// TFE API docs: https://developer.hashicorp.com/terraform/enterprise/api-docs/admin/scim-tokens
type AdminSCIMTokens interface {
	// List all Admin SCIM tokens.
	List(ctx context.Context) (*AdminSCIMTokenList, error)

	// Create an Admin SCIM token.
	Create(ctx context.Context, description string) (*AdminSCIMToken, error)

	// Create an Admin SCIM token with options.
	CreateWithOptions(ctx context.Context, options AdminSCIMTokenCreateOptions) (*AdminSCIMToken, error)

	// Read an Admin SCIM token by its ID.
	Read(ctx context.Context, scimTokenID string) (*AdminSCIMToken, error)

	// Delete an Admin SCIM token.
	Delete(ctx context.Context, scimTokenID string) error
}

// adminSCIMTokens implements AdminSCIMTokens
type adminSCIMTokens struct {
	client *Client
}

// AdminSCIMTokenList represents a list of Admin SCIM tokens
type AdminSCIMTokenList struct {
	Items []*AdminSCIMToken
}

// AdminSCIMToken represents a Terraform Enterprise Admin SCIM token.
type AdminSCIMToken struct {
	ID          string    `jsonapi:"primary,authentication-tokens"`
	CreatedAt   time.Time `jsonapi:"attr,created-at,iso8601"`
	LastUsedAt  time.Time `jsonapi:"attr,last-used-at,iso8601"`
	ExpiredAt   time.Time `jsonapi:"attr,expired-at,iso8601"`
	Description string    `jsonapi:"attr,description"`
	Token       string    `jsonapi:"attr,token,omitempty"`
}

// AdminSCIMTokenCreateOptions represents the options for creating an Admin SCIM token
type AdminSCIMTokenCreateOptions struct {
	// Required: A human-readable description of the token's purpose
	// (for example, Okta SCIM Integration).
	Description *string `jsonapi:"attr,description"`

	// Optional: Optional ISO-8601 timestamp for token expiration.
	// Defaults to 365 days in the future. Must be between 29 and 365 days in the future.
	ExpiredAt *time.Time `jsonapi:"attr,expired-at,iso8601,omitempty"`
}

// List all Admin SCIM tokens.
func (a *adminSCIMTokens) List(ctx context.Context) (*AdminSCIMTokenList, error) {
	req, err := a.client.NewRequest("GET", AdminSCIMTokensPath, nil)
	if err != nil {
		return nil, err
	}

	scimTokens := &AdminSCIMTokenList{}
	err = req.Do(ctx, scimTokens)
	if err != nil {
		return nil, err
	}
	return scimTokens, nil
}

// Create an Admin SCIM token.
func (a *adminSCIMTokens) Create(ctx context.Context, description string) (*AdminSCIMToken, error) {
	return a.CreateWithOptions(ctx, AdminSCIMTokenCreateOptions{
		Description: &description,
	})
}

// Create an Admin SCIM token with options.
func (a *adminSCIMTokens) CreateWithOptions(ctx context.Context, options AdminSCIMTokenCreateOptions) (*AdminSCIMToken, error) {
	if !validString(options.Description) {
		return nil, ErrSCIMTokenDescription
	}
	req, err := a.client.NewRequest("POST", AdminSCIMTokensPath, &options)
	if err != nil {
		return nil, err
	}
	scimToken := &AdminSCIMToken{}
	err = req.Do(ctx, scimToken)
	if err != nil {
		return nil, err
	}
	return scimToken, nil
}

// Read an Admin SCIM token by its ID.
func (a *adminSCIMTokens) Read(ctx context.Context, scimTokenID string) (*AdminSCIMToken, error) {
	if !validStringID(&scimTokenID) {
		return nil, ErrInvalidTokenID
	}
	u := fmt.Sprintf("%s/%s", AdminSCIMTokensPath, url.PathEscape(scimTokenID))
	req, err := a.client.NewRequest("GET", u, nil)
	if err != nil {
		return nil, err
	}
	scimToken := &AdminSCIMToken{}
	err = req.Do(ctx, scimToken)
	if err != nil {
		return nil, err
	}
	return scimToken, nil
}

// Delete an Admin SCIM token.
func (a *adminSCIMTokens) Delete(ctx context.Context, scimTokenID string) error {
	if !validStringID(&scimTokenID) {
		return ErrInvalidTokenID
	}
	u := fmt.Sprintf(AuthenticationTokensPath, url.PathEscape(scimTokenID))
	req, err := a.client.NewRequest("DELETE", u, nil)
	if err != nil {
		return err
	}
	return req.Do(ctx, nil)
}

// Copyright IBM Corp. 2018, 2026
// SPDX-License-Identifier: MPL-2.0

// Compile-time proof of interface implementation.
var _ SCIMSettings = (*adminSCIMSettings)(nil)

// SCIMSettings describes all the scim settings related methods that the Terraform
// Enterprise API supports
//
// TFE API docs: https://developer.hashicorp.com/terraform/enterprise/api-docs/admin/settings
type SCIMSettings interface {
	// Read scim settings
	Read(ctx context.Context) (*AdminSCIMSetting, error)

	// Update scim settings
	Update(ctx context.Context, options AdminSCIMSettingUpdateOptions) (*AdminSCIMSetting, error)

	// Delete scim settings
	Delete(ctx context.Context) error
}

// adminSCIMSettings implements SCIMSettings.
type adminSCIMSettings struct {
	client *Client
}

// AdminSCIMSetting represents the SCIM setting in Terraform Enterprise
type AdminSCIMSetting struct {
	ID                        string `jsonapi:"primary,scim-settings"`
	Enabled                   bool   `jsonapi:"attr,enabled"`
	Paused                    bool   `jsonapi:"attr,paused"`
	SiteAdminGroupSCIMID      string `jsonapi:"attr,site-admin-group-scim-id"`
	SiteAdminGroupDisplayName string `jsonapi:"attr,site-admin-group-display-name"`
}

// AdminSCIMSettingUpdateOptions represents the options for updating an admin SCIM setting.
//
// SiteAdminGroupSCIMID is a nullable attribute. Use NullableString(value) to set the
// site admin group, or NullString() to explicitly clear (unlink) it. Leaving the
// field as its zero value omits it from the request so the existing value on
// the server is preserved.
type AdminSCIMSettingUpdateOptions struct {
	// Enabled toggles SCIM provisioning on or off.
	Enabled *bool `jsonapi:"attr,enabled,omitempty"`
	// Paused toggles whether SCIM provisioning is paused.
	Paused *bool `jsonapi:"attr,paused,omitempty"`
	// SiteAdminGroupSCIMID sets the SCIM group linked to the site admin role.
	// Use NullableString(value) to link a group, or NullString() to explicitly unlink it.
	// Leaving it as the zero value omits the field so the server-side value is preserved.
	SiteAdminGroupSCIMID jsonapi.NullableAttr[string] `jsonapi:"attr,site-admin-group-scim-id,omitempty"`
}

// Read scim setting.
func (a *adminSCIMSettings) Read(ctx context.Context) (*AdminSCIMSetting, error) {
	req, err := a.client.NewRequest("GET", "admin/scim-settings", nil)
	if err != nil {
		return nil, err
	}

	scim := &AdminSCIMSetting{}
	err = req.Do(ctx, scim)
	if err != nil {
		return nil, err
	}

	return scim, nil
}

// Update scim setting.
func (a *adminSCIMSettings) Update(ctx context.Context, options AdminSCIMSettingUpdateOptions) (*AdminSCIMSetting, error) {
	req, err := a.client.NewRequest("PATCH", "admin/scim-settings", &options)
	if err != nil {
		return nil, err
	}

	scim := &AdminSCIMSetting{}
	err = req.Do(ctx, scim)
	if err != nil {
		return nil, err
	}
	return scim, nil
}

// Delete scim setting.
func (a *adminSCIMSettings) Delete(ctx context.Context) error {
	req, err := a.client.NewRequest("DELETE", "admin/scim-settings", nil)
	if err != nil {
		return err
	}

	return req.Do(ctx, nil)
}

// Compile-time proof of interface implementation.
var _ SMTPSettings = (*adminSMTPSettings)(nil)

// SMTPSettings describes all the SMTP admin settings for the Admin Setting API https://developer.hashicorp.com/terraform/enterprise/api-docs/admin/settings
type SMTPSettings interface {
	// Read returns the SMTP settings.
	Read(ctx context.Context) (*AdminSMTPSetting, error)

	// Update updates SMTP settings.
	Update(ctx context.Context, options AdminSMTPSettingsUpdateOptions) (*AdminSMTPSetting, error)
}

type adminSMTPSettings struct {
	client *Client
}

// SMTPAuthType represents valid SMTP Auth types.
type SMTPAuthType string

// List of all SMTP auth types.
const (
	SMTPAuthNone  SMTPAuthType = "none"
	SMTPAuthPlain SMTPAuthType = "plain"
	SMTPAuthLogin SMTPAuthType = "login"
)

// AdminSMTPSetting represents a the SMTP settings in Terraform Enterprise.
type AdminSMTPSetting struct {
	ID       string       `jsonapi:"primary,smtp-settings"`
	Enabled  bool         `jsonapi:"attr,enabled"`
	Host     string       `jsonapi:"attr,host"`
	Port     int          `jsonapi:"attr,port"`
	Sender   string       `jsonapi:"attr,sender"`
	Auth     SMTPAuthType `jsonapi:"attr,auth"`
	Username string       `jsonapi:"attr,username"`
}

// Read returns the SMTP settings.
func (a *adminSMTPSettings) Read(ctx context.Context) (*AdminSMTPSetting, error) {
	req, err := a.client.NewRequest("GET", "admin/smtp-settings", nil)
	if err != nil {
		return nil, err
	}

	smtp := &AdminSMTPSetting{}
	err = req.Do(ctx, smtp)
	if err != nil {
		return nil, err
	}

	return smtp, nil
}

// AdminSMTPSettingsUpdateOptions represents the admin options for updating
// SMTP settings.
// https://developer.hashicorp.com/terraform/enterprise/api-docs/admin/settings#request-body-3
type AdminSMTPSettingsUpdateOptions struct {
	Enabled          *bool         `jsonapi:"attr,enabled,omitempty"`
	Host             *string       `jsonapi:"attr,host,omitempty"`
	Port             *int          `jsonapi:"attr,port,omitempty"`
	Sender           *string       `jsonapi:"attr,sender,omitempty"`
	Auth             *SMTPAuthType `jsonapi:"attr,auth,omitempty"`
	Username         *string       `jsonapi:"attr,username,omitempty"`
	Password         *string       `jsonapi:"attr,password,omitempty"`
	TestEmailAddress *string       `jsonapi:"attr,test-email-address,omitempty"`
}

// Update updates the SMTP settings.
func (a *adminSMTPSettings) Update(ctx context.Context, options AdminSMTPSettingsUpdateOptions) (*AdminSMTPSetting, error) {
	if err := options.valid(); err != nil {
		return nil, err
	}

	req, err := a.client.NewRequest("PATCH", "admin/smtp-settings", &options)
	if err != nil {
		return nil, err
	}

	smtp := &AdminSMTPSetting{}
	err = req.Do(ctx, smtp)
	if err != nil {
		return nil, err
	}

	return smtp, nil
}

func (o AdminSMTPSettingsUpdateOptions) valid() error {
	if validString((*string)(o.Auth)) {
		if err := validateAdminSettingSMTPAuth(*o.Auth); err != nil {
			return err
		}
	}

	return nil
}

func validateAdminSettingSMTPAuth(authVal SMTPAuthType) error {
	switch authVal {
	case SMTPAuthNone, SMTPAuthPlain, SMTPAuthLogin:
		// do nothing
	default:
		return ErrInvalidSMTPAuth
	}

	return nil
}

// Compile-time proof of interface implementation.
var _ TwilioSettings = (*adminTwilioSettings)(nil)

// TwilioSettings describes all the Twilio admin settings for the Admin Setting API.
// https://developer.hashicorp.com/terraform/enterprise/api-docs/admin/settings
type TwilioSettings interface {
	// Read returns the Twilio settings.
	Read(ctx context.Context) (*AdminTwilioSetting, error)

	// Update updates Twilio settings.
	Update(ctx context.Context, options AdminTwilioSettingsUpdateOptions) (*AdminTwilioSetting, error)

	// Verify verifies Twilio settings.
	Verify(ctx context.Context, options AdminTwilioSettingsVerifyOptions) error
}

type adminTwilioSettings struct {
	client *Client
}

// AdminTwilioSetting represents the Twilio settings in Terraform Enterprise.
type AdminTwilioSetting struct {
	ID         string `jsonapi:"primary,twilio-settings"`
	Enabled    bool   `jsonapi:"attr,enabled"`
	AccountSid string `jsonapi:"attr,account-sid"`
	FromNumber string `jsonapi:"attr,from-number"`
}

// Read returns the Twilio settings.
func (a *adminTwilioSettings) Read(ctx context.Context) (*AdminTwilioSetting, error) {
	req, err := a.client.NewRequest("GET", "admin/twilio-settings", nil)
	if err != nil {
		return nil, err
	}

	twilio := &AdminTwilioSetting{}
	err = req.Do(ctx, twilio)
	if err != nil {
		return nil, err
	}

	return twilio, nil
}

// AdminTwilioSettingsUpdateOptions represents the admin options for updating
// Twilio settings.
// https://developer.hashicorp.com/terraform/enterprise/api-docs/admin/settings#request-body-4
type AdminTwilioSettingsUpdateOptions struct {
	Enabled    *bool   `jsonapi:"attr,enabled,omitempty"`
	AccountSid *string `jsonapi:"attr,account-sid,omitempty"`
	AuthToken  *string `jsonapi:"attr,auth-token,omitempty"`
	FromNumber *string `jsonapi:"attr,from-number,omitempty"`
}

// AdminTwilioSettingsVerifyOptions represents the test number to verify Twilio.
// https://developer.hashicorp.com/terraform/enterprise/api-docs/admin/settings#verify-twilio-settings
type AdminTwilioSettingsVerifyOptions struct {
	TestNumber *string `jsonapi:"attr,test-number"` // Required
}

// Update updates the Twilio settings.
func (a *adminTwilioSettings) Update(ctx context.Context, options AdminTwilioSettingsUpdateOptions) (*AdminTwilioSetting, error) {
	req, err := a.client.NewRequest("PATCH", "admin/twilio-settings", &options)
	if err != nil {
		return nil, err
	}

	twilio := &AdminTwilioSetting{}
	err = req.Do(ctx, twilio)
	if err != nil {
		return nil, err
	}

	return twilio, nil
}

// Verify verifies Twilio settings.
func (a *adminTwilioSettings) Verify(ctx context.Context, options AdminTwilioSettingsVerifyOptions) error {
	if err := options.valid(); err != nil {
		return err
	}
	req, err := a.client.NewRequest("PATCH", "admin/twilio-settings/verify", &options)
	if err != nil {
		return err
	}

	return req.Do(ctx, nil)
}

func (o AdminTwilioSettingsVerifyOptions) valid() error {
	if !validString(o.TestNumber) {
		return ErrRequiredTestNumber
	}

	return nil
}

// Copyright IBM Corp. 2018, 2026
// SPDX-License-Identifier: MPL-2.0

// SCIMResource groups the SCIM related resources together.
// This struct should be constructed with keyed fields only or obtained via the client
// to prevent breakages when new fields are added.
type SCIMResource struct {
	SCIMSettings
	Tokens            AdminSCIMTokens
	Groups            AdminSCIMGroups
	SCIMGroupMappings AdminSCIMGroupMappings
}

// AdminSettings describes all the admin settings related methods that the Terraform Enterprise API supports.
// Note that admin settings are only available in Terraform Enterprise.
//
// TFE API docs: https://developer.hashicorp.com/terraform/enterprise/api-docs/admin/settings
type AdminSettings struct {
	General        GeneralSettings
	SAML           SAMLSettings
	CostEstimation CostEstimationSettings
	SMTP           SMTPSettings
	Twilio         TwilioSettings
	Customization  CustomizationSettings
	OIDC           OIDCSettings
	SCIM           *SCIMResource
}

func newAdminSettings(client *Client) *AdminSettings {
	return &AdminSettings{
		General:        &adminGeneralSettings{client: client},
		SAML:           &adminSAMLSettings{client: client},
		CostEstimation: &adminCostEstimationSettings{client: client},
		SMTP:           &adminSMTPSettings{client: client},
		Twilio:         &adminTwilioSettings{client: client},
		Customization:  &adminCustomizationSettings{client: client},
		OIDC:           &adminOIDCSettings{client: client},
		SCIM: &SCIMResource{
			SCIMSettings:      &adminSCIMSettings{client: client},
			Tokens:            &adminSCIMTokens{client: client},
			Groups:            &adminSCIMGroups{client: client},
			SCIMGroupMappings: &adminSCIMGroupMappings{client: client},
		},
	}
}

// Compile-time proof of interface implementation.
var _ AdminTerraformVersions = (*adminTerraformVersions)(nil)

const (
	linux = "linux"
	amd64 = "amd64"
	arm64 = "arm64"
	s390x = "s390x"
)

// AdminTerraformVersions describes all the admin terraform versions related methods that
// the Terraform Enterprise API supports.
// Note that admin terraform versions are only available in Terraform Enterprise.
//
// TFE API docs: https://developer.hashicorp.com/terraform/enterprise/api-docs/admin/terraform-versions
type AdminTerraformVersions interface {
	// List all the terraform versions.
	List(ctx context.Context, options *AdminTerraformVersionsListOptions) (*AdminTerraformVersionsList, error)

	// Read a terraform version by its ID.
	Read(ctx context.Context, id string) (*AdminTerraformVersion, error)

	// Create a terraform version.
	Create(ctx context.Context, options AdminTerraformVersionCreateOptions) (*AdminTerraformVersion, error)

	// Update a terraform version.
	Update(ctx context.Context, id string, options AdminTerraformVersionUpdateOptions) (*AdminTerraformVersion, error)

	// Delete a terraform version
	Delete(ctx context.Context, id string) error
}

// adminTerraformVersions implements AdminTerraformVersions.
type adminTerraformVersions struct {
	client *Client
}

// AdminTerraformVersion represents a Terraform Version
type AdminTerraformVersion struct {
	ID               string                     `jsonapi:"primary,terraform-versions"`
	Version          string                     `jsonapi:"attr,version"`
	URL              string                     `jsonapi:"attr,url,omitempty"`
	Sha              string                     `jsonapi:"attr,sha,omitempty"`
	Deprecated       bool                       `jsonapi:"attr,deprecated"`
	DeprecatedReason *string                    `jsonapi:"attr,deprecated-reason,omitempty"`
	Official         bool                       `jsonapi:"attr,official"`
	Enabled          bool                       `jsonapi:"attr,enabled"`
	Beta             bool                       `jsonapi:"attr,beta"`
	Usage            int                        `jsonapi:"attr,usage"`
	CreatedAt        time.Time                  `jsonapi:"attr,created-at,iso8601"`
	Archs            []*ToolVersionArchitecture `jsonapi:"attr,archs,omitempty"`
}

type ToolVersionArchitecture struct {
	URL  string `jsonapi:"attr,url"`
	Sha  string `jsonapi:"attr,sha"`
	OS   string `jsonapi:"attr,os"`
	Arch string `jsonapi:"attr,arch"`
}

// AdminTerraformVersionsListOptions represents the options for listing
// terraform versions.
type AdminTerraformVersionsListOptions struct {
	ListOptions

	// Optional: A query string to find an exact version
	Filter string `url:"filter[version],omitempty"`

	// Optional: A search query string to find all versions that match version substring
	Search string `url:"search[version],omitempty"`
}

// AdminTerraformVersionCreateOptions for creating a terraform version.
// https://developer.hashicorp.com/terraform/enterprise/api-docs/admin/terraform-versions#request-body
type AdminTerraformVersionCreateOptions struct {
	Type             string                     `jsonapi:"primary,terraform-versions"`
	Version          *string                    `jsonapi:"attr,version"` // Required
	URL              *string                    `jsonapi:"attr,url,omitempty"`
	Sha              *string                    `jsonapi:"attr,sha,omitempty"`
	Official         *bool                      `jsonapi:"attr,official,omitempty"`
	Deprecated       *bool                      `jsonapi:"attr,deprecated,omitempty"`
	DeprecatedReason *string                    `jsonapi:"attr,deprecated-reason,omitempty"`
	Enabled          *bool                      `jsonapi:"attr,enabled,omitempty"`
	Beta             *bool                      `jsonapi:"attr,beta,omitempty"`
	Archs            []*ToolVersionArchitecture `jsonapi:"attr,archs,omitempty"`
}

// AdminTerraformVersionUpdateOptions for updating terraform version.
// https://developer.hashicorp.com/terraform/enterprise/api-docs/admin/terraform-versions#request-body
type AdminTerraformVersionUpdateOptions struct {
	Type             string                     `jsonapi:"primary,terraform-versions"`
	Version          *string                    `jsonapi:"attr,version,omitempty"`
	URL              *string                    `jsonapi:"attr,url,omitempty"`
	Sha              *string                    `jsonapi:"attr,sha,omitempty"`
	Official         *bool                      `jsonapi:"attr,official,omitempty"`
	Deprecated       *bool                      `jsonapi:"attr,deprecated,omitempty"`
	DeprecatedReason *string                    `jsonapi:"attr,deprecated-reason,omitempty"`
	Enabled          *bool                      `jsonapi:"attr,enabled,omitempty"`
	Beta             *bool                      `jsonapi:"attr,beta,omitempty"`
	Archs            []*ToolVersionArchitecture `jsonapi:"attr,archs,omitempty"`
}

// AdminTerraformVersionsList represents a list of terraform versions.
type AdminTerraformVersionsList struct {
	*Pagination
	Items []*AdminTerraformVersion
}

// List all the terraform versions.
func (a *adminTerraformVersions) List(ctx context.Context, options *AdminTerraformVersionsListOptions) (*AdminTerraformVersionsList, error) {
	req, err := a.client.NewRequest("GET", "admin/terraform-versions", options)
	if err != nil {
		return nil, err
	}

	tvl := &AdminTerraformVersionsList{}
	err = req.Do(ctx, tvl)
	if err != nil {
		return nil, err
	}

	return tvl, nil
}

// Read a terraform version by its ID.
func (a *adminTerraformVersions) Read(ctx context.Context, id string) (*AdminTerraformVersion, error) {
	if !validStringID(&id) {
		return nil, ErrInvalidTerraformVersionID
	}

	u := fmt.Sprintf("admin/terraform-versions/%s", url.PathEscape(id))
	req, err := a.client.NewRequest("GET", u, nil)
	if err != nil {
		return nil, err
	}

	tfv := &AdminTerraformVersion{}
	err = req.Do(ctx, tfv)
	if err != nil {
		return nil, err
	}

	return tfv, nil
}

// Create a new terraform version.
func (a *adminTerraformVersions) Create(ctx context.Context, options AdminTerraformVersionCreateOptions) (*AdminTerraformVersion, error) {
	if err := options.valid(); err != nil {
		return nil, err
	}
	req, err := a.client.NewRequest("POST", "admin/terraform-versions", &options)
	if err != nil {
		return nil, err
	}

	tfv := &AdminTerraformVersion{}
	err = req.Do(ctx, tfv)
	if err != nil {
		return nil, err
	}
	return tfv, nil
}

// Update an existing terraform version.
func (a *adminTerraformVersions) Update(ctx context.Context, id string, options AdminTerraformVersionUpdateOptions) (*AdminTerraformVersion, error) {
	if !validStringID(&id) {
		return nil, ErrInvalidTerraformVersionID
	}

	u := fmt.Sprintf("admin/terraform-versions/%s", url.PathEscape(id))
	req, err := a.client.NewRequest("PATCH", u, &options)
	if err != nil {
		return nil, err
	}

	tfv := &AdminTerraformVersion{}
	err = req.Do(ctx, tfv)
	if err != nil {
		return nil, err
	}

	return tfv, nil
}

// Delete a terraform version.
func (a *adminTerraformVersions) Delete(ctx context.Context, id string) error {
	if !validStringID(&id) {
		return ErrInvalidTerraformVersionID
	}

	u := fmt.Sprintf("admin/terraform-versions/%s", url.PathEscape(id))
	req, err := a.client.NewRequest("DELETE", u, nil)
	if err != nil {
		return err
	}

	return req.Do(ctx, nil)
}

func (o AdminTerraformVersionCreateOptions) valid() error {
	if (reflect.DeepEqual(o, AdminTerraformVersionCreateOptions{})) {
		return ErrRequiredTFVerCreateOps
	}
	if !validString(o.Version) {
		return ErrRequiredVersion
	}
	if !o.validArchs() {
		return ErrRequiredArchsOrURLAndSha
	}
	return nil
}

func (o AdminTerraformVersionCreateOptions) validArchs() bool {
	if o.Archs == nil && validString(o.URL) && validString(o.Sha) {
		return true
	}

	for _, a := range o.Archs {
		if !validArch(a) {
			return false
		}
	}

	return true
}

func validArch(a *ToolVersionArchitecture) bool {
	return a.URL != "" && a.Sha != "" && a.OS == linux && (a.Arch == amd64 || a.Arch == arm64 || a.Arch == s390x)
}

// Copyright IBM Corp. 2018, 2026
// SPDX-License-Identifier: MPL-2.0

// Compile-time proof of interface implementation.
var _ AdminUsers = (*adminUsers)(nil)

// AdminUsers describes all the admin user related methods that the Terraform
// Enterprise  API supports.
// It contains endpoints to help site administrators manage their users.
//
// TFE API docs: https://developer.hashicorp.com/terraform/enterprise/api-docs/admin/users
type AdminUsers interface {
	// List all the users of the given installation.
	List(ctx context.Context, options *AdminUserListOptions) (*AdminUserList, error)

	// Delete a user by its ID.
	Delete(ctx context.Context, userID string) error

	// Suspend a user by its ID.
	Suspend(ctx context.Context, userID string) (*AdminUser, error)

	// Unsuspend a user by its ID.
	Unsuspend(ctx context.Context, userID string) (*AdminUser, error)

	// GrantAdmin grants admin privileges to a user by its ID.
	GrantAdmin(ctx context.Context, userID string) (*AdminUser, error)

	// RevokeAdmin revokees admin privileges to a user by its ID.
	RevokeAdmin(ctx context.Context, userID string) (*AdminUser, error)

	// Disable2FA disables a user's two-factor authentication in the situation
	// where they have lost access to their device and recovery codes.
	Disable2FA(ctx context.Context, userID string) (*AdminUser, error)
}

// adminUsers implements the AdminUsers interface.
type adminUsers struct {
	client *Client
}

// AdminUser represents a user as seen by an Admin.
type AdminUser struct {
	ID               string     `jsonapi:"primary,users"`
	Email            string     `jsonapi:"attr,email"`
	Username         string     `jsonapi:"attr,username"`
	AvatarURL        string     `jsonapi:"attr,avatar-url"`
	TwoFactor        *TwoFactor `jsonapi:"attr,two-factor"`
	IsAdmin          bool       `jsonapi:"attr,is-admin"`
	IsSuspended      bool       `jsonapi:"attr,is-suspended"`
	IsServiceAccount bool       `jsonapi:"attr,is-service-account"`

	// Relations
	Organizations []*Organization `jsonapi:"relation,organizations"`

	// SCIM Attributes
	IsSCIMManaged *bool      `jsonapi:"attr,is-scim-managed"`
	SCIMUsername  *string    `jsonapi:"attr,scim-username"`
	SCIMUpdatedAt *time.Time `jsonapi:"attr,scim-updated-at,iso8601"`
}

// AdminUserList represents a list of users.
type AdminUserList struct {
	*Pagination
	Items []*AdminUser
}

// AdminUserIncludeOpt represents the available options for include query params.
// https://developer.hashicorp.com/terraform/enterprise/api-docs/admin/users#available-related-resources
type AdminUserIncludeOpt string

const AdminUserOrgs AdminUserIncludeOpt = "organizations"

// AdminUserListOptions represents the options for listing users.
// https://developer.hashicorp.com/terraform/enterprise/api-docs/admin/users#query-parameters
type AdminUserListOptions struct {
	ListOptions

	// Optional: A search query string. Users are searchable by username and email address.
	Query string `url:"q,omitempty"`

	// Optional: Can be "true" or "false" to show only administrators or non-administrators.
	Administrators string `url:"filter[admin],omitempty"`

	// Optional: Can be "true" or "false" to show only suspended users or users who are not suspended.
	SuspendedUsers string `url:"filter[suspended],omitempty"`

	// Optional: A list of relations to include. See available resources
	// https://developer.hashicorp.com/terraform/enterprise/api-docs/admin/users#available-related-resources
	Include []AdminUserIncludeOpt `url:"include,omitempty"`
}

// List all user accounts in the Terraform Enterprise installation
func (a *adminUsers) List(ctx context.Context, options *AdminUserListOptions) (*AdminUserList, error) {
	if err := options.valid(); err != nil {
		return nil, err
	}

	u := "admin/users"
	req, err := a.client.NewRequest("GET", u, options)
	if err != nil {
		return nil, err
	}

	aul := &AdminUserList{}
	err = req.Do(ctx, aul)
	if err != nil {
		return nil, err
	}

	return aul, nil
}

// Delete a user by its ID.
func (a *adminUsers) Delete(ctx context.Context, userID string) error {
	if !validStringID(&userID) {
		return ErrInvalidUserValue
	}

	u := fmt.Sprintf("admin/users/%s", url.PathEscape(userID))
	req, err := a.client.NewRequest("DELETE", u, nil)
	if err != nil {
		return err
	}

	return req.Do(ctx, nil)
}

// Suspend a user by its ID.
func (a *adminUsers) Suspend(ctx context.Context, userID string) (*AdminUser, error) {
	if !validStringID(&userID) {
		return nil, ErrInvalidUserValue
	}

	u := fmt.Sprintf("admin/users/%s/actions/suspend", url.PathEscape(userID))
	req, err := a.client.NewRequest("POST", u, nil)
	if err != nil {
		return nil, err
	}

	au := &AdminUser{}
	err = req.Do(ctx, au)
	if err != nil {
		return nil, err
	}

	return au, nil
}

// Unsuspend a user by its ID.
func (a *adminUsers) Unsuspend(ctx context.Context, userID string) (*AdminUser, error) {
	if !validStringID(&userID) {
		return nil, ErrInvalidUserValue
	}

	u := fmt.Sprintf("admin/users/%s/actions/unsuspend", url.PathEscape(userID))
	req, err := a.client.NewRequest("POST", u, nil)
	if err != nil {
		return nil, err
	}

	au := &AdminUser{}
	err = req.Do(ctx, au)
	if err != nil {
		return nil, err
	}

	return au, nil
}

// GrantAdmin grants admin privileges to a user by its ID.
func (a *adminUsers) GrantAdmin(ctx context.Context, userID string) (*AdminUser, error) {
	if !validStringID(&userID) {
		return nil, ErrInvalidUserValue
	}

	u := fmt.Sprintf("admin/users/%s/actions/grant_admin", url.PathEscape(userID))
	req, err := a.client.NewRequest("POST", u, nil)
	if err != nil {
		return nil, err
	}

	au := &AdminUser{}
	err = req.Do(ctx, au)
	if err != nil {
		return nil, err
	}

	return au, nil
}

// RevokeAdmin revokes admin privileges to a user by its ID.
func (a *adminUsers) RevokeAdmin(ctx context.Context, userID string) (*AdminUser, error) {
	if !validStringID(&userID) {
		return nil, ErrInvalidUserValue
	}

	u := fmt.Sprintf("admin/users/%s/actions/revoke_admin", url.PathEscape(userID))
	req, err := a.client.NewRequest("POST", u, nil)
	if err != nil {
		return nil, err
	}

	au := &AdminUser{}
	err = req.Do(ctx, au)
	if err != nil {
		return nil, err
	}

	return au, nil
}

// Disable2FA disables a user's two-factor authentication in the situation
// where they have lost access to their device and recovery codes.
func (a *adminUsers) Disable2FA(ctx context.Context, userID string) (*AdminUser, error) {
	if !validStringID(&userID) {
		return nil, ErrInvalidUserValue
	}

	u := fmt.Sprintf("admin/users/%s/actions/disable_two_factor", url.PathEscape(userID))
	req, err := a.client.NewRequest("POST", u, nil)
	if err != nil {
		return nil, err
	}

	au := &AdminUser{}
	err = req.Do(ctx, au)
	if err != nil {
		return nil, err
	}

	return au, nil
}

func (o *AdminUserListOptions) valid() error {
	return nil
}

// Compile-time proof of interface implementation.
var _ AdminWorkspaces = (*adminWorkspaces)(nil)

// AdminWorkspaces describes all the admin workspace related methods that the Terraform Enterprise API supports.
// Note that admin settings are only available in Terraform Enterprise.
//
// TFE API docs: https://developer.hashicorp.com/terraform/enterprise/api-docs/admin/workspaces
type AdminWorkspaces interface {
	// List all the workspaces within a workspace.
	List(ctx context.Context, options *AdminWorkspaceListOptions) (*AdminWorkspaceList, error)

	// Read a workspace by its ID.
	Read(ctx context.Context, workspaceID string) (*AdminWorkspace, error)

	// Delete a workspace by its ID.
	Delete(ctx context.Context, workspaceID string) error
}

// adminWorkspaces implements AdminWorkspaces interface.
type adminWorkspaces struct {
	client *Client
}

// AdminVCSRepo represents a VCS repository
type AdminVCSRepo struct {
	Identifier string `jsonapi:"attr,identifier"`
}

// AdminWorkspaces represents a Terraform Enterprise admin workspace.
type AdminWorkspace struct {
	ID      string        `jsonapi:"primary,workspaces"`
	Name    string        `jsonapi:"attr,name"`
	Locked  bool          `jsonapi:"attr,locked"`
	VCSRepo *AdminVCSRepo `jsonapi:"attr,vcs-repo"`

	// Relations
	Organization *Organization `jsonapi:"relation,organization"`
	CurrentRun   *Run          `jsonapi:"relation,current-run"`
}

// AdminWorkspaceIncludeOpt represents the available options for include query params.
// https://developer.hashicorp.com/terraform/enterprise/api-docs/admin/workspaces#available-related-resources
type AdminWorkspaceIncludeOpt string

const (
	AdminWorkspaceOrg        AdminWorkspaceIncludeOpt = "organization"
	AdminWorkspaceCurrentRun AdminWorkspaceIncludeOpt = "current_run"
	AdminWorkspaceOrgOwners  AdminWorkspaceIncludeOpt = "organization.owners"
)

// AdminWorkspaceListOptions represents the options for listing workspaces.
type AdminWorkspaceListOptions struct {
	ListOptions

	// A query string (partial workspace name) used to filter the results.
	// https://developer.hashicorp.com/terraform/enterprise/api-docs/admin/workspaces#query-parameters
	Query string `url:"q,omitempty"`

	// Optional: A list of relations to include. See available resources
	// https://developer.hashicorp.com/terraform/enterprise/api-docs/admin/workspaces#available-related-resources
	Include []AdminWorkspaceIncludeOpt `url:"include,omitempty"`

	// Optional: A comma-separated list of Run statuses to restrict results. See available resources
	// https://developer.hashicorp.com/terraform/enterprise/api-docs/admin/workspaces#query-parameters
	Filter string `url:"filter[current_run][status],omitempty"`

	// Optional: May sort on "name" (the default) and "current-run.created-at" (which sorts by the time of the current run)
	// Prepending a hyphen to the sort parameter will reverse the order (e.g. "-name" to reverse the default order)
	Sort string `url:"sort,omitempty"`
}

// AdminWorkspaceList represents a list of workspaces.
type AdminWorkspaceList struct {
	*Pagination
	Items []*AdminWorkspace
}

// List all the workspaces within a workspace.
func (s *adminWorkspaces) List(ctx context.Context, options *AdminWorkspaceListOptions) (*AdminWorkspaceList, error) {
	if err := options.valid(); err != nil {
		return nil, err
	}

	u := "admin/workspaces"
	req, err := s.client.NewRequest("GET", u, options)
	if err != nil {
		return nil, err
	}

	awl := &AdminWorkspaceList{}
	err = req.Do(ctx, awl)
	if err != nil {
		return nil, err
	}

	return awl, nil
}

// Read a workspace by its ID.
func (s *adminWorkspaces) Read(ctx context.Context, workspaceID string) (*AdminWorkspace, error) {
	if !validStringID(&workspaceID) {
		return nil, ErrInvalidWorkspaceValue
	}

	u := fmt.Sprintf("admin/workspaces/%s", url.PathEscape(workspaceID))
	req, err := s.client.NewRequest("GET", u, nil)
	if err != nil {
		return nil, err
	}

	aw := &AdminWorkspace{}
	err = req.Do(ctx, aw)
	if err != nil {
		return nil, err
	}

	return aw, nil
}

// Delete a workspace by its ID.
func (s *adminWorkspaces) Delete(ctx context.Context, workspaceID string) error {
	if !validStringID(&workspaceID) {
		return ErrInvalidWorkspaceValue
	}

	u := fmt.Sprintf("admin/workspaces/%s", url.PathEscape(workspaceID))
	req, err := s.client.NewRequest("DELETE", u, nil)
	if err != nil {
		return err
	}

	return req.Do(ctx, nil)
}

func (o *AdminWorkspaceListOptions) valid() error {
	return nil
}

// Compile-time proof of interface implementation.
var _ AgentPools = (*agentPools)(nil)

// AgentPools describes all the agent pool related methods that the HCP Terraform
// API supports. Note that agents are not available in Terraform Enterprise.
//
// TFE API docs: https://developer.hashicorp.com/terraform/cloud-docs/api-docs/agents
type AgentPools interface {
	// List all the agent pools of the given organization.
	List(ctx context.Context, organization string, options *AgentPoolListOptions) (*AgentPoolList, error)

	// Create a new agent pool with the given options.
	Create(ctx context.Context, organization string, options AgentPoolCreateOptions) (*AgentPool, error)

	// Read an agent pool by its ID.
	Read(ctx context.Context, agentPoolID string) (*AgentPool, error)

	// Read an agent pool by its ID with the given options.
	ReadWithOptions(ctx context.Context, agentPoolID string, options *AgentPoolReadOptions) (*AgentPool, error)

	// Update an agent pool by its ID.
	Update(ctx context.Context, agentPool string, options AgentPoolUpdateOptions) (*AgentPool, error)

	// UpdateAllowedWorkspaces updates the list of allowed workspaces associated with an agent pool.
	UpdateAllowedWorkspaces(ctx context.Context, agentPool string, options AgentPoolAllowedWorkspacesUpdateOptions) (*AgentPool, error)

	// UpdateAllowedProjects updates the list of allowed projects associated with an agent pool.
	UpdateAllowedProjects(ctx context.Context, agentPool string, options AgentPoolAllowedProjectsUpdateOptions) (*AgentPool, error)

	// UpdateExcludedWorkspaces updates the list of excluded workspaces associated with an agent pool.
	UpdateExcludedWorkspaces(ctx context.Context, agentPool string, options AgentPoolExcludedWorkspacesUpdateOptions) (*AgentPool, error)

	// Delete an agent pool by its ID.
	Delete(ctx context.Context, agentPoolID string) error
}

// agentPools implements AgentPools.
type agentPools struct {
	client *Client
}

// AgentPoolList represents a list of agent pools.
type AgentPoolList struct {
	*Pagination
	Items []*AgentPool
}

// AgentPool represents a HCP Terraform agent pool.
type AgentPool struct {
	ID                 string    `jsonapi:"primary,agent-pools"`
	Name               string    `jsonapi:"attr,name"`
	AgentCount         int       `jsonapi:"attr,agent-count"`
	OrganizationScoped bool      `jsonapi:"attr,organization-scoped"`
	CreatedAt          time.Time `jsonapi:"attr,created-at,iso8601"`

	// Relations
	Organization       *Organization        `jsonapi:"relation,organization"`
	HYOKConfigurations []*HYOKConfiguration `jsonapi:"relation,hyok-configurations"`
	Workspaces         []*Workspace         `jsonapi:"relation,workspaces"`
	AllowedWorkspaces  []*Workspace         `jsonapi:"relation,allowed-workspaces"`
	AllowedProjects    []*Project           `jsonapi:"relation,allowed-projects"`
	ExcludedWorkspaces []*Workspace         `jsonapi:"relation,excluded-workspaces"`
}

// A list of relations to include
// https://developer.hashicorp.com/terraform/cloud-docs/api-docs/agents#available-related-resources
type AgentPoolIncludeOpt string

const (
	AgentPoolWorkspaces         AgentPoolIncludeOpt = "workspaces"
	AgentPoolHYOKConfigurations AgentPoolIncludeOpt = "hyok-configurations"
)

type AgentPoolReadOptions struct {
	Include []AgentPoolIncludeOpt `url:"include,omitempty"`
}

// AgentPoolListOptions represents the options for listing agent pools.
type AgentPoolListOptions struct {
	ListOptions
	// Optional: A list of relations to include. See available resources
	// https://developer.hashicorp.com/terraform/cloud-docs/api-docs/agents#available-related-resources
	Include []AgentPoolIncludeOpt `url:"include,omitempty"`

	// Optional: A search query string used to filter agent pool. Agent pools are searchable by name
	Query string `url:"q,omitempty"`

	// Optional: String (workspace name) used to filter the results.
	AllowedWorkspacesName string `url:"filter[allowed_workspaces][name],omitempty"`

	// Optional: String (project name) used to filter the results.
	AllowedProjectsName string `url:"filter[allowed_projects][name],omitempty"`

	// Optional: Allows sorting the agent pools by "created-by" or "name"
	Sort string `url:"sort,omitempty"`
}

// AgentPoolCreateOptions represents the options for creating an agent pool.
type AgentPoolCreateOptions struct {
	// Type is a public field utilized by JSON:API to
	// set the resource type via the field tag.
	// It is not a user-defined value and does not need to be set.
	// https://jsonapi.org/format/#crud-creating
	Type string `jsonapi:"primary,agent-pools"`

	// Required: A name to identify the agent pool.
	Name *string `jsonapi:"attr,name"`

	// True if the agent pool is organization scoped, false otherwise.
	OrganizationScoped *bool `jsonapi:"attr,organization-scoped,omitempty"`

	// List of workspaces that are associated with an agent pool.
	AllowedWorkspaces []*Workspace `jsonapi:"relation,allowed-workspaces,omitempty"`

	// List of projects that are associated with an agent pool.
	AllowedProjects []*Project `jsonapi:"relation,allowed-projects,omitempty"`

	// List of workspaces that are excluded from the scope of an agent pool.
	ExcludedWorkspaces []*Workspace `jsonapi:"relation,excluded-workspaces,omitempty"`
}

// List all the agent pools of the given organization.
func (s *agentPools) List(ctx context.Context, organization string, options *AgentPoolListOptions) (*AgentPoolList, error) {
	if !validStringID(&organization) {
		return nil, ErrInvalidOrg
	}
	if err := options.valid(); err != nil {
		return nil, err
	}

	u := fmt.Sprintf("organizations/%s/agent-pools", url.PathEscape(organization))
	req, err := s.client.NewRequest("GET", u, options)
	if err != nil {
		return nil, err
	}

	poolList := &AgentPoolList{}
	err = req.Do(ctx, poolList)
	if err != nil {
		return nil, err
	}

	return poolList, nil
}

// Create a new agent pool with the given options.
func (s *agentPools) Create(ctx context.Context, organization string, options AgentPoolCreateOptions) (*AgentPool, error) {
	if !validStringID(&organization) {
		return nil, ErrInvalidOrg
	}

	if err := options.valid(); err != nil {
		return nil, err
	}

	u := fmt.Sprintf("organizations/%s/agent-pools", url.PathEscape(organization))
	req, err := s.client.NewRequest("POST", u, &options)
	if err != nil {
		return nil, err
	}

	pool := &AgentPool{}
	err = req.Do(ctx, pool)
	if err != nil {
		return nil, err
	}

	return pool, nil
}

// Read a single agent pool by its ID
func (s *agentPools) Read(ctx context.Context, agentpoolID string) (*AgentPool, error) {
	return s.ReadWithOptions(ctx, agentpoolID, nil)
}

// Read a single agent pool by its ID with options.
func (s *agentPools) ReadWithOptions(ctx context.Context, agentpoolID string, options *AgentPoolReadOptions) (*AgentPool, error) {
	if !validStringID(&agentpoolID) {
		return nil, ErrInvalidAgentPoolID
	}
	if err := options.valid(); err != nil {
		return nil, err
	}

	u := fmt.Sprintf("agent-pools/%s", url.PathEscape(agentpoolID))
	req, err := s.client.NewRequest("GET", u, &options)
	if err != nil {
		return nil, err
	}

	pool := &AgentPool{}
	err = req.Do(ctx, pool)
	if err != nil {
		return nil, err
	}

	return pool, nil
}

// AgentPoolUpdateOptions represents the options for updating an agent pool.
type AgentPoolUpdateOptions struct {
	// Type is a public field utilized by JSON:API to
	// set the resource type via the field tag.
	// It is not a user-defined value and does not need to be set.
	// https://jsonapi.org/format/#crud-creating
	Type string `jsonapi:"primary,agent-pools"`

	// A new name to identify the agent pool.
	Name *string `jsonapi:"attr,name,omitempty"`

	// True if the agent pool is organization scoped, false otherwise.
	OrganizationScoped *bool `jsonapi:"attr,organization-scoped,omitempty"`

	// A new list of workspaces that are associated with an agent pool.
	AllowedWorkspaces []*Workspace `jsonapi:"relation,allowed-workspaces,omitempty"`

	// A new list of projects that are associated with an agent pool.
	AllowedProjects []*Project `jsonapi:"relation,allowed-projects,omitempty"`

	// A new list of workspaces that are excluded from the scope of an agent pool.
	ExcludedWorkspaces []*Workspace `jsonapi:"relation,excluded-workspaces,omitempty"`
}

// AgentPoolAllowedWorkspacesUpdateOptions represents the options for updating the allowed workspace on an agent pool
type AgentPoolAllowedWorkspacesUpdateOptions struct {
	// Type is a public field utilized by JSON:API to
	// set the resource type via the field tag.
	// It is not a user-defined value and does not need to be set.
	// https://jsonapi.org/format/#crud-creating
	Type string `jsonapi:"primary,agent-pools"`

	// A new list of workspaces that are associated with an agent pool.
	AllowedWorkspaces []*Workspace `jsonapi:"relation,allowed-workspaces"`
}

// AgentPoolAllowedProjectsUpdateOptions represents the options for updating the allowed projects on an agent pool
type AgentPoolAllowedProjectsUpdateOptions struct {
	// Type is a public field utilized by JSON:API to
	// set the resource type via the field tag.
	// It is not a user-defined value and does not need to be set.
	// https://jsonapi.org/format/#crud-creating
	Type string `jsonapi:"primary,agent-pools"`

	// A new list of projects that are associated with an agent pool.
	AllowedProjects []*Project `jsonapi:"relation,allowed-projects"`
}

// AgentPoolExcludedWorkspacesUpdateOptions represents the options for updating the excluded workspace on an agent pool
type AgentPoolExcludedWorkspacesUpdateOptions struct {
	// Type is a public field utilized by JSON:API to
	// set the resource type via the field tag.
	// It is not a user-defined value and does not need to be set.
	// https://jsonapi.org/format/#crud-creating
	Type string `jsonapi:"primary,agent-pools"`

	// A new list of workspaces that are excluded from the scope of an agent pool.
	ExcludedWorkspaces []*Workspace `jsonapi:"relation,excluded-workspaces"`
}

// Update an agent pool by its ID.
// **Note:** This method cannot be used to clear the allowed workspaces, allowed projects, or excluded workspaces fields.
// instead use UpdateAllowedWorkspaces, UpdateAllowedProjects, or UpdateExcludedWorkspaces methods respectively.
func (s *agentPools) Update(ctx context.Context, agentPoolID string, options AgentPoolUpdateOptions) (*AgentPool, error) {
	if !validStringID(&agentPoolID) {
		return nil, ErrInvalidAgentPoolID
	}

	if err := options.valid(); err != nil {
		return nil, err
	}

	u := fmt.Sprintf("agent-pools/%s", url.PathEscape(agentPoolID))
	req, err := s.client.NewRequest("PATCH", u, &options)
	if err != nil {
		return nil, err
	}

	k := &AgentPool{}
	err = req.Do(ctx, k)
	if err != nil {
		return nil, err
	}

	return k, nil
}

func (s *agentPools) UpdateAllowedWorkspaces(ctx context.Context, agentPoolID string, options AgentPoolAllowedWorkspacesUpdateOptions) (*AgentPool, error) {
	return s.updateArrayAttribute(ctx, agentPoolID, &options)
}

func (s *agentPools) UpdateAllowedProjects(ctx context.Context, agentPoolID string, options AgentPoolAllowedProjectsUpdateOptions) (*AgentPool, error) {
	return s.updateArrayAttribute(ctx, agentPoolID, &options)
}

func (s *agentPools) UpdateExcludedWorkspaces(ctx context.Context, agentPoolID string, options AgentPoolExcludedWorkspacesUpdateOptions) (*AgentPool, error) {
	return s.updateArrayAttribute(ctx, agentPoolID, &options)
}

// Delete an agent pool by its ID.
func (s *agentPools) Delete(ctx context.Context, agentPoolID string) error {
	if !validStringID(&agentPoolID) {
		return ErrInvalidAgentPoolID
	}

	u := fmt.Sprintf("agent-pools/%s", url.PathEscape(agentPoolID))
	req, err := s.client.NewRequest("DELETE", u, nil)
	if err != nil {
		return err
	}

	return req.Do(ctx, nil)
}

// updateArrayAttribute is a helper function to update array attributes of an agent pool, such as allowed workspaces, allowed projects, or excluded workspaces.
// Note: This function does not validate the options parameter, so it should be used with caution.  It is intended to be used with options structs
// (e.g. AgentPoolAllowedWorkspacesUpdateOptions, AgentPoolAllowedProjectsUpdateOptions, AgentPoolExcludedWorkspacesUpdateOptions) whose array
// attributes are NOT marked `omitempty`, so that an empty array is sent to the API to clear the existing values.
func (s *agentPools) updateArrayAttribute(ctx context.Context, agentPoolID string, options any) (*AgentPool, error) {
	if !validStringID(&agentPoolID) {
		return nil, ErrInvalidAgentPoolID
	}

	u := fmt.Sprintf("agent-pools/%s", url.PathEscape(agentPoolID))
	req, err := s.client.NewRequest("PATCH", u, options)
	if err != nil {
		return nil, err
	}

	k := &AgentPool{}
	err = req.Do(ctx, k)
	if err != nil {
		return nil, err
	}

	return k, nil
}

func (o AgentPoolCreateOptions) valid() error {
	if !validString(o.Name) {
		return ErrRequiredName
	}
	if !validStringID(o.Name) {
		return ErrInvalidName
	}
	return nil
}

func (o AgentPoolUpdateOptions) valid() error {
	if o.Name != nil && !validStringID(o.Name) {
		return ErrInvalidName
	}
	return nil
}

func (o *AgentPoolReadOptions) valid() error {
	return nil
}

func (o *AgentPoolListOptions) valid() error {
	return nil
}

// Compile-time proof of interface implementation.
var _ AgentTokens = (*agentTokens)(nil)

// AgentTokens describes all the agent token related methods that the
// HCP Terraform API supports.
//
// TFE API docs:
// https://developer.hashicorp.com/terraform/cloud-docs/api-docs/agent-tokens
type AgentTokens interface {
	// List all the agent tokens of the given agent pool.
	List(ctx context.Context, agentPoolID string) (*AgentTokenList, error)

	// Create a new agent token with the given options.
	Create(ctx context.Context, agentPoolID string, options AgentTokenCreateOptions) (*AgentToken, error)

	// Read an agent token by its ID.
	Read(ctx context.Context, agentTokenID string) (*AgentToken, error)

	// Delete an agent token by its ID.
	Delete(ctx context.Context, agentTokenID string) error
}

// agentTokens implements AgentTokens.
type agentTokens struct {
	client *Client
}

// AgentToken represents a HCP Terraform agent token.
type AgentToken struct {
	ID          string    `jsonapi:"primary,authentication-tokens"`
	CreatedAt   time.Time `jsonapi:"attr,created-at,iso8601"`
	Description string    `jsonapi:"attr,description"`
	LastUsedAt  time.Time `jsonapi:"attr,last-used-at,iso8601"`
	Token       string    `jsonapi:"attr,token"`

	// Relations
	CreatedBy *User `jsonapi:"relation,created-by"`
}

// AgentTokenList represents a list of agent tokens.
type AgentTokenList struct {
	*Pagination
	Items []*AgentToken
}

// AgentTokenCreateOptions represents the options for creating an agent token.
type AgentTokenCreateOptions struct {
	// Type is a public field utilized by JSON:API to
	// set the resource type via the field tag.
	// It is not a user-defined value and does not need to be set.
	// https://jsonapi.org/format/#crud-creating
	Type string `jsonapi:"primary,agent-tokens"`

	// Description of the token
	Description *string `jsonapi:"attr,description"`
}

// List all the agent tokens of the given agent pool.
func (s *agentTokens) List(ctx context.Context, agentPoolID string) (*AgentTokenList, error) {
	if !validStringID(&agentPoolID) {
		return nil, ErrInvalidAgentPoolID
	}

	u := fmt.Sprintf("agent-pools/%s/authentication-tokens", url.PathEscape(agentPoolID))
	req, err := s.client.NewRequest("GET", u, nil)
	if err != nil {
		return nil, err
	}

	tokenList := &AgentTokenList{}
	err = req.Do(ctx, tokenList)
	if err != nil {
		return nil, err
	}

	return tokenList, nil
}

// Create a new agent token with the given options.
func (s *agentTokens) Create(ctx context.Context, agentPoolID string, options AgentTokenCreateOptions) (*AgentToken, error) {
	if !validStringID(&agentPoolID) {
		return nil, ErrInvalidAgentPoolID
	}

	if !validString(options.Description) {
		return nil, ErrAgentTokenDescription
	}

	u := fmt.Sprintf("agent-pools/%s/authentication-tokens", url.PathEscape(agentPoolID))
	req, err := s.client.NewRequest("POST", u, &options)
	if err != nil {
		return nil, err
	}

	at := &AgentToken{}
	err = req.Do(ctx, at)
	if err != nil {
		return nil, err
	}

	return at, err
}

// Read an agent token by its ID.
func (s *agentTokens) Read(ctx context.Context, agentTokenID string) (*AgentToken, error) {
	if !validStringID(&agentTokenID) {
		return nil, ErrInvalidAgentTokenID
	}

	u := fmt.Sprintf(AuthenticationTokensPath, url.PathEscape(agentTokenID))
	req, err := s.client.NewRequest("GET", u, nil)
	if err != nil {
		return nil, err
	}

	at := &AgentToken{}
	err = req.Do(ctx, at)
	if err != nil {
		return nil, err
	}

	return at, err
}

// Delete an agent token by its ID.
func (s *agentTokens) Delete(ctx context.Context, agentTokenID string) error {
	if !validStringID(&agentTokenID) {
		return ErrInvalidAgentTokenID
	}

	u := fmt.Sprintf(AuthenticationTokensPath, url.PathEscape(agentTokenID))
	req, err := s.client.NewRequest("DELETE", u, nil)
	if err != nil {
		return err
	}

	return req.Do(ctx, nil)
}

// Compile-time proof of interface implementation.
var _ Agents = (*agents)(nil)

// Agents describes all the agent-related methods that the
// HCP Terraform API supports.
// TFE API docs: https://developer.hashicorp.com/terraform/cloud-docs/api-docs/agents
type Agents interface {
	// Read an agent by its ID.
	Read(ctx context.Context, agentID string) (*Agent, error)

	// List all the agents of the given pool.
	List(ctx context.Context, agentPoolID string, options *AgentListOptions) (*AgentList, error)
}

// agents implements Agents.
type agents struct {
	client *Client
}

// AgentList represents a list of agents.
type AgentList struct {
	*Pagination
	Items []*Agent
}

// Agent represents a HCP Terraform agent.
type Agent struct {
	ID         string `jsonapi:"primary,agents"`
	Name       string `jsonapi:"attr,name"`
	IP         string `jsonapi:"attr,ip-address"`
	Status     string `jsonapi:"attr,status"`
	LastPingAt string `jsonapi:"attr,last-ping-at"`
}

type AgentListOptions struct {
	ListOptions

	//Optional:
	LastPingSince time.Time `url:"filter[last-ping-since],omitempty,iso8601"`

	// Optional: Allows sorting the agents by "created-by"
	Sort string `url:"sort,omitempty"`
}

// Read a single agent by its ID
func (s *agents) Read(ctx context.Context, agentID string) (*Agent, error) {
	if !validStringID(&agentID) {
		return nil, ErrInvalidAgentID
	}

	u := fmt.Sprintf("agents/%s", url.PathEscape(agentID))
	req, err := s.client.NewRequest("GET", u, nil)
	if err != nil {
		return nil, err
	}

	agent := &Agent{}
	err = req.Do(ctx, agent)
	if err != nil {
		return nil, err
	}

	return agent, nil
}

// List all the agents of the given organization.
func (s *agents) List(ctx context.Context, agentPoolID string, options *AgentListOptions) (*AgentList, error) {
	if !validStringID(&agentPoolID) {
		return nil, ErrInvalidOrg
	}

	u := fmt.Sprintf("agent-pools/%s/agents", url.PathEscape(agentPoolID))
	req, err := s.client.NewRequest("GET", u, options)
	if err != nil {
		return nil, err
	}

	agentList := &AgentList{}
	err = req.Do(ctx, agentList)
	if err != nil {
		return nil, err
	}

	return agentList, nil
}

// Compile-time proof of interface implementation.
var _ Applies = (*applies)(nil)

// Applies describes all the apply related methods that the Terraform
// Enterprise API supports.
//
// TFE API docs: https://developer.hashicorp.com/terraform/cloud-docs/api-docs/applies
type Applies interface {
	// Read an apply by its ID.
	Read(ctx context.Context, applyID string) (*Apply, error)

	// Logs retrieves the logs of an apply.
	Logs(ctx context.Context, applyID string) (io.Reader, error)
}

// applies implements Applies interface.
type applies struct {
	client *Client
}

// ApplyStatus represents an apply state.
type ApplyStatus string

// List all available apply statuses.
const (
	ApplyCanceled    ApplyStatus = "canceled"
	ApplyCreated     ApplyStatus = "created"
	ApplyErrored     ApplyStatus = "errored"
	ApplyFinished    ApplyStatus = "finished"
	ApplyMFAWaiting  ApplyStatus = "mfa_waiting"
	ApplyPending     ApplyStatus = "pending"
	ApplyQueued      ApplyStatus = "queued"
	ApplyRunning     ApplyStatus = "running"
	ApplyUnreachable ApplyStatus = "unreachable"
)

// Apply represents a Terraform Enterprise apply.
type Apply struct {
	ID                   string                 `jsonapi:"primary,applies"`
	LogReadURL           string                 `jsonapi:"attr,log-read-url"`
	ResourceAdditions    int                    `jsonapi:"attr,resource-additions"`
	ResourceChanges      int                    `jsonapi:"attr,resource-changes"`
	ResourceDestructions int                    `jsonapi:"attr,resource-destructions"`
	ResourceImports      int                    `jsonapi:"attr,resource-imports"`
	Status               ApplyStatus            `jsonapi:"attr,status"`
	StatusTimestamps     *ApplyStatusTimestamps `jsonapi:"attr,status-timestamps"`
}

// ApplyStatusTimestamps holds the timestamps for individual apply statuses.
type ApplyStatusTimestamps struct {
	CanceledAt      time.Time `jsonapi:"attr,canceled-at,rfc3339"`
	ErroredAt       time.Time `jsonapi:"attr,errored-at,rfc3339"`
	FinishedAt      time.Time `jsonapi:"attr,finished-at,rfc3339"`
	ForceCanceledAt time.Time `jsonapi:"attr,force-canceled-at,rfc3339"`
	QueuedAt        time.Time `jsonapi:"attr,queued-at,rfc3339"`
	StartedAt       time.Time `jsonapi:"attr,started-at,rfc3339"`
}

// Read an apply by its ID.
func (s *applies) Read(ctx context.Context, applyID string) (*Apply, error) {
	if !validStringID(&applyID) {
		return nil, ErrInvalidApplyID
	}

	u := fmt.Sprintf("applies/%s", url.PathEscape(applyID))
	req, err := s.client.NewRequest("GET", u, nil)
	if err != nil {
		return nil, err
	}

	a := &Apply{}
	err = req.Do(ctx, a)
	if err != nil {
		return nil, err
	}

	return a, nil
}

// Logs retrieves the logs of an apply.
func (s *applies) Logs(ctx context.Context, applyID string) (io.Reader, error) {
	if !validStringID(&applyID) {
		return nil, ErrInvalidApplyID
	}

	// Get the apply to make sure it exists.
	a, err := s.Read(ctx, applyID)
	if err != nil {
		return nil, err
	}

	// Return an error if the log URL is empty.
	if a.LogReadURL == "" {
		return nil, fmt.Errorf("apply %s does not have a log URL", applyID)
	}

	u, err := url.Parse(a.LogReadURL)
	if err != nil {
		return nil, fmt.Errorf("invalid log URL: %w", err)
	}

	done := func() (bool, error) {
		a, err := s.Read(ctx, a.ID)
		if err != nil {
			return false, err
		}

		switch a.Status {
		case ApplyCanceled, ApplyErrored, ApplyFinished, ApplyUnreachable:
			return true, nil
		default:
			return false, nil
		}
	}

	return &LogReader{
		client: s.client,
		ctx:    ctx,
		done:   done,
		logURL: u,
	}, nil
}

// Copyright IBM Corp. 2018, 2026
// SPDX-License-Identifier: MPL-2.0

// Compile-time proof of interface implementation
var _ AuditTrails = (*auditTrails)(nil)

// AuditTrails describes all the audit event related methods that the HCP Terraform
// API supports.
// **Note:** These methods require the client to be configured with an organization token for
// an organization in the Business tier. Furthermore, these methods are only available in HCP Terraform.
//
// HCP Terraform API Docs: https://developer.hashicorp.com/terraform/cloud-docs/api-docs/audit-trails
type AuditTrails interface {
	// Read all the audit events in an organization.
	List(ctx context.Context, options *AuditTrailListOptions) (*AuditTrailList, error)
}

// auditTrails implements AuditTrails
type auditTrails struct {
	client *Client
}

// AuditTrailRequest represents the request details of the audit event.
type AuditTrailRequest struct {
	ID string `json:"id"`
}

// AuditTrailAuth represents the details of the actor that invoked the audit event.
type AuditTrailAuth struct {
	AccessorID     string  `json:"accessor_id"`
	Description    string  `json:"description"`
	Type           string  `json:"type"`
	ImpersonatorID *string `json:"impersonator_id"`
	OrganizationID string  `json:"organization_id"`
}

// AuditTrailResource represents the details of the API resource in the audit event.
type AuditTrailResource struct {
	ID     string                 `json:"id"`
	Type   string                 `json:"type"`
	Action string                 `json:"action"`
	Meta   map[string]interface{} `json:"meta"`
}

type AuditTrailPagination struct {
	CurrentPage  int `json:"current_page"`
	PreviousPage int `json:"prev_page"`
	NextPage     int `json:"next_page"`
	TotalPages   int `json:"total_pages"`
	TotalCount   int `json:"total_count"`
}

// AuditTrail represents an event in the HCP Terraform audit log.
type AuditTrail struct {
	ID        string    `json:"id"`
	Version   string    `json:"version"`
	Type      string    `json:"type"`
	Timestamp time.Time `json:"timestamp"`

	Auth     AuditTrailAuth     `json:"auth"`
	Request  AuditTrailRequest  `json:"request"`
	Resource AuditTrailResource `json:"resource"`
}

// AuditTrailList represents a list of audit trails.
type AuditTrailList struct {
	*AuditTrailPagination `json:"pagination"`
	Items                 []*AuditTrail `json:"data"`
}

// AuditTrailListOptions represents the options for listing audit trails.
type AuditTrailListOptions struct {
	// Optional: Returns only audit trails created after this date
	Since time.Time `url:"since,omitempty"`
	*ListOptions
}

// List all the audit events in an organization.
func (s *auditTrails) List(ctx context.Context, options *AuditTrailListOptions) (*AuditTrailList, error) {
	u, err := s.client.baseURL.Parse("/api/v2/organization/audit-trail")
	if err != nil {
		return nil, err
	}

	headers := make(http.Header)
	headers.Set("User-Agent", _userAgent)
	headers.Set("Authorization", "Bearer "+s.client.token)
	headers.Set("Content-Type", "application/json")

	if options != nil {
		q, err := query.Values(options)
		if err != nil {
			return nil, err
		}

		u.RawQuery = encodeQueryParams(q)
	}

	req, err := retryablehttp.NewRequest("GET", u.String(), nil)
	if err != nil {
		return nil, err
	}

	// Attach the headers to the request
	for k, v := range headers {
		req.Header[k] = v
	}

	if err := s.client.limiter.Wait(ctx); err != nil {
		return nil, err
	}

	resp, err := s.client.http.Do(req.WithContext(ctx))
	if err != nil {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			return nil, err
		}
	}
	defer resp.Body.Close() //nolint:errcheck

	if err := checkResponseCode(resp); err != nil {
		return nil, err
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	atl := &AuditTrailList{}
	if err := json.Unmarshal(body, atl); err != nil {
		return nil, err
	}

	return atl, nil
}

// Copyright IBM Corp. 2018, 2026
// SPDX-License-Identifier: MPL-2.0

const OIDCConfigPathFormat = "oidc-configurations/%s"

// AWSOIDCConfigurations describes all the AWS OIDC configuration related methods that the HCP Terraform API supports.
// HCP Terraform API docs:
// https://developer.hashicorp.com/terraform/cloud-docs/api-docs/hold-your-own-key/oidc-configurations/aws
type AWSOIDCConfigurations interface {
	Create(ctx context.Context, organization string, options AWSOIDCConfigurationCreateOptions) (*AWSOIDCConfiguration, error)

	Read(ctx context.Context, oidcID string) (*AWSOIDCConfiguration, error)

	Update(ctx context.Context, oidcID string, options AWSOIDCConfigurationUpdateOptions) (*AWSOIDCConfiguration, error)

	Delete(ctx context.Context, oidcID string) error
}

type awsOIDCConfigurations struct {
	client *Client
}

var _ AWSOIDCConfigurations = &awsOIDCConfigurations{}

type AWSOIDCConfiguration struct {
	ID      string `jsonapi:"primary,aws-oidc-configurations"`
	RoleARN string `jsonapi:"attr,role-arn"`

	Organization *Organization `jsonapi:"relation,organization"`
}

type AWSOIDCConfigurationCreateOptions struct {
	// Type is a public field utilized by JSON:API to
	// set the resource type via the field tag.
	// It is not a user-defined value and does not need to be set.
	// https://jsonapi.org/format/#crud-creating
	Type string `jsonapi:"primary,aws-oidc-configurations"`

	// Attributes
	RoleARN string `jsonapi:"attr,role-arn"`
}

type AWSOIDCConfigurationUpdateOptions struct {
	// Type is a public field utilized by JSON:API to
	// set the resource type via the field tag.
	// It is not a user-defined value and does not need to be set.
	// https://jsonapi.org/format/#crud-creating
	Type string `jsonapi:"primary,aws-oidc-configurations"`

	// Attributes
	RoleARN string `jsonapi:"attr,role-arn"`
}

func (o *AWSOIDCConfigurationCreateOptions) valid() error {
	if o.RoleARN == "" {
		return ErrRequiredRoleARN
	}

	return nil
}

func (o *AWSOIDCConfigurationUpdateOptions) valid() error {
	if o.RoleARN == "" {
		return ErrRequiredRoleARN
	}

	return nil
}

func (aoc *awsOIDCConfigurations) Create(ctx context.Context, organization string, options AWSOIDCConfigurationCreateOptions) (*AWSOIDCConfiguration, error) {
	if !validStringID(&organization) {
		return nil, ErrInvalidOrg
	}

	if err := options.valid(); err != nil {
		return nil, err
	}

	req, err := aoc.client.NewRequest("POST", fmt.Sprintf("organizations/%s/oidc-configurations", url.PathEscape(organization)), &options)
	if err != nil {
		return nil, err
	}

	awsOIDCConfiguration := &AWSOIDCConfiguration{}
	err = req.Do(ctx, awsOIDCConfiguration)
	if err != nil {
		return nil, err
	}

	return awsOIDCConfiguration, nil
}

func (aoc *awsOIDCConfigurations) Read(ctx context.Context, oidcID string) (*AWSOIDCConfiguration, error) {
	req, err := aoc.client.NewRequest("GET", fmt.Sprintf(OIDCConfigPathFormat, url.PathEscape(oidcID)), nil)
	if err != nil {
		return nil, err
	}

	awsOIDCConfiguration := &AWSOIDCConfiguration{}
	err = req.Do(ctx, awsOIDCConfiguration)
	if err != nil {
		return nil, err
	}

	return awsOIDCConfiguration, nil
}

func (aoc *awsOIDCConfigurations) Update(ctx context.Context, oidcID string, options AWSOIDCConfigurationUpdateOptions) (*AWSOIDCConfiguration, error) {
	if !validStringID(&oidcID) {
		return nil, ErrInvalidOIDC
	}

	if err := options.valid(); err != nil {
		return nil, err
	}

	req, err := aoc.client.NewRequest("PATCH", fmt.Sprintf(OIDCConfigPathFormat, url.PathEscape(oidcID)), &options)
	if err != nil {
		return nil, err
	}

	awsOIDCConfiguration := &AWSOIDCConfiguration{}
	err = req.Do(ctx, awsOIDCConfiguration)
	if err != nil {
		return nil, err
	}

	return awsOIDCConfiguration, nil
}

func (aoc *awsOIDCConfigurations) Delete(ctx context.Context, oidcID string) error {
	if !validStringID(&oidcID) {
		return ErrInvalidOIDC
	}

	req, err := aoc.client.NewRequest("DELETE", fmt.Sprintf(OIDCConfigPathFormat, url.PathEscape(oidcID)), nil)
	if err != nil {
		return err
	}

	return req.Do(ctx, nil)
}

// Copyright IBM Corp. 2018, 2026
// SPDX-License-Identifier: MPL-2.0

// AzureOIDCConfigurations describes all the Azure OIDC configuration related methods that the HCP Terraform API supports.
// HCP Terraform API docs:
// https://developer.hashicorp.com/terraform/cloud-docs/api-docs/hold-your-own-key/oidc-configurations/azure
type AzureOIDCConfigurations interface {
	Create(ctx context.Context, organization string, options AzureOIDCConfigurationCreateOptions) (*AzureOIDCConfiguration, error)

	Read(ctx context.Context, oidcID string) (*AzureOIDCConfiguration, error)

	Update(ctx context.Context, oidcID string, options AzureOIDCConfigurationUpdateOptions) (*AzureOIDCConfiguration, error)

	Delete(ctx context.Context, oidcID string) error
}

type azureOIDCConfigurations struct {
	client *Client
}

var _ AzureOIDCConfigurations = &azureOIDCConfigurations{}

type AzureOIDCConfiguration struct {
	ID             string `jsonapi:"primary,azure-oidc-configurations"`
	ClientID       string `jsonapi:"attr,client-id"`
	SubscriptionID string `jsonapi:"attr,subscription-id"`
	TenantID       string `jsonapi:"attr,tenant-id"`

	Organization *Organization `jsonapi:"relation,organization"`
}

type AzureOIDCConfigurationCreateOptions struct {
	// Type is a public field utilized by JSON:API to
	// set the resource type via the field tag.
	// It is not a user-defined value and does not need to be set.
	// https://jsonapi.org/format/#crud-creating
	Type string `jsonapi:"primary,azure-oidc-configurations"`

	// Attributes
	ClientID       string `jsonapi:"attr,client-id"`
	SubscriptionID string `jsonapi:"attr,subscription-id"`
	TenantID       string `jsonapi:"attr,tenant-id"`
}

type AzureOIDCConfigurationUpdateOptions struct {
	// Type is a public field utilized by JSON:API to
	// set the resource type via the field tag.
	// It is not a user-defined value and does not need to be set.
	// https://jsonapi.org/format/#crud-creating
	Type string `jsonapi:"primary,azure-oidc-configurations"`

	// Attributes
	ClientID       *string `jsonapi:"attr,client-id,omitempty"`
	SubscriptionID *string `jsonapi:"attr,subscription-id,omitempty"`
	TenantID       *string `jsonapi:"attr,tenant-id,omitempty"`
}

func (o *AzureOIDCConfigurationCreateOptions) valid() error {
	if o.ClientID == "" {
		return ErrRequiredClientID
	}

	if o.SubscriptionID == "" {
		return ErrRequiredSubscriptionID
	}

	if o.TenantID == "" {
		return ErrRequiredTenantID
	}

	return nil
}

func (aoc *azureOIDCConfigurations) Create(ctx context.Context, organization string, options AzureOIDCConfigurationCreateOptions) (*AzureOIDCConfiguration, error) {
	if !validStringID(&organization) {
		return nil, ErrInvalidOrg
	}

	if err := options.valid(); err != nil {
		return nil, err
	}

	req, err := aoc.client.NewRequest("POST", fmt.Sprintf("organizations/%s/oidc-configurations", url.PathEscape(organization)), &options)
	if err != nil {
		return nil, err
	}

	azureOIDCConfiguration := &AzureOIDCConfiguration{}
	err = req.Do(ctx, azureOIDCConfiguration)
	if err != nil {
		return nil, err
	}

	return azureOIDCConfiguration, nil
}

func (aoc *azureOIDCConfigurations) Read(ctx context.Context, oidcID string) (*AzureOIDCConfiguration, error) {
	req, err := aoc.client.NewRequest("GET", fmt.Sprintf(OIDCConfigPathFormat, url.PathEscape(oidcID)), nil)
	if err != nil {
		return nil, err
	}

	azureOIDCConfiguration := &AzureOIDCConfiguration{}
	err = req.Do(ctx, azureOIDCConfiguration)
	if err != nil {
		return nil, err
	}

	return azureOIDCConfiguration, nil
}

func (aoc *azureOIDCConfigurations) Update(ctx context.Context, oidcID string, options AzureOIDCConfigurationUpdateOptions) (*AzureOIDCConfiguration, error) {
	if !validStringID(&oidcID) {
		return nil, ErrInvalidOIDC
	}

	req, err := aoc.client.NewRequest("PATCH", fmt.Sprintf(OIDCConfigPathFormat, url.PathEscape(oidcID)), &options)
	if err != nil {
		return nil, err
	}

	azureOIDCConfiguration := &AzureOIDCConfiguration{}
	err = req.Do(ctx, azureOIDCConfiguration)
	if err != nil {
		return nil, err
	}

	return azureOIDCConfiguration, nil
}

func (aoc *azureOIDCConfigurations) Delete(ctx context.Context, oidcID string) error {
	if !validStringID(&oidcID) {
		return ErrInvalidOIDC
	}

	req, err := aoc.client.NewRequest("DELETE", fmt.Sprintf(OIDCConfigPathFormat, url.PathEscape(oidcID)), nil)
	if err != nil {
		return err
	}

	return req.Do(ctx, nil)
}

// Compile-time proof of interface implementation.
var _ Comments = (*comments)(nil)

// Comments describes all the comment related methods that the
// Terraform Enterprise API supports.
//
// TFE API docs:
// https://developer.hashicorp.com/terraform/cloud-docs/api-docs/comments
type Comments interface {
	// List all comments of the given run.
	List(ctx context.Context, runID string) (*CommentList, error)

	// Read a comment by its ID.
	Read(ctx context.Context, commentID string) (*Comment, error)

	// Create a new comment with the given options.
	Create(ctx context.Context, runID string, options CommentCreateOptions) (*Comment, error)
}

// Comments implements Comments.
type comments struct {
	client *Client
}

// CommentList represents a list of comments.
type CommentList struct {
	*Pagination
	Items []*Comment
}

// Comment represents a Terraform Enterprise comment.
type Comment struct {
	ID   string `jsonapi:"primary,comments"`
	Body string `jsonapi:"attr,body"`
}

type CommentCreateOptions struct {
	// Type is a public field utilized by JSON:API to
	// set the resource type via the field tag.
	// It is not a user-defined value and does not need to be set.
	// https://jsonapi.org/format/#crud-creating
	Type string `jsonapi:"primary,comments"`

	// Required: Body of the comment.
	Body string `jsonapi:"attr,body"`
}

// List all comments of the given run.
func (s *comments) List(ctx context.Context, runID string) (*CommentList, error) {
	if !validStringID(&runID) {
		return nil, ErrInvalidRunID
	}

	u := fmt.Sprintf("runs/%s/comments", url.PathEscape(runID))
	req, err := s.client.NewRequest("GET", u, nil)
	if err != nil {
		return nil, err
	}

	cl := &CommentList{}
	err = req.Do(ctx, cl)
	if err != nil {
		return nil, err
	}

	return cl, nil
}

// Create a new comment with the given options.
func (s *comments) Create(ctx context.Context, runID string, options CommentCreateOptions) (*Comment, error) {
	if err := options.valid(); err != nil {
		return nil, err
	}

	if !validStringID(&runID) {
		return nil, ErrInvalidRunID
	}

	u := fmt.Sprintf("runs/%s/comments", url.PathEscape(runID))
	req, err := s.client.NewRequest("POST", u, &options)
	if err != nil {
		return nil, err
	}

	comm := &Comment{}
	err = req.Do(ctx, comm)
	if err != nil {
		return nil, err
	}

	return comm, err
}

// Read a comment by its ID.
func (s *comments) Read(ctx context.Context, commentID string) (*Comment, error) {
	if !validStringID(&commentID) {
		return nil, ErrInvalidCommentID
	}

	u := fmt.Sprintf("comments/%s", url.PathEscape(commentID))
	req, err := s.client.NewRequest("GET", u, nil)
	if err != nil {
		return nil, err
	}

	comm := &Comment{}
	err = req.Do(ctx, comm)
	if err != nil {
		return nil, err
	}

	return comm, nil
}

func (o CommentCreateOptions) valid() error {
	if !validString(&o.Body) {
		return ErrInvalidCommentBody
	}

	return nil
}

// Compile-time proof of interface implementation.
var _ ConfigurationVersions = (*configurationVersions)(nil)

// ConfigurationVersions describes all the configuration version related
// methods that the Terraform Enterprise API supports.
//
// TFE API docs:
// https://developer.hashicorp.com/terraform/cloud-docs/api-docs/configuration-versions
type ConfigurationVersions interface {
	// List returns all configuration versions of a workspace.
	List(ctx context.Context, workspaceID string, options *ConfigurationVersionListOptions) (*ConfigurationVersionList, error)

	// Create is used to create a new configuration version. The created
	// configuration version will be usable once data is uploaded to it.
	Create(ctx context.Context, workspaceID string, options ConfigurationVersionCreateOptions) (*ConfigurationVersion, error)

	// CreateForRegistryModule is used to create a new configuration version
	// keyed to a registry module instead of a workspace. The created
	// configuration version will be usable once data is uploaded to it.
	//
	// **Note: This function is still in BETA and subject to change.**
	CreateForRegistryModule(ctx context.Context, moduleID RegistryModuleID) (*ConfigurationVersion, error)

	// Read a configuration version by its ID.
	Read(ctx context.Context, cvID string) (*ConfigurationVersion, error)

	// ReadWithOptions reads a configuration version by its ID using the options supplied
	ReadWithOptions(ctx context.Context, cvID string, options *ConfigurationVersionReadOptions) (*ConfigurationVersion, error)

	// Upload packages and uploads Terraform configuration files. It requires
	// the upload URL from a configuration version and the full path to the
	// configuration files on disk.
	Upload(ctx context.Context, url string, path string) error

	// Upload a tar gzip archive to the specified configuration version upload URL.
	UploadTarGzip(ctx context.Context, url string, archive io.Reader) error

	// Archive a configuration version. This can only be done on configuration versions that
	// were created with the API or CLI, are in an uploaded state, and have no runs in progress.
	Archive(ctx context.Context, cvID string) error

	// Download a configuration version.  Only configuration versions in the uploaded state may be downloaded.
	Download(ctx context.Context, cvID string) ([]byte, error)

	// SoftDeleteBackingData soft deletes the configuration version's backing data
	// **Note: This functionality is only available in Terraform Enterprise.**
	SoftDeleteBackingData(ctx context.Context, svID string) error

	// RestoreBackingData restores a soft deleted configuration version's backing data
	// **Note: This functionality is only available in Terraform Enterprise.**
	RestoreBackingData(ctx context.Context, svID string) error

	// PermanentlyDeleteBackingData permanently deletes a soft deleted configuration version's backing data
	// **Note: This functionality is only available in Terraform Enterprise.**
	PermanentlyDeleteBackingData(ctx context.Context, svID string) error
}

// configurationVersions implements ConfigurationVersions.
type configurationVersions struct {
	client *Client
}

// ConfigurationStatus represents a configuration version status.
type ConfigurationStatus string

// List all available configuration version statuses.
const (
	ConfigurationArchived ConfigurationStatus = "archived"
	ConfigurationErrored  ConfigurationStatus = "errored"
	ConfigurationFetching ConfigurationStatus = "fetching"
	ConfigurationPending  ConfigurationStatus = "pending"
	ConfigurationUploaded ConfigurationStatus = "uploaded"
)

// ConfigurationSource represents a source of a configuration version.
type ConfigurationSource string

// List all available configuration version sources.
const (
	ConfigurationSourceAPI       ConfigurationSource = "tfe-api"
	ConfigurationSourceBitbucket ConfigurationSource = "bitbucket"
	ConfigurationSourceGithub    ConfigurationSource = "github"
	ConfigurationSourceGitlab    ConfigurationSource = "gitlab"
	ConfigurationSourceAdo       ConfigurationSource = "ado"
	ConfigurationSourceTerraform ConfigurationSource = "terraform"
)

// ConfigurationVersionList represents a list of configuration versions.
type ConfigurationVersionList struct {
	*Pagination
	Items []*ConfigurationVersion
}

// ConfigurationVersion is a representation of an uploaded or ingressed
// Terraform configuration in TFE. A workspace must have at least one
// configuration version before any runs may be queued on it.
type ConfigurationVersion struct {
	ID               string              `jsonapi:"primary,configuration-versions"`
	AutoQueueRuns    bool                `jsonapi:"attr,auto-queue-runs"`
	Error            string              `jsonapi:"attr,error"`
	ErrorMessage     string              `jsonapi:"attr,error-message"`
	Source           ConfigurationSource `jsonapi:"attr,source"`
	Speculative      bool                `jsonapi:"attr,speculative"`
	Provisional      bool                `jsonapi:"attr,provisional"`
	Status           ConfigurationStatus `jsonapi:"attr,status"`
	StatusTimestamps *CVStatusTimestamps `jsonapi:"attr,status-timestamps"`
	UploadURL        string              `jsonapi:"attr,upload-url"`

	// Relations
	IngressAttributes *IngressAttributes `jsonapi:"relation,ingress-attributes"`
}

// CVStatusTimestamps holds the timestamps for individual configuration version
// statuses.
type CVStatusTimestamps struct {
	ArchivedAt time.Time `jsonapi:"attr,archived-at,rfc3339"`
	FetchingAt time.Time `jsonapi:"attr,fetching-at,rfc3339"`
	FinishedAt time.Time `jsonapi:"attr,finished-at,rfc3339"`
	QueuedAt   time.Time `jsonapi:"attr,queued-at,rfc3339"`
	StartedAt  time.Time `jsonapi:"attr,started-at,rfc3339"`
}

// ConfigVerIncludeOpt represents the available options for include query params.
// https://developer.hashicorp.com/terraform/cloud-docs/api-docs/configuration-versions#available-related-resources
type ConfigVerIncludeOpt string

const (
	ConfigVerIngressAttributes ConfigVerIncludeOpt = "ingress_attributes"
	ConfigVerRun               ConfigVerIncludeOpt = "run"
)

// ConfigurationVersionReadOptions represents the options for reading a configuration version.
type ConfigurationVersionReadOptions struct {
	// Optional: A list of relations to include. See available resources:
	// https://developer.hashicorp.com/terraform/cloud-docs/api-docs/configuration-versions#available-related-resources
	Include []ConfigVerIncludeOpt `url:"include,omitempty"`
}

// ConfigurationVersionListOptions represents the options for listing
// configuration versions.
type ConfigurationVersionListOptions struct {
	ListOptions
	// Optional: A list of relations to include. See available resources:
	// https://developer.hashicorp.com/terraform/cloud-docs/api-docs/configuration-versions#available-related-resources
	Include []ConfigVerIncludeOpt `url:"include,omitempty"`
}

// ConfigurationVersionCreateOptions represents the options for creating a
// configuration version.
type ConfigurationVersionCreateOptions struct {
	// Type is a public field utilized by JSON:API to
	// set the resource type via the field tag.
	// It is not a user-defined value and does not need to be set.
	// https://jsonapi.org/format/#crud-creating
	Type string `jsonapi:"primary,configuration-versions"`

	// Optional: When true, runs are queued automatically when the configuration version
	// is uploaded.
	AutoQueueRuns *bool `jsonapi:"attr,auto-queue-runs,omitempty"`

	// Optional: When true, this configuration version can only be used for planning.
	Speculative *bool `jsonapi:"attr,speculative,omitempty"`

	// Optional: When true, does not become the workspace's current configuration until
	// a run referencing it is ultimately applied.
	Provisional *bool `jsonapi:"attr,provisional,omitempty"`
}

// IngressAttributes include commit information associated with configuration versions sourced from VCS.
type IngressAttributes struct {
	ID                string `jsonapi:"primary,ingress-attributes"`
	Branch            string `jsonapi:"attr,branch"`
	CloneURL          string `jsonapi:"attr,clone-url"`
	CommitMessage     string `jsonapi:"attr,commit-message"`
	CommitSHA         string `jsonapi:"attr,commit-sha"`
	CommitURL         string `jsonapi:"attr,commit-url"`
	CompareURL        string `jsonapi:"attr,compare-url"`
	Identifier        string `jsonapi:"attr,identifier"`
	IsPullRequest     bool   `jsonapi:"attr,is-pull-request"`
	OnDefaultBranch   bool   `jsonapi:"attr,on-default-branch"`
	PullRequestNumber int    `jsonapi:"attr,pull-request-number"`
	PullRequestURL    string `jsonapi:"attr,pull-request-url"`
	PullRequestTitle  string `jsonapi:"attr,pull-request-title"`
	PullRequestBody   string `jsonapi:"attr,pull-request-body"`
	Tag               string `jsonapi:"attr,tag"`
	SenderUsername    string `jsonapi:"attr,sender-username"`
	SenderAvatarURL   string `jsonapi:"attr,sender-avatar-url"`
	SenderHTMLURL     string `jsonapi:"attr,sender-html-url"`

	// Links
	Links map[string]interface{} `jsonapi:"links,omitempty"`
}

// List returns all configuration versions of a workspace.
func (s *configurationVersions) List(ctx context.Context, workspaceID string, options *ConfigurationVersionListOptions) (*ConfigurationVersionList, error) {
	if !validStringID(&workspaceID) {
		return nil, ErrInvalidWorkspaceID
	}
	if err := options.valid(); err != nil {
		return nil, err
	}

	u := fmt.Sprintf("workspaces/%s/configuration-versions", url.PathEscape(workspaceID))
	req, err := s.client.NewRequest("GET", u, options)
	if err != nil {
		return nil, err
	}

	cvl := &ConfigurationVersionList{}
	err = req.Do(ctx, cvl)
	if err != nil {
		return nil, err
	}

	return cvl, nil
}

// Create is used to create a new configuration version. The created
// configuration version will be usable once data is uploaded to it.
func (s *configurationVersions) Create(ctx context.Context, workspaceID string, options ConfigurationVersionCreateOptions) (*ConfigurationVersion, error) {
	if !validStringID(&workspaceID) {
		return nil, ErrInvalidWorkspaceID
	}

	u := fmt.Sprintf("workspaces/%s/configuration-versions", url.PathEscape(workspaceID))
	req, err := s.client.NewRequest("POST", u, &options)
	if err != nil {
		return nil, err
	}

	cv := &ConfigurationVersion{}
	err = req.Do(ctx, cv)
	if err != nil {
		return nil, err
	}

	return cv, nil
}

func (s *configurationVersions) CreateForRegistryModule(ctx context.Context, moduleID RegistryModuleID) (*ConfigurationVersion, error) {
	if err := moduleID.valid(); err != nil {
		return nil, err
	}

	u := fmt.Sprintf("%s/configuration-versions", testRunsPath(moduleID))
	req, err := s.client.NewRequest("POST", u, nil)
	if err != nil {
		return nil, err
	}

	cv := &ConfigurationVersion{}
	err = req.Do(ctx, cv)
	if err != nil {
		return nil, err
	}

	return cv, nil
}

// Read a configuration version by its ID.
func (s *configurationVersions) Read(ctx context.Context, cvID string) (*ConfigurationVersion, error) {
	return s.ReadWithOptions(ctx, cvID, nil)
}

// Read a configuration version by its ID with the given options.
func (s *configurationVersions) ReadWithOptions(ctx context.Context, cvID string, options *ConfigurationVersionReadOptions) (*ConfigurationVersion, error) {
	if !validStringID(&cvID) {
		return nil, ErrInvalidConfigVersionID
	}
	if err := options.valid(); err != nil {
		return nil, err
	}

	u := fmt.Sprintf("configuration-versions/%s", url.PathEscape(cvID))
	req, err := s.client.NewRequest("GET", u, options)
	if err != nil {
		return nil, err
	}

	cv := &ConfigurationVersion{}
	err = req.Do(ctx, cv)
	if err != nil {
		return nil, err
	}

	return cv, nil
}

// Upload packages and uploads Terraform configuration files. It requires the
// upload URL from a configuration version and the path to the configuration
// files on disk.
func (s *configurationVersions) Upload(ctx context.Context, uploadURL, path string) error {
	body, err := packContents(path)
	if err != nil {
		return err
	}

	return s.UploadTarGzip(ctx, uploadURL, body)
}

// UploadTarGzip is used to upload Terraform configuration files contained a tar gzip archive.
// Any stream implementing io.Reader can be passed into this method. This method is also
// particularly useful for tar streams created by non-default go-slug configurations.
//
// **Note**: This method does not validate the content being uploaded and is therefore the caller's
// responsibility to ensure the raw content is a valid Terraform configuration.
func (s *configurationVersions) UploadTarGzip(ctx context.Context, uploadURL string, archive io.Reader) error {
	return s.client.doForeignPUTRequest(ctx, uploadURL, archive)
}

// Archive a configuration version. This can only be done on configuration versions that
// were created with the API or CLI, are in an uploaded state, and have no runs in progress.
func (s *configurationVersions) Archive(ctx context.Context, cvID string) error {
	if !validStringID(&cvID) {
		return ErrInvalidConfigVersionID
	}

	body := bytes.NewBuffer(nil)

	u := fmt.Sprintf("configuration-versions/%s/actions/archive", url.PathEscape(cvID))
	req, err := s.client.NewRequest("POST", u, body)
	if err != nil {
		return err
	}

	return req.Do(ctx, nil)
}

func (o *ConfigurationVersionReadOptions) valid() error {
	return nil
}

func (o *ConfigurationVersionListOptions) valid() error {
	return nil
}

// Download a configuration version.  Only configuration versions in the uploaded state may be downloaded.
func (s *configurationVersions) Download(ctx context.Context, cvID string) ([]byte, error) {
	if !validStringID(&cvID) {
		return nil, ErrInvalidConfigVersionID
	}

	u := fmt.Sprintf("configuration-versions/%s/download", url.PathEscape(cvID))
	req, err := s.client.NewRequest("GET", u, nil)
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	err = req.Do(ctx, &buf)
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func (s *configurationVersions) SoftDeleteBackingData(ctx context.Context, cvID string) error {
	return s.manageBackingData(ctx, cvID, "soft_delete_backing_data")
}

func (s *configurationVersions) RestoreBackingData(ctx context.Context, cvID string) error {
	return s.manageBackingData(ctx, cvID, "restore_backing_data")
}

func (s *configurationVersions) PermanentlyDeleteBackingData(ctx context.Context, cvID string) error {
	return s.manageBackingData(ctx, cvID, "permanently_delete_backing_data")
}

func (s *configurationVersions) manageBackingData(ctx context.Context, cvID, action string) error {
	if !validStringID(&cvID) {
		return ErrInvalidConfigVersionID
	}

	u := fmt.Sprintf("configuration-versions/%s/actions/%s", cvID, action)
	req, err := s.client.NewRequest("POST", u, nil)
	if err != nil {
		return err
	}

	return req.Do(ctx, nil)
}

// Copyright IBM Corp. 2018, 2026
// SPDX-License-Identifier: MPL-2.0

const (
	// AuthenticationTokensPath is the API path for authentication tokens.
	AuthenticationTokensPath = "authentication-tokens/%s"

	// AdminSCIMTokensPath is the API path for admin SCIM tokens.
	AdminSCIMTokensPath = "admin/scim-tokens"

	// AdminSCIMGroupsPath is the API path for admin SCIM groups.
	AdminSCIMGroupsPath = "admin/scim-groups"

	// AdminSCIMGroupMappingPath is the API path for admin SCIM group mapping.
	AdminSCIMGroupMappingPath = "admin/teams/%s/scim-group-mapping"
)

// Compile-time proof of interface implementation.
var _ CostEstimates = (*costEstimates)(nil)

// CostEstimates describes all the costEstimate related methods that
// the Terraform Enterprise API supports.
//
// TFE API docs: https://developer.hashicorp.com/terraform/cloud-docs/api-docs/cost-estimates
type CostEstimates interface {
	// Read a costEstimate by its ID.
	Read(ctx context.Context, costEstimateID string) (*CostEstimate, error)

	// Logs retrieves the logs of a costEstimate.
	Logs(ctx context.Context, costEstimateID string) (io.Reader, error)
}

// costEstimates implements CostEstimates.
type costEstimates struct {
	client *Client
}

// CostEstimateStatus represents a costEstimate state.
type CostEstimateStatus string

// List all available costEstimate statuses.
const (
	CostEstimateCanceled              CostEstimateStatus = "canceled"
	CostEstimateErrored               CostEstimateStatus = "errored"
	CostEstimateFinished              CostEstimateStatus = "finished"
	CostEstimatePending               CostEstimateStatus = "pending"
	CostEstimateQueued                CostEstimateStatus = "queued"
	CostEstimateSkippedDueToTargeting CostEstimateStatus = "skipped_due_to_targeting"
)

// CostEstimate represents a Terraform Enterprise costEstimate.
type CostEstimate struct {
	ID                      string                        `jsonapi:"primary,cost-estimates"`
	DeltaMonthlyCost        string                        `jsonapi:"attr,delta-monthly-cost"`
	ErrorMessage            string                        `jsonapi:"attr,error-message"`
	MatchedResourcesCount   int                           `jsonapi:"attr,matched-resources-count"`
	PriorMonthlyCost        string                        `jsonapi:"attr,prior-monthly-cost"`
	ProposedMonthlyCost     string                        `jsonapi:"attr,proposed-monthly-cost"`
	ResourcesCount          int                           `jsonapi:"attr,resources-count"`
	Status                  CostEstimateStatus            `jsonapi:"attr,status"`
	StatusTimestamps        *CostEstimateStatusTimestamps `jsonapi:"attr,status-timestamps"`
	UnmatchedResourcesCount int                           `jsonapi:"attr,unmatched-resources-count"`
}

// CostEstimateStatusTimestamps holds the timestamps for individual costEstimate statuses.
type CostEstimateStatusTimestamps struct {
	CanceledAt              time.Time `jsonapi:"attr,canceled-at,rfc3339"`
	ErroredAt               time.Time `jsonapi:"attr,errored-at,rfc3339"`
	FinishedAt              time.Time `jsonapi:"attr,finished-at,rfc3339"`
	PendingAt               time.Time `jsonapi:"attr,pending-at,rfc3339"`
	QueuedAt                time.Time `jsonapi:"attr,queued-at,rfc3339"`
	SkippedDueToTargetingAt time.Time `jsonapi:"attr,skipped-due-to-targeting-at,rfc3339"`
}

// Read a costEstimate by its ID.
func (s *costEstimates) Read(ctx context.Context, costEstimateID string) (*CostEstimate, error) {
	if !validStringID(&costEstimateID) {
		return nil, ErrInvalidCostEstimateID
	}

	u := fmt.Sprintf("cost-estimates/%s", url.PathEscape(costEstimateID))
	req, err := s.client.NewRequest("GET", u, nil)
	if err != nil {
		return nil, err
	}

	ce := &CostEstimate{}
	err = req.Do(ctx, ce)
	if err != nil {
		return nil, err
	}

	return ce, nil
}

// Logs retrieves the logs of a costEstimate.
func (s *costEstimates) Logs(ctx context.Context, costEstimateID string) (io.Reader, error) {
	if !validStringID(&costEstimateID) {
		return nil, ErrInvalidCostEstimateID
	}

	// Loop until the context is canceled or the cost estimate is finished
	// running. The cost estimate logs are not streamed and so only available
	// once the estimate is finished.
	for {
		// Get the costEstimate to make sure it exists.
		ce, err := s.Read(ctx, costEstimateID)
		if err != nil {
			return nil, err
		}

		switch ce.Status {
		case CostEstimateQueued:
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(1000 * time.Millisecond):
				continue
			}
		}

		u := fmt.Sprintf("cost-estimates/%s/output", url.PathEscape(costEstimateID))
		req, err := s.client.NewRequest("GET", u, nil)
		if err != nil {
			return nil, err
		}

		logs := bytes.NewBuffer(nil)
		err = req.Do(ctx, logs)
		if err != nil {
			return nil, err
		}

		return logs, nil
	}
}

// DataRetentionPolicyChoice is a choice type struct that represents the possible types
// of a drp returned by a polymorphic relationship. If a value is available, exactly one field
// will be non-nil.
type DataRetentionPolicyChoice struct {
	DataRetentionPolicy            *DataRetentionPolicy
	DataRetentionPolicyDeleteOlder *DataRetentionPolicyDeleteOlder
	DataRetentionPolicyDontDelete  *DataRetentionPolicyDontDelete
}

// Returns whether one of the choices is populated
func (d DataRetentionPolicyChoice) IsPopulated() bool {
	return d.DataRetentionPolicy != nil ||
		d.DataRetentionPolicyDeleteOlder != nil ||
		d.DataRetentionPolicyDontDelete != nil
}

// Convert the DataRetentionPolicyChoice to the legacy DataRetentionPolicy struct
// Returns nil if the policy cannot be represented by a legacy DataRetentionPolicy
func (d *DataRetentionPolicyChoice) ConvertToLegacyStruct() *DataRetentionPolicy {
	if d == nil {
		return nil
	}
	if d.DataRetentionPolicy != nil {
		// TFE v202311-1 and v202312-1 will return a deprecated DataRetentionPolicy in the DataRetentionPolicyChoice struct
		return d.DataRetentionPolicy
	} else if d.DataRetentionPolicyDeleteOlder != nil {
		// DataRetentionPolicy was functionally replaced by DataRetentionPolicyDeleteOlder in TFE v202401
		return &DataRetentionPolicy{
			ID:                   d.DataRetentionPolicyDeleteOlder.ID,
			DeleteOlderThanNDays: d.DataRetentionPolicyDeleteOlder.DeleteOlderThanNDays,
		}
	}
	return nil
}

// DataRetentionPolicy describes the retention policy of deleting records older than the specified number of days.
//
// Deprecated: Use DataRetentionPolicyDeleteOlder instead. This is the original representation of a
// data retention policy, only present in TFE v202311-1 and v202312-1
type DataRetentionPolicy struct {
	ID                   string `jsonapi:"primary,data-retention-policies"`
	DeleteOlderThanNDays int    `jsonapi:"attr,delete-older-than-n-days"`
}

// DataRetentionPolicySetOptions is the options for a creating a DataRetentionPolicy.
//
// Deprecated: Use DataRetentionPolicyDeleteOlder variations instead
type DataRetentionPolicySetOptions struct {
	// Type is a public field utilized by JSON:API to
	// set the resource type via the field tag.
	// It is not a user-defined value and does not need to be set.
	// https://jsonapi.org/format/#crud-creating
	Type string `jsonapi:"primary,data-retention-policies"`

	// DeleteOlderThanNDays is the number of days to retain records for.
	DeleteOlderThanNDays int `jsonapi:"attr,delete-older-than-n-days"`
}

// DataRetentionPolicyDeleteOlder describes the retention policy of deleting records older than the specified number of days.
type DataRetentionPolicyDeleteOlder struct {
	ID string `jsonapi:"primary,data-retention-policy-delete-olders"`

	// DeleteOlderThanNDays is the number of days to retain records for.
	DeleteOlderThanNDays int `jsonapi:"attr,delete-older-than-n-days"`
}

// DataRetentionPolicyDontDelete describes the retention policy of never deleting records.
type DataRetentionPolicyDontDelete struct {
	ID string `jsonapi:"primary,data-retention-policy-dont-deletes"`
}

// DataRetentionPolicyDeleteOlderSetOptions describes the options for a creating a DataRetentionPolicyDeleteOlder.
type DataRetentionPolicyDeleteOlderSetOptions struct {
	// Type is a public field utilized by JSON:API to
	// set the resource type via the field tag.
	// It is not a user-defined value and does not need to be set.
	// https://jsonapi.org/format/#crud-creating
	Type string `jsonapi:"primary,data-retention-policy-delete-olders"`

	// DeleteOlderThanNDays is the number of days records will be retained for after their creation.
	DeleteOlderThanNDays int `jsonapi:"attr,delete-older-than-n-days"`
}

// DataRetentionPolicyDontDeleteSetOptions describes the options for a creating a DataRetentionPolicyDontDelete.
type DataRetentionPolicyDontDeleteSetOptions struct {
	// Type is a public field utilized by JSON:API to
	// set the resource type via the field tag.
	// It is not a user-defined value and does not need to be set.
	// https://jsonapi.org/format/#crud-creating
	Type string `jsonapi:"primary,data-retention-policy-dont-deletes"`
}

// error we get when trying to unmarshal a data retention policy from TFE v202401+ into the deprecated DataRetentionPolicy struct
var drpUnmarshalEr = regexp.MustCompile(`Trying to Unmarshal an object of type \".+\", but \"data-retention-policies\" does not match`)

// Copyright IBM Corp. 2018, 2026
// SPDX-License-Identifier: MPL-2.0

// Generic errors applicable to all resources.
var (
	// ErrUnauthorized is returned when receiving a 401.
	ErrUnauthorized = errors.New("unauthorized")

	// ErrResourceNotFound is returned when receiving a 404.
	ErrResourceNotFound = errors.New("resource not found")

	// ErrMissingDirectory is returned when the path does not have an existing directory.
	ErrMissingDirectory = errors.New("path needs to be an existing directory")

	// ErrNamespaceNotAuthorized is returned when a user attempts to perform an action
	// on a namespace (organization) they do not have access to.
	ErrNamespaceNotAuthorized = errors.New("namespace not authorized")
)

// Options/fields that cannot be defined
var (
	ErrUnsupportedOperations = errors.New("operations is deprecated and cannot be specified when execution mode is used")

	ErrUnsupportedPrivateKey = errors.New("private Key can only be present with Azure DevOps Server service provider")

	ErrUnsupportedBothTagsRegexAndFileTriggersEnabled = errors.New(`"TagsRegex" cannot be populated when "FileTriggersEnabled" is true`)

	ErrUnsupportedBothTagsRegexAndTriggerPatterns = errors.New(`"TagsRegex" and "TriggerPrefixes" cannot be populated at the same time`)

	ErrUnsupportedBothTagsRegexAndTriggerPrefixes = errors.New(`"TagsRegex" and "TriggerPatterns" cannot be populated at the same time`)

	ErrUnsupportedRunTriggerType = errors.New(`"RunTriggerType" must be "inbound" when requesting "include" query params`)

	ErrUnsupportedBothTriggerPatternsAndPrefixes = errors.New(`"TriggerPatterns" and "TriggerPrefixes" cannot be populated at the same time`)

	ErrUnsupportedBothNamespaceAndPrivateRegistryName = errors.New(`"Namespace" cannot be populated when "RegistryName" is "private"`)
)

// Library errors that usually indicate a bug in the implementation of go-tfe
var (
	ErrItemsMustBeSlice = errors.New(`model field "Items" must be a slice`) // ErrItemsMustBeSlice is returned when an API response attribute called Items is not a slice

	ErrInvalidRequestBody = errors.New("go-tfe bug: DELETE/PATCH/POST body must be nil, ptr, or ptr slice") // ErrInvalidRequestBody is returned when a request body for DELETE/PATCH/POST is not a reference type

	ErrInvalidStructFormat = errors.New("go-tfe bug: struct can't use both json and jsonapi attributes") // ErrInvalidStructFormat is returned when a mix of json and jsonapi tagged fields are used in the same struct
)

// Resource Errors
var (
	// ErrWorkspaceLocked is returned when trying to lock a locked workspace.
	ErrWorkspaceLocked = errors.New("workspace already locked")

	// ErrWorkspaceNotLocked is returned when trying to unlock a unlocked workspace.
	ErrWorkspaceNotLocked = errors.New("workspace already unlocked")

	// ErrWorkspaceLockedByRun is returned when trying to unlock a workspace locked by a run.
	ErrWorkspaceLockedByRun = errors.New("unable to unlock workspace locked by run")

	// ErrWorkspaceLockedByTeam is returned when trying to unlock a workspace locked by a team.
	ErrWorkspaceLockedByTeam = errors.New("unable to unlock workspace locked by team")

	// ErrWorkspaceLockedByUser is returned when trying to unlock a workspace locked by a user.
	ErrWorkspaceLockedByUser = errors.New("unable to unlock workspace locked by user")

	// ErrWorkspaceLockedStateVersionStillPending is returned when trying to unlock whose
	// latest state version is still pending.
	ErrWorkspaceLockedStateVersionStillPending = errors.New("unable to unlock workspace while state version upload is still pending")

	// ErrWorkspaceStillProcessing is returned when a workspace is still processing state
	// to determine if it is safe to delete. "conflict" followed by newline is used to
	// preserve go-tfe version compatibility with the error constructed at runtime before it was
	// defined here.
	ErrWorkspaceStillProcessing = errors.New("conflict\nLatest workspace state is being processed to discover resources, please try again later")

	// ErrWorkspaceNotSafeToDelete is returned when a workspace has processed state and
	// is determined to still have resources present. "conflict" followed by newline is used to
	// preserve go-tfe version compatibility with the error constructed at runtime before it was
	// defined here.
	ErrWorkspaceNotSafeToDelete = errors.New("conflict\nworkspace cannot be safely deleted because it is still managing resources")

	// ErrWorkspaceLockedCannotDelete is returned when a workspace cannot be safely deleted when
	// it is locked. "conflict" followed by newline is used to preserve go-tfe version
	// compatibility with the error constructed at runtime before it was defined here.
	ErrWorkspaceLockedCannotDelete = errors.New("conflict\nWorkspace is currently locked. Workspace must be unlocked before it can be safely deleted")

	// ErrHYOKCannotBeDisabled is returned when attempting to disable HYOK on a workspace that already has it enabled.
	ErrHYOKCannotBeDisabled = errors.New("bad request\n\nhyok may not be disabled once it has been turned on for a workspace")

	// ErrSCIMTeamAlreadyMapped is returned when attempting to create a SCIM group mapping
	// for a team that is already mapped to a SCIM group.
	ErrSCIMTeamAlreadyMapped = errors.New("conflict\n\nTeam is already linked to a SCIM group")

	// ErrSCIMGroupMappingOwnersTeam is the stable detail substring of the API error
	// returned when attempting to link the owners team to a SCIM group. The HTTP
	// status-code title (e.g., "unprocessable entity" vs "unprocessable content")
	// varies across TFE releases, so the title is intentionally omitted and this
	// value is meant for substring matching only — use require.ErrorContains, not
	// errors.Is or require.EqualError. The leading capital matches the server's response verbatim.
	//nolint:staticcheck // ST1005: server response begins with a capital letter; preserved for verbatim substring matching.
	ErrSCIMGroupMappingOwnersTeam = errors.New("Owners team SCIM linking is not yet supported")

	// ErrSCIMGroupMappingSiteAdminGroup is the stable detail substring of the API error
	// returned when attempting to link a team to the site admin SCIM group, which is
	// not allowed. The HTTP status-code title (e.g., "unprocessable entity" vs
	// "unprocessable content") varies across TFE releases, so the title is
	// intentionally omitted and this value is meant for substring matching only —
	// use require.ErrorContains, not errors.Is or require.EqualError. The leading
	// capital matches the server's response verbatim.
	//nolint:staticcheck // ST1005: server response begins with a capital letter; preserved for verbatim substring matching.
	ErrSCIMGroupMappingSiteAdminGroup = errors.New("The site admin group cannot be linked to a team")

	// ErrSCIMGroupMappingTeamNotLinked is returned when attempting to update a SCIM
	// group mapping for a team that is not linked to a SCIM group.
	ErrSCIMGroupMappingTeamNotLinked = errors.New("conflict\n\nTeam is not linked to a SCIM group")
)

// Invalid values for resources/struct fields
var (
	ErrInvalidWorkspaceID = errors.New("invalid value for workspace ID")

	ErrInvalidWorkspaceValue = errors.New("invalid value for workspace")

	ErrInvalidTerraformVersionID = errors.New("invalid value for terraform version ID")

	ErrInvalidTerraformVersionType = errors.New("invalid type for terraform version. Please use 'terraform-version'")

	ErrInvalidOPAVersionID = errors.New("invalid value for OPA version ID")

	ErrInvalidSentinelVersionID = errors.New("invalid value for Sentinel version ID")

	ErrInvalidConfigVersionID = errors.New("invalid value for configuration version ID")

	ErrInvalidCostEstimateID = errors.New("invalid value for cost estimate ID")

	ErrInvalidSMTPAuth = errors.New("invalid smtp auth type")

	ErrInvalidAgentPoolID = errors.New("invalid value for agent pool ID")

	ErrInvalidAgentTokenID = errors.New("invalid value for agent token ID")

	ErrInvalidRunID = errors.New("invalid value for run ID")

	ErrInvalidRunEventID = errors.New("invalid value for run event ID")

	ErrInvalidProjectID = errors.New("invalid value for project ID")

	ErrInvalidRegistryComponentID = errors.New("invalid value for registry component ID")

	ErrInvalidRegistryModuleID = errors.New("invalid value for registry module ID")

	ErrInvalidRegistryProviderID = errors.New("invalid value for registry provider ID")

	ErrInvalidPagination = errors.New("invalid value for page size or number")

	ErrInvalidReservedTagKeyID = errors.New("invalid value for reserved tag key ID")

	ErrInvalidRunTaskCategory = errors.New(`category must be "task"`)

	ErrInvalidRunTaskID = errors.New("invalid value for run task ID")

	ErrInvalidRunTaskURL = errors.New("invalid url for run task URL")

	ErrInvalidWorkspaceRunTaskID = errors.New("invalid value for workspace run task ID")

	ErrInvalidWorkspaceRunTaskType = errors.New(`invalid value for type, please use "workspace-tasks"`)

	ErrInvalidTaskResultID = errors.New("invalid value for task result ID")

	ErrInvalidTaskStageID = errors.New("invalid value for task stage ID")

	ErrInvalidApplyID = errors.New("invalid value for apply ID")

	ErrInvalidOrg = errors.New("invalid value for organization")

	ErrInvalidName = errors.New("invalid value for name")

	ErrInvalidNotificationConfigID = errors.New("invalid value for notification configuration ID")

	ErrInvalidMembership = errors.New("invalid value for membership")

	ErrInvalidMembershipIDs = errors.New("invalid value for organization membership ids")

	ErrInvalidOauthClientID = errors.New("invalid value for OAuth client ID")

	ErrInvalidOauthTokenID = errors.New("invalid value for OAuth token ID")

	ErrInvalidPolicySetID = errors.New("invalid value for policy set ID")

	ErrInvalidPolicyCheckID = errors.New("invalid value for policy check ID")

	ErrInvalidPolicyEvaluationID = errors.New("invalid value for policy evaluation ID")

	ErrInvalidPolicySetOutcomeID = errors.New("invalid value for policy set outcome ID")

	ErrInvalidTag = errors.New("invalid tag id")

	ErrInvalidPlanExportID = errors.New("invalid value for plan export ID")

	ErrInvalidPlanID = errors.New("invalid value for plan ID")

	ErrInvalidParamID = errors.New("invalid value for parameter ID")

	ErrInvalidPolicyID = errors.New("invalid value for policy ID")

	ErrInvalidProvider = errors.New("invalid value for provider")

	ErrInvalidProviderSetID = errors.New("invalid value for provider set ID")

	ErrInvalidVersion = errors.New("invalid value for version")

	ErrInvalidRunTriggerID = errors.New("invalid value for run trigger ID")

	ErrInvalidRunTriggerType = errors.New(`invalid value or no value for RunTriggerType. It must be either "inbound" or "outbound"`)

	ErrInvalidIncludeValue = errors.New(`invalid value for "include" field`)

	ErrInvalidSHHKeyID = errors.New("invalid value for SSH key ID")

	ErrInvalidStateVerID = errors.New("invalid value for state version ID")

	ErrInvalidOutputID = errors.New("invalid value for state version output ID")

	ErrInvalidAccessTeamID = errors.New("invalid value for team access ID")

	ErrInvalidTeamProjectAccessID = errors.New("invalid value for team project access ID")

	ErrInvalidTeamProjectAccessType = errors.New("invalid type for team project access")

	ErrInvalidTeamID = errors.New("invalid value for team ID")

	ErrInvalidUsernames = errors.New("invalid value for usernames")

	ErrInvalidUserID = errors.New("invalid value for user ID")

	ErrInvalidUserValue = errors.New("invalid value for user")

	ErrInvalidTokenID = errors.New("invalid value for token ID")

	ErrInvalidCategory = errors.New("category must be policy-set")

	ErrInvalidPolicies = errors.New("must provide at least one policy")

	ErrInvalidVariableID = errors.New("invalid value for variable ID")

	ErrInvalidNotificationTrigger = errors.New("invalid value for notification trigger")

	ErrInvalidVariableSetID = errors.New("invalid variable set ID")

	ErrInvalidCommentID = errors.New("invalid value for comment ID")

	ErrInvalidCommentBody = errors.New("invalid value for comment body")

	ErrInvalidNamespace = errors.New("invalid value for namespace")

	ErrInvalidKeyID = errors.New("invalid value for key-id")

	ErrInvalidOS = errors.New("invalid value for OS")

	ErrInvalidArch = errors.New("invalid value for arch")

	ErrInvalidAgentID = errors.New("invalid value for Agent ID")

	ErrInvalidModuleID = errors.New("invalid value for module ID")

	ErrInvalidRegistryName = errors.New(`invalid value for registry-name. It must be either "private" or "public"`)

	ErrInvalidCallbackURL = errors.New("invalid value for callback URL")

	ErrInvalidAccessToken = errors.New("invalid value for access token")

	ErrInvalidTaskResultsCallbackStatus = fmt.Errorf("invalid value for task result status. Must be either `%s`, `%s`, or `%s`", TaskFailed, TaskPassed, TaskRunning)

	ErrInvalidDescriptionConflict = errors.New("invalid attributes\n\nValidation failed: Description has already been taken")

	ErrInvalidOIDC = errors.New("invalid value for OIDC configuration ID")

	ErrInvalidHYOK = errors.New("invalid value for HYOK configuration ID")

	ErrInvalidHYOKCustomerKeyVersion = errors.New("invalid value for HYOK Customer key version ID")

	ErrInvalidHYOKEncryptedDataKey = errors.New("invalid value for HYOK encrypted data key ID")

	ErrInvalidStackID = errors.New("invalid value for stack ID")

	ErrInvalidRemoteStateOptions = errors.New("invalid attribute\n\nProject remote state cannot be enabled when global remote state sharing is enabled")

	ErrInvalidSAMLProviderType = errors.New("invalid SAML provider type")

	ErrInvalidTFPolicyEvaluationID = errors.New("invalid value for tfpolicy evaluation ID")

	ErrInvalidExplorerViewType = errors.New("explorer query type is required")

	ErrInvalidExplorerFilterField = errors.New("explorer filter field is required")

	ErrInvalidExplorerFilterOperator = errors.New("explorer filter operator is required")
)

var (
	ErrRequiredAccess = errors.New("access is required")

	ErrRequiredAgentPoolID = errors.New("'agent' execution mode requires an agent pool ID to be specified")

	ErrRequiredAgentMode                      = errors.New("specifying an agent pool ID requires 'agent' execution mode")
	ErrRequiredBranchWhenTestsEnabled         = errors.New("VCS branch is required when enabling tests")
	ErrBranchMustBeEmptyWhenTagsEnabled       = errors.New("VCS branch must be empty to enable tags")
	ErrRequiredCategory                       = errors.New("category is required")
	ErrAgentPoolNotRequiredForRemoteExecution = errors.New("'remote' execution mode does not support agent pool IDs")
	ErrRequiredDestinationType                = errors.New("destination type is required")

	ErrRequiredDataType = errors.New("data type is required")

	ErrRequiredKey = errors.New("key is required")

	ErrRequiredName = errors.New("name is required")

	ErrRequiredQuery = errors.New("query cannot be empty")

	ErrRequiredEnabled = errors.New("enabled is required")

	ErrRequiredEnforce = errors.New("enforce or enforcement-level is required")

	ErrConflictingEnforceEnforcementLevel = errors.New("enforce and enforcement-level may not both be specified together")

	ErrProviderSetGlobalRelationships = errors.New("global provider set cannot be assigned to workspace or project")

	ErrRequiredEnforcementPath = errors.New("enforcement path is required")

	ErrRequiredEnforcementMode = errors.New("enforcement mode is required")

	ErrRequiredEmail = errors.New("email is required")

	ErrRequiredM5 = errors.New("MD5 is required")

	ErrRequiredProviderSource = errors.New("provider source is required")

	ErrRequiredConfigurationHcl = errors.New("configuration HCL is required")

	ErrRequiredURL = errors.New("url is required")

	ErrRequiredArchsOrURLAndSha = errors.New("valid archs or url and sha are required")

	ErrRequiredAPIURL = errors.New("API URL is required")

	ErrRequiredHTTPURL = errors.New("HTTP URL is required")

	ErrRequiredServiceProvider = errors.New("service provider is required")

	ErrRequiredProvider = errors.New("provider is required")

	ErrRequiredProviderSetID = errors.New("provider set ID is required")

	ErrRequiredOauthToken = errors.New("OAuth token is required")

	ErrRequiredOauthTokenOrGithubAppInstallationID = errors.New("either oauth token ID or github app installation ID is required")

	ErrRequiredTestNumber = errors.New("TestNumber is required")

	ErrMissingTagIdentifier = errors.New("must specify at least one tag by ID or name")

	ErrAgentTokenDescription = errors.New("agent token description can't be blank")

	ErrRequiredTagID = errors.New("you must specify at least one tag id to remove")

	ErrRequiredTagWorkspaceID = errors.New("you must specify at least one workspace to add tag to")

	ErrRequiredWorkspace = errors.New("workspace is required")

	ErrRequiredProject = errors.New("project is required")

	ErrRequiredWorkspaceID = errors.New("workspace ID is required")

	ErrRequiredProjectID = errors.New("project ID is required")

	ErrRequiredStackID = errors.New("stack ID is required")

	ErrWorkspacesRequired = errors.New("workspaces is required")

	ErrWorkspaceMinLimit = errors.New("must provide at least one workspace")

	ErrProjectMinLimit = errors.New("must provide at least one project")

	ErrRequiredTagSelectors = errors.New("tag selectors is required")

	ErrTagSelectorMinLimit = errors.New("must provide at least one tag selector")

	ErrRequiredPlan = errors.New("plan is required")

	ErrRequiredPolicies = errors.New("policies is required")

	ErrRequiredVersion = errors.New("version is required")

	ErrRequiredVCSRepo = errors.New("vcs repo is required")

	ErrRequiredIdentifier = errors.New("identifier is required")

	ErrRequiredDisplayIdentifier = errors.New("display identifier is required")

	ErrRequiredSha = errors.New("sha is required")

	ErrRequiredSourceable = errors.New("sourceable is required")

	ErrRequiredValue = errors.New("value is required")

	ErrRequiredOrg = errors.New("organization is required")

	ErrRequiredTeam = errors.New("team is required")

	ErrRequiredStateVerListOps = errors.New("StateVersionListOptions is required")

	ErrRequiredTeamAccessListOps = errors.New("TeamAccessListOptions is required")

	ErrRequiredTeamProjectAccessListOps = errors.New("TeamProjectAccessListOptions is required")

	ErrRequiredRunTriggerListOps = errors.New("RunTriggerListOptions is required")

	ErrRequiredTFVerCreateOps = errors.New("version, URL and sha is required for AdminTerraformVersionCreateOptions")

	ErrRequiredOPAVerCreateOps = errors.New("version, URL and sha is required for AdminOPAVersionCreateOptions")

	ErrRequiredSentinelVerCreateOps = errors.New("version, URL and sha is required for AdminSentinelVersionCreateOptions")

	ErrRequiredRegistryComponentCreateOps = errors.New("type and name is required for RegistryComponentCreateOptions")

	ErrRequiredSerial = errors.New("serial is required")

	ErrRequiredState = errors.New("state is required")

	ErrRequiredSHHKeyID = errors.New("SSH key ID is required")

	ErrRequiredOnlyOneField = errors.New("only one of usernames or organization membership ids can be provided")

	ErrRequiredUsernameOrMembershipIds = errors.New("usernames or organization membership ids are required")

	ErrRequiredGlobalFlag = errors.New("global flag is required")

	ErrRequiredWorkspacesList = errors.New("no workspaces list provided")

	ErrRequiredStacksList = errors.New("no stacks list provided")

	ErrCommentBody = errors.New("comment body is required")

	ErrEmptyTeamName = errors.New("team name can not be empty")

	ErrInvalidEmail = errors.New("email is invalid")

	ErrRequiredPrivateRegistry = errors.New("only private registry is allowed")

	ErrRequiredOS = errors.New("OS is required")

	ErrRequiredArch = errors.New("arch is required")

	ErrRequiredShasum = errors.New("shasum is required")

	ErrRequiredFilename = errors.New("filename is required")

	ErrInvalidAsciiArmor = errors.New("ASCII Armor is invalid")

	ErrRequiredNamespace = errors.New("namespace is required for public registry")

	ErrRequiredRegistryModule = errors.New("registry module is required")

	ErrRequiredTagBindings = errors.New("TagBindings are required")

	ErrInvalidTestRunID = errors.New("invalid value for test run id")

	ErrInvalidQueryRunID = errors.New("invalid value for query run id")

	ErrTerraformVersionValidForPlanOnly = errors.New("setting terraform-version is only valid when plan-only is set to true")

	ErrStateMustBeOmitted = errors.New("when uploading state, the State and JSONState strings must be omitted from options")

	ErrRequiredRawState = errors.New("RawState is required")

	ErrStateVersionUploadNotSupported = errors.New("upload not supported by this version of Terraform Enterprise")

	ErrSanitizedStateUploadURLMissing = errors.New("sanitized state upload URL is missing")

	ErrRequiredRoleARN = errors.New("role-arn is required for AWS OIDC configuration")

	ErrRequiredServiceAccountEmail = errors.New("service-account-email is required for GCP OIDC configuration")

	ErrRequiredProjectNumber = errors.New("project-number is required for GCP OIDC configuration")

	ErrRequiredWorkloadProviderName = errors.New("workload-provider-name is required for GCP OIDC configuration")

	ErrRequiredClientID = errors.New("client-id is required for Azure OIDC configuration")

	ErrRequiredSubscriptionID = errors.New("subscription-id is required for Azure OIDC configuration")

	ErrRequiredTenantID = errors.New("tenant-id is required for Azure OIDC configuration")

	ErrRequiredVaultAddress = errors.New("address is required for Vault OIDC configuration")

	ErrRequiredRoleName = errors.New("role is required for Vault OIDC configuration")

	ErrRequiredKEKID = errors.New("kek-id is required for HYOK configuration")

	ErrRequiredOIDCConfiguration = errors.New("oidc-configuration is required for HYOK configuration")

	ErrRequiredAgentPool = errors.New("agent-pool is required for HYOK configuration")

	ErrRequiredKMSOptions = errors.New("kms-options is required for HYOK configuration")

	ErrRequiredKMSOptionsKeyRegion = errors.New("kms-options.key-region is required for HYOK configuration with AWS OIDC")

	ErrRequiredKMSOptionsKeyLocation = errors.New("kms-options.key-location is required for HYOK configuration with GCP OIDC")

	ErrRequiredKMSOptionsKeyRingID = errors.New("kms-options.key-ring-id is required for HYOK configuration with GCP OIDC")

	ErrSCIMTokenDescription = errors.New("SCIM token description can't be blank")

	ErrInvalidSCIMGroupID = errors.New("invalid value for SCIM group ID")

	ErrSCIMSyncPausedNil = errors.New("SCIM Sync can either be paused or unpaused, can not be nil")

	ErrRequiredSCIMGroupMappingCreateOps = errors.New("Create Options are required to create a SCIM Group Mapping")

	ErrRequiredSCIMGroupMappingUpdateOps = errors.New("Update Options are required to update SCIM Group Mapping")
)

// Copyright IBM Corp. 2018, 2026
// SPDX-License-Identifier: MPL-2.0

// GCPOIDCConfigurations describes all the GCP OIDC configuration related methods that the HCP Terraform API supports.
// HCP Terraform API docs:
// https://developer.hashicorp.com/terraform/cloud-docs/api-docs/hold-your-own-key/oidc-configurations/gcp
type GCPOIDCConfigurations interface {
	Create(ctx context.Context, organization string, options GCPOIDCConfigurationCreateOptions) (*GCPOIDCConfiguration, error)

	Read(ctx context.Context, oidcID string) (*GCPOIDCConfiguration, error)

	Update(ctx context.Context, oidcID string, options GCPOIDCConfigurationUpdateOptions) (*GCPOIDCConfiguration, error)

	Delete(ctx context.Context, oidcID string) error
}

type gcpOIDCConfigurations struct {
	client *Client
}

var _ GCPOIDCConfigurations = &gcpOIDCConfigurations{}

type GCPOIDCConfiguration struct {
	ID                   string `jsonapi:"primary,gcp-oidc-configurations"`
	ServiceAccountEmail  string `jsonapi:"attr,service-account-email"`
	ProjectNumber        string `jsonapi:"attr,project-number"`
	WorkloadProviderName string `jsonapi:"attr,workload-provider-name"`

	Organization *Organization `jsonapi:"relation,organization"`
}

type GCPOIDCConfigurationCreateOptions struct {
	// Type is a public field utilized by JSON:API to
	// set the resource type via the field tag.
	// It is not a user-defined value and does not need to be set.
	// https://jsonapi.org/format/#crud-creating
	Type string `jsonapi:"primary,gcp-oidc-configurations"`

	// Attributes
	ServiceAccountEmail  string `jsonapi:"attr,service-account-email"`
	ProjectNumber        string `jsonapi:"attr,project-number"`
	WorkloadProviderName string `jsonapi:"attr,workload-provider-name"`
}

type GCPOIDCConfigurationUpdateOptions struct {
	// Type is a public field utilized by JSON:API to
	// set the resource type via the field tag.
	// It is not a user-defined value and does not need to be set.
	// https://jsonapi.org/format/#crud-creating
	Type string `jsonapi:"primary,gcp-oidc-configurations"`

	// Attributes
	ServiceAccountEmail  *string `jsonapi:"attr,service-account-email,omitempty"`
	ProjectNumber        *string `jsonapi:"attr,project-number,omitempty"`
	WorkloadProviderName *string `jsonapi:"attr,workload-provider-name,omitempty"`
}

func (o *GCPOIDCConfigurationCreateOptions) valid() error {
	if o.ServiceAccountEmail == "" {
		return ErrRequiredServiceAccountEmail
	}

	if o.ProjectNumber == "" {
		return ErrRequiredProjectNumber
	}

	if o.WorkloadProviderName == "" {
		return ErrRequiredWorkloadProviderName
	}

	return nil
}

func (goc *gcpOIDCConfigurations) Create(ctx context.Context, organization string, options GCPOIDCConfigurationCreateOptions) (*GCPOIDCConfiguration, error) {
	if !validStringID(&organization) {
		return nil, ErrInvalidOrg
	}

	if err := options.valid(); err != nil {
		return nil, err
	}

	req, err := goc.client.NewRequest("POST", fmt.Sprintf("organizations/%s/oidc-configurations", url.PathEscape(organization)), &options)
	if err != nil {
		return nil, err
	}

	gcpOIDCConfiguration := &GCPOIDCConfiguration{}
	err = req.Do(ctx, gcpOIDCConfiguration)
	if err != nil {
		return nil, err
	}

	return gcpOIDCConfiguration, nil
}

func (goc *gcpOIDCConfigurations) Read(ctx context.Context, oidcID string) (*GCPOIDCConfiguration, error) {
	req, err := goc.client.NewRequest("GET", fmt.Sprintf(OIDCConfigPathFormat, url.PathEscape(oidcID)), nil)
	if err != nil {
		return nil, err
	}

	gcpOIDCConfiguration := &GCPOIDCConfiguration{}
	err = req.Do(ctx, gcpOIDCConfiguration)
	if err != nil {
		return nil, err
	}

	return gcpOIDCConfiguration, nil
}

func (goc *gcpOIDCConfigurations) Update(ctx context.Context, oidcID string, options GCPOIDCConfigurationUpdateOptions) (*GCPOIDCConfiguration, error) {
	if !validStringID(&oidcID) {
		return nil, ErrInvalidOIDC
	}

	req, err := goc.client.NewRequest("PATCH", fmt.Sprintf(OIDCConfigPathFormat, url.PathEscape(oidcID)), &options)
	if err != nil {
		return nil, err
	}

	gcpOIDCConfiguration := &GCPOIDCConfiguration{}
	err = req.Do(ctx, gcpOIDCConfiguration)
	if err != nil {
		return nil, err
	}

	return gcpOIDCConfiguration, nil
}

func (goc *gcpOIDCConfigurations) Delete(ctx context.Context, oidcID string) error {
	if !validStringID(&oidcID) {
		return ErrInvalidOIDC
	}

	req, err := goc.client.NewRequest("DELETE", fmt.Sprintf(OIDCConfigPathFormat, url.PathEscape(oidcID)), nil)
	if err != nil {
		return err
	}

	return req.Do(ctx, nil)
}

// Compile-time proof of interface implementation.
var _ GHAInstallations = (*gHAInstallations)(nil)

// GHAInstallations describes all the GitHub App Installation related methods that the
// Terraform Enterprise API supports. The APIs require the user token for the user who
// already has the GitHub App Installation set up via the UI.
// (https://developer.hashicorp.com/terraform/enterprise/admin/application/github-app-integration)
type GHAInstallations interface {
	// List all the GitHub App Installations for the user.
	List(ctx context.Context, options *GHAInstallationListOptions) (*GHAInstallationList, error)

	// Read a GitHub App Installations by its external id.
	Read(ctx context.Context, GHAInstallationID string) (*GHAInstallation, error)
}

// gHAInstallations implements GHAInstallations.
type gHAInstallations struct {
	client *Client
}

// GHAInstallationList represents a list of github installations.
type GHAInstallationList struct {
	*Pagination
	Items []*GHAInstallation
}

// GHAInstallation represents a github app installation
type GHAInstallation struct {
	ID               *string `jsonapi:"primary,github-app-installations"`
	IconURL          *string `jsonapi:"attr,icon-url"`
	InstallationID   *int    `jsonapi:"attr,installation-id"`
	InstallationType *string `jsonapi:"attr,installation-type"`
	InstallationURL  *string `jsonapi:"attr,installation-url"`
	Name             *string `jsonapi:"attr,name"`
}

// GHAInstallationListOptions represents the options for listing.
type GHAInstallationListOptions struct {
	ListOptions
}

// List all the github app installations.
func (s *gHAInstallations) List(ctx context.Context, options *GHAInstallationListOptions) (*GHAInstallationList, error) {
	u := "github-app/installations"
	req, err := s.client.NewRequest("GET", u, options)
	if err != nil {
		return nil, err
	}

	ghil := &GHAInstallationList{}

	err = req.Do(ctx, ghil)
	if err != nil {
		return nil, err
	}

	return ghil, nil
}

// Read a GitHub App Installations by its ID.
func (s *gHAInstallations) Read(ctx context.Context, id string) (*GHAInstallation, error) {
	if !validStringID(&id) {
		return nil, ErrInvalidOauthClientID
	}

	u := fmt.Sprintf("github-app/installation/%s", url.PathEscape(id))
	req, err := s.client.NewRequest("GET", u, nil)
	if err != nil {
		return nil, err
	}

	ghi := &GHAInstallation{}
	err = req.Do(ctx, ghi)
	if err != nil {
		return nil, err
	}

	return ghi, err
}

// Compile-time proof of interface implementation
var _ GPGKeys = (*gpgKeys)(nil)

// GPGKeys describes all the GPG key related methods that the Terraform Private Registry API supports.
//
// TFE API Docs: https://developer.hashicorp.com/terraform/cloud-docs/api-docs/private-registry/gpg-keys
type GPGKeys interface {
	// Lists GPG keys in a private registry.
	ListPrivate(ctx context.Context, options GPGKeyListOptions) (*GPGKeyList, error)

	// Uploads a GPG Key to a private registry scoped with a namespace.
	Create(ctx context.Context, registryName RegistryName, options GPGKeyCreateOptions) (*GPGKey, error)

	// Read a GPG key.
	Read(ctx context.Context, keyID GPGKeyID) (*GPGKey, error)

	// Update a GPG key.
	Update(ctx context.Context, keyID GPGKeyID, options GPGKeyUpdateOptions) (*GPGKey, error)

	// Delete a GPG key.
	Delete(ctx context.Context, keyID GPGKeyID) error
}

// gpgKeys implements GPGKeys
type gpgKeys struct {
	client *Client
}

// GPGKeyList represents a list of GPG keys.
type GPGKeyList struct {
	*Pagination
	Items []*GPGKey
}

// GPGKey represents a signed GPG key for a HCP Terraform or Terraform Enterprise private provider.
type GPGKey struct {
	ID             string    `jsonapi:"primary,gpg-keys"`
	AsciiArmor     string    `jsonapi:"attr,ascii-armor"`
	CreatedAt      time.Time `jsonapi:"attr,created-at,iso8601"`
	KeyID          string    `jsonapi:"attr,key-id"`
	Namespace      string    `jsonapi:"attr,namespace"`
	Source         string    `jsonapi:"attr,source"`
	SourceURL      *string   `jsonapi:"attr,source-url"`
	TrustSignature string    `jsonapi:"attr,trust-signature"`
	UpdatedAt      time.Time `jsonapi:"attr,updated-at,iso8601"`
}

// GPGKeyID represents the set of identifiers used to fetch a GPG key.
type GPGKeyID struct {
	RegistryName RegistryName
	Namespace    string
	KeyID        string
}

// GPGKeyListOptions represents all the available options to list keys in a registry.
type GPGKeyListOptions struct {
	ListOptions

	// Required: A list of one or more namespaces. Must be authorized HCP Terraform or Terraform Enterprise organization names.
	Namespaces []string `url:"filter[namespace]"`
}

// GPGKeyCreateOptions represents all the available options used to create a GPG key.
type GPGKeyCreateOptions struct {
	Type       string `jsonapi:"primary,gpg-keys"`
	Namespace  string `jsonapi:"attr,namespace"`
	AsciiArmor string `jsonapi:"attr,ascii-armor"`
}

// GPGKeyCreateOptions represents all the available options used to update a GPG key.
type GPGKeyUpdateOptions struct {
	Type      string `jsonapi:"primary,gpg-keys"`
	Namespace string `jsonapi:"attr,namespace"`
}

// ListPrivate lists the private registry GPG keys for specified namespaces.
func (s *gpgKeys) ListPrivate(ctx context.Context, options GPGKeyListOptions) (*GPGKeyList, error) {
	if err := options.valid(); err != nil {
		return nil, err
	}

	u := fmt.Sprintf("/api/registry/%s/v2/gpg-keys", url.PathEscape(string(PrivateRegistry)))
	req, err := s.client.NewRequest("GET", u, &options)
	if err != nil {
		return nil, err
	}

	keyl := &GPGKeyList{}
	err = req.Do(ctx, keyl)
	if err != nil {
		return nil, err
	}

	return keyl, nil
}

func (s *gpgKeys) Create(ctx context.Context, registryName RegistryName, options GPGKeyCreateOptions) (*GPGKey, error) {
	if err := options.valid(); err != nil {
		return nil, err
	}

	if registryName != PrivateRegistry {
		return nil, ErrInvalidRegistryName
	}

	u := fmt.Sprintf("/api/registry/%s/v2/gpg-keys", url.PathEscape(string(registryName)))
	req, err := s.client.NewRequest("POST", u, &options)
	if err != nil {
		return nil, err
	}

	g := &GPGKey{}
	err = req.Do(ctx, g)
	if err != nil {
		return nil, err
	}

	return g, nil
}

func (s *gpgKeys) Read(ctx context.Context, keyID GPGKeyID) (*GPGKey, error) {
	if err := keyID.valid(); err != nil {
		return nil, err
	}

	u := fmt.Sprintf("/api/registry/%s/v2/gpg-keys/%s/%s",
		url.PathEscape(string(keyID.RegistryName)),
		url.PathEscape(keyID.Namespace),
		url.PathEscape(keyID.KeyID),
	)
	req, err := s.client.NewRequest("GET", u, nil)
	if err != nil {
		return nil, err
	}

	g := &GPGKey{}
	err = req.Do(ctx, g)
	if err != nil {
		return nil, err
	}

	return g, nil
}

func (s *gpgKeys) Update(ctx context.Context, keyID GPGKeyID, options GPGKeyUpdateOptions) (*GPGKey, error) {
	if err := options.valid(); err != nil {
		return nil, err
	}

	if err := keyID.valid(); err != nil {
		return nil, err
	}

	u := fmt.Sprintf("/api/registry/%s/v2/gpg-keys/%s/%s",
		url.PathEscape(string(keyID.RegistryName)),
		url.PathEscape(keyID.Namespace),
		url.PathEscape(keyID.KeyID),
	)
	req, err := s.client.NewRequest("PATCH", u, &options)
	if err != nil {
		return nil, err
	}

	g := &GPGKey{}
	err = req.Do(ctx, g)
	if err != nil {
		if strings.Contains(err.Error(), "namespace not authorized") {
			return nil, ErrNamespaceNotAuthorized
		}
		return nil, err
	}

	return g, nil
}

func (s *gpgKeys) Delete(ctx context.Context, keyID GPGKeyID) error {
	if err := keyID.valid(); err != nil {
		return err
	}

	u := fmt.Sprintf("/api/registry/%s/v2/gpg-keys/%s/%s",
		url.PathEscape(string(keyID.RegistryName)),
		url.PathEscape(keyID.Namespace),
		url.PathEscape(keyID.KeyID),
	)
	req, err := s.client.NewRequest("DELETE", u, nil)
	if err != nil {
		return err
	}

	return req.Do(ctx, nil)
}

func (o GPGKeyID) valid() error {
	if o.RegistryName != PrivateRegistry {
		return ErrInvalidRegistryName
	}

	if !validString(&o.Namespace) {
		return ErrInvalidNamespace
	}

	if !validString(&o.KeyID) {
		return ErrInvalidKeyID
	}

	return nil
}

func (o *GPGKeyListOptions) valid() error {
	if len(o.Namespaces) == 0 {
		return ErrInvalidNamespace
	}

	for _, namespace := range o.Namespaces {
		if namespace == "" || !validString(&namespace) {
			return ErrInvalidNamespace
		}
	}

	return nil
}

func (o GPGKeyCreateOptions) valid() error {
	if !validString(&o.Namespace) {
		return ErrInvalidNamespace
	}

	if !validString(&o.AsciiArmor) {
		return ErrInvalidAsciiArmor
	}

	return nil
}

func (o GPGKeyUpdateOptions) valid() error {
	if !validString(&o.Namespace) {
		return ErrInvalidNamespace
	}

	return nil
}

// Copyright IBM Corp. 2018, 2026
// SPDX-License-Identifier: MPL-2.0

// HYOKConfigurations describes all the HYOK configuration related methods that the HCP Terraform API supports.
// HCP Terraform API docs:
// https://developer.hashicorp.com/terraform/cloud-docs/api-docs/hold-your-own-key/configurations
type HYOKConfigurations interface {
	List(ctx context.Context, organization string, options *HYOKConfigurationsListOptions) (*HYOKConfigurationsList, error)

	Create(ctx context.Context, organization string, options HYOKConfigurationsCreateOptions) (*HYOKConfiguration, error)

	Read(ctx context.Context, hyokID string, options *HYOKConfigurationsReadOptions) (*HYOKConfiguration, error)

	Update(ctx context.Context, hyokID string, options HYOKConfigurationsUpdateOptions) (*HYOKConfiguration, error)

	Delete(ctx context.Context, hyokID string) error

	// Test checks the HYOK configuration and returns success if the configuration is valid.
	// It returns an error along with the error message if any issues are found.
	Test(ctx context.Context, hyokID string) error

	Revoke(ctx context.Context, hyokID string) error
}

type hyokConfigurations struct {
	client *Client
}

var _ HYOKConfigurations = &hyokConfigurations{}

type HYOKConfigurationStatus string

const (
	HYOKConfigurationUntested   HYOKConfigurationStatus = "untested"
	HYOKConfigurationTesting    HYOKConfigurationStatus = "testing"
	HYOKConfigurationTestFailed HYOKConfigurationStatus = "test_failed"
	HYOKConfigurationAvailable  HYOKConfigurationStatus = "available"
	HYOKConfigurationErrored    HYOKConfigurationStatus = "errored"
	HYOKConfigurationRevoking   HYOKConfigurationStatus = "revoking"
	HYOKConfigurationRevoked    HYOKConfigurationStatus = "revoked"
)

type OIDCConfigurationTypeChoice struct {
	AWSOIDCConfiguration   *AWSOIDCConfiguration
	GCPOIDCConfiguration   *GCPOIDCConfiguration
	AzureOIDCConfiguration *AzureOIDCConfiguration
	VaultOIDCConfiguration *VaultOIDCConfiguration
}

type KMSOptions struct {
	// AWS
	KeyRegion string `jsonapi:"attr,key-region,omitempty"`
	// GCP
	KeyLocation string `jsonapi:"attr,key-location,omitempty"`
	KeyRingID   string `jsonapi:"attr,key-ring-id,omitempty"`
}

type HYOKConfiguration struct {
	ID string `jsonapi:"primary,hyok-configurations"`

	// Attributes
	KEKID      string                  `jsonapi:"attr,kek-id"`
	KMSOptions *KMSOptions             `jsonapi:"attr,kms-options,omitempty"`
	Name       string                  `jsonapi:"attr,name"`
	Primary    bool                    `jsonapi:"attr,primary"`
	Status     HYOKConfigurationStatus `jsonapi:"attr,status"`
	Error      *string                 `jsonapi:"attr,error"`

	// Relationships
	Organization      *Organization                `jsonapi:"relation,organization"`
	OIDCConfiguration *OIDCConfigurationTypeChoice `jsonapi:"polyrelation,oidc-configuration"`
	AgentPool         *AgentPool                   `jsonapi:"relation,agent-pool"`
	KeyVersions       []*HYOKCustomerKeyVersion    `jsonapi:"relation,hyok-customer-key-versions"`
}

type HYOKConfigurationsList struct {
	*Pagination
	Items []*HYOKConfiguration
}

type HYOKConfigurationsIncludeOpt string

const (
	HYOKConfigurationsIncludeHYOKCustomerKeyVersions HYOKConfigurationsIncludeOpt = "hyok_customer_key_versions"
	HYOKConfigurationsIncludeOIDCConfiguration       HYOKConfigurationsIncludeOpt = "oidc_configuration"
)

type HYOKConfigurationsListOptions struct {
	ListOptions
	SearchQuery string                         `url:"q,omitempty"`
	Include     []HYOKConfigurationsIncludeOpt `url:"include,omitempty"`
}

type HYOKConfigurationsCreateOptions struct {
	// Type is a public field utilized by JSON:API to
	// set the resource type via the field tag.
	// It is not a user-defined value and does not need to be set.
	// https://jsonapi.org/format/#crud-creating
	Type string `jsonapi:"primary,hyok-configurations"`

	// Attributes
	KEKID      string      `jsonapi:"attr,kek-id"`
	KMSOptions *KMSOptions `jsonapi:"attr,kms-options"`
	Name       string      `jsonapi:"attr,name"`

	// Relationships
	OIDCConfiguration *OIDCConfigurationTypeChoice `jsonapi:"polyrelation,oidc-configuration"`
	AgentPool         *AgentPool                   `jsonapi:"relation,agent-pool"`
}

type HYOKConfigurationsReadOptions struct {
	Include []HYOKConfigurationsIncludeOpt `url:"include,omitempty"`
}

type HYOKConfigurationsUpdateOptions struct {
	// Type is a public field utilized by JSON:API to
	// set the resource type via the field tag.
	// It is not a user-defined value and does not need to be set.
	// https://jsonapi.org/format/#crud-creating
	Type string `jsonapi:"primary,hyok-configurations"`

	// Attributes
	KEKID      *string     `jsonapi:"attr,kek-id,omitempty"`
	KMSOptions *KMSOptions `jsonapi:"attr,kms-options,omitempty"`
	Name       *string     `jsonapi:"attr,name,omitempty"`
	Primary    *bool       `jsonapi:"attr,primary,omitempty"`

	// Relationships
	AgentPool *AgentPool `jsonapi:"relation,agent-pool,omitempty"`
}

func (h hyokConfigurations) List(ctx context.Context, organization string, options *HYOKConfigurationsListOptions) (*HYOKConfigurationsList, error) {
	if !validStringID(&organization) {
		return nil, ErrInvalidOrg
	}

	req, err := h.client.NewRequest("GET", fmt.Sprintf("organizations/%s/hyok-configurations", url.PathEscape(organization)), options)
	if err != nil {
		return nil, err
	}

	hyokConfigurationList := &HYOKConfigurationsList{}
	err = req.Do(ctx, hyokConfigurationList)
	if err != nil {
		return nil, err
	}

	return hyokConfigurationList, nil
}

func (h hyokConfigurations) Read(ctx context.Context, hyokID string, options *HYOKConfigurationsReadOptions) (*HYOKConfiguration, error) {
	if !validStringID(&hyokID) {
		return nil, ErrInvalidHYOK
	}

	req, err := h.client.NewRequest("GET", fmt.Sprintf("hyok-configurations/%s", url.PathEscape(hyokID)), options)
	if err != nil {
		return nil, err
	}

	hyokConfiguration := &HYOKConfiguration{}
	err = req.Do(ctx, hyokConfiguration)
	if err != nil {
		return nil, err
	}

	return hyokConfiguration, nil
}

func (h *HYOKConfigurationsCreateOptions) valid() error {
	if h.KEKID == "" {
		return ErrRequiredKEKID
	}

	if h.Name == "" {
		return ErrRequiredName
	}

	if h.OIDCConfiguration == nil {
		return ErrRequiredOIDCConfiguration
	}

	if h.AgentPool == nil {
		return ErrRequiredAgentPool
	}

	if h.OIDCConfiguration.AWSOIDCConfiguration != nil {
		if h.KMSOptions == nil {
			return ErrRequiredKMSOptions
		}

		if h.KMSOptions.KeyRegion == "" {
			return ErrRequiredKMSOptionsKeyRegion
		}
	}

	if h.OIDCConfiguration.GCPOIDCConfiguration != nil {
		if h.KMSOptions == nil {
			return ErrRequiredKMSOptions
		}

		if h.KMSOptions.KeyLocation == "" {
			return ErrRequiredKMSOptionsKeyLocation
		}

		if h.KMSOptions.KeyRingID == "" {
			return ErrRequiredKMSOptionsKeyRingID
		}
	}

	return nil
}

func (h hyokConfigurations) Create(ctx context.Context, organization string, options HYOKConfigurationsCreateOptions) (*HYOKConfiguration, error) {
	if !validStringID(&organization) {
		return nil, ErrInvalidOrg
	}

	if err := options.valid(); err != nil {
		return nil, err
	}

	req, err := h.client.NewRequest("POST", fmt.Sprintf("organizations/%s/hyok-configurations", url.PathEscape(organization)), &options)
	if err != nil {
		return nil, err
	}

	hyokConfiguration := &HYOKConfiguration{}
	err = req.Do(ctx, hyokConfiguration)
	if err != nil {
		return nil, err
	}

	return hyokConfiguration, nil
}

func (h hyokConfigurations) Update(ctx context.Context, hyokID string, options HYOKConfigurationsUpdateOptions) (*HYOKConfiguration, error) {
	if !validStringID(&hyokID) {
		return nil, ErrInvalidHYOK
	}

	req, err := h.client.NewRequest("PATCH", fmt.Sprintf("hyok-configurations/%s", url.PathEscape(hyokID)), &options)
	if err != nil {
		return nil, err
	}

	hyokConfiguration := &HYOKConfiguration{}
	err = req.Do(ctx, hyokConfiguration)
	if err != nil {
		return nil, err
	}

	return hyokConfiguration, nil
}

func (h hyokConfigurations) Delete(ctx context.Context, hyokID string) error {
	if !validStringID(&hyokID) {
		return ErrInvalidHYOK
	}

	req, err := h.client.NewRequest("DELETE", fmt.Sprintf("hyok-configurations/%s", url.PathEscape(hyokID)), nil)
	if err != nil {
		return err
	}

	return req.Do(ctx, nil)
}

func (h hyokConfigurations) Test(ctx context.Context, hyokID string) error {
	if !validStringID(&hyokID) {
		return ErrInvalidHYOK
	}

	req, err := h.client.NewRequest("POST", fmt.Sprintf("hyok-configurations/%s/actions/test", url.PathEscape(hyokID)), nil)
	if err != nil {
		return err
	}

	return req.Do(ctx, nil)
}

func (h hyokConfigurations) Revoke(ctx context.Context, hyokID string) error {
	if !validStringID(&hyokID) {
		return ErrInvalidHYOK
	}

	req, err := h.client.NewRequest("POST", fmt.Sprintf("hyok-configurations/%s/actions/revoke", url.PathEscape(hyokID)), nil)
	if err != nil {
		return err
	}

	return req.Do(ctx, nil)
}

// Copyright IBM Corp. 2018, 2026
// SPDX-License-Identifier: MPL-2.0

var _ HYOKCustomerKeyVersions = (*hyokCustomerKeyVersions)(nil)

// HYOKCustomerKeyVersions describes all the hyok customer key version related methods that the HCP Terraform API supports.
// HCP Terraform API docs:
// https://developer.hashicorp.com/terraform/cloud-docs/api-docs/hold-your-own-key/key-versions
type HYOKCustomerKeyVersions interface {
	// List all hyok customer key versions associated to a HYOK configuration.
	List(ctx context.Context, hyokConfigurationID string, options *HYOKCustomerKeyVersionListOptions) (*HYOKCustomerKeyVersionList, error)

	// Read a hyok customer key version by its ID.
	Read(ctx context.Context, hyokCustomerKeyVersionID string) (*HYOKCustomerKeyVersion, error)

	// Revoke a hyok customer key version.
	Revoke(ctx context.Context, hyokCustomerKeyVersionID string) error

	// Delete a hyok customer key version.
	Delete(ctx context.Context, hyokCustomerKeyVersionID string) error
}

// hyokCustomerKeyVersions implements HYOKCustomerKeyVersions
type hyokCustomerKeyVersions struct {
	client *Client
}

// HYOKCustomerKeyVersionList represents a list of hyok customer key versions
type HYOKCustomerKeyVersionList struct {
	*Pagination
	Items []*HYOKCustomerKeyVersion
}

// HYOKCustomerKeyVersion represents the resource
type HYOKCustomerKeyVersion struct {
	// Attributes
	ID                string               `jsonapi:"primary,hyok-customer-key-versions"`
	KeyVersion        string               `jsonapi:"attr,key-version"`
	CreatedAt         time.Time            `jsonapi:"attr,created-at,iso8601"`
	Status            HYOKKeyVersionStatus `jsonapi:"attr,status"`
	WorkspacesSecured int                  `jsonapi:"attr,workspaces-secured"`
	Error             string               `jsonapi:"attr,error"`

	// Relationships
	HYOKConfiguration *HYOKConfiguration `jsonapi:"relation,hyok-configuration"`
}

// HYOKKeyVersionStatus represents a key version status.
type HYOKKeyVersionStatus string

// List all available configuration version statuses.
const (
	KeyVersionStatusAvailable        HYOKKeyVersionStatus = "available"
	KeyVersionStatusRevoking         HYOKKeyVersionStatus = "revoking"
	KeyVersionStatusRevoked          HYOKKeyVersionStatus = "revoked"
	KeyVersionStatusRevocationFailed HYOKKeyVersionStatus = "revocation_failed"
)

// HYOKCustomerKeyVersionListOptions represents the options for listing hyok customer key versions
type HYOKCustomerKeyVersionListOptions struct {
	ListOptions
	Refresh bool `url:"refresh,omitempty"`
}

// List all hyok customer key versions.
func (s *hyokCustomerKeyVersions) List(ctx context.Context, hyokConfigurationID string, options *HYOKCustomerKeyVersionListOptions) (*HYOKCustomerKeyVersionList, error) {
	if !validStringID(&hyokConfigurationID) {
		return nil, ErrInvalidHYOK
	}

	path := fmt.Sprintf("hyok-configurations/%s/hyok-customer-key-versions", url.PathEscape(hyokConfigurationID))
	req, err := s.client.NewRequest("GET", path, options)
	if err != nil {
		return nil, err
	}

	kvs := &HYOKCustomerKeyVersionList{}
	err = req.Do(ctx, kvs)
	if err != nil {
		return nil, err
	}

	return kvs, nil
}

// Read a hyok customer key version by its ID.
func (s *hyokCustomerKeyVersions) Read(ctx context.Context, hyokCustomerKeyVersionID string) (*HYOKCustomerKeyVersion, error) {
	if !validStringID(&hyokCustomerKeyVersionID) {
		return nil, ErrInvalidHYOKCustomerKeyVersion
	}

	path := fmt.Sprintf("hyok-customer-key-versions/%s", url.PathEscape(hyokCustomerKeyVersionID))
	req, err := s.client.NewRequest("GET", path, nil)
	if err != nil {
		return nil, err
	}

	kv := &HYOKCustomerKeyVersion{}
	err = req.Do(ctx, kv)
	if err != nil {
		return nil, err
	}

	return kv, nil
}

// Revoke a hyok customer key version. This process is asynchronous.
// Returns `error` if there was a problem triggering the revocation. Otherwise revocation has been triggered successfully.
func (s *hyokCustomerKeyVersions) Revoke(ctx context.Context, hyokCustomerKeyVersionID string) error {
	if !validStringID(&hyokCustomerKeyVersionID) {
		return ErrInvalidHYOKCustomerKeyVersion
	}

	path := fmt.Sprintf("hyok-customer-key-versions/%s/actions/revoke", url.PathEscape(hyokCustomerKeyVersionID))
	req, err := s.client.NewRequest("POST", path, nil)
	if err != nil {
		return err
	}

	return req.Do(ctx, nil)
}

// Delete a hyok customer key version.
func (s *hyokCustomerKeyVersions) Delete(ctx context.Context, hyokCustomerKeyVersionID string) error {
	if !validStringID(&hyokCustomerKeyVersionID) {
		return ErrInvalidHYOKCustomerKeyVersion
	}

	path := fmt.Sprintf("hyok-customer-key-versions/%s", url.PathEscape(hyokCustomerKeyVersionID))
	req, err := s.client.NewRequest("DELETE", path, nil)
	if err != nil {
		return err
	}

	return req.Do(ctx, nil)
}

// Copyright IBM Corp. 2018, 2026
// SPDX-License-Identifier: MPL-2.0

var _ HYOKEncryptedDataKeys = (*hyokEncryptedDataKeys)(nil)

// HYOKEncryptedDataKeys describes all the hyok customer key version related methods that the HCP Terraform API supports.
// HCP Terraform API docs:
// https://developer.hashicorp.com/terraform/cloud-docs/api-docs/hold-your-own-key/encrypted-data-keys
type HYOKEncryptedDataKeys interface {
	// Read a HYOK encrypted data key by its ID.
	Read(ctx context.Context, hyokEncryptedDataKeyID string) (*HYOKEncryptedDataKey, error)
}

// hyokEncryptedDataKeys implements HYOKEncryptedDataKeys
type hyokEncryptedDataKeys struct {
	client *Client
}

// HYOKEncryptedDataKey represents the resource
type HYOKEncryptedDataKey struct {
	// Attributes
	ID              string    `jsonapi:"primary,hyok-encrypted-data-keys"`
	EncryptedDEK    string    `jsonapi:"attr,encrypted-dek"`
	CustomerKeyName string    `jsonapi:"attr,customer-key-name"`
	CreatedAt       time.Time `jsonapi:"attr,created-at,iso8601"`

	// Relationships
	KeyVersion *HYOKCustomerKeyVersion `jsonapi:"relation,hyok-customer-key-versions"`
}

// Read a HYOK encrypted data key by its ID.
func (h hyokEncryptedDataKeys) Read(ctx context.Context, hyokEncryptedDataKeyID string) (*HYOKEncryptedDataKey, error) {
	if !validStringID(&hyokEncryptedDataKeyID) {
		return nil, ErrInvalidHYOKEncryptedDataKey
	}

	path := fmt.Sprintf("hyok-encrypted-data-keys/%s", url.PathEscape(hyokEncryptedDataKeyID))
	req, err := h.client.NewRequest("GET", path, nil)
	if err != nil {
		return nil, err
	}

	dek := &HYOKEncryptedDataKey{}
	err = req.Do(ctx, dek)
	if err != nil {
		return nil, err
	}

	return dek, nil
}

// A private struct we need for unmarshalling
type internalRunTask struct {
	ID          string                 `jsonapi:"primary,tasks"`
	Name        string                 `jsonapi:"attr,name"`
	URL         string                 `jsonapi:"attr,url"`
	Description string                 `jsonapi:"attr,description"`
	Category    string                 `jsonapi:"attr,category"`
	HMACKey     *string                `jsonapi:"attr,hmac-key,omitempty"`
	Enabled     bool                   `jsonapi:"attr,enabled"`
	RawGlobal   map[string]interface{} `jsonapi:"attr,global-configuration,omitempty"`

	Organization      *Organization               `jsonapi:"relation,organization"`
	WorkspaceRunTasks []*internalWorkspaceRunTask `jsonapi:"relation,workspace-tasks"`
}

// Due to https://github.com/google/jsonapi/issues/74 we must first unmarshall using map[string]interface{}
// and then perform our own conversion from the map into a GlobalRunTask struct
func (irt internalRunTask) ToRunTask() *RunTask {
	obj := RunTask{
		ID:          irt.ID,
		Name:        irt.Name,
		URL:         irt.URL,
		Description: irt.Description,
		Category:    irt.Category,
		HMACKey:     irt.HMACKey,
		Enabled:     irt.Enabled,

		Organization: irt.Organization,
	}

	// Convert the WorkspaceRunTasks
	workspaceTasks := make([]*WorkspaceRunTask, len(irt.WorkspaceRunTasks))
	for idx, rawTask := range irt.WorkspaceRunTasks {
		if rawTask != nil {
			workspaceTasks[idx] = rawTask.ToWorkspaceRunTask()
		}
	}
	obj.WorkspaceRunTasks = workspaceTasks

	var boolVal bool
	// Check if the global configuration exists
	if val, ok := irt.RawGlobal["enabled"]; !ok {
		// The enabled property is required so we can assume now that the
		// global configuration was not supplied
		return &obj
	} else if boolVal, ok = val.(bool); !ok {
		// The enabled property exists but it is invalid (Couldn't cast to boolean)
		// so assume the global configuration was not supplied
		return &obj
	}

	obj.Global = &GlobalRunTask{
		Enabled: boolVal,
	}

	// Global Enforcement Level
	if val, ok := irt.RawGlobal["enforcement-level"]; ok {
		if stringVal, ok := val.(string); ok {
			obj.Global.EnforcementLevel = TaskEnforcementLevel(stringVal)
		}
	}

	// Global Stages
	if val, ok := irt.RawGlobal["stages"]; ok {
		if stringsVal, ok := val.([]interface{}); ok {
			obj.Global.Stages = make([]Stage, len(stringsVal))
			for idx, stageName := range stringsVal {
				if stringVal, ok := stageName.(string); ok {
					obj.Global.Stages[idx] = Stage(stringVal)
				}
			}
		}
	}

	return &obj
}

// A private struct we need for unmarshalling
type internalRunTaskList struct {
	*Pagination
	Items []*internalRunTask
}

// Due to https://github.com/google/jsonapi/issues/74 we must first unmarshall using
// the internal RunTask struct and convert that a RunTask
func (irt internalRunTaskList) ToRunTaskList() *RunTaskList {
	obj := RunTaskList{
		Pagination: irt.Pagination,
		Items:      make([]*RunTask, len(irt.Items)),
	}

	for idx, src := range irt.Items {
		if src != nil {
			obj.Items[idx] = src.ToRunTask()
		}
	}

	return &obj
}

// A private struct we need for unmarshalling
type internalWorkspaceRunTask struct {
	ID               string               `jsonapi:"primary,workspace-tasks"`
	EnforcementLevel TaskEnforcementLevel `jsonapi:"attr,enforcement-level"`
	Stage            Stage                `jsonapi:"attr,stage"`
	Stages           []string             `jsonapi:"attr,stages"`

	RunTask   *RunTask   `jsonapi:"relation,task"`
	Workspace *Workspace `jsonapi:"relation,workspace"`
}

// Due to https://github.com/google/jsonapi/issues/74 we must first unmarshall using map[string]interface{}
// and then perform our own conversion for the Stages
func (irt internalWorkspaceRunTask) ToWorkspaceRunTask() *WorkspaceRunTask {
	obj := WorkspaceRunTask{
		ID:               irt.ID,
		EnforcementLevel: irt.EnforcementLevel,
		Stage:            irt.Stage,
		Stages:           make([]Stage, len(irt.Stages)),
		RunTask:          irt.RunTask,
		Workspace:        irt.Workspace,
	}

	for idx, val := range irt.Stages {
		obj.Stages[idx] = Stage(val)
	}

	return &obj
}

// A private struct we need for unmarshalling
type internalWorkspaceRunTaskList struct {
	*Pagination
	Items []*internalWorkspaceRunTask
}

// Due to https://github.com/google/jsonapi/issues/74 we must first unmarshall using
// the internal WorkspaceRunTask struct and convert that a WorkspaceRunTask
func (irt internalWorkspaceRunTaskList) ToWorkspaceRunTaskList() *WorkspaceRunTaskList {
	obj := WorkspaceRunTaskList{
		Pagination: irt.Pagination,
		Items:      make([]*WorkspaceRunTask, len(irt.Items)),
	}

	for idx, src := range irt.Items {
		if src != nil {
			obj.Items[idx] = src.ToWorkspaceRunTask()
		}
	}

	return &obj
}

// Compile-time proof of interface implementation.
var _ IPRanges = (*ipRanges)(nil)

// IP Ranges provides a list of HCP Terraform or Terraform Enterprise's IP ranges.
//
// TFE API docs: https://developer.hashicorp.com/terraform/cloud-docs/api-docs/ip-ranges
type IPRanges interface {
	// Retrieve HCP Terraform IP ranges. If `modifiedSince` is not an empty string
	// then it will only return the IP ranges changes since that date.
	// The format for `modifiedSince` can be found here:
	// https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/If-Modified-Since
	Read(ctx context.Context, modifiedSince string) (*IPRange, error)
}

// ipRanges implements IPRanges interface.
type ipRanges struct {
	client *Client
}

// IPRange represents a list of HCP Terraform's IP ranges
type IPRange struct {
	// List of IP ranges in CIDR notation used for connections from user site to HCP Terraform APIs
	API []string `json:"api"`
	// List of IP ranges in CIDR notation used for notifications
	Notifications []string `json:"notifications"`
	// List of IP ranges in CIDR notation used for outbound requests from Sentinel policies
	Sentinel []string `json:"sentinel"`
	// List of IP ranges in CIDR notation used for connecting to VCS providers
	VCS []string `json:"vcs"`
}

// Read an IPRange that was not modified since the specified date.
func (i *ipRanges) Read(ctx context.Context, modifiedSince string) (*IPRange, error) {
	req, err := i.client.NewRequest("GET", "/api/meta/ip-ranges", nil)
	if err != nil {
		return nil, err
	}

	if modifiedSince != "" {
		req.Header.Add("If-Modified-Since", modifiedSince)
	}

	ir := &IPRange{}
	err = req.DoJSON(ctx, ir)
	if err != nil {
		return nil, err
	}

	return ir, nil
}

// Copyright IBM Corp. 2018, 2026
// SPDX-License-Identifier: MPL-2.0

// These helpers represent JSON:API relationship linkage data for custom request payloads.
// They are shared by endpoints that need plain `json` payload structs instead of
// the standard `jsonapi` request models.

type relationshipData struct {
	Data []relationshipItem `json:"data"`
}

type relationshipItem struct {
	Type string `json:"type"`
	ID   string `json:"id"`
}

func (p *Project) relationshipItem() relationshipItem {
	return relationshipItem{
		Type: "projects",
		ID:   p.ID,
	}
}

func (w *Workspace) relationshipItem() relationshipItem {
	return relationshipItem{
		Type: "workspaces",
		ID:   w.ID,
	}
}

// Copyright IBM Corp. 2018, 2026
// SPDX-License-Identifier: MPL-2.0

// LogReader implements io.Reader for streaming logs.
type LogReader struct {
	client      *Client
	ctx         context.Context
	done        func() (bool, error)
	logURL      *url.URL
	offset      int64
	reads       int
	startOfText bool
	endOfText   bool
}

func (r *LogReader) Read(l []byte) (int, error) {
	if written, err := r.read(l); !errors.Is(err, io.ErrNoProgress) {
		return written, err
	}

	// Loop until we can any data, the context is canceled or the
	// run is finsished. If we would return right away without any
	// data, we could end up causing a io.ErrNoProgress error.
	for r.reads = 1; ; r.reads++ {
		select {
		case <-r.ctx.Done():
			return 0, r.ctx.Err()
		case <-time.After(backoff(500, 2000, r.reads)):
			if written, err := r.read(l); !errors.Is(err, io.ErrNoProgress) {
				return written, err
			}
		}
	}
}

func (r *LogReader) read(l []byte) (int, error) {
	// Update the query string.
	r.logURL.RawQuery = fmt.Sprintf("limit=%d&offset=%d", len(l), r.offset)

	// Create a new request.
	req, err := http.NewRequest("GET", r.logURL.String(), nil)
	if err != nil {
		return 0, err
	}
	req = req.WithContext(r.ctx)

	// Attach the default headers.
	for k, v := range r.client.headers {
		req.Header[k] = v
	}

	// Retrieve the next chunk.
	resp, err := r.client.http.HTTPClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close() //nolint:errcheck

	// Basic response checking.
	if err := checkResponseCode(resp); err != nil {
		return 0, err
	}

	// Read the retrieved chunk.
	written, err := resp.Body.Read(l)
	if err != nil && !errors.Is(err, io.EOF) {
		// Ignore io.EOF errors returned when reading from the response
		// body as this indicates the end of the chunk and not the end
		// of the logfile.
		return written, err
	}

	if written > 0 {
		// Check for an STX (Start of Text) ASCII control marker.
		if !r.startOfText && l[0] == byte(2) {
			r.startOfText = true

			// Remove the STX marker from the received chunk.
			copy(l[:written-1], l[1:])
			l[written-1] = byte(0)
			r.offset++
			written--

			// Return early if we only received the STX marker.
			if written == 0 {
				return 0, io.ErrNoProgress
			}
		}

		// If we found an STX ASCII control character, start looking for
		// the ETX (End of Text) control character.
		if r.startOfText && l[written-1] == byte(3) {
			r.endOfText = true

			// Remove the ETX marker from the received chunk.
			l[written-1] = byte(0)
			r.offset++
			written--
		}
	}

	// Check if we need to continue the loop and wait 500 miliseconds
	// before checking if there is a new chunk available or that the
	// run is finished and we are done reading all chunks.
	if written != 0 {
		// Update the offset for the next read.
		r.offset += int64(written)
		return written, nil
	}

	if (r.startOfText && r.endOfText) || // The logstream finished without issues.
		(r.startOfText && r.reads%10 == 0) || // The logstream terminated unexpectedly.
		(!r.startOfText && r.reads > 1) { // The logstream doesn't support STX/ETX.
		done, err := r.done()
		if err != nil {
			return 0, err
		}
		if done {
			return 0, io.EOF
		}
	}
	return 0, io.ErrNoProgress
}

// backoff will perform exponential backoff based on the iteration and
// limited by the provided minimum and maximum (in milliseconds) durations.
func backoff(minimum, maximum float64, iter int) time.Duration {
	backoff := math.Pow(2, float64(iter)/5) * minimum
	if backoff > maximum {
		backoff = maximum
	}
	return time.Duration(backoff) * time.Millisecond
}

// Compile-time proof of interface implementation.
var _ NotificationConfigurations = (*notificationConfigurations)(nil)

// NotificationConfigurations describes all the Notification Configuration
// related methods that the Terraform Enterprise API supports.
//
// TFE API docs:
// https://developer.hashicorp.com/terraform/cloud-docs/api-docs/notification-configurations
type NotificationConfigurations interface {
	// List all the notification configurations within a workspace.
	List(ctx context.Context, subscribableID string, options *NotificationConfigurationListOptions) (*NotificationConfigurationList, error)

	// Create a new notification configuration with the given options.
	Create(ctx context.Context, subscribableID string, options NotificationConfigurationCreateOptions) (*NotificationConfiguration, error)

	// Read a notification configuration by its ID.
	Read(ctx context.Context, notificationConfigurationID string) (*NotificationConfiguration, error)

	// Update an existing notification configuration.
	Update(ctx context.Context, notificationConfigurationID string, options NotificationConfigurationUpdateOptions) (*NotificationConfiguration, error)

	// Delete a notification configuration by its ID.
	Delete(ctx context.Context, notificationConfigurationID string) error

	// Verify a notification configuration by its ID.
	Verify(ctx context.Context, notificationConfigurationID string) (*NotificationConfiguration, error)
}

// notificationConfigurations implements NotificationConfigurations.
type notificationConfigurations struct {
	client *Client
}

// NotificationTriggerType represents the different TFE notifications that can be sent
// as a run's progress transitions between different states
type NotificationTriggerType string

const (
	NotificationTriggerCreated                        NotificationTriggerType = "run:created"
	NotificationTriggerPlanning                       NotificationTriggerType = "run:planning"
	NotificationTriggerNeedsAttention                 NotificationTriggerType = "run:needs_attention"
	NotificationTriggerApplying                       NotificationTriggerType = "run:applying"
	NotificationTriggerCompleted                      NotificationTriggerType = "run:completed"
	NotificationTriggerErrored                        NotificationTriggerType = "run:errored"
	NotificationTriggerAssessmentDrifted              NotificationTriggerType = "assessment:drifted"
	NotificationTriggerAssessmentFailed               NotificationTriggerType = "assessment:failed"
	NotificationTriggerAssessmentCheckFailed          NotificationTriggerType = "assessment:check_failure"
	NotificationTriggerWorkspaceAutoDestroyReminder   NotificationTriggerType = "workspace:auto_destroy_reminder"
	NotificationTriggerWorkspaceAutoDestroyRunResults NotificationTriggerType = "workspace:auto_destroy_run_results"
	NotificationTriggerChangeRequestCreated           NotificationTriggerType = "change_request:created"
)

// NotificationDestinationType represents the destination type of the
// notification configuration.
type NotificationDestinationType string

// List of available notification destination types.
const (
	NotificationDestinationTypeEmail          NotificationDestinationType = "email"
	NotificationDestinationTypeGeneric        NotificationDestinationType = "generic"
	NotificationDestinationTypeSlack          NotificationDestinationType = "slack"
	NotificationDestinationTypeMicrosoftTeams NotificationDestinationType = "microsoft-teams"
)

// NotificationConfigurationList represents a list of Notification
// Configurations.
type NotificationConfigurationList struct {
	*Pagination
	Items []*NotificationConfiguration
}

// NotificationConfigurationSubscribableChoice is a choice type struct that represents the possible values
// within a polymorphic relation. If a value is available, exactly one field
// will be non-nil.
type NotificationConfigurationSubscribableChoice struct {
	Project   *Project
	Team      *Team
	Workspace *Workspace
}

// NotificationConfiguration represents a Notification Configuration.
type NotificationConfiguration struct {
	ID                string                      `jsonapi:"primary,notification-configurations"`
	CreatedAt         time.Time                   `jsonapi:"attr,created-at,iso8601"`
	DeliveryResponses []*DeliveryResponse         `jsonapi:"attr,delivery-responses"`
	DestinationType   NotificationDestinationType `jsonapi:"attr,destination-type"`
	Enabled           bool                        `jsonapi:"attr,enabled"`
	Name              string                      `jsonapi:"attr,name"`
	Token             string                      `jsonapi:"attr,token"`
	Triggers          []string                    `jsonapi:"attr,triggers"`
	UpdatedAt         time.Time                   `jsonapi:"attr,updated-at,iso8601"`
	URL               string                      `jsonapi:"attr,url"`

	// EmailAddresses is only available for TFE users. It is not available in HCP Terraform.
	EmailAddresses []string `jsonapi:"attr,email-addresses"`

	// Relations
	// DEPRECATED. The subscribable field is polymorphic. Use NotificationConfigurationSubscribableChoice instead.
	Subscribable       *Workspace                                   `jsonapi:"relation,subscribable,omitempty"`
	SubscribableChoice *NotificationConfigurationSubscribableChoice `jsonapi:"polyrelation,subscribable"`

	EmailUsers []*User `jsonapi:"relation,users"`
}

// DeliveryResponse represents a notification configuration delivery response.
type DeliveryResponse struct {
	Body       string              `jsonapi:"attr,body"`
	Code       string              `jsonapi:"attr,code"`
	Headers    map[string][]string `jsonapi:"attr,headers"`
	SentAt     time.Time           `jsonapi:"attr,sent-at,rfc3339"`
	Successful string              `jsonapi:"attr,successful"`
	URL        string              `jsonapi:"attr,url"`
}

// NotificationConfigurationListOptions represents the options for listing
// notification configurations.
type NotificationConfigurationListOptions struct {
	ListOptions

	SubscribableChoice *NotificationConfigurationSubscribableChoice
}

// NotificationConfigurationCreateOptions represents the options for
// creating a new notification configuration.
type NotificationConfigurationCreateOptions struct {
	// Type is a public field utilized by JSON:API to
	// set the resource type via the field tag.
	// It is not a user-defined value and does not need to be set.
	// https://jsonapi.org/format/#crud-creating
	Type string `jsonapi:"primary,notification-configurations"`

	// Required: The destination type of the notification configuration
	DestinationType *NotificationDestinationType `jsonapi:"attr,destination-type"`

	// Required: Whether the notification configuration should be enabled or not
	Enabled *bool `jsonapi:"attr,enabled"`

	// Required: The name of the notification configuration
	Name *string `jsonapi:"attr,name"`

	// Optional: The token of the notification configuration
	Token *string `jsonapi:"attr,token,omitempty"`

	// Optional: The list of run events that will trigger notifications.
	Triggers []NotificationTriggerType `jsonapi:"attr,triggers,omitempty"`

	// Optional: The url of the notification configuration
	URL *string `jsonapi:"attr,url,omitempty"`

	// Optional: The list of email addresses that will receive notification emails.
	// EmailAddresses is only available for TFE users. It is not available in HCP Terraform.
	EmailAddresses []string `jsonapi:"attr,email-addresses,omitempty"`

	// Optional: The list of users belonging to the organization that will receive notification emails.
	EmailUsers []*User `jsonapi:"relation,users,omitempty"`

	// Required: The workspace, team, or project that the notification configuration is associated with.
	SubscribableChoice *NotificationConfigurationSubscribableChoice `jsonapi:"polyrelation,subscribable,omitempty"`
}

// NotificationConfigurationUpdateOptions represents the options for
// updating a existing notification configuration.
type NotificationConfigurationUpdateOptions struct {
	// Type is a public field utilized by JSON:API to
	// set the resource type via the field tag.
	// It is not a user-defined value and does not need to be set.
	// https://jsonapi.org/format/#crud-creating
	Type string `jsonapi:"primary,notification-configurations"`

	// Optional: Whether the notification configuration should be enabled or not
	Enabled *bool `jsonapi:"attr,enabled,omitempty"`

	// Optional: The name of the notification configuration
	Name *string `jsonapi:"attr,name,omitempty"`

	// Optional: The token of the notification configuration
	Token *string `jsonapi:"attr,token,omitempty"`

	// Optional: The list of run events that will trigger notifications.
	Triggers []NotificationTriggerType `jsonapi:"attr,triggers,omitempty"`

	// Optional: The url of the notification configuration
	URL *string `jsonapi:"attr,url,omitempty"`

	// Optional: The list of email addresses that will receive notification emails.
	// EmailAddresses is only available for TFE users. It is not available in HCP Terraform.
	EmailAddresses []string `jsonapi:"attr,email-addresses,omitempty"`

	// Optional: The list of users belonging to the organization that will receive notification emails.
	EmailUsers []*User `jsonapi:"relation,users,omitempty"`
}

// List all the notification configurations associated with a workspace.
func (s *notificationConfigurations) List(ctx context.Context, subscribableID string, options *NotificationConfigurationListOptions) (*NotificationConfigurationList, error) {
	if options == nil {
		options = &NotificationConfigurationListOptions{
			SubscribableChoice: &NotificationConfigurationSubscribableChoice{
				Workspace: &Workspace{ID: subscribableID},
			},
		}
	} else if options.SubscribableChoice == nil {
		options.SubscribableChoice = &NotificationConfigurationSubscribableChoice{
			Workspace: &Workspace{ID: subscribableID},
		}
	}

	u, err := notificationSubscribableURL(subscribableID, options.SubscribableChoice)
	if err != nil {
		return nil, err
	}

	req, err := s.client.NewRequest("GET", u, options)
	if err != nil {
		return nil, err
	}

	ncl := &NotificationConfigurationList{}
	err = req.Do(ctx, ncl)
	if err != nil {
		return nil, err
	}

	for i := range ncl.Items {
		backfillDeprecatedSubscribable(ncl.Items[i])
	}

	return ncl, nil
}

// Create a notification configuration with the given options.
func (s *notificationConfigurations) Create(ctx context.Context, subscribableID string, options NotificationConfigurationCreateOptions) (*NotificationConfiguration, error) {
	if options.SubscribableChoice != nil && options.SubscribableChoice.Team != nil {
		options.SubscribableChoice = &NotificationConfigurationSubscribableChoice{Team: &Team{ID: subscribableID}}
	} else if options.SubscribableChoice != nil && options.SubscribableChoice.Project != nil {
		options.SubscribableChoice = &NotificationConfigurationSubscribableChoice{Project: &Project{ID: subscribableID}}
	} else {
		options.SubscribableChoice = &NotificationConfigurationSubscribableChoice{Workspace: &Workspace{ID: subscribableID}}
	}

	if err := options.valid(); err != nil {
		return nil, err
	}

	u, err := notificationSubscribableURL(subscribableID, options.SubscribableChoice)
	if err != nil {
		return nil, err
	}

	req, err := s.client.NewRequest("POST", u, &options)
	if err != nil {
		return nil, err
	}

	nc := &NotificationConfiguration{}
	err = req.Do(ctx, nc)

	if err != nil {
		return nil, err
	}

	backfillDeprecatedSubscribable(nc)

	return nc, nil
}

// Read a notification configuration by its ID.
func (s *notificationConfigurations) Read(ctx context.Context, notificationConfigurationID string) (*NotificationConfiguration, error) {
	if !validStringID(&notificationConfigurationID) {
		return nil, ErrInvalidNotificationConfigID
	}

	u := fmt.Sprintf("notification-configurations/%s", url.PathEscape(notificationConfigurationID))
	req, err := s.client.NewRequest("GET", u, nil)
	if err != nil {
		return nil, err
	}

	nc := &NotificationConfiguration{}
	err = req.Do(ctx, nc)
	if err != nil {
		return nil, err
	}

	backfillDeprecatedSubscribable(nc)

	return nc, nil
}

// Updates a notification configuration with the given options.
func (s *notificationConfigurations) Update(ctx context.Context, notificationConfigurationID string, options NotificationConfigurationUpdateOptions) (*NotificationConfiguration, error) {
	if !validStringID(&notificationConfigurationID) {
		return nil, ErrInvalidNotificationConfigID
	}

	if err := options.valid(); err != nil {
		return nil, err
	}

	u := fmt.Sprintf("notification-configurations/%s", url.PathEscape(notificationConfigurationID))
	req, err := s.client.NewRequest("PATCH", u, &options)
	if err != nil {
		return nil, err
	}

	nc := &NotificationConfiguration{}
	err = req.Do(ctx, nc)
	if err != nil {
		return nil, err
	}

	backfillDeprecatedSubscribable(nc)

	return nc, nil
}

// Delete a notifications configuration by its ID.
func (s *notificationConfigurations) Delete(ctx context.Context, notificationConfigurationID string) error {
	if !validStringID(&notificationConfigurationID) {
		return ErrInvalidNotificationConfigID
	}

	u := fmt.Sprintf("notification-configurations/%s", url.PathEscape(notificationConfigurationID))
	req, err := s.client.NewRequest("DELETE", u, nil)
	if err != nil {
		return err
	}

	return req.Do(ctx, nil)
}

// Verify a notification configuration by delivering a verification
// payload to the configured url.
func (s *notificationConfigurations) Verify(ctx context.Context, notificationConfigurationID string) (*NotificationConfiguration, error) {
	if !validStringID(&notificationConfigurationID) {
		return nil, ErrInvalidNotificationConfigID
	}

	u := fmt.Sprintf(
		"notification-configurations/%s/actions/verify", url.PathEscape(notificationConfigurationID))
	req, err := s.client.NewRequest("POST", u, nil)
	if err != nil {
		return nil, err
	}

	nc := &NotificationConfiguration{}
	err = req.Do(ctx, nc)
	if err != nil {
		return nil, err
	}

	return nc, nil
}

func (o NotificationConfigurationCreateOptions) valid() error {
	if err := validateSubscribableChoice(o.SubscribableChoice); err != nil {
		return err
	}

	if o.DestinationType == nil {
		return ErrRequiredDestinationType
	}
	if o.Enabled == nil {
		return ErrRequiredEnabled
	}
	if !validString(o.Name) {
		return ErrRequiredName
	}

	if !validNotificationTriggerType(o.Triggers) {
		return ErrInvalidNotificationTrigger
	}

	if *o.DestinationType == NotificationDestinationTypeGeneric ||
		*o.DestinationType == NotificationDestinationTypeSlack ||
		*o.DestinationType == NotificationDestinationTypeMicrosoftTeams {
		if o.URL == nil {
			return ErrRequiredURL
		}
	}
	return nil
}

func (o NotificationConfigurationUpdateOptions) valid() error {
	if o.Name != nil && !validString(o.Name) {
		return ErrRequiredName
	}

	if !validNotificationTriggerType(o.Triggers) {
		return ErrInvalidNotificationTrigger
	}

	return nil
}

func backfillDeprecatedSubscribable(notification *NotificationConfiguration) {
	if notification.Subscribable != nil || notification.SubscribableChoice == nil {
		return
	}

	if notification.SubscribableChoice.Workspace != nil {
		notification.Subscribable = notification.SubscribableChoice.Workspace
	}
}

func notificationSubscribableURL(subscribableID string, choice *NotificationConfigurationSubscribableChoice) (string, error) {
	if choice != nil && choice.Team != nil {
		if !validStringID(&subscribableID) {
			return "", ErrInvalidTeamID
		}
		return fmt.Sprintf("teams/%s/notification-configurations", url.PathEscape(subscribableID)), nil
	}
	if choice != nil && choice.Project != nil {
		if !validStringID(&subscribableID) {
			return "", ErrInvalidProjectID
		}
		return fmt.Sprintf("projects/%s/notification-configurations", url.PathEscape(subscribableID)), nil
	}
	if choice == nil || !validStringID(&subscribableID) {
		return "", ErrInvalidWorkspaceID
	}
	return fmt.Sprintf("workspaces/%s/notification-configurations", url.PathEscape(subscribableID)), nil
}

func validateSubscribableChoice(choice *NotificationConfigurationSubscribableChoice) error {
	if choice != nil && choice.Team != nil {
		if !validStringID(&choice.Team.ID) {
			return ErrInvalidTeamID
		}
		return nil
	}
	if choice != nil && choice.Project != nil {
		if !validStringID(&choice.Project.ID) {
			return ErrInvalidProjectID
		}
		return nil
	}
	if choice == nil || !validStringID(&choice.Workspace.ID) {
		return ErrInvalidWorkspaceID
	}
	return nil
}

func validNotificationTriggerType(triggers []NotificationTriggerType) bool {
	for _, t := range triggers {
		switch t {
		case NotificationTriggerApplying,
			NotificationTriggerNeedsAttention,
			NotificationTriggerCompleted,
			NotificationTriggerCreated,
			NotificationTriggerErrored,
			NotificationTriggerPlanning,
			NotificationTriggerAssessmentDrifted,
			NotificationTriggerAssessmentFailed,
			NotificationTriggerWorkspaceAutoDestroyReminder,
			NotificationTriggerWorkspaceAutoDestroyRunResults,
			NotificationTriggerChangeRequestCreated,
			NotificationTriggerAssessmentCheckFailed:
			continue
		default:
			return false
		}
	}

	return true
}

// Compile-time proof of interface implementation.
var _ OAuthClients = (*oAuthClients)(nil)

// OAuthClients describes all the OAuth client related methods that the
// Terraform Enterprise API supports.
//
// TFE API docs:
// https://developer.hashicorp.com/terraform/cloud-docs/api-docs/oauth-clients
type OAuthClients interface {
	// List all the OAuth clients for a given organization.
	List(ctx context.Context, organization string, options *OAuthClientListOptions) (*OAuthClientList, error)

	// Create an OAuth client to connect an organization and a VCS provider.
	Create(ctx context.Context, organization string, options OAuthClientCreateOptions) (*OAuthClient, error)

	// Read an OAuth client by its ID.
	Read(ctx context.Context, oAuthClientID string) (*OAuthClient, error)

	// ReadWithOptions reads an oauth client by its ID using the options supplied.
	ReadWithOptions(ctx context.Context, oAuthClientID string, options *OAuthClientReadOptions) (*OAuthClient, error)

	// Update an existing OAuth client by its ID.
	Update(ctx context.Context, oAuthClientID string, options OAuthClientUpdateOptions) (*OAuthClient, error)

	// Delete an OAuth client by its ID.
	Delete(ctx context.Context, oAuthClientID string) error

	// AddProjects add projects to an oauth client.
	AddProjects(ctx context.Context, oAuthClientID string, options OAuthClientAddProjectsOptions) error

	// RemoveProjects remove projects from an oauth client.
	RemoveProjects(ctx context.Context, oAuthClientID string, options OAuthClientRemoveProjectsOptions) error
}

// oAuthClients implements OAuthClients.
type oAuthClients struct {
	client *Client
}

// ServiceProviderType represents a VCS type.
type ServiceProviderType string

// List of available VCS types.
const (
	ServiceProviderAzureDevOpsServer   ServiceProviderType = "ado_server"
	ServiceProviderAzureDevOpsServices ServiceProviderType = "ado_services"
	ServiceProviderBitbucketDataCenter ServiceProviderType = "bitbucket_data_center"
	ServiceProviderBitbucket           ServiceProviderType = "bitbucket_hosted"
	// Bitbucket Server v5.4.0 and above
	ServiceProviderBitbucketServer ServiceProviderType = "bitbucket_server"
	// Bitbucket Server v5.3.0 and below
	ServiceProviderBitbucketServerLegacy ServiceProviderType = "bitbucket_server_legacy"
	ServiceProviderGithub                ServiceProviderType = "github"
	ServiceProviderGithubEE              ServiceProviderType = "github_enterprise"
	ServiceProviderGitlab                ServiceProviderType = "gitlab_hosted"
	ServiceProviderGitlabCE              ServiceProviderType = "gitlab_community_edition"
	ServiceProviderGitlabEE              ServiceProviderType = "gitlab_enterprise_edition"
)

// OAuthClientList represents a list of OAuth clients.
type OAuthClientList struct {
	*Pagination
	Items []*OAuthClient
}

// OAuthClient represents a connection between an organization and a VCS
// provider.
type OAuthClient struct {
	ID                  string              `jsonapi:"primary,oauth-clients"`
	APIURL              string              `jsonapi:"attr,api-url"`
	CallbackURL         string              `jsonapi:"attr,callback-url"`
	ConnectPath         string              `jsonapi:"attr,connect-path"`
	CreatedAt           time.Time           `jsonapi:"attr,created-at,iso8601"`
	HTTPURL             string              `jsonapi:"attr,http-url"`
	Key                 string              `jsonapi:"attr,key"`
	RSAPublicKey        string              `jsonapi:"attr,rsa-public-key"`
	Name                *string             `jsonapi:"attr,name"`
	Secret              string              `jsonapi:"attr,secret"`
	ServiceProvider     ServiceProviderType `jsonapi:"attr,service-provider"`
	ServiceProviderName string              `jsonapi:"attr,service-provider-display-name"`
	OrganizationScoped  *bool               `jsonapi:"attr,organization-scoped"`

	// Relations
	Organization *Organization `jsonapi:"relation,organization"`
	OAuthTokens  []*OAuthToken `jsonapi:"relation,oauth-tokens"`
	AgentPool    *AgentPool    `jsonapi:"relation,agent-pool"`
	// The projects to which the oauth client applies.
	Projects []*Project `jsonapi:"relation,projects"`
}

// A list of relations to include
type OAuthClientIncludeOpt string

const (
	OauthClientOauthTokens OAuthClientIncludeOpt = "oauth_tokens"
	OauthClientProjects    OAuthClientIncludeOpt = "projects"
)

// OAuthClientListOptions represents the options for listing
// OAuth clients.
type OAuthClientListOptions struct {
	ListOptions

	Include []OAuthClientIncludeOpt `url:"include,omitempty"`
}

// OAuthClientReadOptions are read options.
// For a full list of relations, please see:
// https://developer.hashicorp.com/terraform/cloud-docs/api-docs/oauth-clients#relationships
type OAuthClientReadOptions struct {
	// Optional: A list of relations to include. See available resources
	// https://developer.hashicorp.com/terraform/cloud-docs/api-docs/oauth-clients#available-related-resources
	Include []OAuthClientIncludeOpt `url:"include,omitempty"`
}

// OAuthClientCreateOptions represents the options for creating an OAuth client.
type OAuthClientCreateOptions struct {
	// Type is a public field utilized by JSON:API to
	// set the resource type via the field tag.
	// It is not a user-defined value and does not need to be set.
	// https://jsonapi.org/format/#crud-creating
	Type string `jsonapi:"primary,oauth-clients"`

	// A display name for the OAuth Client.
	Name *string `jsonapi:"attr,name"`

	// Required: The base URL of your VCS provider's API.
	APIURL *string `jsonapi:"attr,api-url"`

	// Required: The homepage of your VCS provider.
	HTTPURL *string `jsonapi:"attr,http-url"`

	// Optional: The OAuth Client key.
	Key *string `jsonapi:"attr,key,omitempty"`

	// Optional: The token string you were given by your VCS provider.
	OAuthToken *string `jsonapi:"attr,oauth-token-string,omitempty"`

	// Optional: The initial list of projects for which the oauth client should be associated with.
	Projects []*Project `jsonapi:"relation,projects,omitempty"`

	// Optional: Private key associated with this vcs provider - only available for ado_server
	PrivateKey *string `jsonapi:"attr,private-key,omitempty"`

	// Optional: Secret key associated with this vcs provider - only available for ado_server
	Secret *string `jsonapi:"attr,secret,omitempty"`

	// Optional: RSAPublicKey the text of the SSH public key associated with your
	// BitBucket Data Center Application Link.
	RSAPublicKey *string `jsonapi:"attr,rsa-public-key,omitempty"`

	// Required: The VCS provider being connected with.
	ServiceProvider *ServiceProviderType `jsonapi:"attr,service-provider"`

	// Optional: AgentPool to associate the VCS Provider with, for PrivateVCS support
	AgentPool *AgentPool `jsonapi:"relation,agent-pool,omitempty"`

	// Optional: Whether the OAuthClient is available to all workspaces in the organization.
	// True if the oauth client is organization scoped, false otherwise.
	OrganizationScoped *bool `jsonapi:"attr,organization-scoped,omitempty"`
}

// OAuthClientUpdateOptions represents the options for updating an OAuth client.
type OAuthClientUpdateOptions struct {
	// Type is a public field utilized by JSON:API to
	// set the resource type via the field tag.
	// It is not a user-defined value and does not need to be set.
	// https://jsonapi.org/format/#crud-creating
	Type string `jsonapi:"primary,oauth-clients"`

	// Optional: A display name for the OAuth Client.
	Name *string `jsonapi:"attr,name,omitempty"`

	// Optional: The OAuth Client key.
	Key *string `jsonapi:"attr,key,omitempty"`

	// Optional: Secret key associated with this vcs provider - only available for ado_server
	Secret *string `jsonapi:"attr,secret,omitempty"`

	// Optional: RSAPublicKey the text of the SSH public key associated with your BitBucket
	// Server Application Link.
	RSAPublicKey *string `jsonapi:"attr,rsa-public-key,omitempty"`

	// Optional: The token string you were given by your VCS provider.
	OAuthToken *string `jsonapi:"attr,oauth-token-string,omitempty"`

	// Optional: AgentPool to associate the VCS Provider with, for PrivateVCS support
	AgentPool *AgentPool `jsonapi:"relation,agent-pool,omitempty"`

	// Optional: Whether the OAuthClient is available to all workspaces in the organization.
	// True if the oauth client is organization scoped, false otherwise.
	OrganizationScoped *bool `jsonapi:"attr,organization-scoped,omitempty"`
}

// OAuthClientAddProjectsOptions represents the options for adding projects
// to an oauth client.
type OAuthClientAddProjectsOptions struct {
	// The projects to add to an oauth client.
	Projects []*Project
}

// OAuthClientRemoveProjectsOptions represents the options for removing
// projects from an oauth client.
type OAuthClientRemoveProjectsOptions struct {
	// The projects to remove from an oauth client.
	Projects []*Project
}

// List all the OAuth clients for a given organization.
func (s *oAuthClients) List(ctx context.Context, organization string, options *OAuthClientListOptions) (*OAuthClientList, error) {
	if !validStringID(&organization) {
		return nil, ErrInvalidOrg
	}
	if err := options.valid(); err != nil {
		return nil, err
	}

	u := fmt.Sprintf("organizations/%s/oauth-clients", url.PathEscape(organization))
	req, err := s.client.NewRequest("GET", u, options)
	if err != nil {
		return nil, err
	}

	ocl := &OAuthClientList{}
	err = req.Do(ctx, ocl)
	if err != nil {
		return nil, err
	}

	return ocl, nil
}

// Create an OAuth client to connect an organization and a VCS provider.
func (s *oAuthClients) Create(ctx context.Context, organization string, options OAuthClientCreateOptions) (*OAuthClient, error) {
	if !validStringID(&organization) {
		return nil, ErrInvalidOrg
	}
	if err := options.valid(); err != nil {
		return nil, err
	}

	u := fmt.Sprintf("organizations/%s/oauth-clients", url.PathEscape(organization))
	req, err := s.client.NewRequest("POST", u, &options)
	if err != nil {
		return nil, err
	}

	oc := &OAuthClient{}
	err = req.Do(ctx, oc)
	if err != nil {
		return nil, err
	}

	return oc, nil
}

// Read an OAuth client by its ID.
func (s *oAuthClients) Read(ctx context.Context, oAuthClientID string) (*OAuthClient, error) {
	return s.ReadWithOptions(ctx, oAuthClientID, nil)
}

func (s *oAuthClients) ReadWithOptions(ctx context.Context, oAuthClientID string, options *OAuthClientReadOptions) (*OAuthClient, error) {
	if !validStringID(&oAuthClientID) {
		return nil, ErrInvalidOauthClientID
	}
	if err := options.valid(); err != nil {
		return nil, err
	}

	u := fmt.Sprintf("oauth-clients/%s", url.PathEscape(oAuthClientID))
	req, err := s.client.NewRequest("GET", u, options)
	if err != nil {
		return nil, err
	}

	oc := &OAuthClient{}
	err = req.Do(ctx, oc)
	if err != nil {
		return nil, err
	}

	return oc, err
}

// Update an OAuth client by its ID.
func (s *oAuthClients) Update(ctx context.Context, oAuthClientID string, options OAuthClientUpdateOptions) (*OAuthClient, error) {
	if !validStringID(&oAuthClientID) {
		return nil, ErrInvalidOauthClientID
	}

	u := fmt.Sprintf("oauth-clients/%s", url.PathEscape(oAuthClientID))
	req, err := s.client.NewRequest("PATCH", u, &options)
	if err != nil {
		return nil, err
	}

	oc := &OAuthClient{}
	err = req.Do(ctx, oc)
	if err != nil {
		return nil, err
	}

	return oc, err
}

// Delete an OAuth client by its ID.
func (s *oAuthClients) Delete(ctx context.Context, oAuthClientID string) error {
	if !validStringID(&oAuthClientID) {
		return ErrInvalidOauthClientID
	}

	u := fmt.Sprintf("oauth-clients/%s", url.PathEscape(oAuthClientID))
	req, err := s.client.NewRequest("DELETE", u, nil)
	if err != nil {
		return err
	}

	return req.Do(ctx, nil)
}

func (o OAuthClientCreateOptions) valid() error {
	if !validString(o.APIURL) {
		return ErrRequiredAPIURL
	}
	if !validString(o.HTTPURL) {
		return ErrRequiredHTTPURL
	}
	if o.ServiceProvider == nil {
		return ErrRequiredServiceProvider
	}
	if !validString(o.OAuthToken) &&
		*o.ServiceProvider != *ServiceProvider(ServiceProviderBitbucketServer) &&
		*o.ServiceProvider != *ServiceProvider(ServiceProviderBitbucketDataCenter) {
		return ErrRequiredOauthToken
	}
	if validString(o.PrivateKey) && *o.ServiceProvider != *ServiceProvider(ServiceProviderAzureDevOpsServer) {
		return ErrUnsupportedPrivateKey
	}
	return nil
}

func (o *OAuthClientListOptions) valid() error {
	return nil
}

// AddProjects adds projects to a given oauth client.
func (s *oAuthClients) AddProjects(ctx context.Context, oAuthClientID string, options OAuthClientAddProjectsOptions) error {
	if !validStringID(&oAuthClientID) {
		return ErrInvalidOauthClientID
	}
	if err := options.valid(); err != nil {
		return err
	}

	u := fmt.Sprintf("oauth-clients/%s/relationships/projects", url.PathEscape(oAuthClientID))
	req, err := s.client.NewRequest("POST", u, options.Projects)
	if err != nil {
		return err
	}

	return req.Do(ctx, nil)
}

// RemoveProjects removes projects from an oauth client.
func (s *oAuthClients) RemoveProjects(ctx context.Context, oAuthClientID string, options OAuthClientRemoveProjectsOptions) error {
	if !validStringID(&oAuthClientID) {
		return ErrInvalidOauthClientID
	}
	if err := options.valid(); err != nil {
		return err
	}

	u := fmt.Sprintf("oauth-clients/%s/relationships/projects", url.PathEscape(oAuthClientID))
	req, err := s.client.NewRequest("DELETE", u, options.Projects)
	if err != nil {
		return err
	}

	return req.Do(ctx, nil)
}

func (o OAuthClientAddProjectsOptions) valid() error {
	if o.Projects == nil {
		return ErrRequiredProject
	}
	if len(o.Projects) == 0 {
		return ErrProjectMinLimit
	}
	return nil
}

func (o OAuthClientRemoveProjectsOptions) valid() error {
	if o.Projects == nil {
		return ErrRequiredProject
	}
	if len(o.Projects) == 0 {
		return ErrProjectMinLimit
	}
	return nil
}

func (o *OAuthClientReadOptions) valid() error {
	return nil
}

// Compile-time proof of interface implementation.
var _ OAuthTokens = (*oAuthTokens)(nil)

// OAuthTokens describes all the OAuth token related methods that the
// Terraform Enterprise API supports.
//
// TFE API docs:
// https://developer.hashicorp.com/terraform/cloud-docs/api-docs/oauth-tokens
type OAuthTokens interface {
	// List all the OAuth tokens for a given organization.
	List(ctx context.Context, organization string, options *OAuthTokenListOptions) (*OAuthTokenList, error)
	// Read a OAuth token by its ID.
	Read(ctx context.Context, oAuthTokenID string) (*OAuthToken, error)

	// Update an existing OAuth token.
	Update(ctx context.Context, oAuthTokenID string, options OAuthTokenUpdateOptions) (*OAuthToken, error)

	// Delete a OAuth token by its ID.
	Delete(ctx context.Context, oAuthTokenID string) error
}

// oAuthTokens implements OAuthTokens.
type oAuthTokens struct {
	client *Client
}

// OAuthTokenList represents a list of OAuth tokens.
type OAuthTokenList struct {
	*Pagination
	Items []*OAuthToken
}

// OAuthToken represents a VCS configuration including the associated
// OAuth token
type OAuthToken struct {
	ID                  string    `jsonapi:"primary,oauth-tokens"`
	UID                 string    `jsonapi:"attr,uid"`
	CreatedAt           time.Time `jsonapi:"attr,created-at,iso8601"`
	HasSSHKey           bool      `jsonapi:"attr,has-ssh-key"`
	ServiceProviderUser string    `jsonapi:"attr,service-provider-user"`

	// Relations
	OAuthClient *OAuthClient `jsonapi:"relation,oauth-client"`
}

// OAuthTokenListOptions represents the options for listing
// OAuth tokens.
type OAuthTokenListOptions struct {
	ListOptions
}

// OAuthTokenUpdateOptions represents the options for updating an OAuth token.
type OAuthTokenUpdateOptions struct {
	// Type is a public field utilized by JSON:API to
	// set the resource type via the field tag.
	// It is not a user-defined value and does not need to be set.
	// https://jsonapi.org/format/#crud-creating
	Type string `jsonapi:"primary,oauth-tokens"`

	// Optional: A private SSH key to be used for git clone operations.
	PrivateSSHKey *string `jsonapi:"attr,ssh-key,omitempty"`
}

// List all the OAuth tokens for a given organization.
func (s *oAuthTokens) List(ctx context.Context, organization string, options *OAuthTokenListOptions) (*OAuthTokenList, error) {
	if !validStringID(&organization) {
		return nil, ErrInvalidOrg
	}

	u := fmt.Sprintf("organizations/%s/oauth-tokens", url.PathEscape(organization))
	req, err := s.client.NewRequest("GET", u, options)
	if err != nil {
		return nil, err
	}

	otl := &OAuthTokenList{}
	err = req.Do(ctx, otl)
	if err != nil {
		return nil, err
	}

	return otl, nil
}

// Read an OAuth token by its ID.
func (s *oAuthTokens) Read(ctx context.Context, oAuthTokenID string) (*OAuthToken, error) {
	if !validStringID(&oAuthTokenID) {
		return nil, ErrInvalidOauthTokenID
	}

	u := fmt.Sprintf("oauth-tokens/%s", url.PathEscape(oAuthTokenID))
	req, err := s.client.NewRequest("GET", u, nil)
	if err != nil {
		return nil, err
	}

	ot := &OAuthToken{}
	err = req.Do(ctx, ot)
	if err != nil {
		return nil, err
	}

	return ot, err
}

// Update an existing OAuth token.
func (s *oAuthTokens) Update(ctx context.Context, oAuthTokenID string, options OAuthTokenUpdateOptions) (*OAuthToken, error) {
	if !validStringID(&oAuthTokenID) {
		return nil, ErrInvalidOauthTokenID
	}

	u := fmt.Sprintf("oauth-tokens/%s", url.PathEscape(oAuthTokenID))
	req, err := s.client.NewRequest("PATCH", u, &options)
	if err != nil {
		return nil, err
	}

	ot := &OAuthToken{}
	err = req.Do(ctx, ot)
	if err != nil {
		return nil, err
	}

	return ot, err
}

// Delete an OAuth token by its ID.
func (s *oAuthTokens) Delete(ctx context.Context, oAuthTokenID string) error {
	if !validStringID(&oAuthTokenID) {
		return ErrInvalidOauthTokenID
	}

	u := fmt.Sprintf("oauth-tokens/%s", url.PathEscape(oAuthTokenID))
	req, err := s.client.NewRequest("DELETE", u, nil)
	if err != nil {
		return err
	}

	return req.Do(ctx, nil)
}

var _ OrganizationAuditConfigurations = (*organizationAuditConfigurations)(nil)

// OrganizationAuditConfigurations describes the configuration for auditing events for the organization.
type OrganizationAuditConfigurations interface {
	// Read the audit configuration of an organization by its name.
	Read(ctx context.Context, organization string) (*OrganizationAuditConfiguration, error)

	// Send a test audit event for an organization by its name.
	Test(ctx context.Context, organization string) (*OrganizationAuditConfigurationTest, error)

	// Update the audit configuration of an organization by its name.
	Update(ctx context.Context, organization string, options OrganizationAuditConfigurationOptions) (*OrganizationAuditConfiguration, error)
}

// OrganizationAuditConfiguration represents the auditing configuration for a HCP Terraform Organization.
type OrganizationAuditConfiguration struct {
	AuditTrails          *OrganizationAuditConfigAuditTrails    `jsonapi:"attr,audit-trails,omitempty"`
	HCPAuditLogStreaming *OrganizationAuditConfigAuditStreaming `jsonapi:"attr,hcp-audit-log-streaming,omitempty"`
	ID                   string                                 `jsonapi:"primary,audit-configurations"`
	Permissions          *OrganizationAuditConfigPermissions    `jsonapi:"attr,permissions,omitempty"`
	Timestamps           *OrganizationAuditConfigTimestamps     `jsonapi:"attr,timestamps,omitempty"`
	UpdatedAt            time.Time                              `jsonapi:"attr,updated-at,iso8601"`

	Organization *Organization `jsonapi:"relation,organization"`
}

type OrganizationAuditConfigAuditTrails struct {
	Enabled bool `jsonapi:"attr,enabled"`
}

type OrganizationAuditConfigAuditStreaming struct {
	Enabled                bool   `jsonapi:"attr,enabled"`
	OrganizationID         string `jsonapi:"attr,organization-id"`
	UseDefaultOrganization bool   `jsonapi:"attr,use-default-organization"`
}

type OrganizationAuditConfigPermissions struct {
	CanEnableHCPAuditLogStreaming              bool `jsonapi:"attr,can-enable-hcp-audit-log-streaming"`
	CanSetHCPAuditLogStreamingOrganization     bool `jsonapi:"attr,can-set-hcp-audit-log-streaming-organization-id"`
	CanUseDefaultAuditLogStreamingOrganization bool `jsonapi:"attr,can-use-default-audit-log-streaming-organization"`
}

type OrganizationAuditConfigTimestamps struct {
	AuditTrailsDisabledAt           *time.Time `jsonapi:"attr,audit-trails-disabled-at,iso8601,omitempty"`
	AuditTrailsEnabledAt            *time.Time `jsonapi:"attr,audit-trails-enabled-at,iso8601,omitempty"`
	AuditTrailsLastFailure          *time.Time `jsonapi:"attr,audit-trails-last-failure,iso8601,omitempty"`
	AuditTrailsLastSuccess          *time.Time `jsonapi:"attr,audit-trails-last-success,iso8601,omitempty"`
	HCPAuditLogStreamingDisabledAt  *time.Time `jsonapi:"attr,hcp-audit-log-streaming-disabled-at,iso8601,omitempty"`
	HCPAuditLogStreamingEnabledAt   *time.Time `jsonapi:"attr,hcp-audit-log-streaming-enabled-at,iso8601,omitempty"`
	HCPAuditLogStreamingLastFailure *time.Time `jsonapi:"attr,hcp-audit-log-streaming-last-failure,iso8601,omitempty"`
	HCPAuditLogStreamingLastSuccess *time.Time `jsonapi:"attr,hcp-audit-log-streaming-last-success,iso8601,omitempty"`
}

type OrganizationAuditConfigurationTest struct {
	RequestID *string `json:"request-id,omitempty"`
}

type OrganizationAuditConfigurationOptions struct {
	AuditTrails          *OrganizationAuditConfigAuditTrails    `jsonapi:"attr,audit-trails,omitempty"`
	HCPAuditLogStreaming *OrganizationAuditConfigAuditStreaming `jsonapi:"attr,hcp-audit-log-streaming,omitempty"`
}

type organizationAuditConfigurations struct {
	client *Client
}

// Read the audit configuration of an organization by its name.
func (s *organizationAuditConfigurations) Read(ctx context.Context, organization string) (*OrganizationAuditConfiguration, error) {
	if !validStringID(&organization) {
		return nil, ErrInvalidOrg
	}

	u := fmt.Sprintf("organizations/%s/audit-configuration", url.PathEscape(organization))
	req, err := s.client.NewRequest("GET", u, nil)
	if err != nil {
		return nil, err
	}

	ac := &OrganizationAuditConfiguration{}
	err = req.Do(ctx, ac)
	if err != nil {
		return nil, err
	}

	return ac, err
}

// Send a test audit event for an organization by its name.
func (s *organizationAuditConfigurations) Test(ctx context.Context, organization string) (*OrganizationAuditConfigurationTest, error) {
	if !validStringID(&organization) {
		return nil, ErrInvalidOrg
	}

	u := fmt.Sprintf("organizations/%s/audit-configuration/test", url.PathEscape(organization))
	req, err := s.client.NewRequest("POST", u, nil)
	if err != nil {
		return nil, err
	}

	result := &OrganizationAuditConfigurationTest{}
	err = req.DoJSON(ctx, result)
	if err != nil {
		return nil, err
	}

	return result, err
}

// Update the audit configuration of an organization by its name.
func (s *organizationAuditConfigurations) Update(ctx context.Context, organization string, options OrganizationAuditConfigurationOptions) (*OrganizationAuditConfiguration, error) {
	if !validStringID(&organization) {
		return nil, ErrInvalidOrg
	}

	u := fmt.Sprintf("organizations/%s/audit-configuration", url.PathEscape(organization))
	req, err := s.client.NewRequest("PATCH", u, &options)
	if err != nil {
		return nil, err
	}

	ac := &OrganizationAuditConfiguration{}
	err = req.Do(ctx, ac)
	if err != nil {
		return nil, err
	}

	return ac, err
}

// Compile-time proof of interface implementation.
var _ OrganizationMemberships = (*organizationMemberships)(nil)

// OrganizationMemberships describes all the organization membership related methods that
// the Terraform Enterprise API supports.
//
// TFE API docs:
// https://developer.hashicorp.com/terraform/cloud-docs/api-docs/organization-memberships
type OrganizationMemberships interface {
	// List all the organization memberships of the given organization.
	List(ctx context.Context, organization string, options *OrganizationMembershipListOptions) (*OrganizationMembershipList, error)

	// Create a new organization membership with the given options.
	Create(ctx context.Context, organization string, options OrganizationMembershipCreateOptions) (*OrganizationMembership, error)

	// Read an organization membership by ID
	Read(ctx context.Context, organizationMembershipID string) (*OrganizationMembership, error)

	// Read an organization membership by ID with options
	ReadWithOptions(ctx context.Context, organizationMembershipID string, options OrganizationMembershipReadOptions) (*OrganizationMembership, error)

	// Delete an organization membership by its ID.
	Delete(ctx context.Context, organizationMembershipID string) error
}

// organizationMemberships implements OrganizationMemberships.
type organizationMemberships struct {
	client *Client
}

// OrganizationMembershipStatus represents an organization membership status.
type OrganizationMembershipStatus string

const (
	OrganizationMembershipActive  OrganizationMembershipStatus = "active"
	OrganizationMembershipInvited OrganizationMembershipStatus = "invited"
)

// OrganizationMembershipList represents a list of organization memberships.
type OrganizationMembershipList struct {
	*Pagination
	Items []*OrganizationMembership
}

// OrganizationMembership represents a Terraform Enterprise organization membership.
type OrganizationMembership struct {
	ID     string                       `jsonapi:"primary,organization-memberships"`
	Status OrganizationMembershipStatus `jsonapi:"attr,status"`
	Email  string                       `jsonapi:"attr,email"`

	// Relations
	Organization *Organization `jsonapi:"relation,organization"`
	User         *User         `jsonapi:"relation,user"`
	Teams        []*Team       `jsonapi:"relation,teams"`
}

// OrgMembershipIncludeOpt represents the available options for include query params.
// https://developer.hashicorp.com/terraform/cloud-docs/api-docs/organization-memberships#available-related-resources
type OrgMembershipIncludeOpt string

const (
	OrgMembershipUser OrgMembershipIncludeOpt = "user"
	OrgMembershipTeam OrgMembershipIncludeOpt = "teams"
)

// OrganizationMembershipListOptions represents the options for listing organization memberships.
type OrganizationMembershipListOptions struct {
	ListOptions
	// Optional: A list of relations to include. See available resources
	// https://developer.hashicorp.com/terraform/cloud-docs/api-docs/organization-memberships#available-related-resources
	Include []OrgMembershipIncludeOpt `url:"include,omitempty"`

	// Optional: A list of organization member emails to filter by.
	Emails []string `url:"filter[email],omitempty"`

	// Optional: If specified, restricts results to those matching status value.
	Status OrganizationMembershipStatus `url:"filter[status],omitempty"`

	// Optional: A query string to search organization memberships by user name
	// and email.
	Query string `url:"q,omitempty"`
}

// OrganizationMembershipCreateOptions represents the options for creating an organization membership.
type OrganizationMembershipCreateOptions struct {
	// Type is a public field utilized by JSON:API to
	// set the resource type via the field tag.
	// It is not a user-defined value and does not need to be set.
	// https://jsonapi.org/format/#crud-creating
	Type string `jsonapi:"primary,organization-memberships"`

	// Required: User's email address.
	Email *string `jsonapi:"attr,email"`

	// Optional: A list of teams in the organization to add the user to
	Teams []*Team `jsonapi:"relation,teams,omitempty"`
}

// OrganizationMembershipReadOptions represents the options for reading organization memberships.
type OrganizationMembershipReadOptions struct {
	// Optional: A list of relations to include. See available resources
	// https://developer.hashicorp.com/terraform/cloud-docs/api-docs/organization-memberships#available-related-resources
	Include []OrgMembershipIncludeOpt `url:"include,omitempty"`
}

// List all the organization memberships of the given organization.
func (s *organizationMemberships) List(ctx context.Context, organization string, options *OrganizationMembershipListOptions) (*OrganizationMembershipList, error) {
	if !validStringID(&organization) {
		return nil, ErrInvalidOrg
	}
	if err := options.valid(); err != nil {
		return nil, err
	}

	u := fmt.Sprintf("organizations/%s/organization-memberships", url.PathEscape(organization))
	req, err := s.client.NewRequest("GET", u, options)
	if err != nil {
		return nil, err
	}

	ml := &OrganizationMembershipList{}
	err = req.Do(ctx, ml)
	if err != nil {
		return nil, err
	}

	return ml, nil
}

// Create an organization membership with the given options.
func (s *organizationMemberships) Create(ctx context.Context, organization string, options OrganizationMembershipCreateOptions) (*OrganizationMembership, error) {
	if !validStringID(&organization) {
		return nil, ErrInvalidOrg
	}
	if err := options.valid(); err != nil {
		return nil, err
	}

	u := fmt.Sprintf("organizations/%s/organization-memberships", url.PathEscape(organization))
	req, err := s.client.NewRequest("POST", u, &options)
	if err != nil {
		return nil, err
	}

	m := &OrganizationMembership{}
	err = req.Do(ctx, m)
	if err != nil {
		return nil, err
	}

	return m, nil
}

// Read an organization membership by its ID.
func (s *organizationMemberships) Read(ctx context.Context, organizationMembershipID string) (*OrganizationMembership, error) {
	return s.ReadWithOptions(ctx, organizationMembershipID, OrganizationMembershipReadOptions{})
}

// Read an organization membership by ID with options
func (s *organizationMemberships) ReadWithOptions(ctx context.Context, organizationMembershipID string, options OrganizationMembershipReadOptions) (*OrganizationMembership, error) {
	if !validStringID(&organizationMembershipID) {
		return nil, ErrInvalidMembership
	}
	if err := options.valid(); err != nil {
		return nil, err
	}

	u := fmt.Sprintf("organization-memberships/%s", url.PathEscape(organizationMembershipID))
	req, err := s.client.NewRequest("GET", u, &options)
	if err != nil {
		return nil, err
	}

	mem := &OrganizationMembership{}
	err = req.Do(ctx, mem)
	if err != nil {
		return nil, err
	}

	return mem, nil
}

// Delete an organization membership by its ID.
func (s *organizationMemberships) Delete(ctx context.Context, organizationMembershipID string) error {
	if !validStringID(&organizationMembershipID) {
		return ErrInvalidMembership
	}

	u := fmt.Sprintf("organization-memberships/%s", url.PathEscape(organizationMembershipID))
	req, err := s.client.NewRequest("DELETE", u, nil)
	if err != nil {
		return err
	}

	return req.Do(ctx, nil)
}

func (o OrganizationMembershipCreateOptions) valid() error {
	if o.Email == nil {
		return ErrRequiredEmail
	}
	return nil
}

func (o *OrganizationMembershipListOptions) valid() error {
	if o == nil {
		return nil
	}

	if err := validateOrgMembershipEmailParams(o.Emails); err != nil {
		return err
	}

	return nil
}

func (o OrganizationMembershipReadOptions) valid() error {
	return nil
}

func validateOrgMembershipEmailParams(emails []string) error {
	for _, email := range emails {
		if !validEmail(email) {
			return ErrInvalidEmail
		}
	}

	return nil
}

var _ OrganizationTags = (*organizationTags)(nil)

// OrganizationMemberships describes all the list of tags used with all resources across the organization.
//
// TFE API docs:
// https://developer.hashicorp.com/terraform/cloud-docs/api-docs/organization-tags
type OrganizationTags interface {
	// List all tags within an organization
	List(ctx context.Context, organization string, options *OrganizationTagsListOptions) (*OrganizationTagsList, error)

	// Delete tags from an organization
	Delete(ctx context.Context, organization string, options OrganizationTagsDeleteOptions) error

	// Associate an organization's workspace with a tag
	AddWorkspaces(ctx context.Context, tag string, options AddWorkspacesToTagOptions) error
}

// organizationTags implements OrganizationTags.
type organizationTags struct {
	client *Client
}

// OrganizationTagsList represents a list of organization tags
type OrganizationTagsList struct {
	*Pagination
	Items []*OrganizationTag
}

// OrganizationTag represents a Terraform Enterprise Organization tag
type OrganizationTag struct {
	ID string `jsonapi:"primary,tags"`
	// Optional:
	Name string `jsonapi:"attr,name,omitempty"`

	// Optional: Number of workspaces that have this tag
	InstanceCount int `jsonapi:"attr,instance-count,omitempty"`

	// The org this tag belongs to
	Organization *Organization `jsonapi:"relation,organization"`
}

// OrganizationTagsListOptions represents the options for listing organization tags
type OrganizationTagsListOptions struct {
	ListOptions
	// Optional:
	Filter string `url:"filter[exclude][taggable][id],omitempty"`

	// Optional: A search query string. Organization tags are searchable by name likeness.
	Query string `url:"q,omitempty"`
}

// OrganizationTagsDeleteOptions represents the request body for deleting a tag in an organization
type OrganizationTagsDeleteOptions struct {
	IDs []string // Required
}

// AddWorkspacesToTagOptions represents the request body to add a workspace to a tag
type AddWorkspacesToTagOptions struct {
	WorkspaceIDs []string // Required
}

// this represents a single tag ID
type tagID struct {
	ID string `jsonapi:"primary,tags"`
}

// this represents a single workspace ID
type workspaceID struct {
	ID string `jsonapi:"primary,workspaces"`
}

// List all the tags in an organization. You can provide query params through OrganizationTagsListOptions
func (s *organizationTags) List(ctx context.Context, organization string, options *OrganizationTagsListOptions) (*OrganizationTagsList, error) {
	if !validStringID(&organization) {
		return nil, ErrInvalidOrg
	}

	u := fmt.Sprintf("organizations/%s/tags", url.PathEscape(organization))
	req, err := s.client.NewRequest("GET", u, options)
	if err != nil {
		return nil, err
	}

	tags := &OrganizationTagsList{}
	err = req.Do(ctx, tags)
	if err != nil {
		return nil, err
	}

	return tags, nil
}

// Delete tags from a Terraform Enterprise organization
func (s *organizationTags) Delete(ctx context.Context, organization string, options OrganizationTagsDeleteOptions) error {
	if !validStringID(&organization) {
		return ErrInvalidOrg
	}

	if err := options.valid(); err != nil {
		return err
	}

	u := fmt.Sprintf("organizations/%s/tags", url.PathEscape(organization))
	var tagsToRemove []*tagID
	for _, id := range options.IDs {
		tagsToRemove = append(tagsToRemove, &tagID{ID: id})
	}

	req, err := s.client.NewRequest("DELETE", u, tagsToRemove)
	if err != nil {
		return err
	}

	return req.Do(ctx, nil)
}

// Add workspaces to a tag
func (s *organizationTags) AddWorkspaces(ctx context.Context, tag string, options AddWorkspacesToTagOptions) error {
	if !validStringID(&tag) {
		return ErrInvalidTag
	}

	if err := options.valid(); err != nil {
		return err
	}

	var workspaces []*workspaceID
	for _, id := range options.WorkspaceIDs {
		workspaces = append(workspaces, &workspaceID{ID: id})
	}

	u := fmt.Sprintf("tags/%s/relationships/workspaces", url.PathEscape(tag))
	req, err := s.client.NewRequest("POST", u, workspaces)
	if err != nil {
		return err
	}

	return req.Do(ctx, nil)
}

func (opts *OrganizationTagsDeleteOptions) valid() error {
	if len(opts.IDs) == 0 {
		return ErrRequiredTagID
	}

	for _, id := range opts.IDs {
		if !validStringID(&id) {
			errorMsg := fmt.Sprintf("%s is not a valid id value", id)
			return errors.New(errorMsg)
		}
	}

	return nil
}

func (w *AddWorkspacesToTagOptions) valid() error {
	if len(w.WorkspaceIDs) == 0 {
		return ErrRequiredTagWorkspaceID
	}

	for _, id := range w.WorkspaceIDs {
		if !validStringID(&id) {
			errorMsg := fmt.Sprintf("%s is not a valid id value", id)
			return errors.New(errorMsg)
		}
	}

	return nil
}

// Copyright IBM Corp. 2018, 2026
// SPDX-License-Identifier: MPL-2.0

var _ OrganizationTokenTTLPolicies = (*organizationTokenTTLPolicies)(nil)

type OrganizationTokenTTLPolicies interface {
	List(ctx context.Context, organization string, options *OrganizationTokenTTLPolicyListOptions) (*OrganizationTokenTTLPolicyList, error)
	Update(ctx context.Context, organization string, options OrganizationTokenTTLPolicyUpdateOptions) ([]*OrganizationTokenTTLPolicy, error)
}

type organizationTokenTTLPolicies struct {
	client *Client
}

type OrganizationTokenTTLPolicy struct {
	ID        string    `jsonapi:"primary,organization-token-ttl-policies"`
	TokenType TokenType `jsonapi:"attr,token-type"`
	MaxTTLMs  int64     `jsonapi:"attr,max-ttl-ms"`
	CreatedAt time.Time `jsonapi:"attr,created-at,iso8601"`
	UpdatedAt time.Time `jsonapi:"attr,updated-at,iso8601"`
}

type OrganizationTokenTTLPolicyList struct {
	*Pagination
	Items []*OrganizationTokenTTLPolicy
}

type OrganizationTokenTTLPolicyListOptions struct {
	ListOptions
}

type OrganizationTokenTTLPolicyUpdateItem struct {
	TokenType TokenType `jsonapi:"attr,token-type"`
	MaxTTLMs  int64     `jsonapi:"attr,max-ttl-ms"`
}

type OrganizationTokenTTLPolicyUpdateOptions struct {
	Type     string                                 `jsonapi:"primary,organization-token-ttl-policies"`
	Policies []OrganizationTokenTTLPolicyUpdateItem `jsonapi:"attr,token-ttl-policies"`
}

func (s *organizationTokenTTLPolicies) List(ctx context.Context, organization string, options *OrganizationTokenTTLPolicyListOptions) (*OrganizationTokenTTLPolicyList, error) {
	if !validStringID(&organization) {
		return nil, ErrInvalidOrg
	}

	u := fmt.Sprintf("organizations/%s/token-ttl-policies", url.PathEscape(organization))
	req, err := s.client.NewRequest("GET", u, options)
	if err != nil {
		return nil, err
	}

	policyList := &OrganizationTokenTTLPolicyList{}
	err = req.Do(ctx, policyList)
	if err != nil {
		return nil, err
	}

	return policyList, nil
}

func (s *organizationTokenTTLPolicies) Update(ctx context.Context, organization string, options OrganizationTokenTTLPolicyUpdateOptions) ([]*OrganizationTokenTTLPolicy, error) {
	if !validStringID(&organization) {
		return nil, ErrInvalidOrg
	}

	if len(options.Policies) == 0 {
		return nil, ErrRequiredPolicies
	}

	u := fmt.Sprintf("organizations/%s/token-ttl-policies", url.PathEscape(organization))
	req, err := s.client.NewRequest("PATCH", u, &options)
	if err != nil {
		return nil, err
	}

	policyList := &OrganizationTokenTTLPolicyList{}
	err = req.Do(ctx, policyList)
	if err != nil {
		return nil, err
	}

	return policyList.Items, nil
}

// Copyright IBM Corp. 2018, 2026
// SPDX-License-Identifier: MPL-2.0

// Compile-time proof of interface implementation.
var _ OrganizationTokens = (*organizationTokens)(nil)

type TokenType string

const (
	// A token which can only access the Audit Trails of an HCP Terraform Organization.
	// See https://developer.hashicorp.com/terraform/cloud-docs/api-docs/audit-trails-tokens
	AuditTrailToken TokenType = "audit-trails"

	// Token types for TTL policies
	TokenTypeOrganization TokenType = "organization"
	TokenTypeTeam         TokenType = "team"
	TokenTypeUser         TokenType = "user"
	TokenTypeAuditTrails  TokenType = "audit_trails"
)

// OrganizationTokens describes all the organization token related methods
// that the Terraform Enterprise API supports.
//
// TFE API docs:
// https://developer.hashicorp.com/terraform/cloud-docs/api-docs/organization-tokens
type OrganizationTokens interface {
	// Create a new organization token, replacing any existing token.
	Create(ctx context.Context, organization string) (*OrganizationToken, error)

	// CreateWithOptions a new organization token with options, replacing any existing token.
	CreateWithOptions(ctx context.Context, organization string, options OrganizationTokenCreateOptions) (*OrganizationToken, error)

	// Read an organization token.
	Read(ctx context.Context, organization string) (*OrganizationToken, error)

	// Read an organization token with options.
	ReadWithOptions(ctx context.Context, organization string, options OrganizationTokenReadOptions) (*OrganizationToken, error)

	// Delete an organization token.
	Delete(ctx context.Context, organization string) error

	// Delete an organization token with options.
	DeleteWithOptions(ctx context.Context, organization string, options OrganizationTokenDeleteOptions) error
}

// organizationTokens implements OrganizationTokens.
type organizationTokens struct {
	client *Client
}

// OrganizationToken represents a Terraform Enterprise organization token.
type OrganizationToken struct {
	ID          string           `jsonapi:"primary,authentication-tokens"`
	CreatedAt   time.Time        `jsonapi:"attr,created-at,iso8601"`
	Description string           `jsonapi:"attr,description"`
	LastUsedAt  time.Time        `jsonapi:"attr,last-used-at,iso8601"`
	Token       string           `jsonapi:"attr,token"`
	ExpiredAt   time.Time        `jsonapi:"attr,expired-at,iso8601"`
	CreatedBy   *CreatedByChoice `jsonapi:"polyrelation,created-by"`
}

// OrganizationTokenCreateOptions contains the options for creating an organization token.
type OrganizationTokenCreateOptions struct {
	// Optional: The token's expiration date.
	// This feature is available in TFE release v202305-1 and later
	ExpiredAt *time.Time `jsonapi:"attr,expired-at,iso8601,omitempty" url:"-"`
	// Optional: What type of token to create
	// This option is only applicable to HCP Terraform and is ignored by TFE.
	TokenType *TokenType `url:"token,omitempty"`
}

// OrganizationTokenReadOptions contains the options for reading an organization token.
type OrganizationTokenReadOptions struct {
	// Optional: What type of token to read
	// This option is only applicable to HCP Terraform and is ignored by TFE.
	TokenType *TokenType `url:"token,omitempty"`
}

// OrganizationTokenDeleteOptions contains the options for deleting an organization token.
type OrganizationTokenDeleteOptions struct {
	// Optional: What type of token to delete
	// This option is only applicable to HCP Terraform and is ignored by TFE.
	TokenType *TokenType `url:"token,omitempty"`
}

// Create a new organization token, replacing any existing token.
func (s *organizationTokens) Create(ctx context.Context, organization string) (*OrganizationToken, error) {
	return s.CreateWithOptions(ctx, organization, OrganizationTokenCreateOptions{})
}

// CreateWithOptions a new organization token with options, replacing any existing token.
func (s *organizationTokens) CreateWithOptions(ctx context.Context, organization string, options OrganizationTokenCreateOptions) (*OrganizationToken, error) {
	if !validStringID(&organization) {
		return nil, ErrInvalidOrg
	}

	u := fmt.Sprintf("organizations/%s/authentication-token", url.PathEscape(organization))
	qp, err := decodeQueryParams(options)
	if err != nil {
		return nil, err
	}

	req, err := s.client.NewRequestWithAdditionalQueryParams("POST", u, &options, qp)
	if err != nil {
		return nil, err
	}

	ot := &OrganizationToken{}
	err = req.Do(ctx, ot)
	if err != nil {
		return nil, err
	}

	return ot, err
}

// Read an organization token.
func (s *organizationTokens) Read(ctx context.Context, organization string) (*OrganizationToken, error) {
	return s.ReadWithOptions(ctx, organization, OrganizationTokenReadOptions{})
}

// Read an organization token with options.
func (s *organizationTokens) ReadWithOptions(ctx context.Context, organization string, options OrganizationTokenReadOptions) (*OrganizationToken, error) {
	if !validStringID(&organization) {
		return nil, ErrInvalidOrg
	}

	u := fmt.Sprintf("organizations/%s/authentication-token", url.PathEscape(organization))
	req, err := s.client.NewRequest("GET", u, options)
	if err != nil {
		return nil, err
	}

	ot := &OrganizationToken{}
	err = req.Do(ctx, ot)
	if err != nil {
		return nil, err
	}

	return ot, err
}

// Delete an organization token.
func (s *organizationTokens) Delete(ctx context.Context, organization string) error {
	return s.DeleteWithOptions(ctx, organization, OrganizationTokenDeleteOptions{})
}

// Delete an organization token with options
func (s *organizationTokens) DeleteWithOptions(ctx context.Context, organization string, options OrganizationTokenDeleteOptions) error {
	if !validStringID(&organization) {
		return ErrInvalidOrg
	}

	u := fmt.Sprintf("organizations/%s/authentication-token", url.PathEscape(organization))
	qp, err := decodeQueryParams(options)
	if err != nil {
		return err
	}

	req, err := s.client.NewRequestWithAdditionalQueryParams("DELETE", u, nil, qp)
	if err != nil {
		return err
	}

	return req.Do(ctx, nil)
}

// Copyright IBM Corp. 2018, 2026
// SPDX-License-Identifier: MPL-2.0

// Compile-time proof of interface implementation.
var _ Organizations = (*organizations)(nil)

// Organizations describes all the organization related methods that the
// Terraform Enterprise API supports.
//
// TFE API docs:
// https://developer.hashicorp.com/terraform/cloud-docs/api-docs/organizations
type Organizations interface {
	// List all the organizations visible to the current user.
	List(ctx context.Context, options *OrganizationListOptions) (*OrganizationList, error)

	// Create a new organization with the given options.
	Create(ctx context.Context, options OrganizationCreateOptions) (*Organization, error)

	// Read an organization by its name.
	Read(ctx context.Context, organization string) (*Organization, error)

	// Read an organization by its name with options
	ReadWithOptions(ctx context.Context, organization string, options OrganizationReadOptions) (*Organization, error)

	// Update attributes of an existing organization.
	Update(ctx context.Context, organization string, options OrganizationUpdateOptions) (*Organization, error)

	// Delete an organization by its name.
	Delete(ctx context.Context, organization string) error

	// ReadCapacity shows the current run capacity of an organization.
	ReadCapacity(ctx context.Context, organization string) (*Capacity, error)

	// ReadEntitlements shows the entitlements of an organization.
	ReadEntitlements(ctx context.Context, organization string) (*Entitlements, error)

	// ReadRunQueue shows the current run queue of an organization.
	ReadRunQueue(ctx context.Context, organization string, options ReadRunQueueOptions) (*RunQueue, error)

	// ReadDataRetentionPolicy reads an organization's data retention policy
	// **Note: This functionality is only available in Terraform Enterprise versions v202311-1 and v202312-1.**
	//
	// Deprecated: Use ReadDataRetentionPolicyChoice instead.
	ReadDataRetentionPolicy(ctx context.Context, organization string) (*DataRetentionPolicy, error)

	// ReadDataRetentionPolicyChoice reads an organization's data retention policy
	// **Note: This functionality is only available in Terraform Enterprise.**
	ReadDataRetentionPolicyChoice(ctx context.Context, organization string) (*DataRetentionPolicyChoice, error)

	// SetDataRetentionPolicy sets an organization's data retention policy
	// **Note: This functionality is only available in Terraform Enterprise versions v202311-1 and v202312-1.**
	//
	// Deprecated: Use SetDataRetentionPolicyDeleteOlder instead
	SetDataRetentionPolicy(ctx context.Context, organization string, options DataRetentionPolicySetOptions) (*DataRetentionPolicy, error)

	// SetDataRetentionPolicyDeleteOlder sets an organization's data retention policy to delete data older than a certain number of days
	// **Note: This functionality is only available in Terraform Enterprise.**
	SetDataRetentionPolicyDeleteOlder(ctx context.Context, organization string, options DataRetentionPolicyDeleteOlderSetOptions) (*DataRetentionPolicyDeleteOlder, error)

	// SetDataRetentionPolicyDontDelete sets an organization's data retention policy to explicitly not delete data
	// **Note: This functionality is only available in Terraform Enterprise.**
	SetDataRetentionPolicyDontDelete(ctx context.Context, organization string, options DataRetentionPolicyDontDeleteSetOptions) (*DataRetentionPolicyDontDelete, error)

	// DeleteDataRetentionPolicy deletes an organization's data retention policy
	// **Note: This functionality is only available in Terraform Enterprise.**
	DeleteDataRetentionPolicy(ctx context.Context, organization string) error
}

// organizations implements Organizations.
type organizations struct {
	client *Client
}

// AuthPolicyType represents an authentication policy type.
type AuthPolicyType string

// List of available authentication policies.
const (
	AuthPolicyPassword  AuthPolicyType = "password"
	AuthPolicyTwoFactor AuthPolicyType = "two_factor_mandatory"
)

// OrganizationList represents a list of organizations.
type OrganizationList struct {
	*Pagination
	Items []*Organization
}

// Organization represents a Terraform Enterprise organization.
type Organization struct {
	Name                                              string                   `jsonapi:"primary,organizations"`
	AssessmentsEnforced                               bool                     `jsonapi:"attr,assessments-enforced"`
	CollaboratorAuthPolicy                            AuthPolicyType           `jsonapi:"attr,collaborator-auth-policy"`
	CostEstimationEnabled                             bool                     `jsonapi:"attr,cost-estimation-enabled"`
	CreatedAt                                         time.Time                `jsonapi:"attr,created-at,iso8601"`
	DefaultExecutionMode                              string                   `jsonapi:"attr,default-execution-mode"`
	Email                                             string                   `jsonapi:"attr,email"`
	ExternalID                                        string                   `jsonapi:"attr,external-id"`
	IsUnified                                         bool                     `jsonapi:"attr,is-unified"`
	OwnersTeamSAMLRoleID                              string                   `jsonapi:"attr,owners-team-saml-role-id"`
	Permissions                                       *OrganizationPermissions `jsonapi:"attr,permissions"`
	SAMLEnabled                                       bool                     `jsonapi:"attr,saml-enabled"`
	StacksEnabled                                     bool                     `jsonapi:"attr,stacks-enabled"`
	SessionRemember                                   int                      `jsonapi:"attr,session-remember"`
	SessionTimeout                                    int                      `jsonapi:"attr,session-timeout"`
	TrialExpiresAt                                    time.Time                `jsonapi:"attr,trial-expires-at,iso8601"`
	TwoFactorConformant                               bool                     `jsonapi:"attr,two-factor-conformant"`
	SendPassingStatusesForUntriggeredSpeculativePlans bool                     `jsonapi:"attr,send-passing-statuses-for-untriggered-speculative-plans"`
	RemainingTestableCount                            int                      `jsonapi:"attr,remaining-testable-count"`
	SpeculativePlanManagementEnabled                  bool                     `jsonapi:"attr,speculative-plan-management-enabled"`
	EnforceHYOK                                       bool                     `jsonapi:"attr,enforce-hyok"`
	UserTokensEnabled                                 *bool                    `jsonapi:"attr,user-tokens-enabled"`
	MaxTTLEnabled                                     bool                     `jsonapi:"attr,max-ttl-enabled"`
	// Optional: If enabled, SendPassingStatusesForUntriggeredSpeculativePlans needs to be false.
	AggregatedCommitStatusEnabled bool `jsonapi:"attr,aggregated-commit-status-enabled,omitempty"`
	// Note: This will be false for TFE versions older than v202211, where the setting was introduced.
	// On those TFE versions, safe delete does not exist, so ALL deletes will be force deletes.
	AllowForceDeleteWorkspaces bool `jsonapi:"attr,allow-force-delete-workspaces"`

	// Relations
	DefaultProject           *Project           `jsonapi:"relation,default-project"`
	DefaultAgentPool         *AgentPool         `jsonapi:"relation,default-agent-pool"`
	PrimaryHYOKConfiguration *HYOKConfiguration `jsonapi:"relation,primary-hyok-configuration,omitempty"`

	// Deprecated: Use DataRetentionPolicyChoice instead.
	DataRetentionPolicy *DataRetentionPolicy

	// **Note: This functionality is only available in Terraform Enterprise.**
	DataRetentionPolicyChoice *DataRetentionPolicyChoice `jsonapi:"polyrelation,data-retention-policy"`
}

// OrganizationIncludeOpt represents the available options for include query params.
// https://developer.hashicorp.com/terraform/cloud-docs/api-docs/organizations#available-related-resources
type OrganizationIncludeOpt string

const (
	// **Note: This include option is still in BETA and subject to change.**
	OrganizationDefaultProject OrganizationIncludeOpt = "default-project"
)

// OrganizationReadOptions represents the options for reading organizations.
type OrganizationReadOptions struct {
	// Optional: A list of relations to include. See available resources
	// https://developer.hashicorp.com/terraform/cloud-docs/api-docs/organizations#available-related-resources
	Include []OrganizationIncludeOpt `url:"include,omitempty"`
}

// Capacity represents the current run capacity of an organization.
type Capacity struct {
	Organization string `jsonapi:"primary,organization-capacity"`
	Pending      int    `jsonapi:"attr,pending"`
	Running      int    `jsonapi:"attr,running"`
}

// Entitlements represents the entitlements of an organization.
type Entitlements struct {
	ID                         string `jsonapi:"primary,entitlement-sets"`
	Agents                     bool   `jsonapi:"attr,agents"`
	AuditLogging               bool   `jsonapi:"attr,audit-logging"`
	CostEstimation             bool   `jsonapi:"attr,cost-estimation"`
	GlobalRunTasks             bool   `jsonapi:"attr,global-run-tasks"`
	Operations                 bool   `jsonapi:"attr,operations"`
	PrivateModuleRegistry      bool   `jsonapi:"attr,private-module-registry"`
	PrivateRunTasks            bool   `jsonapi:"attr,private-run-tasks"`
	RunTasks                   bool   `jsonapi:"attr,run-tasks"`
	SSO                        bool   `jsonapi:"attr,sso"`
	Sentinel                   bool   `jsonapi:"attr,sentinel"`
	StateStorage               bool   `jsonapi:"attr,state-storage"`
	Teams                      bool   `jsonapi:"attr,teams"`
	VCSIntegrations            bool   `jsonapi:"attr,vcs-integrations"`
	WaypointActions            bool   `jsonapi:"attr,waypoint-actions"`
	WaypointTemplatesAndAddons bool   `jsonapi:"attr,waypoint-templates-and-addons"`
	Infragraph                 bool   `jsonapi:"attr,infragraph"`
	InfragraphWithNRTU         bool   `jsonapi:"attr,infragraph-with-nrtu"`
}

// RunQueue represents the current run queue of an organization.
type RunQueue struct {
	*Pagination
	Items []*Run
}

// OrganizationPermissions represents the organization permissions.
type OrganizationPermissions struct {
	CanCreateTeam               bool `jsonapi:"attr,can-create-team"`
	CanCreateWorkspace          bool `jsonapi:"attr,can-create-workspace"`
	CanCreateWorkspaceMigration bool `jsonapi:"attr,can-create-workspace-migration"`
	CanDeployNoCodeModules      bool `jsonapi:"attr,can-deploy-no-code-modules"`
	CanDestroy                  bool `jsonapi:"attr,can-destroy"`
	CanManageAuditing           bool `jsonapi:"attr,can-manage-auditing"`
	CanManageNoCodeModules      bool `jsonapi:"attr,can-manage-no-code-modules"`
	CanManageRunTasks           bool `jsonapi:"attr,can-manage-run-tasks"`
	CanTraverse                 bool `jsonapi:"attr,can-traverse"`
	CanUpdate                   bool `jsonapi:"attr,can-update"`
	CanUpdateAPIToken           bool `jsonapi:"attr,can-update-api-token"`
	CanUpdateOAuth              bool `jsonapi:"attr,can-update-oauth"`
	CanUpdateSentinel           bool `jsonapi:"attr,can-update-sentinel"`
	CanUpdateHYOKConfiguration  bool `jsonapi:"attr,can-update-hyok-configuration"`
	CanViewHYOKFeatureInfo      bool `jsonapi:"attr,can-view-hyok-feature-info"`
	CanEnableStacks             bool `jsonapi:"attr,can-enable-stacks"`
	CanCreateProject            bool `jsonapi:"attr,can-create-project"`
}

// OrganizationListOptions represents the options for listing organizations.
type OrganizationListOptions struct {
	ListOptions

	// Optional: A query string used to filter organizations.
	// Organizations with a name or email partially matching this value will be returned.
	Query string `url:"q,omitempty"`
}

// OrganizationCreateOptions represents the options for creating an organization.
type OrganizationCreateOptions struct {
	// Type is a public field utilized by JSON:API to
	// set the resource type via the field tag.
	// It is not a user-defined value and does not need to be set.
	// https://jsonapi.org/format/#crud-creating
	Type string `jsonapi:"primary,organizations"`

	// Required: Name of the organization.
	Name *string `jsonapi:"attr,name"`

	// Optional: AssessmentsEnforced toggles whether health assessment enablement is enforced across all assessable workspaces (those with a minimum terraform version of 0.15.4 and not running in local execution mode) or if the decision to enabled health assessments is delegated to the workspace setting AssessmentsEnabled.
	AssessmentsEnforced *bool `jsonapi:"attr,assessments-enforced,omitempty"`

	// Required: Admin email address.
	Email *string `jsonapi:"attr,email"`

	// Optional: Session expiration (minutes).
	SessionRemember *int `jsonapi:"attr,session-remember,omitempty"`

	// Optional: Session timeout after inactivity (minutes).
	SessionTimeout *int `jsonapi:"attr,session-timeout,omitempty"`

	// Optional: Authentication policy.
	CollaboratorAuthPolicy *AuthPolicyType `jsonapi:"attr,collaborator-auth-policy,omitempty"`

	// Optional: Enable Cost Estimation
	CostEstimationEnabled *bool `jsonapi:"attr,cost-estimation-enabled,omitempty"`

	// Optional: The name of the "owners" team
	OwnersTeamSAMLRoleID *string `jsonapi:"attr,owners-team-saml-role-id,omitempty"`

	// Optional: SendPassingStatusesForUntriggeredSpeculativePlans toggles behavior of untriggered speculative plans to send status updates to version control systems like GitHub.
	SendPassingStatusesForUntriggeredSpeculativePlans *bool `jsonapi:"attr,send-passing-statuses-for-untriggered-speculative-plans,omitempty"`

	// Optional: If enabled, SendPassingStatusesForUntriggeredSpeculativePlans needs to be false.
	AggregatedCommitStatusEnabled *bool `jsonapi:"attr,aggregated-commit-status-enabled,omitempty"`

	// Optional: SpeculativePlanManagementEnabled toggles whether pending speculative plans from outdated commits will be cancelled if a newer commit is pushed to the same branch.
	SpeculativePlanManagementEnabled *bool `jsonapi:"attr,speculative-plan-management-enabled,omitempty"`

	// Optional: AllowForceDeleteWorkspaces toggles behavior of allowing workspace admins to delete workspaces with resources under management.
	AllowForceDeleteWorkspaces *bool `jsonapi:"attr,allow-force-delete-workspaces,omitempty"`

	// Optional: DefaultExecutionMode the default execution mode for workspaces
	DefaultExecutionMode *string `jsonapi:"attr,default-execution-mode,omitempty"`

	// Optional: EnforceHYOK if HYOK is enforced for the organization.
	EnforceHYOK *bool `jsonapi:"attr,enforce-hyok,omitempty"`

	// Optional: StacksEnabled toggles whether stacks are enabled for the organization. This setting
	// is considered BETA, SUBJECT TO CHANGE, and likely unavailable to most users.
	StacksEnabled *bool `jsonapi:"attr,stacks-enabled,omitempty"`

	// Optional: RegistryMonorepoSupportEnabled toggles whether monorepo support is enabled for the organization
	RegistryMonorepoSupportEnabled *bool `jsonapi:"attr,registry-monorepo-support-enabled,omitempty"`

	// Optional: UserTokensEnabled toggles whether user tokens may be used to access resources in this organization.
	UserTokensEnabled *bool `jsonapi:"attr,user-tokens-enabled,omitempty"`

	// Optional: MaxTTLEnabled toggles whether maximum TTL enforcement is enabled for API tokens in this organization.
	MaxTTLEnabled *bool `jsonapi:"attr,max-ttl-enabled,omitempty"`
}

// OrganizationUpdateOptions represents the options for updating an organization.
type OrganizationUpdateOptions struct {
	// Type is a public field utilized by JSON:API to
	// set the resource type via the field tag.
	// It is not a user-defined value and does not need to be set.
	// https://jsonapi.org/format/#crud-creating
	Type string `jsonapi:"primary,organizations"`

	// New name for the organization.
	Name *string `jsonapi:"attr,name,omitempty"`

	// Optional: AssessmentsEnforced toggles whether health assessment enablement is enforced across all assessable workspaces (those with a minimum terraform version of 0.15.4 and not running in local execution mode) or if the decision to enabled health assessments is delegated to the workspace setting AssessmentsEnabled.
	AssessmentsEnforced *bool `jsonapi:"attr,assessments-enforced,omitempty"`

	// New admin email address.
	Email *string `jsonapi:"attr,email,omitempty"`

	// Session expiration (minutes).
	SessionRemember *int `jsonapi:"attr,session-remember,omitempty"`

	// Session timeout after inactivity (minutes).
	SessionTimeout *int `jsonapi:"attr,session-timeout,omitempty"`

	// Authentication policy.
	CollaboratorAuthPolicy *AuthPolicyType `jsonapi:"attr,collaborator-auth-policy,omitempty"`

	// Enable Cost Estimation
	CostEstimationEnabled *bool `jsonapi:"attr,cost-estimation-enabled,omitempty"`

	// The name of the "owners" team
	OwnersTeamSAMLRoleID *string `jsonapi:"attr,owners-team-saml-role-id,omitempty"`

	// SendPassingStatusesForUntriggeredSpeculativePlans toggles behavior of untriggered speculative plans to send status updates to version control systems like GitHub.
	SendPassingStatusesForUntriggeredSpeculativePlans *bool `jsonapi:"attr,send-passing-statuses-for-untriggered-speculative-plans,omitempty"`

	// Optional: If enabled, SendPassingStatusesForUntriggeredSpeculativePlans needs to be false.
	AggregatedCommitStatusEnabled *bool `jsonapi:"attr,aggregated-commit-status-enabled,omitempty"`

	// Optional: SpeculativePlanManagementEnabled toggles whether pending speculative plans from outdated commits will be cancelled if a newer commit is pushed to the same branch.
	SpeculativePlanManagementEnabled *bool `jsonapi:"attr,speculative-plan-management-enabled,omitempty"`

	// Optional: AllowForceDeleteWorkspaces toggles behavior of allowing workspace admins to delete workspaces with resources under management.
	AllowForceDeleteWorkspaces *bool `jsonapi:"attr,allow-force-delete-workspaces,omitempty"`

	// Optional: DefaultExecutionMode the default execution mode for workspaces
	DefaultExecutionMode *string `jsonapi:"attr,default-execution-mode,omitempty"`

	// Optional: DefaultAgentPoolId default agent pool for workspaces, requires DefaultExecutionMode to be set to `agent`
	DefaultAgentPool *AgentPool `jsonapi:"relation,default-agent-pool,omitempty"`

	// Optional: EnforceHYOK if HYOK is enforced for the organization.
	EnforceHYOK *bool `jsonapi:"attr,enforce-hyok,omitempty"`

	// Optional: StacksEnabled toggles whether stacks are enabled for the organization. This setting
	// is considered BETA, SUBJECT TO CHANGE, and likely unavailable to most users.
	StacksEnabled *bool `jsonapi:"attr,stacks-enabled,omitempty"`

	// Optional: RegistryMonorepoSupportEnabled toggles whether monorepo support is enabled for the organization
	RegistryMonorepoSupportEnabled *bool `jsonapi:"attr,registry-monorepo-support-enabled,omitempty"`

	// Optional: UserTokensEnabled toggles whether user tokens may be used to access resources in this organization.
	UserTokensEnabled *bool `jsonapi:"attr,user-tokens-enabled,omitempty"`

	// Optional: MaxTTLEnabled toggles whether maximum TTL enforcement is enabled for API tokens in this organization.
	MaxTTLEnabled *bool `jsonapi:"attr,max-ttl-enabled,omitempty"`
}

// ReadRunQueueOptions represents the options for showing the queue.
type ReadRunQueueOptions struct {
	ListOptions
}

// List all the organizations visible to the current user.
func (s *organizations) List(ctx context.Context, options *OrganizationListOptions) (*OrganizationList, error) {
	req, err := s.client.NewRequest("GET", "organizations", options)
	if err != nil {
		return nil, err
	}

	orgl := &OrganizationList{}
	err = req.Do(ctx, orgl)
	if err != nil {
		return nil, err
	}

	return orgl, nil
}

// Create a new organization with the given options.
func (s *organizations) Create(ctx context.Context, options OrganizationCreateOptions) (*Organization, error) {
	if err := options.valid(); err != nil {
		return nil, err
	}

	req, err := s.client.NewRequest("POST", "organizations", &options)
	if err != nil {
		return nil, err
	}

	org := &Organization{}
	err = req.Do(ctx, org)
	if err != nil {
		return nil, err
	}

	return org, nil
}

// Read an organization by its name.
func (s *organizations) Read(ctx context.Context, organization string) (*Organization, error) {
	return s.ReadWithOptions(ctx, organization, OrganizationReadOptions{})
}

// Read an organization by its name with options
func (s *organizations) ReadWithOptions(ctx context.Context, organization string, options OrganizationReadOptions) (*Organization, error) {
	if !validStringID(&organization) {
		return nil, ErrInvalidOrg
	}

	u := fmt.Sprintf("organizations/%s", url.PathEscape(organization))
	req, err := s.client.NewRequest("GET", u, &options)
	if err != nil {
		return nil, err
	}

	org := &Organization{}
	err = req.Do(ctx, org)
	if err != nil {
		return nil, err
	}

	// Manually populate the deprecated DataRetentionPolicy field
	org.DataRetentionPolicy = org.DataRetentionPolicyChoice.ConvertToLegacyStruct()

	return org, nil
}

// Update attributes of an existing organization.
func (s *organizations) Update(ctx context.Context, organization string, options OrganizationUpdateOptions) (*Organization, error) {
	if !validStringID(&organization) {
		return nil, ErrInvalidOrg
	}

	u := fmt.Sprintf("organizations/%s", url.PathEscape(organization))
	req, err := s.client.NewRequest("PATCH", u, &options)
	if err != nil {
		return nil, err
	}

	org := &Organization{}
	err = req.Do(ctx, org)
	if err != nil {
		return nil, err
	}

	return org, nil
}

// Delete an organization by its name.
func (s *organizations) Delete(ctx context.Context, organization string) error {
	if !validStringID(&organization) {
		return ErrInvalidOrg
	}

	u := fmt.Sprintf("organizations/%s", url.PathEscape(organization))
	req, err := s.client.NewRequest("DELETE", u, nil)
	if err != nil {
		return err
	}

	return req.Do(ctx, nil)
}

// ReadCapacity shows the currently used capacity of an organization.
func (s *organizations) ReadCapacity(ctx context.Context, organization string) (*Capacity, error) {
	if !validStringID(&organization) {
		return nil, ErrInvalidOrg
	}

	u := fmt.Sprintf("organizations/%s/capacity", url.PathEscape(organization))
	req, err := s.client.NewRequest("GET", u, nil)
	if err != nil {
		return nil, err
	}

	c := &Capacity{}
	err = req.Do(ctx, c)
	if err != nil {
		return nil, err
	}

	return c, nil
}

// ReadEntitlements shows the entitlements of an organization.
func (s *organizations) ReadEntitlements(ctx context.Context, organization string) (*Entitlements, error) {
	if !validStringID(&organization) {
		return nil, ErrInvalidOrg
	}

	u := fmt.Sprintf("organizations/%s/entitlement-set", url.PathEscape(organization))
	req, err := s.client.NewRequest("GET", u, nil)
	if err != nil {
		return nil, err
	}

	e := &Entitlements{}
	err = req.Do(ctx, e)
	if err != nil {
		return nil, err
	}

	return e, nil
}

// ReadRunQueue shows the current run queue of an organization.
func (s *organizations) ReadRunQueue(ctx context.Context, organization string, options ReadRunQueueOptions) (*RunQueue, error) {
	if !validStringID(&organization) {
		return nil, ErrInvalidOrg
	}

	u := fmt.Sprintf("organizations/%s/runs/queue", url.PathEscape(organization))
	req, err := s.client.NewRequest("GET", u, &options)
	if err != nil {
		return nil, err
	}

	rq := &RunQueue{}
	err = req.Do(ctx, rq)
	if err != nil {
		return nil, err
	}

	return rq, nil
}

func (s *organizations) ReadDataRetentionPolicy(ctx context.Context, organization string) (*DataRetentionPolicy, error) {
	if !validStringID(&organization) {
		return nil, ErrInvalidOrg
	}

	u := fmt.Sprintf("organizations/%s/relationships/data-retention-policy", url.PathEscape(organization))
	req, err := s.client.NewRequest("GET", u, nil)
	if err != nil {
		return nil, err
	}

	dataRetentionPolicy := &DataRetentionPolicy{}
	err = req.Do(ctx, dataRetentionPolicy)

	if err != nil {
		// try to detect known issue where this function is used with TFE >= 202401,
		// and direct user towards the V2 function
		if drpUnmarshalEr.MatchString(err.Error()) {
			return nil, fmt.Errorf("error reading deprecated DataRetentionPolicy, use ReadDataRetentionPolicyChoice instead")
		}
		return nil, err
	}

	return dataRetentionPolicy, nil
}

func (s *organizations) ReadDataRetentionPolicyChoice(ctx context.Context, organization string) (*DataRetentionPolicyChoice, error) {
	if !validStringID(&organization) {
		return nil, ErrInvalidOrg
	}

	// The API to read the drp is org/<name>/relationships/data-retention-policy
	// However, this API can return multiple "types" (e.g. data-retention-policy-delete-olders, or data-retention-policy-dont-deletes)
	// Ideally we would deserialize this directly into the choice type (DataRetentionPolicyChoice)...however, there isn't a way to
	// tell the current jsonapi implementation that the direct result of an endpoint could be different types. Relationships can be polymorphic,
	// but the direct result of an endpoint can't be (as far as the jsonapi implementation is concerned)

	// Instead, we need to figure out the type of the data retention policy first, and deserialize it into the matching model. We
	// can then create a choice type manually
	org, err := s.Read(ctx, organization)
	if err != nil {
		return nil, err
	}

	// there is no drp (of a known type)
	if org.DataRetentionPolicyChoice == nil || !org.DataRetentionPolicyChoice.IsPopulated() {
		return org.DataRetentionPolicyChoice, nil
	}

	u := s.dataRetentionPolicyLink(organization)
	req, err := s.client.NewRequest("GET", u, nil)
	if err != nil {
		return nil, err
	}

	dataRetentionPolicy := &DataRetentionPolicyChoice{}
	// if reading the org told us it was a "delete older policy" deserialize into the DeleteOlder portion of the choice model
	if org.DataRetentionPolicyChoice.DataRetentionPolicyDeleteOlder != nil {
		deleteOlder := &DataRetentionPolicyDeleteOlder{}
		err = req.Do(ctx, deleteOlder)
		dataRetentionPolicy.DataRetentionPolicyDeleteOlder = deleteOlder

		// if reading the org told us it was a "delete older policy" deserialize into the DeleteOlder portion of the choice model
	} else if org.DataRetentionPolicyChoice.DataRetentionPolicyDontDelete != nil {
		dontDelete := &DataRetentionPolicyDontDelete{}
		err = req.Do(ctx, dontDelete)
		dataRetentionPolicy.DataRetentionPolicyDontDelete = dontDelete
	} else if org.DataRetentionPolicyChoice.DataRetentionPolicy != nil {
		legacyDrp := &DataRetentionPolicy{}
		err = req.Do(ctx, legacyDrp)
		dataRetentionPolicy.DataRetentionPolicy = legacyDrp
	}

	if err != nil {
		return nil, err
	}

	return dataRetentionPolicy, nil
}

// Deprecated: Use SetDataRetentionPolicyDeleteOlder instead
// **Note: This functionality is only available in Terraform Enterprise versions v202311-1 and v202312-1.**
func (s *organizations) SetDataRetentionPolicy(ctx context.Context, organization string, options DataRetentionPolicySetOptions) (*DataRetentionPolicy, error) {
	if !validStringID(&organization) {
		return nil, ErrInvalidOrg
	}

	u := s.dataRetentionPolicyLink(organization)
	req, err := s.client.NewRequest("PATCH", u, &options)
	if err != nil {
		return nil, err
	}

	dataRetentionPolicy := &DataRetentionPolicy{}
	err = req.Do(ctx, dataRetentionPolicy)

	if err != nil {
		return nil, err
	}

	return dataRetentionPolicy, nil
}

func (s *organizations) SetDataRetentionPolicyDeleteOlder(ctx context.Context, organization string, options DataRetentionPolicyDeleteOlderSetOptions) (*DataRetentionPolicyDeleteOlder, error) {
	if !validStringID(&organization) {
		return nil, ErrInvalidOrg
	}

	u := s.dataRetentionPolicyLink(organization)
	req, err := s.client.NewRequest("POST", u, &options)
	if err != nil {
		return nil, err
	}

	dataRetentionPolicy := &DataRetentionPolicyDeleteOlder{}
	err = req.Do(ctx, dataRetentionPolicy)

	if err != nil {
		return nil, err
	}

	return dataRetentionPolicy, nil
}

func (s *organizations) SetDataRetentionPolicyDontDelete(ctx context.Context, organization string, options DataRetentionPolicyDontDeleteSetOptions) (*DataRetentionPolicyDontDelete, error) {
	if !validStringID(&organization) {
		return nil, ErrInvalidOrg
	}

	u := s.dataRetentionPolicyLink(organization)
	req, err := s.client.NewRequest("POST", u, &options)
	if err != nil {
		return nil, err
	}

	dataRetentionPolicy := &DataRetentionPolicyDontDelete{}
	err = req.Do(ctx, dataRetentionPolicy)

	if err != nil {
		return nil, err
	}

	return dataRetentionPolicy, nil
}

func (s *organizations) DeleteDataRetentionPolicy(ctx context.Context, organization string) error {
	if !validStringID(&organization) {
		return ErrInvalidOrg
	}

	u := s.dataRetentionPolicyLink(organization)
	req, err := s.client.NewRequest("DELETE", u, nil)
	if err != nil {
		return err
	}

	return req.Do(ctx, nil)
}

func (o OrganizationCreateOptions) valid() error {
	if !validString(o.Name) {
		return ErrRequiredName
	}
	if !validStringID(o.Name) {
		return ErrInvalidName
	}
	if !validString(o.Email) {
		return ErrRequiredEmail
	}
	return nil
}

func (s *organizations) dataRetentionPolicyLink(name string) string {
	return fmt.Sprintf("organizations/%s/relationships/data-retention-policy", url.PathEscape(name))
}

// Compile-time proof of interface implementation.
var _ PlanExports = (*planExports)(nil)

// PlanExports describes all the plan export related methods that the Terraform
// Enterprise API supports.
//
// TFE API docs: https://developer.hashicorp.com/terraform/cloud-docs/api-docs/plan-exports
type PlanExports interface {
	// Export a plan by its ID with the given options.
	Create(ctx context.Context, options PlanExportCreateOptions) (*PlanExport, error)

	// Read a plan export by its ID.
	Read(ctx context.Context, planExportID string) (*PlanExport, error)

	// Delete a plan export by its ID.
	Delete(ctx context.Context, planExportID string) error

	// Download the data of an plan export.
	Download(ctx context.Context, planExportID string) ([]byte, error)
}

// planExports implements PlanExports.
type planExports struct {
	client *Client
}

// PlanExportDataType represents the type of data exported from a plan.
type PlanExportDataType string

// List all available plan export data types.
const (
	PlanExportSentinelMockBundleV0 PlanExportDataType = "sentinel-mock-bundle-v0"
)

// PlanExportStatus represents a plan export state.
type PlanExportStatus string

// List all available plan export statuses.
const (
	PlanExportCanceled PlanExportStatus = "canceled"
	PlanExportErrored  PlanExportStatus = "errored"
	PlanExportExpired  PlanExportStatus = "expired"
	PlanExportFinished PlanExportStatus = "finished"
	PlanExportPending  PlanExportStatus = "pending"
	PlanExportQueued   PlanExportStatus = "queued"
)

// PlanExportStatusTimestamps holds the timestamps for plan export statuses.
type PlanExportStatusTimestamps struct {
	CanceledAt time.Time `jsonapi:"attr,canceled-at,rfc3339"`
	ErroredAt  time.Time `jsonapi:"attr,errored-at,rfc3339"`
	ExpiredAt  time.Time `jsonapi:"attr,expired-at,rfc3339"`
	FinishedAt time.Time `jsonapi:"attr,finished-at,rfc3339"`
	QueuedAt   time.Time `jsonapi:"attr,queued-at,rfc3339"`
}

// PlanExport represents an export of Terraform Enterprise plan data.
type PlanExport struct {
	ID               string                      `jsonapi:"primary,plan-exports"`
	DataType         PlanExportDataType          `jsonapi:"attr,data-type"`
	Status           PlanExportStatus            `jsonapi:"attr,status"`
	StatusTimestamps *PlanExportStatusTimestamps `jsonapi:"attr,status-timestamps"`
}

// PlanExportCreateOptions represents the options for exporting data from a plan.
type PlanExportCreateOptions struct {
	// Type is a public field utilized by JSON:API to
	// set the resource type via the field tag.
	// It is not a user-defined value and does not need to be set.
	// https://jsonapi.org/format/#crud-creating
	Type string `jsonapi:"primary,plan-exports"`

	// Required: The plan to export.
	Plan *Plan `jsonapi:"relation,plan"`

	// Required: The name of the policy set.
	DataType *PlanExportDataType `jsonapi:"attr,data-type"`
}

// Create a plan export
func (s *planExports) Create(ctx context.Context, options PlanExportCreateOptions) (*PlanExport, error) {
	if err := options.valid(); err != nil {
		return nil, err
	}

	req, err := s.client.NewRequest("POST", "plan-exports", &options)
	if err != nil {
		return nil, err
	}

	pe := &PlanExport{}
	err = req.Do(ctx, pe)
	if err != nil {
		return nil, err
	}

	return pe, err
}

// Read a plan export by its ID.
func (s *planExports) Read(ctx context.Context, planExportID string) (*PlanExport, error) {
	if !validStringID(&planExportID) {
		return nil, ErrInvalidPlanExportID
	}

	u := fmt.Sprintf("plan-exports/%s", url.PathEscape(planExportID))
	req, err := s.client.NewRequest("GET", u, nil)
	if err != nil {
		return nil, err
	}

	pe := &PlanExport{}
	err = req.Do(ctx, pe)
	if err != nil {
		return nil, err
	}

	return pe, nil
}

// Delete a plan export by ID.
func (s *planExports) Delete(ctx context.Context, planExportID string) error {
	if !validStringID(&planExportID) {
		return ErrInvalidPlanExportID
	}

	u := fmt.Sprintf("plan-exports/%s", url.PathEscape(planExportID))
	req, err := s.client.NewRequest("DELETE", u, nil)
	if err != nil {
		return err
	}

	return req.Do(ctx, nil)
}

// Download a plan export's data. Data is exported in a .tar.gz format.
func (s *planExports) Download(ctx context.Context, planExportID string) ([]byte, error) {
	if !validStringID(&planExportID) {
		return nil, ErrInvalidPlanExportID
	}

	u := fmt.Sprintf("plan-exports/%s/download", url.PathEscape(planExportID))
	req, err := s.client.NewRequest("GET", u, nil)
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	err = req.Do(ctx, &buf)
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func (o PlanExportCreateOptions) valid() error {
	if o.Plan == nil {
		return ErrRequiredPlan
	}
	if o.DataType == nil {
		return ErrRequiredDataType
	}
	return nil
}

// Compile-time proof of interface implementation.
var _ Plans = (*plans)(nil)

// Plans describes all the plan related methods that the Terraform Enterprise
// API supports.
//
// TFE API docs: https://developer.hashicorp.com/terraform/cloud-docs/api-docs/plans
type Plans interface {
	// Read a plan by its ID.
	Read(ctx context.Context, planID string) (*Plan, error)

	// Logs retrieves the logs of a plan.
	Logs(ctx context.Context, planID string) (io.Reader, error)

	// Retrieve the JSON execution plan
	ReadJSONOutput(ctx context.Context, planID string) ([]byte, error)
}

// plans implements Plans.
type plans struct {
	client *Client
}

// PlanStatus represents a plan state.
type PlanStatus string

// List all available plan statuses.
const (
	PlanCanceled    PlanStatus = "canceled"
	PlanCreated     PlanStatus = "created"
	PlanErrored     PlanStatus = "errored"
	PlanFinished    PlanStatus = "finished"
	PlanMFAWaiting  PlanStatus = "mfa_waiting"
	PlanPending     PlanStatus = "pending"
	PlanQueued      PlanStatus = "queued"
	PlanRunning     PlanStatus = "running"
	PlanUnreachable PlanStatus = "unreachable"
)

// Plan represents a Terraform Enterprise plan.
type Plan struct {
	ID                     string                `jsonapi:"primary,plans"`
	HasChanges             bool                  `jsonapi:"attr,has-changes"`
	GeneratedConfiguration bool                  `jsonapi:"attr,generated-configuration"`
	LogReadURL             string                `jsonapi:"attr,log-read-url"`
	ResourceAdditions      int                   `jsonapi:"attr,resource-additions"`
	ResourceChanges        int                   `jsonapi:"attr,resource-changes"`
	ResourceDestructions   int                   `jsonapi:"attr,resource-destructions"`
	ResourceImports        int                   `jsonapi:"attr,resource-imports"`
	Status                 PlanStatus            `jsonapi:"attr,status"`
	StatusTimestamps       *PlanStatusTimestamps `jsonapi:"attr,status-timestamps"`

	// Relations
	Exports              []*PlanExport         `jsonapi:"relation,exports"`
	HYOKEncryptedDataKey *HYOKEncryptedDataKey `jsonapi:"relation,hyok-encrypted-data-key"`

	// Links
	Links map[string]interface{} `jsonapi:"links,omitempty"`
}

// PlanStatusTimestamps holds the timestamps for individual plan statuses.
type PlanStatusTimestamps struct {
	CanceledAt      time.Time `jsonapi:"attr,canceled-at,rfc3339"`
	ErroredAt       time.Time `jsonapi:"attr,errored-at,rfc3339"`
	FinishedAt      time.Time `jsonapi:"attr,finished-at,rfc3339"`
	ForceCanceledAt time.Time `jsonapi:"attr,force-canceled-at,rfc3339"`
	QueuedAt        time.Time `jsonapi:"attr,queued-at,rfc3339"`
	StartedAt       time.Time `jsonapi:"attr,started-at,rfc3339"`
}

// Read a plan by its ID.
func (s *plans) Read(ctx context.Context, planID string) (*Plan, error) {
	if !validStringID(&planID) {
		return nil, ErrInvalidPlanID
	}

	u := fmt.Sprintf("plans/%s", url.PathEscape(planID))
	req, err := s.client.NewRequest("GET", u, nil)
	if err != nil {
		return nil, err
	}

	p := &Plan{}
	err = req.Do(ctx, p)
	if err != nil {
		return nil, err
	}

	return p, nil
}

// Logs retrieves the logs of a plan.
func (s *plans) Logs(ctx context.Context, planID string) (io.Reader, error) {
	if !validStringID(&planID) {
		return nil, ErrInvalidPlanID
	}

	// Get the plan to make sure it exists.
	p, err := s.Read(ctx, planID)
	if err != nil {
		return nil, err
	}

	// Return an error if the log URL is empty.
	if p.LogReadURL == "" {
		return nil, fmt.Errorf("plan %s does not have a log URL", planID)
	}

	u, err := url.Parse(p.LogReadURL)
	if err != nil {
		return nil, fmt.Errorf("invalid log URL: %w", err)
	}

	done := func() (bool, error) {
		p, err := s.Read(ctx, p.ID)
		if err != nil {
			return false, err
		}

		switch p.Status {
		case PlanCanceled, PlanErrored, PlanFinished, PlanUnreachable:
			return true, nil
		default:
			return false, nil
		}
	}

	return &LogReader{
		client: s.client,
		ctx:    ctx,
		done:   done,
		logURL: u,
	}, nil
}

// Retrieve the JSON execution plan
func (s *plans) ReadJSONOutput(ctx context.Context, planID string) ([]byte, error) {
	if !validStringID(&planID) {
		return nil, ErrInvalidPlanID
	}

	u := fmt.Sprintf("plans/%s/json-output", url.PathEscape(planID))
	req, err := s.client.NewRequest("GET", u, nil)
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	err = req.Do(ctx, &buf)
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// Compile-time proof of interface implementation.
var _ PolicyChecks = (*policyChecks)(nil)

// PolicyChecks describes all the policy check related methods that the
// Terraform Enterprise API supports.
//
// TFE API docs:
// https://developer.hashicorp.com/terraform/cloud-docs/api-docs/policy-checks
type PolicyChecks interface {
	// List all policy checks of the given run.
	List(ctx context.Context, runID string, options *PolicyCheckListOptions) (*PolicyCheckList, error)

	// Read a policy check by its ID.
	Read(ctx context.Context, policyCheckID string) (*PolicyCheck, error)

	// Override a soft-mandatory or warning policy.
	Override(ctx context.Context, policyCheckID string) (*PolicyCheck, error)

	// Logs retrieves the logs of a policy check.
	Logs(ctx context.Context, policyCheckID string) (io.Reader, error)
}

// policyChecks implements PolicyChecks.
type policyChecks struct {
	client *Client
}

// PolicyScope represents a policy scope.
type PolicyScope string

// List all available policy scopes.
const (
	PolicyScopeOrganization PolicyScope = "organization"
	PolicyScopeWorkspace    PolicyScope = "workspace"
)

// PolicyStatus represents a policy check state.
type PolicyStatus string

// List all available policy check statuses.
const (
	PolicyCanceled    PolicyStatus = "canceled"
	PolicyErrored     PolicyStatus = "errored"
	PolicyHardFailed  PolicyStatus = "hard_failed"
	PolicyOverridden  PolicyStatus = "overridden"
	PolicyPasses      PolicyStatus = "passed"
	PolicyPending     PolicyStatus = "pending"
	PolicyQueued      PolicyStatus = "queued"
	PolicySoftFailed  PolicyStatus = "soft_failed"
	PolicyUnreachable PolicyStatus = "unreachable"
)

// PolicyCheckList represents a list of policy checks.
type PolicyCheckList struct {
	*Pagination
	Items []*PolicyCheck
}

// PolicyCheck represents a Terraform Enterprise policy check..
type PolicyCheck struct {
	ID               string                  `jsonapi:"primary,policy-checks"`
	Actions          *PolicyActions          `jsonapi:"attr,actions"`
	Permissions      *PolicyPermissions      `jsonapi:"attr,permissions"`
	Result           *PolicyResult           `jsonapi:"attr,result"`
	Scope            PolicyScope             `jsonapi:"attr,scope"`
	Status           PolicyStatus            `jsonapi:"attr,status"`
	StatusTimestamps *PolicyStatusTimestamps `jsonapi:"attr,status-timestamps"`
	Run              *Run                    `jsonapi:"relation,run"`
}

// PolicyActions represents the policy check actions.
type PolicyActions struct {
	IsOverridable bool `jsonapi:"attr,is-overridable"`
}

// PolicyPermissions represents the policy check permissions.
type PolicyPermissions struct {
	CanOverride bool `jsonapi:"attr,can-override"`
}

// PolicyResult represents the complete policy check result,
type PolicyResult struct {
	AdvisoryFailed int  `jsonapi:"attr,advisory-failed"`
	Duration       int  `jsonapi:"attr,duration"`
	HardFailed     int  `jsonapi:"attr,hard-failed"`
	Passed         int  `jsonapi:"attr,passed"`
	Result         bool `jsonapi:"attr,result"`
	SoftFailed     int  `jsonapi:"attr,soft-failed"`
	TotalFailed    int  `jsonapi:"attr,total-failed"`
	Sentinel       any  `jsonapi:"attr,sentinel"`
}

// PolicyStatusTimestamps holds the timestamps for individual policy check
// statuses.
type PolicyStatusTimestamps struct {
	ErroredAt    time.Time `jsonapi:"attr,errored-at,rfc3339"`
	HardFailedAt time.Time `jsonapi:"attr,hard-failed-at,rfc3339"`
	PassedAt     time.Time `jsonapi:"attr,passed-at,rfc3339"`
	QueuedAt     time.Time `jsonapi:"attr,queued-at,rfc3339"`
	SoftFailedAt time.Time `jsonapi:"attr,soft-failed-at,rfc3339"`
}

// A list of relations to include
// https://developer.hashicorp.com/terraform/cloud-docs/api-docs/policy-checks#available-related-resources
type PolicyCheckIncludeOpt string

const (
	PolicyCheckRunWorkspace PolicyCheckIncludeOpt = "run.workspace"
	PolicyCheckRun          PolicyCheckIncludeOpt = "run"
)

// PolicyCheckListOptions represents the options for listing policy checks.
type PolicyCheckListOptions struct {
	ListOptions

	// Optional: A list of relations to include. See available resources
	// https://developer.hashicorp.com/terraform/cloud-docs/api-docs/policy-checks#available-related-resources
	Include []PolicyCheckIncludeOpt `url:"include,omitempty"`
}

// List all policy checks of the given run.
func (s *policyChecks) List(ctx context.Context, runID string, options *PolicyCheckListOptions) (*PolicyCheckList, error) {
	if !validStringID(&runID) {
		return nil, ErrInvalidRunID
	}
	if err := options.valid(); err != nil {
		return nil, err
	}

	u := fmt.Sprintf("runs/%s/policy-checks", url.PathEscape(runID))
	req, err := s.client.NewRequest("GET", u, options)
	if err != nil {
		return nil, err
	}

	pcl := &PolicyCheckList{}
	err = req.Do(ctx, pcl)
	if err != nil {
		return nil, err
	}

	return pcl, nil
}

// Read a policy check by its ID.
func (s *policyChecks) Read(ctx context.Context, policyCheckID string) (*PolicyCheck, error) {
	if !validStringID(&policyCheckID) {
		return nil, ErrInvalidPolicyCheckID
	}

	u := fmt.Sprintf("policy-checks/%s", url.PathEscape(policyCheckID))
	req, err := s.client.NewRequest("GET", u, nil)
	if err != nil {
		return nil, err
	}

	pc := &PolicyCheck{}
	err = req.Do(ctx, pc)
	if err != nil {
		return nil, err
	}

	return pc, nil
}

// Override a soft-mandatory or warning policy.
func (s *policyChecks) Override(ctx context.Context, policyCheckID string) (*PolicyCheck, error) {
	if !validStringID(&policyCheckID) {
		return nil, ErrInvalidPolicyCheckID
	}

	u := fmt.Sprintf("policy-checks/%s/actions/override", url.PathEscape(policyCheckID))
	req, err := s.client.NewRequest("POST", u, nil)
	if err != nil {
		return nil, err
	}

	pc := &PolicyCheck{}
	err = req.Do(ctx, pc)
	if err != nil {
		return nil, err
	}

	return pc, nil
}

// Logs retrieves the logs of a policy check.
func (s *policyChecks) Logs(ctx context.Context, policyCheckID string) (io.Reader, error) {
	if !validStringID(&policyCheckID) {
		return nil, ErrInvalidPolicyCheckID
	}

	// Loop until the context is canceled or the policy check is finished
	// running. The policy check logs are not streamed and so only available
	// once the check is finished.
	for {
		pc, err := s.Read(ctx, policyCheckID)
		if err != nil {
			return nil, err
		}

		switch pc.Status {
		case PolicyPending, PolicyQueued:
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(500 * time.Millisecond):
				continue
			}
		}

		u := fmt.Sprintf("policy-checks/%s/output", url.PathEscape(policyCheckID))
		req, err := s.client.NewRequest("GET", u, nil)
		if err != nil {
			return nil, err
		}

		logs := bytes.NewBuffer(nil)
		err = req.Do(ctx, logs)
		if err != nil {
			return nil, err
		}

		return logs, nil
	}
}

func (o *PolicyCheckListOptions) valid() error {
	return nil
}

// Copyright IBM Corp. 2018, 2026
// SPDX-License-Identifier: MPL-2.0

// Compile-time proof of interface implementation.
var _ PolicyEvaluations = (*policyEvaluation)(nil)

// PolicyEvaluationStatus is an enum that represents all possible statuses for a policy evaluation
type PolicyEvaluationStatus string

const (
	PolicyEvaluationPassed      PolicyEvaluationStatus = "passed"
	PolicyEvaluationFailed      PolicyEvaluationStatus = "failed"
	PolicyEvaluationPending     PolicyEvaluationStatus = "pending"
	PolicyEvaluationRunning     PolicyEvaluationStatus = "running"
	PolicyEvaluationUnreachable PolicyEvaluationStatus = "unreachable"
	PolicyEvaluationOverridden  PolicyEvaluationStatus = "overridden"
	PolicyEvaluationCanceled    PolicyEvaluationStatus = "canceled"
	PolicyEvaluationErrored     PolicyEvaluationStatus = "errored"
)

// PolicyResultCount represents the count of the policy results
type PolicyResultCount struct {
	AdvisoryFailed  int `jsonapi:"attr,advisory-failed"`
	MandatoryFailed int `jsonapi:"attr,mandatory-failed"`
	Passed          int `jsonapi:"attr,passed"`
	Errored         int `jsonapi:"attr,errored"`
}

// The task stage the policy evaluation belongs to
type PolicyAttachable struct {
	ID   string `jsonapi:"attr,id"`
	Type string `jsonapi:"attr,type"`
}

// PolicyEvaluationStatusTimestamps represents the set of timestamps recorded for a policy evaluation
type PolicyEvaluationStatusTimestamps struct {
	ErroredAt  time.Time `jsonapi:"attr,errored-at,rfc3339"`
	RunningAt  time.Time `jsonapi:"attr,running-at,rfc3339"`
	CanceledAt time.Time `jsonapi:"attr,canceled-at,rfc3339"`
	FailedAt   time.Time `jsonapi:"attr,failed-at,rfc3339"`
	PassedAt   time.Time `jsonapi:"attr,passed-at,rfc3339"`
}

// PolicyEvaluation represents the policy evaluations that are part of the task stage.
type PolicyEvaluation struct {
	ID               string                           `jsonapi:"primary,policy-evaluations"`
	Status           PolicyEvaluationStatus           `jsonapi:"attr,status"`
	PolicyKind       PolicyKind                       `jsonapi:"attr,policy-kind"`
	StatusTimestamps PolicyEvaluationStatusTimestamps `jsonapi:"attr,status-timestamps"`
	ResultCount      *PolicyResultCount               `jsonapi:"attr,result-count"`
	CreatedAt        time.Time                        `jsonapi:"attr,created-at,iso8601"`
	UpdatedAt        time.Time                        `jsonapi:"attr,updated-at,iso8601"`

	// The task stage this evaluation belongs to
	TaskStage *PolicyAttachable `jsonapi:"relation,policy-attachable"`
}

// PolicyEvalutations describes all the policy evaluation related methods that the
// Terraform Enterprise API supports.
//
// TFE API docs:
// https://developer.hashicorp.com/terraform/cloud-docs/api-docs/policy-checks
type PolicyEvaluations interface {
	// **Note: This method is still in BETA and subject to change.**
	// List all policy evaluations in the task stage. Only available for OPA policies.
	List(ctx context.Context, taskStageID string, options *PolicyEvaluationListOptions) (*PolicyEvaluationList, error)
}

// policyEvaluation implements PolicyEvaluations.
type policyEvaluation struct {
	client *Client
}

// PolicyEvaluationListOptions represents the options for listing policy evaluations.
type PolicyEvaluationListOptions struct {
	ListOptions
}

// PolicyEvaluationList represents a list of policy evaluation.
type PolicyEvaluationList struct {
	*Pagination
	Items []*PolicyEvaluation
}

// List all policy evaluations in a task stage.
func (s *policyEvaluation) List(ctx context.Context, taskStageID string, options *PolicyEvaluationListOptions) (*PolicyEvaluationList, error) {
	if !validStringID(&taskStageID) {
		return nil, ErrInvalidTaskStageID
	}

	u := fmt.Sprintf("task-stages/%s/policy-evaluations", url.PathEscape(taskStageID))
	req, err := s.client.NewRequest("GET", u, options)
	if err != nil {
		return nil, err
	}

	pcl := &PolicyEvaluationList{}
	err = req.Do(ctx, pcl)
	if err != nil {
		return nil, err
	}

	return pcl, nil
}

// Compile-time proof of interface implementation.
var _ PolicySetOutcomes = (*policySetOutcome)(nil)

// PolicySetOutcomes describes all the policy set outcome related methods that the
// Terraform Enterprise API supports.
//
// TFE API docs:
// https://developer.hashicorp.com/terraform/cloud-docs/api-docs/policy-checks
type PolicySetOutcomes interface {
	// **Note: This method is still in BETA and subject to change.**
	// List all policy set outcomes in the policy evaluation. Only available for OPA policies.
	List(ctx context.Context, policyEvaluationID string, options *PolicySetOutcomeListOptions) (*PolicySetOutcomeList, error)

	// **Note: This method is still in BETA and subject to change.**
	// Read a policy set outcome by its ID. Only available for OPA policies.
	Read(ctx context.Context, policySetOutcomeID string) (*PolicySetOutcome, error)
}

// policySetOutcome implements PolicySetOutcomes.
type policySetOutcome struct {
	client *Client
}

// PolicySetOutcomeListFilter represents the filters that are supported while listing a policy set outcome
type PolicySetOutcomeListFilter struct {
	// Optional: A status string used to filter the results.
	// Must be either "passed", "failed", or "errored".
	Status string

	// Optional: The enforcement level used to filter the results.
	// Must be either "advisory" or "mandatory".
	EnforcementLevel string
}

// PolicySetOutcomeListOptions represents the options for listing policy set outcomes.
type PolicySetOutcomeListOptions struct {
	*ListOptions

	// Optional: A filter map used to filter the results of the policy outcome.
	// You can use filter[n] to combine combinations of statuses and enforcement levels filters
	Filter map[string]PolicySetOutcomeListFilter
}

// PolicySetOutcomeList represents a list of policy set outcomes.
type PolicySetOutcomeList struct {
	*Pagination
	Items []*PolicySetOutcome
}

// OutcomeOutput represents a single print output entry from a policy outcome.
type OutcomeOutput struct {
	Print string `jsonapi:"attr,print"`
}

// Outcome represents the outcome of the individual policy
type Outcome struct {
	EnforcementLevel EnforcementLevel `jsonapi:"attr,enforcement_level"`
	Query            string           `jsonapi:"attr,query"`
	Status           string           `jsonapi:"attr,status"`
	PolicyName       string           `jsonapi:"attr,policy_name"`
	Description      string           `jsonapi:"attr,description"`
	Output           []OutcomeOutput  `jsonapi:"attr,output,omitempty"`
}

// PolicySetOutcome represents outcome of the policy set that are part of the policy evaluation
type PolicySetOutcome struct {
	ID                   string            `jsonapi:"primary,policy-set-outcomes"`
	Outcomes             []Outcome         `jsonapi:"attr,outcomes"`
	Error                string            `jsonapi:"attr,error"`
	Overridable          *bool             `jsonapi:"attr,overridable"`
	PolicySetName        string            `jsonapi:"attr,policy-set-name"`
	PolicySetDescription string            `jsonapi:"attr,policy-set-description"`
	ResultCount          PolicyResultCount `jsonapi:"attr,result_count"`

	// The policy evaluation that this outcome belongs to
	PolicyEvaluation *PolicyEvaluation `jsonapi:"relation,policy-evaluation"`
}

// List all policy set outcomes in a policy evaluation.
func (s *policySetOutcome) List(ctx context.Context, policyEvaluationID string, options *PolicySetOutcomeListOptions) (*PolicySetOutcomeList, error) {
	if !validStringID(&policyEvaluationID) {
		return nil, ErrInvalidPolicyEvaluationID
	}

	additionalQueryParams := options.buildQueryString()

	u := fmt.Sprintf("policy-evaluations/%s/policy-set-outcomes", url.QueryEscape(policyEvaluationID))

	var opts *ListOptions
	if options != nil && options.ListOptions != nil {
		opts = options.ListOptions
	}

	req, err := s.client.NewRequestWithAdditionalQueryParams("GET", u, opts, additionalQueryParams)
	if err != nil {
		return nil, err
	}

	psol := &PolicySetOutcomeList{}
	err = req.Do(ctx, psol)
	if err != nil {
		return nil, err
	}

	return psol, nil
}

// buildQueryString takes the PolicySetOutcomeListOptions and returns a filters map.
// This function is required due to the limitations of the current library,
// we cannot encode map of objects using the current library that is used by go-tfe: https://github.com/google/go-querystring/issues/7
func (opts *PolicySetOutcomeListOptions) buildQueryString() map[string][]string {
	result := make(map[string][]string)
	if opts == nil || opts.Filter == nil {
		return nil
	}
	for k, v := range opts.Filter {
		if v.Status != "" {
			newKey := fmt.Sprintf("filter[%s][status]", k)
			result[newKey] = append(result[newKey], v.Status)
		}
		if v.EnforcementLevel != "" {
			newKey := fmt.Sprintf("filter[%s][enforcement_level]", k)
			result[newKey] = append(result[newKey], v.EnforcementLevel)
		}
	}
	return result
}

// Read reads a policy set outcome by its ID
func (s *policySetOutcome) Read(ctx context.Context, policySetOutcomeID string) (*PolicySetOutcome, error) {
	if !validStringID(&policySetOutcomeID) {
		return nil, ErrInvalidPolicySetOutcomeID
	}

	u := fmt.Sprintf("policy-set-outcomes/%s", url.PathEscape(policySetOutcomeID))
	req, err := s.client.NewRequest("GET", u, nil)
	if err != nil {
		return nil, err
	}

	pso := &PolicySetOutcome{}
	err = req.Do(ctx, pso)
	if err != nil {
		return nil, err
	}

	return pso, err
}

// Compile-time proof of interface implementation.
var _ PolicySetParameters = (*policySetParameters)(nil)

// PolicySetParameters describes all the parameter related methods that the Terraform
// Enterprise API supports.
//
// TFE API docs: https://developer.hashicorp.com/terraform/cloud-docs/api-docs/policy-set-params
type PolicySetParameters interface {
	// List all the parameters associated with the given policy-set.
	List(ctx context.Context, policySetID string, options *PolicySetParameterListOptions) (*PolicySetParameterList, error)

	// Create is used to create a new parameter.
	Create(ctx context.Context, policySetID string, options PolicySetParameterCreateOptions) (*PolicySetParameter, error)

	// Read a parameter by its ID.
	Read(ctx context.Context, policySetID string, parameterID string) (*PolicySetParameter, error)

	// Update values of an existing parameter.
	Update(ctx context.Context, policySetID string, parameterID string, options PolicySetParameterUpdateOptions) (*PolicySetParameter, error)

	// Delete a parameter by its ID.
	Delete(ctx context.Context, policySetID string, parameterID string) error
}

// policySetParameters implements Parameters.
type policySetParameters struct {
	client *Client
}

// PolicySetParameterList represents a list of parameters.
type PolicySetParameterList struct {
	*Pagination
	Items []*PolicySetParameter
}

// PolicySetParameter represents a Policy Set parameter
type PolicySetParameter struct {
	ID        string       `jsonapi:"primary,vars"`
	Key       string       `jsonapi:"attr,key"`
	Value     string       `jsonapi:"attr,value"`
	Category  CategoryType `jsonapi:"attr,category"`
	Sensitive bool         `jsonapi:"attr,sensitive"`

	// Relations
	PolicySet *PolicySet `jsonapi:"relation,configurable"`
}

// PolicySetParameterListOptions represents the options for listing parameters.
type PolicySetParameterListOptions struct {
	ListOptions
}

// PolicySetParameterCreateOptions represents the options for creating a new parameter.
type PolicySetParameterCreateOptions struct {
	// Type is a public field utilized by JSON:API to
	// set the resource type via the field tag.
	// It is not a user-defined value and does not need to be set.
	// https://jsonapi.org/format/#crud-creating
	Type string `jsonapi:"primary,vars"`

	// Required: The name of the parameter.
	Key *string `jsonapi:"attr,key"`

	// Optional: The value of the parameter.
	Value *string `jsonapi:"attr,value,omitempty"`

	// Required: The Category of the parameter, should always be "policy-set"
	Category *CategoryType `jsonapi:"attr,category"`

	// Optional: Whether the value is sensitive.
	Sensitive *bool `jsonapi:"attr,sensitive,omitempty"`
}

// PolicySetParameterUpdateOptions represents the options for updating a parameter.
type PolicySetParameterUpdateOptions struct {
	// Type is a public field utilized by JSON:API to
	// set the resource type via the field tag.
	// It is not a user-defined value and does not need to be set.
	// https://jsonapi.org/format/#crud-creating
	Type string `jsonapi:"primary,vars"`

	// Optional: The name of the parameter.
	Key *string `jsonapi:"attr,key,omitempty"`

	// Optional: The value of the parameter.
	Value *string `jsonapi:"attr,value,omitempty"`

	// Optional: Whether the value is sensitive.
	Sensitive *bool `jsonapi:"attr,sensitive,omitempty"`
}

// List all the parameters associated with the given policy-set.
func (s *policySetParameters) List(ctx context.Context, policySetID string, options *PolicySetParameterListOptions) (*PolicySetParameterList, error) {
	if !validStringID(&policySetID) {
		return nil, ErrInvalidPolicySetID
	}

	u := fmt.Sprintf("policy-sets/%s/parameters", policySetID)
	req, err := s.client.NewRequest("GET", u, options)
	if err != nil {
		return nil, err
	}

	vl := &PolicySetParameterList{}
	err = req.Do(ctx, vl)
	if err != nil {
		return nil, err
	}

	return vl, nil
}

// Create is used to create a new parameter.
func (s *policySetParameters) Create(ctx context.Context, policySetID string, options PolicySetParameterCreateOptions) (*PolicySetParameter, error) {
	if !validStringID(&policySetID) {
		return nil, ErrInvalidPolicySetID
	}
	if err := options.valid(); err != nil {
		return nil, err
	}

	u := fmt.Sprintf("policy-sets/%s/parameters", url.PathEscape(policySetID))
	req, err := s.client.NewRequest("POST", u, &options)
	if err != nil {
		return nil, err
	}

	p := &PolicySetParameter{}
	err = req.Do(ctx, p)
	if err != nil {
		return nil, err
	}

	return p, nil
}

// Read a parameter by its ID.
func (s *policySetParameters) Read(ctx context.Context, policySetID, parameterID string) (*PolicySetParameter, error) {
	if !validStringID(&policySetID) {
		return nil, ErrInvalidPolicySetID
	}
	if !validStringID(&parameterID) {
		return nil, ErrInvalidParamID
	}

	u := fmt.Sprintf("policy-sets/%s/parameters/%s", url.PathEscape(policySetID), url.PathEscape(parameterID))
	req, err := s.client.NewRequest("GET", u, nil)
	if err != nil {
		return nil, err
	}

	p := &PolicySetParameter{}
	err = req.Do(ctx, p)
	if err != nil {
		return nil, err
	}

	return p, err
}

// Update values of an existing parameter.
func (s *policySetParameters) Update(ctx context.Context, policySetID, parameterID string, options PolicySetParameterUpdateOptions) (*PolicySetParameter, error) {
	if !validStringID(&policySetID) {
		return nil, ErrInvalidPolicySetID
	}
	if !validStringID(&parameterID) {
		return nil, ErrInvalidParamID
	}

	u := fmt.Sprintf("policy-sets/%s/parameters/%s", url.PathEscape(policySetID), url.PathEscape(parameterID))
	req, err := s.client.NewRequest("PATCH", u, &options)
	if err != nil {
		return nil, err
	}

	p := &PolicySetParameter{}
	err = req.Do(ctx, p)
	if err != nil {
		return nil, err
	}

	return p, nil
}

// Delete a parameter by its ID.
func (s *policySetParameters) Delete(ctx context.Context, policySetID, parameterID string) error {
	if !validStringID(&policySetID) {
		return ErrInvalidPolicySetID
	}
	if !validStringID(&parameterID) {
		return ErrInvalidParamID
	}

	u := fmt.Sprintf("policy-sets/%s/parameters/%s", url.PathEscape(policySetID), url.PathEscape(parameterID))
	req, err := s.client.NewRequest("DELETE", u, nil)
	if err != nil {
		return err
	}

	return req.Do(ctx, nil)
}

func (o PolicySetParameterCreateOptions) valid() error {
	if !validString(o.Key) {
		return ErrRequiredKey
	}
	if o.Category == nil {
		return ErrRequiredCategory
	}
	if *o.Category != CategoryPolicySet {
		return ErrInvalidCategory
	}
	return nil
}

// Compile-time proof of interface implementation.
var _ PolicySetVersions = (*policySetVersions)(nil)

// PolicySetVersions describes all the Policy Set Version related methods that the Terraform
// Enterprise API supports.
//
// TFE API docs: https://developer.hashicorp.com/terraform/cloud-docs/api-docs/policy-sets#create-a-policy-set-version
type PolicySetVersions interface {
	// Create is used to create a new Policy Set Version.
	Create(ctx context.Context, policySetID string) (*PolicySetVersion, error)

	// Read is used to read a Policy Set Version by its ID.
	Read(ctx context.Context, policySetVersionID string) (*PolicySetVersion, error)

	// Upload uploads policy files. It takes a Policy Set Version and a path
	// to the set of sentinel files, which will be packaged by hashicorp/go-slug
	// before being uploaded.
	Upload(ctx context.Context, psv PolicySetVersion, path string) error
}

// policySetVersions implements PolicySetVersions.
type policySetVersions struct {
	client *Client
}

// PolicySetVersionSource represents a source type of a policy set version.
type PolicySetVersionSource string

// List all available sources for a Policy Set Version.
const (
	PolicySetVersionSourceAPI       PolicySetVersionSource = "tfe-api"
	PolicySetVersionSourceADO       PolicySetVersionSource = "ado"
	PolicySetVersionSourceBitBucket PolicySetVersionSource = "bitbucket"
	PolicySetVersionSourceGitHub    PolicySetVersionSource = "github"
	PolicySetVersionSourceGitLab    PolicySetVersionSource = "gitlab"
)

// PolicySetVersionStatus represents a policy set version status.
type PolicySetVersionStatus string

// List all available policy set version statuses.
const (
	PolicySetVersionErrored    PolicySetVersionStatus = "errored"
	PolicySetVersionIngressing PolicySetVersionStatus = "ingressing"
	PolicySetVersionPending    PolicySetVersionStatus = "pending"
	PolicySetVersionReady      PolicySetVersionStatus = "ready"
)

// PolicySetVersionStatusTimestamps holds the timestamps for individual policy
// set version statuses.
type PolicySetVersionStatusTimestamps struct {
	PendingAt    time.Time `jsonapi:"attr,pending-at,rfc3339"`
	IngressingAt time.Time `jsonapi:"attr,ingressing-at,rfc3339"`
	ReadyAt      time.Time `jsonapi:"attr,ready-at,rfc3339"`
	ErroredAt    time.Time `jsonapi:"attr,errored-at,rfc3339"`
}

type PolicySetIngressAttributes struct {
	CommitSHA  string `jsonapi:"attr,commit-sha"`
	CommitURL  string `jsonapi:"attr,commit-url"`
	Identifier string `jsonapi:"attr,identifier"`
}

// PolicySetVersion represents a Terraform Enterprise Policy Set Version
type PolicySetVersion struct {
	ID                string                           `jsonapi:"primary,policy-set-versions"`
	Source            PolicySetVersionSource           `jsonapi:"attr,source"`
	Status            PolicySetVersionStatus           `jsonapi:"attr,status"`
	StatusTimestamps  PolicySetVersionStatusTimestamps `jsonapi:"attr,status-timestamps"`
	Error             string                           `jsonapi:"attr,error"`
	ErrorMessage      string                           `jsonapi:"attr,error-message"`
	CreatedAt         time.Time                        `jsonapi:"attr,created-at,iso8601"`
	UpdatedAt         time.Time                        `jsonapi:"attr,updated-at,iso8601"`
	IngressAttributes *PolicySetIngressAttributes      `jsonapi:"attr,ingress-attributes"`

	// Relations
	PolicySet *PolicySet `jsonapi:"relation,policy-set"`

	// Links
	Links map[string]interface{} `jsonapi:"links,omitempty"`
}

func (p PolicySetVersion) uploadURL() (string, error) {
	uploadURL, ok := p.Links["upload"].(string)
	if !ok {
		return uploadURL, fmt.Errorf("the Policy Set Version does not contain an upload link")
	}

	if uploadURL == "" {
		return uploadURL, fmt.Errorf("the Policy Set Version upload URL is empty")
	}

	return uploadURL, nil
}

// Create is used to create a new Policy Set Version.
func (p *policySetVersions) Create(ctx context.Context, policySetID string) (*PolicySetVersion, error) {
	if !validStringID(&policySetID) {
		return nil, ErrInvalidPolicySetID
	}

	u := fmt.Sprintf("policy-sets/%s/versions", url.PathEscape(policySetID))
	req, err := p.client.NewRequest("POST", u, nil)
	if err != nil {
		return nil, err
	}

	psv := &PolicySetVersion{}
	err = req.Do(ctx, psv)
	if err != nil {
		return nil, err
	}

	return psv, nil
}

// Read is used to read a Policy Set Version by its ID.
func (p *policySetVersions) Read(ctx context.Context, policySetVersionID string) (*PolicySetVersion, error) {
	if !validStringID(&policySetVersionID) {
		return nil, ErrInvalidPolicySetID
	}

	u := fmt.Sprintf("policy-set-versions/%s", url.PathEscape(policySetVersionID))
	req, err := p.client.NewRequest("GET", u, nil)
	if err != nil {
		return nil, err
	}

	psv := &PolicySetVersion{}
	err = req.Do(ctx, psv)
	if err != nil {
		return nil, err
	}

	return psv, nil
}

// Upload uploads policy files. It takes a Policy Set Version and a path
// to the set of sentinel files, which will be packaged by hashicorp/go-slug
// before being uploaded.
func (p *policySetVersions) Upload(ctx context.Context, psv PolicySetVersion, path string) error {
	uploadURL, err := psv.uploadURL()
	if err != nil {
		return err
	}

	body, err := packContents(path)
	if err != nil {
		return err
	}

	return p.client.doForeignPUTRequest(ctx, uploadURL, body)
}

// Copyright IBM Corp. 2018, 2026
// SPDX-License-Identifier: MPL-2.0

// Compile-time proof of interface implementation.
var _ PolicySets = (*policySets)(nil)

// PolicyKind is an indicator of the underlying technology that the policy or policy set supports.
// There are two kinds documented in the enum.
type PolicyKind string

const (
	OPA      PolicyKind = "opa"
	Sentinel PolicyKind = "sentinel"
	TFPolicy PolicyKind = "tfpolicy"
)

// PolicySets describes all the policy set related methods that the Terraform
// Enterprise API supports.
//
// TFE API docs: https://developer.hashicorp.com/terraform/cloud-docs/api-docs/policy-sets
type PolicySets interface {
	// List all the policy sets for a given organization.
	List(ctx context.Context, organization string, options *PolicySetListOptions) (*PolicySetList, error)

	// Create a policy set and associate it with an organization.
	Create(ctx context.Context, organization string, options PolicySetCreateOptions) (*PolicySet, error)

	// Read a policy set by its ID.
	Read(ctx context.Context, policySetID string) (*PolicySet, error)

	// ReadWithOptions reads a policy set by its ID using the options supplied.
	ReadWithOptions(ctx context.Context, policySetID string, options *PolicySetReadOptions) (*PolicySet, error)

	// Update an existing policy set.
	Update(ctx context.Context, policySetID string, options PolicySetUpdateOptions) (*PolicySet, error)

	// Add policies to a policy set. This function can only be used when
	// there is no VCS repository associated with the policy set.
	AddPolicies(ctx context.Context, policySetID string, options PolicySetAddPoliciesOptions) error

	// Remove policies from a policy set. This function can only be used
	// when there is no VCS repository associated with the policy set.
	RemovePolicies(ctx context.Context, policySetID string, options PolicySetRemovePoliciesOptions) error

	// Add workspaces to a policy set.
	AddWorkspaces(ctx context.Context, policySetID string, options PolicySetAddWorkspacesOptions) error

	// Remove workspaces from a policy set.
	RemoveWorkspaces(ctx context.Context, policySetID string, options PolicySetRemoveWorkspacesOptions) error

	// Add workspace exclusions to a policy set.
	AddWorkspaceExclusions(ctx context.Context, policySetID string, options PolicySetAddWorkspaceExclusionsOptions) error

	// Remove workspace exclusions from a policy set.
	RemoveWorkspaceExclusions(ctx context.Context, policySetID string, options PolicySetRemoveWorkspaceExclusionsOptions) error

	// Add projects to a policy set.
	AddProjects(ctx context.Context, policySetID string, options PolicySetAddProjectsOptions) error

	// Remove projects from a policy set.
	RemoveProjects(ctx context.Context, policySetID string, options PolicySetRemoveProjectsOptions) error

	// Add Project exclusions to a policy set.
	AddProjectExclusions(ctx context.Context, policySetID string, options PolicySetAddProjectExclusionsOptions) error

	// Remove project exclusions from a policy set.
	RemoveProjectExclusions(ctx context.Context, policySetID string, options PolicySetRemoveProjectExclusionsOptions) error

	// Delete a policy set by its ID.
	Delete(ctx context.Context, policyID string) error

	// BETA: AddTagSelectors adds tag selectors (i.e. tag inclusion / exclusion) to a policy set.
	AddTagSelectors(ctx context.Context, policySetID string, options PolicySetAddTagSelectorsOptions) error

	// BETA: RemoveTagSelectors removes tag selectors (i.e. tag inclusion / exclusion) from a policy set.
	RemoveTagSelectors(ctx context.Context, policySetID string, options PolicySetRemoveTagSelectorsOptions) error
}

// policySets implements PolicySets.
type policySets struct {
	client *Client
}

// PolicySetList represents a list of policy sets.
type PolicySetList struct {
	*Pagination
	Items []*PolicySet
}

// PolicySet represents a Terraform Enterprise policy set.
type PolicySet struct {
	ID           string     `jsonapi:"primary,policy-sets"`
	Name         string     `jsonapi:"attr,name"`
	Description  string     `jsonapi:"attr,description"`
	Kind         PolicyKind `jsonapi:"attr,kind"`
	Overridable  *bool      `jsonapi:"attr,overridable"`
	Global       bool       `jsonapi:"attr,global"`
	PoliciesPath string     `jsonapi:"attr,policies-path"`
	// **Note: This field is still in BETA and subject to change.**
	PolicyCount       int       `jsonapi:"attr,policy-count"`
	VCSRepo           *VCSRepo  `jsonapi:"attr,vcs-repo"`
	WorkspaceCount    int       `jsonapi:"attr,workspace-count"`
	ProjectCount      int       `jsonapi:"attr,project-count"`
	CreatedAt         time.Time `jsonapi:"attr,created-at,iso8601"`
	UpdatedAt         time.Time `jsonapi:"attr,updated-at,iso8601"`
	AgentEnabled      bool      `jsonapi:"attr,agent-enabled"`
	PolicyToolVersion string    `jsonapi:"attr,policy-tool-version"`

	PolicyUpdatePatterns []string `jsonapi:"attr,policy-update-patterns"`
	// BETA: The tag selectors for this policy set.
	TagSelectors []*PolicySetTagSelectorAttr `jsonapi:"attr,tag-selectors"`

	// Relations
	// The organization to which the policy set belongs to.
	Organization *Organization `jsonapi:"relation,organization"`
	// The workspaces to which the policy set applies.
	Workspaces []*Workspace `jsonapi:"relation,workspaces"`
	// Individually managed policies which are associated with the policy set.
	Policies []*Policy `jsonapi:"relation,policies"`
	// The most recently created policy set version, regardless of status.
	// Note that this relationship may include an errored and unusable version,
	// and is intended to allow checking for errors.
	NewestVersion *PolicySetVersion `jsonapi:"relation,newest-version"`
	// The most recent successful policy set version.
	CurrentVersion *PolicySetVersion `jsonapi:"relation,current-version"`
	// The workspace exclusions to which the policy set applies.
	WorkspaceExclusions []*Workspace `jsonapi:"relation,workspace-exclusions"`
	// The projects to which the policy set applies.
	Projects []*Project `jsonapi:"relation,projects"`
	// The project exclusions to which the policy set applies.
	ProjectExclusions []*Project `jsonapi:"relation,project-exclusions"`
}

// PolicySetTagSelectorAttr represents a tag selector as returned by the read API.
type PolicySetTagSelectorAttr struct {
	Key       string  `jsonapi:"attr,tag-key"`
	Value     *string `jsonapi:"attr,tag-value"`
	IsExclude bool    `jsonapi:"attr,is-exclude"`
}

// PolicySetIncludeOpt represents the available options for include query params.
// https://developer.hashicorp.com/terraform/cloud-docs/api-docs/policy-sets#available-related-resources
type PolicySetIncludeOpt string

const (
	PolicySetPolicies            PolicySetIncludeOpt = "policies"
	PolicySetWorkspaces          PolicySetIncludeOpt = "workspaces"
	PolicySetCurrentVersion      PolicySetIncludeOpt = "current_version"
	PolicySetNewestVersion       PolicySetIncludeOpt = "newest_version"
	PolicySetProjects            PolicySetIncludeOpt = "projects"
	PolicySetWorkspaceExclusions PolicySetIncludeOpt = "workspace_exclusions"
	PolicySetProjectExclusions   PolicySetIncludeOpt = "project_exclusions"
)

// PolicySetListOptions represents the options for listing policy sets.
type PolicySetListOptions struct {
	ListOptions

	// Optional: A search string (partial policy set name) used to filter the results.
	Search string `url:"search[name],omitempty"`

	// Optional: A kind string used to filter the results by the policy set kind.
	Kind PolicyKind `url:"filter[kind],omitempty"`

	// Optional: A list of relations to include. See available resources
	// https://developer.hashicorp.com/terraform/cloud-docs/api-docs/policy-sets#available-related-resources
	Include []PolicySetIncludeOpt `url:"include,omitempty"`
}

// PolicySetReadOptions are read options.
// For a full list of relations, please see:
// https://developer.hashicorp.com/terraform/cloud-docs/api-docs/policy-sets#relationships
type PolicySetReadOptions struct {
	// Optional: A list of relations to include. See available resources
	// https://developer.hashicorp.com/terraform/cloud-docs/api-docs/policy-sets#available-related-resources
	Include []PolicySetIncludeOpt `url:"include,omitempty"`
}

// PolicySetCreateOptions represents the options for creating a new policy set.
type PolicySetCreateOptions struct {
	// Type is a public field utilized by JSON:API to
	// set the resource type via the field tag.
	// It is not a user-defined value and does not need to be set.
	// https://jsonapi.org/format/#crud-creating
	Type string `jsonapi:"primary,policy-sets"`

	// Required: The name of the policy set.
	Name *string `jsonapi:"attr,name"`

	// Optional: The description of the policy set.
	Description *string `jsonapi:"attr,description,omitempty"`

	// Optional: Whether or not the policy set is global.
	Global *bool `jsonapi:"attr,global,omitempty"`

	// Optional: The underlying technology that the policy set supports
	Kind PolicyKind `jsonapi:"attr,kind,omitempty"`

	// Optional: Whether or not users can override this policy when it fails during a run. Only valid for policy evaluations.
	// https://developer.hashicorp.com/terraform/cloud-docs/policy-enforcement/manage-policy-sets#policy-checks-versus-policy-evaluations
	Overridable *bool `jsonapi:"attr,overridable,omitempty"`

	// Optional: Whether or not the policy is run as an evaluation inside the agent.
	AgentEnabled *bool `jsonapi:"attr,agent-enabled,omitempty"`

	// Optional: The policy tool version to run the evaluation against.
	PolicyToolVersion *string `jsonapi:"attr,policy-tool-version,omitempty"`

	// Optional: A list of glob patterns that trigger policy set updates.
	PolicyUpdatePatterns []string `jsonapi:"attr,policy-update-patterns,omitempty"`

	// Optional: The sub-path within the attached VCS repository to ingress. All
	// files and directories outside of this sub-path will be ignored.
	// This option may only be specified when a VCS repo is present.
	PoliciesPath *string `jsonapi:"attr,policies-path,omitempty"`

	// Optional: The initial members of the policy set.
	Policies []*Policy `jsonapi:"relation,policies,omitempty"`

	// Optional: VCS repository information. When present, the policies and
	// configuration will be sourced from the specified VCS repository
	// instead of being defined within the policy set itself. Note that
	// this option is mutually exclusive with the Policies option and
	// both cannot be used at the same time.
	VCSRepo *VCSRepoOptions `jsonapi:"attr,vcs-repo,omitempty"`

	// Optional: The initial list of workspaces for which the policy set should be enforced.
	Workspaces []*Workspace `jsonapi:"relation,workspaces,omitempty"`

	// Optional: The initial list of workspace exclusions for which the policy set should be enforced.
	WorkspaceExclusions []*Workspace `jsonapi:"relation,workspace-exclusions,omitempty"`

	// Optional: The initial list of projects for which the policy set should be enforced.
	Projects []*Project `jsonapi:"relation,projects,omitempty"`

	// Optional: The initial list of project exclusions for which the policy set should be enforced.
	ProjectExclusions []*Project `jsonapi:"relation,project-exclusions,omitempty"`

	// BETA: Optional: A list of tag selectors for enforcement/exclusion based on tags
	TagSelectors []*PolicySetTagSelector `jsonapi:"attr,tag-selectors,omitempty"`
}

// PolicySetUpdateOptions represents the options for updating a policy set.
type PolicySetUpdateOptions struct {
	// Type is a public field utilized by JSON:API to
	// set the resource type via the field tag.
	// It is not a user-defined value and does not need to be set.
	// https://jsonapi.org/format/#crud-creating
	Type string `jsonapi:"primary,policy-sets"`

	// Optional: The name of the policy set.
	Name *string `jsonapi:"attr,name,omitempty"`

	// Optional: The description of the policy set.
	Description *string `jsonapi:"attr,description,omitempty"`

	// Optional: Whether or not the policy set is global.
	Global *bool `jsonapi:"attr,global,omitempty"`

	// Optional: Whether or not users can override this policy when it fails during a run. Only valid for policy evaluations.
	// https://developer.hashicorp.com/terraform/cloud-docs/policy-enforcement/manage-policy-sets#policy-checks-versus-policy-evaluations
	Overridable *bool `jsonapi:"attr,overridable,omitempty"`

	// Optional: Whether or not the policy is run as an evaluation inside the agent.
	AgentEnabled *bool `jsonapi:"attr,agent-enabled,omitempty"`

	// Optional: The policy tool version to run the evaluation against.
	PolicyToolVersion *string `jsonapi:"attr,policy-tool-version,omitempty"`

	// Optional: A list of glob patterns that trigger policy set updates.
	PolicyUpdatePatterns []string `jsonapi:"attr,policy-update-patterns,omitempty"`

	// Optional: The sub-path within the attached VCS repository to ingress. All
	// files and directories outside of this sub-path will be ignored.
	// This option may only be specified when a VCS repo is present.
	PoliciesPath *string `jsonapi:"attr,policies-path,omitempty"`

	// Optional: VCS repository information. When present, the policies and
	// configuration will be sourced from the specified VCS repository
	// instead of being defined within the policy set itself. Note that
	// specifying this option may only be used on policy sets with no
	// directly-attached policies (*PolicySet.Policies). Specifying this
	// option when policies are already present will result in an error.
	VCSRepo *VCSRepoOptions `jsonapi:"attr,vcs-repo,omitempty"`
}

// PolicySetAddPoliciesOptions represents the options for adding policies
// to a policy set.
type PolicySetAddPoliciesOptions struct {
	// The policies to add to the policy set.
	Policies []*Policy
}

// PolicySetRemovePoliciesOptions represents the options for removing
// policies from a policy set.
type PolicySetRemovePoliciesOptions struct {
	// The policies to remove from the policy set.
	Policies []*Policy
}

// PolicySetAddWorkspacesOptions represents the options for adding workspaces
// to a policy set.
type PolicySetAddWorkspacesOptions struct {
	// The workspaces to add to the policy set.
	Workspaces []*Workspace
}

// PolicySetRemoveWorkspacesOptions represents the options for removing
// workspaces from a policy set.
type PolicySetRemoveWorkspacesOptions struct {
	// The workspaces to remove from the policy set.
	Workspaces []*Workspace
}

// PolicySetAddWorkspaceExclusionsOptions represents the options for adding workspace exclusions to a policy set.
type PolicySetAddWorkspaceExclusionsOptions struct {
	// The workspaces to add to the policy set exclusion list.
	WorkspaceExclusions []*Workspace
}

// PolicySetRemoveWorkspaceExclusionsOptions represents the options for removing workspace exclusions from a policy set.
type PolicySetRemoveWorkspaceExclusionsOptions struct {
	// The workspaces to remove from the policy set exclusion list.
	WorkspaceExclusions []*Workspace
}

// PolicySetAddProjectExclusionsOptions represents the options for adding project exclusions to a policy set.
type PolicySetAddProjectExclusionsOptions struct {
	// The projects to add to the policy set exclusion list.
	ProjectExclusions []*Project
}

// PolicySetRemoveProjectExclusionsOptions represents the options for removing project exclusions from a policy set.
type PolicySetRemoveProjectExclusionsOptions struct {
	// The projects to remove from the policy set exclusion list.
	ProjectExclusions []*Project
}

// PolicySetAddProjectsOptions represents the options for adding projects
// to a policy set.
type PolicySetAddProjectsOptions struct {
	// The projects to add to the policy set.
	Projects []*Project
}

// PolicySetRemoveProjectsOptions represents the options for removing
// projects from a policy set.
type PolicySetRemoveProjectsOptions struct {
	// The projects to remove from the policy set.
	Projects []*Project
}

// PolicySetAddTagSelectorsOptions represents the options for adding
// tag selectors to a policy set.
type PolicySetAddTagSelectorsOptions struct {
	// The tag selectors to add to the policy set.
	TagSelectors []*PolicySetTagSelector
}

// PolicySetRemoveTagSelectorsOptions represents the options for removing
// tag selectors from a policy set.
type PolicySetRemoveTagSelectorsOptions struct {
	// The tag selectors to remove from the policy set.
	TagSelectors []*PolicySetTagSelector
}

// PolicySetTagSelectors represents a tag selector for a policy set.
// Tag selectors control whether a policy set applies to (includes) or
// is exempted from (excludes) workspaces that carry a matching tag.
// The IsExclude field determines the behavior: false means inclusion,
// true means exclusion. For tags that have only a key and no value,
// set Value to nil.
type PolicySetTagSelector struct {
	Key       string  `json:"tag-key"`
	Value     *string `json:"tag-value"`
	IsExclude bool    `json:"is-exclude"`
}

// policySetTagSelectorsRequest is the wire format for the tag-selectors
// POST and DELETE endpoints, which expect {"data": [...]}.
type policySetTagSelectorsRequest struct {
	Data []*PolicySetTagSelector `json:"data"`
}

// List all the policies for a given organization.
func (s *policySets) List(ctx context.Context, organization string, options *PolicySetListOptions) (*PolicySetList, error) {
	if !validStringID(&organization) {
		return nil, ErrInvalidOrg
	}

	u := fmt.Sprintf("organizations/%s/policy-sets", url.PathEscape(organization))
	req, err := s.client.NewRequest("GET", u, options)
	if err != nil {
		return nil, err
	}

	psl := &PolicySetList{}
	err = req.Do(ctx, psl)
	if err != nil {
		return nil, err
	}

	return psl, nil
}

// Create a policy set and associate it with an organization.
func (s *policySets) Create(ctx context.Context, organization string, options PolicySetCreateOptions) (*PolicySet, error) {
	if !validStringID(&organization) {
		return nil, ErrInvalidOrg
	}
	if err := options.valid(); err != nil {
		return nil, err
	}

	u := fmt.Sprintf("organizations/%s/policy-sets", url.PathEscape(organization))
	req, err := s.client.NewRequest("POST", u, &options)
	if err != nil {
		return nil, err
	}

	ps := &PolicySet{}
	err = req.Do(ctx, ps)
	if err != nil {
		return nil, err
	}

	return ps, err
}

// Read a policy set by its ID.
func (s *policySets) Read(ctx context.Context, policySetID string) (*PolicySet, error) {
	return s.ReadWithOptions(ctx, policySetID, nil)
}

// ReadWithOptions reads a policy by its ID using the options supplied.
func (s *policySets) ReadWithOptions(ctx context.Context, policySetID string, options *PolicySetReadOptions) (*PolicySet, error) {
	if !validStringID(&policySetID) {
		return nil, ErrInvalidPolicySetID
	}
	if err := options.valid(); err != nil {
		return nil, err
	}

	u := fmt.Sprintf("policy-sets/%s", url.PathEscape(policySetID))
	req, err := s.client.NewRequest("GET", u, options)
	if err != nil {
		return nil, err
	}

	ps := &PolicySet{}
	err = req.Do(ctx, ps)
	if err != nil {
		return nil, err
	}

	return ps, err
}

// Update an existing policy set.
func (s *policySets) Update(ctx context.Context, policySetID string, options PolicySetUpdateOptions) (*PolicySet, error) {
	if !validStringID(&policySetID) {
		return nil, ErrInvalidPolicySetID
	}
	if err := options.valid(); err != nil {
		return nil, err
	}

	u := fmt.Sprintf("policy-sets/%s", url.PathEscape(policySetID))
	req, err := s.client.NewRequest("PATCH", u, &options)
	if err != nil {
		return nil, err
	}

	ps := &PolicySet{}
	err = req.Do(ctx, ps)
	if err != nil {
		return nil, err
	}

	return ps, err
}

// AddPolicies adds policies to a policy set
func (s *policySets) AddPolicies(ctx context.Context, policySetID string, options PolicySetAddPoliciesOptions) error {
	if !validStringID(&policySetID) {
		return ErrInvalidPolicySetID
	}
	if err := options.valid(); err != nil {
		return err
	}

	u := fmt.Sprintf("policy-sets/%s/relationships/policies", url.PathEscape(policySetID))
	req, err := s.client.NewRequest("POST", u, options.Policies)
	if err != nil {
		return err
	}

	return req.Do(ctx, nil)
}

// RemovePolicies remove policies from a policy set
func (s *policySets) RemovePolicies(ctx context.Context, policySetID string, options PolicySetRemovePoliciesOptions) error {
	if !validStringID(&policySetID) {
		return ErrInvalidPolicySetID
	}
	if err := options.valid(); err != nil {
		return err
	}

	u := fmt.Sprintf("policy-sets/%s/relationships/policies", url.PathEscape(policySetID))
	req, err := s.client.NewRequest("DELETE", u, options.Policies)
	if err != nil {
		return err
	}

	return req.Do(ctx, nil)
}

// Addworkspaces adds workspaces to a policy set.
func (s *policySets) AddWorkspaces(ctx context.Context, policySetID string, options PolicySetAddWorkspacesOptions) error {
	if !validStringID(&policySetID) {
		return ErrInvalidPolicySetID
	}
	if err := options.valid(); err != nil {
		return err
	}

	u := fmt.Sprintf("policy-sets/%s/relationships/workspaces", url.PathEscape(policySetID))
	req, err := s.client.NewRequest("POST", u, options.Workspaces)
	if err != nil {
		return err
	}

	return req.Do(ctx, nil)
}

// RemoveWorkspaces removes workspaces from a policy set.
func (s *policySets) RemoveWorkspaces(ctx context.Context, policySetID string, options PolicySetRemoveWorkspacesOptions) error {
	if !validStringID(&policySetID) {
		return ErrInvalidPolicySetID
	}
	if err := options.valid(); err != nil {
		return err
	}

	u := fmt.Sprintf("policy-sets/%s/relationships/workspaces", url.PathEscape(policySetID))
	req, err := s.client.NewRequest("DELETE", u, options.Workspaces)
	if err != nil {
		return err
	}

	return req.Do(ctx, nil)
}

// AddWorkspaceExclusions adds workspace exclusions to a policy set.
func (s *policySets) AddWorkspaceExclusions(ctx context.Context, policySetID string, options PolicySetAddWorkspaceExclusionsOptions) error {
	if !validStringID(&policySetID) {
		return ErrInvalidPolicySetID
	}
	if err := options.valid(); err != nil {
		return err
	}

	u := fmt.Sprintf("policy-sets/%s/relationships/workspace-exclusions", url.PathEscape(policySetID))
	req, err := s.client.NewRequest("POST", u, options.WorkspaceExclusions)
	if err != nil {
		return err
	}

	return req.Do(ctx, nil)
}

// RemoveWorkspaceExclusions removes workspace exclusions from a policy set.
func (s *policySets) RemoveWorkspaceExclusions(ctx context.Context, policySetID string, options PolicySetRemoveWorkspaceExclusionsOptions) error {
	if !validStringID(&policySetID) {
		return ErrInvalidPolicySetID
	}
	if err := options.valid(); err != nil {
		return err
	}

	u := fmt.Sprintf("policy-sets/%s/relationships/workspace-exclusions", url.PathEscape(policySetID))
	req, err := s.client.NewRequest("DELETE", u, options.WorkspaceExclusions)
	if err != nil {
		return err
	}

	return req.Do(ctx, nil)
}

// AddProjects adds projects to a given policy set.
func (s *policySets) AddProjects(ctx context.Context, policySetID string, options PolicySetAddProjectsOptions) error {
	if !validStringID(&policySetID) {
		return ErrInvalidPolicySetID
	}
	if err := options.valid(); err != nil {
		return err
	}

	u := fmt.Sprintf("policy-sets/%s/relationships/projects", url.PathEscape(policySetID))
	req, err := s.client.NewRequest("POST", u, options.Projects)
	if err != nil {
		return err
	}

	return req.Do(ctx, nil)
}

// RemoveProjects removes projects from a policy set.
func (s *policySets) RemoveProjects(ctx context.Context, policySetID string, options PolicySetRemoveProjectsOptions) error {
	if !validStringID(&policySetID) {
		return ErrInvalidPolicySetID
	}
	if err := options.valid(); err != nil {
		return err
	}

	u := fmt.Sprintf("policy-sets/%s/relationships/projects", url.PathEscape(policySetID))
	req, err := s.client.NewRequest("DELETE", u, options.Projects)
	if err != nil {
		return err
	}

	return req.Do(ctx, nil)
}

// AddProjectExclusions adds project exclusions to a given policy set.
func (s *policySets) AddProjectExclusions(ctx context.Context, policySetID string, options PolicySetAddProjectExclusionsOptions) error {
	if !validStringID(&policySetID) {
		return ErrInvalidPolicySetID
	}

	if err := options.valid(); err != nil {
		return err
	}

	u := fmt.Sprintf("policy-sets/%s/relationships/project-exclusions", url.PathEscape(policySetID))
	req, err := s.client.NewRequest("POST", u, options.ProjectExclusions)
	if err != nil {
		return err
	}

	return req.Do(ctx, nil)
}

// RemoveProjectExclusions removes project exclusions to a given policy set.
func (s *policySets) RemoveProjectExclusions(ctx context.Context, policySetID string, options PolicySetRemoveProjectExclusionsOptions) error {
	if !validStringID(&policySetID) {
		return ErrInvalidPolicySetID
	}
	if err := options.valid(); err != nil {
		return err
	}

	u := fmt.Sprintf("policy-sets/%s/relationships/project-exclusions", url.PathEscape(policySetID))
	req, err := s.client.NewRequest("DELETE", u, options.ProjectExclusions)
	if err != nil {
		return err
	}

	return req.Do(ctx, nil)
}

// BETA: AddTagSelectors adds tag selectors to a policy set.
func (s *policySets) AddTagSelectors(ctx context.Context, policySetID string, options PolicySetAddTagSelectorsOptions) error {
	if !validStringID(&policySetID) {
		return ErrInvalidPolicySetID
	}
	if err := options.valid(); err != nil {
		return err
	}

	u := fmt.Sprintf("policy-sets/%s/tag-selectors", url.PathEscape(policySetID))
	req, err := s.client.NewRequest("POST", u, &policySetTagSelectorsRequest{Data: options.TagSelectors})
	if err != nil {
		return err
	}

	return req.Do(ctx, nil)
}

// BETA: RemoveTagSelectors removes tag selectors from a policy set.
func (s *policySets) RemoveTagSelectors(ctx context.Context, policySetID string, options PolicySetRemoveTagSelectorsOptions) error {
	if !validStringID(&policySetID) {
		return ErrInvalidPolicySetID
	}
	if err := options.valid(); err != nil {
		return err
	}

	u := fmt.Sprintf("policy-sets/%s/tag-selectors", url.PathEscape(policySetID))
	req, err := s.client.NewRequest("DELETE", u, &policySetTagSelectorsRequest{Data: options.TagSelectors})
	if err != nil {
		return err
	}

	return req.Do(ctx, nil)
}

// Delete a policy set by its ID.
func (s *policySets) Delete(ctx context.Context, policySetID string) error {
	if !validStringID(&policySetID) {
		return ErrInvalidPolicySetID
	}

	u := fmt.Sprintf("policy-sets/%s", url.PathEscape(policySetID))
	req, err := s.client.NewRequest("DELETE", u, nil)
	if err != nil {
		return err
	}

	return req.Do(ctx, nil)
}

func (o PolicySetCreateOptions) valid() error {
	if !validString(o.Name) {
		return ErrRequiredName
	}
	if !validStringID(o.Name) {
		return ErrInvalidName
	}
	return nil
}

func (o PolicySetRemoveWorkspacesOptions) valid() error {
	if o.Workspaces == nil {
		return ErrWorkspacesRequired
	}
	if len(o.Workspaces) == 0 {
		return ErrWorkspaceMinLimit
	}
	return nil
}

func (o PolicySetRemoveWorkspaceExclusionsOptions) valid() error {
	if o.WorkspaceExclusions == nil {
		return ErrWorkspacesRequired
	}
	if len(o.WorkspaceExclusions) == 0 {
		return ErrWorkspaceMinLimit
	}
	return nil
}

func (o PolicySetRemoveProjectsOptions) valid() error {
	if o.Projects == nil {
		return ErrRequiredProject
	}
	if len(o.Projects) == 0 {
		return ErrProjectMinLimit
	}
	return nil
}

func (o PolicySetUpdateOptions) valid() error {
	if o.Name != nil && !validStringID(o.Name) {
		return ErrInvalidName
	}
	return nil
}

func (o PolicySetAddPoliciesOptions) valid() error {
	if o.Policies == nil {
		return ErrRequiredPolicies
	}
	if len(o.Policies) == 0 {
		return ErrInvalidPolicies
	}
	return nil
}

func (o PolicySetRemovePoliciesOptions) valid() error {
	if o.Policies == nil {
		return ErrRequiredPolicies
	}
	if len(o.Policies) == 0 {
		return ErrInvalidPolicies
	}
	return nil
}

func (o PolicySetAddWorkspacesOptions) valid() error {
	if o.Workspaces == nil {
		return ErrWorkspacesRequired
	}
	if len(o.Workspaces) == 0 {
		return ErrWorkspaceMinLimit
	}
	return nil
}

func (o PolicySetAddWorkspaceExclusionsOptions) valid() error {
	if o.WorkspaceExclusions == nil {
		return ErrWorkspacesRequired
	}
	if len(o.WorkspaceExclusions) == 0 {
		return ErrWorkspaceMinLimit
	}
	return nil
}

func (o PolicySetAddProjectsOptions) valid() error {
	if o.Projects == nil {
		return ErrRequiredProject
	}
	if len(o.Projects) == 0 {
		return ErrProjectMinLimit
	}
	return nil
}

func (o PolicySetAddProjectExclusionsOptions) valid() error {
	if o.ProjectExclusions == nil {
		return ErrRequiredProject
	}
	if len(o.ProjectExclusions) == 0 {
		return ErrProjectMinLimit
	}
	return nil
}

func (o PolicySetRemoveProjectExclusionsOptions) valid() error {
	if o.ProjectExclusions == nil {
		return ErrRequiredProject
	}
	if len(o.ProjectExclusions) == 0 {
		return ErrProjectMinLimit
	}
	return nil
}

func (o PolicySetAddTagSelectorsOptions) valid() error {
	if o.TagSelectors == nil {
		return ErrRequiredTagSelectors
	}
	if len(o.TagSelectors) == 0 {
		return ErrTagSelectorMinLimit
	}
	return nil
}

func (o PolicySetRemoveTagSelectorsOptions) valid() error {
	if o.TagSelectors == nil {
		return ErrRequiredTagSelectors
	}
	if len(o.TagSelectors) == 0 {
		return ErrTagSelectorMinLimit
	}
	return nil
}

func (o *PolicySetReadOptions) valid() error {
	return nil
}

// Compile-time proof of interface implementation.
var _ Policies = (*policies)(nil)

// Policies describes all the policy related methods that the Terraform
// Enterprise API supports.
//
// TFE API docs: https://developer.hashicorp.com/terraform/cloud-docs/api-docs/policies
type Policies interface {
	// List all the policies for a given organization
	List(ctx context.Context, organization string, options *PolicyListOptions) (*PolicyList, error)

	// Create a policy and associate it with an organization.
	Create(ctx context.Context, organization string, options PolicyCreateOptions) (*Policy, error)

	// Read a policy by its ID.
	Read(ctx context.Context, policyID string) (*Policy, error)

	// Update an existing policy.
	Update(ctx context.Context, policyID string, options PolicyUpdateOptions) (*Policy, error)

	// Delete a policy by its ID.
	Delete(ctx context.Context, policyID string) error

	// Upload the policy content of the policy.
	Upload(ctx context.Context, policyID string, content []byte) error

	// Download the policy content of the policy.
	Download(ctx context.Context, policyID string) ([]byte, error)
}

// policies implements Policies.
type policies struct {
	client *Client
}

// EnforcementLevel represents an enforcement level.
type EnforcementLevel string

// List the available enforcement types.
const (
	EnforcementAdvisory  EnforcementLevel = "advisory"
	EnforcementHard      EnforcementLevel = "hard-mandatory"
	EnforcementSoft      EnforcementLevel = "soft-mandatory"
	EnforcementMandatory EnforcementLevel = "mandatory"
)

// PolicyList represents a list of policies..
type PolicyList struct {
	*Pagination
	Items []*Policy
}

// Policy represents a Terraform Enterprise policy.
type Policy struct {
	ID          string     `jsonapi:"primary,policies"`
	Name        string     `jsonapi:"attr,name"`
	Kind        PolicyKind `jsonapi:"attr,kind"`
	Query       *string    `jsonapi:"attr,query"`
	Description string     `jsonapi:"attr,description"`
	// Deprecated: Use EnforcementLevel instead.
	Enforce          []*Enforcement   `jsonapi:"attr,enforce"`
	EnforcementLevel EnforcementLevel `jsonapi:"attr,enforcement-level"`
	PolicySetCount   int              `jsonapi:"attr,policy-set-count"`
	UpdatedAt        time.Time        `jsonapi:"attr,updated-at,iso8601"`

	// Relations
	Organization *Organization `jsonapi:"relation,organization"`
}

// Enforcement describes a enforcement.
type Enforcement struct {
	Path string           `jsonapi:"attr,path"`
	Mode EnforcementLevel `jsonapi:"attr,mode"`
}

// EnforcementOptions represents the enforcement options of a policy.
type EnforcementOptions struct {
	Path *string           `json:"path"`
	Mode *EnforcementLevel `json:"mode"`
}

// PolicyListOptions represents the options for listing policies.
type PolicyListOptions struct {
	ListOptions

	// Optional: A search string (partial policy name) used to filter the results.
	Search string `url:"search[name],omitempty"`

	// Optional: A kind string used to filter the results by the policy kind.
	Kind PolicyKind `url:"filter[kind],omitempty"`
}

// PolicyCreateOptions represents the options for creating a new policy.
type PolicyCreateOptions struct {
	// Type is a public field utilized by JSON:API to
	// set the resource type via the field tag.
	// It is not a user-defined value and does not need to be set.
	// https://jsonapi.org/format/#crud-creating
	Type string `jsonapi:"primary,policies"`

	// Required: The name of the policy.
	Name *string `jsonapi:"attr,name"`

	// Optional: The underlying technology that the policy supports. Defaults to Sentinel if not specified for PolicyCreate.
	Kind PolicyKind `jsonapi:"attr,kind,omitempty"`

	// Optional: The query passed to policy evaluation to determine the result of the policy. Only valid for OPA.
	Query *string `jsonapi:"attr,query,omitempty"`

	// Optional: A description of the policy's purpose.
	Description *string `jsonapi:"attr,description,omitempty"`

	// The enforcements of the policy.
	//
	// Deprecated: Use EnforcementLevel instead.
	Enforce []*EnforcementOptions `jsonapi:"attr,enforce,omitempty"`

	// Required: The enforcement level of the policy.
	// Either EnforcementLevel or Enforce must be set.
	EnforcementLevel *EnforcementLevel `jsonapi:"attr,enforcement-level,omitempty"`
}

// PolicyUpdateOptions represents the options for updating a policy.
type PolicyUpdateOptions struct {
	// Type is a public field utilized by JSON:API to
	// set the resource type via the field tag.
	// It is not a user-defined value and does not need to be set.
	// https://jsonapi.org/format/#crud-creating
	Type string `jsonapi:"primary,policies"`

	// Optional: A description of the policy's purpose.
	Description *string `jsonapi:"attr,description,omitempty"`

	// Optional: The query passed to policy evaluation to determine the result of the policy. Only valid for OPA.
	Query *string `jsonapi:"attr,query,omitempty"`

	// Optional: The enforcements of the policy.
	//
	// Deprecated: Use EnforcementLevel instead.
	Enforce []*EnforcementOptions `jsonapi:"attr,enforce,omitempty"`

	// Optional: The enforcement level of the policy.
	EnforcementLevel *EnforcementLevel `jsonapi:"attr,enforcement-level,omitempty"`
}

// List all the policies for a given organization
func (s *policies) List(ctx context.Context, organization string, options *PolicyListOptions) (*PolicyList, error) {
	if !validStringID(&organization) {
		return nil, ErrInvalidOrg
	}

	u := fmt.Sprintf("organizations/%s/policies", url.PathEscape(organization))
	req, err := s.client.NewRequest("GET", u, options)
	if err != nil {
		return nil, err
	}

	pl := &PolicyList{}
	err = req.Do(ctx, pl)
	if err != nil {
		return nil, err
	}

	return pl, nil
}

// Create a policy and associate it with an organization.
func (s *policies) Create(ctx context.Context, organization string, options PolicyCreateOptions) (*Policy, error) {
	if !validStringID(&organization) {
		return nil, ErrInvalidOrg
	}
	if err := options.valid(); err != nil {
		return nil, err
	}

	u := fmt.Sprintf("organizations/%s/policies", url.PathEscape(organization))
	req, err := s.client.NewRequest("POST", u, &options)
	if err != nil {
		return nil, err
	}

	p := &Policy{}
	err = req.Do(ctx, p)
	if err != nil {
		return nil, err
	}

	return p, err
}

// Read a policy by its ID.
func (s *policies) Read(ctx context.Context, policyID string) (*Policy, error) {
	if !validStringID(&policyID) {
		return nil, ErrInvalidPolicyID
	}

	u := fmt.Sprintf("policies/%s", url.PathEscape(policyID))
	req, err := s.client.NewRequest("GET", u, nil)
	if err != nil {
		return nil, err
	}

	p := &Policy{}
	err = req.Do(ctx, p)
	if err != nil {
		return nil, err
	}

	return p, err
}

// Update an existing policy.
func (s *policies) Update(ctx context.Context, policyID string, options PolicyUpdateOptions) (*Policy, error) {
	if !validStringID(&policyID) {
		return nil, ErrInvalidPolicyID
	}

	u := fmt.Sprintf("policies/%s", url.PathEscape(policyID))
	req, err := s.client.NewRequest("PATCH", u, &options)
	if err != nil {
		return nil, err
	}

	p := &Policy{}
	err = req.Do(ctx, p)
	if err != nil {
		return nil, err
	}

	return p, err
}

// Delete a policy by its ID.
func (s *policies) Delete(ctx context.Context, policyID string) error {
	if !validStringID(&policyID) {
		return ErrInvalidPolicyID
	}

	u := fmt.Sprintf("policies/%s", url.PathEscape(policyID))
	req, err := s.client.NewRequest("DELETE", u, nil)
	if err != nil {
		return err
	}

	return req.Do(ctx, nil)
}

// Upload the policy content of the policy.
func (s *policies) Upload(ctx context.Context, policyID string, content []byte) error {
	if !validStringID(&policyID) {
		return ErrInvalidPolicyID
	}

	u := fmt.Sprintf("policies/%s/upload", url.PathEscape(policyID))
	req, err := s.client.NewRequest("PUT", u, content)
	if err != nil {
		return err
	}

	return req.Do(ctx, nil)
}

// Download the policy content of the policy.
func (s *policies) Download(ctx context.Context, policyID string) ([]byte, error) {
	if !validStringID(&policyID) {
		return nil, ErrInvalidPolicyID
	}

	u := fmt.Sprintf("policies/%s/download", url.PathEscape(policyID))
	req, err := s.client.NewRequest("GET", u, nil)
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	err = req.Do(ctx, &buf)
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func (o PolicyCreateOptions) valid() error {
	if !validString(o.Name) {
		return ErrRequiredName
	}
	if !validStringID(o.Name) {
		return ErrInvalidName
	}
	if o.Kind == OPA && !validString(o.Query) {
		return ErrRequiredQuery
	}
	if o.Enforce == nil && o.EnforcementLevel == nil {
		return ErrRequiredEnforce
	}
	if o.Enforce != nil && o.EnforcementLevel != nil {
		return ErrConflictingEnforceEnforcementLevel
	}
	if o.Enforce != nil {
		for _, e := range o.Enforce {
			if !validString(e.Path) {
				return ErrRequiredEnforcementPath
			}
			if e.Mode == nil {
				return ErrRequiredEnforcementMode
			}
		}
	}
	return nil
}

// Compile-time proof of interface implementation.
var _ Projects = (*projects)(nil)

// Projects describes all the project related methods that the Terraform
// Enterprise API supports
//
// TFE API docs: https://developer.hashicorp.com/terraform/cloud-docs/api-docs/projects
type Projects interface {
	// List all projects in the given organization
	List(ctx context.Context, organization string, options *ProjectListOptions) (*ProjectList, error)

	// Create a new project.
	Create(ctx context.Context, organization string, options ProjectCreateOptions) (*Project, error)

	// Read a project by its ID.
	Read(ctx context.Context, projectID string) (*Project, error)

	// ReadWithOptions a project by its ID.
	ReadWithOptions(ctx context.Context, projectID string, options ProjectReadOptions) (*Project, error)

	// Update a project.
	Update(ctx context.Context, projectID string, options ProjectUpdateOptions) (*Project, error)

	// Delete a project.
	Delete(ctx context.Context, projectID string) error

	// ListTagBindings lists all tag bindings associated with the project.
	ListTagBindings(ctx context.Context, projectID string) ([]*TagBinding, error)

	// ListEffectiveTagBindings lists all tag bindings associated with the project. In practice,
	// this should be the same as ListTagBindings since projects do not currently inherit
	// tag bindings.
	ListEffectiveTagBindings(ctx context.Context, workspaceID string) ([]*EffectiveTagBinding, error)

	// AddTagBindings adds or modifies the value of existing tag binding keys for a project.
	AddTagBindings(ctx context.Context, projectID string, options ProjectAddTagBindingsOptions) ([]*TagBinding, error)

	// DeleteAllTagBindings removes all existing tag bindings for a project.
	DeleteAllTagBindings(ctx context.Context, projectID string) error
}

// projects implements Projects
type projects struct {
	client *Client
}

// ProjectList represents a list of projects
type ProjectList struct {
	*Pagination
	Items []*Project
}

// Project represents a Terraform Enterprise project
type Project struct {
	ID                          string                       `jsonapi:"primary,projects"`
	AutoDestroyActivityDuration jsonapi.NullableAttr[string] `jsonapi:"attr,auto-destroy-activity-duration,omitempty"`
	DefaultExecutionMode        string                       `jsonapi:"attr,default-execution-mode"`
	Description                 string                       `jsonapi:"attr,description"`
	IsUnified                   bool                         `jsonapi:"attr,is-unified"`
	Name                        string                       `jsonapi:"attr,name"`
	SettingOverwrites           *ProjectSettingOverwrites    `jsonapi:"attr,setting-overwrites"`

	// Relations
	DefaultAgentPool     *AgentPool             `jsonapi:"relation,default-agent-pool"`
	EffectiveTagBindings []*EffectiveTagBinding `jsonapi:"relation,effective-tag-bindings"`
	Organization         *Organization          `jsonapi:"relation,organization"`
}

// Note: the fields of this struct are bool pointers instead of bool values, in order to simplify support for
// future TFE versions that support *some but not all* of the inherited defaults that go-tfe knows about.
type ProjectSettingOverwrites struct {
	ExecutionMode *bool `jsonapi:"attr,default-execution-mode"`
	AgentPool     *bool `jsonapi:"attr,default-agent-pool"`
}

type ProjectIncludeOpt string

const (
	ProjectEffectiveTagBindings ProjectIncludeOpt = "effective_tag_bindings"
)

// ProjectListOptions represents the options for listing projects
type ProjectListOptions struct {
	ListOptions

	// Optional: String (complete project name) used to filter the results.
	// If multiple, comma separated values are specified, projects matching
	// any of the names are returned.
	Name string `url:"filter[names],omitempty"`

	// Optional: A query string to search projects by names.
	Query string `url:"q,omitempty"`

	// Optional: A filter string to list projects filtered by key/value tags.
	// These are not annotated and therefore not encoded by go-querystring
	TagBindings []*TagBinding

	// Optional: A list of relations to include
	Include []ProjectIncludeOpt `url:"include,omitempty"`
}

type ProjectReadOptions struct {
	// Optional: A list of relations to include
	Include []ProjectIncludeOpt `url:"include,omitempty"`
}

// ProjectCreateOptions represents the options for creating a project
type ProjectCreateOptions struct {
	// Type is a public field utilized by JSON:API to
	// set the resource type via the field tag.
	// It is not a user-defined value and does not need to be set.
	// https://jsonapi.org/format/#crud-creating
	Type string `jsonapi:"primary,projects"`

	// Required: A name to identify the project.
	Name string `jsonapi:"attr,name"`

	// Optional: A description for the project.
	Description *string `jsonapi:"attr,description,omitempty"`

	// Associated TagBindings of the project.
	TagBindings []*TagBinding `jsonapi:"relation,tag-bindings,omitempty"`

	// Optional: For all workspaces in the project, the period of time to wait
	// after workspace activity to trigger a destroy run. The format should roughly
	// match a Go duration string limited to days and hours, e.g. "24h" or "1d".
	AutoDestroyActivityDuration jsonapi.NullableAttr[string] `jsonapi:"attr,auto-destroy-activity-duration,omitempty"`

	// Optional: DefaultExecutionMode the default execution mode for workspaces in the project
	DefaultExecutionMode *string `jsonapi:"attr,default-execution-mode,omitempty"`

	// Optional: DefaultAgentPoolID default agent pool for workspaces in the project,
	// required when DefaultExecutionMode is set to `agent`
	DefaultAgentPoolID *string `jsonapi:"attr,default-agent-pool-id,omitempty"`

	// Optional: Struct of booleans, which indicate whether the project
	// specifies its own values for various settings. If you mark a setting as
	// `false` in this struct, it will clear the project's existing value for
	// that setting and defer to the default value that its organization provides.
	//
	// In general, it's not necessary to mark a setting as `true` in this
	// struct; if you provide a literal value for a setting, HCP Terraform will
	// automatically update its overwrites field to `true`. If you do choose to
	// manually mark a setting as overwritten, you must provide a value for that
	// setting at the same time.
	SettingOverwrites *ProjectSettingOverwrites `jsonapi:"attr,setting-overwrites,omitempty"`
}

// ProjectUpdateOptions represents the options for updating a project
type ProjectUpdateOptions struct {
	// Type is a public field utilized by JSON:API to
	// set the resource type via the field tag.
	// It is not a user-defined value and does not need to be set.
	// https://jsonapi.org/format/#crud-creating
	Type string `jsonapi:"primary,projects"`

	// Optional: A name to identify the project
	Name *string `jsonapi:"attr,name,omitempty"`

	// Optional: A description for the project.
	Description *string `jsonapi:"attr,description,omitempty"`

	// Associated TagBindings of the project. Note that this will replace
	// all existing tag bindings.
	TagBindings []*TagBinding `jsonapi:"relation,tag-bindings,omitempty"`

	// Optional: For all workspaces in the project, the period of time to wait
	// after workspace activity to trigger a destroy run. The format should roughly
	// match a Go duration string limited to days and hours, e.g. "24h" or "1d".
	AutoDestroyActivityDuration jsonapi.NullableAttr[string] `jsonapi:"attr,auto-destroy-activity-duration,omitempty"`

	// Optional: DefaultExecutionMode the default execution mode for workspaces
	DefaultExecutionMode *string `jsonapi:"attr,default-execution-mode,omitempty"`

	// Optional: DefaultAgentPoolID default agent pool for workspaces in the project,
	// required when DefaultExecutionMode is set to `agent`
	DefaultAgentPoolID *string `jsonapi:"attr,default-agent-pool-id,omitempty"`

	// Optional: Struct of booleans, which indicate whether the project
	// specifies its own values for various settings. If you mark a setting as
	// `false` in this struct, it will clear the project's existing value for
	// that setting and defer to the default value that its organization provides.
	//
	// In general, it's not necessary to mark a setting as `true` in this
	// struct; if you provide a literal value for a setting, HCP Terraform will
	// automatically update its overwrites field to `true`. If you do choose to
	// manually mark a setting as overwritten, you must provide a value for that
	// setting at the same time.
	SettingOverwrites *ProjectSettingOverwrites `jsonapi:"attr,setting-overwrites,omitempty"`
}

// ProjectAddTagBindingsOptions represents the options for adding tag bindings
// to a project.
type ProjectAddTagBindingsOptions struct {
	TagBindings []*TagBinding
}

// List all projects.
func (s *projects) List(ctx context.Context, organization string, options *ProjectListOptions) (*ProjectList, error) {
	if !validStringID(&organization) {
		return nil, ErrInvalidOrg
	}

	var tagFilters map[string][]string
	if options != nil {
		tagFilters = encodeTagFiltersAsParams(options.TagBindings)
	}

	u := fmt.Sprintf("organizations/%s/projects", url.PathEscape(organization))
	req, err := s.client.NewRequestWithAdditionalQueryParams("GET", u, options, tagFilters)
	if err != nil {
		return nil, err
	}

	p := &ProjectList{}
	err = req.Do(ctx, p)
	if err != nil {
		return nil, err
	}

	return p, nil
}

// Create a project with the given options
func (s *projects) Create(ctx context.Context, organization string, options ProjectCreateOptions) (*Project, error) {
	if !validStringID(&organization) {
		return nil, ErrInvalidOrg
	}

	if err := options.valid(); err != nil {
		return nil, err
	}

	u := fmt.Sprintf("organizations/%s/projects", url.PathEscape(organization))
	req, err := s.client.NewRequest("POST", u, &options)
	if err != nil {
		return nil, err
	}

	p := &Project{}
	err = req.Do(ctx, p)
	if err != nil {
		return nil, err
	}

	return p, nil
}

// ReadWithOptions a project by its ID.
func (s *projects) ReadWithOptions(ctx context.Context, projectID string, options ProjectReadOptions) (*Project, error) {
	if !validStringID(&projectID) {
		return nil, ErrInvalidProjectID
	}

	u := fmt.Sprintf("projects/%s", url.PathEscape(projectID))
	req, err := s.client.NewRequest("GET", u, options)
	if err != nil {
		return nil, err
	}

	p := &Project{}
	err = req.Do(ctx, p)
	if err != nil {
		return nil, err
	}

	return p, nil
}

// Read a single project by its ID.
func (s *projects) Read(ctx context.Context, projectID string) (*Project, error) {
	if !validStringID(&projectID) {
		return nil, ErrInvalidProjectID
	}

	u := fmt.Sprintf("projects/%s", url.PathEscape(projectID))
	req, err := s.client.NewRequest("GET", u, nil)
	if err != nil {
		return nil, err
	}

	p := &Project{}
	err = req.Do(ctx, p)
	if err != nil {
		return nil, err
	}

	return p, nil
}

func (s *projects) ListTagBindings(ctx context.Context, projectID string) ([]*TagBinding, error) {
	if !validStringID(&projectID) {
		return nil, ErrInvalidProjectID
	}

	u := fmt.Sprintf("projects/%s/tag-bindings", url.PathEscape(projectID))
	req, err := s.client.NewRequest("GET", u, nil)
	if err != nil {
		return nil, err
	}

	var list struct {
		*Pagination
		Items []*TagBinding
	}

	err = req.Do(ctx, &list)
	if err != nil {
		return nil, err
	}

	return list.Items, nil
}

func (s *projects) ListEffectiveTagBindings(ctx context.Context, projectID string) ([]*EffectiveTagBinding, error) {
	if !validStringID(&projectID) {
		return nil, ErrInvalidProjectID
	}

	u := fmt.Sprintf("projects/%s/effective-tag-bindings", url.PathEscape(projectID))
	req, err := s.client.NewRequest("GET", u, nil)
	if err != nil {
		return nil, err
	}

	var list struct {
		*Pagination
		Items []*EffectiveTagBinding
	}

	err = req.Do(ctx, &list)
	if err != nil {
		return nil, err
	}

	return list.Items, nil
}

// AddTagBindings adds or modifies the value of existing tag binding keys for a project
func (s *projects) AddTagBindings(ctx context.Context, projectID string, options ProjectAddTagBindingsOptions) ([]*TagBinding, error) {
	if !validStringID(&projectID) {
		return nil, ErrInvalidProjectID
	}

	if err := options.valid(); err != nil {
		return nil, err
	}

	u := fmt.Sprintf("projects/%s/tag-bindings", url.PathEscape(projectID))
	req, err := s.client.NewRequest("PATCH", u, options.TagBindings)
	if err != nil {
		return nil, err
	}

	var response = struct {
		*Pagination
		Items []*TagBinding
	}{}
	err = req.Do(ctx, &response)

	return response.Items, err
}

// Update a project by its ID
func (s *projects) Update(ctx context.Context, projectID string, options ProjectUpdateOptions) (*Project, error) {
	if !validStringID(&projectID) {
		return nil, ErrInvalidProjectID
	}

	if err := options.valid(); err != nil {
		return nil, err
	}

	u := fmt.Sprintf("projects/%s", url.PathEscape(projectID))
	req, err := s.client.NewRequest("PATCH", u, &options)
	if err != nil {
		return nil, err
	}

	p := &Project{}
	err = req.Do(ctx, p)
	if err != nil {
		return nil, err
	}

	return p, nil
}

// Delete a project by its ID
func (s *projects) Delete(ctx context.Context, projectID string) error {
	if !validStringID(&projectID) {
		return ErrInvalidProjectID
	}

	u := fmt.Sprintf("projects/%s", url.PathEscape(projectID))
	req, err := s.client.NewRequest("DELETE", u, nil)
	if err != nil {
		return err
	}

	return req.Do(ctx, nil)
}

// Delete all tag bindings associated with a project.
func (s *projects) DeleteAllTagBindings(ctx context.Context, projectID string) error {
	if !validStringID(&projectID) {
		return ErrInvalidProjectID
	}

	type aliasOpts struct {
		Type        string        `jsonapi:"primary,projects"`
		TagBindings []*TagBinding `jsonapi:"relation,tag-bindings"`
	}

	opts := &aliasOpts{
		TagBindings: []*TagBinding{},
	}

	u := fmt.Sprintf("projects/%s", url.PathEscape(projectID))
	req, err := s.client.NewRequest("PATCH", u, opts)
	if err != nil {
		return err
	}

	return req.Do(ctx, nil)
}

func (o ProjectCreateOptions) valid() error {
	if !validString(&o.Name) {
		return ErrRequiredName
	}
	return nil
}

func (o ProjectUpdateOptions) valid() error {
	return nil
}

func (o ProjectAddTagBindingsOptions) valid() error {
	if len(o.TagBindings) == 0 {
		return ErrRequiredTagBindings
	}

	return nil
}

// Compile-time proof of interface implementation
var _ ProviderSets = (*providerSets)(nil)

// ProviderSets describes all the Provider Set related methods that the
// Terraform Enterprise API supports.
//
// Note: This API is experimental and intended for internal use only. It is
// subject to change or removal without notice.
type ProviderSets interface {
	// Create is used to create a new provider set.
	Create(ctx context.Context, organization string, options ProviderSetCreateOptions) (*ProviderSet, error)

	// Read a provider set by its ID.
	Read(ctx context.Context, providerSetID string) (*ProviderSet, error)

	// Read a provider set by its name.
	ReadByName(ctx context.Context, organization string, name string) (*ProviderSet, error)

	// Update values of an existing provider set.
	Update(ctx context.Context, providerSetID string, options ProviderSetUpdateOptions) (*ProviderSet, error)

	// Delete a provider set by its ID.
	Delete(ctx context.Context, providerSetID string) error
}

// ProviderSet represents a Terraform enterprise provider set.
//
// Note: This API is experimental and intended for internal use only. It is
// subject to change or removal without notice.
type ProviderSet struct {
	ID               string `jsonapi:"primary,provider-sets"`
	Name             string `jsonapi:"attr,name"`
	Description      string `jsonapi:"attr,description"`
	ProviderSource   string `jsonapi:"attr,provider-source"`
	ConfigurationHcl string `jsonapi:"attr,configuration-hcl"`
	Global           bool   `jsonapi:"attr,global"`

	// Relations
	Organization *Organization `jsonapi:"relation,organization"`
	Workspaces   []*Workspace  `jsonapi:"relation,workspaces,omitempty"`
	Projects     []*Project    `jsonapi:"relation,projects,omitempty"`
}

// providerSets implements ProviderSets.
type providerSets struct {
	client *Client
}

// ProviderSetCreateOptions represents the options for creating a new provider set.

// Note: This API is experimental and intended for internal use only. It is
// subject to change or removal without notice.
type ProviderSetCreateOptions struct {
	// Type is a public field utilized by JSON:API to
	// set the resource type via the field tag.
	// It is not a user-defined value and does not need to be set.
	// https://jsonapi.org/format/#crud-creating
	Type string `jsonapi:"primary,provider-sets"`

	// Required: Name of the provider set.
	Name string `jsonapi:"attr,name"`

	// Optional: Description to provide context for the provider set.
	Description *string `jsonapi:"attr,description,omitempty"`

	// Required: Provider source represents the source of the provider set.
	// (ie: "registry.terraform.io/hashicorp/aws")
	ProviderSource string `jsonapi:"attr,provider-source"`

	// Required: ConfigurationHcl represents the HCL configuration for the provider set.
	ConfigurationHcl string `jsonapi:"attr,configuration-hcl"`

	// Optional: If true the provider set is considered in all runs in the organization.
	Global *bool `jsonapi:"attr,global,omitempty"`

	// Optional: Workspaces are the workspaces assigned to the provider set.
	Workspaces []*Workspace `jsonapi:"relation,workspaces,omitempty"`
	// Optional: Projects are the projects assigned to the provider set.
	Projects []*Project `jsonapi:"relation,projects,omitempty"`
}

// ProviderSetUpdateOptions represents the options for updating a new provider set.

// Note: This API is experimental and intended for internal use only. It is
// subject to change or removal without notice.
type ProviderSetUpdateOptions struct {
	// Optional: Name of the provider set.
	Name *string

	// Optional: Description to provide context for the provider set.
	Description *string

	// Optional: Provider source represents the source of the provider set.
	// (ie: "registry.terraform.io/hashicorp/aws")
	ProviderSource *string

	// Optional: ConfigurationHcl represents the HCL configuration for the provider set.
	ConfigurationHcl *string

	// Optional: If true the provider set is considered in all runs in the organization.
	Global *bool

	// Optional: Workspaces are the workspaces assigned to the provider set. Providing
	// nil will be a NOP and empty array will remove all workspaces from the provider set.
	Workspaces []*Workspace
	// Optional: Projects are the projects assigned to the provider set. Providing
	// nil will be a NOP and empty array will remove all projects from the provider set.
	Projects []*Project
}

// These payload structs exist because partial updates need custom relationship encoding:
// omitted relationships must be left unchanged, while empty arrays must clear them.
// The generic JSON:API struct tags used elsewhere in go-tfe do not cleanly express
// that omitted-vs-empty distinction for this single PATCH request.

type providerSetUpdatePayload struct {
	Data providerSetUpdatePayloadData `json:"data"`
}

type providerSetUpdatePayloadData struct {
	Type          string                             `json:"type"`
	Attributes    providerSetUpdatePayloadAttributes `json:"attributes"`
	Relationships map[string]relationshipData        `json:"relationships"`
}

type providerSetUpdatePayloadAttributes struct {
	Name             *string `json:"name,omitempty"`
	Description      *string `json:"description,omitempty"`
	ProviderSource   *string `json:"provider_source,omitempty"`
	ConfigurationHcl *string `json:"configuration_hcl,omitempty"`
	Global           *bool   `json:"global,omitempty"`
}

func (o ProviderSetUpdateOptions) payload() *providerSetUpdatePayload {
	payload := providerSetUpdatePayload{
		Data: providerSetUpdatePayloadData{
			Type: "provider-sets",
			Attributes: providerSetUpdatePayloadAttributes{
				Name:             o.Name,
				Description:      o.Description,
				ProviderSource:   o.ProviderSource,
				ConfigurationHcl: o.ConfigurationHcl,
				Global:           o.Global,
			},
			Relationships: make(map[string]relationshipData),
		},
	}

	if o.Workspaces != nil {
		data := make([]relationshipItem, len(o.Workspaces))
		for i, ws := range o.Workspaces {
			data[i] = ws.relationshipItem()
		}
		payload.Data.Relationships["workspaces"] = relationshipData{Data: data}
	}

	if o.Projects != nil {
		data := make([]relationshipItem, len(o.Projects))
		for i, proj := range o.Projects {
			data[i] = proj.relationshipItem()
		}
		payload.Data.Relationships["projects"] = relationshipData{Data: data}
	}

	return &payload
}

// Create is used to create a new provider set.
//
// Note: This API is experimental and intended for internal use only. It is
// subject to change or removal without notice.
func (p *providerSets) Create(ctx context.Context, organization string, options ProviderSetCreateOptions) (*ProviderSet, error) {
	if !validStringID(&organization) {
		return nil, ErrInvalidOrg
	}
	if err := options.valid(); err != nil {
		return nil, err
	}

	u := fmt.Sprintf("organizations/%s/provider-sets", url.PathEscape(organization))
	req, err := p.client.NewRequest("POST", u, &options)
	if err != nil {
		return nil, err
	}

	ps := &ProviderSet{}
	err = req.Do(ctx, ps)
	if err != nil {
		return nil, err
	}

	return ps, nil
}

// Read a provider set by its ID.
//
// Note: This API is experimental and intended for internal use only. It is
// subject to change or removal without notice.
func (p *providerSets) Read(ctx context.Context, providerSetID string) (*ProviderSet, error) {
	if !validString(&providerSetID) {
		return nil, ErrRequiredProviderSetID
	}
	if !validStringID(&providerSetID) {
		return nil, ErrInvalidProviderSetID
	}

	u := fmt.Sprintf("provider-sets/%s", url.PathEscape(providerSetID))
	req, err := p.client.NewRequest("GET", u, nil)
	if err != nil {
		return nil, err
	}

	ps := &ProviderSet{}
	err = req.Do(ctx, ps)
	if err != nil {
		return nil, err
	}

	return ps, nil
}

// ReadByName is used to read a provider set by its name.
//
// Note: This API is experimental and intended for internal use only. It is
// subject to change or removal without notice.
func (p *providerSets) ReadByName(ctx context.Context, organization, name string) (*ProviderSet, error) {
	if !validStringID(&organization) {
		return nil, ErrInvalidOrg
	}
	if !validString(&name) {
		return nil, ErrRequiredName
	}
	if !validStringID(&name) {
		return nil, ErrInvalidName
	}

	u := fmt.Sprintf("organizations/%s/provider-sets/%s",
		url.PathEscape(organization), url.PathEscape(name))
	req, err := p.client.NewRequest("GET", u, nil)
	if err != nil {
		return nil, err
	}

	ps := &ProviderSet{}
	err = req.Do(ctx, ps)
	if err != nil {
		return nil, err
	}

	return ps, nil
}

// Update values of an existing provider set.
//
// Note: This API is experimental and intended for internal use only. It is
// subject to change or removal without notice.
func (p *providerSets) Update(ctx context.Context, providerSetID string, options ProviderSetUpdateOptions) (*ProviderSet, error) {
	if !validStringID(&providerSetID) {
		return nil, ErrInvalidProviderSetID
	}
	if err := options.valid(); err != nil {
		return nil, err
	}
	u := fmt.Sprintf("provider-sets/%s", url.PathEscape(providerSetID))
	req, err := p.client.NewRequest("PATCH", u, options.payload())
	if err != nil {
		return nil, err
	}

	ps := &ProviderSet{}
	err = req.Do(ctx, ps)
	if err != nil {
		return nil, err
	}

	return ps, nil
}

// Delete a provider set by its ID.
//
// Note: This API is experimental and intended for internal use only. It is
// subject to change or removal without notice.
func (p *providerSets) Delete(ctx context.Context, providerSetID string) error {
	if !validStringID(&providerSetID) {
		return ErrInvalidProviderSetID
	}

	u := fmt.Sprintf("provider-sets/%s", url.PathEscape(providerSetID))
	req, err := p.client.NewRequest("DELETE", u, nil)
	if err != nil {
		return err
	}

	return req.Do(ctx, nil)
}

func (o ProviderSetCreateOptions) valid() error {
	if !validString(&o.Name) {
		return ErrRequiredName
	}
	if !validStringID(&o.Name) {
		return ErrInvalidName
	}
	if !validString(&o.ProviderSource) {
		return ErrRequiredProviderSource
	}
	if !validString(&o.ConfigurationHcl) {
		return ErrRequiredConfigurationHcl
	}
	if o.Global != nil && *o.Global && (len(o.Workspaces) > 0 || len(o.Projects) > 0) {
		return ErrProviderSetGlobalRelationships
	}
	for _, w := range o.Workspaces {
		if !validString(&w.ID) {
			return ErrRequiredWorkspaceID
		}
		if !validStringID(&w.ID) {
			return ErrInvalidWorkspaceID
		}
	}
	for _, p := range o.Projects {
		if !validString(&p.ID) {
			return ErrRequiredProjectID
		}
		if !validStringID(&p.ID) {
			return ErrInvalidProjectID
		}
	}
	return nil
}

func (o ProviderSetUpdateOptions) valid() error {
	if o.Name != nil && !validStringID(o.Name) {
		return ErrInvalidName
	}
	if o.Global != nil && *o.Global && (len(o.Workspaces) > 0 || len(o.Projects) > 0) {
		return ErrProviderSetGlobalRelationships
	}
	for _, w := range o.Workspaces {
		if !validString(&w.ID) {
			return ErrRequiredWorkspaceID
		}
		if !validStringID(&w.ID) {
			return ErrInvalidWorkspaceID
		}
	}
	for _, p := range o.Projects {
		if !validString(&p.ID) {
			return ErrRequiredProjectID
		}
		if !validStringID(&p.ID) {
			return ErrInvalidProjectID
		}
	}
	return nil
}

// Copyright IBM Corp. 2018, 2026
// SPDX-License-Identifier: MPL-2.0

// Compile-time proof of interface implementation.
var _ QueryRuns = (*queryRuns)(nil)

// QueryRuns describes all the run related methods that the Terraform Enterprise
// API supports.
//
// TFE API docs:
// https://developer.hashicorp.com/terraform/cloud-docs/api-docs/query-runs
type QueryRuns interface {
	// List all the query runs of the given workspace.
	List(ctx context.Context, workspaceID string, options *QueryRunListOptions) (*QueryRunList, error)

	// Create a new query run with the given options.
	Create(ctx context.Context, options QueryRunCreateOptions) (*QueryRun, error)

	// Read a query run by its ID.
	Read(ctx context.Context, queryRunID string) (*QueryRun, error)

	// ReadWithOptions reads a query run by its ID using the options supplied
	ReadWithOptions(ctx context.Context, queryRunID string, options *QueryRunReadOptions) (*QueryRun, error)

	// Logs retrieves the logs of a query run.
	Logs(ctx context.Context, queryRunID string) (io.Reader, error)

	// Cancel a query run by its ID.
	Cancel(ctx context.Context, runID string) error
}

// QueryRunCreateOptions represents the options for creating a new run.
type QueryRunCreateOptions struct {
	// Type is a public field utilized by JSON:API to
	// set the resource type via the field tag.
	// It is not a user-defined value and does not need to be set.
	// https://jsonapi.org/format/#crud-creating
	Type string `jsonapi:"primary,queries"`

	// TerraformVersion specifies the Terraform version to use in this query run.
	TerraformVersion *string `jsonapi:"attr,terraform-version,omitempty"`

	Source QueryRunSource `jsonapi:"attr,source"`

	// Specifies the configuration version to use for this query run. If the
	// configuration version object is omitted, the run will be created using the
	// workspace's latest configuration version.
	ConfigurationVersion *ConfigurationVersion `jsonapi:"relation,configuration-version"`

	// Specifies the workspace where the query run will be executed.
	Workspace *Workspace `jsonapi:"relation,workspace"`

	// Variables allows you to specify terraform input variables for
	// a particular run, prioritized over variables defined on the workspace.
	Variables []*RunVariable `jsonapi:"attr,variables,omitempty"`

	// Specifies whether the Terraform query CLI execution passes the
	// -generate-config-out= flag. When set to true, Terraform generates resource configuration
	// output as a side effect of the query run. Defaults to true when omitted from the request.
	GenerateConfigOut *bool `jsonapi:"attr,generate-config-out,omitempty"`
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

// QueryRunStatus is the query run state
type QueryRunStatus string

// List all available run statuses.
const (
	QueryRunCanceled QueryRunStatus = "canceled"
	QueryRunErrored  QueryRunStatus = "errored"
	QueryRunPending  QueryRunStatus = "pending"
	QueryRunQueued   QueryRunStatus = "queued"
	QueryRunRunning  QueryRunStatus = "running"
	QueryRunFinished QueryRunStatus = "finished"
)

// List all available run sources.
const (
	QueryRunSourceAPI QueryRunSource = "tfe-api"
)

const (
	QueryRunCreatedBy QueryRunIncludeOpt = "created_by"
	QueryRunConfigVer QueryRunIncludeOpt = "configuration_version"
)

// queryRuns implements QueryRuns.
type queryRuns struct {
	client *Client
}

// QueryRunList represents a list of query runs.
type QueryRunList struct {
	*Pagination
	Items []*QueryRun
}

// QueryRunListOptions represents the options for listing query runs.
type QueryRunListOptions struct {
	ListOptions
	Include []QueryRunIncludeOpt `url:"include,omitempty"`
}

// QueryRunReadOptions represents the options for reading a query run.
type QueryRunReadOptions struct {
	Include []QueryRunIncludeOpt `url:"include,omitempty"`
}

// QueryRun represents a Terraform Enterprise query run.
type QueryRun struct {
	ID               string                    `jsonapi:"primary,queries"`
	CreatedAt        time.Time                 `jsonapi:"attr,created-at,iso8601"`
	Source           QueryRunSource            `jsonapi:"attr,source"`
	Status           QueryRunStatus            `jsonapi:"attr,status"`
	StatusTimestamps *QueryRunStatusTimestamps `jsonapi:"attr,status-timestamps"`
	TerraformVersion string                    `jsonapi:"attr,terraform-version"`
	Variables        []*RunVariableAttr        `jsonapi:"attr,variables"`
	// GenerateConfigOut indicates whether the Terraform query CLI execution passed the
	// -generate-config-out= flag during this run. When true, Terraform generated resource
	// configuration output as a side effect of the query run.
	GenerateConfigOut bool   `jsonapi:"attr,generate-config-out"`
	LogReadURL        string `jsonapi:"attr,log-read-url"`

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

func (r *queryRuns) Logs(ctx context.Context, queryRunID string) (io.Reader, error) {
	if !validStringID(&queryRunID) {
		return nil, ErrInvalidQueryRunID
	}

	// Get the query to make sure it exists.
	q, err := r.Read(ctx, queryRunID)
	if err != nil {
		return nil, err
	}

	// Return an error if the log URL is empty.
	if q.LogReadURL == "" {
		return nil, fmt.Errorf("query %s does not have a log URL", queryRunID)
	}

	u, err := url.Parse(q.LogReadURL)
	if err != nil {
		return nil, fmt.Errorf("invalid log URL: %w", err)
	}

	done := func() (bool, error) {
		p, err := r.Read(ctx, q.ID)
		if err != nil {
			return false, err
		}

		switch p.Status {
		case QueryRunCanceled, QueryRunErrored, QueryRunFinished:
			return true, nil
		default:
			return false, nil
		}
	}

	return &LogReader{
		client: r.client,
		ctx:    ctx,
		done:   done,
		logURL: u,
	}, nil
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

// Compile-time proof of interface implementation.
var _ RegistryComponents = (*registryComponents)(nil)

// RegistryComponents describes all the registry component-related methods that the Terraform
// Enterprise API supports.
//
// TFE API docs: https://developer.hashicorp.com/terraform/cloud-docs/api-docs/private-registry/stack-component-configurations
type RegistryComponents interface {
	// Create a registry component. Note that this function creates registry components via API-only workflow.
	Create(ctx context.Context, organization string, options RegistryComponentCreateOptions) (*RegistryComponent, error)

	// Update a registry component. Only tag bindings can be updated on a component, so the update options are limited to that field.
	Update(ctx context.Context, componentID string, options *RegistryComponentUpdateOptions) (*RegistryComponent, error)

	// ListTagBindings lists all tag bindings associated with the component.
	ListTagBindings(ctx context.Context, componentID string) ([]*TagBinding, error)

	// Delete a registry component.
	Delete(ctx context.Context, componentID string) error
}

// registryComponents implements RegistryComponents.
type registryComponents struct {
	client *Client
}

type RegistryComponentVersionStatuses struct {
	Version string `jsonapi:"attr,version"`
	Status  string `jsonapi:"attr,status"`
}

// RegistryComponent represents a registry component
type RegistryComponent struct {
	ID              string                             `jsonapi:"primary,registry-components"`
	Name            string                             `jsonapi:"attr,name"`
	Namespace       string                             `jsonapi:"attr,namespace"`
	Description     string                             `jsonapi:"attr,description"`
	Status          string                             `jsonapi:"attr,status"`
	VCSRepo         *VCSRepo                           `jsonapi:"attr,vcs-repo"`
	VersionStatuses []RegistryComponentVersionStatuses `jsonapi:"attr,version-statuses"`
	CreatedAt       string                             `jsonapi:"attr,created-at,iso8601"`
	UpdatedAt       string                             `jsonapi:"attr,updated-at,iso8601"`

	// Relations
	Organization *Organization `jsonapi:"relation,organization"`
	TagBindings  []*TagBinding `jsonapi:"relation,tag-bindings,omitempty"`

	// Links
	Links map[string]interface{} `jsonapi:"links,omitempty"`
}

// RegistryComponentUpdateOptions is used when updating a registry component config
type RegistryComponentUpdateOptions struct {
	// Optional: Tag bindings for the registry component. Note that this
	// will replace all existing tag bindings.
	TagBindings []*TagBinding `jsonapi:"relation,tag-bindings"`
}

// RegistryComponentCreateOptions is used when creating a registry component config via API-only workflow
type RegistryComponentCreateOptions struct {
	Type string `jsonapi:"primary,registry-components"`
	Name string `jsonapi:"attr,name"`
}

func (r *registryComponents) Create(ctx context.Context, organization string, options RegistryComponentCreateOptions) (*RegistryComponent, error) {
	if !validStringID(&organization) {
		return nil, ErrInvalidOrg
	}
	if (reflect.DeepEqual(options, RegistryComponentCreateOptions{})) {
		return nil, ErrRequiredRegistryComponentCreateOps
	}

	if !validStringID(&options.Name) {
		return nil, ErrInvalidName
	}

	u := fmt.Sprintf(
		"organizations/%s/registry-components",
		url.PathEscape(organization),
	)
	req, err := r.client.NewRequest("POST", u, &options)
	if err != nil {
		return nil, err
	}

	rc := &RegistryComponent{}
	err = req.Do(ctx, rc)
	if err != nil {
		return nil, err
	}

	return rc, nil
}

func (r *registryComponents) Update(ctx context.Context, componentID string, options *RegistryComponentUpdateOptions) (*RegistryComponent, error) {
	if !validStringID(&componentID) {
		return nil, ErrInvalidRegistryComponentID
	}

	u := fmt.Sprintf(
		"registry-components/%s",
		url.PathEscape(componentID),
	)
	req, err := r.client.NewRequest("PATCH", u, options)
	if err != nil {
		return nil, err
	}
	rc := &RegistryComponent{}
	err = req.Do(ctx, rc)
	if err != nil {
		return nil, err
	}

	return rc, nil
}

func (r *registryComponents) ListTagBindings(ctx context.Context, componentID string) ([]*TagBinding, error) {
	if !validStringID(&componentID) {
		return nil, ErrInvalidProjectID
	}

	u := fmt.Sprintf("registry-components/%s/tag-bindings", url.PathEscape(componentID))
	req, err := r.client.NewRequest("GET", u, nil)
	if err != nil {
		return nil, err
	}

	var list struct {
		*Pagination
		Items []*TagBinding
	}

	err = req.Do(ctx, &list)
	if err != nil {
		return nil, err
	}

	return list.Items, nil
}

// Delete a specified registry component.
func (r *registryComponents) Delete(ctx context.Context, componentID string) error {
	if !validStringID(&componentID) {
		return ErrInvalidRegistryComponentID
	}

	u := fmt.Sprintf("registry-components/%s", url.PathEscape(componentID))

	req, err := r.client.NewRequest("DELETE", u, nil)

	if err != nil {
		return err
	}

	return req.Do(ctx, nil)
}

type AgentExecutionMode string

const (
	AgentExecutionModeAgent  AgentExecutionMode = "agent"
	AgentExecutionModeRemote AgentExecutionMode = "remote"
)

func (a *AgentExecutionMode) UnmarshalText(text []byte) error {
	*a = AgentExecutionMode(string(text))
	return nil
}

func (a AgentExecutionMode) MarshalText() ([]byte, error) {
	return []byte(string(a)), nil
}

// Compile-time proof of interface implementation.
var _ RegistryModules = (*registryModules)(nil)

// RegistryModules describes all the registry module related methods that the Terraform
// Enterprise API supports.
//
// TFE API docs: https://developer.hashicorp.com/terraform/cloud-docs/api-docs/private-registry/modules
type RegistryModules interface {
	// List all the registry modules within an organization
	List(ctx context.Context, organization string, options *RegistryModuleListOptions) (*RegistryModuleList, error)

	// ListCommits List the commits for the registry module
	// This returns the latest 20 commits for the connected VCS repo.
	// Pagination is not applicable due to inconsistent support from the VCS providers.
	ListCommits(ctx context.Context, moduleID RegistryModuleID) (*CommitList, error)

	// Create a registry module without a VCS repo
	Create(ctx context.Context, organization string, options RegistryModuleCreateOptions) (*RegistryModule, error)

	// Create a registry module version
	CreateVersion(ctx context.Context, moduleID RegistryModuleID, options RegistryModuleCreateVersionOptions) (*RegistryModuleVersion, error)

	// Create and publish a registry module with a VCS repo
	CreateWithVCSConnection(ctx context.Context, options RegistryModuleCreateWithVCSConnectionOptions) (*RegistryModule, error)

	// Read a registry module
	Read(ctx context.Context, moduleID RegistryModuleID) (*RegistryModule, error)

	// ReadVersion Read a registry module version
	ReadVersion(ctx context.Context, moduleID RegistryModuleID, version string) (*RegistryModuleVersion, error)

	// ReadTerraformRegistryModule Reads a registry module from the Terraform
	// Registry, as opposed to Read or ReadVersion which read from the private
	// registry of a Terraform organization.
	// https://developer.hashicorp.com/terraform/enterprise/api-docs/private-registry/modules#hcp-terraform-registry-implementation
	ReadTerraformRegistryModule(ctx context.Context, moduleID RegistryModuleID, version string) (*TerraformRegistryModule, error)

	// Delete a registry module
	// Warning: This method is deprecated and will be removed from a future version of go-tfe. Use DeleteByName instead.
	Delete(ctx context.Context, organization string, name string) error

	// Delete a registry module by name
	DeleteByName(ctx context.Context, module RegistryModuleID) error

	// Delete a specified provider for the given module along with all its versions
	DeleteProvider(ctx context.Context, moduleID RegistryModuleID) error

	// Delete a specified version for the given provider of the module
	DeleteVersion(ctx context.Context, moduleID RegistryModuleID, version string) error

	// Update properties of a registry module
	Update(ctx context.Context, moduleID RegistryModuleID, options RegistryModuleUpdateOptions) (*RegistryModule, error)

	// Upload Terraform configuration files for the provided registry module version. It
	// requires a path to the configuration files on disk, which will be packaged by
	// hashicorp/go-slug before being uploaded.
	Upload(ctx context.Context, rmv RegistryModuleVersion, path string) error

	// Upload a tar gzip archive to the specified configuration version upload URL.
	UploadTarGzip(ctx context.Context, url string, r io.Reader) error

	// ListTagBindings lists all tag bindings associated with the module.
	ListTagBindings(ctx context.Context, moduleID string) ([]*TagBinding, error)
}

// TerraformRegistryModule contains data about a module from the Terraform Registry.
type TerraformRegistryModule struct {
	ID              string   `json:"id"`
	Owner           string   `json:"owner"`
	Namespace       string   `json:"namespace"`
	Name            string   `json:"name"`
	Version         string   `json:"version"`
	Provider        string   `json:"provider"`
	ProviderLogoURL string   `json:"provider_logo_url"`
	Description     string   `json:"description"`
	Source          string   `json:"source"`
	Tag             string   `json:"tag"`
	PublishedAt     string   `json:"published_at"`
	Downloads       int      `json:"downloads"`
	Verified        bool     `json:"verified"`
	Root            Root     `json:"root"`
	Providers       []string `json:"providers"`
	Versions        []string `json:"versions"`
}

type Root struct {
	Path                 string               `json:"path"`
	Name                 string               `json:"name"`
	Readme               string               `json:"readme"`
	Empty                bool                 `json:"empty"`
	Inputs               []Input              `json:"inputs"`
	Outputs              []Output             `json:"outputs"`
	ProviderDependencies []ProviderDependency `json:"provider_dependencies"`
	Resources            []Resource           `json:"resources"`
}

type Input struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Description string `json:"description"`
	Default     string `json:"default"`
	Required    bool   `json:"required"`
}

type Output struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type ProviderDependency struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
	Source    string `json:"source"`
	Version   string `json:"version"`
}

type Resource struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

// registryModules implements RegistryModules.
type registryModules struct {
	client *Client
}

// RegistryModuleStatus represents the status of the registry module
type RegistryModuleStatus string

// List of available registry module statuses
const (
	RegistryModuleStatusPending       RegistryModuleStatus = "pending"
	RegistryModuleStatusNoVersionTags RegistryModuleStatus = "no_version_tags"
	RegistryModuleStatusSetupFailed   RegistryModuleStatus = "setup_failed"
	RegistryModuleStatusSetupComplete RegistryModuleStatus = "setup_complete"
)

// RegistryModuleVersionStatus represents the status of a specific version of a registry module
type RegistryModuleVersionStatus string

// List of available registry module version statuses
const (
	RegistryModuleVersionStatusPending             RegistryModuleVersionStatus = "pending"
	RegistryModuleVersionStatusCloning             RegistryModuleVersionStatus = "cloning"
	RegistryModuleVersionStatusCloneFailed         RegistryModuleVersionStatus = "clone_failed"
	RegistryModuleVersionStatusRegIngressReqFailed RegistryModuleVersionStatus = "reg_ingress_req_failed"
	RegistryModuleVersionStatusRegIngressing       RegistryModuleVersionStatus = "reg_ingressing"
	RegistryModuleVersionStatusRegIngressFailed    RegistryModuleVersionStatus = "reg_ingress_failed"
	RegistryModuleVersionStatusOk                  RegistryModuleVersionStatus = "ok"
)

type PublishingMechanism string

const (
	PublishingMechanismBranch PublishingMechanism = "branch"
	PublishingMechanismTag    PublishingMechanism = "git_tag"
)

// RegistryModuleID represents the set of IDs that identify a RegistryModule
// Use NewPublicRegistryModuleID or NewPrivateRegistryModuleID to build one

type RegistryModuleID struct {
	// The unique ID of the module. If given, the other fields are ignored.
	ID string
	// The organization the module belongs to, see RegistryModule.Organization.Name
	Organization string
	// The name of the module, see RegistryModule.Name
	Name string
	// The module's provider, see RegistryModule.Provider
	Provider string
	// The namespace of the module. For private modules this is the name of the organization that owns the module
	// Required for public modules
	Namespace string
	// Either public or private. If not provided, defaults to private
	RegistryName RegistryName
}

// RegistryModuleList represents a list of registry modules.
type RegistryModuleList struct {
	*Pagination
	Items []*RegistryModule
}

// CommitList represents a list of the latest commits from the registry module
type CommitList struct {
	*Pagination
	Items []*Commit
}

// RegistryModule represents a registry module
type RegistryModule struct {
	ID                  string                          `jsonapi:"primary,registry-modules"`
	Name                string                          `jsonapi:"attr,name"`
	Provider            string                          `jsonapi:"attr,provider"`
	RegistryName        RegistryName                    `jsonapi:"attr,registry-name"`
	Namespace           string                          `jsonapi:"attr,namespace"`
	NoCode              bool                            `jsonapi:"attr,no-code"`
	Permissions         *RegistryModulePermissions      `jsonapi:"attr,permissions"`
	PublishingMechanism PublishingMechanism             `jsonapi:"attr,publishing-mechanism"`
	Status              RegistryModuleStatus            `jsonapi:"attr,status"`
	TestConfig          *TestConfig                     `jsonapi:"attr,test-config"`
	VCSRepo             *VCSRepo                        `jsonapi:"attr,vcs-repo"`
	VersionStatuses     []RegistryModuleVersionStatuses `jsonapi:"attr,version-statuses"`
	CreatedAt           string                          `jsonapi:"attr,created-at"`
	UpdatedAt           string                          `jsonapi:"attr,updated-at"`

	// Relations
	Organization *Organization `jsonapi:"relation,organization"`
	TagBindings  []*TagBinding `jsonapi:"relation,tag-bindings,omitempty"`

	RegistryNoCodeModule []*RegistryNoCodeModule `jsonapi:"relation,no-code-modules"`
}

// Commit represents a commit
type Commit struct {
	ID              string `jsonapi:"primary,commit"`
	Sha             string `jsonapi:"attr,sha"`
	Date            string `jsonapi:"attr,date"`
	URL             string `jsonapi:"attr,url"`
	Author          string `jsonapi:"attr,author"`
	AuthorAvatarURL string `jsonapi:"attr,author-avatar-url"`
	AuthorHTMLURL   string `jsonapi:"attr,author-html-url"`
	Message         string `jsonapi:"attr,message"`
}

// RegistryModuleVersion represents a registry module version
type RegistryModuleVersion struct {
	ID        string                      `jsonapi:"primary,registry-module-versions"`
	Source    string                      `jsonapi:"attr,source"`
	Status    RegistryModuleVersionStatus `jsonapi:"attr,status"`
	Version   string                      `jsonapi:"attr,version"`
	CreatedAt string                      `jsonapi:"attr,created-at"`
	UpdatedAt string                      `jsonapi:"attr,updated-at"`

	// Relations
	RegistryModule *RegistryModule `jsonapi:"relation,registry-module"`

	// Links
	Links map[string]interface{} `jsonapi:"links,omitempty"`
}

type RegistryModulePermissions struct {
	CanDelete bool `jsonapi:"attr,can-delete"`
	CanResync bool `jsonapi:"attr,can-resync"`
	CanRetry  bool `jsonapi:"attr,can-retry"`
}

type RegistryModuleVersionStatuses struct {
	Version string                      `jsonapi:"attr,version"`
	Status  RegistryModuleVersionStatus `jsonapi:"attr,status"`
	Error   string                      `jsonapi:"attr,error"`
}

// RegistryModuleListOptions represents the options for listing registry modules.
type RegistryModuleListOptions struct {
	ListOptions

	// Include is a list of relations to include.
	Include []RegistryModuleListIncludeOpt `url:"include,omitempty"`

	// Search is a search query string. Modules are searchable by name, namespace, provider fields.
	Search string `url:"q,omitempty"`

	// Provider filters results by provider name
	Provider string `url:"filter[provider],omitempty"`

	// RegistryName filters results by registry name (public or private)
	RegistryName RegistryName `url:"filter[registry_name],omitempty"`

	// OrganizationName filters results by organization name
	OrganizationName string `url:"filter[organization_name],omitempty"`
}

type RegistryModuleListIncludeOpt string

const IncludeNoCodeModules RegistryModuleListIncludeOpt = "no-code-modules"

// RegistryModuleCreateOptions is used when creating a registry module without a VCS repo
type RegistryModuleCreateOptions struct {
	// Type is a public field utilized by JSON:API to
	// set the resource type via the field tag.
	// It is not a user-defined value and does not need to be set.
	// https://jsonapi.org/format/#crud-creating
	Type string `jsonapi:"primary,registry-modules"`
	// Required:
	Name *string `jsonapi:"attr,name"`
	// Required:
	Provider *string `jsonapi:"attr,provider"`
	// Optional: Whether this is a publicly maintained module or private. Must be either public or private.
	// Defaults to private if not specified
	RegistryName RegistryName `jsonapi:"attr,registry-name,omitempty"`
	// Optional: The namespace of this module. Required for public modules only.
	Namespace string `jsonapi:"attr,namespace,omitempty"`
	// Optional: If set to true the module is enabled for no-code provisioning.
	// **Note: This field is still in BETA and subject to change.**
	NoCode *bool `jsonapi:"attr,no-code,omitempty"`
}

// RegistryModuleCreateVersionOptions is used when creating a registry module version
type RegistryModuleCreateVersionOptions struct {
	// Type is a public field utilized by JSON:API to
	// set the resource type via the field tag.
	// It is not a user-defined value and does not need to be set.
	// https://jsonapi.org/format/#crud-creating
	Type string `jsonapi:"primary,registry-module-versions"`

	Version *string `jsonapi:"attr,version"`

	CommitSHA *string `jsonapi:"attr,commit-sha"`
}

// RegistryModuleCreateWithVCSConnectionOptions is used when creating a registry module with a VCS repo
type RegistryModuleCreateWithVCSConnectionOptions struct {
	// Type is a public field utilized by JSON:API to
	// set the resource type via the field tag.
	// It is not a user-defined value and does not need to be set.
	// https://jsonapi.org/format/#crud-creating
	Type string `jsonapi:"primary,registry-modules"`

	// Optional: The Name of the Module. If not provided, will be inferred from the VCS repository identifier.
	// Required for monorepos with source_directory where the repository name doesn't follow the terraform-<provider>-<name> convention.
	Name *string `jsonapi:"attr,name,omitempty"`

	// Optional: The Name of the Provider. If not provided, will be inferred from the VCS repository identifier.
	// Required for monorepos with source_directory where the repository name doesn't follow the terraform-<provider>-<name> convention.
	Provider *string `jsonapi:"attr,provider,omitempty"`

	// Required: VCS repository information
	VCSRepo *RegistryModuleVCSRepoOptions `jsonapi:"attr,vcs-repo"`

	// Optional: If Branch is set within VCSRepo then InitialVersion sets the
	// initial version of the newly created branch-based registry module. If
	// Branch is not set within VCSRepo then InitialVersion is ignored.
	//
	// Defaults to "0.0.0".
	//
	// **Note: This field is still in BETA and subject to change.**
	InitialVersion *string `jsonapi:"attr,initial-version,omitempty"`

	// Optional: Flag to enable tests for the module
	// **Note: This field is still in BETA and subject to change.**
	TestConfig *RegistryModuleTestConfigOptions `jsonapi:"attr,test-config,omitempty"`
}

// RegistryModuleCreateVersionOptions is used when updating a registry module
type RegistryModuleUpdateOptions struct {
	// Type is a public field utilized by JSON:API to
	// set the resource type via the field tag.
	// It is not a user-defined value and does not need to be set.
	// https://jsonapi.org/format/#crud-updating
	Type string `jsonapi:"primary,registry-modules"`

	// Optional: Flag to enable no-code provisioning for the whole module.
	// **Note: This field is still in BETA and subject to change.**
	NoCode *bool `jsonapi:"attr,no-code,omitempty"`

	// Optional: Flag to enable tests for the module
	// **Note: This field is still in BETA and subject to change.**
	TestConfig *RegistryModuleTestConfigOptions `jsonapi:"attr,test-config,omitempty"`

	VCSRepo *RegistryModuleVCSRepoUpdateOptions `jsonapi:"attr,vcs-repo,omitempty"`

	// Optional: Tag bindings for the registry provider. Note that this
	// will replace all existing tag bindings.
	TagBindings []*TagBinding `jsonapi:"relation,tag-bindings"`
}

type RegistryModuleTestConfigOptions struct {
	TestsEnabled       *bool               `jsonapi:"attr,tests-enabled,omitempty"`
	AgentExecutionMode *AgentExecutionMode `jsonapi:"attr,agent-execution-mode,omitempty"`
	AgentPoolID        *string             `jsonapi:"attr,agent-pool-id,omitempty"`
}

type RegistryModuleVCSRepoOptions struct {
	Identifier        *string `json:"identifier"` // Required
	OAuthTokenID      *string `json:"oauth-token-id,omitempty"`
	DisplayIdentifier *string `json:"display-identifier,omitempty"` // Required
	GHAInstallationID *string `json:"github-app-installation-id,omitempty"`
	OrganizationName  *string `json:"organization-name,omitempty"`

	// Optional: If set, the newly created registry module will be branch-based
	// with the starting branch set to Branch.
	//
	// **Note: This field is still in BETA and subject to change.**
	Branch *string `json:"branch,omitempty"`
	Tags   *bool   `json:"tags,omitempty"`

	// Optional: If set, the registry module will be branch-based or tag-based
	SourceDirectory *string `json:"source-directory,omitempty"`
	TagPrefix       *string `json:"tag-prefix,omitempty"`
}

type RegistryModuleVCSRepoUpdateOptions struct {
	// The Branch and Tag fields are used to determine
	// the PublishingMechanism for a RegistryModule that has a VCS a connection.
	// When a value for Branch is provided, the Tags field is removed on the server
	// When a value for Tags is provided, the Branch field is removed on the server
	// **Note: This field is still in BETA and subject to change.**
	Branch *string `json:"branch,omitempty"`
	Tags   *bool   `json:"tags,omitempty"`

	// Optional: If set, the registry module will be branch-based or tag-based
	SourceDirectory *string `json:"source-directory,omitempty"`
	TagPrefix       *string `json:"tag-prefix,omitempty"`
}

// List all the registry modules within an organization.
func (r *registryModules) List(ctx context.Context, organization string, options *RegistryModuleListOptions) (*RegistryModuleList, error) {
	if !validStringID(&organization) {
		return nil, ErrInvalidOrg
	}

	u := fmt.Sprintf("organizations/%s/registry-modules", url.PathEscape(organization))
	req, err := r.client.NewRequest("GET", u, options)
	if err != nil {
		return nil, err
	}

	ml := &RegistryModuleList{}
	err = req.Do(ctx, ml)
	if err != nil {
		return nil, err
	}

	return ml, nil
}

// List the last 20 commits for the registry modules within an organization.
func (r *registryModules) ListCommits(ctx context.Context, moduleID RegistryModuleID) (*CommitList, error) {
	if !validStringID(&moduleID.Organization) {
		return nil, ErrInvalidOrg
	}

	u := fmt.Sprintf(
		"organizations/%s/registry-modules/private/%s/%s/%s/commits",
		url.PathEscape(moduleID.Organization),
		url.PathEscape(moduleID.Organization),
		url.PathEscape(moduleID.Name),
		url.PathEscape(moduleID.Provider),
	)
	req, err := r.client.NewRequest("GET", u, nil)
	if err != nil {
		return nil, err
	}

	cl := &CommitList{}
	err = req.Do(ctx, cl)
	if err != nil {
		return nil, err
	}

	return cl, nil
}

// Upload uploads Terraform configuration files for the provided registry module version. It
// requires a path to the configuration files on disk, which will be packaged by
// hashicorp/go-slug before being uploaded.
func (r *registryModules) Upload(ctx context.Context, rmv RegistryModuleVersion, path string) error {
	uploadURL, ok := rmv.Links["upload"].(string)
	if !ok {
		return fmt.Errorf("provided RegistryModuleVersion does not contain an upload link")
	}

	body, err := packContents(path)
	if err != nil {
		return err
	}

	return r.UploadTarGzip(ctx, uploadURL, body)
}

// UploadTarGzip is used to upload Terraform configuration files contained a tar gzip archive.
// Any stream implementing io.Reader can be passed into this method. This method is also
// particularly useful for tar streams created by non-default go-slug configurations.
//
// **Note**: This method does not validate the content being uploaded and is therefore the caller's
// responsibility to ensure the raw content is a valid Terraform configuration.
func (r *registryModules) UploadTarGzip(ctx context.Context, uploadURL string, archive io.Reader) error {
	return r.client.doForeignPUTRequest(ctx, uploadURL, archive)
}

// Create a new registry module without a VCS repo
func (r *registryModules) Create(ctx context.Context, organization string, options RegistryModuleCreateOptions) (*RegistryModule, error) {
	if !validStringID(&organization) {
		return nil, ErrInvalidOrg
	}
	if err := options.valid(); err != nil {
		return nil, err
	}

	if options.NoCode != nil {
		log.Println("[WARN] Support for using the NoCode field is deprecated as of release 1.22.0 and may be removed in a future version. The preferred way to create a no-code module is with the registryNoCodeModules.Create method.")
	}

	u := fmt.Sprintf(
		"organizations/%s/registry-modules",
		url.PathEscape(organization),
	)
	req, err := r.client.NewRequest("POST", u, &options)
	if err != nil {
		return nil, err
	}

	rm := &RegistryModule{}
	err = req.Do(ctx, rm)
	if err != nil {
		return nil, err
	}

	return rm, nil
}

func (r *registryModules) Update(ctx context.Context, moduleID RegistryModuleID, options RegistryModuleUpdateOptions) (*RegistryModule, error) {
	if err := moduleID.valid(); err != nil {
		return nil, err
	}

	if moduleID.RegistryName == "" {
		log.Println("[WARN] Support for using the RegistryModuleID without RegistryName is deprecated as of release 1.5.0 and may be removed in a future version. The preferred method is to include the RegistryName in RegistryModuleID.")
		moduleID.RegistryName = PrivateRegistry
	}

	if moduleID.RegistryName == PrivateRegistry && strings.TrimSpace(moduleID.Namespace) == "" {
		log.Println("[WARN] Support for using the RegistryModuleID without Namespace is deprecated as of release 1.5.0 and may be removed in a future version. The preferred method is to include the Namespace in RegistryModuleID.")
		moduleID.Namespace = moduleID.Organization
	}

	if options.NoCode != nil {
		log.Println("[WARN] Support for using the NoCode field is deprecated as of release 1.22.0 and may be removed in a future version. The preferred way to update a no-code module is with the registryNoCodeModules.Update method.")
	}

	if options.VCSRepo != nil {
		if options.VCSRepo.Tags != nil && *options.VCSRepo.Tags && validString(options.VCSRepo.Branch) {
			return nil, ErrBranchMustBeEmptyWhenTagsEnabled
		}
	}

	if options.TestConfig != nil && options.TestConfig.AgentExecutionMode != nil {
		if *options.TestConfig.AgentExecutionMode == AgentExecutionModeRemote && options.TestConfig.AgentPoolID != nil {
			return nil, ErrAgentPoolNotRequiredForRemoteExecution
		}
	}

	org := url.PathEscape(moduleID.Organization)
	registryName := url.PathEscape(string(moduleID.RegistryName))
	namespace := url.PathEscape(moduleID.Namespace)
	name := url.PathEscape(moduleID.Name)
	provider := url.PathEscape(moduleID.Provider)
	registryModuleURL := fmt.Sprintf("organizations/%s/registry-modules/%s/%s/%s/%s", org, registryName, namespace, name, provider)

	req, err := r.client.NewRequest(http.MethodPatch, registryModuleURL, &options)
	if err != nil {
		return nil, err
	}

	rm := &RegistryModule{}
	if err := req.Do(ctx, rm); err != nil {
		return nil, err
	}

	return rm, nil
}

func (r *registryModules) ListTagBindings(ctx context.Context, moduleID string) ([]*TagBinding, error) {
	if !validStringID(&moduleID) {
		return nil, ErrInvalidModuleID
	}

	u := fmt.Sprintf("registry-modules/%s/tag-bindings", url.PathEscape(moduleID))
	req, err := r.client.NewRequest("GET", u, nil)
	if err != nil {
		return nil, err
	}

	var list struct {
		*Pagination
		Items []*TagBinding
	}

	err = req.Do(ctx, &list)
	if err != nil {
		return nil, err
	}

	return list.Items, nil
}

// CreateVersion creates a new registry module version
func (r *registryModules) CreateVersion(ctx context.Context, moduleID RegistryModuleID, options RegistryModuleCreateVersionOptions) (*RegistryModuleVersion, error) {
	if err := moduleID.valid(); err != nil {
		return nil, err
	}

	if err := options.valid(); err != nil {
		return nil, err
	}

	u := fmt.Sprintf(
		"registry-modules/%s/%s/%s/versions",
		url.PathEscape(moduleID.Organization),
		url.PathEscape(moduleID.Name),
		url.PathEscape(moduleID.Provider),
	)
	req, err := r.client.NewRequest("POST", u, &options)
	if err != nil {
		return nil, err
	}

	rmv := &RegistryModuleVersion{}
	err = req.Do(ctx, rmv)
	if err != nil {
		return nil, err
	}

	return rmv, nil
}

// CreateWithVCSConnection is used to create and publish a new registry module with a VCS repo
func (r *registryModules) CreateWithVCSConnection(ctx context.Context, options RegistryModuleCreateWithVCSConnectionOptions) (*RegistryModule, error) {
	if err := options.valid(); err != nil {
		return nil, err
	}
	var u string
	if options.VCSRepo.OAuthTokenID != nil && options.VCSRepo.Branch == nil {
		u = "registry-modules"
	} else {
		u = fmt.Sprintf(
			"organizations/%s/registry-modules/vcs",
			url.PathEscape(*options.VCSRepo.OrganizationName),
		)
	}

	if options.TestConfig != nil && options.TestConfig.AgentExecutionMode != nil {
		if *options.TestConfig.AgentExecutionMode == AgentExecutionModeRemote && options.TestConfig.AgentPoolID != nil {
			return nil, ErrAgentPoolNotRequiredForRemoteExecution
		}
	}

	req, err := r.client.NewRequest("POST", u, &options)
	if err != nil {
		return nil, err
	}

	rm := &RegistryModule{}
	err = req.Do(ctx, rm)
	if err != nil {
		return nil, err
	}

	return rm, nil
}

// Read a specific registry module
func (r *registryModules) Read(ctx context.Context, moduleID RegistryModuleID) (*RegistryModule, error) {
	if err := moduleID.valid(); err != nil {
		return nil, err
	}

	var u string
	if moduleID.ID == "" {
		if moduleID.RegistryName == "" {
			log.Println("[WARN] Support for using the RegistryModuleID without RegistryName is deprecated as of release 1.5.0 and may be removed in a future version. The preferred method is to include the RegistryName in RegistryModuleID.")
			moduleID.RegistryName = PrivateRegistry
		}

		if moduleID.RegistryName == PrivateRegistry && strings.TrimSpace(moduleID.Namespace) == "" {
			log.Println("[WARN] Support for using the RegistryModuleID without Namespace is deprecated as of release 1.5.0 and may be removed in a future version. The preferred method is to include the Namespace in RegistryModuleID.")
			moduleID.Namespace = moduleID.Organization
		}

		u = fmt.Sprintf(
			"organizations/%s/registry-modules/%s/%s/%s/%s",
			url.PathEscape(moduleID.Organization),
			url.PathEscape(string(moduleID.RegistryName)),
			url.PathEscape(moduleID.Namespace),
			url.PathEscape(moduleID.Name),
			url.PathEscape(moduleID.Provider),
		)
	} else {
		u = fmt.Sprintf("registry-modules/%s", url.PathEscape(moduleID.ID))
	}

	req, err := r.client.NewRequest("GET", u, nil)
	if err != nil {
		return nil, err
	}

	rm := &RegistryModule{}
	err = req.Do(ctx, rm)
	if err != nil {
		return nil, err
	}

	return rm, nil
}

// ReadTerraformRegistryModule fetches a registry module from the Terraform Registry.
func (r *registryModules) ReadTerraformRegistryModule(ctx context.Context, moduleID RegistryModuleID, ver string) (*TerraformRegistryModule, error) {
	u := fmt.Sprintf("/api/registry/v1/modules/%s/%s/%s/%s",
		moduleID.Namespace,
		moduleID.Name,
		moduleID.Provider,
		ver,
	)

	if moduleID.RegistryName == PublicRegistry {
		u = fmt.Sprintf("/api/registry/public/v1/modules/%s/%s/%s/%s",
			moduleID.Namespace,
			moduleID.Name,
			moduleID.Provider,
			ver,
		)
	}
	req, err := r.client.NewRequest("GET", u, nil)
	if err != nil {
		return nil, err
	}

	trm := &TerraformRegistryModule{}
	err = req.DoJSON(ctx, trm)
	if err != nil {
		return nil, err
	}
	return trm, nil
}

func (r *registryModules) ReadVersion(ctx context.Context, moduleID RegistryModuleID, modVersion string) (*RegistryModuleVersion, error) {
	if err := moduleID.valid(); err != nil {
		return nil, err
	}
	if !validString(&modVersion) {
		return nil, ErrRequiredVersion
	}
	if !validStringID(&modVersion) {
		return nil, ErrInvalidVersion
	}
	u := fmt.Sprintf(
		"organizations/%s/registry-modules/private/%s/%s/%s/version?module_version=%s",
		url.PathEscape(moduleID.Organization),
		url.PathEscape(moduleID.Organization),
		url.PathEscape(moduleID.Name),
		url.PathEscape(moduleID.Provider),
		url.PathEscape(modVersion),
	)
	req, err := r.client.NewRequest("GET", u, nil)
	if err != nil {
		return nil, err
	}

	rmv := &RegistryModuleVersion{}
	err = req.Do(ctx, rmv)
	if err != nil {
		return nil, err
	}

	return rmv, nil
}

// Delete is used to delete the entire registry module
// Warning: This method is deprecated and will be removed from a future version of go-tfe. Use DeleteByName instead.
// See API Docs: https://developer.hashicorp.com/terraform/cloud-docs/api-docs/private-registry/modules#delete-a-module
func (r *registryModules) Delete(ctx context.Context, organization, name string) error {
	if !validStringID(&organization) {
		return ErrInvalidOrg
	}
	if !validString(&name) {
		return ErrRequiredName
	}
	if !validStringID(&name) {
		return ErrInvalidName
	}

	u := fmt.Sprintf(
		"registry-modules/actions/delete/%s/%s",
		url.PathEscape(organization),
		url.PathEscape(name),
	)
	req, err := r.client.NewRequest("POST", u, nil)
	if err != nil {
		return err
	}

	return req.Do(ctx, nil)
}

// DeleteByName is used to delete the entire registry module
func (r *registryModules) DeleteByName(ctx context.Context, module RegistryModuleID) error {
	if err := module.validWhenDeleteByName(); err != nil {
		return err
	}

	u := fmt.Sprintf(
		"organizations/%s/registry-modules/%s/%s/%s",
		url.PathEscape(module.Organization),
		url.PathEscape(string(module.RegistryName)),
		url.PathEscape(module.Namespace),
		url.PathEscape(module.Name),
	)

	req, err := r.client.NewRequest("DELETE", u, nil)
	if err != nil && errors.Is(err, ErrResourceNotFound) {
		return r.Delete(ctx, module.Organization, module.Name)
	}

	return req.Do(ctx, nil)
}

// Delete a specified provider for the given module along with all its versions
func (r *registryModules) DeleteProvider(ctx context.Context, moduleID RegistryModuleID) error {
	if err := moduleID.validWhenDeleteByProvider(); err != nil {
		return err
	}

	u := fmt.Sprintf(
		"organizations/%s/registry-modules/%s/%s/%s/%s",
		url.PathEscape(moduleID.Organization),
		url.PathEscape(string(moduleID.RegistryName)),
		url.PathEscape(moduleID.Namespace),
		url.PathEscape(moduleID.Name),
		url.PathEscape(moduleID.Provider),
	)

	req, err := r.client.NewRequest("DELETE", u, nil)

	if err != nil && errors.Is(err, ErrResourceNotFound) {
		return r.deprecatedDeleteProvider(ctx, moduleID)
	}

	return req.Do(ctx, nil)
}

// Delete a specified version for the given provider of the module
func (r *registryModules) DeleteVersion(ctx context.Context, moduleID RegistryModuleID, modVersion string) error {
	if err := moduleID.valid(); err != nil {
		return err
	}
	if !validString(&modVersion) {
		return ErrRequiredVersion
	}
	if !validVersion(modVersion) {
		return ErrInvalidVersion
	}

	u := fmt.Sprintf(
		"organizations/%s/registry-modules/%s/%s/%s/%s/%s",
		url.PathEscape(moduleID.Organization),
		url.PathEscape(string(moduleID.RegistryName)),
		url.PathEscape(moduleID.Namespace),
		url.PathEscape(moduleID.Name),
		url.PathEscape(moduleID.Provider),
		url.PathEscape(modVersion),
	)
	req, err := r.client.NewRequest("DELETE", u, nil)
	if err != nil && errors.Is(err, ErrResourceNotFound) {
		return r.deprecatedDeleteVersion(ctx, moduleID, modVersion)
	}

	return req.Do(ctx, nil)
}

func (o RegistryModuleID) valid() error {
	if validString(&o.ID) && validStringID(&o.ID) {
		return nil
	}

	if !validStringID(&o.Organization) {
		return ErrInvalidOrg
	}

	if !validString(&o.Name) {
		return ErrRequiredName
	}

	if !validStringID(&o.Name) {
		return ErrInvalidName
	}

	if !validString(&o.Provider) {
		return ErrRequiredProvider
	}

	if !validStringID(&o.Provider) {
		return ErrInvalidProvider
	}

	switch o.RegistryName {
	case PublicRegistry:
		if !validString(&o.Namespace) {
			return ErrRequiredNamespace
		}
	case PrivateRegistry:
	case "":
		// no-op:  RegistryName is optional
	// for all other string
	default:
		return ErrInvalidRegistryName
	}

	return nil
}

func (o RegistryModuleID) validWhenDeleteByProvider() error {
	if !validStringID(&o.Organization) {
		return ErrInvalidOrg
	}

	if !validString(&o.Name) {
		return ErrRequiredName
	}

	if !validStringID(&o.Name) {
		return ErrInvalidName
	}

	if !validString(&o.Provider) {
		return ErrRequiredProvider
	}

	if !validStringID(&o.Provider) {
		return ErrInvalidProvider
	}
	// RegistryName is required in this DELETE call
	switch o.RegistryName {
	case PublicRegistry:
		if !validString(&o.Namespace) {
			return ErrRequiredNamespace
		}
	case PrivateRegistry:
	case "":
		return ErrInvalidRegistryName
	default:
		return ErrInvalidRegistryName
	}

	return nil
}

func (o RegistryModuleID) validWhenDeleteByName() error {
	if !validStringID(&o.Organization) {
		return ErrInvalidOrg
	}

	if !validString(&o.Name) {
		return ErrRequiredName
	}

	if !validStringID(&o.Name) {
		return ErrInvalidName
	}

	// RegistryName is required in this DELETE call
	switch o.RegistryName {
	case PublicRegistry:
		if !validString(&o.Namespace) {
			return ErrRequiredNamespace
		}
	case PrivateRegistry:
	case "":
		return ErrInvalidRegistryName
	default:
		return ErrInvalidRegistryName
	}

	return nil
}

func (o RegistryModuleCreateOptions) valid() error {
	if !validString(o.Name) {
		return ErrRequiredName
	}
	if !validStringID(o.Name) {
		return ErrInvalidName
	}
	if !validString(o.Provider) {
		return ErrRequiredProvider
	}
	if !validStringID(o.Provider) {
		return ErrInvalidProvider
	}

	switch o.RegistryName {
	case PublicRegistry:
		if !validString(&o.Namespace) {
			return ErrRequiredNamespace
		}
	case PrivateRegistry:
		if validString(&o.Namespace) {
			return ErrUnsupportedBothNamespaceAndPrivateRegistryName
		}
	case "":
		// no-op:  RegistryName is optional
	// for all other string
	default:
		return ErrInvalidRegistryName
	}
	return nil
}

func (o RegistryModuleCreateVersionOptions) valid() error {
	if !validString(o.Version) {
		return ErrRequiredVersion
	}
	if !validVersion(*o.Version) {
		return ErrInvalidVersion
	}
	return nil
}

func (o RegistryModuleCreateWithVCSConnectionOptions) valid() error {
	if o.VCSRepo == nil {
		return ErrRequiredVCSRepo
	}

	if o.TestConfig != nil && o.TestConfig.TestsEnabled != nil {
		if *o.TestConfig.TestsEnabled {
			if !validString(o.VCSRepo.Branch) {
				return ErrRequiredBranchWhenTestsEnabled
			}
		}
	}

	if o.VCSRepo.Tags != nil && *o.VCSRepo.Tags {
		if validString(o.VCSRepo.Branch) {
			return ErrBranchMustBeEmptyWhenTagsEnabled
		}
	}

	return o.VCSRepo.valid()
}

func (o RegistryModuleVCSRepoOptions) valid() error {
	if !validString(o.Identifier) {
		return ErrRequiredIdentifier
	}
	if !validString(o.OAuthTokenID) && !validString(o.GHAInstallationID) {
		return ErrRequiredOauthTokenOrGithubAppInstallationID
	}
	if (!validString(o.OAuthTokenID) && validString(o.GHAInstallationID)) || validString(o.Branch) {
		if !validString(o.OrganizationName) {
			return ErrInvalidOrg
		}
	}
	if !validString(o.DisplayIdentifier) {
		return ErrRequiredDisplayIdentifier
	}
	return nil
}

func (r *registryModules) deprecatedDeleteProvider(ctx context.Context, moduleID RegistryModuleID) error {
	if err := moduleID.valid(); err != nil {
		return err
	}

	u := fmt.Sprintf(
		"registry-modules/actions/delete/%s/%s/%s",
		url.PathEscape(moduleID.Organization),
		url.PathEscape(moduleID.Name),
		url.PathEscape(moduleID.Provider),
	)
	req, err := r.client.NewRequest("POST", u, nil)
	if err != nil {
		return err
	}

	return req.Do(ctx, nil)
}

func (r *registryModules) deprecatedDeleteVersion(ctx context.Context, moduleID RegistryModuleID, modVersion string) error {
	if err := moduleID.valid(); err != nil {
		return err
	}
	if !validString(&modVersion) {
		return ErrRequiredVersion
	}
	if !validVersion(modVersion) {
		return ErrInvalidVersion
	}

	u := fmt.Sprintf(
		"registry-modules/actions/delete/%s/%s/%s/%s",
		url.PathEscape(moduleID.Organization),
		url.PathEscape(moduleID.Name),
		url.PathEscape(moduleID.Provider),
		url.PathEscape(modVersion),
	)
	req, err := r.client.NewRequest("POST", u, nil)
	if err != nil {
		return err
	}

	return req.Do(ctx, nil)
}

func NewPublicRegistryModuleID(organization, namespace, name, provider string) RegistryModuleID {
	return RegistryModuleID{
		Organization: organization,
		Namespace:    namespace,
		Name:         name,
		RegistryName: PublicRegistry,
		Provider:     provider,
	}
}

func NewPrivateRegistryModuleID(organization, name, provider string) RegistryModuleID {
	return RegistryModuleID{
		Organization: organization,
		Namespace:    organization,
		Name:         name,
		RegistryName: PrivateRegistry,
		Provider:     provider,
	}
}

// Compile-time proof of interface implementation.
var _ RegistryNoCodeModules = (*registryNoCodeModules)(nil)

// RegistryNoCodeModules describes all the registry no-code module related methods that the Terraform
// Enterprise API supports.
//
// TFE API docs: (TODO: Add link to API docs)
type RegistryNoCodeModules interface {

	// Create a registry no-code module
	// **Note: This API is still in BETA and subject to change.**
	Create(ctx context.Context, organization string, options RegistryNoCodeModuleCreateOptions) (*RegistryNoCodeModule, error)

	// Read a registry no-code  module
	// **Note: This API is still in BETA and subject to change.**
	Read(ctx context.Context, noCodeModuleID string, options *RegistryNoCodeModuleReadOptions) (*RegistryNoCodeModule, error)

	// ReadVariables returns the variables for a version of a no-code module
	// **Note: This API is still in BETA and subject to change.**
	ReadVariables(ctx context.Context, noCodeModuleID, noCodeModuleVersion string, options *RegistryNoCodeModuleReadVariablesOptions) (*RegistryModuleVariableList, error)

	// Update a registry no-code module
	// **Note: This API is still in BETA and subject to change.**
	Update(ctx context.Context, noCodeModuleID string, options RegistryNoCodeModuleUpdateOptions) (*RegistryNoCodeModule, error)

	// Delete a registry no-code module
	// **Note: This API is still in BETA and subject to change.**
	Delete(ctx context.Context, ID string) error

	// CreateWorkspace creates a workspace using a no-code module.
	CreateWorkspace(ctx context.Context, noCodeModuleID string, options *RegistryNoCodeModuleCreateWorkspaceOptions) (*Workspace, error)

	// UpgradeWorkspace initiates an upgrade of an existing no-code module workspace.
	UpgradeWorkspace(ctx context.Context, noCodeModuleID string, workspaceID string, options *RegistryNoCodeModuleUpgradeWorkspaceOptions) (*WorkspaceUpgrade, error)
}

// RegistryModuleVariableList is a list of registry module variables.
// **Note: This API is still in BETA and subject to change.**
type RegistryModuleVariableList struct {
	Items []*RegistryModuleVariable

	// NOTE: At the time of authoring this comment, the API endpoint to fetch
	// registry module variables does not support pagination. This field is
	// included to satisfy jsonapi unmarshaler implementation here:
	// https://github.com/hashicorp/go-tfe/blob/3d29602707fa4b10469d1a02685644bd159d3ccc/tfe.go#L859
	*Pagination
}

// RegistryModuleVariable represents a registry module variable.
type RegistryModuleVariable struct {
	// ID is the ID of the variable.
	ID string `jsonapi:"primary,registry-module-variables"`

	// Name is the name of the variable.
	Name string `jsonapi:"attr,name"`

	// VariableType is the type of the variable.
	VariableType string `jsonapi:"attr,type"`

	// Description is the description of the variable.
	Description string `jsonapi:"attr,description"`

	// Required is a boolean indicating if the variable is required.
	Required bool `jsonapi:"attr,required"`

	// Sensitive is a boolean indicating if the variable is sensitive.
	Sensitive bool `jsonapi:"attr,sensitive"`

	// Options is a slice of strings representing the options for the variable.
	Options []string `jsonapi:"attr,options"`

	// HasGlobal is a boolean indicating if the variable is global.
	HasGlobal bool `jsonapi:"attr,has-global"`
}

type RegistryNoCodeModuleCreateWorkspaceOptions struct {
	Type string `jsonapi:"primary,no-code-module-workspace"`

	// Name is the name of the workspace, which can only include letters,
	// numbers, and _. This will be used as an identifier and must be unique in
	// the organization.
	Name string `jsonapi:"attr,name"`

	// Description is a description for the workspace.
	Description *string `jsonapi:"attr,description,omitempty"`

	AutoApply *bool `jsonapi:"attr,auto-apply,omitempty"`

	// Project is the associated project with the workspace. If not provided,
	// default project of the organization will be assigned to the workspace.
	Project *Project `jsonapi:"relation,project,omitempty"`

	// Variables is the slice of variables to be configured for the no-code
	// workspace.
	Variables []*Variable `jsonapi:"relation,vars,omitempty"`

	// SourceName is the name of the source of the workspace.
	SourceName *string `jsonapi:"attr,source-name,omitempty"`

	// SourceUrl is the URL of the source of the workspace.
	SourceURL *string `jsonapi:"attr,source-url,omitempty"`

	// ExecutionMode is the execution mode of the workspace.
	ExecutionMode *string `jsonapi:"attr,execution-mode,omitempty"`

	// AgentPoolId is the ID of the agent pool to use for the workspace.
	// This is required when execution mode is set to "agent".
	// This must not be specified when execution mode is set to "remote".
	AgentPoolID *string `jsonapi:"attr,agent-pool-id,omitempty"`

	// TerraformVersion is the version of Terraform to use for this workspace.
	// Must be a valid semver string, such as "1.5.0". If not specified, the
	// workspace will use the latest Terraform version available on the platform.
	TerraformVersion *string `jsonapi:"attr,terraform-version,omitempty"`
}

type RegistryNoCodeModuleUpgradeWorkspaceOptions struct {
	Type string `jsonapi:"primary,no-code-module-workspace"`

	// Variables is the slice of variables to be configured for the no-code
	// workspace.
	Variables []*Variable `jsonapi:"relation,vars,omitempty"`
}

// registryNoCodeModules implements RegistryNoCodeModules.
type registryNoCodeModules struct {
	client *Client
}

// RegistryNoCodeModule represents a registry no-code module
type RegistryNoCodeModule struct {
	ID         string `jsonapi:"primary,no-code-modules"`
	VersionPin string `jsonapi:"attr,version-pin"`
	Enabled    bool   `jsonapi:"attr,enabled"`

	// Relations
	Organization    *Organization           `jsonapi:"relation,organization"`
	RegistryModule  *RegistryModule         `jsonapi:"relation,registry-module"`
	VariableOptions []*NoCodeVariableOption `jsonapi:"relation,variable-options"`
}

// NoCodeVariableOption represents a registry no-code module variable and its
// options.
type NoCodeVariableOption struct {
	// Type is a public field utilized by JSON:API to
	// set the resource type via the field tag.
	Type string `jsonapi:"primary,variable-options"`

	// Required: The variable name
	VariableName string `jsonapi:"attr,variable-name"`

	// Required: The variable type
	VariableType string `jsonapi:"attr,variable-type"`

	// Optional: The options for the variable
	Options []string `jsonapi:"attr,options"`
}

// RegistryNoCodeModuleCreateOptions is used when creating a registry no-code module
type RegistryNoCodeModuleCreateOptions struct {
	// Type is a public field utilized by JSON:API to
	// set the resource type via the field tag.
	// It is not a user-defined value and does not need to be set.
	// https://jsonapi.org/format/#crud-creating
	Type string `jsonapi:"primary,no-code-modules"`

	// Required: the registry module to use for the no-code module (only the ID is used)
	RegistryModule *RegistryModule `jsonapi:"relation,registry-module"`

	// Optional: whether no-code is enabled for the module
	Enabled *bool `jsonapi:"attr,enabled,omitempty"`

	// Optional: the version pin for the module. valid values are "latest" or a semver string
	VersionPin string `jsonapi:"attr,version-pin,omitempty"`

	// Optional: the variable options for the registry module
	VariableOptions []*NoCodeVariableOption `jsonapi:"relation,variable-options,omitempty"`
}

// RegistryNoCodeModuleIncludeOpt represents the available options for include query params.
type RegistryNoCodeModuleIncludeOpt string

var (
	// RegistryNoCodeIncludeVariableOptions is used to include variable options in the response
	RegistryNoCodeIncludeVariableOptions RegistryNoCodeModuleIncludeOpt = "variable-options"
)

// RegistryNoCodeModuleReadOptions is used when reading a registry no-code module
type RegistryNoCodeModuleReadOptions struct {
	// Type is a public field utilized by JSON:API to
	// set the resource type via the field tag.
	// It is not a user-defined value and does not need to be set.
	// https://jsonapi.org/format/#crud-updating
	Type string `jsonapi:"primary,no-code-modules"`

	// Optional: Include is used to specify the related resources to include in the response.
	Include []RegistryNoCodeModuleIncludeOpt `url:"include,omitempty"`
}

// RegistryNoCodeModuleReadVariablesOptions is used when reading the variables
// for a no-code module.
type RegistryNoCodeModuleReadVariablesOptions struct {
	// Type is a public field utilized by JSON:API to
	// set the resource type via the field tag.
	// It is not a user-defined value and does not need to be set.
	// https://jsonapi.org/format/#crud-updating
	Type string `jsonapi:"primary,no-code-modules"`
}

// RegistryNoCodeModuleUpdateOptions is used when updating a registry no-code module
type RegistryNoCodeModuleUpdateOptions struct {
	// Type is a public field utilized by JSON:API to
	// set the resource type via the field tag.
	// It is not a user-defined value and does not need to be set.
	// https://jsonapi.org/format/#crud-updating
	Type string `jsonapi:"primary,no-code-modules"`

	// Required: the registry module to use for the no-code module (only the ID is used)
	RegistryModule *RegistryModule `jsonapi:"relation,registry-module"`

	// Optional: the version pin for the module. valid values are "latest" or a semver string
	VersionPin string `jsonapi:"attr,version-pin,omitempty"`

	// Optional: whether no-code is enabled for the module
	Enabled *bool `jsonapi:"attr,enabled,omitempty"`

	// Optional: are the variable options for the module
	VariableOptions []*NoCodeVariableOption `jsonapi:"relation,variable-options,omitempty"`
}

// WorkspaceUpgrade contains the data returned by the no-code workspace upgrade
// API endpoint.
type WorkspaceUpgrade struct {
	// Status is the status of the run of the upgrade
	Status string `jsonapi:"attr,status"`

	// PlanURL is the URL to the plan of the upgrade
	PlanURL string `jsonapi:"attr,plan-url"`

	// Message is the message returned by the API when an upgrade is not available.
	Message string `jsonapi:"attr,message"`
}

// Create a new registry no-code module
func (r *registryNoCodeModules) Create(ctx context.Context, organization string, options RegistryNoCodeModuleCreateOptions) (*RegistryNoCodeModule, error) {
	if !validStringID(&organization) {
		return nil, ErrInvalidOrg
	}
	if err := options.valid(); err != nil {
		return nil, err
	}

	u := fmt.Sprintf("organizations/%s/no-code-modules", url.PathEscape(organization))
	req, err := r.client.NewRequest("POST", u, &options)
	if err != nil {
		return nil, err
	}

	rm := &RegistryNoCodeModule{}
	err = req.Do(ctx, rm)
	if err != nil {
		return nil, err
	}

	return rm, nil
}

// Read a registry no-code module
func (r *registryNoCodeModules) Read(ctx context.Context, noCodeModuleID string, options *RegistryNoCodeModuleReadOptions) (*RegistryNoCodeModule, error) {
	if !validStringID(&noCodeModuleID) {
		return nil, ErrInvalidModuleID
	}

	if err := options.valid(); err != nil {
		return nil, err
	}

	u := fmt.Sprintf("no-code-modules/%s", url.PathEscape(noCodeModuleID))
	req, err := r.client.NewRequest("GET", u, options)
	if err != nil {
		return nil, err
	}

	rm := &RegistryNoCodeModule{}
	err = req.Do(ctx, rm)
	if err != nil {
		return nil, err
	}

	return rm, nil
}

// ReadVariables retrieves the no-code variable options for a version of a
// module.
func (r *registryNoCodeModules) ReadVariables(
	ctx context.Context,
	noCodeModuleID, noCodeModuleVersion string,
	options *RegistryNoCodeModuleReadVariablesOptions,
) (*RegistryModuleVariableList, error) {
	if !validStringID(&noCodeModuleID) {
		return nil, ErrInvalidModuleID
	}
	if !validVersion(noCodeModuleVersion) {
		return nil, ErrInvalidVersion
	}

	u := fmt.Sprintf(
		"no-code-modules/%s/versions/%s/module-variables",
		url.PathEscape(noCodeModuleID),
		url.PathEscape(noCodeModuleVersion),
	)
	req, err := r.client.NewRequest("GET", u, options)
	if err != nil {
		return nil, err
	}

	resp := &RegistryModuleVariableList{}
	err = req.Do(ctx, resp)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

// Update a registry no-code module
func (r *registryNoCodeModules) Update(ctx context.Context, noCodeModuleID string, options RegistryNoCodeModuleUpdateOptions) (*RegistryNoCodeModule, error) {
	if !validString(&noCodeModuleID) {
		return nil, ErrInvalidModuleID
	}
	if !validStringID(&noCodeModuleID) {
		return nil, ErrInvalidModuleID
	}

	if err := options.valid(); err != nil {
		return nil, err
	}

	u := fmt.Sprintf("no-code-modules/%s", url.PathEscape(noCodeModuleID))
	req, err := r.client.NewRequest("PATCH", u, &options)
	if err != nil {
		return nil, err
	}

	rm := &RegistryNoCodeModule{}
	err = req.Do(ctx, rm)
	if err != nil {
		return nil, err
	}

	return rm, nil
}

// Delete is used to delete the registry no-code module
func (r *registryNoCodeModules) Delete(ctx context.Context, noCodeModuleID string) error {
	if !validStringID(&noCodeModuleID) {
		return ErrInvalidModuleID
	}

	u := fmt.Sprintf("no-code-modules/%s", url.PathEscape(noCodeModuleID))
	req, err := r.client.NewRequest("DELETE", u, nil)
	if err != nil {
		return err
	}

	return req.Do(ctx, nil)
}

// CreateWorkspace creates a no-code workspace using a no-code module.
func (r *registryNoCodeModules) CreateWorkspace(
	ctx context.Context,
	noCodeModuleID string,
	options *RegistryNoCodeModuleCreateWorkspaceOptions,
) (*Workspace, error) {
	if err := options.valid(); err != nil {
		return nil, err
	}

	u := fmt.Sprintf("no-code-modules/%s/workspaces", url.PathEscape(noCodeModuleID))
	req, err := r.client.NewRequest("POST", u, options)
	if err != nil {
		return nil, err
	}

	w := &Workspace{}
	err = req.Do(ctx, w)
	if err != nil {
		return nil, err
	}

	return w, nil
}

// UpgradeWorkspace initiates an upgrade of an existing no-code module workspace.
func (r *registryNoCodeModules) UpgradeWorkspace(
	ctx context.Context,
	noCodeModuleID string,
	workspaceID string,
	options *RegistryNoCodeModuleUpgradeWorkspaceOptions,
) (*WorkspaceUpgrade, error) {
	if err := options.valid(); err != nil {
		return nil, err
	}

	u := fmt.Sprintf("no-code-modules/%s/workspaces/%s/upgrade",
		url.PathEscape(noCodeModuleID),
		workspaceID,
	)
	req, err := r.client.NewRequest("POST", u, options)
	if err != nil {
		return nil, err
	}

	wu := &WorkspaceUpgrade{}
	err = req.Do(ctx, wu)
	if err != nil {
		return nil, err
	}

	return wu, nil
}

func (o RegistryNoCodeModuleCreateOptions) valid() error {
	if o.RegistryModule == nil || o.RegistryModule.ID == "" {
		return ErrRequiredRegistryModule
	}

	return nil
}

func (o *RegistryNoCodeModuleUpdateOptions) valid() error {
	if o == nil {
		return nil // nothing to validate
	}

	if o.RegistryModule == nil || o.RegistryModule.ID == "" {
		return ErrRequiredRegistryModule
	}

	return nil
}

func (o *RegistryNoCodeModuleReadOptions) valid() error {
	return nil
}

func (o *RegistryNoCodeModuleCreateWorkspaceOptions) valid() error {
	if !validString(&o.Name) {
		return ErrRequiredName
	}

	return nil
}

func (o *RegistryNoCodeModuleUpgradeWorkspaceOptions) valid() error {
	return nil
}

// Compile-time proof of interface implementation
var _ RegistryProviderPlatforms = (*registryProviderPlatforms)(nil)

// RegistryProviderPlatforms describes the registry provider platform methods supported by the Terraform Enterprise API.
//
// TFE API docs: https://developer.hashicorp.com/terraform/cloud-docs/api-docs/private-registry/provider-versions-platforms#private-provider-versions-and-platforms-api
type RegistryProviderPlatforms interface {
	// Create a provider platform for an organization
	Create(ctx context.Context, versionID RegistryProviderVersionID, options RegistryProviderPlatformCreateOptions) (*RegistryProviderPlatform, error)

	// List all provider platforms for a single version
	List(ctx context.Context, versionID RegistryProviderVersionID, options *RegistryProviderPlatformListOptions) (*RegistryProviderPlatformList, error)

	// Read a provider platform by ID
	Read(ctx context.Context, platformID RegistryProviderPlatformID) (*RegistryProviderPlatform, error)

	// Delete a provider platform
	Delete(ctx context.Context, platformID RegistryProviderPlatformID) error
}

// registryProviders implements RegistryProviders
type registryProviderPlatforms struct {
	client *Client
}

// RegistryProviderPlatform represents a registry provider platform
type RegistryProviderPlatform struct {
	ID                     string `jsonapi:"primary,registry-provider-platforms"`
	OS                     string `jsonapi:"attr,os"`
	Arch                   string `jsonapi:"attr,arch"`
	Filename               string `jsonapi:"attr,filename"`
	Shasum                 string `jsonapi:"attr,shasum"`
	ProviderBinaryUploaded bool   `jsonapi:"attr,provider-binary-uploaded"`

	// Relations
	RegistryProviderVersion *RegistryProviderVersion `jsonapi:"relation,registry-provider-version"`

	// Links
	Links map[string]interface{} `jsonapi:"links,omitempty"`
}

// RegistryProviderPlatformID is the multi key ID for identifying a provider platform
type RegistryProviderPlatformID struct {
	RegistryProviderVersionID
	OS   string
	Arch string
}

// RegistryProviderPlatformCreateOptions represents the set of options for creating a registry provider platform
type RegistryProviderPlatformCreateOptions struct {
	// Required: A valid operating system string
	OS string `jsonapi:"attr,os"`

	// Required: A valid architecture string
	Arch string `jsonapi:"attr,arch"`

	// Required: A valid shasum string
	Shasum string `jsonapi:"attr,shasum"`

	// Required: A valid filename string
	Filename string `jsonapi:"attr,filename"`
}

type RegistryProviderPlatformList struct {
	*Pagination
	Items []*RegistryProviderPlatform
}

type RegistryProviderPlatformListOptions struct {
	ListOptions
}

// Create a new registry provider platform
func (r *registryProviderPlatforms) Create(ctx context.Context, versionID RegistryProviderVersionID, options RegistryProviderPlatformCreateOptions) (*RegistryProviderPlatform, error) {
	if err := versionID.valid(); err != nil {
		return nil, err
	}

	if err := options.valid(); err != nil {
		return nil, err
	}

	// POST /organizations/:organization_name/registry-providers/:registry_name/:namespace/:name/versions/:version/platforms
	u := fmt.Sprintf(
		"organizations/%s/registry-providers/%s/%s/%s/versions/%s/platforms",
		url.PathEscape(versionID.OrganizationName),
		url.PathEscape(string(versionID.RegistryName)),
		url.PathEscape(versionID.Namespace),
		url.PathEscape(versionID.Name),
		url.PathEscape(versionID.Version),
	)
	req, err := r.client.NewRequest("POST", u, &options)
	if err != nil {
		return nil, err
	}

	rpp := &RegistryProviderPlatform{}
	err = req.Do(ctx, rpp)
	if err != nil {
		return nil, err
	}

	return rpp, nil
}

// List all provider platforms for a single version
func (r *registryProviderPlatforms) List(ctx context.Context, versionID RegistryProviderVersionID, options *RegistryProviderPlatformListOptions) (*RegistryProviderPlatformList, error) {
	if err := versionID.valid(); err != nil {
		return nil, err
	}
	if err := options.valid(); err != nil {
		return nil, err
	}

	// GET /organizations/:organization_name/registry-providers/:registry_name/:namespace/:name/versions/:version/platforms
	u := fmt.Sprintf(
		"organizations/%s/registry-providers/%s/%s/%s/versions/%s/platforms",
		url.PathEscape(versionID.OrganizationName),
		url.PathEscape(string(versionID.RegistryName)),
		url.PathEscape(versionID.Namespace),
		url.PathEscape(versionID.Name),
		url.PathEscape(versionID.Version),
	)
	req, err := r.client.NewRequest("GET", u, options)
	if err != nil {
		return nil, err
	}

	ppl := &RegistryProviderPlatformList{}
	err = req.Do(ctx, ppl)
	if err != nil {
		return nil, err
	}

	return ppl, nil
}

// Read is used to read an organization's example by ID
func (r *registryProviderPlatforms) Read(ctx context.Context, platformID RegistryProviderPlatformID) (*RegistryProviderPlatform, error) {
	if err := platformID.valid(); err != nil {
		return nil, err
	}

	// GET /organizations/:organization_name/registry-providers/:registry_name/:namespace/:name/versions/:version/platforms/:os/:arch
	u := fmt.Sprintf(
		"organizations/%s/registry-providers/%s/%s/%s/versions/%s/platforms/%s/%s",
		url.PathEscape(platformID.OrganizationName),
		url.PathEscape(string(platformID.RegistryName)),
		url.PathEscape(platformID.Namespace),
		url.PathEscape(platformID.Name),
		url.PathEscape(platformID.Version),
		url.PathEscape(platformID.OS),
		url.PathEscape(platformID.Arch),
	)
	req, err := r.client.NewRequest("GET", u, nil)
	if err != nil {
		return nil, err
	}

	rpp := &RegistryProviderPlatform{}
	err = req.Do(ctx, rpp)

	if err != nil {
		return nil, err
	}

	return rpp, nil
}

// Delete a registry provider platform
func (r *registryProviderPlatforms) Delete(ctx context.Context, platformID RegistryProviderPlatformID) error {
	if err := platformID.valid(); err != nil {
		return err
	}

	// DELETE /organizations/:organization_name/registry-providers/:registry_name/:namespace/:name/versions/:version/platforms/:os/:arch
	u := fmt.Sprintf(
		"organizations/%s/registry-providers/%s/%s/%s/versions/%s/platforms/%s/%s",
		url.PathEscape(platformID.OrganizationName),
		url.PathEscape(string(platformID.RegistryName)),
		url.PathEscape(platformID.Namespace),
		url.PathEscape(platformID.Name),
		url.PathEscape(platformID.Version),
		url.PathEscape(platformID.OS),
		url.PathEscape(platformID.Arch),
	)
	req, err := r.client.NewRequest("DELETE", u, nil)
	if err != nil {
		return err
	}

	return req.Do(ctx, nil)
}

func (id RegistryProviderPlatformID) valid() error {
	if err := id.RegistryProviderID.valid(); err != nil {
		return err
	}
	if !validString(&id.OS) {
		return ErrInvalidOS
	}
	if !validString(&id.Arch) {
		return ErrInvalidArch
	}
	return nil
}

func (o RegistryProviderPlatformCreateOptions) valid() error {
	if !validString(&o.OS) {
		return ErrRequiredOS
	}
	if !validString(&o.Arch) {
		return ErrRequiredArch
	}
	if !validStringID(&o.Shasum) {
		return ErrRequiredShasum
	}
	if !validStringID(&o.Filename) {
		return ErrRequiredFilename
	}
	return nil
}

func (o *RegistryProviderPlatformListOptions) valid() error {
	return nil
}

// Compile-time proof of interface implementation.
var _ RegistryProviderVersions = (*registryProviderVersions)(nil)

// RegistryProviderVersions describes the registry provider version methods that
// the Terraform Enterprise API supports.
//
// TFE API docs: https://developer.hashicorp.com/terraform/cloud-docs/api-docs/private-registry/provider-versions-platforms
type RegistryProviderVersions interface {
	// List all versions for a single provider.
	List(ctx context.Context, providerID RegistryProviderID, options *RegistryProviderVersionListOptions) (*RegistryProviderVersionList, error)

	// Create a registry provider version.
	Create(ctx context.Context, providerID RegistryProviderID, options RegistryProviderVersionCreateOptions) (*RegistryProviderVersion, error)

	// Read a registry provider version.
	Read(ctx context.Context, versionID RegistryProviderVersionID) (*RegistryProviderVersion, error)

	// Delete a registry provider version.
	Delete(ctx context.Context, versionID RegistryProviderVersionID) error
}

// registryProvidersVersions implements RegistryProvidersVersions
type registryProviderVersions struct {
	client *Client
}

// RegistryProviderVersion represents a registry provider version
type RegistryProviderVersion struct {
	ID                 string                             `jsonapi:"primary,registry-provider-versions"`
	Version            string                             `jsonapi:"attr,version"`
	CreatedAt          string                             `jsonapi:"attr,created-at,iso8601"`
	UpdatedAt          string                             `jsonapi:"attr,updated-at,iso8601"`
	KeyID              string                             `jsonapi:"attr,key-id"`
	Protocols          []string                           `jsonapi:"attr,protocols"`
	Permissions        RegistryProviderVersionPermissions `jsonapi:"attr,permissions"`
	ShasumsUploaded    bool                               `jsonapi:"attr,shasums-uploaded"`
	ShasumsSigUploaded bool                               `jsonapi:"attr,shasums-sig-uploaded"`

	// Relations
	RegistryProvider          *RegistryProvider           `jsonapi:"relation,registry-provider"`
	RegistryProviderPlatforms []*RegistryProviderPlatform `jsonapi:"relation,platforms"`

	// Links
	Links map[string]interface{} `jsonapi:"links,omitempty"`
}

// RegistryProviderVersionID is the multi key ID for addressing a version provider
type RegistryProviderVersionID struct {
	RegistryProviderID
	Version string
}

type RegistryProviderVersionPermissions struct {
	CanDelete      bool `jsonapi:"attr,can-delete"`
	CanUploadAsset bool `jsonapi:"attr,can-upload-asset"`
}

type RegistryProviderVersionList struct {
	*Pagination
	Items []*RegistryProviderVersion
}

type RegistryProviderVersionListOptions struct {
	ListOptions
}

type RegistryProviderVersionCreateOptions struct {
	// Required: A valid semver version string.
	Version string `jsonapi:"attr,version"`

	// Required: A valid gpg-key string.
	KeyID string `jsonapi:"attr,key-id"`

	// Required: An array of Terraform provider API versions that this version supports.
	Protocols []string `jsonapi:"attr,protocols"`
}

// List registry provider versions
func (r *registryProviderVersions) List(ctx context.Context, providerID RegistryProviderID, options *RegistryProviderVersionListOptions) (*RegistryProviderVersionList, error) {
	if err := providerID.valid(); err != nil {
		return nil, err
	}
	if err := options.valid(); err != nil {
		return nil, err
	}

	u := fmt.Sprintf(
		"organizations/%s/registry-providers/%s/%s/%s/versions",
		url.PathEscape(providerID.OrganizationName),
		url.PathEscape(string(providerID.RegistryName)),
		url.PathEscape(providerID.Namespace),
		url.PathEscape(providerID.Name),
	)
	req, err := r.client.NewRequest("GET", u, options)
	if err != nil {
		return nil, err
	}

	pvl := &RegistryProviderVersionList{}
	err = req.Do(ctx, pvl)
	if err != nil {
		return nil, err
	}

	return pvl, nil
}

// Create a registry provider version
func (r *registryProviderVersions) Create(ctx context.Context, providerID RegistryProviderID, options RegistryProviderVersionCreateOptions) (*RegistryProviderVersion, error) {
	if err := providerID.valid(); err != nil {
		return nil, err
	}

	if providerID.RegistryName != PrivateRegistry {
		return nil, ErrRequiredPrivateRegistry
	}

	if err := options.valid(); err != nil {
		return nil, err
	}

	u := fmt.Sprintf(
		"organizations/%s/registry-providers/%s/%s/%s/versions",
		url.PathEscape(providerID.OrganizationName),
		url.PathEscape(string(providerID.RegistryName)),
		url.PathEscape(providerID.Namespace),
		url.PathEscape(providerID.Name),
	)
	req, err := r.client.NewRequest("POST", u, &options)
	if err != nil {
		return nil, err
	}

	prvv := &RegistryProviderVersion{}
	err = req.Do(ctx, prvv)
	if err != nil {
		return nil, err
	}

	return prvv, nil
}

// Read a registry provider version
func (r *registryProviderVersions) Read(ctx context.Context, versionID RegistryProviderVersionID) (*RegistryProviderVersion, error) {
	if err := versionID.valid(); err != nil {
		return nil, err
	}

	u := fmt.Sprintf(
		"organizations/%s/registry-providers/%s/%s/%s/versions/%s",
		url.PathEscape(versionID.OrganizationName),
		url.PathEscape(string(versionID.RegistryName)),
		url.PathEscape(versionID.Namespace),
		url.PathEscape(versionID.Name),
		url.PathEscape(versionID.Version),
	)
	req, err := r.client.NewRequest("GET", u, nil)
	if err != nil {
		return nil, err
	}

	prvv := &RegistryProviderVersion{}
	err = req.Do(ctx, prvv)
	if err != nil {
		return nil, err
	}

	return prvv, nil
}

// Delete a registry provider version
func (r *registryProviderVersions) Delete(ctx context.Context, versionID RegistryProviderVersionID) error {
	if err := versionID.valid(); err != nil {
		return err
	}

	u := fmt.Sprintf(
		"organizations/%s/registry-providers/%s/%s/%s/versions/%s",
		url.PathEscape(versionID.OrganizationName),
		url.PathEscape(string(versionID.RegistryName)),
		url.PathEscape(versionID.Namespace),
		url.PathEscape(versionID.Name),
		url.PathEscape(versionID.Version),
	)
	req, err := r.client.NewRequest("DELETE", u, nil)
	if err != nil {
		return err
	}

	return req.Do(ctx, nil)
}

// ShasumsUploadURL returns the upload URL to upload shasums if one is available
func (v *RegistryProviderVersion) ShasumsUploadURL() (string, error) {
	uploadURL, ok := v.Links["shasums-upload"].(string)
	if !ok {
		return uploadURL, fmt.Errorf("the Registry Provider Version does not contain a shasums upload link")
	}
	if uploadURL == "" {
		return uploadURL, fmt.Errorf("the Registry Provider Version shasums upload URL is empty")
	}
	return uploadURL, nil
}

// ShasumsSigUploadURL returns the URL to upload a shasums sig
func (v *RegistryProviderVersion) ShasumsSigUploadURL() (string, error) {
	uploadURL, ok := v.Links["shasums-sig-upload"].(string)
	if !ok {
		return uploadURL, fmt.Errorf("the Registry Provider Version does not contain a shasums sig upload link")
	}
	if uploadURL == "" {
		return uploadURL, fmt.Errorf("the Registry Provider Version shasums sig upload URL is empty")
	}
	return uploadURL, nil
}

// ShasumsDownloadURL returns the URL to download the shasums for the registry version
func (v *RegistryProviderVersion) ShasumsDownloadURL() (string, error) {
	downloadURL, ok := v.Links["shasums-download"].(string)
	if !ok {
		return downloadURL, fmt.Errorf("the Registry Provider Version does not contain a shasums download link")
	}
	if downloadURL == "" {
		return downloadURL, fmt.Errorf("the Registry Provider Version shasums download URL is empty")
	}
	return downloadURL, nil
}

// ShasumsSigDownloadURL returns the URL to download the shasums sig for the registry version
func (v *RegistryProviderVersion) ShasumsSigDownloadURL() (string, error) {
	downloadURL, ok := v.Links["shasums-sig-download"].(string)
	if !ok {
		return downloadURL, fmt.Errorf("the Registry Provider Version does not contain a shasums sig download link")
	}
	if downloadURL == "" {
		return downloadURL, fmt.Errorf("the Registry Provider Version shasums sig download URL is empty")
	}
	return downloadURL, nil
}

func (id RegistryProviderVersionID) valid() error {
	if !validStringID(&id.Version) {
		return ErrInvalidVersion
	}
	if id.RegistryName != PrivateRegistry {
		return ErrRequiredPrivateRegistry
	}
	if err := id.RegistryProviderID.valid(); err != nil {
		return err
	}
	return nil
}

func (o *RegistryProviderVersionListOptions) valid() error {
	return nil
}

func (o RegistryProviderVersionCreateOptions) valid() error {
	if !validStringID(&o.Version) {
		return ErrInvalidVersion
	}
	if !validStringID(&o.KeyID) {
		return ErrInvalidKeyID
	}
	return nil
}

// Compile-time proof of interface implementation.
var _ RegistryProviders = (*registryProviders)(nil)

// RegistryProviders describes all the registry provider-related methods that the Terraform
// Enterprise API supports.
//
// TFE API docs: https://developer.hashicorp.com/terraform/cloud-docs/api-docs/private-registry/providers
type RegistryProviders interface {
	// List all the providers within an organization.
	List(ctx context.Context, organization string, options *RegistryProviderListOptions) (*RegistryProviderList, error)

	// Create a registry provider.
	Create(ctx context.Context, organization string, options RegistryProviderCreateOptions) (*RegistryProvider, error)

	// Update a registry provider. Only tag bindings can be updated on a provider, so the update options are limited to that field.
	Update(ctx context.Context, providerID RegistryProviderID, options *RegistryProviderUpdateOptions) (*RegistryProvider, error)

	// Read a registry provider.
	Read(ctx context.Context, providerID RegistryProviderID, options *RegistryProviderReadOptions) (*RegistryProvider, error)

	// Delete a registry provider.
	Delete(ctx context.Context, providerID RegistryProviderID) error

	// ListTagBindings lists all tag bindings associated with the provider.
	ListTagBindings(ctx context.Context, providerID string) ([]*TagBinding, error)
}

// registryProviders implements RegistryProviders.
type registryProviders struct {
	client *Client
}

// RegistryName represents which registry is being targeted
type RegistryName string

// List of available registry names
const (
	PrivateRegistry RegistryName = "private"
	PublicRegistry  RegistryName = "public"
)

// RegistryProviderIncludeOps represents which jsonapi include can be used with registry providers
type RegistryProviderIncludeOps string

// List of available includes
const (
	RegistryProviderVersionsInclude RegistryProviderIncludeOps = "registry-provider-versions"
)

// RegistryProvider represents a registry provider
type RegistryProvider struct {
	ID           string                      `jsonapi:"primary,registry-providers"`
	Name         string                      `jsonapi:"attr,name"`
	Namespace    string                      `jsonapi:"attr,namespace"`
	CreatedAt    string                      `jsonapi:"attr,created-at,iso8601"`
	UpdatedAt    string                      `jsonapi:"attr,updated-at,iso8601"`
	RegistryName RegistryName                `jsonapi:"attr,registry-name"`
	Permissions  RegistryProviderPermissions `jsonapi:"attr,permissions"`

	// Relations
	Organization             *Organization              `jsonapi:"relation,organization"`
	RegistryProviderVersions []*RegistryProviderVersion `jsonapi:"relation,registry-provider-versions"`
	TagBindings              []*TagBinding              `jsonapi:"relation,tag-bindings,omitempty"`

	// Links
	Links map[string]interface{} `jsonapi:"links,omitempty"`
}

type RegistryProviderPermissions struct {
	CanDelete bool `jsonapi:"attr,can-delete"`
}

type RegistryProviderListOptions struct {
	ListOptions

	// Optional: A query string to filter by registry_name
	RegistryName RegistryName `url:"filter[registry_name],omitempty"`

	// Optional: A query string to filter by organization
	OrganizationName string `url:"filter[organization_name],omitempty"`

	// Optional: A query string to do a fuzzy search
	Search string `url:"q,omitempty"`

	// Optional: Include related jsonapi relationships
	Include *[]RegistryProviderIncludeOps `url:"include,omitempty"`
}

type RegistryProviderList struct {
	*Pagination
	Items []*RegistryProvider
}

// RegistryProviderID is the multi key ID for addressing a provider
type RegistryProviderID struct {
	// The unique ID of the provider. If given, the other fields are ignored.
	ID               string
	OrganizationName string
	RegistryName     RegistryName
	Namespace        string
	Name             string
}

// RegistryProviderCreateOptions is used when creating a registry provider
type RegistryProviderCreateOptions struct {
	// Type is a public field utilized by JSON:API to
	// set the resource type via the field tag.
	// It is not a user-defined value and does not need to be set.
	// https://jsonapi.org/format/#crud-creating
	Type string `jsonapi:"primary,registry-providers"`

	// Required: The name of the registry provider
	Name string `jsonapi:"attr,name"`

	// Required: The namespace of the provider. For private providers, this is the same as the organization name
	Namespace string `jsonapi:"attr,namespace"`

	// Required: Whether this is a publicly maintained provider or private. Must be either public or private.
	RegistryName RegistryName `jsonapi:"attr,registry-name"`
}

// RegistryProviderUpdateOptions is used when creating a registry provider
type RegistryProviderUpdateOptions struct {
	// Optional: Tag bindings for the registry provider. Note that this
	// will replace all existing tag bindings.
	TagBindings []*TagBinding `jsonapi:"relation,tag-bindings"`
}

type RegistryProviderReadOptions struct {
	// Optional: Include related jsonapi relationships
	Include []RegistryProviderIncludeOps `url:"include,omitempty"`
}

func (r *registryProviders) List(ctx context.Context, organization string, options *RegistryProviderListOptions) (*RegistryProviderList, error) {
	if !validStringID(&organization) {
		return nil, ErrInvalidOrg
	}
	if err := options.valid(); err != nil {
		return nil, err
	}

	u := fmt.Sprintf("organizations/%s/registry-providers", url.PathEscape(organization))
	req, err := r.client.NewRequest("GET", u, options)
	if err != nil {
		return nil, err
	}

	pl := &RegistryProviderList{}
	err = req.Do(ctx, pl)
	if err != nil {
		return nil, err
	}

	return pl, nil
}

func (r *registryProviders) Create(ctx context.Context, organization string, options RegistryProviderCreateOptions) (*RegistryProvider, error) {
	if !validStringID(&organization) {
		return nil, ErrInvalidOrg
	}

	if err := options.valid(); err != nil {
		return nil, err
	}

	u := fmt.Sprintf(
		"organizations/%s/registry-providers",
		url.PathEscape(organization),
	)
	req, err := r.client.NewRequest("POST", u, &options)
	if err != nil {
		return nil, err
	}

	prv := &RegistryProvider{}
	err = req.Do(ctx, prv)
	if err != nil {
		return nil, err
	}

	return prv, nil
}

func (r *registryProviders) Update(ctx context.Context, providerID RegistryProviderID, options *RegistryProviderUpdateOptions) (*RegistryProvider, error) {
	if err := providerID.valid(); err != nil {
		return nil, err
	}

	u := fmt.Sprintf(
		"organizations/%s/registry-providers/%s/%s/%s",
		url.PathEscape(providerID.OrganizationName),
		url.PathEscape(string(providerID.RegistryName)),
		url.PathEscape(providerID.Namespace),
		url.PathEscape(providerID.Name),
	)
	req, err := r.client.NewRequest("PATCH", u, options)
	if err != nil {
		return nil, err
	}
	prv := &RegistryProvider{}
	err = req.Do(ctx, prv)
	if err != nil {
		return nil, err
	}

	return prv, nil
}

func (r *registryProviders) ListTagBindings(ctx context.Context, providerID string) ([]*TagBinding, error) {
	if !validStringID(&providerID) {
		return nil, ErrInvalidRegistryProviderID
	}

	u := fmt.Sprintf("registry-providers/%s/tag-bindings", url.PathEscape(providerID))
	req, err := r.client.NewRequest("GET", u, nil)
	if err != nil {
		return nil, err
	}

	var list struct {
		*Pagination
		Items []*TagBinding
	}

	err = req.Do(ctx, &list)
	if err != nil {
		return nil, err
	}

	return list.Items, nil
}

func (r *registryProviders) Read(ctx context.Context, providerID RegistryProviderID, options *RegistryProviderReadOptions) (*RegistryProvider, error) {
	if err := providerID.valid(); err != nil {
		return nil, err
	}

	var u string
	if providerID.ID == "" {
		u = fmt.Sprintf(
			"organizations/%s/registry-providers/%s/%s/%s",
			url.PathEscape(providerID.OrganizationName),
			url.PathEscape(string(providerID.RegistryName)),
			url.PathEscape(providerID.Namespace),
			url.PathEscape(providerID.Name),
		)
	} else {
		u = fmt.Sprintf("registry-providers/%s", url.PathEscape(providerID.ID))
	}

	req, err := r.client.NewRequest("GET", u, options)
	if err != nil {
		return nil, err
	}

	prv := &RegistryProvider{}
	err = req.Do(ctx, prv)
	if err != nil {
		return nil, err
	}

	return prv, nil
}

func (r *registryProviders) Delete(ctx context.Context, providerID RegistryProviderID) error {
	if err := providerID.valid(); err != nil {
		return err
	}

	u := fmt.Sprintf(
		"organizations/%s/registry-providers/%s/%s/%s",
		url.PathEscape(providerID.OrganizationName),
		url.PathEscape(string(providerID.RegistryName)),
		url.PathEscape(providerID.Namespace),
		url.PathEscape(providerID.Name),
	)
	req, err := r.client.NewRequest("DELETE", u, nil)
	if err != nil {
		return err
	}

	return req.Do(ctx, nil)
}

func (o RegistryProviderCreateOptions) valid() error {
	if !validStringID(&o.Name) {
		return ErrInvalidName
	}
	if !validStringID(&o.Namespace) {
		return ErrInvalidNamespace
	}
	return nil
}

func (id RegistryProviderID) valid() error {
	if validString(&id.ID) && validStringID(&id.ID) {
		return nil
	}
	if !validStringID(&id.OrganizationName) {
		return ErrInvalidOrg
	}
	if !validStringID(&id.Name) {
		return ErrInvalidName
	}
	if !validStringID(&id.Namespace) {
		return ErrInvalidNamespace
	}
	if !validStringID((*string)(&id.RegistryName)) {
		return ErrInvalidRegistryName
	}
	return nil
}

func (o *RegistryProviderListOptions) valid() error {
	return nil
}

// ContextWithResponseHeaderHook returns a context that will, if passed to
// [ClientRequest.Do] or to any of the wrapper methods that call it, arrange
// for the given callback to be called with the headers from the raw HTTP
// response.
//
// This is intended for allowing callers to respond to out-of-band metadata
// such as cache-control-related headers, rate limiting headers, etc. Hooks
// must not modify the given [http.Header] or otherwise attempt to change how
// the response is handled by [ClientRequest.Do].
//
// If the given context already has a response header hook then the returned
// context will call both the existing hook and the newly-provided one, with
// the newer being called first.
func ContextWithResponseHeaderHook(parentCtx context.Context, cb func(status int, header http.Header)) context.Context {
	// If the given context already has a notification callback then we'll
	// arrange to notify both the previous and the new one. This is not
	// a super efficient way to achieve that but we expect it to be rare
	// for there to be more than one or two hooks associated with a particular
	// request, so it's not warranted to optimize this further.
	existingI := parentCtx.Value(contextResponseHeaderHookKey)
	finalCb := cb
	if existingI != nil {
		existing, ok := existingI.(func(int, http.Header))
		// This explicit check-and-panic is redundant but required by our linter.
		if !ok {
			panic(fmt.Sprintf("context has response header hook of invalid type %T", existingI))
		}
		finalCb = func(status int, header http.Header) {
			cb(status, header)
			existing(status, header)
		}
	}
	return context.WithValue(parentCtx, contextResponseHeaderHookKey, finalCb)
}

func contextResponseHeaderHook(ctx context.Context) (func(int, http.Header), error) {
	cbI := ctx.Value(contextResponseHeaderHookKey)
	if cbI == nil {
		// Stub callback that does absolutely nothing, then.
		return func(int, http.Header) {}, nil
	}

	cb, ok := cbI.(func(int, http.Header))
	if !ok {
		return nil, fmt.Errorf("context has response header hook of invalid type %T", cbI)
	}

	return cb, nil
}

// contextResponseHeaderHookKey is the type of the internal key used to store
// the callback for [ContextWithResponseHeaderHook] inside a [context.Context]
// object.
type contextResponseHeaderHookKeyType struct{}

// contextResponseHeaderHookKey is the internal key used to store the callback
// for [ContextWithResponseHeaderHook] inside a [context.Context] object.
var contextResponseHeaderHookKey contextResponseHeaderHookKeyType

// ClientRequest encapsulates a request sent by the Client
type ClientRequest struct {
	retryableRequest *retryablehttp.Request
	http             *retryablehttp.Client
	limiter          *rate.Limiter

	// Header are the headers that will be sent in this request
	Header http.Header
}

func (r ClientRequest) Do(ctx context.Context, model interface{}) error {
	// Wait will block until the limiter can obtain a new token
	// or returns an error if the given context is canceled.
	if r.limiter != nil {
		if err := r.limiter.Wait(ctx); err != nil {
			return err
		}
	}

	// If the caller provided a response header hook then we'll call it
	// once we have a response.
	respHeaderHook, err := contextResponseHeaderHook(ctx)
	if err != nil {
		return err
	}

	// Add the context to the request.
	reqWithCxt := r.retryableRequest.WithContext(ctx)

	// Execute the request and check the response.
	resp, err := r.http.Do(reqWithCxt)
	if resp != nil {
		// We call the callback whenever there's any sort of response,
		// even if it's returned in conjunction with an error.
		respHeaderHook(resp.StatusCode, resp.Header)
	}
	if err != nil {
		// If we got an error, and the context has been canceled,
		// the context's error is probably more useful.
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			return err
		}
	}
	defer resp.Body.Close() //nolint:errcheck

	// Basic response checking.
	if err := checkResponseCode(resp); err != nil {
		return err
	}

	// Return here if decoding the response isn't needed.
	if model == nil {
		return nil
	}

	// If v implements io.Writer, write the raw response body.
	if w, ok := model.(io.Writer); ok {
		_, err := io.Copy(w, resp.Body)
		return err
	}

	return unmarshalResponse(resp.Body, model)
}

// DoJSON is similar to Do except that it should be used when a plain JSON response is expected
// as opposed to json-api.
func (r *ClientRequest) DoJSON(ctx context.Context, model any) error {
	// Wait will block until the limiter can obtain a new token
	// or returns an error if the given context is canceled.
	if r.limiter != nil {
		if err := r.limiter.Wait(ctx); err != nil {
			return err
		}
	}

	// Add the context to the request.
	contextReq := r.retryableRequest.WithContext(ctx)

	// If the caller provided a response header hook then we'll call it
	// once we have a response.
	respHeaderHook, err := contextResponseHeaderHook(ctx)
	if err != nil {
		return err
	}

	// Execute the request and check the response.
	resp, err := r.http.Do(contextReq)
	if resp != nil {
		// We call the callback whenever there's any sort of response,
		// even if it's returned in conjunction with an error.
		respHeaderHook(resp.StatusCode, resp.Header)
	}
	if err != nil {
		// If we got an error, and the context has been canceled,
		// the context's error is probably more useful.
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			return err
		}
	}
	defer resp.Body.Close() //nolint:errcheck

	if resp.StatusCode < 200 || resp.StatusCode >= 400 {
		return fmt.Errorf("error HTTP response: %d", resp.StatusCode)
	} else if resp.StatusCode == 304 {
		// Got a "Not Modified" response, but we can't return a model because there is no response body.
		// This is necessary to support the IPRanges endpoint, which has the peculiar behavior
		// of not returning content but allowing a 304 response by optionally sending an
		// If-Modified-Since header.
		return nil
	}

	// Return here if decoding the response isn't needed.
	if model == nil {
		return nil
	}

	// If v implements io.Writer, write the raw response body.
	if w, ok := model.(io.Writer); ok {
		_, err := io.Copy(w, resp.Body)
		return err
	}

	return json.NewDecoder(resp.Body).Decode(model)
}

// DoRaw exposes the underlying io.ReadCloser for the response body.
// The caller is responsible for closing the ReadCloser and unmarshaling the
// results.
func (r *ClientRequest) DoRaw(ctx context.Context) (io.ReadCloser, error) {
	// Wait will block until the limiter can obtain a new token
	// or returns an error if the given context is canceled.
	if r.limiter != nil {
		if err := r.limiter.Wait(ctx); err != nil {
			return nil, err
		}
	}

	// Add the context to the request.
	contextReq := r.retryableRequest.WithContext(ctx)

	// If the caller provided a response header hook then we'll call it
	// once we have a response.
	respHeaderHook, err := contextResponseHeaderHook(ctx)
	if err != nil {
		return nil, err
	}

	// Execute the request and check the response.
	resp, err := r.http.Do(contextReq)
	if resp != nil {
		// We call the callback whenever there's any sort of response,
		// even if it's returned in conjunction with an error.
		respHeaderHook(resp.StatusCode, resp.Header)
	}
	if err != nil {
		// If we got an error, and the context has been canceled,
		// the context's error is probably more useful.
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			return nil, err
		}
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 400 {
		// Close the body here since we won't be returning it to the caller.
		resp.Body.Close() //nolint:errcheck
		return nil, fmt.Errorf("error HTTP response: %d", resp.StatusCode)
	} else if resp.StatusCode == 304 {
		// Got a "Not Modified" response, but we can't return a model because there is no response body.
		// This is necessary to support the IPRanges endpoint, which has the peculiar behavior
		// of not returning content but allowing a 304 response by optionally sending an
		// If-Modified-Since header.
		return nil, nil
	}

	return resp.Body, nil
}

// Compile-time proof of interface implementation.
var _ ReservedTagKeys = (*reservedTagKeys)(nil)

// ReservedTagKeys describes all the reserved tag key endpoints that the
// Terraform Enterprise API supports.
//
// TFE API docs: https://developer.hashicorp.com/terraform/cloud-docs/api-docs/reserved-tag-keys
type ReservedTagKeys interface {
	// List all the reserved tag keys for the given organization.
	List(ctx context.Context, organization string, options *ReservedTagKeyListOptions) (*ReservedTagKeyList, error)

	// Create a new reserved tag key for the given organization.
	Create(ctx context.Context, organization string, options ReservedTagKeyCreateOptions) (*ReservedTagKey, error)

	// Update the reserved tag key with the given ID.
	Update(ctx context.Context, reservedTagKeyID string, options ReservedTagKeyUpdateOptions) (*ReservedTagKey, error)

	// Delete the reserved tag key with the given ID.
	Delete(ctx context.Context, reservedTagKeyID string) error
}

// reservedTagKeys implements ReservedTagKeys.
type reservedTagKeys struct {
	client *Client
}

// ReservedTagKeyList represents a list of reserved tag keys.
type ReservedTagKeyList struct {
	*Pagination
	Items []*ReservedTagKey
}

// ReservedTagKey represents a Terraform Enterprise reserved tag key.
type ReservedTagKey struct {
	ID               string    `jsonapi:"primary,reserved-tag-keys"`
	Key              string    `jsonapi:"attr,key"`
	DisableOverrides bool      `jsonapi:"attr,disable-overrides"`
	CreatedAt        time.Time `jsonapi:"attr,created_at,iso8601"`
	UpdatedAt        time.Time `jsonapi:"attr,updated_at,iso8601"`
}

// ReservedTagKeyListOptions represents the options for listing reserved tag
// keys.
type ReservedTagKeyListOptions struct {
	ListOptions
}

// List all the reserved tag keys for the given organization.
func (s *reservedTagKeys) List(ctx context.Context, organization string, options *ReservedTagKeyListOptions) (*ReservedTagKeyList, error) {
	if !validStringID(&organization) {
		return nil, ErrInvalidOrg
	}

	u := fmt.Sprintf("organizations/%s/reserved-tag-keys", url.PathEscape(organization))

	req, err := s.client.NewRequest("GET", u, options)
	if err != nil {
		return nil, err
	}

	rl := &ReservedTagKeyList{}
	err = req.Do(ctx, rl)
	if err != nil {
		return nil, err
	}

	return rl, nil
}

// ReservedTagKeyCreateOptions represents the options for creating a
// reserved tag key.
type ReservedTagKeyCreateOptions struct {
	// Type is a public field utilized by JSON:API to
	// set the resource type via the field tag.
	// It is not a user-defined value and does not need to be set.
	// https://jsonapi.org/format/#crud-creating
	Type string `jsonapi:"primary,reserved-tag-keys"`

	// Required: The reserved tag key's key string.
	Key string `jsonapi:"attr,key"`

	// Optional: When true, project tag bindings that match this reserved tag key can not
	// be overridden at the workspace level.
	DisableOverrides *bool `jsonapi:"attr,disable-overrides,omitempty"`
}

func (o ReservedTagKeyCreateOptions) valid() error {
	if !validString(&o.Key) {
		return ErrRequiredKey
	}
	return nil
}

// Create a reserved tag key.
func (s *reservedTagKeys) Create(ctx context.Context, organization string, options ReservedTagKeyCreateOptions) (*ReservedTagKey, error) {
	if !validStringID(&organization) {
		return nil, ErrInvalidOrg
	}

	if err := options.valid(); err != nil {
		return nil, err
	}

	u := fmt.Sprintf("organizations/%s/reserved-tag-keys", url.PathEscape(organization))
	req, err := s.client.NewRequest("POST", u, &options)
	if err != nil {
		return nil, err
	}

	r := &ReservedTagKey{}
	err = req.Do(ctx, r)
	if err != nil {
		return nil, err
	}

	return r, nil
}

// ReservedTagKeyUpdateOptions represents the options for updating a
// reserved tag key.
type ReservedTagKeyUpdateOptions struct {
	// Type is a public field utilized by JSON:API to
	// set the resource type via the field tag.
	// It is not a user-defined value and does not need to be set.
	// https://jsonapi.org/format/#crud-creating
	Type string `jsonapi:"primary,reserved-tag-keys"`

	// Optional: The reserved tag key's key string.
	Key *string `jsonapi:"attr,key,omitempty"`

	// Optional: When true, project tag bindings that match this reserved tag key can not
	// be overridden at the workspace level.
	DisableOverrides *bool `jsonapi:"attr,disable-overrides,omitempty"`
}

// Update the reserved tag key with the given ID.
func (s *reservedTagKeys) Update(ctx context.Context, reservedTagKeyID string, options ReservedTagKeyUpdateOptions) (*ReservedTagKey, error) {
	if !validStringID(&reservedTagKeyID) {
		return nil, ErrInvalidReservedTagKeyID
	}

	u := fmt.Sprintf("reserved-tag-keys/%s", url.PathEscape(reservedTagKeyID))
	req, err := s.client.NewRequest("PATCH", u, &options)
	if err != nil {
		return nil, err
	}

	r := &ReservedTagKey{}
	err = req.Do(ctx, r)
	if err != nil {
		return nil, err
	}

	return r, nil
}

// Delete the reserved tag key with the given ID.
func (s *reservedTagKeys) Delete(ctx context.Context, reservedTagKeyID string) error {
	if !validStringID(&reservedTagKeyID) {
		return ErrInvalidReservedTagKeyID
	}

	u := fmt.Sprintf("reserved-tag-keys/%s", url.PathEscape(reservedTagKeyID))
	req, err := s.client.NewRequest("DELETE", u, nil)
	if err != nil {
		return err
	}

	return req.Do(ctx, nil)
}

// Compile-time proof of interface implementation.
var _ RunEvents = (*runEvents)(nil)

// RunEvents describes all the run events that the Terraform Enterprise
// API supports.
//
// TFE API docs: https://developer.hashicorp.com/terraform/cloud-docs/api-docs/run
type RunEvents interface {
	// List all the runs events of the given run.
	List(ctx context.Context, runID string, options *RunEventListOptions) (*RunEventList, error)

	// Read a run event by its ID.
	Read(ctx context.Context, runEventID string) (*RunEvent, error)

	// ReadWithOptions reads a run event by its ID using the options supplied
	ReadWithOptions(ctx context.Context, runEventID string, options *RunEventReadOptions) (*RunEvent, error)
}

// runEvents implements RunEvents.
type runEvents struct {
	client *Client
}

// RunEventList represents a list of run events.
type RunEventList struct {
	// Pagination is not supported by the API
	*Pagination
	Items []*RunEvent
}

// RunEvent represents a Terraform Enterprise run event.
type RunEvent struct {
	ID          string    `jsonapi:"primary,run-events"`
	Action      string    `jsonapi:"attr,action"`
	CreatedAt   time.Time `jsonapi:"attr,created-at,iso8601"`
	Description string    `jsonapi:"attr,description"`

	// Relations - Note that `target` is not supported yet
	Actor   *User    `jsonapi:"relation,actor"`
	Comment *Comment `jsonapi:"relation,comment"`
}

// RunEventIncludeOpt represents the available options for include query params.
type RunEventIncludeOpt string

const (
	RunEventComment RunEventIncludeOpt = "comment"
	RunEventActor   RunEventIncludeOpt = "actor"
)

// RunEventListOptions represents the options for listing run events.
type RunEventListOptions struct {
	// Optional: A list of relations to include. See available resources:
	Include []RunEventIncludeOpt `url:"include,omitempty"`
}

// RunEventReadOptions represents the options for reading a run event.
type RunEventReadOptions struct {
	// Optional: A list of relations to include. See available resources:
	Include []RunEventIncludeOpt `url:"include,omitempty"`
}

// List all the run events of the given run.
func (s *runEvents) List(ctx context.Context, runID string, options *RunEventListOptions) (*RunEventList, error) {
	if !validStringID(&runID) {
		return nil, ErrInvalidRunID
	}
	if err := options.valid(); err != nil {
		return nil, err
	}

	u := fmt.Sprintf("runs/%s/run-events", url.PathEscape(runID))

	req, err := s.client.NewRequest("GET", u, options)
	if err != nil {
		return nil, err
	}

	rl := &RunEventList{}
	err = req.Do(ctx, rl)
	if err != nil {
		return nil, err
	}

	return rl, nil
}

// Read a run by its ID.
func (s *runEvents) Read(ctx context.Context, runEventID string) (*RunEvent, error) {
	return s.ReadWithOptions(ctx, runEventID, nil)
}

// ReadWithOptions reads a run by its ID with the given options.
func (s *runEvents) ReadWithOptions(ctx context.Context, runEventID string, options *RunEventReadOptions) (*RunEvent, error) {
	if !validStringID(&runEventID) {
		return nil, ErrInvalidRunEventID
	}
	if err := options.valid(); err != nil {
		return nil, err
	}

	u := fmt.Sprintf("run-events/%s", url.PathEscape(runEventID))
	req, err := s.client.NewRequest("GET", u, options)
	if err != nil {
		return nil, err
	}

	r := &RunEvent{}
	err = req.Do(ctx, r)
	if err != nil {
		return nil, err
	}

	return r, nil
}

func (o *RunEventReadOptions) valid() error {
	return nil
}

func (o *RunEventListOptions) valid() error {
	return nil
}

// RunTaskRequest is the payload object that TFC/E sends to the Run Task's URL.
// https://developer.hashicorp.com/terraform/enterprise/api-docs/run-tasks/run-tasks-integration#common-properties
type RunTaskRequest struct {
	AccessToken                     string                      `json:"access_token"`
	Capabilitites                   RunTaskRequestCapabilitites `json:"capabilitites,omitempty"`
	ConfigurationVersionDownloadURL string                      `json:"configuration_version_download_url,omitempty"`
	ConfigurationVersionID          string                      `json:"configuration_version_id,omitempty"`
	IsSpeculative                   bool                        `json:"is_speculative"`
	OrganizationName                string                      `json:"organization_name"`
	PayloadVersion                  int                         `json:"payload_version"`
	PlanJSONAPIURL                  string                      `json:"plan_json_api_url,omitempty"` // Specific to post_plan, pre_apply or post_apply stage
	RunAppURL                       string                      `json:"run_app_url"`
	RunCreatedAt                    time.Time                   `json:"run_created_at"`
	RunCreatedBy                    string                      `json:"run_created_by"`
	RunID                           string                      `json:"run_id"`
	RunMessage                      string                      `json:"run_message"`
	Stage                           string                      `json:"stage"`
	TaskResultCallbackURL           string                      `json:"task_result_callback_url"`
	TaskResultEnforcementLevel      string                      `json:"task_result_enforcement_level"`
	TaskResultID                    string                      `json:"task_result_id"`
	VcsBranch                       string                      `json:"vcs_branch,omitempty"`
	VcsCommitURL                    string                      `json:"vcs_commit_url,omitempty"`
	VcsPullRequestURL               string                      `json:"vcs_pull_request_url,omitempty"`
	VcsRepoURL                      string                      `json:"vcs_repo_url,omitempty"`
	WorkspaceAppURL                 string                      `json:"workspace_app_url"`
	WorkspaceID                     string                      `json:"workspace_id"`
	WorkspaceName                   string                      `json:"workspace_name"`
	WorkspaceWorkingDirectory       string                      `json:"workspace_working_directory,omitempty"`
}

// RunTaskRequestCapabilitites defines the capabilities that the caller supports.
type RunTaskRequestCapabilitites struct {
	Outcomes bool `json:"outcomes"`
}

// Compile-time proof of interface implementation
var _ RunTasks = (*runTasks)(nil)

// RunTasks represents all the run task related methods in the context of an organization
// that the HCP Terraform and Terraform Enterprise API supports.
// https://developer.hashicorp.com/terraform/cloud-docs/api-docs/run-tasks/run-tasks#run-tasks-api
type RunTasks interface {
	// Create a run task for an organization
	Create(ctx context.Context, organization string, options RunTaskCreateOptions) (*RunTask, error)

	// List all run tasks for an organization
	List(ctx context.Context, organization string, options *RunTaskListOptions) (*RunTaskList, error)

	// Read an organization's run task by ID
	Read(ctx context.Context, runTaskID string) (*RunTask, error)

	// Read an organization's run task by ID with given options
	ReadWithOptions(ctx context.Context, runTaskID string, options *RunTaskReadOptions) (*RunTask, error)

	// Update a run task for an organization
	Update(ctx context.Context, runTaskID string, options RunTaskUpdateOptions) (*RunTask, error)

	// Delete an organization's run task
	Delete(ctx context.Context, runTaskID string) error

	// Attach a run task to an organization's workspace
	AttachToWorkspace(ctx context.Context, workspaceID string, runTaskID string, enforcementLevel TaskEnforcementLevel) (*WorkspaceRunTask, error)
}

// runTasks implements RunTasks
type runTasks struct {
	client *Client
}

// RunTask represents a HCP Terraform or Terraform Enterprise run task
type RunTask struct {
	ID          string         `jsonapi:"primary,tasks"`
	Name        string         `jsonapi:"attr,name"`
	URL         string         `jsonapi:"attr,url"`
	Description string         `jsonapi:"attr,description"`
	Category    string         `jsonapi:"attr,category"`
	HMACKey     *string        `jsonapi:"attr,hmac-key,omitempty"`
	Enabled     bool           `jsonapi:"attr,enabled"`
	Global      *GlobalRunTask `jsonapi:"attr,global-configuration,omitempty"`

	AgentPool         *AgentPool          `jsonapi:"relation,agent-pool"`
	Organization      *Organization       `jsonapi:"relation,organization"`
	WorkspaceRunTasks []*WorkspaceRunTask `jsonapi:"relation,workspace-tasks"`
}

// GlobalRunTask represents the global configuration of a HCP Terraform or Terraform Enterprise run task
type GlobalRunTask struct {
	Enabled          bool                 `jsonapi:"attr,enabled"`
	Stages           []Stage              `jsonapi:"attr,stages"`
	EnforcementLevel TaskEnforcementLevel `jsonapi:"attr,enforcement-level"`
}

// RunTaskList represents a list of run tasks
type RunTaskList struct {
	*Pagination
	Items []*RunTask
}

// RunTaskIncludeOpt represents the available options for include query params.
// https://developer.hashicorp.com/terraform/cloud-docs/api-docs/run-tasks/run-tasks#list-run-tasks
type RunTaskIncludeOpt string

const (
	RunTaskWorkspaceTasks RunTaskIncludeOpt = "workspace_tasks"
	RunTaskWorkspace      RunTaskIncludeOpt = "workspace_tasks.workspace"
)

// RunTaskListOptions represents the set of options for listing run tasks
type RunTaskListOptions struct {
	ListOptions
	// Optional: A list of relations to include with a run task. See available resources:
	// https://developer.hashicorp.com/terraform/cloud-docs/api-docs/run-tasks/run-tasks#list-run-tasks
	Include []RunTaskIncludeOpt `url:"include,omitempty"`
}

// RunTaskReadOptions represents the set of options for reading a run task
type RunTaskReadOptions struct {
	// Optional: A list of relations to include with a run task. See available resources:
	// https://developer.hashicorp.com/terraform/cloud-docs/api-docs/run-tasks/run-tasks#list-run-tasks
	Include []RunTaskIncludeOpt `url:"include,omitempty"`
}

// GlobalRunTask represents the optional global configuration of a HCP Terraform or Terraform Enterprise run task
type GlobalRunTaskOptions struct {
	Enabled          *bool                 `jsonapi:"attr,enabled,omitempty"`
	Stages           *[]Stage              `jsonapi:"attr,stages,omitempty"`
	EnforcementLevel *TaskEnforcementLevel `jsonapi:"attr,enforcement-level,omitempty"`
}

// RunTaskCreateOptions represents the set of options for creating a run task
type RunTaskCreateOptions struct {
	// Type is a public field utilized by JSON:API to
	// set the resource type via the field tag.
	// It is not a user-defined value and does not need to be set.
	// https://jsonapi.org/format/#crud-creating
	Type string `jsonapi:"primary,tasks"`

	// Required: The name of the run task
	Name string `jsonapi:"attr,name"`

	// Required: The URL to send a run task payload
	URL string `jsonapi:"attr,url"`

	// Optional: Description of the task
	Description *string `jsonapi:"attr,description"`

	// Required: Must be "task"
	Category string `jsonapi:"attr,category"`

	// Optional: An HMAC key to verify the run task
	HMACKey *string `jsonapi:"attr,hmac-key,omitempty"`

	// Optional: Whether the task should be enabled
	Enabled *bool `jsonapi:"attr,enabled,omitempty"`

	// Optional: Whether the task contains global configuration
	Global *GlobalRunTaskOptions `jsonapi:"attr,global-configuration,omitempty"`

	// Optional: Whether the task will be executed using an Agent Pool
	// Requires the PrivateRunTasks entitlement
	AgentPool *AgentPool `jsonapi:"relation,agent-pool,omitempty"`
}

// RunTaskUpdateOptions represents the set of options for updating an organization's run task
type RunTaskUpdateOptions struct {
	// Type is a public field utilized by JSON:API to
	// set the resource type via the field tag.
	// It is not a user-defined value and does not need to be set.
	// https://jsonapi.org/format/#crud-creating
	Type string `jsonapi:"primary,tasks"`

	// Optional: The name of the run task, defaults to previous value
	Name *string `jsonapi:"attr,name,omitempty"`

	// Optional: The URL to send a run task payload, defaults to previous value
	URL *string `jsonapi:"attr,url,omitempty"`

	// Optional: An optional description of the task
	Description *string `jsonapi:"attr,description,omitempty"`

	// Optional: Must be "task", defaults to "task"
	Category *string `jsonapi:"attr,category,omitempty"`

	// Optional: An HMAC key to verify the run task
	HMACKey *string `jsonapi:"attr,hmac-key,omitempty"`

	// Optional: Whether the task should be enabled
	Enabled *bool `jsonapi:"attr,enabled,omitempty"`

	// Optional: Whether the task contains global configuration
	Global *GlobalRunTaskOptions `jsonapi:"attr,global-configuration,omitempty"`

	// Optional: Whether the task will be executed using an Agent Pool
	// Requires the PrivateRunTasks entitlement
	AgentPool *AgentPool `jsonapi:"relation,agent-pool,omitempty"`
}

// Create is used to create a new run task for an organization
func (s *runTasks) Create(ctx context.Context, organization string, options RunTaskCreateOptions) (*RunTask, error) {
	if !validStringID(&organization) {
		return nil, ErrInvalidOrg
	}

	if err := options.valid(); err != nil {
		return nil, err
	}

	u := fmt.Sprintf("organizations/%s/tasks", url.PathEscape(organization))
	req, err := s.client.NewRequest("POST", u, &options)
	if err != nil {
		return nil, err
	}

	r := &internalRunTask{}
	err = req.Do(ctx, r)
	if err != nil {
		return nil, err
	}

	return r.ToRunTask(), nil
}

// List all the run tasks for an organization
func (s *runTasks) List(ctx context.Context, organization string, options *RunTaskListOptions) (*RunTaskList, error) {
	if !validStringID(&organization) {
		return nil, ErrInvalidOrg
	}
	if err := options.valid(); err != nil {
		return nil, err
	}

	u := fmt.Sprintf("organizations/%s/tasks", url.PathEscape(organization))
	req, err := s.client.NewRequest("GET", u, options)
	if err != nil {
		return nil, err
	}

	rl := &internalRunTaskList{}
	err = req.Do(ctx, rl)
	if err != nil {
		return nil, err
	}

	return rl.ToRunTaskList(), nil
}

// Read is used to read an organization's run task by ID
func (s *runTasks) Read(ctx context.Context, runTaskID string) (*RunTask, error) {
	return s.ReadWithOptions(ctx, runTaskID, nil)
}

// Read is used to read an organization's run task by ID with options
func (s *runTasks) ReadWithOptions(ctx context.Context, runTaskID string, options *RunTaskReadOptions) (*RunTask, error) {
	if !validStringID(&runTaskID) {
		return nil, ErrInvalidRunTaskID
	}
	if err := options.valid(); err != nil {
		return nil, err
	}

	u := fmt.Sprintf("tasks/%s", url.PathEscape(runTaskID))
	req, err := s.client.NewRequest("GET", u, options)
	if err != nil {
		return nil, err
	}

	r := &internalRunTask{}
	err = req.Do(ctx, r)
	if err != nil {
		return nil, err
	}

	return r.ToRunTask(), nil
}

// Update an existing run task for an organization by ID
func (s *runTasks) Update(ctx context.Context, runTaskID string, options RunTaskUpdateOptions) (*RunTask, error) {
	if !validStringID(&runTaskID) {
		return nil, ErrInvalidRunTaskID
	}

	if err := options.valid(); err != nil {
		return nil, err
	}

	u := fmt.Sprintf("tasks/%s", url.PathEscape(runTaskID))
	req, err := s.client.NewRequest("PATCH", u, &options)
	if err != nil {
		return nil, err
	}

	r := &internalRunTask{}
	err = req.Do(ctx, r)
	if err != nil {
		return nil, err
	}

	return r.ToRunTask(), nil
}

// Delete an existing run task for an organization by ID
func (s *runTasks) Delete(ctx context.Context, runTaskID string) error {
	if !validStringID(&runTaskID) {
		return ErrInvalidRunTaskID
	}

	u := fmt.Sprintf("tasks/%s", runTaskID)
	req, err := s.client.NewRequest("DELETE", u, nil)
	if err != nil {
		return err
	}

	return req.Do(ctx, nil)
}

// AttachToWorkspace is a convenient method to attach a run task to a workspace. See: WorkspaceRunTasks.Create()
func (s *runTasks) AttachToWorkspace(ctx context.Context, workspaceID, runTaskID string, enforcement TaskEnforcementLevel) (*WorkspaceRunTask, error) {
	return s.client.WorkspaceRunTasks.Create(ctx, workspaceID, WorkspaceRunTaskCreateOptions{
		EnforcementLevel: enforcement,
		RunTask:          &RunTask{ID: runTaskID},
	})
}

func (o *RunTaskCreateOptions) valid() error {
	if !validString(&o.Name) {
		return ErrRequiredName
	}

	if !validString(&o.URL) {
		return ErrInvalidRunTaskURL
	}

	if o.Category != "task" {
		return ErrInvalidRunTaskCategory
	}

	return nil
}

func (o *RunTaskUpdateOptions) valid() error {
	if o.Name != nil && !validString(o.Name) {
		return ErrRequiredName
	}

	if o.URL != nil && !validString(o.URL) {
		return ErrInvalidRunTaskURL
	}

	if o.Category != nil && *o.Category != "task" {
		return ErrInvalidRunTaskCategory
	}

	return nil
}

func (o *RunTaskListOptions) valid() error {
	return nil
}

func (o *RunTaskReadOptions) valid() error {
	return nil
}

// Compile-time proof of interface implementation.
var _ RunTasksIntegration = (*runTaskIntegration)(nil)

// RunTasksIntegration describes all the Run Tasks Integration Callback API methods.
//
// TFE API docs:
// https://developer.hashicorp.com/terraform/enterprise/api-docs/run-tasks/run-tasks-integration
type RunTasksIntegration interface {
	// Update sends updates to TFC/E Run Task Callback URL
	Callback(ctx context.Context, callbackURL string, accessToken string, options TaskResultCallbackRequestOptions) error
}

// taskResultsCallback implements RunTasksIntegration.
type runTaskIntegration struct {
	client *Client
}

// TaskResultCallbackRequestOptions represents the TFC/E Task result callback request
// https://developer.hashicorp.com/terraform/enterprise/api-docs/run-tasks/run-tasks-integration#request-body-1
type TaskResultCallbackRequestOptions struct {
	Type     string               `jsonapi:"primary,task-results"`
	Status   TaskResultStatus     `jsonapi:"attr,status"`
	Message  string               `jsonapi:"attr,message,omitempty"`
	URL      string               `jsonapi:"attr,url,omitempty"`
	Outcomes []*TaskResultOutcome `jsonapi:"relation,outcomes,omitempty"`
}

// TaskResultOutcome represents a detailed TFC/E run task outcome, which improves result visibility and content in the TFC/E UI.
// https://developer.hashicorp.com/terraform/enterprise/api-docs/run-tasks/run-tasks-integration#outcomes-payload-body
type TaskResultOutcome struct {
	Type        string                      `jsonapi:"primary,task-result-outcomes"`
	OutcomeID   string                      `jsonapi:"attr,outcome-id,omitempty"`
	Description string                      `jsonapi:"attr,description,omitempty"`
	Body        string                      `jsonapi:"attr,body,omitempty"`
	URL         string                      `jsonapi:"attr,url,omitempty"`
	Tags        map[string][]*TaskResultTag `jsonapi:"attr,tags,omitempty"`
}

// TaskResultTag can be used to enrich outcomes display list in TFC/E.
// https://developer.hashicorp.com/terraform/enterprise/api-docs/run-tasks/run-tasks-integration#severity-and-status-tags
type TaskResultTag struct {
	Label string `json:"label"`
	Level string `json:"level,omitempty"`
}

// Update sends updates to TFC/E Run Task Callback URL
func (s *runTaskIntegration) Callback(ctx context.Context, callbackURL, accessToken string, options TaskResultCallbackRequestOptions) error {
	if !validString(&callbackURL) {
		return ErrInvalidCallbackURL
	}
	if !validString(&accessToken) {
		return ErrInvalidAccessToken
	}
	if err := options.valid(); err != nil {
		return err
	}
	req, err := s.client.NewRequest(http.MethodPatch, callbackURL, &options)
	if err != nil {
		return err
	}
	// The PATCH request must use the token supplied in the originating request (access_token) for authentication.
	// https://developer.hashicorp.com/terraform/enterprise/api-docs/run-tasks/run-tasks-integration#request-headers-1
	req.Header.Set("Authorization", "Bearer "+accessToken)
	return req.Do(ctx, nil)
}

func (o *TaskResultCallbackRequestOptions) valid() error {
	if o.Status != TaskFailed && o.Status != TaskPassed && o.Status != TaskRunning {
		return ErrInvalidTaskResultsCallbackStatus
	}
	return nil
}

// Compile-time proof of interface implementation.
var _ RunTriggers = (*runTriggers)(nil)

// RunTriggers describes all the Run Trigger
// related methods that the HCP Terraform API supports.
//
// TFE API docs:
// https://developer.hashicorp.com/terraform/cloud-docs/api-docs/run-triggers
type RunTriggers interface {
	// List all the run triggers within a workspace.
	List(ctx context.Context, workspaceID string, options *RunTriggerListOptions) (*RunTriggerList, error)

	// Create a new run trigger with the given options.
	Create(ctx context.Context, workspaceID string, options RunTriggerCreateOptions) (*RunTrigger, error)

	// Read a run trigger by its ID.
	Read(ctx context.Context, RunTriggerID string) (*RunTrigger, error)

	// ReadWithOptions reads a run trigger by its ID using the options supplied
	ReadWithOptions(ctx context.Context, runID string, options *RunTriggerReadOptions) (*RunTrigger, error)

	// Delete a run trigger by its ID.
	Delete(ctx context.Context, RunTriggerID string) error
}

// runTriggers implements RunTriggers.
type runTriggers struct {
	client *Client
}

// RunTriggerList represents a list of Run Triggers
type RunTriggerList struct {
	*Pagination
	Items []*RunTrigger
}

// SourceableChoice is a choice type struct that represents the possible values
// within a polymorphic relation. If a value is available, exactly one field
// will be non-nil.
type SourceableChoice struct {
	Workspace *Workspace
}

// RunTrigger represents a run trigger.
type RunTrigger struct {
	ID             string    `jsonapi:"primary,run-triggers"`
	CreatedAt      time.Time `jsonapi:"attr,created-at,iso8601"`
	SourceableName string    `jsonapi:"attr,sourceable-name"`
	WorkspaceName  string    `jsonapi:"attr,workspace-name"`
	// DEPRECATED. The sourceable field is polymorphic. Use SourceableChoice instead.
	Sourceable       *Workspace        `jsonapi:"relation,sourceable"`
	SourceableChoice *SourceableChoice `jsonapi:"polyrelation,sourceable"`
	Workspace        *Workspace        `jsonapi:"relation,workspace"`
}

// https://developer.hashicorp.com/terraform/cloud-docs/api-docs/run-triggers#query-parameters
type RunTriggerFilterOp string

const (
	RunTriggerOutbound RunTriggerFilterOp = "outbound" // create runs in other workspaces.
	RunTriggerInbound  RunTriggerFilterOp = "inbound"  // create runs in the specified workspace
)

// A list of relations to include
// https://developer.hashicorp.com/terraform/cloud-docs/api-docs/run-triggers#available-related-resources
type RunTriggerIncludeOpt string

const (
	RunTriggerWorkspace  RunTriggerIncludeOpt = "workspace"
	RunTriggerSourceable RunTriggerIncludeOpt = "sourceable"
)

// RunTriggerListOptions represents the options for listing
// run triggers.
type RunTriggerListOptions struct {
	ListOptions
	RunTriggerType RunTriggerFilterOp     `url:"filter[run-trigger][type]"` // Required
	Include        []RunTriggerIncludeOpt `url:"include,omitempty"`         // optional
}

// RunTriggerCreateOptions represents the options for
// creating a new run trigger.
type RunTriggerCreateOptions struct {
	// Type is a public field utilized by JSON:API to
	// set the resource type via the field tag.
	// It is not a user-defined value and does not need to be set.
	// https://jsonapi.org/format/#crud-creating
	Type string `jsonapi:"primary,run-triggers"`

	// The source workspace
	Sourceable *Workspace `jsonapi:"relation,sourceable"`
}

// RunTriggerCreateOptions represents the options for reading a run.
type RunTriggerReadOptions struct {
	Include []RunTriggerIncludeOpt `url:"include,omitempty"` // optional`
}

// List all the run triggers associated with a workspace.
func (s *runTriggers) List(ctx context.Context, workspaceID string, options *RunTriggerListOptions) (*RunTriggerList, error) {
	if !validStringID(&workspaceID) {
		return nil, ErrInvalidWorkspaceID
	}
	if err := options.valid(); err != nil {
		return nil, err
	}

	u := fmt.Sprintf("workspaces/%s/run-triggers", url.PathEscape(workspaceID))
	req, err := s.client.NewRequest("GET", u, options)
	if err != nil {
		return nil, err
	}

	rtl := &RunTriggerList{}
	err = req.Do(ctx, rtl)
	if err != nil {
		return nil, err
	}

	for i := range rtl.Items {
		backfillDeprecatedSourceable(rtl.Items[i])
	}

	return rtl, nil
}

// Create a run trigger with the given options.
func (s *runTriggers) Create(ctx context.Context, workspaceID string, options RunTriggerCreateOptions) (*RunTrigger, error) {
	if !validStringID(&workspaceID) {
		return nil, ErrInvalidWorkspaceID
	}
	if err := options.valid(); err != nil {
		return nil, err
	}

	u := fmt.Sprintf("workspaces/%s/run-triggers", url.PathEscape(workspaceID))
	req, err := s.client.NewRequest("POST", u, &options)
	if err != nil {
		return nil, err
	}

	rt := &RunTrigger{}
	err = req.Do(ctx, rt)
	if err != nil {
		return nil, err
	}

	backfillDeprecatedSourceable(rt)

	return rt, nil
}

// Read a run trigger by its ID.
func (s *runTriggers) Read(ctx context.Context, runTriggerID string) (*RunTrigger, error) {
	return s.ReadWithOptions(ctx, runTriggerID, nil)
}

// Read a run trigger by its ID.
func (s *runTriggers) ReadWithOptions(ctx context.Context, runTriggerID string, options *RunTriggerReadOptions) (*RunTrigger, error) {
	if !validStringID(&runTriggerID) {
		return nil, ErrInvalidRunTriggerID
	}

	u := fmt.Sprintf("run-triggers/%s", url.PathEscape(runTriggerID))
	req, err := s.client.NewRequest("GET", u, options)
	if err != nil {
		return nil, err
	}

	rt := &RunTrigger{}
	err = req.Do(ctx, rt)
	if err != nil {
		return nil, err
	}

	backfillDeprecatedSourceable(rt)

	return rt, nil
}

// Delete a run trigger by its ID.
func (s *runTriggers) Delete(ctx context.Context, runTriggerID string) error {
	if !validStringID(&runTriggerID) {
		return ErrInvalidRunTriggerID
	}

	u := fmt.Sprintf("run-triggers/%s", url.PathEscape(runTriggerID))
	req, err := s.client.NewRequest("DELETE", u, nil)
	if err != nil {
		return err
	}

	return req.Do(ctx, nil)
}

func (o RunTriggerCreateOptions) valid() error {
	if o.Sourceable == nil {
		return ErrRequiredSourceable
	}
	return nil
}

func (o *RunTriggerListOptions) valid() error {
	if o == nil {
		return ErrRequiredRunTriggerListOps
	}

	if err := validateRunTriggerFilterParam(o.RunTriggerType, o.Include); err != nil {
		return err
	}

	return nil
}

func backfillDeprecatedSourceable(runTrigger *RunTrigger) {
	if runTrigger.Sourceable != nil || runTrigger.SourceableChoice == nil {
		return
	}

	runTrigger.Sourceable = runTrigger.SourceableChoice.Workspace
}

func validateRunTriggerFilterParam(filterParam RunTriggerFilterOp, includeParams []RunTriggerIncludeOpt) error {
	switch filterParam {
	case RunTriggerOutbound, RunTriggerInbound:
		// Do nothing
	default:
		return ErrInvalidRunTriggerType // return an error even if string is empty because this a required field
	}

	if len(includeParams) > 0 {
		if filterParam != RunTriggerInbound {
			return ErrUnsupportedRunTriggerType // if user passes RunTriggerOutbound the platform will not return any "include" data
		}
	}

	return nil
}

// Compile-time proof of interface implementation.
var _ Runs = (*runs)(nil)

// Runs describes all the run related methods that the Terraform Enterprise
// API supports.
//
// TFE API docs: https://developer.hashicorp.com/terraform/cloud-docs/api-docs/run
type Runs interface {
	// List all the runs of the given workspace.
	List(ctx context.Context, workspaceID string, options *RunListOptions) (*RunList, error)

	// List all the runs of the given organization.
	ListForOrganization(ctx context.Context, organization string, options *RunListForOrganizationOptions) (*OrganizationRunList, error)

	// Create a new run with the given options.
	Create(ctx context.Context, options RunCreateOptions) (*Run, error)

	// Read a run by its ID.
	Read(ctx context.Context, runID string) (*Run, error)

	// ReadWithOptions reads a run by its ID using the options supplied
	ReadWithOptions(ctx context.Context, runID string, options *RunReadOptions) (*Run, error)

	// Apply a run by its ID.
	Apply(ctx context.Context, runID string, options RunApplyOptions) error

	// Cancel a run by its ID.
	Cancel(ctx context.Context, runID string, options RunCancelOptions) error

	// Force-cancel a run by its ID.
	ForceCancel(ctx context.Context, runID string, options RunForceCancelOptions) error

	// Force execute a run by its ID.
	ForceExecute(ctx context.Context, runID string) error

	// Discard a run by its ID.
	Discard(ctx context.Context, runID string, options RunDiscardOptions) error
}

// runs implements Runs.
type runs struct {
	client *Client
}

// RunStatus represents a run state.
type RunStatus string

// List all available run statuses.
const (
	RunApplied                  RunStatus = "applied"
	RunApplying                 RunStatus = "applying"
	RunApplyQueued              RunStatus = "apply_queued"
	RunCanceled                 RunStatus = "canceled"
	RunConfirmed                RunStatus = "confirmed"
	RunCostEstimated            RunStatus = "cost_estimated"
	RunCostEstimating           RunStatus = "cost_estimating"
	RunDiscarded                RunStatus = "discarded"
	RunErrored                  RunStatus = "errored"
	RunFetching                 RunStatus = "fetching"
	RunFetchingCompleted        RunStatus = "fetching_completed"
	RunPending                  RunStatus = "pending"
	RunPlanned                  RunStatus = "planned"
	RunPlannedAndFinished       RunStatus = "planned_and_finished"
	RunPlannedAndSaved          RunStatus = "planned_and_saved"
	RunPlanning                 RunStatus = "planning"
	RunPlanQueued               RunStatus = "plan_queued"
	RunPolicyChecked            RunStatus = "policy_checked"
	RunPolicyChecking           RunStatus = "policy_checking"
	RunPolicyOverride           RunStatus = "policy_override"
	RunPolicySoftFailed         RunStatus = "policy_soft_failed"
	RunPostPlanAwaitingDecision RunStatus = "post_plan_awaiting_decision"
	RunPostPlanCompleted        RunStatus = "post_plan_completed"
	RunPostPlanRunning          RunStatus = "post_plan_running"
	RunPostApplyRunning         RunStatus = "post_apply_running"
	RunPostApplyCompleted       RunStatus = "post_apply_completed"
	RunPreApplyRunning          RunStatus = "pre_apply_running"
	RunPreApplyCompleted        RunStatus = "pre_apply_completed"
	RunPrePlanCompleted         RunStatus = "pre_plan_completed"
	RunPrePlanRunning           RunStatus = "pre_plan_running"
	RunQueuing                  RunStatus = "queuing"
	RunQueuingApply             RunStatus = "queuing_apply"
)

// RunSource represents a source type of a run.
type RunSource string

// List all available run sources.
const (
	RunSourceAPI                  RunSource = "tfe-api"
	RunSourceConfigurationVersion RunSource = "tfe-configuration-version"
	RunSourceUI                   RunSource = "tfe-ui"
)

// RunOperation represents an operation type of run.
type RunOperation string

// List all available run operations.
const (
	RunOperationPlanApply   RunOperation = "plan_and_apply"
	RunOperationPlanOnly    RunOperation = "plan_only"
	RunOperationRefreshOnly RunOperation = "refresh_only"
	RunOperationDestroy     RunOperation = "destroy"
	RunOperationEmptyApply  RunOperation = "empty_apply"
	RunOperationSavePlan    RunOperation = "save_plan"
)

// RunList represents a list of runs.
type RunList struct {
	*Pagination
	Items []*Run
}

// OrganizationRunList represents a list of runs across an organization. It
// differs from the RunList in that it does not include a TotalCount of records
// in the pagination details
type OrganizationRunList struct {
	*PaginationNextPrev
	Items []*Run
}

// Run represents a Terraform Enterprise run.
type Run struct {
	ID                     string               `jsonapi:"primary,runs"`
	Actions                *RunActions          `jsonapi:"attr,actions"`
	AutoApply              bool                 `jsonapi:"attr,auto-apply,omitempty"`
	AllowConfigGeneration  *bool                `jsonapi:"attr,allow-config-generation,omitempty"`
	AllowEmptyApply        bool                 `jsonapi:"attr,allow-empty-apply"`
	CanceledAt             time.Time            `jsonapi:"attr,canceled-at,iso8601"`
	CreatedAt              time.Time            `jsonapi:"attr,created-at,iso8601"`
	ForceCancelAvailableAt time.Time            `jsonapi:"attr,force-cancel-available-at,iso8601"`
	HasChanges             bool                 `jsonapi:"attr,has-changes"`
	IsDestroy              bool                 `jsonapi:"attr,is-destroy"`
	InvokeActionAddrs      []string             `jsonapi:"attr,invoke-action-addrs,omitempty"`
	Message                string               `jsonapi:"attr,message"`
	Permissions            *RunPermissions      `jsonapi:"attr,permissions"`
	PolicyPaths            []string             `jsonapi:"attr,policy-paths,omitempty"`
	PositionInQueue        int                  `jsonapi:"attr,position-in-queue"`
	PlanOnly               bool                 `jsonapi:"attr,plan-only"`
	Refresh                bool                 `jsonapi:"attr,refresh"`
	RefreshOnly            bool                 `jsonapi:"attr,refresh-only"`
	ReplaceAddrs           []string             `jsonapi:"attr,replace-addrs,omitempty"`
	SavePlan               bool                 `jsonapi:"attr,save-plan,omitempty"`
	Source                 RunSource            `jsonapi:"attr,source"`
	Status                 RunStatus            `jsonapi:"attr,status"`
	StatusTimestamps       *RunStatusTimestamps `jsonapi:"attr,status-timestamps"`
	TargetAddrs            []string             `jsonapi:"attr,target-addrs,omitempty"`
	TerraformVersion       string               `jsonapi:"attr,terraform-version"`
	TriggerReason          string               `jsonapi:"attr,trigger-reason"`
	Variables              []*RunVariableAttr   `jsonapi:"attr,variables"`

	// Relations
	Apply                *Apply                `jsonapi:"relation,apply"`
	ConfigurationVersion *ConfigurationVersion `jsonapi:"relation,configuration-version"`
	CostEstimate         *CostEstimate         `jsonapi:"relation,cost-estimate"`
	CreatedBy            *User                 `jsonapi:"relation,created-by"`
	ConfirmedBy          *User                 `jsonapi:"relation,confirmed-by"`
	Plan                 *Plan                 `jsonapi:"relation,plan"`
	PolicyChecks         []*PolicyCheck        `jsonapi:"relation,policy-checks"`
	RunEvents            []*RunEvent           `jsonapi:"relation,run-events"`
	TaskStages           []*TaskStage          `jsonapi:"relation,task-stages,omitempty"`
	Workspace            *Workspace            `jsonapi:"relation,workspace"`
	Comments             []*Comment            `jsonapi:"relation,comments"`

	// **Note: This field is still in BETA and subject to change.**
	TFPolicyEvaluations []*TFPolicyEvaluation `jsonapi:"relation,tf-policy-evaluations,omitempty"`
}

// RunActions represents the run actions.
type RunActions struct {
	IsCancelable      bool `jsonapi:"attr,is-cancelable"`
	IsConfirmable     bool `jsonapi:"attr,is-confirmable"`
	IsDiscardable     bool `jsonapi:"attr,is-discardable"`
	IsForceCancelable bool `jsonapi:"attr,is-force-cancelable"`
}

// RunPermissions represents the run permissions.
type RunPermissions struct {
	CanApply        bool `jsonapi:"attr,can-apply"`
	CanCancel       bool `jsonapi:"attr,can-cancel"`
	CanDiscard      bool `jsonapi:"attr,can-discard"`
	CanForceCancel  bool `jsonapi:"attr,can-force-cancel"`
	CanForceExecute bool `jsonapi:"attr,can-force-execute"`
}

// RunStatusTimestamps holds the timestamps for individual run statuses.
type RunStatusTimestamps struct {
	AppliedAt            time.Time `jsonapi:"attr,applied-at,rfc3339"`
	ApplyingAt           time.Time `jsonapi:"attr,applying-at,rfc3339"`
	ApplyQueuedAt        time.Time `jsonapi:"attr,apply-queued-at,rfc3339"`
	CanceledAt           time.Time `jsonapi:"attr,canceled-at,rfc3339"`
	ConfirmedAt          time.Time `jsonapi:"attr,confirmed-at,rfc3339"`
	CostEstimatedAt      time.Time `jsonapi:"attr,cost-estimated-at,rfc3339"`
	CostEstimatingAt     time.Time `jsonapi:"attr,cost-estimating-at,rfc3339"`
	DiscardedAt          time.Time `jsonapi:"attr,discarded-at,rfc3339"`
	ErroredAt            time.Time `jsonapi:"attr,errored-at,rfc3339"`
	FetchedAt            time.Time `jsonapi:"attr,fetched-at,rfc3339"`
	FetchingAt           time.Time `jsonapi:"attr,fetching-at,rfc3339"`
	ForceCanceledAt      time.Time `jsonapi:"attr,force-canceled-at,rfc3339"`
	PlannedAndFinishedAt time.Time `jsonapi:"attr,planned-and-finished-at,rfc3339"`
	PlannedAndSavedAt    time.Time `jsonapi:"attr,planned-and-saved-at,rfc3339"`
	PlannedAt            time.Time `jsonapi:"attr,planned-at,rfc3339"`
	PlanningAt           time.Time `jsonapi:"attr,planning-at,rfc3339"`
	PlanQueueableAt      time.Time `jsonapi:"attr,plan-queueable-at,rfc3339"`
	PlanQueuedAt         time.Time `jsonapi:"attr,plan-queued-at,rfc3339"`
	PolicyCheckedAt      time.Time `jsonapi:"attr,policy-checked-at,rfc3339"`
	PolicySoftFailedAt   time.Time `jsonapi:"attr,policy-soft-failed-at,rfc3339"`
	PostPlanCompletedAt  time.Time `jsonapi:"attr,post-plan-completed-at,rfc3339"`
	PostPlanRunningAt    time.Time `jsonapi:"attr,post-plan-running-at,rfc3339"`
	PrePlanCompletedAt   time.Time `jsonapi:"attr,pre-plan-completed-at,rfc3339"`
	PrePlanRunningAt     time.Time `jsonapi:"attr,pre-plan-running-at,rfc3339"`
	QueuingAt            time.Time `jsonapi:"attr,queuing-at,rfc3339"`
}

// RunIncludeOpt represents the available options for include query params.
// https://developer.hashicorp.com/terraform/cloud-docs/api-docs/run#available-related-resources
type RunIncludeOpt string

const (
	RunPlan             RunIncludeOpt = "plan"
	RunApply            RunIncludeOpt = "apply"
	RunCreatedBy        RunIncludeOpt = "created_by"
	RunCostEstimate     RunIncludeOpt = "cost_estimate"
	RunConfigVer        RunIncludeOpt = "configuration_version"
	RunConfigVerIngress RunIncludeOpt = "configuration_version.ingress_attributes"
	RunWorkspace        RunIncludeOpt = "workspace"
	RunTaskStages       RunIncludeOpt = "task_stages"
	// **Note: This field is still in BETA and subject to change.**
	RunTFPolicyEvaluation RunIncludeOpt = "tf_policy_evaluations"
)

// RunListOptions represents the options for listing runs.
type RunListOptions struct {
	ListOptions

	// Optional: Searches runs that matches the supplied VCS username.
	User string `url:"search[user],omitempty"`

	// Optional: Searches runs that matches the supplied commit sha.
	Commit string `url:"search[commit],omitempty"`

	// Optional: Searches runs that matches the supplied VCS username, commit sha, run_id, and run message.
	// The presence of search[commit] or search[user] takes priority over this parameter and will be omitted.
	Search string `url:"search[basic],omitempty"`

	// Optional: Comma-separated list of acceptable run statuses.
	// Options are listed at https://developer.hashicorp.com/terraform/cloud-docs/api-docs/run#run-states,
	// or as constants with the RunStatus string type.
	Status string `url:"filter[status],omitempty"`

	// Optional: Comma-separated list of acceptable run sources.
	// Options are listed at https://developer.hashicorp.com/terraform/cloud-docs/api-docs/run#run-sources,
	// or as constants with the RunSource string type.
	Source string `url:"filter[source],omitempty"`

	// Optional: Comma-separated list of acceptable run operation types.
	// Options are listed at https://developer.hashicorp.com/terraform/cloud-docs/api-docs/run#run-operations,
	// or as constants with the RunOperation string type.
	Operation string `url:"filter[operation],omitempty"`

	// Optional: A list of relations to include. See available resources:
	// https://developer.hashicorp.com/terraform/cloud-docs/api-docs/run#available-related-resources
	Include []RunIncludeOpt `url:"include,omitempty"`
}

// RunListForOrganizationOptions represents the options for listing runs for an organization.
type RunListForOrganizationOptions struct {
	ListOptions

	// Optional: Searches runs that matches the supplied VCS username.
	User string `url:"search[user],omitempty"`

	// Optional: Searches runs that matches the supplied commit sha.
	Commit string `url:"search[commit],omitempty"`

	// Optional: Searches for runs that match the VCS username, commit sha, run_id, or run message your specify.
	// The presence of search[commit] or search[user] takes priority over this parameter and will be omitted.
	Basic string `url:"search[basic],omitempty"`

	// Optional: Comma-separated list of acceptable run statuses.
	// Options are listed at https://developer.hashicorp.com/terraform/cloud-docs/api-docs/run#run-states,
	// or as constants with the RunStatus string type.
	Status string `url:"filter[status],omitempty"`

	// Optional: Comma-separated list of acceptable run sources.
	// Options are listed at https://developer.hashicorp.com/terraform/cloud-docs/api-docs/run#run-sources,
	// or as constants with the RunSource string type.
	Source string `url:"filter[source],omitempty"`

	// Optional: Comma-separated list of acceptable run operation types.
	// Options are listed at https://developer.hashicorp.com/terraform/cloud-docs/api-docs/run#run-operations,
	// or as constants with the RunOperation string type.
	Operation string `url:"filter[operation],omitempty"`

	// Optional: Comma-separated list of agent pool names.
	AgentPoolNames string `url:"filter[agent_pool_names],omitempty"`

	// Optional: Comma-separated list of run status groups.
	StatusGroup string `url:"filter[status_group],omitempty"`

	// Optional: Comma-separated list of run timeframe.
	Timeframe string `url:"filter[timeframe],omitempty"`

	// Optional: Comma-separated list of workspace names. The result lists runs that belong to one of the workspaces your specify.
	WorkspaceNames string `url:"filter[workspace_names],omitempty"`

	// Optional: A list of relations to include. See available resources:
	// https://developer.hashicorp.com/terraform/cloud-docs/api-docs/run#available-related-resources
	Include []RunIncludeOpt `url:"include,omitempty"`
}

// RunReadOptions represents the options for reading a run.
type RunReadOptions struct {
	// Optional: A list of relations to include. See available resources:
	// https://developer.hashicorp.com/terraform/cloud-docs/api-docs/run#available-related-resources
	Include []RunIncludeOpt `url:"include,omitempty"`
}

// RunCreateOptions represents the options for creating a new run.
type RunCreateOptions struct {
	// Type is a public field utilized by JSON:API to
	// set the resource type via the field tag.
	// It is not a user-defined value and does not need to be set.
	// https://jsonapi.org/format/#crud-creating
	Type string `jsonapi:"primary,runs"`

	// AllowConfigGeneration specifies whether generated resource configuration may be created as a side
	// effect of an import block in this run. Setting this does not mean that configuration _will_ be generated,
	// only that it can be.
	AllowConfigGeneration *bool `jsonapi:"attr,allow-config-generation,omitempty"`

	// AllowEmptyApply specifies whether Terraform can apply the run even when the plan contains no changes.
	// Often used to upgrade state after upgrading a workspace to a new terraform version.
	AllowEmptyApply *bool `jsonapi:"attr,allow-empty-apply,omitempty"`

	// TerraformVersion specifies the Terraform version to use in this run.
	// Only valid for plan-only runs; must be a valid Terraform version available to the organization.
	TerraformVersion *string `jsonapi:"attr,terraform-version,omitempty"`

	// PlanOnly specifies if this is a speculative, plan-only run that Terraform cannot apply.
	// Often used in conjunction with terraform-version in order to test whether an upgrade would succeed.
	PlanOnly *bool `jsonapi:"attr,plan-only,omitempty"`

	// Specifies if this plan is a destroy plan, which will destroy all
	// provisioned resources.
	IsDestroy *bool `jsonapi:"attr,is-destroy,omitempty"`

	// Refresh determines if the run should
	// update the state prior to checking for differences
	Refresh *bool `jsonapi:"attr,refresh,omitempty"`

	// RefreshOnly determines whether the run should ignore config changes
	// and refresh the state only
	RefreshOnly *bool `jsonapi:"attr,refresh-only,omitempty"`

	// SavePlan determines whether this should be a saved-plan run. Saved-plan
	// runs perform their plan and checks immediately, but won't lock the
	// workspace and become its current run until they are confirmed for apply.
	SavePlan *bool `jsonapi:"attr,save-plan,omitempty"`

	// Specifies the message to be associated with this run.
	Message *string `jsonapi:"attr,message,omitempty"`

	// Specifies the configuration version to use for this run. If the
	// configuration version object is omitted, the run will be created using the
	// workspace's latest configuration version.
	ConfigurationVersion *ConfigurationVersion `jsonapi:"relation,configuration-version"`

	// Specifies the workspace where the run will be executed.
	Workspace *Workspace `jsonapi:"relation,workspace"`

	// If non-empty, requests that Terraform should create a plan including
	// actions only for the given objects (specified using resource address
	// syntax) and the objects they depend on.
	//
	// This capability is provided for exceptional circumstances only, such as
	// recovering from mistakes or working around existing Terraform
	// limitations. Terraform will generally mention the -target command line
	// option in its error messages describing situations where setting this
	// argument may be appropriate. This argument should not be used as part
	// of routine workflow and Terraform will emit warnings reminding about
	// this whenever this property is set.
	TargetAddrs []string `jsonapi:"attr,target-addrs,omitempty"`

	// If non-empty, requests that Terraform create a plan that replaces
	// (destroys and then re-creates) the objects specified by the given
	// resource addresses.
	ReplaceAddrs []string `jsonapi:"attr,replace-addrs,omitempty"`

	// PolicyPaths is a list of relative directory paths that point to policy
	// configuration files.
	//
	// **Note: This field is in BETA and subject to change.**
	PolicyPaths []string `jsonapi:"attr,policy-paths,omitempty"`

	// AutoApply determines if the run should be applied automatically without
	// user confirmation. It defaults to the Workspace.AutoApply setting.
	AutoApply *bool `jsonapi:"attr,auto-apply,omitempty"`

	// Variables allows you to specify terraform input variables for
	// a particular run, prioritized over variables defined on the workspace.
	Variables []*RunVariable `jsonapi:"attr,variables,omitempty"`

	// Action Addresses to invoke.
	InvokeActionAddrs []string `jsonapi:"attr,invoke-action-addrs,omitempty"`
}

// RunApplyOptions represents the options for applying a run.
type RunApplyOptions struct {
	// An optional comment about the run.
	Comment *string `json:"comment,omitempty"`
}

// RunCancelOptions represents the options for canceling a run.
type RunCancelOptions struct {
	// An optional explanation for why the run was canceled.
	Comment *string `json:"comment,omitempty"`
}

type RunVariableAttr struct {
	Key   string `jsonapi:"attr,key"`
	Value string `jsonapi:"attr,value"`
}

// RunVariableAttr represents a variable that can be applied to a run. All values must be expressed as an HCL literal
// in the same syntax you would use when writing terraform code. See https://developer.hashicorp.com/terraform/language/expressions/types#types
// for more details.
type RunVariable struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// RunForceCancelOptions represents the options for force-canceling a run.
type RunForceCancelOptions struct {
	// An optional comment explaining the reason for the force-cancel.
	Comment *string `json:"comment,omitempty"`
}

// RunDiscardOptions represents the options for discarding a run.
type RunDiscardOptions struct {
	// An optional explanation for why the run was discarded.
	Comment *string `json:"comment,omitempty"`
}

// List all the runs of the given workspace.
func (s *runs) List(ctx context.Context, workspaceID string, options *RunListOptions) (*RunList, error) {
	if !validStringID(&workspaceID) {
		return nil, ErrInvalidWorkspaceID
	}
	if err := options.valid(); err != nil {
		return nil, err
	}

	u := fmt.Sprintf("workspaces/%s/runs", url.PathEscape(workspaceID))
	req, err := s.client.NewRequest("GET", u, options)
	if err != nil {
		return nil, err
	}

	rl := &RunList{}
	err = req.Do(ctx, rl)
	if err != nil {
		return nil, err
	}

	return rl, nil
}

// List all the runs of the given workspace.
func (s *runs) ListForOrganization(ctx context.Context, organization string, options *RunListForOrganizationOptions) (*OrganizationRunList, error) {
	if !validStringID(&organization) {
		return nil, ErrInvalidOrg
	}
	if err := options.valid(); err != nil {
		return nil, err
	}

	u := fmt.Sprintf("organizations/%s/runs", url.PathEscape(organization))
	req, err := s.client.NewRequest("GET", u, options)
	if err != nil {
		return nil, err
	}

	rl := &OrganizationRunList{}
	err = req.Do(ctx, rl)
	if err != nil {
		return nil, err
	}

	return rl, nil
}

// Create a new run with the given options.
func (s *runs) Create(ctx context.Context, options RunCreateOptions) (*Run, error) {
	if err := options.valid(); err != nil {
		return nil, err
	}

	req, err := s.client.NewRequest("POST", "runs", &options)
	if err != nil {
		return nil, err
	}

	r := &Run{}
	err = req.Do(ctx, r)
	if err != nil {
		return nil, err
	}

	return r, nil
}

// Read a run by its ID.
func (s *runs) Read(ctx context.Context, runID string) (*Run, error) {
	return s.ReadWithOptions(ctx, runID, nil)
}

// Read a run by its ID with the given options.
func (s *runs) ReadWithOptions(ctx context.Context, runID string, options *RunReadOptions) (*Run, error) {
	if !validStringID(&runID) {
		return nil, ErrInvalidRunID
	}
	if err := options.valid(); err != nil {
		return nil, err
	}

	u := fmt.Sprintf("runs/%s", url.PathEscape(runID))
	req, err := s.client.NewRequest("GET", u, options)
	if err != nil {
		return nil, err
	}

	r := &Run{}
	err = req.Do(ctx, r)
	if err != nil {
		return nil, err
	}

	return r, nil
}

// Apply a run by its ID.
func (s *runs) Apply(ctx context.Context, runID string, options RunApplyOptions) error {
	if !validStringID(&runID) {
		return ErrInvalidRunID
	}

	u := fmt.Sprintf("runs/%s/actions/apply", url.PathEscape(runID))
	req, err := s.client.NewRequest("POST", u, &options)
	if err != nil {
		return err
	}

	return req.Do(ctx, nil)
}

// Cancel a run by its ID.
func (s *runs) Cancel(ctx context.Context, runID string, options RunCancelOptions) error {
	if !validStringID(&runID) {
		return ErrInvalidRunID
	}

	u := fmt.Sprintf("runs/%s/actions/cancel", url.PathEscape(runID))
	req, err := s.client.NewRequest("POST", u, &options)
	if err != nil {
		return err
	}

	return req.Do(ctx, nil)
}

// ForceCancel is used to forcefully cancel a run by its ID.
func (s *runs) ForceCancel(ctx context.Context, runID string, options RunForceCancelOptions) error {
	if !validStringID(&runID) {
		return ErrInvalidRunID
	}

	u := fmt.Sprintf("runs/%s/actions/force-cancel", url.PathEscape(runID))
	req, err := s.client.NewRequest("POST", u, &options)
	if err != nil {
		return err
	}

	return req.Do(ctx, nil)
}

// ForceExecute is used to forcefully execute a run by its ID.
//
// Note: While useful at times, force-executing a run circumvents the typical
// workflow of applying runs using HCP Terraform. It is not intended for
// regular use. If you find yourself using it frequently, please reach out to
// HashiCorp Support for help in developing an alternative approach.
func (s *runs) ForceExecute(ctx context.Context, runID string) error {
	if !validStringID(&runID) {
		return ErrInvalidRunID
	}

	u := fmt.Sprintf("runs/%s/actions/force-execute", url.PathEscape(runID))
	req, err := s.client.NewRequest("POST", u, nil)
	if err != nil {
		return err
	}

	return req.Do(ctx, nil)
}

// Discard a run by its ID.
func (s *runs) Discard(ctx context.Context, runID string, options RunDiscardOptions) error {
	if !validStringID(&runID) {
		return ErrInvalidRunID
	}

	u := fmt.Sprintf("runs/%s/actions/discard", url.PathEscape(runID))
	req, err := s.client.NewRequest("POST", u, &options)
	if err != nil {
		return err
	}

	return req.Do(ctx, nil)
}

func (o RunCreateOptions) valid() error {
	if o.Workspace == nil {
		return ErrRequiredWorkspace
	}

	if validString(o.TerraformVersion) && (o.PlanOnly == nil || !*o.PlanOnly) {
		return ErrTerraformVersionValidForPlanOnly
	}

	return nil
}

func (o *RunReadOptions) valid() error {
	return nil
}

func (o *RunListOptions) valid() error {
	return nil
}

func (o *RunListForOrganizationOptions) valid() error {
	return nil
}

// Compile-time proof of interface implementation.
var _ SSHKeys = (*sshKeys)(nil)

// SSHKeys describes all the SSH key related methods that the Terraform
// Enterprise API supports.
//
// TFE API docs:
// https://developer.hashicorp.com/terraform/cloud-docs/api-docs/ssh-keys
type SSHKeys interface {
	// List all the SSH keys for a given organization
	List(ctx context.Context, organization string, options *SSHKeyListOptions) (*SSHKeyList, error)

	// Create an SSH key and associate it with an organization.
	Create(ctx context.Context, organization string, options SSHKeyCreateOptions) (*SSHKey, error)

	// Read an SSH key by its ID.
	Read(ctx context.Context, sshKeyID string) (*SSHKey, error)

	// Update an SSH key by its ID.
	Update(ctx context.Context, sshKeyID string, options SSHKeyUpdateOptions) (*SSHKey, error)

	// Delete an SSH key by its ID.
	Delete(ctx context.Context, sshKeyID string) error
}

// sshKeys implements SSHKeys.
type sshKeys struct {
	client *Client
}

// SSHKeyList represents a list of SSH keys.
type SSHKeyList struct {
	*Pagination
	Items []*SSHKey
}

// SSHKey represents a SSH key.
type SSHKey struct {
	ID   string `jsonapi:"primary,ssh-keys"`
	Name string `jsonapi:"attr,name"`
}

// SSHKeyListOptions represents the options for listing SSH keys.
type SSHKeyListOptions struct {
	ListOptions
}

// SSHKeyCreateOptions represents the options for creating an SSH key.
type SSHKeyCreateOptions struct {
	// Type is a public field utilized by JSON:API to
	// set the resource type via the field tag.
	// It is not a user-defined value and does not need to be set.
	// https://jsonapi.org/format/#crud-creating
	Type string `jsonapi:"primary,ssh-keys"`

	// A name to identify the SSH key.
	Name *string `jsonapi:"attr,name"`

	// The content of the SSH private key.
	Value *string `jsonapi:"attr,value"`
}

// SSHKeyUpdateOptions represents the options for updating an SSH key.
type SSHKeyUpdateOptions struct {
	// For internal use only!
	ID string `jsonapi:"primary,ssh-keys"`

	// Optional: A new name to identify the SSH key.
	Name *string `jsonapi:"attr,name,omitempty"`
}

// List all the SSH keys for a given organization
func (s *sshKeys) List(ctx context.Context, organization string, options *SSHKeyListOptions) (*SSHKeyList, error) {
	if !validStringID(&organization) {
		return nil, ErrInvalidOrg
	}

	u := fmt.Sprintf("organizations/%s/ssh-keys", url.PathEscape(organization))
	req, err := s.client.NewRequest("GET", u, options)
	if err != nil {
		return nil, err
	}

	kl := &SSHKeyList{}
	err = req.Do(ctx, kl)
	if err != nil {
		return nil, err
	}

	return kl, nil
}

// Create an SSH key and associate it with an organization.
func (s *sshKeys) Create(ctx context.Context, organization string, options SSHKeyCreateOptions) (*SSHKey, error) {
	if !validStringID(&organization) {
		return nil, ErrInvalidOrg
	}

	if err := options.valid(); err != nil {
		return nil, err
	}

	u := fmt.Sprintf("organizations/%s/ssh-keys", url.PathEscape(organization))
	req, err := s.client.NewRequest("POST", u, &options)
	if err != nil {
		return nil, err
	}

	k := &SSHKey{}
	err = req.Do(ctx, k)
	if err != nil {
		return nil, err
	}

	return k, nil
}

// Read an SSH key by its ID.
func (s *sshKeys) Read(ctx context.Context, sshKeyID string) (*SSHKey, error) {
	if !validStringID(&sshKeyID) {
		return nil, ErrInvalidSHHKeyID
	}

	u := fmt.Sprintf("ssh-keys/%s", url.PathEscape(sshKeyID))
	req, err := s.client.NewRequest("GET", u, nil)
	if err != nil {
		return nil, err
	}

	k := &SSHKey{}
	err = req.Do(ctx, k)
	if err != nil {
		return nil, err
	}

	return k, nil
}

// Update an SSH key by its ID.
func (s *sshKeys) Update(ctx context.Context, sshKeyID string, options SSHKeyUpdateOptions) (*SSHKey, error) {
	if !validStringID(&sshKeyID) {
		return nil, ErrInvalidSHHKeyID
	}

	u := fmt.Sprintf("ssh-keys/%s", url.PathEscape(sshKeyID))
	req, err := s.client.NewRequest("PATCH", u, &options)
	if err != nil {
		return nil, err
	}

	k := &SSHKey{}
	err = req.Do(ctx, k)
	if err != nil {
		return nil, err
	}

	return k, nil
}

// Delete an SSH key by its ID.
func (s *sshKeys) Delete(ctx context.Context, sshKeyID string) error {
	if !validStringID(&sshKeyID) {
		return ErrInvalidSHHKeyID
	}

	u := fmt.Sprintf("ssh-keys/%s", url.PathEscape(sshKeyID))
	req, err := s.client.NewRequest("DELETE", u, nil)
	if err != nil {
		return err
	}

	return req.Do(ctx, nil)
}

func (o SSHKeyCreateOptions) valid() error {
	if !validString(o.Name) {
		return ErrRequiredName
	}
	if !validString(o.Value) {
		return ErrRequiredValue
	}
	return nil
}

type StackConfigurationSummaries interface {
	// List lists all the stack configuration summaries for a stack.
	List(ctx context.Context, stackID string, options *StackConfigurationSummaryListOptions) (*StackConfigurationSummaryList, error)
}

type stackConfigurationSummaries struct {
	client *Client
}

var _ StackConfigurationSummaries = &stackConfigurationSummaries{}

type StackConfigurationSummaryList struct {
	*Pagination
	Items []*StackConfigurationSummary
}

type StackConfigurationSummaryListOptions struct {
	ListOptions
}

type StackConfigurationSummary struct {
	ID             string `jsonapi:"primary,stack-configuration-summaries"`
	Status         string `jsonapi:"attr,status"`
	SequenceNumber int    `jsonapi:"attr,sequence-number"`
}

func (s stackConfigurationSummaries) List(ctx context.Context, stackID string, options *StackConfigurationSummaryListOptions) (*StackConfigurationSummaryList, error) {
	if !validStringID(&stackID) {
		return nil, fmt.Errorf("invalid stack ID: %s", stackID)
	}

	if options == nil {
		options = &StackConfigurationSummaryListOptions{}
	}

	req, err := s.client.NewRequest("GET", fmt.Sprintf("stacks/%s/stack-configuration-summaries", url.PathEscape(stackID)), options)
	if err != nil {
		return nil, err
	}

	scl := &StackConfigurationSummaryList{}
	err = req.Do(ctx, scl)
	if err != nil {
		return nil, err
	}

	return scl, nil
}

// StackConfigurations describes all the stacks configurations-related methods that the
// HCP Terraform API supports.
type StackConfigurations interface {
	// CreateAndUpload packages and uploads the specified Terraform Stacks
	// configuration files in association with a Stack.
	CreateAndUpload(ctx context.Context, stackID string, path string, opts *CreateStackConfigurationOptions) (*StackConfiguration, error)

	// Upload a tar gzip archive to the specified stack configuration upload URL.
	UploadTarGzip(ctx context.Context, url string, archive io.Reader) error

	// ReadConfiguration returns a stack configuration by its ID.
	Read(ctx context.Context, id string) (*StackConfiguration, error)

	// ListStackConfigurations returns a list of stack configurations for a stack.
	List(ctx context.Context, stackID string, opts *StackConfigurationListOptions) (*StackConfigurationList, error)

	// JSONSchemas returns a byte slice of the JSON schema for the stack configuration.
	JSONSchemas(ctx context.Context, stackConfigurationID string) ([]byte, error)

	// AwaitCompleted generates a channel that will receive the status of the
	// stack configuration as it progresses, until that status is "converged",
	// "converging", "errored", "canceled".
	AwaitCompleted(ctx context.Context, stackConfigurationID string) <-chan WaitForStatusResult

	// AwaitPrepared generates a channel that will receive the status of the
	// stack configuration as it progresses, until that status is "<status>",
	// "errored", "canceled".
	AwaitStatus(ctx context.Context, stackConfigurationID string, status StackConfigurationStatus) <-chan WaitForStatusResult

	// Diagnostics returns the diagnostics for this stack configuration.
	Diagnostics(ctx context.Context, stackConfigurationID string) (*StackDiagnosticsList, error)
}

type StackConfigurationStatus string

const (
	StackConfigurationStatusPending   StackConfigurationStatus = "pending"
	StackConfigurationStatusQueued    StackConfigurationStatus = "queued"
	StackConfigurationStatusPreparing StackConfigurationStatus = "preparing"
	StackConfigurationStatusCompleted StackConfigurationStatus = "completed"
	StackConfigurationStatusFailed    StackConfigurationStatus = "failed"
)

func (s StackConfigurationStatus) String() string {
	return string(s)
}

type stackConfigurations struct {
	client *Client
}

var _ StackConfigurations = &stackConfigurations{}

func (s stackConfigurations) Read(ctx context.Context, id string) (*StackConfiguration, error) {
	req, err := s.client.NewRequest("GET", fmt.Sprintf("stack-configurations/%s", url.PathEscape(id)), nil)
	if err != nil {
		return nil, err
	}

	stackConfiguration := &StackConfiguration{}
	err = req.Do(ctx, stackConfiguration)
	if err != nil {
		return nil, err
	}

	return stackConfiguration, nil
}

/**
* Returns the JSON schema for the stack configuration as a byte slice.
* The return value needs to be unmarshalled into a struct to be useful.
* It is meant to be unmarshalled with terraform/internal/command/jsonproivder.Providers.
 */
func (s stackConfigurations) JSONSchemas(ctx context.Context, stackConfigurationID string) ([]byte, error) {
	req, err := s.client.NewRequest("GET", fmt.Sprintf("stack-configurations/%s/json-schemas", url.PathEscape(stackConfigurationID)), nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Accept", "application/json")

	var raw bytes.Buffer
	err = req.Do(ctx, &raw)
	if err != nil {
		return nil, err
	}

	return raw.Bytes(), nil
}

// AwaitCompleted generates a channel that will receive the status of the stack configuration as it progresses.
// The channel will be closed when the stack configuration reaches a status indicating that or an error occurs. The
// read will be retried dependending on the configuration of the client. When the channel is closed,
// the last value will either be a completed status or an error.
func (s stackConfigurations) AwaitCompleted(ctx context.Context, stackConfigurationID string) <-chan WaitForStatusResult {
	return awaitPoll(ctx, stackConfigurationID, func(ctx context.Context) (string, error) {
		stackConfiguration, err := s.Read(ctx, stackConfigurationID)
		if err != nil {
			return "", err
		}

		return stackConfiguration.Status.String(), nil
	}, []string{StackConfigurationStatusCompleted.String(), StackConfigurationStatusFailed.String()})
}

// AwaitStatus generates a channel that will receive the status of the stack configuration as it progresses.
// The channel will be closed when the stack configuration reaches a status indicating that or an error occurs. The
// read will be retried dependending on the configuration of the client. When the channel is closed,
// the last value will either be the specified status, "errored" status, or "canceled" status, or an error.
func (s stackConfigurations) AwaitStatus(ctx context.Context, stackConfigurationID string, status StackConfigurationStatus) <-chan WaitForStatusResult {
	return awaitPoll(ctx, stackConfigurationID, func(ctx context.Context) (string, error) {
		stackConfiguration, err := s.Read(ctx, stackConfigurationID)
		if err != nil {
			return "", err
		}

		return stackConfiguration.Status.String(), nil
	}, []string{status.String(), StackConfigurationStatusFailed.String()})
}

// StackConfigurationList represents a paginated list of stack configurations.
type StackConfigurationList struct {
	Pagination *Pagination
	Items      []*StackConfiguration
}

// StackConfigurationListOptions represents the options for listing stack configurations.
type StackConfigurationListOptions struct {
	ListOptions
}

func (s stackConfigurations) List(ctx context.Context, stackID string, options *StackConfigurationListOptions) (*StackConfigurationList, error) {
	if options == nil {
		options = &StackConfigurationListOptions{}
	}

	req, err := s.client.NewRequest("GET", fmt.Sprintf("stacks/%s/stack-configurations", url.PathEscape(stackID)), options)
	if err != nil {
		return nil, err
	}

	result := &StackConfigurationList{}
	err = req.Do(ctx, result)
	if err != nil {
		return nil, err
	}

	return result, nil
}

type CreateStackConfigurationOptions struct {
	SelectedDeployments []string `jsonapi:"attr,selected-deployments,omitempty"`
	SpeculativeEnabled  *bool    `jsonapi:"attr,speculative,omitempty"`
}

// CreateAndUpload packages and uploads the specified Terraform Stacks
// configuration files in association with a Stack.
func (s stackConfigurations) CreateAndUpload(ctx context.Context, stackID, path string, opts *CreateStackConfigurationOptions) (*StackConfiguration, error) {
	if opts == nil {
		opts = &CreateStackConfigurationOptions{}
	}
	u := fmt.Sprintf("stacks/%s/stack-configurations", url.PathEscape(stackID))
	req, err := s.client.NewRequest("POST", u, opts)
	if err != nil {
		return nil, fmt.Errorf("error creating stack configuration request for stack %q: %w", stackID, err)
	}

	sc := &StackConfiguration{}
	err = req.Do(ctx, sc)
	if err != nil {
		return nil, fmt.Errorf("error creating stack configuration for stack %q: %w", stackID, err)
	}

	uploadURL, err := s.pollForUploadURL(ctx, sc.ID)
	if err != nil {
		return nil, fmt.Errorf("error retrieving upload URL for stack configuration %q: %w", sc.ID, err)
	}

	body, err := packContents(path)
	if err != nil {
		return nil, err
	}

	err = s.UploadTarGzip(ctx, uploadURL, body)
	if err != nil {
		return nil, err
	}

	return sc, nil
}

// PollForUploadURL polls for the upload URL of a stack configuration until it becomes available.
// It makes a request every 2 seconds until the upload URL is present in the response.
// It will timeout after 10 seconds.
func (s stackConfigurations) pollForUploadURL(ctx context.Context, stackConfigurationID string) (string, error) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	timeout := time.NewTimer(15 * time.Second)
	defer timeout.Stop()

	for {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-timeout.C:
			return "", fmt.Errorf("timeout waiting for upload URL for stack configuration %q", stackConfigurationID)
		case <-ticker.C:
			urlReq, err := s.client.NewRequest("GET", fmt.Sprintf("stack-configurations/%s/upload-url", stackConfigurationID), nil)
			if err != nil {
				return "", fmt.Errorf("error creating upload URL request for stack configuration %q: %w", stackConfigurationID, err)
			}

			type UploadURLResponse struct {
				Data struct {
					SourceUploadURL *string `json:"source-upload-url"`
				} `json:"data"`
			}

			uploadResp := &UploadURLResponse{}
			err = urlReq.DoJSON(ctx, uploadResp)
			if err != nil {
				return "", fmt.Errorf("error getting upload URL for stack configuration %q: %w", stackConfigurationID, err)
			}

			if uploadResp.Data.SourceUploadURL != nil {
				return *uploadResp.Data.SourceUploadURL, nil
			}
		}
	}
}

// UploadTarGzip is used to upload Terraform configuration files contained a tar gzip archive.
// Any stream implementing io.Reader can be passed into this method. This method is also
// particularly useful for tar streams created by non-default go-slug configurations.
//
// **Note**: This method does not validate the content being uploaded and is therefore the caller's
// responsibility to ensure the raw content is a valid Terraform configuration.
func (s stackConfigurations) UploadTarGzip(ctx context.Context, uploadURL string, archive io.Reader) error {
	return s.client.doForeignPUTRequest(ctx, uploadURL, archive)
}

// Diagnostics returns the diagnostics for this stack configuration.
func (s stackConfigurations) Diagnostics(ctx context.Context, stackConfigurationID string) (*StackDiagnosticsList, error) {
	req, err := s.client.NewRequest("GET", fmt.Sprintf("stack-configurations/%s/stack-diagnostics", url.PathEscape(stackConfigurationID)), nil)

	if err != nil {
		return nil, err
	}

	diagnostics := &StackDiagnosticsList{}
	err = req.Do(ctx, diagnostics)
	if err != nil {
		return nil, err
	}
	return diagnostics, nil
}

type StackDeploymentGroupSummaries interface {
	// List lists all the stack deployment group summaries for a stack.
	List(ctx context.Context, configurationID string, options *StackDeploymentGroupSummaryListOptions) (*StackDeploymentGroupSummaryList, error)
}

type stackDeploymentGroupSummaries struct {
	client *Client
}

var _ StackDeploymentGroupSummaries = &stackDeploymentGroupSummaries{}

type StackDeploymentGroupSummaryList struct {
	*Pagination
	Items []*StackDeploymentGroupSummary
}

type StackDeploymentGroupSummaryListOptions struct {
	ListOptions
}

type StackDeploymentGroupStatusCounts struct {
	Pending                     int `jsonapi:"attr,pending"`
	PreDeploying                int `jsonapi:"attr,pre-deploying"`
	PreDeployingPendingOperator int `jsonapi:"attr,pending-operator"`
	AcquiringLock               int `jsonapi:"attr,acquiring-lock"`
	Deploying                   int `jsonapi:"attr,deploying"`
	Succeeded                   int `jsonapi:"attr,succeeded"`
	Failed                      int `jsonapi:"attr,failed"`
	Abandoned                   int `jsonapi:"attr,abandoned"`
}

type StackDeploymentGroupSummary struct {
	ID string `jsonapi:"primary,stack-deployment-group-summaries"`

	// Attributes
	Name         string                            `jsonapi:"attr,name"`
	Status       string                            `jsonapi:"attr,status"`
	StatusCounts *StackDeploymentGroupStatusCounts `jsonapi:"attr,status-counts"`

	// Relationships
	StackDeploymentGroup *StackDeploymentGroup `jsonapi:"relation,stack-deployment-group"`
}

func (s stackDeploymentGroupSummaries) List(ctx context.Context, stackID string, options *StackDeploymentGroupSummaryListOptions) (*StackDeploymentGroupSummaryList, error) {
	if !validStringID(&stackID) {
		return nil, fmt.Errorf("invalid stack ID: %s", stackID)
	}

	if options == nil {
		options = &StackDeploymentGroupSummaryListOptions{}
	}

	req, err := s.client.NewRequest("GET", fmt.Sprintf("stack-configurations/%s/stack-deployment-group-summaries", url.PathEscape(stackID)), options)
	if err != nil {
		return nil, err
	}

	scl := &StackDeploymentGroupSummaryList{}
	err = req.Do(ctx, scl)
	if err != nil {
		return nil, err
	}

	return scl, nil
}

// StackDeploymentGroups describes all the stack-deployment-groups related methods that the HCP Terraform API supports.
type StackDeploymentGroups interface {
	// List returns a list of Deployment Groups in a stack.
	List(ctx context.Context, stackConfigID string, options *StackDeploymentGroupListOptions) (*StackDeploymentGroupList, error)

	// Read retrieves a stack deployment group by its ID.
	Read(ctx context.Context, stackDeploymentGroupID string) (*StackDeploymentGroup, error)

	// ReadByName retrieves a stack deployment group by its Name.
	ReadByName(ctx context.Context, stackConfigurationID, stackDeploymentName string) (*StackDeploymentGroup, error)

	// ApproveAllPlans approves all pending plans in a stack deployment group.
	ApproveAllPlans(ctx context.Context, stackDeploymentGroupID string) error

	// Rerun re-runs all the stack deployment runs in a deployment group.
	Rerun(ctx context.Context, stackDeploymentGroupID string, options *StackDeploymentGroupRerunOptions) error
}

type DeploymentGroupStatus string

const (
	DeploymentGroupStatusPending   DeploymentGroupStatus = "pending"
	DeploymentGroupStatusDeploying DeploymentGroupStatus = "deploying"
	DeploymentGroupStatusSucceeded DeploymentGroupStatus = "succeeded"
	DeploymentGroupStatusFailed    DeploymentGroupStatus = "failed"
	DeploymentGroupStatusAbandoned DeploymentGroupStatus = "abandoned"
)

func (s DeploymentGroupStatus) String() string {
	return string(s)
}

// stackDeploymentGroups implements StackDeploymentGroups.
type stackDeploymentGroups struct {
	client *Client
}

var _ StackDeploymentGroups = &stackDeploymentGroups{}

// StackDeploymentGroup represents a stack deployment group.
type StackDeploymentGroup struct {
	// Attributes
	ID        string                `jsonapi:"primary,stack-deployment-groups"`
	Name      string                `jsonapi:"attr,name"`
	Status    DeploymentGroupStatus `jsonapi:"attr,status"`
	CreatedAt time.Time             `jsonapi:"attr,created-at,iso8601"`
	UpdatedAt time.Time             `jsonapi:"attr,updated-at,iso8601"`

	// Relationships
	StackConfiguration *StackConfiguration `jsonapi:"relation,stack-configuration"`
}

// StackDeploymentGroupList represents a list of stack deployment groups.
type StackDeploymentGroupList struct {
	*Pagination
	Items []*StackDeploymentGroup
}

// StackDeploymentGroupListOptions represents additional options when listing stack deployment groups.
type StackDeploymentGroupListOptions struct {
	ListOptions
}

// StackDeploymentGroupRerunOptions represents options for rerunning deployments in a stack deployment group.
type StackDeploymentGroupRerunOptions struct {
	// Required query parameter: A list of deployment run IDs to rerun.
	Deployments []string
}

// List returns a list of Deployment Groups in a stack, optionally filtered by additional parameters.
func (s stackDeploymentGroups) List(ctx context.Context, stackConfigID string, options *StackDeploymentGroupListOptions) (*StackDeploymentGroupList, error) {
	if !validStringID(&stackConfigID) {
		return nil, fmt.Errorf("invalid stack configuration ID: %s", stackConfigID)
	}

	if options == nil {
		options = &StackDeploymentGroupListOptions{}
	}

	req, err := s.client.NewRequest("GET", fmt.Sprintf("stack-configurations/%s/stack-deployment-groups", url.PathEscape(stackConfigID)), options)
	if err != nil {
		return nil, err
	}

	sdgl := &StackDeploymentGroupList{}
	err = req.Do(ctx, sdgl)
	if err != nil {
		return nil, err
	}

	return sdgl, nil
}

// ReadByName retrieves a stack deployment group by its Name.
func (s stackDeploymentGroups) ReadByName(ctx context.Context, stackConfigurationID, stackDeploymentName string) (*StackDeploymentGroup, error) {
	if !validStringID(&stackConfigurationID) {
		return nil, fmt.Errorf("invalid stack configuration id: %s", stackConfigurationID)
	}
	if !validStringID(&stackDeploymentName) {
		return nil, fmt.Errorf("invalid stack deployment group name: %s", stackDeploymentName)
	}

	req, err := s.client.NewRequest("GET", fmt.Sprintf("stack-configurations/%s/stack-deployment-groups/%s", url.PathEscape(stackConfigurationID), url.PathEscape(stackDeploymentName)), nil)
	if err != nil {
		return nil, err
	}

	sdg := &StackDeploymentGroup{}
	err = req.Do(ctx, sdg)
	if err != nil {
		return nil, err
	}

	return sdg, nil
}

// Read retrieves a stack deployment group by its ID.
func (s stackDeploymentGroups) Read(ctx context.Context, stackDeploymentGroupID string) (*StackDeploymentGroup, error) {
	if !validStringID(&stackDeploymentGroupID) {
		return nil, fmt.Errorf("invalid stack deployment group ID: %s", stackDeploymentGroupID)
	}

	req, err := s.client.NewRequest("GET", fmt.Sprintf("stack-deployment-groups/%s", url.PathEscape(stackDeploymentGroupID)), nil)
	if err != nil {
		return nil, err
	}

	sdg := &StackDeploymentGroup{}
	err = req.Do(ctx, sdg)
	if err != nil {
		return nil, err
	}

	return sdg, nil
}

// ApproveAllPlans approves all pending plans in a stack deployment group.
func (s stackDeploymentGroups) ApproveAllPlans(ctx context.Context, stackDeploymentGroupID string) error {
	if !validStringID(&stackDeploymentGroupID) {
		return fmt.Errorf("invalid stack deployment group ID: %s", stackDeploymentGroupID)
	}

	req, err := s.client.NewRequest("POST", fmt.Sprintf("stack-deployment-groups/%s/approve-all-plans", url.PathEscape(stackDeploymentGroupID)), nil)
	if err != nil {
		return err
	}

	return req.Do(ctx, nil)
}

// Rerun re-runs all the stack deployment runs in a deployment group.
func (s stackDeploymentGroups) Rerun(ctx context.Context, stackDeploymentGroupID string, options *StackDeploymentGroupRerunOptions) error {
	if !validStringID(&stackDeploymentGroupID) {
		return fmt.Errorf("invalid stack deployment group ID: %s", stackDeploymentGroupID)
	}

	if options == nil || len(options.Deployments) == 0 {
		return fmt.Errorf("no deployments specified for rerun")
	}

	u := fmt.Sprintf("stack-deployment-groups/%s/rerun", url.PathEscape(stackDeploymentGroupID))

	type DeploymentQueryParams struct {
		Deployments string `url:"deployments"`
	}

	qp, err := decodeQueryParams(&DeploymentQueryParams{
		Deployments: strings.Join(options.Deployments, ","),
	})
	if err != nil {
		return err
	}
	req, err := s.client.NewRequestWithAdditionalQueryParams("POST", u, nil, qp)
	if err != nil {
		return err
	}

	return req.Do(ctx, nil)
}

// StackDeploymentRuns describes all the stack deployment runs-related methods that the HCP Terraform API supports.
type StackDeploymentRuns interface {
	// List returns a list of stack deployment runs for a given deployment group.
	List(ctx context.Context, deploymentGroupID string, options *StackDeploymentRunListOptions) (*StackDeploymentRunList, error)
	Read(ctx context.Context, stackDeploymentRunID string) (*StackDeploymentRun, error)
	ReadWithOptions(ctx context.Context, stackDeploymentRunID string, options *StackDeploymentRunReadOptions) (*StackDeploymentRun, error)
	ApproveAllPlans(ctx context.Context, deploymentRunID string) error
	Cancel(ctx context.Context, stackDeploymentRunID string) error
}

type DeploymentRunStatus string

const (
	DeploymentRunStatusPending                     DeploymentRunStatus = "pending"
	DeploymentRunStatusPreDeploying                DeploymentRunStatus = "pre-deploying"
	DeploymentRunStatusPreDeployingPendingOperator DeploymentRunStatus = "pre-deploying-pending-operator"
	DeploymentRunStatusAcquiringLock               DeploymentRunStatus = "acquiring-lock"
	DeploymentRunStatusDeploying                   DeploymentRunStatus = "deploying"
	DeploymentRunStatusDeployingPendingOperator    DeploymentRunStatus = "deploying-pending-operator"
	DeploymentRunStatusSucceeded                   DeploymentRunStatus = "succeeded"
	DeploymentRunStatusFailed                      DeploymentRunStatus = "failed"
	DeploymentRunStatusAbandoned                   DeploymentRunStatus = "abandoned"
)

func (s DeploymentRunStatus) String() string {
	return string(s)
}

// stackDeploymentRuns implements StackDeploymentRuns.
type stackDeploymentRuns struct {
	client *Client
}

var _ StackDeploymentRuns = &stackDeploymentRuns{}

// StackDeploymentRun represents a stack deployment run.
type StackDeploymentRun struct {
	ID        string              `jsonapi:"primary,stack-deployment-runs"`
	Status    DeploymentRunStatus `jsonapi:"attr,status"`
	CreatedAt time.Time           `jsonapi:"attr,created-at,iso8601"`
	UpdatedAt time.Time           `jsonapi:"attr,updated-at,iso8601"`

	// Relationships
	StackDeploymentGroup *StackDeploymentGroup `jsonapi:"relation,stack-deployment-group"`
}

type SDRIncludeOpt string

const (
	SDRDeploymentGroup SDRIncludeOpt = "stack-deployment-group"
)

// StackDeploymentRunList represents a list of stack deployment runs.
type StackDeploymentRunList struct {
	*Pagination
	Items []*StackDeploymentRun
}

type StackDeploymentRunReadOptions struct {
	// Optional: A list of relations to include.
	Include []SDRIncludeOpt `url:"include,omitempty"`
}

// StackDeploymentRunListOptions represents the options for listing stack deployment runs.
type StackDeploymentRunListOptions struct {
	ListOptions
	// Optional: A list of relations to include.
	Include []SDRIncludeOpt `url:"include,omitempty"`
}

// List returns a list of stack deployment runs for a given deployment group.
func (s *stackDeploymentRuns) List(ctx context.Context, deploymentGroupID string, options *StackDeploymentRunListOptions) (*StackDeploymentRunList, error) {
	req, err := s.client.NewRequest("GET", fmt.Sprintf("stack-deployment-groups/%s/stack-deployment-runs", url.PathEscape(deploymentGroupID)), options)
	if err != nil {
		return nil, err
	}

	sdrl := &StackDeploymentRunList{}
	err = req.Do(ctx, sdrl)
	if err != nil {
		return nil, err
	}

	return sdrl, nil
}

func (s stackDeploymentRuns) Read(ctx context.Context, stackDeploymentRunID string) (*StackDeploymentRun, error) {
	req, err := s.client.NewRequest("GET", fmt.Sprintf("stack-deployment-runs/%s", url.PathEscape(stackDeploymentRunID)), nil)
	if err != nil {
		return nil, err
	}

	run := StackDeploymentRun{}
	err = req.Do(ctx, &run)
	if err != nil {
		return nil, err
	}

	return &run, nil
}

func (s stackDeploymentRuns) ReadWithOptions(ctx context.Context, stackDeploymentRunID string, options *StackDeploymentRunReadOptions) (*StackDeploymentRun, error) {
	if err := options.valid(); err != nil {
		return nil, err
	}

	req, err := s.client.NewRequest("GET", fmt.Sprintf("stack-deployment-runs/%s", url.PathEscape(stackDeploymentRunID)), options)
	if err != nil {
		return nil, err
	}

	run := StackDeploymentRun{}
	err = req.Do(ctx, &run)
	if err != nil {
		return nil, err
	}

	return &run, nil
}

func (s stackDeploymentRuns) ApproveAllPlans(ctx context.Context, stackDeploymentRunID string) error {
	req, err := s.client.NewRequest("POST", fmt.Sprintf("stack-deployment-runs/%s/approve-all-plans", url.PathEscape(stackDeploymentRunID)), nil)
	if err != nil {
		return err
	}

	return req.Do(ctx, nil)
}

func (s stackDeploymentRuns) Cancel(ctx context.Context, stackDeploymentRunID string) error {
	req, err := s.client.NewRequest("POST", fmt.Sprintf("stack-deployment-runs/%s/cancel", url.PathEscape(stackDeploymentRunID)), nil)
	if err != nil {
		return err
	}

	return req.Do(ctx, nil)
}

func (o *StackDeploymentRunReadOptions) valid() error {
	for _, include := range o.Include {
		switch include {
		case SDRDeploymentGroup:
			// Valid option, do nothing.
		default:
			return fmt.Errorf("invalid include option: %s", include)
		}
	}
	return nil
}

// StackDeploymentSteps describes all the stacks deployment step-related methods that the
// HCP Terraform API supports.
type StackDeploymentSteps interface {
	// List returns the stack deployment steps for a stack deployment run.
	List(ctx context.Context, stackDeploymentRunID string, opts *StackDeploymentStepsListOptions) (*StackDeploymentStepList, error)
	// Read returns a stack deployment step by its ID.
	Read(ctx context.Context, stackDeploymentStepID string) (*StackDeploymentStep, error)
	// Advance advances the stack deployment step when in the "pending_operator" state.
	Advance(ctx context.Context, stackDeploymentStepID string) error
	// Diagnostics returns the diagnostics for this stack deployment step.
	Diagnostics(ctx context.Context, stackConfigurationID string) (*StackDiagnosticsList, error)
	// Artifacts returns the artifacts for this stack deployment step.
	// Valid artifact names are "plan-description" and "apply-description".
	Artifacts(ctx context.Context, stackDeploymentStepID string, artifactType StackDeploymentStepArtifactType) (io.ReadCloser, error)
}

type StackDeploymentStepArtifactType string

const (
	// StackDeploymentStepArtifactPlanDescription represents the plan description artifact type.
	StackDeploymentStepArtifactPlanDescription StackDeploymentStepArtifactType = "plan-description"
	// StackDeploymentStepArtifactApplyDescription represents the apply description artifact type.
	StackDeploymentStepArtifactApplyDescription StackDeploymentStepArtifactType = "apply-description"
	// StackDeploymentStepArtifactPlanDescription represents the plan debug log artifact type.
	StackDeploymentStepArtifactPlanDebugLog StackDeploymentStepArtifactType = "plan-debug-log"
	// StackDeploymentStepArtifactApplyDescription represents the apply debug log artifact type.
	StackDeploymentStepArtifactApplyDebugLog StackDeploymentStepArtifactType = "apply-debug-log"
)

type DeploymentStepStatus string

const (
	DeploymentStepStatusBlocked         DeploymentStepStatus = "blocked"
	DeploymentStepStatusAbandoned       DeploymentStepStatus = "abandoned"
	DeploymentStepStatusQueued          DeploymentStepStatus = "queued"
	DeploymentStepStatusRunning         DeploymentStepStatus = "running"
	DeploymentStepStatusPendingOperator DeploymentStepStatus = "pending-operator"
	DeploymentStepStatusCompleted       DeploymentStepStatus = "completed"
	DeploymentStepStatusFailed          DeploymentStepStatus = "failed"
)

func (s DeploymentStepStatus) String() string {
	return string(s)
}

// StackDeploymentStep represents a step from a stack deployment
type StackDeploymentStep struct {
	// Attributes
	ID            string               `jsonapi:"primary,stack-deployment-steps"`
	Status        DeploymentStepStatus `jsonapi:"attr,status"`
	OperationType string               `jsonapi:"attr,operation-type"`
	CreatedAt     time.Time            `jsonapi:"attr,created-at,iso8601"`
	UpdatedAt     time.Time            `jsonapi:"attr,updated-at,iso8601"`

	// Links
	Links map[string]interface{} `jsonapi:"links,omitempty"`

	// Relationships
	StackDeploymentRun *StackDeploymentRun `jsonapi:"relation,stack-deployment-run"`
}

// StackDeploymentStepList represents a list of stack deployment steps
type StackDeploymentStepList struct {
	*Pagination
	Items []*StackDeploymentStep
}

type stackDeploymentSteps struct {
	client *Client
}

// StackDeploymentStepsListOptions represents the options for listing stack
// deployment steps.
type StackDeploymentStepsListOptions struct {
	ListOptions
}

func (s stackDeploymentSteps) List(ctx context.Context, stackDeploymentRunID string, opts *StackDeploymentStepsListOptions) (*StackDeploymentStepList, error) {
	req, err := s.client.NewRequest("GET", fmt.Sprintf("stack-deployment-runs/%s/stack-deployment-steps", url.PathEscape(stackDeploymentRunID)), opts)
	if err != nil {
		return nil, err
	}

	steps := StackDeploymentStepList{}
	err = req.Do(ctx, &steps)
	if err != nil {
		return nil, err
	}

	return &steps, nil
}

// Read returns a stack deployment step by its ID.
func (s stackDeploymentSteps) Read(ctx context.Context, stackDeploymentStepID string) (*StackDeploymentStep, error) {
	req, err := s.client.NewRequest("GET", fmt.Sprintf("stack-deployment-steps/%s", url.PathEscape(stackDeploymentStepID)), nil)
	if err != nil {
		return nil, err
	}

	step := StackDeploymentStep{}
	err = req.Do(ctx, &step)
	if err != nil {
		return nil, err
	}

	return &step, nil
}

// Advance advances the stack deployment step when in the "pending_operator" state.
func (s stackDeploymentSteps) Advance(ctx context.Context, stackDeploymentStepID string) error {
	req, err := s.client.NewRequest("POST", fmt.Sprintf("stack-deployment-steps/%s/advance", url.PathEscape(stackDeploymentStepID)), nil)
	if err != nil {
		return err
	}

	return req.Do(ctx, nil)
}

// Diagnostics returns the diagnostics for this stack deployment step.
func (s stackDeploymentSteps) Diagnostics(ctx context.Context, stackDeploymentStepID string) (*StackDiagnosticsList, error) {
	req, err := s.client.NewRequest("GET", fmt.Sprintf("stack-deployment-steps/%s/stack-diagnostics", url.PathEscape(stackDeploymentStepID)), nil)
	if err != nil {
		return nil, err
	}
	diagnostics := &StackDiagnosticsList{}
	err = req.Do(ctx, diagnostics)
	if err != nil {
		return nil, err
	}
	return diagnostics, nil
}

// Artifacts returns the artifacts for this stack deployment step.
// Valid artifact names are "plan-description" and "apply-description".
func (s stackDeploymentSteps) Artifacts(ctx context.Context, stackDeploymentStepID string, artifactType StackDeploymentStepArtifactType) (io.ReadCloser, error) {
	req, err := s.client.NewRequestWithAdditionalQueryParams("GET",
		fmt.Sprintf("stack-deployment-steps/%s/artifacts", url.PathEscape(stackDeploymentStepID)),
		nil,
		map[string][]string{"name": {url.PathEscape(string(artifactType))}},
	)
	if err != nil {
		return nil, err
	}

	return req.DoRaw(ctx)
}

type StackDeployments interface {
	// List returns a list of stack deployments for a given stack.
	List(ctx context.Context, stackID string, opts *StackDeploymentListOptions) (*StackDeploymentList, error)
}

type StackDeployment struct {
	// Attributes
	ID   string `jsonapi:"primary,stack-deployments"`
	Name string `jsonapi:"attr,name"`

	// Relationships
	Stack               *Stack              `jsonapi:"relation,stack"`
	LatestDeploymentRun *StackDeploymentRun `jsonapi:"relation,latest-deployment-run"`
}

type stackDeployments struct {
	client *Client
}

type StackDeploymentListOptions struct {
	ListOptions
}

type StackDeploymentList struct {
	*Pagination
	Items []*StackDeployment
}

func (s stackDeployments) List(ctx context.Context, stackID string, opts *StackDeploymentListOptions) (*StackDeploymentList, error) {
	if !validStringID(&stackID) {
		return nil, ErrInvalidStackID
	}

	req, err := s.client.NewRequest("GET", fmt.Sprintf("stacks/%s/stack-deployments", url.PathEscape(stackID)), opts)
	if err != nil {
		return nil, err
	}

	var deployments StackDeploymentList
	if err := req.Do(ctx, &deployments); err != nil {
		return nil, err
	}

	return &deployments, nil
}

type StackDiagnostics interface {
	// Read retrieves a stack diagnostic associated with a stack configuration by its ID.
	Read(ctx context.Context, stackConfigurationID string) (*StackDiagnostic, error)
}

// StackDiagnostic represents any sourcebundle.Diagnostic value. The simplest form has
// just a severity, single line summary, and optional detail. If there is more
// information about the source of the diagnostic, this is represented in the
// range field.
type StackDiagnostic struct {
	ID        string                    `jsonapi:"primary,stack-diagnostics"`
	Severity  string                    `jsonapi:"attr,severity"`
	Summary   string                    `jsonapi:"attr,summary"`
	Detail    string                    `jsonapi:"attr,detail"`
	Diags     []*StackDiagnosticSummary `jsonapi:"attr,diags"`
	CreatedAt *time.Time                `jsonapi:"attr,created-at,iso8601"`

	// Relationships
	StackDeploymentStep *StackDeploymentStep `jsonapi:"relation,stack-deployment-step"`
	StackConfiguration  *StackConfiguration  `jsonapi:"relation,stack-configuration"`
}

type StackDiagnosticSummary struct {
	Severity string             `jsonapi:"attr,severity"`
	Summary  string             `jsonapi:"attr,summary"`
	Detail   string             `jsonapi:"attr,detail"`
	Range    *DiagnosticRange   `jsonapi:"attr,range"`
	Origin   string             `jsonapi:"attr,origin"`
	Snippet  *DiagnosticSnippet `jsonapi:"attr,snippet"`
}

type DiagnosticSnippet struct {
	Code                 string   `jsonapi:"attr,code"`
	Values               []string `jsonapi:"attr,values"`
	Context              *string  `jsonapi:"attr,context"`
	StartLine            int      `jsonapi:"attr,start_line"`
	HighlightEndOffset   int      `jsonapi:"attr,highlight_end_offset"`
	HighlightStartOffset int      `jsonapi:"attr,highlight_start_offset"`
}

type stackDiagnostics struct {
	client *Client
}

type StackDiagnosticsList struct {
	Items []*StackDiagnostic
}

// DiagnosticPos represents a position in the source code.
type DiagnosticPos struct {
	// Line is a one-based count for the line in the indicated file.
	Line int `jsonapi:"attr,line"`

	// Column is a one-based count of Unicode characters from the start of the line.
	Column int `jsonapi:"attr,column"`

	// Byte is a zero-based offset into the indicated file.
	Byte int `jsonapi:"attr,byte"`
}

// DiagnosticRange represents the filename and position of the diagnostic
// subject. This defines the range of the source to be highlighted in the
// output. Note that the snippet may include additional surrounding source code
// if the diagnostic has a context range.
//
// The stacks-specific source field represents the full source bundle address
// of the file, while the filename field is the sub path relative to its
// enclosing package. This represents an attempt to be somewhat backwards
// compatible with the existing Terraform JSON diagnostic format, where
// filename is root module relative.
//
// The Start position is inclusive, and the End position is exclusive. Exact
// positions are intended for highlighting for human interpretation only and
// are subject to change.
type DiagnosticRange struct {
	Filename string        `jsonapi:"attr,filename"`
	Source   string        `jsonapi:"attr,source"`
	Start    DiagnosticPos `jsonapi:"attr,start"`
	End      DiagnosticPos `jsonapi:"attr,end"`
}

// Read retrieves a stack diagnostic by its ID.
func (s stackDiagnostics) Read(ctx context.Context, stackDiagnosticID string) (*StackDiagnostic, error) {
	req, err := s.client.NewRequest("GET", fmt.Sprintf("stack-diagnostics/%s", url.PathEscape(stackDiagnosticID)), nil)
	if err != nil {
		return nil, err
	}

	var diagnostics StackDiagnostic
	if err := req.Do(ctx, &diagnostics); err != nil {
		return nil, err
	}

	return &diagnostics, nil
}

// StackState describes all the stack state-related methods that the
// HCP Terraform API supports.
type StackStates interface {
	// List returns the stack states for a stack.
	List(ctx context.Context, stackID string, opts *StackStateListOptions) (*StackStateList, error)
	// Read returns a stack state by its ID.
	Read(ctx context.Context, stackStateID string) (*StackState, error)
	// Description returns the state description for the given stack state.
	// The description is returned as an io.ReadCloser and should be closed and
	// unmarshaled by the caller.
	Description(ctx context.Context, stackStateID string) (io.ReadCloser, error)
}

// StackState represents a stack state.
type StackState struct {
	// Attributes
	ID                    string            `jsonapi:"primary,stack-states"`
	Generation            int               `jsonapi:"attr,generation"`
	Status                string            `jsonapi:"attr,status"`
	Deployment            string            `jsonapi:"attr,deployment"`
	Components            []*StackComponent `jsonapi:"attr,components"`
	IsCurrent             bool              `jsonapi:"attr,is-current"`
	ResourceInstanceCount int               `jsonapi:"attr,resource-instance-count"`

	// Relationships
	Stack              *Stack              `jsonapi:"relation,stack"`
	StackDeploymentRun *StackDeploymentRun `jsonapi:"relation,stack-deployment-run"`

	// Links
	Links map[string]interface{} `jsonapi:"links,omitempty"`
}

// StackStateList represents a list of stack states.
type StackStateList struct {
	*Pagination
	Items []*StackState
}

type stackStates struct {
	client *Client
}

// StackStateListOptions represents the options for listing stack states.
type StackStateListOptions struct {
	ListOptions
}

// List returns the stack states for a stack.
func (s stackStates) List(ctx context.Context, stackID string, opts *StackStateListOptions) (*StackStateList, error) {
	req, err := s.client.NewRequest("GET", fmt.Sprintf("stacks/%s/stack-states", url.PathEscape(stackID)), opts)
	if err != nil {
		return nil, err
	}

	states := StackStateList{}
	if err := req.Do(ctx, &states); err != nil {
		return nil, err
	}

	return &states, nil
}

// Read returns a stack state by its ID.
func (s stackStates) Read(ctx context.Context, stackStateID string) (*StackState, error) {
	req, err := s.client.NewRequest("GET", fmt.Sprintf("stack-states/%s", url.PathEscape(stackStateID)), nil)
	if err != nil {
		return nil, err
	}

	state := StackState{}
	if err := req.Do(ctx, &state); err != nil {
		return nil, err
	}

	return &state, nil
}

// Description returns the state description for the given stack state.
func (s stackStates) Description(ctx context.Context, stackStateID string) (io.ReadCloser, error) {
	req, err := s.client.NewRequest("GET", fmt.Sprintf("stack-states/%s/description", url.PathEscape(stackStateID)), nil)
	if err != nil {
		return nil, err
	}

	return req.DoRaw(ctx)
}

// Stacks describes all the stacks-related methods that the HCP Terraform API supports.
type Stacks interface {
	// List returns a list of stacks, optionally filtered by project.
	List(ctx context.Context, organization string, options *StackListOptions) (*StackList, error)

	// Read returns a stack by its ID.
	Read(ctx context.Context, stackID string) (*Stack, error)

	// Create creates a new stack.
	Create(ctx context.Context, options StackCreateOptions) (*Stack, error)

	// Update updates a stack.
	Update(ctx context.Context, stackID string, options StackUpdateOptions) (*Stack, error)

	// Delete deletes a stack.
	Delete(ctx context.Context, stackID string) error

	// ForceDelete deletes a stack.
	ForceDelete(ctx context.Context, stackID string) error

	// FetchLatestFromVcs updates the configuration of a stack, triggering stack preparation.
	FetchLatestFromVcs(ctx context.Context, stackID string) (*Stack, error)
}

// stacks implements Stacks.
type stacks struct {
	client *Client
}

var _ Stacks = &stacks{}

// StackSortColumn represents a string that can be used to sort items when using
// the List method.
type StackSortColumn string

const (
	// StackSortByName sorts by the name attribute.
	StackSortByName StackSortColumn = "name"

	// StackSortByUpdatedAt sorts by the updated-at attribute.
	StackSortByUpdatedAt StackSortColumn = "updated-at"

	// StackSortByNameDesc sorts by the name attribute in descending order.
	StackSortByNameDesc StackSortColumn = "-name"

	// StackSortByUpdatedAtDesc sorts by the updated-at attribute in descending order.
	StackSortByUpdatedAtDesc StackSortColumn = "-updated-at"
)

// StackList represents a list of stacks.
type StackList struct {
	*Pagination
	Items []*Stack
}

// StackVCSRepo represents the version control system repository for a stack.
type StackVCSRepo struct {
	Identifier        string `jsonapi:"attr,identifier"`
	Branch            string `jsonapi:"attr,branch,omitempty"`
	GHAInstallationID string `jsonapi:"attr,github-app-installation-id,omitempty"`
	OAuthTokenID      string `jsonapi:"attr,oauth-token-id,omitempty"`
}

// StackVCSRepoOptions
type StackVCSRepoOptions struct {
	Identifier        string `json:"identifier"`
	Branch            string `json:"branch,omitempty"`
	GHAInstallationID string `json:"github-app-installation-id,omitempty"`
	OAuthTokenID      string `json:"oauth-token-id,omitempty"`
}

// Stack represents a stack.
type Stack struct {
	ID                 string        `jsonapi:"primary,stacks"`
	Name               string        `jsonapi:"attr,name"`
	Description        string        `jsonapi:"attr,description"`
	VCSRepo            *StackVCSRepo `jsonapi:"attr,vcs-repo"`
	SpeculativeEnabled bool          `jsonapi:"attr,speculative-enabled"`
	CreatedAt          time.Time     `jsonapi:"attr,created-at,iso8601"`
	UpdatedAt          time.Time     `jsonapi:"attr,updated-at,iso8601"`
	UpstreamCount      int           `jsonapi:"attr,upstream-count"`
	DownstreamCount    int           `jsonapi:"attr,downstream-count"`
	InputsCount        int           `jsonapi:"attr,inputs-count"`
	OutputsCount       int           `jsonapi:"attr,outputs-count"`
	CreationSource     string        `jsonapi:"attr,creation-source"`
	WorkingDirectory   string        `jsonapi:"attr,working-directory,omitempty"`
	TriggerPatterns    []string      `jsonapi:"attr,trigger-patterns,omitempty"`

	// Relationships
	Project                  *Project            `jsonapi:"relation,project"`
	AgentPool                *AgentPool          `jsonapi:"relation,agent-pool"`
	LatestStackConfiguration *StackConfiguration `jsonapi:"relation,latest-stack-configuration"`
}

// StackConfigurationStatusTimestamps represents the timestamps for a stack configuration
type StackConfigurationStatusTimestamps struct {
	QueuedAt     *time.Time `jsonapi:"attr,queued-at,omitempty,rfc3339"`
	CompletedAt  *time.Time `jsonapi:"attr,completed-at,omitempty,rfc3339"`
	PreparingAt  *time.Time `jsonapi:"attr,preparing-at,omitempty,rfc3339"`
	EnqueueingAt *time.Time `jsonapi:"attr,enqueueing-at,omitempty,rfc3339"`
	CanceledAt   *time.Time `jsonapi:"attr,canceled-at,omitempty,rfc3339"`
	ErroredAt    *time.Time `jsonapi:"attr,errored-at,omitempty,rfc3339"`
}

// StackComponent represents a stack component, specified by configuration
type StackComponent struct {
	Name       string `json:"name"`
	Correlator string `json:"correlator"`
	Expanded   bool   `json:"expanded"`
	Removed    bool   `json:"removed"`
}

// StackConfiguration represents a stack configuration snapshot
type StackConfiguration struct {
	// Attributes
	ID                      string                   `jsonapi:"primary,stack-configurations"`
	Status                  StackConfigurationStatus `jsonapi:"attr,status"`
	SequenceNumber          int                      `jsonapi:"attr,sequence-number"`
	Components              []*StackComponent        `jsonapi:"attr,components"`
	PreparingEventStreamURL string                   `jsonapi:"attr,preparing-event-stream-url"`
	CreatedAt               time.Time                `jsonapi:"attr,created-at,iso8601"`
	UpdatedAt               time.Time                `jsonapi:"attr,updated-at,iso8601"`
	Speculative             bool                     `jsonapi:"attr,speculative"`

	// Relationships
	Stack             *Stack             `jsonapi:"relation,stack"`
	IngressAttributes *IngressAttributes `jsonapi:"relation,ingress-attributes"`
}

// StackIncludeOpt represents the include options for a stack.
type StackIncludeOpt string

const (
	StackIncludeOrganization             StackIncludeOpt = "organization"
	StackIncludeProject                  StackIncludeOpt = "project"
	StackIncludeLatestStackConfiguration StackIncludeOpt = "latest_stack_configuration"
	StackIncludeStackDiagnostics         StackIncludeOpt = "latest_stack_configuration.stack_diagnostics"
)

// StackListOptions represents the options for listing stacks.
type StackListOptions struct {
	ListOptions
	ProjectID    string          `url:"filter[project][id],omitempty"`
	Sort         StackSortColumn `url:"sort,omitempty"`
	SearchByName string          `url:"search[name],omitempty"`
}

// StackCreateOptions represents the options for creating a stack. The project
// relation is required.
type StackCreateOptions struct {
	Type               string               `jsonapi:"primary,stacks"`
	Name               string               `jsonapi:"attr,name"`
	Migration          *bool                `jsonapi:"attr,migration,omitempty"`
	SpeculativeEnabled *bool                `jsonapi:"attr,speculative-enabled,omitempty"`
	Description        *string              `jsonapi:"attr,description,omitempty"`
	VCSRepo            *StackVCSRepoOptions `jsonapi:"attr,vcs-repo"`
	Project            *Project             `jsonapi:"relation,project"`
	AgentPool          *AgentPool           `jsonapi:"relation,agent-pool"`
	WorkingDirectory   *string              `jsonapi:"attr,working-directory,omitempty"`
	TriggerPatterns    []string             `jsonapi:"attr,trigger-patterns"`
}

// StackUpdateOptions represents the options for updating a stack.
type StackUpdateOptions struct {
	Name               *string              `jsonapi:"attr,name,omitempty"`
	Description        *string              `jsonapi:"attr,description,omitempty"`
	SpeculativeEnabled *bool                `jsonapi:"attr,speculative-enabled,omitempty"`
	VCSRepo            *StackVCSRepoOptions `jsonapi:"attr,vcs-repo"`
	AgentPool          *AgentPool           `jsonapi:"relation,agent-pool"`
	WorkingDirectory   *string              `jsonapi:"attr,working-directory,omitempty"`
	TriggerPatterns    []string             `jsonapi:"attr,trigger-patterns"`
}

// WaitForStatusResult is the data structure that is sent over the channel
// returned by various status polling functions. For each result, either the
// Error or the Status will be set, but not both. If the Quit field is set,
// the channel will be closed. If the Quit field is set and the Error is
// nil, the Status field will be set to a specified quit status.
type WaitForStatusResult struct {
	ID           string
	Status       string
	ReadAttempts int
	Error        error
	Quit         bool
}

const minimumPollingIntervalMs = 3000
const maximumPollingIntervalMs = 5000

// FetchLatestFromVcs fetches the latest configuration of a stack from VCS, triggering stack operations
func (s *stacks) FetchLatestFromVcs(ctx context.Context, stackID string) (*Stack, error) {
	req, err := s.client.NewRequest("POST", fmt.Sprintf("stacks/%s/fetch-latest-from-vcs", url.PathEscape(stackID)), nil)
	if err != nil {
		return nil, err
	}

	stack := &Stack{}
	err = req.Do(ctx, stack)
	if err != nil {
		return nil, err
	}

	return stack, nil
}

// List returns a list of stacks, optionally filtered by additional paameters.
func (s stacks) List(ctx context.Context, organization string, options *StackListOptions) (*StackList, error) {
	if err := options.valid(); err != nil {
		return nil, err
	}

	req, err := s.client.NewRequest("GET", fmt.Sprintf("organizations/%s/stacks", organization), options)
	if err != nil {
		return nil, err
	}

	sl := &StackList{}
	err = req.Do(ctx, sl)
	if err != nil {
		return nil, err
	}

	return sl, nil
}

// Read returns a stack by its ID.
func (s stacks) Read(ctx context.Context, stackID string) (*Stack, error) {
	req, err := s.client.NewRequest("GET", fmt.Sprintf("stacks/%s", url.PathEscape(stackID)), nil)
	if err != nil {
		return nil, err
	}

	stack := &Stack{}
	err = req.Do(ctx, stack)
	if err != nil {
		return nil, err
	}

	return stack, nil
}

// Create creates a new stack.
func (s stacks) Create(ctx context.Context, options StackCreateOptions) (*Stack, error) {
	if err := options.valid(); err != nil {
		return nil, err
	}

	req, err := s.client.NewRequest("POST", "stacks", &options)
	if err != nil {
		return nil, err
	}

	stack := &Stack{}
	err = req.Do(ctx, stack)
	if err != nil {
		return nil, err
	}

	return stack, nil
}

// Update updates a stack.
func (s stacks) Update(ctx context.Context, stackID string, options StackUpdateOptions) (*Stack, error) {
	req, err := s.client.NewRequest("PATCH", fmt.Sprintf("stacks/%s", url.PathEscape(stackID)), &options)
	if err != nil {
		return nil, err
	}

	stack := &Stack{}
	err = req.Do(ctx, stack)
	if err != nil {
		return nil, err
	}

	return stack, nil
}

// Delete deletes a stack.
func (s stacks) Delete(ctx context.Context, stackID string) error {
	req, err := s.client.NewRequest("DELETE", fmt.Sprintf("stacks/%s", url.PathEscape(stackID)), nil)
	if err != nil {
		return err
	}

	return req.Do(ctx, nil)
}

// ForceDelete deletes a stack that still has deployments.
func (s stacks) ForceDelete(ctx context.Context, stackID string) error {
	req, err := s.client.NewRequest("DELETE", fmt.Sprintf("stacks/%s?force=true", url.PathEscape(stackID)), nil)
	if err != nil {
		return err
	}

	return req.Do(ctx, nil)
}

func (s *StackListOptions) valid() error {
	return nil
}

func (s StackCreateOptions) valid() error {
	if s.Name == "" {
		return ErrRequiredName
	}

	if s.Project == nil || s.Project.ID == "" {
		return ErrRequiredProject
	}

	return nil
}

// awaitPoll is a helper function that uses a callback to read a status, then
// waits for a terminal status or an error. The callback should return the
// current status, or an error. For each time the status changes, the channel
// emits a new result. The id parameter should be the ID of the resource being
// polled, which is used in the result to help identify the resource being polled.
func awaitPoll(ctx context.Context, id string, reader func(ctx context.Context) (string, error), quitStatus []string) <-chan WaitForStatusResult {
	resultCh := make(chan WaitForStatusResult)

	mapStatus := make(map[string]struct{}, len(quitStatus))
	for _, status := range quitStatus {
		mapStatus[status] = struct{}{}
	}

	go func() {
		defer close(resultCh)

		reads := 0
		lastStatus := ""
		for {
			select {
			case <-ctx.Done():
				resultCh <- WaitForStatusResult{ID: id, Error: fmt.Errorf("context canceled: %w", ctx.Err())}
				return
			case <-time.After(backoff(minimumPollingIntervalMs, maximumPollingIntervalMs, reads)):
				status, err := reader(ctx)
				if err != nil {
					resultCh <- WaitForStatusResult{ID: id, Error: err, Quit: true}
					return
				}

				_, terminal := mapStatus[status]

				if status != lastStatus {
					resultCh <- WaitForStatusResult{
						ID:           id,
						Status:       status,
						ReadAttempts: reads + 1,
						Quit:         terminal,
					}
				}

				lastStatus = status

				if terminal {
					return
				}

				reads += 1
			}
		}
	}()

	return resultCh
}

// Compile-time proof of interface implementation.
var _ StateVersionOutputs = (*stateVersionOutputs)(nil)

// State version outputs are the output values from a Terraform state file.
// They include the name and value of the output, as well as a sensitive boolean
// if the value should be hidden by default in UIs.
//
// TFE API docs: https://developer.hashicorp.com/terraform/cloud-docs/api-docs/state-version-outputs
type StateVersionOutputs interface {
	Read(ctx context.Context, outputID string) (*StateVersionOutput, error)
	ReadCurrent(ctx context.Context, workspaceID string) (*StateVersionOutputsList, error)
}

// stateVersionOutputs implements StateVersionOutputs.
type stateVersionOutputs struct {
	client *Client
}

// StateVersionOutput represents a State Version Outputs
type StateVersionOutput struct {
	ID        string      `jsonapi:"primary,state-version-outputs"`
	Name      string      `jsonapi:"attr,name"`
	Sensitive bool        `jsonapi:"attr,sensitive"`
	Type      string      `jsonapi:"attr,type"`
	Value     interface{} `jsonapi:"attr,value"`
	// BETA: This field is experimental and not universally present in all versions of TFE/Terraform
	DetailedType interface{} `jsonapi:"attr,detailed-type"`
}

// ReadCurrent reads the current state version outputs for the specified workspace
func (s *stateVersionOutputs) ReadCurrent(ctx context.Context, workspaceID string) (*StateVersionOutputsList, error) {
	if !validStringID(&workspaceID) {
		return nil, ErrInvalidWorkspaceID
	}

	u := fmt.Sprintf("workspaces/%s/current-state-version-outputs", url.PathEscape(workspaceID))
	req, err := s.client.NewRequest("GET", u, nil)
	if err != nil {
		return nil, err
	}

	so := &StateVersionOutputsList{}
	err = req.Do(ctx, so)
	if err != nil {
		return nil, err
	}

	return so, nil
}

// Read a State Version Output
func (s *stateVersionOutputs) Read(ctx context.Context, outputID string) (*StateVersionOutput, error) {
	if !validStringID(&outputID) {
		return nil, ErrInvalidOutputID
	}

	u := fmt.Sprintf("state-version-outputs/%s", url.PathEscape(outputID))
	req, err := s.client.NewRequest("GET", u, nil)
	if err != nil {
		return nil, err
	}

	so := &StateVersionOutput{}
	err = req.Do(ctx, so)
	if err != nil {
		return nil, err
	}

	return so, nil
}

// Compile-time proof of interface implementation.
var _ StateVersions = (*stateVersions)(nil)

// StateVersionStatus are available state version status values
type StateVersionStatus string

// Available state version statuses.
const (
	StateVersionPending   StateVersionStatus = "pending"
	StateVersionFinalized StateVersionStatus = "finalized"
	StateVersionDiscarded StateVersionStatus = "discarded"
)

// StateVersions describes all the state version related methods that
// the Terraform Enterprise API supports.
//
// TFE API docs:
// https://developer.hashicorp.com/terraform/cloud-docs/api-docs/state-versions
type StateVersions interface {
	// List all the state versions for a given workspace.
	List(ctx context.Context, options *StateVersionListOptions) (*StateVersionList, error)

	// Create a new state version for the given workspace.
	Create(ctx context.Context, workspaceID string, options StateVersionCreateOptions) (*StateVersion, error)

	// Upload creates a new state version but uploads the state content directly to the object store.
	// This is a more resilient form of Create and is the recommended approach to creating state versions.
	Upload(ctx context.Context, workspaceID string, options StateVersionUploadOptions) (*StateVersion, error)

	// UploadSanitizedState uploads a sanitized version of the state to the provided sanitized state upload url.
	// The SanitizedStateUploadURL cannot be empty.
	UploadSanitizedState(ctx context.Context, sanitizedStateUploadURL *string, sanitizedState []byte) error

	// Read a state version by its ID.
	Read(ctx context.Context, svID string) (*StateVersion, error)

	// ReadWithOptions reads a state version by its ID using the options supplied
	ReadWithOptions(ctx context.Context, svID string, options *StateVersionReadOptions) (*StateVersion, error)

	// ReadCurrent reads the latest available state from the given workspace.
	ReadCurrent(ctx context.Context, workspaceID string) (*StateVersion, error)

	// ReadCurrentWithOptions reads the latest available state from the given workspace using the options supplied
	ReadCurrentWithOptions(ctx context.Context, workspaceID string, options *StateVersionCurrentOptions) (*StateVersion, error)

	// Download retrieves the actual stored state of a state version
	Download(ctx context.Context, url string) ([]byte, error)

	// ListOutputs retrieves all the outputs of a state version by its ID. IMPORTANT: HCP Terraform might
	// process outputs asynchronously. When consuming outputs or other async StateVersion fields, be sure to
	// wait for ResourcesProcessed to become `true` before assuming they are empty.
	ListOutputs(ctx context.Context, svID string, options *StateVersionOutputsListOptions) (*StateVersionOutputsList, error)

	// SoftDeleteBackingData soft deletes the state version's backing data
	// **Note: This functionality is only available in Terraform Enterprise.**
	SoftDeleteBackingData(ctx context.Context, svID string) error

	// RestoreBackingData restores a soft deleted state version's backing data
	// **Note: This functionality is only available in Terraform Enterprise.**
	RestoreBackingData(ctx context.Context, svID string) error

	// PermanentlyDeleteBackingData permanently deletes a soft deleted state version's backing data
	// **Note: This functionality is only available in Terraform Enterprise.**
	PermanentlyDeleteBackingData(ctx context.Context, svID string) error
}

// stateVersions implements StateVersions.
type stateVersions struct {
	client *Client
}

// StateVersionList represents a list of state versions.
type StateVersionList struct {
	*Pagination
	Items []*StateVersion
}

// StateVersion represents a Terraform Enterprise state version.
type StateVersion struct {
	ID                        string             `jsonapi:"primary,state-versions"`
	CreatedAt                 time.Time          `jsonapi:"attr,created-at,iso8601"`
	DownloadURL               string             `jsonapi:"attr,hosted-state-download-url"`
	UploadURL                 string             `jsonapi:"attr,hosted-state-upload-url"`
	Status                    StateVersionStatus `jsonapi:"attr,status"`
	JSONUploadURL             string             `jsonapi:"attr,hosted-json-state-upload-url"`
	JSONDownloadURL           string             `jsonapi:"attr,hosted-json-state-download-url"`
	Serial                    int64              `jsonapi:"attr,serial"`
	Size                      int64              `jsonapi:"attr,size"`
	VCSCommitSHA              string             `jsonapi:"attr,vcs-commit-sha"`
	VCSCommitURL              string             `jsonapi:"attr,vcs-commit-url"`
	BillableRUMCount          *uint32            `jsonapi:"attr,billable-rum-count"`
	EncryptedStateDownloadURL *string            `jsonapi:"attr,encrypted-state-download-url,omitempty"`
	SanitizedStateUploadURL   *string            `jsonapi:"attr,sanitized-state-upload-url,omitempty"`
	SanitizedStateDownloadURL *string            `jsonapi:"attr,sanitized-state-download-url,omitempty"`

	// Whether HCP Terraform has finished populating any StateVersion fields that required async processing.
	// If `false`, some fields may appear empty even if they should actually contain data; see comments on
	// individual fields for details.
	ResourcesProcessed bool `jsonapi:"attr,resources-processed"`
	StateVersion       int  `jsonapi:"attr,state-version"`
	// Populated asynchronously.
	TerraformVersion string `jsonapi:"attr,terraform-version"`
	// Populated asynchronously.
	Modules *StateVersionModules `jsonapi:"attr,modules"`
	// Populated asynchronously.
	Providers *StateVersionProviders `jsonapi:"attr,providers"`
	// Populated asynchronously.
	Resources []*StateVersionResources `jsonapi:"attr,resources"`

	// Relations
	Run                  *Run                  `jsonapi:"relation,run"`
	Outputs              []*StateVersionOutput `jsonapi:"relation,outputs"`
	HYOKEncryptedDataKey *HYOKEncryptedDataKey `jsonapi:"relation,hyok-encrypted-data-key,omitempty"`
}

// StateVersionOutputsList represents a list of StateVersionOutput items.
type StateVersionOutputsList struct {
	*Pagination
	Items []*StateVersionOutput
}

// StateVersionListOptions represents the options for listing state versions.
type StateVersionListOptions struct {
	ListOptions
	Organization string `url:"filter[organization][name]"`
	Workspace    string `url:"filter[workspace][name]"`
}

// StateVersionIncludeOpt represents the available options for include query params.
// https://developer.hashicorp.com/terraform/cloud-docs/api-docs/state-versions#available-related-resources
type StateVersionIncludeOpt string

const (
	SVcreatedby               StateVersionIncludeOpt = "created_by"
	SVrun                     StateVersionIncludeOpt = "run"
	SVrunCreatedBy            StateVersionIncludeOpt = "run.created_by"
	SVrunConfigurationVersion StateVersionIncludeOpt = "run.configuration_version"
	SVoutputs                 StateVersionIncludeOpt = "outputs"
)

// StateVersionReadOptions represents the options for reading state version.
type StateVersionReadOptions struct {
	// Optional: A list of relations to include. See available resources:
	// https://developer.hashicorp.com/terraform/cloud-docs/api-docs/state-versions#available-related-resources
	Include []StateVersionIncludeOpt `url:"include,omitempty"`
}

// StateVersionOutputsListOptions represents the options for listing state
// version outputs.
type StateVersionOutputsListOptions struct {
	ListOptions
}

// StateVersionCurrentOptions represents the options for reading the current state version.
type StateVersionCurrentOptions struct {
	// Optional: A list of relations to include. See available resources:
	// https://developer.hashicorp.com/terraform/cloud-docs/api-docs/state-versions#available-related-resources
	Include []StateVersionIncludeOpt `url:"include,omitempty"`
}

// StateVersionCreateOptions represents the options for creating a state version.
type StateVersionCreateOptions struct {
	// Type is a public field utilized by JSON:API to
	// set the resource type via the field tag.
	// It is not a user-defined value and does not need to be set.
	// https://jsonapi.org/format/#crud-creating
	Type string `jsonapi:"primary,state-versions"`

	// Optional: The lineage of the state.
	Lineage *string `jsonapi:"attr,lineage,omitempty"`

	// Required: The MD5 hash of the state version.
	MD5 *string `jsonapi:"attr,md5"`

	// Required: The serial of the state.
	Serial *int64 `jsonapi:"attr,serial"`

	// Optional: The base64 encoded state.
	State *string `jsonapi:"attr,state,omitempty"`

	// Optional: Force can be set to skip certain validations. Wrong use
	// of this flag can cause data loss, so USE WITH CAUTION!
	Force *bool `jsonapi:"attr,force,omitempty"`

	// Optional: Specifies the run to associate the state with.
	Run *Run `jsonapi:"relation,run,omitempty"`

	// Optional: The external, json representation of state data, base64 encoded.
	// https://developer.hashicorp.com/terraform/internals/json-format#state-representation
	// Supplying this state representation can provide more details to the platform
	// about the current terraform state.
	JSONState *string `jsonapi:"attr,json-state,omitempty"`
	// Optional: The external, json representation of state outputs, base64 encoded. Supplying this field
	// will provide more detailed output type information to TFE.
	// For more information on the contents of this field: https://developer.hashicorp.com/terraform/internals/json-format#values-representation
	// about the current terraform state.
	JSONStateOutputs *string `jsonapi:"attr,json-state-outputs,omitempty"`
}

type StateVersionUploadOptions struct {
	StateVersionCreateOptions

	RawState     []byte
	RawJSONState []byte
}

type StateVersionModules struct {
	Root StateVersionModuleRoot `jsonapi:"attr,root"`
}

type StateVersionModuleRoot struct {
	NullResource         int `jsonapi:"attr,null-resource"`
	TerraformRemoteState int `jsonapi:"attr,data.terraform-remote-state"`
}

type StateVersionProviders struct {
	Data ProviderData `jsonapi:"attr,provider[map]string"`
}

type ProviderData struct {
	NullResource         int `json:"null-resource"`
	TerraformRemoteState int `json:"data.terraform-remote-state"`
}

type StateVersionResources struct {
	Name     string `jsonapi:"attr,name"`
	Count    int    `jsonapi:"attr,count"`
	Type     string `jsonapi:"attr,type"`
	Module   string `jsonapi:"attr,module"`
	Provider string `jsonapi:"attr,provider"`
}

// List all the state versions for a given workspace.
func (s *stateVersions) List(ctx context.Context, options *StateVersionListOptions) (*StateVersionList, error) {
	if err := options.valid(); err != nil {
		return nil, err
	}

	req, err := s.client.NewRequest("GET", "state-versions", options)
	if err != nil {
		return nil, err
	}

	svl := &StateVersionList{}
	err = req.Do(ctx, svl)
	if err != nil {
		return nil, err
	}

	return svl, nil
}

// Create a new state version for the given workspace.
func (s *stateVersions) Create(ctx context.Context, workspaceID string, options StateVersionCreateOptions) (*StateVersion, error) {
	if !validStringID(&workspaceID) {
		return nil, ErrInvalidWorkspaceID
	}
	if err := options.valid(); err != nil {
		return nil, err
	}

	u := fmt.Sprintf("workspaces/%s/state-versions", url.PathEscape(workspaceID))
	req, err := s.client.NewRequest("POST", u, &options)
	if err != nil {
		return nil, err
	}

	sv := &StateVersion{}
	err = req.Do(ctx, sv)
	if err != nil {
		return nil, err
	}

	return sv, nil
}

// Upload creates a new state version but uploads the state content directly to the object store.
// This is a more resilient form of Create and is the recommended approach to creating state versions.
func (s *stateVersions) Upload(ctx context.Context, workspaceID string, options StateVersionUploadOptions) (*StateVersion, error) {
	if err := options.valid(); err != nil {
		return nil, err
	}

	sv, err := s.Create(ctx, workspaceID, options.StateVersionCreateOptions)
	if err != nil {
		if strings.Contains(err.Error(), "param is missing or the value is empty: state") {
			return nil, ErrStateVersionUploadNotSupported
		}
		return nil, err
	}

	g, _ := errgroup.WithContext(ctx)
	g.Go(func() error {
		return s.client.doForeignPUTRequest(ctx, sv.UploadURL, bytes.NewReader(options.RawState))
	})
	if options.RawJSONState != nil {
		g.Go(func() error {
			return s.client.doForeignPUTRequest(ctx, sv.JSONUploadURL, bytes.NewReader(options.RawJSONState))
		})
	}

	if err := g.Wait(); err != nil {
		return nil, err
	}

	// Re-read the state version to get the updated status, if available
	return s.Read(ctx, sv.ID)
}

// UploadSanitizedState uploads a sanitized version of the state to the provided sanitized state upload url.
// The SanitizedStateUploadURL cannot be empty.
func (s *stateVersions) UploadSanitizedState(ctx context.Context, sanitizedStateUploadURL *string, sanitizedState []byte) error {
	if sanitizedStateUploadURL == nil {
		return ErrSanitizedStateUploadURLMissing
	}

	return s.client.doForeignPUTRequest(ctx, *sanitizedStateUploadURL, bytes.NewReader(sanitizedState))
}

// Read a state version by its ID.
func (s *stateVersions) ReadWithOptions(ctx context.Context, svID string, options *StateVersionReadOptions) (*StateVersion, error) {
	if !validStringID(&svID) {
		return nil, ErrInvalidStateVerID
	}
	if err := options.valid(); err != nil {
		return nil, err
	}

	u := fmt.Sprintf("state-versions/%s", url.PathEscape(svID))
	req, err := s.client.NewRequest("GET", u, options)
	if err != nil {
		return nil, err
	}

	sv := &StateVersion{}
	err = req.Do(ctx, sv)
	if err != nil {
		return nil, err
	}

	return sv, nil
}

// Read a state version by its ID.
func (s *stateVersions) Read(ctx context.Context, svID string) (*StateVersion, error) {
	return s.ReadWithOptions(ctx, svID, nil)
}

// ReadCurrentWithOptions reads the latest available state from the given workspace using the options supplied.
func (s *stateVersions) ReadCurrentWithOptions(ctx context.Context, workspaceID string, options *StateVersionCurrentOptions) (*StateVersion, error) {
	if !validStringID(&workspaceID) {
		return nil, ErrInvalidWorkspaceID
	}
	if err := options.valid(); err != nil {
		return nil, err
	}

	u := fmt.Sprintf("workspaces/%s/current-state-version", url.PathEscape(workspaceID))
	req, err := s.client.NewRequest("GET", u, options)
	if err != nil {
		return nil, err
	}

	sv := &StateVersion{}
	err = req.Do(ctx, sv)
	if err != nil {
		return nil, err
	}

	return sv, nil
}

// ReadCurrent reads the latest available state from the given workspace.
func (s *stateVersions) ReadCurrent(ctx context.Context, workspaceID string) (*StateVersion, error) {
	return s.ReadCurrentWithOptions(ctx, workspaceID, nil)
}

// Download retrieves the actual stored state of a state version
func (s *stateVersions) Download(ctx context.Context, u string) ([]byte, error) {
	req, err := s.client.NewRequest("GET", u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")

	var buf bytes.Buffer
	err = req.Do(ctx, &buf)
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// ListOutputs retrieves all the outputs of a state version by its ID. IMPORTANT: HCP Terraform might
// process outputs asynchronously. When consuming outputs or other async StateVersion fields, be sure to
// wait for ResourcesProcessed to become `true` before assuming they are empty.
func (s *stateVersions) ListOutputs(ctx context.Context, svID string, options *StateVersionOutputsListOptions) (*StateVersionOutputsList, error) {
	if !validStringID(&svID) {
		return nil, ErrInvalidStateVerID
	}

	u := fmt.Sprintf("state-versions/%s/outputs", url.PathEscape(svID))
	req, err := s.client.NewRequest("GET", u, options)
	if err != nil {
		return nil, err
	}

	sv := &StateVersionOutputsList{}
	err = req.Do(ctx, sv)
	if err != nil {
		return nil, err
	}

	return sv, nil
}

func (s *stateVersions) SoftDeleteBackingData(ctx context.Context, svID string) error {
	return s.manageBackingData(ctx, svID, "soft_delete_backing_data")
}

func (s *stateVersions) RestoreBackingData(ctx context.Context, svID string) error {
	return s.manageBackingData(ctx, svID, "restore_backing_data")
}

func (s *stateVersions) PermanentlyDeleteBackingData(ctx context.Context, svID string) error {
	return s.manageBackingData(ctx, svID, "permanently_delete_backing_data")
}

func (s *stateVersions) manageBackingData(ctx context.Context, svID, action string) error {
	if !validStringID(&svID) {
		return ErrInvalidStateVerID
	}

	u := fmt.Sprintf("state-versions/%s/actions/%s", svID, action)
	req, err := s.client.NewRequest("POST", u, nil)
	if err != nil {
		return err
	}

	return req.Do(ctx, nil)
}

// check that StateVersionListOptions fields had valid values
func (o *StateVersionListOptions) valid() error {
	if o == nil {
		return ErrRequiredStateVerListOps
	}
	if !validString(&o.Organization) {
		return ErrRequiredOrg
	}
	if !validString(&o.Workspace) {
		return ErrRequiredWorkspace
	}
	return nil
}

func (o StateVersionCreateOptions) valid() error {
	if !validString(o.MD5) {
		return ErrRequiredM5
	}
	if o.Serial == nil {
		return ErrRequiredSerial
	}
	return nil
}

func (o StateVersionUploadOptions) valid() error {
	if err := o.StateVersionCreateOptions.valid(); err != nil {
		return err
	}
	if o.State != nil || o.JSONState != nil {
		return ErrStateMustBeOmitted
	}
	if o.RawState == nil {
		return ErrRequiredRawState
	}
	return nil
}

func (o *StateVersionReadOptions) valid() error {
	return nil
}
func (o *StateVersionCurrentOptions) valid() error {
	return nil
}

type TagList struct {
	*Pagination
	Items []*Tag
}

// Tag is owned by an organization and applied to workspaces. Used for grouping and search.
type Tag struct {
	ID   string `jsonapi:"primary,tags"`
	Name string `jsonapi:"attr,name,omitempty"`
}

type TagBinding struct {
	ID    string `jsonapi:"primary,tag-bindings"`
	Key   string `jsonapi:"attr,key"`
	Value string `jsonapi:"attr,value,omitempty"`
}

type EffectiveTagBinding struct {
	ID    string                 `jsonapi:"primary,effective-tag-bindings"`
	Key   string                 `jsonapi:"attr,key"`
	Value string                 `jsonapi:"attr,value,omitempty"`
	Links map[string]interface{} `jsonapi:"links,omitempty"`
}

func encodeTagFiltersAsParams(filters []*TagBinding) map[string][]string {
	if len(filters) == 0 {
		return nil
	}

	var tagFilter = make(map[string][]string, len(filters))
	for index, tag := range filters {
		tagFilter[fmt.Sprintf("filter[tagged][%d][key]", index)] = []string{tag.Key}
		tagFilter[fmt.Sprintf("filter[tagged][%d][value]", index)] = []string{tag.Value}
	}

	return tagFilter
}

// Compile-time proof of interface implementation
var _ TaskResults = (*taskResults)(nil)

// TaskResults describes all the task result related methods that the HCP Terraform or Terraform Enterprise API supports.
type TaskResults interface {
	// Read a task result by ID
	Read(ctx context.Context, taskResultID string) (*TaskResult, error)
}

// taskResults implements TaskResults
type taskResults struct {
	client *Client
}

// TaskResultStatus is an enum that represents all possible statuses for a task result
type TaskResultStatus string

const (
	TaskPassed      TaskResultStatus = "passed"
	TaskFailed      TaskResultStatus = "failed"
	TaskPending     TaskResultStatus = "pending"
	TaskRunning     TaskResultStatus = "running"
	TaskUnreachable TaskResultStatus = "unreachable"
	TaskErrored     TaskResultStatus = "errored"
)

// TaskEnforcementLevel is an enum that describes the enforcement levels for a run task
type TaskEnforcementLevel string

const (
	Advisory  TaskEnforcementLevel = "advisory"
	Mandatory TaskEnforcementLevel = "mandatory"
)

// TaskResultStatusTimestamps represents the set of timestamps recorded for a task result
type TaskResultStatusTimestamps struct {
	ErroredAt  time.Time `jsonapi:"attr,errored-at,rfc3339"`
	RunningAt  time.Time `jsonapi:"attr,running-at,rfc3339"`
	CanceledAt time.Time `jsonapi:"attr,canceled-at,rfc3339"`
	FailedAt   time.Time `jsonapi:"attr,failed-at,rfc3339"`
	PassedAt   time.Time `jsonapi:"attr,passed-at,rfc3339"`
}

// TaskResult represents the result of a HCP Terraform or Terraform Enterprise run task
type TaskResult struct {
	ID                            string                     `jsonapi:"primary,task-results"`
	Status                        TaskResultStatus           `jsonapi:"attr,status"`
	Message                       string                     `jsonapi:"attr,message"`
	StatusTimestamps              TaskResultStatusTimestamps `jsonapi:"attr,status-timestamps"`
	URL                           string                     `jsonapi:"attr,url"`
	CreatedAt                     time.Time                  `jsonapi:"attr,created-at,iso8601"`
	UpdatedAt                     time.Time                  `jsonapi:"attr,updated-at,iso8601"`
	TaskID                        string                     `jsonapi:"attr,task-id"`
	TaskName                      string                     `jsonapi:"attr,task-name"`
	TaskURL                       string                     `jsonapi:"attr,task-url"`
	WorkspaceTaskID               string                     `jsonapi:"attr,workspace-task-id"`
	WorkspaceTaskEnforcementLevel TaskEnforcementLevel       `jsonapi:"attr,workspace-task-enforcement-level"`
	AgentPoolID                   *string                    `jsonapi:"attr,agent-pool-id,omitempty"`

	// The task stage this result belongs to
	TaskStage *TaskStage `jsonapi:"relation,task_stage"`
}

// Read a task result by ID
func (t *taskResults) Read(ctx context.Context, taskResultID string) (*TaskResult, error) {
	if !validStringID(&taskResultID) {
		return nil, ErrInvalidTaskResultID
	}

	u := fmt.Sprintf("task-results/%s", taskResultID)
	req, err := t.client.NewRequest("GET", u, nil)
	if err != nil {
		return nil, err
	}

	r := &TaskResult{}
	err = req.Do(ctx, r)
	if err != nil {
		return nil, err
	}

	return r, nil
}

// Compile-time proof of interface  implementation
var _ TaskStages = (*taskStages)(nil)

// TaskStages describes all the task stage related methods that the HCP Terraform and Terraform Enterprise API
// supports.
type TaskStages interface {
	// Read a task stage by ID
	Read(ctx context.Context, taskStageID string, options *TaskStageReadOptions) (*TaskStage, error)

	// List all task stages for a given run
	List(ctx context.Context, runID string, options *TaskStageListOptions) (*TaskStageList, error)

	// **Note: This function is still in BETA and subject to change.**
	// Override a task stage for a given run
	Override(ctx context.Context, taskStageID string, options TaskStageOverrideOptions) (*TaskStage, error)
}

// taskStages implements TaskStages
type taskStages struct {
	client *Client
}

// Stage is an enum that represents the possible run stages for run tasks
type Stage string

const (
	PrePlan   Stage = "pre_plan"
	PostPlan  Stage = "post_plan"
	PreApply  Stage = "pre_apply"
	PostApply Stage = "post_apply"
)

// TaskStageStatus is an enum that represents all possible statuses for a task stage
type TaskStageStatus string

const (
	TaskStagePending          TaskStageStatus = "pending"
	TaskStageRunning          TaskStageStatus = "running"
	TaskStagePassed           TaskStageStatus = "passed"
	TaskStageFailed           TaskStageStatus = "failed"
	TaskStageAwaitingOverride TaskStageStatus = "awaiting_override"
	TaskStageCanceled         TaskStageStatus = "canceled"
	TaskStageErrored          TaskStageStatus = "errored"
	TaskStageUnreachable      TaskStageStatus = "unreachable"
)

// Permissions represents the permission types for overridding a task stage
type Permissions struct {
	CanOverridePolicy *bool `jsonapi:"attr,can-override-policy"`
	CanOverrideTasks  *bool `jsonapi:"attr,can-override-tasks"`
	CanOverride       *bool `jsonapi:"attr,can-override"`
}

// Actions represents a task stage actions
type Actions struct {
	IsOverridable *bool `jsonapi:"attr,is-overridable"`
}

// TaskStage represents a HCP Terraform or Terraform Enterprise run's stage where run tasks can occur
type TaskStage struct {
	ID               string                    `jsonapi:"primary,task-stages"`
	Stage            Stage                     `jsonapi:"attr,stage"`
	Status           TaskStageStatus           `jsonapi:"attr,status"`
	StatusTimestamps TaskStageStatusTimestamps `jsonapi:"attr,status-timestamps"`
	CreatedAt        time.Time                 `jsonapi:"attr,created-at,iso8601"`
	UpdatedAt        time.Time                 `jsonapi:"attr,updated-at,iso8601"`
	Permissions      *Permissions              `jsonapi:"attr,permissions"`
	Actions          *Actions                  `jsonapi:"attr,actions"`

	Run               *Run                `jsonapi:"relation,run"`
	TaskResults       []*TaskResult       `jsonapi:"relation,task-results"`
	PolicyEvaluations []*PolicyEvaluation `jsonapi:"relation,policy-evaluations"`
}

// TaskStageOverrideOptions represents the options for overriding a TaskStage.
type TaskStageOverrideOptions struct {
	// An optional explanation for why the stage was overridden
	Comment *string `json:"comment,omitempty"`
}

// TaskStageList represents a list of task stages
type TaskStageList struct {
	*Pagination
	Items []*TaskStage
}

// TaskStageStatusTimestamps represents the set of timestamps recorded for a task stage
type TaskStageStatusTimestamps struct {
	ErroredAt  time.Time `jsonapi:"attr,errored-at,rfc3339"`
	RunningAt  time.Time `jsonapi:"attr,running-at,rfc3339"`
	CanceledAt time.Time `jsonapi:"attr,canceled-at,rfc3339"`
	FailedAt   time.Time `jsonapi:"attr,failed-at,rfc3339"`
	PassedAt   time.Time `jsonapi:"attr,passed-at,rfc3339"`
}

// TaskStageIncludeOpt represents the available options for include query params.
type TaskStageIncludeOpt string

const TaskStageTaskResults TaskStageIncludeOpt = "task_results"

// **Note: This field is still in BETA and subject to change.**
const PolicyEvaluationsTaskResults TaskStageIncludeOpt = "policy_evaluations"

// TaskStageReadOptions represents the set of options when reading a task stage
type TaskStageReadOptions struct {
	// Optional: A list of relations to include.
	Include []TaskStageIncludeOpt `url:"include,omitempty"`
}

// TaskStageListOptions represents the options for listing task stages for a run
type TaskStageListOptions struct {
	ListOptions
}

// Read a task stage by ID
func (s *taskStages) Read(ctx context.Context, taskStageID string, options *TaskStageReadOptions) (*TaskStage, error) {
	if !validStringID(&taskStageID) {
		return nil, ErrInvalidTaskStageID
	}
	if err := options.valid(); err != nil {
		return nil, err
	}

	u := fmt.Sprintf("task-stages/%s", taskStageID)
	req, err := s.client.NewRequest("GET", u, options)
	if err != nil {
		return nil, err
	}

	t := &TaskStage{}
	err = req.Do(ctx, t)
	if err != nil {
		return nil, err
	}

	return t, nil
}

// List task stages for a run
func (s *taskStages) List(ctx context.Context, runID string, options *TaskStageListOptions) (*TaskStageList, error) {
	if !validStringID(&runID) {
		return nil, ErrInvalidRunID
	}

	u := fmt.Sprintf("runs/%s/task-stages", runID)
	req, err := s.client.NewRequest("GET", u, options)
	if err != nil {
		return nil, err
	}

	tlist := &TaskStageList{}

	err = req.Do(ctx, tlist)
	if err != nil {
		return nil, err
	}

	return tlist, nil
}

// **Note: This function is still in BETA and subject to change.**
// Override a task stages for a run
func (s *taskStages) Override(ctx context.Context, taskStageID string, options TaskStageOverrideOptions) (*TaskStage, error) {
	if !validStringID(&taskStageID) {
		return nil, ErrInvalidTaskStageID
	}

	u := fmt.Sprintf("task-stages/%s/actions/override", taskStageID)
	req, err := s.client.NewRequest("POST", u, &options)
	if err != nil {
		return nil, err
	}

	t := &TaskStage{}
	err = req.Do(ctx, t)
	if err != nil {
		return nil, err
	}

	return t, nil
}

func (o *TaskStageReadOptions) valid() error {
	return nil
}

// Compile-time proof of interface implementation.
var _ TeamAccesses = (*teamAccesses)(nil)

// TeamAccesses describes all the team access related methods that the
// Terraform Enterprise API supports.
//
// TFE API docs:
// https://developer.hashicorp.com/terraform/cloud-docs/api-docs/team-access
type TeamAccesses interface {
	// List all the team accesses for a given workspace.
	List(ctx context.Context, options *TeamAccessListOptions) (*TeamAccessList, error)

	// Add team access for a workspace.
	Add(ctx context.Context, options TeamAccessAddOptions) (*TeamAccess, error)

	// Read a team access by its ID.
	Read(ctx context.Context, teamAccessID string) (*TeamAccess, error)

	// Update a team access by its ID.
	Update(ctx context.Context, teamAccessID string, options TeamAccessUpdateOptions) (*TeamAccess, error)

	// Remove team access from a workspace.
	Remove(ctx context.Context, teamAccessID string) error
}

// teamAccesses implements TeamAccesses.
type teamAccesses struct {
	client *Client
}

// AccessType represents a team access type.
type AccessType string

const (
	AccessAdmin  AccessType = "admin"
	AccessPlan   AccessType = "plan"
	AccessRead   AccessType = "read"
	AccessWrite  AccessType = "write"
	AccessCustom AccessType = "custom"
)

// RunsPermissionType represents the permissiontype to a workspace's runs.
type RunsPermissionType string

const (
	RunsPermissionRead  RunsPermissionType = "read"
	RunsPermissionPlan  RunsPermissionType = "plan"
	RunsPermissionApply RunsPermissionType = "apply"
)

// VariablesPermissionType represents the permissiontype to a workspace's variables.
type VariablesPermissionType string

const (
	VariablesPermissionNone  VariablesPermissionType = "none"
	VariablesPermissionRead  VariablesPermissionType = "read"
	VariablesPermissionWrite VariablesPermissionType = "write"
)

// StateVersionsPermissionType represents the permissiontype to a workspace's state versions.
type StateVersionsPermissionType string

const (
	StateVersionsPermissionNone        StateVersionsPermissionType = "none"
	StateVersionsPermissionReadOutputs StateVersionsPermissionType = "read-outputs"
	StateVersionsPermissionRead        StateVersionsPermissionType = "read"
	StateVersionsPermissionWrite       StateVersionsPermissionType = "write"
)

// SentinelMocksPermissionType represents the permissiontype to a workspace's Sentinel mocks.
type SentinelMocksPermissionType string

const (
	SentinelMocksPermissionNone SentinelMocksPermissionType = "none"
	SentinelMocksPermissionRead SentinelMocksPermissionType = "read"
)

// TeamAccessList represents a list of team accesses.
type TeamAccessList struct {
	*Pagination
	Items []*TeamAccess
}

// TeamAccess represents the workspace access for a team.
type TeamAccess struct {
	ID               string                      `jsonapi:"primary,team-workspaces"`
	Access           AccessType                  `jsonapi:"attr,access"`
	Runs             RunsPermissionType          `jsonapi:"attr,runs"`
	Variables        VariablesPermissionType     `jsonapi:"attr,variables"`
	StateVersions    StateVersionsPermissionType `jsonapi:"attr,state-versions"`
	SentinelMocks    SentinelMocksPermissionType `jsonapi:"attr,sentinel-mocks"`
	WorkspaceLocking bool                        `jsonapi:"attr,workspace-locking"`
	RunTasks         bool                        `jsonapi:"attr,run-tasks"`
	// **Note: This API is still in BETA and subject to change.**
	PolicyOverrides bool `jsonapi:"attr,policy-overrides"`

	// Relations
	Team      *Team      `jsonapi:"relation,team"`
	Workspace *Workspace `jsonapi:"relation,workspace"`
}

// TeamAccessListOptions represents the options for listing team accesses.
type TeamAccessListOptions struct {
	ListOptions
	WorkspaceID string `url:"filter[workspace][id]"`
}

// TeamAccessAddOptions represents the options for adding team access.
type TeamAccessAddOptions struct {
	// Type is a public field utilized by JSON:API to
	// set the resource type via the field tag.
	// It is not a user-defined value and does not need to be set.
	// https://jsonapi.org/format/#crud-creating
	Type string `jsonapi:"primary,team-workspaces"`

	// The type of access to grant.
	Access *AccessType `jsonapi:"attr,access"`

	// Custom workspace access permissions. These can only be edited when Access is 'custom'; otherwise, they are
	// read-only and reflect the Access level's implicit permissions.
	Runs             *RunsPermissionType          `jsonapi:"attr,runs,omitempty"`
	Variables        *VariablesPermissionType     `jsonapi:"attr,variables,omitempty"`
	StateVersions    *StateVersionsPermissionType `jsonapi:"attr,state-versions,omitempty"`
	SentinelMocks    *SentinelMocksPermissionType `jsonapi:"attr,sentinel-mocks,omitempty"`
	WorkspaceLocking *bool                        `jsonapi:"attr,workspace-locking,omitempty"`
	RunTasks         *bool                        `jsonapi:"attr,run-tasks,omitempty"`
	// **Note: This API is still in BETA and subject to change.**
	PolicyOverrides *bool `jsonapi:"attr,policy-overrides,omitempty"`

	// The team to add to the workspace
	Team *Team `jsonapi:"relation,team"`

	// The workspace to which the team is to be added.
	Workspace *Workspace `jsonapi:"relation,workspace"`
}

// TeamAccessUpdateOptions represents the options for updating team access.
type TeamAccessUpdateOptions struct {
	// Type is a public field utilized by JSON:API to
	// set the resource type via the field tag.
	// It is not a user-defined value and does not need to be set.
	// https://jsonapi.org/format/#crud-creating
	Type string `jsonapi:"primary,team-workspaces"`

	// The type of access to grant.
	Access *AccessType `jsonapi:"attr,access,omitempty"`

	// Custom workspace access permissions. These can only be edited when Access is 'custom'; otherwise, they are
	// read-only and reflect the Access level's implicit permissions.
	Runs             *RunsPermissionType          `jsonapi:"attr,runs,omitempty"`
	Variables        *VariablesPermissionType     `jsonapi:"attr,variables,omitempty"`
	StateVersions    *StateVersionsPermissionType `jsonapi:"attr,state-versions,omitempty"`
	SentinelMocks    *SentinelMocksPermissionType `jsonapi:"attr,sentinel-mocks,omitempty"`
	WorkspaceLocking *bool                        `jsonapi:"attr,workspace-locking,omitempty"`
	RunTasks         *bool                        `jsonapi:"attr,run-tasks,omitempty"`
	// **Note: This API is still in BETA and subject to change.**
	PolicyOverrides *bool `jsonapi:"attr,policy-overrides,omitempty"`
}

// List all the team accesses for a given workspace.
func (s *teamAccesses) List(ctx context.Context, options *TeamAccessListOptions) (*TeamAccessList, error) {
	if err := options.valid(); err != nil {
		return nil, err
	}

	req, err := s.client.NewRequest("GET", "team-workspaces", options)
	if err != nil {
		return nil, err
	}

	tal := &TeamAccessList{}
	err = req.Do(ctx, tal)
	if err != nil {
		return nil, err
	}

	return tal, nil
}

// Add team access for a workspace.
func (s *teamAccesses) Add(ctx context.Context, options TeamAccessAddOptions) (*TeamAccess, error) {
	if err := options.valid(); err != nil {
		return nil, err
	}

	req, err := s.client.NewRequest("POST", "team-workspaces", &options)
	if err != nil {
		return nil, err
	}

	ta := &TeamAccess{}
	err = req.Do(ctx, ta)
	if err != nil {
		return nil, err
	}

	return ta, nil
}

// Read a team access by its ID.
func (s *teamAccesses) Read(ctx context.Context, teamAccessID string) (*TeamAccess, error) {
	if !validStringID(&teamAccessID) {
		return nil, ErrInvalidAccessTeamID
	}

	u := fmt.Sprintf("team-workspaces/%s", url.PathEscape(teamAccessID))
	req, err := s.client.NewRequest("GET", u, nil)
	if err != nil {
		return nil, err
	}

	ta := &TeamAccess{}
	err = req.Do(ctx, ta)
	if err != nil {
		return nil, err
	}

	return ta, nil
}

// Update team access for a workspace
func (s *teamAccesses) Update(ctx context.Context, teamAccessID string, options TeamAccessUpdateOptions) (*TeamAccess, error) {
	if !validStringID(&teamAccessID) {
		return nil, ErrInvalidAccessTeamID
	}

	u := fmt.Sprintf("team-workspaces/%s", url.PathEscape(teamAccessID))
	req, err := s.client.NewRequest("PATCH", u, &options)
	if err != nil {
		return nil, err
	}

	ta := &TeamAccess{}
	err = req.Do(ctx, ta)
	if err != nil {
		return nil, err
	}

	return ta, err
}

// Remove team access from a workspace.
func (s *teamAccesses) Remove(ctx context.Context, teamAccessID string) error {
	if !validStringID(&teamAccessID) {
		return ErrInvalidAccessTeamID
	}

	u := fmt.Sprintf("team-workspaces/%s", url.PathEscape(teamAccessID))
	req, err := s.client.NewRequest("DELETE", u, nil)
	if err != nil {
		return err
	}

	return req.Do(ctx, nil)
}

func (o *TeamAccessListOptions) valid() error {
	if o == nil {
		return ErrRequiredTeamAccessListOps
	}
	if !validString(&o.WorkspaceID) {
		return ErrRequiredWorkspaceID
	}
	if !validStringID(&o.WorkspaceID) {
		return ErrInvalidWorkspaceID
	}

	return nil
}

func (o TeamAccessAddOptions) valid() error {
	if o.Access == nil {
		return ErrRequiredAccess
	}
	if o.Team == nil {
		return ErrRequiredTeam
	}
	if o.Workspace == nil {
		return ErrRequiredWorkspace
	}
	return nil
}

// Compile-time proof of interface implementation.
var _ TeamMembers = (*teamMembers)(nil)

// TeamMembers describes all the team member related methods that the
// Terraform Enterprise API supports.
//
// TFE API docs:
// https://developer.hashicorp.com/terraform/cloud-docs/api-docs/team-members
type TeamMembers interface {
	// List returns all Users of a team calling ListUsers
	// See ListOrganizationMemberships for fetching memberships
	List(ctx context.Context, teamID string) ([]*User, error)

	// ListUsers returns the Users of this team.
	ListUsers(ctx context.Context, teamID string) ([]*User, error)

	// ListOrganizationMemberships returns the OrganizationMemberships of this team.
	ListOrganizationMemberships(ctx context.Context, teamID string) ([]*OrganizationMembership, error)

	// Add multiple users to a team.
	Add(ctx context.Context, teamID string, options TeamMemberAddOptions) error

	// Remove multiple users from a team.
	Remove(ctx context.Context, teamID string, options TeamMemberRemoveOptions) error
}

// teamMembers implements TeamMembers.
type teamMembers struct {
	client *Client
}

type teamMemberUser struct {
	Username string `jsonapi:"primary,users"`
}

type teamMemberOrgMembership struct {
	ID string `jsonapi:"primary,organization-memberships"`
}

// TeamMemberAddOptions represents the options for
// adding or removing team members.
type TeamMemberAddOptions struct {
	Usernames                 []string
	OrganizationMembershipIDs []string
}

// TeamMemberRemoveOptions represents the options for
// adding or removing team members.
type TeamMemberRemoveOptions struct {
	Usernames                 []string
	OrganizationMembershipIDs []string
}

// List returns all Users of a team calling ListUsers
// See ListOrganizationMemberships for fetching memberships
func (s *teamMembers) List(ctx context.Context, teamID string) ([]*User, error) {
	return s.ListUsers(ctx, teamID)
}

// ListUsers returns the Users of this team.
func (s *teamMembers) ListUsers(ctx context.Context, teamID string) ([]*User, error) {
	if !validStringID(&teamID) {
		return nil, ErrInvalidTeamID
	}

	options := struct {
		Include []TeamIncludeOpt `url:"include,omitempty"`
	}{
		Include: []TeamIncludeOpt{TeamUsers},
	}

	u := fmt.Sprintf("teams/%s", url.PathEscape(teamID))
	req, err := s.client.NewRequest("GET", u, options)
	if err != nil {
		return nil, err
	}

	t := &Team{}
	err = req.Do(ctx, t)
	if err != nil {
		return nil, err
	}

	return t.Users, nil
}

// ListOrganizationMemberships returns the OrganizationMemberships of this team.
func (s *teamMembers) ListOrganizationMemberships(ctx context.Context, teamID string) ([]*OrganizationMembership, error) {
	if !validStringID(&teamID) {
		return nil, ErrInvalidTeamID
	}

	options := struct {
		Include []TeamIncludeOpt `url:"include,omitempty"`
	}{
		Include: []TeamIncludeOpt{TeamOrganizationMemberships},
	}

	u := fmt.Sprintf("teams/%s", url.PathEscape(teamID))
	req, err := s.client.NewRequest("GET", u, options)
	if err != nil {
		return nil, err
	}

	t := &Team{}
	err = req.Do(ctx, t)
	if err != nil {
		return nil, err
	}

	return t.OrganizationMemberships, nil
}

// Add multiple users to a team.
func (s *teamMembers) Add(ctx context.Context, teamID string, options TeamMemberAddOptions) error {
	if !validStringID(&teamID) {
		return ErrInvalidTeamID
	}
	if err := options.valid(); err != nil {
		return err
	}

	usersOrMemberships := options.kind()
	u := fmt.Sprintf("teams/%s/relationships/%s", url.PathEscape(teamID), usersOrMemberships)

	var req *ClientRequest

	if usersOrMemberships == "users" {
		var err error
		var members []*teamMemberUser
		for _, name := range options.Usernames {
			members = append(members, &teamMemberUser{Username: name})
		}
		req, err = s.client.NewRequest("POST", u, members)
		if err != nil {
			return err
		}
	} else {
		var err error
		var members []*teamMemberOrgMembership
		for _, ID := range options.OrganizationMembershipIDs {
			members = append(members, &teamMemberOrgMembership{ID: ID})
		}
		req, err = s.client.NewRequest("POST", u, members)
		if err != nil {
			return err
		}
	}

	return req.Do(ctx, nil)
}

// Remove multiple users from a team.
func (s *teamMembers) Remove(ctx context.Context, teamID string, options TeamMemberRemoveOptions) error {
	if !validStringID(&teamID) {
		return ErrInvalidTeamID
	}
	if err := options.valid(); err != nil {
		return err
	}

	usersOrMemberships := options.kind()
	u := fmt.Sprintf("teams/%s/relationships/%s", url.PathEscape(teamID), usersOrMemberships)

	var req *ClientRequest

	if usersOrMemberships == "users" {
		var err error
		var members []*teamMemberUser
		for _, name := range options.Usernames {
			members = append(members, &teamMemberUser{Username: name})
		}
		req, err = s.client.NewRequest("DELETE", u, members)
		if err != nil {
			return err
		}
	} else {
		var err error
		var members []*teamMemberOrgMembership
		for _, ID := range options.OrganizationMembershipIDs {
			members = append(members, &teamMemberOrgMembership{ID: ID})
		}
		req, err = s.client.NewRequest("DELETE", u, members)
		if err != nil {
			return err
		}
	}

	return req.Do(ctx, nil)
}

// kind returns "users" or "organization-memberships"
// depending on which is defined
func (o *TeamMemberAddOptions) kind() string {
	if len(o.Usernames) != 0 {
		return "users"
	}
	return "organization-memberships"
}

// kind returns "users" or "organization-memberships"
// depending on which is defined
func (o *TeamMemberRemoveOptions) kind() string {
	if len(o.Usernames) != 0 {
		return "users"
	}
	return "organization-memberships"
}

func (o *TeamMemberAddOptions) valid() error {
	if o.Usernames == nil && o.OrganizationMembershipIDs == nil {
		return ErrRequiredUsernameOrMembershipIds
	}
	if o.Usernames != nil && o.OrganizationMembershipIDs != nil {
		return ErrRequiredOnlyOneField
	}
	if o.Usernames != nil && len(o.Usernames) == 0 {
		return ErrInvalidUsernames
	}
	if o.OrganizationMembershipIDs != nil && len(o.OrganizationMembershipIDs) == 0 {
		return ErrInvalidMembershipIDs
	}
	return nil
}

func (o *TeamMemberRemoveOptions) valid() error {
	if o.Usernames == nil && o.OrganizationMembershipIDs == nil {
		return ErrRequiredUsernameOrMembershipIds
	}
	if o.Usernames != nil && o.OrganizationMembershipIDs != nil {
		return ErrRequiredOnlyOneField
	}
	if o.Usernames != nil && len(o.Usernames) == 0 {
		return ErrInvalidUsernames
	}
	if o.OrganizationMembershipIDs != nil && len(o.OrganizationMembershipIDs) == 0 {
		return ErrInvalidMembershipIDs
	}
	return nil
}

// Compile-time proof of interface implementation.
var _ TeamProjectAccesses = (*teamProjectAccesses)(nil)

// TeamProjectAccesses describes all the team project access related methods that the Terraform
// Enterprise API supports
//
// TFE API docs: https://developer.hashicorp.com/terraform/cloud-docs/api-docs/project-team-access
type TeamProjectAccesses interface {
	// List all project accesses for a given project.
	List(ctx context.Context, options TeamProjectAccessListOptions) (*TeamProjectAccessList, error)

	// Add team access for a project.
	Add(ctx context.Context, options TeamProjectAccessAddOptions) (*TeamProjectAccess, error)

	// Read team access by project ID.
	Read(ctx context.Context, teamProjectAccessID string) (*TeamProjectAccess, error)

	// Update team access on a project.
	Update(ctx context.Context, teamProjectAccessID string, options TeamProjectAccessUpdateOptions) (*TeamProjectAccess, error)

	// Remove team access from a project.
	Remove(ctx context.Context, teamProjectAccessID string) error
}

// teamProjectAccesses implements TeamProjectAccesses
type teamProjectAccesses struct {
	client *Client
}

// TeamProjectAccessType represents a team project access type.
type TeamProjectAccessType string

const (
	TeamProjectAccessAdmin    TeamProjectAccessType = "admin"
	TeamProjectAccessMaintain TeamProjectAccessType = "maintain"
	TeamProjectAccessWrite    TeamProjectAccessType = "write"
	TeamProjectAccessRead     TeamProjectAccessType = "read"
	TeamProjectAccessCustom   TeamProjectAccessType = "custom"
)

// TeamProjectAccessList represents a list of team project accesses
type TeamProjectAccessList struct {
	*Pagination
	Items []*TeamProjectAccess
}

// TeamProjectAccess represents a project access for a team
type TeamProjectAccess struct {
	ID              string                                 `jsonapi:"primary,team-projects"`
	Access          TeamProjectAccessType                  `jsonapi:"attr,access"`
	ProjectAccess   *TeamProjectAccessProjectPermissions   `jsonapi:"attr,project-access"`
	WorkspaceAccess *TeamProjectAccessWorkspacePermissions `jsonapi:"attr,workspace-access"`

	// Relations
	Team    *Team    `jsonapi:"relation,team"`
	Project *Project `jsonapi:"relation,project"`
}

// ProjectPermissions represents the team's permissions on its project
type TeamProjectAccessProjectPermissions struct {
	ProjectSettingsPermission ProjectSettingsPermissionType `jsonapi:"attr,settings"`
	ProjectTeamsPermission    ProjectTeamsPermissionType    `jsonapi:"attr,teams"`
	// ProjectVariableSetsPermission represents read, manage, and no access custom permission for project-level variable sets
	ProjectVariableSetsPermission ProjectVariableSetsPermissionType `jsonapi:"attr,variable-sets"`
}

// WorkspacePermissions represents the team's permission on all workspaces in its project
type TeamProjectAccessWorkspacePermissions struct {
	WorkspaceRunsPermission          WorkspaceRunsPermissionType          `jsonapi:"attr,runs"`
	WorkspaceSentinelMocksPermission WorkspaceSentinelMocksPermissionType `jsonapi:"attr,sentinel-mocks"`
	WorkspaceStateVersionsPermission WorkspaceStateVersionsPermissionType `jsonapi:"attr,state-versions"`
	WorkspaceVariablesPermission     WorkspaceVariablesPermissionType     `jsonapi:"attr,variables"`
	WorkspaceCreatePermission        bool                                 `jsonapi:"attr,create"`
	WorkspaceLockingPermission       bool                                 `jsonapi:"attr,locking"`
	WorkspaceMovePermission          bool                                 `jsonapi:"attr,move"`
	WorkspaceDeletePermission        bool                                 `jsonapi:"attr,delete"`
	WorkspaceRunTasksPermission      bool                                 `jsonapi:"attr,run-tasks"`
	// **Note: This API is still in BETA and subject to change.**
	WorkspacePolicyOverridesPermission bool `jsonapi:"attr,policy-overrides"`
}

// ProjectSettingsPermissionType represents the permissiontype to a project's settings
type ProjectSettingsPermissionType string

const (
	ProjectSettingsPermissionRead   ProjectSettingsPermissionType = "read"
	ProjectSettingsPermissionUpdate ProjectSettingsPermissionType = "update"
	ProjectSettingsPermissionDelete ProjectSettingsPermissionType = "delete"
)

// ProjectTeamsPermissionType represents the permissiontype to a project's teams
type ProjectTeamsPermissionType string

const (
	ProjectTeamsPermissionNone   ProjectTeamsPermissionType = "none"
	ProjectTeamsPermissionRead   ProjectTeamsPermissionType = "read"
	ProjectTeamsPermissionManage ProjectTeamsPermissionType = "manage"
)

// ProjectVariableSetsPermissionType represents the permission type to a project's variable sets
type ProjectVariableSetsPermissionType string

const (
	ProjectVariableSetsPermissionNone  ProjectVariableSetsPermissionType = "none"
	ProjectVariableSetsPermissionRead  ProjectVariableSetsPermissionType = "read"
	ProjectVariableSetsPermissionWrite ProjectVariableSetsPermissionType = "write"
)

// WorkspaceRunsPermissionType represents the permissiontype to project workspaces' runs
type WorkspaceRunsPermissionType string

const (
	WorkspaceRunsPermissionRead  WorkspaceRunsPermissionType = "read"
	WorkspaceRunsPermissionPlan  WorkspaceRunsPermissionType = "plan"
	WorkspaceRunsPermissionApply WorkspaceRunsPermissionType = "apply"
)

// WorkspaceSentinelMocksPermissionType represents the permissiontype to project workspaces' sentinel-mocks
type WorkspaceSentinelMocksPermissionType string

const (
	WorkspaceSentinelMocksPermissionNone WorkspaceSentinelMocksPermissionType = "none"
	WorkspaceSentinelMocksPermissionRead WorkspaceSentinelMocksPermissionType = "read"
)

// WorkspaceStateVersionsPermissionType represents the permissiontype to project workspaces' state-versions
type WorkspaceStateVersionsPermissionType string

const (
	WorkspaceStateVersionsPermissionNone        WorkspaceStateVersionsPermissionType = "none"
	WorkspaceStateVersionsPermissionReadOutputs WorkspaceStateVersionsPermissionType = "read-outputs"
	WorkspaceStateVersionsPermissionRead        WorkspaceStateVersionsPermissionType = "read"
	WorkspaceStateVersionsPermissionWrite       WorkspaceStateVersionsPermissionType = "write"
)

// WorkspaceVariablesPermissionType represents the permissiontype to project workspaces' variables
type WorkspaceVariablesPermissionType string

const (
	WorkspaceVariablesPermissionNone  WorkspaceVariablesPermissionType = "none"
	WorkspaceVariablesPermissionRead  WorkspaceVariablesPermissionType = "read"
	WorkspaceVariablesPermissionWrite WorkspaceVariablesPermissionType = "write"
)

type TeamProjectAccessProjectPermissionsOptions struct {
	Settings     *ProjectSettingsPermissionType     `json:"settings,omitempty"`
	Teams        *ProjectTeamsPermissionType        `json:"teams,omitempty"`
	VariableSets *ProjectVariableSetsPermissionType `json:"variable-sets,omitempty"`
}

type TeamProjectAccessWorkspacePermissionsOptions struct {
	Runs          *WorkspaceRunsPermissionType          `json:"runs,omitempty"`
	SentinelMocks *WorkspaceSentinelMocksPermissionType `json:"sentinel-mocks,omitempty"`
	StateVersions *WorkspaceStateVersionsPermissionType `json:"state-versions,omitempty"`
	Variables     *WorkspaceVariablesPermissionType     `json:"variables,omitempty"`
	Create        *bool                                 `json:"create,omitempty"`
	Locking       *bool                                 `json:"locking,omitempty"`
	Move          *bool                                 `json:"move,omitempty"`
	Delete        *bool                                 `json:"delete,omitempty"`
	RunTasks      *bool                                 `json:"run-tasks,omitempty"`
	// **Note: This API is still in BETA and subject to change.**
	PolicyOverrides *bool `json:"policy-overrides,omitempty"`
}

// TeamProjectAccessListOptions represents the options for listing team project accesses
type TeamProjectAccessListOptions struct {
	ListOptions
	ProjectID string `url:"filter[project][id]"`
}

// TeamProjectAccessAddOptions represents the options for adding team access for a project
type TeamProjectAccessAddOptions struct {
	// Type is a public field utilized by JSON:API to
	// set the resource type via the field tag.
	// It is not a user-defined value and does not need to be set.
	// https://jsonapi.org/format/#crud-creating
	Type string `jsonapi:"primary,team-projects"`
	// The type of access to grant.
	Access TeamProjectAccessType `jsonapi:"attr,access"`
	// The levels that project and workspace permissions grant
	ProjectAccess   *TeamProjectAccessProjectPermissionsOptions   `jsonapi:"attr,project-access,omitempty"`
	WorkspaceAccess *TeamProjectAccessWorkspacePermissionsOptions `jsonapi:"attr,workspace-access,omitempty"`

	// The team to add to the project
	Team *Team `jsonapi:"relation,team"`
	// The project to which the team is to be added.
	Project *Project `jsonapi:"relation,project"`
}

// TeamProjectAccessUpdateOptions represents the options for updating a team project access
type TeamProjectAccessUpdateOptions struct {
	// Type is a public field utilized by JSON:API to
	// set the resource type via the field tag.
	// It is not a user-defined value and does not need to be set.
	// https://jsonapi.org/format/#crud-creating
	Type string `jsonapi:"primary,team-projects"`
	// The type of access to grant.
	Access          *TeamProjectAccessType                        `jsonapi:"attr,access,omitempty"`
	ProjectAccess   *TeamProjectAccessProjectPermissionsOptions   `jsonapi:"attr,project-access,omitempty"`
	WorkspaceAccess *TeamProjectAccessWorkspacePermissionsOptions `jsonapi:"attr,workspace-access,omitempty"`
}

// List all team accesses for a given project.
func (s *teamProjectAccesses) List(ctx context.Context, options TeamProjectAccessListOptions) (*TeamProjectAccessList, error) {
	if err := options.valid(); err != nil {
		return nil, err
	}

	req, err := s.client.NewRequest("GET", "team-projects", &options)
	if err != nil {
		return nil, err
	}

	tpal := &TeamProjectAccessList{}
	err = req.Do(ctx, tpal)
	if err != nil {
		return nil, err
	}

	return tpal, nil
}

// Add team access for a project.
func (s *teamProjectAccesses) Add(ctx context.Context, options TeamProjectAccessAddOptions) (*TeamProjectAccess, error) {
	if err := options.valid(); err != nil {
		return nil, err
	}

	if err := validateTeamProjectAccessType(options.Access); err != nil {
		return nil, err
	}

	req, err := s.client.NewRequest("POST", "team-projects", &options)
	if err != nil {
		return nil, err
	}

	tpa := &TeamProjectAccess{}
	err = req.Do(ctx, tpa)
	if err != nil {
		return nil, err
	}

	return tpa, nil
}

// Read a team project access by its ID.
func (s *teamProjectAccesses) Read(ctx context.Context, teamProjectAccessID string) (*TeamProjectAccess, error) {
	if !validStringID(&teamProjectAccessID) {
		return nil, ErrInvalidTeamProjectAccessID
	}

	u := fmt.Sprintf("team-projects/%s", url.PathEscape(teamProjectAccessID))
	req, err := s.client.NewRequest("GET", u, nil)
	if err != nil {
		return nil, err
	}

	tpa := &TeamProjectAccess{}
	err = req.Do(ctx, tpa)
	if err != nil {
		return nil, err
	}

	return tpa, nil
}

// Update team access for a project.
func (s *teamProjectAccesses) Update(ctx context.Context, teamProjectAccessID string, options TeamProjectAccessUpdateOptions) (*TeamProjectAccess, error) {
	if !validStringID(&teamProjectAccessID) {
		return nil, ErrInvalidTeamProjectAccessID
	}

	if err := validateTeamProjectAccessType(*options.Access); err != nil {
		return nil, err
	}
	u := fmt.Sprintf("team-projects/%s", url.PathEscape(teamProjectAccessID))
	req, err := s.client.NewRequest("PATCH", u, &options)
	if err != nil {
		return nil, err
	}

	ta := &TeamProjectAccess{}
	err = req.Do(ctx, ta)
	if err != nil {
		return nil, err
	}

	return ta, err
}

// Remove team access from a project.
func (s *teamProjectAccesses) Remove(ctx context.Context, teamProjectAccessID string) error {
	if !validStringID(&teamProjectAccessID) {
		return ErrInvalidTeamProjectAccessID
	}

	u := fmt.Sprintf("team-projects/%s", url.PathEscape(teamProjectAccessID))
	req, err := s.client.NewRequest("DELETE", u, nil)
	if err != nil {
		return err
	}

	return req.Do(ctx, nil)
}

func (o TeamProjectAccessListOptions) valid() error {
	if !validStringID(&o.ProjectID) {
		return ErrInvalidProjectID
	}

	return nil
}

func (o TeamProjectAccessAddOptions) valid() error {
	if err := validateTeamProjectAccessType(o.Access); err != nil {
		return err
	}
	if o.Team == nil {
		return ErrRequiredTeam
	}
	if o.Project == nil {
		return ErrRequiredProject
	}

	return nil
}

func validateTeamProjectAccessType(t TeamProjectAccessType) error {
	switch t {
	case TeamProjectAccessAdmin,
		TeamProjectAccessMaintain,
		TeamProjectAccessWrite,
		TeamProjectAccessRead,
		TeamProjectAccessCustom:
		// do nothing
	default:
		return ErrInvalidTeamProjectAccessType
	}
	return nil
}

// Compile-time proof of interface implementation.
var _ TeamTokens = (*teamTokens)(nil)

// TeamTokens describes all the team token related methods that the
// Terraform Enterprise API supports.
//
// TFE API docs:
// https://developer.hashicorp.com/terraform/cloud-docs/api-docs/team-tokens
type TeamTokens interface {
	// Create a new team token using the legacy creation behavior, which creates a token without a description
	// or regenerates the existing, descriptionless token.
	Create(ctx context.Context, teamID string) (*TeamToken, error)

	// CreateWithOptions creates a team token, with options. If no description is provided, it uses the legacy
	// creation behavior, which regenerates the descriptionless token if it already exists. Otherwise, it create
	//  a new token with the given unique description, allowing for the creation of multiple team tokens.
	CreateWithOptions(ctx context.Context, teamID string, options TeamTokenCreateOptions) (*TeamToken, error)

	// Read a team token by its team ID.
	Read(ctx context.Context, teamID string) (*TeamToken, error)

	// Read a team token by its token ID.
	ReadByID(ctx context.Context, teamID string) (*TeamToken, error)

	// List an organization's team tokens.
	List(ctx context.Context, organizationID string, options *TeamTokenListOptions) (*TeamTokenList, error)

	// Delete a team token by its team ID.
	Delete(ctx context.Context, teamID string) error

	// Delete a team token by its token ID.
	DeleteByID(ctx context.Context, tokenID string) error
}

// teamTokens implements TeamTokens.
type teamTokens struct {
	client *Client
}

// TeamToken represents a Terraform Enterprise team token.
type TeamToken struct {
	ID          string           `jsonapi:"primary,authentication-tokens"`
	CreatedAt   time.Time        `jsonapi:"attr,created-at,iso8601"`
	Description *string          `jsonapi:"attr,description"`
	LastUsedAt  time.Time        `jsonapi:"attr,last-used-at,iso8601"`
	Token       string           `jsonapi:"attr,token"`
	ExpiredAt   time.Time        `jsonapi:"attr,expired-at,iso8601"`
	CreatedBy   *CreatedByChoice `jsonapi:"polyrelation,created-by"`
	Team        *Team            `jsonapi:"relation,team"`
}

// TeamTokenCreateOptions contains the options for creating a team token.
type TeamTokenCreateOptions struct {
	// Optional: The token's expiration date.
	// This feature is available in TFE release v202305-1 and later
	ExpiredAt *time.Time `jsonapi:"attr,expired-at,iso8601,omitempty"`

	// Optional: The token's description, which must unique per team.
	// This feature is considered BETA, SUBJECT TO CHANGE, and likely unavailable to most users.
	Description *string `jsonapi:"attr,description,omitempty"`
}

// TeamTokenListOptions contains the options for listing team tokens.
type TeamTokenListOptions struct {
	ListOptions

	// Optional: A query string used to filter team tokens by
	// a specified team name.
	Query string `url:"q,omitempty"`

	// Optional: Allows sorting the team tokens by "team-name",
	// "created-by", "expired-at", and "last-used-at"
	Sort string `url:"sort,omitempty"`
}

// TeamTokenList represents a list of team tokens.
type TeamTokenList struct {
	*Pagination
	Items []*TeamToken
}

// Create a new team token using the legacy creation behavior, which creates a token without a description
// or regenerates the existing, descriptionless token.
func (s *teamTokens) Create(ctx context.Context, teamID string) (*TeamToken, error) {
	return s.CreateWithOptions(ctx, teamID, TeamTokenCreateOptions{})
}

// CreateWithOptions creates a team token, with options. If no description is provided, it uses the legacy
// creation behavior, which regenerates the descriptionless token if it already exists. Otherwise, it create
// a new token with the given unique description, allowing for the creation of multiple team tokens.
func (s *teamTokens) CreateWithOptions(ctx context.Context, teamID string, options TeamTokenCreateOptions) (*TeamToken, error) {
	if !validStringID(&teamID) {
		return nil, ErrInvalidTeamID
	}

	var u string
	if options.Description != nil {
		u = fmt.Sprintf("teams/%s/authentication-tokens", url.PathEscape(teamID))
	} else {
		u = fmt.Sprintf("teams/%s/authentication-token", url.PathEscape(teamID))
	}

	req, err := s.client.NewRequest("POST", u, &options)
	if err != nil {
		return nil, err
	}

	tt := &TeamToken{}
	err = req.Do(ctx, tt)
	if err != nil {
		return nil, err
	}

	return tt, err
}

// Read a team token by its team ID.
func (s *teamTokens) Read(ctx context.Context, teamID string) (*TeamToken, error) {
	if !validStringID(&teamID) {
		return nil, ErrInvalidTeamID
	}

	u := fmt.Sprintf("teams/%s/authentication-token", url.PathEscape(teamID))
	req, err := s.client.NewRequest("GET", u, nil)
	if err != nil {
		return nil, err
	}

	tt := &TeamToken{}
	err = req.Do(ctx, tt)
	if err != nil {
		return nil, err
	}

	return tt, err
}

// Read a team token by its token ID.
func (s *teamTokens) ReadByID(ctx context.Context, tokenID string) (*TeamToken, error) {
	if !validStringID(&tokenID) {
		return nil, ErrInvalidTokenID
	}

	u := fmt.Sprintf(AuthenticationTokensPath, url.PathEscape(tokenID))
	req, err := s.client.NewRequest("GET", u, nil)
	if err != nil {
		return nil, err
	}

	tt := &TeamToken{}
	err = req.Do(ctx, tt)
	if err != nil {
		return nil, err
	}

	return tt, err
}

// List an organization's team tokens with the option to filter by team name.
func (s *teamTokens) List(ctx context.Context, organizationID string, options *TeamTokenListOptions) (*TeamTokenList, error) {
	if !validStringID(&organizationID) {
		return nil, ErrInvalidOrg
	}

	u := fmt.Sprintf("organizations/%s/team-tokens", url.PathEscape(organizationID))

	req, err := s.client.NewRequest("GET", u, options)
	if err != nil {
		return nil, err
	}

	tt := &TeamTokenList{}
	err = req.Do(ctx, tt)
	if err != nil {
		return nil, err
	}

	return tt, err
}

// Delete a team token by its team ID.
func (s *teamTokens) Delete(ctx context.Context, teamID string) error {
	if !validStringID(&teamID) {
		return ErrInvalidTeamID
	}

	u := fmt.Sprintf("teams/%s/authentication-token", url.PathEscape(teamID))
	req, err := s.client.NewRequest("DELETE", u, nil)
	if err != nil {
		return err
	}

	return req.Do(ctx, nil)
}

// Delete a team token by its token ID.
func (s *teamTokens) DeleteByID(ctx context.Context, tokenID string) error {
	if !validStringID(&tokenID) {
		return ErrInvalidTokenID
	}

	u := fmt.Sprintf(AuthenticationTokensPath, url.PathEscape(tokenID))
	req, err := s.client.NewRequest("DELETE", u, nil)
	if err != nil {
		return err
	}

	return req.Do(ctx, nil)
}

// Compile-time proof of interface implementation.
var _ Teams = (*teams)(nil)

// Teams describes all the team related methods that the Terraform
// Enterprise API supports.
//
// TFE API docs: https://developer.hashicorp.com/terraform/cloud-docs/api-docs/teams
type Teams interface {
	// List all the teams of the given organization.
	List(ctx context.Context, organization string, options *TeamListOptions) (*TeamList, error)

	// Create a new team with the given options.
	Create(ctx context.Context, organization string, options TeamCreateOptions) (*Team, error)

	// Read a team by its ID.
	Read(ctx context.Context, teamID string) (*Team, error)

	// Update a team by its ID.
	Update(ctx context.Context, teamID string, options TeamUpdateOptions) (*Team, error)

	// Delete a team by its ID.
	Delete(ctx context.Context, teamID string) error
}

// teams implements Teams.
type teams struct {
	client *Client
}

// TeamList represents a list of teams.
type TeamList struct {
	*Pagination
	Items []*Team
}

// Team represents a Terraform Enterprise team.
type Team struct {
	ID                 string              `jsonapi:"primary,teams"`
	IsUnified          bool                `jsonapi:"attr,is-unified"`
	Name               string              `jsonapi:"attr,name"`
	OrganizationAccess *OrganizationAccess `jsonapi:"attr,organization-access"`
	Visibility         string              `jsonapi:"attr,visibility"`
	Permissions        *TeamPermissions    `jsonapi:"attr,permissions"`
	UserCount          int                 `jsonapi:"attr,users-count"`
	SSOTeamID          string              `jsonapi:"attr,sso-team-id"`
	// AllowMemberTokenManagement is false for TFE versions older than v202408
	AllowMemberTokenManagement bool `jsonapi:"attr,allow-member-token-management"`

	// Relations
	Users                   []*User                   `jsonapi:"relation,users"`
	OrganizationMemberships []*OrganizationMembership `jsonapi:"relation,organization-memberships"`

	// SCIM Attributes
	SCIMLinked     *bool      `jsonapi:"attr,scim-linked"`
	SCIMSyncPaused *bool      `jsonapi:"attr,scim-sync-paused"`
	SCIMGroupName  *string    `jsonapi:"attr,scim-group-name"`
	SCIMUpdatedAt  *time.Time `jsonapi:"attr,scim-updated-at,iso8601"`
}

// OrganizationAccess represents the team's permissions on its organization
type OrganizationAccess struct {
	ManagePolicies        bool `jsonapi:"attr,manage-policies"`
	ManagePolicyOverrides bool `jsonapi:"attr,manage-policy-overrides"`
	// **Note: This API is still in BETA and subject to change.**
	DelegatePolicyOverrides  bool `jsonapi:"attr,delegate-policy-overrides"`
	ManageWorkspaces         bool `jsonapi:"attr,manage-workspaces"`
	ManageVCSSettings        bool `jsonapi:"attr,manage-vcs-settings"`
	ManageProviders          bool `jsonapi:"attr,manage-providers"`
	ManageModules            bool `jsonapi:"attr,manage-modules"`
	ManageRunTasks           bool `jsonapi:"attr,manage-run-tasks"`
	ManageProjects           bool `jsonapi:"attr,manage-projects"`
	ReadWorkspaces           bool `jsonapi:"attr,read-workspaces"`
	ReadProjects             bool `jsonapi:"attr,read-projects"`
	ManageMembership         bool `jsonapi:"attr,manage-membership"`
	ManageTeams              bool `jsonapi:"attr,manage-teams"`
	ManageOrganizationAccess bool `jsonapi:"attr,manage-organization-access"`
	AccessSecretTeams        bool `jsonapi:"attr,access-secret-teams"`
	ManageAgentPools         bool `jsonapi:"attr,manage-agent-pools"`
}

// TeamPermissions represents the current user's permissions on the team.
type TeamPermissions struct {
	CanDestroy          bool `jsonapi:"attr,can-destroy"`
	CanUpdateMembership bool `jsonapi:"attr,can-update-membership"`
}

// TeamIncludeOpt represents the available options for include query params.
// https://developer.hashicorp.com/terraform/cloud-docs/api-docs/teams#available-related-resources
type TeamIncludeOpt string

const (
	TeamUsers                   TeamIncludeOpt = "users"
	TeamOrganizationMemberships TeamIncludeOpt = "organization-memberships"
)

// TeamListOptions represents the options for listing teams.
type TeamListOptions struct {
	ListOptions
	// Optional: A list of relations to include.
	// https://developer.hashicorp.com/terraform/cloud-docs/api-docs/teams#available-related-resources
	Include []TeamIncludeOpt `url:"include,omitempty"`

	// Optional: A list of team names to filter by.
	Names []string `url:"filter[names],omitempty"`

	// Optional: A query string to search teams by names.
	Query string `url:"q,omitempty"`
}

// TeamCreateOptions represents the options for creating a team.
type TeamCreateOptions struct {
	// Type is a public field utilized by JSON:API to
	// set the resource type via the field tag.
	// It is not a user-defined value and does not need to be set.
	// https://jsonapi.org/format/#crud-creating
	Type string `jsonapi:"primary,teams"`

	// Name of the team.
	Name *string `jsonapi:"attr,name"`

	// Optional: Unique Identifier to control team membership via SAML
	SSOTeamID *string `jsonapi:"attr,sso-team-id,omitempty"`

	// The team's organization access
	OrganizationAccess *OrganizationAccessOptions `jsonapi:"attr,organization-access,omitempty"`

	// The team's visibility ("secret", "organization")
	Visibility *string `jsonapi:"attr,visibility,omitempty"`

	// Optional: Used by Owners and users with "Manage Teams" permissions to control whether team members can manage team tokens
	AllowMemberTokenManagement *bool `jsonapi:"attr,allow-member-token-management,omitempty"`
}

// TeamUpdateOptions represents the options for updating a team.
type TeamUpdateOptions struct {
	// Type is a public field utilized by JSON:API to
	// set the resource type via the field tag.
	// It is not a user-defined value and does not need to be set.
	// https://jsonapi.org/format/#crud-creating
	Type string `jsonapi:"primary,teams"`

	// Optional: New name for the team
	Name *string `jsonapi:"attr,name,omitempty"`

	// Optional: Unique Identifier to control team membership via SAML
	SSOTeamID *string `jsonapi:"attr,sso-team-id,omitempty"`

	// Optional: The team's organization access
	OrganizationAccess *OrganizationAccessOptions `jsonapi:"attr,organization-access,omitempty"`

	// Optional: The team's visibility ("secret", "organization")
	Visibility *string `jsonapi:"attr,visibility,omitempty"`

	// Optional: Used by Owners and users with "Manage Teams" permissions to control whether team members can manage team tokens
	AllowMemberTokenManagement *bool `jsonapi:"attr,allow-member-token-management,omitempty"`
}

// OrganizationAccessOptions represents the organization access options of a team.
type OrganizationAccessOptions struct {
	ManagePolicies        *bool `json:"manage-policies,omitempty"`
	ManagePolicyOverrides *bool `json:"manage-policy-overrides,omitempty"`
	// **Note: This API is still in BETA and subject to change.**
	DelegatePolicyOverrides  *bool `json:"delegate-policy-overrides,omitempty"`
	ManageWorkspaces         *bool `json:"manage-workspaces,omitempty"`
	ManageVCSSettings        *bool `json:"manage-vcs-settings,omitempty"`
	ManageProviders          *bool `json:"manage-providers,omitempty"`
	ManageModules            *bool `json:"manage-modules,omitempty"`
	ManageRunTasks           *bool `json:"manage-run-tasks,omitempty"`
	ManageProjects           *bool `json:"manage-projects,omitempty"`
	ReadWorkspaces           *bool `json:"read-workspaces,omitempty"`
	ReadProjects             *bool `json:"read-projects,omitempty"`
	ManageMembership         *bool `json:"manage-membership,omitempty"`
	ManageTeams              *bool `json:"manage-teams,omitempty"`
	ManageOrganizationAccess *bool `json:"manage-organization-access,omitempty"`
	AccessSecretTeams        *bool `json:"access-secret-teams,omitempty"`
	ManageAgentPools         *bool `json:"manage-agent-pools,omitempty"`
}

// List all the teams of the given organization.
func (s *teams) List(ctx context.Context, organization string, options *TeamListOptions) (*TeamList, error) {
	if !validStringID(&organization) {
		return nil, ErrInvalidOrg
	}
	if err := options.valid(); err != nil {
		return nil, err
	}
	u := fmt.Sprintf("organizations/%s/teams", url.PathEscape(organization))
	req, err := s.client.NewRequest("GET", u, options)
	if err != nil {
		return nil, err
	}

	tl := &TeamList{}
	err = req.Do(ctx, tl)
	if err != nil {
		return nil, err
	}

	return tl, nil
}

// Create a new team with the given options.
func (s *teams) Create(ctx context.Context, organization string, options TeamCreateOptions) (*Team, error) {
	if !validStringID(&organization) {
		return nil, ErrInvalidOrg
	}
	if err := options.valid(); err != nil {
		return nil, err
	}

	u := fmt.Sprintf("organizations/%s/teams", url.PathEscape(organization))
	req, err := s.client.NewRequest("POST", u, &options)
	if err != nil {
		return nil, err
	}

	t := &Team{}
	err = req.Do(ctx, t)
	if err != nil {
		return nil, err
	}

	return t, nil
}

// Read a single team by its ID.
func (s *teams) Read(ctx context.Context, teamID string) (*Team, error) {
	if !validStringID(&teamID) {
		return nil, ErrInvalidTeamID
	}

	u := fmt.Sprintf("teams/%s", url.PathEscape(teamID))
	req, err := s.client.NewRequest("GET", u, nil)
	if err != nil {
		return nil, err
	}

	t := &Team{}
	err = req.Do(ctx, t)
	if err != nil {
		return nil, err
	}

	return t, nil
}

// Update a team by its ID.
func (s *teams) Update(ctx context.Context, teamID string, options TeamUpdateOptions) (*Team, error) {
	if !validStringID(&teamID) {
		return nil, ErrInvalidTeamID
	}

	u := fmt.Sprintf("teams/%s", url.PathEscape(teamID))
	req, err := s.client.NewRequest("PATCH", u, &options)
	if err != nil {
		return nil, err
	}

	t := &Team{}
	err = req.Do(ctx, t)
	if err != nil {
		return nil, err
	}

	return t, nil
}

// Delete a team by its ID.
func (s *teams) Delete(ctx context.Context, teamID string) error {
	if !validStringID(&teamID) {
		return ErrInvalidTeamID
	}

	u := fmt.Sprintf("teams/%s", url.PathEscape(teamID))
	req, err := s.client.NewRequest("DELETE", u, nil)
	if err != nil {
		return err
	}

	return req.Do(ctx, nil)
}

func (o TeamCreateOptions) valid() error {
	if !validString(o.Name) {
		return ErrRequiredName
	}
	return nil
}

func (o *TeamListOptions) valid() error {
	if o == nil {
		return nil // nothing to validate
	}

	if err := validateTeamNames(o.Names); err != nil {
		return err
	}

	return nil
}

func validateTeamNames(names []string) error {
	for _, name := range names {
		if name == "" {
			return ErrEmptyTeamName
		}
	}

	return nil
}

type TestConfig struct {
	TestsEnabled       bool    `jsonapi:"attr,tests-enabled"`
	AgentExecutionMode *string `jsonapi:"attr,agent-execution-mode,omitempty"`
	AgentPoolID        *string `jsonapi:"attr,agent-pool-id,omitempty"`
}

// Compile-time proof of interface implementation.
var _ TestRuns = (*testRuns)(nil)

// TestRuns describes all the test run related methods that the Terraform
// Enterprise API supports.
//
// **Note: These methods are still in BETA and subject to change.**
type TestRuns interface {
	// List all the test runs for a given private registry module.
	List(ctx context.Context, moduleID RegistryModuleID, options *TestRunListOptions) (*TestRunList, error)

	// Read a test run by its ID.
	Read(ctx context.Context, moduleID RegistryModuleID, testRunID string) (*TestRun, error)

	// Create a new test run with the given options.
	Create(ctx context.Context, options TestRunCreateOptions) (*TestRun, error)

	// Logs retrieves the logs for a test run by its ID.
	Logs(ctx context.Context, moduleID RegistryModuleID, testRunID string) (io.Reader, error)

	// Cancel a test run by its ID.
	Cancel(ctx context.Context, moduleID RegistryModuleID, testRunID string) error

	// ForceCancel a test run by its ID.
	ForceCancel(ctx context.Context, moduleID RegistryModuleID, testRunID string) error
}

// testRuns implements TestRuns.
type testRuns struct {
	client *Client
}

// TestRunStatus represents the status of a test run.
type TestRunStatus string

// List all available test run statuses.
const (
	TestRunPending  TestRunStatus = "pending"
	TestRunQueued   TestRunStatus = "queued"
	TestRunRunning  TestRunStatus = "running"
	TestRunErrored  TestRunStatus = "errored"
	TestRunCanceled TestRunStatus = "canceled"
	TestRunFinished TestRunStatus = "finished"
)

// TestStatus represents the status of an individual test within an overall test
// run.
type TestStatus string

// List all available test statuses.
const (
	TestPending TestStatus = "pending"
	TestSkip    TestStatus = "skip"
	TestPass    TestStatus = "pass"
	TestFail    TestStatus = "fail"
	TestError   TestStatus = "error"
)

// TestRun represents a Terraform Enterprise test run.
type TestRun struct {
	ID               string                  `jsonapi:"primary,test-runs"`
	Status           TestRunStatus           `jsonapi:"attr,status"`
	StatusTimestamps TestRunStatusTimestamps `jsonapi:"attr,status-timestamps"`
	TestStatus       TestStatus              `jsonapi:"attr,test-status"`
	TestsPassed      int                     `jsonapi:"attr,tests-passed"`
	TestsFailed      int                     `jsonapi:"attr,tests-failed"`
	TestsErrored     int                     `jsonapi:"attr,tests-errored"`
	TestsSkipped     int                     `jsonapi:"attr,tests-skipped"`
	LogReadURL       string                  `jsonapi:"attr,log-read-url"`

	// Relations
	ConfigurationVersion *ConfigurationVersion `jsonapi:"relation,configuration-version"`
	RegistryModule       *RegistryModule       `jsonapi:"relation,registry-module"`
}

// TestRunStatusTimestamps holds the timestamps for individual test run
// statuses.
type TestRunStatusTimestamps struct {
	CanceledAt      time.Time `jsonapi:"attr,canceled-at,rfc3339"`
	ErroredAt       time.Time `jsonapi:"attr,errored-at,rfc3339"`
	FinishedAt      time.Time `jsonapi:"attr,finished-at,rfc3339"`
	ForceCanceledAt time.Time `jsonapi:"attr,force-canceled-at,rfc3339"`
	QueuedAt        time.Time `jsonapi:"attr,queued-at,rfc3339"`
	StartedAt       time.Time `jsonapi:"attr,started-at,rfc3339"`
}

// TestRunCreateOptions represents the options for creating a run.
type TestRunCreateOptions struct {
	// Type is a public field utitilized by JSON:API to set the resource type
	// via the field tag. It is not a user-defined value and does not need to
	// be set.
	// https://jsonapi.org/format/#crud-creating
	Type string `jsonapi:"primary,test-runs"`

	// If non-empty, requests that only a subset of testing files within the
	// ConfigurationVersion should be executed.
	Filters []string `jsonapi:"attr,filters,omitempty"`

	// Specifies the directory within the ConfigurationVersion that test files
	// should be loaded from. Defaults to "tests" if empty.
	TestDirectory *string `jsonapi:"attr,test-directory,omitempty"`

	// Verbose prints out the plan and state files for each run block that is
	// executed by this TestRun.
	Verbose *bool `jsonapi:"attr,verbose,omitempty"`

	// Parallelism controls the number of parallel operations to execute within a single test run.
	Parallelism *int `jsonapi:"attr,parallelism,omitempty"`

	// Variables allows you to specify terraform input variables for
	// a particular run, prioritized over variables defined on the workspace.
	Variables []*RunVariable `jsonapi:"attr,variables,omitempty"`

	// ConfigurationVersion specifies the configuration version to use for this
	// test run.
	ConfigurationVersion *ConfigurationVersion `jsonapi:"relation,configuration-version"`

	// RegistryModule specifies the registry module this test run should be
	// assigned to.
	RegistryModule *RegistryModule `jsonapi:"relation,registry-module"`
}

// TestRunList represents a list of test runs.
type TestRunList struct {
	*Pagination
	Items []*TestRun
}

// TestRunListOptions represents the options for listing runs.
type TestRunListOptions struct {
	ListOptions
}

// List all the test runs for a given private registry module.
func (s *testRuns) List(ctx context.Context, moduleID RegistryModuleID, options *TestRunListOptions) (*TestRunList, error) {
	if err := moduleID.valid(); err != nil {
		return nil, err
	}

	req, err := s.client.NewRequest("GET", testRunsPath(moduleID), options)
	if err != nil {
		return nil, err
	}

	trl := &TestRunList{}
	err = req.Do(ctx, trl)
	if err != nil {
		return nil, err
	}

	return trl, nil
}

// Read a test run by its ID.
func (s *testRuns) Read(ctx context.Context, moduleID RegistryModuleID, testRunID string) (*TestRun, error) {
	if err := moduleID.valid(); err != nil {
		return nil, err
	}

	if !validStringID(&testRunID) {
		return nil, ErrInvalidTestRunID
	}

	u := fmt.Sprintf("%s/%s", testRunsPath(moduleID), url.PathEscape(testRunID))
	req, err := s.client.NewRequest("GET", u, nil)
	if err != nil {
		return nil, err
	}

	tr := &TestRun{}
	err = req.Do(ctx, tr)
	if err != nil {
		return nil, err
	}

	return tr, nil
}

// Create a new test run with the given options.
func (s *testRuns) Create(ctx context.Context, options TestRunCreateOptions) (*TestRun, error) {
	if err := options.valid(); err != nil {
		return nil, err
	}

	moduleID := RegistryModuleID{
		Organization: options.RegistryModule.Organization.Name,
		Name:         options.RegistryModule.Name,
		Provider:     options.RegistryModule.Provider,
		Namespace:    options.RegistryModule.Namespace,
		RegistryName: options.RegistryModule.RegistryName,
	}

	if err := moduleID.valid(); err != nil {
		return nil, err
	}

	if err := options.valid(); err != nil {
		return nil, err
	}

	req, err := s.client.NewRequest("POST", testRunsPath(moduleID), &options)
	if err != nil {
		return nil, err
	}

	tr := &TestRun{}
	err = req.Do(ctx, tr)
	if err != nil {
		return nil, err
	}

	return tr, nil
}

// Logs retrieves the logs for a test run by its ID.
func (s *testRuns) Logs(ctx context.Context, moduleID RegistryModuleID, testRunID string) (io.Reader, error) {
	if err := moduleID.valid(); err != nil {
		return nil, err
	}

	if !validStringID(&testRunID) {
		return nil, ErrInvalidTestRunID
	}

	tr, err := s.Read(ctx, moduleID, testRunID)
	if err != nil {
		return nil, err
	}

	if tr.LogReadURL == "" {
		return nil, fmt.Errorf("test run %s does not have a log URL", testRunID)
	}

	u, err := url.Parse(tr.LogReadURL)
	if err != nil {
		return nil, fmt.Errorf("invalid log URL: %w", err)
	}

	done := func() (bool, error) {
		tr, err := s.Read(ctx, moduleID, testRunID)
		if err != nil {
			return false, err
		}

		switch tr.Status {
		case TestRunErrored, TestRunCanceled, TestRunFinished:
			return true, nil
		default:
			return false, nil
		}
	}

	return &LogReader{
		client: s.client,
		ctx:    ctx,
		done:   done,
		logURL: u,
	}, nil
}

// Cancel a test run by its ID.
func (s *testRuns) Cancel(ctx context.Context, moduleID RegistryModuleID, testRunID string) error {
	if err := moduleID.valid(); err != nil {
		return err
	}

	if !validStringID(&testRunID) {
		return ErrInvalidTestRunID
	}

	u := fmt.Sprintf("%s/%s/cancel", testRunsPath(moduleID), url.PathEscape(testRunID))
	req, err := s.client.NewRequest("POST", u, nil)
	if err != nil {
		return err
	}

	return req.Do(ctx, nil)
}

// ForceCancel a test run by its ID.
func (s *testRuns) ForceCancel(ctx context.Context, moduleID RegistryModuleID, testRunID string) error {
	if err := moduleID.valid(); err != nil {
		return err
	}

	if !validStringID(&testRunID) {
		return ErrInvalidTestRunID
	}

	u := fmt.Sprintf("%s/%s/force-cancel", testRunsPath(moduleID), url.PathEscape(testRunID))
	req, err := s.client.NewRequest("POST", u, nil)
	if err != nil {
		return err
	}

	return req.Do(ctx, nil)
}

func (o TestRunCreateOptions) valid() error {
	if o.ConfigurationVersion == nil {
		return ErrInvalidConfigVersionID
	}

	if o.RegistryModule == nil {
		return ErrRequiredRegistryModule
	}

	if o.RegistryModule.Organization == nil {
		return ErrRequiredOrg
	}

	return nil
}

func testRunsPath(moduleID RegistryModuleID) string {
	return fmt.Sprintf("organizations/%s/tests/registry-modules/%s/%s/%s/%s/test-runs",
		url.PathEscape(moduleID.Organization),
		url.PathEscape(string(moduleID.RegistryName)),
		url.PathEscape(moduleID.Namespace),
		url.PathEscape(moduleID.Name),
		url.PathEscape(moduleID.Provider))
}

// Compile-time proof of interface implementation.
var _ TestVariables = (*testVariables)(nil)

// Variables describes all the variable related methods that the Terraform
// Enterprise API supports.
//
// TFE API docs: https://developer.hashicorp.com/terraform/cloud-docs/api-docs/private-registry/tests
type TestVariables interface {
	// List all the test variables associated with the given module.
	List(ctx context.Context, moduleID RegistryModuleID, options *VariableListOptions) (*VariableList, error)

	// Read a test variable by its ID.
	Read(ctx context.Context, moduleID RegistryModuleID, variableID string) (*Variable, error)

	// Create is used to create a new variable.
	Create(ctx context.Context, moduleID RegistryModuleID, options VariableCreateOptions) (*Variable, error)

	// Update values of an existing variable.
	Update(ctx context.Context, moduleID RegistryModuleID, variableID string, options VariableUpdateOptions) (*Variable, error)

	// Delete a variable by its ID.
	Delete(ctx context.Context, moduleID RegistryModuleID, variableID string) error
}

// variables implements Variables.
type testVariables struct {
	client *Client
}

// List all the variables associated with the given module.
func (s *testVariables) List(ctx context.Context, moduleID RegistryModuleID, options *VariableListOptions) (*VariableList, error) {
	if err := moduleID.valid(); err != nil {
		return nil, err
	}

	req, err := s.client.NewRequest("GET", testVarsPath(moduleID), options)
	if err != nil {
		return nil, err
	}

	vl := &VariableList{}
	err = req.Do(ctx, vl)
	if err != nil {
		return nil, err
	}

	return vl, nil
}

// Read a variable by its ID.
func (s *testVariables) Read(ctx context.Context, moduleID RegistryModuleID, variableID string) (*Variable, error) {
	if err := moduleID.valid(); err != nil {
		return nil, err
	}

	if !validStringID(&variableID) {
		return nil, ErrInvalidVariableID
	}

	u := fmt.Sprintf("%s/%s", testVarsPath(moduleID), url.PathEscape(variableID))

	req, err := s.client.NewRequest("GET", u, nil)
	if err != nil {
		return nil, err
	}

	v := &Variable{}
	err = req.Do(ctx, v)
	if err != nil {
		return nil, err
	}

	return v, err
}

// Create is used to create a new variable.
func (s *testVariables) Create(ctx context.Context, moduleID RegistryModuleID, options VariableCreateOptions) (*Variable, error) {
	if err := moduleID.valid(); err != nil {
		return nil, err
	}
	if err := options.valid(); err != nil {
		return nil, err
	}

	req, err := s.client.NewRequest("POST", testVarsPath(moduleID), &options)
	if err != nil {
		return nil, err
	}

	v := &Variable{}
	err = req.Do(ctx, v)
	if err != nil {
		return nil, err
	}

	return v, nil
}

// Update values of an existing variable.
func (s *testVariables) Update(ctx context.Context, moduleID RegistryModuleID, variableID string, options VariableUpdateOptions) (*Variable, error) {
	if err := moduleID.valid(); err != nil {
		return nil, err
	}
	if !validStringID(&variableID) {
		return nil, ErrInvalidVariableID
	}

	u := fmt.Sprintf("%s/%s", testVarsPath(moduleID), url.PathEscape(variableID))
	req, err := s.client.NewRequest("PATCH", u, &options)
	if err != nil {
		return nil, err
	}

	v := &Variable{}
	err = req.Do(ctx, v)
	if err != nil {
		return nil, err
	}

	return v, nil
}

// Delete a variable by its ID.
func (s *testVariables) Delete(ctx context.Context, moduleID RegistryModuleID, variableID string) error {
	if err := moduleID.valid(); err != nil {
		return err
	}
	if !validStringID(&variableID) {
		return ErrInvalidVariableID
	}

	u := fmt.Sprintf("%s/%s", testVarsPath(moduleID), url.PathEscape(variableID))
	req, err := s.client.NewRequest("DELETE", u, nil)
	if err != nil {
		return err
	}

	return req.Do(ctx, nil)
}

func testVarsPath(moduleID RegistryModuleID) string {
	return fmt.Sprintf("organizations/%s/tests/registry-modules/%s/%s/%s/%s/vars",
		url.PathEscape(moduleID.Organization),
		url.PathEscape(string(moduleID.RegistryName)),
		url.PathEscape(moduleID.Namespace),
		url.PathEscape(moduleID.Name),
		url.PathEscape(moduleID.Provider))
}

const (
	_userAgent               = "go-tfe"
	_headerRateLimit         = "X-RateLimit-Limit"
	_headerRateReset         = "X-RateLimit-Reset"
	_headerAppName           = "TFP-AppName"
	_headerAPIVersion        = "TFP-API-Version"
	_headerTFEVersion        = "X-TFE-Version"
	_headerTFENumericVersion = "X-TFE-Current-Version"
	_includeQueryParam       = "include"

	DefaultAddress      = "https://app.terraform.io"
	DefaultBasePath     = "/api/v2/"
	DefaultRegistryPath = "/api/registry/"
	// PingEndpoint is a no-op API endpoint used to configure the rate limiter
	PingEndpoint       = "ping"
	ContentTypeJSONAPI = "application/vnd.api+json"
)

// RetryLogHook allows a function to run before each retry.

type RetryLogHook func(attemptNum int, resp *http.Response)

// Config provides configuration details to the API client.

type Config struct {
	// The address of the Terraform Enterprise API.
	Address string

	// The base path on which the API is served.
	BasePath string

	// The base path for the Registry API
	RegistryBasePath string

	// API token used to access the Terraform Enterprise API.
	Token string

	// Headers that will be added to every request.
	Headers http.Header

	// A custom HTTP client to use.
	HTTPClient *http.Client

	// RetryLogHook is invoked each time a request is retried.
	RetryLogHook RetryLogHook

	// RetryServerErrors enables the retry logic in the client.
	RetryServerErrors bool
}

// DefaultConfig returns a default config structure.

func DefaultConfig() *Config {
	config := &Config{
		Address:           os.Getenv("TFE_ADDRESS"),
		BasePath:          DefaultBasePath,
		RegistryBasePath:  DefaultRegistryPath,
		Token:             os.Getenv("TFE_TOKEN"),
		Headers:           make(http.Header),
		HTTPClient:        cleanhttp.DefaultPooledClient(),
		RetryServerErrors: false,
	}

	// Set the default address if none is given.
	if config.Address == "" {
		if host := os.Getenv("TFE_HOSTNAME"); host != "" {
			config.Address = fmt.Sprintf("https://%s", host)
		} else {
			config.Address = DefaultAddress
		}
	}

	// Set the default user agent.
	config.Headers.Set("User-Agent", _userAgent)

	return config
}

// Client is the Terraform Enterprise API client. It provides the basic
// connectivity and configuration for accessing the TFE API
type Client struct {
	baseURL                 *url.URL
	registryBaseURL         *url.URL
	token                   string
	headers                 http.Header
	http                    *retryablehttp.Client
	limiter                 *rate.Limiter
	retryLogHook            RetryLogHook
	retryServerErrors       bool
	remoteAPIVersion        string
	remoteTFEVersion        string
	remoteTFENumericVersion string
	appName                 string

	Admin                           Admin
	Agents                          Agents
	AgentPools                      AgentPools
	AgentTokens                     AgentTokens
	Applies                         Applies
	AuditTrails                     AuditTrails
	AWSOIDCConfigurations           AWSOIDCConfigurations
	GCPOIDCConfigurations           GCPOIDCConfigurations
	AzureOIDCConfigurations         AzureOIDCConfigurations
	VaultOIDCConfigurations         VaultOIDCConfigurations
	Comments                        Comments
	ConfigurationVersions           ConfigurationVersions
	CostEstimates                   CostEstimates
	Explorer                        Explorer
	GHAInstallations                GHAInstallations
	GPGKeys                         GPGKeys
	NotificationConfigurations      NotificationConfigurations
	OAuthClients                    OAuthClients
	OAuthTokens                     OAuthTokens
	OrganizationAuditConfigurations OrganizationAuditConfigurations
	OrganizationMemberships         OrganizationMemberships
	Organizations                   Organizations
	OrganizationTags                OrganizationTags
	OrganizationTokens              OrganizationTokens
	OrganizationTokenTTLPolicies    OrganizationTokenTTLPolicies
	Plans                           Plans
	PlanExports                     PlanExports
	Policies                        Policies
	PolicyChecks                    PolicyChecks
	PolicyEvaluations               PolicyEvaluations
	PolicySetOutcomes               PolicySetOutcomes
	PolicySetParameters             PolicySetParameters
	PolicySetVersions               PolicySetVersions
	PolicySets                      PolicySets
	ProviderSets                    ProviderSets
	QueryRuns                       QueryRuns
	RegistryModules                 RegistryModules
	RegistryNoCodeModules           RegistryNoCodeModules
	RegistryProviders               RegistryProviders
	RegistryProviderPlatforms       RegistryProviderPlatforms
	RegistryProviderVersions        RegistryProviderVersions
	RegistryComponents              RegistryComponents
	ReservedTagKeys                 ReservedTagKeys
	Runs                            Runs
	RunEvents                       RunEvents
	RunTasks                        RunTasks
	RunTasksIntegration             RunTasksIntegration
	RunTriggers                     RunTriggers
	SSHKeys                         SSHKeys
	Stacks                          Stacks
	HYOKConfigurations              HYOKConfigurations
	HYOKCustomerKeyVersions         HYOKCustomerKeyVersions
	HYOKEncryptedDataKeys           HYOKEncryptedDataKeys
	StackConfigurations             StackConfigurations
	StackConfigurationSummaries     StackConfigurationSummaries
	StackDeployments                StackDeployments
	StackDeploymentGroups           StackDeploymentGroups
	StackDeploymentGroupSummaries   StackDeploymentGroupSummaries
	StackDeploymentRuns             StackDeploymentRuns
	StackDeploymentSteps            StackDeploymentSteps
	StackDiagnostics                StackDiagnostics
	StackStates                     StackStates
	StateVersionOutputs             StateVersionOutputs
	StateVersions                   StateVersions
	TaskResults                     TaskResults
	TaskStages                      TaskStages
	Teams                           Teams
	TeamAccess                      TeamAccesses
	TeamMembers                     TeamMembers
	TeamProjectAccess               TeamProjectAccesses
	TeamTokens                      TeamTokens
	TestRuns                        TestRuns
	TestVariables                   TestVariables
	Users                           Users
	UserTokens                      UserTokens
	Variables                       Variables
	VariableSets                    VariableSets
	VariableSetVariables            VariableSetVariables
	Workspaces                      Workspaces
	WorkspaceResources              WorkspaceResources
	WorkspaceRunTasks               WorkspaceRunTasks
	Projects                        Projects
	TFPolicyEvaluationOutcomes      TFPolicyEvaluationOutcomes

	Meta Meta
}

// Admin is the the Terraform Enterprise Admin API. It provides access to site
// wide admin settings. These are only available for Terraform Enterprise and
// do not function against HCP Terraform
type Admin struct {
	Organizations     AdminOrganizations
	Workspaces        AdminWorkspaces
	Runs              AdminRuns
	TerraformVersions AdminTerraformVersions
	OPAVersions       AdminOPAVersions
	SentinelVersions  AdminSentinelVersions
	Users             AdminUsers
	Settings          *AdminSettings
}

// Meta contains any HCP Terraform APIs which provide data about the API itself.
type Meta struct {
	IPRanges IPRanges
}

// doForeignPUTRequest performs a PUT request using the specific data body. The Content-Type
// header is set to application/octet-stream but no Authentication header is sent. No response
// body is decoded.
func (c *Client) doForeignPUTRequest(ctx context.Context, foreignURL string, data io.Reader) error {
	u, err := url.Parse(foreignURL)
	if err != nil {
		return fmt.Errorf("specified URL was not valid: %w", err)
	}

	reqHeaders := make(http.Header)
	reqHeaders.Set("Accept", "application/json, */*")
	reqHeaders.Set("Content-Type", "application/octet-stream")

	req, err := retryablehttp.NewRequest("PUT", u.String(), data)
	if err != nil {
		return err
	}

	// Set the default headers.
	for k, v := range c.headers {
		req.Header[k] = v
	}

	// Set the request specific headers.
	for k, v := range reqHeaders {
		req.Header[k] = v
	}

	request := &ClientRequest{
		retryableRequest: req,
		http:             c.http,
		Header:           req.Header,
	}

	return request.DoJSON(ctx, nil)
}

// NewRequest performs some basic API request preparation based on the method
// specified. For GET requests, the reqBody is encoded as query parameters.
// For DELETE, PATCH, and POST requests, the request body is serialized as JSONAPI.
// For PUT requests, the request body is sent as a stream of bytes.
func (c *Client) NewRequest(method, path string, reqBody any) (*ClientRequest, error) {
	return c.NewRequestWithAdditionalQueryParams(method, path, reqBody, nil)
}

// NewRequestWithAdditionalQueryParams performs some basic API request
// preparation based on the method specified. For GET requests, the reqBody is
// encoded as query parameters. For DELETE, PATCH, and POST requests, the
// request body is serialized as JSONAPI. For PUT requests, the request body is
// sent as a stream of bytes. Additional query parameters can be added to the
// request as a string map. Note that if a key exists in both the reqBody and
// additionalQueryParams, the value in additionalQueryParams will be used.
func (c *Client) NewRequestWithAdditionalQueryParams(method, path string, reqBody any, additionalQueryParams map[string][]string) (*ClientRequest, error) {
	var u *url.URL
	var err error
	if strings.Contains(path, "/api/registry/") {
		u, err = c.registryBaseURL.Parse(path)
		if err != nil {
			return nil, err
		}
	} else {
		u, err = c.baseURL.Parse(path)
		if err != nil {
			return nil, err
		}
	}

	// Will contain combined query values from path parsing and
	// additionalQueryParams parameter
	q := make(url.Values)

	// Create a request specific headers map.
	reqHeaders := make(http.Header)
	reqHeaders.Set("Authorization", "Bearer "+c.token)

	var body any
	switch method {
	case "GET":
		reqHeaders.Set("Accept", ContentTypeJSONAPI)

		// Encode the reqBody as query parameters
		if reqBody != nil {
			q, err = query.Values(reqBody)
			if err != nil {
				return nil, err
			}
		}
	case "DELETE", "PATCH", "POST":
		reqHeaders.Set("Accept", ContentTypeJSONAPI)
		reqHeaders.Set("Content-Type", ContentTypeJSONAPI)

		if reqBody != nil {
			if body, err = serializeRequestBody(reqBody); err != nil {
				return nil, err
			}
		}
	case "PUT":
		reqHeaders.Set("Accept", "application/json")
		reqHeaders.Set("Content-Type", "application/octet-stream")
		body = reqBody
	}

	for k, v := range u.Query() {
		q[k] = v
	}
	for k, v := range additionalQueryParams {
		q[k] = v
	}

	u.RawQuery = encodeQueryParams(q)

	req, err := retryablehttp.NewRequest(method, u.String(), body)
	if err != nil {
		return nil, err
	}

	// Set the default headers.
	for k, v := range c.headers {
		req.Header[k] = v
	}

	// Set the request specific headers.
	for k, v := range reqHeaders {
		req.Header[k] = v
	}

	return &ClientRequest{
		retryableRequest: req,
		http:             c.http,
		limiter:          c.limiter,
		Header:           req.Header,
	}, nil
}

// NewClient creates a new Terraform Enterprise API client.
func NewClient(cfg *Config) (*Client, error) {
	config := DefaultConfig()

	// Layer in the provided config for any non-blank values.
	if cfg != nil { // nolint
		if cfg.Address != "" {
			config.Address = cfg.Address
		}
		if cfg.BasePath != "" {
			config.BasePath = cfg.BasePath
		}
		if cfg.RegistryBasePath != "" {
			config.RegistryBasePath = cfg.RegistryBasePath
		}
		if cfg.Token != "" {
			config.Token = cfg.Token
		}
		for k, v := range cfg.Headers {
			config.Headers[k] = v
		}
		if cfg.HTTPClient != nil {
			config.HTTPClient = cfg.HTTPClient
		}
		if cfg.RetryLogHook != nil {
			config.RetryLogHook = cfg.RetryLogHook
		}
		config.RetryServerErrors = cfg.RetryServerErrors
	}

	// Parse the address to make sure its a valid URL.
	baseURL, err := url.Parse(config.Address)
	if err != nil {
		return nil, fmt.Errorf("invalid address: %w", err)
	}

	baseURL.Path = config.BasePath
	if !strings.HasSuffix(baseURL.Path, "/") {
		baseURL.Path += "/"
	}

	registryURL, err := url.Parse(config.Address)
	if err != nil {
		return nil, fmt.Errorf("invalid address: %w", err)
	}

	registryURL.Path = config.RegistryBasePath
	if !strings.HasSuffix(registryURL.Path, "/") {
		registryURL.Path += "/"
	}

	// This value must be provided by the user.
	if config.Token == "" {
		return nil, fmt.Errorf("missing API token")
	}

	// Create the client.
	client := &Client{
		baseURL:           baseURL,
		registryBaseURL:   registryURL,
		token:             config.Token,
		headers:           config.Headers,
		retryLogHook:      config.RetryLogHook,
		retryServerErrors: config.RetryServerErrors,
	}

	client.http = &retryablehttp.Client{
		Backoff:      client.retryHTTPBackoff,
		CheckRetry:   client.retryHTTPCheck,
		ErrorHandler: retryablehttp.PassthroughErrorHandler,
		HTTPClient:   config.HTTPClient,
		RetryWaitMin: 100 * time.Millisecond,
		RetryWaitMax: 400 * time.Millisecond,
		RetryMax:     30,
	}

	meta, err := client.getRawAPIMetadata()
	if err != nil {
		return nil, err
	}

	// Configure the rate limiter.
	client.configureLimiter(meta.RateLimit)

	// Save the API version so we can return it from the RemoteAPIVersion
	// method later.
	client.remoteAPIVersion = meta.APIVersion

	// Save the TFE version
	client.remoteTFEVersion = meta.TFEVersion

	// Save the TFE Numeric version
	client.remoteTFENumericVersion = meta.TFENumericVersion

	// Save the app name
	client.appName = meta.AppName

	// Create Admin
	client.Admin = Admin{
		Organizations:     &adminOrganizations{client: client},
		Workspaces:        &adminWorkspaces{client: client},
		Runs:              &adminRuns{client: client},
		Settings:          newAdminSettings(client),
		TerraformVersions: &adminTerraformVersions{client: client},
		OPAVersions:       &adminOPAVersions{client: client},
		SentinelVersions:  &adminSentinelVersions{client: client},
		Users:             &adminUsers{client: client},
	}

	// Create the services.
	client.AgentPools = &agentPools{client: client}
	client.Agents = &agents{client: client}
	client.AgentTokens = &agentTokens{client: client}
	client.Applies = &applies{client: client}
	client.AuditTrails = &auditTrails{client: client}
	client.AWSOIDCConfigurations = &awsOIDCConfigurations{client: client}
	client.GCPOIDCConfigurations = &gcpOIDCConfigurations{client: client}
	client.AzureOIDCConfigurations = &azureOIDCConfigurations{client: client}
	client.VaultOIDCConfigurations = &vaultOIDCConfigurations{client: client}
	client.Comments = &comments{client: client}
	client.ConfigurationVersions = &configurationVersions{client: client}
	client.CostEstimates = &costEstimates{client: client}
	client.Explorer = &explorer{client: client}
	client.GHAInstallations = &gHAInstallations{client: client}
	client.GPGKeys = &gpgKeys{client: client}
	client.RegistryNoCodeModules = &registryNoCodeModules{client: client}
	client.NotificationConfigurations = &notificationConfigurations{client: client}
	client.OAuthClients = &oAuthClients{client: client}
	client.OAuthTokens = &oAuthTokens{client: client}
	client.OrganizationMemberships = &organizationMemberships{client: client}
	client.Organizations = &organizations{client: client}
	client.OrganizationTags = &organizationTags{client: client}
	client.OrganizationTokens = &organizationTokens{client: client}
	client.OrganizationTokenTTLPolicies = &organizationTokenTTLPolicies{client: client}
	client.OrganizationAuditConfigurations = &organizationAuditConfigurations{client: client}
	client.PlanExports = &planExports{client: client}
	client.Plans = &plans{client: client}
	client.Policies = &policies{client: client}
	client.PolicyChecks = &policyChecks{client: client}
	client.PolicyEvaluations = &policyEvaluation{client: client}
	client.PolicySetOutcomes = &policySetOutcome{client: client}
	client.PolicySetParameters = &policySetParameters{client: client}
	client.PolicySets = &policySets{client: client}
	client.PolicySetVersions = &policySetVersions{client: client}
	client.ProviderSets = &providerSets{client: client}
	client.Projects = &projects{client: client}
	client.QueryRuns = &queryRuns{client: client}
	client.RegistryModules = &registryModules{client: client}
	client.RegistryProviderPlatforms = &registryProviderPlatforms{client: client}
	client.RegistryProviders = &registryProviders{client: client}
	client.RegistryProviderVersions = &registryProviderVersions{client: client}
	client.RegistryComponents = &registryComponents{client: client}
	client.ReservedTagKeys = &reservedTagKeys{client: client}
	client.Runs = &runs{client: client}
	client.RunEvents = &runEvents{client: client}
	client.RunTasks = &runTasks{client: client}
	client.RunTasksIntegration = &runTaskIntegration{client: client}
	client.RunTriggers = &runTriggers{client: client}
	client.SSHKeys = &sshKeys{client: client}
	client.Stacks = &stacks{client: client}
	client.HYOKConfigurations = &hyokConfigurations{client: client}
	client.HYOKCustomerKeyVersions = &hyokCustomerKeyVersions{client: client}
	client.HYOKEncryptedDataKeys = &hyokEncryptedDataKeys{client: client}
	client.StackConfigurations = &stackConfigurations{client: client}
	client.StackConfigurationSummaries = &stackConfigurationSummaries{client: client}
	client.StackDeployments = &stackDeployments{client: client}
	client.StackDeploymentGroups = &stackDeploymentGroups{client: client}
	client.StackDeploymentGroupSummaries = &stackDeploymentGroupSummaries{client: client}
	client.StackDeploymentRuns = &stackDeploymentRuns{client: client}
	client.StackDeploymentSteps = &stackDeploymentSteps{client: client}
	client.StackDiagnostics = &stackDiagnostics{client: client}
	client.StackStates = &stackStates{client: client}
	client.StateVersionOutputs = &stateVersionOutputs{client: client}
	client.StateVersions = &stateVersions{client: client}
	client.TaskResults = &taskResults{client: client}
	client.TaskStages = &taskStages{client: client}
	client.TeamAccess = &teamAccesses{client: client}
	client.TeamMembers = &teamMembers{client: client}
	client.TeamProjectAccess = &teamProjectAccesses{client: client}
	client.Teams = &teams{client: client}
	client.TeamTokens = &teamTokens{client: client}
	client.TestRuns = &testRuns{client: client}
	client.TestVariables = &testVariables{client: client}
	client.Users = &users{client: client}
	client.UserTokens = &userTokens{client: client}
	client.Variables = &variables{client: client}
	client.VariableSets = &variableSets{client: client}
	client.VariableSetVariables = &variableSetVariables{client: client}
	client.WorkspaceRunTasks = &workspaceRunTasks{client: client}
	client.Workspaces = &workspaces{client: client}
	client.WorkspaceResources = &workspaceResources{client: client}
	client.TFPolicyEvaluationOutcomes = &tfPolicyEvaluationOutcomes{client: client}

	client.Meta = Meta{
		IPRanges: &ipRanges{client: client},
	}

	client.StackDeploymentRuns = &stackDeploymentRuns{client: client}

	return client, nil
}

// AppName returns the name of the instance.
func (c Client) AppName() string {
	return c.appName
}

// IsCloud returns true if the client is configured against a HCP Terraform
// instance.
//
// Whether an instance is HCP Terraform or Terraform Enterprise is derived from the TFP-AppName header.
func (c Client) IsCloud() bool {
	return c.appName == "HCP Terraform"
}

// IsEnterprise returns true if the client is configured against a Terraform
// Enterprise instance.
//
// Whether an instance is HCP Terraform or TFE is derived from the TFP-AppName header. Note:
// not all TFE releases include this header in API responses.
func (c Client) IsEnterprise() bool {
	return !c.IsCloud()
}

// RemoteAPIVersion returns the server's declared API version string.
//
// A HCP Terraform or Enterprise API server returns its API version in an
// HTTP header field in all responses. The NewClient function saves the
// version number returned in its initial setup request and RemoteAPIVersion
// returns that cached value.
//
// The API protocol calls for this string to be a dotted-decimal version number
// like 2.3.0, where the first number indicates the API major version while the
// second indicates a minor version which may have introduced some
// backward-compatible additional features compared to its predecessor.
//
// Explicit API versioning was added to the HCP Terraform and Enterprise
// APIs as a later addition, so older servers will not return version
// information. In that case, this function returns an empty string as the
// version.
func (c Client) RemoteAPIVersion() string {
	return c.remoteAPIVersion
}

// BaseURL returns the base URL as configured in the client
func (c Client) BaseURL() url.URL {
	return *c.baseURL
}

// BaseRegistryURL returns the registry base URL as configured in the client
func (c Client) BaseRegistryURL() url.URL {
	return *c.registryBaseURL
}

// SetFakeRemoteAPIVersion allows setting a given string as the client's remoteAPIVersion,
// overriding the value pulled from the API header during client initialization.
//
// This is intended for use in tests, when you may want to configure your TFE client to
// return something different than the actual API version in order to test error handling.
func (c *Client) SetFakeRemoteAPIVersion(fakeAPIVersion string) {
	c.remoteAPIVersion = fakeAPIVersion
}

// RemoteTFEVersion returns the server's declared TFE monthly version string.
//
// A Terraform Enterprise API server includes its current version in an
// HTTP header field in all responses. This value is saved by the client
// during the initial setup request and RemoteTFEVersion returns that cached
// value. This function returns an empty string for any Terraform Enterprise version
// earlier than v202208-3 and for HCP Terraform.
func (c Client) RemoteTFEVersion() string {
	return c.remoteTFEVersion
}

// RemoteTFENumericVersion returns the server's declared TFE version string.
//
// A Terraform Enterprise API server includes its current numeric version in an
// HTTP header field in all responses. This value is saved by the client
// during the initial setup request and RemoteTFENumericVersion returns that cached
// value. This function returns an empty string for any Terraform Enterprise version
// earlier than 1.0.3 and for HCP Terraform.
func (c Client) RemoteTFENumericVersion() string {
	return c.remoteTFENumericVersion
}

// RetryServerErrors configures the retry HTTP check to also retry
// unexpected errors or requests that failed with a server error.
func (c *Client) RetryServerErrors(retry bool) {
	c.retryServerErrors = retry
}

// retryHTTPCheck provides a callback for Client.CheckRetry which
// will retry both rate limit (429) and server (>= 500) errors.
func (c *Client) retryHTTPCheck(ctx context.Context, resp *http.Response, err error) (bool, error) {
	if ctx.Err() != nil {
		return false, ctx.Err()
	}
	if err != nil {
		return c.retryServerErrors, err
	}
	if resp.StatusCode == 429 || (c.retryServerErrors && resp.StatusCode >= 500) {
		return true, nil
	}
	return false, nil
}

// retryHTTPBackoff provides a generic callback for Client.Backoff which
// will pass through all calls based on the status code of the response.
func (c *Client) retryHTTPBackoff(minimum, maximum time.Duration, attemptNum int, resp *http.Response) time.Duration {
	if c.retryLogHook != nil {
		c.retryLogHook(attemptNum, resp)
	}

	// Use the rate limit backoff function when we are rate limited.
	if resp != nil && resp.StatusCode == 429 {
		return rateLimitBackoff(minimum, maximum, resp)
	}

	// Set custom duration's when we experience a service interruption.
	minimum = 700 * time.Millisecond
	maximum = 900 * time.Millisecond

	return retryablehttp.LinearJitterBackoff(minimum, maximum, attemptNum, resp)
}

// rateLimitBackoff provides a callback for Client.Backoff which will use the
// X-RateLimit_Reset header to determine the time to wait. We add some jitter
// to prevent a thundering herd.
//
// minimum and maximum are mainly used for bounding the jitter that will be added to
// the reset time retrieved from the headers. But if the final wait time is
// less than minimum, minimum will be used instead.
func rateLimitBackoff(minimum, maximum time.Duration, resp *http.Response) time.Duration {
	// rnd is used to generate pseudo-random numbers.
	rnd := rand.New(rand.NewSource(time.Now().UnixNano()))

	// First create some jitter bounded by the min and max durations.
	jitter := time.Duration(rnd.Float64() * float64(maximum-minimum))

	if resp != nil && resp.Header.Get(_headerRateReset) != "" {
		v := resp.Header.Get(_headerRateReset)
		reset, err := strconv.ParseFloat(v, 64)
		if err != nil {
			log.Fatal(err)
		}
		// Only update min if the given time to wait is longer
		if reset > 0 && time.Duration(reset*1e9) > minimum {
			minimum = time.Duration(reset * 1e9)
		}
	}

	return minimum + jitter
}

type rawAPIMetadata struct {
	// APIVersion is the raw API version string reported by the server in the
	// TFP-API-Version response header, or an empty string if that header
	// field was not included in the response.
	APIVersion string

	// TFEVersion is the raw TFE monthly version string reported by the server in the
	// X-TFE-Version response header, or an empty string if that header
	// field was not included in the response.
	TFEVersion string

	// TFENumericVersion is the raw TFE Numeric version string reported by the server in the
	// X-TFE-Current-Version response header, or an empty string if that header
	// field was not included in the response.
	TFENumericVersion string

	// RateLimit is the raw API version string reported by the server in the
	// X-RateLimit-Limit response header, or an empty string if that header
	// field was not included in the response.
	RateLimit string

	// AppName is either 'HCP Terraform' or 'Terraform Enterprise'
	AppName string
}

func (c *Client) getRawAPIMetadata() (rawAPIMetadata, error) {
	var meta rawAPIMetadata

	// Create a new request.
	u, err := c.baseURL.Parse(PingEndpoint)
	if err != nil {
		return meta, err
	}
	req, err := http.NewRequest("GET", u.String(), nil)
	if err != nil {
		return meta, err
	}

	// Attach the default headers.
	for k, v := range c.headers {
		req.Header[k] = v
	}
	req.Header.Set("Accept", ContentTypeJSONAPI)
	req.Header.Set("Authorization", "Bearer "+c.token)

	// Make a single request to retrieve the rate limit headers.
	resp, err := c.http.HTTPClient.Do(req)
	if err != nil {
		return meta, err
	}
	resp.Body.Close() //nolint:errcheck

	meta.APIVersion = resp.Header.Get(_headerAPIVersion)
	meta.RateLimit = resp.Header.Get(_headerRateLimit)
	meta.TFEVersion = resp.Header.Get(_headerTFEVersion)
	meta.TFENumericVersion = resp.Header.Get(_headerTFENumericVersion)
	meta.AppName = resp.Header.Get(_headerAppName)

	return meta, nil
}

// configureLimiter configures the rate limiter.
func (c *Client) configureLimiter(rawLimit string) {
	// Set default values for when rate limiting is disabled.
	limit := rate.Inf
	burst := 0

	if v := rawLimit; v != "" {
		if rateLimit, err := strconv.ParseFloat(v, 64); rateLimit > 0 {
			if err != nil {
				log.Fatal(err)
			}
			// Configure the limit and burst using a split of 2/3 for the limit and
			// 1/3 for the burst. This enables clients to burst 1/3 of the allowed
			// calls before the limiter kicks in. The remaining calls will then be
			// spread out evenly using intervals of time.Second / limit which should
			// prevent hitting the rate limit.
			limit = rate.Limit(rateLimit * 0.66)
			burst = int(rateLimit * 0.33)
		}
	}

	// Create a new limiter using the calculated values.
	c.limiter = rate.NewLimiter(limit, burst)
}

// encodeQueryParams encodes the values into "URL encoded" form
// ("bar=baz&foo=quux") sorted by key. This version behaves as url.Values
// Encode, except that it encodes certain keys as comma-separated values instead
// of using multiple keys.
func encodeQueryParams(v url.Values) string {
	if v == nil {
		return ""
	}
	var buf strings.Builder
	keys := make([]string, 0, len(v))
	for k := range v {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		vs := v[k]
		if len(vs) > 1 && validSliceKey(k) {
			val := strings.Join(vs, ",")
			vs = vs[:0]
			vs = append(vs, val)
		}
		keyEscaped := url.QueryEscape(k)

		for _, v := range vs {
			if buf.Len() > 0 {
				buf.WriteByte('&')
			}
			buf.WriteString(keyEscaped)
			buf.WriteByte('=')
			buf.WriteString(url.QueryEscape(v))
		}
	}
	return buf.String()
}

// decodeQueryParams types an object and converts the struct fields into
// Query Parameters, which can be used with NewRequestWithAdditionalQueryParams
// Note that a field without a `url` annotation will be converted into a query
// parameter. Use url:"-" to ignore struct fields.
func decodeQueryParams(v any) (url.Values, error) {
	if v == nil {
		return make(url.Values, 0), nil
	}
	return query.Values(v)
}

// serializeRequestBody serializes the given ptr or ptr slice into a JSON
// request. It automatically uses jsonapi or json serialization, depending
// on the body type's tags.
func serializeRequestBody(v interface{}) (interface{}, error) {
	// The body can be a slice of pointers or a pointer. In either
	// case we want to choose the serialization type based on the
	// individual record type. To determine that type, we need
	// to either follow the pointer or examine the slice element type.
	// There are other theoretical possibilities (e. g. maps,
	// non-pointers) but they wouldn't work anyway because the
	// json-api library doesn't support serializing other things.
	var modelType reflect.Type
	bodyType := reflect.TypeOf(v)
	switch bodyType.Kind() {
	case reflect.Slice:
		sliceElem := bodyType.Elem()
		if sliceElem.Kind() != reflect.Pointer {
			return nil, ErrInvalidRequestBody
		}
		modelType = sliceElem.Elem()
	case reflect.Pointer:
		modelType = reflect.ValueOf(v).Elem().Type()
	default:
		return nil, ErrInvalidRequestBody
	}

	// Infer whether the request uses jsonapi or regular json
	// serialization based on how the fields are tagged.
	jsonAPIFields := 0
	jsonFields := 0
	for i := 0; i < modelType.NumField(); i++ {
		structField := modelType.Field(i)
		if structField.Tag.Get("jsonapi") != "" {
			jsonAPIFields++
		}
		if structField.Tag.Get("json") != "" {
			jsonFields++
		}
	}
	if jsonAPIFields > 0 && jsonFields > 0 {
		// Defining a struct with both json and jsonapi tags doesn't
		// make sense, because a struct can only be serialized
		// as one or another. If this does happen, it's a bug
		// in the library that should be fixed at development time
		return nil, ErrInvalidStructFormat
	}

	if jsonFields > 0 {
		return json.Marshal(v)
	}
	buf := bytes.NewBuffer(nil)
	if err := jsonapi.MarshalPayloadWithoutIncluded(buf, v); err != nil {
		return nil, err
	}
	return buf, nil
}

func unmarshalResponse(responseBody io.Reader, model interface{}) error {
	// Get the value of model so we can test if it's a struct.
	dst := reflect.Indirect(reflect.ValueOf(model))

	// Return an error if model is not a struct or an io.Writer.
	if dst.Kind() != reflect.Struct {
		return fmt.Errorf("%v must be a struct or an io.Writer", dst)
	}

	// Try to get the Items and Pagination struct fields.
	items := dst.FieldByName("Items")

	// Unmarshal a single value if model does not contain the
	// Items and Pagination struct fields.
	if !items.IsValid() {
		return jsonapi.UnmarshalPayload(responseBody, model)
	}

	// Return an error if model.Items is not a slice.
	if items.Type().Kind() != reflect.Slice {
		return ErrItemsMustBeSlice
	}

	// Create a temporary buffer and copy all the read data into it.
	body := bytes.NewBuffer(nil)
	reader := io.TeeReader(responseBody, body)

	// Unmarshal as a list of values as model.Items is a slice.
	raw, err := jsonapi.UnmarshalManyPayload(reader, items.Type().Elem())
	if err != nil {
		return err
	}

	// Make a new slice to hold the results.
	sliceType := reflect.SliceOf(items.Type().Elem())
	result := reflect.MakeSlice(sliceType, 0, len(raw))

	// Add all of the results to the new slice.
	for _, v := range raw {
		result = reflect.Append(result, reflect.ValueOf(v))
	}

	// Pointer-swap the result.
	items.Set(result)

	pagination := dst.FieldByName("Pagination")
	paginationWithoutTotals := dst.FieldByName("PaginationNextPrev")

	// As we are getting a list of values, we need to decode
	// the pagination details out of the response body.
	// Pointer-swap the decoded pagination details.
	if paginationWithoutTotals.IsValid() {
		p, err := parsePaginationWithoutTotal(body)
		if err != nil {
			return err
		}
		paginationWithoutTotals.Set(reflect.ValueOf(p))
	} else if pagination.IsValid() {
		p, err := parsePagination(body)
		if err != nil {
			return err
		}
		pagination.Set(reflect.ValueOf(p))
	}

	return nil
}

// ListOptions is used to specify pagination options when making API requests.
// Pagination allows breaking up large result sets into chunks, or "pages".
type ListOptions struct {
	// The page number to request. The results vary based on the PageSize.
	PageNumber int `url:"page[number],omitempty"`

	// The number of elements returned in a single page.
	PageSize int `url:"page[size],omitempty"`
}

// PaginationNextPrev is used to return the pagination details of an API request.
type PaginationNextPrev struct {
	CurrentPage  int `json:"current-page"`
	PreviousPage int `json:"prev-page"`
	NextPage     int `json:"next-page"`
}

// Pagination is used to return the pagination details of an API request including TotalCount.
type Pagination struct {
	CurrentPage  int `json:"current-page"`
	PreviousPage int `json:"prev-page"`
	NextPage     int `json:"next-page"`
	TotalCount   int `json:"total-count"`
	TotalPages   int `json:"total-pages"`
}

func parsePaginationWithoutTotal(body io.Reader) (*PaginationNextPrev, error) {
	var raw struct {
		Meta struct {
			Pagination PaginationNextPrev `jsonapi:"pagination"`
		} `jsonapi:"meta"`
	}

	// JSON decode the raw response.
	if err := json.NewDecoder(body).Decode(&raw); err != nil {
		return &PaginationNextPrev{}, err
	}

	return &raw.Meta.Pagination, nil
}

func parsePagination(body io.Reader) (*Pagination, error) {
	var raw struct {
		Meta struct {
			Pagination Pagination `jsonapi:"pagination"`
		} `jsonapi:"meta"`
	}

	// JSON decode the raw response.
	if err := json.NewDecoder(body).Decode(&raw); err != nil {
		return &Pagination{}, err
	}

	return &raw.Meta.Pagination, nil
}

// checkResponseCode refines typical API errors into more specific errors
// if possible. It returns nil if the response code < 400
func checkResponseCode(r *http.Response) error {
	if r.StatusCode >= 200 && r.StatusCode <= 399 {
		return nil
	}

	var errs []string
	var err error

	switch r.StatusCode {
	case 400:
		errs, err = decodeErrorPayload(r)
		if err != nil {
			return err
		}

		if errorPayloadContains(errs, "include parameter") {
			return ErrInvalidIncludeValue
		}
		return errors.New(strings.Join(errs, "\n"))
	case 401:
		return ErrUnauthorized
	case 404:
		return ErrResourceNotFound
	case 409:
		switch {
		case strings.HasSuffix(r.Request.URL.Path, "actions/lock"):
			return ErrWorkspaceLocked
		case strings.HasSuffix(r.Request.URL.Path, "actions/unlock"):
			errs, err = decodeErrorPayload(r)
			if err != nil {
				return err
			}

			if errorPayloadContains(errs, "is locked by Run") {
				return ErrWorkspaceLockedByRun
			}

			if errorPayloadContains(errs, "is locked by Team") {
				return ErrWorkspaceLockedByTeam
			}

			if errorPayloadContains(errs, "is locked by User") {
				return ErrWorkspaceLockedByUser
			}

			return ErrWorkspaceNotLocked
		case strings.HasSuffix(r.Request.URL.Path, "actions/force-unlock"):
			return ErrWorkspaceNotLocked
		case strings.HasSuffix(r.Request.URL.Path, "actions/safe-delete"):
			errs, err = decodeErrorPayload(r)
			if err != nil {
				return err
			}
			if errorPayloadContains(errs, "locked") {
				return ErrWorkspaceLockedCannotDelete
			}
			if errorPayloadContains(errs, "being processed") {
				return ErrWorkspaceStillProcessing
			}

			return ErrWorkspaceNotSafeToDelete
		}
	}

	errs, err = decodeErrorPayload(r)
	if err != nil {
		return err
	}

	return errors.New(strings.Join(errs, "\n"))
}

func decodeErrorPayload(r *http.Response) ([]string, error) {
	// Decode the error payload.
	var errs []string
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return errs, errors.New(r.Status)
	}

	// attempt JSON:API error payloads unwrapping
	errPayload := &jsonapi.ErrorsPayload{}
	if err := json.Unmarshal(body, errPayload); err == nil && len(errPayload.Errors) > 0 {
		for _, e := range errPayload.Errors {
			if e.Detail == "" {
				errs = append(errs, e.Title)
			} else {
				errs = append(errs, fmt.Sprintf("%s\n\n%s", e.Title, e.Detail))
			}
		}
		return errs, nil
	}

	// attempt JSON error payloads unwrapping: like {"errors":["..."]}.
	var rawErrs struct {
		Errors []string `json:"errors"`
	}
	if err := json.Unmarshal(body, &rawErrs); err == nil && len(rawErrs.Errors) > 0 {
		return rawErrs.Errors, nil
	}

	return errs, errors.New(r.Status)
}

func errorPayloadContains(payloadErrors []string, match string) bool {
	for _, e := range payloadErrors {
		if strings.Contains(e, match) {
			return true
		}
	}
	return false
}

func packContents(path string) (*bytes.Buffer, error) {
	body := bytes.NewBuffer(nil)

	file, err := os.Stat(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return body, fmt.Errorf(`failed to find files under the path "%v": %w`, path, err)
		}
		return body, fmt.Errorf(`unable to upload files from the path "%v": %w`, path, err)
	}

	if !file.Mode().IsDir() {
		return body, ErrMissingDirectory
	}

	_, errSlug := slug.Pack(path, body, true)
	if errSlug != nil {
		return body, errSlug
	}

	return body, nil
}

func validSliceKey(key string) bool {
	return key == _includeQueryParam || strings.Contains(key, "filter[")
}

// NOTE: TFpolicy is still a beta feature and is subject to change.

var _ TFPolicyEvaluationOutcomes = (*tfPolicyEvaluationOutcomes)(nil)

type TFPolicyEvaluationStatus string

const (
	TFPolicyEvaluationStatusPending          TFPolicyEvaluationStatus = "pending"
	TFPolicyEvaluationStatusQueued           TFPolicyEvaluationStatus = "queued"
	TFPolicyEvaluationStatusRunning          TFPolicyEvaluationStatus = "running"
	TFPolicyEvaluationStatusAwaitingOverride TFPolicyEvaluationStatus = "awaiting_override"
	TFPolicyEvaluationStatusPassed           TFPolicyEvaluationStatus = "passed"
	TFPolicyEvaluationStatusFailed           TFPolicyEvaluationStatus = "failed"
	TFPolicyEvaluationStatusOverridden       TFPolicyEvaluationStatus = "overridden"
	TFPolicyEvaluationStatusErrored          TFPolicyEvaluationStatus = "errored"
	TFPolicyEvaluationStatusCanceled         TFPolicyEvaluationStatus = "canceled"
	TFPolicyEvaluationStatusUnreachable      TFPolicyEvaluationStatus = "unreachable"
)

type TFPolicyEvaluationStageType string

const (
	TFPolicyEvaluationStageTypeInit  TFPolicyEvaluationStageType = "Init"
	TFPolicyEvaluationStageTypePlan  TFPolicyEvaluationStageType = "Plan"
	TFPolicyEvaluationStageTypeApply TFPolicyEvaluationStageType = "Apply"
)

type TFPolicyEvaluationStatusTimestamps struct {
	PendingAt          time.Time `jsonapi:"attr,pending-at,rfc3339"`
	QueuedAt           time.Time `jsonapi:"attr,queued-at,rfc3339"`
	RunningAt          time.Time `jsonapi:"attr,running-at,rfc3339"`
	AwaitingOverrideAt time.Time `jsonapi:"attr,awaiting-override-at,rfc3339"`
	PassedAt           time.Time `jsonapi:"attr,passed-at,rfc3339"`
	FailedAt           time.Time `jsonapi:"attr,failed-at,rfc3339"`
	OverriddenAt       time.Time `jsonapi:"attr,overridden-at,rfc3339"`
	ErroredAt          time.Time `jsonapi:"attr,errored-at,rfc3339"`
	CanceledAt         time.Time `jsonapi:"attr,canceled-at,rfc3339"`
}

type TFPolicyEvaluationResultCount struct {
	AdvisoryFailed  int `jsonapi:"attr,advisory-failed"`
	MandatoryFailed int `jsonapi:"attr,mandatory-failed"`
	Passed          int `jsonapi:"attr,passed"`
	Errored         int `jsonapi:"attr,errored"`
	Unknown         int `jsonapi:"attr,unknown"`
}

type TFPolicyEvaluationErrorType string

const (
	TFPolicyEvaluationErrorTypeSetupError               TFPolicyEvaluationErrorType = "setup_error"
	TFPolicyEvaluationErrorTypeIncompatibleAgentVersion TFPolicyEvaluationErrorType = "incompatible_agent_version"
)

type TFPolicyEvaluationError struct {
	Type    TFPolicyEvaluationErrorType `jsonapi:"attr,type"`
	Summary string                      `jsonapi:"attr,summary"`
	Detail  string                      `jsonapi:"attr,detail"`
}

type TFPolicyEvaluationPermissions struct {
	CanOverride bool `jsonapi:"attr,can-override"`
}

type TFPolicyEvaluationActions struct {
	IsOverridable bool `jsonapi:"attr,is-overridable"`
}

type TFPolicyEvaluation struct {
	ID               string                              `jsonapi:"primary,tf-policy-evaluations"`
	Status           TFPolicyEvaluationStatus            `jsonapi:"attr,status"`
	StageType        TFPolicyEvaluationStageType         `jsonapi:"attr,stage-type"`
	StatusTimestamps *TFPolicyEvaluationStatusTimestamps `jsonapi:"attr,status-timestamps"`
	ResultCount      *TFPolicyEvaluationResultCount      `jsonapi:"attr,result-count"`
	Error            *TFPolicyEvaluationError            `jsonapi:"attr,error,omitempty"`
	OrganizedLog     bool                                `jsonapi:"attr,organized-log"`
	Permissions      *TFPolicyEvaluationPermissions      `jsonapi:"attr,permissions,omitempty"`
	Actions          *TFPolicyEvaluationActions          `jsonapi:"attr,actions,omitempty"`

	// Relations
	Run                        *Run                  `jsonapi:"relation,run,omitempty"`
	TFPolicyEvaluationOutcomes []*TFPolicySetOutcome `jsonapi:"relation,outcomes,omitempty"`
	// Links
	Links map[string]interface{} `jsonapi:"links,omitempty"`
}

type TFPolicyEvaluationOutcomes interface {
	List(ctx context.Context, tfPolicyEvaluationID string, options *TFPolicyEvaluationListOptions) (*TFPolicyEvaluationOutcomeList, error)
}

type TFPolicyEvaluationOutcomeEnforcementLevel string

const (
	TFPolicyEvaluationOutcomeEnforcementLevelAdvisory             TFPolicyEvaluationOutcomeEnforcementLevel = "advisory"
	TFPolicyEvaluationOutcomeEnforcementLevelMandatory            TFPolicyEvaluationOutcomeEnforcementLevel = "mandatory"
	TFPolicyEvaluationOutcomeEnforcementLevelMandatoryOverridable TFPolicyEvaluationOutcomeEnforcementLevel = "mandatory_overridable"
)

type TFPolicyEvaluationOutcomeStatus string

type TFPolicySetOutcomeDiagnostic struct {
	Code         string                        `jsonapi:"attr,code"`
	Context      string                        `jsonapi:"attr,context"`
	StartLine    int                           `jsonapi:"attr,start_line"`
	Summary      string                        `jsonapi:"attr,summary"`
	Resources    []*TFPolicySetOutcomeResource `jsonapi:"attr,resources,omitempty"`
	ErrorMessage string                        `jsonapi:"attr,error_message,omitempty"`
}

type TFPolicySetOutcomeResource struct {
	ResourceName string   `jsonapi:"attr,resource_name"`
	ErrorMessage string   `jsonapi:"attr,error_message,omitempty"`
	InfoMessage  string   `jsonapi:"attr,info_message"`
	InfoMessages []string `jsonapi:"attr,info_messages,omitempty"`
	Code         string   `jsonapi:"attr,code,omitempty"`
	FileName     string   `jsonapi:"attr,file_name,omitempty"`
	StartLine    int      `jsonapi:"attr,start_line,omitempty"`
	Context      string   `jsonapi:"attr,context,omitempty"`
	Start        int      `jsonapi:"attr,start,omitempty"`
	Values       []*struct {
		Traversal string `jsonapi:"attr,traversal"`
		Statement string `jsonapi:"attr,statement"`
	} `jsonapi:"attr,values,omitempty"`
}

type TFPolicySetPolicyOutcome struct {
	EnforcementLevel TFPolicyEvaluationOutcomeEnforcementLevel `jsonapi:"attr,enforcement_level"`
	Status           string                                    `jsonapi:"attr,status"`
	Description      string                                    `jsonapi:"attr,description"`
	FileName         string                                    `jsonapi:"attr,file_name"`
	PolicyName       string                                    `jsonapi:"attr,policy_name"`
	Diagnostics      []*TFPolicySetOutcomeDiagnostic           `jsonapi:"attr,diagnostics,omitempty"`
	PassedResources  []*TFPolicySetOutcomeResource             `jsonapi:"attr,passed_resources,omitempty"`
}

type TFPolicySetOutcome struct {
	ID                   string                         `jsonapi:"primary,tf-policy-set-outcomes"`
	Outcomes             []*TFPolicySetPolicyOutcome    `jsonapi:"attr,outcomes,omitempty"`
	Error                *TFPolicyEvaluationError       `jsonapi:"attr,error,omitempty"`
	Overridable          bool                           `jsonapi:"attr,overridable"`
	PolicySetName        string                         `jsonapi:"attr,policy-set-name"`
	PolicySetDescription string                         `jsonapi:"attr,policy-set-description"`
	ResultCount          *TFPolicyEvaluationResultCount `jsonapi:"attr,result-count,omitempty"`

	// Relations
	TFPolicyEvaluation *TFPolicyEvaluation `jsonapi:"relation,tf-policy-evaluation,omitempty"`
}

type TFPolicyEvaluationListOptions struct {
	ListOptions

	Status           string `url:"filter[status],omitempty"`
	EnforcementLevel string `url:"filter[enforcement-level],omitempty"`
}

type TFPolicyEvaluationOutcomeList struct {
	*Pagination
	Items []*TFPolicySetOutcome
}

type tfPolicyEvaluationOutcomes struct {
	client *Client
}

func (s *tfPolicyEvaluationOutcomes) List(ctx context.Context, tfPolicyEvaluationID string, options *TFPolicyEvaluationListOptions) (*TFPolicyEvaluationOutcomeList, error) {
	if !validStringID(&tfPolicyEvaluationID) {
		return nil, ErrInvalidTFPolicyEvaluationID
	}

	u := fmt.Sprintf("tf-policy-evaluations/%s/tf-policy-set-outcomes", url.PathEscape(tfPolicyEvaluationID))
	req, err := s.client.NewRequest("GET", u, options)
	if err != nil {
		return nil, err
	}

	tfpo := &TFPolicyEvaluationOutcomeList{}
	err = req.Do(ctx, tfpo)
	if err != nil {
		return nil, err
	}

	return tfpo, nil
}

// Access returns a pointer to the given team access type.
func Access(v AccessType) *AccessType {
	return &v
}

func AgentExecutionModePtr(v AgentExecutionMode) *AgentExecutionMode {
	return &v
}

// ProjectAccess returns a pointer to the given team access project type.
func ProjectAccess(v TeamProjectAccessType) *TeamProjectAccessType {
	return &v
}

// ProjectSettingsPermission returns a pointer to the given team access project type.
func ProjectSettingsPermission(v ProjectSettingsPermissionType) *ProjectSettingsPermissionType {
	return &v
}

// ProjectTeamsPermission returns a pointer to the given team access project type.
func ProjectTeamsPermission(v ProjectTeamsPermissionType) *ProjectTeamsPermissionType {
	return &v
}

// ProjectVariableSetsPermission returns a pointer to the given team access project type.
func ProjectVariableSetsPermission(v ProjectVariableSetsPermissionType) *ProjectVariableSetsPermissionType {
	return &v
}

// WorkspaceRunsPermission returns a pointer to the given team access project type.
func WorkspaceRunsPermission(v WorkspaceRunsPermissionType) *WorkspaceRunsPermissionType {
	return &v
}

// WorkspaceSentinelMocksPermission returns a pointer to the given team access project type.
func WorkspaceSentinelMocksPermission(v WorkspaceSentinelMocksPermissionType) *WorkspaceSentinelMocksPermissionType {
	return &v
}

// WorkspaceStateVersionsPermission returns a pointer to the given team access project type.
func WorkspaceStateVersionsPermission(v WorkspaceStateVersionsPermissionType) *WorkspaceStateVersionsPermissionType {
	return &v
}

// WorkspaceStateVersionsPermission returns a pointer to the given team access project type.
func WorkspaceVariablesPermission(v WorkspaceVariablesPermissionType) *WorkspaceVariablesPermissionType {
	return &v
}

// RunsPermission returns a pointer to the given team runs permission type.
func RunsPermission(v RunsPermissionType) *RunsPermissionType {
	return &v
}

// VariablesPermission returns a pointer to the given team variables permission type.
func VariablesPermission(v VariablesPermissionType) *VariablesPermissionType {
	return &v
}

// StateVersionsPermission returns a pointer to the given team state versions permission type.
func StateVersionsPermission(v StateVersionsPermissionType) *StateVersionsPermissionType {
	return &v
}

// SentinelMocksPermission returns a pointer to the given team Sentinel mocks permission type.
func SentinelMocksPermission(v SentinelMocksPermissionType) *SentinelMocksPermissionType {
	return &v
}

// AuthPolicy returns a pointer to the given authentication poliy.
func AuthPolicy(v AuthPolicyType) *AuthPolicyType {
	return &v
}

// Bool returns a pointer to the given bool
func Bool(v bool) *bool {
	return &v
}

// Category returns a pointer to the given category type.
func Category(v CategoryType) *CategoryType {
	return &v
}

// EnforcementMode returns a pointer to the given enforcement level.
func EnforcementMode(v EnforcementLevel) *EnforcementLevel {
	return &v
}

// Int returns a pointer to the given int.
func Int(v int) *int {
	return &v
}

// Int64 returns a pointer to the given int64.
func Int64(v int64) *int64 {
	return &v
}

// NotificationDestination returns a pointer to the given notification configuration destination type
func NotificationDestination(v NotificationDestinationType) *NotificationDestinationType {
	return &v
}

// PlanExportType returns a pointer to the given plan export data type.
func PlanExportType(v PlanExportDataType) *PlanExportDataType {
	return &v
}

// ServiceProvider returns a pointer to the given service provider type.
func ServiceProvider(v ServiceProviderType) *ServiceProviderType {
	return &v
}

// SMTPAuthValue returns a pointer to a given smtp auth type.
func SMTPAuthValue(v SMTPAuthType) *SMTPAuthType {
	return &v
}

// String returns a pointer to the given string.
func String(v string) *string {
	return &v
}

// SAMLProvider returns a pointer to the given SAML provider type.
func SAMLProvider(v SAMLProviderType) *SAMLProviderType {
	return &v
}

func NullableBool(v bool) jsonapi.NullableAttr[bool] {
	return jsonapi.NewNullableAttrWithValue[bool](v)
}

func NullBool() jsonapi.NullableAttr[bool] {
	return jsonapi.NewNullNullableAttr[bool]()
}

func NullableTime(v time.Time) jsonapi.NullableAttr[time.Time] {
	return jsonapi.NewNullableAttrWithValue[time.Time](v)
}

func NullTime() jsonapi.NullableAttr[time.Time] {
	return jsonapi.NewNullNullableAttr[time.Time]()
}

// NullableString returns a NullableAttr wrapping the given string value.
func NullableString(v string) jsonapi.NullableAttr[string] {
	return jsonapi.NewNullableAttrWithValue[string](v)
}

// NullString returns a NullableAttr that explicitly serializes as null.
func NullString() jsonapi.NullableAttr[string] {
	return jsonapi.NewNullNullableAttr[string]()
}

// Ptr returns a pointer to the given value of any type.
func Ptr[T any](v T) *T {
	return &v
}

// Compile-time proof of interface implementation.
var _ UserTokens = (*userTokens)(nil)

// UserTokens describes all the user token related methods that the
// HCP Terraform and Terraform Enterprise API supports.
//
// TFE API docs:
// https://developer.hashicorp.com/terraform/cloud-docs/api-docs/user-tokens
type UserTokens interface {
	// List all the tokens of the given user ID.
	List(ctx context.Context, userID string) (*UserTokenList, error)

	// Create a new user token
	Create(ctx context.Context, userID string, options UserTokenCreateOptions) (*UserToken, error)

	// Read a user token by its ID.
	Read(ctx context.Context, tokenID string) (*UserToken, error)

	// Delete a user token by its ID.
	Delete(ctx context.Context, tokenID string) error
}

// userTokens implements UserTokens.
type userTokens struct {
	client *Client
}

// UserTokenList is a list of tokens for the given user ID.
type UserTokenList struct {
	*Pagination
	Items []*UserToken
}

// CreatedByChoice is a choice type struct that represents the possible values
// within a polymorphic relation. If a value is available, exactly one field
// will be non-nil.
type CreatedByChoice struct {
	Organization *Organization
	Team         *Team
	User         *User
}

// UserToken represents a Terraform Enterprise user token.
type UserToken struct {
	ID          string           `jsonapi:"primary,authentication-tokens"`
	CreatedAt   time.Time        `jsonapi:"attr,created-at,iso8601"`
	Description string           `jsonapi:"attr,description"`
	LastUsedAt  time.Time        `jsonapi:"attr,last-used-at,iso8601"`
	Token       string           `jsonapi:"attr,token"`
	ExpiredAt   time.Time        `jsonapi:"attr,expired-at,iso8601"`
	CreatedBy   *CreatedByChoice `jsonapi:"polyrelation,created-by"`
}

// UserTokenCreateOptions contains the options for creating a user token.
type UserTokenCreateOptions struct {
	Description string `jsonapi:"attr,description,omitempty"`
	// Optional: The token's expiration date.
	// This feature is available in TFE release v202305-1 and later
	ExpiredAt *time.Time `jsonapi:"attr,expired-at,iso8601,omitempty"`
}

// Create a new user token
func (s *userTokens) Create(ctx context.Context, userID string, options UserTokenCreateOptions) (*UserToken, error) {
	if !validStringID(&userID) {
		return nil, ErrInvalidUserID
	}

	u := fmt.Sprintf("users/%s/authentication-tokens", url.PathEscape(userID))
	req, err := s.client.NewRequest("POST", u, &options)
	if err != nil {
		return nil, err
	}

	ut := &UserToken{}
	err = req.Do(ctx, ut)
	if err != nil {
		return nil, err
	}

	return ut, err
}

// List shows existing user tokens
func (s *userTokens) List(ctx context.Context, userID string) (*UserTokenList, error) {
	if !validStringID(&userID) {
		return nil, ErrInvalidUserID
	}

	u := fmt.Sprintf("users/%s/authentication-tokens", url.PathEscape(userID))
	req, err := s.client.NewRequest("GET", u, nil)
	if err != nil {
		return nil, err
	}

	tl := &UserTokenList{}
	err = req.Do(ctx, tl)
	if err != nil {
		return nil, err
	}

	return tl, err
}

// Read a user token by its ID.
func (s *userTokens) Read(ctx context.Context, tokenID string) (*UserToken, error) {
	if !validStringID(&tokenID) {
		return nil, ErrInvalidTokenID
	}

	u := fmt.Sprintf(AuthenticationTokensPath, url.PathEscape(tokenID))
	req, err := s.client.NewRequest("GET", u, nil)
	if err != nil {
		return nil, err
	}

	tt := &UserToken{}
	err = req.Do(ctx, tt)
	if err != nil {
		return nil, err
	}

	return tt, err
}

// Delete a user token by its ID.
func (s *userTokens) Delete(ctx context.Context, tokenID string) error {
	if !validStringID(&tokenID) {
		return ErrInvalidTokenID
	}

	u := fmt.Sprintf(AuthenticationTokensPath, url.PathEscape(tokenID))
	req, err := s.client.NewRequest("DELETE", u, nil)
	if err != nil {
		return err
	}

	return req.Do(ctx, nil)
}

// Compile-time proof of interface implementation.
var _ Users = (*users)(nil)

// Users describes all the user related methods that the Terraform
// Enterprise API supports.
//
// TFE API docs: https://developer.hashicorp.com/terraform/cloud-docs/api-docs/account
type Users interface {
	// ReadCurrent reads the details of the currently authenticated user.
	ReadCurrent(ctx context.Context) (*User, error)

	// UpdateCurrent updates attributes of the currently authenticated user.
	UpdateCurrent(ctx context.Context, options UserUpdateOptions) (*User, error)
}

// users implements Users.
type users struct {
	client *Client
}

// User represents a Terraform Enterprise user.
type User struct {
	ID               string     `jsonapi:"primary,users"`
	AvatarURL        string     `jsonapi:"attr,avatar-url"`
	Email            string     `jsonapi:"attr,email"`
	IsServiceAccount bool       `jsonapi:"attr,is-service-account"`
	TwoFactor        *TwoFactor `jsonapi:"attr,two-factor"`
	UnconfirmedEmail string     `jsonapi:"attr,unconfirmed-email"`
	Username         string     `jsonapi:"attr,username"`
	V2Only           bool       `jsonapi:"attr,v2-only"`
	// Deprecated: IsSiteAdmin was deprecated in v202406 and will be removed in a future version of Terraform Enterprise
	IsSiteAdmin *bool            `jsonapi:"attr,is-site-admin"`
	IsAdmin     *bool            `jsonapi:"attr,is-admin"`
	IsSsoLogin  *bool            `jsonapi:"attr,is-sso-login"`
	Permissions *UserPermissions `jsonapi:"attr,permissions"`

	// Relations
	// AuthenticationTokens *AuthenticationTokens `jsonapi:"relation,authentication-tokens"`

	// SCIM Attributes
	IsSCIMManaged *bool      `jsonapi:"attr,is-scim-managed"`
	SCIMUsername  *string    `jsonapi:"attr,scim-username"`
	SCIMUpdatedAt *time.Time `jsonapi:"attr,scim-updated-at,iso8601"`
}

// UserPermissions represents the user permissions.
type UserPermissions struct {
	CanCreateOrganizations bool `jsonapi:"attr,can-create-organizations"`
	CanChangeEmail         bool `jsonapi:"attr,can-change-email"`
	CanChangeUsername      bool `jsonapi:"attr,can-change-username"`
	CanManageUserTokens    bool `jsonapi:"attr,can-manage-user-tokens"`
	CanView2FaSettings     bool `jsonapi:"attr,can-view2fa-settings"`
	CanManageHcpAccount    bool `jsonapi:"attr,can-manage-hcp-account"`
}

// TwoFactor represents the organization permissions.
type TwoFactor struct {
	Enabled  bool `jsonapi:"attr,enabled"`
	Verified bool `jsonapi:"attr,verified"`
}

// UserUpdateOptions represents the options for updating a user.
type UserUpdateOptions struct {
	// Type is a public field utilized by JSON:API to
	// set the resource type via the field tag.
	// It is not a user-defined value and does not need to be set.
	// https://jsonapi.org/format/#crud-creating
	Type string `jsonapi:"primary,users"`

	// Optional: New username.
	Username *string `jsonapi:"attr,username,omitempty"`

	// Optional: New email address (must be consumed afterwards to take effect).
	Email *string `jsonapi:"attr,email,omitempty"`
}

// ReadCurrent reads the details of the currently authenticated user.
func (s *users) ReadCurrent(ctx context.Context) (*User, error) {
	req, err := s.client.NewRequest("GET", "account/details", nil)
	if err != nil {
		return nil, err
	}

	u := &User{}
	err = req.Do(ctx, u)
	if err != nil {
		return nil, err
	}

	return u, nil
}

// UpdateCurrent updates attributes of the currently authenticated user.
func (s *users) UpdateCurrent(ctx context.Context, options UserUpdateOptions) (*User, error) {
	req, err := s.client.NewRequest("PATCH", "account/update", &options)
	if err != nil {
		return nil, err
	}

	u := &User{}
	err = req.Do(ctx, u)
	if err != nil {
		return nil, err
	}

	return u, nil
}

// A regular expression used to validate common string ID patterns.
var reStringID = regexp.MustCompile(`^[^/\s]+$`)

// validEmail checks if the given input is a correct email
func validEmail(v string) bool {
	_, err := mail.ParseAddress(v)
	return err == nil
}

// validString checks if the given input is present and non-empty.
func validString(v *string) bool {
	return v != nil && *v != ""
}

// validStringID checks if the given string pointer is non-nil and
// contains a typical string identifier.
func validStringID(v *string) bool {
	return v != nil && reStringID.MatchString(*v)
}

// validVersion checks if the given input is a valid version.
func validVersion(v string) bool {
	_, err := version.NewVersion(v)
	return err == nil
}

// Compile-time proof of interface implementation.
var _ VariableSetVariables = (*variableSetVariables)(nil)

// VariableSetVariables describes all variable variable related methods within the scope of
// Variable Sets that the Terraform Enterprise API supports
//
// TFE API docs: https://developer.hashicorp.com/terraform/cloud-docs/api-docs/variable-sets#variable-relationships
type VariableSetVariables interface {
	// List all variables in the variable set.
	List(ctx context.Context, variableSetID string, options *VariableSetVariableListOptions) (*VariableSetVariableList, error)

	// Create is used to create a new variable within a given variable set
	Create(ctx context.Context, variableSetID string, options *VariableSetVariableCreateOptions) (*VariableSetVariable, error)

	// Read a variable by its ID
	Read(ctx context.Context, variableSetID string, variableID string) (*VariableSetVariable, error)

	// Update valuse of an existing variable
	Update(ctx context.Context, variableSetID string, variableID string, options *VariableSetVariableUpdateOptions) (*VariableSetVariable, error)

	// Delete a variable by its ID
	Delete(ctx context.Context, variableSetID string, variableID string) error
}

type variableSetVariables struct {
	client *Client
}

type VariableSetVariableList struct {
	*Pagination
	Items []*VariableSetVariable
}

type VariableSetVariable struct {
	ID          string       `jsonapi:"primary,vars"`
	Key         string       `jsonapi:"attr,key"`
	Value       string       `jsonapi:"attr,value"`
	Description string       `jsonapi:"attr,description"`
	Category    CategoryType `jsonapi:"attr,category"`
	HCL         bool         `jsonapi:"attr,hcl"`
	Sensitive   bool         `jsonapi:"attr,sensitive"`
	VersionID   string       `jsonapi:"attr,version-id"`

	// Relations
	VariableSet *VariableSet `jsonapi:"relation,varset"`
}

type VariableSetVariableListOptions struct {
	ListOptions
}

func (o VariableSetVariableListOptions) valid() error {
	return nil
}

// List all variables associated with the given variable set.
func (s *variableSetVariables) List(ctx context.Context, variableSetID string, options *VariableSetVariableListOptions) (*VariableSetVariableList, error) {
	if !validStringID(&variableSetID) {
		return nil, ErrInvalidVariableSetID
	}
	if options != nil {
		if err := options.valid(); err != nil {
			return nil, err
		}
	}

	u := fmt.Sprintf("varsets/%s/relationships/vars", url.PathEscape(variableSetID))
	req, err := s.client.NewRequest("GET", u, options)
	if err != nil {
		return nil, err
	}

	vl := &VariableSetVariableList{}
	err = req.Do(ctx, vl)
	if err != nil {
		return nil, err
	}

	return vl, nil
}

// VariableSetVariableCreatOptions represents the options for creating a new variable within a variable set
type VariableSetVariableCreateOptions struct {
	// Type is a public field utilized by JSON:API to
	// set the resource type via the field tag.
	// It is not a user-defined value and does not need to be set.
	// https://jsonapi.org/format/#crud-creating
	Type string `jsonapi:"primary,vars"`

	// The name of the variable.
	Key *string `jsonapi:"attr,key"`

	// The value of the variable.
	Value *string `jsonapi:"attr,value,omitempty"`

	// The description of the variable.
	Description *string `jsonapi:"attr,description,omitempty"`

	// Whether this is a Terraform or environment variable.
	Category *CategoryType `jsonapi:"attr,category"`

	// Whether to evaluate the value of the variable as a string of HCL code.
	HCL *bool `jsonapi:"attr,hcl,omitempty"`

	// Whether the value is sensitive.
	Sensitive *bool `jsonapi:"attr,sensitive,omitempty"`
}

func (o VariableSetVariableCreateOptions) valid() error {
	if !validString(o.Key) {
		return ErrRequiredKey
	}
	if o.Category == nil {
		return ErrRequiredCategory
	}
	return nil
}

// Create is used to create a new variable.
func (s *variableSetVariables) Create(ctx context.Context, variableSetID string, options *VariableSetVariableCreateOptions) (*VariableSetVariable, error) {
	if !validStringID(&variableSetID) {
		return nil, ErrInvalidVariableSetID
	}
	if options != nil {
		if err := options.valid(); err != nil {
			return nil, err
		}
	}

	u := fmt.Sprintf("varsets/%s/relationships/vars", url.PathEscape(variableSetID))
	req, err := s.client.NewRequest("POST", u, options)
	if err != nil {
		return nil, err
	}

	v := &VariableSetVariable{}
	err = req.Do(ctx, v)
	if err != nil {
		return nil, err
	}

	return v, nil
}

// Read a variable by its ID.
func (s *variableSetVariables) Read(ctx context.Context, variableSetID, variableID string) (*VariableSetVariable, error) {
	if !validStringID(&variableSetID) {
		return nil, ErrInvalidVariableSetID
	}
	if !validStringID(&variableID) {
		return nil, ErrInvalidVariableID
	}

	u := fmt.Sprintf("varsets/%s/relationships/vars/%s", url.PathEscape(variableSetID), url.PathEscape(variableID))
	req, err := s.client.NewRequest("GET", u, nil)

	if err != nil {
		return nil, err
	}

	v := &VariableSetVariable{}
	err = req.Do(ctx, v)
	if err != nil {
		return nil, err
	}

	return v, err
}

// VariableSetVariableUpdateOptions represents the options for updating a variable.
type VariableSetVariableUpdateOptions struct {
	// Type is a public field utilized by JSON:API to
	// set the resource type via the field tag.
	// It is not a user-defined value and does not need to be set.
	// https://jsonapi.org/format/#crud-creating
	Type string `jsonapi:"primary,vars"`

	// The name of the variable.
	Key *string `jsonapi:"attr,key,omitempty"`

	// The value of the variable.
	Value *string `jsonapi:"attr,value,omitempty"`

	// The description of the variable.
	Description *string `jsonapi:"attr,description,omitempty"`

	// Whether to evaluate the value of the variable as a string of HCL code.
	HCL *bool `jsonapi:"attr,hcl,omitempty"`

	// Whether the value is sensitive.
	Sensitive *bool `jsonapi:"attr,sensitive,omitempty"`
}

// Update values of an existing variable.
func (s *variableSetVariables) Update(ctx context.Context, variableSetID, variableID string, options *VariableSetVariableUpdateOptions) (*VariableSetVariable, error) {
	if !validStringID(&variableSetID) {
		return nil, ErrInvalidVariableSetID
	}
	if !validStringID(&variableID) {
		return nil, ErrInvalidVariableID
	}

	u := fmt.Sprintf("varsets/%s/relationships/vars/%s", url.PathEscape(variableSetID), url.PathEscape(variableID))
	req, err := s.client.NewRequest("PATCH", u, options)
	if err != nil {
		return nil, err
	}

	v := &VariableSetVariable{}
	err = req.Do(ctx, v)
	if err != nil {
		return nil, err
	}

	return v, nil
}

// Delete a variable by its ID.
func (s *variableSetVariables) Delete(ctx context.Context, variableSetID, variableID string) error {
	if !validStringID(&variableSetID) {
		return ErrInvalidVariableSetID
	}
	if !validStringID(&variableID) {
		return ErrInvalidVariableID
	}

	u := fmt.Sprintf("varsets/%s/relationships/vars/%s", url.PathEscape(variableSetID), url.PathEscape(variableID))
	req, err := s.client.NewRequest("DELETE", u, nil)
	if err != nil {
		return err
	}

	return req.Do(ctx, nil)
}

// Compile-time proof of interface implementation.
var _ VariableSets = (*variableSets)(nil)

// VariableSets describes all the Variable Set related methods that the
// Terraform Enterprise API supports.
//
// TFE API docs: https://developer.hashicorp.com/terraform/cloud-docs/api-docs/variable-sets
type VariableSets interface {
	// List all the variable sets within an organization.
	List(ctx context.Context, organization string, options *VariableSetListOptions) (*VariableSetList, error)

	// ListForWorkspace gets the associated variable sets for a workspace.
	ListForWorkspace(ctx context.Context, workspaceID string, options *VariableSetListOptions) (*VariableSetList, error)

	// ListForProject gets the associated variable sets for a project.
	ListForProject(ctx context.Context, projectID string, options *VariableSetListOptions) (*VariableSetList, error)

	// Create is used to create a new variable set.
	Create(ctx context.Context, organization string, options *VariableSetCreateOptions) (*VariableSet, error)

	// Read a variable set by its ID.
	Read(ctx context.Context, variableSetID string, options *VariableSetReadOptions) (*VariableSet, error)

	// Update an existing variable set.
	Update(ctx context.Context, variableSetID string, options *VariableSetUpdateOptions) (*VariableSet, error)

	// Delete a variable set by ID.
	Delete(ctx context.Context, variableSetID string) error

	// Apply variable set to workspaces in the supplied list.
	ApplyToWorkspaces(ctx context.Context, variableSetID string, options *VariableSetApplyToWorkspacesOptions) error

	// Remove variable set from workspaces in the supplied list.
	RemoveFromWorkspaces(ctx context.Context, variableSetID string, options *VariableSetRemoveFromWorkspacesOptions) error

	// Apply variable set to projects in the supplied list.
	ApplyToProjects(ctx context.Context, variableSetID string, options VariableSetApplyToProjectsOptions) error

	// Remove variable set from projects in the supplied list.
	RemoveFromProjects(ctx context.Context, variableSetID string, options VariableSetRemoveFromProjectsOptions) error

	// Apply variable set to stacks in the supplied list.
	ApplyToStacks(ctx context.Context, variableSetID string, options *VariableSetApplyToStacksOptions) error

	// Remove variable set from stacks in the supplied list.
	RemoveFromStacks(ctx context.Context, variableSetID string, options *VariableSetRemoveFromStacksOptions) error

	// Update list of workspaces to which the variable set is applied to match the supplied list.
	UpdateWorkspaces(ctx context.Context, variableSetID string, options *VariableSetUpdateWorkspacesOptions) (*VariableSet, error)

	// Update list of stacks to which the variable set is applied to match the supplied list.
	UpdateStacks(ctx context.Context, variableSetID string, options *VariableSetUpdateStacksOptions) (*VariableSet, error)
}

// variableSets implements VariableSets.
type variableSets struct {
	client *Client
}

// VariableSetList represents a list of variable sets.
type VariableSetList struct {
	*Pagination
	Items []*VariableSet
}

// Parent represents the variable set's parent (currently only organizations and projects are supported).
// This relation is considered BETA, SUBJECT TO CHANGE, and likely unavailable to most users.
type Parent struct {
	Organization *Organization
	Project      *Project
}

// VariableSet represents a Terraform Enterprise variable set.
type VariableSet struct {
	ID          string `jsonapi:"primary,varsets"`
	Name        string `jsonapi:"attr,name"`
	Description string `jsonapi:"attr,description"`
	Global      bool   `jsonapi:"attr,global"`
	Priority    bool   `jsonapi:"attr,priority"`

	// Relations
	Organization *Organization `jsonapi:"relation,organization"`
	// Optional: Parent represents the variable set's parent (currently only organizations and projects are supported).
	// This relation is considered BETA, SUBJECT TO CHANGE, and likely unavailable to most users.
	Parent     *Parent                `jsonapi:"polyrelation,parent"`
	Workspaces []*Workspace           `jsonapi:"relation,workspaces,omitempty"`
	Projects   []*Project             `jsonapi:"relation,projects,omitempty"`
	Stacks     []*Stack               `jsonapi:"relation,stacks,omitempty"`
	Variables  []*VariableSetVariable `jsonapi:"relation,vars,omitempty"`
}

// A list of relations to include. See available resources
// https://developer.hashicorp.com/terraform/enterprise/api-docs/admin/organizations#available-related-resources
type VariableSetIncludeOpt string

const (
	VariableSetWorkspaces VariableSetIncludeOpt = "workspaces"
	VariableSetProjects   VariableSetIncludeOpt = "projects"
	VariableSetStacks     VariableSetIncludeOpt = "stacks"
	VariableSetVars       VariableSetIncludeOpt = "vars"
)

// VariableSetListOptions represents the options for listing variable sets.
type VariableSetListOptions struct {
	ListOptions
	Include string `url:"include"`

	// Optional: A query string used to filter variable sets.
	// Any variable sets with a name partially matching this value will be returned.
	Query string `url:"q,omitempty"`
}

// VariableSetCreateOptions represents the options for creating a new variable set within in a organization.
type VariableSetCreateOptions struct {
	// Type is a public field utilized by JSON:API to
	// set the resource type via the field tag.
	// It is not a user-defined value and does not need to be set.
	// https://jsonapi.org/format/#crud-creating
	Type string `jsonapi:"primary,varsets"`

	// The name of the variable set.
	// Affects variable precedence when there are conflicts between Variable Sets
	// https://developer.hashicorp.com/terraform/cloud-docs/api-docs/variable-sets#apply-variable-set-to-workspaces
	Name *string `jsonapi:"attr,name"`

	// A description to provide context for the variable set.
	Description *string `jsonapi:"attr,description,omitempty"`

	// If true the variable set is considered in all runs in the organization.
	Global *bool `jsonapi:"attr,global,omitempty"`

	// If true the variables in the set override any other variable values set
	// in a more specific scope including values set on the command line.
	Priority *bool `jsonapi:"attr,priority,omitempty"`

	// Optional: Parent represents the variable set's parent (currently only organizations and projects are supported).
	// This relation is considered BETA, SUBJECT TO CHANGE, and likely unavailable to most users.
	Parent *Parent `jsonapi:"polyrelation,parent"`
}

// VariableSetReadOptions represents the options for reading variable sets.
type VariableSetReadOptions struct {
	Include *[]VariableSetIncludeOpt `url:"include,omitempty"`
}

// VariableSetUpdateOptions represents the options for updating a variable set.
type VariableSetUpdateOptions struct {
	// Type is a public field utilized by JSON:API to
	// set the resource type via the field tag.
	// It is not a user-defined value and does not need to be set.
	// https://jsonapi.org/format/#crud-creating
	Type string `jsonapi:"primary,varsets"`

	// The name of the variable set.
	// Affects variable precedence when there are conflicts between Variable Sets
	// https://developer.hashicorp.com/terraform/cloud-docs/api-docs/variable-sets#apply-variable-set-to-workspaces
	Name *string `jsonapi:"attr,name,omitempty"`

	// A description to provide context for the variable set.
	Description *string `jsonapi:"attr,description,omitempty"`

	// If true the variable set is considered in all runs in the organization.
	Global *bool `jsonapi:"attr,global,omitempty"`

	// If true the variables in the set override any other variable values set
	// in a more specific scope including values set on the command line.
	Priority *bool `jsonapi:"attr,priority,omitempty"`
}

// VariableSetApplyToWorkspacesOptions represents the options for applying variable sets to workspaces.
type VariableSetApplyToWorkspacesOptions struct {
	// The workspaces to apply the variable set to (additive).
	Workspaces []*Workspace
}

// VariableSetRemoveFromWorkspacesOptions represents the options for removing variable sets from workspaces.
type VariableSetRemoveFromWorkspacesOptions struct {
	// The workspaces to remove the variable set from.
	Workspaces []*Workspace
}

// VariableSetApplyToProjectsOptions represents the options for applying variable sets to projects.
type VariableSetApplyToProjectsOptions struct {
	// The projects to apply the variable set to (additive).
	Projects []*Project
}

// VariableSetApplyToStacksOptions represents the options for applying variable sets to stacks.
type VariableSetApplyToStacksOptions struct {
	// The stacks to apply the variable set to (additive).
	Stacks []*Stack
}

// VariableSetRemoveFromProjectsOptions represents the options for removing variable sets from projects.
type VariableSetRemoveFromProjectsOptions struct {
	// The projects to remove the variable set from.
	Projects []*Project
}

// VariableSetRemoveFromStacksOptions represents the options for removing variable sets from stacks.
type VariableSetRemoveFromStacksOptions struct {
	// The stacks to remove the variable set from.
	Stacks []*Stack
}

// VariableSetUpdateWorkspacesOptions represents a subset of update options specifically for applying variable sets to workspaces
type VariableSetUpdateWorkspacesOptions struct {
	// Type is a public field utilized by JSON:API to
	// set the resource type via the field tag.
	// It is not a user-defined value and does not need to be set.
	// https://jsonapi.org/format/#crud-creating
	Type string `jsonapi:"primary,varsets"`

	// The workspaces to be applied to. An empty set means remove all applied
	Workspaces []*Workspace `jsonapi:"relation,workspaces"`
}

// VariableSetUpdateStacksOptions represents a subset of update options specifically for applying variable sets to stacks
type VariableSetUpdateStacksOptions struct {
	// Type is a public field utilized by JSON:API to
	// set the resource type via the field tag.
	// It is not a user-defined value and does not need to be set.
	// https://jsonapi.org/format/#crud-creating
	Type string `jsonapi:"primary,varsets"`

	// The stacks to be applied to. An empty set means remove all applied
	Stacks []*Stack `jsonapi:"relation,stacks"`
}

type privateVariableSetUpdateWorkspacesOptions struct {
	Type       string       `jsonapi:"primary,varsets"`
	Global     bool         `jsonapi:"attr,global"`
	Workspaces []*Workspace `jsonapi:"relation,workspaces"`
}

type privateVariableSetUpdateStacksOptions struct {
	Type   string   `jsonapi:"primary,varsets"`
	Global bool     `jsonapi:"attr,global"`
	Stacks []*Stack `jsonapi:"relation,stacks"`
}

// List all Variable Sets in the organization
func (s *variableSets) List(ctx context.Context, organization string, options *VariableSetListOptions) (*VariableSetList, error) {
	if !validStringID(&organization) {
		return nil, ErrInvalidOrg
	}
	if options != nil {
		if err := options.valid(); err != nil {
			return nil, err
		}
	}

	u := fmt.Sprintf("organizations/%s/varsets", url.PathEscape(organization))
	req, err := s.client.NewRequest("GET", u, options)
	if err != nil {
		return nil, err
	}

	vl := &VariableSetList{}
	err = req.Do(ctx, vl)
	if err != nil {
		return nil, err
	}

	return vl, nil
}

// ListForWorkspace gets the associated variable sets for a workspace.
func (s *variableSets) ListForWorkspace(ctx context.Context, workspaceID string, options *VariableSetListOptions) (*VariableSetList, error) {
	if !validStringID(&workspaceID) {
		return nil, ErrInvalidWorkspaceID
	}
	if options != nil {
		if err := options.valid(); err != nil {
			return nil, err
		}
	}

	u := fmt.Sprintf("workspaces/%s/varsets", url.PathEscape(workspaceID))
	req, err := s.client.NewRequest("GET", u, options)
	if err != nil {
		return nil, err
	}

	vl := &VariableSetList{}
	err = req.Do(ctx, vl)
	if err != nil {
		return nil, err
	}

	return vl, nil
}

// ListForProject gets the associated variable sets for a project.
func (s *variableSets) ListForProject(ctx context.Context, projectID string, options *VariableSetListOptions) (*VariableSetList, error) {
	if !validStringID(&projectID) {
		return nil, ErrInvalidProjectID
	}
	if options != nil {
		if err := options.valid(); err != nil {
			return nil, err
		}
	}

	u := fmt.Sprintf("projects/%s/varsets", url.PathEscape(projectID))
	req, err := s.client.NewRequest("GET", u, options)
	if err != nil {
		return nil, err
	}

	vl := &VariableSetList{}
	err = req.Do(ctx, vl)
	if err != nil {
		return nil, err
	}

	return vl, nil
}

// Create is used to create a new variable set.
func (s *variableSets) Create(ctx context.Context, organization string, options *VariableSetCreateOptions) (*VariableSet, error) {
	if !validStringID(&organization) {
		return nil, ErrInvalidOrg
	}
	if err := options.valid(); err != nil {
		return nil, err
	}

	u := fmt.Sprintf("organizations/%s/varsets", url.PathEscape(organization))
	req, err := s.client.NewRequest("POST", u, options)
	if err != nil {
		return nil, err
	}

	vl := &VariableSet{}
	err = req.Do(ctx, vl)
	if err != nil {
		return nil, err
	}

	return vl, nil
}

// Read is used to inspect a given variable set based on ID
func (s *variableSets) Read(ctx context.Context, variableSetID string, options *VariableSetReadOptions) (*VariableSet, error) {
	if !validStringID(&variableSetID) {
		return nil, ErrInvalidVariableSetID
	}

	u := fmt.Sprintf("varsets/%s", url.PathEscape(variableSetID))
	req, err := s.client.NewRequest("GET", u, options)
	if err != nil {
		return nil, err
	}

	vs := &VariableSet{}
	err = req.Do(ctx, vs)
	if err != nil {
		return nil, err
	}

	return vs, err
}

// Update an existing variable set.
func (s *variableSets) Update(ctx context.Context, variableSetID string, options *VariableSetUpdateOptions) (*VariableSet, error) {
	if !validStringID(&variableSetID) {
		return nil, ErrInvalidVariableSetID
	}

	u := fmt.Sprintf("varsets/%s", url.PathEscape(variableSetID))
	req, err := s.client.NewRequest("PATCH", u, options)
	if err != nil {
		return nil, err
	}

	v := &VariableSet{}
	err = req.Do(ctx, v)
	if err != nil {
		return nil, err
	}

	return v, nil
}

// Delete a variable set by its ID.
func (s *variableSets) Delete(ctx context.Context, variableSetID string) error {
	if !validStringID(&variableSetID) {
		return ErrInvalidVariableSetID
	}

	u := fmt.Sprintf("varsets/%s", url.PathEscape(variableSetID))
	req, err := s.client.NewRequest("DELETE", u, nil)
	if err != nil {
		return err
	}

	return req.Do(ctx, nil)
}

// Apply variable set to workspaces in the supplied list.
// Note: this method will return an error if the variable set has global = true.
func (s *variableSets) ApplyToWorkspaces(ctx context.Context, variableSetID string, options *VariableSetApplyToWorkspacesOptions) error {
	if !validStringID(&variableSetID) {
		return ErrInvalidVariableSetID
	}
	if err := options.valid(); err != nil {
		return err
	}

	u := fmt.Sprintf("varsets/%s/relationships/workspaces", url.PathEscape(variableSetID))
	req, err := s.client.NewRequest("POST", u, options.Workspaces)
	if err != nil {
		return err
	}

	return req.Do(ctx, nil)
}

// Remove variable set from workspaces in the supplied list.
// Note: this method will return an error if the variable set has global = true.
func (s *variableSets) RemoveFromWorkspaces(ctx context.Context, variableSetID string, options *VariableSetRemoveFromWorkspacesOptions) error {
	if !validStringID(&variableSetID) {
		return ErrInvalidVariableSetID
	}
	if err := options.valid(); err != nil {
		return err
	}

	u := fmt.Sprintf("varsets/%s/relationships/workspaces", url.PathEscape(variableSetID))
	req, err := s.client.NewRequest("DELETE", u, options.Workspaces)
	if err != nil {
		return err
	}

	return req.Do(ctx, nil)
}

// ApplyToProjects applies the variable set to projects in the supplied list.
// This method will return an error if the variable set has global = true.
func (s variableSets) ApplyToProjects(ctx context.Context, variableSetID string, options VariableSetApplyToProjectsOptions) error {
	if !validStringID(&variableSetID) {
		return ErrInvalidVariableSetID
	}
	if err := options.valid(); err != nil {
		return err
	}

	u := fmt.Sprintf("varsets/%s/relationships/projects", url.PathEscape(variableSetID))
	req, err := s.client.NewRequest("POST", u, options.Projects)
	if err != nil {
		return err
	}

	return req.Do(ctx, nil)
}

// RemoveFromProjects removes the variable set from projects in the supplied list.
// This method will return an error if the variable set has global = true.
func (s variableSets) RemoveFromProjects(ctx context.Context, variableSetID string, options VariableSetRemoveFromProjectsOptions) error {
	if !validStringID(&variableSetID) {
		return ErrInvalidVariableSetID
	}
	if err := options.valid(); err != nil {
		return err
	}

	u := fmt.Sprintf("varsets/%s/relationships/projects", url.PathEscape(variableSetID))
	req, err := s.client.NewRequest("DELETE", u, options.Projects)
	if err != nil {
		return err
	}

	return req.Do(ctx, nil)
}

// ApplyToStacks applies the variable set to stacks in the supplied list.
// This method will return an error if the variable set has global = true.
func (s *variableSets) ApplyToStacks(ctx context.Context, variableSetID string, options *VariableSetApplyToStacksOptions) error {
	if !validStringID(&variableSetID) {
		return ErrInvalidVariableSetID
	}
	if err := options.valid(); err != nil {
		return err
	}

	u := fmt.Sprintf("varsets/%s/relationships/stacks", url.PathEscape(variableSetID))
	req, err := s.client.NewRequest("POST", u, options.Stacks)
	if err != nil {
		return err
	}

	return req.Do(ctx, nil)
}

func (s *variableSets) RemoveFromStacks(ctx context.Context, variableSetID string, options *VariableSetRemoveFromStacksOptions) error {
	if !validStringID(&variableSetID) {
		return ErrInvalidVariableSetID
	}
	if err := options.valid(); err != nil {
		return err
	}

	u := fmt.Sprintf("varsets/%s/relationships/stacks", url.PathEscape(variableSetID))
	req, err := s.client.NewRequest("DELETE", u, options.Stacks)
	if err != nil {
		return err
	}

	return req.Do(ctx, nil)
}

// Update variable set to be applied to only the workspaces in the supplied list.
func (s *variableSets) UpdateWorkspaces(ctx context.Context, variableSetID string, options *VariableSetUpdateWorkspacesOptions) (*VariableSet, error) {
	if err := options.valid(); err != nil {
		return nil, err
	}

	// Use private struct to ensure global is set to false when applying to workspaces
	o := privateVariableSetUpdateWorkspacesOptions{
		Global:     bool(false),
		Workspaces: options.Workspaces,
	}

	// We force inclusion of workspaces as that is the primary data for which we are concerned with confirming changes.
	u := fmt.Sprintf("varsets/%s?include=%s", url.PathEscape(variableSetID), VariableSetWorkspaces)
	req, err := s.client.NewRequest("PATCH", u, &o)
	if err != nil {
		return nil, err
	}

	v := &VariableSet{}
	err = req.Do(ctx, v)
	if err != nil {
		return nil, err
	}

	return v, nil
}

// Update variable set to be applied to only the stacks in the supplied list.
func (s *variableSets) UpdateStacks(ctx context.Context, variableSetID string, options *VariableSetUpdateStacksOptions) (*VariableSet, error) {
	if err := options.valid(); err != nil {
		return nil, err
	}

	// Use private struct to ensure global is set to false when applying to stacks
	o := privateVariableSetUpdateStacksOptions{
		Global: bool(false),
		Stacks: options.Stacks,
	}

	// We force inclusion of stacks as that is the primary data for which we are concerned with confirming changes.
	u := fmt.Sprintf("varsets/%s?include=%s", url.PathEscape(variableSetID), VariableSetStacks)
	req, err := s.client.NewRequest("PATCH", u, &o)
	if err != nil {
		return nil, err
	}

	v := &VariableSet{}
	err = req.Do(ctx, v)
	if err != nil {
		return nil, err
	}

	return v, nil
}

func (o *VariableSetListOptions) valid() error {
	return nil
}

func (o *VariableSetCreateOptions) valid() error {
	if o == nil {
		return nil
	}
	if !validString(o.Name) {
		return ErrRequiredName
	}
	if o.Global == nil {
		return ErrRequiredGlobalFlag
	}
	return nil
}

func (o *VariableSetApplyToWorkspacesOptions) valid() error {
	for _, s := range o.Workspaces {
		if !validStringID(&s.ID) {
			return ErrRequiredWorkspaceID
		}
	}
	return nil
}

func (o *VariableSetRemoveFromWorkspacesOptions) valid() error {
	for _, s := range o.Workspaces {
		if !validStringID(&s.ID) {
			return ErrRequiredWorkspaceID
		}
	}
	return nil
}

func (o *VariableSetApplyToProjectsOptions) valid() error {
	for _, s := range o.Projects {
		if !validStringID(&s.ID) {
			return ErrRequiredProjectID
		}
	}
	return nil
}

func (o VariableSetRemoveFromProjectsOptions) valid() error {
	for _, s := range o.Projects {
		if !validStringID(&s.ID) {
			return ErrRequiredProjectID
		}
	}
	return nil
}

func (o VariableSetApplyToStacksOptions) valid() error {
	for _, s := range o.Stacks {
		if !validStringID(&s.ID) {
			return ErrRequiredStackID
		}
	}
	return nil
}

func (o VariableSetRemoveFromStacksOptions) valid() error {
	for _, s := range o.Stacks {
		if !validStringID(&s.ID) {
			return ErrRequiredStackID
		}
	}
	return nil
}

func (o *VariableSetUpdateWorkspacesOptions) valid() error {
	if o == nil || o.Workspaces == nil {
		return ErrRequiredWorkspacesList
	}
	return nil
}

func (o *VariableSetUpdateStacksOptions) valid() error {
	if o == nil || o.Stacks == nil {
		return ErrRequiredStacksList
	}
	return nil
}

// Compile-time proof of interface implementation.
var _ Variables = (*variables)(nil)

// Variables describes all the variable related methods that the Terraform
// Enterprise API supports.
//
// TFE API docs: https://developer.hashicorp.com/terraform/cloud-docs/api-docs/workspace-variables
type Variables interface {
	// List all the variables associated with the given workspace (doesn't include variables inherited from varsets).
	List(ctx context.Context, workspaceID string, options *VariableListOptions) (*VariableList, error)

	// ListAll all the variables associated with the given workspace including variables inherited from varsets.
	ListAll(ctx context.Context, workspaceID string, options *VariableListOptions) (*VariableList, error)

	// Create is used to create a new variable.
	Create(ctx context.Context, workspaceID string, options VariableCreateOptions) (*Variable, error)

	// Read a variable by its ID.
	Read(ctx context.Context, workspaceID string, variableID string) (*Variable, error)

	// Update values of an existing variable.
	Update(ctx context.Context, workspaceID string, variableID string, options VariableUpdateOptions) (*Variable, error)

	// Delete a variable by its ID.
	Delete(ctx context.Context, workspaceID string, variableID string) error
}

// variables implements Variables.
type variables struct {
	client *Client
}

// CategoryType represents a category type.
type CategoryType string

// List all available categories.
const (
	CategoryEnv       CategoryType = "env"
	CategoryPolicySet CategoryType = "policy-set"
	CategoryTerraform CategoryType = "terraform"
)

// VariableList represents a list of variables.
type VariableList struct {
	*Pagination
	Items []*Variable
}

// Variable represents a Terraform Enterprise variable.
type Variable struct {
	ID          string       `jsonapi:"primary,vars"`
	Key         string       `jsonapi:"attr,key"`
	Value       string       `jsonapi:"attr,value"`
	Description string       `jsonapi:"attr,description"`
	Category    CategoryType `jsonapi:"attr,category"`
	HCL         bool         `jsonapi:"attr,hcl"`
	Sensitive   bool         `jsonapi:"attr,sensitive"`
	VersionID   string       `jsonapi:"attr,version-id"`

	// Relations
	Workspace *Workspace `jsonapi:"relation,configurable"`
}

// VariableListOptions represents the options for listing variables.
type VariableListOptions struct {
	ListOptions
}

// VariableCreateOptions represents the options for creating a new variable.
type VariableCreateOptions struct {
	// Type is a public field utilized by JSON:API to
	// set the resource type via the field tag.
	// It is not a user-defined value and does not need to be set.
	// https://jsonapi.org/format/#crud-creating
	Type string `jsonapi:"primary,vars"`

	// Required: The name of the variable.
	Key *string `jsonapi:"attr,key"`

	// Optional: The value of the variable.
	Value *string `jsonapi:"attr,value,omitempty"`

	// Optional: The description of the variable.
	Description *string `jsonapi:"attr,description,omitempty"`

	// Required: Whether this is a Terraform or environment variable.
	Category *CategoryType `jsonapi:"attr,category"`

	// Optional: Whether to evaluate the value of the variable as a string of HCL code.
	HCL *bool `jsonapi:"attr,hcl,omitempty"`

	// Optional: Whether the value is sensitive.
	Sensitive *bool `jsonapi:"attr,sensitive,omitempty"`
}

// VariableUpdateOptions represents the options for updating a variable.
type VariableUpdateOptions struct {
	// Type is a public field utilized by JSON:API to
	// set the resource type via the field tag.
	// It is not a user-defined value and does not need to be set.
	// https://jsonapi.org/format/#crud-creating
	Type string `jsonapi:"primary,vars"`

	// The name of the variable.
	Key *string `jsonapi:"attr,key,omitempty"`

	// The value of the variable.
	Value *string `jsonapi:"attr,value,omitempty"`

	// The description of the variable.
	Description *string `jsonapi:"attr,description,omitempty"`

	// Whether this is a Terraform or environment variable.
	Category *CategoryType `jsonapi:"attr,category,omitempty"`

	// Whether to evaluate the value of the variable as a string of HCL code.
	HCL *bool `jsonapi:"attr,hcl,omitempty"`

	// Whether the value is sensitive.
	Sensitive *bool `jsonapi:"attr,sensitive,omitempty"`
}

// List all the variables associated with the given workspace (doesn't include variables inherited from varsets).
func (s *variables) List(ctx context.Context, workspaceID string, options *VariableListOptions) (*VariableList, error) {
	return s.getList(ctx, workspaceID, options, "workspaces/%s/vars")
}

// ListAll the variables associated with the given workspace including variables inherited from varsets.
func (s *variables) ListAll(ctx context.Context, workspaceID string, options *VariableListOptions) (*VariableList, error) {
	return s.getList(ctx, workspaceID, options, "workspaces/%s/all-vars")
}

func (s *variables) getList(ctx context.Context, workspaceID string, options *VariableListOptions, path string) (*VariableList, error) {
	if !validStringID(&workspaceID) {
		return nil, ErrInvalidWorkspaceID
	}

	u := fmt.Sprintf(path, url.PathEscape(workspaceID))
	req, err := s.client.NewRequest("GET", u, options)
	if err != nil {
		return nil, err
	}

	vl := &VariableList{}
	err = req.Do(ctx, vl)
	if err != nil {
		return nil, err
	}

	return vl, nil
}

// Create is used to create a new variable.
func (s *variables) Create(ctx context.Context, workspaceID string, options VariableCreateOptions) (*Variable, error) {
	if !validStringID(&workspaceID) {
		return nil, ErrInvalidWorkspaceID
	}
	if err := options.valid(); err != nil {
		return nil, err
	}

	u := fmt.Sprintf("workspaces/%s/vars", url.PathEscape(workspaceID))
	req, err := s.client.NewRequest("POST", u, &options)
	if err != nil {
		return nil, err
	}

	v := &Variable{}
	err = req.Do(ctx, v)
	if err != nil {
		return nil, err
	}

	return v, nil
}

// Read a variable by its ID.
func (s *variables) Read(ctx context.Context, workspaceID, variableID string) (*Variable, error) {
	if !validStringID(&workspaceID) {
		return nil, ErrInvalidWorkspaceID
	}
	if !validStringID(&variableID) {
		return nil, ErrInvalidVariableID
	}

	u := fmt.Sprintf("workspaces/%s/vars/%s", url.PathEscape(workspaceID), url.PathEscape(variableID))
	req, err := s.client.NewRequest("GET", u, nil)
	if err != nil {
		return nil, err
	}

	v := &Variable{}
	err = req.Do(ctx, v)
	if err != nil {
		return nil, err
	}

	return v, err
}

// Update values of an existing variable.
func (s *variables) Update(ctx context.Context, workspaceID, variableID string, options VariableUpdateOptions) (*Variable, error) {
	if !validStringID(&workspaceID) {
		return nil, ErrInvalidWorkspaceID
	}
	if !validStringID(&variableID) {
		return nil, ErrInvalidVariableID
	}

	u := fmt.Sprintf("workspaces/%s/vars/%s", url.PathEscape(workspaceID), url.PathEscape(variableID))
	req, err := s.client.NewRequest("PATCH", u, &options)
	if err != nil {
		return nil, err
	}

	v := &Variable{}
	err = req.Do(ctx, v)
	if err != nil {
		return nil, err
	}

	return v, nil
}

// Delete a variable by its ID.
func (s *variables) Delete(ctx context.Context, workspaceID, variableID string) error {
	if !validStringID(&workspaceID) {
		return ErrInvalidWorkspaceID
	}
	if !validStringID(&variableID) {
		return ErrInvalidVariableID
	}

	u := fmt.Sprintf("workspaces/%s/vars/%s", url.PathEscape(workspaceID), url.PathEscape(variableID))
	req, err := s.client.NewRequest("DELETE", u, nil)
	if err != nil {
		return err
	}

	return req.Do(ctx, nil)
}

func (o VariableCreateOptions) valid() error {
	if !validString(o.Key) {
		return ErrRequiredKey
	}
	if o.Category == nil {
		return ErrRequiredCategory
	}
	return nil
}

// VaultOIDCConfigurations describes all the Vault OIDC configuration related methods that the HCP Terraform API supports.
// HCP Terraform API docs:
// https://developer.hashicorp.com/terraform/cloud-docs/api-docs/hold-your-own-key/oidc-configurations/vault
type VaultOIDCConfigurations interface {
	Create(ctx context.Context, organization string, options VaultOIDCConfigurationCreateOptions) (*VaultOIDCConfiguration, error)

	Read(ctx context.Context, oidcID string) (*VaultOIDCConfiguration, error)

	Update(ctx context.Context, oidcID string, options VaultOIDCConfigurationUpdateOptions) (*VaultOIDCConfiguration, error)

	Delete(ctx context.Context, oidcID string) error
}

type vaultOIDCConfigurations struct {
	client *Client
}

var _ VaultOIDCConfigurations = &vaultOIDCConfigurations{}

type VaultOIDCConfiguration struct {
	ID               string `jsonapi:"primary,vault-oidc-configurations"`
	Address          string `jsonapi:"attr,address"`
	RoleName         string `jsonapi:"attr,role"`
	Namespace        string `jsonapi:"attr,namespace"`
	JWTAuthPath      string `jsonapi:"attr,auth-path"`
	TLSCACertificate string `jsonapi:"attr,encoded-cacert"`

	Organization *Organization `jsonapi:"relation,organization"`
}

type VaultOIDCConfigurationCreateOptions struct {
	// Type is a public field utilized by JSON:API to
	// set the resource type via the field tag.
	// It is not a user-defined value and does not need to be set.
	// https://jsonapi.org/format/#crud-creating
	Type string `jsonapi:"primary,vault-oidc-configurations"`

	// Attributes
	Address          string `jsonapi:"attr,address"`
	RoleName         string `jsonapi:"attr,role"`
	Namespace        string `jsonapi:"attr,namespace"`
	JWTAuthPath      string `jsonapi:"attr,auth-path"`
	TLSCACertificate string `jsonapi:"attr,encoded-cacert"`
}

type VaultOIDCConfigurationUpdateOptions struct {
	// Type is a public field utilized by JSON:API to
	// set the resource type via the field tag.
	// It is not a user-defined value and does not need to be set.
	// https://jsonapi.org/format/#crud-creating
	Type string `jsonapi:"primary,vault-oidc-configurations"`

	// Attributes
	Address          *string `jsonapi:"attr,address,omitempty"`
	RoleName         *string `jsonapi:"attr,role,omitempty"`
	Namespace        *string `jsonapi:"attr,namespace,omitempty"`
	JWTAuthPath      *string `jsonapi:"attr,auth-path,omitempty"`
	TLSCACertificate *string `jsonapi:"attr,encoded-cacert,omitempty"`
}

func (o *VaultOIDCConfigurationCreateOptions) valid() error {
	if o.Address == "" {
		return ErrRequiredVaultAddress
	}

	if o.RoleName == "" {
		return ErrRequiredRoleName
	}

	return nil
}

func (voc *vaultOIDCConfigurations) Create(ctx context.Context, organization string, options VaultOIDCConfigurationCreateOptions) (*VaultOIDCConfiguration, error) {
	if !validStringID(&organization) {
		return nil, ErrInvalidOrg
	}

	if err := options.valid(); err != nil {
		return nil, err
	}

	req, err := voc.client.NewRequest("POST", fmt.Sprintf("organizations/%s/oidc-configurations", url.PathEscape(organization)), &options)
	if err != nil {
		return nil, err
	}

	vaultOIDCConfiguration := &VaultOIDCConfiguration{}
	err = req.Do(ctx, vaultOIDCConfiguration)
	if err != nil {
		return nil, err
	}

	return vaultOIDCConfiguration, nil
}

func (voc *vaultOIDCConfigurations) Read(ctx context.Context, oidcID string) (*VaultOIDCConfiguration, error) {
	req, err := voc.client.NewRequest("GET", fmt.Sprintf(OIDCConfigPathFormat, url.PathEscape(oidcID)), nil)
	if err != nil {
		return nil, err
	}

	vaultOIDCConfiguration := &VaultOIDCConfiguration{}
	err = req.Do(ctx, vaultOIDCConfiguration)
	if err != nil {
		return nil, err
	}

	return vaultOIDCConfiguration, nil
}

func (voc *vaultOIDCConfigurations) Update(ctx context.Context, oidcID string, options VaultOIDCConfigurationUpdateOptions) (*VaultOIDCConfiguration, error) {
	if !validStringID(&oidcID) {
		return nil, ErrInvalidOIDC
	}

	req, err := voc.client.NewRequest("PATCH", fmt.Sprintf(OIDCConfigPathFormat, url.PathEscape(oidcID)), &options)
	if err != nil {
		return nil, err
	}

	vaultOIDCConfiguration := &VaultOIDCConfiguration{}
	err = req.Do(ctx, vaultOIDCConfiguration)
	if err != nil {
		return nil, err
	}

	return vaultOIDCConfiguration, nil
}

func (voc *vaultOIDCConfigurations) Delete(ctx context.Context, oidcID string) error {
	if !validStringID(&oidcID) {
		return ErrInvalidOIDC
	}

	req, err := voc.client.NewRequest("DELETE", fmt.Sprintf(OIDCConfigPathFormat, url.PathEscape(oidcID)), nil)
	if err != nil {
		return err
	}

	return req.Do(ctx, nil)
}

// Compile-time proof of interface implementation.
var _ WorkspaceResources = (*workspaceResources)(nil)

// WorkspaceResources describes all the workspace resources related methods that the Terraform
// Enterprise API supports.
//
// TFE API docs: https://developer.hashicorp.com/terraform/cloud-docs/api-docs/workspace-resources
type WorkspaceResources interface {
	// List all the workspaces resources within a workspace
	List(ctx context.Context, workspaceID string, options *WorkspaceResourceListOptions) (*WorkspaceResourcesList, error)
}

// workspaceResources implements WorkspaceResources.
type workspaceResources struct {
	client *Client
}

// WorkspaceResourcesList represents a list of workspace resources.
type WorkspaceResourcesList struct {
	*Pagination
	Items []*WorkspaceResource
}

// WorkspaceResource represents a Terraform Enterprise workspace resource.
type WorkspaceResource struct {
	ID                       string  `jsonapi:"primary,resources"`
	Address                  string  `jsonapi:"attr,address"`
	Name                     string  `jsonapi:"attr,name"`
	CreatedAt                string  `jsonapi:"attr,created-at"`
	UpdatedAt                string  `jsonapi:"attr,updated-at"`
	Module                   string  `jsonapi:"attr,module"`
	Provider                 string  `jsonapi:"attr,provider"`
	ProviderType             string  `jsonapi:"attr,provider-type"`
	ModifiedByStateVersionID string  `jsonapi:"attr,modified-by-state-version-id"`
	NameIndex                *string `jsonapi:"attr,name-index"`
}

// WorkspaceResourceListOptions represents the options for listing workspace resources.
type WorkspaceResourceListOptions struct {
	ListOptions
}

// List all the workspaces resources within a workspace
func (s *workspaceResources) List(ctx context.Context, workspaceID string, options *WorkspaceResourceListOptions) (*WorkspaceResourcesList, error) {
	if !validStringID(&workspaceID) {
		return nil, ErrInvalidWorkspaceID
	}
	if err := options.valid(); err != nil {
		return nil, err
	}

	u := fmt.Sprintf("workspaces/%s/resources", url.PathEscape(workspaceID))
	req, err := s.client.NewRequest("GET", u, options)
	if err != nil {
		return nil, err
	}

	wl := &WorkspaceResourcesList{}
	err = req.Do(ctx, wl)
	if err != nil {
		return nil, err
	}
	return wl, nil
}

func (o *WorkspaceResourceListOptions) valid() error {
	return nil
}

// Compile-time proof of interface implementation
var _ WorkspaceRunTasks = (*workspaceRunTasks)(nil)

// WorkspaceRunTasks represent all the run task related methods in the context of a workspace that the HCP Terraform and Terraform Enterprise API supports.
type WorkspaceRunTasks interface {
	// Add a run task to a workspace
	Create(ctx context.Context, workspaceID string, options WorkspaceRunTaskCreateOptions) (*WorkspaceRunTask, error)

	// List all run tasks for a workspace
	List(ctx context.Context, workspaceID string, options *WorkspaceRunTaskListOptions) (*WorkspaceRunTaskList, error)

	// Read a workspace run task by ID
	Read(ctx context.Context, workspaceID string, workspaceTaskID string) (*WorkspaceRunTask, error)

	// Update a workspace run task by ID
	Update(ctx context.Context, workspaceID string, workspaceTaskID string, options WorkspaceRunTaskUpdateOptions) (*WorkspaceRunTask, error)

	// Delete a workspace's run task by ID
	Delete(ctx context.Context, workspaceID string, workspaceTaskID string) error
}

// workspaceRunTasks implements WorkspaceRunTasks
type workspaceRunTasks struct {
	client *Client
}

// WorkspaceRunTask represents a HCP Terraform or Terraform Enterprise run task that belongs to a workspace
type WorkspaceRunTask struct {
	ID               string               `jsonapi:"primary,workspace-tasks"`
	EnforcementLevel TaskEnforcementLevel `jsonapi:"attr,enforcement-level"`
	// Deprecated: Use Stages property instead.
	Stage  Stage   `jsonapi:"attr,stage"`
	Stages []Stage `jsonapi:"attr,stages"`

	RunTask   *RunTask   `jsonapi:"relation,task"`
	Workspace *Workspace `jsonapi:"relation,workspace"`
}

// WorkspaceRunTaskList represents a list of workspace run tasks
type WorkspaceRunTaskList struct {
	*Pagination
	Items []*WorkspaceRunTask
}

// WorkspaceRunTaskListOptions represents the set of options for listing workspace run tasks
type WorkspaceRunTaskListOptions struct {
	ListOptions
}

// WorkspaceRunTaskCreateOptions represents the set of options for creating a workspace run task
type WorkspaceRunTaskCreateOptions struct {
	Type string `jsonapi:"primary,workspace-tasks"`
	// Required: The enforcement level for a run task
	EnforcementLevel TaskEnforcementLevel `jsonapi:"attr,enforcement-level"`
	// Required: The run task to attach to the workspace
	RunTask *RunTask `jsonapi:"relation,task"`
	// Deprecated: Use Stages property instead.
	Stage *Stage `jsonapi:"attr,stage,omitempty"`
	// Optional: The stage to run the task in
	Stages *[]Stage `jsonapi:"attr,stages,omitempty"`
}

// WorkspaceRunTaskUpdateOptions represent the set of options for updating a workspace run task.
type WorkspaceRunTaskUpdateOptions struct {
	Type             string               `jsonapi:"primary,workspace-tasks"`
	EnforcementLevel TaskEnforcementLevel `jsonapi:"attr,enforcement-level,omitempty"`
	// Deprecated: Use Stages property instead.
	Stage *Stage `jsonapi:"attr,stage,omitempty"`
	// Optional: The stage to run the task in
	Stages *[]Stage `jsonapi:"attr,stages,omitempty"`
}

// List all run tasks attached to a workspace
func (s *workspaceRunTasks) List(ctx context.Context, workspaceID string, options *WorkspaceRunTaskListOptions) (*WorkspaceRunTaskList, error) {
	if !validStringID(&workspaceID) {
		return nil, ErrInvalidWorkspaceID
	}

	u := fmt.Sprintf("workspaces/%s/tasks", url.PathEscape(workspaceID))
	req, err := s.client.NewRequest("GET", u, options)
	if err != nil {
		return nil, err
	}

	rl := &internalWorkspaceRunTaskList{}
	err = req.Do(ctx, rl)
	if err != nil {
		return nil, err
	}

	return rl.ToWorkspaceRunTaskList(), nil
}

// Read a workspace run task by ID
func (s *workspaceRunTasks) Read(ctx context.Context, workspaceID, workspaceTaskID string) (*WorkspaceRunTask, error) {
	if !validStringID(&workspaceID) {
		return nil, ErrInvalidWorkspaceID
	}

	if !validStringID(&workspaceTaskID) {
		return nil, ErrInvalidWorkspaceRunTaskID
	}

	u := fmt.Sprintf(
		"workspaces/%s/tasks/%s",
		url.PathEscape(workspaceID),
		url.PathEscape(workspaceTaskID),
	)
	req, err := s.client.NewRequest("GET", u, nil)
	if err != nil {
		return nil, err
	}

	wr := &internalWorkspaceRunTask{}
	err = req.Do(ctx, wr)
	if err != nil {
		return nil, err
	}

	return wr.ToWorkspaceRunTask(), nil
}

// Create is used to attach a run task to a workspace, or in other words: create a workspace run task. The run task must exist in the workspace's organization.
func (s *workspaceRunTasks) Create(ctx context.Context, workspaceID string, options WorkspaceRunTaskCreateOptions) (*WorkspaceRunTask, error) {
	if !validStringID(&workspaceID) {
		return nil, ErrInvalidWorkspaceID
	}

	if err := options.valid(); err != nil {
		return nil, err
	}

	u := fmt.Sprintf("workspaces/%s/tasks", workspaceID)
	req, err := s.client.NewRequest("POST", u, &options)
	if err != nil {
		return nil, err
	}

	wr := &internalWorkspaceRunTask{}
	err = req.Do(ctx, wr)
	if err != nil {
		return nil, err
	}

	return wr.ToWorkspaceRunTask(), nil
}

// Update an existing workspace run task by ID
func (s *workspaceRunTasks) Update(ctx context.Context, workspaceID, workspaceTaskID string, options WorkspaceRunTaskUpdateOptions) (*WorkspaceRunTask, error) {
	if !validStringID(&workspaceID) {
		return nil, ErrInvalidWorkspaceID
	}

	if !validStringID(&workspaceTaskID) {
		return nil, ErrInvalidWorkspaceRunTaskID
	}

	u := fmt.Sprintf(
		"workspaces/%s/tasks/%s",
		url.PathEscape(workspaceID),
		url.PathEscape(workspaceTaskID),
	)
	req, err := s.client.NewRequest("PATCH", u, &options)
	if err != nil {
		return nil, err
	}

	wr := &internalWorkspaceRunTask{}
	err = req.Do(ctx, wr)
	if err != nil {
		return nil, err
	}

	return wr.ToWorkspaceRunTask(), nil
}

// Delete a workspace run task by ID
func (s *workspaceRunTasks) Delete(ctx context.Context, workspaceID, workspaceTaskID string) error {
	if !validStringID(&workspaceID) {
		return ErrInvalidWorkspaceID
	}

	if !validStringID(&workspaceTaskID) {
		return ErrInvalidWorkspaceRunTaskType
	}

	u := fmt.Sprintf(
		"workspaces/%s/tasks/%s",
		url.PathEscape(workspaceID),
		url.PathEscape(workspaceTaskID),
	)
	req, err := s.client.NewRequest("DELETE", u, nil)
	if err != nil {
		return err
	}

	return req.Do(ctx, nil)
}

func (o *WorkspaceRunTaskCreateOptions) valid() error {
	if o.RunTask.ID == "" {
		return ErrInvalidRunTaskID
	}

	return nil
}

// Compile-time proof of interface implementation.
var _ Workspaces = (*workspaces)(nil)

// Workspaces describes all the workspace related methods that the Terraform
// Enterprise API supports.
//
// TFE API docs: https://developer.hashicorp.com/terraform/cloud-docs/api-docs/workspaces
type Workspaces interface {
	// List all the workspaces within an organization.
	List(ctx context.Context, organization string, options *WorkspaceListOptions) (*WorkspaceList, error)

	// Create is used to create a new workspace.
	Create(ctx context.Context, organization string, options WorkspaceCreateOptions) (*Workspace, error)

	// Read a workspace by its name and organization name.
	Read(ctx context.Context, organization string, workspace string) (*Workspace, error)

	// ReadWithOptions reads a workspace by name and organization name with given options.
	ReadWithOptions(ctx context.Context, organization string, workspace string, options *WorkspaceReadOptions) (*Workspace, error)

	// Readme gets the readme of a workspace by its ID.
	Readme(ctx context.Context, workspaceID string) (io.Reader, error)

	// ReadByID reads a workspace by its ID.
	ReadByID(ctx context.Context, workspaceID string) (*Workspace, error)

	// ReadByIDWithOptions reads a workspace by its ID with the given options.
	ReadByIDWithOptions(ctx context.Context, workspaceID string, options *WorkspaceReadOptions) (*Workspace, error)

	// Update settings of an existing workspace.
	Update(ctx context.Context, organization string, workspace string, options WorkspaceUpdateOptions) (*Workspace, error)

	// UpdateByID updates the settings of an existing workspace.
	UpdateByID(ctx context.Context, workspaceID string, options WorkspaceUpdateOptions) (*Workspace, error)

	// Delete a workspace by its name.
	Delete(ctx context.Context, organization string, workspace string) error

	// DeleteByID deletes a workspace by its ID.
	DeleteByID(ctx context.Context, workspaceID string) error

	// SafeDelete a workspace by its name.
	SafeDelete(ctx context.Context, organization string, workspace string) error

	// SafeDeleteByID deletes a workspace by its ID.
	SafeDeleteByID(ctx context.Context, workspaceID string) error

	// RemoveVCSConnection from a workspace.
	RemoveVCSConnection(ctx context.Context, organization, workspace string) (*Workspace, error)

	// RemoveVCSConnectionByID removes a VCS connection from a workspace.
	RemoveVCSConnectionByID(ctx context.Context, workspaceID string) (*Workspace, error)

	// Lock a workspace by its ID.
	Lock(ctx context.Context, workspaceID string, options WorkspaceLockOptions) (*Workspace, error)

	// Unlock a workspace by its ID.
	Unlock(ctx context.Context, workspaceID string) (*Workspace, error)

	// ForceUnlock a workspace by its ID.
	ForceUnlock(ctx context.Context, workspaceID string) (*Workspace, error)

	// AssignSSHKey to a workspace.
	AssignSSHKey(ctx context.Context, workspaceID string, options WorkspaceAssignSSHKeyOptions) (*Workspace, error)

	// UnassignSSHKey from a workspace.
	UnassignSSHKey(ctx context.Context, workspaceID string) (*Workspace, error)

	// ListRemoteStateConsumers reads the remote state consumers for a workspace.
	ListRemoteStateConsumers(ctx context.Context, workspaceID string, options *RemoteStateConsumersListOptions) (*WorkspaceList, error)

	// AddRemoteStateConsumers adds remote state consumers to a workspace.
	AddRemoteStateConsumers(ctx context.Context, workspaceID string, options WorkspaceAddRemoteStateConsumersOptions) error

	// RemoveRemoteStateConsumers removes remote state consumers from a workspace.
	RemoveRemoteStateConsumers(ctx context.Context, workspaceID string, options WorkspaceRemoveRemoteStateConsumersOptions) error

	// UpdateRemoteStateConsumers updates all the remote state consumers for a workspace
	// to match the workspaces in the update options.
	UpdateRemoteStateConsumers(ctx context.Context, workspaceID string, options WorkspaceUpdateRemoteStateConsumersOptions) error

	// ListTags reads the tags for a workspace.
	ListTags(ctx context.Context, workspaceID string, options *WorkspaceTagListOptions) (*TagList, error)

	// AddTags appends tags to a workspace
	AddTags(ctx context.Context, workspaceID string, options WorkspaceAddTagsOptions) error

	// RemoveTags removes tags from a workspace
	RemoveTags(ctx context.Context, workspaceID string, options WorkspaceRemoveTagsOptions) error

	// ReadDataRetentionPolicy reads a workspace's data retention policy
	//
	// Deprecated: Use ReadDataRetentionPolicyChoice instead.
	// **Note: This functionality is only available in Terraform Enterprise versions v202311-1 and v202312-1.**
	ReadDataRetentionPolicy(ctx context.Context, workspaceID string) (*DataRetentionPolicy, error)

	// ReadDataRetentionPolicyChoice reads a workspace's data retention policy
	// **Note: This functionality is only available in Terraform Enterprise.**
	ReadDataRetentionPolicyChoice(ctx context.Context, workspaceID string) (*DataRetentionPolicyChoice, error)

	// SetDataRetentionPolicy sets a workspace's data retention policy to delete data older than a certain number of days
	//
	// Deprecated: Use SetDataRetentionPolicyDeleteOlder instead
	// **Note: This functionality is only available in Terraform Enterprise versions v202311-1 and v202312-1.**
	SetDataRetentionPolicy(ctx context.Context, workspaceID string, options DataRetentionPolicySetOptions) (*DataRetentionPolicy, error)

	// SetDataRetentionPolicyDeleteOlder sets a workspace's data retention policy to delete data older than a certain number of days
	// **Note: This functionality is only available in Terraform Enterprise.**
	SetDataRetentionPolicyDeleteOlder(ctx context.Context, workspaceID string, options DataRetentionPolicyDeleteOlderSetOptions) (*DataRetentionPolicyDeleteOlder, error)

	// SetDataRetentionPolicyDontDelete sets a workspace's data retention policy to explicitly not delete data
	// **Note: This functionality is only available in Terraform Enterprise.**
	SetDataRetentionPolicyDontDelete(ctx context.Context, workspaceID string, options DataRetentionPolicyDontDeleteSetOptions) (*DataRetentionPolicyDontDelete, error)

	// DeleteDataRetentionPolicy deletes a workspace's data retention policy
	// **Note: This functionality is only available in Terraform Enterprise.**
	DeleteDataRetentionPolicy(ctx context.Context, workspaceID string) error

	// ListTagBindings lists all tag bindings associated with the workspace.
	ListTagBindings(ctx context.Context, workspaceID string) ([]*TagBinding, error)

	// ListEffectiveTagBindings lists all tag bindings associated with the workspace which may be
	// either inherited from a project or binded to the workspace itself.
	ListEffectiveTagBindings(ctx context.Context, workspaceID string) ([]*EffectiveTagBinding, error)

	// AddTagBindings adds or modifies the value of existing tag binding keys for a workspace.
	AddTagBindings(ctx context.Context, workspaceID string, options WorkspaceAddTagBindingsOptions) ([]*TagBinding, error)

	// DeleteAllTagBindings removes all tag bindings for a workspace.
	DeleteAllTagBindings(ctx context.Context, workspaceID string) error
}

// workspaces implements Workspaces.
type workspaces struct {
	client *Client
}

// WorkspaceSource represents a source type of a workspace.
type WorkspaceSource string

const (
	WorkspaceSourceAPI       WorkspaceSource = "tfe-api"
	WorkspaceSourceModule    WorkspaceSource = "tfe-module"
	WorkspaceSourceUI        WorkspaceSource = "tfe-ui"
	WorkspaceSourceTerraform WorkspaceSource = "terraform"
)

// WorkspaceList represents a list of workspaces.
type WorkspaceList struct {
	*Pagination
	Items []*Workspace
}

// WorkspaceAddTagBindingsOptions represents the options for adding tag bindings
// to a workspace.
type WorkspaceAddTagBindingsOptions struct {
	TagBindings []*TagBinding
}

// LockedByChoice is a choice type struct that represents the possible values
// within a polymorphic relation. If a value is available, exactly one field
// will be non-nil.
type LockedByChoice struct {
	Run  *Run
	User *User
	Team *Team
}

// Workspace represents a Terraform Enterprise workspace.
type Workspace struct {
	ID                          string                          `jsonapi:"primary,workspaces"`
	Actions                     *WorkspaceActions               `jsonapi:"attr,actions"`
	AllowDestroyPlan            bool                            `jsonapi:"attr,allow-destroy-plan"`
	AssessmentsEnabled          bool                            `jsonapi:"attr,assessments-enabled"`
	AutoApply                   bool                            `jsonapi:"attr,auto-apply"`
	AutoApplyRunTrigger         bool                            `jsonapi:"attr,auto-apply-run-trigger"`
	AutoDestroyAt               jsonapi.NullableAttr[time.Time] `jsonapi:"attr,auto-destroy-at,iso8601,omitempty"`
	AutoDestroyActivityDuration jsonapi.NullableAttr[string]    `jsonapi:"attr,auto-destroy-activity-duration,omitempty"`
	CanQueueDestroyPlan         bool                            `jsonapi:"attr,can-queue-destroy-plan"`
	CreatedAt                   time.Time                       `jsonapi:"attr,created-at,iso8601"`
	Description                 string                          `jsonapi:"attr,description"`
	Environment                 string                          `jsonapi:"attr,environment"`
	ExecutionMode               string                          `jsonapi:"attr,execution-mode"`
	FileTriggersEnabled         bool                            `jsonapi:"attr,file-triggers-enabled"`
	GlobalRemoteState           bool                            `jsonapi:"attr,global-remote-state"`
	ProjectRemoteState          bool                            `jsonapi:"attr,project-remote-state"`
	InheritsProjectAutoDestroy  bool                            `jsonapi:"attr,inherits-project-auto-destroy"`
	Locked                      bool                            `jsonapi:"attr,locked"`
	MigrationEnvironment        string                          `jsonapi:"attr,migration-environment"`
	Name                        string                          `jsonapi:"attr,name"`
	NoCodeUpgradeAvailable      bool                            `jsonapi:"attr,no-code-upgrade-available"`
	Operations                  bool                            `jsonapi:"attr,operations"`
	Permissions                 *WorkspacePermissions           `jsonapi:"attr,permissions"`
	QueueAllRuns                bool                            `jsonapi:"attr,queue-all-runs"`
	SpeculativeEnabled          bool                            `jsonapi:"attr,speculative-enabled"`
	Source                      WorkspaceSource                 `jsonapi:"attr,source"`
	SourceName                  string                          `jsonapi:"attr,source-name"`
	SourceURL                   string                          `jsonapi:"attr,source-url"`
	SourceModuleID              string                          `jsonapi:"attr,source-module-id"`
	StructuredRunOutputEnabled  bool                            `jsonapi:"attr,structured-run-output-enabled"`
	TerraformVersion            string                          `jsonapi:"attr,terraform-version"`
	TriggerPrefixes             []string                        `jsonapi:"attr,trigger-prefixes"`
	TriggerPatterns             []string                        `jsonapi:"attr,trigger-patterns"`
	VCSRepo                     *VCSRepo                        `jsonapi:"attr,vcs-repo"`
	WorkingDirectory            string                          `jsonapi:"attr,working-directory"`
	UpdatedAt                   time.Time                       `jsonapi:"attr,updated-at,iso8601"`
	ResourceCount               int                             `jsonapi:"attr,resource-count"`
	ApplyDurationAverage        time.Duration                   `jsonapi:"attr,apply-duration-average"`
	PlanDurationAverage         time.Duration                   `jsonapi:"attr,plan-duration-average"`
	PolicyCheckFailures         int                             `jsonapi:"attr,policy-check-failures"`
	RunFailures                 int                             `jsonapi:"attr,run-failures"`
	RunsCount                   int                             `jsonapi:"attr,workspace-kpis-runs-count"`
	TagNames                    []string                        `jsonapi:"attr,tag-names"`
	SettingOverwrites           *WorkspaceSettingOverwrites     `jsonapi:"attr,setting-overwrites"`
	HYOKEnabled                 *bool                           `jsonapi:"attr,hyok-enabled"`

	// Relations
	AgentPool                   *AgentPool             `jsonapi:"relation,agent-pool"`
	CurrentRun                  *Run                   `jsonapi:"relation,current-run"`
	CurrentStateVersion         *StateVersion          `jsonapi:"relation,current-state-version"`
	Organization                *Organization          `jsonapi:"relation,organization"`
	SSHKey                      *SSHKey                `jsonapi:"relation,ssh-key"`
	Outputs                     []*WorkspaceOutputs    `jsonapi:"relation,outputs"`
	Project                     *Project               `jsonapi:"relation,project"`
	Tags                        []*Tag                 `jsonapi:"relation,tags"`
	CurrentConfigurationVersion *ConfigurationVersion  `jsonapi:"relation,current-configuration-version,omitempty"`
	LockedBy                    *LockedByChoice        `jsonapi:"polyrelation,locked-by"`
	Variables                   []*Variable            `jsonapi:"relation,vars"`
	TagBindings                 []*TagBinding          `jsonapi:"relation,tag-bindings"`
	EffectiveTagBindings        []*EffectiveTagBinding `jsonapi:"relation,effective-tag-bindings"`
	HYOKEncryptedDataKey        *HYOKEncryptedDataKey  `jsonapi:"relation,hyok-data-key-for-encryption"`

	// Deprecated: Use DataRetentionPolicyChoice instead.
	DataRetentionPolicy *DataRetentionPolicy
	// **Note: This functionality is only available in Terraform Enterprise.**
	DataRetentionPolicyChoice *DataRetentionPolicyChoice `jsonapi:"polyrelation,data-retention-policy"`

	// Links
	Links map[string]interface{} `jsonapi:"links,omitempty"`
}

type WorkspaceOutputs struct {
	ID        string      `jsonapi:"primary,workspace-outputs"`
	Name      string      `jsonapi:"attr,name"`
	Sensitive bool        `jsonapi:"attr,sensitive"`
	Type      string      `jsonapi:"attr,output-type"`
	Value     interface{} `jsonapi:"attr,value"`
}

// workspaceWithReadme is the same as a workspace but it has a readme.
type workspaceWithReadme struct {
	ID     string           `jsonapi:"primary,workspaces"`
	Readme *workspaceReadme `jsonapi:"relation,readme"`
}

// workspaceReadme contains the readme of the workspace.
type workspaceReadme struct {
	ID          string `jsonapi:"primary,workspace-readme"`
	RawMarkdown string `jsonapi:"attr,raw-markdown"`
}

// VCSRepo contains the configuration of a VCS integration.
type VCSRepo struct {
	Branch            string `jsonapi:"attr,branch"`
	DisplayIdentifier string `jsonapi:"attr,display-identifier"`
	Identifier        string `jsonapi:"attr,identifier"`
	IngressSubmodules bool   `jsonapi:"attr,ingress-submodules"`
	OAuthTokenID      string `jsonapi:"attr,oauth-token-id"`
	GHAInstallationID string `jsonapi:"attr,github-app-installation-id"`
	RepositoryHTTPURL string `jsonapi:"attr,repository-http-url"`
	ServiceProvider   string `jsonapi:"attr,service-provider"`
	Tags              bool   `jsonapi:"attr,tags"`
	TagsRegex         string `jsonapi:"attr,tags-regex"`
	WebhookURL        string `jsonapi:"attr,webhook-url"`
	SourceDirectory   string `jsonapi:"attr,source-directory"`
	TagPrefix         string `jsonapi:"attr,tag-prefix"`
}

// Note: the fields of this struct are bool pointers instead of bool values, in order to simplify support for
// future TFE versions that support *some but not all* of the inherited defaults that go-tfe knows about.
type WorkspaceSettingOverwrites struct {
	ExecutionMode *bool `jsonapi:"attr,execution-mode"`
	AgentPool     *bool `jsonapi:"attr,agent-pool"`
}

// WorkspaceActions represents the workspace actions.
type WorkspaceActions struct {
	IsDestroyable bool `jsonapi:"attr,is-destroyable"`
}

// WorkspacePermissions represents the workspace permissions.
type WorkspacePermissions struct {
	CanDestroy           bool  `jsonapi:"attr,can-destroy"`
	CanForceUnlock       bool  `jsonapi:"attr,can-force-unlock"`
	CanLock              bool  `jsonapi:"attr,can-lock"`
	CanManageRunTasks    bool  `jsonapi:"attr,can-manage-run-tasks"`
	CanManageHYOK        bool  `jsonapi:"attr,can-manage-hyok"`
	CanQueueApply        bool  `jsonapi:"attr,can-queue-apply"`
	CanQueueDestroy      bool  `jsonapi:"attr,can-queue-destroy"`
	CanQueueRun          bool  `jsonapi:"attr,can-queue-run"`
	CanReadSettings      bool  `jsonapi:"attr,can-read-settings"`
	CanReadStateVersions bool  `jsonapi:"attr,can-read-state-versions"`
	CanReadVariable      bool  `jsonapi:"attr,can-read-variable"`
	CanUnlock            bool  `jsonapi:"attr,can-unlock"`
	CanUpdate            bool  `jsonapi:"attr,can-update"`
	CanUpdateVariable    bool  `jsonapi:"attr,can-update-variable"`
	CanForceDelete       *bool `jsonapi:"attr,can-force-delete"` // pointer b/c it will be useful to check if this property exists, as opposed to having it default to false
}

// WSIncludeOpt represents the available options for include query params.
// https://developer.hashicorp.com/terraform/cloud-docs/api-docs/workspaces#available-related-resources
type WSIncludeOpt string

const (
	WSOrganization               WSIncludeOpt = "organization"
	WSCurrentConfigVer           WSIncludeOpt = "current_configuration_version"
	WSCurrentConfigVerIngress    WSIncludeOpt = "current_configuration_version.ingress_attributes"
	WSCurrentRun                 WSIncludeOpt = "current_run"
	WSCurrentRunPlan             WSIncludeOpt = "current_run.plan"
	WSCurrentRunConfigVer        WSIncludeOpt = "current_run.configuration_version"
	WSCurrentrunConfigVerIngress WSIncludeOpt = "current_run.configuration_version.ingress_attributes"
	WSEffectiveTagBindings       WSIncludeOpt = "effective_tag_bindings"
	WSLockedBy                   WSIncludeOpt = "locked_by"
	WSReadme                     WSIncludeOpt = "readme"
	WSOutputs                    WSIncludeOpt = "outputs"
	WSCurrentStateVer            WSIncludeOpt = "current-state-version"
	WSProject                    WSIncludeOpt = "project"
)

// WorkspaceReadOptions represents the options for reading a workspace.
type WorkspaceReadOptions struct {
	// Optional: A list of relations to include.
	// https://developer.hashicorp.com/terraform/cloud-docs/api-docs/workspaces#available-related-resources
	Include []WSIncludeOpt `url:"include,omitempty"`
}

// WorkspaceListOptions represents the options for listing workspaces.
type WorkspaceListOptions struct {
	ListOptions

	// Optional: A search string (partial workspace name) used to filter the results.
	Search string `url:"search[name],omitempty"`

	// Optional: A search string (comma-separated tag names) used to filter the results.
	Tags string `url:"search[tags],omitempty"`

	// Optional: A search string (comma-separated tag names to exclude) used to filter the results.
	ExcludeTags string `url:"search[exclude-tags],omitempty"`

	// Optional: A search on substring matching to filter the results.
	WildcardName string `url:"search[wildcard-name],omitempty"`

	// Optional: A filter string to list all the workspaces linked to a given project id in the organization.
	ProjectID string `url:"filter[project][id],omitempty"`

	// Optional: A filter string to list all the workspaces filtered by current run status.
	CurrentRunStatus string `url:"filter[current-run][status],omitempty"`

	// Optional: A filter string to list workspaces filtered by key/value tags.
	// These are not annotated and therefore not encoded by go-querystring
	TagBindings []*TagBinding

	// Optional: A list of relations to include. See available resources https://developer.hashicorp.com/terraform/cloud-docs/api-docs/workspaces#available-related-resources
	Include []WSIncludeOpt `url:"include,omitempty"`

	// Optional: May sort on "name" (the default) and "current-run.created-at" (which sorts by the time of the current run)
	// Prepending a hyphen to the sort parameter will reverse the order (e.g. "-name" to reverse the default order)
	Sort string `url:"sort,omitempty"`
}

// WorkspaceCreateOptions represents the options for creating a new workspace.
type WorkspaceCreateOptions struct {
	// Type is a public field utilized by JSON:API to
	// set the resource type via the field tag.
	// It is not a user-defined value and does not need to be set.
	// https://jsonapi.org/format/#crud-creating
	Type string `jsonapi:"primary,workspaces"`

	// Required when: execution-mode is set to agent. The ID of the agent pool
	// belonging to the workspace's organization. This value must not be specified
	// if execution-mode is set to remote or local or if operations is set to true.
	AgentPoolID *string `jsonapi:"attr,agent-pool-id,omitempty"`

	// Optional: Whether destroy plans can be queued on the workspace.
	AllowDestroyPlan *bool `jsonapi:"attr,allow-destroy-plan,omitempty"`

	// Optional: Whether to enable health assessments (drift detection etc.) for the workspace.
	// Reference: https://developer.hashicorp.com/terraform/cloud-docs/api-docs/workspaces#create-a-workspace
	// Requires remote execution mode, HCP Terraform Business entitlement, and a valid agent pool to work
	AssessmentsEnabled *bool `jsonapi:"attr,assessments-enabled,omitempty"`

	// Optional: Whether to automatically apply changes when a Terraform plan is successful.
	AutoApply *bool `jsonapi:"attr,auto-apply,omitempty"`

	// Optional: Whether to automatically apply changes for runs that are created by run triggers
	// from another workspace.
	AutoApplyRunTrigger *bool `jsonapi:"attr,auto-apply-run-trigger,omitempty"`

	// Optional: The time after which an automatic destroy run will be queued
	AutoDestroyAt jsonapi.NullableAttr[time.Time] `jsonapi:"attr,auto-destroy-at,iso8601,omitempty"`

	// Optional: The period of time to wait after workspace activity to trigger a destroy run. The format
	// should roughly match a Go duration string limited to days and hours, e.g. "24h" or "1d".
	AutoDestroyActivityDuration jsonapi.NullableAttr[string] `jsonapi:"attr,auto-destroy-activity-duration,omitempty"`

	// Optional: Whether the workspace inherits auto destroy settings from the project
	InheritsProjectAutoDestroy *bool `jsonapi:"attr,inherits-project-auto-destroy,omitempty"`

	// Optional: A description for the workspace.
	Description *string `jsonapi:"attr,description,omitempty"`

	// Optional: Which execution mode to use. Valid values are remote, local, and agent.
	// When set to local, the workspace will be used for state storage only.
	// This value must not be specified if operations is specified.
	// 'agent' execution mode is not available in Terraform Enterprise.
	ExecutionMode *string `jsonapi:"attr,execution-mode,omitempty"`

	// Optional: Whether to filter runs based on the changed files in a VCS push. If
	// enabled, the working directory and trigger prefixes describe a set of
	// paths which must contain changes for a VCS push to trigger a run. If
	// disabled, any push will trigger a run.
	FileTriggersEnabled *bool `jsonapi:"attr,file-triggers-enabled,omitempty"`

	GlobalRemoteState *bool `jsonapi:"attr,global-remote-state,omitempty"`

	// Optional: Allows the workspace to share remote state at the project level.
	// Default is false.
	ProjectRemoteState *bool `jsonapi:"attr,project-remote-state,omitempty"`

	// Optional: The legacy TFE environment to use as the source of the migration, in the
	// form organization/environment. Omit this unless you are migrating a legacy
	// environment.
	MigrationEnvironment *string `jsonapi:"attr,migration-environment,omitempty"`

	// The name of the workspace, which can only include letters, numbers, -,
	// and _. This will be used as an identifier and must be unique in the
	// organization.
	Name *string `jsonapi:"attr,name"`

	// DEPRECATED. Whether the workspace will use remote or local execution mode.
	// Use ExecutionMode instead.
	Operations *bool `jsonapi:"attr,operations,omitempty"`

	// Whether to queue all runs. Unless this is set to true, runs triggered by
	// a webhook will not be queued until at least one run is manually queued.
	QueueAllRuns *bool `jsonapi:"attr,queue-all-runs,omitempty"`

	// Whether this workspace allows speculative plans. Setting this to false
	// prevents HCP Terraform or the Terraform Enterprise instance from
	// running plans on pull requests, which can improve security if the VCS
	// repository is public or includes untrusted contributors.
	SpeculativeEnabled *bool `jsonapi:"attr,speculative-enabled,omitempty"`

	// BETA. A friendly name for the application or client creating this
	// workspace. If set, this will be displayed on the workspace as
	// "Created via <SOURCE NAME>".
	SourceName *string `jsonapi:"attr,source-name,omitempty"`

	// BETA. A URL for the application or client creating this workspace. This
	// can be the URL of a related resource in another app, or a link to
	// documentation or other info about the client.
	SourceURL *string `jsonapi:"attr,source-url,omitempty"`

	// BETA. Enable the experimental advanced run user interface.
	// This only applies to runs using Terraform version 0.15.2 or newer,
	// and runs executed using older versions will see the classic experience
	// regardless of this setting.
	StructuredRunOutputEnabled *bool `jsonapi:"attr,structured-run-output-enabled,omitempty"`

	// The version of Terraform to use for this workspace. Upon creating a
	// workspace, the latest version is selected unless otherwise specified.
	TerraformVersion *string `jsonapi:"attr,terraform-version,omitempty"`

	// List of repository-root-relative paths which list all locations to be
	// tracked for changes. See FileTriggersEnabled above for more details.
	TriggerPrefixes []string `jsonapi:"attr,trigger-prefixes,omitempty"`

	// Optional: List of patterns used to match against changed files in order
	// to decide whether to trigger a run or not.
	TriggerPatterns []string `jsonapi:"attr,trigger-patterns,omitempty"`

	// Settings for the workspace's VCS repository. If omitted, the workspace is
	// created without a VCS repo. If included, you must specify at least the
	// oauth-token-id and identifier keys below.
	VCSRepo *VCSRepoOptions `jsonapi:"attr,vcs-repo,omitempty"`

	// A relative path that Terraform will execute within. This defaults to the
	// root of your repository and is typically set to a subdirectory matching the
	// environment when multiple environments exist within the same repository.
	WorkingDirectory *string `jsonapi:"attr,working-directory,omitempty"`

	// Optional: Enables HYOK in the workspace.
	// If set to true, the workspace will be created with HYOK enabled.
	// If set to false, the workspace will be created with HYOK disabled.
	// If not specified, the workspace will be created with HYOK disabled.
	// Note: HYOK is only available in HCP Terraform.
	HYOKEnabled *bool `jsonapi:"attr,hyok-enabled,omitempty"`

	// A list of tags to attach to the workspace. If the tag does not already
	// exist, it is created and added to the workspace.
	Tags []*Tag `jsonapi:"relation,tags,omitempty"`

	// Optional: Struct of booleans, which indicate whether the workspace
	// specifies its own values for various settings. If you mark a setting as
	// `false` in this struct, it will clear the workspace's existing value for
	// that setting and defer to the default value that its project or
	// organization provides.
	//
	// In general, it's not necessary to mark a setting as `true` in this
	// struct; if you provide a literal value for a setting, HCP Terraform will
	// automatically update its overwrites field to `true`. If you do choose to
	// manually mark a setting as overwritten, you must provide a value for that
	// setting at the same time.
	SettingOverwrites *WorkspaceSettingOverwritesOptions `jsonapi:"attr,setting-overwrites,omitempty"`

	// Associated Project with the workspace. If not provided, default project
	// of the organization will be assigned to the workspace.
	Project *Project `jsonapi:"relation,project,omitempty"`

	// Associated TagBindings of the workspace.
	TagBindings []*TagBinding `jsonapi:"relation,tag-bindings,omitempty"`
}

// TODO: move this struct out. VCSRepoOptions is used by workspaces, policy sets, and registry modules
// VCSRepoOptions represents the configuration options of a VCS integration.
type VCSRepoOptions struct {
	Branch            *string `json:"branch,omitempty"`
	Identifier        *string `json:"identifier,omitempty"`
	IngressSubmodules *bool   `json:"ingress-submodules,omitempty"`
	OAuthTokenID      *string `json:"oauth-token-id,omitempty"`
	TagsRegex         *string `json:"tags-regex,omitempty"`
	GHAInstallationID *string `json:"github-app-installation-id,omitempty"`
}

type WorkspaceSettingOverwritesOptions struct {
	// If false, the workspace will defer to its organization or project's DefaultExecutionMode value.
	ExecutionMode *bool `json:"execution-mode,omitempty"`
	// If false, the workspace will defer to its organization or project's DefaultAgentPool value.
	AgentPool *bool `json:"agent-pool,omitempty"`
}

// WorkspaceUpdateOptions represents the options for updating a workspace.
type WorkspaceUpdateOptions struct {
	// Type is a public field utilized by JSON:API to
	// set the resource type via the field tag.
	// It is not a user-defined value and does not need to be set.
	// https://jsonapi.org/format/#crud-creating
	Type string `jsonapi:"primary,workspaces"`

	// Required when: execution-mode is set to agent. The ID of the agent pool
	// belonging to the workspace's organization. This value must not be specified
	// if execution-mode is set to remote or local or if operations is set to true.
	AgentPoolID *string `jsonapi:"attr,agent-pool-id,omitempty"`

	// Optional: Whether destroy plans can be queued on the workspace.
	AllowDestroyPlan *bool `jsonapi:"attr,allow-destroy-plan,omitempty"`

	// Optional: Whether to enable health assessments (drift detection etc.) for the workspace.
	// Reference: https://developer.hashicorp.com/terraform/cloud-docs/api-docs/workspaces#update-a-workspace
	// Requires remote execution mode, HCP Terraform Business entitlement, and a valid agent pool to work
	AssessmentsEnabled *bool `jsonapi:"attr,assessments-enabled,omitempty"`

	// Optional: Whether to automatically apply changes when a Terraform plan is successful.
	AutoApply *bool `jsonapi:"attr,auto-apply,omitempty"`

	// Optional: Whether to automatically apply changes for runs that are created by run triggers
	// from another workspace.
	AutoApplyRunTrigger *bool `jsonapi:"attr,auto-apply-run-trigger,omitempty"`

	// Optional: The time after which an automatic destroy run will be queued
	AutoDestroyAt jsonapi.NullableAttr[time.Time] `jsonapi:"attr,auto-destroy-at,iso8601,omitempty"`

	// Optional: The period of time to wait after workspace activity to trigger a destroy run. The format
	// should roughly match a Go duration string limited to days and hours, e.g. "24h" or "1d".
	AutoDestroyActivityDuration jsonapi.NullableAttr[string] `jsonapi:"attr,auto-destroy-activity-duration,omitempty"`

	// Optional: Whether the workspace inherits auto destroy settings from the project
	InheritsProjectAutoDestroy *bool `jsonapi:"attr,inherits-project-auto-destroy,omitempty"`

	// Optional: A new name for the workspace, which can only include letters, numbers, -,
	// and _. This will be used as an identifier and must be unique in the
	// organization. Warning: Changing a workspace's name changes its URL in the
	// API and UI.
	Name *string `jsonapi:"attr,name,omitempty"`

	// Optional: A description for the workspace.
	Description *string `jsonapi:"attr,description,omitempty"`

	// Optional: Which execution mode to use. Valid values are remote, local, and agent.
	// When set to local, the workspace will be used for state storage only.
	// This value must not be specified if operations is specified.
	// 'agent' execution mode is not available in Terraform Enterprise.
	ExecutionMode *string `jsonapi:"attr,execution-mode,omitempty"`

	// Optional: Whether to filter runs based on the changed files in a VCS push. If
	// enabled, the working directory and trigger prefixes describe a set of
	// paths which must contain changes for a VCS push to trigger a run. If
	// disabled, any push will trigger a run.
	FileTriggersEnabled *bool `jsonapi:"attr,file-triggers-enabled,omitempty"`

	// Optional:
	GlobalRemoteState *bool `jsonapi:"attr,global-remote-state,omitempty"`

	// Optional: Allows the workspace to share remote state at the project level.
	// Default is false.
	ProjectRemoteState *bool `jsonapi:"attr,project-remote-state,omitempty"`

	// DEPRECATED. Whether the workspace will use remote or local execution mode.
	// Use ExecutionMode instead.
	Operations *bool `jsonapi:"attr,operations,omitempty"`

	// Optional: Whether to queue all runs. Unless this is set to true, runs triggered by
	// a webhook will not be queued until at least one run is manually queued.
	QueueAllRuns *bool `jsonapi:"attr,queue-all-runs,omitempty"`

	// Optional: Whether this workspace allows speculative plans. Setting this to false
	// prevents HCP Terraform or the Terraform Enterprise instance from
	// running plans on pull requests, which can improve security if the VCS
	// repository is public or includes untrusted contributors.
	SpeculativeEnabled *bool `jsonapi:"attr,speculative-enabled,omitempty"`

	// BETA. Enable the experimental advanced run user interface.
	// This only applies to runs using Terraform version 0.15.2 or newer,
	// and runs executed using older versions will see the classic experience
	// regardless of this setting.
	StructuredRunOutputEnabled *bool `jsonapi:"attr,structured-run-output-enabled,omitempty"`

	// Optional: The version of Terraform to use for this workspace.
	TerraformVersion *string `jsonapi:"attr,terraform-version,omitempty"`

	// Optional: List of repository-root-relative paths which list all locations to be
	// tracked for changes. See FileTriggersEnabled above for more details.
	TriggerPrefixes []string `jsonapi:"attr,trigger-prefixes,omitempty"`

	// Optional: List of patterns used to match against changed files in order
	// to decide whether to trigger a run or not.
	TriggerPatterns []string `jsonapi:"attr,trigger-patterns,omitempty"`

	// Optional: To delete a workspace's existing VCS repo, specify null instead of an
	// object. To modify a workspace's existing VCS repo, include whichever of
	// the keys below you wish to modify. To add a new VCS repo to a workspace
	// that didn't previously have one, include at least the oauth-token-id and
	// identifier keys.
	VCSRepo *VCSRepoOptions `jsonapi:"attr,vcs-repo,omitempty"`

	// Optional: A relative path that Terraform will execute within. This defaults to the
	// root of your repository and is typically set to a subdirectory matching
	// the environment when multiple environments exist within the same
	// repository.
	WorkingDirectory *string `jsonapi:"attr,working-directory,omitempty"`

	// Optional: Struct of booleans, which indicate whether the workspace
	// specifies its own values for various settings. If you mark a setting as
	// `false` in this struct, it will clear the workspace's existing value for
	// that setting and defer to the default value that its project or
	// organization provides.
	//
	// In general, it's not necessary to mark a setting as `true` in this
	// struct; if you provide a literal value for a setting, HCP Terraform will
	// automatically update its overwrites field to `true`. If you do choose to
	// manually mark a setting as overwritten, you must provide a value for that
	// setting at the same time.
	SettingOverwrites *WorkspaceSettingOverwritesOptions `jsonapi:"attr,setting-overwrites,omitempty"`

	// Optional: Enables HYOK in the workspace.
	// If set to true, the workspace will be updated with HYOK enabled.
	// This can't be set to false, as HYOK is a one-way operation.
	HYOKEnabled *bool `jsonapi:"attr,hyok-enabled,omitempty"`

	// Associated Project with the workspace. If not provided, default project
	// of the organization will be assigned to the workspace
	Project *Project `jsonapi:"relation,project,omitempty"`

	// Associated TagBindings of the project. Note that this will replace
	// all existing tag bindings.
	TagBindings []*TagBinding `jsonapi:"relation,tag-bindings,omitempty"`
}

// WorkspaceLockOptions represents the options for locking a workspace.
type WorkspaceLockOptions struct {
	// Specifies the reason for locking the workspace.
	Reason *string `jsonapi:"attr,reason,omitempty"`
}

// workspaceRemoveVCSConnectionOptions
type workspaceRemoveVCSConnectionOptions struct {
	ID      string          `jsonapi:"primary,workspaces"`
	VCSRepo *VCSRepoOptions `jsonapi:"attr,vcs-repo"`
}

// WorkspaceAssignSSHKeyOptions represents the options to assign an SSH key to
// a workspace.
type WorkspaceAssignSSHKeyOptions struct {
	// Type is a public field utilized by JSON:API to
	// set the resource type via the field tag.
	// It is not a user-defined value and does not need to be set.
	// https://jsonapi.org/format/#crud-creating
	Type string `jsonapi:"primary,workspaces"`

	// The SSH key ID to assign.
	SSHKeyID *string `jsonapi:"attr,id"`
}

// workspaceUnassignSSHKeyOptions represents the options to unassign an SSH key
// to a workspace.
type workspaceUnassignSSHKeyOptions struct {
	// Type is a public field utilized by JSON:API to
	// set the resource type via the field tag.
	// It is not a user-defined value and does not need to be set.
	// https://jsonapi.org/format/#crud-creating
	Type string `jsonapi:"primary,workspaces"`

	// Must be nil to unset the currently assigned SSH key.
	SSHKeyID *string `jsonapi:"attr,id"`
}

type RemoteStateConsumersListOptions struct {
	ListOptions
}

// WorkspaceAddRemoteStateConsumersOptions represents the options for adding remote state consumers
// to a workspace.
type WorkspaceAddRemoteStateConsumersOptions struct {
	// The workspaces to add as remote state consumers to the workspace.
	Workspaces []*Workspace
}

// WorkspaceRemoveRemoteStateConsumersOptions represents the options for removing remote state
// consumers from a workspace.
type WorkspaceRemoveRemoteStateConsumersOptions struct {
	// The workspaces to remove as remote state consumers from the workspace.
	Workspaces []*Workspace
}

// WorkspaceUpdateRemoteStateConsumersOptions represents the options for
// updatintg remote state consumers from a workspace.
type WorkspaceUpdateRemoteStateConsumersOptions struct {
	// The workspaces to update remote state consumers for the workspace.
	Workspaces []*Workspace
}

type WorkspaceTagListOptions struct {
	ListOptions

	// A query string used to filter workspace tags.
	// Any workspace tag with a name partially matching this value will be returned.
	Query *string `url:"name,omitempty"`
}

type WorkspaceAddTagsOptions struct {
	Tags []*Tag
}

type WorkspaceRemoveTagsOptions struct {
	Tags []*Tag
}

// List all the workspaces within an organization.
func (s *workspaces) List(ctx context.Context, organization string, options *WorkspaceListOptions) (*WorkspaceList, error) {
	if !validStringID(&organization) {
		return nil, ErrInvalidOrg
	}
	if err := options.valid(); err != nil {
		return nil, err
	}

	var tagFilters map[string][]string
	if options != nil {
		tagFilters = encodeTagFiltersAsParams(options.TagBindings)
	}

	// Encode parameters that cannot be encoded by go-querystring
	u := fmt.Sprintf("organizations/%s/workspaces", url.PathEscape(organization))
	req, err := s.client.NewRequestWithAdditionalQueryParams("GET", u, options, tagFilters)
	if err != nil {
		return nil, err
	}

	wl := &WorkspaceList{}
	err = req.Do(ctx, wl)
	if err != nil {
		return nil, err
	}

	return wl, nil
}

func (s *workspaces) ListTagBindings(ctx context.Context, workspaceID string) ([]*TagBinding, error) {
	if !validStringID(&workspaceID) {
		return nil, ErrInvalidWorkspaceID
	}

	u := fmt.Sprintf("workspaces/%s/tag-bindings", url.PathEscape(workspaceID))
	req, err := s.client.NewRequest("GET", u, nil)
	if err != nil {
		return nil, err
	}

	var list struct {
		*Pagination
		Items []*TagBinding
	}

	err = req.Do(ctx, &list)
	if err != nil {
		return nil, err
	}

	return list.Items, nil
}

func (s *workspaces) ListEffectiveTagBindings(ctx context.Context, workspaceID string) ([]*EffectiveTagBinding, error) {
	if !validStringID(&workspaceID) {
		return nil, ErrInvalidWorkspaceID
	}

	u := fmt.Sprintf("workspaces/%s/effective-tag-bindings", url.PathEscape(workspaceID))
	req, err := s.client.NewRequest("GET", u, nil)
	if err != nil {
		return nil, err
	}

	var list struct {
		*Pagination
		Items []*EffectiveTagBinding
	}

	err = req.Do(ctx, &list)
	if err != nil {
		return nil, err
	}

	return list.Items, nil
}

// AddTagBindings adds or modifies the value of existing tag binding keys for a workspace.
func (s *workspaces) AddTagBindings(ctx context.Context, workspaceID string, options WorkspaceAddTagBindingsOptions) ([]*TagBinding, error) {
	if !validStringID(&workspaceID) {
		return nil, ErrInvalidWorkspaceID
	}

	if err := options.valid(); err != nil {
		return nil, err
	}

	u := fmt.Sprintf("workspaces/%s/tag-bindings", url.PathEscape(workspaceID))
	req, err := s.client.NewRequest("PATCH", u, options.TagBindings)
	if err != nil {
		return nil, err
	}

	var response = struct {
		*Pagination
		Items []*TagBinding
	}{}
	err = req.Do(ctx, &response)

	return response.Items, err
}

// DeleteAllTagBindings removes all tag bindings associated with a workspace.
// This method will not remove any inherited tag bindings, which must be
// explicitly removed from the parent project.
func (s *workspaces) DeleteAllTagBindings(ctx context.Context, workspaceID string) error {
	if !validStringID(&workspaceID) {
		return ErrInvalidWorkspaceID
	}

	type aliasOpts struct {
		Type        string        `jsonapi:"primary,workspaces"`
		TagBindings []*TagBinding `jsonapi:"relation,tag-bindings"`
	}

	opts := &aliasOpts{
		TagBindings: []*TagBinding{},
	}

	u := fmt.Sprintf("workspaces/%s", url.PathEscape(workspaceID))
	req, err := s.client.NewRequest("PATCH", u, opts)
	if err != nil {
		return err
	}

	return req.Do(ctx, nil)
}

// Create is used to create a new workspace.
func (s *workspaces) Create(ctx context.Context, organization string, options WorkspaceCreateOptions) (*Workspace, error) {
	if !validStringID(&organization) {
		return nil, ErrInvalidOrg
	}
	if err := options.valid(); err != nil {
		return nil, err
	}

	u := fmt.Sprintf("organizations/%s/workspaces", url.PathEscape(organization))
	req, err := s.client.NewRequest("POST", u, &options)
	if err != nil {
		return nil, err
	}

	w := &Workspace{}
	err = req.Do(ctx, w)
	if err != nil {
		return nil, err
	}

	return w, nil
}

// Read a workspace by its name and organization name.
func (s *workspaces) Read(ctx context.Context, organization, workspace string) (*Workspace, error) {
	return s.ReadWithOptions(ctx, organization, workspace, nil)
}

// ReadWithOptions reads a workspace by name and organization name with given options.
func (s *workspaces) ReadWithOptions(ctx context.Context, organization, workspace string, options *WorkspaceReadOptions) (*Workspace, error) {
	if !validStringID(&organization) {
		return nil, ErrInvalidOrg
	}
	if !validStringID(&workspace) {
		return nil, ErrInvalidWorkspaceValue
	}
	if err := options.valid(); err != nil {
		return nil, err
	}

	u := fmt.Sprintf(
		"organizations/%s/workspaces/%s",
		url.PathEscape(organization),
		url.PathEscape(workspace),
	)
	req, err := s.client.NewRequest("GET", u, options)
	if err != nil {
		return nil, err
	}

	w := &Workspace{}
	err = req.Do(ctx, w)
	if err != nil {
		return nil, err
	}

	// Manually populate the deprecated DataRetentionPolicy field
	w.DataRetentionPolicy = w.DataRetentionPolicyChoice.ConvertToLegacyStruct()

	// durations come over in ms
	w.ApplyDurationAverage *= time.Millisecond
	w.PlanDurationAverage *= time.Millisecond

	return w, nil
}

// ReadByID reads a workspace by its ID.
func (s *workspaces) ReadByID(ctx context.Context, workspaceID string) (*Workspace, error) {
	return s.ReadByIDWithOptions(ctx, workspaceID, nil)
}

// ReadByIDWithOptions reads a workspace by its ID with the given options.
func (s *workspaces) ReadByIDWithOptions(ctx context.Context, workspaceID string, options *WorkspaceReadOptions) (*Workspace, error) {
	if !validStringID(&workspaceID) {
		return nil, ErrInvalidWorkspaceID
	}

	u := fmt.Sprintf("workspaces/%s", url.PathEscape(workspaceID))
	req, err := s.client.NewRequest("GET", u, options)
	if err != nil {
		return nil, err
	}

	w := &Workspace{}
	err = req.Do(ctx, w)
	if err != nil {
		return nil, err
	}

	// Manually populate the deprecated DataRetentionPolicy field
	if w.DataRetentionPolicyChoice != nil {
		w.DataRetentionPolicy = w.DataRetentionPolicyChoice.ConvertToLegacyStruct()
	}

	// durations come over in ms
	w.ApplyDurationAverage *= time.Millisecond
	w.PlanDurationAverage *= time.Millisecond

	return w, nil
}

// Readme gets the readme of a workspace by its ID.
func (s *workspaces) Readme(ctx context.Context, workspaceID string) (io.Reader, error) {
	if !validStringID(&workspaceID) {
		return nil, ErrInvalidWorkspaceID
	}

	u := fmt.Sprintf("workspaces/%s?include=readme", url.PathEscape(workspaceID))
	req, err := s.client.NewRequest("GET", u, nil)
	if err != nil {
		return nil, err
	}

	r := &workspaceWithReadme{}
	err = req.Do(ctx, r)
	if err != nil {
		return nil, err
	}
	if r.Readme == nil {
		return nil, nil
	}

	return strings.NewReader(r.Readme.RawMarkdown), nil
}

// Update settings of an existing workspace.
func (s *workspaces) Update(ctx context.Context, organization, workspace string, options WorkspaceUpdateOptions) (*Workspace, error) {
	if !validStringID(&organization) {
		return nil, ErrInvalidOrg
	}
	if !validStringID(&workspace) {
		return nil, ErrInvalidWorkspaceValue
	}
	if err := options.valid(); err != nil {
		return nil, err
	}

	u := fmt.Sprintf(
		"organizations/%s/workspaces/%s",
		url.PathEscape(organization),
		url.PathEscape(workspace),
	)
	req, err := s.client.NewRequest("PATCH", u, &options)
	if err != nil {
		return nil, err
	}

	w := &Workspace{}
	err = req.Do(ctx, w)
	if err != nil {
		return nil, err
	}

	return w, nil
}

// UpdateByID updates the settings of an existing workspace.
func (s *workspaces) UpdateByID(ctx context.Context, workspaceID string, options WorkspaceUpdateOptions) (*Workspace, error) {
	if !validStringID(&workspaceID) {
		return nil, ErrInvalidWorkspaceID
	}

	u := fmt.Sprintf("workspaces/%s", url.PathEscape(workspaceID))
	req, err := s.client.NewRequest("PATCH", u, &options)
	if err != nil {
		return nil, err
	}

	w := &Workspace{}
	err = req.Do(ctx, w)
	if err != nil {
		return nil, err
	}

	return w, nil
}

// Delete a workspace by its name.
func (s *workspaces) Delete(ctx context.Context, organization, workspace string) error {
	if !validStringID(&organization) {
		return ErrInvalidOrg
	}
	if !validStringID(&workspace) {
		return ErrInvalidWorkspaceValue
	}

	u := fmt.Sprintf(
		"organizations/%s/workspaces/%s",
		url.PathEscape(organization),
		url.PathEscape(workspace),
	)
	req, err := s.client.NewRequest("DELETE", u, nil)
	if err != nil {
		return err
	}

	return req.Do(ctx, nil)
}

// DeleteByID deletes a workspace by its ID.
func (s *workspaces) DeleteByID(ctx context.Context, workspaceID string) error {
	if !validStringID(&workspaceID) {
		return ErrInvalidWorkspaceID
	}

	u := fmt.Sprintf("workspaces/%s", url.PathEscape(workspaceID))
	req, err := s.client.NewRequest("DELETE", u, nil)
	if err != nil {
		return err
	}

	return req.Do(ctx, nil)
}

// SafeDelete a workspace by its name.
func (s *workspaces) SafeDelete(ctx context.Context, organization, workspace string) error {
	if !validStringID(&organization) {
		return ErrInvalidOrg
	}
	if !validStringID(&workspace) {
		return ErrInvalidWorkspaceValue
	}

	u := fmt.Sprintf(
		"organizations/%s/workspaces/%s/actions/safe-delete",
		url.PathEscape(organization),
		url.PathEscape(workspace),
	)
	req, err := s.client.NewRequest("POST", u, nil)
	if err != nil {
		return err
	}

	return req.Do(ctx, nil)
}

// SafeDeleteByID safely deletes a workspace by its ID.
func (s *workspaces) SafeDeleteByID(ctx context.Context, workspaceID string) error {
	if !validStringID(&workspaceID) {
		return ErrInvalidWorkspaceID
	}

	u := fmt.Sprintf("workspaces/%s/actions/safe-delete", url.PathEscape(workspaceID))
	req, err := s.client.NewRequest("POST", u, nil)
	if err != nil {
		return err
	}

	return req.Do(ctx, nil)
}

// RemoveVCSConnection from a workspace.
func (s *workspaces) RemoveVCSConnection(ctx context.Context, organization, workspace string) (*Workspace, error) {
	if !validStringID(&organization) {
		return nil, ErrInvalidOrg
	}
	if !validStringID(&workspace) {
		return nil, ErrInvalidWorkspaceValue
	}

	u := fmt.Sprintf(
		"organizations/%s/workspaces/%s",
		url.PathEscape(organization),
		url.PathEscape(workspace),
	)

	req, err := s.client.NewRequest("PATCH", u, &workspaceRemoveVCSConnectionOptions{})
	if err != nil {
		return nil, err
	}

	w := &Workspace{}
	err = req.Do(ctx, w)
	if err != nil {
		return nil, err
	}

	return w, nil
}

// RemoveVCSConnectionByID removes a VCS connection from a workspace.
func (s *workspaces) RemoveVCSConnectionByID(ctx context.Context, workspaceID string) (*Workspace, error) {
	if !validStringID(&workspaceID) {
		return nil, ErrInvalidWorkspaceID
	}

	u := fmt.Sprintf("workspaces/%s", url.PathEscape(workspaceID))

	req, err := s.client.NewRequest("PATCH", u, &workspaceRemoveVCSConnectionOptions{})
	if err != nil {
		return nil, err
	}

	w := &Workspace{}
	err = req.Do(ctx, w)
	if err != nil {
		return nil, err
	}

	return w, nil
}

// Lock a workspace by its ID.
func (s *workspaces) Lock(ctx context.Context, workspaceID string, options WorkspaceLockOptions) (*Workspace, error) {
	if !validStringID(&workspaceID) {
		return nil, ErrInvalidWorkspaceID
	}

	u := fmt.Sprintf("workspaces/%s/actions/lock", url.PathEscape(workspaceID))
	req, err := s.client.NewRequest("POST", u, &options)
	if err != nil {
		return nil, err
	}

	w := &Workspace{}
	err = req.Do(ctx, w)
	if err != nil {
		return nil, err
	}

	return w, nil
}

// Unlock a workspace by its ID.
func (s *workspaces) Unlock(ctx context.Context, workspaceID string) (*Workspace, error) {
	if !validStringID(&workspaceID) {
		return nil, ErrInvalidWorkspaceID
	}

	u := fmt.Sprintf("workspaces/%s/actions/unlock", url.PathEscape(workspaceID))
	req, err := s.client.NewRequest("POST", u, nil)
	if err != nil {
		return nil, err
	}

	w := &Workspace{}
	err = req.Do(ctx, w)
	if err != nil {
		if strings.Contains(err.Error(), "latest state version is still pending") {
			return nil, ErrWorkspaceLockedStateVersionStillPending
		}
		return nil, err
	}

	return w, nil
}

// ForceUnlock a workspace by its ID.
func (s *workspaces) ForceUnlock(ctx context.Context, workspaceID string) (*Workspace, error) {
	if !validStringID(&workspaceID) {
		return nil, ErrInvalidWorkspaceID
	}

	u := fmt.Sprintf("workspaces/%s/actions/force-unlock", url.PathEscape(workspaceID))
	req, err := s.client.NewRequest("POST", u, nil)
	if err != nil {
		return nil, err
	}

	w := &Workspace{}
	err = req.Do(ctx, w)
	if err != nil {
		return nil, err
	}

	return w, nil
}

// AssignSSHKey to a workspace.
func (s *workspaces) AssignSSHKey(ctx context.Context, workspaceID string, options WorkspaceAssignSSHKeyOptions) (*Workspace, error) {
	if !validStringID(&workspaceID) {
		return nil, ErrInvalidWorkspaceID
	}
	if err := options.valid(); err != nil {
		return nil, err
	}

	u := fmt.Sprintf("workspaces/%s/relationships/ssh-key", url.PathEscape(workspaceID))
	req, err := s.client.NewRequest("PATCH", u, &options)
	if err != nil {
		return nil, err
	}

	w := &Workspace{}
	err = req.Do(ctx, w)
	if err != nil {
		return nil, err
	}

	return w, nil
}

// UnassignSSHKey from a workspace.
func (s *workspaces) UnassignSSHKey(ctx context.Context, workspaceID string) (*Workspace, error) {
	if !validStringID(&workspaceID) {
		return nil, ErrInvalidWorkspaceID
	}

	u := fmt.Sprintf("workspaces/%s/relationships/ssh-key", url.PathEscape(workspaceID))
	req, err := s.client.NewRequest("PATCH", u, &workspaceUnassignSSHKeyOptions{})
	if err != nil {
		return nil, err
	}

	w := &Workspace{}
	err = req.Do(ctx, w)
	if err != nil {
		return nil, err
	}

	return w, nil
}

// RemoteStateConsumers returns the remote state consumers for a given workspace.
func (s *workspaces) ListRemoteStateConsumers(ctx context.Context, workspaceID string, options *RemoteStateConsumersListOptions) (*WorkspaceList, error) {
	if !validStringID(&workspaceID) {
		return nil, ErrInvalidWorkspaceID
	}

	u := fmt.Sprintf("workspaces/%s/relationships/remote-state-consumers", url.PathEscape(workspaceID))

	req, err := s.client.NewRequest("GET", u, options)
	if err != nil {
		return nil, err
	}

	wl := &WorkspaceList{}
	err = req.Do(ctx, wl)
	if err != nil {
		return nil, err
	}

	return wl, nil
}

// AddRemoteStateConsumere adds the remote state consumers to a given workspace.
func (s *workspaces) AddRemoteStateConsumers(ctx context.Context, workspaceID string, options WorkspaceAddRemoteStateConsumersOptions) error {
	if !validStringID(&workspaceID) {
		return ErrInvalidWorkspaceID
	}
	if err := options.valid(); err != nil {
		return err
	}

	u := fmt.Sprintf("workspaces/%s/relationships/remote-state-consumers", url.PathEscape(workspaceID))
	req, err := s.client.NewRequest("POST", u, options.Workspaces)
	if err != nil {
		return err
	}

	return req.Do(ctx, nil)
}

// RemoveRemoteStateConsumers removes the remote state consumers for a given workspace.
func (s *workspaces) RemoveRemoteStateConsumers(ctx context.Context, workspaceID string, options WorkspaceRemoveRemoteStateConsumersOptions) error {
	if !validStringID(&workspaceID) {
		return ErrInvalidWorkspaceID
	}
	if err := options.valid(); err != nil {
		return err
	}

	u := fmt.Sprintf("workspaces/%s/relationships/remote-state-consumers", url.PathEscape(workspaceID))
	req, err := s.client.NewRequest("DELETE", u, options.Workspaces)
	if err != nil {
		return err
	}

	return req.Do(ctx, nil)
}

// UpdateRemoteStateConsumers removes the remote state consumers for a given workspace.
func (s *workspaces) UpdateRemoteStateConsumers(ctx context.Context, workspaceID string, options WorkspaceUpdateRemoteStateConsumersOptions) error {
	if !validStringID(&workspaceID) {
		return ErrInvalidWorkspaceID
	}
	if err := options.valid(); err != nil {
		return err
	}

	u := fmt.Sprintf("workspaces/%s/relationships/remote-state-consumers", url.PathEscape(workspaceID))
	req, err := s.client.NewRequest("PATCH", u, options.Workspaces)
	if err != nil {
		return err
	}

	return req.Do(ctx, nil)
}

// ListTags returns the tags for a given workspace.
func (s *workspaces) ListTags(ctx context.Context, workspaceID string, options *WorkspaceTagListOptions) (*TagList, error) {
	if !validStringID(&workspaceID) {
		return nil, ErrInvalidWorkspaceID
	}

	u := fmt.Sprintf("workspaces/%s/relationships/tags", url.PathEscape(workspaceID))

	req, err := s.client.NewRequest("GET", u, options)
	if err != nil {
		return nil, err
	}

	tl := &TagList{}
	err = req.Do(ctx, tl)
	if err != nil {
		return nil, err
	}

	return tl, nil
}

// AddTags adds a list of tags to a workspace.
func (s *workspaces) AddTags(ctx context.Context, workspaceID string, options WorkspaceAddTagsOptions) error {
	if !validStringID(&workspaceID) {
		return ErrInvalidWorkspaceID
	}
	if err := options.valid(); err != nil {
		return err
	}

	u := fmt.Sprintf("workspaces/%s/relationships/tags", url.PathEscape(workspaceID))
	req, err := s.client.NewRequest("POST", u, options.Tags)
	if err != nil {
		return err
	}

	return req.Do(ctx, nil)
}

// RemoveTags removes a list of tags from a workspace.
func (s *workspaces) RemoveTags(ctx context.Context, workspaceID string, options WorkspaceRemoveTagsOptions) error {
	if !validStringID(&workspaceID) {
		return ErrInvalidWorkspaceID
	}
	if err := options.valid(); err != nil {
		return err
	}

	u := fmt.Sprintf("workspaces/%s/relationships/tags", url.PathEscape(workspaceID))
	req, err := s.client.NewRequest("DELETE", u, options.Tags)
	if err != nil {
		return err
	}

	return req.Do(ctx, nil)
}

func (s *workspaces) ReadDataRetentionPolicy(ctx context.Context, workspaceID string) (*DataRetentionPolicy, error) {
	if !validStringID(&workspaceID) {
		return nil, ErrInvalidWorkspaceID
	}

	u := fmt.Sprintf("workspaces/%s/relationships/data-retention-policy", url.PathEscape(workspaceID))
	req, err := s.client.NewRequest("GET", u, nil)
	if err != nil {
		return nil, err
	}

	dataRetentionPolicy := &DataRetentionPolicy{}
	err = req.Do(ctx, dataRetentionPolicy)

	if err != nil {
		// try to detect known issue where this function is used with TFE >= 202401,
		// and direct user towards the V2 function
		if drpUnmarshalEr.MatchString(err.Error()) {
			return nil, fmt.Errorf("error reading deprecated DataRetentionPolicy, use ReadDataRetentionPolicyChoice instead")
		}
		return nil, err
	}

	return dataRetentionPolicy, nil
}

func (s *workspaces) ReadDataRetentionPolicyChoice(ctx context.Context, workspaceID string) (*DataRetentionPolicyChoice, error) {
	if !validStringID(&workspaceID) {
		return nil, ErrInvalidWorkspaceID
	}

	// The API to read the drp is workspaces/<id>/relationships/data-retention-policy
	// However, this API can return multiple "types" (e.g. data-retention-policy-delete-olders, or data-retention-policy-dont-deletes)
	// Ideally we would deserialize this directly into the choice type (DataRetentionPolicyChoice)...however, there isn't a way to
	// tell the current jsonapi implementation that the direct result of an endpoint could be different types. Relationships can be polymorphic,
	// but the direct result of an endpoint can't be (as far as the jsonapi implementation is concerned)

	// Instead, we need to figure out the type of the data retention policy first, and deserialize it into the matching model. We
	// can then create a choice type manually
	ws, err := s.ReadByID(ctx, workspaceID)
	if err != nil {
		return nil, err
	}

	// there is no drp (of a known type)
	if ws.DataRetentionPolicyChoice == nil || !ws.DataRetentionPolicyChoice.IsPopulated() {
		return ws.DataRetentionPolicyChoice, nil
	}

	u := s.dataRetentionPolicyLink(workspaceID)
	req, err := s.client.NewRequest("GET", u, nil)
	if err != nil {
		return nil, err
	}

	dataRetentionPolicy := &DataRetentionPolicyChoice{}
	// if reading the workspace told us it was a "delete older policy" deserialize into the DeleteOlder portion of the choice model
	if ws.DataRetentionPolicyChoice.DataRetentionPolicyDeleteOlder != nil {
		deleteOlder := &DataRetentionPolicyDeleteOlder{}
		err = req.Do(ctx, deleteOlder)
		dataRetentionPolicy.DataRetentionPolicyDeleteOlder = deleteOlder

		// if reading the workspace told us it was a "delete older policy" deserialize into the DeleteOlder portion of the choice model
	} else if ws.DataRetentionPolicyChoice.DataRetentionPolicyDontDelete != nil {
		dontDelete := &DataRetentionPolicyDontDelete{}
		err = req.Do(ctx, dontDelete)
		dataRetentionPolicy.DataRetentionPolicyDontDelete = dontDelete
	} else if ws.DataRetentionPolicyChoice != nil {
		legacyDrp := &DataRetentionPolicy{}
		err = req.Do(ctx, legacyDrp)
		dataRetentionPolicy.DataRetentionPolicy = legacyDrp
	}

	if err != nil {
		return nil, err
	}

	return dataRetentionPolicy, nil
}

func (s *workspaces) SetDataRetentionPolicy(ctx context.Context, workspaceID string, options DataRetentionPolicySetOptions) (*DataRetentionPolicy, error) {
	if !validStringID(&workspaceID) {
		return nil, ErrInvalidWorkspaceID
	}

	u := s.dataRetentionPolicyLink(workspaceID)
	req, err := s.client.NewRequest("PATCH", u, &options)
	if err != nil {
		return nil, err
	}

	dataRetentionPolicy := &DataRetentionPolicy{}
	err = req.Do(ctx, dataRetentionPolicy)

	if err != nil {
		return nil, err
	}

	return dataRetentionPolicy, nil
}

func (s *workspaces) SetDataRetentionPolicyDeleteOlder(ctx context.Context, workspaceID string, options DataRetentionPolicyDeleteOlderSetOptions) (*DataRetentionPolicyDeleteOlder, error) {
	if !validStringID(&workspaceID) {
		return nil, ErrInvalidWorkspaceID
	}

	u := s.dataRetentionPolicyLink(workspaceID)
	req, err := s.client.NewRequest("POST", u, &options)
	if err != nil {
		return nil, err
	}

	dataRetentionPolicy := &DataRetentionPolicyDeleteOlder{}
	err = req.Do(ctx, dataRetentionPolicy)

	if err != nil {
		return nil, err
	}

	return dataRetentionPolicy, nil
}

func (s *workspaces) SetDataRetentionPolicyDontDelete(ctx context.Context, workspaceID string, options DataRetentionPolicyDontDeleteSetOptions) (*DataRetentionPolicyDontDelete, error) {
	if !validStringID(&workspaceID) {
		return nil, ErrInvalidWorkspaceID
	}

	u := s.dataRetentionPolicyLink(workspaceID)
	req, err := s.client.NewRequest("POST", u, &options)
	if err != nil {
		return nil, err
	}

	dataRetentionPolicy := &DataRetentionPolicyDontDelete{}
	err = req.Do(ctx, dataRetentionPolicy)

	if err != nil {
		return nil, err
	}

	return dataRetentionPolicy, nil
}

func (s *workspaces) DeleteDataRetentionPolicy(ctx context.Context, workspaceID string) error {
	if !validStringID(&workspaceID) {
		return ErrInvalidWorkspaceID
	}

	u := s.dataRetentionPolicyLink(workspaceID)
	req, err := s.client.NewRequest("DELETE", u, nil)
	if err != nil {
		return err
	}

	return req.Do(ctx, nil)
}

func (o WorkspaceAddTagBindingsOptions) valid() error {
	if len(o.TagBindings) == 0 {
		return ErrRequiredTagBindings
	}

	return nil
}

func (o WorkspaceCreateOptions) valid() error {
	if !validString(o.Name) {
		return ErrRequiredName
	}
	if !validStringID(o.Name) {
		return ErrInvalidName
	}
	if o.Operations != nil && o.ExecutionMode != nil {
		return ErrUnsupportedOperations
	}
	if o.AgentPoolID != nil && (o.ExecutionMode == nil || *o.ExecutionMode != "agent") {
		return ErrRequiredAgentMode
	}
	if o.AgentPoolID == nil && (o.ExecutionMode != nil && *o.ExecutionMode == "agent") {
		return ErrRequiredAgentPoolID
	}
	if len(o.TriggerPrefixes) > 0 &&
		o.TriggerPatterns != nil && len(o.TriggerPatterns) > 0 {
		return ErrUnsupportedBothTriggerPatternsAndPrefixes
	}
	if tagRegexDefined(o.VCSRepo) &&
		o.TriggerPatterns != nil && len(o.TriggerPatterns) > 0 {
		return ErrUnsupportedBothTagsRegexAndTriggerPatterns
	}
	if tagRegexDefined(o.VCSRepo) &&
		o.TriggerPrefixes != nil && len(o.TriggerPrefixes) > 0 {
		return ErrUnsupportedBothTagsRegexAndTriggerPrefixes
	}
	if tagRegexDefined(o.VCSRepo) &&
		o.FileTriggersEnabled != nil && *o.FileTriggersEnabled {
		return ErrUnsupportedBothTagsRegexAndFileTriggersEnabled
	}

	return nil
}

func (o WorkspaceUpdateOptions) valid() error {
	if o.Name != nil && !validStringID(o.Name) {
		return ErrInvalidName
	}
	if o.Operations != nil && o.ExecutionMode != nil {
		return ErrUnsupportedOperations
	}
	if o.AgentPoolID == nil && (o.ExecutionMode != nil && *o.ExecutionMode == "agent") {
		return ErrRequiredAgentPoolID
	}
	if len(o.TriggerPrefixes) > 0 &&
		o.TriggerPatterns != nil && len(o.TriggerPatterns) > 0 {
		return ErrUnsupportedBothTriggerPatternsAndPrefixes
	}

	if tagRegexDefined(o.VCSRepo) &&
		o.TriggerPatterns != nil && len(o.TriggerPatterns) > 0 {
		return ErrUnsupportedBothTagsRegexAndTriggerPatterns
	}
	if tagRegexDefined(o.VCSRepo) &&
		o.TriggerPrefixes != nil && len(o.TriggerPrefixes) > 0 {
		return ErrUnsupportedBothTagsRegexAndTriggerPrefixes
	}
	if tagRegexDefined(o.VCSRepo) &&
		o.FileTriggersEnabled != nil && *o.FileTriggersEnabled {
		return ErrUnsupportedBothTagsRegexAndFileTriggersEnabled
	}

	return nil
}

func (o WorkspaceAssignSSHKeyOptions) valid() error {
	if !validString(o.SSHKeyID) {
		return ErrRequiredSHHKeyID
	}
	if !validStringID(o.SSHKeyID) {
		return ErrInvalidSHHKeyID
	}
	return nil
}

func (o WorkspaceAddRemoteStateConsumersOptions) valid() error {
	if o.Workspaces == nil {
		return ErrWorkspacesRequired
	}
	if len(o.Workspaces) == 0 {
		return ErrWorkspaceMinLimit
	}
	return nil
}

func (o WorkspaceRemoveRemoteStateConsumersOptions) valid() error {
	if o.Workspaces == nil {
		return ErrWorkspacesRequired
	}
	if len(o.Workspaces) == 0 {
		return ErrWorkspaceMinLimit
	}
	return nil
}

func (o WorkspaceUpdateRemoteStateConsumersOptions) valid() error {
	if o.Workspaces == nil {
		return ErrWorkspacesRequired
	}
	if len(o.Workspaces) == 0 {
		return ErrWorkspaceMinLimit
	}
	return nil
}

func (o WorkspaceAddTagsOptions) valid() error {
	if len(o.Tags) == 0 {
		return ErrMissingTagIdentifier
	}
	for _, s := range o.Tags {
		if s.Name == "" && s.ID == "" {
			return ErrMissingTagIdentifier
		}
	}

	return nil
}

func (o WorkspaceRemoveTagsOptions) valid() error {
	if len(o.Tags) == 0 {
		return ErrMissingTagIdentifier
	}
	for _, s := range o.Tags {
		if s.Name == "" && s.ID == "" {
			return ErrMissingTagIdentifier
		}
	}

	return nil
}

func (o *WorkspaceListOptions) valid() error {
	return nil
}

func (o *WorkspaceReadOptions) valid() error {
	return nil
}

func tagRegexDefined(options *VCSRepoOptions) bool {
	if options == nil {
		return false
	}
	if options.TagsRegex != nil && *options.TagsRegex != "" {
		return true
	}
	return false
}

func (s *workspaces) dataRetentionPolicyLink(wsID string) string {
	return fmt.Sprintf("workspaces/%s/relationships/data-retention-policy", url.PathEscape(wsID))
}

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
