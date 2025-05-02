// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package tfe

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const waitForPolicySetVersionUpload = 500 * time.Millisecond

func TestPolicySetVersionsCreate(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	psTest, psTestCleanup := createPolicySet(t, client, nil, nil, nil, nil, nil, "")
	defer psTestCleanup()

	t.Run("with valid identifier", func(t *testing.T) {
		psv, err := client.PolicySetVersions.Create(ctx, psTest.ID)
		require.NoError(t, err)

		assert.NotEmpty(t, psv.ID)
		assert.Equal(t, psv.Source, PolicySetVersionSourceAPI)
		assert.Equal(t, psv.PolicySet.ID, psTest.ID)
	})

	t.Run("with invalid identifier", func(t *testing.T) {
		_, err := client.PolicySetVersions.Create(ctx, badIdentifier)
		assert.Equal(t, err, ErrInvalidPolicySetID)
	})
}

func TestPolicySetVersionsRead(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	psTest, psTestCleanup := createPolicySet(t, client, nil, nil, nil, nil, nil, "")
	defer psTestCleanup()

	origPSV, err := client.PolicySetVersions.Create(ctx, psTest.ID)
	require.NoError(t, err)

	t.Run("with valid id", func(t *testing.T) {
		psv, err := client.PolicySetVersions.Read(ctx, origPSV.ID)
		require.NoError(t, err)

		assert.Equal(t, psv.Source, origPSV.Source)
		assert.Equal(t, psv.Status, origPSV.Status)
	})

	t.Run("with invalid id", func(t *testing.T) {
		_, err := client.PolicySetVersions.Read(ctx, randomString(t))
		require.Error(t, err)
	})
}

func TestPolicySetVersionsUpload(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	psv, psvCleanup := createPolicySetVersion(t, client, nil)
	defer psvCleanup()

	t.Run("with valid upload URL", func(t *testing.T) {
		psv, err := client.PolicySetVersions.Read(ctx, psv.ID)
		require.NoError(t, err)
		assert.Equal(t, psv.Status, PolicySetVersionPending)

		err = client.PolicySetVersions.Upload(
			ctx,
			*psv,
			"test-fixtures/policy-set-version",
		)
		require.NoError(t, err)

		// give HCP Terraform some time to process uploading the
		// policy set version before reading.
		time.Sleep(waitForPolicySetVersionUpload)

		psv, err = client.PolicySetVersions.Read(ctx, psv.ID)
		require.NoError(t, err)
		assert.Equal(t, PolicySetVersionReady, psv.Status)
	})

	t.Run("with missing upload URL", func(t *testing.T) {
		delete(psv.Links, "upload")

		err := client.PolicySetVersions.Upload(
			ctx,
			*psv,
			"test-fixtures/policy-set-version",
		)
		assert.EqualError(t, err, "the Policy Set Version does not contain an upload link")
	})
}

func TestPolicySetVersionsUploadURL(t *testing.T) {
	t.Run("successfully returns upload link", func(t *testing.T) {
		links := map[string]interface{}{
			"upload": "example.com",
		}
		psv := PolicySetVersion{
			Links: links,
		}

		uploadURL, err := psv.uploadURL()
		require.NoError(t, err)

		assert.Equal(t, uploadURL, "example.com")
	})

	t.Run("errors when there is no upload key in the Links", func(t *testing.T) {
		links := map[string]interface{}{
			"bad-field": "example.com",
		}
		psv := PolicySetVersion{
			Links: links,
		}

		_, err := psv.uploadURL()
		assert.EqualError(t, err, "the Policy Set Version does not contain an upload link")
	})

	t.Run("errors when the upload link is empty", func(t *testing.T) {
		links := map[string]interface{}{
			"upload": "",
		}
		psv := PolicySetVersion{
			Links: links,
		}

		_, err := psv.uploadURL()
		assert.EqualError(t, err, "the Policy Set Version upload URL is empty")
	})
}

func TestPolicySetVersionsIngressAttributes(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	t.Cleanup(orgTestCleanup)

	t.Run("with vcs", func(t *testing.T) {
		githubIdentifier := os.Getenv("GITHUB_POLICY_SET_IDENTIFIER")
		if githubIdentifier == "" {
			t.Skip("Export a valid GITHUB_POLICY_SET_IDENTIFIER before running this test")
		}

		otTest, otTestCleanup := createOAuthToken(t, client, orgTest)
		t.Cleanup(otTestCleanup)

		options := PolicySetCreateOptions{
			Name:         String("vcs-policy-set"),
			Kind:         Sentinel,
			PoliciesPath: String("policy-sets/foo"),
			VCSRepo: &VCSRepoOptions{
				Branch:            String("policies"),
				Identifier:        String(githubIdentifier),
				OAuthTokenID:      String(otTest.ID),
				IngressSubmodules: Bool(true),
			},
		}

		ps, err := client.PolicySets.Create(ctx, orgTest.Name, options)
		require.NoError(t, err)

		ps, err = client.PolicySets.ReadWithOptions(ctx, ps.ID, &PolicySetReadOptions{
			Include: []PolicySetIncludeOpt{
				PolicySetNewestVersion,
			},
		})
		require.NoError(t, err)

		psv, err := client.PolicySetVersions.Read(ctx, ps.NewestVersion.ID)
		require.NoError(t, err)

		require.NotNil(t, psv.IngressAttributes)
		assert.NotZero(t, psv.IngressAttributes.CommitSHA)
		assert.NotZero(t, psv.IngressAttributes.CommitURL)
		assert.NotZero(t, psv.IngressAttributes.Identifier)
	})

	t.Run("without vcs", func(t *testing.T) {
		psTest, psTestCleanup := createPolicySet(t, client, nil, nil, nil, nil, nil, "")
		t.Cleanup(psTestCleanup)

		psv, err := client.PolicySetVersions.Create(ctx, psTest.ID)
		require.NoError(t, err)

		assert.NotEmpty(t, psv.ID)
		assert.Equal(t, psv.Source, PolicySetVersionSourceAPI)
		assert.Equal(t, psv.PolicySet.ID, psTest.ID)

		assert.Nil(t, psv.IngressAttributes)
	})
}
