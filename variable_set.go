package tfe

import (
	"context"
	"errors"
	"fmt"
	"net/url"
)

// Compile-time proof of interface implementation.
var _ VariableSets = (*variableSets)(nil)

// VariableSets describes all the Variable Set related methods that the
// Terraform Enterprise API supports.
//
// TFE API docs: https://www.terraform.io/cloud-docs/api-docs/variable-sets
type VariableSets interface {
	// List all the variable sets within an organization.
	List(ctx context.Context, organization string, options *VariableSetListOptions) (*VariableSetList, error)

	// Create is used to create a new variable set.
	Create(ctx context.Context, organization string, options *VariableSetCreateOptions) (*VariableSet, error)

	// Read a variable set by its ID.
	Read(ctx context.Context, variableSetVariableSetReadOptionsID string, options *VariableSetReadOptions) (*VariableSet, error)

	// Update an existing variable set.
	Update(ctx context.Context, variableSetID string, options *VariableSetUpdateOptions) (*VariableSet, error)

	// Delete a variable set by ID.
	Delete(ctx context.Context, variableSetID string) error

	// Assign a variable set to workspaces
	Assign(ctx context.Context, variableSetID string, options *VariableSetAssignOptions) (*VariableSet, error)
}

type variableSets struct {
	client *Client
}

type VariableSetList struct {
	*Pagination
	Items []*VariableSet
}

type VariableSet struct {
	ID          string `jsonapi:"primary,varsets"`
	Name        string `jsonapi:"attr,name"`
	Description string `jsonapi:"attr,description"`
	Global      bool   `jsonapi:"attr,global"`

	// Relations
	Organization *Organization          `jsonapi:"relation,organization"`
	Workspaces   []*Workspace           `jsonapi:"relation,workspaces,omitempty"`
	Variables    []*VariableSetVariable `jsonapi:"relation,vars,omitempty"`
}

// A list of relations to include. See available resources
// https://www.terraform.io/docs/cloud/api/admin/organizations.html#available-related-resources
type VariableSetIncludeOps string

const (
	VariableSetWorkspaces VariableSetIncludeOps = "workspaces"
	VariableSetVars       VariableSetIncludeOps = "vars"
)

type VariableSetListOptions struct {
	ListOptions
	Include string `url:"include"`
}

func (o VariableSetListOptions) valid() error {
	return nil
}

// List all Variable Sets in the organization
func (s *variableSets) List(ctx context.Context, organization string, options *VariableSetListOptions) (*VariableSetList, error) {
	if !validStringID(&organization) {
		return nil, ErrInvalidOrg
	}
	if options != nil {
		if err := options.valid(); err != nil {
			return nil, err
		}
	}

	u := fmt.Sprintf("organizations/%s/varsets", url.QueryEscape(organization))
	req, err := s.client.newRequest("GET", u, options)
	if err != nil {
		return nil, err
	}

	vl := &VariableSetList{}
	err = s.client.do(ctx, req, vl)
	if err != nil {
		return nil, err
	}

	return vl, nil
}

// VariableSetCreateOptions represents the options for creating a new variable set within in a organization.
type VariableSetCreateOptions struct {
	// Type is a public field utilized by JSON:API to
	// set the resource type via the field tag.
	// It is not a user-defined value and does not need to be set.
	// https://jsonapi.org/format/#crud-creating
	Type string `jsonapi:"primary,vars"`

	// The name of the variable set.
	// Affects variable precedence when there are conflicts between Variable Sets
	// https://www.terraform.io/cloud-docs/api-docs/variable-sets#apply-variable-set-to-workspaces
	Name *string `jsonapi:"attr,name"`

	// A description to provide context for the variable set.
	Description *string `jsonapi:"attr,description"`

	// If true the variable set is considered in all runs in the organization.
	Global *bool `jsonapi:"attr,global"`
}

func (o VariableSetCreateOptions) valid() error {
	if !validString(o.Name) {
		return ErrRequiredName
	}
	if o.Global == nil {
		return errors.New("global flag is required")
	}
	return nil
}

// Create is used to create a new variable set.
func (s *variableSets) Create(ctx context.Context, organization string, options *VariableSetCreateOptions) (*VariableSet, error) {
	if !validStringID(&organization) {
		return nil, ErrInvalidOrg
	}
	if options != nil {
		if err := options.valid(); err != nil {
			return nil, err
		}
	}

	u := fmt.Sprintf("organizations/%s/varsets", url.QueryEscape(organization))
	req, err := s.client.newRequest("POST", u, options)
	if err != nil {
		return nil, err
	}

	vl := &VariableSet{}
	err = s.client.do(ctx, req, vl)
	if err != nil {
		return nil, err
	}

	return vl, nil
}

