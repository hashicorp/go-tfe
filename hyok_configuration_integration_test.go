package tfe

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// These tests are intended for local execution only, as HYOK requires specific conditions.
// To test locally:
//  1. set skipHYOKIntegrationTests to false. The default value is true.
//  2. set hyokOrganizationName to the name of an organization that can use HYOK.
//  3. set hyokAgentPoolID to an agent pool with running agents that have HYOK capabilities turned on.
const skipHYOKIntegrationTests = false
const hyokOrganizationName = "hippos-for-sale"
const hyokAgentPoolID = "apool-vUbcDykKhvoezDoP"

func TestHYOKConfigurationCreateRevokeDelete(t *testing.T) {
	if skipHYOKIntegrationTests {
		t.Skip()
	}

	client := testClient(t)
	ctx := context.Background()

	orgTest, err := client.Organizations.Read(ctx, hyokOrganizationName)
	if err != nil {
		t.Fatal(err)
	}

	agentPool, err := client.AgentPools.Read(ctx, hyokAgentPoolID)
	if err != nil {
		t.Fatal(err)
	}

	t.Run("AWS with valid options", func(t *testing.T) {
		awsOIDCConfig, configCleanup := createAWSOIDCConfiguration(t, client, orgTest)
		t.Cleanup(configCleanup)

		keyRegion := "us-east-1"
		opts := HYOKConfigurationsCreateOptions{
			Name: randomStringWithoutSpecialChar(t),
			KMSOptions: &KMSOptions{
				KeyRegion: keyRegion,
			},
			KEKID:     "arn:aws:kms:us-east-1:123456789012:key/this-is-not-a-real-key",
			AgentPool: agentPool,
			OIDCConfiguration: &OIDCConfigurationTypeChoice{
				AWSOIDCConfiguration: awsOIDCConfig,
			},
		}

		created, err := client.HYOKConfigurations.Create(ctx, orgTest.Name, opts)
		require.NoError(t, err)
		require.NotNil(t, created)
		assert.Equal(t, opts.Name, created.Name)
		assert.Equal(t, opts.KEKID, created.KEKID)
		assert.Equal(t, opts.KMSOptions.KeyRegion, created.KMSOptions.KeyRegion)
		assert.Equal(t, opts.AgentPool.ID, created.AgentPool.ID)
		assert.Equal(t, opts.OIDCConfiguration.AWSOIDCConfiguration.ID, created.OIDCConfiguration.AWSOIDCConfiguration.ID)

		// Must revoke and delete HYOK config or else agent pool and OIDC configs cannot be cleaned up
		err = client.HYOKConfigurations.Revoke(ctx, created.ID)
		if err != nil {
			require.NoError(t, err)
		}

		fetched, err := client.HYOKConfigurations.Read(ctx, created.ID, nil)
		require.NoError(t, err)
		assert.True(t, fetched.Status == HYOKConfigurationRevoked || fetched.Status == HYOKConfigurationRevoking)

		err = client.HYOKConfigurations.Delete(ctx, created.ID)
		require.NoError(t, err)
		_, err = client.HYOKConfigurations.Read(ctx, created.ID, nil)
		require.ErrorIs(t, err, ErrResourceNotFound)
	})

	t.Run("AWS with missing key region", func(t *testing.T) {
		awsOIDCConfig, configCleanup := createAWSOIDCConfiguration(t, client, orgTest)
		t.Cleanup(configCleanup)

		opts := HYOKConfigurationsCreateOptions{
			Name:       randomStringWithoutSpecialChar(t),
			KMSOptions: &KMSOptions{},
			KEKID:      "arn:aws:kms:us-east-1:123456789012:key/this-is-not-a-real-key",
			AgentPool:  agentPool,
			OIDCConfiguration: &OIDCConfigurationTypeChoice{
				AWSOIDCConfiguration: awsOIDCConfig,
			},
		}

		_, err := client.HYOKConfigurations.Create(ctx, orgTest.Name, opts)
		require.ErrorIs(t, err, ErrRequiredKMSOptionsKeyRegion)
	})

	t.Run("GCP with valid options", func(t *testing.T) {
		gcpOIDCConfig, configCleanup := createGCPOIDCConfiguration(t, client, orgTest)
		t.Cleanup(configCleanup)

		keyLocation := "global"
		keyRingID := randomStringWithoutSpecialChar(t)

		opts := HYOKConfigurationsCreateOptions{
			Name: randomStringWithoutSpecialChar(t),
			KMSOptions: &KMSOptions{
				KeyLocation: keyLocation,
				KeyRingID:   keyRingID,
			},
			KEKID:     randomStringWithoutSpecialChar(t),
			AgentPool: agentPool,
			OIDCConfiguration: &OIDCConfigurationTypeChoice{
				GCPOIDCConfiguration: gcpOIDCConfig,
			},
		}

		created, err := client.HYOKConfigurations.Create(ctx, orgTest.Name, opts)
		require.NoError(t, err)
		require.NotNil(t, created)
		assert.Equal(t, opts.Name, created.Name)
		assert.Equal(t, opts.KEKID, created.KEKID)
		assert.Equal(t, opts.KMSOptions.KeyLocation, created.KMSOptions.KeyLocation)
		assert.Equal(t, opts.KMSOptions.KeyRingID, created.KMSOptions.KeyRingID)
		assert.Equal(t, opts.AgentPool.ID, created.AgentPool.ID)
		assert.Equal(t, opts.OIDCConfiguration.GCPOIDCConfiguration.ID, created.OIDCConfiguration.GCPOIDCConfiguration.ID)

		// Must revoke and delete HYOK config or else agent pool and OIDC configs cannot be cleaned up
		err = client.HYOKConfigurations.Revoke(ctx, created.ID)
		require.NoError(t, err)

		fetched, err := client.HYOKConfigurations.Read(ctx, created.ID, nil)
		require.NoError(t, err)
		assert.True(t, fetched.Status == HYOKConfigurationRevoked || fetched.Status == HYOKConfigurationRevoking)

		err = client.HYOKConfigurations.Delete(ctx, created.ID)
		require.NoError(t, err)
		_, err = client.HYOKConfigurations.Read(ctx, created.ID, nil)
		require.ErrorIs(t, err, ErrResourceNotFound)
	})

	t.Run("GCP with missing key location", func(t *testing.T) {
		gcpOIDCConfig, configCleanup := createGCPOIDCConfiguration(t, client, orgTest)
		t.Cleanup(configCleanup)

		keyRingID := randomStringWithoutSpecialChar(t)
		opts := HYOKConfigurationsCreateOptions{
			Name: randomStringWithoutSpecialChar(t),
			KMSOptions: &KMSOptions{
				KeyRingID: keyRingID,
			},
			KEKID:     randomStringWithoutSpecialChar(t),
			AgentPool: agentPool,
			OIDCConfiguration: &OIDCConfigurationTypeChoice{
				GCPOIDCConfiguration: gcpOIDCConfig,
			},
		}

		_, err := client.HYOKConfigurations.Create(ctx, orgTest.Name, opts)
		require.ErrorIs(t, err, ErrRequiredKMSOptionsKeyLocation)
	})

	t.Run("GCP with missing key ring ID", func(t *testing.T) {
		gcpOIDCConfig, configCleanup := createGCPOIDCConfiguration(t, client, orgTest)
		t.Cleanup(configCleanup)

		keyLocation := "global"

		opts := HYOKConfigurationsCreateOptions{
			Name: randomStringWithoutSpecialChar(t),
			KMSOptions: &KMSOptions{
				KeyLocation: keyLocation,
			},
			KEKID:     randomStringWithoutSpecialChar(t),
			AgentPool: agentPool,
			OIDCConfiguration: &OIDCConfigurationTypeChoice{
				GCPOIDCConfiguration: gcpOIDCConfig,
			},
		}

		_, err := client.HYOKConfigurations.Create(ctx, orgTest.Name, opts)
		require.ErrorIs(t, err, ErrRequiredKMSOptionsKeyRingID)
	})

	t.Run("Vault with valid options", func(t *testing.T) {
		vaultOIDCConfig, configCleanup := createVaultOIDCConfiguration(t, client, orgTest)
		t.Cleanup(configCleanup)

		opts := HYOKConfigurationsCreateOptions{
			Name:      randomStringWithoutSpecialChar(t),
			KEKID:     randomStringWithoutSpecialChar(t),
			AgentPool: agentPool,
			OIDCConfiguration: &OIDCConfigurationTypeChoice{
				VaultOIDCConfiguration: vaultOIDCConfig,
			},
		}

		created, err := client.HYOKConfigurations.Create(ctx, orgTest.Name, opts)
		require.NoError(t, err)
		require.NotNil(t, created)
		assert.Equal(t, opts.Name, created.Name)
		assert.Equal(t, opts.KEKID, created.KEKID)
		assert.Equal(t, opts.AgentPool.ID, created.AgentPool.ID)
		assert.Equal(t, opts.OIDCConfiguration.VaultOIDCConfiguration.ID, created.OIDCConfiguration.VaultOIDCConfiguration.ID)

		// Must revoke and delete HYOK config or else agent pool and OIDC configs cannot be cleaned up
		err = client.HYOKConfigurations.Revoke(ctx, created.ID)
		require.NoError(t, err)

		fetched, err := client.HYOKConfigurations.Read(ctx, created.ID, nil)
		require.NoError(t, err)
		assert.True(t, fetched.Status == HYOKConfigurationRevoked || fetched.Status == HYOKConfigurationRevoking)

		err = client.HYOKConfigurations.Delete(ctx, created.ID)
		require.NoError(t, err)
		_, err = client.HYOKConfigurations.Read(ctx, created.ID, nil)
		require.ErrorIs(t, err, ErrResourceNotFound)
	})

	t.Run("Azure with valid options", func(t *testing.T) {
		azureOIDCConfig, configCleanup := createAzureOIDCConfiguration(t, client, orgTest)
		t.Cleanup(configCleanup)

		opts := HYOKConfigurationsCreateOptions{
			Name:      randomStringWithoutSpecialChar(t),
			KEKID:     "https://random.vault.azure.net/keys/some-key",
			AgentPool: agentPool,
			OIDCConfiguration: &OIDCConfigurationTypeChoice{
				AzureOIDCConfiguration: azureOIDCConfig,
			},
		}

		created, err := client.HYOKConfigurations.Create(ctx, orgTest.Name, opts)
		require.NoError(t, err)
		require.NotNil(t, created)
		assert.Equal(t, opts.Name, created.Name)
		assert.Equal(t, opts.KEKID, created.KEKID)
		assert.Equal(t, opts.AgentPool.ID, created.AgentPool.ID)
		assert.Equal(t, opts.OIDCConfiguration.AzureOIDCConfiguration.ID, created.OIDCConfiguration.AzureOIDCConfiguration.ID)

		// Must revoke and delete HYOK config or else agent pool and OIDC configs cannot be cleaned up
		err = client.HYOKConfigurations.Revoke(ctx, created.ID)
		require.NoError(t, err)

		fetched, err := client.HYOKConfigurations.Read(ctx, created.ID, nil)
		require.NoError(t, err)
		assert.True(t, fetched.Status == HYOKConfigurationRevoked || fetched.Status == HYOKConfigurationRevoking)

		err = client.HYOKConfigurations.Delete(ctx, created.ID)
		require.NoError(t, err)
		_, err = client.HYOKConfigurations.Read(ctx, created.ID, nil)
		require.ErrorIs(t, err, ErrResourceNotFound)
	})

	t.Run("with missing KEK ID", func(t *testing.T) {
		awsOIDCConfig, configCleanup := createAWSOIDCConfiguration(t, client, orgTest)
		t.Cleanup(configCleanup)

		keyRegion := "us-east-1"

		opts := HYOKConfigurationsCreateOptions{
			Name: randomStringWithoutSpecialChar(t),
			KMSOptions: &KMSOptions{
				KeyRegion: keyRegion,
			},
			AgentPool: agentPool,
			OIDCConfiguration: &OIDCConfigurationTypeChoice{
				AWSOIDCConfiguration: awsOIDCConfig,
			},
		}

		_, err := client.HYOKConfigurations.Create(ctx, orgTest.Name, opts)
		require.ErrorIs(t, err, ErrRequiredKEKID)
	})

	t.Run("with missing agent pool", func(t *testing.T) {
		awsOIDCConfig, configCleanup := createAWSOIDCConfiguration(t, client, orgTest)
		t.Cleanup(configCleanup)

		keyRegion := "us-east-1"

		opts := HYOKConfigurationsCreateOptions{
			Name: randomStringWithoutSpecialChar(t),
			KMSOptions: &KMSOptions{
				KeyRegion: keyRegion,
			},
			KEKID: randomStringWithoutSpecialChar(t),
			OIDCConfiguration: &OIDCConfigurationTypeChoice{
				AWSOIDCConfiguration: awsOIDCConfig,
			},
		}

		_, err := client.HYOKConfigurations.Create(ctx, orgTest.Name, opts)
		require.ErrorIs(t, err, ErrRequiredAgentPool)
	})

	t.Run("with missing OIDC config", func(t *testing.T) {
		keyRegion := "us-east-1"

		opts := HYOKConfigurationsCreateOptions{
			Name: randomStringWithoutSpecialChar(t),
			KMSOptions: &KMSOptions{
				KeyRegion: keyRegion,
			},
			KEKID:     randomStringWithoutSpecialChar(t),
			AgentPool: agentPool,
		}

		_, err := client.HYOKConfigurations.Create(ctx, orgTest.Name, opts)
		require.ErrorIs(t, err, ErrRequiredOIDCConfiguration)
	})
}

