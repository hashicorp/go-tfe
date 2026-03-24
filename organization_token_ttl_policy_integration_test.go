// Copyright IBM Corp. 2018, 2025
// SPDX-License-Identifier: MPL-2.0

package tfe

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOrganizationTokenTTLPoliciesList(t *testing.T) {
	skipUnlessBeta(t)
	t.Parallel()
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	defer orgTestCleanup()

	t.Run("with valid organization", func(t *testing.T) {
		policyList, err := client.OrganizationTokenTTLPolicies.List(ctx, orgTest.Name, nil)
		require.NoError(t, err)
		require.NotNil(t, policyList)
		assert.NotNil(t, policyList.Items)
	})

	t.Run("without valid organization", func(t *testing.T) {
		policyList, err := client.OrganizationTokenTTLPolicies.List(ctx, badIdentifier, nil)
		assert.Nil(t, policyList)
		assert.EqualError(t, err, ErrInvalidOrg.Error())
	})
}

func TestOrganizationTokenTTLPoliciesUpdate(t *testing.T) {
	skipUnlessBeta(t)
	t.Parallel()
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	defer orgTestCleanup()

	t.Run("update single policy", func(t *testing.T) {
		options := OrganizationTokenTTLPolicyUpdateOptions{
			Policies: []OrganizationTokenTTLPolicyUpdateItem{
				{
					TokenType: TokenTypeOrganization,
					MaxTTLMs:  86400000, // 1 day in milliseconds
				},
			},
		}

		policies, err := client.OrganizationTokenTTLPolicies.Update(ctx, orgTest.Name, options)
		require.NoError(t, err)
		require.NotNil(t, policies)
		require.Len(t, policies, 4) // API returns all 4 token type policies

		// Find the organization token policy and verify its value
		var orgPolicy *OrganizationTokenTTLPolicy
		for _, policy := range policies {
			if policy.TokenType == TokenTypeOrganization {
				orgPolicy = policy
				break
			}
		}
		require.NotNil(t, orgPolicy, "Organization token policy should be present")
		assert.Equal(t, TokenTypeOrganization, orgPolicy.TokenType)
		assert.Equal(t, int64(86400000), orgPolicy.MaxTTLMs)
		assert.NotEmpty(t, orgPolicy.ID)
	})

	t.Run("update multiple policies", func(t *testing.T) {
		options := OrganizationTokenTTLPolicyUpdateOptions{
			Policies: []OrganizationTokenTTLPolicyUpdateItem{
				{
					TokenType: TokenTypeOrganization,
					MaxTTLMs:  31536000000, // 1 year
				},
				{
					TokenType: TokenTypeTeam,
					MaxTTLMs:  15768000000, // 6 months
				},
				{
					TokenType: TokenTypeUser,
					MaxTTLMs:  2592000000, // 30 days
				},
			},
		}

		policies, err := client.OrganizationTokenTTLPolicies.Update(ctx, orgTest.Name, options)
		require.NoError(t, err)
		require.NotNil(t, policies)
		require.Len(t, policies, 4) // API returns all 4 token type policies

		// Build a map of returned policies by token type
		policyMap := make(map[string]*OrganizationTokenTTLPolicy)
		for _, policy := range policies {
			policyMap[policy.TokenType] = policy
			assert.NotEmpty(t, policy.ID)
			assert.Greater(t, policy.MaxTTLMs, int64(0))
		}

		// Verify the three policies we updated have the correct values
		require.Contains(t, policyMap, TokenTypeOrganization)
		assert.Equal(t, int64(31536000000), policyMap[TokenTypeOrganization].MaxTTLMs)

		require.Contains(t, policyMap, TokenTypeTeam)
		assert.Equal(t, int64(15768000000), policyMap[TokenTypeTeam].MaxTTLMs)

		require.Contains(t, policyMap, TokenTypeUser)
		assert.Equal(t, int64(2592000000), policyMap[TokenTypeUser].MaxTTLMs)

		// Verify audit trails policy has the default 2-year value (not updated)
		require.Contains(t, policyMap, TokenTypeAuditTrails)
		assert.Equal(t, int64(63113904000), policyMap[TokenTypeAuditTrails].MaxTTLMs, "Audit trails should have default 2-year TTL")
	})

	t.Run("update all token types", func(t *testing.T) {
		options := OrganizationTokenTTLPolicyUpdateOptions{
			Policies: []OrganizationTokenTTLPolicyUpdateItem{
				{TokenType: TokenTypeOrganization, MaxTTLMs: 31536000000},
				{TokenType: TokenTypeTeam, MaxTTLMs: 15768000000},
				{TokenType: TokenTypeUser, MaxTTLMs: 2592000000},
				{TokenType: TokenTypeAuditTrails, MaxTTLMs: 7776000000},
			},
		}

		policies, err := client.OrganizationTokenTTLPolicies.Update(ctx, orgTest.Name, options)
		require.NoError(t, err)
		require.NotNil(t, policies)
		require.Len(t, policies, 4)
	})

	t.Run("with various TTL values in milliseconds", func(t *testing.T) {
		testCases := []struct {
			name  string
			value int64
		}{
			{"1 hour", 3600000},
			{"12 hours", 43200000},
			{"1 day", 86400000},
			{"7 days", 604800000},
			{"30 days", 2592000000},
			{"6 months", 15768000000},
			{"1 year", 31536000000},
			{"2 years", 63072000000},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				options := OrganizationTokenTTLPolicyUpdateOptions{
					Policies: []OrganizationTokenTTLPolicyUpdateItem{
						{
							TokenType: TokenTypeOrganization,
							MaxTTLMs:  tc.value,
						},
					},
				}

				policies, err := client.OrganizationTokenTTLPolicies.Update(ctx, orgTest.Name, options)
				require.NoError(t, err)
				require.NotNil(t, policies)
				require.Len(t, policies, 4) // API returns all 4 token type policies

				// Find the organization token policy and verify its value
				var orgPolicy *OrganizationTokenTTLPolicy
				for _, policy := range policies {
					if policy.TokenType == TokenTypeOrganization {
						orgPolicy = policy
						break
					}
				}
				require.NotNil(t, orgPolicy, "Organization token policy should be present")
				assert.Equal(t, tc.value, orgPolicy.MaxTTLMs)
			})
		}
	})

	t.Run("without policies array", func(t *testing.T) {
		options := OrganizationTokenTTLPolicyUpdateOptions{
			Policies: []OrganizationTokenTTLPolicyUpdateItem{},
		}

		policies, err := client.OrganizationTokenTTLPolicies.Update(ctx, orgTest.Name, options)
		assert.Nil(t, policies)
		assert.EqualError(t, err, ErrRequiredPolicies.Error())
	})

	t.Run("without valid organization", func(t *testing.T) {
		options := OrganizationTokenTTLPolicyUpdateOptions{
			Policies: []OrganizationTokenTTLPolicyUpdateItem{
				{
					TokenType: TokenTypeOrganization,
					MaxTTLMs:  86400000,
				},
			},
		}

		policies, err := client.OrganizationTokenTTLPolicies.Update(ctx, badIdentifier, options)
		assert.Nil(t, policies)
		assert.EqualError(t, err, ErrInvalidOrg.Error())
	})
}

