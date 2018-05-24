package tfe

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunsList(t *testing.T) {
	client := testClient(t)

	wTest, wTestCleanup := createWorkspace(t, client, nil)
	defer wTestCleanup()

	rTest1, _ := createRun(t, client, wTest)
	rTest2, _ := createRun(t, client, wTest)

	rs, err := client.Runs.List(wTest.ID, nil)
	require.Nil(t, err)

	found := []string{}
	for _, r := range rs {
		found = append(found, r.ID)
	}

	assert.Contains(t, found, rTest1.ID)
	assert.Contains(t, found, rTest2.ID)
}

func TestRunsCreate(t *testing.T) {
	client := testClient(t)

	wTest, wTestCleanup := createWorkspace(t, client, nil)
	defer wTestCleanup()

	cvTest, _ := createUploadedConfigurationVersion(t, client, wTest)

	t.Run("without a configuration version", func(t *testing.T) {
		options := &CreateRunOptions{
			Workspace: wTest,
		}

		_, err := client.Runs.Create(options)
		require.Nil(t, err)
	})

	t.Run("with a configuration version", func(t *testing.T) {
		options := &CreateRunOptions{
			ConfigurationVersion: cvTest,
			Workspace:            wTest,
		}

		r, err := client.Runs.Create(options)
		require.Nil(t, err)

		assert.Equal(t, cvTest.ID, r.ConfigurationVersion.ID)
	})

	t.Run("with additional attributes", func(t *testing.T) {
		options := &CreateRunOptions{
			Message:   String("yo"),
			Workspace: wTest,
		}

		r, err := client.Runs.Create(options)
		require.Nil(t, err)

		assert.Equal(t, *options.Message, r.Message)
	})
}

func TestRunsRetrieve(t *testing.T) {
	client := testClient(t)

	rTest, rTestCleanup := createRun(t, client, nil)
	defer rTestCleanup()

	t.Run("when the run exists", func(t *testing.T) {
		r, err := client.Runs.Retrieve(rTest.ID)
		assert.Nil(t, err)
		assert.Equal(t, rTest, r)
	})

	t.Run("when the run does not exist", func(t *testing.T) {
		r, err := client.Runs.Retrieve("nonexisting")
		assert.Nil(t, r)
		assert.EqualError(t, err, "Resource not found")
	})

	t.Run("with invalid run ID", func(t *testing.T) {
		r, err := client.Runs.Retrieve(badIdentifier)
		assert.Nil(t, r)
		assert.EqualError(t, err, "Invalid value for run ID")
	})
}
