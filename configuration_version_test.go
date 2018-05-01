package tfe

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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

func TestConfigurationVersion(t *testing.T) {
	client := testClient(t)

	cv, cvCleanup := createConfigurationVersion(t, client, nil)
	defer cvCleanup()

	t.Run("when the configuration version exists", func(t *testing.T) {
		result, err := client.ConfigurationVersion(*cv.ID)
		require.Nil(t, err)

		// Don't compare the UploadURL because it will be generated twice in
		// this test - once at creation of the configuration version, and
		// again during the GET.
		cv.UploadURL, result.UploadURL = nil, nil

		assert.Equal(t, cv, result)
	})

	t.Run("when the configuration version does not exist", func(t *testing.T) {
		result, err := client.ConfigurationVersion("nope")
		assert.Nil(t, result)
		assert.EqualError(t, err, "Resource not found")
	})
}