func TestHyokConfigurationList(t *testing.T) {
	if skipHYOKIntegrationTests {
		t.Skip()
	}

	client := testClient(t)
	ctx := context.Background()

	orgTest, err := client.Organizations.Read(ctx, hyokOrganizationName)
	if err != nil {
		t.Fatal(err)
	}

	agentPool, err := client.AgentPools.Read(ctx, hyokAgentPoolID)
	if err != nil {
		t.Fatal(err)
	}

	azureOIDC, azureOIDCCleanup := createAzureOIDCConfiguration(t, client, orgTest)
	t.Cleanup(azureOIDCCleanup)
	hyok1, hyokCleanup1 := azureOIDC.createHYOKConfiguration(t, client, orgTest, agentPool)
	t.Cleanup(hyokCleanup1)

	awsOIDC, awsOIDCCleanup := createAWSOIDCConfiguration(t, client, orgTest)
	t.Cleanup(awsOIDCCleanup)
	hyok2, hyokCleanup2 := awsOIDC.createHYOKConfiguration(t, client, orgTest, agentPool)
	t.Cleanup(hyokCleanup2)

	gcpOIDC, gcpOIDCCleanup := createAWSOIDCConfiguration(t, client, orgTest)
	t.Cleanup(gcpOIDCCleanup)
	hyok3, hyokCleanup3 := gcpOIDC.createHYOKConfiguration(t, client, orgTest, agentPool)
	t.Cleanup(hyokCleanup3)

	vaultOIDC, vaultOIDCCleanup := createAWSOIDCConfiguration(t, client, orgTest)
	t.Cleanup(vaultOIDCCleanup)
	hyok4, hyokCleanup4 := vaultOIDC.createHYOKConfiguration(t, client, orgTest, agentPool)
	t.Cleanup(hyokCleanup4)

	t.Run("without list options", func(t *testing.T) {
		results, err := client.HYOKConfigurations.List(ctx, orgTest.Name, nil)

		var resultingIDs []string
		for _, r := range results.Items {
			resultingIDs = append(resultingIDs, r.ID)
		}
		require.NoError(t, err)
		assert.Contains(t, resultingIDs, hyok1.ID)
		assert.Contains(t, resultingIDs, hyok2.ID)
		assert.Contains(t, resultingIDs, hyok3.ID)
		assert.Contains(t, resultingIDs, hyok4.ID)
	})
}

