//go:build integration
// +build integration

package tfe

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/hashicorp/go-slug"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfigurationVersionsList(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	wTest, wTestCleanup := createWorkspace(t, client, nil)
	defer wTestCleanup()

	cvTest1, cvTest1Cleanup := createConfigurationVersion(t, client, wTest)
	defer cvTest1Cleanup()
	cvTest2, cvTest2Cleanup := createConfigurationVersion(t, client, wTest)
	defer cvTest2Cleanup()

	t.Run("without list options", func(t *testing.T) {
		cvl, err := client.ConfigurationVersions.List(ctx, wTest.ID, nil)
		require.NoError(t, err)

		// We need to strip the upload URL as that is a dynamic link.
		cvTest1.UploadURL = ""
		cvTest2.UploadURL = ""

		// And for the retrieved configuration versions as well.
		for _, cv := range cvl.Items {
			cv.UploadURL = ""
		}

		assert.Contains(t, cvl.Items, cvTest1)
		assert.Contains(t, cvl.Items, cvTest2)
		assert.Equal(t, 1, cvl.CurrentPage)
		assert.Equal(t, 2, cvl.TotalCount)
	})

	t.Run("with list options", func(t *testing.T) {
		// Request a page number which is out of range. The result should
		// be successful, but return no results if the paging options are
		// properly passed along.
		options := &ConfigurationVersionListOptions{
			ListOptions: ListOptions{
				PageNumber: 999,
				PageSize:   100,
			},
		}

		cvl, err := client.ConfigurationVersions.List(ctx, wTest.ID, options)
		require.NoError(t, err)
		assert.Empty(t, cvl.Items)
		assert.Equal(t, 999, cvl.CurrentPage)
		assert.Equal(t, 2, cvl.TotalCount)
	})

	t.Run("without a valid organization", func(t *testing.T) {
		cvl, err := client.ConfigurationVersions.List(ctx, badIdentifier, nil)
		assert.Nil(t, cvl)
		assert.EqualError(t, err, ErrInvalidWorkspaceID.Error())
	})
}

func TestConfigurationVersionsCreate(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	wTest, wTestCleanup := createWorkspace(t, client, nil)
	defer wTestCleanup()

	t.Run("with valid options", func(t *testing.T) {
		cv, err := client.ConfigurationVersions.Create(ctx,
			wTest.ID,
			ConfigurationVersionCreateOptions{},
		)
		assert.NotEmpty(t, cv.UploadURL)
		require.NoError(t, err)

		// Get a refreshed view of the configuration version.
		refreshed, err := client.ConfigurationVersions.Read(ctx, cv.ID)
		require.NoError(t, err)
		assert.Empty(t, refreshed.UploadURL)

		for _, item := range []*ConfigurationVersion{
			cv,
			refreshed,
		} {
			assert.NotEmpty(t, item.ID)
			assert.Empty(t, item.Error)
			assert.Equal(t, item.Source, ConfigurationSourceAPI)
			assert.Equal(t, item.Status, ConfigurationPending)
		}
	})

	t.Run("with invalid workspace id", func(t *testing.T) {
		cv, err := client.ConfigurationVersions.Create(
			ctx,
			badIdentifier,
			ConfigurationVersionCreateOptions{},
		)
		assert.Nil(t, cv)
		assert.EqualError(t, err, ErrInvalidWorkspaceID.Error())
	})
}

func TestConfigurationVersionsRead(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	cvTest, cvTestCleanup := createConfigurationVersion(t, client, nil)
	defer cvTestCleanup()

	t.Run("when the configuration version exists", func(t *testing.T) {
		cv, err := client.ConfigurationVersions.Read(ctx, cvTest.ID)
		require.NoError(t, err)

		// Don't compare the UploadURL because it will be generated twice in
		// this test - once at creation of the configuration version, and
		// again during the GET.
		cvTest.UploadURL, cv.UploadURL = "", ""

		assert.Equal(t, cvTest, cv)
	})

	t.Run("when the configuration version does not exist", func(t *testing.T) {
		cv, err := client.ConfigurationVersions.Read(ctx, "nonexisting")
		assert.Nil(t, cv)
		assert.Equal(t, err, ErrResourceNotFound)
	})

	t.Run("with invalid configuration version id", func(t *testing.T) {
		cv, err := client.ConfigurationVersions.Read(ctx, badIdentifier)
		assert.Nil(t, cv)
		assert.EqualError(t, err, ErrInvalidConfigVersionID.Error())
	})
}

