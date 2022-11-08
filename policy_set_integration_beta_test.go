package tfe

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPolicySetsList_Beta(t *testing.T) {
	skipIfFreeOnly(t)
	skipIfBeta(t)

	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	defer orgTestCleanup()

	upgradeOrganizationSubscription(t, client, orgTest)

	workspace, workspaceCleanup := createWorkspace(t, client, orgTest)
	defer workspaceCleanup()

	psTest1, psTestCleanup1 := createPolicySet(t, client, orgTest, nil, []*Workspace{workspace}, "")
	defer psTestCleanup1()
	psTest2, psTestCleanup2 := createPolicySet(t, client, orgTest, nil, []*Workspace{workspace}, "")
	defer psTestCleanup2()
	psTest3, psTestCleanup3 := createPolicySet(t, client, orgTest, nil, []*Workspace{workspace}, OPA)
	defer psTestCleanup3()

	t.Run("without list options", func(t *testing.T) {
		psl, err := client.PolicySets.List(ctx, orgTest.Name, nil)
		require.NoError(t, err)

		assert.Contains(t, psl.Items, psTest1)
		assert.Contains(t, psl.Items, psTest2)
		assert.Contains(t, psl.Items, psTest3)
		assert.Equal(t, 1, psl.CurrentPage)
		assert.Equal(t, 3, psl.TotalCount)
	})

	t.Run("with pagination", func(t *testing.T) {
		// Request a page number which is out of range. The result should
		// be successful, but return no results if the paging options are
		// properly passed along.
		psl, err := client.PolicySets.List(ctx, orgTest.Name, &PolicySetListOptions{
			ListOptions: ListOptions{
				PageNumber: 999,
				PageSize:   100,
			},
		})
		require.NoError(t, err)

		assert.Empty(t, psl.Items)
		assert.Equal(t, 999, psl.CurrentPage)
		assert.Equal(t, 3, psl.TotalCount)
	})

	t.Run("with search", func(t *testing.T) {
		// Search by one of the policy set's names; we should get only that policy
		// set and pagination data should reflect the search as well
		psl, err := client.PolicySets.List(ctx, orgTest.Name, &PolicySetListOptions{
			Search: psTest1.Name,
		})
		require.NoError(t, err)

		assert.Contains(t, psl.Items, psTest1)
		assert.NotContains(t, psl.Items, psTest2)
		assert.Equal(t, 1, psl.CurrentPage)
		assert.Equal(t, 1, psl.TotalCount)
	})

	t.Run("with include param", func(t *testing.T) {
		psl, err := client.PolicySets.List(ctx, orgTest.Name, &PolicySetListOptions{
			Include: []PolicySetIncludeOpt{PolicySetWorkspaces},
		})
		require.NoError(t, err)

		assert.Equal(t, 3, len(psl.Items))

		assert.NotNil(t, psl.Items[0].Workspaces)
		assert.Equal(t, 1, len(psl.Items[0].Workspaces))
		assert.Equal(t, workspace.ID, psl.Items[0].Workspaces[0].ID)
	})

	t.Run("filter by kind", func(t *testing.T) {
		psl, err := client.PolicySets.List(ctx, orgTest.Name, &PolicySetListOptions{
			Include: []PolicySetIncludeOpt{PolicySetWorkspaces},
			Kind:    OPA,
		})
		require.NoError(t, err)

		assert.Equal(t, 1, len(psl.Items))

		assert.NotNil(t, psl.Items[0].Workspaces)
		assert.Equal(t, 1, len(psl.Items[0].Workspaces))
		assert.Equal(t, workspace.ID, psl.Items[0].Workspaces[0].ID)
	})

	t.Run("without a valid organization", func(t *testing.T) {
		ps, err := client.PolicySets.List(ctx, badIdentifier, nil)
		assert.Nil(t, ps)
		assert.EqualError(t, err, ErrInvalidOrg.Error())
	})
}

