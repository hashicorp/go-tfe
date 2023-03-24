package tfe

import (
	"context"
	"fmt"
	"net/url"
)

// Compile-time proof of interface implementation.
var _ NoCodeRegistryModules = (*noCodeRegistryModules)(nil)

// NoCodeRegistryModules describes all the no-code registry module related methods that the Terraform
// Enterprise API supports.
//
// TFE API docs: https://www.terraform.io/docs/cloud/api/modules.html (TODO: update this link)
type NoCodeRegistryModules interface {

	// Create a no-code registry module
	Create(ctx context.Context, organization string, options RegistryNoCodeModuleCreateOptions) (*RegistryNoCodeModule, error)

	// Read a registryno-code  module
	Read(ctx context.Context, noCodeModuleID string, options *RegistryNoCodeModuleReadOptions) (*RegistryNoCodeModule, error)

	// Update a no-code registry module
	Update(ctx context.Context, noCodeModuleID string, options RegistryNoCodeModuleUpdateOptions) (*RegistryNoCodeModule, error)

	// Delete a no-code registry module
	Delete(ctx context.Context, ID string) error
}

// noCodeRegistryModules implements NoCodeRegistryModules.
type noCodeRegistryModules struct {
	client *Client
}

// RegistryNoCodeModule represents a no-code registry module
type RegistryNoCodeModule struct {
	ID                  string `jsonapi:"primary,no-code-modules"`
	Enabled             bool   `jsonapi:"attr,enabled"`
	FollowLatestVersion bool   `jsonapi:"attr,follow-latest-version"`
	VersionPin          string `jsonapi:"attr,version-pin,omitempty"`

	// Relations
	Organization    *Organization           `jsonapi:"relation,organization"`
	RegistryModule  *RegistryModule         `jsonapi:"relation,registry-module"`
	VariableOptions []*NoCodeVariableOption `jsonapi:"relation,variable-options"`
}

// RegistryNoCodeModuleCreateOptions is used when creating a no-code registry module
type RegistryNoCodeModuleCreateOptions struct {
	// Type is a public field utilized by JSON:API to
	// set the resource type via the field tag.
	// It is not a user-defined value and does not need to be set.
	// https://jsonapi.org/format/#crud-creating
	Type string `jsonapi:"primary,no-code-modules"`

	// FollowLatestVersion indicates whether the module should follow the latest version
	FollowLatestVersion *bool `jsonapi:"attr,follow-latest-version"`

	// Enabled indicates whether no-code is enabled for the module
	Enabled *bool `jsonapi:"attr,enabled"`

	// Optional: Variable options for the module
	VariableOptions []*NoCodeVariableOption `jsonapi:"relation,variable-options,omitempty"`

	RegistryModule *RegistryModule `jsonapi:"relation,registry-module"`
}

// NoCodeModuleIncludeOpt represents the available options for include query params.
// https://developer.hashicorp.com/terraform/enterprise/api-docs/admin/organizations#available-related-resources
type NoCodeModuleIncludeOpt string

// RegistryNoCodeModuleReadOptions is used when reading a no-code registry module
type RegistryNoCodeModuleReadOptions struct {
	// Type is a public field utilized by JSON:API to
	// set the resource type via the field tag.
	// It is not a user-defined value and does not need to be set.
	// https://jsonapi.org/format/#crud-updating
	Type string `jsonapi:"primary,no-code-modules"`

	Include []NoCodeModuleIncludeOpt `url:"include,omitempty"`
}

// RegistryNoCodeModuleUpdateOptions is used when updating a no-code registry module
type RegistryNoCodeModuleUpdateOptions struct {
	// Type is a public field utilized by JSON:API to
	// set the resource type via the field tag.
	// It is not a user-defined value and does not need to be set.
	// https://jsonapi.org/format/#crud-updating
	Type string `jsonapi:"primary,no-code-modules"`

	// Optional:
	Version *string `jsonapi:"attr,version,omitempty"`
}

