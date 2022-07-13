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

	orgTest, orgTestCleanup := createOrganization(t, client)
	defer orgTestCleanup()

	upgradeOrganizationSubscription(t, client, orgTest)

	pool, poolCleanup := createAgentPool(t, client, orgTest)
	defer poolCleanup()

	t.Run("when the agent pool exists", func(t *testing.T) {
		k, err := client.AgentPools.Read(ctx, pool.ID)
		require.NoError(t, err)
		assert.Equal(t, pool, k)
	})

	t.Run("when the agent pool does not exist", func(t *testing.T) {
		k, err := client.AgentPools.Read(ctx, "nonexisting")
		assert.Nil(t, k)
		assert.Equal(t, err, ErrResourceNotFound)
	})

	t.Run("without a valid agent pool ID", func(t *testing.T) {
		k, err := client.AgentPools.Read(ctx, badIdentifier)
		assert.Nil(t, k)
		assert.EqualError(t, err, ErrInvalidAgentPoolID.Error())
	})

	t.Run("with Include option", func(t *testing.T) {
		_, wTestCleanup := createWorkspaceWithVCS(t, client, orgTest, WorkspaceCreateOptions{
			Name:          String("foo"),
			ExecutionMode: String("agent"),
			AgentPoolID:   String(pool.ID),
		})
		defer wTestCleanup()

		k, err := client.AgentPools.ReadWithOptions(ctx, pool.ID, &AgentPoolReadOptions{
			Include: []AgentPoolIncludeOpt{AgentPoolWorkspaces},
		})
		require.NoError(t, err)
		assert.NotEmpty(t, k.Workspaces[0])
	})
}

func TestAgentsList(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	defer orgTestCleanup()

	upgradeOrganizationSubscription(t, client, orgTest)

	agentPool, agentPoolCleanup := createAgentPool(t, client, orgTest)
	defer agentPoolCleanup()

	t.Run("without list options", func(t *testing.T) {
		pools, err := client.AgentPools.List(ctx, orgTest.Name, nil)
		require.NoError(t, err)
		assert.Contains(t, pools.Items, agentPool)

		assert.Equal(t, 1, pools.CurrentPage)
		assert.Equal(t, 1, pools.TotalCount)
	})

	t.Run("with Include option", func(t *testing.T) {
		_, wTestCleanup := createWorkspaceWithVCS(t, client, orgTest, WorkspaceCreateOptions{
			Name:          String("bar"),
			ExecutionMode: String("agent"),
			AgentPoolID:   String(agentPool.ID),
		})
		defer wTestCleanup()

		k, err := client.AgentPools.List(ctx, orgTest.Name, &AgentPoolListOptions{
			Include: []AgentPoolIncludeOpt{AgentPoolWorkspaces},
		})
		require.NoError(t, err)
		assert.NotEmpty(t, k.Items[0].Workspaces[0])
	})

	t.Run("with list options", func(t *testing.T) {
		// Request a page number which is out of range. The result should
		// be successful, but return no results if the paging options are
		// properly passed along.
		pools, err := client.AgentPools.List(ctx, orgTest.Name, &AgentPoolListOptions{
			ListOptions: ListOptions{
				PageNumber: 999,
				PageSize:   100,
			},
		})
		require.NoError(t, err)
		assert.Empty(t, pools.Items)
		assert.Equal(t, 999, pools.CurrentPage)
		assert.Equal(t, 1, pools.TotalCount)
	})

	t.Run("without a valid organization", func(t *testing.T) {
		pools, err := client.AgentPools.List(ctx, badIdentifier, nil)
		assert.Nil(t, pools)
		assert.EqualError(t, err, ErrInvalidOrg.Error())
	})
}

func TestAgentsDelete(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	defer orgTestCleanup()

	upgradeOrganizationSubscription(t, client, orgTest)

	agentPool, _ := createAgentPool(t, client, orgTest)

	t.Run("with valid options", func(t *testing.T) {
		err := client.AgentPools.Delete(ctx, agentPool.ID)
		require.NoError(t, err)

		// Try loading the agent pool - it should fail.
		_, err = client.AgentPools.Read(ctx, agentPool.ID)
		assert.Equal(t, err, ErrResourceNotFound)
	})

	t.Run("when the agent pool does not exist", func(t *testing.T) {
		err := client.AgentPools.Delete(ctx, agentPool.ID)
		assert.Equal(t, err, ErrResourceNotFound)
	})

	t.Run("when the agent pool ID is invalid", func(t *testing.T) {
		err := client.AgentPools.Delete(ctx, badIdentifier)
		assert.EqualError(t, err, ErrInvalidAgentPoolID.Error())
	})
}
