// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package tfe

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAdminSettings_SMTP_Read(t *testing.T) {
	skipUnlessEnterprise(t)

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
	skipUnlessEnterprise(t)

	client := testClient(t)
	ctx := context.Background()

	enabled := true
	disabled := false

	t.Run("with Auth option defined", func(t *testing.T) {
		smtpSettings, err := client.Admin.Settings.SMTP.Update(ctx, AdminSMTPSettingsUpdateOptions{
			Enabled: Bool(disabled),
			Auth:    SMTPAuthValue(SMTPAuthNone),
		})

		require.NoError(t, err)
		assert.Equal(t, disabled, smtpSettings.Enabled)
	})
	t.Run("with no Auth option", func(t *testing.T) {
		smtpSettings, err := client.Admin.Settings.SMTP.Update(ctx, AdminSMTPSettingsUpdateOptions{
			Enabled:          Bool(enabled),
			TestEmailAddress: String("test@example.com"),
			Host:             String("123"),
			Port:             Int(123),
		})

		require.NoError(t, err)
		assert.Equal(t, SMTPAuthNone, smtpSettings.Auth)
		assert.Equal(t, enabled, smtpSettings.Enabled)
	})
	t.Run("with invalid Auth option", func(t *testing.T) {
		var SMTPAuthPlained SMTPAuthType = "plained"
		_, err := client.Admin.Settings.SMTP.Update(ctx, AdminSMTPSettingsUpdateOptions{
			Enabled: Bool(enabled),
			Auth:    &SMTPAuthPlained,
		})

		assert.Equal(t, err, ErrInvalidSMTPAuth)
	})
}
