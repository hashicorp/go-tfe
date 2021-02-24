package tfe

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTeamAccessesList(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	defer orgTestCleanup()

	wTest, _ := createWorkspace(t, client, orgTest)

	tmTest1, tmTest1Cleanup := createTeam(t, client, orgTest)
	defer tmTest1Cleanup()
	tmTest2, tmTest2Cleanup := createTeam(t, client, orgTest)
	defer tmTest2Cleanup()

	taTest1, taTest1Cleanup := createTeamAccess(t, client, tmTest1, wTest, orgTest)
	defer taTest1Cleanup()
	taTest2, taTest2Cleanup := createTeamAccess(t, client, tmTest2, wTest, orgTest)
	defer taTest2Cleanup()

	t.Run("with valid options", func(t *testing.T) {
		tal, err := client.TeamAccess.List(ctx, TeamAccessListOptions{
			WorkspaceID: String(wTest.ID),
		})
		require.NoError(t, err)
		assert.Contains(t, tal.Items, taTest1)
		assert.Contains(t, tal.Items, taTest2)

		t.Skip("paging not supported yet in API")
		assert.Equal(t, 1, tal.CurrentPage)
		assert.Equal(t, 2, tal.TotalCount)
	})

	t.Run("with list options", func(t *testing.T) {
		t.Skip("paging not supported yet in API")
		// Request a page number which is out of range. The result should
		// be successful, but return no results if the paging options are
		// properly passed along.
		tal, err := client.TeamAccess.List(ctx, TeamAccessListOptions{
			ListOptions: ListOptions{
				PageNumber: 999,
				PageSize:   100,
			},
		})
		require.NoError(t, err)
		assert.Empty(t, tal.Items)
		assert.Equal(t, 999, tal.CurrentPage)
		assert.Equal(t, 2, tal.TotalCount)
	})

	t.Run("without list options", func(t *testing.T) {
		tal, err := client.TeamAccess.List(ctx, TeamAccessListOptions{})
		assert.Nil(t, tal)
		assert.EqualError(t, err, "workspace ID is required")
	})

	t.Run("without a valid workspace ID", func(t *testing.T) {
		tal, err := client.TeamAccess.List(ctx, TeamAccessListOptions{
			WorkspaceID: String(badIdentifier),
		})
		assert.Nil(t, tal)
		assert.EqualError(t, err, ErrInvalidWorkspaceID.Error())
	})
}

func TestTeamAccessesAdd(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	defer orgTestCleanup()

	wTest, _ := createWorkspace(t, client, orgTest)

	tmTest, tmTestCleanup := createTeam(t, client, orgTest)
	defer tmTestCleanup()

	t.Run("with valid options", func(t *testing.T) {
		options := TeamAccessAddOptions{
			Access:    Access(AccessAdmin),
			Team:      tmTest,
			Workspace: wTest,
		}

		ta, err := client.TeamAccess.Add(ctx, options)
		defer client.TeamAccess.Remove(ctx, ta.ID)

		require.NoError(t, err)

		// Get a refreshed view from the API.
		refreshed, err := client.TeamAccess.Read(ctx, ta.ID)
		require.NoError(t, err)

		for _, item := range []*TeamAccess{
			ta,
			refreshed,
		} {
			assert.NotEmpty(t, item.ID)
			assert.Equal(t, *options.Access, item.Access)
		}
	})

	t.Run("with valid custom options", func(t *testing.T) {
		options := TeamAccessAddOptions{
			Access:        Access(AccessCustom),
			Runs:          RunsPermission(RunsPermissionRead),
			StateVersions: StateVersionsPermission(StateVersionsPermissionNone),
			Team:          tmTest,
			Workspace:     wTest,
		}

		ta, err := client.TeamAccess.Add(ctx, options)
		defer client.TeamAccess.Remove(ctx, ta.ID)

		require.NoError(t, err)

		// Get a refreshed view from the API.
		refreshed, err := client.TeamAccess.Read(ctx, ta.ID)
		require.NoError(t, err)

		for _, item := range []*TeamAccess{
			ta,
			refreshed,
		} {
			assert.NotEmpty(t, item.ID)
			assert.Equal(t, *options.Access, item.Access)
		}
	})

	t.Run("with invalid custom options", func(t *testing.T) {
		options := TeamAccessAddOptions{
			Access:    Access(AccessRead),
			Runs:      RunsPermission(RunsPermissionApply),
			Team:      tmTest,
			Workspace: wTest,
		}

		_, err := client.TeamAccess.Add(ctx, options)

		assert.EqualError(t, err, "invalid attribute\n\nRuns is read-only when access level is 'read'; use the 'custom' access level to set this attribute.")
	})

	t.Run("when the team already has access", func(t *testing.T) {
		_, taTestCleanup := createTeamAccess(t, client, tmTest, wTest, nil)
		defer taTestCleanup()

		options := TeamAccessAddOptions{
			Access:    Access(AccessAdmin),
			Team:      tmTest,
			Workspace: wTest,
		}

		_, err := client.TeamAccess.Add(ctx, options)
		assert.Error(t, err)
	})

	t.Run("when options is missing access", func(t *testing.T) {
		ta, err := client.TeamAccess.Add(ctx, TeamAccessAddOptions{
			Team:      tmTest,
			Workspace: wTest,
		})
		assert.Nil(t, ta)
		assert.EqualError(t, err, "access is required")
	})

	t.Run("when options is missing team", func(t *testing.T) {
		ta, err := client.TeamAccess.Add(ctx, TeamAccessAddOptions{
			Access:    Access(AccessAdmin),
			Workspace: wTest,
		})
		assert.Nil(t, ta)
		assert.EqualError(t, err, "team is required")
	})

	t.Run("when options is missing workspace", func(t *testing.T) {
		ta, err := client.TeamAccess.Add(ctx, TeamAccessAddOptions{
			Access: Access(AccessAdmin),
			Team:   tmTest,
		})
		assert.Nil(t, ta)
		assert.EqualError(t, err, "workspace is required")
	})
}

