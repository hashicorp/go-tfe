package tfe

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVaultOIDCConfigurationCreate(t *testing.T) {
	skipIfEnterprise(t)

	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	t.Cleanup(orgTestCleanup)

	t.Run("with valid options", func(t *testing.T) {
		opts := VaultOIDCConfigurationCreateOptions{
			Address:          "https://vault.example.com",
			RoleName:         "vault-role-name",
			Namespace:        "admin",
			JWTAuthPath:      "jwt",
			TLSCACertificate: randomString(t),
		}

		oidcConfig, err := client.VaultOIDCConfigurations.Create(ctx, orgTest.Name, opts)
		require.NoError(t, err)
		require.NotNil(t, oidcConfig)
		assert.Equal(t, opts.Address, oidcConfig.Address)
		assert.Equal(t, opts.RoleName, oidcConfig.RoleName)
		assert.Equal(t, opts.Namespace, oidcConfig.Namespace)
		assert.Equal(t, opts.JWTAuthPath, oidcConfig.JWTAuthPath)
	})

	t.Run("missing address", func(t *testing.T) {
		opts := VaultOIDCConfigurationCreateOptions{
			RoleName:         "vault-role-name",
			Namespace:        "admin",
			JWTAuthPath:      "jwt",
			TLSCACertificate: randomString(t),
		}

		_, err := client.VaultOIDCConfigurations.Create(ctx, orgTest.Name, opts)
		assert.ErrorIs(t, err, ErrRequiredVaultAddress)
	})

	t.Run("missing role name", func(t *testing.T) {
		opts := VaultOIDCConfigurationCreateOptions{
			Address:          "https://vault.example.com",
			Namespace:        "admin",
			JWTAuthPath:      "jwt",
			TLSCACertificate: randomString(t),
		}

		_, err := client.VaultOIDCConfigurations.Create(ctx, orgTest.Name, opts)
		assert.ErrorIs(t, err, ErrRequiredRoleName)
	})
}

func TestVaultOIDCConfigurationRead(t *testing.T) {
	skipIfEnterprise(t)

	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	t.Cleanup(orgTestCleanup)

	oidcConfig, oidcConfigCleanup := createVaultOIDCConfiguration(t, client, orgTest)
	t.Cleanup(oidcConfigCleanup)

	t.Run("fetch existing configuration", func(t *testing.T) {
		fetched, err := client.VaultOIDCConfigurations.Read(ctx, oidcConfig.ID)
		require.NoError(t, err)
		require.NotEmpty(t, fetched)
	})

	t.Run("fetching non-existing configuration", func(t *testing.T) {
		_, err := client.VaultOIDCConfigurations.Read(ctx, "voidc-notreal")
		assert.ErrorIs(t, err, ErrResourceNotFound)
	})
}

func TestVaultOIDCConfigurationUpdate(t *testing.T) {
	skipIfEnterprise(t)

	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	t.Cleanup(orgTestCleanup)

	oidcConfig, oidcConfigCleanup := createVaultOIDCConfiguration(t, client, orgTest)
	t.Cleanup(oidcConfigCleanup)

	t.Run("with valid options", func(t *testing.T) {
		opts := VaultOIDCConfigurationUpdateOptions{
			Address:          randomString(t),
			RoleName:         randomString(t),
			Namespace:        randomString(t),
			JWTAuthPath:      randomString(t),
			TLSCACertificate: randomString(t),
		}
		updated, err := client.VaultOIDCConfigurations.Update(ctx, oidcConfig.ID, opts)
		require.NoError(t, err)
		require.NotEmpty(t, updated)
		assert.Equal(t, opts.Address, updated.Address)
		assert.Equal(t, opts.RoleName, updated.RoleName)
		assert.Equal(t, opts.Namespace, updated.Namespace)
		assert.Equal(t, opts.JWTAuthPath, updated.JWTAuthPath)
	})

	t.Run("missing address", func(t *testing.T) {
		opts := VaultOIDCConfigurationUpdateOptions{
			RoleName:         randomString(t),
			Namespace:        randomString(t),
			JWTAuthPath:      randomString(t),
			TLSCACertificate: randomString(t),
		}

		_, err := client.VaultOIDCConfigurations.Update(ctx, orgTest.Name, opts)
		assert.ErrorIs(t, err, ErrRequiredVaultAddress)
	})

	t.Run("missing role name", func(t *testing.T) {
		opts := VaultOIDCConfigurationUpdateOptions{
			Address:          randomString(t),
			Namespace:        randomString(t),
			JWTAuthPath:      randomString(t),
			TLSCACertificate: randomString(t),
		}

		_, err := client.VaultOIDCConfigurations.Update(ctx, orgTest.Name, opts)
		assert.ErrorIs(t, err, ErrRequiredRoleName)
	})
}

func TestVaultOIDCConfigurationDelete(t *testing.T) {
	skipIfEnterprise(t)

	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	t.Cleanup(orgTestCleanup)

	oidcConfig, _ := createVaultOIDCConfiguration(t, client, orgTest)

	t.Run("delete existing configuration", func(t *testing.T) {
		err := client.VaultOIDCConfigurations.Delete(ctx, oidcConfig.ID)
		require.NoError(t, err)
	})

	t.Run("fetching non-existing configuration", func(t *testing.T) {
		err := client.VaultOIDCConfigurations.Delete(ctx, "voidc-notreal")
		require.ErrorIs(t, err, ErrResourceNotFound)
	})
}
