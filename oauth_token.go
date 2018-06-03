package tfe

import (
	"errors"
	"fmt"
	"time"
)

// OAuthTokens handles communication with the oAuth token related methods
// of the Terraform Enterprise API.
//
// TFE API docs:
// https://www.terraform.io/docs/enterprise/api/oauth-tokens.html
type OAuthTokens struct {
	client *Client
}

// OAuthToken represents a VCS configuration including the assocaited
// OAuth token
type OAuthToken struct {
	ID                  string    `jsonapi:"primary,oauth-tokens"`
	UID                 string    `jsonapi:"attr,uid"`
	CreatedAt           time.Time `jsonapi:"attr,created-at,iso8601"`
	HasSSHKey           bool      `jsonapi:"attr,has-ssh-key"`
	ServiceProviderUser string    `jsonapi:"attr,service-provider-user"`

	// Relations
	OAuthClient *OAuthClient `jsonapi:"relation,oauth-client"`
}

// List all the OAuth Tokens for a given organization.
func (s *OAuthTokens) List(organization string) ([]*OAuthToken, error) {
	if !validStringID(&organization) {
		return nil, errors.New("Invalid value for organization")
	}

	u := fmt.Sprintf("organizations/%s/oauth-tokens", organization)
	req, err := s.client.newRequest("GET", u, nil)
	if err != nil {
		return nil, err
	}

	result, err := s.client.do(req, []*OAuthToken{})
	if err != nil {
		return nil, err
	}

	var os []*OAuthToken
	for _, o := range result.([]interface{}) {
		os = append(os, o.(*OAuthToken))
	}

	return os, nil
}
