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
		ws, err := client.Workspaces.List(orgTest.Name, WorkspaceListOptions{})
		require.NoError(t, err)
		assert.Contains(t, ws, wTest1)
		assert.Contains(t, ws, wTest2)
	})

	t.Run("with list options", func(t *testing.T) {
		// Request a page number which is out of range. The result should
		// be successful, but return no results if the paging options are
		// properly passed along.
		ws, err := client.Workspaces.List(orgTest.Name, WorkspaceListOptions{
			ListOptions: ListOptions{
				PageNumber: 999,
				PageSize:   100,
			},
		})
		require.NoError(t, err)
		assert.Empty(t, ws)
	})

	t.Run("without a valid organization", func(t *testing.T) {
		ws, err := client.Workspaces.List(badIdentifier, WorkspaceListOptions{})
		assert.Nil(t, ws)
		assert.EqualError(t, err, "Invalid value for organization")
	})
}

func TestWorkspacesCreate(t *testing.T) {
	client := testClient(t)

	orgTest, orgTestCleanup := createOrganization(t, client)
	defer orgTestCleanup()

	t.Run("with valid options", func(t *testing.T) {
		options := WorkspaceCreateOptions{
			Name:             String("foo"),
			AutoApply:        Bool(true),
			TerraformVersion: String("0.11.0"),
			WorkingDirectory: String("bar/"),
		}

		w, err := client.Workspaces.Create(orgTest.Name, options)
		require.NoError(t, err)

		// Get a refreshed view from the API.
		refreshed, err := client.Workspaces.Retrieve(orgTest.Name, *options.Name)
		require.NoError(t, err)

		for _, item := range []*Workspace{
			w,
			refreshed,
		} {
			assert.NotEmpty(t, item.ID)
			assert.Equal(t, *options.Name, item.Name)
			assert.Equal(t, *options.AutoApply, item.AutoApply)
			assert.Equal(t, *options.WorkingDirectory, item.WorkingDirectory)
			assert.Equal(t, *options.TerraformVersion, item.TerraformVersion)
		}
	})

	t.Run("when options is missing name", func(t *testing.T) {
		w, err := client.Workspaces.Create("foo", WorkspaceCreateOptions{})
		assert.Nil(t, w)
		assert.EqualError(t, err, "Name is required")
	})

	t.Run("when options has an invalid name", func(t *testing.T) {
		w, err := client.Workspaces.Create("foo", WorkspaceCreateOptions{
			Name: String(badIdentifier),
		})
		assert.Nil(t, w)
		assert.EqualError(t, err, "Invalid value for name")
	})

	t.Run("when options has an invalid organization", func(t *testing.T) {
		w, err := client.Workspaces.Create(badIdentifier, WorkspaceCreateOptions{
			Name: String("foo"),
		})
		assert.Nil(t, w)
		assert.EqualError(t, err, "Invalid value for organization")
	})

	t.Run("when an error is returned from the api", func(t *testing.T) {
		w, err := client.Workspaces.Create("bar", WorkspaceCreateOptions{
			Name:             String("bar"),
			TerraformVersion: String("nonexisting"),
		})
		assert.Nil(t, w)
		assert.Error(t, err)
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
		require.NoError(t, err)
		assert.Equal(t, wTest, w)

		t.Run("permissions are properly decoded", func(t *testing.T) {
			assert.True(t, w.Permissions.CanDestroy)
		})

		t.Run("relationships are properly decoded", func(t *testing.T) {
			assert.Equal(t, orgTest.Name, w.Organization.Name)
		})

		t.Run("timestamps are properly decoded", func(t *testing.T) {
			assert.NotEmpty(t, w.CreatedAt)
		})
	})

	t.Run("when the workspace does not exist", func(t *testing.T) {
		w, err := client.Workspaces.Retrieve(orgTest.Name, "nonexisting")
		assert.Nil(t, w)
		assert.Error(t, err)
	})

	t.Run("when the organization does not exist", func(t *testing.T) {
		w, err := client.Workspaces.Retrieve("nonexisting", "nonexisting")
		assert.Nil(t, w)
		assert.Error(t, err)
	})

	t.Run("without a valid organization", func(t *testing.T) {
		w, err := client.Workspaces.Retrieve(badIdentifier, wTest.Name)
		assert.Nil(t, w)
		assert.EqualError(t, err, "Invalid value for organization")
	})

	t.Run("without a valid workspace", func(t *testing.T) {
		w, err := client.Workspaces.Retrieve(orgTest.Name, badIdentifier)
		assert.Nil(t, w)
		assert.EqualError(t, err, "Invalid value for workspace")
	})
}

