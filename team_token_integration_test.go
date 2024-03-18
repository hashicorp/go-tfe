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

func TestTeamTokensCreate(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	tmTest, tmTestCleanup := createTeam(t, client, nil)
	defer tmTestCleanup()

	var tmToken string
	t.Run("with valid options", func(t *testing.T) {
		tt, err := client.TeamTokens.Create(ctx, tmTest.ID)
		require.NoError(t, err)
		require.NotEmpty(t, tt.Token)
		require.NotEmpty(t, tt.CreatedBy)
		requireExactlyOneNotEmpty(t, tt.CreatedBy.Organization, tt.CreatedBy.Team, tt.CreatedBy.User)
		tmToken = tt.Token
	})

	t.Run("when a token already exists", func(t *testing.T) {
		tt, err := client.TeamTokens.Create(ctx, tmTest.ID)
		require.NoError(t, err)
		require.NotEmpty(t, tt.Token)
		assert.NotEqual(t, tmToken, tt.Token)
	})

	t.Run("without valid team ID", func(t *testing.T) {
		tt, err := client.TeamTokens.Create(ctx, badIdentifier)
		assert.Nil(t, tt)
		assert.ErrorIs(t, err, ErrInvalidTeamID)
	})
}

func TestTeamTokens_CreateWithOptions(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	tmTest, tmTestCleanup := createTeam(t, client, nil)
	defer tmTestCleanup()

	var tmToken string
	t.Run("with valid options", func(t *testing.T) {
		tt, err := client.TeamTokens.CreateWithOptions(ctx, tmTest.ID, TeamTokenCreateOptions{})
		require.NoError(t, err)
		require.NotEmpty(t, tt.Token)
		tmToken = tt.Token
	})

	t.Run("when a token already exists", func(t *testing.T) {
		tt, err := client.TeamTokens.CreateWithOptions(ctx, tmTest.ID, TeamTokenCreateOptions{})
		require.NoError(t, err)
		require.NotEmpty(t, tt.Token)
		assert.NotEqual(t, tmToken, tt.Token)
	})

	t.Run("without valid team ID", func(t *testing.T) {
		tt, err := client.TeamTokens.CreateWithOptions(ctx, badIdentifier, TeamTokenCreateOptions{})
		assert.Nil(t, tt)
		assert.ErrorIs(t, err, ErrInvalidTeamID)
	})

	t.Run("without an expiration date", func(t *testing.T) {
		tt, err := client.TeamTokens.CreateWithOptions(ctx, tmTest.ID, TeamTokenCreateOptions{})
		require.NoError(t, err)
		require.NotEmpty(t, tt.Token)
		assert.Empty(t, tt.ExpiredAt)
		tmToken = tt.Token
	})

	t.Run("with an expiration date", func(t *testing.T) {
		currentTime := time.Now().UTC().Truncate(time.Second)
		oneDayLater := currentTime.Add(24 * time.Hour)
		tt, err := client.TeamTokens.CreateWithOptions(ctx, tmTest.ID, TeamTokenCreateOptions{
			ExpiredAt: &oneDayLater,
		})
		require.NoError(t, err)
		require.NotEmpty(t, tt.Token)
		assert.Equal(t, tt.ExpiredAt, oneDayLater)
		tmToken = tt.Token
	})
}

func TestTeamTokensRead(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	tmTest, tmTestCleanup := createTeam(t, client, nil)
	defer tmTestCleanup()

	t.Run("with valid options", func(t *testing.T) {
		_, ttTestCleanup := createTeamToken(t, client, tmTest)

		tt, err := client.TeamTokens.Read(ctx, tmTest.ID)
		require.NoError(t, err)
		assert.NotEmpty(t, tt)

		ttTestCleanup()
	})

	t.Run("with an expiration date passed as a valid option", func(t *testing.T) {
		currentTime := time.Now().UTC().Truncate(time.Second)
		oneDayLater := currentTime.Add(24 * time.Hour)

		_, ttTestCleanup := createTeamTokenWithOptions(t, client, tmTest, TeamTokenCreateOptions{ExpiredAt: &oneDayLater})

		tt, err := client.TeamTokens.Read(ctx, tmTest.ID)
		require.NoError(t, err)
		assert.NotEmpty(t, tt)
		assert.Equal(t, tt.ExpiredAt, oneDayLater)

		ttTestCleanup()
	})

	t.Run("when a token doesn't exists", func(t *testing.T) {
		tt, err := client.TeamTokens.Read(ctx, tmTest.ID)
		assert.ErrorIs(t, ErrResourceNotFound, err)
		assert.Nil(t, tt)
	})

	t.Run("without valid organization", func(t *testing.T) {
		tt, err := client.OrganizationTokens.Read(ctx, badIdentifier)
		assert.Nil(t, tt)
		assert.EqualError(t, err, ErrInvalidOrg.Error())
	})
}

func TestTeamTokensDelete(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	tmTest, tmTestCleanup := createTeam(t, client, nil)
	defer tmTestCleanup()

	createTeamToken(t, client, tmTest)

	t.Run("with valid options", func(t *testing.T) {
		err := client.TeamTokens.Delete(ctx, tmTest.ID)
		require.NoError(t, err)
	})

	t.Run("when a token does not exist", func(t *testing.T) {
		err := client.TeamTokens.Delete(ctx, tmTest.ID)
		assert.ErrorIs(t, err, ErrResourceNotFound)
	})

	t.Run("without valid team ID", func(t *testing.T) {
		err := client.TeamTokens.Delete(ctx, badIdentifier)
		assert.ErrorIs(t, err, ErrInvalidTeamID)
	})
}
