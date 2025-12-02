package tfe

import (
	"context"
	"fmt"
	"net/url"
)

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
