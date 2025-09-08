// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package tfe

import (
	"context"
	"fmt"
	"net/url"
)

type StackConfigurationSummaries interface {
	// List lists all the stack configuration summaries for a stack.
	List(ctx context.Context, stackID string, options *StackConfigurationSummaryListOptions) (*StackConfigurationSummaryList, error)
}

type stackConfigurationSummaries struct {
	client *Client
}

var _ StackConfigurationSummaries = &stackConfigurationSummaries{}

type StackConfigurationSummaryList struct {
	*Pagination
	Items []*StackConfigurationSummary
}

type StackConfigurationSummaryListOptions struct {
	ListOptions
}

type StackConfigurationSummary struct {
	ID             string `jsonapi:"primary,stack-configuration-summaries"`
	Status         string `jsonapi:"attr,status"`
	SequenceNumber int    `jsonapi:"attr,sequence-number"`
}

func (s stackConfigurationSummaries) List(ctx context.Context, stackID string, options *StackConfigurationSummaryListOptions) (*StackConfigurationSummaryList, error) {
	if !validStringID(&stackID) {
		return nil, fmt.Errorf("invalid stack ID: %s", stackID)
	}

	if options == nil {
		options = &StackConfigurationSummaryListOptions{}
	}

	req, err := s.client.NewRequest("GET", fmt.Sprintf("stacks/%s/stack-configuration-summaries", url.PathEscape(stackID)), options)
	if err != nil {
		return nil, err
	}

	scl := &StackConfigurationSummaryList{}
	err = req.Do(ctx, scl)
	if err != nil {
		return nil, err
	}

	return scl, nil
}
