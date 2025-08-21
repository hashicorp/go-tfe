package tfe

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGCPOIDCConfigurationCreate(t *testing.T) {
	skipIfEnterprise(t)

	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	t.Cleanup(orgTestCleanup)

	t.Run("with valid options", func(t *testing.T) {
		opts := GCPOIDCConfigurationCreateOptions{
			ServiceAccountEmail:  "updated-service-account@example.iam.gserviceaccount.com",
			ProjectNumber:        "123456789012",
			WorkloadProviderName: randomString(t),
		}

		oidcConfig, err := client.GCPOIDCConfigurations.Create(ctx, orgTest.Name, opts)
		require.NoError(t, err)
		require.NotNil(t, oidcConfig)
		assert.Equal(t, oidcConfig.ServiceAccountEmail, opts.ServiceAccountEmail)
		assert.Equal(t, oidcConfig.ProjectNumber, opts.ProjectNumber)
		assert.Equal(t, oidcConfig.WorkloadProviderName, opts.WorkloadProviderName)
	})

	t.Run("missing workload provider name", func(t *testing.T) {
		opts := GCPOIDCConfigurationCreateOptions{
			ServiceAccountEmail: "updated-service-account@example.iam.gserviceaccount.com",
			ProjectNumber:       "123456789012",
		}

		_, err := client.GCPOIDCConfigurations.Create(ctx, orgTest.Name, opts)
		assert.ErrorIs(t, err, ErrRequiredWorkloadProviderName)
	})

	t.Run("missing service account email", func(t *testing.T) {
		opts := GCPOIDCConfigurationCreateOptions{
			ProjectNumber:        "123456789012",
			WorkloadProviderName: randomString(t),
		}

		_, err := client.GCPOIDCConfigurations.Create(ctx, orgTest.Name, opts)
		assert.ErrorIs(t, err, ErrRequiredServiceAccountEmail)
	})

	t.Run("missing project number", func(t *testing.T) {
		opts := GCPOIDCConfigurationCreateOptions{
			ServiceAccountEmail:  "updated-service-account@example.iam.gserviceaccount.com",
			WorkloadProviderName: randomString(t),
		}

		_, err := client.GCPOIDCConfigurations.Create(ctx, orgTest.Name, opts)
		assert.ErrorIs(t, err, ErrRequiredProjectNumber)
	})
}

func TestGCPOIDCConfigurationRead(t *testing.T) {
	skipIfEnterprise(t)

	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	t.Cleanup(orgTestCleanup)

	oidcConfig, oidcConfigCleanup := createGCPOIDCConfiguration(t, client, orgTest)
	t.Cleanup(oidcConfigCleanup)

	t.Run("fetch existing configuration", func(t *testing.T) {
		fetched, err := client.GCPOIDCConfigurations.Read(ctx, oidcConfig.ID)
		require.NoError(t, err)
		require.NotEmpty(t, fetched)
	})

	t.Run("fetching non-existing configuration", func(t *testing.T) {
		_, err := client.GCPOIDCConfigurations.Read(ctx, "gcpoidc-notreal")
		assert.ErrorIs(t, err, ErrResourceNotFound)
	})
}

