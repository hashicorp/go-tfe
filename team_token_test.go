package tfe

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTeamTokensGenerate(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	tmTest, tmTestCleanup := createTeam(t, client, nil)
	defer tmTestCleanup()

	var tkToken string
	t.Run("with valid options", func(t *testing.T) {
		tk, err := client.TeamTokens.Generate(ctx, tmTest.ID)
		require.NoError(t, err)
		require.NotEmpty(t, tk.Token)
		tkToken = tk.Token
	})

	t.Run("when a token already exists", func(t *testing.T) {
		tk, err := client.TeamTokens.Generate(ctx, tmTest.ID)
		require.NoError(t, err)
		require.NotEmpty(t, tk.Token)
		assert.NotEqual(t, tkToken, tk.Token)
	})

	t.Run("without valid team ID", func(t *testing.T) {
		tk, err := client.TeamTokens.Generate(ctx, badIdentifier)
		assert.Nil(t, tk)
		assert.EqualError(t, err, "Invalid value for team ID")
	})
}

func TestTeamTokensDelete(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	tmTest, tmTestCleanup := createTeam(t, client, nil)
	defer tmTestCleanup()

	createTeamToken(t, client, tmTest)

	t.Run("with valid options", func(t *testing.T) {
		err := client.TeamTokens.Delete(ctx, tmTest.ID)
		assert.NoError(t, err)
	})

	t.Run("when a token does not exist", func(t *testing.T) {
		err := client.TeamTokens.Delete(ctx, tmTest.ID)
		assert.EqualError(t, err, "Error: not found")
	})

	t.Run("without valid team ID", func(t *testing.T) {
		err := client.TeamTokens.Delete(ctx, badIdentifier)
		assert.EqualError(t, err, "Invalid value for team ID")
	})
}
