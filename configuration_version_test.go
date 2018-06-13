package tfe

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfigurationVersionsCreate(t *testing.T) {
	client := testClient(t)

	wTest, wTestCleanup := createWorkspace(t, client, nil)
	defer wTestCleanup()

	t.Run("with valid options", func(t *testing.T) {
		cv, err := client.ConfigurationVersions.Create(
			wTest.ID,
			ConfigurationVersionCreateOptions{},
		)
		require.Nil(t, err)

		// Get a refreshed view of the configuration version.
		refreshed, err := client.ConfigurationVersions.Retrieve(cv.ID)
		require.Nil(t, err)

		for _, item := range []*ConfigurationVersion{
			cv,
			refreshed,
		} {
			assert.NotNil(t, item.ID)
			// TODO: Fix this. API does not return workspace associations.
			// assert.Equal(t, wTest.ID, item.Workspace.ID)
			assert.NotNil(t, item.UploadURL)
			assert.NotEqual(t, 0, len(item.UploadURL))
			assert.Equal(t, item.Status, ConfigurationPending)
			assert.Equal(t, item.Source, ConfigurationSourceAPI)
			assert.Equal(t, item.Error, "")
		}
	})

	t.Run("with invalid workspace id", func(t *testing.T) {
		cv, err := client.ConfigurationVersions.Create(
			badIdentifier,
			ConfigurationVersionCreateOptions{},
		)
		assert.Nil(t, cv)
		assert.EqualError(t, err, "Invalid value for workspace ID")
	})
}

func TestConfigurationVersionsRetrieve(t *testing.T) {
	client := testClient(t)

	cvTest, cvTestCleanup := createConfigurationVersion(t, client, nil)
	defer cvTestCleanup()

	t.Run("when the configuration version exists", func(t *testing.T) {
		cv, err := client.ConfigurationVersions.Retrieve(cvTest.ID)
		require.Nil(t, err)

		// Don't compare the UploadURL because it will be generated twice in
		// this test - once at creation of the configuration version, and
		// again during the GET.
		cvTest.UploadURL, cv.UploadURL = "", ""

		assert.Equal(t, cvTest, cv)
	})

	t.Run("when the configuration version does not exist", func(t *testing.T) {
		cv, err := client.ConfigurationVersions.Retrieve("nonexisting")
		assert.Nil(t, cv)
		assert.EqualError(t, err, "Resource not found")
	})

	t.Run("with invalid configuration version id", func(t *testing.T) {
		cv, err := client.ConfigurationVersions.Retrieve(badIdentifier)
		assert.Nil(t, cv)
		assert.EqualError(t, err, "Invalid value for configuration version ID")
	})
}
