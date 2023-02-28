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
		_, err := client.GHAInstallations.List(ctx, nil)
		// The API will not error when the github api installation exists.
		// assert.NoError(t, err)
		assert.ErrorContains(t, err, "no github app oauth token for user")
	})
}
func TestGHAInstallationRead(t *testing.T) {
	gHAInstallationID := os.Getenv("GITHUB_APP_INSTALLATION_ID")

	if gHAInstallationID == "" {
		t.Skip("Export a valid GITHUB_APP_INSTALLATION_ID before running this test!")
	}

	var GHAInstallationID = string(gHAInstallationID)
	client := testClient(t)
	ctx := context.Background()

	t.Run("when installation id exists", func(t *testing.T) {
		ghais, err := client.GHAInstallations.Read(ctx, GHAInstallationID)
		require.NoError(t, err)
		assert.NotEmpty(t, ghais.InstallationID)
		assert.NotEmpty(t, ghais.ID)
		assert.NotEmpty(t, ghais.Name)
	})
}
