// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package tfe

import (
	"context"
	"fmt"
	"net/url"
	"time"
)

// StackDeploymentRuns describes all the stack deployment runs-related methods that the HCP Terraform API supports.
type StackDeploymentRuns interface {
	// List returns a list of stack deployment runs for a given deployment group.
	List(ctx context.Context, deploymentGroupID string, options *StackDeploymentRunListOptions) (*StackDeploymentRunList, error)
	Read(ctx context.Context, stackDeploymentRunID string) (*StackDeploymentRun, error)
	ApproveAllPlans(ctx context.Context, deploymentRunID string) error
	Cancel(ctx context.Context, stackDeploymentRunID string) error
}

// stackDeploymentRuns implements StackDeploymentRuns.
type stackDeploymentRuns struct {
	client *Client
}

var _ StackDeploymentRuns = &stackDeploymentRuns{}

// StackDeploymentRun represents a stack deployment run.
type StackDeploymentRun struct {
	ID        string    `jsonapi:"primary,stack-deployment-runs"`
	Status    string    `jsonapi:"attr,status"`
	CreatedAt time.Time `jsonapi:"attr,created-at,iso8601"`
	UpdatedAt time.Time `jsonapi:"attr,updated-at,iso8601"`

	// Relationships
	StackDeploymentGroup *StackDeploymentGroup `jsonapi:"relation,stack-deployment-group"`
}

type SDRIncludeOpt string

const (
	SDRDeploymentGroup SDRIncludeOpt = "stack-deployment-group"
)

type StackDeploymentRunReadOptions struct {
	// Optional: A list of relations to include.
	Include []SDRIncludeOpt `url:"include,omitempty"`
}

// StackDeploymentRunList represents a list of stack deployment runs.
type StackDeploymentRunList struct {
	*Pagination
	Items []*StackDeploymentRun
}

// StackDeploymentRunListOptions represents the options for listing stack deployment runs.
type StackDeploymentRunListOptions struct {
	ListOptions

	// Optional: A list of relations to include.
	Include []SDRIncludeOpt `url:"include,omitempty"`
}

// List returns a list of stack deployment runs for a given deployment group.
func (s *stackDeploymentRuns) List(ctx context.Context, deploymentGroupID string, options *StackDeploymentRunListOptions) (*StackDeploymentRunList, error) {
	req, err := s.client.NewRequest("GET", fmt.Sprintf("stack-deployment-groups/%s/stack-deployment-runs", url.PathEscape(deploymentGroupID)), options)
	if err != nil {
		return nil, err
	}

	sdrl := &StackDeploymentRunList{}
	err = req.Do(ctx, sdrl)
	if err != nil {
		return nil, err
	}

	return sdrl, nil
}

func (s stackDeploymentRuns) Read(ctx context.Context, stackDeploymentRunID string) (*StackDeploymentRun, error) {
	req, err := s.client.NewRequest("GET", fmt.Sprintf("stack-deployment-runs/%s", url.PathEscape(stackDeploymentRunID)), nil)
	if err != nil {
		return nil, err
	}

	run := StackDeploymentRun{}
	err = req.Do(ctx, &run)
	if err != nil {
		return nil, err
	}

	return &run, nil
}

func (s stackDeploymentRuns) ReadWithOptions(ctx context.Context, stackDeploymentRunID string, options *StackDeploymentRunReadOptions) (*StackDeploymentRun, error) {
	if err := options.valid(); err != nil {
		return nil, err
	}

	req, err := s.client.NewRequest("GET", fmt.Sprintf("stack-deployment-runs/%s", url.PathEscape(stackDeploymentRunID)), options)
	if err != nil {
		return nil, err
	}

	run := StackDeploymentRun{}
	err = req.Do(ctx, &run)
	if err != nil {
		return nil, err
	}

	return &run, nil
}

func (s stackDeploymentRuns) ApproveAllPlans(ctx context.Context, stackDeploymentRunID string) error {
	req, err := s.client.NewRequest("POST", fmt.Sprintf("stack-deployment-runs/%s/approve-all-plans", url.PathEscape(stackDeploymentRunID)), nil)
	if err != nil {
		return err
	}

	return req.Do(ctx, nil)
}

func (s stackDeploymentRuns) Cancel(ctx context.Context, stackDeploymentRunID string) error {
	req, err := s.client.NewRequest("POST", fmt.Sprintf("stack-deployment-runs/%s/cancel", url.PathEscape(stackDeploymentRunID)), nil)
	if err != nil {
		return err
	}

	return req.Do(ctx, nil)
}

func (o *StackDeploymentRunReadOptions) valid() error {
	for _, include := range o.Include {
		switch include {
		case SDRDeploymentGroup:
			// Valid option, do nothing.
		default:
			return fmt.Errorf("invalid include option: %s", include)
		}
	}
	return nil
}
