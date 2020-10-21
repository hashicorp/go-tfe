package tfe

import (
	"context"
	"errors"
	"fmt"
	"net/url"
)

// Compile-time proof of interface implementation.
var _ AgentPools = (*agentPools)(nil)

// AgentPools describes all the agent pool related methods that the Terraform
// Enterprise API supports.
//
// TFE API docs: https://www.terraform.io/docs/cloud/api/agents.html
type AgentPools interface {
	// List all the agent pools of the given organization.
	List(ctx context.Context, organization string, options AgentPoolListOptions) (*AgentPoolList, error)

	// Create a new agent pool with the given options.
	Create(ctx context.Context, organization string, options AgentPoolCreateOptions) (*AgentPool, error)

	// Read a agentpool by its ID.
	Read(ctx context.Context, agentpoolID string) (*AgentPool, error)
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

// AgentPool represents a Terraform Enterprise agent pool.
type AgentPool struct {
	ID   string `jsonapi:"primary,agent-pools"`
	Name string `jsonapi:"attr,name"`
}

// AgentPoolListOptions represents the options for listing agent pools.
type AgentPoolListOptions struct {
	ListOptions
}

// List all the agent pools of the given organization. Note this currently is
// limited to a single pool per organization (API enforced).
func (s *agentPools) List(ctx context.Context, organization string, options AgentPoolListOptions) (*AgentPoolList, error) {
	if !validStringID(&organization) {
		return nil, errors.New("invalid value for organization")
	}

	u := fmt.Sprintf("organizations/%s/agent-pools", url.QueryEscape(organization))
	req, err := s.client.newRequest("GET", u, &options)
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

// AgentPoolCreateOptions represents the options for creating an agent pool.
type AgentPoolCreateOptions struct {
	// For internal use only!
	ID string `jsonapi:"primary,agent-pools"`
}

// Create a new agent pool with the given options. Note only a single pool is
// allowed per organization (API enforced).
func (s *agentPools) Create(ctx context.Context, organization string, options AgentPoolCreateOptions) (*AgentPool, error) {
	if !validStringID(&organization) {
		return nil, errors.New("invalid value for organization")
	}

	// Make sure we don't send a user provided ID.
	options.ID = ""

	u := fmt.Sprintf("organizations/%s/agent-pools", url.QueryEscape(organization))
	req, err := s.client.newRequest("POST", u, &options)
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

// Read a single agent pool by its ID.
func (s *agentPools) Read(ctx context.Context, agentpoolID string) (*AgentPool, error) {
	if !validStringID(&agentpoolID) {
		return nil, errors.New("invalid value for agent pool ID")
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
