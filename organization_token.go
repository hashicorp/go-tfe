package tfe

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"time"
)

// OrganizationTokens handles communication with the organization token related
// methods of the Terraform Enterprise API.
//
// TFE API docs:
// https://www.terraform.io/docs/enterprise/api/organization-tokens.html
type OrganizationTokens struct {
	client *Client
}

// OrganizationToken represents a Terraform Enterprise organization token.
type OrganizationToken struct {
	ID          string    `jsonapi:"primary,authentication-tokens"`
	CreatedAt   time.Time `jsonapi:"attr,created-at,iso8601"`
	Description string    `jsonapi:"attr,description"`
	LastUsedAt  time.Time `jsonapi:"attr,last-used-at,iso8601"`
	Token       string    `jsonapi:"attr,token"`
}

// Generate a new organization token, replacing any existing token.
func (s *OrganizationTokens) Generate(ctx context.Context, organization string) (*OrganizationToken, error) {
	if !validStringID(&organization) {
		return nil, errors.New("Invalid value for organization")
	}

	u := fmt.Sprintf("organizations/%s/authentication-token", url.QueryEscape(organization))
	req, err := s.client.newRequest("POST", u, nil)
	if err != nil {
		return nil, err
	}

	t, err := s.client.do(ctx, req, &OrganizationToken{})
	if err != nil {
		return nil, err
	}

	return t.(*OrganizationToken), err
}

// Delete an organization token.
func (s *OrganizationTokens) Delete(ctx context.Context, organization string) error {
	if !validStringID(&organization) {
		return errors.New("Invalid value for organization")
	}

	u := fmt.Sprintf("organizations/%s/authentication-token", url.QueryEscape(organization))
	req, err := s.client.newRequest("DELETE", u, nil)
	if err != nil {
		return err
	}

	_, err = s.client.do(ctx, req, nil)

	return err
}
