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

	// hardcoded currently
	taskStageID := "ts-xzskdpGj36B4ZJGn"

	t.Run("without include param", func(t *testing.T) {
		taskStage, err := client.TaskStages.Read(ctx, taskStageID, nil)
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
		taskStage, err := client.TaskStages.Read(ctx, taskStageID, &TaskStageReadOptions{
			Include: "task_results",
		})
		require.NoError(t, err)

		t.Run("task results are properly decoded", func(t *testing.T) {
			assert.NotEmpty(t, taskStage.TaskResults[0].Status)
			assert.NotEmpty(t, taskStage.TaskResults[0].Message)
		})
	})
}
