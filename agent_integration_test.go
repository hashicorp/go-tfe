// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package tfe

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAgentsRead(t *testing.T) {
	skipUnlessLinuxAMD64(t)

	client := testClient(t)
	ctx := context.Background()

	org, orgCleanup := createOrganization(t, client)
	t.Cleanup(orgCleanup)

	upgradeOrganizationSubscription(t, client, org)

	agent, _, agentCleanup := createAgent(t, client, org, nil, "")
	t.Cleanup(agentCleanup)

	t.Run("when the agent exists", func(t *testing.T) {
		k, err := client.Agents.Read(ctx, agent.ID)
		require.NoError(t, err)
		assert.Equal(t, agent, k)
	})

	t.Run("when the agent does not exist", func(t *testing.T) {
		k, err := client.Agents.Read(ctx, "nonexistent")
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
	skipUnlessLinuxAMD64(t)

	client := testClient(t)
	ctx := context.Background()

	org, orgCleanup := createOrganization(t, client)
	t.Cleanup(orgCleanup)

	upgradeOrganizationSubscription(t, client, org)

	agent1, agentPool, agentCleanup := createAgent(t, client, org, nil, "agent1")
	t.Cleanup(agentCleanup)
	agent2, agentPool2, agentCleanup2 := createAgent(t, client, org, agentPool, "agent2")
	fmt.Println("agent pool stuff")
	fmt.Println(agentPool2.ID)
	fmt.Println(agentPool.ID)
	t.Cleanup(agentCleanup2)

	t.Run("expect an agent to exist", func(t *testing.T) {
		agent, err := client.Agents.List(ctx, agentPool.ID, nil)

		require.NoError(t, err)
		require.NotEmpty(t, agent.Items)
		assert.NotEmpty(t, agent.Items[0].ID)
	})

	t.Run("with sorting", func(t *testing.T) {
		agents, err := client.Agents.List(ctx, agentPool.ID, &AgentListOptions{
			Sort: "created-at",
		})
		fmt.Println("line 78")
		fmt.Println(agents.Items[0].Name)
		fmt.Println(agents.Items[0].ID)
		fmt.Println(agents.Items[1].Name)
		fmt.Println(agents.Items[1].ID)
		require.NoError(t, err)
		require.NotNil(t, agents)
		require.Len(t, agents.Items, 2)
		fmt.Println(agent1)
		fmt.Println(agent1.Name)
		fmt.Println(agent2)
		fmt.Println(agent2.Name)
		fmt.Println(agents)

		require.Equal(t, []string{agent1.ID, agent2.ID}, []string{agents.Items[0].ID, agents.Items[1].ID})

		agents, err = client.Agents.List(ctx, agentPool.ID, &AgentListOptions{
			Sort: "-created-at",
		})
		require.NoError(t, err)
		require.NotNil(t, agents)
		require.Len(t, agents.Items, 2)
		require.Equal(t, []string{agent2.ID, agent1.ID}, []string{agents.Items[0].ID, agents.Items[1].ID})
	})

	t.Run("without a valid agent pool ID", func(t *testing.T) {
		agent, err := client.Agents.List(ctx, badIdentifier, nil)
		assert.Nil(t, agent)
		assert.EqualError(t, err, ErrInvalidOrg.Error())
	})
}
