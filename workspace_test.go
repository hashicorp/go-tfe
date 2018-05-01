package tfe

import (
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListWorkspaces(t *testing.T) {
	client := testClient(t)

	org, orgCleanup := createOrganization(t, client)
	defer orgCleanup()

	ws1, ws1Cleanup := createWorkspace(t, client, org)
	defer ws1Cleanup()
	ws2, ws2Cleanup := createWorkspace(t, client, org)
	defer ws2Cleanup()

	t.Run("without list options", func(t *testing.T) {
		workspaces, err := client.ListWorkspaces(&ListWorkspacesInput{
			OrganizationName: org.Name,
		})
		require.Nil(t, err)

		expect := []*Workspace{ws1, ws2}

		// Sort to ensure we get a non-flaky comparison.
		sort.Stable(WorkspaceNameSort(expect))
		sort.Stable(WorkspaceNameSort(workspaces))

		assert.Equal(t, expect, workspaces)
	})

	t.Run("with list options", func(t *testing.T) {
		// Request a page number which is out of range. The result should
		// be successful, but return no results if the paging options are
		// properly passed along.
		workspaces, err := client.ListWorkspaces(&ListWorkspacesInput{
			OrganizationName: org.Name,
			ListOptions: ListOptions{
				PageNumber: 999,
				PageSize:   100,
			},
		})
		require.Nil(t, err)

		assert.Equal(t, 0, len(workspaces))
	})
}

func TestWorkspace(t *testing.T) {
	client := testClient(t)

	org, orgCleanup := createOrganization(t, client)
	defer orgCleanup()

	ws, wsCleanup := createWorkspace(t, client, org)
	defer wsCleanup()

	t.Run("when the workspace exists", func(t *testing.T) {
		result, err := client.Workspace(*org.Name, *ws.Name)
		require.Nil(t, err)
		assert.Equal(t, ws, result)

		t.Run("permissions are properly decoded", func(t *testing.T) {
			if !result.Permissions.Can("destroy") {
				t.Fatal("should be able to destroy")
			}
		})

		t.Run("relationships are properly decoded", func(t *testing.T) {
			assert.Equal(t, result.OrganizationName, org.Name)
		})

		t.Run("timestamps are properly decoded", func(t *testing.T) {
			assert.False(t, result.CreatedAt.IsZero())
		})
	})

	t.Run("when the workspace does not exist", func(t *testing.T) {
		result, err := client.Workspace(*org.Name, "nope")
		assert.NotNil(t, err)
		assert.Nil(t, result)
	})

	t.Run("when the organization does not exist", func(t *testing.T) {
		result, err := client.Workspace("nope", "nope")
		assert.NotNil(t, err)
		assert.Nil(t, result)
	})
}

func TestCreateWorkspace(t *testing.T) {
	client := testClient(t)

	org, orgCleanup := createOrganization(t, client)
	defer orgCleanup()

	t.Run("with valid input", func(t *testing.T) {
		input := &CreateWorkspaceInput{
			OrganizationName: org.Name,
			Name:             String("foo"),
			AutoApply:        Bool(true),
			TerraformVersion: String("0.11.0"),
			WorkingDirectory: String("bar/"),
		}

		output, err := client.CreateWorkspace(input)
		require.Nil(t, err)

		// Get a refreshed view from the API.
		refreshedWorkspace, err := client.Workspace(*org.Name, *input.Name)
		require.Nil(t, err)

		for _, result := range []*Workspace{
			output.Workspace,
			refreshedWorkspace,
		} {
			assert.NotNil(t, result.ID)
			assert.Equal(t, input.Name, result.Name)
			assert.Equal(t, input.AutoApply, result.AutoApply)
			assert.Equal(t, input.WorkingDirectory, result.WorkingDirectory)
			assert.Equal(t, input.TerraformVersion, result.TerraformVersion)
		}
	})

	t.Run("when input is missing organization", func(t *testing.T) {
		result, err := client.CreateWorkspace(&CreateWorkspaceInput{
			Name: String("foo"),
		})
		assert.EqualError(t, err, "Invalid value for OrganizationName")
		assert.Nil(t, result)
	})

	t.Run("when input is missing name", func(t *testing.T) {
		result, err := client.CreateWorkspace(&CreateWorkspaceInput{
			OrganizationName: String("foo"),
		})
		assert.EqualError(t, err, "Invalid value for Name")
		assert.Nil(t, result)
	})

	t.Run("when input has invalid name", func(t *testing.T) {
		result, err := client.CreateWorkspace(&CreateWorkspaceInput{
			OrganizationName: String("foo"),
			Name:             String("! / nope"),
		})
		assert.EqualError(t, err, "Invalid value for Name")
		assert.Nil(t, result)
	})

	t.Run("when input has invalid organization", func(t *testing.T) {
		result, err := client.CreateWorkspace(&CreateWorkspaceInput{
			OrganizationName: String("! / nope"),
			Name:             String("foo"),
		})
		assert.EqualError(t, err, "Invalid value for OrganizationName")
		assert.Nil(t, result)
	})

	t.Run("when an error is returned from the api", func(t *testing.T) {
		result, err := client.CreateWorkspace(&CreateWorkspaceInput{
			OrganizationName: org.Name,
			Name:             String("bar"),
			TerraformVersion: String("nope"),
		})
		assert.NotNil(t, err)
		assert.Nil(t, result)
	})
}

func TestModifyWorkspace(t *testing.T) {
	client := testClient(t)

	ws, wsCleanup := createWorkspace(t, client, nil)
	defer wsCleanup()

	t.Run("when updating a subset of values", func(t *testing.T) {
		before, err := client.Workspace(*ws.OrganizationName, *ws.Name)
		require.Nil(t, err)

		input := &ModifyWorkspaceInput{
			OrganizationName: ws.OrganizationName,
			Name:             ws.Name,
			TerraformVersion: String("0.10.0"),
		}

		output, err := client.ModifyWorkspace(input)
		require.Nil(t, err)

		after := output.Workspace
		assert.Equal(t, before.Name, after.Name)
		assert.Equal(t, before.AutoApply, after.AutoApply)
		assert.Equal(t, before.WorkingDirectory, after.WorkingDirectory)
		assert.NotEqual(t, before.TerraformVersion, after.TerraformVersion)
	})

	t.Run("with valid input", func(t *testing.T) {
		input := &ModifyWorkspaceInput{
			OrganizationName: ws.OrganizationName,
			Name:             ws.Name,
			Rename:           String(randomString(t)),
			AutoApply:        Bool(false),
			TerraformVersion: String("0.11.1"),
			WorkingDirectory: String("baz/"),
		}

		output, err := client.ModifyWorkspace(input)
		require.Nil(t, err)

		// Get a refreshed view of the workspace from the API
		refreshed, err := client.Workspace(*ws.OrganizationName, *input.Rename)
		require.Nil(t, err)

		for _, result := range []*Workspace{
			output.Workspace,
			refreshed,
		} {
			assert.Equal(t, result.Name, input.Rename)
			assert.Equal(t, result.AutoApply, input.AutoApply)
			assert.Equal(t, result.TerraformVersion, input.TerraformVersion)
			assert.Equal(t, result.WorkingDirectory, input.WorkingDirectory)
		}
	})

	t.Run("when input is missing organization", func(t *testing.T) {
		result, err := client.ModifyWorkspace(&ModifyWorkspaceInput{
			Name: String("foo"),
		})
		assert.EqualError(t, err, "Invalid value for OrganizationName")
		assert.Nil(t, result)
	})

	t.Run("when input is missing name", func(t *testing.T) {
		result, err := client.ModifyWorkspace(&ModifyWorkspaceInput{
			OrganizationName: ws.OrganizationName,
		})
		assert.EqualError(t, err, "Invalid value for Name")
		assert.Nil(t, result)
	})

	t.Run("when an error is returned from the api", func(t *testing.T) {
		result, err := client.ModifyWorkspace(&ModifyWorkspaceInput{
			OrganizationName: ws.OrganizationName,
			Name:             ws.Name,
			TerraformVersion: String("nope"),
		})
		assert.NotNil(t, err)
		assert.Nil(t, result)
	})

	t.Run("when input has invalid name", func(t *testing.T) {
		result, err := client.ModifyWorkspace(&ModifyWorkspaceInput{
			OrganizationName: ws.OrganizationName,
			Name:             String("! / nope"),
		})
		assert.EqualError(t, err, "Invalid value for Name")
		assert.Nil(t, result)
	})

	t.Run("when input has invalid organization", func(t *testing.T) {
		result, err := client.ModifyWorkspace(&ModifyWorkspaceInput{
			OrganizationName: String("! / nope"),
			Name:             ws.Name,
		})
		assert.EqualError(t, err, "Invalid value for OrganizationName")
		assert.Nil(t, result)
	})
}

func TestDeleteWorkspace(t *testing.T) {
	client := testClient(t)

	ws, wsCleanup := createWorkspace(t, client, nil)
	defer wsCleanup()

	t.Run("with valid input", func(t *testing.T) {
		output, err := client.DeleteWorkspace(&DeleteWorkspaceInput{
			OrganizationName: ws.OrganizationName,
			Name:             ws.Name,
		})
		require.Nil(t, err)
		require.Equal(t, &DeleteWorkspaceOutput{}, output)

		// Try loading the workspace - it should fail.
		_, err = client.Workspace(*ws.OrganizationName, *ws.Name)
		assert.EqualError(t, err, "Resource not found")
	})

	t.Run("when input has invalid name", func(t *testing.T) {
		result, err := client.DeleteWorkspace(&DeleteWorkspaceInput{
			OrganizationName: ws.OrganizationName,
			Name:             String("! / nope"),
		})
		assert.EqualError(t, err, "Invalid value for Name")
		assert.Nil(t, result)
	})

	t.Run("when input has invalid organization", func(t *testing.T) {
		result, err := client.DeleteWorkspace(&DeleteWorkspaceInput{
			OrganizationName: String("! / nope"),
			Name:             ws.Name,
		})
		assert.EqualError(t, err, "Invalid value for OrganizationName")
		assert.Nil(t, result)
	})
}
