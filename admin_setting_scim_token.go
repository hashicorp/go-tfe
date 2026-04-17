// Copyright IBM Corp. 2018, 2026
// SPDX-License-Identifier: MPL-2.0

package tfe

import (
	"context"
	"fmt"
	"net/url"
	"time"
)

var _ AdminSCIMTokens = (*adminSCIMTokens)(nil)

// AdminSCIMTokens describes all the Admin SCIM token related methods that the Terraform
// Enterprise API supports
//
// TFE API docs: https://developer.hashicorp.com/terraform/enterprise/api-docs/admin/scim-tokens
type AdminSCIMTokens interface {
	// List all Admin SCIM tokens.
	List(ctx context.Context) (*AdminSCIMTokenList, error)

	// Create an Admin SCIM token.
	Create(ctx context.Context, description string) (*AdminSCIMToken, error)

	// Create an Admin SCIM token with options.
	CreateWithOptions(ctx context.Context, options AdminSCIMTokenCreateOptions) (*AdminSCIMToken, error)

	// Read an Admin SCIM token by its ID.
	Read(ctx context.Context, scimTokenID string) (*AdminSCIMToken, error)

	// Delete an Admin SCIM token.
	Delete(ctx context.Context, scimTokenID string) error
}

// adminSCIMTokens implements AdminSCIMTokens
type adminSCIMTokens struct {
	client *Client
}

// AdminSCIMTokenList represents a list of Admin SCIM tokens
type AdminSCIMTokenList struct {
	Items []*AdminSCIMToken
}

// AdminSCIMToken represents a Terraform Enterprise Admin SCIM token.
type AdminSCIMToken struct {
	ID          string    `jsonapi:"primary,authentication-tokens"`
	CreatedAt   time.Time `jsonapi:"attr,created-at,iso8601"`
	LastUsedAt  time.Time `jsonapi:"attr,last-used-at,iso8601"`
	ExpiredAt   time.Time `jsonapi:"attr,expired-at,iso8601"`
	Description string    `jsonapi:"attr,description"`
	Token       string    `jsonapi:"attr,token,omitempty"`
}

// AdminSCIMTokenCreateOptions represents the options for creating an Admin SCIM token
type AdminSCIMTokenCreateOptions struct {
	// Required: A human-readable description of the token's purpose
	// (for example, Okta SCIM Integration).
	Description *string `jsonapi:"attr,description"`

	// Optional: Optional ISO-8601 timestamp for token expiration.
	// Defaults to 365 days in the future. Must be between 29 and 365 days in the future.
	ExpiredAt *time.Time `jsonapi:"attr,expired-at,iso8601,omitempty"`
}

// List all Admin SCIM tokens.
func (a *adminSCIMTokens) List(ctx context.Context) (*AdminSCIMTokenList, error) {
	req, err := a.client.NewRequest("GET", AdminSCIMTokensPath, nil)
	if err != nil {
		return nil, err
	}

	scimTokens := &AdminSCIMTokenList{}
	err = req.Do(ctx, scimTokens)
	if err != nil {
		return nil, err
	}
	return scimTokens, nil
}

// Create an Admin SCIM token.
func (a *adminSCIMTokens) Create(ctx context.Context, description string) (*AdminSCIMToken, error) {
	return a.CreateWithOptions(ctx, AdminSCIMTokenCreateOptions{
		Description: &description,
	})
}

// Create an Admin SCIM token with options.
func (a *adminSCIMTokens) CreateWithOptions(ctx context.Context, options AdminSCIMTokenCreateOptions) (*AdminSCIMToken, error) {
	if !validString(options.Description) {
		return nil, ErrSCIMTokenDescription
	}
	req, err := a.client.NewRequest("POST", AdminSCIMTokensPath, &options)
	if err != nil {
		return nil, err
	}
	scimToken := &AdminSCIMToken{}
	err = req.Do(ctx, scimToken)
	if err != nil {
		return nil, err
	}
	return scimToken, nil
}

// Read an Admin SCIM token by its ID.
func (a *adminSCIMTokens) Read(ctx context.Context, scimTokenID string) (*AdminSCIMToken, error) {
	if !validStringID(&scimTokenID) {
		return nil, ErrInvalidTokenID
	}
	u := fmt.Sprintf(AdminSCIMTokenPath, url.PathEscape(scimTokenID))
	req, err := a.client.NewRequest("GET", u, nil)
	if err != nil {
		return nil, err
	}
	scimToken := &AdminSCIMToken{}
	err = req.Do(ctx, scimToken)
	if err != nil {
		return nil, err
	}
	return scimToken, nil
}

// Delete an Admin SCIM token.
func (a *adminSCIMTokens) Delete(ctx context.Context, scimTokenID string) error {
	if !validStringID(&scimTokenID) {
		return ErrInvalidTokenID
	}
	u := fmt.Sprintf(AuthenticationTokensPath, url.PathEscape(scimTokenID))
	req, err := a.client.NewRequest("DELETE", u, nil)
	if err != nil {
		return err
	}
	return req.Do(ctx, nil)
}
