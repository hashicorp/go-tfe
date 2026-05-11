// Copyright IBM Corp. 2018, 2025
// SPDX-License-Identifier: MPL-2.0

package tfe

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUsersReadCurrent(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	u, err := client.Users.ReadCurrent(ctx)
	require.NoError(t, err)

	assert.NotEmpty(t, u.ID)
	assert.NotEmpty(t, u.AvatarURL)
	assert.NotEmpty(t, u.Username)

	t.Run("two factor options are decoded", func(t *testing.T) {
		assert.NotNil(t, u.TwoFactor)
	})

	t.Run("permissions are decoded", func(t *testing.T) {
		assert.NotNil(t, u.Permissions)
	})

	t.Run("with no scim attributes", func(t *testing.T) {
		skipUnlessEnterprise(t)
		// When SCIM is disabled, the API omits all SCIM attributes from the response.
		assert.Nil(t, u.IsSCIMManaged)
		assert.Nil(t, u.SCIMUsername)
		assert.Nil(t, u.SCIMUpdatedAt)
	})

	t.Run("with scim attributes", func(t *testing.T) {
		skipUnlessEnterprise(t)
		enableSCIM(ctx, t, client, true)
		t.Cleanup(func() {
			enableSCIM(ctx, t, client, false)
		})

		user, err := client.Users.ReadCurrent(ctx)
		require.NoError(t, err)

		// The current user (test runner) is not a SCIM-managed user, so we can
		// only verify that IsSCIMManaged is populated (and false) when SCIM is
		// enabled. Verifying the SCIM-managed path requires authenticating as
		// the SCIM-provisioned user, which isn't possible here. See
		// TestAdminUsers_List/with_scim_attributes for coverage of a
		// SCIM-managed user via the admin API.
		assert.NotNil(t, user.IsSCIMManaged)
		assert.False(t, *user.IsSCIMManaged)
		assert.Nil(t, user.SCIMUsername)
		assert.Nil(t, user.SCIMUpdatedAt)
	})
}

func TestUsersUpdate(t *testing.T) {
	t.Parallel()
	client := testClient(t)
	ctx := context.Background()

	uTest, err := client.Users.ReadCurrent(ctx)
	require.NoError(t, err)

	// Make sure we reset the current user when we're done.
	defer func() {
		_, err := client.Users.UpdateCurrent(ctx, UserUpdateOptions{
			Email:    String(uTest.Email),
			Username: String(uTest.Username),
		})
		if err != nil {
			t.Logf("Error updating user: %s", err)
		}
	}()

	t.Run("without any options", func(t *testing.T) {
		_, err := client.Users.UpdateCurrent(ctx, UserUpdateOptions{})
		require.NoError(t, err)

		u, err := client.Users.ReadCurrent(ctx)
		require.NoError(t, err)
		assert.Equal(t, u, uTest)
	})

	t.Run("with a new username", func(t *testing.T) {
		_, err := client.Users.UpdateCurrent(ctx, UserUpdateOptions{
			Username: String("NewTestUsername"),
		})
		require.NoError(t, err)

		u, err := client.Users.ReadCurrent(ctx)
		require.NoError(t, err)
		assert.Equal(t, "NewTestUsername", u.Username)
	})

	t.Run("with a new email address", func(t *testing.T) {
		_, err := client.Users.UpdateCurrent(ctx, UserUpdateOptions{
			Email: String("newtestemail@hashicorp.com"),
		})
		require.NoError(t, err)

		u, err := client.Users.ReadCurrent(ctx)

		email := ""
		if u.UnconfirmedEmail != "" {
			email = u.UnconfirmedEmail
		} else if u.Email != "" {
			email = u.Email
		} else {
			t.Fatalf("cannot test with user %q because both email and unconfirmed email are empty", u.ID)
		}

		require.NoError(t, err)
		assert.Equal(t, "newtestemail@hashicorp.com", email)
	})

	t.Run("with invalid email address", func(t *testing.T) {
		u, err := client.Users.UpdateCurrent(ctx, UserUpdateOptions{
			Email: String("notamailaddress"),
		})
		assert.Nil(t, u)
		assert.Error(t, err)
	})
}
