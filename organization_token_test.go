package tfe

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOrganizationTokensGenerate(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	defer orgTestCleanup()

	var tkToken string
	t.Run("with valid options", func(t *testing.T) {
		tk, err := client.OrganizationTokens.Generate(ctx, orgTest.Name)
		require.NoError(t, err)
		require.NotEmpty(t, tk.Token)
		tkToken = tk.Token
	})

	t.Run("when a token already exists", func(t *testing.T) {
		tk, err := client.OrganizationTokens.Generate(ctx, orgTest.Name)
		require.NoError(t, err)
		require.NotEmpty(t, tk.Token)
		assert.NotEqual(t, tkToken, tk.Token)
	})

	t.Run("without valid organization", func(t *testing.T) {
		tk, err := client.OrganizationTokens.Generate(ctx, badIdentifier)
		assert.Nil(t, tk)
		assert.EqualError(t, err, "Invalid value for organization")
	})
}

func TestOrganizationTokensDelete(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	defer orgTestCleanup()

	createOrganizationToken(t, client, orgTest)

	t.Run("with valid options", func(t *testing.T) {
		err := client.OrganizationTokens.Delete(ctx, orgTest.Name)
		assert.NoError(t, err)
	})

	t.Run("when a token does not exist", func(t *testing.T) {
		err := client.OrganizationTokens.Delete(ctx, orgTest.Name)
		assert.EqualError(t, err, "Error: not found")
	})

	t.Run("without valid organization", func(t *testing.T) {
		err := client.OrganizationTokens.Delete(ctx, badIdentifier)
		assert.EqualError(t, err, "Invalid value for organization")
	})
}
