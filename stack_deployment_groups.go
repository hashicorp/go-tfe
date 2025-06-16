// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package tfe

import (
	"context"
	"fmt"
	"net/url"
	"time"
)

type StackDeploymentGroups interface {
	// List returns a list of Deployment Groups in a stack.
	List(ctx context.Context, stackConfigID string, options *StackDeploymentGroupListOptions) (*StackDeploymentGroupList, error)

	// Read retrieves a stack deployment group by its ID.
	Read(ctx context.Context, stackDeploymentGroupID string) (*StackDeploymentGroup, error)
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
	ID        string    `jsonapi:"primary,stacks-deployment-groups"`
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