func TestTeamAccessesRead(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	taTest, taTestCleanup := createTeamAccess(t, client, nil, nil, nil)
	defer taTestCleanup()

	t.Run("when the team access exists", func(t *testing.T) {
		ta, err := client.TeamAccess.Read(ctx, taTest.ID)
		require.NoError(t, err)

		assert.Equal(t, AccessAdmin, ta.Access)

		t.Run("permission attributes are decoded", func(t *testing.T) {
			assert.Equal(t, RunsPermissionApply, ta.Runs)
			assert.Equal(t, VariablesPermissionWrite, ta.Variables)
			assert.Equal(t, StateVersionsPermissionWrite, ta.StateVersions)
			assert.Equal(t, SentinelMocksPermissionRead, ta.SentinelMocks)
			assert.Equal(t, true, ta.WorkspaceLocking)
		})

		t.Run("team relationship is decoded", func(t *testing.T) {
			assert.NotEmpty(t, ta.Team)
		})

		t.Run("workspace relationship is decoded", func(t *testing.T) {
			assert.NotEmpty(t, ta.Workspace)
		})
	})

	t.Run("when the team access does not exist", func(t *testing.T) {
		ta, err := client.TeamAccess.Read(ctx, "nonexisting")
		assert.Nil(t, ta)
		assert.Equal(t, err, ErrResourceNotFound)
	})

	t.Run("without a valid team access ID", func(t *testing.T) {
		ta, err := client.TeamAccess.Read(ctx, badIdentifier)
		assert.Nil(t, ta)
		assert.EqualError(t, err, "invalid value for team access ID")
	})
}

func TestTeamAccessesUpdate(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	defer orgTestCleanup()

	wTest, wTestCleanup := createWorkspace(t, client, orgTest)
	defer wTestCleanup()

	tmTest, tmTestCleanup := createTeam(t, client, orgTest)
	defer tmTestCleanup()

	taTest, taTestCleanup := createTeamAccess(t, client, tmTest, wTest, nil)
	defer taTestCleanup()

	t.Run("with valid attributes", func(t *testing.T) {
		options := TeamAccessUpdateOptions{
			Access: Access(AccessCustom),
			Runs:   RunsPermission(RunsPermissionPlan),
		}

		ta, err := client.TeamAccess.Update(ctx, taTest.ID, options)
		require.NoError(t, err)

		assert.Equal(t, ta.Access, AccessCustom)
		assert.Equal(t, ta.Runs, RunsPermissionPlan)
	})
}

func TestTeamAccessesRemove(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	defer orgTestCleanup()

	tmTest, tmTestCleanup := createTeam(t, client, orgTest)
	defer tmTestCleanup()

	taTest, _ := createTeamAccess(t, client, tmTest, nil, orgTest)

	t.Run("with valid options", func(t *testing.T) {
		err := client.TeamAccess.Remove(ctx, taTest.ID)
		require.NoError(t, err)

		// Try loading the workspace - it should fail.
		_, err = client.TeamAccess.Read(ctx, taTest.ID)
		assert.Equal(t, err, ErrResourceNotFound)
	})

	t.Run("when the team access does not exist", func(t *testing.T) {
		err := client.TeamAccess.Remove(ctx, taTest.ID)
		assert.Equal(t, err, ErrResourceNotFound)
	})

	t.Run("when the team access ID is invalid", func(t *testing.T) {
		err := client.TeamAccess.Remove(ctx, badIdentifier)
		assert.EqualError(t, err, "invalid value for team access ID")
	})
}
