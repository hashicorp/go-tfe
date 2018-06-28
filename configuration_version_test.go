package tfe

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"io"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfigurationVersionsList(t *testing.T) {
	client := testClient(t)

	wTest, wTestCleanup := createWorkspace(t, client, nil)
	defer wTestCleanup()

	cvTest1, cvTest1Cleanup := createConfigurationVersion(t, client, wTest)
	defer cvTest1Cleanup()
	cvTest2, cvTest2Cleanup := createConfigurationVersion(t, client, wTest)
	defer cvTest2Cleanup()

	t.Run("without list options", func(t *testing.T) {
		cvs, err := client.ConfigurationVersions.List(wTest.ID, ConfigurationVersionListOptions{})
		require.NoError(t, err)

		// We need to strip the upload URL as that is a dynamic link.
		cvTest1.UploadURL = ""
		cvTest2.UploadURL = ""

		// And for the retrieved configuration versions as well.
		for _, cv := range cvs {
			cv.UploadURL = ""
		}

		assert.Contains(t, cvs, cvTest1)
		assert.Contains(t, cvs, cvTest2)
	})

	t.Run("with list options", func(t *testing.T) {
		// Request a page number which is out of range. The result should
		// be successful, but return no results if the paging options are
		// properly passed along.
		csv, err := client.ConfigurationVersions.List(wTest.ID, ConfigurationVersionListOptions{
			ListOptions: ListOptions{
				PageNumber: 999,
				PageSize:   100,
			},
		})
		require.NoError(t, err)
		assert.Empty(t, csv)
	})

	t.Run("without a valid organization", func(t *testing.T) {
		csv, err := client.ConfigurationVersions.List(badIdentifier, ConfigurationVersionListOptions{})
		assert.Nil(t, csv)
		assert.EqualError(t, err, "Invalid value for workspace ID")
	})
}

func TestConfigurationVersionsCreate(t *testing.T) {
	client := testClient(t)

	wTest, wTestCleanup := createWorkspace(t, client, nil)
	defer wTestCleanup()

	t.Run("with valid options", func(t *testing.T) {
		cv, err := client.ConfigurationVersions.Create(
			wTest.ID,
			ConfigurationVersionCreateOptions{},
		)
		require.NoError(t, err)

		// Get a refreshed view of the configuration version.
		refreshed, err := client.ConfigurationVersions.Retrieve(cv.ID)
		require.NoError(t, err)

		for _, item := range []*ConfigurationVersion{
			cv,
			refreshed,
		} {
			assert.NotEmpty(t, item.ID)
			assert.Empty(t, item.Error)
			assert.Equal(t, item.Source, ConfigurationSourceAPI)
			assert.Equal(t, item.Status, ConfigurationPending)
			assert.NotEmpty(t, item.UploadURL)
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
		require.NoError(t, err)

		// Don't compare the UploadURL because it will be generated twice in
		// this test - once at creation of the configuration version, and
		// again during the GET.
		cvTest.UploadURL, cv.UploadURL = "", ""

		assert.Equal(t, cvTest, cv)
	})

	t.Run("when the configuration version does not exist", func(t *testing.T) {
		cv, err := client.ConfigurationVersions.Retrieve("nonexisting")
		assert.Nil(t, cv)
		assert.EqualError(t, err, "Error: not found")
	})

	t.Run("with invalid configuration version id", func(t *testing.T) {
		cv, err := client.ConfigurationVersions.Retrieve(badIdentifier)
		assert.Nil(t, cv)
		assert.EqualError(t, err, "Invalid value for configuration version ID")
	})
}

func TestConfigurationVersionsUpload(t *testing.T) {
	client := testClient(t)

	cv, cvCleanup := createConfigurationVersion(t, client, nil)
	defer cvCleanup()

	t.Run("with valid options", func(t *testing.T) {
		err := client.ConfigurationVersions.Upload(
			cv.UploadURL,
			"test-fixtures/config-version",
		)
		require.NoError(t, err)

		// We do this is a small loop, because it can take a second
		// before the upload is finished.
		for i := 0; ; i++ {
			refreshed, err := client.ConfigurationVersions.Retrieve(cv.ID)
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
			cv.UploadURL[:len(cv.UploadURL)-10]+"nonexisting",
			"test-fixtures/config-version",
		)
		assert.Error(t, err)
	})

	t.Run("without a valid path", func(t *testing.T) {
		err := client.ConfigurationVersions.Upload(
			cv.UploadURL,
			"nonexisting",
		)
		assert.Error(t, err)
	})
}

func TestConfigurationVersionsPack(t *testing.T) {
	client := testClient(t)

	t.Run("with a valid path", func(t *testing.T) {
		raw, err := client.ConfigurationVersions.pack("test-fixtures/archive-dir")
		require.NoError(t, err)

		gzipR, err := gzip.NewReader(bytes.NewReader(raw))
		require.NoError(t, err)

		tarR := tar.NewReader(gzipR)
		var (
			symFound bool
			fileList []string
			slugSize int64
		)
		for {
			hdr, err := tarR.Next()
			if err == io.EOF {
				break
			}
			require.NoError(t, err)

			fileList = append(fileList, hdr.Name)
			if hdr.Typeflag == tar.TypeReg || hdr.Typeflag == tar.TypeRegA {
				slugSize += hdr.Size
			}

			if hdr.Name == "sub/foo.txt" {
				require.EqualValues(t, tar.TypeSymlink, hdr.Typeflag, "expect symlink for 'sub/foo.txt'")
				assert.Equal(t, "../foo.txt", hdr.Linkname, "expect target of '../foo.txt'")
				symFound = true
			}
		}

		t.Run("confirm we saw and handled a symlink", func(t *testing.T) {
			assert.True(t, symFound)
		})

		t.Run("check that the archive was created correctly", func(t *testing.T) {
			expectedFiles := []string{"bar.txt", "exe", "foo.txt", "sub/", "sub/foo.txt", "sub/zip.txt"}
			expectedSize := int64(12)

			assert.Equal(t, expectedFiles, fileList)
			assert.Equal(t, expectedSize, slugSize)
		})
	})

	t.Run("without a valid path", func(t *testing.T) {
		raw, err := client.ConfigurationVersions.pack("nonexisting")
		assert.Nil(t, raw)
		assert.Error(t, err)
	})
}
