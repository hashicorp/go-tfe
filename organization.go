package tfe

import (
	"bytes"
	"reflect"

	"github.com/google/jsonapi"
)

// The reflect type of an organization. Used during deserialization.
var organizationType = reflect.TypeOf(&Organization{})

// Organization encapsulates all data fields of a TFE Organization.
type Organization struct {
	// The organization name. Globally unique within a TFE instance.
	Name string `jsonapi:"primary,organizations"`

	// Email address associated with the organization. It is possible for
	// this value to be empty.
	Email string `jsonapi:"attr,email"`

	// Authentication policy for collaborators of the organization. Identifies
	// 2FA requirements or other required authentication for collaborators
	// of the organization.
	CollaboratorAuthPolicy string `jsonapi:"attr,collaborator-auth-policy"`

	// The TFE plan. May be "trial", "pro", or "premium". For private (PTFE)
	// installations this will always be "premium".
	EnterprisePlan string `jsonapi:"attr,enterprise-plan"`

	// Creation time of the organization.
	CreatedAt string `jsonapi:"attr,created-at"`

	// Expiration timestamp of the organization's trial period. Only applicable
	// if the EnterprisePlan is "trial".
	TrialExpiresAt string `jsonapi:"attr,trial-expires-at"`

	// Flag determining if SAML is enabled. This is an installation-wide setting
	// but is exposed through the organization API.
	SAMLEnabled bool `jsonapi:"attr,saml-enabled"`

	// The role ID in SAML which should be mapped to the "owners" team. If
	// empty, then owner access is not enabled via SAML. Any other value
	// grants SAML users with the given role ID owner-level access to the
	// organization.
	SAMLOwnersRoleID string `jsonapi:attr:"owners-team-saml-role-id"`
}

// Organizations returns all of the organizations visible to the current user.
func (c *Client) Organizations() ([]*Organization, error) {
	resp, err := c.do(&request{
		method: "GET",
		path:   "/api/v2/organizations",
	})
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	apiOrgs, err := jsonapi.UnmarshalManyPayload(
		resp.Body,
		organizationType,
	)
	if err != nil {
		return nil, err
	}

	orgs := make([]*Organization, len(apiOrgs))
	for i, org := range apiOrgs {
		orgs[i] = org.(*Organization)
	}
	return orgs, nil
}

// Organization is used to look up a single organization by its name.
func (c *Client) Organization(name string) (*Organization, error) {
	resp, err := c.do(&request{
		method: "GET",
		path:   "/api/v2/organizations/" + name,
	})
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var org Organization
	if err := jsonapi.UnmarshalPayload(resp.Body, &org); err != nil {
		return nil, err
	}

	return &org, nil
}

// CreateOrganizationParams holds all of the settable parameters to pass
// during organization creation.
type CreateOrganizationInput struct {
	// The organization name.
	Name string

	// Email address associated with the organization.
	Email string

	// The optional SAML role ID which maps to the owners team. If this is
	// not set, then the owners team cannot be accessed when logging in with
	// SAML.
	OwnersTeamSAMLRoleID string
}

type createOrganizationJSONAPI struct {
	ID                   string `jsonapi:"primary,organizations"`
	Name                 string `jsonapi:"attr,name"`
	Email                string `jsonapi:"attr,email"`
	OwnersTeamSAMLRoleID string `jsonapi:"attr,owners-team-saml-role-id"`
}

// CreateOrganization creates a new organization with the given parameters.
// Returns the new organization and any error.
func (c *Client) CreateOrganization(input *CreateOrganizationInput) (
	*Organization, error) {

	// Create the special JSONAPI params object.
	jsonapiParams := &createOrganizationJSONAPI{
		Name:                 input.Name,
		Email:                input.Email,
		OwnersTeamSAMLRoleID: input.OwnersTeamSAMLRoleID,
	}

	// Encode the JSONAPI payload.
	payload := bytes.NewBuffer(nil)
	if err := jsonapi.MarshalPayload(payload, jsonapiParams); err != nil {
		return nil, err
	}

	// Send the request.
	if resp, err := c.do(&request{
		method: "POST",
		path:   "/api/v2/organizations",
		body:   payload,
	}); err != nil {
		return nil, err
	} else {
		defer resp.Body.Close()

		// Decode the response and return the new org.
		var org Organization
		if err := jsonapi.UnmarshalPayload(resp.Body, &org); err != nil {
			return nil, err
		}
		return &org, nil
	}
}

// DeleteOrganizationInput holds parameters used during organization deletion.
type DeleteOrganizationInput struct {
	// The name of the organization to delete.
	Name string
}

// DeleteOrganizationOutput stores results from an org deletion request.
type DeleteOrganizationOutput struct{}

// DeleteOrganization deletes the organization matching the given parameters.
func (c *Client) DeleteOrganization(input *DeleteOrganizationInput) (
	*DeleteOrganizationOutput, error) {

	// Send the request.
	if resp, err := c.do(&request{
		method: "DELETE",
		path:   "/api/v2/organizations/" + input.Name,
	}); err != nil {
		return nil, err
	} else {
		resp.Body.Close()
	}

	return &DeleteOrganizationOutput{}, nil
}
