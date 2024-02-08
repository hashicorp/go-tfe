// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package tfe

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOAuthClientsList(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	defer orgTestCleanup()

	ocTest1, ocTestCleanup1 := createOAuthClient(t, client, orgTest, nil)
	defer ocTestCleanup1()
	ocTest2, ocTestCleanup2 := createOAuthClient(t, client, orgTest, nil)
	defer ocTestCleanup2()

	t.Run("without list options", func(t *testing.T) {
		ocl, err := client.OAuthClients.List(ctx, orgTest.Name, nil)
		require.NoError(t, err)

		t.Run("the OAuth tokens relationship is decoded correcly", func(t *testing.T) {
			for _, oc := range ocl.Items {
				assert.Equal(t, 1, len(oc.OAuthTokens))
			}
		})

		// We need to strip some fields before the next test.
		for _, oc := range append(ocl.Items, ocTest1, ocTest2) {
			oc.OAuthTokens = nil
			oc.Organization = nil
		}

		assert.Contains(t, ocl.Items, ocTest1)
		assert.Contains(t, ocl.Items, ocTest2)

		t.Skip("paging not supported yet in API")
		assert.Equal(t, 1, ocl.CurrentPage)
		assert.Equal(t, 2, ocl.TotalCount)
	})

	t.Run("with list options", func(t *testing.T) {
		t.Skip("paging not supported yet in API")
		// Request a page number which is out of range. The result should
		// be successful, but return no results if the paging options are
		// properly passed along.
		options := &OAuthClientListOptions{
			ListOptions: ListOptions{
				PageNumber: 999,
				PageSize:   100,
			},
		}

		ocl, err := client.OAuthClients.List(ctx, orgTest.Name, options)
		require.NoError(t, err)
		assert.Empty(t, ocl.Items)
		assert.Equal(t, 999, ocl.CurrentPage)
		assert.Equal(t, 2, ocl.TotalCount)
	})

	t.Run("with Include options", func(t *testing.T) {
		ocl, err := client.OAuthClients.List(ctx, orgTest.Name, &OAuthClientListOptions{
			Include: []OAuthClientIncludeOpt{OauthClientOauthTokens},
		})
		require.NoError(t, err)
		require.NotEmpty(t, ocl.Items)
		require.NotNil(t, ocl.Items[0])
		require.NotEmpty(t, ocl.Items[0].OAuthTokens)
		assert.NotEmpty(t, ocl.Items[0].OAuthTokens[0].ID)
	})

	t.Run("without a valid organization", func(t *testing.T) {
		ocl, err := client.OAuthClients.List(ctx, badIdentifier, nil)
		assert.Nil(t, ocl)
		assert.EqualError(t, err, ErrInvalidOrg.Error())
	})
}