func TestWorkspacesUpdate(t *testing.T) {
	client := testClient(t)

	orgTest, orgTestCleanup := createOrganization(t, client)
	defer orgTestCleanup()

	wTest, wTestCleanup := createWorkspace(t, client, orgTest)

	t.Run("when updating a subset of values", func(t *testing.T) {
		options := WorkspaceUpdateOptions{
			Name:             String(wTest.Name),
			TerraformVersion: String("0.10.0"),
		}

		wAfter, err := client.Workspaces.Update(orgTest.Name, wTest.Name, options)
		if err != nil {
			wTestCleanup()
		}
		require.NoError(t, err)

		assert.Equal(t, wTest.Name, wAfter.Name)
		assert.Equal(t, wTest.AutoApply, wAfter.AutoApply)
		assert.Equal(t, wTest.WorkingDirectory, wAfter.WorkingDirectory)
		assert.NotEqual(t, wTest.TerraformVersion, wAfter.TerraformVersion)
	})

	t.Run("with valid options", func(t *testing.T) {
		options := WorkspaceUpdateOptions{
			Name:             String(randomString(t)),
			AutoApply:        Bool(false),
			TerraformVersion: String("0.11.1"),
			WorkingDirectory: String("baz/"),
		}

		w, err := client.Workspaces.Update(orgTest.Name, wTest.Name, options)
		if err != nil {
			wTestCleanup()
		}
		require.NoError(t, err)

		// Get a refreshed view of the workspace from the API
		refreshed, err := client.Workspaces.Retrieve(orgTest.Name, *options.Name)
		require.NoError(t, err)

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
		w, err := client.Workspaces.Update(orgTest.Name, wTest.Name, WorkspaceUpdateOptions{
			TerraformVersion: String("nonexisting"),
		})
		assert.Nil(t, w)
		assert.Error(t, err)
	})

	t.Run("when options has an invalid name", func(t *testing.T) {
		w, err := client.Workspaces.Update(orgTest.Name, badIdentifier, WorkspaceUpdateOptions{})
		assert.Nil(t, w)
		assert.EqualError(t, err, "Invalid value for workspace")
	})

	t.Run("when options has an invalid organization", func(t *testing.T) {
		w, err := client.Workspaces.Update(badIdentifier, wTest.Name, WorkspaceUpdateOptions{})
		assert.Nil(t, w)
		assert.EqualError(t, err, "Invalid value for organization")
	})
}

