package tfe

import (
	"context"
	"errors"
	"fmt"
	"net/url"
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

func (o *TeamMemberAddOptions) valid() error {
	if o.Usernames == nil {
		return errors.New("Usernames is required")
	}
	if len(o.Usernames) == 0 {
		return errors.New("Invalid value for usernames")
	}
	return nil
}

// Add multiple users to a team.
func (s *TeamMembers) Add(ctx context.Context, teamID string, options TeamMemberAddOptions) error {
	if !validStringID(&teamID) {
		return errors.New("Invalid value for team ID")
	}
	if err := options.valid(); err != nil {
		return err
	}

	var tms []*teamMember
	for _, name := range options.Usernames {
		tms = append(tms, &teamMember{Username: name})
	}

	u := fmt.Sprintf("teams/%s/relationships/users", url.QueryEscape(teamID))
	req, err := s.client.newRequest("POST", u, tms)
	if err != nil {
		return err
	}

	_, err = s.client.do(ctx, req, nil)

	return err
}

// TeamMemberRemoveOptions represents the options for deleting team members.
type TeamMemberRemoveOptions struct {
	Usernames []string
}

func (o *TeamMemberRemoveOptions) valid() error {
	if o.Usernames == nil {
		return errors.New("Usernames is required")
	}
	if len(o.Usernames) == 0 {
		return errors.New("Invalid value for usernames")
	}
	return nil
}

// Remove multiple users from a team.
func (s *TeamMembers) Remove(ctx context.Context, teamID string, options TeamMemberRemoveOptions) error {
	if !validStringID(&teamID) {
		return errors.New("Invalid value for team ID")
	}
	if err := options.valid(); err != nil {
		return err
	}

	var tms []*teamMember
	for _, name := range options.Usernames {
		tms = append(tms, &teamMember{Username: name})
	}

	u := fmt.Sprintf("teams/%s/relationships/users", url.QueryEscape(teamID))
	req, err := s.client.newRequest("DELETE", u, tms)
	if err != nil {
		return err
	}

	_, err = s.client.do(ctx, req, nil)

	return err
}
