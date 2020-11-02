package tfe

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPolicySetVersionsCreate(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	psTest, psTestCleanup := createPolicySet(t, client, nil, nil, nil)
	defer psTestCleanup()

	t.Run("with valid options", func(t *testing.T) {
		psv, err := client.PolicySetVersions.Create(ctx,
			psTest.ID,
			PolicySetVersionCreateOptions{},
		)
		require.NoError(t, err)

		// Get a refreshed view of the policy set version.
		refreshed, err := client.PolicySetVersions.Read(ctx, psv.ID)
		require.NoError(t, err)

		for _, item := range []*PolicySetVersion{
			psv,
			refreshed,
		} {
			assert.NotEmpty(t, item.ID)
			assert.Empty(t, item.Error)
			assert.Equal(t, item.Source, PolicySetVersionSourceAPI)
			assert.Equal(t, item.Status, PolicySetVersionPending)

			uploadLink := ""
			for k, v := range *(item.Links) {
				if k == "upload" {
					uploadLink = v.(string)
				}
			}
			assert.NotEmpty(t, uploadLink)
		}
	})

	t.Run("when policy set ID is invalid", func(t *testing.T) {
		options := PolicySetVersionCreateOptions{}

		_, err := client.PolicySetVersions.Create(ctx, badIdentifier, options)
		assert.EqualError(t, err, "invalid value for policy set ID")
	})
}

func TestPolicySetVersionsRead(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	psvTest, psvTestCleanup := createPolicySetVersion(t, client, nil)
	defer psvTestCleanup()

	t.Run("when the policy set version exists", func(t *testing.T) {
		psv, err := client.PolicySetVersions.Read(ctx, psvTest.ID)
		require.NoError(t, err)

		assert.Equal(t, psvTest, psv)
	})

	t.Run("when the policy set version does not exist", func(t *testing.T) {
		psv, err := client.PolicySetVersions.Read(ctx, "nonexisting")
		assert.Nil(t, psv)
		assert.Equal(t, err, ErrResourceNotFound)
	})

	t.Run("with invalid policy set version id", func(t *testing.T) {
		psv, err := client.PolicySetVersions.Read(ctx, badIdentifier)
		assert.Nil(t, psv)
		assert.EqualError(t, err, "invalid value for policy set version ID")
	})
}

func TestPolicySetVersionsUpload(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	psv, psvCleanup := createPolicySetVersion(t, client, nil)
	defer psvCleanup()

	t.Run("with valid options", func(t *testing.T) {

		uploadLink := ""
		for k, v := range *(psv.Links) {
			if k == "upload" {
				uploadLink = v.(string)
			}
		}

		err := client.PolicySetVersions.Upload(
			ctx,
			uploadLink,
			"test-fixtures/policy-set-version",
		)
		require.NoError(t, err)

		// We do this in a small loop, because it can take a second
		// before the upload is finished.
		for i := 0; ; i++ {
			refreshed, err := client.PolicySetVersions.Read(ctx, psv.ID)
			require.NoError(t, err)

			if refreshed.Status == PolicySetVersionUploaded {
				break
			}

			if i > 10 {
				t.Fatal("Timeout waiting for the policy set version to be uploaded")
			}

			time.Sleep(1 * time.Second)
		}
	})

	t.Run("without a valid upload URL", func(t *testing.T) {

		uploadLink := ""
		for k, v := range *(psv.Links) {
			if k == "upload" {
				uploadLink = v.(string)
			}
		}

		err := client.PolicySetVersions.Upload(
			ctx,
			uploadLink[:len(uploadLink)-10]+"nonexisting",
			"test-fixtures/policy-set-version",
		)
		assert.Error(t, err)
	})

	t.Run("without a valid path", func(t *testing.T) {

		uploadLink := ""
		for k, v := range *(psv.Links) {
			if k == "upload" {
				uploadLink = v.(string)
			}
		}

		err := client.PolicySetVersions.Upload(
			ctx,
			uploadLink,
			"nonexisting",
		)
		assert.Error(t, err)
	})
}
