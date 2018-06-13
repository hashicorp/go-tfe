package tfe

import (
	"errors"
)

// TeamAccesses handles communication with the team access related methods of
// the Terraform Enterprise API.
//
// TFE API docs:
// https://www.terraform.io/docs/enterprise/api/team-access.html
type TeamAccesses struct {
	client *Client
}

// TeamAccessType represents a team access type.
type TeamAccessType string

// List all available team access types.
const (
	TeamAccessAdmin TeamAccessType = "admin"
	TeamAccessRead  TeamAccessType = "read"
	TeamAccessWrite TeamAccessType = "write"
)

// TeamAccess represents the workspace access for a team.
type TeamAccess struct {
	ID     string         `jsonapi:"primary,team-workspaces"`
	Access TeamAccessType `jsonapi:"attr,access"`

	// Relations
	Team      *Team      `jsonapi:"relation,team"`
	Workspace *Workspace `jsonapi:"relation,workspace"`
}

// TeamAccessListOptions represents the options for listing team accesses.
type TeamAccessListOptions struct {
	ListOptions
	WorkspaceID *string `url:"filter[workspace][id],omitempty"`
}

func (o TeamAccessListOptions) valid() error {
	if !validString(o.WorkspaceID) {
		return errors.New("Workspace ID is required")
	}
	if !validStringID(o.WorkspaceID) {
		return errors.New("Invalid value for workspace ID")
	}
	return nil
}

// List returns the team accesses for a given workspace.
func (s *TeamAccesses) List(options TeamAccessListOptions) ([]*TeamAccess, error) {
	if err := options.valid(); err != nil {
		return nil, err
	}

	req, err := s.client.newRequest("GET", "team-workspaces", &options)
	if err != nil {
		return nil, err
	}

	result, err := s.client.do(req, []*TeamAccess{})
	if err != nil {
		return nil, err
	}

	var tas []*TeamAccess
	for _, ta := range result.([]interface{}) {
		tas = append(tas, ta.(*TeamAccess))
	}

	return tas, nil
}

// TeamAccessAddOptions represents the options for adding team access.
type TeamAccessAddOptions struct {
	// For internal use only!
	ID string `jsonapi:"primary,team-workspaces"`

	// The type of access to grant.
	Access *TeamAccessType `jsonapi:"attr,access"`

	// The team to add to the workspace
	Team *Team `jsonapi:"relation,team"`

	// The workspace to which the team is to be added.
	Workspace *Workspace `jsonapi:"relation,workspace"`
}

func (o TeamAccessAddOptions) valid() error {
	if o.Access == nil {
		return errors.New("Access is required")
	}
	if o.Team == nil {
		return errors.New("Team is required")
	}
	if o.Workspace == nil {
		return errors.New("Workspace is required")
	}
	return nil
}

// Add team access for a workspace.
func (s *TeamAccesses) Add(options TeamAccessAddOptions) (*TeamAccess, error) {
	if err := options.valid(); err != nil {
		return nil, err
	}

	// Make sure we don't send a user provided ID.
	options.ID = ""

	req, err := s.client.newRequest("POST", "team-workspaces", &options)
	if err != nil {
		return nil, err
	}

	ta, err := s.client.do(req, &TeamAccess{})
	if err != nil {
		return nil, err
	}

	return ta.(*TeamAccess), nil
}

// Retrieve a sible team access by its ID.
func (s *TeamAccesses) Retrieve(teamAccessID string) (*TeamAccess, error) {
	if !validStringID(&teamAccessID) {
		return nil, errors.New("Invalid value for team access ID")
	}

	req, err := s.client.newRequest("GET", "team-workspaces/"+teamAccessID, nil)
	if err != nil {
		return nil, err
	}

	ta, err := s.client.do(req, &TeamAccess{})
	if err != nil {
		return nil, err
	}

	return ta.(*TeamAccess), nil
}

// Remove team access from a workspace.
func (s *TeamAccesses) Remove(teamAccessID string) error {
	if !validStringID(&teamAccessID) {
		return errors.New("Invalid value for team access ID")
	}

	req, err := s.client.newRequest("DELETE", "team-workspaces/"+teamAccessID, nil)
	if err != nil {
		return err
	}

	_, err = s.client.do(req, nil)

	return err
}
