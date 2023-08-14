package tfe

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPolicySetsCreate_Beta(t *testing.T) {
	skipUnlessBeta(t)
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	defer orgTestCleanup()

	upgradeOrganizationSubscription(t, client, orgTest)

	t.Run("with valid attributes", func(t *testing.T) {
		options := PolicySetCreateOptions{
			Name:              String(randomString(t)),
			Kind:              Sentinel,
			AgentEnabled:      true,
			PolicyToolVersion: "0.22.1",
		}

		ps, err := client.PolicySets.Create(ctx, orgTest.Name, options)
		require.NoError(t, err)

		assert.Equal(t, ps.Name, *options.Name)
		assert.Equal(t, ps.Description, "")
		assert.Equal(t, ps.Kind, Sentinel)
		assert.Equal(t, ps.AgentEnabled, true)
		assert.Equal(t, ps.PolicyToolVersion, "0.22.1")
		assert.False(t, ps.Global)
	})

	t.Run("with kind missing", func(t *testing.T) {
		options := PolicySetCreateOptions{
			Name:              String(randomString(t)),
			AgentEnabled:      true,
			PolicyToolVersion: "0.22.1",
			Overridable:       Bool(true),
		}

		ps, err := client.PolicySets.Create(ctx, orgTest.Name, options)
		require.NoError(t, err)

		assert.Equal(t, ps.Name, *options.Name)
		assert.Equal(t, ps.Description, "")
		assert.Equal(t, ps.Kind, Sentinel)
		assert.Equal(t, ps.AgentEnabled, true)
		assert.Equal(t, ps.PolicyToolVersion, "0.22.1")
		assert.False(t, ps.Global)
	})

	t.Run("with agent enabled missing", func(t *testing.T) {
		options := PolicySetCreateOptions{
			Name: String(randomString(t)),
			Kind: Sentinel,
		}

		ps, err := client.PolicySets.Create(ctx, orgTest.Name, options)
		require.NoError(t, err)

		assert.Equal(t, ps.Name, *options.Name)
		assert.Equal(t, ps.Description, "")
		assert.Equal(t, ps.Kind, Sentinel)
		assert.Equal(t, ps.AgentEnabled, false)
		assert.False(t, ps.Global)
	})
}

func TestPolicySetsList_Beta(t *testing.T) {
	skipUnlessBeta(t)

	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	defer orgTestCleanup()

	upgradeOrganizationSubscription(t, client, orgTest)

	workspace, workspaceCleanup := createWorkspace(t, client, orgTest)
	defer workspaceCleanup()

	options := PolicySetCreateOptions{
		Kind:              Sentinel,
		AgentEnabled:      true,
		PolicyToolVersion: "0.22.1",
		Overridable:       Bool(true),
	}

	psTest1, psTestCleanup1 := createPolicySetWithOptions(t, client, orgTest, nil, []*Workspace{workspace}, options)
	defer psTestCleanup1()
	psTest2, psTestCleanup2 := createPolicySetWithOptions(t, client, orgTest, nil, []*Workspace{workspace}, options)
	defer psTestCleanup2()

	t.Run("without list options", func(t *testing.T) {
		psl, err := client.PolicySets.List(ctx, orgTest.Name, nil)
		require.NoError(t, err)

		assert.Contains(t, psl.Items, psTest1)
		assert.Contains(t, psl.Items, psTest2)
		assert.Equal(t, true, psl.Items[0].AgentEnabled)
		assert.Equal(t, "0.22.1", psl.Items[0].PolicyToolVersion)
		assert.Equal(t, 1, psl.CurrentPage)
		assert.Equal(t, 2, psl.TotalCount)
	})
}

func TestPolicySetsUpdate_Beta(t *testing.T) {
	skipUnlessBeta(t)

	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	defer orgTestCleanup()

	upgradeOrganizationSubscription(t, client, orgTest)

	options := PolicySetCreateOptions{
		Kind:              Sentinel,
		AgentEnabled:      true,
		PolicyToolVersion: "0.22.1",
		Overridable:       Bool(true),
	}

	psTest, psTestCleanup := createPolicySetWithOptions(t, client, orgTest, nil, nil, options)
	defer psTestCleanup()

	t.Run("with valid attributes", func(t *testing.T) {
		options := PolicySetUpdateOptions{
			AgentEnabled: false,
		}

		ps, err := client.PolicySets.Update(ctx, psTest.ID, options)
		require.NoError(t, err)

		assert.Equal(t, ps.AgentEnabled, false)
		assert.Equal(t, ps.PolicyToolVersion, "")
		assert.Nil(t, ps.Overridable)
	})
}
