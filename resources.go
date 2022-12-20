package tfe

import (
	"context"
	"fmt"
)

// Resources describes all the resource related methods that the Terraform
// Enterprise API supports.
type Resources interface {
	// List all the resources within a workspace
	List(ctx context.Context, workspaceID string, options *ResourceListOptions) (*ResourceList, error)
}

// resources implements Resources
type resources struct {
	client *Client
}

// ResourceList represents a list of resources
type ResourceList struct {
	*Pagination
	Items []*Resource
}

// Resource represents a Terraform Enterprise resource
type Resource struct {
	ID                       string `jsonapi:"primary,resources"`
	Address                  string `jsonapi:"attr,address"`
	Name                     string `jsonapi:"attr,name"`
	CreatedAt                string `jsonapi:"attr,created-at,iso8601"`
	UpdatedAt                string `jsonapi:"attr,updated-at,iso8601"`
	Module                   string `jsonapi:"attr,module"`
	Provider                 string `jsonapi:"attr,provider"`
	ProviderType             string `jsonapi:"attr,provider-type"`
	ModifiedByStateVersionID string `jsonapi:"attr,modified-by-state-version-id"`
	NameIndex                string `jsonapi:"attr,name-index"`
}

// ResourceListOptions represents the options for listing resources.
type ResourceListOptions struct {
	ListOptions
}

// List all the resources within a workspace
func (s *resources) List(ctx context.Context, workspaceID string, options *ResourceListOptions) (*ResourceList, error) {
	if !validStringID(&workspaceID) {
		return nil, ErrInvalidWorkspaceID
	}
	u := fmt.Sprintf("workspaces/%s/resources", workspaceID)
	req, err := s.client.NewRequest("GET", u, &options)
	if err != nil {
		return nil, err
	}
	rl := &ResourceList{}
	err = req.Do(ctx, rl)
	if err != nil {
		return nil, err
	}
	return rl, nil
}
