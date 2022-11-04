//go:build integration
// +build integration

package tfe

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTeamMembersList(t *testing.T) {
	// The TeamMembers.List() endpoint is available for everyone,
	// but this test uses extra functionality that is only available
	// to paid accounts. Organizations under a free account can
	// create team tokens, but they only have access to one team: the
	// owners team. This test creates new teams, and that feature is
	// unavaiable to paid accounts.
	skipIfFreeOnly(t)

	client := testClient(t)
	ctx := context.Background()

	tmTest, tmTestCleanup := createTeam(t, client, nil)
	defer tmTestCleanup()

	testAcct := fetchTestAccountDetails(t, client)

	options := TeamMemberAddOptions{
		Usernames: []string{testAcct.Username},
	}
	err := client.TeamMembers.Add(ctx, tmTest.ID, options)
	require.NoError(t, err)

	t.Run("with valid options", func(t *testing.T) {
		users, err := client.TeamMembers.List(ctx, tmTest.ID)
		require.NoError(t, err)
		require.Equal(t, 1, len(users))

		found := false
		for _, user := range users {
			if user.Username == testAcct.Username {
				found = true
				break
			}
		}

		assert.True(t, found)
	})

	t.Run("when the team ID is invalid", func(t *testing.T) {
		users, err := client.TeamMembers.List(ctx, badIdentifier)
		assert.Equal(t, err, ErrInvalidTeamID)
		assert.Nil(t, users)
	})
}

func TestTeamMembersAddWithInvalidOptions(t *testing.T) {
	skipIfFreeOnly(t)

	client := testClient(t)
	ctx := context.Background()

	tmTest, tmTestCleanup := createTeam(t, client, nil)
	defer tmTestCleanup()

	t.Run("when options is missing usernames and organization membership ids", func(t *testing.T) {
		err := client.TeamMembers.Add(ctx, tmTest.ID, TeamMemberAddOptions{})
		assert.Equal(t, err, ErrRequiredUsernameOrMembershipIds)
	})

	t.Run("when options has both usernames and organization membership ids", func(t *testing.T) {
		err := client.TeamMembers.Add(ctx, tmTest.ID, TeamMemberAddOptions{
			Usernames:                 []string{},
			OrganizationMembershipIDs: []string{},
		})
		assert.Equal(t, err, ErrRequiredOnlyOneField)
	})

	t.Run("when usernames is empty", func(t *testing.T) {
		err := client.TeamMembers.Add(ctx, tmTest.ID, TeamMemberAddOptions{
			Usernames: []string{},
		})
		assert.Equal(t, err, ErrInvalidUsernames)
	})

	t.Run("when organization membership ids is empty", func(t *testing.T) {
		err := client.TeamMembers.Add(ctx, tmTest.ID, TeamMemberAddOptions{
			OrganizationMembershipIDs: []string{},
		})
		assert.Equal(t, err, ErrInvalidMembershipIDs)
	})

	t.Run("when the team ID is invalid", func(t *testing.T) {
		err := client.TeamMembers.Add(ctx, badIdentifier, TeamMemberAddOptions{
			Usernames: []string{"user1"},
		})
		assert.Equal(t, err, ErrInvalidTeamID)
	})
}

func TestTeamMembersAddByUsername(t *testing.T) {
	skipIfFreeOnly(t)

	client := testClient(t)
	ctx := context.Background()

	tmTest, tmTestCleanup := createTeam(t, client, nil)
	defer tmTestCleanup()

	testAcct := fetchTestAccountDetails(t, client)

	t.Run("with valid username option", func(t *testing.T) {
		options := TeamMemberAddOptions{
			Usernames: []string{testAcct.Username},
		}

		err := client.TeamMembers.Add(ctx, tmTest.ID, options)
		require.NoError(t, err)

		users, err := client.TeamMembers.List(ctx, tmTest.ID)
		require.NoError(t, err)

		found := false
		for _, user := range users {
			if user.Username == testAcct.Username {
				found = true
				break
			}
		}

		assert.True(t, found)
	})
}

