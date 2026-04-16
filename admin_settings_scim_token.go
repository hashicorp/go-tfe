package tfe

import (
	"context"
	"fmt"
	"net/url"
	"time"
)

var _ AdminSCIMTokens = (*adminSCIMTokens)(nil)

// AdminSCIMTokens describes all the admin scim token related methods that the Terraform
// Enterprise API supports
//
// TFE API docs: https://developer.hashicorp.com/terraform/enterprise/api-docs/admin/scim-tokens
type AdminSCIMTokens interface {
	// List all admin scim tokens.
	List(ctx context.Context) (*AdminSCIMTokenList, error)

	// Create a admin scim token.
	Create(ctx context.Context) (*AdminSCIMToken, error)

	// Create a admin scim token with options.
	CreateWithOptions(ctx context.Context, options AdminSCIMTokenCreateOptions) (*AdminSCIMToken, error)

	// Read a admin scim token by its ID.
	ReadByID(ctx context.Context, scimTokenID string) (*AdminSCIMToken, error)

	// Delete a admin scim token.
	Delete(ctx context.Context, scimTokenID string) error
}

// adminSCIMTokens implements AdminSCIMTokens
type adminSCIMTokens struct {
	client *Client
}

// AdminSCIMTokenList represents a list of admin scim tokens
type AdminSCIMTokenList struct {
	Items []*AdminSCIMToken
}

// AdminSCIMToken represents a Terraform Enterprise scim token.
type AdminSCIMToken struct {
	ID          string    `jsonapi:"primary,authentication-tokens"`
	CreatedAt   time.Time `jsonapi:"attr,created-at,iso8601"`
	LastUsedAt  time.Time `jsonapi:"attr,last-used-at,iso8601"`
	ExpiredAt   time.Time `jsonapi:"attr,expired-at,iso8601"`
	Description string    `jsonapi:"attr,description"`
	Token       string    `jsonapi:"attr,token,omitempty"`
}

// AdminSCIMTokenCreateOptions represents the options for creating a admin scim token
type AdminSCIMTokenCreateOptions struct {

	// Optional: An optional human-readable description of the token's purpose
	// (for example, Okta SCIM Integration).
	Description *string `jsonapi:"attr,description,omitempty"`

	// Optional: Optional ISO-8601 timestamp for token expiration.
	// Defaults to 365 days in the future. Must be between 29 and 365 days in the future.
	ExpiredAt *time.Time `jsonapi:"attr,expired-at,iso8601,omitempty"`
}

// List all admin scim tokens.
func (a *adminSCIMTokens) List(ctx context.Context) (*AdminSCIMTokenList, error) {
	req, err := a.client.NewRequest("GET", "admin/scim-tokens", nil)
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

// Create a admin scim token.
func (a *adminSCIMTokens) Create(ctx context.Context) (*AdminSCIMToken, error) {
	return a.CreateWithOptions(ctx, AdminSCIMTokenCreateOptions{})
}

// Create a admin scim token with options.
func (a *adminSCIMTokens) CreateWithOptions(ctx context.Context, options AdminSCIMTokenCreateOptions) (*AdminSCIMToken, error) {
	req, err := a.client.NewRequest("POST", "admin/scim-tokens", &options)
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

// Read a admin scim token by its ID.
func (a *adminSCIMTokens) ReadByID(ctx context.Context, scimTokenID string) (*AdminSCIMToken, error) {
	if !validStringID(&scimTokenID) {
		return nil, ErrInvalidTokenID
	}
	u := fmt.Sprintf("admin/scim-tokens/%s", url.PathEscape(scimTokenID))
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

// Delete a admin scim token.
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
