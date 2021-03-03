package tfe

import (
	"context"
	"fmt"
	"net/url"
)

// Compile-time proof of interface implementation.
var _ AdminUsers = (*adminUsers)(nil)

// AdminUsers describes all the admin user related methods that the Terraform
// Enterprise  API supports.
// It contains endpoints to help site administrators manage their users.
//
// TFE API docs: https://www.terraform.io/docs/cloud/api/admin/users.html
type AdminUsers interface {
	// List all the users of the given installation.
	List(ctx context.Context, options AdminUserListOptions) (*AdminUserList, error)

	// Delete a user by its ID.
	Delete(ctx context.Context, userID string) error

	//	// Suspend a user account by its ID.
	//	Suspend(ctx context.Context, userID string) (*AdminUser, error)
	//
	//	// Unsuspend a user account by its ID.
	//	Unsuspend(ctx context.Context, userID string) (*AdminUser, error)
	//
	//	// GrantAdminPrivlages to a user account by its ID.
	//	GrantAdminPrivlages(ctx context.Context, userID string) (*AdminUser, error)
	//
	//	// RevokeAdminPrivlages to a user account by its ID.
	//	RevokeAdminPrivlages(ctx context.Context, userID string) (*AdminUser, error)
	//
	//	// Disable2FA disables a user's two-factor authentication in the situation
	//	// where they have lost access to their device and recovery codes.
	//	Disable2FA(ctx context.Context, userID string) (*AdminUser, error)
	//
	//	// ImpersonateUser allows an admin to begin a new session as another user in
	//	// the system
	//	ImpersonateUser(ctx context.Context, userID string) (*AdminUser, error)
	//
	//	// UnimpersonateUser allows an admin to end an impersonationn session of
	//	// another user in the system
	//	UnimpersonateUser(ctx context.Context, userID string) (*AdminUser, error)
}

// adminUsers implements the AdminUsers interface.
type adminUsers struct {
	client *Client
}

type AdminUser struct {
	ID    string `jsonapi:"primary,users"`
	Email string `jsonapi:"attr,email"`

	// Relations
	Organizations []*Organization `jsonapi:"relation,organizations"`
}

// AdminUserList represents a list of users.
type AdminUserList struct {
	*Pagination
	Items []*AdminUser
}

// AdminUserListOptions represents the options for listing users.
// https://www.terraform.io/docs/cloud/api/admin/users.html#query-parameters
type AdminUserListOptions struct {
	ListOptions

	// A search query string. Users are searchable by username and email address.
	Query *string `url:"q,omitempty"`

	// Can be "true" or "false" to show only administrators or non-administrators.
	Administrators *string `url:"filter[admin]"`

	// Can be "true" or "false" to show only suspended users or users who are not suspended.
	SuspendedUsers *string `url:"filter[suspended]"`

	// A list of relations to include. See available resources
	// https://www.terraform.io/docs/cloud/api/admin/users.html#available-related-resources
	Include *string `url:"include"`
}

// List all user accounts in the Terraform Enterprise installation
func (s *adminUsers) List(ctx context.Context, options AdminUserListOptions) (*AdminUserList, error) {
	u := fmt.Sprintf("admin/users")
	req, err := s.client.newRequest("GET", u, &options)
	if err != nil {
		return nil, err
	}

	awl := &AdminUserList{}
	err = s.client.do(ctx, req, awl)
	if err != nil {
		return nil, err
	}

	return awl, nil
}

// Delete a user by its ID.
func (s *adminUsers) Delete(ctx context.Context, userID string) error {
	if !validStringID(&userID) {
		return ErrInvalidUserValue
	}

	u := fmt.Sprintf("admin/users/%s", url.QueryEscape(userID))
	req, err := s.client.newRequest("DELETE", u, nil)
	if err != nil {
		return err
	}

	return s.client.do(ctx, req, nil)
}
