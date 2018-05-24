package tfe

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWorkspacesList(t *testing.T) {
	client := testClient(t)

	orgTest, orgTestCleanup := createOrganization(t, client)
	defer orgTestCleanup()

	wTest1, wTest1Cleanup := createWorkspace(t, client, orgTest)
	defer wTest1Cleanup()
	wTest2, wTest2Cleanup := createWorkspace(t, client, orgTest)
	defer wTest2Cleanup()

	t.Run("without list options", func(t *testing.T) {
		ws, err := client.Workspaces.List(orgTest.Name, nil)
		require.Nil(t, err)

		assert.Contains(t, ws, wTest1)
		assert.Contains(t, ws, wTest2)
	})

	t.Run("with list options", func(t *testing.T) {
		// Request a page number which is out of range. The result should
		// be successful, but return no results if the paging options are
		// properly passed along.
		ws, err := client.Workspaces.List(orgTest.Name, &ListWorkspacesOptions{
			ListOptions: ListOptions{
				PageNumber: 999,
				PageSize:   100,
			},
		})
		require.Nil(t, err)

		assert.Equal(t, 0, len(ws))
	})
}

func TestWorkspacesCreate(t *testing.T) {
	client := testClient(t)

	orgTest, orgTestCleanup := createOrganization(t, client)
	defer orgTestCleanup()

	t.Run("with valid options", func(t *testing.T) {
		options := &CreateWorkspaceOptions{
			Name:             String("foo"),
			AutoApply:        Bool(true),
			TerraformVersion: String("0.11.0"),
			WorkingDirectory: String("bar/"),
		}

		w, err := client.Workspaces.Create(orgTest.Name, options)
		require.Nil(t, err)

		// Get a refreshed view from the API.
		refreshed, err := client.Workspaces.Retrieve(orgTest.Name, *options.Name)
		require.Nil(t, err)

		for _, item := range []*Workspace{
			w,
			refreshed,
		} {
			assert.NotNil(t, w.ID)
			assert.Equal(t, *options.Name, item.Name)
			assert.Equal(t, *options.AutoApply, item.AutoApply)
			assert.Equal(t, *options.WorkingDirectory, item.WorkingDirectory)
			assert.Equal(t, *options.TerraformVersion, item.TerraformVersion)
		}
	})

	t.Run("when options is missing name", func(t *testing.T) {
		w, err := client.Workspaces.Create("foo", &CreateWorkspaceOptions{})
		assert.EqualError(t, err, "Invalid value for Name")
		assert.Nil(t, w)
	})

	t.Run("when options has an invalid name", func(t *testing.T) {
		w, err := client.Workspaces.Create("foo", &CreateWorkspaceOptions{
			Name: String(badIdentifier),
		})
		assert.EqualError(t, err, "Invalid value for Name")
		assert.Nil(t, w)
	})

	t.Run("when options has an invalid organization", func(t *testing.T) {
		w, err := client.Workspaces.Create(badIdentifier, &CreateWorkspaceOptions{
			Name: String("foo"),
		})
		assert.EqualError(t, err, "Invalid value for organization")
		assert.Nil(t, w)
	})

	t.Run("when an error is returned from the api", func(t *testing.T) {
		w, err := client.Workspaces.Create("bar", &CreateWorkspaceOptions{
			Name:             String("bar"),
			TerraformVersion: String("nonexisting"),
		})
		assert.NotNil(t, err)
		assert.Nil(t, w)
	})
}

func TestWorkspacesRetrieve(t *testing.T) {
	client := testClient(t)

	orgTest, orgTestCleanup := createOrganization(t, client)
	defer orgTestCleanup()

	wTest, wTestCleanup := createWorkspace(t, client, orgTest)
	defer wTestCleanup()

	t.Run("when the workspace exists", func(t *testing.T) {
		w, err := client.Workspaces.Retrieve(orgTest.Name, wTest.Name)
		require.Nil(t, err)
		assert.Equal(t, wTest, w)

		t.Run("permissions are properly decoded", func(t *testing.T) {
			assert.True(t, w.Permissions.CanDestroy)
		})

		t.Run("relationships are properly decoded", func(t *testing.T) {
			assert.Equal(t, orgTest.Name, w.Organization.Name)
		})

		t.Run("timestamps are properly decoded", func(t *testing.T) {
			assert.False(t, w.CreatedAt.IsZero())
		})
	})

	t.Run("when the workspace does not exist", func(t *testing.T) {
		w, err := client.Workspaces.Retrieve(orgTest.Name, "nonexisting")
		assert.NotNil(t, err)
		assert.Nil(t, w)
	})

	t.Run("when the organization does not exist", func(t *testing.T) {
		w, err := client.Workspaces.Retrieve("nonexisting", "nonexisting")
		assert.NotNil(t, err)
		assert.Nil(t, w)
	})
}

