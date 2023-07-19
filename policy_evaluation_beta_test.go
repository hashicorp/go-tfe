// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package tfe

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPolicyEvaluationList_Beta(t *testing.T) {
	skipUnlessBeta(t)

	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	defer orgTestCleanup()

	wkspaceTest, wkspaceTestCleanup := createWorkspace(t, client, orgTest)
	defer wkspaceTestCleanup()

	options := PolicyCreateOptions{
		Description: String("A sample policy"),
		Kind:        OPA,
		Query:       String("data.example.rule"),
		Enforce: []*EnforcementOptions{
			{
				Mode: EnforcementMode(EnforcementAdvisory),
			},
		},
	}
	policyTest, policyTestCleanup := createUploadedPolicyWithOptions(t, client, true, orgTest, options)
	defer policyTestCleanup()

	policySet := []*Policy{policyTest}
	_, psTestCleanup1 := createPolicySet(t, client, orgTest, policySet, []*Workspace{wkspaceTest}, nil, OPA)
	defer psTestCleanup1()

	rTest, rTestCleanup := createRun(t, client, wkspaceTest)
	defer rTestCleanup()

	t.Run("with no params", func(t *testing.T) {
		taskStageList, err := client.TaskStages.List(ctx, rTest.ID, nil)
		require.NoError(t, err)

		require.NotEmpty(t, taskStageList.Items)
		assert.NotEmpty(t, taskStageList.Items[0].ID)
		assert.Equal(t, 1, len(taskStageList.Items[0].PolicyEvaluations))

		polEvaluation, err := client.PolicyEvaluations.List(ctx, taskStageList.Items[0].ID, nil)
		require.NoError(t, err)

		require.NotEmpty(t, polEvaluation.Items)
		assert.NotEmpty(t, polEvaluation.Items[0].ID)
	})

	t.Run("with a invalid policy evaluation ID", func(t *testing.T) {
		policyEvaluationeID := "invalid ID"

		_, err := client.PolicyEvaluations.List(ctx, policyEvaluationeID, nil)
		require.Errorf(t, err, "invalid value for policy evaluation ID")
	})
}

func TestPolicySetOutcomeList_Beta(t *testing.T) {
	skipUnlessBeta(t)

	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	defer orgTestCleanup()

	wkspaceTest, wkspaceTestCleanup := createWorkspace(t, client, orgTest)
	defer wkspaceTestCleanup()

	options := PolicyCreateOptions{
		Description: String("A sample policy"),
		Kind:        OPA,
		Query:       String("data.example.rule"),
		Enforce: []*EnforcementOptions{
			{
				Mode: EnforcementMode(EnforcementAdvisory),
			},
		},
	}
	policyTest, policyTestCleanup := createUploadedPolicyWithOptions(t, client, true, orgTest, options)
	defer policyTestCleanup()

	policySet := []*Policy{policyTest}
	_, psTestCleanup1 := createPolicySet(t, client, orgTest, policySet, []*Workspace{wkspaceTest}, nil, OPA)
	defer psTestCleanup1()

	rTest, rTestCleanup := createPlannedRun(t, client, wkspaceTest)
	defer rTestCleanup()

	t.Run("with no params", func(t *testing.T) {
		taskStageList, err := client.TaskStages.List(ctx, rTest.ID, nil)
		require.NoError(t, err)

		require.NotEmpty(t, taskStageList.Items)
		assert.NotEmpty(t, taskStageList.Items[0].ID)
		assert.Equal(t, 1, len(taskStageList.Items[0].PolicyEvaluations))
		assert.NotEmpty(t, 1, len(taskStageList.Items[0].PolicyEvaluations[0].ID))

		polEvaluationID := taskStageList.Items[0].PolicyEvaluations[0].ID

		polSetOutcomesList, err := client.PolicySetOutcomes.List(ctx, polEvaluationID, nil)
		require.NoError(t, err)

		require.NotEmpty(t, polSetOutcomesList.Items)
		assert.NotEmpty(t, polSetOutcomesList.Items[0].ID)
		assert.NotEmpty(t, polSetOutcomesList.Items[0].Outcomes)
		assert.NotEmpty(t, polSetOutcomesList.Items[0].PolicySetName)
	})

	t.Run("with non-matching filters", func(t *testing.T) {
		taskStageList, err := client.TaskStages.List(ctx, rTest.ID, nil)
		require.NoError(t, err)

		require.NotEmpty(t, taskStageList.Items)
		assert.NotEmpty(t, taskStageList.Items[0].ID)
		assert.Equal(t, 1, len(taskStageList.Items[0].PolicyEvaluations))
		assert.NotEmpty(t, 1, len(taskStageList.Items[0].PolicyEvaluations[0].ID))

		polEvaluationID := taskStageList.Items[0].PolicyEvaluations[0].ID

		opts := &PolicySetOutcomeListOptions{
			Filter: map[string]PolicySetOutcomeListFilter{
				"0": {
					Status: "errored",
				},
				"1": {
					EnforcementLevel: "mandatory",
					Status:           "failed",
				},
			},
		}

		polSetOutcomesList, err := client.PolicySetOutcomes.List(ctx, polEvaluationID, opts)
		require.NoError(t, err)

		require.Empty(t, polSetOutcomesList.Items)
	})

	t.Run("with matching filters", func(t *testing.T) {
		taskStageList, err := client.TaskStages.List(ctx, rTest.ID, nil)
		require.NoError(t, err)

		require.NotEmpty(t, taskStageList.Items)
		assert.NotEmpty(t, taskStageList.Items[0].ID)
		assert.Equal(t, 1, len(taskStageList.Items[0].PolicyEvaluations))
		assert.NotEmpty(t, 1, len(taskStageList.Items[0].PolicyEvaluations[0].ID))

		polEvaluationID := taskStageList.Items[0].PolicyEvaluations[0].ID

		opts := &PolicySetOutcomeListOptions{
			Filter: map[string]PolicySetOutcomeListFilter{
				"0": {
					Status:           "passed",
					EnforcementLevel: "advisory",
				},
			},
		}

		polSetOutcomesList, err := client.PolicySetOutcomes.List(ctx, polEvaluationID, opts)
		require.NoError(t, err)

		require.NotEmpty(t, polSetOutcomesList.Items)
		assert.NotEmpty(t, polSetOutcomesList.Items[0].ID)
		assert.Equal(t, 1, len(polSetOutcomesList.Items[0].Outcomes))
		assert.NotEmpty(t, polSetOutcomesList.Items[0].PolicySetName)
	})
}

