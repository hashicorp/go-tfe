package tfe

import (
	"context"
	"fmt"
	"net/url"
)

// Compile-time proof of interface implementation.
var _ TeamProjectAccesses = (*teamProjectAccesses)(nil)

// TeamProjectAccesses describes all the team project access related methods that the Terraform
// Enterprise API supports
//
// TFE API docs: (TODO: ADD DOCS URL)
// **Note: This functionality is still in BETA and subject to change.**
type TeamProjectAccesses interface {
	// List all project accesses for a given project.
	List(ctx context.Context, options *TeamProjectAccessListOptions) (*TeamProjectAccessList, error)

	// Add team access for a project.
	Add(ctx context.Context, options TeamProjectAccessAddOptions) (*TeamProjectAccess, error)

	// Read team access by project ID.
	Read(ctx context.Context, teamProjectAccessID string) (*TeamProjectAccess, error)

	// Update team access on a project.
	Update(ctx context.Context, teamProjectAccessID string, options TeamProjectAccessUpdateOptions) (*TeamProjectAccess, error)

	// Remove team access from a project.
	Remove(ctx context.Context, teamProjectAccessID string) error
}

// teamProjectAccesses implements TeamProjectAccesses
type teamProjectAccesses struct {
	client *Client
}

// TeamProjectAccessType represents a team project access type.
type TeamProjectAccessType string

const (
	TeamProjectAccessAdmin TeamProjectAccessType = "admin"
	TeamProjectAccessRead  TeamProjectAccessType = "read"
)

// TeamProjectAccessList represents a list of team project accesses
type TeamProjectAccessList struct {
	*Pagination
	Items []*TeamProjectAccess
}

// TeamProjectAccess represents a project access for a team
type TeamProjectAccess struct {
	ID     string                `jsonapi:"primary,team-projects"`
	Access TeamProjectAccessType `jsonapi:"attr,access"`

	// Relations
	Team    *Team    `jsonapi:"relation,team"`
	Project *Project `jsonapi:"relation,project"`
}

// TeamProjectAccessListOptions represents the options for listing team project accesses
type TeamProjectAccessListOptions struct {
	ListOptions
	ProjectID string `url:"filter[project][id]"`
}

// TeamProjectAccessAddOptions represents the options for adding team access for a project
type TeamProjectAccessAddOptions struct {
	// Type is a public field utilized by JSON:API to
	// set the resource type via the field tag.
	// It is not a user-defined value and does not need to be set.
	// https://jsonapi.org/format/#crud-creating
	Type string `jsonapi:"primary,team-projects"`
	// The type of access to grant.
	Access *TeamProjectAccessType `jsonapi:"attr,access"`

	// The team to add to the project
	Team *Team `jsonapi:"relation,team"`
	// The project to which the team is to be added.
	Project *Project `jsonapi:"relation,project"`
}

// TeamProjectAccessUpdateOptions represents the options for updating a team project access
type TeamProjectAccessUpdateOptions struct {
	// Type is a public field utilized by JSON:API to
	// set the resource type via the field tag.
	// It is not a user-defined value and does not need to be set.
	// https://jsonapi.org/format/#crud-creating
	Type string `jsonapi:"primary,team-projects"`
	// The type of access to grant.
	Access *TeamProjectAccessType `jsonapi:"attr,access,omitempty"`
}

// List all team accesses for a given project.
func (s *teamProjectAccesses) List(ctx context.Context, options *TeamProjectAccessListOptions) (*TeamProjectAccessList, error) {
	if err := options.valid(); err != nil {
		return nil, err
	}

	req, err := s.client.NewRequest("GET", "team-projects", options)
	if err != nil {
		return nil, err
	}

	tpal := &TeamProjectAccessList{}
	err = req.Do(ctx, tpal)
	if err != nil {
		return nil, err
	}

	return tpal, nil
}

// Add team access for a project.
func (s *teamProjectAccesses) Add(ctx context.Context, options TeamProjectAccessAddOptions) (*TeamProjectAccess, error) {
	if err := options.valid(); err != nil {
		return nil, err
	}

	req, err := s.client.NewRequest("POST", "team-projects", &options)
	if err != nil {
		return nil, err
	}

	tpa := &TeamProjectAccess{}
	err = req.Do(ctx, tpa)
	if err != nil {
		return nil, err
	}

	return tpa, nil
}

// Read a team project access by its ID.
func (s *teamProjectAccesses) Read(ctx context.Context, teamProjectAccessID string) (*TeamProjectAccess, error) {
	if !validStringID(&teamProjectAccessID) {
		return nil, ErrInvalidTeamProjectAccessID
	}

	u := fmt.Sprintf("team-projects/%s", url.QueryEscape(teamProjectAccessID))
	req, err := s.client.NewRequest("GET", u, nil)
	if err != nil {
		return nil, err
	}

	tpa := &TeamProjectAccess{}
	err = req.Do(ctx, tpa)
	if err != nil {
		return nil, err
	}

	return tpa, nil
}

// Update team access for a project.
func (s *teamProjectAccesses) Update(ctx context.Context, teamProjectAccessID string, options TeamProjectAccessUpdateOptions) (*TeamProjectAccess, error) {
	if !validStringID(&teamProjectAccessID) {
		return nil, ErrInvalidTeamProjectAccessID
	}

	u := fmt.Sprintf("team-projects/%s", url.QueryEscape(teamProjectAccessID))
	req, err := s.client.NewRequest("PATCH", u, &options)
	if err != nil {
		return nil, err
	}

	ta := &TeamProjectAccess{}
	err = req.Do(ctx, ta)
	if err != nil {
		return nil, err
	}

	return ta, err
}

// Remove team access from a project.
func (s *teamProjectAccesses) Remove(ctx context.Context, teamProjectAccessID string) error {
	if !validStringID(&teamProjectAccessID) {
		return ErrInvalidTeamProjectAccessID
	}

	u := fmt.Sprintf("team-projects/%s", url.QueryEscape(teamProjectAccessID))
	req, err := s.client.NewRequest("DELETE", u, nil)
	if err != nil {
		return err
	}

	return req.Do(ctx, nil)
}

func (o *TeamProjectAccessListOptions) valid() error {
	if o == nil {
		return ErrRequiredTeamProjectAccessListOps
	}
	if !validString(&o.ProjectID) {
		return ErrRequiredProjectID
	}
	if !validStringID(&o.ProjectID) {
		return ErrInvalidProjectID
	}

	return nil
}

func (o TeamProjectAccessAddOptions) valid() error {
	if o.Access == nil {
		return ErrRequiredAccess
	}
	if o.Team == nil {
		return ErrRequiredTeam
	}
	if o.Project == nil {
		return ErrRequiredProject
	}
	return nil
}
