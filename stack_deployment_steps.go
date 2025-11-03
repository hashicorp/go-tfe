// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package tfe

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"time"
)

// StackDeploymentSteps describes all the stacks deployment step-related methods that the
// HCP Terraform API supports.
type StackDeploymentSteps interface {
	// List returns the stack deployment steps for a stack deployment run.
	List(ctx context.Context, stackDeploymentRunID string, opts *StackDeploymentStepsListOptions) (*StackDeploymentStepList, error)
	// Read returns a stack deployment step by its ID.
	Read(ctx context.Context, stackDeploymentStepID string) (*StackDeploymentStep, error)
	// Advance advances the stack deployment step when in the "pending_operator" state.
	Advance(ctx context.Context, stackDeploymentStepID string) error
	// Diagnostics returns the diagnostics for this stack deployment step.
	Diagnostics(ctx context.Context, stackConfigurationID string) (*StackDiagnosticsList, error)
	// Artifacts returns the artifacts for this stack deployment step.
	// Valid artifact names are "plan-description" and "apply-description".
	Artifacts(ctx context.Context, stackDeploymentStepID string, artifactType StackDeploymentStepArtifactType) (io.ReadCloser, error)
}

type StackDeploymentStepArtifactType string

const (
	// StackDeploymentStepArtifactPlanDescription represents the plan description artifact type.
	StackDeploymentStepArtifactPlanDescription StackDeploymentStepArtifactType = "plan-description"
	// StackDeploymentStepArtifactApplyDescription represents the apply description artifact type.
	StackDeploymentStepArtifactApplyDescription StackDeploymentStepArtifactType = "apply-description"
	// StackDeploymentStepArtifactPlanDescription represents the plan debug log artifact type.
	StackDeploymentStepArtifactPlanDebugLog StackDeploymentStepArtifactType = "plan-debug-log"
	// StackDeploymentStepArtifactApplyDescription represents the apply debug log artifact type.
	StackDeploymentStepArtifactApplyDebugLog StackDeploymentStepArtifactType = "apply-debug-log"
)

// StackDeploymentStep represents a step from a stack deployment
type StackDeploymentStep struct {
	// Attributes
	ID            string    `jsonapi:"primary,stack-deployment-steps"`
	Status        string    `jsonapi:"attr,status"`
	OperationType string    `jsonapi:"attr,operation-type"`
	CreatedAt     time.Time `jsonapi:"attr,created-at,iso8601"`
	UpdatedAt     time.Time `jsonapi:"attr,updated-at,iso8601"`

	// Links
	Links map[string]interface{} `jsonapi:"links,omitempty"`

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

// Read returns a stack deployment step by its ID.
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

// Advance advances the stack deployment step when in the "pending_operator" state.
func (s stackDeploymentSteps) Advance(ctx context.Context, stackDeploymentStepID string) error {
	req, err := s.client.NewRequest("POST", fmt.Sprintf("stack-deployment-steps/%s/advance", url.PathEscape(stackDeploymentStepID)), nil)
	if err != nil {
		return err
	}

	return req.Do(ctx, nil)
}

// Diagnostics returns the diagnostics for this stack deployment step.
func (s stackDeploymentSteps) Diagnostics(ctx context.Context, stackDeploymentStepID string) (*StackDiagnosticsList, error) {
	req, err := s.client.NewRequest("GET", fmt.Sprintf("stack-deployment-steps/%s/stack-diagnostics", url.PathEscape(stackDeploymentStepID)), nil)
	if err != nil {
		return nil, err
	}
	diagnostics := &StackDiagnosticsList{}
	err = req.Do(ctx, diagnostics)
	if err != nil {
		return nil, err
	}
	return diagnostics, nil
}

// Artifacts returns the artifacts for this stack deployment step.
// Valid artifact names are "plan-description" and "apply-description".
func (s stackDeploymentSteps) Artifacts(ctx context.Context, stackDeploymentStepID string, artifactType StackDeploymentStepArtifactType) (io.ReadCloser, error) {
	req, err := s.client.NewRequestWithAdditionalQueryParams("GET",
		fmt.Sprintf("stack-deployment-steps/%s/artifacts", url.PathEscape(stackDeploymentStepID)),
		nil,
		map[string][]string{"name": {url.PathEscape(string(artifactType))}},
	)
	if err != nil {
		return nil, err
	}

	return req.DoRaw(ctx)
}
