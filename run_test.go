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

	t.Run("without list options", func(t *testing.T) {
		rs, err := client.Runs.List(wTest.ID, RunListOptions{})
		require.NoError(t, err)

		found := []string{}
		for _, r := range rs {
			found = append(found, r.ID)
		}

		assert.Contains(t, found, rTest1.ID)
		assert.Contains(t, found, rTest2.ID)
	})

	t.Run("with list options", func(t *testing.T) {
		t.Skip("paging not supported yet in API")

		// Request a page number which is out of range. The result should
		// be successful, but return no results if the paging options are
		// properly passed along.
		rs, err := client.Runs.List(wTest.ID, RunListOptions{
			ListOptions: ListOptions{
				PageNumber: 999,
				PageSize:   100,
			},
		})
		require.NoError(t, err)
		assert.Empty(t, rs)
	})

	t.Run("without a valid workspace ID", func(t *testing.T) {
		rs, err := client.Runs.List(badIdentifier, RunListOptions{})
		assert.Nil(t, rs)
		assert.EqualError(t, err, "Invalid value for workspace ID")
	})
}

func TestRunsCreate(t *testing.T) {
	client := testClient(t)

	wTest, wTestCleanup := createWorkspace(t, client, nil)
	defer wTestCleanup()

	cvTest, _ := createUploadedConfigurationVersion(t, client, wTest)

	t.Run("without a configuration version", func(t *testing.T) {
		options := RunCreateOptions{
			Workspace: wTest,
		}

		_, err := client.Runs.Create(options)
		assert.NoError(t, err)
	})

	t.Run("with a configuration version", func(t *testing.T) {
		options := RunCreateOptions{
			ConfigurationVersion: cvTest,
			Workspace:            wTest,
		}

		r, err := client.Runs.Create(options)
		require.NoError(t, err)
		assert.Equal(t, cvTest.ID, r.ConfigurationVersion.ID)
	})

	t.Run("without a workspace", func(t *testing.T) {
		r, err := client.Runs.Create(RunCreateOptions{})
		assert.Nil(t, r)
		assert.EqualError(t, err, "Workspace is required")
	})

	t.Run("with additional attributes", func(t *testing.T) {
		options := RunCreateOptions{
			Message:   String("yo"),
			Workspace: wTest,
		}

		r, err := client.Runs.Create(options)
		require.NoError(t, err)
		assert.Equal(t, *options.Message, r.Message)
	})
}

func TestRunsRetrieve(t *testing.T) {
	client := testClient(t)

	rTest, rTestCleanup := createPlannedRun(t, client, nil)
	defer rTestCleanup()

	t.Run("when the run exists", func(t *testing.T) {
		r, err := client.Runs.Retrieve(rTest.ID)
		assert.NoError(t, err)
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

func TestRunsApply(t *testing.T) {
	client := testClient(t)

	rTest, rTestCleanup := createPlannedRun(t, client, nil)
	defer rTestCleanup()

	t.Run("when the run exists", func(t *testing.T) {
		err := client.Runs.Apply(rTest.ID, RunApplyOptions{})
		assert.NoError(t, err)
	})

	t.Run("when the run does not exist", func(t *testing.T) {
		err := client.Runs.Apply("nonexisting", RunApplyOptions{})
		assert.EqualError(t, err, "Resource not found")
	})

	t.Run("with invalid run ID", func(t *testing.T) {
		err := client.Runs.Apply(badIdentifier, RunApplyOptions{})
		assert.EqualError(t, err, "Invalid value for run ID")
	})
}

func TestRunsCancel(t *testing.T) {
	client := testClient(t)

	wTest, wTestCleanup := createWorkspace(t, client, nil)
	defer wTestCleanup()

	// We need to create 2 runs here. The first run will automatically
	// be planned so that one cannot be cancelled. The second one will
	// be pending until the first one is confirmed or discarded, so we
	// can cancel that one.
	_, _ = createRun(t, client, wTest)
	rTest2, _ := createRun(t, client, wTest)

	t.Run("when the run exists", func(t *testing.T) {
		err := client.Runs.Cancel(rTest2.ID, RunCancelOptions{})
		assert.NoError(t, err)
	})

	t.Run("when the run does not exist", func(t *testing.T) {
		err := client.Runs.Cancel("nonexisting", RunCancelOptions{})
		assert.EqualError(t, err, "Resource not found")
	})

	t.Run("with invalid run ID", func(t *testing.T) {
		err := client.Runs.Cancel(badIdentifier, RunCancelOptions{})
		assert.EqualError(t, err, "Invalid value for run ID")
	})
}

func TestRunsDiscard(t *testing.T) {
	client := testClient(t)

	rTest, rTestCleanup := createPlannedRun(t, client, nil)
	defer rTestCleanup()

	t.Run("when the run exists", func(t *testing.T) {
		err := client.Runs.Discard(rTest.ID, RunDiscardOptions{})
		assert.NoError(t, err)
	})

	t.Run("when the run does not exist", func(t *testing.T) {
		err := client.Runs.Discard("nonexisting", RunDiscardOptions{})
		assert.EqualError(t, err, "Resource not found")
	})

	t.Run("with invalid run ID", func(t *testing.T) {
		err := client.Runs.Discard(badIdentifier, RunDiscardOptions{})
		assert.EqualError(t, err, "Invalid value for run ID")
	})
}
