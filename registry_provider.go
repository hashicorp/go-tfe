// Copyright IBM Corp. 2018, 2025
// SPDX-License-Identifier: MPL-2.0

package tfe

import (
	"context"
	"fmt"
	"net/url"
)

// Compile-time proof of interface implementation.
var _ RegistryProviders = (*registryProviders)(nil)

// RegistryProviders describes all the registry provider-related methods that the Terraform
// Enterprise API supports.
//
// TFE API docs: https://developer.hashicorp.com/terraform/cloud-docs/api-docs/private-registry/providers
type RegistryProviders interface {
	// List all the providers within an organization.
	List(ctx context.Context, organization string, options *RegistryProviderListOptions) (*RegistryProviderList, error)

	// Create a registry provider.
	Create(ctx context.Context, organization string, options RegistryProviderCreateOptions) (*RegistryProvider, error)

	// Update a registry provider. Only tag bindings can be updated on a provider, so the update options are limited to that field.
	Update(ctx context.Context, providerID RegistryProviderID, options *RegistryProviderUpdateOptions) (*RegistryProvider, error)

	// Read a registry provider.
	Read(ctx context.Context, providerID RegistryProviderID, options *RegistryProviderReadOptions) (*RegistryProvider, error)

	// Delete a registry provider.
	Delete(ctx context.Context, providerID RegistryProviderID) error

	// ListTagBindings lists all tag bindings associated with the provider.
	ListTagBindings(ctx context.Context, providerID string) ([]*TagBinding, error)
}

// registryProviders implements RegistryProviders.
type registryProviders struct {
	client *Client
}

// RegistryName represents which registry is being targeted
type RegistryName string

// List of available registry names
const (
	PrivateRegistry RegistryName = "private"
	PublicRegistry  RegistryName = "public"
)

// RegistryProviderIncludeOps represents which jsonapi include can be used with registry providers
type RegistryProviderIncludeOps string

// List of available includes
const (
	RegistryProviderVersionsInclude RegistryProviderIncludeOps = "registry-provider-versions"
)

// RegistryProvider represents a registry provider
type RegistryProvider struct {
	ID           string                      `jsonapi:"primary,registry-providers"`
	Name         string                      `jsonapi:"attr,name"`
	Namespace    string                      `jsonapi:"attr,namespace"`
	CreatedAt    string                      `jsonapi:"attr,created-at,iso8601"`
	UpdatedAt    string                      `jsonapi:"attr,updated-at,iso8601"`
	RegistryName RegistryName                `jsonapi:"attr,registry-name"`
	Permissions  RegistryProviderPermissions `jsonapi:"attr,permissions"`

	// Relations
	Organization             *Organization              `jsonapi:"relation,organization"`
	RegistryProviderVersions []*RegistryProviderVersion `jsonapi:"relation,registry-provider-versions"`
	TagBindings              []*TagBinding              `jsonapi:"relation,tag-bindings,omitempty"`

	// Links
	Links map[string]interface{} `jsonapi:"links,omitempty"`
}

type RegistryProviderPermissions struct {
	CanDelete bool `jsonapi:"attr,can-delete"`
}

type RegistryProviderListOptions struct {
	ListOptions

	// Optional: A query string to filter by registry_name
	RegistryName RegistryName `url:"filter[registry_name],omitempty"`

	// Optional: A query string to filter by organization
	OrganizationName string `url:"filter[organization_name],omitempty"`

	// Optional: A query string to do a fuzzy search
	Search string `url:"q,omitempty"`

	// Optional: Include related jsonapi relationships
	Include *[]RegistryProviderIncludeOps `url:"include,omitempty"`
}

type RegistryProviderList struct {
	*Pagination
	Items []*RegistryProvider
}

// RegistryProviderID is the multi key ID for addressing a provider
type RegistryProviderID struct {
	OrganizationName string
	RegistryName     RegistryName
	Namespace        string
	Name             string
}

// RegistryProviderCreateOptions is used when creating a registry provider
type RegistryProviderCreateOptions struct {
	// Type is a public field utilized by JSON:API to
	// set the resource type via the field tag.
	// It is not a user-defined value and does not need to be set.
	// https://jsonapi.org/format/#crud-creating
	Type string `jsonapi:"primary,registry-providers"`

	// Required: The name of the registry provider
	Name string `jsonapi:"attr,name"`

	// Required: The namespace of the provider. For private providers, this is the same as the organization name
	Namespace string `jsonapi:"attr,namespace"`

	// Required: Whether this is a publicly maintained provider or private. Must be either public or private.
	RegistryName RegistryName `jsonapi:"attr,registry-name"`
}

// RegistryProviderUpdateOptions is used when creating a registry provider
type RegistryProviderUpdateOptions struct {
	// Optional: Tag bindings for the registry provider. Note that this
	// will replace all existing tag bindings.
	TagBindings []*TagBinding `jsonapi:"relation,tag-bindings"`
}

