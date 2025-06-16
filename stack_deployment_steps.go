// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package tfe

import (
	"context"
	"fmt"
	"net/url"
)

// StackDeploymentSteps describes all the stacks deployment step-related methods that the
// HCP Terraform API supports.
// NOTE WELL: This is a beta feature and is subject to change until noted otherwise in the
// release notes.
type StackDeploymentSteps interface {
	// List returns the stack deployment steps for a stack deployment run.
	List(ctx context.Context, stackDeploymentRunID string) (*StackDeploymentStepList, error)
	Read(ctx context.Context, stackDeploymentStepID string) (*StackDeploymentStep, error)
}

type stackDeploymentSteps struct {
	client *Client
}

func (s stackDeploymentSteps) List(ctx context.Context, stackDeploymentRunID string) (*StackDeploymentStepList, error) {
	req, err := s.client.NewRequest("GET", fmt.Sprintf("stack-deployment-runs/%s/stack-deployment-steps", url.PathEscape(stackDeploymentRunID)), nil)
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
