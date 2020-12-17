package tfe

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
)

// Compile-time proof of interface implementation.
var _ StateVersionOutputs = (*stateVersionOutputs)(nil)

//State version outputs are the output values from a Terraform state file.
//They include the name and value of the output, as well as a sensitive boolean
//if the value should be hidden by default in UIs.
//
// TFE API docs: https://www.terraform.io/docs/cloud/api/state-version-outputs.html
type StateVersionOutputs interface {
	Read(ctx context.Context, outputID string) (*StateVersionOutput, error)

	ReadSimpler(ctx context.Context, outputID string) (*StateVersionOutputAllKinds, error)
}

type stateVersionOutputs struct {
	client *Client
}

type StateVersionOutput struct {
	ID        string `jsonapi:"primary,state-version-outputs"`
	Name      string `jsonapi:"attr,name"`
	Sensitive bool   `jsonapi:"attr,sensitive"`
	Type      string `jsonapi:"attr,type"`
	Value     string `jsonapi:"attr,value"`
}

type StateVersionOutputAllKinds struct {
	ID        string `jsonapi:"primary,state-version-outputs"`
	Name      string `jsonapi:"attr,name"`
	Sensitive bool   `jsonapi:"attr,sensitive"`
	Type      string `jsonapi:"attr,type"`
	Value     interface{} `jsonapi:"attr,value"`
}


func (s *stateVersionOutputs) Read(ctx context.Context, outputID string) (*StateVersionOutput, error) {
	if !validStringID(&outputID) {
		return nil, errors.New("invalid value for run ID")
	}

	u := fmt.Sprintf("state-version-outputs/%s", url.QueryEscape(outputID))
	req, err := s.client.newRequest("GET", u, nil)
	if err != nil {
		return nil, err
	}

	so := &StateVersionOutput{}
	err = s.client.do(ctx, req, so)
	if err != nil {
		return nil, err
	}

	return so, nil
}

// This is here because the original implementation, above, does not support State Outputs that are not strings, and errors when you try to read one.
// State outputs can also be arrays, as in our use case.  So we support the interface{} type for any data type, and the 'type' field can be used to determine what kind of data you have.
// The underlying JSON API parser does not support this type of changeable output, or so it would seem, so we shall use a more raw approach.
func (s *stateVersionOutputs) ReadSimpler(ctx context.Context, outputID string) (*StateVersionOutputAllKinds, error) {
	if !validStringID(&outputID) {
		return nil, errors.New("invalid value for run ID")
	}

	u := fmt.Sprintf("state-version-outputs/%s", url.QueryEscape(outputID))
	req, err := s.client.newRequest("GET", u, nil)
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	err = s.client.do(ctx, req, &buf)
	if err != nil {
		return nil, err
	}

	// Convert to a json map
	var result map[string]interface{}
	err = json.Unmarshal(buf.Bytes(), &result)
	if err != nil {
		return nil, err
	}

	// Create a new value to hold our output
	so := &StateVersionOutputAllKinds{}
	so.ID = outputID

	// Read in the data from the json.
	// Look for an object called Data, and below that, attributes..
	if data, ok := result["data"].(map[string]interface{}); ok {
		if attr, ok := data["attributes"].(map[string]interface{}); ok {
			if name, ok := attr["name"].(string); ok {
				so.Name = name
			}
			if sensitive, ok := attr["name"].(bool); ok {
				so.Sensitive = sensitive
			}
			if tp, ok := attr["type"].(string); ok {
				so.Type = tp
			}
			if value, ok := attr["value"]; ok {
				so.Value = value
			}
		}
	}

	return so, nil
}