func TestPolicySetsCreate_Beta(t *testing.T) {
	skipIfFreeOnly(t)
	skipIfBeta(t)

	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	defer orgTestCleanup()

	upgradeOrganizationSubscription(t, client, orgTest)

	var vcsPolicyID string

	t.Run("with valid attributes", func(t *testing.T) {
		options := PolicySetCreateOptions{
			Name: String("policy-set"),
			Kind: OPA,
		}

		ps, err := client.PolicySets.Create(ctx, orgTest.Name, options)
		require.NoError(t, err)

		assert.Equal(t, ps.Name, *options.Name)
		assert.Equal(t, ps.Description, "")
		assert.Equal(t, ps.Kind, OPA)
		assert.False(t, ps.Global)
	})

	t.Run("with kind missing", func(t *testing.T) {
		options := PolicySetCreateOptions{
			Name: String("policy-set1"),
		}

		ps, err := client.PolicySets.Create(ctx, orgTest.Name, options)
		require.NoError(t, err)

		assert.Equal(t, ps.Name, *options.Name)
		assert.Equal(t, ps.Description, "")
		assert.Equal(t, ps.Kind, Sentinel)
		assert.False(t, ps.Global)
	})

	t.Run("with all attributes provided - sentinel", func(t *testing.T) {
		options := PolicySetCreateOptions{
			Name:        String("global"),
			Description: String("Policies in this set will be checked in ALL workspaces!"),
			Kind:        Sentinel,
			Global:      Bool(true),
		}

		ps, err := client.PolicySets.Create(ctx, orgTest.Name, options)
		require.NoError(t, err)

		assert.Equal(t, ps.Name, *options.Name)
		assert.Equal(t, ps.Description, *options.Description)
		assert.Equal(t, ps.Kind, Sentinel)
		assert.True(t, ps.Global)
	})

	t.Run("with all attributes provided - OPA", func(t *testing.T) {
		options := PolicySetCreateOptions{
			Name:        String("global1"),
			Description: String("Policies in this set will be checked in ALL workspaces!"),
			Kind:        OPA,
			Overridable: Bool(true),
			Global:      Bool(true),
		}

		ps, err := client.PolicySets.Create(ctx, orgTest.Name, options)
		require.NoError(t, err)

		assert.Equal(t, ps.Name, *options.Name)
		assert.Equal(t, ps.Description, *options.Description)
		assert.Equal(t, ps.Overridable, options.Overridable)
		assert.Equal(t, ps.Kind, OPA)
		assert.True(t, ps.Global)
	})

	t.Run("with missing overridable attribute", func(t *testing.T) {
		options := PolicySetCreateOptions{
			Name:        String("global2"),
			Description: String("Policies in this set will be checked in ALL workspaces!"),
			Kind:        OPA,
			Global:      Bool(true),
		}

		ps, err := client.PolicySets.Create(ctx, orgTest.Name, options)
		require.NoError(t, err)

		assert.Equal(t, ps.Name, *options.Name)
		assert.Equal(t, ps.Description, *options.Description)
		assert.Equal(t, ps.Overridable, Bool(false))
		assert.Equal(t, ps.Kind, OPA)
		assert.True(t, ps.Global)
	})

	t.Run("with policies and workspaces provided", func(t *testing.T) {
		pTest, pTestCleanup := createPolicy(t, client, orgTest)
		defer pTestCleanup()
		wTest, wTestCleanup := createWorkspace(t, client, orgTest)
		defer wTestCleanup()

		options := PolicySetCreateOptions{
			Name:       String("populated-policy-set"),
			Policies:   []*Policy{pTest},
			Kind:       Sentinel,
			Workspaces: []*Workspace{wTest},
		}

		ps, err := client.PolicySets.Create(ctx, orgTest.Name, options)
		require.NoError(t, err)

		assert.Equal(t, ps.Name, *options.Name)
		assert.Equal(t, ps.PolicyCount, 1)
		assert.Equal(t, ps.Policies[0].ID, pTest.ID)
		assert.Equal(t, ps.WorkspaceCount, 1)
		assert.Equal(t, ps.Kind, Sentinel)
		assert.Equal(t, ps.Workspaces[0].ID, wTest.ID)
	})

	t.Run("with vcs policy set", func(t *testing.T) {
		githubIdentifier := os.Getenv("GITHUB_POLICY_SET_IDENTIFIER")
		if githubIdentifier == "" {
			t.Skip("Export a valid GITHUB_POLICY_SET_IDENTIFIER before running this test")
		}

		oc, ocTestCleanup := createOAuthToken(t, client, orgTest)
		defer ocTestCleanup()

		options := PolicySetCreateOptions{
			Name:         String("vcs-policy-set1"),
			Kind:         Sentinel,
			PoliciesPath: String("/policy-sets/foo"),
			VCSRepo: &VCSRepoOptions{
				Branch:            String("policies"),
				Identifier:        String(githubIdentifier),
				OAuthTokenID:      String(oc.ID),
				IngressSubmodules: Bool(true),
			},
		}

		ps, err := client.PolicySets.Create(ctx, orgTest.Name, options)
		require.NoError(t, err)

		// Save policy ID to be used by update func
		vcsPolicyID = ps.ID

		assert.Equal(t, ps.Name, *options.Name)
		assert.Equal(t, ps.Description, "")
		assert.False(t, ps.Global)
		assert.Equal(t, ps.PoliciesPath, "/policy-sets/foo")
		assert.Equal(t, ps.VCSRepo.Branch, "policies")
		assert.Equal(t, ps.Kind, Sentinel)
		assert.Equal(t, ps.VCSRepo.DisplayIdentifier, githubIdentifier)
		assert.Equal(t, ps.VCSRepo.Identifier, githubIdentifier)
		assert.Equal(t, ps.VCSRepo.IngressSubmodules, true)
		assert.Equal(t, ps.VCSRepo.OAuthTokenID, oc.ID)
		assert.Equal(t, ps.VCSRepo.RepositoryHTTPURL, fmt.Sprintf("https://github.com/%s", githubIdentifier))
		assert.Equal(t, ps.VCSRepo.ServiceProvider, string(ServiceProviderGithub))
		assert.Regexp(t, fmt.Sprintf("^%s/webhooks/vcs/[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12}$", regexp.QuoteMeta(DefaultConfig().Address)), ps.VCSRepo.WebhookURL)
	})

	t.Run("with vcs policy updated", func(t *testing.T) {
		githubIdentifier := os.Getenv("GITHUB_POLICY_SET_IDENTIFIER")
		if githubIdentifier == "" {
			t.Skip("Export a valid GITHUB_POLICY_SET_IDENTIFIER before running this test")
		}

		oc, ocTestCleanup := createOAuthToken(t, client, orgTest)
		defer ocTestCleanup()

		options := PolicySetUpdateOptions{
			Name:         String("vcs-policy-set"),
			PoliciesPath: String("/policy-sets/bar"),
			VCSRepo: &VCSRepoOptions{
				Branch:            String("policies"),
				Identifier:        String(githubIdentifier),
				OAuthTokenID:      String(oc.ID),
				IngressSubmodules: Bool(false),
			},
		}

		ps, err := client.PolicySets.Update(ctx, vcsPolicyID, options)
		require.NoError(t, err)

		assert.Equal(t, ps.Name, *options.Name)
		assert.Equal(t, ps.Description, "")
		assert.False(t, ps.Global)
		assert.Equal(t, ps.PoliciesPath, "/policy-sets/bar")
		assert.Equal(t, ps.VCSRepo.Branch, "policies")
		assert.Equal(t, ps.VCSRepo.DisplayIdentifier, githubIdentifier)
		assert.Equal(t, ps.VCSRepo.Identifier, githubIdentifier)
		assert.Equal(t, ps.VCSRepo.IngressSubmodules, false)
		assert.Equal(t, ps.VCSRepo.OAuthTokenID, oc.ID)
		assert.Equal(t, ps.VCSRepo.RepositoryHTTPURL, fmt.Sprintf("https://github.com/%s", githubIdentifier))
		assert.Equal(t, ps.VCSRepo.ServiceProvider, string(ServiceProviderGithub))
		assert.Regexp(t, fmt.Sprintf("^%s/webhooks/vcs/[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12}$", regexp.QuoteMeta(DefaultConfig().Address)), ps.VCSRepo.WebhookURL)
	})

	t.Run("without a name provided", func(t *testing.T) {
		ps, err := client.PolicySets.Create(ctx, orgTest.Name, PolicySetCreateOptions{})
		assert.Nil(t, ps)
		assert.EqualError(t, err, ErrRequiredName.Error())
	})

	t.Run("with an invalid name provided", func(t *testing.T) {
		ps, err := client.PolicySets.Create(ctx, orgTest.Name, PolicySetCreateOptions{
			Name: String("nope!"),
		})
		assert.Nil(t, ps)
		assert.EqualError(t, err, ErrInvalidName.Error())
	})

	t.Run("without a valid organization", func(t *testing.T) {
		ps, err := client.PolicySets.Create(ctx, badIdentifier, PolicySetCreateOptions{
			Name: String("policy-set"),
		})
		assert.Nil(t, ps)
		assert.EqualError(t, err, ErrInvalidOrg.Error())
	})
}
