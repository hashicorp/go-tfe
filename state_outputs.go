package tfe

import (
	"context"
	"errors"
	"fmt"
	"net/url"
)

// Compile-time proof of interface implementation.
var _ StateOutputs = (*stateOutputs)(nil)

//State version outputs are the output values from a Terraform state file.
//They include the name and value of the output, as well as a sensitive boolean
//if the value should be hidden by default in UIs.
//
// TFE API docs: https://www.terraform.io/docs/cloud/api/state-version-outputs.html
type StateOutputs interface {
	Read(ctx context.Context, outputID string) (*StateOutput, error)
}

type stateOutputs struct {
	client *Client
}

type StateOutput struct {
	ID        string `jsonapi:"primary,state-version-outputs"`
	Name      string `jsonapi:"attr,name"`
	Sensitive bool   `jsonapi:"attr,sensitive"`
	Type      string `jsonapi:"attr,type"`
	Value     string `jsonapi:"attr,value"`
}

func (s *stateOutputs) Read(ctx context.Context, outputID string) (*StateOutput, error) {
	if !validStringID(&outputID) {
		return nil, errors.New("invalid value for run ID")
	}

	u := fmt.Sprintf("state-version-outputs/%s", url.QueryEscape(outputID))
	req, err := s.client.newRequest("GET", u, nil)
	if err != nil {
		return nil, err
	}

	so := &StateOutput{}
	err = s.client.do(ctx, req, so)
	if err != nil {
		return nil, err
	}

	return so, nil
}
