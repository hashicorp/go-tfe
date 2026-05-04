// Copyright IBM Corp. 2018, 2026
// SPDX-License-Identifier: MPL-2.0

package tfe

import (
	"context"
	"fmt"
	"net/url"
)

var _ AdminSCIMGroupMappings = (*adminSCIMGroupMappings)(nil)

// AdminSCIMGroupMappings describes all the SCIM group mapping related methods that the Terraform
// Enterprise API supports
//
// TFE API docs: https://developer.hashicorp.com/terraform/enterprise/api-docs/admin/team-scim-group-mapping
type AdminSCIMGroupMappings interface {
	// Create a SCIM group mapping.
	Create(ctx context.Context, teamID string, options *AdminSCIMGroupMappingCreateOptions) error

	// Update a SCIM group mapping.
	Update(ctx context.Context, teamID string, options *AdminSCIMGroupMappingUpdateOptions) error

	// Delete a SCIM group mapping.
	Delete(ctx context.Context, teamID string) error
}

// adminSCIMGroupMappings implements AdminSCIMGroupMappings
type adminSCIMGroupMappings struct {
	client *Client
}

// AdminSCIMGroupMappingCreateOptions represents the options for creating a SCIM group mapping
type AdminSCIMGroupMappingCreateOptions struct {
	// Type is a public field utilized by JSON:API to
	// set the resource type via the field tag.
	// It is not a user-defined value and does not need to be set.
	// https://jsonapi.org/format/#crud-creating
	Type        string `jsonapi:"primary,scim-group-mappings"`
	SCIMGroupID string `jsonapi:"attr,scim-group-id"`
}

// AdminSCIMGroupMappingUpdateOptions represents the options for updating a SCIM group mapping
type AdminSCIMGroupMappingUpdateOptions struct {
	// Type is a public field utilized by JSON:API to
	// set the resource type via the field tag.
	// It is not a user-defined value and does not need to be set.
	// https://jsonapi.org/format/#crud-creating
	Type           string `jsonapi:"primary,scim-group-mappings"`
	SCIMSyncPaused *bool  `jsonapi:"attr,scim-sync-paused"`
}

// Create a SCIM group mapping.
func (a *adminSCIMGroupMappings) Create(ctx context.Context, teamID string, options *AdminSCIMGroupMappingCreateOptions) error {
	if !validStringID(&teamID) {
		return ErrInvalidTeamID
	}
	if options == nil {
		return ErrRequiredSCIMGroupMappingCreateOps
	}
	if !validStringID(&options.SCIMGroupID) {
		return ErrInvalidSCIMGroupID
	}

	req, err := a.client.NewRequest("POST", fmt.Sprintf(AdminSCIMGroupMappingPath, url.PathEscape(teamID)), options)
	if err != nil {
		return err
	}

	return req.Do(ctx, nil)
}

// Update a SCIM group mapping.
func (a *adminSCIMGroupMappings) Update(ctx context.Context, teamID string, options *AdminSCIMGroupMappingUpdateOptions) error {
	if !validStringID(&teamID) {
		return ErrInvalidTeamID
	}

	if options == nil {
		return ErrRequiredSCIMGroupMappingUpdateOps
	}

	if options.SCIMSyncPaused == nil {
		return ErrSCIMSyncPausedNil
	}

	req, err := a.client.NewRequest("PATCH", fmt.Sprintf(AdminSCIMGroupMappingPath, url.PathEscape(teamID)), options)
	if err != nil {
		return err
	}

	return req.Do(ctx, nil)
}

// Delete a SCIM group mapping.
func (a *adminSCIMGroupMappings) Delete(ctx context.Context, teamID string) error {
	if !validStringID(&teamID) {
		return ErrInvalidTeamID
	}

	req, err := a.client.NewRequest("DELETE", fmt.Sprintf(AdminSCIMGroupMappingPath, url.PathEscape(teamID)), nil)
	if err != nil {
		return err
	}

	return req.Do(ctx, nil)
}
