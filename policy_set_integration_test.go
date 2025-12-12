// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package tfe

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPolicySetsList(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	defer orgTestCleanup()

	upgradeOrganizationSubscription(t, client, orgTest)

	version := createAdminSentinelVersion()
	opts := AdminSentinelVersionCreateOptions{
		Version:    version,
		URL:        "https://www.hashicorp.com",
		SHA:        genSha(t),
		Official:   Bool(false),
		Deprecated: Bool(false),
		Enabled:    Bool(true),
		Beta:       Bool(false),
	}
	sv, err := client.Admin.SentinelVersions.Create(ctx, opts)
	require.NoError(t, err)

	workspace, workspaceCleanup := createWorkspace(t, client, orgTest)
	defer workspaceCleanup()
	excludedWorkspace, excludedWorkspaceCleanup := createWorkspace(t, client, orgTest)
	defer excludedWorkspaceCleanup()

	options := PolicySetCreateOptions{
		Kind:              Sentinel,
		AgentEnabled:      Bool(true),
		PolicyToolVersion: String(sv.Version),
		Overridable:       Bool(true),
	}

	psTest1, psTestCleanup1 := createPolicySetWithOptions(t, client, orgTest, nil, []*Workspace{workspace}, []*Workspace{excludedWorkspace}, nil, options)
	defer psTestCleanup1()
	psTest2, psTestCleanup2 := createPolicySetWithOptions(t, client, orgTest, nil, []*Workspace{workspace}, []*Workspace{excludedWorkspace}, nil, options)
	defer psTestCleanup2()
	psTest3, psTestCleanup3 := createPolicySet(t, client, orgTest, nil, []*Workspace{workspace}, []*Workspace{excludedWorkspace}, nil, OPA)
	defer psTestCleanup3()
	defer func() {
		err := client.Admin.SentinelVersions.Delete(ctx, sv.ID)
		require.NoError(t, err)
	}()

	t.Run("without list options", func(t *testing.T) {
		psl, err := client.PolicySets.List(ctx, orgTest.Name, nil)
		require.NoError(t, err)

		assert.Contains(t, psl.Items, psTest1)
		assert.Contains(t, psl.Items, psTest2)
		assert.Contains(t, psl.Items, psTest3)
		assert.Equal(t, true, psl.Items[0].AgentEnabled)
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

	t.Run("with workspace exclusion include param", func(t *testing.T) {
		psl, err := client.PolicySets.List(ctx, orgTest.Name, &PolicySetListOptions{
			Include: []PolicySetIncludeOpt{PolicySetWorkspaceExclusions},
		})
		require.NoError(t, err)
		assert.Equal(t, 3, len(psl.Items))

		assert.NotNil(t, psl.Items[0].WorkspaceExclusions)
		assert.Equal(t, 1, len(psl.Items[0].WorkspaceExclusions))
		assert.Equal(t, excludedWorkspace.ID, psl.Items[0].WorkspaceExclusions[0].ID)
		assert.Equal(t, excludedWorkspace.Name, psl.Items[0].WorkspaceExclusions[0].Name)
		assert.Equal(t, excludedWorkspace.CreatedAt, psl.Items[0].WorkspaceExclusions[0].CreatedAt)
	})

	t.Run("without a valid organization", func(t *testing.T) {
		ps, err := client.PolicySets.List(ctx, badIdentifier, nil)
		assert.Nil(t, ps)
		assert.EqualError(t, err, ErrInvalidOrg.Error())
	})
}

func TestPolicySetsCreate(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	defer orgTestCleanup()

	upgradeOrganizationSubscription(t, client, orgTest)

	version := createAdminSentinelVersion()
	opts := AdminSentinelVersionCreateOptions{
		Version:    version,
		URL:        "https://www.hashicorp.com",
		SHA:        genSha(t),
		Official:   Bool(false),
		Deprecated: Bool(false),
		Enabled:    Bool(true),
		Beta:       Bool(false),
	}
	sv, err := client.Admin.SentinelVersions.Create(ctx, opts)
	require.NoError(t, err)

	defer func() {
		err := client.Admin.SentinelVersions.Delete(ctx, sv.ID)
		require.NoError(t, err)
	}()
	require.NoError(t, err)

	var vcsPolicyID string

	t.Run("with valid attributes", func(t *testing.T) {
		options := PolicySetCreateOptions{
			Name:              String(randomString(t)),
			PolicyToolVersion: String(sv.Version),
		}

		ps, err := client.PolicySets.Create(ctx, orgTest.Name, options)
		require.NoError(t, err)

		assert.Equal(t, ps.Name, *options.Name)
		assert.Equal(t, ps.Description, "")
		assert.False(t, ps.Global)
	})

	t.Run("OPA policy set with valid attributes", func(t *testing.T) {
		options := PolicySetCreateOptions{
			Name: String("opa-policy-set"),
			Kind: OPA,
		}

		ps, err := client.PolicySets.Create(ctx, orgTest.Name, options)
		require.NoError(t, err)

		assert.Equal(t, ps.Name, *options.Name)
		assert.Equal(t, ps.Description, "")
		assert.Equal(t, ps.Kind, OPA)
		assert.False(t, ps.Global)
	})

	t.Run("with pinned policy runtime version valid attributes", func(t *testing.T) {
		options := PolicySetCreateOptions{
			Name:              String(randomString(t)),
			Kind:              Sentinel,
			AgentEnabled:      Bool(true),
			PolicyToolVersion: String(sv.Version),
		}

		ps, err := client.PolicySets.Create(ctx, orgTest.Name, options)
		require.NoError(t, err)

		assert.Equal(t, ps.Name, *options.Name)
		assert.Equal(t, ps.Description, "")
		assert.Equal(t, ps.Kind, Sentinel)
		assert.Equal(t, ps.AgentEnabled, true)
		assert.Equal(t, ps.PolicyToolVersion, sv.Version)
		assert.False(t, ps.Global)
	})

	t.Run("with pinned policy runtime version and missing kind", func(t *testing.T) {
		options := PolicySetCreateOptions{
			Name:              String(randomString(t)),
			AgentEnabled:      Bool(true),
			PolicyToolVersion: String(sv.Version),
			Overridable:       Bool(true),
		}
		ps, err := client.PolicySets.Create(ctx, orgTest.Name, options)
		require.NoError(t, err)

		assert.Equal(t, ps.Name, *options.Name)
		assert.Equal(t, ps.Description, "")
		assert.Equal(t, ps.Kind, Sentinel)
		assert.Equal(t, ps.AgentEnabled, true)
		assert.Equal(t, ps.PolicyToolVersion, sv.Version)
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

	t.Run("with policies, workspaces and projects provided", func(t *testing.T) {
		pTest, pTestCleanup := createPolicy(t, client, orgTest)
		defer pTestCleanup()
		wTest, wTestCleanup := createWorkspace(t, client, orgTest)
		defer wTestCleanup()
		prjTest, prjTestCleanup := createProject(t, client, orgTest)
		defer prjTestCleanup()

		options := PolicySetCreateOptions{
			Name:       String("project-policy-set"),
			Policies:   []*Policy{pTest},
			Workspaces: []*Workspace{wTest},
			Projects:   []*Project{prjTest},
		}

		ps, err := client.PolicySets.Create(ctx, orgTest.Name, options)
		require.NoError(t, err)

		assert.Equal(t, ps.Name, *options.Name)
		assert.Equal(t, ps.PolicyCount, 1)
		assert.Equal(t, ps.Policies[0].ID, pTest.ID)
		assert.Equal(t, ps.WorkspaceCount, 1)
		assert.Equal(t, ps.Workspaces[0].ID, wTest.ID)
		assert.Equal(t, ps.ProjectCount, 1)
		assert.Equal(t, ps.Projects[0].ID, prjTest.ID)
	})

	t.Run("with policies and excluded workspaces provided", func(t *testing.T) {
		pTest, pTestCleanup := createPolicy(t, client, orgTest)
		defer pTestCleanup()
		wTest, wTestCleanup := createWorkspace(t, client, orgTest)
		defer wTestCleanup()

		options := PolicySetCreateOptions{
			Name:                String("exclusion-policy-set"),
			Policies:            []*Policy{pTest},
			WorkspaceExclusions: []*Workspace{wTest},
		}

		ps, err := client.PolicySets.Create(ctx, orgTest.Name, options)
		require.NoError(t, err)

		assert.Equal(t, ps.Name, *options.Name)
		assert.Equal(t, ps.PolicyCount, 1)
		assert.Equal(t, ps.Policies[0].ID, pTest.ID)
		assert.Equal(t, ps.WorkspaceExclusions[0].ID, wTest.ID)
		assert.Equal(t, len(ps.WorkspaceExclusions), 1)
	})

	t.Run("with vcs policy set", func(t *testing.T) {
		githubIdentifier := os.Getenv("GITHUB_POLICY_SET_IDENTIFIER")
		if githubIdentifier == "" {
			t.Skip("Export a valid GITHUB_POLICY_SET_IDENTIFIER before running this test")
		}

		// We are deliberately ignoring the cleanup func here, because there's a potential race condition
		// against the subsequent subtest -- HCP Terraform performs some async cleanup on VCS repos when deleting an
		// OAuthClient, and we've seen evidence that it will zero out the next test's NEW VCSRepo values if
		// they manage to slip in before the async stuff completes, even though the new values link it to a
		// new OAuthToken. Anyway, there's a deferred cleanup for orgTest in the outer scope, so the org's
		// dependent: destroy clause on OAuthClients will clean this up when the test as a whole ends.
		oc, _ := createOAuthToken(t, client, orgTest)

		options := PolicySetCreateOptions{
			Name:         String("vcs-policy-set"),
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
		assert.Equal(t, ps.Kind, Sentinel)
		assert.Equal(t, ps.PoliciesPath, "/policy-sets/foo")
		assert.Equal(t, ps.VCSRepo.Branch, "policies")
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

		// We are deliberately ignoring the cleanup func here, because it's not really necessary: there's a
		// deferred cleanup for orgTest in the outer scope, so the org's dependent: destroy clause on
		// OAuthClients will clean this up when the test as a whole ends. Unlike the one in the previous
		// subtest, there's no known race condition here because there aren't any later subtests that modify
		// this same policy set. But I'm being consistent with the prior case just to reduce risks from future
		// copypasta code.
		oc, _ := createOAuthToken(t, client, orgTest)

		options := PolicySetUpdateOptions{
			Name:         String("vcs-policy-set"),
			PoliciesPath: String("/policy-sets/bar"),
			VCSRepo: &VCSRepoOptions{
				Branch:            String("policies"),
				Identifier:        String(githubIdentifier),
				OAuthTokenID:      String(oc.ID),
				IngressSubmodules: Bool(false),
			},
			PolicyUpdatePattern: []*string{String("**"), String("*.sentinel")},
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

		assert.Equal(t, len(ps.PolicyUpdatePattern), 2)
		assert.Contains(t, ps.PolicyUpdatePattern, "*.sentinel")
	})

	t.Run("without a name provided", func(t *testing.T) {
		ps, err := client.PolicySets.Create(ctx, orgTest.Name, PolicySetCreateOptions{})
		assert.Nil(t, ps)
		assert.EqualError(t, err, ErrRequiredName.Error())
	})

	t.Run("with an invalid name provided", func(t *testing.T) {
		ps, err := client.PolicySets.Create(ctx, orgTest.Name, PolicySetCreateOptions{
			Name: String("nope/nope!"),
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

func TestPolicySetsCreateWithGithubApp(t *testing.T) {
	gHAInstallationID := os.Getenv("GITHUB_APP_INSTALLATION_ID")
	if gHAInstallationID == "" {
		t.Skip("Export a valid GITHUB_APP_INSTALLATION_ID before running this test!")
	}

	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	defer orgTestCleanup()

	upgradeOrganizationSubscription(t, client, orgTest)

	var vcsPolicyID string
	t.Run("with vcs policy set", func(t *testing.T) {
		githubIdentifier := os.Getenv("GITHUB_POLICY_SET_IDENTIFIER")
		if githubIdentifier == "" {
			t.Skip("Export a valid GITHUB_POLICY_SET_IDENTIFIER before running this test")
		}

		options := PolicySetCreateOptions{
			Name:         String("vcs-policy-set"),
			PoliciesPath: String("/policy-sets/foo"),
			VCSRepo: &VCSRepoOptions{
				Branch:            String("policies"),
				Identifier:        String(githubIdentifier),
				GHAInstallationID: String(gHAInstallationID),
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
		assert.Equal(t, ps.VCSRepo.DisplayIdentifier, githubIdentifier)
		assert.Equal(t, ps.VCSRepo.Identifier, githubIdentifier)
		assert.Equal(t, ps.VCSRepo.IngressSubmodules, true)
		assert.Equal(t, ps.VCSRepo.GHAInstallationID, gHAInstallationID)
		assert.Equal(t, ps.VCSRepo.RepositoryHTTPURL, fmt.Sprintf("https://github.com/%s", githubIdentifier))
	})

	t.Run("with vcs policy updated", func(t *testing.T) {
		githubIdentifier := os.Getenv("GITHUB_POLICY_SET_IDENTIFIER")
		if githubIdentifier == "" {
			t.Skip("Export a valid GITHUB_POLICY_SET_IDENTIFIER before running this test")
		}

		options := PolicySetUpdateOptions{
			Name:         String("vcs-policy-set"),
			PoliciesPath: String("/policy-sets/bar"),
			VCSRepo: &VCSRepoOptions{
				Branch:            String("policies"),
				Identifier:        String(githubIdentifier),
				GHAInstallationID: String(gHAInstallationID),
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
		assert.Equal(t, ps.VCSRepo.GHAInstallationID, gHAInstallationID)
		assert.Equal(t, ps.VCSRepo.RepositoryHTTPURL, fmt.Sprintf("https://github.com/%s", githubIdentifier))
		assert.Equal(t, ps.VCSRepo.ServiceProvider, string("github_app"))
	})
}
func TestPolicySetsRead(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()
	orgTest, orgTestCleanup := createOrganization(t, client)
	defer orgTestCleanup()

	upgradeOrganizationSubscription(t, client, orgTest)

	psTest, psTestCleanup := createPolicySet(t, client, orgTest, nil, nil, nil, nil, "")
	defer psTestCleanup()

	t.Run("with a valid ID", func(t *testing.T) {
		ps, err := client.PolicySets.Read(ctx, psTest.ID)
		require.NoError(t, err)

		assert.Equal(t, ps.ID, psTest.ID)
	})

	t.Run("without a valid ID", func(t *testing.T) {
		ps, err := client.PolicySets.Read(ctx, badIdentifier)
		assert.Nil(t, ps)
		assert.Equal(t, err, ErrInvalidPolicySetID)
	})

	t.Run("with policy set version", func(t *testing.T) {
		psv, psvCleanup := createPolicySetVersion(t, client, psTest)
		defer psvCleanup()

		ps, err := client.PolicySets.Read(ctx, psTest.ID)
		require.NoError(t, err)

		// The newest one is the policy set version created in this test.
		assert.Equal(t, ps.NewestVersion.ID, psv.ID)
		// The current policy set version is nil because nothing has been uploaded
		assert.Nil(t, ps.CurrentVersion)

		psvNew, psvCleanupNew := createPolicySetVersion(t, client, psTest)
		defer psvCleanupNew()
		err = client.PolicySetVersions.Upload(
			ctx,
			*psv,
			"test-fixtures/policy-set-version",
		)
		require.NoError(t, err)

		// give HCP Terraform some time to process uploading the
		// policy set version before reading.
		time.Sleep(waitForPolicySetVersionUpload)

		opts := &PolicySetReadOptions{
			Include: []PolicySetIncludeOpt{PolicySetCurrentVersion, PolicySetNewestVersion},
		}
		psWithOptions, err := client.PolicySets.ReadWithOptions(ctx, psTest.ID, opts)
		require.NoError(t, err)

		// The newest policy set version is changed to the most recent one
		// that was created.
		require.NotNil(t, psWithOptions.NewestVersion)
		assert.Equal(t, psWithOptions.NewestVersion.ID, psvNew.ID)
		assert.Equal(t, psWithOptions.NewestVersion.Status, PolicySetVersionPending)
		// The current one is now set because policies were uploaded to the
		// policy set version. Notice how it is set to the one that was uplaoded,
		// not the newest policy set version.
		require.NotNil(t, psWithOptions.CurrentVersion)
		assert.Equal(t, psWithOptions.CurrentVersion.ID, psv.ID)
		assert.Equal(t, psWithOptions.CurrentVersion.Status, PolicySetVersionReady)
	})
}

func TestPolicySetsUpdate(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	defer orgTestCleanup()

	upgradeOrganizationSubscription(t, client, orgTest)

	version := createAdminSentinelVersion()
	opts := AdminSentinelVersionCreateOptions{
		Version:    version,
		URL:        "https://www.hashicorp.com",
		SHA:        genSha(t),
		Official:   Bool(false),
		Deprecated: Bool(false),
		Enabled:    Bool(true),
		Beta:       Bool(false),
	}
	sv, err := client.Admin.SentinelVersions.Create(ctx, opts)
	require.NoError(t, err)
	defer func() {
		delErr := client.Admin.SentinelVersions.Delete(ctx, sv.ID)
		require.NoError(t, delErr)
	}()

	options := PolicySetCreateOptions{
		Kind:              Sentinel,
		AgentEnabled:      Bool(true),
		PolicyToolVersion: String(sv.Version),
		Overridable:       Bool(true),
	}
	if sv != nil {
		options.PolicyToolVersion = String(sv.Version)
	}

	psTest, psTestCleanup := createPolicySetWithOptions(t, client, orgTest, nil, nil, nil, nil, options)
	defer psTestCleanup()
	psTest2, psTestCleanup2 := createPolicySet(t, client, orgTest, nil, nil, nil, nil, "opa")
	defer psTestCleanup2()

	t.Run("with valid attributes", func(t *testing.T) {
		options := PolicySetUpdateOptions{
			AgentEnabled: Bool(false),
			Name:         String("global"),
			Description:  String("Policies in this set will be checked in ALL workspaces!"),
			Global:       Bool(true),
		}

		ps, err := client.PolicySets.Update(ctx, psTest.ID, options)
		require.NoError(t, err)

		assert.Equal(t, ps.AgentEnabled, false)
		assert.Equal(t, ps.PolicyToolVersion, "")
		assert.Nil(t, ps.Overridable)
		assert.Equal(t, ps.Name, *options.Name)
		assert.Equal(t, ps.Description, *options.Description)
		assert.True(t, ps.Global)
	})

	t.Run("with valid attributes-OPA", func(t *testing.T) {
		options := PolicySetUpdateOptions{
			Name:        String("global2"),
			Description: String("Policies in this set will be checked in ALL workspaces!"),
			Global:      Bool(true),
			Overridable: Bool(true),
		}

		ps, err := client.PolicySets.Update(ctx, psTest2.ID, options)
		require.NoError(t, err)

		assert.Equal(t, ps.Name, *options.Name)
		assert.Equal(t, ps.Description, *options.Description)
		assert.True(t, ps.Global)
		assert.True(t, *ps.Overridable)
	})

	t.Run("with invalid attributes", func(t *testing.T) {
		ps, err := client.PolicySets.Update(ctx, psTest.ID, PolicySetUpdateOptions{
			Name: String("nope/nope!"),
		})
		assert.Nil(t, ps)
		assert.EqualError(t, err, ErrInvalidName.Error())
	})

	t.Run("without a valid ID", func(t *testing.T) {
		ps, err := client.PolicySets.Update(ctx, badIdentifier, PolicySetUpdateOptions{
			Name: String("policy-set"),
		})
		assert.Nil(t, ps)
		assert.Equal(t, err, ErrInvalidPolicySetID)
	})
}

func TestPolicySetsAddPolicies(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	defer orgTestCleanup()

	upgradeOrganizationSubscription(t, client, orgTest)

	pTest1, pTestCleanup1 := createPolicy(t, client, orgTest)
	defer pTestCleanup1()
	pTest2, pTestCleanup2 := createPolicy(t, client, orgTest)
	defer pTestCleanup2()
	psTest, psTestCleanup := createPolicySet(t, client, orgTest, nil, nil, nil, nil, "")
	defer psTestCleanup()

	t.Run("with policies provided", func(t *testing.T) {
		err := client.PolicySets.AddPolicies(ctx, psTest.ID, PolicySetAddPoliciesOptions{
			Policies: []*Policy{pTest1, pTest2},
		})
		require.NoError(t, err)

		ps, err := client.PolicySets.Read(ctx, psTest.ID)
		require.NoError(t, err)
		assert.Equal(t, ps.PolicyCount, 2)

		ids := []string{}
		for _, policy := range ps.Policies {
			ids = append(ids, policy.ID)
		}

		assert.Contains(t, ids, pTest1.ID)
		assert.Contains(t, ids, pTest2.ID)
	})

	t.Run("without policies provided", func(t *testing.T) {
		err := client.PolicySets.AddPolicies(ctx, psTest.ID, PolicySetAddPoliciesOptions{})
		assert.Equal(t, err, ErrRequiredPolicies)
	})

	t.Run("with empty policies slice", func(t *testing.T) {
		err := client.PolicySets.AddPolicies(ctx, psTest.ID, PolicySetAddPoliciesOptions{
			Policies: []*Policy{},
		})
		assert.Equal(t, err, ErrInvalidPolicies)
	})

	t.Run("without a valid ID", func(t *testing.T) {
		err := client.PolicySets.AddPolicies(ctx, badIdentifier, PolicySetAddPoliciesOptions{
			Policies: []*Policy{pTest1, pTest2},
		})
		assert.Equal(t, err, ErrInvalidPolicySetID)
	})
}

func TestPolicySetsRemovePolicies(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	defer orgTestCleanup()

	upgradeOrganizationSubscription(t, client, orgTest)

	pTest1, pTestCleanup1 := createPolicy(t, client, orgTest)
	defer pTestCleanup1()
	pTest2, pTestCleanup2 := createPolicy(t, client, orgTest)
	defer pTestCleanup2()
	psTest, psTestCleanup := createPolicySet(t, client, orgTest, nil, nil, nil, nil, "")
	defer psTestCleanup()

	t.Run("with policies provided", func(t *testing.T) {
		err := client.PolicySets.RemovePolicies(ctx, psTest.ID, PolicySetRemovePoliciesOptions{
			Policies: []*Policy{pTest1, pTest2},
		})
		require.NoError(t, err)

		ps, err := client.PolicySets.Read(ctx, psTest.ID)
		require.NoError(t, err)

		assert.Equal(t, 0, ps.PolicyCount)
		assert.Empty(t, ps.Policies)
	})

	t.Run("without policies provided", func(t *testing.T) {
		err := client.PolicySets.RemovePolicies(ctx, psTest.ID, PolicySetRemovePoliciesOptions{})
		assert.Equal(t, err, ErrRequiredPolicies)
	})

	t.Run("with empty policies slice", func(t *testing.T) {
		err := client.PolicySets.RemovePolicies(ctx, psTest.ID, PolicySetRemovePoliciesOptions{
			Policies: []*Policy{},
		})
		assert.Equal(t, err, ErrInvalidPolicies)
	})

	t.Run("without a valid ID", func(t *testing.T) {
		err := client.PolicySets.RemovePolicies(ctx, badIdentifier, PolicySetRemovePoliciesOptions{
			Policies: []*Policy{pTest1, pTest2},
		})
		assert.Equal(t, err, ErrInvalidPolicySetID)
	})
}

func TestPolicySetsAddWorkspaces(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	defer orgTestCleanup()

	upgradeOrganizationSubscription(t, client, orgTest)

	wTest1, wTestCleanup1 := createWorkspace(t, client, orgTest)
	defer wTestCleanup1()
	wTest2, wTestCleanup2 := createWorkspace(t, client, orgTest)
	defer wTestCleanup2()
	psTest, psTestCleanup := createPolicySet(t, client, orgTest, nil, nil, nil, nil, "")
	defer psTestCleanup()

	t.Run("with workspaces provided", func(t *testing.T) {
		err := client.PolicySets.AddWorkspaces(
			ctx,
			psTest.ID,
			PolicySetAddWorkspacesOptions{
				Workspaces: []*Workspace{wTest1, wTest2},
			},
		)
		require.NoError(t, err)

		ps, err := client.PolicySets.Read(ctx, psTest.ID)
		require.NoError(t, err)
		assert.Equal(t, 2, ps.WorkspaceCount)

		ids := []string{}
		for _, ws := range ps.Workspaces {
			ids = append(ids, ws.ID)
		}

		assert.Contains(t, ids, wTest1.ID)
		assert.Contains(t, ids, wTest2.ID)
	})

	t.Run("without workspaces provided", func(t *testing.T) {
		err := client.PolicySets.AddWorkspaces(
			ctx,
			psTest.ID,
			PolicySetAddWorkspacesOptions{},
		)
		assert.Equal(t, err, ErrWorkspacesRequired)
	})

	t.Run("with empty workspaces slice", func(t *testing.T) {
		err := client.PolicySets.AddWorkspaces(
			ctx,
			psTest.ID,
			PolicySetAddWorkspacesOptions{Workspaces: []*Workspace{}},
		)
		assert.Equal(t, err, ErrWorkspaceMinLimit)
	})

	t.Run("without a valid ID", func(t *testing.T) {
		err := client.PolicySets.AddWorkspaces(
			ctx,
			badIdentifier,
			PolicySetAddWorkspacesOptions{
				Workspaces: []*Workspace{wTest1, wTest2},
			},
		)
		assert.Equal(t, err, ErrInvalidPolicySetID)
	})
}

func TestPolicySetsRemoveWorkspaces(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	defer orgTestCleanup()

	upgradeOrganizationSubscription(t, client, orgTest)

	wTest1, wTestCleanup1 := createWorkspace(t, client, orgTest)
	defer wTestCleanup1()
	wTest2, wTestCleanup2 := createWorkspace(t, client, orgTest)
	defer wTestCleanup2()
	psTest, psTestCleanup := createPolicySet(t, client, orgTest, nil, []*Workspace{wTest1, wTest2}, nil, nil, "")
	defer psTestCleanup()

	t.Run("with workspaces provided", func(t *testing.T) {
		err := client.PolicySets.RemoveWorkspaces(
			ctx,
			psTest.ID,
			PolicySetRemoveWorkspacesOptions{
				Workspaces: []*Workspace{wTest1, wTest2},
			},
		)
		require.NoError(t, err)

		ps, err := client.PolicySets.Read(ctx, psTest.ID)
		require.NoError(t, err)

		assert.Equal(t, 0, ps.WorkspaceCount)
		assert.Empty(t, ps.Workspaces)
	})

	t.Run("without workspaces provided", func(t *testing.T) {
		err := client.PolicySets.RemoveWorkspaces(
			ctx,
			psTest.ID,
			PolicySetRemoveWorkspacesOptions{},
		)
		assert.Equal(t, err, ErrWorkspacesRequired)
	})

	t.Run("with empty workspaces slice", func(t *testing.T) {
		err := client.PolicySets.RemoveWorkspaces(
			ctx,
			psTest.ID,
			PolicySetRemoveWorkspacesOptions{Workspaces: []*Workspace{}},
		)
		assert.Equal(t, err, ErrWorkspaceMinLimit)
	})

	t.Run("without a valid ID", func(t *testing.T) {
		err := client.PolicySets.RemoveWorkspaces(
			ctx,
			badIdentifier,
			PolicySetRemoveWorkspacesOptions{
				Workspaces: []*Workspace{wTest1, wTest2},
			},
		)
		assert.Equal(t, err, ErrInvalidPolicySetID)
	})
}

func TestPolicySetsAddWorkspaceExclusions(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	defer orgTestCleanup()

	upgradeOrganizationSubscription(t, client, orgTest)

	wTest1, wTestCleanup1 := createWorkspace(t, client, orgTest)
	defer wTestCleanup1()
	wTest2, wTestCleanup2 := createWorkspace(t, client, orgTest)
	defer wTestCleanup2()
	psTest, psTestCleanup := createPolicySet(t, client, orgTest, nil, nil, nil, nil, "")
	defer psTestCleanup()

	t.Run("with workspace exclusions provided", func(t *testing.T) {
		err := client.PolicySets.AddWorkspaceExclusions(
			ctx,
			psTest.ID,
			PolicySetAddWorkspaceExclusionsOptions{
				WorkspaceExclusions: []*Workspace{wTest1, wTest2},
			},
		)
		require.NoError(t, err)

		ps, err := client.PolicySets.Read(ctx, psTest.ID)
		require.NoError(t, err)
		assert.Equal(t, 2, len(ps.WorkspaceExclusions))

		ids := []string{}
		for _, ws := range ps.WorkspaceExclusions {
			ids = append(ids, ws.ID)
		}

		assert.Contains(t, ids, wTest1.ID)
		assert.Contains(t, ids, wTest2.ID)
	})

	t.Run("without workspace exclusions provided", func(t *testing.T) {
		err := client.PolicySets.AddWorkspaceExclusions(
			ctx,
			psTest.ID,
			PolicySetAddWorkspaceExclusionsOptions{},
		)
		assert.Equal(t, err, ErrWorkspacesRequired)
	})

	t.Run("with empty workspace exclusions slice", func(t *testing.T) {
		err := client.PolicySets.AddWorkspaceExclusions(
			ctx,
			psTest.ID,
			PolicySetAddWorkspaceExclusionsOptions{WorkspaceExclusions: []*Workspace{}},
		)
		assert.Equal(t, err, ErrWorkspaceMinLimit)
	})

	t.Run("without a valid ID", func(t *testing.T) {
		err := client.PolicySets.AddWorkspaceExclusions(
			ctx,
			badIdentifier,
			PolicySetAddWorkspaceExclusionsOptions{
				WorkspaceExclusions: []*Workspace{wTest1, wTest2},
			},
		)
		assert.Equal(t, err, ErrInvalidPolicySetID)
	})
}

func TestPolicySetsRemoveWorkspaceExclusions(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	defer orgTestCleanup()

	upgradeOrganizationSubscription(t, client, orgTest)

	wTest1, wTestCleanup1 := createWorkspace(t, client, orgTest)
	defer wTestCleanup1()
	wTest2, wTestCleanup2 := createWorkspace(t, client, orgTest)
	defer wTestCleanup2()
	psTest, psTestCleanup := createPolicySet(t, client, orgTest, nil, nil, []*Workspace{wTest1, wTest2}, nil, "")
	defer psTestCleanup()

	t.Run("with workspace exclusions provided", func(t *testing.T) {
		err := client.PolicySets.RemoveWorkspaceExclusions(
			ctx,
			psTest.ID,
			PolicySetRemoveWorkspaceExclusionsOptions{
				WorkspaceExclusions: []*Workspace{wTest1, wTest2},
			},
		)
		require.NoError(t, err)

		ps, err := client.PolicySets.Read(ctx, psTest.ID)
		require.NoError(t, err)

		assert.Equal(t, 0, len(ps.WorkspaceExclusions))
		assert.Empty(t, ps.WorkspaceExclusions)
	})

	t.Run("without workspaces provided", func(t *testing.T) {
		err := client.PolicySets.RemoveWorkspaceExclusions(
			ctx,
			psTest.ID,
			PolicySetRemoveWorkspaceExclusionsOptions{},
		)
		assert.Equal(t, err, ErrWorkspacesRequired)
	})

	t.Run("with empty workspaces slice", func(t *testing.T) {
		err := client.PolicySets.RemoveWorkspaceExclusions(
			ctx,
			psTest.ID,
			PolicySetRemoveWorkspaceExclusionsOptions{WorkspaceExclusions: []*Workspace{}},
		)
		assert.Equal(t, err, ErrWorkspaceMinLimit)
	})

	t.Run("without a valid ID", func(t *testing.T) {
		err := client.PolicySets.RemoveWorkspaceExclusions(
			ctx,
			badIdentifier,
			PolicySetRemoveWorkspaceExclusionsOptions{
				WorkspaceExclusions: []*Workspace{wTest1, wTest2},
			},
		)
		assert.Equal(t, err, ErrInvalidPolicySetID)
	})
}

func TestPolicySetsAddProjects(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	defer orgTestCleanup()

	upgradeOrganizationSubscription(t, client, orgTest)

	pTest1, pTestCleanup1 := createProject(t, client, orgTest)
	defer pTestCleanup1()
	pTest2, pTestCleanup2 := createProject(t, client, orgTest)
	defer pTestCleanup2()
	psTest, psTestCleanup := createPolicySet(t, client, orgTest, nil, nil, nil, nil, "")
	defer psTestCleanup()

	t.Run("with projects provided", func(t *testing.T) {
		err := client.PolicySets.AddProjects(
			ctx,
			psTest.ID,
			PolicySetAddProjectsOptions{
				Projects: []*Project{pTest1, pTest2},
			},
		)
		require.NoError(t, err)

		ps, err := client.PolicySets.Read(ctx, psTest.ID)
		require.NoError(t, err)
		assert.Equal(t, 2, ps.ProjectCount)

		ids := []string{}
		for _, ws := range ps.Projects {
			ids = append(ids, ws.ID)
		}

		assert.Contains(t, ids, pTest1.ID)
		assert.Contains(t, ids, pTest2.ID)
	})

	t.Run("without projects provided", func(t *testing.T) {
		err := client.PolicySets.AddProjects(
			ctx,
			psTest.ID,
			PolicySetAddProjectsOptions{},
		)
		assert.Equal(t, err, ErrRequiredProject)
	})

	t.Run("with empty projects slice", func(t *testing.T) {
		err := client.PolicySets.AddProjects(
			ctx,
			psTest.ID,
			PolicySetAddProjectsOptions{Projects: []*Project{}},
		)
		assert.Equal(t, err, ErrProjectMinLimit)
	})

	t.Run("without a valid ID", func(t *testing.T) {
		err := client.PolicySets.AddProjects(
			ctx,
			badIdentifier,
			PolicySetAddProjectsOptions{
				Projects: []*Project{pTest1, pTest2},
			},
		)
		assert.Equal(t, err, ErrInvalidPolicySetID)
	})
}

func TestPolicySetsRemoveProjects(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	defer orgTestCleanup()

	upgradeOrganizationSubscription(t, client, orgTest)

	pTest1, pTestCleanup1 := createProject(t, client, orgTest)
	defer pTestCleanup1()
	pTest2, pTestCleanup2 := createProject(t, client, orgTest)
	defer pTestCleanup2()
	psTest, psTestCleanup := createPolicySet(t, client, orgTest, nil, nil, nil, []*Project{pTest1, pTest2}, "")
	defer psTestCleanup()

	t.Run("with projects provided", func(t *testing.T) {
		err := client.PolicySets.RemoveProjects(
			ctx,
			psTest.ID,
			PolicySetRemoveProjectsOptions{
				Projects: []*Project{pTest1, pTest2},
			},
		)
		require.NoError(t, err)

		ps, err := client.PolicySets.Read(ctx, psTest.ID)
		require.NoError(t, err)

		assert.Equal(t, 0, ps.ProjectCount)
		assert.Empty(t, ps.Projects)
	})

	t.Run("without projects provided", func(t *testing.T) {
		err := client.PolicySets.RemoveProjects(
			ctx,
			psTest.ID,
			PolicySetRemoveProjectsOptions{},
		)
		assert.Equal(t, err, ErrRequiredProject)
	})

	t.Run("with empty projects slice", func(t *testing.T) {
		err := client.PolicySets.RemoveProjects(
			ctx,
			psTest.ID,
			PolicySetRemoveProjectsOptions{Projects: []*Project{}},
		)
		assert.Equal(t, err, ErrProjectMinLimit)
	})

	t.Run("without a valid ID", func(t *testing.T) {
		err := client.PolicySets.RemoveProjects(
			ctx,
			badIdentifier,
			PolicySetRemoveProjectsOptions{
				Projects: []*Project{pTest1, pTest2},
			},
		)
		assert.Equal(t, err, ErrInvalidPolicySetID)
	})
}

func TestPolicySetsDelete(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	defer orgTestCleanup()

	upgradeOrganizationSubscription(t, client, orgTest)

	psTest, _ := createPolicySet(t, client, orgTest, nil, nil, nil, nil, "")

	t.Run("with valid options", func(t *testing.T) {
		err := client.PolicySets.Delete(ctx, psTest.ID)
		require.NoError(t, err)

		// Try loading the policy - it should fail.
		_, err = client.PolicySets.Read(ctx, psTest.ID)
		assert.Equal(t, err, ErrResourceNotFound)
	})

	t.Run("when the policy does not exist", func(t *testing.T) {
		err := client.PolicySets.Delete(ctx, psTest.ID)
		assert.Equal(t, err, ErrResourceNotFound)
	})

	t.Run("when the policy ID is invalid", func(t *testing.T) {
		err := client.PolicySets.Delete(ctx, badIdentifier)
		assert.Equal(t, err, ErrInvalidPolicySetID)
	})
}
