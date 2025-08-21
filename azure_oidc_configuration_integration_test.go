package tfe

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAzureOIDCConfigurationCreate(t *testing.T) {
	skipIfEnterprise(t)

	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	t.Cleanup(orgTestCleanup)

	t.Run("with valid options", func(t *testing.T) {
		opts := AzureOIDCConfigurationCreateOptions{
			ClientID:       "your-azure-client-id",
			SubscriptionID: "your-azure-subscription-id",
			TenantID:       "your-azure-tenant-id",
		}

		oidcConfig, err := client.AzureOIDCConfigurations.Create(ctx, orgTest.Name, opts)
		require.NoError(t, err)
		require.NotNil(t, oidcConfig)
		assert.Equal(t, oidcConfig.ClientID, opts.ClientID)
		assert.Equal(t, oidcConfig.SubscriptionID, opts.SubscriptionID)
		assert.Equal(t, oidcConfig.TenantID, opts.TenantID)
	})

	t.Run("missing client ID", func(t *testing.T) {
		opts := AzureOIDCConfigurationCreateOptions{
			SubscriptionID: "your-azure-subscription-id",
			TenantID:       "your-azure-tenant-id",
		}

		_, err := client.AzureOIDCConfigurations.Create(ctx, orgTest.Name, opts)
		assert.ErrorIs(t, err, ErrRequiredClientID)
	})

	t.Run("missing subscription ID", func(t *testing.T) {
		opts := AzureOIDCConfigurationCreateOptions{
			ClientID: "your-azure-client-id",
			TenantID: "your-azure-tenant-id",
		}

		_, err := client.AzureOIDCConfigurations.Create(ctx, orgTest.Name, opts)
		assert.ErrorIs(t, err, ErrRequiredSubscriptionID)
	})

	t.Run("missing tenant ID", func(t *testing.T) {
		opts := AzureOIDCConfigurationCreateOptions{
			ClientID:       "your-azure-client-id",
			SubscriptionID: "your-azure-subscription-id",
		}

		_, err := client.AzureOIDCConfigurations.Create(ctx, orgTest.Name, opts)
		assert.ErrorIs(t, err, ErrRequiredTenantID)
	})
}

func TestAzureOIDCConfigurationRead(t *testing.T) {
	skipIfEnterprise(t)

	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	t.Cleanup(orgTestCleanup)

	oidcConfig, oidcConfigCleanup := createAzureOIDCConfiguration(t, client, orgTest)
	t.Cleanup(oidcConfigCleanup)

	t.Run("fetch existing configuration", func(t *testing.T) {
		fetched, err := client.AzureOIDCConfigurations.Read(ctx, oidcConfig.ID)
		require.NoError(t, err)
		require.NotEmpty(t, fetched)
	})

	t.Run("fetching non-existing configuration", func(t *testing.T) {
		_, err := client.AzureOIDCConfigurations.Read(ctx, "azoidc-notreal")
		assert.ErrorIs(t, err, ErrResourceNotFound)
	})
}

func TestAzureOIDCConfigurationUpdate(t *testing.T) {
	skipIfEnterprise(t)

	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	t.Cleanup(orgTestCleanup)

	oidcConfig, oidcConfigCleanup := createAzureOIDCConfiguration(t, client, orgTest)
	t.Cleanup(oidcConfigCleanup)

	t.Run("with valid options", func(t *testing.T) {
		opts := AzureOIDCConfigurationUpdateOptions{
			ClientID:       "your-azure-client-id",
			SubscriptionID: "your-azure-subscription-id",
			TenantID:       "your-azure-tenant-id",
		}
		updated, err := client.AzureOIDCConfigurations.Update(ctx, oidcConfig.ID, opts)
		require.NoError(t, err)
		require.NotEmpty(t, updated)
		assert.NotEqual(t, oidcConfig.ClientID, updated.ClientID)
		assert.NotEqual(t, oidcConfig.SubscriptionID, updated.SubscriptionID)
		assert.NotEqual(t, oidcConfig.TenantID, updated.TenantID)
	})

	t.Run("missing client ID", func(t *testing.T) {
		opts := AzureOIDCConfigurationUpdateOptions{
			SubscriptionID: "your-azure-subscription-id",
			TenantID:       "your-azure-tenant-id",
		}

		_, err := client.AzureOIDCConfigurations.Update(ctx, orgTest.Name, opts)
		assert.ErrorIs(t, err, ErrRequiredClientID)
	})

	t.Run("missing subscription ID", func(t *testing.T) {
		opts := AzureOIDCConfigurationUpdateOptions{
			ClientID: "your-azure-client-id",
			TenantID: "your-azure-tenant-id",
		}

		_, err := client.AzureOIDCConfigurations.Update(ctx, orgTest.Name, opts)
		assert.ErrorIs(t, err, ErrRequiredSubscriptionID)
	})

	t.Run("missing tenant ID", func(t *testing.T) {
		opts := AzureOIDCConfigurationUpdateOptions{
			ClientID:       "your-azure-client-id",
			SubscriptionID: "your-azure-subscription-id",
		}

		_, err := client.AzureOIDCConfigurations.Update(ctx, orgTest.Name, opts)
		assert.ErrorIs(t, err, ErrRequiredTenantID)
	})
}

func TestAzureOIDCConfigurationDelete(t *testing.T) {
	skipIfEnterprise(t)

	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	t.Cleanup(orgTestCleanup)

	oidcConfig, _ := createAzureOIDCConfiguration(t, client, orgTest)

	t.Run("delete existing configuration", func(t *testing.T) {
		err := client.AzureOIDCConfigurations.Delete(ctx, oidcConfig.ID)
		require.NoError(t, err)
	})

	t.Run("fetching non-existing configuration", func(t *testing.T) {
		err := client.AzureOIDCConfigurations.Delete(ctx, "azoidc-notreal")
		require.ErrorIs(t, err, ErrResourceNotFound)
	})
}
