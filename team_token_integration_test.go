// Copyright IBM Corp. 2018, 2025
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
			Description: &desc1,
		})
		require.NoError(t, err)
		require.NotEmpty(t, tt.Token)
		require.NotNil(t, tt.Description)
		require.Equal(t, *tt.Description, desc1)

		desc2 := fmt.Sprintf("go-tfe-team-token-test-%s", randomString(t))
		tt, err = client.TeamTokens.CreateWithOptions(ctx, tmTest.ID, TeamTokenCreateOptions{
			Description: &desc2,
		})
		require.NoError(t, err)
		require.NotEmpty(t, tt.Token)
		require.NotNil(t, tt.Description)
		require.Equal(t, *tt.Description, desc2)

		emptyString := ""
		tt, err = client.TeamTokens.CreateWithOptions(ctx, tmTest.ID, TeamTokenCreateOptions{
			Description: &emptyString,
		})
		require.NoError(t, err)
		require.NotEmpty(t, tt.Token)
		require.NotNil(t, tt.Description)
		require.Equal(t, *tt.Description, emptyString)

		tt, err = client.TeamTokens.CreateWithOptions(ctx, tmTest.ID, TeamTokenCreateOptions{})
		require.NoError(t, err)
		require.NotEmpty(t, tt.Token)
		require.Nil(t, tt.Description)
	})

	t.Run("with an expiration date", func(t *testing.T) {
		desc := fmt.Sprintf("go-tfe-team-token-test-%s", randomString(t))
		currentTime := time.Now().UTC().Truncate(time.Second)
		oneDayLater := currentTime.Add(24 * time.Hour)
		tt, err := client.TeamTokens.CreateWithOptions(ctx, tmTest.ID, TeamTokenCreateOptions{
			Description: &desc,
			ExpiredAt:   &oneDayLater,
		})
		require.NoError(t, err)
		require.NotEmpty(t, tt.Token)
		assert.Equal(t, tt.ExpiredAt, oneDayLater)
		require.NotNil(t, tt.Description)
		require.Equal(t, *tt.Description, desc)
	})

	t.Run("without an expiration date", func(t *testing.T) {
		desc := fmt.Sprintf("go-tfe-team-token-test-%s", randomString(t))
		tt, err := client.TeamTokens.CreateWithOptions(ctx, tmTest.ID, TeamTokenCreateOptions{
			Description: &desc,
		})
		require.NoError(t, err)
		require.NotEmpty(t, tt.Token)
		assert.Empty(t, tt.ExpiredAt)
		require.NotNil(t, tt.Description)
		require.Equal(t, *tt.Description, desc)
	})

	t.Run("when a token already exists with the same description", func(t *testing.T) {
		desc := fmt.Sprintf("go-tfe-team-token-test-%s", randomString(t))
		tt, err := client.TeamTokens.CreateWithOptions(ctx, tmTest.ID, TeamTokenCreateOptions{
			Description: &desc,
		})
		require.NoError(t, err)
		require.NotEmpty(t, tt.Token)
		require.NotNil(t, tt.Description)
		require.Equal(t, *tt.Description, desc)

		tt, err = client.TeamTokens.CreateWithOptions(ctx, tmTest.ID, TeamTokenCreateOptions{
			Description: &desc,
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
		require.NotEmpty(t, tt.Team)
		assert.Equal(t, tt.Team.ID, tmTest.ID)

		ttTestCleanup()
	})

	t.Run("with an expiration date passed as a valid option", func(t *testing.T) {
		currentTime := time.Now().UTC().Truncate(time.Second)
		oneDayLater := currentTime.Add(24 * time.Hour)

		_, ttTestCleanup := createTeamTokenWithOptions(t, client, tmTest, TeamTokenCreateOptions{ExpiredAt: &oneDayLater})

		tt, err := client.TeamTokens.Read(ctx, tmTest.ID)
		require.NoError(t, err)
		require.NotEmpty(t, tt)
		assert.Equal(t, tt.ExpiredAt, oneDayLater)
		require.NotEmpty(t, tt.Team)
		assert.Equal(t, tt.Team.ID, tmTest.ID)

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

func TestTeamTokensReadByID(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	tmTest, tmTestCleanup := createTeam(t, client, nil)
	t.Cleanup(tmTestCleanup)

	currentTime := time.Now().UTC().Truncate(time.Second)
	oneDayLater := currentTime.Add(24 * time.Hour)
	t.Run("with legacy, descriptionless tokens", func(t *testing.T) {
		token, ttTestCleanup := createTeamTokenWithOptions(t, client, tmTest, TeamTokenCreateOptions{
			ExpiredAt: &oneDayLater,
		})
		t.Cleanup(ttTestCleanup)

		tt, err := client.TeamTokens.ReadByID(ctx, token.ID)
		require.NoError(t, err)
		require.NotEmpty(t, tt)
		assert.Nil(t, tt.Description)
		assert.Equal(t, tt.ExpiredAt, oneDayLater)
		require.NotEmpty(t, tt.Team)
		assert.Equal(t, tt.Team.ID, tmTest.ID)
	})

	t.Run("with multiple team tokens", func(t *testing.T) {
		skipUnlessBeta(t)
		desc1 := fmt.Sprintf("go-tfe-team-token-test-%s", randomString(t))
		token, ttTestCleanup := createTeamTokenWithOptions(t, client, tmTest, TeamTokenCreateOptions{
			Description: &desc1,
		})
		t.Cleanup(ttTestCleanup)

		tt, err := client.TeamTokens.ReadByID(ctx, token.ID)
		require.NoError(t, err)
		require.NotEmpty(t, tt)
		require.NotNil(t, tt.Description)
		assert.Equal(t, *tt.Description, desc1)
		assert.Empty(t, tt.ExpiredAt)
		require.NotEmpty(t, tt.Team)
		assert.Equal(t, tt.Team.ID, tmTest.ID)

		desc2 := fmt.Sprintf("go-tfe-team-token-test-%s", randomString(t))
		tokenWithExpiration, ttTestCleanup2 := createTeamTokenWithOptions(t, client, tmTest, TeamTokenCreateOptions{
			ExpiredAt:   &oneDayLater,
			Description: &desc2,
		})
		t.Cleanup(ttTestCleanup2)

		tt2, err := client.TeamTokens.ReadByID(ctx, tokenWithExpiration.ID)
		require.NoError(t, err)
		require.NotEmpty(t, tt2)
		require.NotNil(t, tt2.Description)
		assert.Equal(t, *tt2.Description, desc2)
		assert.Equal(t, tt2.ExpiredAt, oneDayLater)
		require.NotEmpty(t, tt.Team)
		assert.Equal(t, tt.Team.ID, tmTest.ID)
	})

	t.Run("when a token doesn't exists", func(t *testing.T) {
		tt, err := client.TeamTokens.ReadByID(ctx, "nonexistent-token-id")
		assert.Equal(t, ErrResourceNotFound, err)
		assert.Nil(t, tt)
	})
}

func TestTeamTokensList(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	org, orgTestCleanup := createOrganization(t, client)
	t.Cleanup(orgTestCleanup)

	// Create a team with a token
	team1, tmTestCleanup1 := createTeam(t, client, org)
	t.Cleanup(tmTestCleanup1)

	currentTime := time.Now().UTC().Truncate(time.Second)
	oneDayLater := currentTime.Add(24 * time.Hour)
	token1, ttTestCleanup := createTeamTokenWithOptions(t, client, team1, TeamTokenCreateOptions{
		ExpiredAt: &oneDayLater,
	})
	t.Cleanup(ttTestCleanup)

	// Create a second team with a token that has a later expiration date
	team2, tmTestCleanup2 := createTeam(t, client, org)
	t.Cleanup(tmTestCleanup2)

	twoDaysLater := currentTime.Add(48 * time.Hour)
	token2, ttTestCleanup := createTeamTokenWithOptions(t, client, team2, TeamTokenCreateOptions{
		ExpiredAt: &twoDaysLater,
	})
	t.Cleanup(ttTestCleanup)

	t.Run("with team tokens across multiple teams", func(t *testing.T) {
		tokens, err := client.TeamTokens.List(ctx, org.Name, nil)
		require.NoError(t, err)
		require.NotNil(t, tokens)
		require.Len(t, tokens.Items, 2)
		require.ElementsMatch(t, []string{token1.ID, token2.ID}, []string{tokens.Items[0].ID, tokens.Items[1].ID})
	})

	t.Run("with filtering by team name", func(t *testing.T) {
		tokens, err := client.TeamTokens.List(ctx, org.Name, &TeamTokenListOptions{
			Query: team1.Name,
		})
		require.NoError(t, err)
		require.NotNil(t, tokens)
		require.Len(t, tokens.Items, 1)
		require.Equal(t, token1.ID, tokens.Items[0].ID)
	})

	t.Run("with sorting", func(t *testing.T) {
		tokens, err := client.TeamTokens.List(ctx, org.Name, &TeamTokenListOptions{
			Sort: "expired-at",
		})
		require.NoError(t, err)
		require.NotNil(t, tokens)
		require.Len(t, tokens.Items, 2)
		require.Equal(t, []string{token1.ID, token2.ID}, []string{tokens.Items[0].ID, tokens.Items[1].ID})

		tokens, err = client.TeamTokens.List(ctx, org.Name, &TeamTokenListOptions{
			Sort: "-expired-at",
		})
		require.NoError(t, err)
		require.NotNil(t, tokens)
		require.Len(t, tokens.Items, 2)
		require.Equal(t, []string{token2.ID, token1.ID}, []string{tokens.Items[0].ID, tokens.Items[1].ID})
	})

	t.Run("with multiple team tokens in a single team", func(t *testing.T) {
		skipUnlessBeta(t)
		desc1 := fmt.Sprintf("go-tfe-team-token-test-%s", randomString(t))
		multiToken1, ttTestCleanup := createTeamTokenWithOptions(t, client, team1, TeamTokenCreateOptions{
			Description: &desc1,
		})
		t.Cleanup(ttTestCleanup)

		desc2 := fmt.Sprintf("go-tfe-team-token-test-%s", randomString(t))
		multiToken2, ttTestCleanup := createTeamTokenWithOptions(t, client, team1, TeamTokenCreateOptions{
			Description: &desc2,
		})
		t.Cleanup(ttTestCleanup)

		tokens, err := client.TeamTokens.List(ctx, org.Name, nil)
		require.NoError(t, err)
		require.NotNil(t, tokens)
		require.Len(t, tokens.Items, 4)
		actualIDs := []string{}
		for _, token := range tokens.Items {
			actualIDs = append(actualIDs, token.ID)
		}
		require.ElementsMatch(t, []string{token1.ID, token2.ID, multiToken1.ID, multiToken2.ID},
			actualIDs)
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

func TestTeamTokensDeleteByID(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	tmTest, tmTestCleanup := createTeam(t, client, nil)
	t.Cleanup(tmTestCleanup)

	t.Run("with legacy, descriptionless tokens", func(t *testing.T) {
		token, _ := createTeamToken(t, client, tmTest)
		err := client.TeamTokens.DeleteByID(ctx, token.ID)
		require.NoError(t, err)
	})

	t.Run("with multiple team tokens", func(t *testing.T) {
		skipUnlessBeta(t)
		desc1 := fmt.Sprintf("go-tfe-team-token-test-%s", randomString(t))
		token1, _ := createTeamTokenWithOptions(t, client, tmTest, TeamTokenCreateOptions{
			Description: &desc1,
		})

		desc2 := fmt.Sprintf("go-tfe-team-token-test-%s", randomString(t))
		token2, _ := createTeamTokenWithOptions(t, client, tmTest, TeamTokenCreateOptions{
			Description: &desc2,
		})

		err := client.TeamTokens.DeleteByID(ctx, token1.ID)
		require.NoError(t, err)

		err = client.TeamTokens.DeleteByID(ctx, token2.ID)
		require.NoError(t, err)
	})

	t.Run("when a token does not exist", func(t *testing.T) {
		err := client.TeamTokens.DeleteByID(ctx, "nonexistent-token-id")
		assert.Equal(t, err, ErrResourceNotFound)
	})

	t.Run("with invalid token ID", func(t *testing.T) {
		err := client.TeamTokens.DeleteByID(ctx, badIdentifier)
		assert.Equal(t, err, ErrInvalidTokenID)
	})
}
