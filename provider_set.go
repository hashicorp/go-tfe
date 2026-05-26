// Copyright IBM Corp. 2018, 2025
// SPDX-License-Identifier: MPL-2.0
package tfe

import (
	"context"
	"fmt"
	"net/url"
)

// Compile-time proof of interface implementation
var _ ProviderSets = (*providerSets)(nil)

// ProviderSets describes all the Provider Set related methods that the
// Terraform Enterprise API supports.
//
// TFE API docs: https://developer.hashicorp.com/terraform/cloud-docs/api-docs/provider-sets
type ProviderSets interface {
	// Create is used to create a new provider set.
	Create(ctx context.Context, organization string, options ProviderSetCreateOptions) (*ProviderSet, error)

	// Read a provider set by its ID.
	Read(ctx context.Context, providerSetID string) (*ProviderSet, error)

	// Update values of an existing provider set.
	Update(ctx context.Context, providerSetID string, options ProviderSetUpdateOptions) (*ProviderSet, error)

	// Delete a provider set by its ID.
	Delete(ctx context.Context, providerSetID string) error
}

// ProviderSet describes all the Provider Set related methods that the
type ProviderSet struct {
	ID               string `jsonapi:"primary,provider-sets"`
	Name             string `jsonapi:"attr,name"`
	Description      string `jsonapi:"attr,description"`
	ProviderSource   string `jsonapi:"attr,provider-source"`
	ConfigurationHcl string `jsonapi:"attr,configuration-hcl"`
	Global           bool   `jsonapi:"attr,global"`

	// Relations
	Organization *Organization `jsonapi:"relation,organization"`
	Workspaces   []*Workspace  `jsonapi:"relation,workspaces,omitempty"`
	Projects     []*Project    `jsonapi:"relation,projects,omitempty"`
}

// providerSets implements ProviderSets.
type providerSets struct {
	client *Client
}

// ProviderSetCreateOptions represents the options for creating a new provider set.
type ProviderSetCreateOptions struct {
	// Type is a public field utilized by JSON:API to
	// set the resource type via the field tag.
	// It is not a user-defined value and does not need to be set.
	// https://jsonapi.org/format/#crud-creating
	Type string `jsonapi:"primary,provider-sets"`

	// Required: Name of the provider set.
	Name string `jsonapi:"attr,name"`

	// Optional: Description to provide context for the provider set.
	Description *string `jsonapi:"attr,description,omitempty"`

	// Required: Provider source represents the source of the provider set.
	// (ie: "registry.terraform.io/hashicorp/aws")
	ProviderSource string `jsonapi:"attr,provider-source"`

	// Required: ConfigurationHcl represents the HCL configuration for the provider set.
	ConfigurationHcl string `jsonapi:"attr,configuration-hcl"`

	// Optional: If true the provider set is considered in all runs in the organization.
	Global *bool `jsonapi:"attr,global,omitempty"`

	// Optional: Workspaces are the workspaces assigned to the provider set.
	Workspaces []*Workspace `jsonapi:"relation,workspaces,omitempty"`
	// Optional: Projects are the projects assigned to the provider set.
	Projects []*Project `jsonapi:"relation,projects,omitempty"`
}

// ProviderSetUpdateOptions represents the options for creating a new provider set.
type ProviderSetUpdateOptions struct {
	// Optional: Name of the provider set.
	Name *string

	// Optional: Description to provide context for the provider set.
	Description *string

	// Optional: Provider source represents the source of the provider set.
	// (ie: "registry.terraform.io/hashicorp/aws")
	ProviderSource *string

	// Optional: ConfigurationHcl represents the HCL configuration for the provider set.
	ConfigurationHcl *string

	// Optional: If true the provider set is considered in all runs in the organization.
	Global *bool

	// Optional: Workspaces are the workspaces assigned to the provider set. Providing
	// nil will be a NOP and empty array will remove all workspaces from the provider set.
	Workspaces []*Workspace
	// Optional: Projects are the projects assigned to the provider set. Providing
	// nil will be a NOP and empty array will remove all projects from the provider set.
	Projects []*Project
}

// These payload structs exist because partial updates need custom relationship encoding:
// omitted relationships must be left unchanged, while empty arrays must clear them.
// The generic JSON:API struct tags used elsewhere in go-tfe do not cleanly express
// that omitted-vs-empty distinction for this single PATCH request.

type providerSetUpdatePayload struct {
	Data providerSetUpdatePayloadData `json:"data"`
}

type providerSetUpdatePayloadData struct {
	Type          string                             `json:"type"`
	Attributes    providerSetUpdatePayloadAttributes `json:"attributes"`
	Relationships map[string]relationshipData        `json:"relationships"`
}

type providerSetUpdatePayloadAttributes struct {
	Name             *string `json:"name,omitempty"`
	Description      *string `json:"description,omitempty"`
	ProviderSource   *string `json:"provider_source,omitempty"`
	ConfigurationHcl *string `json:"configuration_hcl,omitempty"`
	Global           *bool   `json:"global,omitempty"`
}

