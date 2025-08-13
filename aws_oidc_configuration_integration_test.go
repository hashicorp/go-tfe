package tfe

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAWSOIDCConfigurationsCreateReadUpdateDelete(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	t.Cleanup(orgTestCleanup)

	// Using "silly_name" because of the hyok feature flag.
	// Put in the name of the organization you want to test with.
	orgTest, err := client.Organizations.Read(ctx, "silly_name")

	create_aws_oidc_configuration, err := client.AWSOIDCConfigurations.Create(ctx, orgTest.Name, AWSOIDCConfigurationCreateOptions{
		RoleARN: "arn:aws:iam::123456789012:role/rocket-hyok",
		Organization: &Organization{
			Name: orgTest.Name,
		},
	})
	require.NoError(t, err)
	require.NotNil(t, create_aws_oidc_configuration)

	read_aws_oidc_configuration, err := client.AWSOIDCConfigurations.Read(ctx, create_aws_oidc_configuration.ID)
	require.NoError(t, err)
	require.NotNil(t, read_aws_oidc_configuration)

	update_aws_oidc_configuration, err := client.AWSOIDCConfigurations.Update(ctx, create_aws_oidc_configuration.ID, AWSOIDCConfigurationUpdateOptions{
		RoleARN: "arn:aws:iam::123456789012:role/rocket-hyok-updated",
	})
	require.NoError(t, err)
	require.NotNil(t, update_aws_oidc_configuration)
	assert.Equal(t, create_aws_oidc_configuration.ID, update_aws_oidc_configuration.ID)
	assert.Equal(t, "arn:aws:iam::123456789012:role/rocket-hyok-updated", update_aws_oidc_configuration.RoleARN)

	err = client.AWSOIDCConfigurations.Delete(ctx, create_aws_oidc_configuration.ID)
	require.NoError(t, err)

	t.Run("with empty role arn", func(t *testing.T) {
		invalidConfig, err := client.AWSOIDCConfigurations.Create(ctx, orgTest.Name, AWSOIDCConfigurationCreateOptions{
			RoleARN: "",
			Organization: &Organization{
				Name: orgTest.Name,
			},
		})
		assert.Nil(t, invalidConfig)
		assert.EqualError(t, err, ErrRequiredRoleARN.Error())
	})
}
