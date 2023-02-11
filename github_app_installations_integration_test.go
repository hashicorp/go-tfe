package tfe

import (
	"context"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestGHAInstallationsList(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	defer orgTestCleanup()

	otTest1, otTest1Cleanup := createOAuthToken(t, client, orgTest)
	defer otTest1Cleanup()
	otTest2, otTest2Cleanup := createOAuthToken(t, client, orgTest)
	defer otTest2Cleanup()

	t.Run("without list options", func(t *testing.T) {
		otl, err := client.GHAInstallations.List(ctx, nil)
		require.NoError(t, err)

		assert.Contains(t, otl.Items, otTest1)
		assert.Contains(t, otl.Items, otTest2)

		t.Skip("paging not supported yet in API")
		assert.Equal(t, 1, otl.CurrentPage)
		assert.Equal(t, 2, otl.TotalCount)
	})
}
