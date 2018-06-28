package tfe

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOrganizationTokensGenerate(t *testing.T) {
	client := testClient(t)

	orgTest, orgTestCleanup := createOrganization(t, client)
	defer orgTestCleanup()

	var tkToken string
	t.Run("with valid options", func(t *testing.T) {
		tk, err := client.OrganizationTokens.Generate(orgTest.Name)
		require.NoError(t, err)
		require.NotEmpty(t, tk.Token)
		tkToken = tk.Token
	})

	t.Run("when a token already exists", func(t *testing.T) {
		tk, err := client.OrganizationTokens.Generate(orgTest.Name)
		require.NoError(t, err)
		require.NotEmpty(t, tk.Token)
		assert.NotEqual(t, tkToken, tk.Token)
	})

	t.Run("without valid organization", func(t *testing.T) {
		tk, err := client.OrganizationTokens.Generate(badIdentifier)
		assert.Nil(t, tk)
		assert.EqualError(t, err, "Invalid value for organization")
	})
}

func TestOrganizationTokensDelete(t *testing.T) {
	client := testClient(t)

	orgTest, orgTestCleanup := createOrganization(t, client)
	defer orgTestCleanup()

	createOrganizationToken(t, client, orgTest)

	t.Run("with valid options", func(t *testing.T) {
		err := client.OrganizationTokens.Delete(orgTest.Name)
		assert.NoError(t, err)
	})

	t.Run("when a token does not exist", func(t *testing.T) {
		err := client.OrganizationTokens.Delete(orgTest.Name)
		assert.EqualError(t, err, "Error: not found")
	})

	t.Run("without valid organization", func(t *testing.T) {
		err := client.OrganizationTokens.Delete(badIdentifier)
		assert.EqualError(t, err, "Invalid value for organization")
	})
}
