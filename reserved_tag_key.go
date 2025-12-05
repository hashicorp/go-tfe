// Copyright IBM Corp. 2018, 2025
// SPDX-License-Identifier: MPL-2.0

package tfe

import (
	"context"
	"fmt"
	"net/url"
	"time"
)

// Compile-time proof of interface implementation.
var _ ReservedTagKeys = (*reservedTagKeys)(nil)

// ReservedTagKeys describes all the reserved tag key endpoints that the
// Terraform Enterprise API supports.
//
// TFE API docs: https://developer.hashicorp.com/terraform/cloud-docs/api-docs/reserved-tag-keys
type ReservedTagKeys interface {
	// List all the reserved tag keys for the given organization.
	List(ctx context.Context, organization string, options *ReservedTagKeyListOptions) (*ReservedTagKeyList, error)

	// Create a new reserved tag key for the given organization.
	Create(ctx context.Context, organization string, options ReservedTagKeyCreateOptions) (*ReservedTagKey, error)

	// Update the reserved tag key with the given ID.
	Update(ctx context.Context, reservedTagKeyID string, options ReservedTagKeyUpdateOptions) (*ReservedTagKey, error)

	// Delete the reserved tag key with the given ID.
	Delete(ctx context.Context, reservedTagKeyID string) error
}

// reservedTagKeys implements ReservedTagKeys.
type reservedTagKeys struct {
	client *Client
}

// ReservedTagKeyList represents a list of reserved tag keys.
type ReservedTagKeyList struct {
	*Pagination
	Items []*ReservedTagKey
}

// ReservedTagKey represents a Terraform Enterprise reserved tag key.
type ReservedTagKey struct {
	ID               string    `jsonapi:"primary,reserved-tag-keys"`
	Key              string    `jsonapi:"attr,key"`
	DisableOverrides bool      `jsonapi:"attr,disable-overrides"`
	CreatedAt        time.Time `jsonapi:"attr,created_at,iso8601"`
	UpdatedAt        time.Time `jsonapi:"attr,updated_at,iso8601"`
}

// ReservedTagKeyListOptions represents the options for listing reserved tag
// keys.
type ReservedTagKeyListOptions struct {
	ListOptions
}

// List all the reserved tag keys for the given organization.
func (s *reservedTagKeys) List(ctx context.Context, organization string, options *ReservedTagKeyListOptions) (*ReservedTagKeyList, error) {
	if !validStringID(&organization) {
		return nil, ErrInvalidOrg
	}

	u := fmt.Sprintf("organizations/%s/reserved-tag-keys", url.PathEscape(organization))

	req, err := s.client.NewRequest("GET", u, options)
	if err != nil {
		return nil, err
	}

	rl := &ReservedTagKeyList{}
	err = req.Do(ctx, rl)
	if err != nil {
		return nil, err
	}

	return rl, nil
}

// ReservedTagKeyCreateOptions represents the options for creating a
// reserved tag key.
type ReservedTagKeyCreateOptions struct {
	// Type is a public field utilized by JSON:API to
	// set the resource type via the field tag.
	// It is not a user-defined value and does not need to be set.
	// https://jsonapi.org/format/#crud-creating
	Type string `jsonapi:"primary,reserved-tag-keys"`

	// Required: The reserved tag key's key string.
	Key string `jsonapi:"attr,key"`

	// Optional: When true, project tag bindings that match this reserved tag key can not
	// be overridden at the workspace level.
	DisableOverrides *bool `jsonapi:"attr,disable-overrides,omitempty"`
}

func (o ReservedTagKeyCreateOptions) valid() error {
	if !validString(&o.Key) {
		return ErrRequiredKey
	}
	return nil
}

// Create a reserved tag key.
func (s *reservedTagKeys) Create(ctx context.Context, organization string, options ReservedTagKeyCreateOptions) (*ReservedTagKey, error) {
	if !validStringID(&organization) {
		return nil, ErrInvalidOrg
	}

	if err := options.valid(); err != nil {
		return nil, err
	}

	u := fmt.Sprintf("organizations/%s/reserved-tag-keys", url.PathEscape(organization))
	req, err := s.client.NewRequest("POST", u, &options)
	if err != nil {
		return nil, err
	}

	r := &ReservedTagKey{}
	err = req.Do(ctx, r)
	if err != nil {
		return nil, err
	}

	return r, nil
}

// ReservedTagKeyUpdateOptions represents the options for updating a
// reserved tag key.
type ReservedTagKeyUpdateOptions struct {
	// Type is a public field utilized by JSON:API to
	// set the resource type via the field tag.
	// It is not a user-defined value and does not need to be set.
	// https://jsonapi.org/format/#crud-creating
	Type string `jsonapi:"primary,reserved-tag-keys"`

	// Optional: The reserved tag key's key string.
	Key *string `jsonapi:"attr,key,omitempty"`

	// Optional: When true, project tag bindings that match this reserved tag key can not
	// be overridden at the workspace level.
	DisableOverrides *bool `jsonapi:"attr,disable-overrides,omitempty"`
}

// Update the reserved tag key with the given ID.
func (s *reservedTagKeys) Update(ctx context.Context, reservedTagKeyID string, options ReservedTagKeyUpdateOptions) (*ReservedTagKey, error) {
	if !validStringID(&reservedTagKeyID) {
		return nil, ErrInvalidReservedTagKeyID
	}

	u := fmt.Sprintf("reserved-tag-keys/%s", url.PathEscape(reservedTagKeyID))
	req, err := s.client.NewRequest("PATCH", u, &options)
	if err != nil {
		return nil, err
	}

	r := &ReservedTagKey{}
	err = req.Do(ctx, r)
	if err != nil {
		return nil, err
	}

	return r, nil
}

// Delete the reserved tag key with the given ID.
func (s *reservedTagKeys) Delete(ctx context.Context, reservedTagKeyID string) error {
	if !validStringID(&reservedTagKeyID) {
		return ErrInvalidReservedTagKeyID
	}

	u := fmt.Sprintf("reserved-tag-keys/%s", url.PathEscape(reservedTagKeyID))
	req, err := s.client.NewRequest("DELETE", u, nil)
	if err != nil {
		return err
	}

	return req.Do(ctx, nil)
}
