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

	t.Run("without list options", func(t *testing.T) {
		ghais, err := client.GHAInstallations.List(ctx, nil)
		require.NoError(t, err)
		assert.NotEmpty(t, ghais.Items)
	})
}
