// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package tfe

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGPGKeyList(t *testing.T) {
	t.Skip()

	client := testClient(t)
	ctx := context.Background()

	org1, org1Cleanup := createOrganization(t, client)
	t.Cleanup(org1Cleanup)

	org2, org2Cleanup := createOrganization(t, client)
	t.Cleanup(org2Cleanup)

	upgradeOrganizationSubscription(t, client, org1)
	upgradeOrganizationSubscription(t, client, org2)

	provider1, provider1Cleanup := createRegistryProvider(t, client, org1, PrivateRegistry)
	t.Cleanup(provider1Cleanup)

	provider2, provider2Cleanup := createRegistryProvider(t, client, org2, PrivateRegistry)
	t.Cleanup(provider2Cleanup)

	gpgKey1, gpgKey1Cleanup := createGPGKey(t, client, org1, provider1)
	t.Cleanup(gpgKey1Cleanup)

	gpgKey2, gpgKey2Cleanup := createGPGKey(t, client, org2, provider2)
	t.Cleanup(gpgKey2Cleanup)

	t.Run("with single namespace", func(t *testing.T) {
		opts := GPGKeyListOptions{
			Namespaces: []string{org1.Name},
		}

		keyl, err := client.GPGKeys.ListPrivate(ctx, opts)
		require.NoError(t, err)

		require.Len(t, keyl.Items, 1)
		assert.Equal(t, gpgKey1.ID, keyl.Items[0].ID)
		assert.Equal(t, gpgKey1.KeyID, keyl.Items[0].KeyID)
	})

	t.Run("with multiple namespaces", func(t *testing.T) {
		t.Skip("Skipping due to GPG Key API not returning keys for multiple namespaces")

		opts := GPGKeyListOptions{
			Namespaces: []string{org1.Name, org2.Name},
		}

		keyl, err := client.GPGKeys.ListPrivate(ctx, opts)
		require.NoError(t, err)

		require.Len(t, keyl.Items, 2)
		for i, key := range []*GPGKey{
			gpgKey1,
			gpgKey2,
		} {
			assert.Equal(t, key.ID, keyl.Items[i].ID)
			assert.Equal(t, key.KeyID, keyl.Items[i].KeyID)
		}
	})

	t.Run("with list options", func(t *testing.T) {
		opts := GPGKeyListOptions{
			Namespaces: []string{org1.Name},
			ListOptions: ListOptions{
				PageNumber: 999,
				PageSize:   100,
			},
		}

		keyl, err := client.GPGKeys.ListPrivate(ctx, opts)
		require.NoError(t, err)
		require.Empty(t, keyl.Items)
		assert.Equal(t, 999, keyl.CurrentPage)
		assert.Equal(t, 1, keyl.TotalCount)
	})

	t.Run("with invalid options", func(t *testing.T) {
		t.Run("invalid namespace", func(t *testing.T) {
			opts := GPGKeyListOptions{
				Namespaces: []string{},
			}
			_, err := client.GPGKeys.ListPrivate(ctx, opts)
			require.EqualError(t, err, ErrInvalidNamespace.Error())
		})
	})
}

func TestGPGKeyCreate(t *testing.T) {
	t.Skip()

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
	t.Skip()

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
	t.Skip()

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
	t.Skip()

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
