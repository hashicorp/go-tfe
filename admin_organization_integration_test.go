//go:build integration
// +build integration

package tfe

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAdminOrganizations_List(t *testing.T) {
	skipIfCloud(t)

	client := testClient(t)
	ctx := context.Background()

	org, orgTestCleanup := createOrganization(t, client)
	defer orgTestCleanup()

	t.Run("with no list options", func(t *testing.T) {
		adminOrgList, err := client.Admin.Organizations.List(ctx, nil)
		require.NoError(t, err)

		// Given that org creation occurs on every test, the ordering is not
		// guaranteed. It may be that the `org` created in this test does not appear
		// in this list, so we want to test that the items are filled.
		assert.NotEmpty(t, adminOrgList.Items)
	})

	t.Run("with list options", func(t *testing.T) {
		// creating second org so that the query can only find the main org
		_, orgTestCleanup := createOrganization(t, client)
		defer orgTestCleanup()

		adminOrgList, err := client.Admin.Organizations.List(ctx, &AdminOrganizationListOptions{
			Query: org.Name,
		})
		require.NoError(t, err)
		assert.Equal(t, true, adminOrgItemsContainsName(adminOrgList.Items, org.Name))
		assert.Equal(t, 1, adminOrgList.CurrentPage)
		assert.Equal(t, 1, adminOrgList.TotalCount)
	})

	t.Run("with list options and org name that doesn't exist", func(t *testing.T) {
		randomName := "random-org-name"

		adminOrgList, err := client.Admin.Organizations.List(ctx, &AdminOrganizationListOptions{
			Query: randomName,
		})
		require.NoError(t, err)
		assert.Equal(t, false, adminOrgItemsContainsName(adminOrgList.Items, org.Name))
		assert.Equal(t, 1, adminOrgList.CurrentPage)
		assert.Equal(t, 0, adminOrgList.TotalCount)
	})

	t.Run("with owners included", func(t *testing.T) {
		adminOrgList, err := client.Admin.Organizations.List(ctx, &AdminOrganizationListOptions{
			Include: []AdminOrgIncludeOpt{AdminOrgOwners},
		})
		require.NoError(t, err)

		require.NotEmpty(t, adminOrgList.Items)
		assert.NotNil(t, adminOrgList.Items[0].Owners)
		assert.NotEmpty(t, adminOrgList.Items[0].Owners[0].Email)
	})
}

func TestAdminOrganizations_Read(t *testing.T) {
	skipIfCloud(t)

	client := testClient(t)
	ctx := context.Background()

	t.Run("it fails to read an organization with an invalid id", func(t *testing.T) {
		adminOrg, err := client.Admin.Organizations.Read(ctx, "")
		require.Error(t, err)
		assert.EqualError(t, err, ErrInvalidOrg.Error())
		assert.Nil(t, adminOrg)
	})

	t.Run("it returns ErrResourceNotFound for an organization that doesn't exist", func(t *testing.T) {
		orgName := fmt.Sprintf("non-existing-%s", randomString(t))
		adminOrg, err := client.Admin.Organizations.Read(ctx, orgName)
		require.Error(t, err)
		assert.EqualError(t, err, ErrResourceNotFound.Error())
		assert.Nil(t, adminOrg)
	})

	t.Run("it reads an organization successfully", func(t *testing.T) {
		org, orgTestCleanup := createOrganization(t, client)
		defer orgTestCleanup()

		adminOrg, err := client.Admin.Organizations.Read(ctx, org.Name)
		require.NoError(t, err)
		require.NotNilf(t, adminOrg, "Organization is not nil")
		assert.Equal(t, adminOrg.Name, org.Name)

		// attributes part of an AdminOrganization response that are not null
		assert.NotNilf(t, adminOrg.AccessBetaTools, "AccessBetaTools is not nil")
		assert.NotNilf(t, adminOrg.ExternalID, "ExternalID is not nil")
		assert.NotNilf(t, adminOrg.IsDisabled, "IsDisabled is not nil")
		assert.NotNilf(t, adminOrg.NotificationEmail, "NotificationEmail is not nil")
		assert.NotNilf(t, adminOrg.SsoEnabled, "SsoEnabled is not nil")
		assert.NotNilf(t, adminOrg.TerraformWorkerSudoEnabled, "TerraformWorkerSudoEnabledis not nil")
		assert.Nilf(t, adminOrg.WorkspaceLimit, "WorkspaceLimit is nil")
	})
}

