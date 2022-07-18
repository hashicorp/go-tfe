//go:build integration
// +build integration

package tfe

import (
	"testing"
)

func TestAgentsRead(t *testing.T) {
	client := testClient(t)
	// ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	defer orgTestCleanup()

	//an upgrade is necessary because the use of agents, agent pools is a paid feature
	upgradeOrganizationSubscription(t, client, orgTest)

	agentPool, agentPoolCleanup := createAgentPool(t, client, orgTest)
	t.Log(agentPool)
	defer agentPoolCleanup()

	agent, agentCleanup := createAgent(t, client, orgTest, agentPool, nil)
	defer agentCleanup()
	t.Log(agent)
	//createAgent fn that associates an org and agent pool
	//defer createAgent fn

}

func TestAgentsList(t *testing.T) {
	client := testClient(t)
	// ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	defer orgTestCleanup()

	upgradeOrganizationSubscription(t, client, orgTest)

	agentPool, agentPoolCleanup := createAgentPool(t, client, orgTest)
	t.Log(agentPool)
	defer agentPoolCleanup()

}

func TestAgentsDelete(t *testing.T) {
	client := testClient(t)
	// ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	defer orgTestCleanup()

	upgradeOrganizationSubscription(t, client, orgTest)

	agentPool, _ := createAgentPool(t, client, orgTest)
	t.Log(agentPool)

}
