package tfe

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVariableSetsList(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	defer orgTestCleanup()

	vsTest1, vsTestCleanup1 := createVariableSet(t, client, orgTest, VariableSetCreateOptions{})
	defer vsTestCleanup1()
	vsTest2, vsTestCleanup2 := createVariableSet(t, client, orgTest, VariableSetCreateOptions{})
	defer vsTestCleanup2()

	t.Run("without list options", func(t *testing.T) {
		vsl, err := client.VariableSets.List(ctx, orgTest.Name, nil)
		require.NoError(t, err)
		require.NotEmpty(t, vsl.Items)
		assert.Contains(t, vsl.Items, vsTest1)
		assert.Contains(t, vsl.Items, vsTest2)

		t.Skip("paging not supported yet in API")
		assert.Equal(t, 1, vsl.CurrentPage)
		assert.Equal(t, 2, vsl.TotalCount)
	})

	t.Run("with list options", func(t *testing.T) {
		t.Skip("paging not supported yet in API")
		// Request a page number which is out of range. The result should
		// be successful, but return no results if the paging options are
		// properly passed along.
		vsl, err := client.VariableSets.List(ctx, orgTest.Name, &VariableSetListOptions{
			ListOptions: ListOptions{
				PageNumber: 999,
				PageSize:   100,
			},
		})
		require.NoError(t, err)
		assert.Empty(t, vsl.Items)
		assert.Equal(t, 999, vsl.CurrentPage)
		assert.Equal(t, 2, vsl.TotalCount)
	})

	t.Run("when Organization name is invalid ID", func(t *testing.T) {
		vsl, err := client.VariableSets.List(ctx, badIdentifier, nil)
		assert.Nil(t, vsl)
		assert.EqualError(t, err, ErrInvalidOrg.Error())
	})
}

func TestVariableSetsCreate(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	defer orgTestCleanup()

	t.Run("with valid options", func(t *testing.T) {
		options := VariableSetCreateOptions{
			Name:        String("varset"),
			Description: String("a variable set"),
			Global:      Bool(false),
		}

		vs, err := client.VariableSets.Create(ctx, orgTest.Name, &options)
		require.NoError(t, err)

		// Get refreshed view from the API
		refreshed, err := client.VariableSets.Read(ctx, vs.ID, nil)
		require.NoError(t, err)

		for _, item := range []*VariableSet{
			vs,
			refreshed,
		} {
			assert.NotEmpty(t, item.ID)
			assert.Equal(t, *options.Name, item.Name)
			assert.Equal(t, *options.Description, item.Description)
			assert.Equal(t, *options.Global, item.Global)
		}
	})

	t.Run("when options is missing name", func(t *testing.T) {
		vs, err := client.VariableSets.Create(ctx, "foo", &VariableSetCreateOptions{
			Global: Bool(true),
		})
		assert.Nil(t, vs)
		assert.EqualError(t, err, ErrRequiredName.Error())
	})

	t.Run("when options is missing global flag", func(t *testing.T) {
		vs, err := client.VariableSets.Create(ctx, "foo", &VariableSetCreateOptions{
			Name: String("foo"),
		})
		assert.Nil(t, vs)
		assert.EqualError(t, err, ErrRequiredGlobalFlag.Error())
	})
}

func TestVariableSetsRead(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	defer orgTestCleanup()

	vsTest, vsTestCleanup := createVariableSet(t, client, orgTest, VariableSetCreateOptions{})
	defer vsTestCleanup()

	t.Run("when the variable set exists", func(t *testing.T) {
		vs, err := client.VariableSets.Read(ctx, vsTest.ID, nil)
		require.NoError(t, err)
		assert.Equal(t, vsTest, vs)
	})

	t.Run("when variable set does not exist", func(t *testing.T) {
		vs, err := client.VariableSets.Read(ctx, "nonexisting", nil)
		assert.Nil(t, vs)
		assert.Error(t, err)
	})
}

func TestVariableSetsUpdate(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	defer orgTestCleanup()

	vsTest, _ := createVariableSet(t, client, orgTest, VariableSetCreateOptions{
		Name:        String("OriginalName"),
		Description: String("Original Description"),
		Global:      Bool(false),
	})

	t.Run("when updating a subset of values", func(t *testing.T) {
		options := VariableSetUpdateOptions{
			Name:        String("UpdatedName"),
			Description: String("Updated Description"),
			Global:      Bool(true),
		}

		vsAfter, err := client.VariableSets.Update(ctx, vsTest.ID, &options)
		require.NoError(t, err)

		assert.Equal(t, *options.Name, vsAfter.Name)
		assert.Equal(t, *options.Description, vsAfter.Description)
		assert.Equal(t, *options.Global, vsAfter.Global)
	})

	t.Run("when options has an invalid variable set ID", func(t *testing.T) {
		vsAfter, err := client.VariableSets.Update(ctx, badIdentifier, &VariableSetUpdateOptions{
			Name:        String("UpdatedName"),
			Description: String("Updated Description"),
			Global:      Bool(true),
		})
		assert.Nil(t, vsAfter)
		assert.EqualError(t, err, ErrInvalidVariableSetID.Error())
	})
}

func TestVariableSetsDelete(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	defer orgTestCleanup()

	// Do not defer cleanup since the next step in this test is to delete it
	vsTest, _ := createVariableSet(t, client, orgTest, VariableSetCreateOptions{})

	t.Run("with valid ID", func(t *testing.T) {
		err := client.VariableSets.Delete(ctx, vsTest.ID)
		require.NoError(t, err)

		// Try loading the variable set - it should fail.
		_, err = client.VariableSets.Read(ctx, vsTest.ID, nil)
		assert.Equal(t, ErrResourceNotFound, err)
	})

	t.Run("when ID is invalid", func(t *testing.T) {
		err := client.VariableSets.Delete(ctx, badIdentifier)
		assert.EqualError(t, err, ErrInvalidVariableSetID.Error())
	})
}

