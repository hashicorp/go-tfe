// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package tfe

import (
	"context"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// createQueryRun creates a query run in the given workspace.
// A configuration version is created and uploaded to ensure the run can be processed.
func createQueryRun(t *testing.T, client *Client, workspace *Workspace) *QueryRun {
	t.Helper()
	createUploadedConfigurationVersion(t, client, workspace)
	options := QueryRunCreateOptions{
		Workspace: workspace,
		Source:    QueryRunSourceAPI,
	}
	queryRun, err := client.QueryRuns.Create(context.Background(), options)
	require.NoError(t, err)
	return queryRun
}

func TestQueryRunsList(t *testing.T) {
	skipUnlessBeta(t)
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	defer orgTestCleanup()

	wTest, _ := createWorkspace(t, client, orgTest)
	_ = createQueryRun(t, client, wTest)
	_ = createQueryRun(t, client, wTest)

	t.Run("without list options", func(t *testing.T) {
		qrl, err := client.QueryRuns.List(ctx, wTest.ID, nil)
		require.NoError(t, err)

		// The API returns Run objects, not QueryRun objects.
		// We can't easily correlate the created QueryRun with the returned Run.
		// So we just check the count.
		assert.Len(t, qrl.Items, 2)
		assert.Equal(t, 1, qrl.CurrentPage)
		assert.Equal(t, 2, qrl.TotalCount)
	})

	t.Run("without list options and include as nil", func(t *testing.T) {
		qrl, err := client.QueryRuns.List(ctx, wTest.ID, &QueryRunListOptions{
			Include: []QueryRunIncludeOpt{},
		})
		require.NoError(t, err)
		require.NotEmpty(t, qrl.Items)

		assert.Len(t, qrl.Items, 2)
		assert.Equal(t, 1, qrl.CurrentPage)
		assert.Equal(t, 2, qrl.TotalCount)
	})

	t.Run("with list options", func(t *testing.T) {
		qrl, err := client.QueryRuns.List(ctx, wTest.ID, &QueryRunListOptions{
			ListOptions: ListOptions{
				PageNumber: 999,
				PageSize:   100,
			},
		})
		require.NoError(t, err)
		assert.Empty(t, qrl.Items)
		assert.Equal(t, 999, qrl.CurrentPage)
		assert.Equal(t, 2, qrl.TotalCount)
	})

	t.Run("with created_by included", func(t *testing.T) {
		qrl, err := client.QueryRuns.List(ctx, wTest.ID, &QueryRunListOptions{
			// The QueryRunIncludeOpt constants in query_runs.go have the wrong type.
			// We use a string literal here as a workaround.
			Include: []QueryRunIncludeOpt{"created-by"},
		})
		require.NoError(t, err)

		require.NotEmpty(t, qrl.Items)
		// The items are of type *Run, which has a CreatedBy field.
		require.NotNil(t, qrl.Items[0].CreatedBy)
		assert.NotEmpty(t, qrl.Items[0].CreatedBy.Username)
	})

	t.Run("without a valid workspace ID", func(t *testing.T) {
		qrl, err := client.QueryRuns.List(ctx, badIdentifier, nil)
		assert.Nil(t, qrl)
		assert.EqualError(t, err, ErrResourceNotFound.Error())
	})
}

func TestQueryRunsCreate(t *testing.T) {
	skipUnlessBeta(t)
	client := testClient(t)
	ctx := context.Background()

	wTest, wTestCleanup := createWorkspace(t, client, nil)
	defer wTestCleanup()

	cvTest, _ := createUploadedConfigurationVersion(t, client, wTest)

	t.Run("without a configuration version", func(t *testing.T) {
		options := QueryRunCreateOptions{
			Workspace: wTest,
			Source:    QueryRunSourceAPI,
		}

		qr, err := client.QueryRuns.Create(ctx, options)
		require.NoError(t, err)
		assert.NotNil(t, qr.ID)
		assert.NotNil(t, qr.CreatedAt)
		assert.NotNil(t, qr.Source)
		require.NotNil(t, qr.StatusTimestamps)
	})

	t.Run("with a configuration version", func(t *testing.T) {
		options := QueryRunCreateOptions{
			ConfigurationVersion: cvTest,
			Workspace:            wTest,
			Source:               QueryRunSourceAPI,
		}

		qr, err := client.QueryRuns.Create(ctx, options)
		require.NoError(t, err)
		require.NotNil(t, qr.ConfigurationVersion)
		assert.Equal(t, cvTest.ID, qr.ConfigurationVersion.ID)
	})

	t.Run("without a workspace", func(t *testing.T) {
		qr, err := client.QueryRuns.Create(ctx, QueryRunCreateOptions{
			Source: QueryRunSourceAPI,
		})
		assert.Nil(t, qr)
		assert.Equal(t, err, ErrRequiredWorkspace)
	})

	t.Run("with variables", func(t *testing.T) {
		t.Skip("Variables not yet implemented")

		vars := []*RunVariable{
			{
				Key:   "test_variable",
				Value: "Hello, World!",
			},
			{
				Key:   "test_foo",
				Value: "Hello, Foo!",
			},
		}

		options := QueryRunCreateOptions{
			Workspace: wTest,
			Variables: vars,
			Source:    QueryRunSourceAPI,
		}

		qr, err := client.QueryRuns.Create(ctx, options)
		require.NoError(t, err)
		assert.NotNil(t, qr.Variables)
		assert.Equal(t, len(vars), len(qr.Variables))

		for _, v := range qr.Variables {
			switch v.Key {
			case "test_foo":
				assert.Equal(t, v.Value, "Hello, Foo!")
			case "test_variable":
				assert.Equal(t, v.Value, "Hello, World!")
			default:
				t.Fatalf("Unexpected variable key: %s", v.Key)
			}
		}
	})
}

func TestQueryRunsRead(t *testing.T) {
	skipUnlessBeta(t)
	client := testClient(t)
	ctx := context.Background()

	wTest, wTestCleanup := createWorkspace(t, client, nil)
	defer wTestCleanup()
	qrTest := createQueryRun(t, client, wTest)

	t.Run("when the query run exists", func(t *testing.T) {
		qr, err := client.QueryRuns.Read(ctx, qrTest.ID)
		require.NoError(t, err)
		assert.Equal(t, qrTest.ID, qr.ID)
	})

	t.Run("when the query run does not exist", func(t *testing.T) {
		qr, err := client.QueryRuns.Read(ctx, "nonexisting")
		assert.Nil(t, qr)
		assert.Equal(t, err, ErrResourceNotFound)
	})

	t.Run("with invalid query run ID", func(t *testing.T) {
		qr, err := client.QueryRuns.Read(ctx, badIdentifier)
		assert.Nil(t, qr)
		assert.EqualError(t, err, ErrResourceNotFound.Error())
	})
}

func TestQueryRunsReadWithOptions(t *testing.T) {
	skipUnlessBeta(t)
	client := testClient(t)
	ctx := context.Background()

	wTest, wTestCleanup := createWorkspace(t, client, nil)
	defer wTestCleanup()
	qrTest := createQueryRun(t, client, wTest)

	t.Run("when the query run exists", func(t *testing.T) {
		curOpts := &QueryRunReadOptions{
			// The QueryRunIncludeOpt constants in query_runs.go have the wrong type.
			// We use a string literal here as a workaround.
			Include: []QueryRunIncludeOpt{"created-by"},
		}

		qr, err := client.QueryRuns.ReadWithOptions(ctx, qrTest.ID, curOpts)
		require.NoError(t, err)

		require.NotEmpty(t, qr.CreatedBy)
		assert.NotEmpty(t, qr.CreatedBy.Username)
	})
}

func TestQueryRunsCancel(t *testing.T) {
	t.Skip("Cancel not yet implemented")

	skipUnlessBeta(t)
	client := testClient(t)
	ctx := context.Background()

	wTest, wTestCleanup := createWorkspace(t, client, nil)
	defer wTestCleanup()

	// We need to create 2 query runs here. The first run associated with the query
	// will automatically be planned so that one cannot be cancelled. The second one will
	// be pending until the first one is confirmed or discarded, so we
	// can cancel that one.
	_ = createQueryRun(t, client, wTest)
	qrTest2 := createQueryRun(t, client, wTest)

	t.Run("when the query run exists and is cancelable", func(t *testing.T) {
		// We assume the second query run is in a state that can be canceled.
		err := client.QueryRuns.Cancel(ctx, qrTest2.ID)
		require.NoError(t, err)
	})

	t.Run("when the query run does not exist", func(t *testing.T) {
		err := client.QueryRuns.Cancel(ctx, "nonexisting")
		assert.Equal(t, err, ErrResourceNotFound)
	})

	t.Run("with invalid query run ID", func(t *testing.T) {
		err := client.QueryRuns.Cancel(ctx, badIdentifier)
		assert.EqualError(t, err, ErrResourceNotFound.Error())
	})
}

func TestQueryRunsLogs(t *testing.T) {
	skipUnlessBeta(t)
	client := testClient(t)
	ctx := context.Background()

	wTest, wTestCleanup := createWorkspace(t, client, nil)
	defer wTestCleanup()

	qr, cleanup := createQueryRunWaitForAnyStatuses(t, client, wTest, []QueryRunStatus{QueryRunErrored, QueryRunFinished})
	t.Cleanup(cleanup)

	t.Run("when the query run exists", func(t *testing.T) {
		// We assume the second query run is in a state that can be canceled.
		reader, err := client.QueryRuns.Logs(ctx, qr.ID)
		require.NoError(t, err)

		logs, err := io.ReadAll(reader)
		require.NoError(t, err)
		assert.NotEmpty(t, logs, "some logs should be returned")
	})
}

func TestQueryRunsForceCancel(t *testing.T) {
	skipUnlessBeta(t)
	client := testClient(t)
	ctx := context.Background()

	wTest, wTestCleanup := createWorkspace(t, client, nil)
	defer wTestCleanup()

	_ = createQueryRun(t, client, wTest)
	qrTest2 := createQueryRun(t, client, wTest)

	// A force-cancel is not needed in any normal circumstance.
	// We can't easily get a query run into a state where it can be force-canceled.
	// So we'll just test the negative paths.

	t.Run("when the query run is not in a force-cancelable state", func(t *testing.T) {
		// This will likely return an error, but we are testing that the call can be made.
		// The API should return a 409 Conflict in this case.
		err := client.QueryRuns.ForceCancel(ctx, qrTest2.ID)
		assert.Error(t, err)
	})

	t.Run("when the query run does not exist", func(t *testing.T) {
		err := client.QueryRuns.ForceCancel(ctx, "nonexisting")
		assert.Equal(t, err, ErrResourceNotFound)
	})

	t.Run("with invalid query run ID", func(t *testing.T) {
		err := client.QueryRuns.ForceCancel(ctx, badIdentifier)
		assert.EqualError(t, err, ErrResourceNotFound.Error())
	})
}