func TestWorkspacesDelete(t *testing.T) {
	client := testClient(t)

	orgTest, orgTestCleanup := createOrganization(t, client)
	defer orgTestCleanup()

	wTest, _ := createWorkspace(t, client, orgTest)

	t.Run("with valid options", func(t *testing.T) {
		err := client.Workspaces.Delete(orgTest.Name, wTest.Name)
		require.NoError(t, err)

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

func TestWorkspacesLock(t *testing.T) {
	client := testClient(t)

	orgTest, orgTestCleanup := createOrganization(t, client)
	defer orgTestCleanup()

	wTest, _ := createWorkspace(t, client, orgTest)

	t.Run("with valid options", func(t *testing.T) {
		w, err := client.Workspaces.Lock(wTest.ID, WorkspaceLockOptions{})
		require.NoError(t, err)
		assert.True(t, w.Locked)
	})

	t.Run("when workspace is already locked", func(t *testing.T) {
		w, err := client.Workspaces.Lock(wTest.ID, WorkspaceLockOptions{})
		require.NoError(t, err)
		assert.True(t, w.Locked)
	})

	t.Run("without a valid workspace ID", func(t *testing.T) {
		w, err := client.Workspaces.Lock(badIdentifier, WorkspaceLockOptions{})
		assert.Nil(t, w)
		assert.EqualError(t, err, "Invalid value for workspace ID")
	})
}

func TestWorkspacesUnlock(t *testing.T) {
	client := testClient(t)

	orgTest, orgTestCleanup := createOrganization(t, client)
	defer orgTestCleanup()

	wTest, _ := createWorkspace(t, client, orgTest)

	w, err := client.Workspaces.Lock(wTest.ID, WorkspaceLockOptions{})
	if err != nil {
		orgTestCleanup()
	}
	require.NoError(t, err)
	require.True(t, w.Locked)

	t.Run("with valid options", func(t *testing.T) {
		w, err := client.Workspaces.Unlock(wTest.ID)
		require.NoError(t, err)
		assert.False(t, w.Locked)
	})

	t.Run("when workspace is already locked", func(t *testing.T) {
		w, err := client.Workspaces.Unlock(wTest.ID)
		require.NoError(t, err)
		assert.False(t, w.Locked)
	})

	t.Run("without a valid workspace ID", func(t *testing.T) {
		w, err := client.Workspaces.Unlock(badIdentifier)
		assert.Nil(t, w)
		assert.EqualError(t, err, "Invalid value for workspace ID")
	})
}

func TestWorkspacesAssignSSHKey(t *testing.T) {
	client := testClient(t)

	orgTest, orgTestCleanup := createOrganization(t, client)
	defer orgTestCleanup()

	wTest, wTestCleanup := createWorkspace(t, client, orgTest)
	defer wTestCleanup()

	sshKeyTest, sshKeyTestCleanup := createSSHKey(t, client, orgTest)
	defer sshKeyTestCleanup()

	t.Run("with valid options", func(t *testing.T) {
		w, err := client.Workspaces.AssignSSHKey(wTest.ID, WorkspaceAssignSSHKeyOptions{
			SSHKeyID: String(sshKeyTest.ID),
		})
		require.NoError(t, err)
		require.NotNil(t, w.SSHKey)
		assert.Equal(t, w.SSHKey.ID, sshKeyTest.ID)
	})

	t.Run("without an SSH key ID", func(t *testing.T) {
		w, err := client.Workspaces.AssignSSHKey(wTest.ID, WorkspaceAssignSSHKeyOptions{})
		assert.Nil(t, w)
		assert.EqualError(t, err, "SSH key ID is required")
	})

	t.Run("without a valid SSH key ID", func(t *testing.T) {
		w, err := client.Workspaces.AssignSSHKey(wTest.ID, WorkspaceAssignSSHKeyOptions{
			SSHKeyID: String(badIdentifier),
		})
		assert.Nil(t, w)
		assert.EqualError(t, err, "Invalid value for SSH key ID")
	})

	t.Run("without a valid workspace ID", func(t *testing.T) {
		w, err := client.Workspaces.AssignSSHKey(badIdentifier, WorkspaceAssignSSHKeyOptions{
			SSHKeyID: String(sshKeyTest.ID),
		})
		assert.Nil(t, w)
		assert.EqualError(t, err, "Invalid value for workspace ID")
	})
}

func TestWorkspacesUnassignSSHKey(t *testing.T) {
	client := testClient(t)

	orgTest, orgTestCleanup := createOrganization(t, client)
	defer orgTestCleanup()

	wTest, wTestCleanup := createWorkspace(t, client, orgTest)
	defer wTestCleanup()

	sshKeyTest, sshKeyTestCleanup := createSSHKey(t, client, orgTest)
	defer sshKeyTestCleanup()

	w, err := client.Workspaces.AssignSSHKey(wTest.ID, WorkspaceAssignSSHKeyOptions{
		SSHKeyID: String(sshKeyTest.ID),
	})
	if err != nil {
		orgTestCleanup()
	}
	require.NoError(t, err)
	require.NotNil(t, w.SSHKey)
	require.Equal(t, w.SSHKey.ID, sshKeyTest.ID)

	t.Run("with valid options", func(t *testing.T) {
		w, err := client.Workspaces.UnassignSSHKey(wTest.ID)
		assert.Nil(t, err)
		assert.Nil(t, w.SSHKey)
	})

	t.Run("without a valid workspace ID", func(t *testing.T) {
		w, err := client.Workspaces.UnassignSSHKey(badIdentifier)
		assert.Nil(t, w)
		assert.EqualError(t, err, "Invalid value for workspace ID")
	})
}
