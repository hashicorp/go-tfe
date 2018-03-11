package tfe

import (
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
