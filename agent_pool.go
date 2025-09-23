// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package tfe

import (
	"context"
	"fmt"
	"net/url"
	"time"
)

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
