package tfe

import (
	"errors"
	"fmt"
	"time"
)

// TeamTokens handles communication with the team token related methods of the
// Terraform Enterprise API.
//
// TFE API docs:
// https://www.terraform.io/docs/enterprise/api/team-tokens.html
type TeamTokens struct {
	client *Client
}

// TeamToken represents a Terraform Enterprise team token.
type TeamToken struct {
	ID          string    `jsonapi:"primary,authentication-tokens"`
	CreatedAt   time.Time `jsonapi:"attr,created-at,iso8601"`
	Description string    `jsonapi:"attr,description"`
	LastUsedAt  time.Time `jsonapi:"attr,last-used-at,iso8601"`
	Token       string    `jsonapi:"attr,token"`
}

// Generate a new team token, replacing any existing token.
func (s *TeamTokens) Generate(teamID string) (*TeamToken, error) {
	if !validStringID(&teamID) {
		return nil, errors.New("Invalid value for team ID")
	}

	u := fmt.Sprintf("teams/%s/authentication-token", teamID)
	req, err := s.client.newRequest("POST", u, nil)
	if err != nil {
		return nil, err
	}

	t, err := s.client.do(req, &TeamToken{})
	if err != nil {
		return nil, err
	}

	return t.(*TeamToken), err
}

// Delete a team token.
func (s *TeamTokens) Delete(teamID string) error {
	if !validStringID(&teamID) {
		return errors.New("Invalid value for team ID")
	}

	u := fmt.Sprintf("teams/%s/authentication-token", teamID)
	req, err := s.client.newRequest("DELETE", u, nil)
	if err != nil {
		return err
	}

	_, err = s.client.do(req, nil)

	return err
}
