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
// Module sharing docs: https://www.terraform.io/docs/cloud/api/admin/module-sharing.html
type AdminOrganizations interface {

	// List the module sharing partnerships that an organization has
	ListModuleConsumers(ctx context.Context, organization string) (*ModulePartnershipList, error)

	// Update the module sharing partnerships that an organization has
	UpdateModuleConsumers(ctx context.Context, organization string, options ModulePartnershipUpdateOptions) (*ModulePartnershipList, error)
}

// adminOrganizations implements AdminOrganizations.
type adminOrganizations struct {
	client *Client
}

// ModulePartnershipList represents the list of module sharing partnerships
type ModulePartnershipList struct {
	*Pagination
	Items []*ModulePartnership
}

// ModulePartnership represents the module sharing partnership between two organizations
type ModulePartnership struct {
	ConsumingOrganizationID   *string `jsonapi:"attr,consuming-organization-id"`
	ConsumingOrganizationName *string `jsonapi:"attr,consuming-organization-name"`
	ProducingOrganizationID   *string `jsonapi:"attr,producing-organization-id"`
	ProducingOrganizationName *string `jsonapi:"attr,producing-organization-name"`
}

// ModulePartnershipUpdateOptions represents the options for updating an organization's module sharing partnerships
type ModulePartnershipUpdateOptions struct {
	ModuleConsumingOrganizationIDs []*string `jsonapi:"attr,module-consuming-organization-ids"`
}

func (s *adminOrganizations) ListModuleConsumers(ctx context.Context, organization string) (*ModulePartnershipList, error) {
	u := fmt.Sprintf("admin/organizations/%s/module-consumers", url.QueryEscape(organization))
	req, err := s.client.newRequest("GET", u, nil)
	if err != nil {
		return nil, err
	}

	partnerships := &ModulePartnershipList{}
	err = s.client.do(ctx, req, partnerships)
	if err != nil {
		return nil, err
	}

	return partnerships, nil
}

func (s *adminOrganizations) UpdateModuleConsumers(ctx context.Context, organization string, options ModulePartnershipUpdateOptions) (*ModulePartnershipList, error) {
	u := fmt.Sprintf("admin/organizations/%s/module-consumers", url.QueryEscape(organization))
	req, err := s.client.newRequest("PATCH", u, &options)
	if err != nil {
		return nil, err
	}

	partnerships := &ModulePartnershipList{}
	err = s.client.do(ctx, req, partnerships)
	if err != nil {
		return nil, err
	}

	return partnerships, nil
}
