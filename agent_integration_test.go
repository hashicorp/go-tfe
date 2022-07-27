//go:build integration
// +build integration

package tfe

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAgentsRead(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	agent, _, agentCleanup := createAgent(t, client, nil, nil, nil)
	defer agentCleanup()
	t.Log("log agent: ", agent)
	// t.Log("log pool: ", pool)

	t.Run("when the agent exists", func(t *testing.T) {
		k, err := client.Agents.Read(ctx, agent.ID)
		require.NoError(t, err)
		assert.Equal(t, agent, k)
	})

	t.Run("when the agent does not exist", func(t *testing.T) {
		k, err := client.Agents.Read(ctx, "nonexisting")
		assert.Nil(t, k)
		assert.Equal(t, err, ErrResourceNotFound)
	})

	t.Run("without a valid agent ID", func(t *testing.T) {
		k, err := client.Agents.Read(ctx, badIdentifier)
		assert.Nil(t, k)
		assert.EqualError(t, err, ErrInvalidAgentID.Error())
	})
}

func TestAgentsList(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	agent, agentPool, agentCleanup := createAgent(t, client, nil, nil, nil)
	defer agentCleanup()
	t.Log("log agent: ", agent)
	t.Log("log agent pool: ", agentPool)

	t.Run("expect an agent to exist", func(t *testing.T) {
		agent, err := client.Agents.List(ctx, agentPool.ID, nil)

		require.NoError(t, err)
		require.NotEmpty(t, agent.Items)
		assert.NotEmpty(t, agent.Items[0].ID)
	})

	t.Run("without a valid agent pool ID", func(t *testing.T) {
		agents, err := client.Agents.List(ctx, badIdentifier, nil)
		assert.Nil(t, agents)
		assert.EqualError(t, err, ErrInvalidOrg.Error())
	})
}