func TestWorkspacesUpdate(t *testing.T) {
	client := testClient(t)

	orgTest, orgTestCleanup := createOrganization(t, client)
	defer orgTestCleanup()

	wTest, wTestCleanup := createWorkspace(t, client, orgTest)

	t.Run("when updating a subset of values", func(t *testing.T) {
		wBefore, err := client.Workspaces.Retrieve(orgTest.Name, wTest.Name)
		require.Nil(t, err)

		options := &UpdateWorkspaceOptions{
			Name:             String(wTest.Name),
			TerraformVersion: String("0.10.0"),
		}

		wAfter, err := client.Workspaces.Update(orgTest.Name, wTest.Name, options)
		if err != nil {
			wTestCleanup()
		}
		require.Nil(t, err)

		assert.Equal(t, wBefore.Name, wAfter.Name)
		assert.Equal(t, wBefore.AutoApply, wAfter.AutoApply)
		assert.Equal(t, wBefore.WorkingDirectory, wAfter.WorkingDirectory)
		assert.NotEqual(t, wBefore.TerraformVersion, wAfter.TerraformVersion)
	})

	t.Run("with valid options", func(t *testing.T) {
		options := &UpdateWorkspaceOptions{
			Name:             String(randomString(t)),
			AutoApply:        Bool(false),
			TerraformVersion: String("0.11.1"),
			WorkingDirectory: String("baz/"),
		}

		w, err := client.Workspaces.Update(orgTest.Name, wTest.Name, options)
		if err != nil {
			wTestCleanup()
		}
		require.Nil(t, err)

		// Get a refreshed view of the workspace from the API
		refreshed, err := client.Workspaces.Retrieve(orgTest.Name, *options.Name)
		require.Nil(t, err)

		for _, item := range []*Workspace{
			w,
			refreshed,
		} {
			assert.Equal(t, *options.Name, item.Name)
			assert.Equal(t, *options.AutoApply, item.AutoApply)
			assert.Equal(t, *options.TerraformVersion, item.TerraformVersion)
			assert.Equal(t, *options.WorkingDirectory, item.WorkingDirectory)
		}
	})

	t.Run("when an error is returned from the api", func(t *testing.T) {
		w, err := client.Workspaces.Update(orgTest.Name, wTest.Name, &UpdateWorkspaceOptions{
			TerraformVersion: String("nonexisting"),
		})
		assert.NotNil(t, err)
		assert.Nil(t, w)
	})

	t.Run("when options has an invalid name", func(t *testing.T) {
		w, err := client.Workspaces.Update(orgTest.Name, badIdentifier, nil)
		assert.EqualError(t, err, "Invalid value for workspace")
		assert.Nil(t, w)
	})

	t.Run("when options has an invalid organization", func(t *testing.T) {
		w, err := client.Workspaces.Update(badIdentifier, wTest.Name, nil)
		assert.EqualError(t, err, "Invalid value for organization")
		assert.Nil(t, w)
	})
}

func TestWorkspacesDelete(t *testing.T) {
	client := testClient(t)

	orgTest, orgTestCleanup := createOrganization(t, client)
	defer orgTestCleanup()

	wTest, _ := createWorkspace(t, client, orgTest)

	t.Run("with valid options", func(t *testing.T) {
		err := client.Workspaces.Delete(orgTest.Name, wTest.Name)
		require.Nil(t, err)

		// Try loading the workspace - it should fail.
		_, err = client.Workspaces.Retrieve(orgTest.Name, wTest.Name)
		assert.EqualError(t, err, "Resource not found")
	})

	t.Run("when organization is invalid", func(t *testing.T) {
		err := client.Workspaces.Delete(badIdentifier, wTest.Name)
		assert.EqualError(t, err, "Invalid value for organization")
	})

	t.Run("when workspace is invalid", func(t *testing.T) {
		err := client.Workspaces.Delete(orgTest.Name, badIdentifier)
		assert.EqualError(t, err, "Invalid value for workspace")
	})
}
