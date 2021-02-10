package tfe

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAdminOrganizationModulePartnershipsList(t *testing.T) {

	skipIfNotEnterprise(t)

	client := testClient(t)
	ctx := context.Background()

	org, orgTestCleanup := createOrganization(t, client)
	defer orgTestCleanup()

	t.Run("creates and destroys consumers", func(t *testing.T) {
		consumerList, err := client.Admin.Organizations.ListModuleConsumers(ctx, org.Name, OrganizationListOptions{})
		assert.Nilf(t, err, "Failed to read org consumers %v", err)
		assert.Empty(t, consumerList.Items)

		org2, orgTestCleanup2 := createOrganization(t, client)
		defer orgTestCleanup2()
		org3, orgTestCleanup3 := createOrganization(t, client)
		defer orgTestCleanup3()

		_listContains := func(name string, items []string) bool {
			for _, item := range items {
				if name == item {
					return true
				}
			}
			return false
		}
		_toNameList := func(orgs []*Organization) []string {
			names := []string{}
			for _, org := range orgs {
				names = append(names, org.Name)
			}
			return names
		}

		opts := ModuleConsumers{
			&org2.Name,
		}
		err = client.Admin.Organizations.UpdateModuleConsumers(ctx, org.Name, opts)
		assert.Nilf(t, err, "Failed to update consumers %v", err)

		consumerList, err = client.Admin.Organizations.ListModuleConsumers(ctx, org.Name, OrganizationListOptions{})
		assert.Nilf(t, err, "Failed to read org consumers %v", err)
		nameList := _toNameList(consumerList.Items)
		assert.Truef(t, _listContains(org2.Name, nameList), "Expected %v to be in returned list", org2.Name)

		opts = ModuleConsumers{
			&org2.Name,
			&org3.Name,
		}
		err = client.Admin.Organizations.UpdateModuleConsumers(ctx, org.Name, opts)
		assert.Nilf(t, err, "Failed to update consumers %v", err)
		nameList = _toNameList(consumerList.Items)

		consumerList, err = client.Admin.Organizations.ListModuleConsumers(ctx, org.Name, OrganizationListOptions{})
		assert.Nilf(t, err, "Failed to read org consumers %v", err)
		nameList = _toNameList(consumerList.Items)
		assert.Truef(t, _listContains(org2.Name, nameList), "Expected %v to be in returned list", org2.Name)
		assert.Truef(t, _listContains(org3.Name, nameList), "Expected %v to be in returned list", org3.Name)

		opts = ModuleConsumers{
			&org3.Name,
		}
		err = client.Admin.Organizations.UpdateModuleConsumers(ctx, org.Name, opts)
		assert.Nilf(t, err, "Failed to update consumers %v", err)

		consumerList, err = client.Admin.Organizations.ListModuleConsumers(ctx, org.Name, OrganizationListOptions{})
		assert.Nilf(t, err, "Failed to read org consumers %v", err)
		nameList = _toNameList(consumerList.Items)
		assert.Truef(t, _listContains(org3.Name, nameList), "Expected %v to be in returned list", org3.Name)

		opts = ModuleConsumers{}
		err = client.Admin.Organizations.UpdateModuleConsumers(ctx, org.Name, opts)
		assert.Nilf(t, err, "Failed to update consumers %v", err)

		consumerList, err = client.Admin.Organizations.ListModuleConsumers(ctx, org.Name, OrganizationListOptions{})
		assert.Nilf(t, err, "Failed to read org consumers %v", err)
		assert.Empty(t, consumerList.Items)
	})

	t.Run("lists consumers with pagination options", func(t *testing.T) {
		consumerList, err := client.Admin.Organizations.ListModuleConsumers(ctx, org.Name, OrganizationListOptions{})
		assert.Nilf(t, err, "Failed to read org consumers %v", err)
		assert.Empty(t, consumerList.Items)

		org2, orgTestCleanup2 := createOrganization(t, client)
		defer orgTestCleanup2()
		org3, orgTestCleanup3 := createOrganization(t, client)
		defer orgTestCleanup3()

		_listContains := func(name string, items []string) bool {
			for _, item := range items {
				if name == item {
					return true
				}
			}
			return false
		}
		_toNameList := func(orgs []*Organization) []string {
			names := []string{}
			for _, org := range orgs {
				names = append(names, org.Name)
			}
			return names
		}

		opts := ModuleConsumers{
			&org2.Name,
			&org3.Name,
		}
		err = client.Admin.Organizations.UpdateModuleConsumers(ctx, org.Name, opts)
		assert.Nilf(t, err, "Failed to update consumers %v", err)

		consumerList, err = client.Admin.Organizations.ListModuleConsumers(ctx, org.Name, OrganizationListOptions{
			ListOptions{
				PageSize:   1,
				PageNumber: 1,
			},
		})
		assert.Nilf(t, err, "Failed to read org consumers %v", err)
		nameList1 := _toNameList(consumerList.Items)

		consumerList, err = client.Admin.Organizations.ListModuleConsumers(ctx, org.Name, OrganizationListOptions{
			ListOptions{
				PageSize:   1,
				PageNumber: 2,
			},
		})
		assert.Nilf(t, err, "Failed to read org consumers %v", err)
		nameList2 := _toNameList(consumerList.Items)

		nameList := append(nameList1, nameList2...)

		assert.Truef(t, _listContains(org2.Name, nameList), "Expected %v to be in returned list", org2.Name)
		assert.Truef(t, _listContains(org3.Name, nameList), "Expected %v to be in returned list", org3.Name)
	})
}

