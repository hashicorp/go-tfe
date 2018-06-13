package tfe

import (
	"errors"
	"fmt"
)

// TeamMembers handles communication with the team member related methods of
// the Terraform Enterprise API.
//
// TFE API docs:
// https://www.terraform.io/docs/enterprise/api/team-members.html
type TeamMembers struct {
	client *Client
}

type teamMember struct {
	Username string `jsonapi:"primary,users"`
}

// TeamMemberAddOptions represents the options for adding team members.
type TeamMemberAddOptions struct {
	Usernames []string
}

// Add multiple users to a team.
func (s *TeamMembers) Add(teamID string, options TeamMemberAddOptions) error {
	if !validStringID(&teamID) {
		return errors.New("Invalid value for team ID")
	}

	var tms []*teamMember
	for _, name := range options.Usernames {
		tms = append(tms, &teamMember{Username: name})
	}

	u := fmt.Sprintf("teams/%s/relationships/users", teamID)
	req, err := s.client.newRequest("POST", u, &options)
	if err != nil {
		return err
	}

	_, err = s.client.do(req, nil)

	return err
}

// TeamMemberRemoveOptions represents the options for deleting team members.
type TeamMemberRemoveOptions struct {
	Usernames []string
}

// Remove multiple users from a team.
func (s *TeamMembers) Remove(teamID string, options TeamMemberRemoveOptions) error {
	if !validStringID(&teamID) {
		return errors.New("Invalid value for team ID")
	}

	var tms []*teamMember
	for _, name := range options.Usernames {
		tms = append(tms, &teamMember{Username: name})
	}

	u := fmt.Sprintf("teams/%s/relationships/users", teamID)
	req, err := s.client.newRequest("DELETE", u, &options)
	if err != nil {
		return err
	}

	_, err = s.client.do(req, nil)

	return err
}
