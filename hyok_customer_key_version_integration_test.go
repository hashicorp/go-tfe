package tfe

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

// These tests are intended for local execution only, as OIDC configurations for HYOK requires specific conditions.
// To run them locally, follow the instructions outlined in hyok_configuration_integration_test.go

func TestHYOKCustomerKeyVersionsList(t *testing.T) {
	if skipHYOKIntegrationTests {
		t.Skip()
	}

	client := testClient(t)
	ctx := context.Background()

	orgTest, err := client.Organizations.Read(ctx, hyokOrganizationName)
	if err != nil {
		t.Fatal(err)
	}

	agentPool, agentPoolCleanup := createAgentPool(t, client, orgTest)
	t.Cleanup(agentPoolCleanup)

	oidc, oidcCleanup := createGCPOIDCConfiguration(t, client, orgTest)
	t.Cleanup(oidcCleanup)
	hyok, hyokCleanup := oidc.createHYOKConfiguration(t, client, orgTest, agentPool)
	t.Cleanup(hyokCleanup)

	t.Run("with no list options", func(t *testing.T) {
		_, err := client.HYOKCustomerKeyVersions.List(ctx, hyok.ID, nil)
		require.NoError(t, err)
	})
}
