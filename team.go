package tfe

import (
	"errors"
	"fmt"
)

// Teams handles communication with the team related methods of the Terraform
// Enterprise API.
//
// TFE API docs: https://www.terraform.io/docs/enterprise/api/teams.html
type Teams struct {
	client *Client
}

// Team represents a Terraform Enterprise team.
type Team struct {
	ID          string           `jsonapi:"primary,teams"`
	Name        string           `jsonapi:"attr,name"`
	Permissions *TeamPermissions `jsonapi:"attr,permissions"`
	UserCount   int              `jsonapi:"attr,users-count"`

	// Relations
	//User []*User `jsonapi:"relation,users"`
}

// TeamPermissions represents the team permissions.
type TeamPermissions struct {
	CanDestroy          bool `json:"can-destroy"`
	CanUpdateMembership bool `json:"can-update-membership"`
}

// ListTeamsOptions represents the options for listing teams.
type ListTeamsOptions struct {
	ListOptions
}

// List returns all the organizations visible to the current user.
func (s *Teams) List(organization string, options ListTeamsOptions) ([]*Team, error) {
	if !validStringID(&organization) {
		return nil, errors.New("Invalid value for organization")
	}

	u := fmt.Sprintf("organizations/%s/teams", organization)
	req, err := s.client.newRequest("GET", u, &options)
	if err != nil {
		return nil, err
	}

	result, err := s.client.do(req, []*Team{})
	if err != nil {
		return nil, err
	}

	var ts []*Team
	for _, t := range result.([]interface{}) {
		ts = append(ts, t.(*Team))
	}

	return ts, nil
}

// CreateTeamOptions represents the options for creating a team.
type CreateTeamOptions struct {
	// For internal use only!
	ID string `jsonapi:"primary,teams"`

	// Name of the team.
	Name *string `jsonapi:"attr,name"`
}

func (o CreateTeamOptions) valid() error {
	if !validStringID(o.Name) {
		return errors.New("Invalid value for name")
	}
	return nil
}

// Create a new team with the given name.
func (s *Teams) Create(organization string, options CreateTeamOptions) (*Team, error) {
	if !validStringID(&organization) {
		return nil, errors.New("Invalid value for organization")
	}
	if err := options.valid(); err != nil {
		return nil, err
	}

	// Make sure we don't send a user provided ID.
	options.ID = ""

	u := fmt.Sprintf("organizations/%s/teams", organization)
	req, err := s.client.newRequest("POST", u, &options)
	if err != nil {
		return nil, err
	}

	t, err := s.client.do(req, &Team{})
	if err != nil {
		return nil, err
	}

	return t.(*Team), nil
}

// Retrieve a single team by its ID.
func (s *Teams) Retrieve(teamID string) (*Team, error) {
	if !validStringID(&teamID) {
		return nil, errors.New("Invalid value for team ID")
	}

	req, err := s.client.newRequest("GET", "teams/"+teamID, nil)
	if err != nil {
		return nil, err
	}

	t, err := s.client.do(req, &Team{})
	if err != nil {
		return nil, err
	}

	return t.(*Team), nil
}

// Delete a team by its ID.
func (s *Teams) Delete(teamID string) error {
	if !validStringID(&teamID) {
		return errors.New("Invalid value for team ID")
	}

	req, err := s.client.newRequest("DELETE", "teams/"+teamID, nil)
	if err != nil {
		return err
	}

	_, err = s.client.do(req, nil)

	return err
}
