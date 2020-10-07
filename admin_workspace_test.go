package tfe

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAdminWorkspacesList(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	defer orgTestCleanup()

	wTest1, wTest1Cleanup := createWorkspace(t, client, orgTest)
	defer wTest1Cleanup()
	wTest2, wTest2Cleanup := createWorkspace(t, client, orgTest)
	defer wTest2Cleanup()

	t.Run("without list options", func(t *testing.T) {
		wl, err := client.AdminWorkspaces.List(ctx, AdminWorkspacesListOptions{})
		require.NoError(t, err)
		assert.Contains(t, wl.Items, wTest1)
		assert.Contains(t, wl.Items, wTest2)
		assert.Equal(t, 1, wl.CurrentPage)
		assert.Equal(t, 2, wl.TotalCount)
	})

	t.Run("with list options", func(t *testing.T) {
		// Request a page number which is out of range. The result should
		// be successful, but return no results if the paging options are
		// properly passed along.
		wl, err := client.AdminWorkspaces.List(ctx, AdminWorkspacesListOptions{
			ListOptions: ListOptions{
				PageNumber: 999,
				PageSize:   100,
			},
		})
		require.NoError(t, err)
		assert.Empty(t, wl.Items)
		assert.Equal(t, 999, wl.CurrentPage)
		assert.Equal(t, 2, wl.TotalCount)
	})
}

func TestAdminWorkspacesRead(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	defer orgTestCleanup()

	wTest, wTestCleanup := createWorkspace(t, client, orgTest)
	defer wTestCleanup()

	t.Run("when the workspace exists", func(t *testing.T) {
		w, err := client.AdminWorkspaces.Read(ctx, wTest.ID)
		require.NoError(t, err)
		assert.Equal(t, wTest, w)

		t.Run("relationships are properly decoded", func(t *testing.T) {
			assert.Equal(t, orgTest.Name, w.Organization.Name)
		})
	})

	t.Run("when the workspace does not exist", func(t *testing.T) {
		w, err := client.AdminWorkspaces.Read(ctx, "nonexisting")
		assert.Nil(t, w)
		assert.Error(t, err)
	})

	t.Run("without a valid workspace ID", func(t *testing.T) {
		w, err := client.AdminWorkspaces.Read(ctx, badIdentifier)
		assert.Nil(t, w)
		assert.EqualError(t, err, "invalid value for workspace ID")
	})
}

func TestAdminWorkspacesDelete(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	defer orgTestCleanup()

	wTest, _ := createWorkspace(t, client, orgTest)

	t.Run("with valid options", func(t *testing.T) {
		err := client.AdminWorkspaces.Delete(ctx, wTest.ID)
		require.NoError(t, err)

		// Try loading the workspace - it should fail.
		_, err = client.AdminWorkspaces.Read(ctx, wTest.ID)
		assert.Equal(t, ErrResourceNotFound, err)
	})

	t.Run("without a valid workspace ID", func(t *testing.T) {
		err := client.AdminWorkspaces.Delete(ctx, badIdentifier)
		assert.EqualError(t, err, "invalid value for workspace ID")
	})
}
