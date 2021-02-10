package tfe

import (
	"context"
	"errors"
	"fmt"
	"net/url"
)

// Compile-time proof of interface implementation.
var _ AdminOrganizations = (*adminOrganizations)(nil)

// AdminOrganizations describes all of the admin organization related methods that the Terraform
// Enterprise API supports. Note that admin settings are only available in Terraform Enterprise.
//
// TFE API docs: https://www.terraform.io/docs/cloud/api/admin/organizations.html
// Module sharing docs: https://www.terraform.io/docs/cloud/api/admin/module-sharing.html
type AdminOrganizations interface {

	// List the module sharing partnerships that an organization has
	ListModuleConsumers(ctx context.Context, organization string, options OrganizationListOptions) (*OrganizationList, error)

	// Update the module sharing consumers that an organization has
	UpdateModuleConsumers(ctx context.Context, organization string, consumers ModuleConsumers) error

	// Read attributes of an existing organization via admin API.
	Read(ctx context.Context, organization string) (*AdminOrganization, error)

	// Update attributes of an existing organization via admin API.
	Update(ctx context.Context, organization string, options AdminOrganizationUpdateOptions) (*AdminOrganization, error)

	// Delete an organization by its name via admin API
	Delete(ctx context.Context, organization string) error
}

// adminOrganizations implements AdminOrganizations.
type adminOrganizations struct {
	client *Client
}

// ModuleConsumers is a typed list of organization names to represent the
// list of module consumers for updates
type ModuleConsumers []*string

// modulePartnershipUpdateOption represents the option for updating an organization's module sharing partnerships
// this is private as it's used internally
type modulePartnershipUpdateOption struct {
	ID *string `jsonapi:"primary,organizations"`
}

// AdminOrganization represents a Terraform Enterprise organization returned from the Admin API.
type AdminOrganization struct {
	Name string `jsonapi:"primary,organizations"`

	AccessBetaTools                  bool   `jsonapi:"attr,access-beta-tools"`
	ExternalID                       string `jsonapi:"attr,external-id"`
	GlobalModuleSharing              bool   `jsonapi:"attr,global-module-sharing"`
	IsDisabled                       bool   `jsonapi:"attr,is-disabled"`
	NotificationEmail                string `jsonapi:"attr,notification-email"`
	SsoEnabled                       bool   `jsonapi:"attr,sso-enabled"`
	TerraformBuildWorkerApplyTimeout string `jsonapi:"attr,terraform-build-worker-apply-timeout"`
	TerraformBuildWorkerPlanTimeout  string `jsonapi:"attr,terraform-build-worker-plan-timeout"`
	TerraformWorkerSudoEnabled       bool   `jsonapi:"attr,terraform-worker-sudo-enabled"`
}

// AdminOrganizationUpdateOptions represents the admin options for updating an organization.
// https://www.terraform.io/docs/cloud/api/admin/organizations.html#request-body
type AdminOrganizationUpdateOptions struct {
	AccessBetaTools                  *bool   `jsonapi:"attr,access-beta-tools,omitempty"`
	GlobalModuleSharing              *bool   `jsonapi:"attr,global-module-sharing,omitempty"`
	IsDisabled                       *bool   `jsonapi:"attr,is-disabled,omitempty"`
	TerraformBuildWorkerApplyTimeout *string `jsonapi:"attr,terraform-build-worker-apply-timeout,omitempty"`
	TerraformBuildWorkerPlanTimeout  *string `jsonapi:"attr,terraform-build-worker-plan-timeout,omitempty"`
}

func (s *adminOrganizations) ListModuleConsumers(ctx context.Context, organization string, options OrganizationListOptions) (*OrganizationList, error) {
	if !validStringID(&organization) {
		return nil, errors.New("invalid value for organization")
	}

	u := fmt.Sprintf("admin/organizations/%s/relationships/module-consumers", url.QueryEscape(organization))
	req, err := s.client.newRequest("GET", u, options)
	if err != nil {
		return nil, err
	}

	partnerships := &OrganizationList{}
	err = s.client.do(ctx, req, partnerships)
	if err != nil {
		return nil, err
	}

	return partnerships, nil
}

func (s *adminOrganizations) UpdateModuleConsumers(ctx context.Context, organization string, consumers ModuleConsumers) error {
	if !validStringID(&organization) {
		return errors.New("invalid value for organization")
	}

	options := []*modulePartnershipUpdateOption{}
	for _, consumer := range consumers {
		options = append(options, &modulePartnershipUpdateOption{
			ID: consumer,
		})
	}

	u := fmt.Sprintf("admin/organizations/%s/relationships/module-consumers", url.QueryEscape(organization))
	req, err := s.client.newRequest("PATCH", u, options)
	if err != nil {
		return err
	}

	return s.client.do(ctx, req, nil)
}

func (s *adminOrganizations) Read(ctx context.Context, organization string) (*AdminOrganization, error) {
	if !validStringID(&organization) {
		return nil, errors.New("invalid value for organization")
	}

	u := fmt.Sprintf("admin/organizations/%s", url.QueryEscape(organization))
	req, err := s.client.newRequest("GET", u, nil)
	if err != nil {
		return nil, err
	}

	org := &AdminOrganization{}
	err = s.client.do(ctx, req, org)
	if err != nil {
		return nil, err
	}

	return org, nil
}

func (s *adminOrganizations) Update(ctx context.Context, organization string, options AdminOrganizationUpdateOptions) (*AdminOrganization, error) {
	if !validStringID(&organization) {
		return nil, errors.New("invalid value for organization")
	}

	u := fmt.Sprintf("admin/organizations/%s", url.QueryEscape(organization))
	req, err := s.client.newRequest("PATCH", u, &options)
	if err != nil {
		return nil, err
	}

	org := &AdminOrganization{}
	err = s.client.do(ctx, req, org)
	if err != nil {
		return nil, err
	}

	return org, nil
}

// Delete an organization by its name.
func (s *adminOrganizations) Delete(ctx context.Context, organization string) error {
	if !validStringID(&organization) {
		return errors.New("invalid value for organization")
	}

	u := fmt.Sprintf("admin/organizations/%s", url.QueryEscape(organization))
	req, err := s.client.newRequest("DELETE", u, nil)
	if err != nil {
		return err
	}

	return s.client.do(ctx, req, nil)
}