func TestConfigurationVersionsReadWithOptions(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	defer orgTestCleanup()

	wTest, wTestCleanup := createWorkspaceWithVCS(t, client, orgTest, WorkspaceCreateOptions{QueueAllRuns: Bool(true)})
	defer wTestCleanup()

	// Hack: Wait for TFC to ingress the configuration and queue a run
	time.Sleep(3 * time.Second)

	w, err := client.Workspaces.ReadByIDWithOptions(ctx, wTest.ID, &WorkspaceReadOptions{
		Include: []WSIncludeOpt{WSCurrentRunConfigVer},
	})

	if err != nil {
		require.NoError(t, err)
	}

	if w.CurrentRun == nil {
		t.Fatal("A run was expected to be found on this workspace as a test pre-condition")
	}

	cv := w.CurrentRun.ConfigurationVersion

	t.Run("when the configuration version exists", func(t *testing.T) {
		options := &ConfigurationVersionReadOptions{
			Include: []ConfigVerIncludeOpt{ConfigVerIngressAttributes},
		}

		cv, err := client.ConfigurationVersions.ReadWithOptions(ctx, cv.ID, options)
		require.NoError(t, err)

		assert.NotZero(t, cv.IngressAttributes)
		assert.NotZero(t, cv.IngressAttributes.CommitURL)
		assert.NotZero(t, cv.IngressAttributes.CommitSHA)
	})
}

func TestConfigurationVersionsUpload(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	cv, cvCleanup := createConfigurationVersion(t, client, nil)
	defer cvCleanup()

	t.Run("with valid options", func(t *testing.T) {
		err := client.ConfigurationVersions.Upload(
			ctx,
			cv.UploadURL,
			"test-fixtures/config-version",
		)
		require.NoError(t, err)

		// We do this is a small loop, because it can take a second
		// before the upload is finished.
		for i := 0; ; i++ {
			refreshed, err := client.ConfigurationVersions.Read(ctx, cv.ID)
			require.NoError(t, err)

			if refreshed.Status == ConfigurationUploaded {
				break
			}

			if i > 10 {
				t.Fatal("Timeout waiting for the configuration version to be uploaded")
			}

			time.Sleep(1 * time.Second)
		}
	})

	t.Run("without a valid upload URL", func(t *testing.T) {
		err := client.ConfigurationVersions.Upload(
			ctx,
			cv.UploadURL[:len(cv.UploadURL)-10]+"nonexisting",
			"test-fixtures/config-version",
		)
		assert.Error(t, err)
	})

	t.Run("without a valid path", func(t *testing.T) {
		err := client.ConfigurationVersions.Upload(
			ctx,
			cv.UploadURL,
			"nonexisting",
		)
		assert.Error(t, err)
	})
}

func TestConfigurationVersionsArchive(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	w, wCleanup := createWorkspace(t, client, nil)
	defer wCleanup()

	cv, cvCleanup := createConfigurationVersion(t, client, w)
	defer cvCleanup()

	t.Run("when the configuration version exists and has been uploaded", func(t *testing.T) {
		err := client.ConfigurationVersions.Upload(
			ctx,
			cv.UploadURL,
			"test-fixtures/config-version",
		)
		require.NoError(t, err)

		// We do this is a small loop, because it can take a second
		// before the upload is finished.
		for i := 0; ; i++ {
			refreshed, err := client.ConfigurationVersions.Read(ctx, cv.ID)
			require.NoError(t, err)

			if refreshed.Status == ConfigurationUploaded {
				break
			}

			if i > 10 {
				t.Fatal("Timeout waiting for the configuration version to be uploaded")
			}

			time.Sleep(1 * time.Second)
		}

		// configuration version should not be archived, since it's the latest version
		err = client.ConfigurationVersions.Archive(ctx, cv.ID)
		assert.Error(t, err)
		assert.EqualError(t, err, "transition not allowed")

		// create subsequent version, since the latest configuration version cannot be archived
		newCv, newCvCleanup := createConfigurationVersion(t, client, w)
		err = client.ConfigurationVersions.Upload(
			ctx,
			newCv.UploadURL,
			"test-fixtures/config-version",
		)
		require.NoError(t, err)
		defer newCvCleanup()

		err = client.ConfigurationVersions.Archive(ctx, cv.ID)
		require.NoError(t, err)

		// We do this is a small loop, because it can take a second
		// before the archive is finished.
		for i := 0; ; i++ {
			refreshed, err := client.ConfigurationVersions.Read(ctx, cv.ID)
			require.NoError(t, err)

			if refreshed.Status == ConfigurationArchived {
				break
			}

			if i > 10 {
				t.Fatal("Timeout waiting for the configuration version to be archived")
			}

			time.Sleep(1 * time.Second)
		}
	})

	t.Run("when the configuration version does not exist", func(t *testing.T) {
		err := client.ConfigurationVersions.Archive(ctx, "nonexisting")
		assert.Equal(t, err, ErrResourceNotFound)
	})

	t.Run("with invalid configuration version id", func(t *testing.T) {
		err := client.ConfigurationVersions.Archive(ctx, badIdentifier)
		assert.EqualError(t, err, ErrInvalidConfigVersionID.Error())
	})
}