func TestOAuthClientsCreate(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	githubToken := os.Getenv("OAUTH_CLIENT_GITHUB_TOKEN")
	if githubToken == "" {
		t.Skip("Export a valid OAUTH_CLIENT_GITHUB_TOKEN before running this test!")
	}

	orgTest, orgTestCleanup := createOrganization(t, client)
	defer orgTestCleanup()

	t.Run("with valid options", func(t *testing.T) {
		options := OAuthClientCreateOptions{
			APIURL:          String("https://api.github.com"),
			HTTPURL:         String("https://github.com"),
			OAuthToken:      String(githubToken),
			ServiceProvider: ServiceProvider(ServiceProviderGithub),
		}

		oc, err := client.OAuthClients.Create(ctx, orgTest.Name, options)
		require.NoError(t, err)
		assert.NotEmpty(t, oc.ID)
		assert.Nil(t, oc.Name)
		assert.Equal(t, "https://api.github.com", oc.APIURL)
		assert.Equal(t, "https://github.com", oc.HTTPURL)
		assert.Equal(t, 1, len(oc.OAuthTokens))
		assert.Equal(t, ServiceProviderGithub, oc.ServiceProvider)

		t.Run("the organization relationship is decoded correctly", func(t *testing.T) {
			assert.NotEmpty(t, oc.Organization)
		})
	})

	t.Run("without an valid organization", func(t *testing.T) {
		options := OAuthClientCreateOptions{
			APIURL:          String("https://api.github.com"),
			HTTPURL:         String("https://github.com"),
			OAuthToken:      String(githubToken),
			ServiceProvider: ServiceProvider(ServiceProviderGithub),
		}

		_, err := client.OAuthClients.Create(ctx, badIdentifier, options)
		assert.EqualError(t, err, ErrInvalidOrg.Error())
	})

	t.Run("without an API URL", func(t *testing.T) {
		options := OAuthClientCreateOptions{
			HTTPURL:         String("https://github.com"),
			OAuthToken:      String(githubToken),
			ServiceProvider: ServiceProvider(ServiceProviderGithub),
		}

		_, err := client.OAuthClients.Create(ctx, orgTest.Name, options)
		assert.Equal(t, err, ErrRequiredAPIURL)
	})

	t.Run("without a HTTP URL", func(t *testing.T) {
		options := OAuthClientCreateOptions{
			APIURL:          String("https://api.github.com"),
			OAuthToken:      String(githubToken),
			ServiceProvider: ServiceProvider(ServiceProviderGithub),
		}

		_, err := client.OAuthClients.Create(ctx, orgTest.Name, options)
		assert.Equal(t, err, ErrRequiredHTTPURL)
	})

	t.Run("without an OAuth token", func(t *testing.T) {
		options := OAuthClientCreateOptions{
			APIURL:          String("https://api.github.com"),
			HTTPURL:         String("https://github.com"),
			ServiceProvider: ServiceProvider(ServiceProviderGithub),
		}

		_, err := client.OAuthClients.Create(ctx, orgTest.Name, options)
		assert.Equal(t, err, ErrRequiredOauthToken)
	})

	t.Run("without a service provider", func(t *testing.T) {
		options := OAuthClientCreateOptions{
			APIURL:     String("https://api.github.com"),
			HTTPURL:    String("https://github.com"),
			OAuthToken: String(githubToken),
		}

		_, err := client.OAuthClients.Create(ctx, orgTest.Name, options)
		assert.Equal(t, err, ErrRequiredServiceProvider)
	})

	t.Run("with projects provided", func(t *testing.T) {
		skipUnlessBeta(t)
		prjTest, prjTestCleanup := createProject(t, client, orgTest)
		defer prjTestCleanup()

		options := OAuthClientCreateOptions{
			Name:     String("project-oauth-client"),
			Projects: []*Project{prjTest},
		}

		ps, err := client.OAuthClients.Create(ctx, orgTest.Name, options)
		require.NoError(t, err)

		assert.Equal(t, ps.Name, *options.Name)
		assert.Equal(t, len(ps.Projects), 1)
		assert.Equal(t, ps.Projects[0].ID, prjTest.ID)
	})
}

func TestOAuthClientsCreate_rsaKeyPair(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	defer orgTestCleanup()

	t.Run("with key, rsa public/private key options", func(t *testing.T) {
		key := randomString(t)
		options := OAuthClientCreateOptions{
			APIURL:          String("https://bbs.com"),
			HTTPURL:         String("https://bbs.com"),
			ServiceProvider: ServiceProvider(ServiceProviderBitbucketServer),
			Key:             String(key),
			Secret:          String(privateKey),
			RSAPublicKey:    String(publicKey),
		}

		oc, err := client.OAuthClients.Create(ctx, orgTest.Name, options)
		require.NoError(t, err)
		assert.NotEmpty(t, oc.ID)
		assert.Equal(t, "https://bbs.com", oc.APIURL)
		assert.Equal(t, "https://bbs.com", oc.HTTPURL)
		assert.Equal(t, ServiceProviderBitbucketServer, oc.ServiceProvider)
		assert.Equal(t, publicKey, oc.RSAPublicKey)
		assert.Equal(t, key, oc.Key)
	})
}

