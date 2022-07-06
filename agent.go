package tfe

import (
	"context"
	"fmt"
	"net/url"
)

// Compile-time proof of interface implementation.
var _ Agents = (*agents)(nil)

// Agents describes all the agent-related methods that the
// Terraform Cloud API supports.
// TFE API docs: https://www.terraform.io/docs/cloud/api/agents.html
type Agents interface {
	// Read an agent by its ID.
	Read(ctx context.Context, agentID string) (*Agent, error)

	// Read an agent by its ID with the given options.
	ReadWithOptions(ctx context.Context, agentID string, options *AgentReadOptions) (*Agent, error)

	// List all the agents of the given pool.
	List(ctx context.Context, agentPoolID string, options *AgentPoolListOptions) (*AgentList, error)

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
func (s *agents) Read(ctx context.Context, agentID string) (*Agent, error) {
	return s.ReadWithOptions(ctx, agentID, nil)
}

// Read a single agent pool by its ID with options.
func (s *agents) ReadWithOptions(ctx context.Context, agentID string, options *AgentReadOptions) (*Agent, error) {
	if !validStringID(&agentID) {
		return nil, ErrInvalidAgentID //undeclared var name
	}
	if err := options.valid(); err != nil {
		return nil, err
	}

	u := fmt.Sprintf("agents/%s", url.QueryEscape(agentID))
	req, err := s.client.newRequest("GET", u, nil)
	if err != nil {
		return nil, err
	}

	agent := &AgentPool{}
	err = s.client.do(ctx, req, agent)
	if err != nil {
		return nil, err
	}

	return agent, nil //cannot use agent as *Agent value in return statement
}

// List all the agent pools of the given organization.
func (s *agents) List(ctx context.Context, agentPoolID string, options *AgentListOptions) (*AgentList, error) {
	if !validStringID(&agentPoolID) {
		return nil, ErrInvalidOrg
	}
	if err := options.valid(); err != nil {
		return nil, err
	}

	u := fmt.Sprintf("agent-pools/%s/agents", url.QueryEscape(agentPoolID))
	req, err := s.client.newRequest("GET", u, options)
	if err != nil {
		return nil, err
	}

	agentList := &AgentList{}
	err = s.client.do(ctx, req, agentList)
	if err != nil {
		return nil, err
	}

	return agentList, nil
}

// Delete an agent pool by its ID.
func (s *agents) Delete(ctx context.Context, agentID string) error {
	if !validStringID(&agentID) {
		return ErrInvalidAgentID
	}

	u := fmt.Sprintf("agents/%s", url.QueryEscape(agentID))
	req, err := s.client.newRequest("DELETE", u, nil)
	if err != nil {
		return err
	}

	return s.client.do(ctx, req, nil)
}
