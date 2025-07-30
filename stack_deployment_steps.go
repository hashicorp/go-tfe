// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package tfe

import (
	"context"
	"fmt"
	"net/url"
	"time"
)

// StackDeploymentSteps describes all the stacks deployment step-related methods that the
// HCP Terraform API supports.
// NOTE WELL: This is a beta feature and is subject to change until noted otherwise in the
// release notes.
type StackDeploymentSteps interface {
	// List returns the stack deployment steps for a stack deployment run.
	List(ctx context.Context, stackDeploymentRunID string, opts *StackDeploymentStepsListOptions) (*StackDeploymentStepList, error)
	Read(ctx context.Context, stackDeploymentStepID string) (*StackDeploymentStep, error)
}

// StackDeploymentStep represents a step from a stack deployment
type StackDeploymentStep struct {
	// Attributes
	ID        string    `jsonapi:"primary,stack-deployment-steps"`
	Status    string    `jsonapi:"attr,status"`
	CreatedAt time.Time `jsonapi:"attr,created-at,iso8601"`
	UpdatedAt time.Time `jsonapi:"attr,created-at,iso8601"`

	// Links
	UploadUrl string `jsonapi:"links,upload-url,omitempty"`

	// Relationships
	StackDeploymentRun *StackDeploymentRun `jsonapi:"relation,stack-deployment-run"`
}

// StackDeploymentStepList represents a list of stack deployment steps
type StackDeploymentStepList struct {
	*Pagination
	Items []*StackDeploymentStep
}

type stackDeploymentSteps struct {
	client *Client
}

// StackDeploymentStepsListOptions represents the options for listing stack
// deployment steps.
type StackDeploymentStepsListOptions struct {
	ListOptions
}

func (s stackDeploymentSteps) List(ctx context.Context, stackDeploymentRunID string, opts *StackDeploymentStepsListOptions) (*StackDeploymentStepList, error) {
	req, err := s.client.NewRequest("GET", fmt.Sprintf("stack-deployment-runs/%s/stack-deployment-steps", url.PathEscape(stackDeploymentRunID)), opts)
	if err != nil {
		return nil, err
	}

	steps := StackDeploymentStepList{}
	err = req.Do(ctx, &steps)
	if err != nil {
		return nil, err
	}

	return &steps, nil
}

func (s stackDeploymentSteps) Read(ctx context.Context, stackDeploymentStepID string) (*StackDeploymentStep, error) {
	req, err := s.client.NewRequest("GET", fmt.Sprintf("stack-deployment-steps/%s", url.PathEscape(stackDeploymentStepID)), nil)
	if err != nil {
		return nil, err
	}

	step := StackDeploymentStep{}
	err = req.Do(ctx, &step)
	if err != nil {
		return nil, err
	}

	return &step, nil
}
