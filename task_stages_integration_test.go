package tfe

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTaskStagesRead(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	defer orgTestCleanup()

	runTaskTest, runTaskTestCleanup := createRunTask(t, client, orgTest)
	defer runTaskTestCleanup()

	wkspaceTest, wkspaceTestCleanup := createWorkspace(t, client, orgTest)
	defer wkspaceTestCleanup()

	wrTaskTest, wrTaskTestCleanup := createWorkspaceRunTask(t, client, wkspaceTest, runTaskTest)
	defer wrTaskTestCleanup()

	rTest, rTestCleanup := createRun(t, client, wkspaceTest)
	defer rTestCleanup()

	t.Run("without include param", func(t *testing.T) {
		r, err := client.Runs.ReadWithOptions(ctx, rTest.ID, &RunReadOptions{
			Include: "task_stages",
		})
		require.NoError(t, err)

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
	})

	t.Run("with include param task_results", func(t *testing.T) {
		r, err := client.Runs.ReadWithOptions(ctx, rTest.ID, &RunReadOptions{
			Include: "task_stages",
		})
		require.NoError(t, err)

		taskStage, err := client.TaskStages.Read(ctx, r.TaskStages[0].ID, &TaskStageReadOptions{
			Include: "task_results",
		})
		require.NoError(t, err)

		t.Run("task results are properly decoded", func(t *testing.T) {
			assert.NotEmpty(t, taskStage.TaskResults[0].ID)
			assert.NotEmpty(t, taskStage.TaskResults[0].Status)
			assert.NotEmpty(t, taskStage.TaskResults[0].CreatedAt)
			assert.Equal(t, wrTaskTest.ID, taskStage.TaskResults[0].WorkspaceTaskID)
			assert.Equal(t, runTaskTest.Name, taskStage.TaskResults[0].TaskName)
		})
	})
}

func TestTaskStagesList(t *testing.T) {
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

	_, wrTaskTestCleanup := createWorkspaceRunTask(t, client, wkspaceTest, runTaskTest)
	defer wrTaskTestCleanup()

	_, wrTaskTest2Cleanup := createWorkspaceRunTask(t, client, wkspaceTest, runTaskTest2)
	defer wrTaskTest2Cleanup()

	rTest, rTestCleanup := createRun(t, client, wkspaceTest)
	defer rTestCleanup()

	t.Run("with no params", func(t *testing.T) {
		taskStageList, err := client.TaskStages.List(ctx, rTest.ID, nil)
		require.NoError(t, err)

		assert.NotNil(t, taskStageList.Items)
		assert.NotEmpty(t, taskStageList.Items[0].ID)
		assert.Equal(t, 2, len(taskStageList.Items[0].TaskResults))
	})
}
