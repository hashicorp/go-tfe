package tfe

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// These tests are intended for local execution only, as OIDC configurations for HYOK requires specific conditions.
// To run them locally, follow the instructions outlined in hyok_configuration_integration_test.go

func TestVaultOIDCConfigurationCreateDelete(t *testing.T) {
	skipHYOKIntegrationTests := os.Getenv("SKIP_HYOK_INTEGRATION_TESTS") != "false"
	if skipHYOKIntegrationTests {
		t.Skip()
	}

	client := testClient(t)
	ctx := context.Background()

	// replace the environment variable with a valid organization name that has Vault OIDC HYOK configurations
	hyokOrganizationName := os.Getenv("HYOK_ORGANIZATION_NAME")
	if hyokOrganizationName == "" {
		t.Fatal("Export a valid HYOK_ORGANIZATION_NAME before running this test!")
	}

	orgTest, err := client.Organizations.Read(ctx, hyokOrganizationName)
	if err != nil {
		t.Fatal(err)
	}

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

		// delete the created configuration
		err = client.VaultOIDCConfigurations.Delete(ctx, oidcConfig.ID)
		require.NoError(t, err)
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
	skipHYOKIntegrationTests := os.Getenv("SKIP_HYOK_INTEGRATION_TESTS") != "false"
	if skipHYOKIntegrationTests {
		t.Skip()
	}

	client := testClient(t)
	ctx := context.Background()

	// replace the environment variable with a valid organization name that has Vault OIDC HYOK configurations
	hyokOrganizationName := os.Getenv("HYOK_ORGANIZATION_NAME")
	if hyokOrganizationName == "" {
		t.Fatal("Export a valid HYOK_ORGANIZATION_NAME before running this test!")
	}

	orgTest, err := client.Organizations.Read(ctx, hyokOrganizationName)
	if err != nil {
		t.Fatal(err)
	}

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
	skipHYOKIntegrationTests := os.Getenv("SKIP_HYOK_INTEGRATION_TESTS") != "false"
	if skipHYOKIntegrationTests {
		t.Skip()
	}

	client := testClient(t)
	ctx := context.Background()

	// replace the environment variable with a valid organization name that has Vault OIDC HYOK configurations
	hyokOrganizationName := os.Getenv("HYOK_ORGANIZATION_NAME")
	if hyokOrganizationName == "" {
		t.Fatal("Export a valid HYOK_ORGANIZATION_NAME before running this test!")
	}

	orgTest, err := client.Organizations.Read(ctx, hyokOrganizationName)
	if err != nil {
		t.Fatal(err)
	}

	t.Run("update all fields", func(t *testing.T) {
		oidcConfig, oidcConfigCleanup := createVaultOIDCConfiguration(t, client, orgTest)
		t.Cleanup(oidcConfigCleanup)

		address := randomString(t)
		roleName := randomString(t)
		namespace := randomString(t)
		jwtAuthPath := randomString(t)
		tlscaCertificate := randomString(t)

		opts := VaultOIDCConfigurationUpdateOptions{
			Address:          &address,
			RoleName:         &roleName,
			Namespace:        &namespace,
			JWTAuthPath:      &jwtAuthPath,
			TLSCACertificate: &tlscaCertificate,
		}
		updated, err := client.VaultOIDCConfigurations.Update(ctx, oidcConfig.ID, opts)
		require.NoError(t, err)
		require.NotEmpty(t, updated)
		assert.Equal(t, *opts.Address, updated.Address)
		assert.Equal(t, *opts.RoleName, updated.RoleName)
		assert.Equal(t, *opts.Namespace, updated.Namespace)
		assert.Equal(t, *opts.JWTAuthPath, updated.JWTAuthPath)
		assert.Equal(t, *opts.TLSCACertificate, updated.TLSCACertificate)
	})

	t.Run("address not provided", func(t *testing.T) {
		oidcConfig, oidcConfigCleanup := createVaultOIDCConfiguration(t, client, orgTest)
		t.Cleanup(oidcConfigCleanup)

		roleName := randomString(t)
		namespace := randomString(t)
		jwtAuthPath := randomString(t)
		tlscaCertificate := randomString(t)

		opts := VaultOIDCConfigurationUpdateOptions{
			RoleName:         &roleName,
			Namespace:        &namespace,
			JWTAuthPath:      &jwtAuthPath,
			TLSCACertificate: &tlscaCertificate,
		}
		updated, err := client.VaultOIDCConfigurations.Update(ctx, oidcConfig.ID, opts)
		require.NoError(t, err)
		require.NotEmpty(t, updated)
		assert.Equal(t, oidcConfig.Address, updated.Address) // not updated
		assert.Equal(t, *opts.RoleName, updated.RoleName)
		assert.Equal(t, *opts.Namespace, updated.Namespace)
		assert.Equal(t, *opts.JWTAuthPath, updated.JWTAuthPath)
		assert.Equal(t, *opts.TLSCACertificate, updated.TLSCACertificate)
	})

	t.Run("role name not provided", func(t *testing.T) {
		oidcConfig, oidcConfigCleanup := createVaultOIDCConfiguration(t, client, orgTest)
		t.Cleanup(oidcConfigCleanup)

		address := randomString(t)
		namespace := randomString(t)
		jwtAuthPath := randomString(t)
		tlscaCertificate := randomString(t)

		opts := VaultOIDCConfigurationUpdateOptions{
			Address:          &address,
			Namespace:        &namespace,
			JWTAuthPath:      &jwtAuthPath,
			TLSCACertificate: &tlscaCertificate,
		}
		updated, err := client.VaultOIDCConfigurations.Update(ctx, oidcConfig.ID, opts)
		require.NoError(t, err)
		require.NotEmpty(t, updated)
		assert.Equal(t, *opts.Address, updated.Address)
		assert.Equal(t, oidcConfig.RoleName, updated.RoleName) // not updated
		assert.Equal(t, *opts.Namespace, updated.Namespace)
		assert.Equal(t, *opts.JWTAuthPath, updated.JWTAuthPath)
		assert.Equal(t, *opts.TLSCACertificate, updated.TLSCACertificate)
	})

	t.Run("namespace not provided", func(t *testing.T) {
		oidcConfig, oidcConfigCleanup := createVaultOIDCConfiguration(t, client, orgTest)
		t.Cleanup(oidcConfigCleanup)

		address := randomString(t)
		roleName := randomString(t)
		jwtAuthPath := randomString(t)
		tlscaCertificate := randomString(t)

		opts := VaultOIDCConfigurationUpdateOptions{
			Address:          &address,
			RoleName:         &roleName,
			JWTAuthPath:      &jwtAuthPath,
			TLSCACertificate: &tlscaCertificate,
		}
		updated, err := client.VaultOIDCConfigurations.Update(ctx, oidcConfig.ID, opts)
		require.NoError(t, err)
		require.NotEmpty(t, updated)
		assert.Equal(t, *opts.Address, updated.Address)
		assert.Equal(t, *opts.RoleName, updated.RoleName)
		assert.Equal(t, oidcConfig.Namespace, updated.Namespace) // not updated
		assert.Equal(t, *opts.JWTAuthPath, updated.JWTAuthPath)
		assert.Equal(t, *opts.TLSCACertificate, updated.TLSCACertificate)
	})

	t.Run("JWTAuthPath not provided", func(t *testing.T) {
		oidcConfig, oidcConfigCleanup := createVaultOIDCConfiguration(t, client, orgTest)
		t.Cleanup(oidcConfigCleanup)

		address := randomString(t)
		roleName := randomString(t)
		namespace := randomString(t)
		tlscaCertificate := randomString(t)

		opts := VaultOIDCConfigurationUpdateOptions{
			Address:          &address,
			RoleName:         &roleName,
			Namespace:        &namespace,
			TLSCACertificate: &tlscaCertificate,
		}
		updated, err := client.VaultOIDCConfigurations.Update(ctx, oidcConfig.ID, opts)
		require.NoError(t, err)
		require.NotEmpty(t, updated)
		assert.Equal(t, *opts.Address, updated.Address)
		assert.Equal(t, *opts.RoleName, updated.RoleName)
		assert.Equal(t, *opts.Namespace, updated.Namespace)
		assert.Equal(t, oidcConfig.JWTAuthPath, updated.JWTAuthPath) // not updated
		assert.Equal(t, *opts.TLSCACertificate, updated.TLSCACertificate)
	})

	t.Run("TLSCACertificate not provided", func(t *testing.T) {
		oidcConfig, oidcConfigCleanup := createVaultOIDCConfiguration(t, client, orgTest)
		t.Cleanup(oidcConfigCleanup)

		address := randomString(t)
		roleName := randomString(t)
		namespace := randomString(t)
		jwtAuthPath := randomString(t)

		opts := VaultOIDCConfigurationUpdateOptions{
			Address:     &address,
			RoleName:    &roleName,
			Namespace:   &namespace,
			JWTAuthPath: &jwtAuthPath,
		}
		updated, err := client.VaultOIDCConfigurations.Update(ctx, oidcConfig.ID, opts)
		require.NoError(t, err)
		require.NotEmpty(t, updated)
		assert.Equal(t, *opts.Address, updated.Address)
		assert.Equal(t, *opts.RoleName, updated.RoleName)
		assert.Equal(t, *opts.Namespace, updated.Namespace)
		assert.Equal(t, *opts.JWTAuthPath, updated.JWTAuthPath)
		assert.Equal(t, oidcConfig.TLSCACertificate, updated.TLSCACertificate) // not updated
	})
}
