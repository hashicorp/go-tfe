package tfe

import (
	"context"
	"fmt"
	"net/url"
)

// Compile-time proof of interface implementation.
var _ AdminOrganizations = (*adminOrganizations)(nil)

// AdminOrganizations describes all of the admin organization related methods that the Terraform
// Enterprise API supports. Note that admin settings are only available in Terraform Enterprise.
//
// TFE API docs: https://www.terraform.io/docs/cloud/api/admin/organizations.html
type AdminOrganizations interface {
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

// AdminOrganization represents a Terraform Enterprise organization returned from the Admin API.
type AdminOrganization struct {
	Name string `jsonapi:"primary,organizations"`

	AccessBetaTools                  bool   `jsonapi:"attr,access-beta-tools"`
	ExternalID                       string `jsonapi:"attr,external-id"`
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
	IsDisabled                       *bool   `jsonapi:"attr,is-disabled,omitempty"`
	TerraformBuildWorkerApplyTimeout *string `jsonapi:"attr,terraform-build-worker-apply-timeout,omitempty"`
	TerraformBuildWorkerPlanTimeout  *string `jsonapi:"attr,terraform-build-worker-plan-timeout,omitempty"`
}

func (s *adminOrganizations) Read(ctx context.Context, organization string) (*AdminOrganization, error) {
	if !validStringID(&organization) {
		return nil, ErrInvalidOrg
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
		return nil, ErrInvalidOrg
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
		return ErrInvalidOrg
	}

	u := fmt.Sprintf("admin/organizations/%s", url.QueryEscape(organization))
	req, err := s.client.newRequest("DELETE", u, nil)
	if err != nil {
		return err
	}

	return s.client.do(ctx, req, nil)
}