func TestAdminOrganizations_Delete(t *testing.T) {
	skipIfCloud(t)

	client := testClient(t)
	ctx := context.Background()

	t.Run("it fails to delete an organization with an invalid id", func(t *testing.T) {
		err := client.Admin.Organizations.Delete(ctx, "")
		require.Error(t, err)
		assert.EqualError(t, err, ErrInvalidOrg.Error())
	})

	t.Run("it returns ErrResourceNotFound during an attempt to delete an organization that doesn't exist", func(t *testing.T) {
		orgName := fmt.Sprintf("non-existing-%s", randomString(t))
		err := client.Admin.Organizations.Delete(ctx, orgName)
		require.Error(t, err)
		assert.EqualError(t, err, ErrResourceNotFound.Error())
	})

	t.Run("it deletes an organization successfully", func(t *testing.T) {
		originalOrg, _ := createOrganization(t, client)

		adminOrg, err := client.Admin.Organizations.Read(ctx, originalOrg.Name)
		require.NoError(t, err)
		require.NotNil(t, adminOrg)
		assert.Equal(t, adminOrg.Name, originalOrg.Name)

		err = client.Admin.Organizations.Delete(ctx, adminOrg.Name)
		require.NoError(t, err)

		// Cannot find deleted org
		_, err = client.Admin.Organizations.Read(ctx, originalOrg.Name)
		assert.Error(t, err)
		assert.EqualError(t, err, ErrResourceNotFound.Error())
	})
}

func TestAdminOrganizations_ModuleConsumers(t *testing.T) {
	skipIfCloud(t)

	client := testClient(t)
	ctx := context.Background()

	t.Run("returns error if invalid org string is used", func(t *testing.T) {
		org1, org1TestCleanup := createOrganization(t, client)
		defer org1TestCleanup()

		err := client.Admin.Organizations.UpdateModuleConsumers(ctx, org1.Name, []string{"1Hello!"})
		assert.EqualError(t, err, ErrInvalidOrg.Error())
	})

	t.Run("can list and update module consumers", func(t *testing.T) {
		org1, org1TestCleanup := createOrganization(t, client)
		defer org1TestCleanup()

		org2, org2TestCleanup := createOrganization(t, client)
		defer org2TestCleanup()

		err := client.Admin.Organizations.UpdateModuleConsumers(ctx, org1.Name, []string{org2.Name})
		require.NoError(t, err)

		adminModuleConsumerList, err := client.Admin.Organizations.ListModuleConsumers(ctx, org1.Name, nil)
		require.NoError(t, err)

		assert.Equal(t, len(adminModuleConsumerList.Items), 1)
		assert.Equal(t, adminModuleConsumerList.Items[0].Name, org2.Name)

		org3, org3TestCleanup := createOrganization(t, client)
		defer org3TestCleanup()

		err = client.Admin.Organizations.UpdateModuleConsumers(ctx, org1.Name, []string{org3.Name})
		require.NoError(t, err)

		adminModuleConsumerList, err = client.Admin.Organizations.ListModuleConsumers(ctx, org1.Name, nil)
		require.NoError(t, err)

		assert.Equal(t, len(adminModuleConsumerList.Items), 1)
		assert.Equal(t, adminModuleConsumerList.Items[0].Name, org3.Name)
	})
}

