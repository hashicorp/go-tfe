// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package tfe

import (
	"context"
	"fmt"
	"net/url"
)

// NOTE WELL: This is a beta feature and is subject to change until noted otherwise in the
// release notes.
type StackPlans interface {
	// Read returns a stack plan by its ID.
	Read(ctx context.Context, stackPlanId string) (*StackPlan, error)

	// Get Stack Plans from Configuration Version

}

type stackPlans struct {
	client *Client
}

var _ StackPlans = &stackPlans{}

type StackPlan struct {
	ID             string     `jsonapi:"primary,stack-plans"`
	Status         string     `jsonapi:"attr,status"`
	Deployment     string     `jsonapi:"attr,deployment"`
	SequenceNumber int        `jsonapi:"attr,sequence-number"`
	ParentPlan     *StackPlan `jsonapi:"relation,parent-plan"`
	ChangeSummary  string     `jsonapi:"attr,change-summary"`

	// Relations
	StackPlanOperations []*StackPlanOperation `jsonapi:"relation,stack-plan-operations"`
	Stack               *Stack                `jsonapi:"relation,stack"`
	StackConfiguration  *StackConfiguration   `jsonapi:"relation,stack-configuration"`
}

func (s stackPlans) Read(ctx context.Context, stackPlanId string) (*StackPlan, error) {
	req, err := s.client.NewRequest("GET", fmt.Sprintf("stack-plans/%s", url.PathEscape(stackPlanId)), nil)
	if err != nil {
		return nil, err
	}

	ucs := &StackPlan{}
	err = req.Do(ctx, ucs)
	if err != nil {
		return nil, err
	}

	return ucs, nil
}
