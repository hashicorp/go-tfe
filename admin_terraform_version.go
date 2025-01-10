// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package tfe

import (
	"context"
	"fmt"
	"net/url"
	"reflect"
	"time"
)

// Compile-time proof of interface implementation.
var _ AdminTerraformVersions = (*adminTerraformVersions)(nil)

const (
	linux = "linux"
	amd64 = "amd64"
	arm64 = "arm64"
)

// AdminTerraformVersions describes all the admin terraform versions related methods that
// the Terraform Enterprise API supports.
// Note that admin terraform versions are only available in Terraform Enterprise.
//
// TFE API docs: https://developer.hashicorp.com/terraform/enterprise/api-docs/admin/terraform-versions
type AdminTerraformVersions interface {
	// List all the terraform versions.
	List(ctx context.Context, options *AdminTerraformVersionsListOptions) (*AdminTerraformVersionsList, error)

	// Read a terraform version by its ID.
	Read(ctx context.Context, id string) (*AdminTerraformVersion, error)

	// Create a terraform version.
	Create(ctx context.Context, options AdminTerraformVersionCreateOptions) (*AdminTerraformVersion, error)

	// Update a terraform version.
	Update(ctx context.Context, id string, options AdminTerraformVersionUpdateOptions) (*AdminTerraformVersion, error)

	// Delete a terraform version
	Delete(ctx context.Context, id string) error
}

// adminTerraformVersions implements AdminTerraformVersions.
type adminTerraformVersions struct {
	client *Client
}

