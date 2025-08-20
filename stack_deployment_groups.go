// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package tfe

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"
)

// StackDeploymentGroups describes all the stack-deployment-groups related methods that the HCP Terraform API supports.
type StackDeploymentGroups interface {
	// List returns a list of Deployment Groups in a stack.
	List(ctx context.Context, stackConfigID string, options *StackDeploymentGroupListOptions) (*StackDeploymentGroupList, error)

	// Read retrieves a stack deployment group by its ID.
	Read(ctx context.Context, stackDeploymentGroupID string) (*StackDeploymentGroup, error)

	// ReadByName retrieves a stack deployment group by its Name.
	ReadByName(ctx context.Context, stackConfigurationID, stackDeploymentName string) (*StackDeploymentGroup, error)

	// ApproveAllPlans approves all pending plans in a stack deployment group.
	ApproveAllPlans(ctx context.Context, stackDeploymentGroupID string) error

	// Rerun re-runs all the stack deployment runs in a deployment group.
	Rerun(ctx context.Context, stackDeploymentGroupID string, options *StackDeploymentGroupRerunOptions) error
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
	// Attributes
	ID        string    `jsonapi:"primary,stack-deployment-groups"`
	Name      string    `jsonapi:"attr,name"`
	Status    string    `jsonapi:"attr,status"`
	CreatedAt time.Time `jsonapi:"attr,created-at,iso8601"`
	UpdatedAt time.Time `jsonapi:"attr,updated-at,iso8601"`

	// Relationships
	StackConfiguration *StackConfiguration `jsonapi:"relation,stack-configuration"`
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

// StackDeploymentGroupRerunOptions represents options for rerunning deployments in a stack deployment group.
type StackDeploymentGroupRerunOptions struct {
	// Required query parameter: A list of deployment run IDs to rerun.
	Deployments []string
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

// ReadByName retrieves a stack deployment group by its Name.
func (s stackDeploymentGroups) ReadByName(ctx context.Context, stackConfigurationID, stackDeploymentName string) (*StackDeploymentGroup, error) {
	if !validStringID(&stackConfigurationID) {
		return nil, fmt.Errorf("invalid stack configuration id: %s", stackConfigurationID)
	}
	if !validStringID(&stackDeploymentName) {
		return nil, fmt.Errorf("invalid stack deployment group name: %s", stackDeploymentName)
	}

	req, err := s.client.NewRequest("GET", fmt.Sprintf("stack-configurations/%s/stack-deployment-groups/%s", url.PathEscape(stackConfigurationID), url.PathEscape(stackDeploymentName)), nil)
	if err != nil {
		return nil, err
	}

	sdg := &StackDeploymentGroup{}
	err = req.Do(ctx, sdg)
	if err != nil {
		return nil, err
	}

	return sdg, nil
}

// Read retrieves a stack deployment group by its ID.
func (s stackDeploymentGroups) Read(ctx context.Context, stackDeploymentGroupID string) (*StackDeploymentGroup, error) {
	if !validStringID(&stackDeploymentGroupID) {
		return nil, fmt.Errorf("invalid stack deployment group ID: %s", stackDeploymentGroupID)
	}

	req, err := s.client.NewRequest("GET", fmt.Sprintf("stack-deployment-groups/%s", url.PathEscape(stackDeploymentGroupID)), nil)
	if err != nil {
		return nil, err
	}

	sdg := &StackDeploymentGroup{}
	err = req.Do(ctx, sdg)
	if err != nil {
		return nil, err
	}

	return sdg, nil
}

// ApproveAllPlans approves all pending plans in a stack deployment group.
func (s stackDeploymentGroups) ApproveAllPlans(ctx context.Context, stackDeploymentGroupID string) error {
	if !validStringID(&stackDeploymentGroupID) {
		return fmt.Errorf("invalid stack deployment group ID: %s", stackDeploymentGroupID)
	}

	req, err := s.client.NewRequest("POST", fmt.Sprintf("stack-deployment-groups/%s/approve-all-plans", url.PathEscape(stackDeploymentGroupID)), nil)
	if err != nil {
		return err
	}

	return req.Do(ctx, nil)
}

// Rerun re-runs all the stack deployment runs in a deployment group.
func (s stackDeploymentGroups) Rerun(ctx context.Context, stackDeploymentGroupID string, options *StackDeploymentGroupRerunOptions) error {
	if !validStringID(&stackDeploymentGroupID) {
		return fmt.Errorf("invalid stack deployment group ID: %s", stackDeploymentGroupID)
	}

	if options == nil || len(options.Deployments) == 0 {
		return fmt.Errorf("no deployments specified for rerun")
	}

	u := fmt.Sprintf("stack-deployment-groups/%s/rerun", url.PathEscape(stackDeploymentGroupID))

	type DeploymentQueryParams struct {
		Deployments string `url:"deployments"`
	}

	qp, err := decodeQueryParams(&DeploymentQueryParams{
		Deployments: strings.Join(options.Deployments, ","),
	})
	if err != nil {
		return err
	}
	req, err := s.client.NewRequestWithAdditionalQueryParams("POST", u, nil, qp)
	if err != nil {
		return err
	}

	return req.Do(ctx, nil)
}
