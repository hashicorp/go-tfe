package tfe

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAdminSettings_GetGeneral(t *testing.T) {
	skipIfCloud(t)

	client := testClient(t)
	ctx := context.Background()

	generalSettings, err := client.Admin.Settings.GetGeneral(ctx)
	require.NoError(t, err)

	assert.Equal(t, "general", generalSettings.ID)
	assert.NotNil(t, generalSettings.LimitUserOrganizationCreation)
	assert.NotNil(t, generalSettings.APIRateLimitingEnabled)
	assert.NotNil(t, generalSettings.APIRateLimit)
	assert.NotNil(t, generalSettings.SendPassingStatusesEnabled)
	assert.NotNil(t, generalSettings.AllowSpeculativePlansOnPR)
}

func TestAdminSettings_UpdateGeneral(t *testing.T) {
	skipIfCloud(t)

	client := testClient(t)
	ctx := context.Background()

	generalSettings, err := client.Admin.Settings.GetGeneral(ctx)
	require.NoError(t, err)

	origLimitOrgCreation := generalSettings.LimitUserOrganizationCreation
	origAPIRateLimitEnabled := generalSettings.APIRateLimitingEnabled
	origAPIRateLimit := generalSettings.APIRateLimit

	limitOrgCreation := true
	apiRateLimitEnabled := true
	apiRateLimit := 50

	generalSettings, err = client.Admin.Settings.UpdateGeneral(ctx, AdminGeneralSettingsUpdateOptions{
		LimitUserOrgCreation:   Bool(limitOrgCreation),
		APIRateLimitingEnabled: Bool(apiRateLimitEnabled),
		APIRateLimit:           Int(apiRateLimit),
	})
	require.NoError(t, err)
	assert.Equal(t, limitOrgCreation, generalSettings.LimitUserOrganizationCreation)
	assert.Equal(t, apiRateLimitEnabled, generalSettings.APIRateLimitingEnabled)
	assert.Equal(t, apiRateLimit, generalSettings.APIRateLimit)

	// Undo Updates, revert back to original
	generalSettings, err = client.Admin.Settings.UpdateGeneral(ctx, AdminGeneralSettingsUpdateOptions{
		LimitUserOrgCreation:   Bool(origLimitOrgCreation),
		APIRateLimitingEnabled: Bool(origAPIRateLimitEnabled),
		APIRateLimit:           Int(origAPIRateLimit),
	})
	require.NoError(t, err)
	assert.Equal(t, origLimitOrgCreation, generalSettings.LimitUserOrganizationCreation)
	assert.Equal(t, origAPIRateLimitEnabled, generalSettings.APIRateLimitingEnabled)
	assert.Equal(t, origAPIRateLimit, generalSettings.APIRateLimit)
}

func TestAdminSettings_GetCostEstimation(t *testing.T) {
	skipIfCloud(t)

	client := testClient(t)
	ctx := context.Background()

	costEstimationSettings, err := client.Admin.Settings.GetCostEstimation(ctx)
	require.NoError(t, err)
	assert.Equal(t, "cost-estimation", costEstimationSettings.ID)
	assert.NotNil(t, costEstimationSettings.Enabled)
}

func TestAdminSettings_UpdateCostEstimation(t *testing.T) {
	skipIfCloud(t)

	client := testClient(t)
	ctx := context.Background()

	costEstimationSettings, err := client.Admin.Settings.GetCostEstimation(ctx)
	require.NoError(t, err)

	costEnabled := false
	costEstimationSettings, err = client.Admin.Settings.UpdateCostEstimation(ctx, AdminCostEstimationSettingOptions{
		Enabled: Bool(costEnabled),
	})
	require.NoError(t, err)
	assert.Equal(t, costEnabled, costEstimationSettings.Enabled)

	enableCostEstimation := true
	// Undo Updates, revert back to original
	costEstimationSettings, err = client.Admin.Settings.UpdateCostEstimation(ctx, AdminCostEstimationSettingOptions{
		Enabled: Bool(enableCostEstimation),
	})
	require.NoError(t, err)
	assert.Equal(t, enableCostEstimation, costEstimationSettings.Enabled)
}

func TestAdminSettings_GetSAML(t *testing.T) {
	skipIfCloud(t)

	client := testClient(t)
	ctx := context.Background()

	samlSettings, err := client.Admin.Settings.GetSAML(ctx)
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
}

func TestAdminSettings_UpdateSAML(t *testing.T) {
	skipIfCloud(t)

	client := testClient(t)
	ctx := context.Background()

	samlSettings, err := client.Admin.Settings.GetSAML(ctx)
	require.NoError(t, err)

	enabled := false
	debug := false

	samlSettings, err = client.Admin.Settings.UpdateSAML(ctx, AdminSAMLSettingsUpdateOptions{
		Enabled: Bool(enabled),
		Debug:   Bool(debug),
	})
	require.NoError(t, err)
	assert.Equal(t, enabled, samlSettings.Enabled)
	assert.Equal(t, debug, samlSettings.Debug)
}

