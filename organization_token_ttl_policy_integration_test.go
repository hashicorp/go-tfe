// Copyright IBM Corp. 2018, 2025
// SPDX-License-Identifier: MPL-2.0

package tfe

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	ttl1HourInMs         int64 = 3600000
	ttl12HoursInMs       int64 = 43200000
	ttl1DayInMs          int64 = 86400000
	ttl7DaysInMs         int64 = 604800000
	ttl30DaysInMs        int64 = 2592000000
	ttl6MonthsInMs       int64 = 15768000000
	ttl1YearInMs         int64 = 31536000000
	ttl2YearsInMs        int64 = 63072000000
	ttl2YearsDefaultInMs int64 = 63113904000 // Default 2-year TTL used by the API
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

	setupOptions := OrganizationTokenTTLPolicyUpdateOptions{
		Policies: []OrganizationTokenTTLPolicyUpdateItem{
			{TokenType: TokenTypeOrganization, MaxTTLMs: ttl2YearsDefaultInMs},
			{TokenType: TokenTypeTeam, MaxTTLMs: ttl2YearsDefaultInMs},
			{TokenType: TokenTypeUser, MaxTTLMs: ttl2YearsDefaultInMs},
			{TokenType: TokenTypeAuditTrails, MaxTTLMs: ttl2YearsDefaultInMs},
		},
	}
	_, err := client.OrganizationTokenTTLPolicies.Update(ctx, orgTest.Name, setupOptions)
	require.NoError(t, err, "Failed to create initial policies")

	t.Run("update single policy", func(t *testing.T) {
		options := OrganizationTokenTTLPolicyUpdateOptions{
			Policies: []OrganizationTokenTTLPolicyUpdateItem{
				{
					TokenType: TokenTypeOrganization,
					MaxTTLMs:  ttl1DayInMs,
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
		assert.Equal(t, ttl1DayInMs, orgPolicy.MaxTTLMs)
		assert.NotEmpty(t, orgPolicy.ID)
	})

	t.Run("update multiple policies", func(t *testing.T) {
		options := OrganizationTokenTTLPolicyUpdateOptions{
			Policies: []OrganizationTokenTTLPolicyUpdateItem{
				{
					TokenType: TokenTypeOrganization,
					MaxTTLMs:  ttl1YearInMs,
				},
				{
					TokenType: TokenTypeTeam,
					MaxTTLMs:  ttl6MonthsInMs,
				},
				{
					TokenType: TokenTypeUser,
					MaxTTLMs:  ttl30DaysInMs,
				},
			},
		}

		policies, err := client.OrganizationTokenTTLPolicies.Update(ctx, orgTest.Name, options)
		require.NoError(t, err)
		require.NotNil(t, policies)
		require.Len(t, policies, 4) // API returns all 4 token type policies

		// Build a map of returned policies by token type
		policyMap := make(map[TokenType]*OrganizationTokenTTLPolicy)
		for _, policy := range policies {
			policyMap[policy.TokenType] = policy
			assert.NotEmpty(t, policy.ID)
			assert.Greater(t, policy.MaxTTLMs, int64(0))
		}

		// Verify the three policies we updated have the correct values
		require.Contains(t, policyMap, TokenTypeOrganization)
		assert.Equal(t, ttl1YearInMs, policyMap[TokenTypeOrganization].MaxTTLMs)

		require.Contains(t, policyMap, TokenTypeTeam)
		assert.Equal(t, ttl6MonthsInMs, policyMap[TokenTypeTeam].MaxTTLMs)

		require.Contains(t, policyMap, TokenTypeUser)
		assert.Equal(t, ttl30DaysInMs, policyMap[TokenTypeUser].MaxTTLMs)

		// Verify audit trails policy has the default 2-year value (not updated)
		require.Contains(t, policyMap, TokenTypeAuditTrails)
		assert.Equal(t, ttl2YearsDefaultInMs, policyMap[TokenTypeAuditTrails].MaxTTLMs, "Audit trails should have default 2-year TTL")
	})

	t.Run("update all token types", func(t *testing.T) {
		options := OrganizationTokenTTLPolicyUpdateOptions{
			Policies: []OrganizationTokenTTLPolicyUpdateItem{
				{TokenType: TokenTypeOrganization, MaxTTLMs: ttl1YearInMs},
				{TokenType: TokenTypeTeam, MaxTTLMs: ttl6MonthsInMs},
				{TokenType: TokenTypeUser, MaxTTLMs: ttl30DaysInMs},
				{TokenType: TokenTypeAuditTrails, MaxTTLMs: ttl6MonthsInMs},
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
			{"1 hour", ttl1HourInMs},
			{"12 hours", ttl12HoursInMs},
			{"1 day", ttl1DayInMs},
			{"7 days", ttl7DaysInMs},
			{"30 days", ttl30DaysInMs},
			{"6 months", ttl6MonthsInMs},
			{"1 year", ttl1YearInMs},
			{"2 years", ttl2YearsInMs},
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
				{TokenType: TokenTypeOrganization, MaxTTLMs: ttl1YearInMs},
				{TokenType: TokenTypeTeam, MaxTTLMs: ttl6MonthsInMs},
				{TokenType: TokenTypeUser, MaxTTLMs: ttl30DaysInMs},
				{TokenType: TokenTypeAuditTrails, MaxTTLMs: ttl6MonthsInMs},
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

		policyMap := make(map[TokenType]*OrganizationTokenTTLPolicy)
		for _, policy := range policyList.Items {
			policyMap[policy.TokenType] = policy
		}

		assert.Equal(t, ttl1YearInMs, policyMap[TokenTypeOrganization].MaxTTLMs)
		assert.Equal(t, ttl6MonthsInMs, policyMap[TokenTypeTeam].MaxTTLMs)
		assert.Equal(t, ttl30DaysInMs, policyMap[TokenTypeUser].MaxTTLMs)
		assert.Equal(t, ttl6MonthsInMs, policyMap[TokenTypeAuditTrails].MaxTTLMs)

		updateSingleOptions := OrganizationTokenTTLPolicyUpdateOptions{
			Policies: []OrganizationTokenTTLPolicyUpdateItem{
				{TokenType: TokenTypeOrganization, MaxTTLMs: ttl2YearsInMs},
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
		assert.Equal(t, ttl2YearsInMs, orgPolicy.MaxTTLMs)

		finalList, err := client.OrganizationTokenTTLPolicies.List(ctx, orgTest.Name, nil)
		require.NoError(t, err)
		require.NotNil(t, finalList)

		// Verify the organization policy was updated in the final list
		var foundOrgPolicy bool
		for _, policy := range finalList.Items {
			if policy.TokenType == TokenTypeOrganization {
				assert.Equal(t, ttl2YearsInMs, policy.MaxTTLMs)
				foundOrgPolicy = true
				break
			}
		}
		require.True(t, foundOrgPolicy, "Organization token policy should be present in final list")
	})
}
