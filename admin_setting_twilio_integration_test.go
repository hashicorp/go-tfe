package tfe

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAdminSettings_Twilio_Read(t *testing.T) {
	skipUnlessEnterprise(t)

	client := testClient(t)
	ctx := context.Background()

	twilioSettings, err := client.Admin.Settings.Twilio.Read(ctx)
	require.NoError(t, err)

	assert.Equal(t, "twilio", twilioSettings.ID)
	assert.NotNil(t, twilioSettings.Enabled)
	assert.NotNil(t, twilioSettings.AccountSid)
	assert.NotNil(t, twilioSettings.FromNumber)
}

func TestAdminSettings_Twilio_Update(t *testing.T) {
	skipUnlessEnterprise(t)

	client := testClient(t)
	ctx := context.Background()

	twilioSettings, err := client.Admin.Settings.Twilio.Update(ctx, AdminTwilioSettingsUpdateOptions{
		Enabled: Bool(false),
	})

	require.NoError(t, err)
	assert.Equal(t, false, twilioSettings.Enabled)
}

func TestAdminSettings_Twilio_Verify(t *testing.T) {
	skipUnlessEnterprise(t)

	client := testClient(t)
	ctx := context.Background()

	err := client.Admin.Settings.Twilio.Verify(ctx, AdminTwilioSettingsVerifyOptions{})

	assert.Equal(t, err, ErrRequiredTestNumber)
}
