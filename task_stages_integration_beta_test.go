//go:build integration
// +build integration

package tfe

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTaskStagesRead_Beta(t *testing.T) {
	skipIfNotCINode(t)
	skipIfFreeOnly(t)
	skipIfBeta(t)

	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	defer orgTestCleanup()

	runTaskTest, runTaskTestCleanup := createRunTask(t, client, orgTest)
	defer runTaskTestCleanup()

	wkspaceTest, wkspaceTestCleanup := createWorkspace(t, client, orgTest)
	defer wkspaceTestCleanup()

	options := PolicyCreateOptions{
		Description: String("A sample policy"),
		Kind:        OPA,
		Query:       String("data.example.rule"),
		Enforce: []*EnforcementOptions{
			{
				Path: String(".rego"),
				Mode: EnforcementMode(EnforcementAdvisory),
			},
		},
	}
	policyTest, policyTestCleanup := createUploadedPolicyWithOptions(t, client, true, orgTest, options)
	defer policyTestCleanup()

	policySet := []*Policy{policyTest}
	_, psTestCleanup1 := createPolicySet(t, client, orgTest, policySet, []*Workspace{wkspaceTest}, OPA)
	defer psTestCleanup1()

	wrTaskTest, wrTaskTestCleanup := createWorkspaceRunTask(t, client, wkspaceTest, runTaskTest)
	defer wrTaskTestCleanup()

	rTest, rTestCleanup := createRun(t, client, wkspaceTest)
	defer rTestCleanup()

	r, err := client.Runs.ReadWithOptions(ctx, rTest.ID, &RunReadOptions{
		Include: []RunIncludeOpt{RunTaskStages},
	})
	require.NoError(t, err)
	require.NotEmpty(t, r.TaskStages)
	require.NotNil(t, r.TaskStages[0])

	t.Run("without read options", func(t *testing.T) {
		taskStage, err := client.TaskStages.Read(ctx, r.TaskStages[0].ID, nil)
		require.NoError(t, err)

		assert.NotEmpty(t, taskStage.ID)
		assert.NotEmpty(t, taskStage.Stage)
		assert.NotNil(t, taskStage.StatusTimestamps.ErroredAt)
		assert.NotNil(t, taskStage.StatusTimestamps.RunningAt)
		assert.NotNil(t, taskStage.CreatedAt)
		assert.NotNil(t, taskStage.UpdatedAt)
		assert.NotNil(t, taskStage.Run)
		assert.NotNil(t, taskStage.TaskResults)

		// so this bit is interesting, if the relation is not specified in the include
		// param, the fields of the struct will be zeroed out, minus the ID
		assert.NotEmpty(t, taskStage.TaskResults[0].ID)
		assert.Empty(t, taskStage.TaskResults[0].Status)
		assert.Empty(t, taskStage.TaskResults[0].Message)

		assert.NotEmpty(t, taskStage.PolicyEvaluation[0].ID)
	})

	t.Run("with include param task_results", func(t *testing.T) {
		taskStage, err := client.TaskStages.Read(ctx, r.TaskStages[0].ID, &TaskStageReadOptions{
			Include: []TaskStageIncludeOpt{TaskStageTaskResults, PolicyEvaluationsTaskResults},
		})
		require.NoError(t, err)
		require.NotEmpty(t, taskStage.TaskResults)
		require.NotNil(t, taskStage.TaskResults[0])
		require.NotEmpty(t, taskStage.PolicyEvaluation)
		require.NotNil(t, taskStage.PolicyEvaluation[0])

		t.Run("task results are properly decoded", func(t *testing.T) {
			assert.NotEmpty(t, taskStage.TaskResults[0].ID)
			assert.NotEmpty(t, taskStage.TaskResults[0].Status)
			assert.NotEmpty(t, taskStage.TaskResults[0].CreatedAt)
			assert.Equal(t, wrTaskTest.ID, taskStage.TaskResults[0].WorkspaceTaskID)
			assert.Equal(t, runTaskTest.Name, taskStage.TaskResults[0].TaskName)
		})

		t.Run("policy evaluations are properly decoded", func(t *testing.T) {
			assert.NotEmpty(t, taskStage.PolicyEvaluation[0].ID)
			assert.NotEmpty(t, taskStage.PolicyEvaluation[0].Status)
			assert.NotEmpty(t, taskStage.PolicyEvaluation[0].CreatedAt)
			assert.Equal(t, OPA, taskStage.PolicyEvaluation[0].PolicyKind)
			assert.NotEmpty(t, taskStage.PolicyEvaluation[0].UpdatedAt)
			assert.NotNil(t, taskStage.PolicyEvaluation[0].ResultCount)
		})

	})
}

