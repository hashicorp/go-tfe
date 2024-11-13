// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package tfe

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAdminSettings_SAML_Read(t *testing.T) {
	skipUnlessEnterprise(t)

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
	assert.NotNil(t, samlSettings.SignatureSigningMethod)
	assert.NotNil(t, samlSettings.SignatureDigestMethod)
}

func TestAdminSettings_SAML_Update(t *testing.T) {
	skipUnlessEnterprise(t)

	client := testClient(t)
	ctx := context.Background()

	_, err := client.Admin.Settings.SAML.Read(ctx)
	require.NoError(t, err)

	enabled := false
	debug := false

	samlSettings, err := client.Admin.Settings.SAML.Update(ctx, AdminSAMLSettingsUpdateOptions{
		Enabled: Bool(enabled),
		Debug:   Bool(debug),
	})
	require.NoError(t, err)
	assert.Equal(t, enabled, samlSettings.Enabled)
	assert.Equal(t, debug, samlSettings.Debug)
	assert.Empty(t, samlSettings.PrivateKey)

	t.Run("with certificate defined", func(t *testing.T) {
		cert := "testCert"
		pKey := "testPrivateKey"
		signatureSigningMethod := "SHA1"
		signatureDigestMethod := "SHA1"
		samlSettingsUpd, err := client.Admin.Settings.SAML.Update(ctx, AdminSAMLSettingsUpdateOptions{
			Certificate:            String(cert),
			PrivateKey:             String(pKey),
			IDPCert:                String(cert),
			SLOEndpointURL:         String("https://example.com/slo"),
			SSOEndpointURL:         String("https://example.com/sso"),
			SignatureSigningMethod: String(signatureSigningMethod),
			SignatureDigestMethod:  String(signatureDigestMethod),
		})
		require.NoError(t, err)
		assert.Equal(t, cert, samlSettingsUpd.Certificate)
		assert.NotNil(t, samlSettingsUpd.PrivateKey)
		assert.Equal(t, signatureSigningMethod, samlSettingsUpd.SignatureSigningMethod)
		assert.Equal(t, signatureDigestMethod, samlSettingsUpd.SignatureDigestMethod)
	})

	t.Run("with team management enabled", func(t *testing.T) {
		cert := "testCert"
		pKey := "testPrivateKey"
		signatureSigningMethod := "SHA1"
		signatureDigestMethod := "SHA1"

		samlSettingsUpd, err := client.Admin.Settings.SAML.Update(ctx, AdminSAMLSettingsUpdateOptions{
			Enabled:                Bool(true),
			TeamManagementEnabled:  Bool(true),
			Certificate:            String(cert),
			PrivateKey:             String(pKey),
			SignatureSigningMethod: String(signatureSigningMethod),
			SignatureDigestMethod:  String(signatureDigestMethod),
		})
		require.NoError(t, err)
		assert.True(t, samlSettingsUpd.TeamManagementEnabled)
	})

	t.Run("with invalid signature digest method", func(t *testing.T) {
		_, err := client.Admin.Settings.SAML.Update(ctx, AdminSAMLSettingsUpdateOptions{
			AuthnRequestsSigned:   Bool(true),
			SignatureDigestMethod: String("SHA1234"),
		})
		require.Error(t, err)
	})

	t.Run("with invalid signature signing method", func(t *testing.T) {
		_, err := client.Admin.Settings.SAML.Update(ctx, AdminSAMLSettingsUpdateOptions{
			AuthnRequestsSigned:    Bool(true),
			SignatureSigningMethod: String("SHA1234"),
		})
		require.Error(t, err)
	})
}

func TestAdminSettings_SAML_RevokeIdpCert(t *testing.T) {
	skipUnlessEnterprise(t)

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
