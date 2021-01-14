package tfe

import (
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
}

type stateVersionOutputs struct {
	client *Client
}

type StateVersionOutput struct {
	ID        string `jsonapi:"primary,state-version-outputs"`
	Name      string `jsonapi:"attr,name"`
	Sensitive bool   `jsonapi:"attr,sensitive"`
	Type      string `jsonapi:"attr,type"`
	Value     OutputValue `jsonapi:"attr,value"`
}

// Since the Output can be one of many types, and we don't want to use interface{} here, this type
// can store all types of output.   There may be more types that are not yet implemented here.
type OutputValue struct {
	ValueBool 	bool
	ValueInt	int
	ValueString	string
	ValueArray	[]interface{}
	ValueMap	map[string]interface{}
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

// Allow the 'value' component of this state output to be unmarshaled to the right kind of data.
func (ov *OutputValue) UnmarshalJSON(b []byte) error {
	// unmarshal the data and then store it.
	var result interface{}
	err := json.Unmarshal(b, &result)
	if err != nil {
		return err
	}

	// Test what kind of data we have here and store it in the right place.
	switch v := result.(type) {
	case int:
		ov.ValueInt = v
	case string:
		ov.ValueString = v
	case bool:
		ov.ValueBool = v
	case []interface{}:
		ov.ValueArray = v
	case map[string]interface{}:
		ov.ValueMap = v
	default:
		return fmt.Errorf("unknown output type: %v", v)
	}
	return nil
}