var (
	// NoCodeIncludeVariableOptions is used to include variable options in the response
	NoCodeIncludeVariableOptions NoCodeModuleIncludeOpt = "variable-options"
)

// Create a new no-code registry module
func (r *noCodeRegistryModules) Create(ctx context.Context, organization string, options RegistryNoCodeModuleCreateOptions) (*RegistryNoCodeModule, error) {
	if !validStringID(&organization) {
		return nil, ErrInvalidOrg
	}
	if err := options.valid(); err != nil {
		return nil, err
	}

	u := fmt.Sprintf(
		"organizations/%s/no-code-modules",
		url.QueryEscape(organization),
	)
	req, err := r.client.NewRequest("POST", u, &options)
	if err != nil {
		return nil, err
	}

	rm := &RegistryNoCodeModule{}
	err = req.Do(ctx, rm)
	if err != nil {
		return nil, err
	}

	return rm, nil
}

// Read a no-code registry module
func (r *noCodeRegistryModules) Read(ctx context.Context, noCodeModuleID string, options *RegistryNoCodeModuleReadOptions) (*RegistryNoCodeModule, error) {
	if !validStringID(&noCodeModuleID) {
		return nil, ErrInvalidModuleID
	}

	if err := options.valid(); err != nil {
		return nil, err
	}

	u := fmt.Sprintf(
		"no-code-modules/%s",
		url.QueryEscape(noCodeModuleID),
	)

	req, err := r.client.NewRequest("GET", u, options)
	if err != nil {
		return nil, err
	}

	rm := &RegistryNoCodeModule{}
	err = req.Do(ctx, rm)
	if err != nil {
		return nil, err
	}

	return rm, nil
}

// Update a no-code registry module
func (r *noCodeRegistryModules) Update(ctx context.Context, noCodeModuleID string, options RegistryNoCodeModuleUpdateOptions) (*RegistryNoCodeModule, error) {
	if !validStringID(&noCodeModuleID) {
		return nil, ErrInvalidModuleID
	}
	if !validStringID(&noCodeModuleID) {
		return nil, ErrInvalidModuleID
	}

	u := fmt.Sprintf(
		"no-code-modules/%s",
		url.QueryEscape(noCodeModuleID),
	)

	req, err := r.client.NewRequest("PATCH", u, &options)
	if err != nil {
		return nil, err
	}

	rm := &RegistryNoCodeModule{}
	err = req.Do(ctx, rm)
	if err != nil {
		return nil, err
	}

	return rm, nil
}

// Delete is used to delete the no-code registry module
func (r *noCodeRegistryModules) Delete(ctx context.Context, noCodeModuleID string) error {
	if !validStringID(&noCodeModuleID) {
		return ErrInvalidModuleID
	}

	u := fmt.Sprintf("no-code-modules/%s", url.QueryEscape(noCodeModuleID))
	req, err := r.client.NewRequest("DELETE", u, nil)
	if err != nil {
		return err
	}

	return req.Do(ctx, nil)
}

func (o RegistryNoCodeModuleCreateOptions) valid() error {
	if o.RegistryModule == nil {
		return fmt.Errorf("registry module is required")
	}

	if o.FollowLatestVersion == nil {
		return fmt.Errorf("follow_latest_version field is required")
	}

	if o.Enabled == nil {
		return fmt.Errorf("enabled field is required")
	}
	return nil
}

func (o *RegistryNoCodeModuleReadOptions) valid() error {
	if o == nil {
		return nil // nothing to validate
	}

	if err := validateNoCodeIncludeParams(o.Include); err != nil {
		return err
	}

	return nil
}

func validateNoCodeIncludeParams(params []NoCodeModuleIncludeOpt) error {
	for _, p := range params {
		switch p {
		case NoCodeIncludeVariableOptions:
			// do nothing
		default:
			return ErrInvalidIncludeValue
		}
	}

	return nil
}
