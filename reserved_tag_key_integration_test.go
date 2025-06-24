// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package tfe

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/stretchr/testify/require"
)

func TestReservedTagKeysList(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	defer orgTestCleanup()

	rtkTest1, rtkTestCleanup := createReservedTagKey(t, client, orgTest,
		ReservedTagKeyCreateOptions{
			Key: randomString(t),
		})

	defer rtkTestCleanup()

	rtkTest2, rtkTestCleanup := createReservedTagKey(t, client, orgTest,
		ReservedTagKeyCreateOptions{
			Key: randomString(t),
		})
	defer rtkTestCleanup()

	rtks, err := client.ReservedTagKeys.List(ctx, orgTest.Name, nil)
	require.NoError(t, err)
	require.Len(t, rtks.Items, 2)

	t.Run("without list options", func(t *testing.T) {
		pl, err := client.ReservedTagKeys.List(ctx, orgTest.Name, nil)
		require.NoError(t, err)
		assert.Contains(t, pl.Items, rtkTest1)

		assert.Equal(t, 1, pl.CurrentPage)
		assert.Equal(t, 2, pl.TotalCount)
	})

	t.Run("with pagination list options", func(t *testing.T) {
		rtks, err := client.ReservedTagKeys.List(ctx, orgTest.Name, &ReservedTagKeyListOptions{
			ListOptions: ListOptions{
				PageNumber: 1,
				PageSize:   100,
			},
		})
		require.NoError(t, err)
		assert.Contains(t, rtks.Items, rtkTest1)
		assert.Contains(t, rtks.Items, rtkTest2)
		assert.Equal(t, 2, len(rtks.Items))
	})

	t.Run("without a valid organization", func(t *testing.T) {
		pl, err := client.ReservedTagKeys.List(ctx, badIdentifier, nil)
		assert.Nil(t, pl)
		assert.EqualError(t, err, ErrInvalidOrg.Error())
	})
}

func TestReservedTagKeysCreate(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	defer orgTestCleanup()

	newSubscriptionUpdater(orgTest).WithBusinessPlan().Update(t)

	t.Run("with valid options", func(t *testing.T) {
		options := ReservedTagKeyCreateOptions{
			Key:              randomString(t),
			DisableOverrides: Bool(true),
		}

		rtk, err := client.ReservedTagKeys.Create(ctx, orgTest.Name, options)
		require.NoError(t, err)

		rtks, err := client.ReservedTagKeys.List(ctx, orgTest.Name, nil)
		require.NoError(t, err)
		require.Len(t, rtks.Items, 1)

		assert.NotEmpty(t, rtk.ID)
		assert.Equal(t, options.Key, rtk.Key)
		assert.Equal(t, *options.DisableOverrides, rtk.DisableOverrides)
	})

	t.Run("when key has already been taken", func(t *testing.T) {
		rtkExisting, rtkTestCleanup := createReservedTagKey(t, client, orgTest, ReservedTagKeyCreateOptions{
			Key:              randomString(t),
			DisableOverrides: Bool(true),
		})
		t.Cleanup(rtkTestCleanup)

		rtks, err := client.ReservedTagKeys.List(ctx, orgTest.Name, nil)
		assert.NoError(t, err)
		assert.Len(t, rtks.Items, 2)

		rtk, err := client.ReservedTagKeys.Create(ctx, orgTest.Name, ReservedTagKeyCreateOptions{
			Key: rtkExisting.Key,
		})
		assert.Nil(t, rtk)
		assert.Contains(t, err.Error(), "invalid attribute\n\nKey has already been taken")
	})

	t.Run("when options is missing key", func(t *testing.T) {
		w, err := client.ReservedTagKeys.Create(ctx, orgTest.Name, ReservedTagKeyCreateOptions{
			DisableOverrides: Bool(true),
		})
		assert.Nil(t, w)
		assert.EqualError(t, err, "invalid attribute\n\nKey can't be blank")
	})

	t.Run("when options has an invalid key", func(t *testing.T) {
		rtk, err := client.ReservedTagKeys.Create(ctx, orgTest.Name, ReservedTagKeyCreateOptions{
			Key: badIdentifier,
		})
		assert.Nil(t, rtk)
		assert.Contains(t, err.Error(), "invalid attribute\n\nKey is invalid")
	})

	t.Run("when options has an invalid organization", func(t *testing.T) {
		rtk, err := client.ReservedTagKeys.Create(ctx, badIdentifier, ReservedTagKeyCreateOptions{
			Key: randomString(t),
		})
		assert.Nil(t, rtk)
		assert.EqualError(t, err, ErrInvalidOrg.Error())
	})

	t.Run("when organization does not exist", func(t *testing.T) {
		rtk, err := client.ReservedTagKeys.Create(ctx, "nonexistent", ReservedTagKeyCreateOptions{
			Key: randomString(t),
		})
		assert.Nil(t, rtk)
		assert.EqualError(t, err, ErrResourceNotFound.Error())
	})
}

