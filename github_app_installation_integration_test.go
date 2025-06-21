// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package tfe

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGHAInstallationList(t *testing.T) {
	gHAInstallationID := os.Getenv("GITHUB_APP_INSTALLATION_ID")

	if gHAInstallationID == "" {
		t.Skip("Export a valid GITHUB_APP_INSTALLATION_ID before running this test!")
	}
	client := testClient(t)
	ctx := context.Background()

	t.Run("without list options", func(t *testing.T) {
		_, err := client.GHAInstallations.List(ctx, nil)
		assert.NoError(t, err)
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
		assert.NotEmpty(t, ghais.IconUrl)
		assert.NotEmpty(t, ghais.ID)
		assert.NotEmpty(t, ghais.InstallationID)
		assert.NotEmpty(t, ghais.InstallationType)
		assert.NotEmpty(t, ghais.InstallationURL)
		assert.NotEmpty(t, ghais.Name)
		assert.Equal(t, *ghais.ID, gHAInstallationID)
	})
}