func TestAdminOrganizations_Update(t *testing.T) {
	skipIfCloud(t)

	client := testClient(t)
	ctx := context.Background()

	t.Run("it fails to update an organization with an invalid id", func(t *testing.T) {
		_, err := client.Admin.Organizations.Update(ctx, "", AdminOrganizationUpdateOptions{})
		require.Error(t, err)
		assert.EqualError(t, err, ErrInvalidOrg.Error())
	})

	t.Run("it returns ErrResourceNotFound for during an update on an organization that doesn't exist", func(t *testing.T) {
		orgName := fmt.Sprintf("non-existing-%s", randomString(t))
		_, err := client.Admin.Organizations.Update(ctx, orgName, AdminOrganizationUpdateOptions{})
		require.Error(t, err)
		assert.EqualError(t, err, ErrResourceNotFound.Error())
	})

	t.Run("fetches and updates organization", func(t *testing.T) {
		org, orgTestCleanup := createOrganization(t, client)
		defer orgTestCleanup()

		adminOrg, err := client.Admin.Organizations.Read(ctx, org.Name)
		require.NoError(t, err)
		require.NotNilf(t, adminOrg, "Org returned as nil")

		accessBetaTools := true
		globalModuleSharing := false
		isDisabled := false
		terraformBuildWorkerApplyTimeout := "24h"
		terraformBuildWorkerPlanTimeout := "24h"
		terraformWorkerSudoEnabled := true

		opts := AdminOrganizationUpdateOptions{
			AccessBetaTools:                  &accessBetaTools,
			GlobalModuleSharing:              &globalModuleSharing,
			IsDisabled:                       &isDisabled,
			TerraformBuildWorkerApplyTimeout: &terraformBuildWorkerApplyTimeout,
			TerraformBuildWorkerPlanTimeout:  &terraformBuildWorkerPlanTimeout,
			TerraformWorkerSudoEnabled:       terraformWorkerSudoEnabled,
		}

		adminOrg, err = client.Admin.Organizations.Update(ctx, org.Name, opts)
		require.NotNilf(t, adminOrg, "Org returned as nil when it shouldn't be.")
		require.NoError(t, err)

		assert.Equal(t, accessBetaTools, adminOrg.AccessBetaTools)
		assert.Equal(t, adminOrg.GlobalModuleSharing, &globalModuleSharing)
		assert.Equal(t, isDisabled, adminOrg.IsDisabled)
		assert.Equal(t, terraformBuildWorkerApplyTimeout, adminOrg.TerraformBuildWorkerApplyTimeout)
		assert.Equal(t, terraformBuildWorkerPlanTimeout, adminOrg.TerraformBuildWorkerPlanTimeout)
		assert.Equal(t, terraformWorkerSudoEnabled, adminOrg.TerraformWorkerSudoEnabled)
		assert.Nil(t, adminOrg.WorkspaceLimit, "default workspace limit should be nil")

		isDisabled = true
		globalModuleSharing = true
		workspaceLimit := 42
		opts = AdminOrganizationUpdateOptions{
			GlobalModuleSharing: &globalModuleSharing,
			IsDisabled:          &isDisabled,
			WorkspaceLimit:      &workspaceLimit,
		}

		adminOrg, err = client.Admin.Organizations.Update(ctx, org.Name, opts)
		require.NoError(t, err)
		require.NotNilf(t, adminOrg, "Org returned as nil when it shouldn't be.")

		assert.Equal(t, adminOrg.GlobalModuleSharing, &globalModuleSharing)
		assert.Equal(t, adminOrg.IsDisabled, isDisabled)
		assert.Equal(t, &workspaceLimit, adminOrg.WorkspaceLimit)

		globalModuleSharing = false
		isDisabled = false
		workspaceLimit = 0
		opts = AdminOrganizationUpdateOptions{
			GlobalModuleSharing: &globalModuleSharing,
			IsDisabled:          &isDisabled,
			WorkspaceLimit:      &workspaceLimit,
		}

		adminOrg, err = client.Admin.Organizations.Update(ctx, org.Name, opts)
		require.NoError(t, err)
		require.NotNilf(t, adminOrg, "Org returned as nil when it shouldn't be.")

		assert.Equal(t, &globalModuleSharing, adminOrg.GlobalModuleSharing)
		assert.Equal(t, adminOrg.IsDisabled, isDisabled)

		assert.Equal(t, &workspaceLimit, adminOrg.WorkspaceLimit)
	})
}

func adminOrgItemsContainsName(items []*AdminOrganization, name string) bool {
	hasName := false
	for _, item := range items {
		if item.Name == name {
			hasName = true
			break
		}
	}

	return hasName
}
