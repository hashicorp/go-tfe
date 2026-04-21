// Copyright IBM Corp. 2018, 2026
// SPDX-License-Identifier: MPL-2.0

package tfe

import (
	"context"
)

var _ AdminSCIMGroups = (*adminSCIMGroups)(nil)

// AdminSCIMGroups describes all the SCIM group related methods that the Terraform
// Enterprise API supports
//
// TFE API docs: https://developer.hashicorp.com/terraform/enterprise/api-docs/admin/scim-groups
type AdminSCIMGroups interface {
	// List all SCIM groups.
	List(ctx context.Context, options *AdminSCIMGroupListOptions) (*AdminSCIMGroupList, error)
}

// adminSCIMGroups implements AdminSCIMGroups
type adminSCIMGroups struct {
	client *Client
}

// AdminSCIMGroupList represents a list of SCIM groups
type AdminSCIMGroupList struct {
	*Pagination
	Items []*AdminSCIMGroup
}

// AdminSCIMGroup represents a Terraform Enterprise SCIM group
type AdminSCIMGroup struct {
	ID   string `jsonapi:"primary,scim-groups"`
	Name string `jsonapi:"attr,name"`
}

// AdminSCIMGroupListOptions represents the options for listing SCIM groups
type AdminSCIMGroupListOptions struct {
	ListOptions
	Query string `url:"q,omitempty"`
}

// List all SCIM groups.
func (a *adminSCIMGroups) List(ctx context.Context, options *AdminSCIMGroupListOptions) (*AdminSCIMGroupList, error) {
	req, err := a.client.NewRequest("GET", AdminSCIMGroupsPath, options)
	if err != nil {
		return nil, err
	}

	scimGroups := &AdminSCIMGroupList{}
	err = req.Do(ctx, scimGroups)
	if err != nil {
		return nil, err
	}

	return scimGroups, nil
}