func TestAdminSettings_RevokeSAMLIdpCert(t *testing.T) {
	skipIfCloud(t)

	client := testClient(t)
	ctx := context.Background()

	samlSettings, err := client.Admin.Settings.GetSAML(ctx)
	require.NoError(t, err)
	if !samlSettings.Enabled {
		t.Skip("SAML is not enabled, skipping Revoke IDP Cert test.")
	}
	samlSettings, err = client.Admin.Settings.RevokeSAMLIdpCert(ctx)
	require.NoError(t, err)
	assert.NotNil(t, samlSettings.IDPCert)
}

func TestAdminSettings_GetSMTP(t *testing.T) {
	skipIfCloud(t)

	client := testClient(t)
	ctx := context.Background()

	smtpSettings, err := client.Admin.Settings.GetSMTP(ctx)
	require.NoError(t, err)

	assert.Equal(t, "smtp", smtpSettings.ID)
	assert.NotNil(t, smtpSettings.Enabled)
	assert.NotNil(t, smtpSettings.Host)
	assert.NotNil(t, smtpSettings.Port)
	assert.NotNil(t, smtpSettings.Sender)
	assert.NotNil(t, smtpSettings.Auth)
	assert.NotNil(t, smtpSettings.Username)
}

func TestAdminSettings_UpdateSMTP(t *testing.T) {
	skipIfCloud(t)

	client := testClient(t)
	ctx := context.Background()

	enabled := false
	auth := string(SMTPAuthNone)
	smtpSettings, err := client.Admin.Settings.UpdateSMTP(ctx, AdminSMTPSettingsUpdateOptions{
		Enabled: Bool(enabled),
		Auth:    String(auth),
	})

	require.NoError(t, err)
	assert.Equal(t, enabled, smtpSettings.Enabled)
}

func TestAdminSettings_GetTwlio(t *testing.T) {
	skipIfCloud(t)

	client := testClient(t)
	ctx := context.Background()

	twilioSettings, err := client.Admin.Settings.GetTwilio(ctx)
	require.NoError(t, err)

	assert.Equal(t, "twilio", twilioSettings.ID)
	assert.NotNil(t, twilioSettings.Enabled)
	assert.NotNil(t, twilioSettings.AccountSid)
	assert.NotNil(t, twilioSettings.FromNumber)
}

func TestAdminSettings_UpdateTwilio(t *testing.T) {
	skipIfCloud(t)

	client := testClient(t)
	ctx := context.Background()

	twilioSettings, err := client.Admin.Settings.UpdateTwilio(ctx, AdminTwilioSettingsUpdateOptions{
		Enabled: Bool(false),
	})

	require.NoError(t, err)
	assert.Equal(t, false, twilioSettings.Enabled)
}

func TestAdminSettings_GetCustomization(t *testing.T) {
	skipIfCloud(t)

	client := testClient(t)
	ctx := context.Background()

	customizationSettings, err := client.Admin.Settings.GetCustomization(ctx)
	require.NoError(t, err)

	assert.Equal(t, "customization", customizationSettings.ID)
	assert.NotNil(t, customizationSettings.SupportEmail)
	assert.NotNil(t, customizationSettings.LoginHelp)
	assert.NotNil(t, customizationSettings.Footer)
	assert.NotNil(t, customizationSettings.Error)
	assert.NotNil(t, customizationSettings.NewUser)
}

func TestAdminSettings_UpdateCustomization(t *testing.T) {
	skipIfCloud(t)

	client := testClient(t)
	ctx := context.Background()

	email := "test@example.com"
	loginHelp := "<div>Login Help</div>"
	footer := "<p>Custom Footer Content</p>"
	customError := "<em>Custom Error Instructions</em>"
	newUser := "New user? <a href=\"#\">Click Here</a>"

	customizationSettings, err := client.Admin.Settings.UpdateCustomization(ctx, AdminCustomizationSettingsUpdateOptions{
		SupportEmail: String(email),
		LoginHelp:    String(loginHelp),
		Footer:       String(footer),
		Error:        String(customError),
		NewUser:      String(newUser),
	})

	require.NoError(t, err)
	assert.Equal(t, email, customizationSettings.SupportEmail)
	assert.Equal(t, loginHelp, customizationSettings.LoginHelp)
	assert.Equal(t, footer, customizationSettings.Footer)
	assert.Equal(t, customError, customizationSettings.Error)
	assert.Equal(t, newUser, customizationSettings.NewUser)
}
