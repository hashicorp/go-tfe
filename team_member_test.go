package tfe

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTeamMembersList(t *testing.T) {
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
		assert.EqualError(t, err, "invalid value for team ID")
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
		assert.EqualError(t, err, "usernames or organization membership ids are required")
	})

	t.Run("when options has both usernames and organization membership ids", func(t *testing.T) {
		err := client.TeamMembers.Add(ctx, tmTest.ID, TeamMemberAddOptions{
			Usernames:                 []string{},
			OrganizationMembershipIDs: []string{},
		})
		assert.EqualError(t, err, "only one of usernames or organization membership ids can be provided")
	})

	t.Run("when usernames is empty", func(t *testing.T) {
		err := client.TeamMembers.Add(ctx, tmTest.ID, TeamMemberAddOptions{
			Usernames: []string{},
		})
		assert.EqualError(t, err, "invalid value for usernames")
	})

	t.Run("when organization membership ids is empty", func(t *testing.T) {
		err := client.TeamMembers.Add(ctx, tmTest.ID, TeamMemberAddOptions{
			OrganizationMembershipIDs: []string{},
		})
		assert.EqualError(t, err, "invalid value for organization membership ids")
	})

	t.Run("when the team ID is invalid", func(t *testing.T) {
		err := client.TeamMembers.Add(ctx, badIdentifier, TeamMemberAddOptions{
			Usernames: []string{"user1"},
		})
		assert.EqualError(t, err, "invalid value for team ID")
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
		assert.EqualError(t, err, "usernames or organization membership ids are required")
	})

	t.Run("when options has both usernames and organization membership ids", func(t *testing.T) {
		err := client.TeamMembers.Remove(ctx, tmTest.ID, TeamMemberRemoveOptions{
			Usernames:                 []string{},
			OrganizationMembershipIDs: []string{},
		})
		assert.EqualError(t, err, "only one of usernames or organization membership ids can be provided")
	})

	t.Run("when usernames is empty", func(t *testing.T) {
		err := client.TeamMembers.Remove(ctx, tmTest.ID, TeamMemberRemoveOptions{
			Usernames: []string{},
		})
		assert.EqualError(t, err, "invalid value for usernames")
	})

	t.Run("when organization membership ids is empty", func(t *testing.T) {
		err := client.TeamMembers.Remove(ctx, tmTest.ID, TeamMemberRemoveOptions{
			OrganizationMembershipIDs: []string{},
		})
		assert.EqualError(t, err, "invalid value for organization membership ids")
	})

	t.Run("when the team ID is invalid", func(t *testing.T) {
		err := client.TeamMembers.Remove(ctx, badIdentifier, TeamMemberRemoveOptions{
			Usernames: []string{"user1"},
		})
		assert.EqualError(t, err, "invalid value for team ID")
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
		assert.NoError(t, err)
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
		assert.NoError(t, err)
	})
}
