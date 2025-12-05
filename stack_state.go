// Copyright IBM Corp. 2018, 2025
// SPDX-License-Identifier: MPL-2.0

package tfe

import (
	"context"
	"fmt"
	"io"
	"net/url"
)

// StackState describes all the stack state-related methods that the
// HCP Terraform API supports.
type StackStates interface {
	// List returns the stack states for a stack.
	List(ctx context.Context, stackID string, opts *StackStateListOptions) (*StackStateList, error)
	// Read returns a stack state by its ID.
	Read(ctx context.Context, stackStateID string) (*StackState, error)
	// Description returns the state description for the given stack state.
	// The description is returned as an io.ReadCloser and should be closed and
	// unmarshaled by the caller.
	Description(ctx context.Context, stackStateID string) (io.ReadCloser, error)
}

// StackState represents a stack state.
type StackState struct {
	// Attributes
	ID                    string            `jsonapi:"primary,stack-states"`
	Generation            int               `jsonapi:"attr,generation"`
	Status                string            `jsonapi:"attr,status"`
	Deployment            string            `jsonapi:"attr,deployment"`
	Components            []*StackComponent `jsonapi:"attr,components"`
	IsCurrent             bool              `jsonapi:"attr,is-current"`
	ResourceInstanceCount int               `jsonapi:"attr,resource-instance-count"`

	// Relationships
	Stack              *Stack              `jsonapi:"relation,stack"`
	StackDeploymentRun *StackDeploymentRun `jsonapi:"relation,stack-deployment-run"`

	// Links
	Links map[string]interface{} `jsonapi:"links,omitempty"`
}

// StackStateList represents a list of stack states.
type StackStateList struct {
	*Pagination
	Items []*StackState
}

type stackStates struct {
	client *Client
}

// StackStateListOptions represents the options for listing stack states.
type StackStateListOptions struct {
	ListOptions
}

// List returns the stack states for a stack.
func (s stackStates) List(ctx context.Context, stackID string, opts *StackStateListOptions) (*StackStateList, error) {
	req, err := s.client.NewRequest("GET", fmt.Sprintf("stacks/%s/stack-states", url.PathEscape(stackID)), opts)
	if err != nil {
		return nil, err
	}

	states := StackStateList{}
	if err := req.Do(ctx, &states); err != nil {
		return nil, err
	}

	return &states, nil
}

// Read returns a stack state by its ID.
func (s stackStates) Read(ctx context.Context, stackStateID string) (*StackState, error) {
	req, err := s.client.NewRequest("GET", fmt.Sprintf("stack-states/%s", url.PathEscape(stackStateID)), nil)
	if err != nil {
		return nil, err
	}

	state := StackState{}
	if err := req.Do(ctx, &state); err != nil {
		return nil, err
	}

	return &state, nil
}

// Description returns the state description for the given stack state.
func (s stackStates) Description(ctx context.Context, stackStateID string) (io.ReadCloser, error) {
	req, err := s.client.NewRequest("GET", fmt.Sprintf("stack-states/%s/description", url.PathEscape(stackStateID)), nil)
	if err != nil {
		return nil, err
	}

	return req.DoRaw(ctx)
}
