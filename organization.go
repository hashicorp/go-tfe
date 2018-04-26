package tfe

import (
	"errors"
)

// Organization encapsulates all data fields of a TFE Organization.
type Organization struct {
	// The organization name. Globally unique within a TFE instance.
	Name *string `json:"name,omitempty"`

	// Email address associated with the organization. It is possible for
	// this value to be empty.
	Email *string `json:"email,omitempty"`

	// Authentication policy for collaborators of the organization. Identifies
	// 2FA requirements or other required authentication for collaborators
	// of the organization.
	CollaboratorAuthPolicy *string `json:"collaborator-auth-policy,omitempty"`

	// The TFE plan. May be "trial", "pro", or "premium". For private (PTFE)
	// installations this will always be "premium".
	EnterprisePlan *string `json:"enterprise-plan,omitempty"`

	// Creation time of the organization.
	CreatedAt *string `json:"created-at,omitempty"`

	// Expiration timestamp of the organization's trial period. Only applicable
	// if the EnterprisePlan is "trial".
	TrialExpiresAt *string `json:"trial-expires-at,omitempty"`

	// Permissions the current user can perform against the organization.
	Permissions Permissions `json:"permissions"`
}

// Organizations returns all of the organizations visible to the current user.
func (c *Client) Organizations() ([]*Organization, error) {
	var result jsonapiOrganizations

	if _, err := c.do(&request{
		method: "GET",
		path:   "/api/v2/organizations",
		output: &result,
	}); err != nil {
		return nil, err
	}

	output := make([]*Organization, len(result))
	for i, org := range result {
		output[i] = org.Organization
	}

	return output, nil
}

// Organization is used to look up a single organization by its name.
func (c *Client) Organization(name string) (*Organization, error) {
	var output jsonapiOrganization

	if _, err := c.do(&request{
		method: "GET",
		path:   "/api/v2/organizations/" + name,
		output: &output,
	}); err != nil {
		return nil, err
	}

	return output.Organization, nil
}

// CreateOrganizationParams holds all of the settable parameters to pass
// during organization creation.
type CreateOrganizationInput struct {
	// The organization name.
	Name *string

	// Email address associated with the organization.
	Email *string
}

// Valid checks if the input is filled sufficiently.
func (i *CreateOrganizationInput) Valid() error {
	if !validStringID(i.Name) {
		return errors.New("Invalid value for Name")
	}
	if !validString(i.Email) {
		return errors.New("Email is required")
	}
	return nil
}

// CreateOrganizationOutput holds the return values from an organization
// creation request.
type CreateOrganizationOutput struct {
	// A reference to the newly-created organization.
	Organization *Organization
}

// CreateOrganization creates a new organization with the given parameters.
func (c *Client) CreateOrganization(input *CreateOrganizationInput) (
	*CreateOrganizationOutput, error) {

	if err := input.Valid(); err != nil {
		return nil, err
	}

	// Create the special JSONAPI params object.
	jsonapiParams := jsonapiOrganization{
		Organization: &Organization{
			Name:  input.Name,
			Email: input.Email,
		},
	}

	var output jsonapiOrganization

	// Send the request.
	if _, err := c.do(&request{
		method: "POST",
		path:   "/api/v2/organizations",
		input:  jsonapiParams,
		output: &output,
	}); err != nil {
		return nil, err
	}

	return &CreateOrganizationOutput{
		Organization: output.Organization,
	}, nil
}

// DeleteOrganizationInput holds parameters used during organization deletion.
type DeleteOrganizationInput struct {
	// The name of the organization to delete. Required.
	Name *string
}

// Valid checks if the input is sufficiently filled.
func (i *DeleteOrganizationInput) Valid() error {
	if !validStringID(i.Name) {
		return errors.New("Invalid value for Name")
	}
	return nil
}

// DeleteOrganizationOutput stores results from an org deletion request.
type DeleteOrganizationOutput struct{}

// DeleteOrganization deletes the organization matching the given parameters.
func (c *Client) DeleteOrganization(input *DeleteOrganizationInput) (
	*DeleteOrganizationOutput, error) {

	if err := input.Valid(); err != nil {
		return nil, err
	}

	// Send the request.
	if resp, err := c.do(&request{
		method: "DELETE",
		path:   "/api/v2/organizations/" + *input.Name,
	}); err != nil {
		return nil, err
	} else {
		resp.Body.Close()
	}

	return &DeleteOrganizationOutput{}, nil
}

// ModifyOrganizationInput contains the parameters used for modifying an
// existing organization. Any optional values left empty will be left intact
// on the organization.
type ModifyOrganizationInput struct {
	// The organization to modify. Required.
	Name *string

	// Renames the organization to the given string.
	Rename *string

	// The email address associated with the organization.
	Email *string
}

// Valid checks that the input is sufficiently filled.
func (i *ModifyOrganizationInput) Valid() error {
	if !validStringID(i.Name) {
		return errors.New("Invalid value for Name")
	}
	return nil
}

// ModifyOrganizationOutput contains response values from an organization
// modify request.
type ModifyOrganizationOutput struct {
	// The updated view of the organization.
	Organization *Organization
}

// ModifyOrganization is used to adjust attributes on an existing organization.
func (c *Client) ModifyOrganization(input *ModifyOrganizationInput) (
	*ModifyOrganizationOutput, error) {

	if err := input.Valid(); err != nil {
		return nil, err
	}

	// Create the special JSON API payload.
	jsonapiParams := jsonapiOrganization{
		Organization: &Organization{
			Name:  input.Rename,
			Email: input.Email,
		},
	}

	var output jsonapiOrganization

	// Send the request
	if _, err := c.do(&request{
		method: "PATCH",
		path:   "/api/v2/organizations/" + *input.Name,
		input:  jsonapiParams,
		output: &output,
	}); err != nil {
		return nil, err
	}

	return &ModifyOrganizationOutput{
		Organization: output.Organization,
	}, nil
}

// OrganizationNameSort provides sorting by the organization name.
type OrganizationNameSort []*Organization

func (o OrganizationNameSort) Len() int           { return len(o) }
func (o OrganizationNameSort) Less(a, b int) bool { return *o[a].Name < *o[b].Name }
func (o OrganizationNameSort) Swap(a, b int)      { o[a], o[b] = o[b], o[a] }

// Internal type to satisfy the jsonapi interface for a single organization.
type jsonapiOrganization struct{ *Organization }

func (jsonapiOrganization) GetName() string    { return "organizations" }
func (jsonapiOrganization) GetID() string      { return "" }
func (jsonapiOrganization) SetID(string) error { return nil }
func (jsonapiOrganization) SetToOneReferenceID(string, string) error {
	return nil
}

// Internal type to satisfy the jsonapi interface for org indexes.
type jsonapiOrganizations []jsonapiOrganization

func (jsonapiOrganizations) GetName() string    { return "organizations" }
func (jsonapiOrganizations) GetID() string      { return "" }
func (jsonapiOrganizations) SetID(string) error { return nil }
func (jsonapiOrganizations) SetToOneReferenceID(string, string) error {
	return nil
}