func TestHyokConfigurationRead(t *testing.T) {
	if skipHYOKIntegrationTests {
		t.Skip()
	}

	client := testClient(t)
	ctx := context.Background()

	orgTest, err := client.Organizations.Read(ctx, hyokOrganizationName)
	if err != nil {
		t.Fatal(err)
	}

	agentPool, err := client.AgentPools.Read(ctx, hyokAgentPoolID)
	if err != nil {
		t.Fatal(err)
	}

	t.Run("AWS", func(t *testing.T) {
		oidc, oidcCleanup := createAWSOIDCConfiguration(t, client, orgTest)
		t.Cleanup(oidcCleanup)
		hyok, hyokCleanup := oidc.createHYOKConfiguration(t, client, orgTest, agentPool)
		t.Cleanup(hyokCleanup)

		fetched, err := client.HYOKConfigurations.Read(ctx, hyok.ID, nil)
		require.NoError(t, err)
		require.NotNil(t, fetched)
		assert.Equal(t, hyok.Name, fetched.Name)
		assert.Equal(t, hyok.KEKID, fetched.KEKID)
		assert.Equal(t, hyok.KMSOptions.KeyRegion, fetched.KMSOptions.KeyRegion)
		assert.Equal(t, hyok.Organization.Name, fetched.Organization.Name)
		assert.Equal(t, hyok.AgentPool.ID, fetched.AgentPool.ID)
		assert.Equal(t, hyok.OIDCConfiguration.AWSOIDCConfiguration.ID, fetched.OIDCConfiguration.AWSOIDCConfiguration.ID)
	})

	t.Run("Azure", func(t *testing.T) {
		oidc, oidcCleanup := createAzureOIDCConfiguration(t, client, orgTest)
		t.Cleanup(oidcCleanup)
		hyok, hyokCleanup := oidc.createHYOKConfiguration(t, client, orgTest, agentPool)
		t.Cleanup(hyokCleanup)

		fetched, err := client.HYOKConfigurations.Read(ctx, hyok.ID, nil)
		require.NoError(t, err)
		require.NotNil(t, fetched)
		assert.Equal(t, hyok.Name, fetched.Name)
		assert.Equal(t, hyok.KEKID, fetched.KEKID)
		assert.Equal(t, hyok.KMSOptions, fetched.KMSOptions)
		assert.Equal(t, hyok.Organization.Name, fetched.Organization.Name)
		assert.Equal(t, hyok.AgentPool.ID, fetched.AgentPool.ID)
		assert.Equal(t, hyok.OIDCConfiguration.AzureOIDCConfiguration.ID, fetched.OIDCConfiguration.AzureOIDCConfiguration.ID)
	})

	t.Run("GCP", func(t *testing.T) {
		oidc, oidcCleanup := createGCPOIDCConfiguration(t, client, orgTest)
		t.Cleanup(oidcCleanup)
		hyok, hyokCleanup := oidc.createHYOKConfiguration(t, client, orgTest, agentPool)
		t.Cleanup(hyokCleanup)

		fetched, err := client.HYOKConfigurations.Read(ctx, hyok.ID, nil)
		require.NoError(t, err)
		require.NotNil(t, fetched)
		assert.Equal(t, hyok.Name, fetched.Name)
		assert.Equal(t, hyok.KEKID, fetched.KEKID)
		assert.Equal(t, hyok.KMSOptions.KeyLocation, fetched.KMSOptions.KeyLocation)
		assert.Equal(t, hyok.KMSOptions.KeyRingID, fetched.KMSOptions.KeyRingID)
		assert.Equal(t, hyok.Organization.Name, fetched.Organization.Name)
		assert.Equal(t, hyok.AgentPool.ID, fetched.AgentPool.ID)
		assert.Equal(t, hyok.OIDCConfiguration.GCPOIDCConfiguration.ID, fetched.OIDCConfiguration.GCPOIDCConfiguration.ID)
	})

	t.Run("Vault", func(t *testing.T) {
		oidc, oidcCleanup := createVaultOIDCConfiguration(t, client, orgTest)
		t.Cleanup(oidcCleanup)
		hyok, hyokCleanup := oidc.createHYOKConfiguration(t, client, orgTest, agentPool)
		t.Cleanup(hyokCleanup)

		fetched, err := client.HYOKConfigurations.Read(ctx, hyok.ID, nil)
		require.NoError(t, err)
		require.NotNil(t, fetched)
		assert.Equal(t, hyok.Name, fetched.Name)
		assert.Equal(t, hyok.KEKID, fetched.KEKID)
		assert.Equal(t, hyok.KMSOptions, fetched.KMSOptions)
		assert.Equal(t, hyok.Organization.Name, fetched.Organization.Name)
		assert.Equal(t, hyok.AgentPool.ID, fetched.AgentPool.ID)
		assert.Equal(t, hyok.OIDCConfiguration.VaultOIDCConfiguration.ID, fetched.OIDCConfiguration.VaultOIDCConfiguration.ID)
	})

	t.Run("fetching non-existing configuration", func(t *testing.T) {
		_, err := client.HYOKConfigurations.Read(ctx, "hyokc-notreal", nil)
		assert.ErrorIs(t, err, ErrResourceNotFound)
	})
}

