//go:build integration
// +build integration

package tfe

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAdminSettings_General_Read(t *testing.T) {
	checkTestNodeEnv(t)
	skipIfCloud(t)

	client := testClient(t)
	ctx := context.Background()

	generalSettings, err := client.Admin.Settings.General.Read(ctx)
	require.NoError(t, err)

	assert.Equal(t, "general", generalSettings.ID)
	assert.NotNil(t, generalSettings.LimitUserOrganizationCreation)
	assert.NotNil(t, generalSettings.APIRateLimitingEnabled)
	assert.NotNil(t, generalSettings.APIRateLimit)
	assert.NotNil(t, generalSettings.SendPassingStatusesEnabled)
	assert.NotNil(t, generalSettings.AllowSpeculativePlansOnPR)
	assert.NotNil(t, generalSettings.RequireTwoFactorForAdmin)
	assert.NotNil(t, generalSettings.FairRunQueuingEnabled)
	assert.NotNil(t, generalSettings.LimitOrgsPerUser)
	assert.NotNil(t, generalSettings.DefaultOrgsPerUserCeiling)
	assert.NotNil(t, generalSettings.LimitWorkspacesPerOrg)
	assert.NotNil(t, generalSettings.DefaultWorkspacesPerOrgCeiling)
	assert.NotNil(t, generalSettings.TerraformBuildWorkerApplyTimeout)
	assert.NotNil(t, generalSettings.TerraformBuildWorkerPlanTimeout)
	assert.NotNil(t, generalSettings.DefaultRemoteStateAccess)
}

func TestAdminSettings_General_Update(t *testing.T) {
	checkTestNodeEnv(t)
	skipIfCloud(t)

	client := testClient(t)
	ctx := context.Background()

	generalSettings, err := client.Admin.Settings.General.Read(ctx)
	require.NoError(t, err)

	origLimitOrgCreation := generalSettings.LimitUserOrganizationCreation
	origAPIRateLimitEnabled := generalSettings.APIRateLimitingEnabled
	origAPIRateLimit := generalSettings.APIRateLimit
	origDefaultRemoteState := generalSettings.DefaultRemoteStateAccess

	limitOrgCreation := true
	apiRateLimitEnabled := true
	apiRateLimit := 50
	defaultRemoteStateAccess := false

	generalSettings, err = client.Admin.Settings.General.Update(ctx, AdminGeneralSettingsUpdateOptions{
		LimitUserOrgCreation:     Bool(limitOrgCreation),
		APIRateLimitingEnabled:   Bool(apiRateLimitEnabled),
		APIRateLimit:             Int(apiRateLimit),
		DefaultRemoteStateAccess: Bool(defaultRemoteStateAccess),
	})
	require.NoError(t, err)
	assert.Equal(t, limitOrgCreation, generalSettings.LimitUserOrganizationCreation)
	assert.Equal(t, apiRateLimitEnabled, generalSettings.APIRateLimitingEnabled)
	assert.Equal(t, apiRateLimit, generalSettings.APIRateLimit)
	assert.Equal(t, defaultRemoteStateAccess, generalSettings.DefaultRemoteStateAccess)

	// Undo Updates, revert back to original
	generalSettings, err = client.Admin.Settings.General.Update(ctx, AdminGeneralSettingsUpdateOptions{
		LimitUserOrgCreation:     Bool(origLimitOrgCreation),
		APIRateLimitingEnabled:   Bool(origAPIRateLimitEnabled),
		APIRateLimit:             Int(origAPIRateLimit),
		DefaultRemoteStateAccess: Bool(origDefaultRemoteState),
	})
	require.NoError(t, err)
	assert.Equal(t, origLimitOrgCreation, generalSettings.LimitUserOrganizationCreation)
	assert.Equal(t, origAPIRateLimitEnabled, generalSettings.APIRateLimitingEnabled)
	assert.Equal(t, origAPIRateLimit, generalSettings.APIRateLimit)
	assert.Equal(t, origDefaultRemoteState, generalSettings.DefaultRemoteStateAccess)
}
