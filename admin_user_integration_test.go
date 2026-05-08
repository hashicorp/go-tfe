// Copyright IBM Corp. 2018, 2025
// SPDX-License-Identifier: MPL-2.0

package tfe

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAdminUsers_List(t *testing.T) {
	t.Parallel()
	skipUnlessEnterprise(t)

	client := testClient(t)
	ctx := context.Background()

	currentUser, err := client.Users.ReadCurrent(ctx)
	require.NoError(t, err)

	org, orgCleanup := createOrganization(t, client)
	defer orgCleanup()

	t.Run("without list options", func(t *testing.T) {
		ul, err := client.Admin.Users.List(ctx, nil)
		require.NoError(t, err)

		assert.NotEmpty(t, ul.Items)
	})

	t.Run("with list options", func(t *testing.T) {
		ul, err := client.Admin.Users.List(ctx, &AdminUserListOptions{
			ListOptions: ListOptions{
				PageNumber: 999,
				PageSize:   100,
			},
		})

		require.NoError(t, err)
		// Out of range page number, so the items should be empty
		assert.Empty(t, ul.Items)
		assert.Equal(t, 999, ul.CurrentPage)

		ul, err = client.Admin.Users.List(ctx, &AdminUserListOptions{
			ListOptions: ListOptions{
				PageNumber: 1,
				PageSize:   100,
			},
		})
		require.NoError(t, err)
		assert.NotEmpty(t, ul.Items)
		assert.Equal(t, 1, ul.CurrentPage)
	})

	t.Run("query by username or email", func(t *testing.T) {
		ul, err := client.Admin.Users.List(ctx, &AdminUserListOptions{
			Query: "admin-security-maintenance",
		})
		require.NoError(t, err)
		assert.Equal(t, 1, ul.CurrentPage)
		assert.Equal(t, true, ul.TotalCount == 1)

		member, memberCleanup := createOrganizationMembership(t, client, org)
		defer memberCleanup()

		ul, err = client.Admin.Users.List(ctx, &AdminUserListOptions{
			Query: member.User.Email,
		})
		require.NoError(t, err)
		assert.Equal(t, member.User.Email, ul.Items[0].Email)
		assert.Equal(t, 1, ul.CurrentPage)
		assert.Equal(t, true, ul.TotalCount == 1)
	})

	t.Run("with organization included", func(t *testing.T) {
		ul, err := client.Admin.Users.List(ctx, &AdminUserListOptions{
			Include: []AdminUserIncludeOpt{AdminUserOrgs},
		})

		require.NoError(t, err)
		require.NotEmpty(t, ul.Items)
		require.NotEmpty(t, ul.Items[0].Organizations)

		assert.NotEmpty(t, ul.Items[0].Organizations[0].Name)
	})

	t.Run("filter by admin", func(t *testing.T) {
		ul, err := client.Admin.Users.List(ctx, &AdminUserListOptions{
			Administrators: "true",
		})

		require.NoError(t, err)
		require.NotEmpty(t, ul.Items)
		require.NotNil(t, ul.Items[0])
		// We use this `includesEmail` helper function because throughout
		// the tests, there could be multiple admins, depending on the
		// ordering of the test runs.
		assert.Equal(t, true, includesEmail(currentUser.Email, ul.Items))
	})

	t.Run("with scim attributes", func(t *testing.T) {
		skipUnlessEnterprise(t)
		enableSCIM(ctx, t, client, true)
		t.Cleanup(func() {
			enableSCIM(ctx, t, client, false)
		})

		scimToken, err := client.Admin.Settings.SCIM.Tokens.Create(ctx, "user integration test")
		require.NoError(t, err)
		require.NotNil(t, scimToken.Token)
		t.Cleanup(func() {
			err = client.Admin.Settings.SCIM.Tokens.Delete(ctx, scimToken.ID)
			require.NoError(t, err)
		})

		userSCIMID, username := createSCIMUser(ctx, t, client, scimToken.Token, "")
		t.Cleanup(func() {
			deleteSCIMUser(ctx, t, client, scimToken.Token, userSCIMID)
		})

		users, err := client.Admin.Users.List(ctx, &AdminUserListOptions{Query: username})
		require.NoError(t, err)
		require.NotEmpty(t, users.Items)

		var user *AdminUser
		for _, u := range users.Items {
			if u.SCIMUsername != nil && *u.SCIMUsername == username {
				user = u
				break
			}
		}
		require.NotNil(t, user, "expected to find SCIM user %q in admin users list", username)
		assert.Equal(t, username, *user.SCIMUsername)
		require.NotNil(t, user.IsSCIMManaged)
		assert.True(t, *user.IsSCIMManaged)
		require.NotNil(t, user.SCIMUpdatedAt)
		assert.WithinDuration(t, time.Now(), *user.SCIMUpdatedAt, 10*time.Second)
	})
}

func TestAdminUsers_Delete(t *testing.T) {
	t.Parallel()
	skipUnlessEnterprise(t)

	client := testClient(t)
	ctx := context.Background()

	org, orgCleanup := createOrganization(t, client)
	defer orgCleanup()

	t.Run("an existing user", func(t *testing.T) {
		// Avoid the member cleanup function because the user
		// gets deleted below.
		member, _ := createOrganizationMembership(t, client, org)

		ul, err := client.Admin.Users.List(ctx, &AdminUserListOptions{
			Query: member.User.Email,
		})
		require.NoError(t, err)
		assert.Equal(t, member.User.Email, ul.Items[0].Email)
		assert.Equal(t, 1, ul.CurrentPage)
		assert.Equal(t, true, ul.TotalCount == 1)

		err = client.Admin.Users.Delete(ctx, member.User.ID)
		require.NoError(t, err)

		ul, err = client.Admin.Users.List(ctx, &AdminUserListOptions{
			Query: member.User.Email,
		})
		require.NoError(t, err)
		assert.Empty(t, ul.Items)
		assert.Equal(t, 1, ul.CurrentPage)
		assert.Equal(t, 0, ul.TotalCount)
	})

	t.Run("an non-existing user", func(t *testing.T) {
		err := client.Admin.Users.Delete(ctx, "non-existing-user-id")
		require.Error(t, err)
	})
}

func TestAdminUsers_Disable2FA(t *testing.T) {
	t.Parallel()
	skipUnlessEnterprise(t)

	client := testClient(t)
	ctx := context.Background()

	org, orgCleanup := createOrganization(t, client)
	defer orgCleanup()

	member, memberCleanup := createOrganizationMembership(t, client, org)
	defer memberCleanup()

	if !member.User.TwoFactor.Enabled {
		t.Skip("User does not have 2FA enabled. Skipping")
	}
	user, err := client.Admin.Users.Disable2FA(ctx, member.User.ID)
	require.NoError(t, err)
	require.NotNil(t, user)
}

func includesEmail(email string, userList []*AdminUser) bool {
	for _, user := range userList {
		if user.Email == email {
			return true
		}
	}

	return false
}
