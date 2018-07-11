package tfe

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTeamMembersAdd(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	tmTest, tmTestCleanup := createTeam(t, client, nil)
	defer tmTestCleanup()

	t.Run("with valid options", func(t *testing.T) {
		t.Skip("the users API isn't available yet")
		options := TeamMemberAddOptions{
			Usernames: []string{"user1", "user2"},
		}

		err := client.TeamMembers.Add(ctx, tmTest.ID, options)
		assert.NoError(t, err)
	})

	t.Run("when options is missing usernames", func(t *testing.T) {
		err := client.TeamMembers.Add(ctx, tmTest.ID, TeamMemberAddOptions{})
		assert.EqualError(t, err, "Usernames is required")
	})

	t.Run("when usernames is empty", func(t *testing.T) {
		err := client.TeamMembers.Add(ctx, tmTest.ID, TeamMemberAddOptions{
			Usernames: []string{},
		})
		assert.EqualError(t, err, "Invalid value for usernames")
	})

	t.Run("when the team ID is invalid", func(t *testing.T) {
		err := client.TeamMembers.Add(ctx, badIdentifier, TeamMemberAddOptions{
			Usernames: []string{"user1"},
		})
		assert.EqualError(t, err, "Invalid value for team ID")
	})
}

func TestTeamMembersRemove(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	tmTest, tmTestCleanup := createTeam(t, client, nil)
	defer tmTestCleanup()

	t.Run("with valid options", func(t *testing.T) {
		t.Skip("the users API isn't available yet")
		options := TeamMemberRemoveOptions{
			Usernames: []string{"user1", "user2"},
		}

		err := client.TeamMembers.Remove(ctx, tmTest.ID, options)
		assert.NoError(t, err)
	})

	t.Run("when options is missing usernames", func(t *testing.T) {
		err := client.TeamMembers.Remove(ctx, tmTest.ID, TeamMemberRemoveOptions{})
		assert.EqualError(t, err, "Usernames is required")
	})

	t.Run("when usernames is empty", func(t *testing.T) {
		err := client.TeamMembers.Remove(ctx, tmTest.ID, TeamMemberRemoveOptions{
			Usernames: []string{},
		})
		assert.EqualError(t, err, "Invalid value for usernames")
	})

	t.Run("when the team ID is invalid", func(t *testing.T) {
		err := client.TeamMembers.Remove(ctx, badIdentifier, TeamMemberRemoveOptions{
			Usernames: []string{"user1"},
		})
		assert.EqualError(t, err, "Invalid value for team ID")
	})
}
