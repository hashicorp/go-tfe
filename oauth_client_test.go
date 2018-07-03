package tfe

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestOAuthClientCreate(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgwTestCleanup := createOrganization(t, client)
	defer orgwTestCleanup()

	t.Run("with valid options", func(t *testing.T) {
		options := OAuthClientCreateOptions{
			APIURL:          String("https://api.github.com"),
			HTTPURL:         String("https://github.com"),
			Key:             String("26960d75bc03e5535757"),
			Secret:          String("a5f32ed18aa9fe251052c8fa98d04570a1515466"),
			ServiceProvider: ServiceProvider(ServiceProviderGithub),
		}

		oc, err := client.OAuthClients.Create(ctx, orgTest.Name, options)
		assert.NoError(t, err)
		assert.NotEmpty(t, oc.ID)
		assert.Equal(t, "https://api.github.com", oc.APIURL)
		assert.Equal(t, "https://github.com", oc.HTTPURL)
		assert.Equal(t, "26960d75bc03e5535757", oc.Key)
		assert.Equal(t, ServiceProviderGithub, oc.ServiceProvider)

		t.Run("the organization relationship is decoded correcly", func(t *testing.T) {
			assert.NotEmpty(t, oc.Organization)
		})
	})

	t.Run("without an valid organization", func(t *testing.T) {
		options := OAuthClientCreateOptions{
			APIURL:          String("https://api.github.com"),
			HTTPURL:         String("https://github.com"),
			Key:             String("26960d75bc03e5535757"),
			Secret:          String("a5f32ed18aa9fe251052c8fa98d04570a1515466"),
			ServiceProvider: ServiceProvider(ServiceProviderGithub),
		}

		_, err := client.OAuthClients.Create(ctx, badIdentifier, options)
		assert.EqualError(t, err, "Invalid value for organization")
	})

	t.Run("without an API URL", func(t *testing.T) {
		options := OAuthClientCreateOptions{
			HTTPURL:         String("https://github.com"),
			Key:             String("26960d75bc03e5535757"),
			Secret:          String("a5f32ed18aa9fe251052c8fa98d04570a1515466"),
			ServiceProvider: ServiceProvider(ServiceProviderGithub),
		}

		_, err := client.OAuthClients.Create(ctx, orgTest.Name, options)
		assert.EqualError(t, err, "APIURL is required")
	})

	t.Run("without a HTTP URL", func(t *testing.T) {
		options := OAuthClientCreateOptions{
			APIURL:          String("https://api.github.com"),
			Key:             String("26960d75bc03e5535757"),
			Secret:          String("a5f32ed18aa9fe251052c8fa98d04570a1515466"),
			ServiceProvider: ServiceProvider(ServiceProviderGithub),
		}

		_, err := client.OAuthClients.Create(ctx, orgTest.Name, options)
		assert.EqualError(t, err, "HTTPURL is required")
	})

	t.Run("without an key", func(t *testing.T) {
		options := OAuthClientCreateOptions{
			APIURL:          String("https://api.github.com"),
			HTTPURL:         String("https://github.com"),
			Secret:          String("a5f32ed18aa9fe251052c8fa98d04570a1515466"),
			ServiceProvider: ServiceProvider(ServiceProviderGithub),
		}

		_, err := client.OAuthClients.Create(ctx, orgTest.Name, options)
		assert.EqualError(t, err, "Key is required")
	})

	t.Run("without an secret", func(t *testing.T) {
		options := OAuthClientCreateOptions{
			APIURL:          String("https://api.github.com"),
			HTTPURL:         String("https://github.com"),
			Key:             String("26960d75bc03e5535757"),
			ServiceProvider: ServiceProvider(ServiceProviderGithub),
		}

		_, err := client.OAuthClients.Create(ctx, orgTest.Name, options)
		assert.EqualError(t, err, "Secret is required")
	})

	t.Run("without a service provider", func(t *testing.T) {
		options := OAuthClientCreateOptions{
			APIURL:  String("https://api.github.com"),
			HTTPURL: String("https://github.com"),
			Key:     String("26960d75bc03e5535757"),
			Secret:  String("a5f32ed18aa9fe251052c8fa98d04570a1515466"),
		}

		_, err := client.OAuthClients.Create(ctx, orgTest.Name, options)
		assert.EqualError(t, err, "ServiceProvider is required")
	})
}
