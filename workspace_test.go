package tfe

import (
	"bytes"
	"context"
	"encoding/json"
	"io/ioutil"
	"strings"
	"testing"
	"time"

	"github.com/hashicorp/go-retryablehttp"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWorkspacesList(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	defer orgTestCleanup()

	wTest1, wTest1Cleanup := createWorkspace(t, client, orgTest)
	defer wTest1Cleanup()
	wTest2, wTest2Cleanup := createWorkspace(t, client, orgTest)
	defer wTest2Cleanup()

	t.Run("without list options", func(t *testing.T) {
		wl, err := client.Workspaces.List(ctx, orgTest.Name, WorkspaceListOptions{})
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
		wl, err := client.Workspaces.List(ctx, orgTest.Name, WorkspaceListOptions{
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

	t.Run("when searching a known workspace", func(t *testing.T) {
		// Use a known workspace prefix as search attribute. The result
		// should be successful and only contain the matching workspace.
		wl, err := client.Workspaces.List(ctx, orgTest.Name, WorkspaceListOptions{
			Search: String(wTest1.Name[:len(wTest1.Name)-5]),
		})
		require.NoError(t, err)
		assert.Contains(t, wl.Items, wTest1)
		assert.NotContains(t, wl.Items, wTest2)
		assert.Equal(t, 1, wl.CurrentPage)
		assert.Equal(t, 1, wl.TotalCount)
	})

	t.Run("when searching an unknown workspace", func(t *testing.T) {
		// Use a nonexisting workspace name as search attribute. The result
		// should be successful, but return no results.
		wl, err := client.Workspaces.List(ctx, orgTest.Name, WorkspaceListOptions{
			Search: String("nonexisting"),
		})
		require.NoError(t, err)
		assert.Empty(t, wl.Items)
		assert.Equal(t, 1, wl.CurrentPage)
		assert.Equal(t, 0, wl.TotalCount)
	})

	t.Run("without a valid organization", func(t *testing.T) {
		wl, err := client.Workspaces.List(ctx, badIdentifier, WorkspaceListOptions{})
		assert.Nil(t, wl)
		assert.EqualError(t, err, ErrInvalidOrg.Error())
	})

	t.Run("with organization included", func(t *testing.T) {
		wl, err := client.Workspaces.List(ctx, orgTest.Name, WorkspaceListOptions{
			Include: String("organization"),
		})

		assert.NoError(t, err)

		assert.NotEmpty(t, wl.Items)
		assert.NotNil(t, wl.Items[0].Organization)
		assert.NotEmpty(t, wl.Items[0].Organization.Email)
	})
}

func TestWorkspacesCreate(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	defer orgTestCleanup()

	t.Run("with valid options", func(t *testing.T) {
		options := WorkspaceCreateOptions{
			Name:                       String("foo"),
			AllowDestroyPlan:           Bool(false),
			AutoApply:                  Bool(true),
			Description:                String("qux"),
			FileTriggersEnabled:        Bool(true),
			Operations:                 Bool(true),
			QueueAllRuns:               Bool(true),
			SpeculativeEnabled:         Bool(true),
			SourceName:                 String("my-app"),
			SourceURL:                  String("http://my-app-hostname.io"),
			StructuredRunOutputEnabled: Bool(true),
			TerraformVersion:           String("0.11.0"),
			TriggerPrefixes:            []string{"/modules", "/shared"},
			WorkingDirectory:           String("bar/"),
		}

		w, err := client.Workspaces.Create(ctx, orgTest.Name, options)
		require.NoError(t, err)

		// Get a refreshed view from the API.
		refreshed, err := client.Workspaces.Read(ctx, orgTest.Name, *options.Name)
		require.NoError(t, err)

		for _, item := range []*Workspace{
			w,
			refreshed,
		} {
			assert.NotEmpty(t, item.ID)
			assert.Equal(t, *options.Name, item.Name)
			assert.Equal(t, *options.Description, item.Description)
			assert.Equal(t, *options.AllowDestroyPlan, item.AllowDestroyPlan)
			assert.Equal(t, *options.AutoApply, item.AutoApply)
			assert.Equal(t, *options.FileTriggersEnabled, item.FileTriggersEnabled)
			assert.Equal(t, *options.Operations, item.Operations)
			assert.Equal(t, *options.QueueAllRuns, item.QueueAllRuns)
			assert.Equal(t, *options.SpeculativeEnabled, item.SpeculativeEnabled)
			assert.Equal(t, *options.SourceName, item.SourceName)
			assert.Equal(t, *options.SourceURL, item.SourceURL)
			assert.Equal(t, *options.StructuredRunOutputEnabled, item.StructuredRunOutputEnabled)
			assert.Equal(t, *options.TerraformVersion, item.TerraformVersion)
			assert.Equal(t, options.TriggerPrefixes, item.TriggerPrefixes)
			assert.Equal(t, *options.WorkingDirectory, item.WorkingDirectory)
		}
	})

	t.Run("when options is missing name", func(t *testing.T) {
		w, err := client.Workspaces.Create(ctx, "foo", WorkspaceCreateOptions{})
		assert.Nil(t, w)
		assert.EqualError(t, err, ErrRequiredName.Error())
	})

	t.Run("when options has an invalid name", func(t *testing.T) {
		w, err := client.Workspaces.Create(ctx, "foo", WorkspaceCreateOptions{
			Name: String(badIdentifier),
		})
		assert.Nil(t, w)
		assert.EqualError(t, err, ErrInvalidName.Error())
	})

	t.Run("when options has an invalid organization", func(t *testing.T) {
		w, err := client.Workspaces.Create(ctx, badIdentifier, WorkspaceCreateOptions{
			Name: String("foo"),
		})
		assert.Nil(t, w)
		assert.EqualError(t, err, ErrInvalidOrg.Error())
	})

	t.Run("when options includes both an operations value and an enforcement mode value", func(t *testing.T) {
		options := WorkspaceCreateOptions{
			Name:          String("foo"),
			ExecutionMode: String("remote"),
			Operations:    Bool(true),
		}

		w, err := client.Workspaces.Create(ctx, orgTest.Name, options)
		assert.Nil(t, w)
		assert.EqualError(t, err, "operations is deprecated and cannot be specified when execution mode is used")
	})

	t.Run("when an agent pool ID is specified without 'agent' execution mode", func(t *testing.T) {
		options := WorkspaceCreateOptions{
			Name:        String("foo"),
			AgentPoolID: String("apool-xxxxx"),
		}

		w, err := client.Workspaces.Create(ctx, orgTest.Name, options)
		assert.Nil(t, w)
		assert.EqualError(t, err, "specifying an agent pool ID requires 'agent' execution mode")
	})

	t.Run("when 'agent' execution mode is specified without an an agent pool ID", func(t *testing.T) {
		options := WorkspaceCreateOptions{
			Name:          String("foo"),
			ExecutionMode: String("agent"),
		}

		w, err := client.Workspaces.Create(ctx, orgTest.Name, options)
		assert.Nil(t, w)
		assert.EqualError(t, err, "'agent' execution mode requires an agent pool ID to be specified")
	})

	t.Run("when an error is returned from the API", func(t *testing.T) {
		w, err := client.Workspaces.Create(ctx, "bar", WorkspaceCreateOptions{
			Name:             String("bar"),
			TerraformVersion: String("nonexisting"),
		})
		assert.Nil(t, w)
		assert.Error(t, err)
	})
}

func TestWorkspacesRead(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	defer orgTestCleanup()

	wTest, wTestCleanup := createWorkspace(t, client, orgTest)
	defer wTestCleanup()

	t.Run("when the workspace exists", func(t *testing.T) {
		w, err := client.Workspaces.Read(ctx, orgTest.Name, wTest.Name)
		require.NoError(t, err)
		assert.Equal(t, wTest, w)

		assert.True(t, w.Permissions.CanDestroy)
		assert.NotEmpty(t, w.Actions)
		assert.Equal(t, orgTest.Name, w.Organization.Name)
		assert.NotEmpty(t, w.CreatedAt)
	})

	t.Run("when the workspace does not exist", func(t *testing.T) {
		w, err := client.Workspaces.Read(ctx, orgTest.Name, "nonexisting")
		assert.Nil(t, w)
		assert.Error(t, err)
	})

	t.Run("when the organization does not exist", func(t *testing.T) {
		w, err := client.Workspaces.Read(ctx, "nonexisting", "nonexisting")
		assert.Nil(t, w)
		assert.Error(t, err)
	})

	t.Run("without a valid organization", func(t *testing.T) {
		w, err := client.Workspaces.Read(ctx, badIdentifier, wTest.Name)
		assert.Nil(t, w)
		assert.EqualError(t, err, ErrInvalidOrg.Error())
	})

	t.Run("without a valid workspace", func(t *testing.T) {
		w, err := client.Workspaces.Read(ctx, orgTest.Name, badIdentifier)
		assert.Nil(t, w)
		assert.EqualError(t, err, ErrInvalidWorkspaceValue.Error())
	})
}

func TestWorkspacesReadWithOptions(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	defer orgTestCleanup()

	wTest, wTestCleanup := createWorkspace(t, client, orgTest)
	defer wTestCleanup()

	svTest, svTestCleanup := createStateVersion(t, client, 0, wTest)
	defer svTestCleanup()

	// give TFC some time to process the statefile and extract the outputs.
	time.Sleep(waitForStateVersionOutputs)

	t.Run("when options to include resource", func(t *testing.T) {
		opts := &WorkspaceReadOptions{
			Include: "outputs",
		}
		w, err := client.Workspaces.ReadWithOptions(ctx, orgTest.Name, wTest.Name, opts)
		require.NoError(t, err)

		assert.Equal(t, wTest.ID, w.ID)
		assert.NotEmpty(t, w.Outputs)

		svOutputs, err := client.StateVersions.Outputs(ctx, svTest.ID, StateVersionOutputsListOptions{})
		require.NoError(t, err)

		assert.Len(t, w.Outputs, len(svOutputs))

		wsOutputs := map[string]interface{}{}
		wsOutputsTypes := map[string]string{}
		for _, op := range w.Outputs {
			wsOutputs[op.Name] = op.Value
			wsOutputsTypes[op.Name] = op.Type
		}
		for _, svop := range svOutputs {
			val, ok := wsOutputs[svop.Name]
			assert.True(t, ok)
			assert.Equal(t, svop.Value, val)

			val, ok = wsOutputsTypes[svop.Name]
			assert.True(t, ok)
			assert.Equal(t, svop.Type, val)
		}
	})
}

func TestWorkspacesReadWithHistory(t *testing.T) {
	client := testClient(t)

	orgTest, orgTestCleanup := createOrganization(t, client)
	defer orgTestCleanup()

	wTest, wTestCleanup := createWorkspace(t, client, orgTest)
	defer wTestCleanup()

	_, rCleanup := createAppliedRun(t, client, wTest)
	defer rCleanup()

	w, err := client.Workspaces.Read(context.Background(), orgTest.Name, wTest.Name)
	require.NoError(t, err)

	assert.Equal(t, 1, w.RunsCount)
	assert.Equal(t, 1, w.ResourceCount)
}

func TestWorkspacesReadReadme(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	defer orgTestCleanup()

	wTest, wTestCleanup := createWorkspaceWithVCS(t, client, orgTest, WorkspaceCreateOptions{})
	defer wTestCleanup()

	_, rCleanup := createAppliedRun(t, client, wTest)
	defer rCleanup()

	t.Run("when the readme exists", func(t *testing.T) {
		w, err := client.Workspaces.Readme(ctx, wTest.ID)
		require.NoError(t, err)
		require.NotNil(t, w)

		readme, err := ioutil.ReadAll(w)
		require.NoError(t, err)
		require.True(
			t,
			strings.HasPrefix(string(readme), `This is a simple test`),
			"got: %s", readme,
		)
	})

	t.Run("when the readme does not exist", func(t *testing.T) {
		w, err := client.Workspaces.Readme(ctx, "nonexisting")
		assert.Nil(t, w)
		assert.Error(t, err)
	})

	t.Run("without a valid workspace ID", func(t *testing.T) {
		w, err := client.Workspaces.Readme(ctx, badIdentifier)
		assert.Nil(t, w)
		assert.EqualError(t, err, ErrInvalidWorkspaceID.Error())
	})
}

func TestWorkspacesReadByID(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	defer orgTestCleanup()

	wTest, wTestCleanup := createWorkspace(t, client, orgTest)
	defer wTestCleanup()

	t.Run("when the workspace exists", func(t *testing.T) {
		w, err := client.Workspaces.ReadByID(ctx, wTest.ID)
		require.NoError(t, err)
		assert.Equal(t, wTest, w)

		assert.True(t, w.Permissions.CanDestroy)
		assert.Equal(t, orgTest.Name, w.Organization.Name)
		assert.NotEmpty(t, w.CreatedAt)
		assert.NotEmpty(t, w.Actions)
	})

	t.Run("when the workspace does not exist", func(t *testing.T) {
		w, err := client.Workspaces.ReadByID(ctx, "nonexisting")
		assert.Nil(t, w)
		assert.Error(t, err)
	})

	t.Run("without a valid workspace ID", func(t *testing.T) {
		w, err := client.Workspaces.ReadByID(ctx, badIdentifier)
		assert.Nil(t, w)
		assert.EqualError(t, err, ErrInvalidWorkspaceID.Error())
	})
}

func TestWorkspacesUpdate(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	defer orgTestCleanup()

	wTest, _ := createWorkspace(t, client, orgTest)

	t.Run("when updating a subset of values", func(t *testing.T) {
		options := WorkspaceUpdateOptions{
			Name:             String(wTest.Name),
			AllowDestroyPlan: Bool(false),
			AutoApply:        Bool(true),
			Operations:       Bool(true),
			QueueAllRuns:     Bool(true),
			TerraformVersion: String("0.10.0"),
		}

		wAfter, err := client.Workspaces.Update(ctx, orgTest.Name, wTest.Name, options)
		require.NoError(t, err)

		assert.Equal(t, wTest.Name, wAfter.Name)
		assert.NotEqual(t, wTest.AllowDestroyPlan, wAfter.AllowDestroyPlan)
		assert.NotEqual(t, wTest.AutoApply, wAfter.AutoApply)
		assert.NotEqual(t, wTest.QueueAllRuns, wAfter.QueueAllRuns)
		assert.NotEqual(t, wTest.TerraformVersion, wAfter.TerraformVersion)
		assert.Equal(t, wTest.WorkingDirectory, wAfter.WorkingDirectory)
	})

	t.Run("with valid options", func(t *testing.T) {
		options := WorkspaceUpdateOptions{
			Name:                       String(randomString(t)),
			AllowDestroyPlan:           Bool(true),
			AutoApply:                  Bool(false),
			FileTriggersEnabled:        Bool(true),
			Operations:                 Bool(false),
			QueueAllRuns:               Bool(false),
			SpeculativeEnabled:         Bool(true),
			Description:                String("updated description"),
			StructuredRunOutputEnabled: Bool(true),
			TerraformVersion:           String("0.11.1"),
			TriggerPrefixes:            []string{"/modules", "/shared"},
			WorkingDirectory:           String("baz/"),
		}

		w, err := client.Workspaces.Update(ctx, orgTest.Name, wTest.Name, options)
		require.NoError(t, err)

		// Get a refreshed view of the workspace from the API
		refreshed, err := client.Workspaces.Read(ctx, orgTest.Name, *options.Name)
		require.NoError(t, err)

		for _, item := range []*Workspace{
			w,
			refreshed,
		} {
			assert.Equal(t, *options.Name, item.Name)
			assert.Equal(t, *options.AllowDestroyPlan, item.AllowDestroyPlan)
			assert.Equal(t, *options.AutoApply, item.AutoApply)
			assert.Equal(t, *options.FileTriggersEnabled, item.FileTriggersEnabled)
			assert.Equal(t, *options.Description, item.Description)
			assert.Equal(t, *options.Operations, item.Operations)
			assert.Equal(t, *options.QueueAllRuns, item.QueueAllRuns)
			assert.Equal(t, *options.SpeculativeEnabled, item.SpeculativeEnabled)
			assert.Equal(t, *options.StructuredRunOutputEnabled, item.StructuredRunOutputEnabled)
			assert.Equal(t, *options.TerraformVersion, item.TerraformVersion)
			assert.Equal(t, options.TriggerPrefixes, item.TriggerPrefixes)
			assert.Equal(t, *options.WorkingDirectory, item.WorkingDirectory)
		}
	})

	t.Run("when options includes both an operations value and an enforcement mode value", func(t *testing.T) {
		options := WorkspaceUpdateOptions{
			ExecutionMode: String("remote"),
			Operations:    Bool(true),
		}

		wAfter, err := client.Workspaces.Update(ctx, orgTest.Name, wTest.Name, options)
		assert.Nil(t, wAfter)
		assert.EqualError(t, err, "operations is deprecated and cannot be specified when execution mode is used")
	})

	t.Run("when 'agent' execution mode is specified without an an agent pool ID", func(t *testing.T) {
		options := WorkspaceUpdateOptions{
			ExecutionMode: String("agent"),
		}

		wAfter, err := client.Workspaces.Update(ctx, orgTest.Name, wTest.Name, options)
		assert.Nil(t, wAfter)
		assert.EqualError(t, err, "'agent' execution mode requires an agent pool ID to be specified")
	})

	t.Run("when an error is returned from the api", func(t *testing.T) {
		w, err := client.Workspaces.Update(ctx, orgTest.Name, wTest.Name, WorkspaceUpdateOptions{
			TerraformVersion: String("nonexisting"),
		})
		assert.Nil(t, w)
		assert.Error(t, err)
	})

	t.Run("when options has an invalid name", func(t *testing.T) {
		w, err := client.Workspaces.Update(ctx, orgTest.Name, badIdentifier, WorkspaceUpdateOptions{})
		assert.Nil(t, w)
		assert.EqualError(t, err, ErrInvalidWorkspaceValue.Error())
	})

	t.Run("when options has an invalid organization", func(t *testing.T) {
		w, err := client.Workspaces.Update(ctx, badIdentifier, wTest.Name, WorkspaceUpdateOptions{})
		assert.Nil(t, w)
		assert.EqualError(t, err, ErrInvalidOrg.Error())
	})
}

func TestWorkspacesUpdateByID(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	defer orgTestCleanup()

	wTest, _ := createWorkspace(t, client, orgTest)

	t.Run("when updating a subset of values", func(t *testing.T) {
		options := WorkspaceUpdateOptions{
			Name:             String(wTest.Name),
			AllowDestroyPlan: Bool(false),
			AutoApply:        Bool(true),
			Operations:       Bool(true),
			QueueAllRuns:     Bool(true),
			TerraformVersion: String("0.10.0"),
		}

		wAfter, err := client.Workspaces.UpdateByID(ctx, wTest.ID, options)
		require.NoError(t, err)

		assert.Equal(t, wTest.Name, wAfter.Name)
		assert.NotEqual(t, wTest.AllowDestroyPlan, wAfter.AllowDestroyPlan)
		assert.NotEqual(t, wTest.AutoApply, wAfter.AutoApply)
		assert.NotEqual(t, wTest.QueueAllRuns, wAfter.QueueAllRuns)
		assert.NotEqual(t, wTest.TerraformVersion, wAfter.TerraformVersion)
		assert.Equal(t, wTest.WorkingDirectory, wAfter.WorkingDirectory)
	})

	t.Run("with valid options", func(t *testing.T) {
		options := WorkspaceUpdateOptions{
			Name:                       String(randomString(t)),
			AllowDestroyPlan:           Bool(true),
			AutoApply:                  Bool(false),
			FileTriggersEnabled:        Bool(true),
			Operations:                 Bool(false),
			QueueAllRuns:               Bool(false),
			SpeculativeEnabled:         Bool(true),
			StructuredRunOutputEnabled: Bool(true),
			TerraformVersion:           String("0.11.1"),
			TriggerPrefixes:            []string{"/modules", "/shared"},
			WorkingDirectory:           String("baz/"),
		}

		w, err := client.Workspaces.UpdateByID(ctx, wTest.ID, options)
		require.NoError(t, err)

		// Get a refreshed view of the workspace from the API
		refreshed, err := client.Workspaces.Read(ctx, orgTest.Name, *options.Name)
		require.NoError(t, err)

		for _, item := range []*Workspace{
			w,
			refreshed,
		} {
			assert.Equal(t, *options.Name, item.Name)
			assert.Equal(t, *options.AllowDestroyPlan, item.AllowDestroyPlan)
			assert.Equal(t, *options.AutoApply, item.AutoApply)
			assert.Equal(t, *options.FileTriggersEnabled, item.FileTriggersEnabled)
			assert.Equal(t, *options.Operations, item.Operations)
			assert.Equal(t, *options.QueueAllRuns, item.QueueAllRuns)
			assert.Equal(t, *options.SpeculativeEnabled, item.SpeculativeEnabled)
			assert.Equal(t, *options.StructuredRunOutputEnabled, item.StructuredRunOutputEnabled)
			assert.Equal(t, *options.TerraformVersion, item.TerraformVersion)
			assert.Equal(t, options.TriggerPrefixes, item.TriggerPrefixes)
			assert.Equal(t, *options.WorkingDirectory, item.WorkingDirectory)
		}
	})

	t.Run("when an error is returned from the api", func(t *testing.T) {
		w, err := client.Workspaces.UpdateByID(ctx, wTest.ID, WorkspaceUpdateOptions{
			TerraformVersion: String("nonexisting"),
		})
		assert.Nil(t, w)
		assert.Error(t, err)
	})

	t.Run("without a valid workspace ID", func(t *testing.T) {
		w, err := client.Workspaces.UpdateByID(ctx, badIdentifier, WorkspaceUpdateOptions{})
		assert.Nil(t, w)
		assert.EqualError(t, err, ErrInvalidWorkspaceID.Error())
	})
}

func TestWorkspacesDelete(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	defer orgTestCleanup()

	wTest, _ := createWorkspace(t, client, orgTest)

	t.Run("with valid options", func(t *testing.T) {
		err := client.Workspaces.Delete(ctx, orgTest.Name, wTest.Name)
		require.NoError(t, err)

		// Try loading the workspace - it should fail.
		_, err = client.Workspaces.Read(ctx, orgTest.Name, wTest.Name)
		assert.Equal(t, ErrResourceNotFound, err)
	})

	t.Run("when organization is invalid", func(t *testing.T) {
		err := client.Workspaces.Delete(ctx, badIdentifier, wTest.Name)
		assert.EqualError(t, err, ErrInvalidOrg.Error())
	})

	t.Run("when workspace is invalid", func(t *testing.T) {
		err := client.Workspaces.Delete(ctx, orgTest.Name, badIdentifier)
		assert.EqualError(t, err, ErrInvalidWorkspaceValue.Error())
	})
}

func TestWorkspacesDeleteByID(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	defer orgTestCleanup()

	wTest, _ := createWorkspace(t, client, orgTest)

	t.Run("with valid options", func(t *testing.T) {
		err := client.Workspaces.DeleteByID(ctx, wTest.ID)
		require.NoError(t, err)

		// Try loading the workspace - it should fail.
		_, err = client.Workspaces.ReadByID(ctx, wTest.ID)
		assert.Equal(t, ErrResourceNotFound, err)
	})

	t.Run("without a valid workspace ID", func(t *testing.T) {
		err := client.Workspaces.DeleteByID(ctx, badIdentifier)
		assert.EqualError(t, err, ErrInvalidWorkspaceID.Error())
	})
}

func TestWorkspacesRemoveVCSConnection(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	defer orgTestCleanup()

	wTest, wTestCleanup := createWorkspaceWithVCS(t, client, orgTest, WorkspaceCreateOptions{})
	defer wTestCleanup()

	t.Run("remove vcs integration", func(t *testing.T) {
		w, err := client.Workspaces.RemoveVCSConnection(ctx, orgTest.Name, wTest.Name)
		require.NoError(t, err)
		assert.Equal(t, (*VCSRepo)(nil), w.VCSRepo)
	})
}

func TestWorkspacesRemoveVCSConnectionByID(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	defer orgTestCleanup()

	wTest, wTestCleanup := createWorkspaceWithVCS(t, client, orgTest, WorkspaceCreateOptions{})
	defer wTestCleanup()

	t.Run("remove vcs integration", func(t *testing.T) {
		w, err := client.Workspaces.RemoveVCSConnectionByID(ctx, wTest.ID)
		require.NoError(t, err)
		assert.Equal(t, (*VCSRepo)(nil), w.VCSRepo)
	})
}

func TestWorkspacesLock(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	defer orgTestCleanup()

	wTest, wTestCleanup := createWorkspace(t, client, orgTest)
	defer wTestCleanup()

	t.Run("with valid options", func(t *testing.T) {
		w, err := client.Workspaces.Lock(ctx, wTest.ID, WorkspaceLockOptions{})
		require.NoError(t, err)
		assert.True(t, w.Locked)
	})

	t.Run("when workspace is already locked", func(t *testing.T) {
		_, err := client.Workspaces.Lock(ctx, wTest.ID, WorkspaceLockOptions{})
		assert.Equal(t, ErrWorkspaceLocked, err)
	})

	t.Run("without a valid workspace ID", func(t *testing.T) {
		w, err := client.Workspaces.Lock(ctx, badIdentifier, WorkspaceLockOptions{})
		assert.Nil(t, w)
		assert.EqualError(t, err, ErrInvalidWorkspaceID.Error())
	})
}

func TestWorkspacesUnlock(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	defer orgTestCleanup()

	wTest, wTestCleanup := createWorkspace(t, client, orgTest)
	defer wTestCleanup()

	w, err := client.Workspaces.Lock(ctx, wTest.ID, WorkspaceLockOptions{})
	if err != nil {
		orgTestCleanup()
	}
	require.NoError(t, err)
	require.True(t, w.Locked)

	t.Run("with valid options", func(t *testing.T) {
		w, err := client.Workspaces.Unlock(ctx, wTest.ID)
		require.NoError(t, err)
		assert.False(t, w.Locked)
	})

	t.Run("when workspace is already unlocked", func(t *testing.T) {
		_, err := client.Workspaces.Unlock(ctx, wTest.ID)
		assert.Equal(t, ErrWorkspaceNotLocked, err)
	})

	t.Run("without a valid workspace ID", func(t *testing.T) {
		w, err := client.Workspaces.Unlock(ctx, badIdentifier)
		assert.Nil(t, w)
		assert.EqualError(t, err, ErrInvalidWorkspaceID.Error())
	})
}

func TestWorkspacesForceUnlock(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	defer orgTestCleanup()

	wTest, wTestCleanup := createWorkspace(t, client, orgTest)
	defer wTestCleanup()

	w, err := client.Workspaces.Lock(ctx, wTest.ID, WorkspaceLockOptions{})
	if err != nil {
		orgTestCleanup()
	}
	require.NoError(t, err)
	require.True(t, w.Locked)

	t.Run("with valid options", func(t *testing.T) {
		w, err := client.Workspaces.ForceUnlock(ctx, wTest.ID)
		require.NoError(t, err)
		assert.False(t, w.Locked)
	})

	t.Run("when workspace is already unlocked", func(t *testing.T) {
		_, err := client.Workspaces.ForceUnlock(ctx, wTest.ID)
		assert.Equal(t, ErrWorkspaceNotLocked, err)
	})

	t.Run("without a valid workspace ID", func(t *testing.T) {
		w, err := client.Workspaces.ForceUnlock(ctx, badIdentifier)
		assert.Nil(t, w)
		assert.EqualError(t, err, ErrInvalidWorkspaceID.Error())
	})
}

func TestWorkspacesAssignSSHKey(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	defer orgTestCleanup()

	wTest, wTestCleanup := createWorkspace(t, client, orgTest)
	defer wTestCleanup()

	sshKeyTest, sshKeyTestCleanup := createSSHKey(t, client, orgTest)
	defer sshKeyTestCleanup()

	t.Run("with valid options", func(t *testing.T) {
		w, err := client.Workspaces.AssignSSHKey(ctx, wTest.ID, WorkspaceAssignSSHKeyOptions{
			SSHKeyID: String(sshKeyTest.ID),
		})
		require.NoError(t, err)
		require.NotNil(t, w.SSHKey)
		assert.Equal(t, w.SSHKey.ID, sshKeyTest.ID)
	})

	t.Run("without an SSH key ID", func(t *testing.T) {
		w, err := client.Workspaces.AssignSSHKey(ctx, wTest.ID, WorkspaceAssignSSHKeyOptions{})
		assert.Nil(t, w)
		assert.EqualError(t, err, "SSH key ID is required")
	})

	t.Run("without a valid SSH key ID", func(t *testing.T) {
		w, err := client.Workspaces.AssignSSHKey(ctx, wTest.ID, WorkspaceAssignSSHKeyOptions{
			SSHKeyID: String(badIdentifier),
		})
		assert.Nil(t, w)
		assert.EqualError(t, err, "invalid value for SSH key ID")
	})

	t.Run("without a valid workspace ID", func(t *testing.T) {
		w, err := client.Workspaces.AssignSSHKey(ctx, badIdentifier, WorkspaceAssignSSHKeyOptions{
			SSHKeyID: String(sshKeyTest.ID),
		})
		assert.Nil(t, w)
		assert.EqualError(t, err, ErrInvalidWorkspaceID.Error())
	})
}

func TestWorkspacesUnassignSSHKey(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	defer orgTestCleanup()

	wTest, wTestCleanup := createWorkspace(t, client, orgTest)
	defer wTestCleanup()

	sshKeyTest, sshKeyTestCleanup := createSSHKey(t, client, orgTest)
	defer sshKeyTestCleanup()

	w, err := client.Workspaces.AssignSSHKey(ctx, wTest.ID, WorkspaceAssignSSHKeyOptions{
		SSHKeyID: String(sshKeyTest.ID),
	})
	if err != nil {
		orgTestCleanup()
	}
	require.NoError(t, err)
	require.NotNil(t, w.SSHKey)
	require.Equal(t, w.SSHKey.ID, sshKeyTest.ID)

	t.Run("with valid options", func(t *testing.T) {
		w, err := client.Workspaces.UnassignSSHKey(ctx, wTest.ID)
		assert.Nil(t, err)
		assert.Nil(t, w.SSHKey)
	})

	t.Run("without a valid workspace ID", func(t *testing.T) {
		w, err := client.Workspaces.UnassignSSHKey(ctx, badIdentifier)
		assert.Nil(t, w)
		assert.EqualError(t, err, ErrInvalidWorkspaceID.Error())
	})
}

func TestWorkspaces_AddRemoteStateConsumers(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	defer orgTestCleanup()

	wTest, wTestCleanup := createWorkspace(t, client, orgTest)
	defer wTestCleanup()

	// Update workspace to not allow global remote state
	options := WorkspaceUpdateOptions{
		GlobalRemoteState: Bool(false),
	}
	wTest, err := client.Workspaces.Update(ctx, orgTest.Name, wTest.Name, options)
	require.NoError(t, err)

	t.Run("successfully adds a remote state consumer", func(t *testing.T) {
		wTestConsumer1, wTestCleanupConsumer1 := createWorkspace(t, client, orgTest)
		defer wTestCleanupConsumer1()
		wTestConsumer2, wTestCleanupConsumer2 := createWorkspace(t, client, orgTest)
		defer wTestCleanupConsumer2()

		err := client.Workspaces.AddRemoteStateConsumers(ctx, wTest.ID, WorkspaceAddRemoteStateConsumersOptions{
			Workspaces: []*Workspace{wTestConsumer1, wTestConsumer2},
		})
		require.NoError(t, err)

		_, err = client.Workspaces.Read(ctx, orgTest.Name, wTest.Name)
		require.NoError(t, err)

		rsc, err := client.Workspaces.RemoteStateConsumers(ctx, wTest.ID)
		require.NoError(t, err)
		assert.Equal(t, 2, len(rsc.Items))
		assert.Contains(t, rsc.Items, wTestConsumer1)
		assert.Contains(t, rsc.Items, wTestConsumer2)
	})

	t.Run("with invalid options", func(t *testing.T) {
		err := client.Workspaces.AddRemoteStateConsumers(ctx, wTest.ID, WorkspaceAddRemoteStateConsumersOptions{})
		require.Error(t, err)
		assert.EqualError(t, err, ErrWorkspacesRequired.Error())

		err = client.Workspaces.AddRemoteStateConsumers(ctx, wTest.ID, WorkspaceAddRemoteStateConsumersOptions{
			Workspaces: []*Workspace{},
		})
		require.Error(t, err)
		assert.EqualError(t, err, ErrWorkspaceMinLimit.Error())
	})

	t.Run("without a valid workspace ID", func(t *testing.T) {
		err := client.Workspaces.AddRemoteStateConsumers(ctx, badIdentifier, WorkspaceAddRemoteStateConsumersOptions{})
		require.Error(t, err)
		assert.EqualError(t, err, ErrInvalidWorkspaceID.Error())
	})
}

func TestWorkspaces_RemoveRemoteStateConsumers(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	defer orgTestCleanup()

	wTest, wTestCleanup := createWorkspace(t, client, orgTest)
	defer wTestCleanup()

	// Update workspace to not allow global remote state
	options := WorkspaceUpdateOptions{
		GlobalRemoteState: Bool(false),
	}
	wTest, err := client.Workspaces.Update(ctx, orgTest.Name, wTest.Name, options)
	require.NoError(t, err)

	t.Run("successfully removes a remote state consumer", func(t *testing.T) {
		wTestConsumer1, wTestCleanupConsumer1 := createWorkspace(t, client, orgTest)
		defer wTestCleanupConsumer1()
		wTestConsumer2, wTestCleanupConsumer2 := createWorkspace(t, client, orgTest)
		defer wTestCleanupConsumer2()

		err := client.Workspaces.AddRemoteStateConsumers(ctx, wTest.ID, WorkspaceAddRemoteStateConsumersOptions{
			Workspaces: []*Workspace{wTestConsumer1, wTestConsumer2},
		})
		require.NoError(t, err)

		rsc, err := client.Workspaces.RemoteStateConsumers(ctx, wTest.ID)
		require.NoError(t, err)
		assert.Equal(t, 2, len(rsc.Items))
		assert.Contains(t, rsc.Items, wTestConsumer1)
		assert.Contains(t, rsc.Items, wTestConsumer2)

		err = client.Workspaces.RemoveRemoteStateConsumers(ctx, wTest.ID, WorkspaceRemoveRemoteStateConsumersOptions{
			Workspaces: []*Workspace{wTestConsumer1},
		})
		require.NoError(t, err)

		_, err = client.Workspaces.Read(ctx, orgTest.Name, wTest.Name)
		require.NoError(t, err)

		rsc, err = client.Workspaces.RemoteStateConsumers(ctx, wTest.ID)
		require.NoError(t, err)
		assert.Contains(t, rsc.Items, wTestConsumer2)
		assert.Equal(t, 1, len(rsc.Items))

		err = client.Workspaces.RemoveRemoteStateConsumers(ctx, wTest.ID, WorkspaceRemoveRemoteStateConsumersOptions{
			Workspaces: []*Workspace{wTestConsumer2},
		})
		require.NoError(t, err)

		rsc, err = client.Workspaces.RemoteStateConsumers(ctx, wTest.ID)
		require.NoError(t, err)
		assert.Empty(t, len(rsc.Items))
	})

	t.Run("with invalid options", func(t *testing.T) {
		err := client.Workspaces.RemoveRemoteStateConsumers(ctx, wTest.ID, WorkspaceRemoveRemoteStateConsumersOptions{})
		require.Error(t, err)
		assert.EqualError(t, err, ErrWorkspacesRequired.Error())

		err = client.Workspaces.RemoveRemoteStateConsumers(ctx, wTest.ID, WorkspaceRemoveRemoteStateConsumersOptions{
			Workspaces: []*Workspace{},
		})
		require.Error(t, err)
		assert.EqualError(t, err, ErrWorkspaceMinLimit.Error())
	})

	t.Run("without a valid workspace ID", func(t *testing.T) {
		err := client.Workspaces.RemoveRemoteStateConsumers(ctx, badIdentifier, WorkspaceRemoveRemoteStateConsumersOptions{})
		require.Error(t, err)
		assert.EqualError(t, err, ErrInvalidWorkspaceID.Error())
	})
}

func TestWorkspaces_UpdateRemoteStateConsumers(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	defer orgTestCleanup()

	wTest, wTestCleanup := createWorkspace(t, client, orgTest)
	defer wTestCleanup()

	// Update workspace to not allow global remote state
	options := WorkspaceUpdateOptions{
		GlobalRemoteState: Bool(false),
	}
	wTest, err := client.Workspaces.Update(ctx, orgTest.Name, wTest.Name, options)
	require.NoError(t, err)

	t.Run("successfully updates a remote state consumer", func(t *testing.T) {
		wTestConsumer1, wTestCleanupConsumer1 := createWorkspace(t, client, orgTest)
		defer wTestCleanupConsumer1()
		wTestConsumer2, wTestCleanupConsumer2 := createWorkspace(t, client, orgTest)
		defer wTestCleanupConsumer2()

		err := client.Workspaces.AddRemoteStateConsumers(ctx, wTest.ID, WorkspaceAddRemoteStateConsumersOptions{
			Workspaces: []*Workspace{wTestConsumer1},
		})
		require.NoError(t, err)

		rsc, err := client.Workspaces.RemoteStateConsumers(ctx, wTest.ID)
		require.NoError(t, err)
		assert.Equal(t, 1, len(rsc.Items))
		assert.Contains(t, rsc.Items, wTestConsumer1)

		err = client.Workspaces.UpdateRemoteStateConsumers(ctx, wTest.ID, WorkspaceUpdateRemoteStateConsumersOptions{
			Workspaces: []*Workspace{wTestConsumer2},
		})
		require.NoError(t, err)

		rsc, err = client.Workspaces.RemoteStateConsumers(ctx, wTest.ID)
		require.NoError(t, err)
		assert.Equal(t, 1, len(rsc.Items))
		assert.Contains(t, rsc.Items, wTestConsumer2)

	})

	t.Run("with invalid options", func(t *testing.T) {
		err := client.Workspaces.UpdateRemoteStateConsumers(ctx, wTest.ID, WorkspaceUpdateRemoteStateConsumersOptions{})
		require.Error(t, err)
		assert.EqualError(t, err, ErrWorkspacesRequired.Error())

		err = client.Workspaces.UpdateRemoteStateConsumers(ctx, wTest.ID, WorkspaceUpdateRemoteStateConsumersOptions{
			Workspaces: []*Workspace{},
		})
		require.Error(t, err)
		assert.EqualError(t, err, ErrWorkspaceMinLimit.Error())
	})

	t.Run("without a valid workspace ID", func(t *testing.T) {
		err := client.Workspaces.UpdateRemoteStateConsumers(ctx, badIdentifier, WorkspaceUpdateRemoteStateConsumersOptions{})
		require.Error(t, err)
		assert.EqualError(t, err, ErrInvalidWorkspaceID.Error())
	})
}

func TestWorkspace_Unmarshal(t *testing.T) {
	data := map[string]interface{}{
		"data": map[string]interface{}{
			"type": "workspaces",
			"id":   "ws-1234",
			"attributes": map[string]interface{}{
				"name":           "my-workspace",
				"auto-apply":     true,
				"created-at":     "2020-07-15T23:38:43.821Z",
				"resource-count": 2,
				"permissions": map[string]interface{}{
					"can-update": true,
					"can-lock":   true,
				},
				"vcs-repo": map[string]interface{}{
					"branch":              "main",
					"display-identifier":  "repo-name",
					"identifier":          "hashicorp/repo-name",
					"ingress-submodules":  true,
					"oauth-token-id":      "token",
					"repository-http-url": "github.com",
					"service-provider":    "github",
				},
				"actions": map[string]interface{}{
					"is-destroyable": true,
				},
				"trigger-prefixes": []string{"prefix-"},
			},
		},
	}

	byteData, err := json.Marshal(data)
	require.NoError(t, err)

	responseBody := bytes.NewReader(byteData)
	ws := &Workspace{}
	err = unmarshalResponse(responseBody, ws)
	require.NoError(t, err)

	iso8601TimeFormat := "2006-01-02T15:04:05Z"
	parsedTime, _ := time.Parse(iso8601TimeFormat, "2020-07-15T23:38:43.821Z")

	assert.Equal(t, ws.ID, "ws-1234")
	assert.Equal(t, ws.Name, "my-workspace")
	assert.Equal(t, ws.AutoApply, true)
	assert.Equal(t, ws.CreatedAt, parsedTime)
	assert.Equal(t, ws.ResourceCount, 2)
	assert.Equal(t, ws.Permissions.CanUpdate, true)
	assert.Equal(t, ws.Permissions.CanLock, true)
	assert.Equal(t, ws.VCSRepo.Branch, "main")
	assert.Equal(t, ws.VCSRepo.DisplayIdentifier, "repo-name")
	assert.Equal(t, ws.VCSRepo.Identifier, "hashicorp/repo-name")
	assert.Equal(t, ws.VCSRepo.IngressSubmodules, true)
	assert.Equal(t, ws.VCSRepo.OAuthTokenID, "token")
	assert.Equal(t, ws.VCSRepo.RepositoryHTTPURL, "github.com")
	assert.Equal(t, ws.VCSRepo.ServiceProvider, "github")
	assert.Equal(t, ws.Actions.IsDestroyable, true)
	assert.Equal(t, ws.TriggerPrefixes, []string{"prefix-"})
}

func TestWorkspaceCreateOptions_Marshal(t *testing.T) {
	opts := WorkspaceCreateOptions{
		AllowDestroyPlan: Bool(true),
		Name:             String("my-workspace"),
		TriggerPrefixes:  []string{"prefix-"},
		VCSRepo: &VCSRepoOptions{
			Identifier:   String("id"),
			OAuthTokenID: String("token"),
		},
	}

	reqBody, err := serializeRequestBody(&opts)
	require.NoError(t, err)
	req, err := retryablehttp.NewRequest("POST", "url", reqBody)
	require.NoError(t, err)
	bodyBytes, err := req.BodyBytes()
	require.NoError(t, err)

	expectedBody := `{"data":{"type":"workspaces","attributes":{"allow-destroy-plan":true,"name":"my-workspace","trigger-prefixes":["prefix-"],"vcs-repo":{"identifier":"id","oauth-token-id":"token"}}}}
`
	assert.Equal(t, expectedBody, string(bodyBytes))
}
