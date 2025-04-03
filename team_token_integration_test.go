// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package tfe

import (
	"context"
	"fmt"
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
		assert.Equal(t, err, ErrInvalidTeamID)
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
		assert.Equal(t, err, ErrInvalidTeamID)
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

func TestTeamTokens_CreateWithOptions_MultipleTokens(t *testing.T) {
	skipUnlessBeta(t)

	client := testClient(t)
	ctx := context.Background()

	tmTest, tmTestCleanup := createTeam(t, client, nil)
	t.Cleanup(tmTestCleanup)

	t.Run("with multiple tokens", func(t *testing.T) {
		desc1 := fmt.Sprintf("go-tfe-team-token-test-%s", randomString(t))
		tt, err := client.TeamTokens.CreateWithOptions(ctx, tmTest.ID, TeamTokenCreateOptions{
			Description: desc1,
		})
		require.NoError(t, err)
		require.NotEmpty(t, tt.Token)
		require.Equal(t, tt.Description, desc1)

		desc2 := fmt.Sprintf("go-tfe-team-token-test-%s", randomString(t))
		tt, err = client.TeamTokens.CreateWithOptions(ctx, tmTest.ID, TeamTokenCreateOptions{
			Description: desc2,
		})
		require.NoError(t, err)
		require.NotEmpty(t, tt.Token)
		require.Equal(t, tt.Description, desc2)
	})

	t.Run("with an expiration date", func(t *testing.T) {
		desc := fmt.Sprintf("go-tfe-team-token-test-%s", randomString(t))
		currentTime := time.Now().UTC().Truncate(time.Second)
		oneDayLater := currentTime.Add(24 * time.Hour)
		tt, err := client.TeamTokens.CreateWithOptions(ctx, tmTest.ID, TeamTokenCreateOptions{
			Description: desc,
			ExpiredAt:   &oneDayLater,
		})
		require.NoError(t, err)
		require.NotEmpty(t, tt.Token)
		assert.Equal(t, tt.ExpiredAt, oneDayLater)
		require.Equal(t, tt.Description, desc)
	})

	t.Run("without an expiration date", func(t *testing.T) {
		desc := fmt.Sprintf("go-tfe-team-token-test-%s", randomString(t))
		tt, err := client.TeamTokens.CreateWithOptions(ctx, tmTest.ID, TeamTokenCreateOptions{
			Description: desc,
		})
		require.NoError(t, err)
		require.NotEmpty(t, tt.Token)
		assert.Empty(t, tt.ExpiredAt)
		require.Equal(t, tt.Description, desc)
	})

	t.Run("when a token already exists with the same description", func(t *testing.T) {
		desc := fmt.Sprintf("go-tfe-team-token-test-%s", randomString(t))
		tt, err := client.TeamTokens.CreateWithOptions(ctx, tmTest.ID, TeamTokenCreateOptions{
			Description: desc,
		})
		require.NoError(t, err)
		require.NotEmpty(t, tt.Token)
		require.Equal(t, tt.Description, desc)

		tt, err = client.TeamTokens.CreateWithOptions(ctx, tmTest.ID, TeamTokenCreateOptions{
			Description: desc,
		})
		assert.Nil(t, tt)
		assert.Equal(t, err, ErrInvalidDescriptionConflict)
	})

	t.Run("without valid team ID", func(t *testing.T) {
		tt, err := client.TeamTokens.CreateWithOptions(ctx, badIdentifier, TeamTokenCreateOptions{})
		assert.Nil(t, tt)
		assert.Equal(t, err, ErrInvalidTeamID)
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
		assert.Equal(t, ErrResourceNotFound, err)
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
		assert.Equal(t, err, ErrResourceNotFound)
	})

	t.Run("without valid team ID", func(t *testing.T) {
		err := client.TeamTokens.Delete(ctx, badIdentifier)
		assert.Equal(t, err, ErrInvalidTeamID)
	})
}
