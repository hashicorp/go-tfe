// Copyright IBM Corp. 2018, 2026
// SPDX-License-Identifier: MPL-2.0

package tfe

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAdminSettings_SCIM_Read(t *testing.T) {
	skipUnlessEnterprise(t)
	client := testClient(t)
	ctx := context.Background()

	t.Run("read scim settings with default values", func(t *testing.T) {
		scimSettings, err := client.Admin.Settings.SCIM.Read(ctx)
		require.NoError(t, err)

		assert.Equal(t, "scim", scimSettings.ID)
		assert.False(t, scimSettings.Enabled)
		assert.False(t, scimSettings.Paused)
		assert.Empty(t, scimSettings.SiteAdminGroupScimID)
		assert.Empty(t, scimSettings.SiteAdminGroupDisplayName)
	})
}

func TestAdminSettings_SCIM_Update(t *testing.T) {
	skipUnlessEnterprise(t)
	client := testClient(t)
	ctx := context.Background()

	enableSAML(ctx, t, client, true)
	defer enableSAML(ctx, t, client, false)

	scimClient := client.Admin.Settings.SCIM

	t.Run("enable scim settings", func(t *testing.T) {
		err := setSAMLProviderType(ctx, t, client, true)
		if err != nil {
			t.Fatalf("failed to set SAML provider type: %v", err)
		}
		require.NoError(t, err)
		defer cleanupSCIMSettings(ctx, t, client)

		scimSettings, err := scimClient.Update(ctx, AdminSCIMSettingUpdateOptions{Enabled: Bool(true)})
		require.NoError(t, err)

		assert.True(t, scimSettings.Enabled)
	})

	t.Run("pause scim settings", func(t *testing.T) {
		err := setSAMLProviderType(ctx, t, client, true)
		if err != nil {
			t.Fatalf("failed to set SAML provider type: %v", err)
		}
		require.NoError(t, err)
		defer cleanupSCIMSettings(ctx, t, client)

		_, err = scimClient.Update(ctx, AdminSCIMSettingUpdateOptions{
			Enabled: Bool(true),
		})
		require.NoError(t, err)

		testCases := []struct {
			name   string
			paused bool
		}{
			{"pause scim provisioning", true},
			{"unpause scim provisioning", false},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				_, err := scimClient.Update(ctx, AdminSCIMSettingUpdateOptions{Paused: &tc.paused})
				require.NoError(t, err)
				scimSettings, err := scimClient.Read(ctx)
				require.NoError(t, err)
				assert.Equal(t, tc.paused, scimSettings.Paused)
			})
		}
	})

	t.Run("update site admin group scim id", func(t *testing.T) {
		err := setSAMLProviderType(ctx, t, client, true)
		if err != nil {
			t.Fatalf("failed to set SAML provider type: %v", err)
		}
		require.NoError(t, err)
		defer cleanupSCIMSettings(ctx, t, client)

		_, err = scimClient.Update(ctx, AdminSCIMSettingUpdateOptions{Enabled: Bool(true)})
		require.NoError(t, err)

		scimToken := generateSCIMToken(ctx, t, client)
		scimGroupID := createSCIMGroup(ctx, t, client, "foo", scimToken)

		testCases := []struct {
			name        string
			scimGroupID string
			raiseError  bool
		}{
			{"link scim group to site admin role", scimGroupID, false},
			{"trying to link non-existent group - should raise error", "this-group-doesn't-exist", true},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				_, err := scimClient.Update(ctx, AdminSCIMSettingUpdateOptions{SiteAdminGroupScimID: &tc.scimGroupID})
				if tc.raiseError {
					require.Error(t, err)
					return
				}
				require.NoError(t, err)
				scimSettings, err := scimClient.Read(ctx)
				require.NoError(t, err)
				assert.Equal(t, tc.scimGroupID, scimSettings.SiteAdminGroupScimID)
				assert.Equal(t, "foo", scimSettings.SiteAdminGroupDisplayName)
			})
		}
	})
}

func TestAdminSettings_SCIM_Delete(t *testing.T) {
	skipUnlessEnterprise(t)
	client := testClient(t)
	ctx := context.Background()

	enableSAML(ctx, t, client, true)
	defer enableSAML(ctx, t, client, false)

	scimClient := client.Admin.Settings.SCIM

	t.Run("disable scim settings", func(t *testing.T) {
		err := setSAMLProviderType(ctx, t, client, true)
		if err != nil {
			t.Fatalf("failed to set SAML provider type: %v", err)
		}
		require.NoError(t, err)
		defer cleanupSCIMSettings(ctx, t, client)

		testCases := []struct {
			name          string
			isScimEnabled bool
		}{
			{"disable scim provisioning when it's already enabled", true},
			{"disable scim provisioning when it's already disabled - should not raise error", false},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				if tc.isScimEnabled {
					_, err := scimClient.Update(ctx, AdminSCIMSettingUpdateOptions{Enabled: Bool(true)})
					require.NoError(t, err)
				}

				err := scimClient.Delete(ctx)
				require.NoError(t, err)

				scimSettings, err := scimClient.Read(ctx)
				require.NoError(t, err)
				assert.False(t, scimSettings.Enabled)
			})
		}
	})
}

// cleanup scim settings by disabling scim provisioning and setting saml provider type to unknown.
func cleanupSCIMSettings(ctx context.Context, t *testing.T, client *Client) {
	scimSettings, err := client.Admin.Settings.SCIM.Read(ctx)
	if err == nil && scimSettings.Enabled {
		err = client.Admin.Settings.SCIM.Delete(ctx)
		if err != nil {
			t.Fatalf("failed to disable SCIM provisioning: %v", err)
		}
	}

	err = setSAMLProviderType(ctx, t, client, false)
	if err != nil {
		t.Fatalf("failed to set SAML provider type: %v", err)
	}
}

// generate a SCIM token for testing
func generateSCIMToken(ctx context.Context, t *testing.T, client *Client) string {
	expiredAt := time.Now().Add(30 * 24 * time.Hour)

	options := struct {
		Description *string    `jsonapi:"attr,description"`
		ExpiredAt   *time.Time `jsonapi:"attr,expired-at,iso8601"`
	}{
		Description: String("test-scim-token"),
		ExpiredAt:   &expiredAt,
	}
	req, err := client.NewRequest("POST", "admin/scim-tokens", &options)
	require.NoError(t, err)

	var res struct {
		Token string `jsonapi:"attr,token"`
	}
	err = req.Do(ctx, &res)
	require.NoError(t, err)

	return res.Token
}
