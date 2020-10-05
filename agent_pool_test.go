package tfe

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAgentPoolsList(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	defer orgTestCleanup()

	agentPool, _ := createAgentPool(t, client, orgTest)

	t.Run("without list options", func(t *testing.T) {
		pools, err := client.AgentPools.List(ctx, orgTest.Name, AgentPoolListOptions{})
		require.NoError(t, err)
		assert.Contains(t, pools.Items, agentPool)

		assert.Equal(t, 1, pools.CurrentPage)
		assert.Equal(t, 1, pools.TotalCount)
	})

	t.Run("with list options", func(t *testing.T) {
		// Request a page number which is out of range. The result should
		// be successful, but return no results if the paging options are
		// properly passed along.
		pools, err := client.AgentPools.List(ctx, orgTest.Name, AgentPoolListOptions{
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
		pools, err := client.AgentPools.List(ctx, badIdentifier, AgentPoolListOptions{})
		assert.Nil(t, pools)
		assert.EqualError(t, err, "invalid value for organization")
	})
}

func TestAgentPoolsCreate(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	defer orgTestCleanup()

	t.Run("with valid options", func(t *testing.T) {
		options := AgentPoolCreateOptions{}

		pool, err := client.AgentPools.Create(ctx, orgTest.Name, options)
		require.NoError(t, err)

		// Get a refreshed view from the API.
		refreshed, err := client.AgentPools.Read(ctx, pool.ID)
		require.NoError(t, err)

		for _, item := range []*AgentPool{
			pool,
			refreshed,
		} {
			assert.NotEmpty(t, item.ID)
		}
	})

	t.Run("with an invalid organization", func(t *testing.T) {
		pool, err := client.AgentPools.Create(ctx, badIdentifier, AgentPoolCreateOptions{})
		assert.Nil(t, pool)
		assert.EqualError(t, err, "invalid value for organization")
	})
}

func TestAgentPoolsRead(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	defer orgTestCleanup()

	pool, _ := createAgentPool(t, client, orgTest)

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
		assert.EqualError(t, err, "invalid value for agent pool ID")
	})
}