// AdminTerraformVersion represents a Terraform Version
type AdminTerraformVersion struct {
	ID               string    `jsonapi:"primary,terraform-versions"`
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

type ToolVersionArchitectureOptions struct {
	URL  string `json:"url"`
	Sha  string `json:"sha"`
	OS   string `json:"os"`
	Arch string `json:"arch"`
}

// AdminTerraformVersionsListOptions represents the options for listing
// terraform versions.
type AdminTerraformVersionsListOptions struct {
	ListOptions

	// Optional: A query string to find an exact version
	Filter string `url:"filter[version],omitempty"`

	// Optional: A search query string to find all versions that match version substring
	Search string `url:"search[version],omitempty"`
}

// AdminTerraformVersionCreateOptions for creating a terraform version.
// https://developer.hashicorp.com/terraform/enterprise/api-docs/admin/terraform-versions#request-body
type AdminTerraformVersionCreateOptions struct {
	Type             string                            `jsonapi:"primary,terraform-versions"`
	Version          *string                           `jsonapi:"attr,version"` // Required
	URL              *string                           `jsonapi:"attr,url"`     // Required
	Sha              *string                           `jsonapi:"attr,sha"`     // Required
	Official         *bool                             `jsonapi:"attr,official,omitempty"`
	Deprecated       *bool                             `jsonapi:"attr,deprecated,omitempty"`
	DeprecatedReason *string                           `jsonapi:"attr,deprecated-reason,omitempty"`
	Enabled          *bool                             `jsonapi:"attr,enabled,omitempty"`
	Beta             *bool                             `jsonapi:"attr,beta,omitempty"`
	Archs            []*ToolVersionArchitectureOptions `jsonapi:"attr,archs,omitempty"`
}

// AdminTerraformVersionUpdateOptions for updating terraform version.
// https://developer.hashicorp.com/terraform/enterprise/api-docs/admin/terraform-versions#request-body
type AdminTerraformVersionUpdateOptions struct {
	Type             string  `jsonapi:"primary,terraform-versions"`
	Version          *string `jsonapi:"attr,version,omitempty"`
	URL              *string `jsonapi:"attr,url,omitempty"`
	Sha              *string `jsonapi:"attr,sha,omitempty"`
	Official         *bool   `jsonapi:"attr,official,omitempty"`
	Deprecated       *bool   `jsonapi:"attr,deprecated,omitempty"`
	DeprecatedReason *string `jsonapi:"attr,deprecated-reason,omitempty"`
	Enabled          *bool   `jsonapi:"attr,enabled,omitempty"`
	Beta             *bool   `jsonapi:"attr,beta,omitempty"`
}

// AdminTerraformVersionsList represents a list of terraform versions.
type AdminTerraformVersionsList struct {
	*Pagination
	Items []*AdminTerraformVersion
}

// List all the terraform versions.
func (a *adminTerraformVersions) List(ctx context.Context, options *AdminTerraformVersionsListOptions) (*AdminTerraformVersionsList, error) {
	req, err := a.client.NewRequest("GET", "admin/terraform-versions", options)
	if err != nil {
		return nil, err
	}

	tvl := &AdminTerraformVersionsList{}
	err = req.Do(ctx, tvl)
	if err != nil {
		return nil, err
	}

	return tvl, nil
}

// Read a terraform version by its ID.
func (a *adminTerraformVersions) Read(ctx context.Context, id string) (*AdminTerraformVersion, error) {
	if !validStringID(&id) {
		return nil, ErrInvalidTerraformVersionID
	}

	u := fmt.Sprintf("admin/terraform-versions/%s", url.PathEscape(id))
	req, err := a.client.NewRequest("GET", u, nil)
	if err != nil {
		return nil, err
	}

	tfv := &AdminTerraformVersion{}
	err = req.Do(ctx, tfv)
	if err != nil {
		return nil, err
	}

	return tfv, nil
}

// Create a new terraform version.
func (a *adminTerraformVersions) Create(ctx context.Context, options AdminTerraformVersionCreateOptions) (*AdminTerraformVersion, error) {
	if err := options.valid(); err != nil {
		return nil, err
	}
	req, err := a.client.NewRequest("POST", "admin/terraform-versions", &options)
	if err != nil {
		return nil, err
	}

	tfv := &AdminTerraformVersion{}
	err = req.Do(ctx, tfv)
	if err != nil {
		return nil, err
	}
	return tfv, nil
}

// Update an existing terraform version.
func (a *adminTerraformVersions) Update(ctx context.Context, id string, options AdminTerraformVersionUpdateOptions) (*AdminTerraformVersion, error) {
	if !validStringID(&id) {
		return nil, ErrInvalidTerraformVersionID
	}

	u := fmt.Sprintf("admin/terraform-versions/%s", url.PathEscape(id))
	req, err := a.client.NewRequest("PATCH", u, &options)
	if err != nil {
		return nil, err
	}

	tfv := &AdminTerraformVersion{}
	err = req.Do(ctx, tfv)
	if err != nil {
		return nil, err
	}

	return tfv, nil
}

// Delete a terraform version.
func (a *adminTerraformVersions) Delete(ctx context.Context, id string) error {
	if !validStringID(&id) {
		return ErrInvalidTerraformVersionID
	}

	u := fmt.Sprintf("admin/terraform-versions/%s", url.PathEscape(id))
	req, err := a.client.NewRequest("DELETE", u, nil)
	if err != nil {
		return err
	}

	return req.Do(ctx, nil)
}

func (o AdminTerraformVersionCreateOptions) valid() error {
	if (reflect.DeepEqual(o, AdminTerraformVersionCreateOptions{})) {
		return ErrRequiredTFVerCreateOps
	}
	if !validString(o.Version) {
		return ErrRequiredVersion
	}
	if !o.validArch() && (!validString(o.URL) || !validString(o.Sha)) {
		return ErrRequiredArchOrURLAndSha
	}
	return nil
}

func (o AdminTerraformVersionCreateOptions) validArch() bool {
	var valid bool
	for _, a := range o.Archs {
		valid = validString(&a.URL) && validString(&a.Sha) && a.OS == linux && (a.Arch == amd64 || a.Arch == arm64)
		if valid {
			break
		}
	}
	return valid
}