func TestAdminOrganizations(t *testing.T) {

	skipIfNotEnterprise(t)

	client := testClient(t)
	ctx := context.Background()

	org, orgTestCleanup := createOrganization(t, client)
	defer orgTestCleanup()

	t.Run("deletes organization", func(t *testing.T) {
		org2, org2TestCleanup := createOrganization(t, client)
		cleanupFailed := false
		var err error = nil
		defer func() {
			if cleanupFailed {
				t.Error("Error destroying organization!", org.Name, err)
				org2TestCleanup()
			}
		}()

		err = client.Admin.Organizations.Delete(ctx, org2.Name)
		if err != nil {
			cleanupFailed = true
		}
	})

	t.Run("fetches and updates organization", func(t *testing.T) {
		adminOrg, err := client.Admin.Organizations.Read(ctx, org.Name)
		assert.NotNilf(t, adminOrg, "Org returned as nil")
		assert.Nilf(t, err, "Failed to update org %v", err)

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

		adminOrg, err = client.Admin.Organizations.Update(ctx, org.Name, opts)

		assert.NotNilf(t, adminOrg, "Org returned as nil")
		assert.Nilf(t, err, "Failed to update org %v", err)

		assert.Equal(t, adminOrg.AccessBetaTools, accessBetaTools)
		assert.Equal(t, adminOrg.GlobalModuleSharing, globalModuleSharing)
		assert.Equal(t, adminOrg.IsDisabled, isDisabled)
		assert.Equal(t, adminOrg.TerraformBuildWorkerApplyTimeout, terraformBuildWorkerApplyTimeout)
		assert.Equal(t, adminOrg.TerraformBuildWorkerPlanTimeout, terraformBuildWorkerPlanTimeout)

		isDisabled = true
		opts = AdminOrganizationUpdateOptions{
			IsDisabled: &isDisabled,
		}

		adminOrg, err = client.Admin.Organizations.Update(ctx, org.Name, opts)
		assert.NotNilf(t, adminOrg, "Org returned as nil")
		assert.Nilf(t, err, "Failed to update org %v", err)

		assert.Equal(t, adminOrg.IsDisabled, isDisabled)

		isDisabled = false
		opts = AdminOrganizationUpdateOptions{
			IsDisabled: &isDisabled,
		}

		adminOrg, err = client.Admin.Organizations.Update(ctx, org.Name, opts)
		assert.NotNilf(t, adminOrg, "Org returned as nil")
		assert.Nilf(t, err, "Failed to update org %v", err)

		assert.Equal(t, adminOrg.IsDisabled, isDisabled)
	})
}
