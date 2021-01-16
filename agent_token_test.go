package tfe

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAgentTokensList(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	apTest, apTestCleanup := createAgentPool(t, client, nil)
	defer apTestCleanup()

	agentToken1, agentToken1Cleanup := createAgentToken(t, client, apTest)
	defer agentToken1Cleanup()
	agentToken2, agentToken2Cleanup := createAgentToken(t, client, apTest)
	defer agentToken2Cleanup()

	t.Run("with no list options", func(t *testing.T) {
		tokenlist, err := client.AgentTokens.List(ctx, apTest.ID)
		require.NoError(t, err)
		assert.Contains(t, tokenlist.Items, agentToken1)
		assert.Contains(t, tokenlist.Items, agentToken2)

		t.Skip("paging not supported yet in API")
		assert.Equal(t, 1, tokenlist.CurrentPage)
		assert.Equal(t, 2, tokenlist.TotalCount)
	})

	t.Run("without a valid agent pool ID", func(t *testing.T) {
		tokenlist, err := client.AgentTokens.List(ctx, badIdentifier)
		assert.Nil(t, tokenlist)
		assert.EqualError(t, err, "invalid value for agent pool ID")
	})
}

func TestAgentTokensGenerate(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	apTest, apTestCleanup := createAgentPool(t, client, nil)
	defer apTestCleanup()

	t.Run("with valid description", func(t *testing.T) {
		token, err := client.AgentTokens.Generate(ctx, apTest.ID, AgentTokenGenerateOptions{
			Description: String(randomString(t)),
		})
		require.NoError(t, err)
		require.NotEmpty(t, token.Token)
	})

	t.Run("without valid description", func(t *testing.T) {
		at, err := client.AgentTokens.Generate(ctx, badIdentifier, AgentTokenGenerateOptions{})
		assert.Nil(t, at)
		assert.EqualError(t, err, "invalid value for agent pool ID")
	})

	t.Run("without valid agent pool ID", func(t *testing.T) {
		at, err := client.AgentTokens.Generate(ctx, badIdentifier, AgentTokenGenerateOptions{
			Description: String(randomString(t)),
		})
		assert.Nil(t, at)
		assert.EqualError(t, err, "invalid value for agent pool ID")
	})
}
func TestAgentTokensRead(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	apTest, apTestCleanup := createAgentPool(t, client, nil)
	defer apTestCleanup()

	token, tokenTestCleanup := createAgentToken(t, client, apTest)
	defer tokenTestCleanup()

	t.Run("read token with valid token ID", func(t *testing.T) {
		at, err := client.AgentTokens.Read(ctx, token.ID)
		assert.NoError(t, err)
		assert.Equal(t, token.Token, at.Token)
	})

	t.Run("read token without valid token ID", func(t *testing.T) {
		_, err := client.AgentTokens.Read(ctx, badIdentifier)
		assert.EqualError(t, err, "invalid value for agent token ID")
	})
}

func TestAgentTokensDelete(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	apTest, apTestCleanup := createAgentPool(t, client, nil)
	defer apTestCleanup()

	token, atTestCleanup := createAgentToken(t, client, apTest)
	defer atTestCleanup()

	t.Run("with valid token ID", func(t *testing.T) {
		err := client.AgentTokens.Delete(ctx, token.ID)
		assert.NoError(t, err)
	})

	t.Run("without valid token ID", func(t *testing.T) {
		err := client.AgentTokens.Delete(ctx, badIdentifier)
		assert.EqualError(t, err, "invalid value for agent token ID")
	})
}
