package tfe

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

// These tests are intended for local execution only, as key versions for HYOK requires specific conditions
// for tests to run successfully. To test locally:
// 1. Follow the instructions outlined in hyok_configuration_integration_test.go.
// 2. Set hyokCustomerKeyVersionID to the ID of an existing HYOK customer key version

func TestHYOKCustomerKeyVersionsList(t *testing.T) {
	skipHYOKIntegrationTests(t)

	client := testClient(t)
	ctx := context.Background()

	orgTest := testHyokOrganization(t, client)

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

func TestHYOKCustomerKeyVersionsRead(t *testing.T) {
	skipHYOKIntegrationTests(t)

	client := testClient(t)
	ctx := context.Background()

	t.Run("read an existing key version", func(t *testing.T) {
		hyokCustomerKeyVersionID := os.Getenv("HYOK_CUSTOMER_KEY_VERSION_ID")
		if hyokCustomerKeyVersionID == "" {
			t.Fatal("Export a valid HYOK_CUSTOMER_KEY_VERSION_ID before running this test!")
		}

		_, err := client.HYOKCustomerKeyVersions.Read(ctx, hyokCustomerKeyVersionID)
		require.NoError(t, err)
	})
}
