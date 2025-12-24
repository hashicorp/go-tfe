// Copyright IBM Corp. 2018, 2025
// SPDX-License-Identifier: MPL-2.0

package tfe

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOrganizationAuditConfigurationRead(t *testing.T) {
	t.Parallel()
	skipUnlessBeta(t)

	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	defer orgTestCleanup()

	newSubscriptionUpdater(orgTest).WithBusinessPlan().Update(t)

	if v, err := hasAuditLogging(client, orgTest.Name); err != nil {
		t.Fatalf("Could not retrieve the entitlements for the test organization.: %s", err)
	} else if !v {
		t.Fatal("The test organization requires the audit-logging entitlement but is not entitled.")
		return
	}

	ac, err := client.OrganizationAuditConfigurations.Read(ctx, orgTest.Name)
	require.NoError(t, err)

	// By default audit trails is enabled
	assert.Equal(t, ac.AuditTrails.Enabled, true)
	assert.NotNil(t, ac.Organization)
	assert.Equal(t, orgTest.Name, ac.Organization.Name)
}

func TestOrganizationAuditConfigurationTest(t *testing.T) {
	t.Parallel()
	skipUnlessBeta(t)

	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	defer orgTestCleanup()

	newSubscriptionUpdater(orgTest).WithBusinessPlan().Update(t)

	if v, err := hasAuditLogging(client, orgTest.Name); err != nil {
		t.Fatalf("Could not retrieve the entitlements for the test organization.: %s", err)
	} else if !v {
		t.Fatal("The test organization requires the audit-logging entitlement but is not entitled.")
		return
	}

	result, err := client.OrganizationAuditConfigurations.Test(ctx, orgTest.Name)
	require.NoError(t, err)

	// Expect a Request ID is returned
	assert.NotNil(t, result.RequestID)
}

func TestOrganizationAuditConfigurationUpdate(t *testing.T) {
	t.Parallel()
	skipUnlessBeta(t)

	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	defer orgTestCleanup()

	newSubscriptionUpdater(orgTest).WithBusinessPlan().Update(t)

	if v, err := hasAuditLogging(client, orgTest.Name); err != nil {
		t.Fatalf("Could not retrieve the entitlements for the test organization.: %s", err)
	} else if !v {
		t.Fatal("The test organization requires the audit-logging entitlement but is not entitled.")
		return
	}

	ac, err := client.OrganizationAuditConfigurations.Read(ctx, orgTest.Name)
	require.NoError(t, err)

	// Unfortunately we can't really test the HCP Log Streaming because it requires either an integrated HCP organization,
	// or a valid HCP login session. Neither of which are setup for the test organization. Instead we just "update" the settings
	// with the existing ones. This doesn't prove that the endpoint behaves properly, but just tests that we can at least send
	// a payload to the expected API route.
	newCfg, err := client.OrganizationAuditConfigurations.Update(ctx, orgTest.Name, OrganizationAuditConfigurationOptions{
		AuditTrails: &OrganizationAuditConfigAuditTrails{
			Enabled: ac.AuditTrails.Enabled,
		},
		HCPAuditLogStreaming: &OrganizationAuditConfigAuditStreaming{
			Enabled: ac.HCPAuditLogStreaming.Enabled,
		},
	})

	require.NoError(t, err)

	assert.Equal(t, ac.AuditTrails.Enabled, newCfg.AuditTrails.Enabled)
	assert.Equal(t, ac.HCPAuditLogStreaming.Enabled, newCfg.HCPAuditLogStreaming.Enabled)
}
