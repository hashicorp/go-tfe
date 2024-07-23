// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package tfe

import (
	"bytes"
	"context"
	"fmt"
	"net/url"
)

// NOTE WELL: This is a beta feature and is subject to change until noted otherwise in the
// release notes.
type StackPlanOperations interface {
	// Read returns a stack plan operation by its ID.
	Read(ctx context.Context, stackPlanOperationId string) (*StackPlanOperation, error)

	// Get Stack Plans from Configuration Version
	DownloadEventStream(ctx context.Context, stackPlanOperationId string) ([]byte, error)
}

type stackPlanOperations struct {
	client *Client
}

var _ StackPlanOperations = &stackPlanOperations{}

type StackPlanOperation struct {
	ID             string `jsonapi:"primary,stack-plan-operations"`
	Type           string `jsonapi:"attr,operation-type"`
	Status         string `jsonapi:"attr,status"`
	EventStreamUrl string `jsonapi:"attr,event-stream-url"`
	Diagnostics    string `jsonapi:"attr,diags"`

	// Relations
	StackPlan *StackPlan `jsonapi:"relation,stack-plan"`
}

func (s stackPlanOperations) Read(ctx context.Context, stackPlanOperationId string) (*StackPlanOperation, error) {
	req, err := s.client.NewRequest("GET", fmt.Sprintf("stack-plans-operations/%s", url.PathEscape(stackPlanOperationId)), nil)
	if err != nil {
		return nil, err
	}

	ucs := &StackPlanOperation{}
	err = req.Do(ctx, ucs)
	if err != nil {
		return nil, err
	}

	return ucs, nil
}

func (s stackPlanOperations) DownloadEventStream(ctx context.Context, eventStreamUrl string) ([]byte, error) {
	req, err := s.client.NewRequest("GET", eventStreamUrl, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Accept", "application/json")

	var buf bytes.Buffer
	err = req.Do(ctx, &buf)
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}
