//go:build integration
// +build integration

package tfe

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGPGKeyCreate(t *testing.T) {
	checkTestNodeEnv(t)
	
	client := testClient(t)
	ctx := context.Background()

	org, orgCleanup := createOrganization(t, client)
	t.Cleanup(orgCleanup)

	upgradeOrganizationSubscription(t, client, org)

	provider, providerCleanup := createRegistryProvider(t, client, org, PrivateRegistry)
	t.Cleanup(providerCleanup)

	t.Run("with valid options", func(t *testing.T) {
		opts := GPGKeyCreateOptions{
			Namespace:  provider.Organization.Name,
			AsciiArmor: testGpgArmor,
		}

		gpgKey, err := client.GPGKeys.Create(ctx, PrivateRegistry, opts)
		require.NoError(t, err)

		assert.NotEmpty(t, gpgKey.ID)
		assert.Equal(t, gpgKey.AsciiArmor, opts.AsciiArmor)
		assert.Equal(t, gpgKey.Namespace, opts.Namespace)
		assert.NotEmpty(t, gpgKey.CreatedAt)
		assert.NotEmpty(t, gpgKey.UpdatedAt)

		// The default value for these two fields is an empty string
		assert.Empty(t, gpgKey.Source)
		assert.Empty(t, gpgKey.TrustSignature)
	})

	t.Run("with invalid registry name", func(t *testing.T) {
		opts := GPGKeyCreateOptions{
			Namespace:  provider.Organization.Name,
			AsciiArmor: testGpgArmor,
		}

		_, err := client.GPGKeys.Create(ctx, "foobar", opts)
		assert.ErrorIs(t, err, ErrInvalidRegistryName)
	})

	t.Run("with invalid options", func(t *testing.T) {
		missingNamespaceOpts := GPGKeyCreateOptions{
			Namespace:  "",
			AsciiArmor: testGpgArmor,
		}
		_, err := client.GPGKeys.Create(ctx, PrivateRegistry, missingNamespaceOpts)
		assert.ErrorIs(t, err, ErrInvalidNamespace)

		missingAsciiArmorOpts := GPGKeyCreateOptions{
			Namespace:  provider.Organization.Name,
			AsciiArmor: "",
		}
		_, err = client.GPGKeys.Create(ctx, PrivateRegistry, missingAsciiArmorOpts)
		assert.ErrorIs(t, err, ErrInvalidAsciiArmor)
	})
}

func TestGPGKeyRead(t *testing.T) {
	checkTestNodeEnv(t)

	client := testClient(t)
	ctx := context.Background()

	org, orgCleanup := createOrganization(t, client)
	t.Cleanup(orgCleanup)

	upgradeOrganizationSubscription(t, client, org)

	provider, providerCleanup := createRegistryProvider(t, client, org, PrivateRegistry)
	t.Cleanup(providerCleanup)

	gpgKey, gpgKeyCleanup := createGPGKey(t, client, org, provider)
	t.Cleanup(gpgKeyCleanup)

	t.Run("when the gpg key exists", func(t *testing.T) {
		fetched, err := client.GPGKeys.Read(ctx, GPGKeyID{
			RegistryName: PrivateRegistry,
			Namespace:    provider.Organization.Name,
			KeyID:        gpgKey.KeyID,
		})
		require.NoError(t, err)

		assert.NotEmpty(t, gpgKey.ID)
		assert.NotEmpty(t, gpgKey.KeyID)
		assert.Greater(t, len(gpgKey.AsciiArmor), 0)
		assert.Equal(t, fetched.Namespace, provider.Organization.Name)
	})

	t.Run("when the key does not exist", func(t *testing.T) {
		_, err := client.GPGKeys.Read(ctx, GPGKeyID{
			RegistryName: PrivateRegistry,
			Namespace:    provider.Organization.Name,
			KeyID:        "foobar",
		})
		assert.ErrorIs(t, err, ErrResourceNotFound)
	})
}

func TestGPGKeyUpdate(t *testing.T) {
	checkTestNodeEnv(t)

	client := testClient(t)
	ctx := context.Background()

	org, orgCleanup := createOrganization(t, client)
	t.Cleanup(orgCleanup)

	upgradeOrganizationSubscription(t, client, org)

	provider, providerCleanup := createRegistryProvider(t, client, org, PrivateRegistry)
	t.Cleanup(providerCleanup)

	// We won't use the cleanup method here as the namespace
	// is used to identify a key and that will change due to the update
	// call. We'll need to manually delete the key.
	gpgKey, _ := createGPGKey(t, client, org, provider)

	t.Run("when using an invalid namespace", func(t *testing.T) {
		keyID := GPGKeyID{
			RegistryName: PrivateRegistry,
			Namespace:    provider.Organization.Name,
			KeyID:        gpgKey.KeyID,
		}
		opts := GPGKeyUpdateOptions{
			Namespace: "invalid_namespace_org",
		}
		_, err := client.GPGKeys.Update(ctx, keyID, opts)
		assert.ErrorIs(t, err, ErrNamespaceNotAuthorized)
	})

	t.Run("when updating to a valid namespace", func(t *testing.T) {
		// Create a new namespace to update the key with
		org2, org2Cleanup := createOrganization(t, client)
		t.Cleanup(org2Cleanup)

		provider2, provider2Cleanup := createRegistryProvider(t, client, org2, PrivateRegistry)
		t.Cleanup(provider2Cleanup)

		keyID := GPGKeyID{
			RegistryName: PrivateRegistry,
			Namespace:    provider.Organization.Name,
			KeyID:        gpgKey.KeyID,
		}
		opts := GPGKeyUpdateOptions{
			Namespace: provider2.Organization.Name,
		}

		updatedKey, err := client.GPGKeys.Update(ctx, keyID, opts)
		require.NoError(t, err)

		assert.Equal(t, gpgKey.KeyID, updatedKey.KeyID)
		assert.Equal(t, updatedKey.Namespace, provider2.Organization.Name)

		// Cleanup
		err = client.GPGKeys.Delete(ctx, GPGKeyID{
			RegistryName: PrivateRegistry,
			Namespace:    provider2.Organization.Name,
			KeyID:        updatedKey.KeyID,
		})
		require.NoError(t, err)
	})
}

func TestGPGKeyDelete(t *testing.T) {
	checkTestNodeEnv(t)

	client := testClient(t)
	ctx := context.Background()

	org, orgCleanup := createOrganization(t, client)
	t.Cleanup(orgCleanup)

	upgradeOrganizationSubscription(t, client, org)

	provider, providerCleanup := createRegistryProvider(t, client, org, PrivateRegistry)
	t.Cleanup(providerCleanup)

	gpgKey, _ := createGPGKey(t, client, org, provider)

	t.Run("when a key exists", func(t *testing.T) {
		err := client.GPGKeys.Delete(ctx, GPGKeyID{
			RegistryName: PrivateRegistry,
			Namespace:    provider.Organization.Name,
			KeyID:        gpgKey.KeyID,
		})
		require.NoError(t, err)
	})
}
