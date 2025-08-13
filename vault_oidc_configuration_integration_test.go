package tfe

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVaultOIDCConfigurationsCreateReadUpdateDelete(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	t.Cleanup(orgTestCleanup)

	// Using "silly_name" because of the hyok feature flag.
	// Put in the name of the organization you want to test with.
	orgTest, err := client.Organizations.Read(ctx, "silly_name")

	create_vault_oidc_configuration, err := client.VaultOIDCConfigurations.Create(ctx, orgTest.Name, VaultOIDCConfigurationCreateOptions{
		Address:          "https://vault.example.com",
		RoleName:         "vault-role-name",
		Namespace:        "admin",
		JWTAuthPath:      "jwt",
		TLSCACertificate: "something",
		Organization: &Organization{
			Name: orgTest.Name,
		},
	})
	require.NoError(t, err)
	require.NotNil(t, create_vault_oidc_configuration)

	read_vault_oidc_configuration, err := client.VaultOIDCConfigurations.Read(ctx, create_vault_oidc_configuration.ID)
	require.NoError(t, err)
	require.NotNil(t, read_vault_oidc_configuration)

	update_vault_oidc_configuration, err := client.VaultOIDCConfigurations.Update(ctx, create_vault_oidc_configuration.ID, VaultOIDCConfigurationUpdateOptions{
		Address:          "https://vault.example.updated.com",
		RoleName:         "vault-role-name-updated",
		Namespace:        "admin-updated",
		JWTAuthPath:      "jwt-updated",
		TLSCACertificate: "something-updated",
	})
	require.NoError(t, err)
	require.NotNil(t, update_vault_oidc_configuration)
	assert.Equal(t, create_vault_oidc_configuration.ID, update_vault_oidc_configuration.ID)
	assert.Equal(t, "https://vault.example.updated.com", update_vault_oidc_configuration.Address)
	assert.Equal(t, "vault-role-name-updated", update_vault_oidc_configuration.RoleName)
	assert.Equal(t, "admin-updated", update_vault_oidc_configuration.Namespace)
	assert.Equal(t, "jwt-updated", update_vault_oidc_configuration.JWTAuthPath)
	assert.Equal(t, "something-updated", update_vault_oidc_configuration.TLSCACertificate)

	err = client.VaultOIDCConfigurations.Delete(ctx, create_vault_oidc_configuration.ID)
	require.NoError(t, err)
}
