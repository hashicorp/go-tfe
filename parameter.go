package tfe

import (
	"context"
	"errors"
	"fmt"
	"net/url"
)

// Compile-time proof of interface implementation.
var _ Parameters = (*parameters)(nil)

// Parameters describes all the parameter related methods that the Terraform
// Enterprise API supports.
//
// TFE API docs: https://www.terraform.io/docs/enterprise/api/policy-set-params.html
type Parameters interface {
	// List all the parameters associated with the given policy-set.
	List(ctx context.Context, policySetID string, options ParameterListOptions) (*ParameterList, error)

	// Create is used to create a new parameter.
	Create(ctx context.Context, policySetID string, options ParameterCreateOptions) (*Parameter, error)

	// Read a parameter by its ID.
	Read(ctx context.Context, policySetID string, parameterID string) (*Parameter, error)

	// Update values of an existing parameter.
	Update(ctx context.Context, policySetID string, parameterID string, options ParameterUpdateOptions) (*Parameter, error)

	// Delete a parameter by its ID.
	Delete(ctx context.Context, policySetID string, parameterID string) error
}

// parameters implements Parameters.
type parameters struct {
	client *Client
}

// ParameterList represents a list of parameters.
type ParameterList struct {
	*Pagination
	Items []*Parameter
}

// Parameter represents a Terraform Enterprise parameter.
type Parameter struct {
	ID        string       `jsonapi:"primary,vars"`
	Key       string       `jsonapi:"attr,key"`
	Value     string       `jsonapi:"attr,value"`
	Category  CategoryType `jsonapi:"attr,category"`
	Sensitive bool         `jsonapi:"attr,sensitive"`

	// Relations
	PolicySet *PolicySet `jsonapi:"relation,configurable"`
}

// ParameterListOptions represents the options for listing parameters.
type ParameterListOptions struct {
	ListOptions
}

func (o ParameterListOptions) valid() error {
	return nil
}

// List all the parameters associated with the given policy-set.
func (s *parameters) List(ctx context.Context, policySetID string, options ParameterListOptions) (*ParameterList, error) {
	if !validStringID(&policySetID) {
		return nil, errors.New("invalid value for policy set ID")
	}
	if err := options.valid(); err != nil {
		return nil, err
	}

	u := fmt.Sprintf("policy-sets/%s/parameters", policySetID)
	req, err := s.client.newRequest("GET", u, &options)
	if err != nil {
		return nil, err
	}

	vl := &ParameterList{}
	err = s.client.do(ctx, req, vl)
	if err != nil {
		return nil, err
	}

	return vl, nil
}

// ParameterCreateOptions represents the options for creating a new parameter.
type ParameterCreateOptions struct {
	// For internal use only!
	ID string `jsonapi:"primary,vars"`

	// The name of the parameter.
	Key *string `jsonapi:"attr,key"`

	// The value of the parameter.
	Value *string `jsonapi:"attr,value,omitempty"`

	// Whether this is a Terraform or environment parameter.
	Category *CategoryType `jsonapi:"attr,category"`

	// Whether the value is sensitive.
	Sensitive *bool `jsonapi:"attr,sensitive,omitempty"`
}

func (o ParameterCreateOptions) valid() error {
	if !validString(o.Key) {
		return errors.New("key is required")
	}
	if o.Category == nil {
		return errors.New("category is required")
	}
	if *o.Category != CategoryPolicySet {
		return errors.New("category must be policy-set")
	}
	return nil
}

// Create is used to create a new parameter.
func (s *parameters) Create(ctx context.Context, policySetID string, options ParameterCreateOptions) (*Parameter, error) {
	if !validStringID(&policySetID) {
		return nil, errors.New("invalid value for policy set ID")
	}
	if err := options.valid(); err != nil {
		return nil, err
	}

	// Make sure we don't send a user provided ID.
	options.ID = ""

	u := fmt.Sprintf("policy-sets/%s/parameters", url.QueryEscape(policySetID))
	req, err := s.client.newRequest("POST", u, &options)
	if err != nil {
		return nil, err
	}

	v := &Parameter{}
	err = s.client.do(ctx, req, v)
	if err != nil {
		return nil, err
	}

	return v, nil
}

// Read a parameter by its ID.
func (s *parameters) Read(ctx context.Context, policySetID string, parameterID string) (*Parameter, error) {
	if !validStringID(&policySetID) {
		return nil, errors.New("invalid value for policy set ID")
	}
	if !validStringID(&parameterID) {
		return nil, errors.New("invalid value for parameter ID")
	}

	u := fmt.Sprintf("policy-sets/%s/parameters/%s", url.QueryEscape(policySetID), url.QueryEscape(parameterID))
	req, err := s.client.newRequest("GET", u, nil)
	if err != nil {
		return nil, err
	}

	v := &Parameter{}
	err = s.client.do(ctx, req, v)
	if err != nil {
		return nil, err
	}

	return v, err
}

// ParameterUpdateOptions represents the options for updating a parameter.
type ParameterUpdateOptions struct {
	// For internal use only!
	ID string `jsonapi:"primary,vars"`

	// The name of the parameter.
	Key *string `jsonapi:"attr,key,omitempty"`

	// The value of the parameter.
	Value *string `jsonapi:"attr,value,omitempty"`

	// Whether the value is sensitive.
	Sensitive *bool `jsonapi:"attr,sensitive,omitempty"`
}

// Update values of an existing parameter.
func (s *parameters) Update(ctx context.Context, policySetID string, parameterID string, options ParameterUpdateOptions) (*Parameter, error) {
	if !validStringID(&policySetID) {
		return nil, errors.New("invalid value for policy set ID")
	}
	if !validStringID(&parameterID) {
		return nil, errors.New("invalid value for parameter ID")
	}

	// Make sure we don't send a user provided ID.
	options.ID = parameterID

	u := fmt.Sprintf("policy-sets/%s/parameters/%s", url.QueryEscape(policySetID), url.QueryEscape(parameterID))
	req, err := s.client.newRequest("PATCH", u, &options)
	if err != nil {
		return nil, err
	}

	v := &Parameter{}
	err = s.client.do(ctx, req, v)
	if err != nil {
		return nil, err
	}

	return v, nil
}

// Delete a parameter by its ID.
func (s *parameters) Delete(ctx context.Context, policySetID string, parameterID string) error {
	if !validStringID(&policySetID) {
		return errors.New("invalid value for policy set ID")
	}
	if !validStringID(&parameterID) {
		return errors.New("invalid value for parameter ID")
	}

	u := fmt.Sprintf("policy-sets/%s/parameters/%s", url.QueryEscape(policySetID), url.QueryEscape(parameterID))
	req, err := s.client.newRequest("DELETE", u, nil)
	if err != nil {
		return err
	}

	return s.client.do(ctx, req, nil)
}
