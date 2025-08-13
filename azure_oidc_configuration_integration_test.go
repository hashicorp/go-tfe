package tfe

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAzureOIDCConfigurationsCreateReadUpdateDelete(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	t.Cleanup(orgTestCleanup)

	// Using "silly_name" because of the hyok feature flag.
	// Put in the name of the organization you want to test with.
	orgTest, err := client.Organizations.Read(ctx, "silly_name")

	create_azure_oidc_configuration, err := client.AzureOIDCConfigurations.Create(ctx, orgTest.Name, AzureOIDCConfigurationCreateOptions{
		ClientID:       "your-azure-client-id",
		SubscriptionID: "your-azure-subscription-id",
		TenantID:       "your-azure-tenant-id",
		Organization: &Organization{
			Name: orgTest.Name,
		},
	})
	require.NoError(t, err)
	require.NotNil(t, create_azure_oidc_configuration)

	read_azure_oidc_configuration, err := client.AzureOIDCConfigurations.Read(ctx, create_azure_oidc_configuration.ID)
	require.NoError(t, err)
	require.NotNil(t, read_azure_oidc_configuration)

	update_azure_oidc_configuration, err := client.AzureOIDCConfigurations.Update(ctx, create_azure_oidc_configuration.ID, AzureOIDCConfigurationUpdateOptions{
		ClientID:       "your-azure-client-id-updated",
		SubscriptionID: "your-azure-subscription-id-updated",
		TenantID:       "your-azure-tenant-id-updated",
	})
	require.NoError(t, err)
	require.NotNil(t, update_azure_oidc_configuration)
	assert.Equal(t, create_azure_oidc_configuration.ID, update_azure_oidc_configuration.ID)
	assert.Equal(t, "your-azure-client-id-updated", update_azure_oidc_configuration.ClientID)
	assert.Equal(t, "your-azure-subscription-id-updated", update_azure_oidc_configuration.SubscriptionID)
	assert.Equal(t, "your-azure-tenant-id-updated", update_azure_oidc_configuration.TenantID)

	err = client.AzureOIDCConfigurations.Delete(ctx, create_azure_oidc_configuration.ID)
	require.NoError(t, err)
}
