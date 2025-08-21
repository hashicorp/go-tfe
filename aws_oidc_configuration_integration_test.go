package tfe

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAWSOIDCConfigurationCreate(t *testing.T) {
	skipIfEnterprise(t)

	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	t.Cleanup(orgTestCleanup)

	t.Run("with valid options", func(t *testing.T) {
		opts := AWSOIDCConfigurationCreateOptions{
			RoleARN: "arn:aws:iam::123456789012:role/some-role",
		}

		oidcConfig, err := client.AWSOIDCConfigurations.Create(ctx, orgTest.Name, opts)
		require.NoError(t, err)
		require.NotNil(t, oidcConfig)
		assert.Equal(t, oidcConfig.RoleARN, opts.RoleARN)
	})

	t.Run("missing role ARN", func(t *testing.T) {
		opts := AWSOIDCConfigurationCreateOptions{}

		_, err := client.AWSOIDCConfigurations.Create(ctx, orgTest.Name, opts)
		assert.ErrorIs(t, err, ErrRequiredRoleARN)
	})
}

func TestAWSOIDCConfigurationRead(t *testing.T) {
	skipIfEnterprise(t)

	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	t.Cleanup(orgTestCleanup)

	oidcConfig, oidcConfigCleanup := createAWSOIDCConfiguration(t, client, orgTest)
	t.Cleanup(oidcConfigCleanup)

	t.Run("fetch existing configuration", func(t *testing.T) {
		fetched, err := client.AWSOIDCConfigurations.Read(ctx, oidcConfig.ID)
		require.NoError(t, err)
		require.NotEmpty(t, fetched)
	})

	t.Run("fetching non-existing configuration", func(t *testing.T) {
		_, err := client.AWSOIDCConfigurations.Read(ctx, "awsoidc-notreal")
		assert.ErrorIs(t, err, ErrResourceNotFound)
	})
}

func TestAWSOIDCConfigurationsUpdate(t *testing.T) {
	skipIfEnterprise(t)

	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	t.Cleanup(orgTestCleanup)

	oidcConfig, oidcConfigCleanup := createAWSOIDCConfiguration(t, client, orgTest)
	t.Cleanup(oidcConfigCleanup)

	t.Run("with valid options", func(t *testing.T) {
		opts := AWSOIDCConfigurationUpdateOptions{
			RoleARN: "arn:aws:iam::123456789012:role/some-role-2",
		}
		updated, err := client.AWSOIDCConfigurations.Update(ctx, oidcConfig.ID, opts)
		require.NoError(t, err)
		require.NotEmpty(t, updated)
		assert.NotEqual(t, oidcConfig.RoleARN, updated.RoleARN)
	})

	t.Run("missing role ARN", func(t *testing.T) {
		opts := AWSOIDCConfigurationUpdateOptions{}
		_, err := client.AWSOIDCConfigurations.Update(ctx, oidcConfig.ID, opts)
		assert.ErrorIs(t, err, ErrRequiredRoleARN)
	})
}

func TestAWSOIDCConfigurationsDelete(t *testing.T) {
	skipIfEnterprise(t)

	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	t.Cleanup(orgTestCleanup)

	oidcConfig, _ := createAWSOIDCConfiguration(t, client, orgTest)

	t.Run("delete existing configuration", func(t *testing.T) {
		err := client.AWSOIDCConfigurations.Delete(ctx, oidcConfig.ID)
		require.NoError(t, err)
	})

	t.Run("fetching non-existing configuration", func(t *testing.T) {
		err := client.AWSOIDCConfigurations.Delete(ctx, "awsoidc-notreal")
		require.ErrorIs(t, err, ErrResourceNotFound)
	})
}