func TestOAuthClientsCreate_agentPool(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	githubToken := os.Getenv("OAUTH_CLIENT_GITHUB_TOKEN")
	if githubToken == "" {
		t.Skip("Export a valid OAUTH_CLIENT_GITHUB_TOKEN before running this test!")
	}

	orgTest, orgTestCleanup := createOrganization(t, client)
	defer orgTestCleanup()
	agentPoolTest, agentPoolCleanup := createAgentPool(t, client, orgTest)
	defer agentPoolCleanup()

	t.Run("with valid agent pool external id", func(t *testing.T) {
		t.Skip()
		orgTest, errOrg := client.Organizations.Read(ctx, "xxxxx")
		require.NoError(t, errOrg)
		agentPoolTest, errAgentPool := client.AgentPools.Read(ctx, "xxxxx")
		require.NoError(t, errAgentPool)
		options := OAuthClientCreateOptions{
			APIURL:          String("https://githubenterprise.xxxxx"),
			HTTPURL:         String("https://githubenterprise.xxxxx"),
			OAuthToken:      String(githubToken),
			ServiceProvider: ServiceProvider(ServiceProviderGithubEE),
			AgentPool:       agentPoolTest,
		}
		oc, errCreate := client.OAuthClients.Create(ctx, orgTest.Name, options)
		require.NoError(t, errCreate)
		assert.NotEmpty(t, oc.ID)
		assert.Equal(t, "https://githubenterprise.xxxxx", oc.APIURL)
		assert.Equal(t, "https://githubenterprise.xxxxx", oc.HTTPURL)
		assert.Equal(t, 1, len(oc.OAuthTokens))
		assert.Equal(t, ServiceProviderGithubEE, oc.ServiceProvider)
		assert.Equal(t, agentPoolTest.ID, oc.AgentPool.ID)
	})

	t.Run("with an invalid agent pool", func(t *testing.T) {
		agentPoolID := agentPoolTest.ID
		agentPoolTest.ID = badIdentifier
		options := OAuthClientCreateOptions{
			APIURL:          String("https://githubenterprise.xxxxx"),
			HTTPURL:         String("https://githubenterprise.xxxxx"),
			OAuthToken:      String(githubToken),
			ServiceProvider: ServiceProvider(ServiceProviderGithubEE),
			AgentPool:       agentPoolTest,
		}
		_, errCreate := client.OAuthClients.Create(ctx, orgTest.Name, options)
		require.Error(t, errCreate)
		assert.Contains(t, errCreate.Error(), "the provided agent pool does not exist or you are not authorized to use it")
		agentPoolTest.ID = agentPoolID
	})

	t.Run("with no agents connected", func(t *testing.T) {
		options := OAuthClientCreateOptions{
			APIURL:          String("https://githubenterprise.xxxxx"),
			HTTPURL:         String("https://githubenterprise.xxxxx"),
			OAuthToken:      String(githubToken),
			ServiceProvider: ServiceProvider(ServiceProviderGithubEE),
			AgentPool:       agentPoolTest,
		}
		_, errCreate := client.OAuthClients.Create(ctx, orgTest.Name, options)
		assert.Contains(t, errCreate.Error(), "the organization does not have private VCS enabled")
		require.Error(t, errCreate)
	})
}

func TestOAuthClientsRead(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	ocTest, ocTestCleanup := createOAuthClient(t, client, nil, nil)
	defer ocTestCleanup()

	t.Run("when the OAuth client exists", func(t *testing.T) {
		oc, err := client.OAuthClients.Read(ctx, ocTest.ID)
		require.NoError(t, err)
		assert.Equal(t, ocTest.ID, oc.ID)
		assert.Equal(t, ocTest.APIURL, oc.APIURL)
		assert.Equal(t, ocTest.CallbackURL, oc.CallbackURL)
		assert.Equal(t, ocTest.ConnectPath, oc.ConnectPath)
		assert.Equal(t, ocTest.HTTPURL, oc.HTTPURL)
		assert.Equal(t, ocTest.ServiceProvider, oc.ServiceProvider)
		assert.Equal(t, ocTest.ServiceProviderName, oc.ServiceProviderName)
		assert.Equal(t, ocTest.OAuthTokens, oc.OAuthTokens)
		assert.Equal(t, ocTest.OrganizationScoped, oc.OrganizationScoped)
	})

	t.Run("when the OAuth client does not exist", func(t *testing.T) {
		oc, err := client.OAuthClients.Read(ctx, "nonexisting")
		assert.Nil(t, oc)
		assert.Equal(t, ErrResourceNotFound, err)
	})

	t.Run("without a valid OAuth client ID", func(t *testing.T) {
		oc, err := client.OAuthClients.Read(ctx, badIdentifier)
		assert.Nil(t, oc)
		assert.Equal(t, err, ErrInvalidOauthClientID)
	})
}

