package tfe

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAdminSettings_SMTP_Read(t *testing.T) {
	skipIfCloud(t)

	client := testClient(t)
	ctx := context.Background()

	smtpSettings, err := client.Admin.Settings.SMTP.Read(ctx)
	require.NoError(t, err)

	assert.Equal(t, "smtp", smtpSettings.ID)
	assert.NotNil(t, smtpSettings.Enabled)
	assert.NotNil(t, smtpSettings.Host)
	assert.NotNil(t, smtpSettings.Port)
	assert.NotNil(t, smtpSettings.Sender)
	assert.NotNil(t, smtpSettings.Auth)
	assert.NotNil(t, smtpSettings.Username)
}

func TestAdminSettings_SMTP_Update(t *testing.T) {
	skipIfCloud(t)

	client := testClient(t)
	ctx := context.Background()

	enabled := false
	smtpSettings, err := client.Admin.Settings.SMTP.Update(ctx, AdminSMTPSettingsUpdateOptions{
		Enabled: Bool(enabled),
		Auth:    SMTPAuthValue(SMTPAuthNone),
	})

	require.NoError(t, err)
	assert.Equal(t, enabled, smtpSettings.Enabled)
}
