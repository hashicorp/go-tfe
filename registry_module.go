package tfe

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"time"
)

// Compile-time proof of interface implementation.
var _ RegistryModules = (*registryModules)(nil)

// RegistryModules describes all the registry module related methods that the
// Terraform Enterprise API supports.
//
// TFE API docs:
// https://www.terraform.io/docs/enterprise/api/modules.html
type RegistryModules interface {
	// Publish a module in the registry
	Publish(ctx context.Context, options RegistryModulePublishOptions) (*RegistryModule, error)
	// Deletes a module from the registry
	Delete(ctx context.Context, organization string, module string, provider string, version string) error
}

// registryModules implements RegistryModules.
type registryModules struct {
	client *Client
}

// RegistryModuleStatus represents a registry module state.
type RegistryModuleStatus string

//List all available registry module statuses.
const (
	RegistryModulePending       RegistryModuleStatus = "pending"
	RegistryModuleSetupComplete RegistryModuleStatus = "setup_complete"
)

// RegistryModule represents a Terraform Enterprise registry module.
type RegistryModule struct {
	ID              string                         `jsonapi:"primary,registry-modules"`
	Name            string                         `jsonapi:"attr,name"`
	Provider        string                         `jsonapi:"attr,provider"`
	Status          string                         `jsonapi:"attr,status"`
	VersionStatuses []*RegistryModuleVersionStatus `jsonapi:"attr,version-statuses"`
	CreatedAt       time.Time                      `jsonapi:"attr,created-at,iso8601"`
	UpdatedAt       time.Time                      `jsonapi:"attr,updated-at,iso8601"`

	// Relations
	Organization *Organization `jsonapi:"relation,organization"`
}

// RegistryModulePermissions represents the registry module permissions.
type RegistryModulePermissions struct {
	CanDestroy bool `json:"can-destroy"`
	CanResync  bool `json:"can-resync"`
	CanRetry   bool `json:"can-retry"`
}

// RegistryModuleVersionStatus is a list of each version and its associated status.
type RegistryModuleVersionStatus struct {
	version string `json:"version"`
	status  string `json:"status"`
}

//RegistryModulePublishOptions is the config for the publish method
type RegistryModulePublishOptions struct {
	// For internal use only!
	ID string `jsonapi:"primary,registry-modules"`

	// Settings for the registry modules's VCS repository.
	VCSRepo *VCSRepo `jsonapi:"attr,vcs-repo"`
}

func (o RegistryModulePublishOptions) valid() error {
	if o.VCSRepo == nil {
		return errors.New("VCSRepo is required")
	}
	return nil
}

// Publish a module in the registry
func (s *registryModules) Publish(ctx context.Context, options RegistryModulePublishOptions) (*RegistryModule, error) {
	if err := options.valid(); err != nil {
		return nil, err
	}

	// Make sure we don't send a user provided ID.
	options.ID = ""

	u := fmt.Sprintf("registry-modules")
	req, err := s.client.newRequest("POST", u, &options)
	if err != nil {
		return nil, err
	}

	rm := &RegistryModule{}
	err = s.client.do(ctx, req, rm)
	if err != nil {
		return nil, err
	}

	return rm, nil
}

//Delete a module in the registry
func (s *registryModules) Delete(ctx context.Context, organization string, module string, provider string, version string) error {
	if !validStringID(&organization) {
		return errors.New("Invalid value for organization")
	}
	if !validString(&module) {
		return errors.New("Invalid module")
	}

	//construct url endpoint
	u := fmt.Sprintf("registry-modules/actions/delete/%s/%s", url.QueryEscape(organization), url.QueryEscape(module))
	if validString(&provider) {
		u = fmt.Sprintf("%s/%s", u, url.QueryEscape(provider))
		if validString(&version) {
			u = fmt.Sprintf("%s/%s", u, url.QueryEscape(version))
		}
	}
	fmt.Println(u)
	req, err := s.client.newRequest("POST", u, nil)
	if err != nil {
		return err
	}

	return s.client.do(ctx, req, nil)
}
