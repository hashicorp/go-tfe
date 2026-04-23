package tfe

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// NOTE: Concrete tests for this feature are intentionally omitted.
// The `tfpolicy` feature is currently in beta and its behavior/interface
// is subject to change. Tests will be added once the feature stabilizes.
func TestTFPolicyEvaluationOutcomes_List(t *testing.T) {
	skipUnlessBeta(t)
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganizationWithOptions(t, client, OrganizationCreateOptions{
		Name:  String("tfpolicy-test" + randomString(t)),
		Email: String("tfpolicy-test-" + randomString(t) + "@hashicorp.com"),
	})
	defer orgTestCleanup()

	ws, wsCleanup := createWorkspace(t, client, orgTest)
	defer wsCleanup()

	githubIdentifer := os.Getenv("GITHUB_POLICY_SET_IDENTIFIER")
	if githubIdentifer == "" {
		t.Skip("Export a valid GITHUB_POLICY_SET_IDENTIFIER before running this test")
	}

	oc, _ := createOAuthToken(t, client, orgTest)

	options := PolicySetCreateOptions{
		Name: String("tfpolicy-policy-set"),
		Kind: TFPolicy,
		VCSRepo: &VCSRepoOptions{
			Branch:       String("tfpolicy"),
			Identifier:   String(githubIdentifer),
			OAuthTokenID: String(oc.ID),
		},
	}

	_, err := client.PolicySets.Create(ctx, orgTest.Name, options)
	require.NoError(t, err)

	_, rTestCleanup := createRun(t, client, ws)
	defer rTestCleanup()

	// NOTE: TFEvaluations for run ID is not yet supported,
	// hence we will be using Run.
	// List with include param to verify that the API is
	// working as expected.

	t.Run("with no params", func(t *testing.T) {
		rData, err := client.Runs.List(ctx, ws.ID, &RunListOptions{
			Include: []RunIncludeOpt{
				RunTFPolicyEvaluation,
			},
		})

		require.NoError(t, err)

		require.NotEmpty(t, rData.Items)
		assert.NotEmpty(t, rData.Items[0].ID)
		assert.NotEmpty(t, rData.Items[0].TFPolicyEvaluations)

		assert.NotEmpty(t, rData.Items[0].TFPolicyEvaluations[0].ID)

		evaluationOutcome, err := client.TFPolicyEvaluationOutcomes.List(ctx, rData.Items[0].TFPolicyEvaluations[0].ID, nil)
		require.NoError(t, err)

		require.NotEmpty(t, evaluationOutcome.Items)
		assert.NotEmpty(t, evaluationOutcome.Items[0].ID)
	})
}
