package tfe

import (
	"context"
	"fmt"
	"net/url"
	"reflect"
)

// Compile-time proof of interface implementation.
var _ RegistryComponents = (*registryComponents)(nil)

// RegistryComponents describes all the registry component-related methods that the Terraform
// Enterprise API supports.
//
// TFE API docs: https://developer.hashicorp.com/terraform/cloud-docs/api-docs/private-registry/stack-component-configurations
type RegistryComponents interface {
	// Create a registry component. Note that this function creates registry components via API-only workflow.
	Create(ctx context.Context, organization string, options RegistryComponentCreateOptions) (*RegistryComponent, error)

	// Update a registry component. Only tag bindings can be updated on a component, so the update options are limited to that field.
	Update(ctx context.Context, componentID string, options *RegistryComponentUpdateOptions) (*RegistryComponent, error)

	// ListTagBindings lists all tag bindings associated with the component.
	ListTagBindings(ctx context.Context, componentID string) ([]*TagBinding, error)

	// Delete a registry component.
	Delete(ctx context.Context, componentID string) error
}

// registryComponents implements RegistryComponents.
type registryComponents struct {
	client *Client
}

type RegistryComponentVersionStatuses struct {
	Version string `jsonapi:"attr,version"`
	Status  string `jsonapi:"attr,status"`
}

// RegistryComponent represents a registry component
type RegistryComponent struct {
	ID              string                             `jsonapi:"primary,registry-components"`
	Name            string                             `jsonapi:"attr,name"`
	Namespace       string                             `jsonapi:"attr,namespace"`
	Description     string                             `jsonapi:"attr,description"`
	Status          string                             `jsonapi:"attr,status"`
	VCSRepo         *VCSRepo                           `jsonapi:"attr,vcs-repo"`
	VersionStatuses []RegistryComponentVersionStatuses `jsonapi:"attr,version-statuses"`
	CreatedAt       string                             `jsonapi:"attr,created-at,iso8601"`
	UpdatedAt       string                             `jsonapi:"attr,updated-at,iso8601"`

	// Relations
	Organization *Organization `jsonapi:"relation,organization"`
	TagBindings  []*TagBinding `jsonapi:"relation,tag-bindings,omitempty"`

	// Links
	Links map[string]interface{} `jsonapi:"links,omitempty"`
}

// RegistryComponentUpdateOptions is used when updating a registry component config
type RegistryComponentUpdateOptions struct {
	// Optional: Tag bindings for the registry component. Note that this
	// will replace all existing tag bindings.
	TagBindings []*TagBinding `jsonapi:"relation,tag-bindings"`
}

// RegistryComponentCreateOptions is used when creating a registry component config via API-only workflow
type RegistryComponentCreateOptions struct {
	Type string `jsonapi:"primary,registry-components"`
	Name string `jsonapi:"attr,name"`
}

func (r *registryComponents) Create(ctx context.Context, organization string, options RegistryComponentCreateOptions) (*RegistryComponent, error) {
	if !validStringID(&organization) {
		return nil, ErrInvalidOrg
	}
	if (reflect.DeepEqual(options, RegistryComponentCreateOptions{})) {
		return nil, ErrRequiredRegistryComponentCreateOps
	}

	if !validStringID(&options.Name) {
		return nil, ErrInvalidName
	}

	u := fmt.Sprintf(
		"organizations/%s/registry-components",
		url.PathEscape(organization),
	)
	req, err := r.client.NewRequest("POST", u, &options)
	if err != nil {
		return nil, err
	}

	rc := &RegistryComponent{}
	err = req.Do(ctx, rc)
	if err != nil {
		return nil, err
	}

	return rc, nil
}

func (r *registryComponents) Update(ctx context.Context, componentID string, options *RegistryComponentUpdateOptions) (*RegistryComponent, error) {
	if !validStringID(&componentID) {
		return nil, ErrInvalidRegistryComponentID
	}

	u := fmt.Sprintf(
		"registry-components/%s",
		url.PathEscape(componentID),
	)
	req, err := r.client.NewRequest("PATCH", u, options)
	if err != nil {
		return nil, err
	}
	rc := &RegistryComponent{}
	err = req.Do(ctx, rc)
	if err != nil {
		return nil, err
	}

	return rc, nil
}

func (r *registryComponents) ListTagBindings(ctx context.Context, componentID string) ([]*TagBinding, error) {
	if !validStringID(&componentID) {
		return nil, ErrInvalidProjectID
	}

	u := fmt.Sprintf("registry-components/%s/tag-bindings", url.PathEscape(componentID))
	req, err := r.client.NewRequest("GET", u, nil)
	if err != nil {
		return nil, err
	}

	var list struct {
		*Pagination
		Items []*TagBinding
	}

	err = req.Do(ctx, &list)
	if err != nil {
		return nil, err
	}

	return list.Items, nil
}

// Delete a specified registry component.
func (r *registryComponents) Delete(ctx context.Context, componentID string) error {
	if !validStringID(&componentID) {
		return ErrInvalidRegistryComponentID
	}

	u := fmt.Sprintf("registry-components/%s", url.PathEscape(componentID))

	req, err := r.client.NewRequest("DELETE", u, nil)

	if err != nil {
		return err
	}

	return req.Do(ctx, nil)
}
