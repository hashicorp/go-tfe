package tfe

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestModulePartnershipsList(t *testing.T) {
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
