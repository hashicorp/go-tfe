//go:build integration
// +build integration

package tfe

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAdminSettings_SAML_Read(t *testing.T) {
	skipIfNotCINode(t)
	skipIfCloud(t)

	client := testClient(t)
	ctx := context.Background()

	samlSettings, err := client.Admin.Settings.SAML.Read(ctx)
	require.NoError(t, err)

	assert.Equal(t, "saml", samlSettings.ID)
	assert.NotNil(t, samlSettings.Enabled)
	assert.NotNil(t, samlSettings.Debug)
	assert.NotNil(t, samlSettings.SLOEndpointURL)
	assert.NotNil(t, samlSettings.SSOEndpointURL)
	assert.NotNil(t, samlSettings.AttrUsername)
	assert.NotNil(t, samlSettings.AttrGroups)
	assert.NotNil(t, samlSettings.AttrSiteAdmin)
	assert.NotNil(t, samlSettings.SiteAdminRole)
	assert.NotNil(t, samlSettings.SSOAPITokenSessionTimeout)
	assert.NotNil(t, samlSettings.ACSConsumerURL)
	assert.NotNil(t, samlSettings.MetadataURL)
	assert.NotNil(t, samlSettings.TeamManagementEnabled)
	assert.NotNil(t, samlSettings.Certificate)
	assert.NotNil(t, samlSettings.AuthnRequestsSigned)
	assert.NotNil(t, samlSettings.WantAssertionsSigned)
	assert.NotNil(t, samlSettings.PrivateKey)
}

func TestAdminSettings_SAML_Update(t *testing.T) {
	skipIfNotCINode(t)
	skipIfCloud(t)

	client := testClient(t)
	ctx := context.Background()

	samlSettings, err := client.Admin.Settings.SAML.Read(ctx)
	require.NoError(t, err)

	enabled := false
	debug := false

	samlSettings, err = client.Admin.Settings.SAML.Update(ctx, AdminSAMLSettingsUpdateOptions{
		Enabled: Bool(enabled),
		Debug:   Bool(debug),
	})
	require.NoError(t, err)
	assert.Equal(t, enabled, samlSettings.Enabled)
	assert.Equal(t, debug, samlSettings.Debug)
}

func TestAdminSettings_SAML_RevokeIdpCert(t *testing.T) {
	skipIfNotCINode(t)
	skipIfCloud(t)

	client := testClient(t)
	ctx := context.Background()

	samlSettings, err := client.Admin.Settings.SAML.Read(ctx)
	require.NoError(t, err)
	if !samlSettings.Enabled {
		t.Skip("SAML is not enabled, skipping Revoke IDP Cert test.")
	}
	samlSettings, err = client.Admin.Settings.SAML.RevokeIdpCert(ctx)
	require.NoError(t, err)
	assert.NotNil(t, samlSettings.IDPCert)
}
