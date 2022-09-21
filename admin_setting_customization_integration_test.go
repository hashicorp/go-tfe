//go:build integration
// +build integration

package tfe

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAdminSettings_Customization_Read(t *testing.T) {
	checkTestNodeEnv(t)
	skipIfCloud(t)

	client := testClient(t)
	ctx := context.Background()

	customizationSettings, err := client.Admin.Settings.Customization.Read(ctx)
	require.NoError(t, err)

	assert.Equal(t, "customization", customizationSettings.ID)
	assert.NotNil(t, customizationSettings.SupportEmail)
	assert.NotNil(t, customizationSettings.LoginHelp)
	assert.NotNil(t, customizationSettings.Footer)
	assert.NotNil(t, customizationSettings.Error)
	assert.NotNil(t, customizationSettings.NewUser)
}

func TestAdminSettings_Customization_Update(t *testing.T) {
	checkTestNodeEnv(t)
	skipIfCloud(t)

	client := testClient(t)
	ctx := context.Background()

	email := "test@example.com"
	loginHelp := "<div>Login Help</div>"
	footer := "<p>Custom Footer Content</p>"
	customError := "<em>Custom Error Instructions</em>"
	newUser := "New user? <a href=\"#\">Click Here</a>"

	customizationSettings, err := client.Admin.Settings.Customization.Update(ctx, AdminCustomizationSettingsUpdateOptions{
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