type RegistryProviderReadOptions struct {
	// Optional: Include related jsonapi relationships
	Include []RegistryProviderIncludeOps `url:"include,omitempty"`
}

func (r *registryProviders) List(ctx context.Context, organization string, options *RegistryProviderListOptions) (*RegistryProviderList, error) {
	if !validStringID(&organization) {
		return nil, ErrInvalidOrg
	}
	if err := options.valid(); err != nil {
		return nil, err
	}

	u := fmt.Sprintf("organizations/%s/registry-providers", url.PathEscape(organization))
	req, err := r.client.NewRequest("GET", u, options)
	if err != nil {
		return nil, err
	}

	pl := &RegistryProviderList{}
	err = req.Do(ctx, pl)
	if err != nil {
		return nil, err
	}

	return pl, nil
}

func (r *registryProviders) Create(ctx context.Context, organization string, options RegistryProviderCreateOptions) (*RegistryProvider, error) {
	if !validStringID(&organization) {
		return nil, ErrInvalidOrg
	}

	if err := options.valid(); err != nil {
		return nil, err
	}

	u := fmt.Sprintf(
		"organizations/%s/registry-providers",
		url.PathEscape(organization),
	)
	req, err := r.client.NewRequest("POST", u, &options)
	if err != nil {
		return nil, err
	}

	prv := &RegistryProvider{}
	err = req.Do(ctx, prv)
	if err != nil {
		return nil, err
	}

	return prv, nil
}

func (r *registryProviders) Update(ctx context.Context, providerID RegistryProviderID, options *RegistryProviderUpdateOptions) (*RegistryProvider, error) {
	if err := providerID.valid(); err != nil {
		return nil, err
	}

	u := fmt.Sprintf(
		"organizations/%s/registry-providers/%s/%s/%s",
		url.PathEscape(providerID.OrganizationName),
		url.PathEscape(string(providerID.RegistryName)),
		url.PathEscape(providerID.Namespace),
		url.PathEscape(providerID.Name),
	)
	req, err := r.client.NewRequest("PATCH", u, options)
	if err != nil {
		return nil, err
	}
	prv := &RegistryProvider{}
	err = req.Do(ctx, prv)
	if err != nil {
		return nil, err
	}

	return prv, nil
}

func (r *registryProviders) ListTagBindings(ctx context.Context, providerID string) ([]*TagBinding, error) {
	if !validStringID(&providerID) {
		return nil, ErrInvalidProjectID
	}

	u := fmt.Sprintf("registry-providers/%s/tag-bindings", url.PathEscape(providerID))
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

func (r *registryProviders) Read(ctx context.Context, providerID RegistryProviderID, options *RegistryProviderReadOptions) (*RegistryProvider, error) {
	if err := providerID.valid(); err != nil {
		return nil, err
	}

	u := fmt.Sprintf(
		"organizations/%s/registry-providers/%s/%s/%s",
		url.PathEscape(providerID.OrganizationName),
		url.PathEscape(string(providerID.RegistryName)),
		url.PathEscape(providerID.Namespace),
		url.PathEscape(providerID.Name),
	)
	req, err := r.client.NewRequest("GET", u, options)
	if err != nil {
		return nil, err
	}

	prv := &RegistryProvider{}
	err = req.Do(ctx, prv)
	if err != nil {
		return nil, err
	}

	return prv, nil
}

func (r *registryProviders) Delete(ctx context.Context, providerID RegistryProviderID) error {
	if err := providerID.valid(); err != nil {
		return err
	}

	u := fmt.Sprintf(
		"organizations/%s/registry-providers/%s/%s/%s",
		url.PathEscape(providerID.OrganizationName),
		url.PathEscape(string(providerID.RegistryName)),
		url.PathEscape(providerID.Namespace),
		url.PathEscape(providerID.Name),
	)
	req, err := r.client.NewRequest("DELETE", u, nil)
	if err != nil {
		return err
	}

	return req.Do(ctx, nil)
}

func (o RegistryProviderCreateOptions) valid() error {
	if !validStringID(&o.Name) {
		return ErrInvalidName
	}
	if !validStringID(&o.Namespace) {
		return ErrInvalidNamespace
	}
	return nil
}

func (id RegistryProviderID) valid() error {
	if !validStringID(&id.OrganizationName) {
		return ErrInvalidOrg
	}
	if !validStringID(&id.Name) {
		return ErrInvalidName
	}
	if !validStringID(&id.Namespace) {
		return ErrInvalidNamespace
	}
	if !validStringID((*string)(&id.RegistryName)) {
		return ErrInvalidRegistryName
	}
	return nil
}

func (o *RegistryProviderListOptions) valid() error {
	return nil
}
