package tfe

import (
	"context"
	"errors"
	"fmt"
	"net/url"
)

// Compile-time proof of interface implementation.
var _ RegistryProviders = (*registryProviders)(nil)

// RegistryProviders describes all the registry provider related methods that the Terraform
// Enterprise API supports.
//
// TFE API docs: https://www.terraform.io/docs/cloud/api/providers.html
type RegistryProviders interface {
	// List all the providers within an organization.
	List(ctx context.Context, organization string, options *RegistryProviderListOptions) (*RegistryProviderList, error)

	// Create a registry provider
	Create(ctx context.Context, organization string, options RegistryProviderCreateOptions) (*RegistryProvider, error)

	// Read a registry provider
	Read(ctx context.Context, organization string, registryName RegistryName, namespace string, name string, options *RegistryProviderReadOptions) (*RegistryProvider, error)

	// Delete a registry provider
	Delete(ctx context.Context, organization string, registryName RegistryName, namespace string, name string) error
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

// RegistryProvider represents a registry provider
type RegistryProvider struct {
	ID           string                       `jsonapi:"primary,registry-providers"`
	Namespace    string                       `jsonapi:"attr,namespace"`
	Name         string                       `jsonapi:"attr,name"`
	RegistryName RegistryName                 `jsonapi:"attr,registry-name"`
	Permissions  *RegistryProviderPermissions `jsonapi:"attr,permissions"`
	CreatedAt    string                       `jsonapi:"attr,created-at"`
	UpdatedAt    string                       `jsonapi:"attr,updated-at"`

	// Relations
	Organization             *Organization             `jsonapi:"relation,organization"`
	RegistryProviderVersions []RegistryProviderVersion `jsonapi:"relation,registry-provider-version"`
}

type RegistryProviderPermissions struct {
	CanDelete bool `jsonapi:"attr,can-delete"`
}

type RegistryProviderListOptions struct {
	ListOptions
	// A query string to filter by registry_name
	RegistryName *RegistryName `url:"filter[registry_name],omitempty"`
	// A query string to filter by organization
	OrganizationName *string `url:"filter[organization_name],omitempty"`
	// A query string to do a fuzzy search
	Search *string `url:"q,omitempty"`
}

type RegistryProviderList struct {
	*Pagination
	Items []*RegistryProvider
}

func (o RegistryProviderListOptions) valid() error {
	return nil
}

func (r *registryProviders) List(ctx context.Context, organization string, options *RegistryProviderListOptions) (*RegistryProviderList, error) {

	if !validStringID(&organization) {
		return nil, ErrInvalidOrg
	}
	if options != nil {
		if err := options.valid(); err != nil {
			return nil, err
		}
	}

	u := fmt.Sprintf("organizations/%s/registry-providers", url.QueryEscape(organization))
	req, err := r.client.newRequest("GET", u, options)
	if err != nil {
		return nil, err
	}

	pl := &RegistryProviderList{}
	err = r.client.do(ctx, req, pl)
	if err != nil {
		return nil, err
	}

	return pl, nil
}

// RegistryProviderCreateOptions is used when creating a registry provider
type RegistryProviderCreateOptions struct {
	// Type is a public field utilized by JSON:API to
	// set the resource type via the field tag.
	// It is not a user-defined value and does not need to be set.
	// https://jsonapi.org/format/#crud-creating
	Type string `jsonapi:"primary,registry-providers"`

	Namespace    *string       `jsonapi:"attr,namespace"`
	Name         *string       `jsonapi:"attr,name"`
	RegistryName *RegistryName `jsonapi:"attr,registry-name"`
}

func (o RegistryProviderCreateOptions) valid() error {
	if !validString(o.Name) {
		return ErrRequiredName
	}
	if !validStringID(o.Name) {
		return ErrInvalidName
	}
	if !validString(o.Namespace) {
		return errors.New("namespace is required")
	}
	if !validStringID(o.Namespace) {
		return errors.New("invalid value for namespace")
	}
	if !validString((*string)(o.RegistryName)) {
		return errors.New("registry-name is required")
	}
	return nil
}

func (r *registryProviders) Create(ctx context.Context, organization string, options RegistryProviderCreateOptions) (*RegistryProvider, error) {
	if !validStringID(&organization) {
		return nil, ErrInvalidOrg
	}
	if err := options.valid(); err != nil {
		return nil, err
	}
	// Private providers must match their namespace and organization name
	// This is enforced by the API as well
	if *options.RegistryName == PrivateRegistry && organization != *options.Namespace {
		return nil, errors.New("namespace must match organization name for private providers")
	}

	u := fmt.Sprintf(
		"organizations/%s/registry-providers",
		url.QueryEscape(organization),
	)
	req, err := r.client.newRequest("POST", u, &options)
	if err != nil {
		return nil, err
	}
	prv := &RegistryProvider{}
	err = r.client.do(ctx, req, prv)
	if err != nil {
		return nil, err
	}

	return prv, nil
}

type RegistryProviderReadOptions struct {
}

func (r *registryProviders) Read(ctx context.Context, organization string, registryName RegistryName, namespace string, name string, options *RegistryProviderReadOptions) (*RegistryProvider, error) {
	if !validStringID(&organization) {
		return nil, ErrInvalidOrg
	}
	if !validString(&name) {
		return nil, ErrRequiredName
	}
	if !validStringID(&name) {
		return nil, ErrInvalidName
	}
	if !validString(&namespace) {
		return nil, errors.New("namespace is required")
	}
	if !validStringID(&namespace) {
		return nil, errors.New("invalid value for namespace")
	}
	if !validString((*string)(&registryName)) {
		return nil, errors.New("registry-name is required")
	}

	u := fmt.Sprintf(
		"organizations/%s/registry-providers/%s/%s/%s",
		url.QueryEscape(organization),
		url.QueryEscape(string(registryName)),
		url.QueryEscape(namespace),
		url.QueryEscape(name),
	)
	req, err := r.client.newRequest("GET", u, options)
	if err != nil {
		return nil, err
	}

	prv := &RegistryProvider{}
	err = r.client.do(ctx, req, prv)
	if err != nil {
		return nil, err
	}

	return prv, nil
}

func (r *registryProviders) Delete(ctx context.Context, organization string, registryName RegistryName, namespace string, name string) error {
	if !validStringID(&organization) {
		return ErrInvalidOrg
	}
	if !validString(&name) {
		return ErrRequiredName
	}
	if !validStringID(&name) {
		return ErrInvalidName
	}
	if !validString(&namespace) {
		return errors.New("namespace is required")
	}
	if !validStringID(&namespace) {
		return errors.New("invalid value for namespace")
	}
	if !validString((*string)(&registryName)) {
		return errors.New("registry-name is required")
	}

	u := fmt.Sprintf(
		"organizations/%s/registry-providers/%s/%s/%s",
		url.QueryEscape(organization),
		url.QueryEscape(string(registryName)),
		url.QueryEscape(namespace),
		url.QueryEscape(name),
	)
	req, err := r.client.newRequest("DELETE", u, nil)
	if err != nil {
		return err
	}

	return r.client.do(ctx, req, nil)
}
