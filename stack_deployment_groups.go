package tfe

import (
	"context"
	"fmt"
	"net/url"
)

// StackDeploymentGroups describes all the stack-deployment-groups related methods that the HCP Terraform API supports.
type StackDeploymentGroups interface {
	// List returns a list of Deployment Groups in a stack.
	List(ctx context.Context, stackConfigID string, options *StackDeploymentGroupListOptions) (*StackDeploymentGroupList, error)
}

type DeploymentGroupStatus string

const (
	DeploymentGroupStatusPending   DeploymentGroupStatus = "pending"
	DeploymentGroupStatusDeploying DeploymentGroupStatus = "deploying"
	DeploymentGroupStatusSucceeded DeploymentGroupStatus = "succeeded"
	DeploymentGroupStatusFailed    DeploymentGroupStatus = "failed"
	DeploymentGroupStatusAbandoned DeploymentGroupStatus = "abandoned"
)

// stackDeploymentGroups implements StackDeploymentGroups.
type stackDeploymentGroups struct {
	client *Client
}

var _ StackDeploymentGroups = &stackDeploymentGroups{}

// StackDeploymentGroup represents a stack deployment group.
type StackDeploymentGroup struct {
	ID        string                `jsonapi:"primary,stacks-deployment-groups"`
	Name      string                `jsonapi:"attr,name"`
	Status    DeploymentGroupStatus `jsonapi:"attr,status"`
	CreatedAt string                `jsonapi:"attr,created-at"`
	UpdatedAt string                `jsonapi:"attr,updated-at"`

	// Relationships
	StackConfiguration StackConfiguration `jsonapi:"relation,stack-configurations"`
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
