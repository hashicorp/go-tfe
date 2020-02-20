package tfe

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAdminUsersList(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	t.Run("without list options", func(t *testing.T) {
		rl, err := client.AdminUsers.List(ctx, AdminUsersListOptions{})
		require.NoError(t, err)

		u, err := client.Users.ReadCurrent(ctx)

		found := []string{}
		for _, r := range rl.Items {
			found = append(found, r.ID)
		}

		assert.Contains(t, found, u.ID)
		assert.Equal(t, 1, rl.CurrentPage)
		assert.Equal(t, 1, rl.TotalCount)
	})

	t.Run("with list options", func(t *testing.T) {
		// Request a page number which is out of range. The result should
		// be successful, but return no results if the paging options are
		// properly passed along.
		rl, err := client.AdminUsers.List(ctx, AdminUsersListOptions{
			ListOptions: ListOptions{
				PageNumber: 999,
				PageSize:   100,
			},
		})
		require.NoError(t, err)
		assert.Empty(t, rl.Items)
		assert.Equal(t, 999, rl.CurrentPage)
		assert.Equal(t, 2, rl.TotalCount)
	})
}
