package tfe

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestModulePartnershipsList(t *testing.T) {

	skipIfNotEnterprise(t)

	client := testClient(t)
	ctx := context.Background()

	org, orgTestCleanup := createOrganization(t, client)
	defer orgTestCleanup()

	t.Run("creates and destroys consumers", func(t *testing.T) {
		consumerList, _ := client.Admin.Organizations.ListModuleConsumers(ctx, org.Name)
		assert.Empty(t, consumerList.Items)

		org2, orgTestCleanup2 := createOrganization(t, client)
		defer orgTestCleanup2()

		opts := ModulePartnershipUpdateOptions{
			ModuleConsumingOrganizationIDs: []*string{&org2.ExternalID},
		}
		consumerList, _ = client.Admin.Organizations.UpdateModuleConsumers(ctx, org.Name, opts)
		assert.Equal(t, org2.ExternalID, *consumerList.Items[0].ConsumingOrganizationID)
		assert.Equal(t, org.ExternalID, *consumerList.Items[0].ProducingOrganizationID)

		opts = ModulePartnershipUpdateOptions{
			ModuleConsumingOrganizationIDs: []*string{},
		}
		consumerList, _ = client.Admin.Organizations.UpdateModuleConsumers(ctx, org.Name, opts)
		assert.Empty(t, consumerList.Items)
	})
}

func TestAdminOrganizations(t *testing.T) {

	skipIfNotEnterprise(t)

	client := testClient(t)
	ctx := context.Background()

	org, orgTestCleanup := createOrganization(t, client)
	defer orgTestCleanup()

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
