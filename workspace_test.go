package tfe

import (
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWorkspaces(t *testing.T) {
	client := testClient(t)

	org, orgCleanup := createOrganization(t, client)
	defer orgCleanup()

	ws1, ws1Cleanup := createWorkspace(t, client, org)
	defer ws1Cleanup()
	ws2, ws2Cleanup := createWorkspace(t, client, org)
	defer ws2Cleanup()

	// List the workspaces within the organization.
	workspaces, err := client.Workspaces(*org.Name)
	require.Nil(t, err)

	expect := []*Workspace{ws1, ws2}

	// Sort to ensure we get a non-flaky comparison.
	sort.Stable(WorkspaceNameSort(expect))
	sort.Stable(WorkspaceNameSort(workspaces))

	assert.Equal(t, expect, workspaces)
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
			Organization:     org.Name,
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
		assert.EqualError(t, err, "Organization and Name are required")
		assert.Nil(t, result)
	})

	t.Run("when input is missing name", func(t *testing.T) {
		result, err := client.CreateWorkspace(&CreateWorkspaceInput{
			Organization: org.Name,
		})
		assert.EqualError(t, err, "Organization and Name are required")
		assert.Nil(t, result)
	})

	t.Run("when an error is returned from the api", func(t *testing.T) {
		result, err := client.CreateWorkspace(&CreateWorkspaceInput{
			Organization:     org.Name,
			Name:             String("bar"),
			TerraformVersion: String("nope"),
		})
		assert.NotNil(t, err)
		println(err.Error())
		assert.Nil(t, result)
	})
}
