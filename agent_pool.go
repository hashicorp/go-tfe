package tfe

import (
	"context"
	"fmt"
	"net/url"
)

// Compile-time proof of interface implementation.
var _ AgentPools = (*agentPools)(nil)

// AgentPools describes all the agent pool related methods that the Terraform
// Cloud API supports. Note that agents are not available in Terraform Enterprise.
//
// TFE API docs: https://www.terraform.io/docs/cloud/api/agents.html
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

// AgentPool represents a Terraform Cloud agent pool.
type AgentPool struct {
	ID                 string `jsonapi:"primary,agent-pools"`
	Name               string `jsonapi:"attr,name"`
	OrganizationScoped bool   `jsonapi:"attr,organization-scoped"`

	// Relations
	Organization      *Organization `jsonapi:"relation,organization"`
	Workspaces        []*Workspace  `jsonapi:"relation,workspaces"`
	AllowedWorkspaces []*Workspace  `jsonapi:"relation,allowed-workspaces"`
}

// A list of relations to include
// https://www.terraform.io/cloud-docs/api-docs/agents#available-related-resources
type AgentPoolIncludeOpt string

const AgentPoolWorkspaces AgentPoolIncludeOpt = "workspaces"

type AgentPoolReadOptions struct {
	Include []AgentPoolIncludeOpt `url:"include,omitempty"`
}

// AgentPoolListOptions represents the options for listing agent pools.
type AgentPoolListOptions struct {
	ListOptions
	// Optional: A list of relations to include. See available resources
	// https://www.terraform.io/cloud-docs/api-docs/agents#available-related-resources
	Include []AgentPoolIncludeOpt `url:"include,omitempty"`

	// Optional: A search query string used to filter agent pool. Agent pools are searchable by name
	Query string `url:"q,omitempty"`
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
}

// List all the agent pools of the given organization.
func (s *agentPools) List(ctx context.Context, organization string, options *AgentPoolListOptions) (*AgentPoolList, error) {
	if !validStringID(&organization) {
		return nil, ErrInvalidOrg
	}
	if err := options.valid(); err != nil {
		return nil, err
	}

	u := fmt.Sprintf("organizations/%s/agent-pools", url.QueryEscape(organization))
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

	u := fmt.Sprintf("organizations/%s/agent-pools", url.QueryEscape(organization))
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

	u := fmt.Sprintf("agent-pools/%s", url.QueryEscape(agentpoolID))
	req, err := s.client.NewRequest("GET", u, nil)
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
	Name *string `jsonapi:"attr,name"`

	// True if the agent pool is organization scoped, false otherwise.
	OrganizationScoped *bool `jsonapi:"attr,organization-scoped,omitempty"`

	// A new list of workspaces that are associated with an agent pool.
	AllowedWorkspaces []*Workspace `jsonapi:"relation,allowed-workspaces"`
}

// Update an agent pool by its ID.
func (s *agentPools) Update(ctx context.Context, agentPoolID string, options AgentPoolUpdateOptions) (*AgentPool, error) {
	if !validStringID(&agentPoolID) {
		return nil, ErrInvalidAgentPoolID
	}

	if err := options.valid(); err != nil {
		return nil, err
	}

	u := fmt.Sprintf("agent-pools/%s", url.QueryEscape(agentPoolID))
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

// Delete an agent pool by its ID.
func (s *agentPools) Delete(ctx context.Context, agentPoolID string) error {
	if !validStringID(&agentPoolID) {
		return ErrInvalidAgentPoolID
	}

	u := fmt.Sprintf("agent-pools/%s", url.QueryEscape(agentPoolID))
	req, err := s.client.NewRequest("DELETE", u, nil)
	if err != nil {
		return err
	}

	return req.Do(ctx, nil)
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
	if o == nil {
		return nil // nothing to validate
	}
	if err := validateAgentPoolIncludeParams(o.Include); err != nil {
		return err
	}

	return nil
}

func (o *AgentPoolListOptions) valid() error {
	if o == nil {
		return nil // nothing to validate
	}
	if err := validateAgentPoolIncludeParams(o.Include); err != nil {
		return err
	}

	return nil
}

func validateAgentPoolIncludeParams(params []AgentPoolIncludeOpt) error {
	for _, p := range params {
		switch p {
		case AgentPoolWorkspaces:
			// do nothing
		default:
			return ErrInvalidIncludeValue
		}
	}

	return nil
}
