package tfe

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAdminUsers_List(t *testing.T) {
	skipIfCloud(t)

	client := testClient(t)
	ctx := context.Background()

	currentUser, err := client.Users.ReadCurrent(ctx)
	fmt.Println(currentUser.ID)
	fmt.Println(currentUser.Email)
	assert.NoError(t, err)

	org, orgCleanup := createOrganization(t, client)
	defer orgCleanup()
	fmt.Println(org.Name)

	t.Run("without list options", func(t *testing.T) {
		ul, err := client.Admin.Users.List(ctx, AdminUserListOptions{})
		require.NoError(t, err)

		assert.NotEmpty(t, ul.Items)
	})

	t.Run("with list options", func(t *testing.T) {
		ul, err := client.Admin.Users.List(ctx, AdminUserListOptions{
			ListOptions: ListOptions{
				PageNumber: 999,
				PageSize:   100,
			},
		})
		require.NoError(t, err)
		// Out of range page number, so the items should be empty
		assert.Empty(t, ul.Items)
		assert.Equal(t, 999, ul.CurrentPage)

		ul, err = client.Admin.Users.List(ctx, AdminUserListOptions{
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
		ul, err := client.Admin.Users.List(ctx, AdminUserListOptions{
			Query: String(currentUser.Username),
		})
		require.NoError(t, err)
		assert.Equal(t, currentUser.ID, ul.Items[0].ID)
		assert.Equal(t, 1, ul.CurrentPage)
		assert.Equal(t, true, ul.TotalCount == 1)

		ul, err = client.Admin.Users.List(ctx, AdminUserListOptions{
			Query: String(currentUser.Email),
		})
		require.NoError(t, err)
		assert.Equal(t, currentUser.Email, ul.Items[0].Email)
		assert.Equal(t, 1, ul.CurrentPage)
		assert.Equal(t, true, ul.TotalCount == 1)
	})

	t.Run("with organization included", func(t *testing.T) {
		ul, err := client.Admin.Users.List(ctx, AdminUserListOptions{
			Include: String("organizations"),
		})

		assert.NoError(t, err)

		assert.NotEmpty(t, ul.Items)
		assert.NotNil(t, ul.Items[0].Organizations)
		assert.NotEmpty(t, ul.Items[0].Organizations[0].Name)
	})

	t.Run("filter by admin", func(t *testing.T) {
		ul, err := client.Admin.Users.List(ctx, AdminUserListOptions{
			Administrators: String("true"),
		})

		assert.NoError(t, err)
		assert.NotEmpty(t, ul.Items)
		assert.NotNil(t, ul.Items[0])
		assert.Equal(t, currentUser.Email, ul.Items[0].Email)
	})
}
