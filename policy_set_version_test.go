package tfe

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPolicySetVersionsCreate(t *testing.T) {
	skipIfFreeOnly(t)

	client := testClient(t)
	ctx := context.Background()

	psTest, psTestCleanup := createPolicySet(t, client, nil, nil, nil)
	defer psTestCleanup()

	t.Run("with valid identifier", func(t *testing.T) {
		psv, err := client.PolicySetVersions.Create(ctx, psTest.ID)
		require.NoError(t, err)

		assert.NotEmpty(t, psv.ID)
		assert.Equal(t, psv.Source, PolciySetVersionSourceAPI)
		assert.Equal(t, psv.PolicySet.ID, psTest.ID)
	})

	t.Run("with invalid identifier", func(t *testing.T) {
		_, err := client.PolicySetVersions.Create(ctx, badIdentifier)
		assert.EqualError(t, err, "invalid value for policy set ID")
	})
}

func TestPolicySetVersionsRead(t *testing.T) {
	skipIfFreeOnly(t)

	client := testClient(t)
	ctx := context.Background()

	psTest, psTestCleanup := createPolicySet(t, client, nil, nil, nil)
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
	skipIfFreeOnly(t)

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

		psv, err = client.PolicySetVersions.Read(ctx, psv.ID)
		require.NoError(t, err)
		assert.Equal(t, psv.Status, PolicySetVersionReady)
	})

	t.Run("with missing upload URL", func(t *testing.T) {
		delete(psv.Links, "upload")

		err := client.PolicySetVersions.Upload(
			ctx,
			*psv,
			"test-fixtures/policy-set-version",
		)
		assert.EqualError(t, err, "The Policy Set Version does not contain an upload link.")
	})
}
