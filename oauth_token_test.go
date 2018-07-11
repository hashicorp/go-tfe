package tfe

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOAuthTokensList(t *testing.T) {
	t.Skip("there isn't a way to create a token through the API")
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	defer orgTestCleanup()

	otTest1, _ := createOAuthToken(t, client, orgTest)
	otTest2, _ := createOAuthToken(t, client, orgTest)

	t.Run("with valid options", func(t *testing.T) {
		ots, err := client.OAuthTokens.List(ctx, orgTest.Name)
		require.NoError(t, err)

		assert.Contains(t, ots, otTest1)
		assert.Contains(t, ots, otTest2)

		t.Run("the OAuth client relationship is decoded correcly", func(t *testing.T) {
			for _, ot := range ots {
				assert.NotEmpty(t, ot.OAuthClient)
			}
		})
	})

	t.Run("without a valid organization", func(t *testing.T) {
		ots, err := client.OAuthTokens.List(ctx, badIdentifier)
		assert.Nil(t, ots)
		assert.EqualError(t, err, "Invalid value for organization")
	})
}
