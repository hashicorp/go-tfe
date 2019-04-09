package tfe

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAdminRunsList(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	wTest, wTestCleanup := createWorkspace(t, client, nil)
	defer wTestCleanup()

	rTest1, _ := createRun(t, client, wTest)
	rTest2, _ := createRun(t, client, wTest)

	t.Run("without list options", func(t *testing.T) {
		rl, err := client.AdminRuns.List(ctx, AdminRunListOptions{})
		require.NoError(t, err)

		found := []string{}
		for _, r := range rl.Items {
			found = append(found, r.ID)
		}

		assert.Contains(t, found, rTest1.ID)
		assert.Contains(t, found, rTest2.ID)
		assert.Equal(t, 1, rl.CurrentPage)
		assert.Equal(t, 2, rl.TotalCount)
	})

	t.Run("with list options", func(t *testing.T) {
		t.Skip("paging not supported yet in API")

		// Request a page number which is out of range. The result should
		// be successful, but return no results if the paging options are
		// properly passed along.
		rl, err := client.AdminRuns.List(ctx, AdminRunListOptions{
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
