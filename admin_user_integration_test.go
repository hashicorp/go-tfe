//go:build integration
// +build integration

package tfe

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAdminUsers_List(t *testing.T) {
	skipIfCloud(t)

	client := testClient(t)
	ctx := context.Background()

	currentUser, err := client.Users.ReadCurrent(ctx)
	assert.NoError(t, err)

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
			Query: String(currentUser.Username),
		})
		require.NoError(t, err)
		assert.Equal(t, currentUser.ID, ul.Items[0].ID)
		assert.Equal(t, 1, ul.CurrentPage)
		assert.Equal(t, true, ul.TotalCount == 1)

		member, memberCleanup := createOrganizationMembership(t, client, org)
		defer memberCleanup()

		ul, err = client.Admin.Users.List(ctx, &AdminUserListOptions{
			Query: String(member.User.Email),
		})
		require.NoError(t, err)
		assert.Equal(t, member.User.Email, ul.Items[0].Email)
		assert.Equal(t, 1, ul.CurrentPage)
		assert.Equal(t, true, ul.TotalCount == 1)
	})

	t.Run("with organization included", func(t *testing.T) {
		ul, err := client.Admin.Users.List(ctx, &AdminUserListOptions{
			Include: &([]AdminUserIncludeOps{AdminUserOrgs}),
		})

		assert.NoError(t, err)

		assert.NotEmpty(t, ul.Items)
		assert.NotNil(t, ul.Items[0].Organizations)
		assert.NotEmpty(t, ul.Items[0].Organizations[0].Name)
	})

	t.Run("filter by admin", func(t *testing.T) {
		ul, err := client.Admin.Users.List(ctx, &AdminUserListOptions{
			Administrators: String("true"),
		})

		assert.NoError(t, err)
		assert.NotEmpty(t, ul.Items)
		assert.NotNil(t, ul.Items[0])
		// We use this `includesEmail` helper function because throughout
		// the tests, there could be multiple admins, depending on the
		// ordering of the test runs.
		assert.Equal(t, true, includesEmail(currentUser.Email, ul.Items))
	})
}

func TestAdminUsers_Delete(t *testing.T) {
	skipIfCloud(t)

	client := testClient(t)
	ctx := context.Background()

	org, orgCleanup := createOrganization(t, client)
	defer orgCleanup()

	t.Run("an existing user", func(t *testing.T) {
		// Avoid the member cleanup function because the user
		// gets deleted below.
		member, _ := createOrganizationMembership(t, client, org)

		ul, err := client.Admin.Users.List(ctx, &AdminUserListOptions{
			Query: String(member.User.Email),
		})
		require.NoError(t, err)
		assert.Equal(t, member.User.Email, ul.Items[0].Email)
		assert.Equal(t, 1, ul.CurrentPage)
		assert.Equal(t, true, ul.TotalCount == 1)

		err = client.Admin.Users.Delete(ctx, member.User.ID)
		require.NoError(t, err)

		ul, err = client.Admin.Users.List(ctx, &AdminUserListOptions{
			Query: String(member.User.Email),
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
	skipIfCloud(t)

	client := testClient(t)
	ctx := context.Background()

	org, orgCleanup := createOrganization(t, client)
	defer orgCleanup()

	member, memberCleanup := createOrganizationMembership(t, client, org)
	defer memberCleanup()

	if !member.User.TwoFactor.Enabled {
		t.Skip("User does not have 2FA enalbed. Skiping")
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
