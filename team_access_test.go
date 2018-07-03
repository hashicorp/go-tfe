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
		tas, err := client.TeamAccess.List(ctx, TeamAccessListOptions{
			WorkspaceID: String(wTest.ID),
		})
		require.NoError(t, err)
		assert.Contains(t, tas, taTest1)
		assert.Contains(t, tas, taTest2)
	})

	t.Run("with list options", func(t *testing.T) {
		t.Skip("paging not supported yet in API")
		// Request a page number which is out of range. The result should
		// be successful, but return no results if the paging options are
		// properly passed along.
		tas, err := client.TeamAccess.List(ctx, TeamAccessListOptions{
			ListOptions: ListOptions{
				PageNumber: 999,
				PageSize:   100,
			},
		})
		require.NoError(t, err)
		assert.Empty(t, tas)
	})

	t.Run("without list options", func(t *testing.T) {
		tas, err := client.TeamAccess.List(ctx, TeamAccessListOptions{})
		assert.Nil(t, tas)
		assert.EqualError(t, err, "Workspace ID is required")
	})

	t.Run("without a valid workspace ID", func(t *testing.T) {
		tas, err := client.TeamAccess.List(ctx, TeamAccessListOptions{
			WorkspaceID: String(badIdentifier),
		})
		assert.Nil(t, tas)
		assert.EqualError(t, err, "Invalid value for workspace ID")
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
			Access:    Access(TeamAccessAdmin),
			Team:      tmTest,
			Workspace: wTest,
		}

		ta, err := client.TeamAccess.Add(ctx, options)
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

	t.Run("when the team already has access", func(t *testing.T) {
		options := TeamAccessAddOptions{
			Access:    Access(TeamAccessAdmin),
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
		assert.EqualError(t, err, "Access is required")
	})

	t.Run("when options is missing team", func(t *testing.T) {
		ta, err := client.TeamAccess.Add(ctx, TeamAccessAddOptions{
			Access:    Access(TeamAccessAdmin),
			Workspace: wTest,
		})
		assert.Nil(t, ta)
		assert.EqualError(t, err, "Team is required")
	})

	t.Run("when options is missing workspace", func(t *testing.T) {
		ta, err := client.TeamAccess.Add(ctx, TeamAccessAddOptions{
			Access: Access(TeamAccessAdmin),
			Team:   tmTest,
		})
		assert.Nil(t, ta)
		assert.EqualError(t, err, "Workspace is required")
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

		assert.Equal(t, TeamAccessAdmin, ta.Access)

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
		assert.EqualError(t, err, "Error: not found")
	})

	t.Run("without a valid team access ID", func(t *testing.T) {
		ta, err := client.TeamAccess.Read(ctx, badIdentifier)
		assert.Nil(t, ta)
		assert.EqualError(t, err, "Invalid value for team access ID")
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
		assert.EqualError(t, err, "Error: not found")
	})

	t.Run("when the team access does not exist", func(t *testing.T) {
		err := client.TeamAccess.Remove(ctx, taTest.ID)
		assert.EqualError(t, err, "Error: not found")
	})

	t.Run("when the team access ID is invalid", func(t *testing.T) {
		err := client.TeamAccess.Remove(ctx, badIdentifier)
		assert.EqualError(t, err, "Invalid value for team access ID")
	})
}
