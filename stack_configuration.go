// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package tfe

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
)

// NOTE WELL: This is a beta feature and is subject to change until noted otherwise in the
// release notes.
type StackConfigurations interface {
	// Read returns a uncontrolled configuration source by its ID.
	Read(ctx context.Context, stackConfigurationId string) (*StackConfiguration, error)

	JsonSchemas(ctx context.Context, stackConfigurationId string) (json.RawMessage, error)

	StackPlans(ctx context.Context, stackConfigurationId string, options *StackPlansOptions) (*StackPlansList, error)
}

type stackConfigurations struct {
	client *Client
}

var _ StackConfigurations = &stackConfigurations{}

type StackConfiguration struct {
	ID                        string `jsonapi:"primary,stack-configurations"`
	Status                    string `jsonapi:"attr,status"`
	SequenceNumber            int    `jsonapi:"attr,sequence-number"`
	StackConfigSourceAddress  string `jsonapi:"attr,stack-config-source-address"`
	TerraformCliVersion       string `jsonapi:"attr,terraform-cli-version"`
	TerraformCliConfigVersion string `jsonapi:"attr,terraform-cli-config-version"`
}

type StackPlansList struct {
	*Pagination
	Items []*StackPlan
}

type StackPlansIncludeOpt string

const (
	StackPlansIncludeOperations StackPlansIncludeOpt = "stack_plan_operations"
)

type StackPlansOptions struct {
	Include []StackPlansIncludeOpt `url:"include,omitempty"`
}

func (s stackConfigurations) Read(ctx context.Context, stackConfigurationId string) (*StackConfiguration, error) {
	req, err := s.client.NewRequest("GET", fmt.Sprintf("stack-configurations/%s", url.PathEscape(stackConfigurationId)), nil)
	if err != nil {
		return nil, err
	}

	ucs := &StackConfiguration{}
	err = req.Do(ctx, ucs)
	if err != nil {
		return nil, err
	}

	return ucs, nil
}

func (s stackConfigurations) JsonSchemas(ctx context.Context, stackConfigurationId string) (json.RawMessage, error) {
	req, err := s.client.NewRequest("GET", fmt.Sprintf("stack-configurations/%s/json-schemas", url.PathEscape(stackConfigurationId)), nil)
	if err != nil {
		return nil, err
	}

	var raw json.RawMessage
	err = req.Do(ctx, &raw)
	if err != nil {
		return nil, err
	}

	return raw, nil
}

func (s stackConfigurations) StackPlans(ctx context.Context, stackConfigurationId string, options *StackPlansOptions) (*StackPlansList, error) {
	if err := options.valid(); err != nil {
		return nil, err
	}

	req, err := s.client.NewRequest("GET", fmt.Sprintf("stack-configurations/%s/stack-plans", url.PathEscape(stackConfigurationId)), options)
	if err != nil {
		return nil, err
	}

	planList := &StackPlansList{}
	err = req.Do(ctx, planList)
	if err != nil {
		return nil, err
	}

	return planList, nil
}

func (o *StackPlansOptions) valid() error {
	return nil
}
