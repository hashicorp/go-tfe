package tfe

import (
	"context"
	"errors"
	"fmt"
	"net/url"
)

// Compile-time proof of interface implementation.
var _ AdminWorkspaces = (*adminWorkspaces)(nil)

// AdminWorkspaces Admin API contains endpoints to help site administrators manage
// workspaces.
//
// TFE API docs: https://www.terraform.io/docs/cloud/api/admin/workspaces.html
type AdminWorkspaces interface {
	// List all the workspaces of the given installation.
	List(ctx context.Context, options AdminWorkspacesListOptions) (*AdminWorkspacesList, error)

	// Read a workspace by its id.
	Read(ctx context.Context, workspaceID string) (*Workspace, error)

	// Delete a workspace by id.
	Delete(ctx context.Context, workspaceID string) error
}

// workspaces implements Users.
type adminWorkspaces struct {
	client *Client
}

// AdminWorkspacesList represents a list of workspaces.
type AdminWorkspacesList struct {
	*Pagination
	Items []*Workspace
}

// AdminWorkspacesListOptions represents the options for listing workspaces.
type AdminWorkspacesListOptions struct {
	ListOptions
	RunStatus *string `url:"filter[current_run][status],omitempty"`
	Query     *string `url:"q"`
}

// List all the workspaces of the terraform enterprise installation.
func (s *adminWorkspaces) List(ctx context.Context, options AdminWorkspacesListOptions) (*AdminWorkspacesList, error) {
	u := fmt.Sprintf("admin/workspaces")
	req, err := s.client.newRequest("GET", u, &options)
	if err != nil {
		return nil, err
	}

	rl := &AdminWorkspacesList{}
	err = s.client.do(ctx, req, rl)
	if err != nil {
		return nil, err
	}

	return rl, nil
}

// Read reads a workspace by its ID.
func (s *adminWorkspaces) Read(ctx context.Context, workspaceID string) (*Workspace, error) {
	if !validStringID(&workspaceID) {
		return nil, errors.New("invalid value for workspace ID")
	}

	u := fmt.Sprintf("admin/workspaces/%s", url.QueryEscape(workspaceID))
	req, err := s.client.newRequest("GET", u, nil)
	if err != nil {
		return nil, err
	}

	w := &Workspace{}
	err = s.client.do(ctx, req, w)
	if err != nil {
		return nil, err
	}

	return w, nil
}

// Delete deletes a workspace by its ID.
func (s *adminWorkspaces) Delete(ctx context.Context, workspaceID string) error {
	if !validStringID(&workspaceID) {
		return errors.New("invalid value for workspace ID")
	}

	u := fmt.Sprintf("admin/workspaces/%s", url.QueryEscape(workspaceID))
	req, err := s.client.newRequest("DELETE", u, nil)
	if err != nil {
		return err
	}

	return s.client.do(ctx, req, nil)
}
