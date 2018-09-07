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

	t.Run("without list options", func(t *testing.T) {
		options := OAuthTokenListOptions{}

		otl, err := client.OAuthTokens.List(ctx, orgTest.Name, options)
		require.NoError(t, err)

		assert.Contains(t, otl.Items, otTest1)
		assert.Contains(t, otl.Items, otTest2)
		assert.Equal(t, 1, otl.CurrentPage)
		assert.Equal(t, 2, otl.TotalCount)

		t.Run("the OAuth client relationship is decoded correcly", func(t *testing.T) {
			for _, ot := range otl.Items {
				assert.NotEmpty(t, ot.OAuthClient)
			}
		})
	})

	t.Run("with list options", func(t *testing.T) {
		// Request a page number which is out of range. The result should
		// be successful, but return no results if the paging options are
		// properly passed along.
		options := OAuthTokenListOptions{
			ListOptions: ListOptions{
				PageNumber: 999,
				PageSize:   100,
			},
		}

		otl, err := client.OAuthTokens.List(ctx, orgTest.Name, options)
		require.NoError(t, err)
		assert.Empty(t, otl.Items)
		assert.Equal(t, 999, otl.CurrentPage)
		assert.Equal(t, 2, otl.TotalCount)
	})

	t.Run("without a valid organization", func(t *testing.T) {
		options := OAuthTokenListOptions{}

		otl, err := client.OAuthTokens.List(ctx, badIdentifier, options)
		assert.Nil(t, otl)
		assert.EqualError(t, err, "Invalid value for organization")
	})
}