func TestReservedTagKeysUpdate(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	defer orgTestCleanup()

	rtkExisting, rtkTestCleanup := createReservedTagKey(t, client, orgTest, ReservedTagKeyCreateOptions{
		Key:              randomString(t),
		DisableOverrides: Bool(true),
	})
	t.Cleanup(rtkTestCleanup)

	rtks, err := client.ReservedTagKeys.List(ctx, orgTest.Name, nil)
	require.NoError(t, err)
	require.Len(t, rtks.Items, 1)

	t.Run("with valid options", func(t *testing.T) {
		rtkAfter, err := client.ReservedTagKeys.Update(ctx, rtkExisting.ID, ReservedTagKeyUpdateOptions{
			Key:              String(randomString(t)),
			DisableOverrides: Bool(false),
		})
		require.NoError(t, err)

		assert.Equal(t, rtkExisting.ID, rtkAfter.ID)
		assert.NotEqual(t, rtkExisting.Key, rtkAfter.Key)
		assert.NotEqual(t, rtkExisting.DisableOverrides, rtkAfter.DisableOverrides)
	})

	t.Run("when updating with invalid key", func(t *testing.T) {
		rtkAfter, err := client.ReservedTagKeys.Update(ctx, rtkExisting.ID, ReservedTagKeyUpdateOptions{
			Key: String(badIdentifier),
		})

		assert.Error(t, err)
		assert.Nil(t, rtkAfter)
		assert.Contains(t, err.Error(), "invalid attribute\n\nKey is invalid")
	})

	t.Run("when key has already been taken", func(t *testing.T) {
		rtkOther, rtkTestCleanup := createReservedTagKey(t, client, orgTest, ReservedTagKeyCreateOptions{
			Key:              randomString(t),
			DisableOverrides: Bool(true),
		})
		t.Cleanup(rtkTestCleanup)

		rtks, err := client.ReservedTagKeys.List(ctx, orgTest.Name, nil)
		assert.NoError(t, err)
		assert.Len(t, rtks.Items, 2)

		rtkAfter, err := client.ReservedTagKeys.Update(ctx, rtkExisting.ID, ReservedTagKeyUpdateOptions{
			Key: String(rtkOther.Key),
		})
		require.Error(t, err)
		assert.Nil(t, rtkAfter)
		assert.Contains(t, err.Error(), "invalid attribute\n\nKey has already been taken")
	})

	t.Run("without a valid reserved tag key ID", func(t *testing.T) {
		rtkAfter, err := client.ReservedTagKeys.Update(ctx, badIdentifier, ReservedTagKeyUpdateOptions{
			Key: String(randomString(t)),
		})
		assert.Nil(t, rtkAfter)
		assert.EqualError(t, err, ErrInvalidReservedTagKeyID.Error())
	})

	t.Run("when the reserved tag key does not exist", func(t *testing.T) {
		rtkAfter, err := client.ReservedTagKeys.Update(ctx, "nonexistent", ReservedTagKeyUpdateOptions{
			Key: String(randomString(t)),
		})
		assert.Nil(t, rtkAfter)
		assert.EqualError(t, err, ErrResourceNotFound.Error())
	})
}

func TestReservedTagKeysDelete(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	defer orgTestCleanup()

	rtkBefore, rtkTestCleanup := createReservedTagKey(t, client, orgTest, ReservedTagKeyCreateOptions{
		Key:              randomString(t),
		DisableOverrides: Bool(true),
	})
	t.Cleanup(rtkTestCleanup)

	rtks, err := client.ReservedTagKeys.List(ctx, orgTest.Name, nil)
	assert.NoError(t, err)
	assert.Len(t, rtks.Items, 1)

	t.Run("when the request is valid", func(t *testing.T) {
		err := client.ReservedTagKeys.Delete(ctx, rtkBefore.ID)
		require.NoError(t, err)

		rtks, err = client.ReservedTagKeys.List(ctx, orgTest.Name, nil)
		assert.NoError(t, err)
		assert.Len(t, rtks.Items, 0)
	})

	t.Run("when the reserved tag key does not exist", func(t *testing.T) {
		err := client.ReservedTagKeys.Delete(ctx, "nonexistent")
		assert.Equal(t, err, ErrResourceNotFound)
	})

	t.Run("when the reserved tag key ID is invalid", func(t *testing.T) {
		err := client.ReservedTagKeys.Delete(ctx, badIdentifier)
		assert.EqualError(t, err, ErrInvalidReservedTagKeyID.Error())
	})
}

func createReservedTagKey(t *testing.T, client *Client, org *Organization, opts ReservedTagKeyCreateOptions) (*ReservedTagKey, func()) {
	t.Helper()

	rtk, err := client.ReservedTagKeys.Create(context.Background(), org.Name, opts)
	require.NoError(t, err)

	cleanup := func() {
		err := client.ReservedTagKeys.Delete(context.Background(), rtk.ID)
		if err != nil && err == ErrResourceNotFound {
			// It's already been deleted
			return
		}

		require.NoError(t, err)
	}

	return rtk, cleanup
}
