package tfe

import (
	"errors"
	"fmt"
	"time"
)

// Policies handles communication with the policy related methods of the
// Terraform Enterprise API.
//
// TFE API docs: https://www.terraform.io/docs/enterprise/api/policies.html
type Policies struct {
	client *Client
}

// EnforcementLevel represents an enforcement level.
type EnforcementLevel string

// List the available enforcement types.
const (
	EnforcementAdvisory EnforcementLevel = "advisory"
	EnforcementHard     EnforcementLevel = "hard-mandatory"
	EnforcementSoft     EnforcementLevel = "soft-mandatory"
)

// Policy represents a Terraform Enterprise policy.
type Policy struct {
	ID        string       `jsonapi:"primary,policies"`
	Name      string       `jsonapi:"attr,name"`
	Enforce   *Enforcement `jsonapi:"attr,enforce"`
	UpdatedAt time.Time    `jsonapi:"attr,updated-at,iso8601"`
}

// Enforcement describes a enforcement.
type Enforcement struct {
	Path string           `json:"path"`
	Mode EnforcementLevel `json:"mode"`
}

// ListPoliciesOptions represents the options for listing policies.
type ListPoliciesOptions struct {
	ListOptions
}

// List all the policies for a given organization
func (s *Policies) List(organization string, options ListPoliciesOptions) ([]*Policy, error) {
	if !validStringID(&organization) {
		return nil, errors.New("Invalid value for organization")
	}

	u := fmt.Sprintf("organizations/%s/policies", organization)
	req, err := s.client.newRequest("GET", u, &options)
	if err != nil {
		return nil, err
	}

	result, err := s.client.do(req, []*Workspace{})
	if err != nil {
		return nil, err
	}

	var ps []*Policy
	for _, p := range result.([]interface{}) {
		ps = append(ps, p.(*Policy))
	}

	return ps, nil
}

// CreatePolicyOptions represents the options for creating a new policy.
type CreatePolicyOptions struct {
	// For internal use only!
	ID string `jsonapi:"primary,policies"`

	// The name of the policy.
	Name *string `jsonapi:"attr,name,omitempty"`

	// The enforcement level of the policy.
	Enforce *EnforcementOptions `jsonapi:"attr,enforce,omitempty"`
}

// EnforcementOptions represents the enforcement options of a policy.
type EnforcementOptions struct {
	Path *string           `json:"name,omitempty"`
	Mode *EnforcementLevel `json:"mode,omitempty"`
}

func (o CreatePolicyOptions) valid() error {
	if !validStringID(o.Name) {
		return errors.New("Invalid value for name")
	}
	if o.Enforce == nil || o.Enforce.Mode == nil {
		return errors.New("Invalid value for enforce mode")
	}
	return nil
}

// Create a policy and associate it with an organization.
func (s *Policies) Create(organization string, options CreatePolicyOptions) (*Policy, error) {
	if !validStringID(&organization) {
		return nil, errors.New("Invalid value for organization")
	}
	if err := options.valid(); err != nil {
		return nil, err
	}

	// Make sure we don't send a user provided ID.
	options.ID = ""

	u := fmt.Sprintf("organizations/%s/policies", organization)
	req, err := s.client.newRequest("POST", u, &options)
	if err != nil {
		return nil, err
	}

	p, err := s.client.do(req, &Policy{})
	if err != nil {
		return nil, err
	}

	return p.(*Policy), err
}

// Upload the policy content of the policy.
func (s *Policies) Upload(policyID string, content []byte) error {
	if !validStringID(&policyID) {
		return errors.New("Invalid value for policy ID")
	}

	u := fmt.Sprintf("policies/%s/upload", policyID)
	req, err := s.client.newRequest("PUT", u, content)
	if err != nil {
		return err
	}

	_, err = s.client.do(req, nil)

	return err
}

// UpdatePolicyOptions represents the options for updating a policy.
type UpdatePolicyOptions struct {
	// For internal use only!
	ID string `jsonapi:"primary,policies"`

	// The name of the policy.
	Name *string `jsonapi:"attr,name,omitempty"`

	// The enforcement level of the policy.
	Enforce *EnforcementOptions `jsonapi:"attr,enforce,omitempty"`
}

func (o UpdatePolicyOptions) valid() error {
	if !validStringID(o.Name) {
		return errors.New("Invalid value for name")
	}
	if o.Enforce == nil || o.Enforce.Mode == nil {
		return errors.New("Invalid value for enforce mode")
	}
	return nil
}

// Update an existing policy.
func (s *Policies) Update(policyID string, options UpdatePolicyOptions) (*Policy, error) {
	if !validStringID(&policyID) {
		return nil, errors.New("Invalid value for policy ID")
	}
	if err := options.valid(); err != nil {
		return nil, err
	}

	// Make sure we don't send a user provided ID.
	options.ID = ""

	req, err := s.client.newRequest("PATCH", "policies/"+policyID, &options)
	if err != nil {
		return nil, err
	}

	p, err := s.client.do(req, &Policy{})
	if err != nil {
		return nil, err
	}

	return p.(*Policy), err
}

// Delete an organization policy.
func (s *Policies) Delete(policyID string) error {
	if !validStringID(&policyID) {
		return errors.New("Invalid value for policy ID")
	}

	req, err := s.client.newRequest("DELETE", "policies/"+policyID, nil)
	if err != nil {
		return err
	}

	_, err = s.client.do(req, nil)

	return err
}
