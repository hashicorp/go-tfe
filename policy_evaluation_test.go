// Copyright IBM Corp. 2018, 2025
// SPDX-License-Identifier: MPL-2.0

package tfe

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPolicySetOutcomes_OutcomeOutputDeserialization(t *testing.T) {
	t.Parallel()

	// This test verifies that the Output field on Outcome is correctly
	// deserialized from the API response. The Output field carries Sentinel
	// print() log entries and is only present when policies emit print output.
	const policyEvaluationID = "poleval-abc123"
	const policySetOutcomeID = "pso-xyz789"

	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		expectedPath := fmt.Sprintf("/api/v2/policy-evaluations/%s/policy-set-outcomes", policyEvaluationID)
		if r.URL.Path != expectedPath {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/vnd.api+json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{
			"data": [
				{
					"id": "pso-xyz789",
					"type": "policy-set-outcomes",
					"attributes": {
						"policy-set-name": "test-sentinel-set",
						"policy-set-description": "",
						"error": "",
						"overridable": false,
						"result_count": {
							"passed": 1,
							"advisory_failed": 0,
							"mandatory_failed": 0,
							"errored": 0
						},
						"outcomes": [
							{
								"policy_name": "my-sentinel-policy",
								"enforcement_level": "advisory",
								"status": "passed",
								"query": "data.example.rule",
								"description": "checks example rule",
								"output": [
									{"print": "checking resource count"},
									{"print": "all resources pass"}
								]
							}
						]
					}
				}
			]
		}`)
	}))
	defer testServer.Close()

	client, err := NewClient(&Config{
		Address: testServer.URL,
		Token:   "fake-token",
	})
	require.NoError(t, err)

	ctx := context.Background()

	outcomes, err := client.PolicySetOutcomes.List(ctx, policyEvaluationID, nil)
	require.NoError(t, err)
	require.Len(t, outcomes.Items, 1)

	pso := outcomes.Items[0]
	assert.Equal(t, policySetOutcomeID, pso.ID)
	assert.Equal(t, "test-sentinel-set", pso.PolicySetName)
	require.Len(t, pso.Outcomes, 1)

	outcome := pso.Outcomes[0]
	assert.Equal(t, "my-sentinel-policy", outcome.PolicyName)
	assert.Equal(t, "passed", outcome.Status)

	require.Len(t, outcome.Output, 2, "expected two print output entries")
	assert.Equal(t, "checking resource count", outcome.Output[0].Print)
	assert.Equal(t, "all resources pass", outcome.Output[1].Print)
}

func TestPolicySetOutcomes_OutcomeOutputEmpty(t *testing.T) {
	t.Parallel()

	// Verifies that an Outcome with no output field deserializes without error
	// and leaves Output as nil (omitempty). This is the common case for OPA policies.
	const policyEvaluationID = "poleval-def456"

	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/vnd.api+json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{
			"data": [
				{
					"id": "pso-empty123",
					"type": "policy-set-outcomes",
					"attributes": {
						"policy-set-name": "test-opa-set",
						"policy-set-description": "",
						"error": "",
						"overridable": false,
						"result_count": {
							"passed": 1,
							"advisory_failed": 0,
							"mandatory_failed": 0,
							"errored": 0
						},
						"outcomes": [
							{
								"policy_name": "my-opa-policy",
								"enforcement_level": "mandatory",
								"status": "passed",
								"query": "data.example.allow",
								"description": ""
							}
						]
					}
				}
			]
		}`)
	}))
	defer testServer.Close()

	client, err := NewClient(&Config{
		Address: testServer.URL,
		Token:   "fake-token",
	})
	require.NoError(t, err)

	ctx := context.Background()

	outcomes, err := client.PolicySetOutcomes.List(ctx, policyEvaluationID, nil)
	require.NoError(t, err)
	require.Len(t, outcomes.Items, 1)
	require.Len(t, outcomes.Items[0].Outcomes, 1)

	outcome := outcomes.Items[0].Outcomes[0]
	assert.Equal(t, "my-opa-policy", outcome.PolicyName)
	assert.Empty(t, outcome.Output, "Output should be empty when no print statements are present")
}
