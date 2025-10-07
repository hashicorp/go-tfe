package tfe

import (
	"context"
	"fmt"
	"net/url"
	"time"
)

type StackDeployments interface {
	// List returns a list of stack deployments for a given stack.
	List(ctx context.Context, stackID string, opts *StackDeploymentListOptions) (*StackDeploymentList, error)
}

type StackDeployment struct {
	// Attributes
	ID            string    `jsonapi:"primary,stack-deployments"`
	Status        string    `jsonapi:"attr,status"`
	Name          string    `jsonapi:"attr,name"`
	DeployedAt    time.Time `jsonapi:"attr,deployed-at,iso8601"`
	ErrorsCount   int       `jsonapi:"attr,errors-count"`
	WarningsCount int       `jsonapi:"attr,warnings-count"`
	PausedCount   int       `jsonapi:"attr,paused-count"`
	HasEmptyState bool      `jsonapi:"attr,has-empty-state"`

	// Relationships
	Stack              *Stack              `jsonapi:"relation,stack"`
	CurrentStackState  *StackState         `jsonapi:"relation,current-stack-state"`
	StackConfiguration *StackConfiguration `jsonapi:"relation,stack-configuration"`
	LatestStackPlan    *StackPlan          `jsonapi:"relation,latest-stack-plan"`
	Diagnostics        []*StackDiagnostic  `jsonapi:"relation,stack-diagnostics"`
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
