// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package tfe

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOrganizationTokensCreate(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	defer orgTestCleanup()

	var tkToken string
	t.Run("with valid options", func(t *testing.T) {
		ot, err := client.OrganizationTokens.Create(ctx, orgTest.Name)
		require.NoError(t, err)
		require.NotEmpty(t, ot.Token)
		requireExactlyOneNotEmpty(t, ot.CreatedBy.Organization, ot.CreatedBy.Team, ot.CreatedBy.User)
		tkToken = ot.Token
	})

	t.Run("when a token already exists", func(t *testing.T) {
		ot, err := client.OrganizationTokens.Create(ctx, orgTest.Name)
		require.NoError(t, err)
		require.NotEmpty(t, ot.Token)
		assert.NotEqual(t, tkToken, ot.Token)
	})

	t.Run("without valid organization", func(t *testing.T) {
		ot, err := client.OrganizationTokens.Create(ctx, badIdentifier)
		assert.Nil(t, ot)
		assert.EqualError(t, err, ErrInvalidOrg.Error())
	})
}

func TestOrganizationTokens_CreateWithOptions(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	defer orgTestCleanup()

	var tkToken string
	t.Run("with valid options", func(t *testing.T) {
		ot, err := client.OrganizationTokens.CreateWithOptions(ctx, orgTest.Name, OrganizationTokenCreateOptions{})
		require.NoError(t, err)
		require.NotEmpty(t, ot.Token)
		tkToken = ot.Token
	})

	t.Run("when a token already exists", func(t *testing.T) {
		ot, err := client.OrganizationTokens.CreateWithOptions(ctx, orgTest.Name, OrganizationTokenCreateOptions{})
		require.NoError(t, err)
		require.NotEmpty(t, ot.Token)
		assert.NotEqual(t, tkToken, ot.Token)
	})

	t.Run("without valid organization", func(t *testing.T) {
		ot, err := client.OrganizationTokens.CreateWithOptions(ctx, badIdentifier, OrganizationTokenCreateOptions{})
		assert.Nil(t, ot)
		assert.EqualError(t, err, ErrInvalidOrg.Error())
	})

	t.Run("without an expiration date", func(t *testing.T) {
		ot, err := client.OrganizationTokens.CreateWithOptions(ctx, orgTest.Name, OrganizationTokenCreateOptions{})
		require.NoError(t, err)
		require.NotEmpty(t, ot.Token)
		assert.Empty(t, ot.ExpiredAt)
		tkToken = ot.Token
	})

	t.Run("with an expiration date", func(t *testing.T) {
		currentTime := time.Now().UTC().Truncate(time.Second)
		oneDayLater := currentTime.Add(24 * time.Hour)
		ot, err := client.OrganizationTokens.CreateWithOptions(ctx, orgTest.Name, OrganizationTokenCreateOptions{
			ExpiredAt: &oneDayLater,
		})
		require.NoError(t, err)
		require.NotEmpty(t, ot.Token)
		assert.Equal(t, ot.ExpiredAt, oneDayLater)
		tkToken = ot.Token
	})
}

func TestOrganizationTokensRead(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	defer orgTestCleanup()

	t.Run("with valid options", func(t *testing.T) {
		_, otTestCleanup := createOrganizationToken(t, client, orgTest)

		ot, err := client.OrganizationTokens.Read(ctx, orgTest.Name)
		require.NoError(t, err)
		assert.NotEmpty(t, ot)

		otTestCleanup()
	})

	t.Run("with an expiration date passed as a valid option", func(t *testing.T) {
		currentTime := time.Now().UTC().Truncate(time.Second)
		oneDayLater := currentTime.Add(24 * time.Hour)

		_, otTestCleanup := createOrganizationTokenWithOptions(t, client, orgTest, OrganizationTokenCreateOptions{ExpiredAt: &oneDayLater})

		ot, err := client.OrganizationTokens.Read(ctx, orgTest.Name)
		require.NoError(t, err)
		assert.NotEmpty(t, ot)
		assert.Equal(t, ot.ExpiredAt, oneDayLater)

		otTestCleanup()
	})

	t.Run("when a token doesn't exists", func(t *testing.T) {
		ot, err := client.OrganizationTokens.Read(ctx, orgTest.Name)
		assert.Equal(t, ErrResourceNotFound, err)
		assert.Nil(t, ot)
	})

	t.Run("without valid organization", func(t *testing.T) {
		ot, err := client.OrganizationTokens.Read(ctx, badIdentifier)
		assert.Nil(t, ot)
		assert.EqualError(t, err, ErrInvalidOrg.Error())
	})
}

func TestOrganizationTokensDelete(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	defer orgTestCleanup()

	createOrganizationToken(t, client, orgTest)

	t.Run("with valid options", func(t *testing.T) {
		err := client.OrganizationTokens.Delete(ctx, orgTest.Name)
		require.NoError(t, err)
	})

	t.Run("when a token does not exist", func(t *testing.T) {
		err := client.OrganizationTokens.Delete(ctx, orgTest.Name)
		assert.Equal(t, err, ErrResourceNotFound)
	})

	t.Run("without valid organization", func(t *testing.T) {
		err := client.OrganizationTokens.Delete(ctx, badIdentifier)
		assert.EqualError(t, err, ErrInvalidOrg.Error())
	})
}