func TestPolicySetOutcomeRead_Beta(t *testing.T) {
	skipUnlessBeta(t)

	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	defer orgTestCleanup()

	wkspaceTest, wkspaceTestCleanup := createWorkspace(t, client, orgTest)
	defer wkspaceTestCleanup()

	options := PolicyCreateOptions{
		Description: String("A sample policy"),
		Kind:        OPA,
		Query:       String("data.example.rule"),
		Enforce: []*EnforcementOptions{
			{
				Mode: EnforcementMode(EnforcementAdvisory),
			},
		},
	}
	policyTest, policyTestCleanup := createUploadedPolicyWithOptions(t, client, true, orgTest, options)
	defer policyTestCleanup()

	policySet := []*Policy{policyTest}
	_, psTestCleanup1 := createPolicySet(t, client, orgTest, policySet, []*Workspace{wkspaceTest}, nil, OPA)
	defer psTestCleanup1()

	rTest, rTestCleanup := createPlannedRun(t, client, wkspaceTest)
	defer rTestCleanup()

	t.Run("with a valid policy set outcome ID", func(t *testing.T) {
		taskStageList, err := client.TaskStages.List(ctx, rTest.ID, nil)
		require.NoError(t, err)

		require.NotEmpty(t, taskStageList.Items)
		assert.NotEmpty(t, taskStageList.Items[0].ID)
		assert.Equal(t, 1, len(taskStageList.Items[0].PolicyEvaluations))
		assert.NotEmpty(t, 1, len(taskStageList.Items[0].PolicyEvaluations[0].ID))

		polEvaluationID := taskStageList.Items[0].PolicyEvaluations[0].ID

		polSetOutcomesList, err := client.PolicySetOutcomes.List(ctx, polEvaluationID, nil)
		require.NoError(t, err)

		require.NotEmpty(t, polSetOutcomesList.Items)
		assert.NotEmpty(t, polSetOutcomesList.Items[0].ID)
		assert.NotEmpty(t, polSetOutcomesList.Items[0].Outcomes)
		assert.NotEmpty(t, polSetOutcomesList.Items[0].PolicySetName)

		policySetOutcomeID := polSetOutcomesList.Items[0].ID

		policyOutcome, err := client.PolicySetOutcomes.Read(ctx, policySetOutcomeID)
		require.NoError(t, err)

		assert.NotEmpty(t, policyOutcome.ID)
		assert.NotEmpty(t, policyOutcome.Outcomes)
	})

	t.Run("with a invalid policy set outcome ID", func(t *testing.T) {
		policySetOutcomeID := "invalid ID"

		_, err := client.PolicySetOutcomes.Read(ctx, policySetOutcomeID)
		require.Errorf(t, err, "invalid value for policy set outcome ID")
	})
}