type VariableSetReadOptions struct {
	Include *[]VariableSetIncludeOps `url:"include:omitempty"`
}

// Read is used to inspect a given variable set based on ID
func (s *variableSets) Read(ctx context.Context, variableSetID string, options *VariableSetReadOptions) (*VariableSet, error) {
	if !validStringID(&variableSetID) {
		return nil, errors.New("invalid variable set ID")
	}

	u := fmt.Sprintf("varsets/%s", url.QueryEscape(variableSetID))
	req, err := s.client.newRequest("GET", u, options)
	if err != nil {
		return nil, err
	}

	vs := &VariableSet{}
	err = s.client.do(ctx, req, vs)
	if err != nil {
		return nil, err
	}

	return vs, err
}

// VariableSetUpdateOptions represents the options for updating a variable set.
type VariableSetUpdateOptions struct {
	// Type is a public field utilized by JSON:API to
	// set the resource type via the field tag.
	// It is not a user-defined value and does not need to be set.
	// https://jsonapi.org/format/#crud-creating
	Type string `jsonapi:"primary,vars"`

	// The name of the variable set.
	// Affects variable precedence when there are conflicts between Variable Sets
	// https://www.terraform.io/cloud-docs/api-docs/variable-sets#apply-variable-set-to-workspaces
	Name *string `jsonapi:"attr,name"`

	// A description to provide context for the variable set.
	Description *string `jsonapi:"attr,description,omitempty"`

	// If true the variable set is considered in all runs in the organization.
	Global *bool `jsonapi:"attr,global,omitempty"`

	Include *[]VariableSetIncludeOps `url:"include:omitempty"`
}

func (s *variableSets) Update(ctx context.Context, variableSetID string, options *VariableSetUpdateOptions) (*VariableSet, error) {
	if !validStringID(&variableSetID) {
		return nil, errors.New("invalid value for variable set ID")
	}

	u := fmt.Sprintf("varsets/%s", url.QueryEscape(variableSetID))
	req, err := s.client.newRequest("PATCH", u, options)
	if err != nil {
		return nil, err
	}

	v := &VariableSet{}
	err = s.client.do(ctx, req, v)
	if err != nil {
		return nil, err
	}

	return v, nil
}

// Delete a variable set by its ID.
func (s *variableSets) Delete(ctx context.Context, variableSetID string) error {
	if !validStringID(&variableSetID) {
		return errors.New("invalid value for variable set ID")
	}

	u := fmt.Sprintf("varsets/%s", url.QueryEscape(variableSetID))
	req, err := s.client.newRequest("DELETE", u, nil)
	if err != nil {
		return err
	}

	return s.client.do(ctx, req, nil)
}

// VariableSetAssignOptions represents a subset of update options specifically for assigning variable sets to workspaces
type VariableSetAssignOptions struct {
	// Type is a public field utilized by JSON:API to
	// set the resource type via the field tag.
	// It is not a user-defined value and does not need to be set.
	// https://jsonapi.org/format/#crud-creating
	Type string `jsonapi:"primary,vars"`

	// Used to set the variable set from Global to not Global if necessary
	Global *bool `jsonapi:"attr,global"`

	// The workspaces to be assigned to. An empty set means remove all assignments
	Workspaces []*Workspace `jsonapi:"relation,workspaces"`
}

// Use Update to assign a variable set to workspaces
func (s *variableSets) Assign(ctx context.Context, variableSetID string, options *VariableSetAssignOptions) (*VariableSet, error) {
	if options == nil || options.Workspaces == nil {
		return nil, errors.New("no workspaces list provided")
	}

	options.Global = Bool(false)

	// We force inclusion of workspaces as that is the primary data for which we are concerned with confirming changes.
	u := fmt.Sprintf("varsets/%s?include=%s", url.QueryEscape(variableSetID), VariableSetWorkspaces)
	req, err := s.client.newRequest("PATCH", u, options)
	if err != nil {
		return nil, err
	}

	v := &VariableSet{}
	err = s.client.do(ctx, req, v)
	if err != nil {
		return nil, err
	}

	return v, nil
}