func TestGCPOIDCConfigurationUpdate(t *testing.T) {
	skipIfEnterprise(t)

	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	t.Cleanup(orgTestCleanup)

	t.Run("update all fields", func(t *testing.T) {
		oidcConfig, oidcConfigCleanup := createGCPOIDCConfiguration(t, client, orgTest)
		t.Cleanup(oidcConfigCleanup)

		serviceAccountEmail := "updated-service-account@example.iam.gserviceaccount.com"
		projectNumber := "123456789012"
		workloadProviderName := randomString(t)

		opts := GCPOIDCConfigurationUpdateOptions{
			ServiceAccountEmail:  &serviceAccountEmail,
			ProjectNumber:        &projectNumber,
			WorkloadProviderName: &workloadProviderName,
		}

		updated, err := client.GCPOIDCConfigurations.Update(ctx, oidcConfig.ID, opts)
		require.NoError(t, err)
		require.NotNil(t, updated)
		assert.Equal(t, opts.ServiceAccountEmail, updated.ServiceAccountEmail)
		assert.Equal(t, opts.ProjectNumber, updated.ProjectNumber)
		assert.Equal(t, opts.WorkloadProviderName, updated.WorkloadProviderName)
	})

	t.Run("workload provider name not provided", func(t *testing.T) {
		oidcConfig, oidcConfigCleanup := createGCPOIDCConfiguration(t, client, orgTest)
		t.Cleanup(oidcConfigCleanup)

		serviceAccountEmail := "updated-service-account@example.iam.gserviceaccount.com"
		projectNumber := "123456789012"

		opts := GCPOIDCConfigurationUpdateOptions{
			ServiceAccountEmail: &serviceAccountEmail,
			ProjectNumber:       &projectNumber,
		}

		updated, err := client.GCPOIDCConfigurations.Update(ctx, oidcConfig.ID, opts)
		require.NoError(t, err)
		require.NotNil(t, updated)
		assert.Equal(t, opts.ServiceAccountEmail, updated.ServiceAccountEmail)
		assert.Equal(t, opts.ProjectNumber, updated.ProjectNumber)
		assert.Equal(t, oidcConfig.WorkloadProviderName, updated.WorkloadProviderName) // not updated
	})

	t.Run("service account email not provided", func(t *testing.T) {
		oidcConfig, oidcConfigCleanup := createGCPOIDCConfiguration(t, client, orgTest)
		t.Cleanup(oidcConfigCleanup)

		projectNumber := "123456789012"
		workloadProviderName := randomString(t)

		opts := GCPOIDCConfigurationUpdateOptions{
			ProjectNumber:        &projectNumber,
			WorkloadProviderName: &workloadProviderName,
		}

		updated, err := client.GCPOIDCConfigurations.Update(ctx, oidcConfig.ID, opts)
		require.NoError(t, err)
		require.NotNil(t, updated)
		assert.Equal(t, oidcConfig.ServiceAccountEmail, updated.ServiceAccountEmail) // not updated
		assert.Equal(t, opts.ProjectNumber, updated.ProjectNumber)
		assert.Equal(t, opts.WorkloadProviderName, updated.WorkloadProviderName)
	})

	t.Run("project number not provided", func(t *testing.T) {
		oidcConfig, oidcConfigCleanup := createGCPOIDCConfiguration(t, client, orgTest)
		t.Cleanup(oidcConfigCleanup)

		serviceAccountEmail := "updated-service-account@example.iam.gserviceaccount.com"
		workloadProviderName := randomString(t)

		opts := GCPOIDCConfigurationUpdateOptions{
			ServiceAccountEmail:  &serviceAccountEmail,
			WorkloadProviderName: &workloadProviderName,
		}

		updated, err := client.GCPOIDCConfigurations.Update(ctx, oidcConfig.ID, opts)
		require.NoError(t, err)
		require.NotNil(t, updated)
		assert.Equal(t, opts.ServiceAccountEmail, updated.ServiceAccountEmail)
		assert.Equal(t, oidcConfig.ProjectNumber, updated.ProjectNumber) // not updated
		assert.Equal(t, opts.WorkloadProviderName, updated.WorkloadProviderName)
	})
}

func TestGCPOIDCConfigurationDelete(t *testing.T) {
	skipIfEnterprise(t)

	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	t.Cleanup(orgTestCleanup)

	oidcConfig, _ := createGCPOIDCConfiguration(t, client, orgTest)

	t.Run("delete existing configuration", func(t *testing.T) {
		err := client.GCPOIDCConfigurations.Delete(ctx, oidcConfig.ID)
		require.NoError(t, err)
	})

	t.Run("fetching non-existing configuration", func(t *testing.T) {
		err := client.GCPOIDCConfigurations.Delete(ctx, "gcpoidc-notreal")
		require.ErrorIs(t, err, ErrResourceNotFound)
	})
}
