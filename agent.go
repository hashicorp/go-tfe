package tfe

import (
	"context"
	"fmt"
	"net/url"
)

// Compile-time proof of interface implementation.
var _ Agents = (*agents)(nil)

// // AgentPools describes all the agent pool related methods that the
// Terraform Cloud API supports.
// Note that agents are not available in Terraform Enterprise.
// TFE API docs: https://www.terraform.io/docs/cloud/api/agents.html
type Agents interface {
	// List all the agents of the given pool.
	List(ctx context.Context, organization string, options *AgentPoolListOptions) (*AgentPoolList, error)

	// Read an agent by its ID.
	Read(ctx context.Context, agentPoolID string) (*Agent, error)

	// Read an agent by its ID with the given options.
	ReadWithOptions(ctx context.Context, agentID string, options *AgentReadOptions) (*AgentPool, error)

	// Delete an agent by its ID.
	Delete(ctx context.Context, agentPoolID string) error
}

// agents implements Agents.
type agents struct {
	client *Client
}

// AgentList represents a list of agents.
type AgentList struct {
	*Pagination
	Items []*Agents
}

// Agent represents a Terraform Cloud agent.
type Agent struct {
	ID   string `jsonapi:"primary,agent-pools"`
	Name string `jsonapi:"attr,name"`
	IP   string `jsonapi:"attr,ip-address"`

	// Relations
	Organization *Organization `jsonapi:"relation,organization"`
	Workspaces   []*Workspace  `jsonapi:"relation,workspaces"`
}

// A list of relations to include
// https://www.terraform.io/cloud-docs/api-docs/agents#available-related-resources
type AgentIncludeOpt string

const (
	AgentWorkspaces AgentIncludeOpt = "workspaces"
)

// AgentReadOptions represents the options for reading an agent.
type AgentReadOptions struct {
	Include []AgentIncludeOpt `url:"include,omitempty"`
}

// AgentListOptions represents the options for listing agents.
type AgentListOptions struct {
	ListOptions
	// Optional: A list of relations to include. See available resources
	// https://www.terraform.io/cloud-docs/api-docs/agents#available-related-resources
	Include []AgentIncludeOpt `url:"include,omitempty"`
}

// Read a single agent by its ID
func (s *agents) Read(ctx context.Context, agentpoolID string) (*Agent, error) {
	return s.ReadWithOptions(ctx, agentID, nil)
}

// Read a single agent pool by its ID with options.
func (s *agents) ReadWithOptions(ctx context.Context, agentpoolID string, options *AgentReadOptions) (*AgentPool, error) {
	if !validStringID(&agentpoolID) {
		return nil, ErrInvalidAgentPoolID
	}
	if err := options.valid(); err != nil {
		return nil, err
	}

	u := fmt.Sprintf("agent-pools/%s", url.QueryEscape(agentpoolID))
	req, err := s.client.newRequest("GET", u, nil)
	if err != nil {
		return nil, err
	}

	pool := &AgentPool{}
	err = s.client.do(ctx, req, pool)
	if err != nil {
		return nil, err
	}

	return pool, nil
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
	req, err := s.client.newRequest("GET", u, options)
	if err != nil {
		return nil, err
	}

	poolList := &AgentPoolList{}
	err = s.client.do(ctx, req, poolList)
	if err != nil {
		return nil, err
	}

	return poolList, nil
}

// Delete an agent pool by its ID.
func (s *agentPools) Delete(ctx context.Context, agentPoolID string) error {
	if !validStringID(&agentPoolID) {
		return ErrInvalidAgentPoolID
	}

	u := fmt.Sprintf("agent-pools/%s", url.QueryEscape(agentPoolID))
	req, err := s.client.newRequest("DELETE", u, nil)
	if err != nil {
		return err
	}

	return s.client.do(ctx, req, nil)
}
