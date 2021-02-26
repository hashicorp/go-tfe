package tfe

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAgentPoolsList(t *testing.T) {
	skipIfEnterprise(t)

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
		assert.EqualError(t, err, ErrInvalidOrg.Error())
	})
}

func TestAgentPoolsCreate(t *testing.T) {
	skipIfEnterprise(t)

	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	defer orgTestCleanup()

	t.Run("with valid options", func(t *testing.T) {
		options := AgentPoolCreateOptions{
			Name: String("cool-pool"),
		}

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

	t.Run("when options is missing name", func(t *testing.T) {
		k, err := client.AgentPools.Create(ctx, "foo", AgentPoolCreateOptions{})
		assert.Nil(t, k)
		assert.EqualError(t, err, ErrRequiredName.Error())
	})

	t.Run("with an invalid organization", func(t *testing.T) {
		pool, err := client.AgentPools.Create(ctx, badIdentifier, AgentPoolCreateOptions{
			Name: String("cool-pool"),
		})
		assert.Nil(t, pool)
		assert.EqualError(t, err, ErrInvalidOrg.Error())
	})
}

func TestAgentPoolsRead(t *testing.T) {
	skipIfEnterprise(t)

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
		assert.EqualError(t, err, ErrInvalidAgentPoolID.Error())
	})
}

func TestAgentPoolsUpdate(t *testing.T) {
	skipIfEnterprise(t)

	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	defer orgTestCleanup()

	t.Run("with valid options", func(t *testing.T) {
		kBefore, kTestCleanup := createAgentPool(t, client, orgTest)
		defer kTestCleanup()

		kAfter, err := client.AgentPools.Update(ctx, kBefore.ID, AgentPoolUpdateOptions{
			Name: String(randomString(t)),
		})
		require.NoError(t, err)

		assert.Equal(t, kBefore.ID, kAfter.ID)
		assert.NotEqual(t, kBefore.Name, kAfter.Name)
	})

	t.Run("when updating the name", func(t *testing.T) {
		kBefore, kTestCleanup := createAgentPool(t, client, orgTest)
		defer kTestCleanup()

		kAfter, err := client.AgentPools.Update(ctx, kBefore.ID, AgentPoolUpdateOptions{
			Name: String("updated-key-name"),
		})
		require.NoError(t, err)

		assert.Equal(t, kBefore.ID, kAfter.ID)
		assert.Equal(t, "updated-key-name", kAfter.Name)
	})

	t.Run("without a valid agent pool ID", func(t *testing.T) {
		w, err := client.AgentPools.Update(ctx, badIdentifier, AgentPoolUpdateOptions{})
		assert.Nil(t, w)
		assert.EqualError(t, err, ErrInvalidAgentPoolID.Error())
	})
}

func TestAgentPoolsDelete(t *testing.T) {
	skipIfEnterprise(t)

	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	defer orgTestCleanup()

	kTest, _ := createAgentPool(t, client, orgTest)

	t.Run("with valid options", func(t *testing.T) {
		err := client.AgentPools.Delete(ctx, kTest.ID)
		require.NoError(t, err)

		// Try loading the agent pool - it should fail.
		_, err = client.AgentPools.Read(ctx, kTest.ID)
		assert.Equal(t, err, ErrResourceNotFound)
	})

	t.Run("when the agent pool does not exist", func(t *testing.T) {
		err := client.AgentPools.Delete(ctx, kTest.ID)
		assert.Equal(t, err, ErrResourceNotFound)
	})

	t.Run("when the agent pool ID is invalid", func(t *testing.T) {
		err := client.AgentPools.Delete(ctx, badIdentifier)
		assert.EqualError(t, err, ErrInvalidAgentPoolID.Error())
	})
}