func TestOAuthClientsReadWithOptions(t *testing.T) {
	skipUnlessBeta(t)
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	defer orgTestCleanup()

	pj, pjCleanup := createProject(t, client, orgTest)
	defer pjCleanup()

	ocTest, ocTestCleanup := createOAuthClient(t, client, nil, []*Project{pj})
	defer ocTestCleanup()

	opts := &OAuthClientReadOptions{
		Include: []OAuthClientIncludeOpt{OauthClientProjects},
	}
	t.Run("when the OAuth client exists", func(t *testing.T) {
		ocWithOptions, err := client.OAuthClients.ReadWithOptions(ctx, ocTest.ID, opts)
		require.NoError(t, err)

		assert.Equal(t, ocTest.Projects, ocWithOptions.Projects)
	})
}

func TestOAuthClientsDelete(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	defer orgTestCleanup()

	ocTest, _ := createOAuthClient(t, client, orgTest, nil)

	t.Run("with valid options", func(t *testing.T) {
		err := client.OAuthClients.Delete(ctx, ocTest.ID)
		require.NoError(t, err)

		_, err = retry(func() (interface{}, error) {
			c, err := client.OAuthClients.Read(ctx, ocTest.ID)
			if err != ErrResourceNotFound {
				return nil, fmt.Errorf("expected %s, but err was %s", ErrResourceNotFound, err)
			}
			return c, err
		})

		assert.Equal(t, err, ErrResourceNotFound)
	})

	t.Run("when the OAuth client does not exist", func(t *testing.T) {
		err := client.OAuthClients.Delete(ctx, ocTest.ID)
		assert.Equal(t, err, ErrResourceNotFound)
	})

	t.Run("when the OAuth client ID is invalid", func(t *testing.T) {
		err := client.OAuthClients.Delete(ctx, badIdentifier)
		assert.Equal(t, err, ErrInvalidOauthClientID)
	})
}

func TestOAuthClientsCreateOptionsValid(t *testing.T) {
	t.Run("with valid options", func(t *testing.T) {
		options := OAuthClientCreateOptions{
			APIURL:          String("https://api.github.com"),
			HTTPURL:         String("https://github.com"),
			OAuthToken:      String("NOTHING"),
			ServiceProvider: ServiceProvider(ServiceProviderGithub),
		}

		err := options.valid()
		assert.Nil(t, err)
	})

	t.Run("without an API URL", func(t *testing.T) {
		options := OAuthClientCreateOptions{
			HTTPURL:         String("https://github.com"),
			OAuthToken:      String("NOTHING"),
			ServiceProvider: ServiceProvider(ServiceProviderGithub),
		}

		err := options.valid()
		assert.Equal(t, err, ErrRequiredAPIURL)
	})

	t.Run("without a HTTP URL", func(t *testing.T) {
		options := OAuthClientCreateOptions{
			APIURL:          String("https://api.github.com"),
			OAuthToken:      String("NOTHING"),
			ServiceProvider: ServiceProvider(ServiceProviderGithub),
		}

		err := options.valid()
		assert.Equal(t, err, ErrRequiredHTTPURL)
	})

	t.Run("without an OAuth token", func(t *testing.T) {
		options := OAuthClientCreateOptions{
			APIURL:          String("https://api.github.com"),
			HTTPURL:         String("https://github.com"),
			ServiceProvider: ServiceProvider(ServiceProviderGithub),
		}

		err := options.valid()
		assert.Equal(t, err, ErrRequiredOauthToken)
	})

	t.Run("without a service provider", func(t *testing.T) {
		options := OAuthClientCreateOptions{
			APIURL:     String("https://api.github.com"),
			HTTPURL:    String("https://github.com"),
			OAuthToken: String("NOTHING"),
		}

		err := options.valid()
		assert.Equal(t, err, ErrRequiredServiceProvider)
	})

	t.Run("without private key and not ado_server options", func(t *testing.T) {
		options := OAuthClientCreateOptions{
			APIURL:          String("https://api.github.com"),
			HTTPURL:         String("https://github.com"),
			OAuthToken:      String("NOTHING"),
			ServiceProvider: ServiceProvider(ServiceProviderGitlabEE),
		}

		err := options.valid()
		assert.Nil(t, err)
	})

	t.Run("with empty private key and not ado_server options", func(t *testing.T) {
		options := OAuthClientCreateOptions{
			APIURL:          String("https://api.github.com"),
			HTTPURL:         String("https://github.com"),
			OAuthToken:      String("NOTHING"),
			ServiceProvider: ServiceProvider(ServiceProviderGitlabEE),
			PrivateKey:      String(""),
		}

		err := options.valid()
		assert.Nil(t, err)
	})

	t.Run("with private key and not ado_server options", func(t *testing.T) {
		options := OAuthClientCreateOptions{
			APIURL:          String("https://api.github.com"),
			HTTPURL:         String("https://github.com"),
			OAuthToken:      String("NOTHING"),
			ServiceProvider: ServiceProvider(ServiceProviderGithub),
			PrivateKey:      String("NOTHING"),
		}

		err := options.valid()
		assert.Equal(t, err, ErrUnsupportedPrivateKey)
	})

	t.Run("with valid options including private key", func(t *testing.T) {
		options := OAuthClientCreateOptions{
			APIURL:          String("https://ado.example.com"),
			HTTPURL:         String("https://ado.example.com"),
			OAuthToken:      String("NOTHING"),
			ServiceProvider: ServiceProvider(ServiceProviderAzureDevOpsServer),
			PrivateKey:      String("NOTHING"),
		}

		err := options.valid()
		assert.Nil(t, err)
	})
}

