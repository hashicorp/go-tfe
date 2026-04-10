// Copyright IBM Corp. 2018, 2026
// SPDX-License-Identifier: MPL-2.0

package tfe

import (
	"context"
)

// Compile-time proof of interface implementation.
var _ ScimSettings = (*adminScimSettings)(nil)

// ScimSettings describes all the scim settings related methods that the Terraform
// Enterprise API supports
//
// TFE API docs: https://developer.hashicorp.com/terraform/enterprise/api-docs/admin/settings
type ScimSettings interface {

	// Read scim settings
	Read(ctx context.Context) (*AdminScimSetting, error)

	// Update scim settings
	Update(ctx context.Context, options AdminScimSettingUpdateOptions) (*AdminScimSetting, error)

	// Delete scim settings
	Delete(ctx context.Context) error
}

// adminScimSettings implements ScimSettings.
type adminScimSettings struct {
	client *Client
}

// AdminScimSetting represents the SCIM setting in Terraform Enterprise
type AdminScimSetting struct {
	ID                        string `jsonapi:"primary,scim-settings"`
	Enabled                   bool   `jsonapi:"attr,enabled"`
	Paused                    bool   `jsonapi:"attr,paused"`
	SiteAdminGroupScimID      string `jsonapi:"attr,site-admin-group-scim-id,omitempty"`
	SiteAdminGroupDisplayName string `jsonapi:"attr,site-admin-group-display-name,omitempty"`
}

// AdminScimSettingUpdateOptions represents the options for updating an admin SCIM setting.
type AdminScimSettingUpdateOptions struct {
	Enabled              *bool   `jsonapi:"attr,enabled,omitempty"`
	Paused               *bool   `jsonapi:"attr,paused,omitempty"`
	SiteAdminGroupScimID *string `jsonapi:"attr,site-admin-group-scim-id,omitempty"`
}

// Read scim setting.
func (a *adminScimSettings) Read(ctx context.Context) (*AdminScimSetting, error) {
	req, err := a.client.NewRequest("GET", "admin/scim-settings", nil)
	if err != nil {
		return nil, err
	}

	scim := &AdminScimSetting{}
	err = req.Do(ctx, scim)
	if err != nil {
		return nil, err
	}

	return scim, nil
}

// Update scim setting.
func (a *adminScimSettings) Update(ctx context.Context, options AdminScimSettingUpdateOptions) (*AdminScimSetting, error) {
	req, err := a.client.NewRequest("PATCH", "admin/scim-settings", &options)
	if err != nil {
		return nil, err
	}

	scim := &AdminScimSetting{}
	err = req.Do(ctx, scim)
	if err != nil {
		return nil, err
	}
	return scim, nil
}

// Delete scim setting.
func (a *adminScimSettings) Delete(ctx context.Context) error {
	req, err := a.client.NewRequest("DELETE", "admin/scim-settings", nil)
	if err != nil {
		return err
	}

	return req.Do(ctx, nil)
}
