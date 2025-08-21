package tfe

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// These tests are intended for local execution only, as OIDC configurations for HYOK requires specific conditions.
// To run them locally, follow the instructions outlined in hyok_configuration_integration_test.go

func TestAWSOIDCConfigurationCreate(t *testing.T) {
	if skipHYOKIntegrationTests {
		t.Skip()
	}

	client := testClient(t)
	ctx := context.Background()

	orgTest, err := client.Organizations.Read(ctx, hyokOrganizationName)
	if err != nil {
		t.Fatal(err)
	}

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
	if skipHYOKIntegrationTests {
		t.Skip()
	}

	client := testClient(t)
	ctx := context.Background()

	orgTest, err := client.Organizations.Read(ctx, hyokOrganizationName)
	if err != nil {
		t.Fatal(err)
	}

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
	if skipHYOKIntegrationTests {
		t.Skip()
	}

	client := testClient(t)
	ctx := context.Background()

	orgTest, err := client.Organizations.Read(ctx, hyokOrganizationName)
	if err != nil {
		t.Fatal(err)
	}

	oidcConfig, oidcConfigCleanup := createAWSOIDCConfiguration(t, client, orgTest)
	t.Cleanup(oidcConfigCleanup)

	t.Run("with valid options", func(t *testing.T) {
		opts := AWSOIDCConfigurationUpdateOptions{
			RoleARN: "arn:aws:iam::123456789012:role/some-role-2",
		}
		updated, err := client.AWSOIDCConfigurations.Update(ctx, oidcConfig.ID, opts)
		require.NoError(t, err)
		require.NotEmpty(t, updated)
		assert.Equal(t, opts.RoleARN, updated.RoleARN)
	})

	t.Run("missing role ARN", func(t *testing.T) {
		opts := AWSOIDCConfigurationUpdateOptions{}
		_, err := client.AWSOIDCConfigurations.Update(ctx, oidcConfig.ID, opts)
		assert.ErrorIs(t, err, ErrRequiredRoleARN)
	})
}

func TestAWSOIDCConfigurationsDelete(t *testing.T) {
	if skipHYOKIntegrationTests {
		t.Skip()
	}

	client := testClient(t)
	ctx := context.Background()

	orgTest, err := client.Organizations.Read(ctx, hyokOrganizationName)
	if err != nil {
		t.Fatal(err)
	}
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