func TestTeamMembersAddByOrganizationMembers(t *testing.T) {
	skipIfFreeOnly(t)

	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	defer orgTestCleanup()

	tmTest, tmTestCleanup := createTeam(t, client, orgTest)
	defer tmTestCleanup()

	memTest, memTestCleanup := createOrganizationMembership(t, client, orgTest)
	defer memTestCleanup()

	t.Run("with valid membership IDs option", func(t *testing.T) {
		options := TeamMemberAddOptions{
			OrganizationMembershipIDs: []string{memTest.ID},
		}

		err := client.TeamMembers.Add(ctx, tmTest.ID, options)
		require.NoError(t, err)

		orgMemberships, err := client.TeamMembers.ListOrganizationMemberships(ctx, tmTest.ID)
		require.NoError(t, err)

		found := false
		for _, orgMembership := range orgMemberships {
			if orgMembership.ID == memTest.ID {
				found = true
				break
			}
		}

		assert.True(t, found)
	})
}

func TestTeamMembersRemoveWithInvalidOptions(t *testing.T) {
	skipIfFreeOnly(t)

	client := testClient(t)
	ctx := context.Background()

	tmTest, tmTestCleanup := createTeam(t, client, nil)
	defer tmTestCleanup()

	t.Run("when options is missing usernames and organization membership ids", func(t *testing.T) {
		err := client.TeamMembers.Remove(ctx, tmTest.ID, TeamMemberRemoveOptions{})
		assert.Equal(t, err, ErrRequiredUsernameOrMembershipIds)
	})

	t.Run("when options has both usernames and organization membership ids", func(t *testing.T) {
		err := client.TeamMembers.Remove(ctx, tmTest.ID, TeamMemberRemoveOptions{
			Usernames:                 []string{},
			OrganizationMembershipIDs: []string{},
		})
		assert.Equal(t, err, ErrRequiredOnlyOneField)
	})

	t.Run("when usernames is empty", func(t *testing.T) {
		err := client.TeamMembers.Remove(ctx, tmTest.ID, TeamMemberRemoveOptions{
			Usernames: []string{},
		})
		assert.Equal(t, err, ErrInvalidUsernames)
	})

	t.Run("when organization membership ids is empty", func(t *testing.T) {
		err := client.TeamMembers.Remove(ctx, tmTest.ID, TeamMemberRemoveOptions{
			OrganizationMembershipIDs: []string{},
		})
		assert.Equal(t, err, ErrInvalidMembershipIDs)
	})

	t.Run("when the team ID is invalid", func(t *testing.T) {
		err := client.TeamMembers.Remove(ctx, badIdentifier, TeamMemberRemoveOptions{
			Usernames: []string{"user1"},
		})
		assert.Equal(t, err, ErrInvalidTeamID)
	})
}

func TestTeamMembersRemoveByUsernames(t *testing.T) {
	skipIfFreeOnly(t)

	client := testClient(t)
	ctx := context.Background()

	tmTest, tmTestCleanup := createTeam(t, client, nil)
	defer tmTestCleanup()

	testAcct := fetchTestAccountDetails(t, client)

	options := TeamMemberAddOptions{
		Usernames: []string{testAcct.Username},
	}
	err := client.TeamMembers.Add(ctx, tmTest.ID, options)
	require.NoError(t, err)

	t.Run("with valid usernames", func(t *testing.T) {
		options := TeamMemberRemoveOptions{
			Usernames: []string{testAcct.Username},
		}

		err := client.TeamMembers.Remove(ctx, tmTest.ID, options)
		require.NoError(t, err)
	})
}

func TestTeamMembersRemoveByOrganizationMemberships(t *testing.T) {
	skipIfFreeOnly(t)

	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	defer orgTestCleanup()

	tmTest, tmTestCleanup := createTeam(t, client, orgTest)
	defer tmTestCleanup()

	memTest, memTestCleanup := createOrganizationMembership(t, client, orgTest)
	defer memTestCleanup()

	options := TeamMemberAddOptions{
		OrganizationMembershipIDs: []string{memTest.ID},
	}
	err := client.TeamMembers.Add(ctx, tmTest.ID, options)
	require.NoError(t, err)

	t.Run("with valid org membership ids", func(t *testing.T) {
		options := TeamMemberRemoveOptions{
			OrganizationMembershipIDs: []string{memTest.ID},
		}

		err := client.TeamMembers.Remove(ctx, tmTest.ID, options)
		require.NoError(t, err)
	})
}
