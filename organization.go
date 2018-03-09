package tfe

import (
	"reflect"

	"github.com/google/jsonapi"
)

// The reflect type of an organization. Used during deserialization.
var organizationType = reflect.TypeOf(&Organization{})

// Organization encapsulates all data fields of a TFE Organization.
type Organization struct {
	Name                   string `jsonapi:"primary,organizations"`
	CreatedAt              string `jsonapi:"attr,created-at"`
	Email                  string `jsonapi:"attr,email"`
	CollaboratorAuthPolicy string `jsonapi:"attr,collaborator-auth-policy"`
	EnterprisePlan         string `jsonapi:"attr,enterprise-plan"`
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
