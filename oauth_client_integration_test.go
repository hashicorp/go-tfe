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

	ocTest1, ocTestCleanup1 := createOAuthClient(t, client, orgTest)
	defer ocTestCleanup1()
	ocTest2, ocTestCleanup2 := createOAuthClient(t, client, orgTest)
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

		t.Run("the organization relationship is decoded correcly", func(t *testing.T) {
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

	agentPool, agentPoolCleanup := createAgentPool(t, client, orgTest)
	defer agentPoolCleanup()

	t.Run("with valid agent pool external id", func(t *testing.T) {
		options := OAuthClientCreateOptions{
			APIURL:          String("https://api.github.com"),
			HTTPURL:         String("https://github.com"),
			OAuthToken:      String(githubToken),
			ServiceProvider: ServiceProvider(ServiceProviderGithub),
			AgentPoolID:     String(agentPool.ID),
		}

		ocTest, errCreate := client.OAuthClients.Create(ctx, orgTest.Name, options)
		require.NoError(t, errCreate)
		errDelete := client.OAuthClients.Delete(ctx, ocTest.ID)
		require.NoError(t, errDelete)
	})

	t.Run("with invalid agent pool external id", func(t *testing.T) {
		options := OAuthClientCreateOptions{
			APIURL:          String("https://api.github.com"),
			HTTPURL:         String("https://github.com"),
			OAuthToken:      String(githubToken),
			ServiceProvider: ServiceProvider(ServiceProviderGithub),
			AgentPoolID:     String(randomString(t)),
		}
		_, err := client.OAuthClients.Create(ctx, orgTest.Name, options)
		assert.EqualError(t, err, "unprocessable entity\n\nAgent Pool is missing")
	})
}

func TestOAuthClientsRead(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	ocTest, ocTestCleanup := createOAuthClient(t, client, nil)
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

func TestOAuthClientsDelete(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	defer orgTestCleanup()

	ocTest, _ := createOAuthClient(t, client, orgTest)

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
