package tfe

import (
	"context"
	"fmt"
	"net/url"
	"time"
)

// Compile-time proof of interface implementation.
var _ Agents = (*agents)(nil)

// Agents describes all the agent-related methods that the
// Terraform Cloud API supports.
// TFE API docs: https://www.terraform.io/docs/cloud/api/agents.html
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

// Agent represents a Terraform Cloud agent.
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
}

// Read a single agent by its ID
func (s *agents) Read(ctx context.Context, agentID string) (*Agent, error) {
	if !validStringID(&agentID) {
		return nil, ErrInvalidAgentID
	}

	u := fmt.Sprintf("agents/%s", url.QueryEscape(agentID))
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

	u := fmt.Sprintf("agent-pools/%s/agents", url.QueryEscape(agentPoolID))
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
