package tfe

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListConfigurationVersions(t *testing.T) {
	client := testClient(t)

	ws, wsCleanup := createWorkspace(t, client, nil)
	defer wsCleanup()

	cv1, _ := createConfigurationVersion(t, client, ws)
	cv2, _ := createConfigurationVersion(t, client, ws)

	resp, err := client.ListConfigurationVersions(
		&ListConfigurationVersionsInput{
			WorkspaceID: ws.ID,
		},
	)
	require.Nil(t, err)

	found := []string{}
	for _, cv := range resp {
		found = append(found, *cv.ID)
	}

	assert.Contains(t, found, *cv1.ID)
	assert.Contains(t, found, *cv2.ID)
}

func TestCreateConfigurationVersion(t *testing.T) {
	client := testClient(t)

	ws, wsCleanup := createWorkspace(t, client, nil)
	defer wsCleanup()

	t.Run("with valid input", func(t *testing.T) {
		input := &CreateConfigurationVersionInput{
			WorkspaceID: ws.ID,
		}
		resp, err := client.CreateConfigurationVersion(input)
		require.Nil(t, err)

		// Get a refreshed view of the configuration version.
		refreshed, err := client.ConfigurationVersion(*resp.ConfigurationVersion.ID)
		require.Nil(t, err)

		for _, cv := range []*ConfigurationVersion{
			resp.ConfigurationVersion,
			refreshed,
		} {
			assert.NotNil(t, cv.ID)
			// TODO: Fix this. API does not return workspace associations.
			//assert.Equal(t, input.WorkspaceID, cv.WorkspaceID)
			assert.NotNil(t, cv.UploadURL)
			assert.NotEqual(t, 0, len(*cv.UploadURL))
			assert.Equal(t, *cv.Status, "pending")
			assert.Equal(t, *cv.Source, "tfe-api")
			assert.Nil(t, cv.Error)
		}
	})

	t.Run("with invalid workspace id", func(t *testing.T) {
		result, err := client.CreateConfigurationVersion(
			&CreateConfigurationVersionInput{
				WorkspaceID: String("! / nope"),
			},
		)
		assert.Nil(t, result)
		assert.EqualError(t, err, "Invalid value for WorkspaceID")
	})
}
