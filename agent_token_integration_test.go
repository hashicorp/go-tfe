// Copyright IBM Corp. 2018, 2025
// SPDX-License-Identifier: MPL-2.0

package tfe

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAgentTokensList(t *testing.T) {
	t.Parallel()
	skipIfEnterprise(t)

	client := testClient(t)
	ctx := context.Background()

	apTest, apTestCleanup := createAgentPool(t, client, nil)
	defer apTestCleanup()

	agentToken1, agentToken1Cleanup := createAgentToken(t, client, apTest)
	defer agentToken1Cleanup()
	_, agentToken2Cleanup := createAgentToken(t, client, apTest)
	defer agentToken2Cleanup()

	t.Run("with no list options", func(t *testing.T) {
		tokenlist, err := client.AgentTokens.List(ctx, apTest.ID)
		require.NoError(t, err)
		var found bool
		for _, j := range tokenlist.Items {
			if j.ID == agentToken1.ID {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("agent token (%s) not found in token list", agentToken1.ID)
		}

		assert.Equal(t, 1, tokenlist.CurrentPage)
		assert.Equal(t, 2, tokenlist.TotalCount)
	})

	t.Run("without a valid agent pool ID", func(t *testing.T) {
		tokenlist, err := client.AgentTokens.List(ctx, badIdentifier)
		assert.Nil(t, tokenlist)
		assert.EqualError(t, err, ErrInvalidAgentPoolID.Error())
	})
}

func TestAgentTokensCreate(t *testing.T) {
	t.Parallel()
	skipIfEnterprise(t)

	client := testClient(t)
	ctx := context.Background()

	apTest, apTestCleanup := createAgentPool(t, client, nil)
	defer apTestCleanup()

	t.Run("with valid description", func(t *testing.T) {
		token, err := client.AgentTokens.Create(ctx, apTest.ID, AgentTokenCreateOptions{
			Description: String(randomString(t)),
		})
		require.NoError(t, err)
		require.NotEmpty(t, token.Token)
	})

	t.Run("without valid description", func(t *testing.T) {
		at, err := client.AgentTokens.Create(ctx, badIdentifier, AgentTokenCreateOptions{})
		assert.Nil(t, at)
		assert.EqualError(t, err, ErrInvalidAgentPoolID.Error())
	})

	t.Run("without valid agent pool ID", func(t *testing.T) {
		at, err := client.AgentTokens.Create(ctx, badIdentifier, AgentTokenCreateOptions{
			Description: String(randomString(t)),
		})
		assert.Nil(t, at)
		assert.EqualError(t, err, ErrInvalidAgentPoolID.Error())
	})
}
func TestAgentTokensRead(t *testing.T) {
	t.Parallel()
	skipIfEnterprise(t)

	client := testClient(t)
	ctx := context.Background()

	apTest, apTestCleanup := createAgentPool(t, client, nil)
	defer apTestCleanup()

	token, tokenTestCleanup := createAgentToken(t, client, apTest)
	defer tokenTestCleanup()

	t.Run("read token with valid token ID", func(t *testing.T) {
		at, err := client.AgentTokens.Read(ctx, token.ID)
		require.NoError(t, err)
		// The initial API call to create a token will return a value in the token
		// object. Empty that out for comparison
		token.Token = ""
		assert.Equal(t, token, at)
	})

	t.Run("read token without valid token ID", func(t *testing.T) {
		_, err := client.AgentTokens.Read(ctx, badIdentifier)
		assert.EqualError(t, err, ErrInvalidAgentTokenID.Error())
	})
}

func TestAgentTokensReadCreatedBy(t *testing.T) {
	t.Parallel()
	skipIfEnterprise(t)

	client := testClient(t)
	ctx := context.Background()

	apTest, apTestCleanup := createAgentPool(t, client, nil)
	defer apTestCleanup()

	token, tokenTestCleanup := createAgentToken(t, client, apTest)
	defer tokenTestCleanup()

	at, err := client.AgentTokens.Read(ctx, token.ID)
	require.NoError(t, err)
	require.NotNil(t, at.CreatedBy)
}

func TestAgentTokensDelete(t *testing.T) {
	t.Parallel()
	skipIfEnterprise(t)

	client := testClient(t)
	ctx := context.Background()

	apTest, apTestCleanup := createAgentPool(t, client, nil)
	defer apTestCleanup()

	token, _ := createAgentToken(t, client, apTest)

	t.Run("with valid token ID", func(t *testing.T) {
		err := client.AgentTokens.Delete(ctx, token.ID)
		require.NoError(t, err)
	})

	t.Run("without valid token ID", func(t *testing.T) {
		err := client.AgentTokens.Delete(ctx, badIdentifier)
		assert.EqualError(t, err, ErrInvalidAgentTokenID.Error())
	})
}
