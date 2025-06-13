package tfe

import (
	"context"
	"fmt"
	"net/url"
)

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

type stackDeploymentGroups struct {
	client *Client
}

var _ StackDeploymentGroups = &stackDeploymentGroups{}

type StackDeploymentGroup struct {
	ID        string                
	Name      string                `jsonapi:"attr,name"`
	Status    DeploymentGroupStatus `jsonapi:"attr,status"`
	CreatedAt string                
	UpdatedAt string                

	// Relationships
	StackConfiguration StackConfiguration 
}

// StackDeploymentGroupList represents a list of stack deployment groups.
type StackDeploymentGroupList struct {
	*Pagination
	Items []*StackDeploymentGroup
}

type StackDeploymentGroupListOptions struct {
	ListOptions
}

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
