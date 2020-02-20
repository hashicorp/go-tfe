package tfe

import (
	"context"
	"fmt"
)

// Compile-time proof of interface implementation.
var _ AdminUsers = (*adminUsers)(nil)

// AdminUsers Users Admin API contains endpoints to help site administrators manage
// user accounts.
//
// TFE API docs: https://www.terraform.io/docs/cloud/api/admin/users.html
type AdminUsers interface {
	// List all the users of the given installation.
	List(ctx context.Context, options AdminUsersListOptions) (*AdminUsersList, error)
}

// users implements Users.
type adminUsers struct {
	client *Client
}

// AdminUser represents a Terraform Enterprise user.
type AdminUser struct {
	ID               string          `jsonapi:"primary,users"`
	AvatarURL        string          `jsonapi:"attr,avatar-url"`
	Email            string          `jsonapi:"attr,email"`
	IsServiceAccount bool            `jsonapi:"attr,is-service-account"`
	TwoFactor        *AdminTwoFactor `jsonapi:"attr,two-factor"`
	UnconfirmedEmail string          `jsonapi:"attr,unconfirmed-email"`
	Username         string          `jsonapi:"attr,username"`
	V2Only           bool            `jsonapi:"attr,v2-only"`

	// Relations
	// AuthenticationTokens *AuthenticationTokens `jsonapi:"relation,authentication-tokens"`
	Organizations []*Organization `jsonapi:"relation,organizations"`
}

// AdminTwoFactor represents the organization permissions.
type AdminTwoFactor struct {
	Enabled  bool `json:"enabled"`
	Verified bool `json:"verified"`
}

// AdminUsersList represents a list of users.
type AdminUsersList struct {
	*Pagination
	Items []*AdminUser
}

// AdminUsersListOptions represents the options for listing users.
type AdminUsersListOptions struct {
	ListOptions

	Include string `url:"include"`
}

// List all the users of the terraform enterprise installation.
func (s *adminUsers) List(ctx context.Context, options AdminUsersListOptions) (*AdminUsersList, error) {
	u := fmt.Sprintf("admin/users")
	req, err := s.client.newRequest("GET", u, &options)
	if err != nil {
		return nil, err
	}

	rl := &AdminUsersList{}
	err = s.client.do(ctx, req, rl)
	if err != nil {
		return nil, err
	}

	return rl, nil
}
