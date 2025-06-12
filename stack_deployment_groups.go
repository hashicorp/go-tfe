package tfe

import (
	"context"
	"fmt"
)

type StackDeploymentGroups interface {
	// List returns a list of Deployment Groups in a stack.
	List(ctx context.Context, stackConfigID string) (*StackDeploymentGroupList, error)
}

type DeploymentGroupStatus string

const (
	DeploymentGroupStatusPending   DeploymentGroupStatus = "pending"
	DeploymentGroupStatusDeploying DeploymentGroupStatus = "deploying"
	DeploymentGroupStatusSucceeded DeploymentGroupStatus = "succeeded"
	DeploymentGroupStatusFailed    DeploymentGroupStatus = "failed"
	DeploymentGroupStatusAbandoned DeploymentGroupStatus = "abandoned"
)

var DeploymentGroupStatuses = []DeploymentGroupStatus{
	DeploymentGroupStatusPending,
	DeploymentGroupStatusDeploying,
	DeploymentGroupStatusSucceeded,
	DeploymentGroupStatusFailed,
	DeploymentGroupStatusAbandoned,
}

type stackDeploymentGroups struct {
	client *Client
}

var _ StackDeploymentGroups = &stackDeploymentGroups{}

type StackDeploymentGroup struct {
	Id                   string
	Name                 string
	Status               DeploymentGroupStatus
	CreatedAt            string
	UpdatedAt            string // time.RFC3339
	StackConfigurationId string
	FailureCount         int
}

// StackDeploymentGroupList represents a list of stack deployment groups.
type StackDeploymentGroupList struct {
	*Pagination
	Items []*StackDeploymentGroup
}

func (s stackDeploymentGroups) List(ctx context.Context, stackConfigID string) (*StackDeploymentGroupList, error) {
	if !validStringID(&stackConfigID) {
		return nil, fmt.Errorf("invalid stack configuration ID: %s", stackConfigID)
	}

	req, err := s.client.NewRequest("GET", fmt.Sprintf("stack-configurations/%s/stack-deployment-groups", stackConfigID), nil)
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