func TestVariableSetsApplyToAndRemoveFromWorkspaces(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	defer orgTestCleanup()

	vsTest, vsTestCleanup := createVariableSet(t, client, orgTest, VariableSetCreateOptions{})
	defer vsTestCleanup()

	wTest1, wTest1Cleanup := createWorkspace(t, client, orgTest)
	defer wTest1Cleanup()
	wTest2, wTest2Cleanup := createWorkspace(t, client, orgTest)
	defer wTest2Cleanup()

	t.Run("with first workspace added", func(t *testing.T) {
		options := VariableSetApplyToWorkspacesOptions{
			Workspaces: []*Workspace{wTest1},
		}

		err := client.VariableSets.ApplyToWorkspaces(ctx, vsTest.ID, &options)
		require.NoError(t, err)

		vsAfter, err := client.VariableSets.Read(ctx, vsTest.ID, nil)
		require.NoError(t, err)

		// Variable set should be applied to [wTest1]
		assert.Equal(t, 1, len(vsAfter.Workspaces))
		assert.Equal(t, wTest1.ID, vsAfter.Workspaces[0].ID)
	})

	t.Run("with second workspace added", func(t *testing.T) {
		options := VariableSetApplyToWorkspacesOptions{
			Workspaces: []*Workspace{wTest2},
		}

		err := client.VariableSets.ApplyToWorkspaces(ctx, vsTest.ID, &options)
		require.NoError(t, err)

		vsAfter, err := client.VariableSets.Read(ctx, vsTest.ID, nil)
		require.NoError(t, err)

		// Variable set should be applied to [wTest1, wTest2]
		assert.Equal(t, 2, len(vsAfter.Workspaces))
		assert.Equal(t, wTest1.ID, vsAfter.Workspaces[0].ID)
		assert.Equal(t, wTest2.ID, vsAfter.Workspaces[1].ID)
	})

	t.Run("with first workspace removed", func(t *testing.T) {
		options := VariableSetRemoveFromWorkspacesOptions{
			Workspaces: []*Workspace{wTest1},
		}

		err := client.VariableSets.RemoveFromWorkspaces(ctx, vsTest.ID, &options)
		require.NoError(t, err)

		vsAfter, err := client.VariableSets.Read(ctx, vsTest.ID, nil)
		require.NoError(t, err)

		// Variable set should be applied to [wTest2]
		assert.Equal(t, 1, len(vsAfter.Workspaces))
		assert.Equal(t, wTest2.ID, vsAfter.Workspaces[0].ID)
	})

	t.Run("when variable set ID is invalid", func(t *testing.T) {
		applyOptions := VariableSetApplyToWorkspacesOptions{
			Workspaces: []*Workspace{wTest1},
		}

		err := client.VariableSets.ApplyToWorkspaces(ctx, badIdentifier, &applyOptions)
		assert.EqualError(t, err, ErrInvalidVariableSetID.Error())

		removeOptions := VariableSetRemoveFromWorkspacesOptions{
			Workspaces: []*Workspace{wTest1},
		}
		err = client.VariableSets.RemoveFromWorkspaces(ctx, badIdentifier, &removeOptions)
		assert.EqualError(t, err, ErrInvalidVariableSetID.Error())
	})

	t.Run("when workspace ID is invalid", func(t *testing.T) {
		badWorkspace := &Workspace{
			ID: badIdentifier,
		}

		applyOptions := VariableSetApplyToWorkspacesOptions{
			Workspaces: []*Workspace{badWorkspace},
		}

		err := client.VariableSets.ApplyToWorkspaces(ctx, vsTest.ID, &applyOptions)
		assert.EqualError(t, err, ErrRequiredWorkspaceID.Error())

		removeOptions := VariableSetRemoveFromWorkspacesOptions{
			Workspaces: []*Workspace{badWorkspace},
		}

		err = client.VariableSets.RemoveFromWorkspaces(ctx, vsTest.ID, &removeOptions)
		assert.EqualError(t, err, ErrRequiredWorkspaceID.Error())
	})
}

func TestVariableSetsUpdateWorkspaces(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	defer orgTestCleanup()

	vsTest, vsTestCleanup := createVariableSet(t, client, orgTest, VariableSetCreateOptions{})
	defer vsTestCleanup()

	wTest, wTestCleanup := createWorkspace(t, client, orgTest)
	defer wTestCleanup()

	t.Run("with valid workspaces", func(t *testing.T) {
		options := VariableSetUpdateWorkspacesOptions{
			Workspaces: []*Workspace{wTest},
		}

		vsAfter, err := client.VariableSets.UpdateWorkspaces(ctx, vsTest.ID, &options)
		require.NoError(t, err)

		assert.Equal(t, len(options.Workspaces), len(vsAfter.Workspaces))
		assert.Equal(t, options.Workspaces[0].ID, vsAfter.Workspaces[0].ID)

		options = VariableSetUpdateWorkspacesOptions{
			Workspaces: []*Workspace{},
		}

		vsAfter, err = client.VariableSets.UpdateWorkspaces(ctx, vsTest.ID, &options)
		require.NoError(t, err)

		assert.Equal(t, len(options.Workspaces), len(vsAfter.Workspaces))
	})
}
