// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package tfe

import (
	"context"
	"fmt"
	"net/url"
	"time"
)

// StackPlans describes all the stacks plans-related methods that the HCP Terraform API supports.
// NOTE WELL: This is a beta feature and is subject to change until noted otherwise in the
// release notes.
type StackPlans interface {
	// Read returns a stack plan by its ID.
	Read(ctx context.Context, stackPlanID string) (*StackPlan, error)

	// ListByConfiguration returns a list of stack plans for a given stack configuration.
	ListByConfiguration(ctx context.Context, stackConfigurationID string, options *StackPlansListOptions) (*StackPlanList, error)

	// Approve approves a stack plan.
	Approve(ctx context.Context, stackPlanID string) error

	// Cancel cancels a stack plan.
	Cancel(ctx context.Context, stackPlanID string) error

	// Discard discards a stack plan.
	Discard(ctx context.Context, stackPlanID string) error
}

type StackPlansStatusFilter string

const (
	StackPlansStatusFilterCreated   StackPlansStatusFilter = "created"
	StackPlansStatusFilterRunning   StackPlansStatusFilter = "running"
	StackPlansStatusFilterPaused    StackPlansStatusFilter = "paused"
	StackPlansStatusFilterFinished  StackPlansStatusFilter = "finished"
	StackPlansStatusFilterDiscarded StackPlansStatusFilter = "discarded"
	StackPlansStatusFilterErrored   StackPlansStatusFilter = "errored"
	StackPlansStatusFilterCanceled  StackPlansStatusFilter = "canceled"
)

type StackPlansListOptions struct {
	ListOptions

	// Optional: A query string to filter plans by status.
	Status StackPlansStatusFilter `url:"filter[status],omitempty"`

	// Optional: A query string to filter plans by deployment.
	Deployment string `url:"filter[deployment],omitempty"`
}

type StackPlanList struct {
	*Pagination
	Items []*StackPlan
}

// stackPlans implements StackPlans.
type stackPlans struct {
	client *Client
}

var _ StackPlans = &stackPlans{}

type StackPlanStatusTimestamps struct {
	CreatedAt  time.Time `jsonapi:"attr,created-at,rfc3339"`
	RunningAt  time.Time `jsonapi:"attr,running-at,rfc3339"`
	PausedAt   time.Time `jsonapi:"attr,paused-at,rfc3339"`
	FinishedAt time.Time `jsonapi:"attr,finished-at,rfc3339"`
}

type PlanChanges struct {
	Add    int `jsonapi:"attr,add"`
	Total  int `jsonapi:"attr,total"`
	Change int `jsonapi:"attr,change"`
	Import int `jsonapi:"attr,import"`
	Remove int `jsonapi:"attr,remove"`
}

// StackPlan represents a plan for a stack.
type StackPlan struct {
	ID               string                     `jsonapi:"primary,stack-plans"`
	PlanMode         string                     `jsonapi:"attr,plan-mode"`
	PlanNumber       string                     `jsonapi:"attr,plan-number"`
	Status           string                     `jsonapi:"attr,status"`
	StatusTimestamps *StackPlanStatusTimestamps `jsonapi:"attr,status-timestamps"`
	IsPlanned        bool                       `jsonapi:"attr,is-planned"`
	Changes          *PlanChanges               `jsonapi:"attr,changes"`
	Deployment       string                     `jsonapi:"attr,deployment"`

	// Relationships
	StackConfiguration *StackConfiguration `jsonapi:"relation,stack-configuration"`
	Stack              *Stack              `jsonapi:"relation,stack"`
}

func (s stackPlans) Read(ctx context.Context, stackPlanID string) (*StackPlan, error) {
	req, err := s.client.NewRequest("GET", fmt.Sprintf("stack-plans/%s", url.PathEscape(stackPlanID)), nil)
	if err != nil {
		return nil, err
	}

	sp := &StackPlan{}
	err = req.Do(ctx, sp)
	if err != nil {
		return nil, err
	}

	return sp, nil
}

func (s stackPlans) ListByConfiguration(ctx context.Context, stackConfigurationID string, options *StackPlansListOptions) (*StackPlanList, error) {
	req, err := s.client.NewRequest("GET", fmt.Sprintf("stack-configurations/%s/stack-plans", url.PathEscape(stackConfigurationID)), options)
	if err != nil {
		return nil, err
	}

	sl := &StackPlanList{}
	err = req.Do(ctx, sl)
	if err != nil {
		return nil, err
	}

	return sl, nil
}

func (s stackPlans) Approve(ctx context.Context, stackPlanID string) error {
	req, err := s.client.NewRequest("POST", fmt.Sprintf("stack-plans/%s/approve", url.PathEscape(stackPlanID)), nil)
	if err != nil {
		return err
	}

	return req.Do(ctx, nil)
}

func (s stackPlans) Discard(ctx context.Context, stackPlanID string) error {
	req, err := s.client.NewRequest("POST", fmt.Sprintf("stack-plans/%s/discard", url.PathEscape(stackPlanID)), nil)
	if err != nil {
		return err
	}

	return req.Do(ctx, nil)
}

func (s stackPlans) Cancel(ctx context.Context, stackPlanID string) error {
	req, err := s.client.NewRequest("POST", fmt.Sprintf("stack-plans/%s/cancel", url.PathEscape(stackPlanID)), nil)
	if err != nil {
		return err
	}

	return req.Do(ctx, nil)
}