func TestHYOKConfigurationUpdate(t *testing.T) {
	if skipHYOKIntegrationTests {
		t.Skip()
	}

	client := testClient(t)
	ctx := context.Background()

	orgTest, err := client.Organizations.Read(ctx, hyokOrganizationName)
	if err != nil {
		t.Fatal(err)
	}

	agentPool, err := client.AgentPools.Read(ctx, hyokAgentPoolID)
	if err != nil {
		t.Fatal(err)
	}

	t.Run("AWS with valid options", func(t *testing.T) {
		oidc, oidcCleanup := createAWSOIDCConfiguration(t, client, orgTest)
		t.Cleanup(oidcCleanup)
		hyok, hyokCleanup := oidc.createHYOKConfiguration(t, client, orgTest, agentPool)
		t.Cleanup(hyokCleanup)

		name := randomStringWithoutSpecialChar(t)
		kekID := "arn:aws:kms:us-east-1:123456789012:key/this-is-a-bad-key"

		opts := HYOKConfigurationsUpdateOptions{
			Name: &name,
			KMSOptions: &KMSOptions{
				KeyRegion: "us-east-2",
			},
			KEKID:     &kekID,
			AgentPool: agentPool,
		}

		updated, err := client.HYOKConfigurations.Update(ctx, hyok.ID, opts)
		require.NoError(t, err)
		assert.Equal(t, *opts.Name, updated.Name)
		assert.Equal(t, *opts.KEKID, updated.KEKID)
		assert.Equal(t, opts.KMSOptions.KeyRegion, updated.KMSOptions.KeyRegion)
	})

	t.Run("GCP with valid options", func(t *testing.T) {
		oidc, oidcCleanup := createGCPOIDCConfiguration(t, client, orgTest)
		t.Cleanup(oidcCleanup)
		hyok, hyokCleanup := oidc.createHYOKConfiguration(t, client, orgTest, agentPool)
		t.Cleanup(hyokCleanup)

		name := randomStringWithoutSpecialChar(t)
		kekId := randomStringWithoutSpecialChar(t)

		opts := HYOKConfigurationsUpdateOptions{
			Name: &name,
			KMSOptions: &KMSOptions{
				KeyLocation: "ca",
				KeyRingID:   randomStringWithoutSpecialChar(t),
			},
			KEKID:     &kekId,
			AgentPool: agentPool,
		}

		updated, err := client.HYOKConfigurations.Update(ctx, hyok.ID, opts)
		require.NoError(t, err)
		assert.Equal(t, *opts.Name, updated.Name)
		assert.Equal(t, *opts.KEKID, updated.KEKID)
		assert.Equal(t, opts.KMSOptions.KeyLocation, updated.KMSOptions.KeyLocation)
		assert.Equal(t, opts.KMSOptions.KeyRingID, updated.KMSOptions.KeyRingID)
	})

	t.Run("Vault with valid options", func(t *testing.T) {
		oidc, oidcCleanup := createVaultOIDCConfiguration(t, client, orgTest)
		t.Cleanup(oidcCleanup)
		hyok, hyokCleanup := oidc.createHYOKConfiguration(t, client, orgTest, agentPool)
		t.Cleanup(hyokCleanup)

		name := randomStringWithoutSpecialChar(t)
		kekID := randomStringWithoutSpecialChar(t)

		opts := HYOKConfigurationsUpdateOptions{
			Name:      &name,
			KEKID:     &kekID,
			AgentPool: agentPool,
		}

		updated, err := client.HYOKConfigurations.Update(ctx, hyok.ID, opts)
		require.NoError(t, err)
		assert.Equal(t, *opts.Name, updated.Name)
		assert.Equal(t, *opts.KEKID, updated.KEKID)
		assert.Equal(t, opts.AgentPool.ID, updated.AgentPool.ID)
	})

	t.Run("Azure with valid options", func(t *testing.T) {
		oidc, oidcCleanup := createAzureOIDCConfiguration(t, client, orgTest)
		t.Cleanup(oidcCleanup)
		hyok, hyokCleanup := oidc.createHYOKConfiguration(t, client, orgTest, agentPool)
		t.Cleanup(hyokCleanup)

		name := randomStringWithoutSpecialChar(t)
		kekID := "https://random.vault.azure.net/keys/some-key-2"

		opts := HYOKConfigurationsUpdateOptions{
			Name:      &name,
			KEKID:     &kekID,
			AgentPool: agentPool,
		}

		updated, err := client.HYOKConfigurations.Update(ctx, hyok.ID, opts)
		require.NoError(t, err)
		assert.Equal(t, *opts.Name, updated.Name)
		assert.Equal(t, *opts.KEKID, updated.KEKID)
		assert.Equal(t, opts.AgentPool.ID, updated.AgentPool.ID)
	})
}