func TestTaskStagesList_Beta(t *testing.T) {
	skipIfNotCINode(t)
	skipIfFreeOnly(t)
	skipIfBeta(t)

	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	defer orgTestCleanup()

	runTaskTest, runTaskTestCleanup := createRunTask(t, client, orgTest)
	defer runTaskTestCleanup()

	runTaskTest2, runTaskTest2Cleanup := createRunTask(t, client, orgTest)
	defer runTaskTest2Cleanup()

	wkspaceTest, wkspaceTestCleanup := createWorkspace(t, client, orgTest)
	defer wkspaceTestCleanup()

	options := PolicyCreateOptions{
		Description: String("A sample policy"),
		Kind:        OPA,
		Query:       String("data.example.rule"),
		Enforce: []*EnforcementOptions{
			{
				Path: String(".rego"),
				Mode: EnforcementMode(EnforcementAdvisory),
			},
		},
	}
	policyTest, policyTestCleanup := createUploadedPolicyWithOptions(t, client, true, orgTest, options)
	defer policyTestCleanup()

	policyTest2, policyTestCleanup2 := createUploadedPolicyWithOptions(t, client, true, orgTest, options)
	defer policyTestCleanup2()

	policySet := []*Policy{policyTest, policyTest2}
	_, psTestCleanup1 := createPolicySet(t, client, orgTest, policySet, []*Workspace{wkspaceTest}, OPA)
	defer psTestCleanup1()

	policySet2 := []*Policy{policyTest2}
	_, psTestCleanup2 := createPolicySet(t, client, orgTest, policySet2, []*Workspace{wkspaceTest}, OPA)
	defer psTestCleanup2()

	_, wrTaskTestCleanup := createWorkspaceRunTask(t, client, wkspaceTest, runTaskTest)
	defer wrTaskTestCleanup()

	_, wrTaskTest2Cleanup := createWorkspaceRunTask(t, client, wkspaceTest, runTaskTest2)
	defer wrTaskTest2Cleanup()

	rTest, rTestCleanup := createRun(t, client, wkspaceTest)
	defer rTestCleanup()

	t.Run("with no params", func(t *testing.T) {
		taskStageList, err := client.TaskStages.List(ctx, rTest.ID, nil)
		require.NoError(t, err)

		require.NotEmpty(t, taskStageList.Items)
		assert.NotEmpty(t, taskStageList.Items[0].ID)
		assert.Equal(t, 2, len(taskStageList.Items[0].TaskResults))
		assert.Equal(t, 1, len(taskStageList.Items[0].PolicyEvaluation))
	})
}

func TestTaskStageOverride_Beta(t *testing.T) {
	skipIfNotCINode(t)
	skipIfFreeOnly(t)
	skipIfBeta(t)

	client := testClient(t)
	ctx := context.Background()

	t.Run("when the policy failed", func(t *testing.T) {
		orgTest, orgTestCleanup := createOrganization(t, client)
		defer orgTestCleanup()

		options := PolicyCreateOptions{
			Description: String("A sample policy"),
			Kind:        OPA,
			Query:       String("data.example.rule"),
			Enforce: []*EnforcementOptions{
				{
					Mode: EnforcementMode(EnforcementMandatory),
				},
			},
		}
		pTest, pTestCleanup := createUploadedPolicyWithOptions(t, client, false, orgTest, options)
		defer pTestCleanup()

		wTest, wTestCleanup := createWorkspace(t, client, orgTest)
		defer wTestCleanup()
		createPolicySet(t, client, orgTest, []*Policy{pTest}, []*Workspace{wTest}, OPA)
		rTest, tTestCleanup := createRunWaitForStatus(t, client, wTest, RunAwaitingDecision)
		defer tTestCleanup()

		taskStageList, err := client.TaskStages.List(ctx, rTest.ID, nil)
		require.NoError(t, err)

		require.NotEmpty(t, taskStageList.Items)
		assert.NotEmpty(t, taskStageList.Items[0].ID)
		assert.Equal(t, TaskStageAwaitingOverride, taskStageList.Items[0].Status)
		assert.Equal(t, 1, len(taskStageList.Items[0].PolicyEvaluation))

		_, err = client.TaskStages.Override(ctx, taskStageList.Items[0].ID)
		require.NoError(t, err)
	})

	t.Run("when the policy passed", func(t *testing.T) {
		orgTest, orgTestCleanup := createOrganization(t, client)
		defer orgTestCleanup()

		options := PolicyCreateOptions{
			Description: String("A sample policy"),
			Kind:        OPA,
			Query:       String("data.example.rule"),
			Enforce: []*EnforcementOptions{
				{
					Mode: EnforcementMode(EnforcementMandatory),
				},
			},
		}
		pTest, pTestCleanup := createUploadedPolicyWithOptions(t, client, true, orgTest, options)
		defer pTestCleanup()
		wTest, wTestCleanup := createWorkspace(t, client, orgTest)
		defer wTestCleanup()
		createPolicySet(t, client, orgTest, []*Policy{pTest}, []*Workspace{wTest}, OPA)
		rTest, tTestCleanup := createRunApply(t, client, wTest)
		defer tTestCleanup()

		taskStageList, err := client.TaskStages.List(ctx, rTest.ID, nil)
		require.NoError(t, err)

		require.NotEmpty(t, taskStageList.Items)
		assert.NotEmpty(t, taskStageList.Items[0].ID)
		assert.Equal(t, TaskStagePassed, taskStageList.Items[0].Status)
		assert.Equal(t, 1, len(taskStageList.Items[0].PolicyEvaluation))

		_, err = client.TaskStages.Override(ctx, taskStageList.Items[0].ID)
		assert.Errorf(t, err, "transition not allowed")
	})
}
