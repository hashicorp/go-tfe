package tfe

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGCPOIDCConfigurationsCreateReadUpdateDelete(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	t.Cleanup(orgTestCleanup)

	// Using "silly_name" because of the hyok feature flag.
	// Put in the name of the organization you want to test with.
	orgTest, err := client.Organizations.Read(ctx, "silly_name")

	create_gcp_oidc_configuration, err := client.GCPOIDCConfigurations.Create(ctx, orgTest.Name, GCPOIDCConfigurationCreateOptions{
		ServiceAccountEmail:  "service-account@example.iam.gserviceaccount.com",
		ProjectNumber:        "123456789012",
		WorkloadProviderName: "rocket",
		Organization: &Organization{
			Name: orgTest.Name,
		},
	})
	require.NoError(t, err)
	require.NotNil(t, create_gcp_oidc_configuration)

	read_gcp_oidc_configuration, err := client.GCPOIDCConfigurations.Read(ctx, create_gcp_oidc_configuration.ID)
	require.NoError(t, err)
	require.NotNil(t, read_gcp_oidc_configuration)

	update_gcp_oidc_configuration, err := client.GCPOIDCConfigurations.Update(ctx, create_gcp_oidc_configuration.ID, GCPOIDCConfigurationUpdateOptions{
		ServiceAccountEmail:  "updated-service-account@example.iam.gserviceaccount.com",
		ProjectNumber:        "987654321012",
		WorkloadProviderName: "rocket-updated",
	})
	require.NoError(t, err)
	require.NotNil(t, update_gcp_oidc_configuration)
	assert.Equal(t, create_gcp_oidc_configuration.ID, update_gcp_oidc_configuration.ID)
	assert.Equal(t, "updated-service-account@example.iam.gserviceaccount.com", update_gcp_oidc_configuration.ServiceAccountEmail)
	assert.Equal(t, "987654321012", update_gcp_oidc_configuration.ProjectNumber)
	assert.Equal(t, "rocket-updated", update_gcp_oidc_configuration.WorkloadProviderName)

	err = client.GCPOIDCConfigurations.Delete(ctx, create_gcp_oidc_configuration.ID)
	require.NoError(t, err)
}
