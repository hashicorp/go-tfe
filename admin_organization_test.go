package tfe

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAdminOrganizations_Read(t *testing.T) {
	skipIfNotEnterprise(t)

	client := testClient(t)
	ctx := context.Background()

	t.Run("it fails to read an organization with an invalid id", func(t *testing.T) {
		adminOrg, err := client.Admin.Organizations.Read(ctx, "")
		require.Error(t, err)
		assert.EqualError(t, err, "invalid value for organization")
		assert.Nil(t, adminOrg)
	})

	t.Run("it fails to read an organization with an bad org name", func(t *testing.T) {
		orgName := fmt.Sprintf("bad-%s", randomString(t))
		adminOrg, err := client.Admin.Organizations.Read(ctx, orgName)
		require.Error(t, err)
		assert.EqualError(t, err, ErrResourceNotFound.Error())
		assert.Nil(t, adminOrg)
	})

	t.Run("it reads an organization successfully", func(t *testing.T) {
		org, orgTestCleanup := createOrganization(t, client)
		defer orgTestCleanup()

		adminOrg, err := client.Admin.Organizations.Read(ctx, org.Name)
		assert.NoError(t, err)
		assert.NotNilf(t, adminOrg, "Organization is not nil")
		assert.Equal(t, adminOrg.Name, org.Name)

		// attributes part of an AdminOrganization response that are not null
		assert.NotNilf(t, adminOrg.AccessBetaTools, "AccessBetaTools is not nil")
		assert.NotNilf(t, adminOrg.ExternalID, "ExternalID is not nil")
		assert.NotNilf(t, adminOrg.GlobalModuleSharing, "GlobalModuleSharing is not nil")
		assert.NotNilf(t, adminOrg.IsDisabled, "IsDisabled is not nil")
		assert.NotNilf(t, adminOrg.NotificationEmail, "NotificationEmail is not nil")
		assert.NotNilf(t, adminOrg.SsoEnabled, "SsoEnabled is not nil")
		assert.NotNilf(t, adminOrg.TerraformWorkerSudoEnabled, "TerraformWorkerSudoEnabledis not nil")
	})
}

func TestAdminOrganizations_Delete(t *testing.T) {
	skipIfNotEnterprise(t)

	client := testClient(t)
	ctx := context.Background()

	t.Run("it fails to delete an organization with an invalid id", func(t *testing.T) {
		err := client.Admin.Organizations.Delete(ctx, "")
		require.Error(t, err)
		assert.EqualError(t, err, "invalid value for organization")
	})

	t.Run("it fails to delete an organization with an bad org name", func(t *testing.T) {
		orgName := fmt.Sprintf("bad-%s", randomString(t))
		err := client.Admin.Organizations.Delete(ctx, orgName)
		require.Error(t, err)
		assert.EqualError(t, err, ErrResourceNotFound.Error())
	})

	t.Run("it deletes an organization successfully", func(t *testing.T) {
		originalOrg, _ := createOrganization(t, client)

		adminOrg, err := client.Admin.Organizations.Read(ctx, originalOrg.Name)
		assert.NoError(t, err)
		assert.NotNilf(t, adminOrg, "Organization is not nil")
		assert.Equal(t, adminOrg.Name, originalOrg.Name)

		err = client.Admin.Organizations.Delete(ctx, adminOrg.Name)
		assert.NoError(t, err)

		// Cannot find deleted org
		_, err = client.Admin.Organizations.Read(ctx, originalOrg.Name)
		assert.Error(t, err)
		assert.EqualError(t, err, ErrResourceNotFound.Error())
	})
}

func TestAdminOrganizations_Update(t *testing.T) {
	skipIfNotEnterprise(t)

	client := testClient(t)
	ctx := context.Background()

	t.Run("it fails to update an organization with an invalid id", func(t *testing.T) {
		_, err := client.Admin.Organizations.Update(ctx, "", AdminOrganizationUpdateOptions{})
		require.Error(t, err)
		assert.EqualError(t, err, "invalid value for organization")
	})

	t.Run("it fails to update an organization with an bad org name", func(t *testing.T) {
		orgName := fmt.Sprintf("bad-%s", randomString(t))
		_, err := client.Admin.Organizations.Update(ctx, orgName, AdminOrganizationUpdateOptions{})
		require.Error(t, err)
		assert.EqualError(t, err, ErrResourceNotFound.Error())
	})

	t.Run("fetches and updates organization", func(t *testing.T) {
		org, orgTestCleanup := createOrganization(t, client)
		defer orgTestCleanup()

		adminOrg, err := client.Admin.Organizations.Read(ctx, org.Name)
		assert.NoError(t, err)
		assert.NotNilf(t, adminOrg, "Org returned as nil")

		accessBetaTools := true
		globalModuleSharing := true
		isDisabled := false
		terraformBuildWorkerApplyTimeout := "24h"
		terraformBuildWorkerPlanTimeout := "24h"

		opts := AdminOrganizationUpdateOptions{
			AccessBetaTools:                  &accessBetaTools,
			GlobalModuleSharing:              &globalModuleSharing,
			IsDisabled:                       &isDisabled,
			TerraformBuildWorkerApplyTimeout: &terraformBuildWorkerApplyTimeout,
			TerraformBuildWorkerPlanTimeout:  &terraformBuildWorkerPlanTimeout,
		}

		fmt.Println(org.Name)
		adminOrg, err = client.Admin.Organizations.Update(ctx, org.Name, opts)
		assert.NotNilf(t, adminOrg, "Org returned as nil when it shouldn't be.")
		assert.NoError(t, err)

		assert.Equal(t, accessBetaTools, adminOrg.AccessBetaTools)
		assert.Equal(t, globalModuleSharing, adminOrg.GlobalModuleSharing)
		assert.Equal(t, isDisabled, adminOrg.IsDisabled)
		assert.Equal(t, terraformBuildWorkerApplyTimeout, adminOrg.TerraformBuildWorkerApplyTimeout)
		assert.Equal(t, terraformBuildWorkerPlanTimeout, adminOrg.TerraformBuildWorkerPlanTimeout)
		assert.Equal(t, false, adminOrg.TerraformWorkerSudoEnabled)

		isDisabled = true
		opts = AdminOrganizationUpdateOptions{
			IsDisabled: &isDisabled,
		}

		adminOrg, err = client.Admin.Organizations.Update(ctx, org.Name, opts)
		assert.NoError(t, err)
		assert.NotNilf(t, adminOrg, "Org returned as nil when it shouldn't be.")

		assert.Equal(t, adminOrg.IsDisabled, isDisabled)

		isDisabled = false
		opts = AdminOrganizationUpdateOptions{
			IsDisabled: &isDisabled,
		}

		adminOrg, err = client.Admin.Organizations.Update(ctx, org.Name, opts)
		assert.NoError(t, err)
		assert.NotNilf(t, adminOrg, "Org returned as nil when it shouldn't be.")

		assert.Equal(t, adminOrg.IsDisabled, isDisabled)
	})
}