func (o ProviderSetUpdateOptions) payload() *providerSetUpdatePayload {
	payload := providerSetUpdatePayload{
		Data: providerSetUpdatePayloadData{
			Type: "provider-sets",
			Attributes: providerSetUpdatePayloadAttributes{
				Name:             o.Name,
				Description:      o.Description,
				ProviderSource:   o.ProviderSource,
				ConfigurationHcl: o.ConfigurationHcl,
				Global:           o.Global,
			},
			Relationships: make(map[string]relationshipData),
		},
	}

	if o.Workspaces != nil {
		data := make([]relationshipItem, len(o.Workspaces))
		for i, ws := range o.Workspaces {
			data[i] = ws.relationshipItem()
		}
		payload.Data.Relationships["workspaces"] = relationshipData{Data: data}
	}

	if o.Projects != nil {
		data := make([]relationshipItem, len(o.Projects))
		for i, proj := range o.Projects {
			data[i] = proj.relationshipItem()
		}
		payload.Data.Relationships["projects"] = relationshipData{Data: data}
	}

	return &payload
}

// Create is used to create a new provider set.
func (p *providerSets) Create(ctx context.Context, organization string, options ProviderSetCreateOptions) (*ProviderSet, error) {
	if !validStringID(&organization) {
		return nil, ErrInvalidOrg
	}
	if err := options.valid(); err != nil {
		return nil, err
	}

	u := fmt.Sprintf("organizations/%s/provider-sets", url.PathEscape(organization))
	req, err := p.client.NewRequest("POST", u, &options)
	if err != nil {
		return nil, err
	}

	ps := &ProviderSet{}
	err = req.Do(ctx, ps)
	if err != nil {
		return nil, err
	}

	return ps, nil
}

// Read a provider set by its ID.
func (p *providerSets) Read(ctx context.Context, providerSetID string) (*ProviderSet, error) {
	if !validStringID(&providerSetID) {
		return nil, ErrInvalidProviderSetID
	}

	u := fmt.Sprintf("provider-sets/%s", url.PathEscape(providerSetID))
	req, err := p.client.NewRequest("GET", u, nil)
	if err != nil {
		return nil, err
	}

	ps := &ProviderSet{}
	err = req.Do(ctx, ps)
	if err != nil {
		return nil, err
	}

	return ps, nil
}

// Update values of an existing provider set.
func (p *providerSets) Update(ctx context.Context, providerSetID string, options ProviderSetUpdateOptions) (*ProviderSet, error) {
	if !validStringID(&providerSetID) {
		return nil, ErrInvalidProviderSetID
	}
	if err := options.valid(); err != nil {
		return nil, err
	}
	u := fmt.Sprintf("provider-sets/%s", url.PathEscape(providerSetID))
	req, err := p.client.NewRequest("PATCH", u, options.payload())
	if err != nil {
		return nil, err
	}

	ps := &ProviderSet{}
	err = req.Do(ctx, ps)
	if err != nil {
		return nil, err
	}

	return ps, nil
}

// Delete a provider set by its ID.
func (p *providerSets) Delete(ctx context.Context, providerSetID string) error {
	if !validStringID(&providerSetID) {
		return ErrInvalidProviderSetID
	}

	u := fmt.Sprintf("provider-sets/%s", url.PathEscape(providerSetID))
	req, err := p.client.NewRequest("DELETE", u, nil)
	if err != nil {
		return err
	}

	return req.Do(ctx, nil)
}

func (o ProviderSetCreateOptions) valid() error {
	if !validString(&o.Name) {
		return ErrRequiredName
	}
	if !validStringID(&o.Name) {
		return ErrInvalidName
	}
	if !validString(&o.ProviderSource) {
		return ErrRequiredProviderSource
	}
	if !validString(&o.ConfigurationHcl) {
		return ErrRequiredConfigurationHcl
	}
	if o.Global != nil && *o.Global && (len(o.Workspaces) > 0 || len(o.Projects) > 0) {
		return ErrProviderSetGlobalRelationships
	}
	for _, w := range o.Workspaces {
		if !validString(&w.ID) {
			return ErrRequiredWorkspaceID
		}
		if !validStringID(&w.ID) {
			return ErrInvalidWorkspaceID
		}
	}
	for _, p := range o.Projects {
		if !validString(&p.ID) {
			return ErrRequiredProjectID
		}
		if !validStringID(&p.ID) {
			return ErrInvalidProjectID
		}
	}
	return nil
}

func (o ProviderSetUpdateOptions) valid() error {
	if o.Name != nil && !validStringID(o.Name) {
		return ErrInvalidName
	}
	if o.Global != nil && *o.Global && (len(o.Workspaces) > 0 || len(o.Projects) > 0) {
		return ErrProviderSetGlobalRelationships
	}
	for _, w := range o.Workspaces {
		if !validString(&w.ID) {
			return ErrRequiredWorkspaceID
		}
		if !validStringID(&w.ID) {
			return ErrInvalidWorkspaceID
		}
	}
	for _, p := range o.Projects {
		if !validString(&p.ID) {
			return ErrRequiredProjectID
		}
		if !validStringID(&p.ID) {
			return ErrInvalidProjectID
		}
	}
	return nil
}
