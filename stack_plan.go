// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package tfe

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

// NOTE WELL: This is a beta feature and is subject to change until noted otherwise in the
// release notes.
type StackPlans interface {
	// Read returns a stack plan by its ID.
	Read(ctx context.Context, stackPlanId string) (*StackPlan, error)
	DownloadPlanDescription(ctx context.Context, stackPlanId string) ([]byte, error)

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

// TODO: Maybe parse the plan description here?
func (s stackPlans) DownloadPlanDescription(ctx context.Context, stackPlanId string) ([]byte, error) {
	// Create a new request.
	baseUrl := s.client.BaseURL()
	href := fmt.Sprintf("stack-plans/%s/plan-description", url.PathEscape(stackPlanId))
	url, err := baseUrl.Parse(href)
	if err != nil {

		return nil, err
	}

	req, err := http.NewRequest("GET", url.String(), nil)
	if err != nil {

		return nil, err
	}
	req = req.WithContext(ctx)

	// Attach the default headers.
	for k, v := range s.client.headers {
		req.Header[k] = v
	}

	req.Header.Set("Authorization", "Bearer "+s.client.token)

	// Retrieve the next chunk.
	resp, err := s.client.http.HTTPClient.Do(req)

	if err != nil {

		return nil, err
	}
	defer resp.Body.Close()

	// Basic response checking.
	if err := checkResponseCode(resp); err != nil {

		return nil, err
	}

	// Read the retrieved chunk.
	b, err := io.ReadAll(resp.Body)
	if err != nil {

		return nil, err
	}

	return b, nil
}
