package tfe

import (
	"context"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"os"
	"testing"
)

func TestGHAInstallationList(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	t.Run("without list options", func(t *testing.T) {
		ghais, err := client.GHAInstallations.List(ctx, nil)
		require.NoError(t, err)
		assert.NotEmpty(t, ghais.Items)
	})
}

func TestGHAInstallationRead(t *testing.T) {

	ID := os.Getenv("GITHUB_APP_INSTALLATION_ID")

	if ID == "" {
		t.Skip("Export a valid GITHUB_APP_INSTALLATION_ID before running this test!")
	}

	var GHAInstallationID = ID
	client := testClient(t)
	ctx := context.Background()

	t.Run("when installation id exists", func(t *testing.T) {
		ghais, err := client.GHAInstallations.Read(ctx, GHAInstallationID)
		require.NoError(t, err)
		assert.NotEmpty(t, ghais.GHInstallationId)
	})
}