func TestOAuthClientsAddProjects(t *testing.T) {
	skipUnlessBeta(t)
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	defer orgTestCleanup()

	upgradeOrganizationSubscription(t, client, orgTest)

	pTest1, pTestCleanup1 := createProject(t, client, orgTest)
	defer pTestCleanup1()
	pTest2, pTestCleanup2 := createProject(t, client, orgTest)
	defer pTestCleanup2()
	psTest, psTestCleanup := createOAuthClient(t, client, orgTest, nil)
	defer psTestCleanup()

	t.Run("with projects provided", func(t *testing.T) {
		err := client.OAuthClients.AddProjects(
			ctx,
			psTest.ID,
			OAuthClientAddProjectsOptions{
				Projects: []*Project{pTest1, pTest2},
			},
		)
		require.NoError(t, err)

		ps, err := client.OAuthClients.Read(ctx, psTest.ID)
		require.NoError(t, err)
		assert.Equal(t, 2, len(ps.Projects))

		var ids []string
		for _, pj := range ps.Projects {
			ids = append(ids, pj.ID)
		}

		assert.Contains(t, ids, pTest1.ID)
		assert.Contains(t, ids, pTest2.ID)
	})

	t.Run("without projects provided", func(t *testing.T) {
		err := client.OAuthClients.AddProjects(
			ctx,
			psTest.ID,
			OAuthClientAddProjectsOptions{},
		)
		assert.Equal(t, err, ErrRequiredProject)
	})

	t.Run("with empty projects slice", func(t *testing.T) {
		err := client.OAuthClients.AddProjects(
			ctx,
			psTest.ID,
			OAuthClientAddProjectsOptions{Projects: []*Project{}},
		)
		assert.Equal(t, err, ErrProjectMinLimit)
	})

	t.Run("without a valid ID", func(t *testing.T) {
		err := client.OAuthClients.AddProjects(
			ctx,
			badIdentifier,
			OAuthClientAddProjectsOptions{
				Projects: []*Project{pTest1, pTest2},
			},
		)
		assert.Equal(t, err, ErrInvalidOauthClientID)
	})
}

func TestOAuthClientsRemoveProjects(t *testing.T) {
	skipUnlessBeta(t)
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	defer orgTestCleanup()

	upgradeOrganizationSubscription(t, client, orgTest)

	pTest1, pTestCleanup1 := createProject(t, client, orgTest)
	defer pTestCleanup1()
	pTest2, pTestCleanup2 := createProject(t, client, orgTest)
	defer pTestCleanup2()
	psTest, psTestCleanup := createOAuthClient(t, client, orgTest, []*Project{pTest1, pTest2})
	defer psTestCleanup()

	t.Run("with projects provided", func(t *testing.T) {
		err := client.OAuthClients.RemoveProjects(
			ctx,
			psTest.ID,
			OAuthClientRemoveProjectsOptions{
				Projects: []*Project{pTest1, pTest2},
			},
		)
		require.NoError(t, err)

		ps, err := client.OAuthClients.Read(ctx, psTest.ID)
		require.NoError(t, err)

		assert.Equal(t, 0, len(ps.Projects))
		assert.Empty(t, ps.Projects)
	})

	t.Run("without projects provided", func(t *testing.T) {
		err := client.OAuthClients.RemoveProjects(
			ctx,
			psTest.ID,
			OAuthClientRemoveProjectsOptions{},
		)
		assert.Equal(t, err, ErrRequiredProject)
	})

	t.Run("with empty projects slice", func(t *testing.T) {
		err := client.OAuthClients.RemoveProjects(
			ctx,
			psTest.ID,
			OAuthClientRemoveProjectsOptions{Projects: []*Project{}},
		)
		assert.Equal(t, err, ErrProjectMinLimit)
	})

	t.Run("without a valid ID", func(t *testing.T) {
		err := client.OAuthClients.RemoveProjects(
			ctx,
			badIdentifier,
			OAuthClientRemoveProjectsOptions{
				Projects: []*Project{pTest1, pTest2},
			},
		)
		assert.Equal(t, err, ErrInvalidOauthClientID)
	})
}

