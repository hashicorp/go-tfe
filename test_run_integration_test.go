package tfe

import (
	"context"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTestRunsList(t *testing.T) {
	skipUnlessBeta(t)
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	defer orgTestCleanup()

	rmTest, registryModuleTestCleanup := createBranchBasedRegistryModule(t, client, orgTest)
	defer registryModuleTestCleanup()

	id := RegistryModuleID{
		Organization: orgTest.Name,
		Name:         rmTest.Name,
		Provider:     rmTest.Provider,
		Namespace:    rmTest.Namespace,
		RegistryName: rmTest.RegistryName,
	}

	trTest1, trTestCleanup1 := createTestRun(t, client, rmTest)
	trTest2, trTestCleanup2 := createTestRun(t, client, rmTest)

	defer trTestCleanup1()
	defer trTestCleanup2()

	t.Run("without list options", func(t *testing.T) {
		trl, err := client.TestRuns.List(ctx, id, nil)
		var found []string
		for _, r := range trl.Items {
			found = append(found, r.ID)
		}

		require.NoError(t, err)
		assert.Contains(t, found, trTest1.ID)
		assert.Contains(t, found, trTest2.ID)
		assert.Equal(t, 1, trl.CurrentPage)
		assert.Equal(t, 2, trl.TotalCount)
	})

	t.Run("empty list options", func(t *testing.T) {
		trl, err := client.TestRuns.List(ctx, id, &TestRunListOptions{})
		var found []string
		for _, r := range trl.Items {
			found = append(found, r.ID)
		}

		require.NoError(t, err)
		assert.Contains(t, found, trTest1.ID)
		assert.Contains(t, found, trTest2.ID)
		assert.Equal(t, 1, trl.CurrentPage)
		assert.Equal(t, 2, trl.TotalCount)
	})

	t.Run("with page size", func(t *testing.T) {
		// Request a page number which is out of range. The result should
		// be successful, but return no results if the paging options are
		// properly passed along.
		trl, err := client.TestRuns.List(ctx, id, &TestRunListOptions{
			ListOptions: ListOptions{
				PageNumber: 999,
				PageSize:   100,
			},
		})

		require.NoError(t, err)
		assert.Empty(t, trl.Items)
		assert.Equal(t, 999, trl.CurrentPage)
		assert.Equal(t, 2, trl.TotalCount)
	})
}

func TestTestRunsRead(t *testing.T) {
	skipUnlessBeta(t)
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	defer orgTestCleanup()

	rmTest, registryModuleTestCleanup := createBranchBasedRegistryModule(t, client, orgTest)
	defer registryModuleTestCleanup()

	id := RegistryModuleID{
		Organization: orgTest.Name,
		Name:         rmTest.Name,
		Provider:     rmTest.Provider,
		Namespace:    rmTest.Namespace,
		RegistryName: rmTest.RegistryName,
	}

	trTest, trTestCleanup := createTestRun(t, client, rmTest)
	defer trTestCleanup()

	t.Run("when the test run exists", func(t *testing.T) {
		tr, err := client.TestRuns.Read(ctx, id, trTest.ID)
		require.NoError(t, err)
		require.Equal(t, trTest.ID, tr.ID)
	})
	t.Run("when the test run does not exist", func(t *testing.T) {
		_, err := client.TestRuns.Read(ctx, id, "trun-NoTaReAlId")
		require.Error(t, err)
		require.Equal(t, ErrResourceNotFound, err)
	})
}

func TestTestRunsCreate(t *testing.T) {
	skipUnlessBeta(t)
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	defer orgTestCleanup()

	rmTest, rmTestCleanup := createBranchBasedRegistryModule(t, client, orgTest)
	defer rmTestCleanup()

	cvTest, cvTestCleanup := createUploadedTestConfigurationVersion(t, client, rmTest)
	defer cvTestCleanup()

	t.Run("with a configuration version", func(t *testing.T) {
		options := TestRunCreateOptions{
			ConfigurationVersion: cvTest,
			RegistryModule:       rmTest,
		}

		_, err := client.TestRuns.Create(ctx, options)
		require.NoError(t, err)
	})
	t.Run("without a configuration version", func(t *testing.T) {
		options := TestRunCreateOptions{
			RegistryModule: rmTest,
		}

		_, err := client.TestRuns.Create(ctx, options)
		require.Equal(t, ErrInvalidConfigVersionID, err)
	})
	t.Run("without a module", func(t *testing.T) {
		options := TestRunCreateOptions{
			ConfigurationVersion: cvTest,
		}

		_, err := client.TestRuns.Create(ctx, options)
		require.Equal(t, ErrRequiredRegistryModule, err)
	})
}

func TestTestRunsLogs(t *testing.T) {
	skipUnlessBeta(t)
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	defer orgTestCleanup()

	rmTest, rmTestCleanup := createBranchBasedRegistryModule(t, client, orgTest)
	defer rmTestCleanup()

	id := RegistryModuleID{
		Organization: orgTest.Name,
		Name:         rmTest.Name,
		Provider:     rmTest.Provider,
		Namespace:    rmTest.Namespace,
		RegistryName: rmTest.RegistryName,
	}

	tr, trCleanup := createTestRun(t, client, rmTest)
	defer trCleanup()

	t.Run("when the log exists", func(t *testing.T) {
		waitUntilTestRunStatus(t, client, id, tr, TestRunFinished, 15)

		logReader, err := client.TestRuns.Logs(ctx, id, tr.ID)
		require.NoError(t, err)

		logs, err := io.ReadAll(logReader)
		require.NoError(t, err)

		assert.Contains(t, string(logs), "Success!")
	})

	t.Run("when the log does not exist", func(t *testing.T) {
		logs, err := client.TestRuns.Logs(ctx, id, "notreal")
		assert.Nil(t, logs)
		assert.Error(t, err)
	})
}

func TestTestRunsCancel(t *testing.T) {
	skipUnlessBeta(t)
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	defer orgTestCleanup()

	rmTest, rmTestCleanup := createBranchBasedRegistryModule(t, client, orgTest)
	defer rmTestCleanup()

	id := RegistryModuleID{
		Organization: orgTest.Name,
		Name:         rmTest.Name,
		Provider:     rmTest.Provider,
		Namespace:    rmTest.Namespace,
		RegistryName: rmTest.RegistryName,
	}

	tr, trCleanup := createTestRun(t, client, rmTest, &RunVariable{
		Key:   "wait_time",
		Value: "5s", // Create a long-running test run that we'll have time to cancel.
	})
	defer trCleanup()

	t.Run("when the run exists", func(t *testing.T) {
		err := client.TestRuns.Cancel(ctx, id, tr.ID)
		require.NoError(t, err)
	})

	/* TODO: Enable force cancel test when supported.
	t.Run("can force cancel", func(t *testing.T) {
		  var err error

		for i := 1; ; i++ {
		  	tr, err = client.TestRuns.Read(ctx, id, tr.ID)
		  	require.NoError(t, err)

			// TODO: Check if we can force cancel yet, not available in the
	        //       API yet.

			if i > 30 {
				t.Fatal("Timeout waiting for run to be canceled")
			}

			time.Sleep(time.Second)
		}
	})
	*/

	t.Run("when the run does not exist", func(t *testing.T) {
		err := client.TestRuns.Cancel(ctx, id, "notreal")
		assert.Equal(t, err, ErrResourceNotFound)
	})
}
