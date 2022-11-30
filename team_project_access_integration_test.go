package tfe

import (
	"context"
	"github.com/stretchr/testify/assert"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTeamProjectAccessesList(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	defer orgTestCleanup()

	pTest, pTestCleanup := createProject(t, client, orgTest)
	defer pTestCleanup()

	tmTest1, tmTest1Cleanup := createTeam(t, client, orgTest)
	defer tmTest1Cleanup()
	tmTest2, tmTest2Cleanup := createTeam(t, client, orgTest)
	defer tmTest2Cleanup()

	tpaTest1, taTest1Cleanup := createTeamProjectAccess(t, client, tmTest1, pTest, orgTest)
	defer taTest1Cleanup()
	tpaTest2, taTest2Cleanup := createTeamProjectAccess(t, client, tmTest2, pTest, orgTest)
	defer taTest2Cleanup()

	t.Run("with valid options", func(t *testing.T) {
		tpal, err := client.TeamProjectAccess.List(ctx, &TeamProjectAccessListOptions{
			ProjectID: pTest.ID,
			ListOptions: ListOptions{
				PageNumber: 1,
				PageSize:   100,
			},
		})
		require.NoError(t, err)
		assert.Contains(t, tpal.Items, tpaTest1)
		assert.Contains(t, tpal.Items, tpaTest2)

		//t.Skip("paging not supported yet in API")
		assert.Equal(t, 1, tpal.CurrentPage)
		assert.Equal(t, 2, tpal.TotalCount)
	})

	t.Run("with list options", func(t *testing.T) {
		t.Skip("paging not supported yet in API")
		// Request a page number which is out of range. The result should
		// be successful, but return no results if the paging options are
		// properly passed along.
		tal, err := client.TeamProjectAccess.List(ctx, &TeamProjectAccessListOptions{
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

	t.Run("without TeamProjectAccessListOptions", func(t *testing.T) {
		tal, err := client.TeamProjectAccess.List(ctx, nil)
		assert.Nil(t, tal)
		assert.Equal(t, err, ErrRequiredTeamProjectAccessListOps)
	})

	t.Run("without projectID options", func(t *testing.T) {
		tal, err := client.TeamProjectAccess.List(ctx, &TeamProjectAccessListOptions{
			ListOptions: ListOptions{
				PageNumber: 2,
				PageSize:   25,
			},
		})
		assert.Nil(t, tal)
		assert.Equal(t, err, ErrRequiredProjectID)
	})

	t.Run("without a valid projectID", func(t *testing.T) {
		tal, err := client.TeamProjectAccess.List(ctx, &TeamProjectAccessListOptions{
			ProjectID: badIdentifier,
		})
		assert.Nil(t, tal)
		assert.EqualError(t, err, ErrInvalidProjectID.Error())
	})
}

func TestTeamProjectAccessesRead(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	defer orgTestCleanup()

	pTest, pTestCleanup := createProject(t, client, orgTest)
	defer pTestCleanup()

	tmTest, tmTestCleanup := createTeam(t, client, orgTest)
	defer tmTestCleanup()

	tpaTest, tpaTestCleanup := createTeamProjectAccess(t, client, tmTest, pTest, orgTest)
	defer tpaTestCleanup()

	t.Run("when the team access exists", func(t *testing.T) {
		ta, err := client.TeamProjectAccess.Read(ctx, tpaTest.ID)
		require.NoError(t, err)

		assert.Equal(t, TeamProjectAccessAdmin, ta.Access)

		t.Run("team relationship is decoded", func(t *testing.T) {
			assert.NotEmpty(t, ta.Team)
		})

		t.Run("project relationship is decoded", func(t *testing.T) {
			assert.NotEmpty(t, ta.Project)
		})
	})

	t.Run("when the team access does not exist", func(t *testing.T) {
		ta, err := client.TeamProjectAccess.Read(ctx, "nonexisting")
		assert.Nil(t, ta)
		assert.Equal(t, err, ErrResourceNotFound)
	})

	t.Run("without a valid team access ID", func(t *testing.T) {
		ta, err := client.TeamProjectAccess.Read(ctx, badIdentifier)
		assert.Nil(t, ta)
		assert.Equal(t, err, ErrInvalidTeamProjectAccessID)
	})
}

func TestTeamProjectAccessesAdd(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	defer orgTestCleanup()

	pTest, pTestCleanup := createProject(t, client, orgTest)
	defer pTestCleanup()

	tmTest, tmTestCleanup := createTeam(t, client, orgTest)
	defer tmTestCleanup()

	t.Run("with valid options", func(t *testing.T) {
		options := TeamProjectAccessAddOptions{
			Access:  ProjectAccess(TeamProjectAccessAdmin),
			Team:    tmTest,
			Project: pTest,
		}

		ta, err := client.TeamProjectAccess.Add(ctx, options)
		defer func() {
			err := client.TeamProjectAccess.Remove(ctx, ta.ID)
			if err != nil {
				t.Logf("error removing team access (%s): %s", ta.ID, err)
			}
		}()

		require.NoError(t, err)

		// Get a refreshed view from the API.
		refreshed, err := client.TeamProjectAccess.Read(ctx, ta.ID)
		require.NoError(t, err)

		for _, item := range []*TeamProjectAccess{
			ta,
			refreshed,
		} {
			assert.NotEmpty(t, item.ID)
			assert.Equal(t, *options.Access, item.Access)
		}
	})

	t.Run("when the team already has access to the project", func(t *testing.T) {
		_, taTestCleanup := createTeamProjectAccess(t, client, tmTest, pTest, nil)
		defer taTestCleanup()

		options := TeamProjectAccessAddOptions{
			Access:  ProjectAccess(TeamProjectAccessAdmin),
			Team:    tmTest,
			Project: pTest,
		}

		_, err := client.TeamProjectAccess.Add(ctx, options)
		assert.Error(t, err)
	})

	t.Run("when options is missing access", func(t *testing.T) {
		ta, err := client.TeamProjectAccess.Add(ctx, TeamProjectAccessAddOptions{
			Team:    tmTest,
			Project: pTest,
		})
		assert.Nil(t, ta)
		assert.Equal(t, err, ErrRequiredAccess)
	})

	t.Run("when options is missing team", func(t *testing.T) {
		ta, err := client.TeamProjectAccess.Add(ctx, TeamProjectAccessAddOptions{
			Access:  ProjectAccess(TeamProjectAccessAdmin),
			Project: pTest,
		})
		assert.Nil(t, ta)
		assert.Equal(t, err, ErrRequiredTeam)
	})

	t.Run("when options is missing project", func(t *testing.T) {
		ta, err := client.TeamProjectAccess.Add(ctx, TeamProjectAccessAddOptions{
			Access: ProjectAccess(TeamProjectAccessAdmin),
			Team:   tmTest,
		})
		assert.Nil(t, ta)
		assert.Equal(t, err, ErrRequiredProject)
	})
}

func TestTeamProjectAccessesUpdate(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	defer orgTestCleanup()

	pTest, pTestCleanup := createProject(t, client, orgTest)
	defer pTestCleanup()

	tmTest, tmTestCleanup := createTeam(t, client, orgTest)
	defer tmTestCleanup()

	tpaTest, tpaTestCleanup := createTeamProjectAccess(t, client, tmTest, pTest, orgTest)
	defer tpaTestCleanup()

	t.Run("with valid attributes", func(t *testing.T) {
		options := TeamProjectAccessUpdateOptions{
			Access: ProjectAccess(TeamProjectAccessRead),
		}

		ta, err := client.TeamProjectAccess.Update(ctx, tpaTest.ID, options)
		require.NoError(t, err)

		assert.Equal(t, ta.Access, TeamProjectAccessRead)
	})
}

func TestTeamProjectAccessesRemove(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	defer orgTestCleanup()

	pTest, pTestCleanup := createProject(t, client, orgTest)
	defer pTestCleanup()

	tmTest, tmTestCleanup := createTeam(t, client, orgTest)
	defer tmTestCleanup()

	tpaTest, _ := createTeamProjectAccess(t, client, tmTest, pTest, orgTest)

	t.Run("with valid options", func(t *testing.T) {
		err := client.TeamProjectAccess.Remove(ctx, tpaTest.ID)
		require.NoError(t, err)

		// Try loading the project - it should fail.
		_, err = client.TeamProjectAccess.Read(ctx, tpaTest.ID)
		assert.Equal(t, err, ErrResourceNotFound)
	})

	t.Run("when the team access does not exist", func(t *testing.T) {
		err := client.TeamProjectAccess.Remove(ctx, tpaTest.ID)
		assert.Equal(t, err, ErrResourceNotFound)
	})

	t.Run("when the team access ID is invalid", func(t *testing.T) {
		err := client.TeamProjectAccess.Remove(ctx, badIdentifier)
		assert.Equal(t, err, ErrInvalidTeamProjectAccessID)
	})
}