func TestConfigurationVersionsDownload(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	t.Run("with a valid ID for downloadable configuration version", func(t *testing.T) {
		uploadedCv, uploadedCvCleanup := createUploadedConfigurationVersion(t, client, nil)
		defer uploadedCvCleanup()

		expectedCvFile := bytes.NewBuffer(nil)
		_, expectedCvFileErr := slug.Pack("test-fixtures/config-version", expectedCvFile, true)
		if expectedCvFileErr != nil {
			t.Fatal(expectedCvFileErr)
		}

		cvFile, err := client.ConfigurationVersions.Download(ctx, uploadedCv.ID)

		assert.NotNil(t, cvFile)
		assert.NoError(t, err)
		assert.True(t, bytes.Equal(cvFile, expectedCvFile.Bytes()), "Configuration version should match")
	})

	t.Run("with a valid ID for a non downloadable configuration version", func(t *testing.T) {
		pendingCv, pendingCvCleanup := createConfigurationVersion(t, client, nil)
		defer pendingCvCleanup()

		cvFile, err := client.ConfigurationVersions.Download(ctx, pendingCv.ID)

		assert.Nil(t, cvFile)
		assert.EqualError(t, err, ErrResourceNotFound.Error())
	})

	t.Run("with an invalid ID", func(t *testing.T) {
		cvFile, err := client.ConfigurationVersions.Download(ctx, "nonexistent")
		assert.Nil(t, cvFile)
		assert.EqualError(t, err, ErrResourceNotFound.Error())
	})
}

func TestConfigurationVersions_Unmarshal(t *testing.T) {
	data := map[string]interface{}{
		"data": map[string]interface{}{
			"type": "configuration-versions",
			"id":   "cv-ntv3HbhJqvFzamy7",
			"attributes": map[string]interface{}{
				"auto-queue-runs": true,
				"error":           "bad error",
				"error-message":   "message",
				"source":          ConfigurationSourceTerraform,
				"status":          ConfigurationUploaded,
				"status-timestamps": map[string]string{
					"finished-at": "2020-03-16T23:15:59+00:00",
					"started-at":  "2019-03-16T23:23:59+00:00",
				},
			},
		},
	}
	byteData, err := json.Marshal(data)
	require.NoError(t, err)

	responseBody := bytes.NewReader(byteData)
	cv := &ConfigurationVersion{}
	err = unmarshalResponse(responseBody, cv)
	require.NoError(t, err)

	finishedParsedTime, err := time.Parse(time.RFC3339, "2020-03-16T23:15:59+00:00")
	require.NoError(t, err)
	startedParsedTime, err := time.Parse(time.RFC3339, "2019-03-16T23:23:59+00:00")
	require.NoError(t, err)

	assert.Equal(t, cv.ID, "cv-ntv3HbhJqvFzamy7")
	assert.Equal(t, cv.AutoQueueRuns, true)
	assert.Equal(t, cv.Error, "bad error")
	assert.Equal(t, cv.ErrorMessage, "message")
	assert.Equal(t, cv.Source, ConfigurationSourceTerraform)
	assert.Equal(t, cv.Status, ConfigurationUploaded)
	assert.Equal(t, cv.StatusTimestamps.FinishedAt, finishedParsedTime)
	assert.Equal(t, cv.StatusTimestamps.StartedAt, startedParsedTime)
}