func TestOAuthClientsUpdate(t *testing.T) {
	skipUnlessBeta(t)
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	defer orgTestCleanup()

	t.Run("updates organization scoped", func(t *testing.T) {
		organizationScoped := false
		organizationScopedTrue := true
		options := OAuthClientCreateOptions{
			APIURL:             String("https://bbs.com"),
			HTTPURL:            String("https://bbs.com"),
			ServiceProvider:    ServiceProvider(ServiceProviderBitbucketServer),
			OrganizationScoped: &organizationScopedTrue,
		}

		origOC, err := client.OAuthClients.Create(ctx, orgTest.Name, options)
		require.NoError(t, err)
		assert.NotEmpty(t, origOC.ID)

		updateOpts := OAuthClientUpdateOptions{
			OrganizationScoped: &organizationScoped,
		}
		oc, err := client.OAuthClients.Update(ctx, origOC.ID, updateOpts)
		require.NoError(t, err)
		assert.NotEmpty(t, oc.ID)
		assert.NotEqual(t, origOC.OrganizationScoped, oc.OrganizationScoped)
	})
}

const publicKey = `
-----BEGIN PUBLIC KEY-----
MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAoKizy4xbN6qZFAwIJV24
-----END PUBLIC KEY-----
`

const privateKey = `
-----BEGIN RSA PRIVATE KEY-----
MIIEpAIBAAKCAQEAoKizy4xbN6qZFAwIJV24liz/vYBSvR3SjEiUzhpp0uMAmICN
-----END RSA PRIVATE KEY-----
`

func TestOAuthClientsUpdate_rsaKeyPair(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	defer orgTestCleanup()

	t.Run("updates a new key", func(t *testing.T) {
		originalKey := randomString(t)
		options := OAuthClientCreateOptions{
			APIURL:          String("https://bbs.com"),
			HTTPURL:         String("https://bbs.com"),
			ServiceProvider: ServiceProvider(ServiceProviderBitbucketServer),
			Key:             String(originalKey),
			Secret:          String(privateKey),
			RSAPublicKey:    String(publicKey),
		}

		origOC, err := client.OAuthClients.Create(ctx, orgTest.Name, options)
		require.NoError(t, err)
		assert.NotEmpty(t, origOC.ID)

		newKey := randomString(t)
		updateOpts := OAuthClientUpdateOptions{
			Key: String(newKey),
		}
		oc, err := client.OAuthClients.Update(ctx, origOC.ID, updateOpts)
		require.NoError(t, err)
		assert.NotEmpty(t, oc.ID)
		assert.Equal(t, ServiceProviderBitbucketServer, oc.ServiceProvider)
		assert.Equal(t, oc.RSAPublicKey, origOC.RSAPublicKey)
		assert.Equal(t, newKey, oc.Key)
	})

	t.Run("errors when missing key", func(t *testing.T) {
		originalKey := randomString(t)
		options := OAuthClientCreateOptions{
			APIURL:          String("https://bbs.com"),
			HTTPURL:         String("https://bbs.com"),
			ServiceProvider: ServiceProvider(ServiceProviderBitbucketServer),
			Key:             String(originalKey),
			Secret:          String(privateKey),
			RSAPublicKey:    String(publicKey),
		}

		origOC, err := client.OAuthClients.Create(ctx, orgTest.Name, options)
		require.NoError(t, err)
		assert.NotEmpty(t, origOC.ID)

		updateOpts := OAuthClientUpdateOptions{
			Key: String(""),
		}
		_, err = client.OAuthClients.Update(ctx, origOC.ID, updateOpts)
		assert.Error(t, err, "The Consumer Key for BitBucket Server must be present. Please add a value for `key`.")
	})
}
