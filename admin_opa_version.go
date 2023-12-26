package tfe

import (
	"context"
	"fmt"
	"net/url"
	"time"
)

// Compile-time proof of interface implementation.
var _ AdminOpaVersions = (*adminOpaVersions)(nil)

// AdminOpaVersions describes all the admin OPA versions related methods that
// the Terraform Enterprise API supports.
// Note that admin OPA versions are only available in Terraform Enterprise.
//
// TFE API docs: https://developer.hashicorp.com/terraform/enterprise/api-docs/admin/opa-versions
type AdminOpaVersions interface {
	// List all the OPA versions.
	List(ctx context.Context, options *AdminOpaVersionsListOptions) (*AdminOpaVersionsList, error)

	// Read a OPA version by its ID.
	Read(ctx context.Context, id string) (*AdminOpaVersion, error)

	// Create a OPA version.
	Create(ctx context.Context, options AdminOpaVersionCreateOptions) (*AdminOpaVersion, error)

	// Update a OPA version.
	Update(ctx context.Context, id string, options AdminOpaVersionUpdateOptions) (*AdminOpaVersion, error)

	// Delete a OPA version
	Delete(ctx context.Context, id string) error
}

// adminOpaVersions implements AdminOpaVersions.
type adminOpaVersions struct {
	client *Client
}

// AdminOpaVersion represents a OPA Version
type AdminOpaVersion struct {
	ID               string    `jsonapi:"primary,opa-versions"`
	Version          string    `jsonapi:"attr,version"`
	URL              string    `jsonapi:"attr,url"`
	Sha              string    `jsonapi:"attr,sha"`
	Deprecated       bool      `jsonapi:"attr,deprecated"`
	DeprecatedReason *string   `jsonapi:"attr,deprecated-reason,omitempty"`
	Official         bool      `jsonapi:"attr,official"`
	Enabled          bool      `jsonapi:"attr,enabled"`
	Beta             bool      `jsonapi:"attr,beta"`
	Usage            int       `jsonapi:"attr,usage"`
	CreatedAt        time.Time `jsonapi:"attr,created-at,iso8601"`
}

// AdminOpaVersionsListOptions represents the options for listing
// OPA versions.
type AdminOpaVersionsListOptions struct {
	ListOptions

	// Optional: A query string to find an exact version
	Filter string `url:"filter[version],omitempty"`

	// Optional: A search query string to find all versions that match version substring
	Search string `url:"search[version],omitempty"`
}

// AdminOpaVersionCreateOptions for creating an OPA version.
type AdminOpaVersionCreateOptions struct {
	Type             string  `jsonapi:"primary,opa-versions"`
	Version          string  `jsonapi:"attr,version"` // Required
	URL              string  `jsonapi:"attr,url"`     // Required
	Sha              string  `jsonapi:"attr,sha"`     // Required
	Official         *bool   `jsonapi:"attr,official,omitempty"`
	Deprecated       *bool   `jsonapi:"attr,deprecated,omitempty"`
	DeprecatedReason *string `jsonapi:"attr,deprecated-reason,omitempty"`
	Enabled          *bool   `jsonapi:"attr,enabled,omitempty"`
	Beta             *bool   `jsonapi:"attr,beta,omitempty"`
}

// AdminOpaVersionUpdateOptions for updating OPA version.
type AdminOpaVersionUpdateOptions struct {
	Type             string  `jsonapi:"primary,opa-versions"`
	Version          *string `jsonapi:"attr,version,omitempty"`
	URL              *string `jsonapi:"attr,url,omitempty"`
	Sha              *string `jsonapi:"attr,sha,omitempty"`
	Official         *bool   `jsonapi:"attr,official,omitempty"`
	Deprecated       *bool   `jsonapi:"attr,deprecated,omitempty"`
	DeprecatedReason *string `jsonapi:"attr,deprecated-reason,omitempty"`
	Enabled          *bool   `jsonapi:"attr,enabled,omitempty"`
	Beta             *bool   `jsonapi:"attr,beta,omitempty"`
}

// AdminOpaVersionsList represents a list of OPA versions.
type AdminOpaVersionsList struct {
	*Pagination
	Items []*AdminOpaVersion
}

// List all the OPA versions.
func (a *adminOpaVersions) List(ctx context.Context, options *AdminOpaVersionsListOptions) (*AdminOpaVersionsList, error) {
	req, err := a.client.NewRequest("GET", "admin/opa-versions", options)
	if err != nil {
		return nil, err
	}

	ol := &AdminOpaVersionsList{}
	err = req.Do(ctx, ol)
	if err != nil {
		return nil, err
	}

	return ol, nil
}

// Read a OPA version by its ID.
func (a *adminOpaVersions) Read(ctx context.Context, id string) (*AdminOpaVersion, error) {
	if !validStringID(&id) {
		return nil, ErrInvalidOpaVersionID
	}

	u := fmt.Sprintf("admin/opa-versions/%s", url.QueryEscape(id))
	req, err := a.client.NewRequest("GET", u, nil)
	if err != nil {
		return nil, err
	}

	ov := &AdminOpaVersion{}
	err = req.Do(ctx, ov)
	if err != nil {
		return nil, err
	}

	return ov, nil
}

// Create a new OPA version.
func (a *adminOpaVersions) Create(ctx context.Context, options AdminOpaVersionCreateOptions) (*AdminOpaVersion, error) {
	if err := options.valid(); err != nil {
		return nil, err
	}
	req, err := a.client.NewRequest("POST", "admin/opa-versions", &options)
	if err != nil {
		return nil, err
	}

	ov := &AdminOpaVersion{}
	err = req.Do(ctx, ov)
	if err != nil {
		return nil, err
	}

	return ov, nil
}

// Update an existing OPA version.
func (a *adminOpaVersions) Update(ctx context.Context, id string, options AdminOpaVersionUpdateOptions) (*AdminOpaVersion, error) {
	if !validStringID(&id) {
		return nil, ErrInvalidOpaVersionID
	}

	u := fmt.Sprintf("admin/opa-versions/%s", url.QueryEscape(id))
	req, err := a.client.NewRequest("PATCH", u, &options)
	if err != nil {
		return nil, err
	}

	ov := &AdminOpaVersion{}
	err = req.Do(ctx, ov)
	if err != nil {
		return nil, err
	}

	return ov, nil
}

// Delete a OPA version.
func (a *adminOpaVersions) Delete(ctx context.Context, id string) error {
	if !validStringID(&id) {
		return ErrInvalidOpaVersionID
	}

	u := fmt.Sprintf("admin/opa-versions/%s", url.QueryEscape(id))
	req, err := a.client.NewRequest("DELETE", u, nil)
	if err != nil {
		return err
	}

	return req.Do(ctx, nil)
}

func (o AdminOpaVersionCreateOptions) valid() error {
	if (o == AdminOpaVersionCreateOptions{}) {
		return ErrRequiredOpaVerCreateOps
	}
	if o.Version == "" {
		return ErrRequiredVersion
	}
	if o.URL == "" {
		return ErrRequiredURL
	}
	if o.Sha == "" {
		return ErrRequiredSha
	}

	return nil
}
