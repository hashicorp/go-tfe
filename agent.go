package tfe

import (
	"context"
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
	Read(ctx context.Context, agentPoolID string) (*AgentPool, error)

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

	// Relations
	Organization *Organization `jsonapi:"relation,organization"`
	Workspaces   []*Workspace  `jsonapi:"relation,workspaces"`
}

// A list of relations to include
// https://www.terraform.io/cloud-docs/api-docs/agents#available-related-resources
