package tfe

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateRun(t *testing.T) {
	client := testClient(t)

	ws, wsCleanup := createWorkspace(t, client, nil)
	defer wsCleanup()

	cv1, _ := createUploadedConfigurationVersion(t, client, ws)
	cv2, _ := createUploadedConfigurationVersion(t, client, ws)

	t.Run("without a configuration version", func(t *testing.T) {
		input := &CreateRunInput{
			WorkspaceID: ws.ID,
		}

		result, err := client.CreateRun(input)
		require.Nil(t, err)

		assert.Equal(t, cv2.ID, result.Run.ConfigurationVersionID)
	})

	t.Run("with a configuration version", func(t *testing.T) {
		result, err := client.CreateRun(&CreateRunInput{
			WorkspaceID:            ws.ID,
			ConfigurationVersionID: cv1.ID,
		})
		require.Nil(t, err)

		assert.Equal(t, cv1.ID, result.Run.ConfigurationVersionID)
	})

	t.Run("with additional attributes", func(t *testing.T) {
		input := &CreateRunInput{
			WorkspaceID: ws.ID,
			Message:     String("yo"),
		}

		result, err := client.CreateRun(input)
		require.Nil(t, err)

		assert.Equal(t, input.Message, result.Run.Message)
	})
}

func TestListRuns(t *testing.T) {
	client := testClient(t)

	ws, wsCleanup := createWorkspace(t, client, nil)
	defer wsCleanup()

	run1, _ := createRun(t, client, ws)
	run2, _ := createRun(t, client, ws)

	result, err := client.ListRuns(&ListRunsInput{
		WorkspaceID: ws.ID,
	})
	require.Nil(t, err)

	found := []string{}
	for _, run := range result {
		found = append(found, *run.ID)
	}

	assert.Contains(t, found, *run1.ID)
	assert.Contains(t, found, *run2.ID)
}