func TestOrganizationTokenTTLPoliciesUpdate_RoundTrip(t *testing.T) {
	skipUnlessBeta(t)
	t.Parallel()
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	defer orgTestCleanup()

	t.Run("create, list, and verify", func(t *testing.T) {
		updateOptions := OrganizationTokenTTLPolicyUpdateOptions{
			Policies: []OrganizationTokenTTLPolicyUpdateItem{
				{TokenType: TokenTypeOrganization, MaxTTLMs: 31536000000},
				{TokenType: TokenTypeTeam, MaxTTLMs: 15768000000},
				{TokenType: TokenTypeUser, MaxTTLMs: 2592000000},
				{TokenType: TokenTypeAuditTrails, MaxTTLMs: 7776000000},
			},
		}

		updatedPolicies, err := client.OrganizationTokenTTLPolicies.Update(ctx, orgTest.Name, updateOptions)
		require.NoError(t, err)
		require.NotNil(t, updatedPolicies)
		require.Len(t, updatedPolicies, 4)

		policyList, err := client.OrganizationTokenTTLPolicies.List(ctx, orgTest.Name, nil)
		require.NoError(t, err)
		require.NotNil(t, policyList)
		require.Len(t, policyList.Items, 4)

		policyMap := make(map[string]*OrganizationTokenTTLPolicy)
		for _, policy := range policyList.Items {
			policyMap[policy.TokenType] = policy
		}

		assert.Equal(t, int64(31536000000), policyMap[TokenTypeOrganization].MaxTTLMs)
		assert.Equal(t, int64(15768000000), policyMap[TokenTypeTeam].MaxTTLMs)
		assert.Equal(t, int64(2592000000), policyMap[TokenTypeUser].MaxTTLMs)
		assert.Equal(t, int64(7776000000), policyMap[TokenTypeAuditTrails].MaxTTLMs)

		updateSingleOptions := OrganizationTokenTTLPolicyUpdateOptions{
			Policies: []OrganizationTokenTTLPolicyUpdateItem{
				{TokenType: TokenTypeOrganization, MaxTTLMs: 63072000000}, // 2 years
			},
		}

		updatedSingle, err := client.OrganizationTokenTTLPolicies.Update(ctx, orgTest.Name, updateSingleOptions)
		require.NoError(t, err)
		require.NotNil(t, updatedSingle)
		require.Len(t, updatedSingle, 4) // API returns all 4 token type policies

		// Find the organization token policy and verify its value
		var orgPolicy *OrganizationTokenTTLPolicy
		for _, policy := range updatedSingle {
			if policy.TokenType == TokenTypeOrganization {
				orgPolicy = policy
				break
			}
		}
		require.NotNil(t, orgPolicy, "Organization token policy should be present")
		assert.Equal(t, int64(63072000000), orgPolicy.MaxTTLMs)

		finalList, err := client.OrganizationTokenTTLPolicies.List(ctx, orgTest.Name, nil)
		require.NoError(t, err)
		require.NotNil(t, finalList)

		for _, policy := range finalList.Items {
			if policy.TokenType == TokenTypeOrganization {
				assert.Equal(t, int64(63072000000), policy.MaxTTLMs)
				break
			}
		}
	})
}
