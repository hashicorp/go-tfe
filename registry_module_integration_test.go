//go:build integration
// +build integration

package tfe

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"
	"testing"

	retryablehttp "github.com/hashicorp/go-retryablehttp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRegistryModulesList(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	defer orgTestCleanup()

	registryModuleTest1, registryModuleTest1Cleanup := createRegistryModule(t, client, orgTest, PrivateRegistry)
	defer registryModuleTest1Cleanup()
	registryModuleTest2, registryModuleTest2Cleanup := createRegistryModule(t, client, orgTest, PrivateRegistry)
	defer registryModuleTest2Cleanup()

	t.Run("with no list options", func(t *testing.T) {
		modl, err := client.RegistryModules.List(ctx, orgTest.Name, &RegistryModuleListOptions{})
		require.NoError(t, err)
		assert.Contains(t, modl.Items, registryModuleTest1)
		assert.Contains(t, modl.Items, registryModuleTest2)
		assert.Equal(t, 1, modl.CurrentPage)
		assert.Equal(t, 2, modl.TotalCount)
	})

	t.Run("with list options", func(t *testing.T) {
		modl, err := client.RegistryModules.List(ctx, orgTest.Name, &RegistryModuleListOptions{
			ListOptions: ListOptions{
				PageNumber: 999,
				PageSize:   100,
			},
		})
		require.NoError(t, err)
		// Out of range page number, so the items should be empty
		assert.Empty(t, modl.Items)
		assert.Equal(t, 999, modl.CurrentPage)

		modl, err = client.RegistryModules.List(ctx, orgTest.Name, &RegistryModuleListOptions{
			ListOptions: ListOptions{
				PageNumber: 1,
				PageSize:   100,
			},
		})
		require.NoError(t, err)
		assert.NotEmpty(t, modl.Items)
		assert.Equal(t, 1, modl.CurrentPage)
	})
}

func TestRegistryModulesCreate(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	defer orgTestCleanup()

	t.Run("with valid options", func(t *testing.T) {
		assertRegistryModuleAttributes := func(t *testing.T, registryModule *RegistryModule) {
			t.Run("permissions are properly decoded", func(t *testing.T) {
				require.NotEmpty(t, registryModule.Permissions)
				assert.True(t, registryModule.Permissions.CanDelete)
				assert.True(t, registryModule.Permissions.CanResync)
				assert.True(t, registryModule.Permissions.CanRetry)
			})

			t.Run("relationships are properly decoded", func(t *testing.T) {
				require.NotEmpty(t, registryModule.Organization)
				assert.Equal(t, orgTest.Name, registryModule.Organization.Name)
			})

			t.Run("timestamps are properly decoded", func(t *testing.T) {
				assert.NotEmpty(t, registryModule.CreatedAt)
				assert.NotEmpty(t, registryModule.UpdatedAt)
			})
		}

		t.Run("without RegistryName", func(t *testing.T) {
			options := RegistryModuleCreateOptions{
				Name:     String("name"),
				Provider: String("provider"),
			}
			rm, err := client.RegistryModules.Create(ctx, orgTest.Name, options)
			require.NoError(t, err)
			assert.NotEmpty(t, rm.ID)
			assert.Equal(t, *options.Name, rm.Name)
			assert.Equal(t, *options.Provider, rm.Provider)
			assert.Equal(t, PrivateRegistry, rm.RegistryName)
			assert.Equal(t, orgTest.Name, rm.Namespace)

			assertRegistryModuleAttributes(t, rm)
		})

		t.Run("with private RegistryName", func(t *testing.T) {
			options := RegistryModuleCreateOptions{
				Name:         String("another_name"),
				Provider:     String("provider"),
				RegistryName: PrivateRegistry,
			}
			rm, err := client.RegistryModules.Create(ctx, orgTest.Name, options)
			require.NoError(t, err)
			assert.NotEmpty(t, rm.ID)
			assert.Equal(t, *options.Name, rm.Name)
			assert.Equal(t, *options.Provider, rm.Provider)
			assert.Equal(t, options.RegistryName, rm.RegistryName)
			assert.Equal(t, orgTest.Name, rm.Namespace)

			assertRegistryModuleAttributes(t, rm)
		})

		t.Run("with public RegistryName", func(t *testing.T) {
			options := RegistryModuleCreateOptions{
				Name:         String("vpc"),
				Provider:     String("aws"),
				RegistryName: PublicRegistry,
				Namespace:    "terraform-aws-modules",
			}
			rm, err := client.RegistryModules.Create(ctx, orgTest.Name, options)
			require.NoError(t, err)
			assert.NotEmpty(t, rm.ID)
			assert.Equal(t, *options.Name, rm.Name)
			assert.Equal(t, *options.Provider, rm.Provider)
			assert.Equal(t, options.RegistryName, rm.RegistryName)
			assert.Equal(t, options.Namespace, rm.Namespace)

			assertRegistryModuleAttributes(t, rm)
		})

		t.Run("with no-code attribute", func(t *testing.T) {
			skipIfBeta(t)
			options := RegistryModuleCreateOptions{
				Name:         String("iam"),
				Provider:     String("aws"),
				NoCode:       true,
				RegistryName: PublicRegistry,
				Namespace:    "terraform-aws-modules",
			}
			rm, err := client.RegistryModules.Create(ctx, orgTest.Name, options)
			require.NoError(t, err)
			assert.NotEmpty(t, rm.ID)
			assert.Equal(t, *options.Name, rm.Name)
			assert.Equal(t, *options.Provider, rm.Provider)
			assert.Equal(t, options.RegistryName, rm.RegistryName)
			assert.Equal(t, options.Namespace, rm.Namespace)
			assert.Equal(t, options.NoCode, rm.NoCode)

			assertRegistryModuleAttributes(t, rm)
		})
	})

	t.Run("with invalid options", func(t *testing.T) {
		t.Run("without a name", func(t *testing.T) {
			options := RegistryModuleCreateOptions{
				Provider: String("provider"),
			}
			rm, err := client.RegistryModules.Create(ctx, orgTest.Name, options)
			assert.Nil(t, rm)
			assert.EqualError(t, err, ErrRequiredName.Error())
		})

		t.Run("with an invalid name", func(t *testing.T) {
			options := RegistryModuleCreateOptions{
				Name:     String("invalid name"),
				Provider: String("provider"),
			}
			rm, err := client.RegistryModules.Create(ctx, orgTest.Name, options)
			assert.Nil(t, rm)
			assert.EqualError(t, err, ErrInvalidName.Error())
		})

		t.Run("without a provider", func(t *testing.T) {
			options := RegistryModuleCreateOptions{
				Name: String("name"),
			}
			rm, err := client.RegistryModules.Create(ctx, orgTest.Name, options)
			assert.Nil(t, rm)
			assert.Equal(t, err, ErrRequiredProvider)
		})

		t.Run("with an invalid provider", func(t *testing.T) {
			options := RegistryModuleCreateOptions{
				Name:     String("name"),
				Provider: String("invalid provider"),
			}
			rm, err := client.RegistryModules.Create(ctx, orgTest.Name, options)
			assert.Nil(t, rm)
			assert.Equal(t, err, ErrInvalidProvider)
		})

		t.Run("with an invalid registry name", func(t *testing.T) {
			options := RegistryModuleCreateOptions{
				Name:         String("name"),
				Provider:     String("provider"),
				RegistryName: "PRIVATE",
			}
			rm, err := client.RegistryModules.Create(ctx, orgTest.Name, options)
			assert.Nil(t, rm)
			assert.Equal(t, err, ErrInvalidRegistryName)
		})

		t.Run("without a namespace for public registry name", func(t *testing.T) {
			options := RegistryModuleCreateOptions{
				Name:         String("name"),
				Provider:     String("provider"),
				RegistryName: PublicRegistry,
			}
			rm, err := client.RegistryModules.Create(ctx, orgTest.Name, options)
			assert.Nil(t, rm)
			assert.Equal(t, err, ErrRequiredNamespace)
		})

		t.Run("with a namespace for private registry name", func(t *testing.T) {
			options := RegistryModuleCreateOptions{
				Name:         String("name"),
				Provider:     String("provider"),
				RegistryName: PrivateRegistry,
				Namespace:    "namespace",
			}
			rm, err := client.RegistryModules.Create(ctx, orgTest.Name, options)
			assert.Nil(t, rm)
			assert.Equal(t, err, ErrUnsupportedBothNamespaceAndPrivateRegistryName)
		})
	})

	t.Run("without a valid organization", func(t *testing.T) {
		options := RegistryModuleCreateOptions{
			Name:     String("name"),
			Provider: String("provider"),
		}
		rm, err := client.RegistryModules.Create(ctx, badIdentifier, options)
		assert.Nil(t, rm)
		assert.EqualError(t, err, ErrInvalidOrg.Error())
	})
}

func TestRegistryModuleUpdate(t *testing.T) {
	skipIfBeta(t)
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	defer orgTestCleanup()

	options := RegistryModuleCreateOptions{
		Name:         String("vpc"),
		Provider:     String("aws"),
		RegistryName: PublicRegistry,
		Namespace:    "terraform-aws-modules",
	}
	rm, err := client.RegistryModules.Create(ctx, orgTest.Name, options)
	require.NoError(t, err)
	assert.NotEmpty(t, rm.ID)

	t.Run("enable no-code", func(t *testing.T) {
		options := RegistryModuleUpdateOptions{
			NoCode: Bool(true),
		}
		rm, err := client.RegistryModules.Update(ctx, RegistryModuleID{
			Organization: orgTest.Name,
			Name:         "vpc",
			Provider:     "aws",
			Namespace:    "terraform-aws-modules",
			RegistryName: PublicRegistry,
		}, options)
		require.NoError(t, err)
		assert.True(t, rm.NoCode)
	})

	t.Run("disable no-code", func(t *testing.T) {
		options := RegistryModuleUpdateOptions{
			NoCode: Bool(false),
		}
		rm, err := client.RegistryModules.Update(ctx, RegistryModuleID{
			Organization: orgTest.Name,
			Name:         "vpc",
			Provider:     "aws",
			Namespace:    "terraform-aws-modules",
			RegistryName: PublicRegistry,
		}, options)
		require.NoError(t, err)
		assert.False(t, rm.NoCode)
	})

}

func TestRegistryModulesCreateVersion(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	defer orgTestCleanup()

	registryModuleTest, registryModuleTestCleanup := createRegistryModule(t, client, orgTest, PrivateRegistry)
	defer registryModuleTestCleanup()

	t.Run("with valid options", func(t *testing.T) {
		options := RegistryModuleCreateVersionOptions{
			Version: String("1.2.3"),
		}

		rmv, err := client.RegistryModules.CreateVersion(ctx, RegistryModuleID{
			Organization: orgTest.Name,
			Name:         registryModuleTest.Name,
			Provider:     registryModuleTest.Provider,
		}, options)
		require.NoError(t, err)
		assert.NotEmpty(t, rmv.ID)
		assert.Equal(t, *options.Version, rmv.Version)

		t.Run("relationships are properly decoded", func(t *testing.T) {
			assert.Equal(t, registryModuleTest.ID, rmv.RegistryModule.ID)
		})

		t.Run("timestamps are properly decoded", func(t *testing.T) {
			assert.NotEmpty(t, rmv.CreatedAt)
			assert.NotEmpty(t, rmv.UpdatedAt)
		})

		t.Run("links are properly decoded", func(t *testing.T) {
			assert.NotEmpty(t, rmv.Links["upload"])
			assert.Contains(t, rmv.Links["upload"], "/object/")
		})
	})

	t.Run("with invalid options", func(t *testing.T) {
		t.Run("without version", func(t *testing.T) {
			options := RegistryModuleCreateVersionOptions{}
			rmv, err := client.RegistryModules.CreateVersion(ctx, RegistryModuleID{
				Organization: orgTest.Name,
				Name:         registryModuleTest.Name,
				Provider:     registryModuleTest.Provider,
			}, options)
			assert.Nil(t, rmv)
			assert.Equal(t, err, ErrRequiredVersion)
		})

		t.Run("with invalid version", func(t *testing.T) {
			options := RegistryModuleCreateVersionOptions{
				Version: String("invalid version"),
			}
			rmv, err := client.RegistryModules.CreateVersion(ctx, RegistryModuleID{
				Organization: orgTest.Name,
				Name:         registryModuleTest.Name,
				Provider:     registryModuleTest.Provider,
			}, options)
			assert.Nil(t, rmv)
			assert.Equal(t, err, ErrInvalidVersion)
		})
	})

	t.Run("without a name", func(t *testing.T) {
		options := RegistryModuleCreateVersionOptions{
			Version: String("1.2.3"),
		}
		rmv, err := client.RegistryModules.CreateVersion(ctx, RegistryModuleID{
			Organization: orgTest.Name,
			Name:         "",
			Provider:     registryModuleTest.Provider,
		}, options)
		assert.Nil(t, rmv)
		assert.EqualError(t, err, ErrRequiredName.Error())
	})

	t.Run("with an invalid name", func(t *testing.T) {
		options := RegistryModuleCreateVersionOptions{
			Version: String("1.2.3"),
		}
		rmv, err := client.RegistryModules.CreateVersion(ctx, RegistryModuleID{
			Organization: orgTest.Name,
			Name:         badIdentifier,
			Provider:     registryModuleTest.Provider,
		}, options)
		assert.Nil(t, rmv)
		assert.EqualError(t, err, ErrInvalidName.Error())
	})

	t.Run("without a provider", func(t *testing.T) {
		options := RegistryModuleCreateVersionOptions{
			Version: String("1.2.3"),
		}
		rmv, err := client.RegistryModules.CreateVersion(ctx, RegistryModuleID{
			Organization: orgTest.Name,
			Name:         registryModuleTest.Name,
			Provider:     "",
		}, options)
		assert.Nil(t, rmv)
		assert.Equal(t, err, ErrRequiredProvider)
	})

	t.Run("with an invalid provider", func(t *testing.T) {
		options := RegistryModuleCreateVersionOptions{
			Version: String("1.2.3"),
		}
		rmv, err := client.RegistryModules.CreateVersion(ctx, RegistryModuleID{
			Organization: orgTest.Name,
			Name:         registryModuleTest.Name,
			Provider:     badIdentifier,
		}, options)
		assert.Nil(t, rmv)
		assert.Equal(t, err, ErrInvalidProvider)
	})

	t.Run("without a valid organization", func(t *testing.T) {
		options := RegistryModuleCreateVersionOptions{
			Version: String("1.2.3"),
		}
		rmv, err := client.RegistryModules.CreateVersion(ctx, RegistryModuleID{
			Organization: badIdentifier,
			Name:         registryModuleTest.Name,
			Provider:     registryModuleTest.Provider,
		}, options)
		assert.Nil(t, rmv)
		assert.EqualError(t, err, ErrInvalidOrg.Error())
	})

}

func TestRegistryModulesCreateWithVCSConnection(t *testing.T) {
	githubIdentifier := os.Getenv("GITHUB_REGISTRY_MODULE_IDENTIFIER")
	if githubIdentifier == "" {
		t.Skip("Export a valid GITHUB_REGISTRY_MODULE_IDENTIFIER before running this test")
	}
	repositoryName := strings.Split(githubIdentifier, "/")[1]
	registryModuleProvider := strings.SplitN(repositoryName, "-", 3)[1]
	registryModuleName := strings.SplitN(repositoryName, "-", 3)[2]

	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	defer orgTestCleanup()

	oauthTokenTest, oauthTokenTestCleanup := createOAuthToken(t, client, orgTest)
	defer oauthTokenTestCleanup()

	t.Run("with valid options", func(t *testing.T) {
		options := RegistryModuleCreateWithVCSConnectionOptions{
			VCSRepo: &RegistryModuleVCSRepoOptions{
				Identifier:        String(githubIdentifier),
				OAuthTokenID:      String(oauthTokenTest.ID),
				DisplayIdentifier: String(githubIdentifier),
			},
		}
		rm, err := client.RegistryModules.CreateWithVCSConnection(ctx, options)
		require.NoError(t, err)
		assert.NotEmpty(t, rm.ID)
		assert.Equal(t, registryModuleName, rm.Name)
		assert.Equal(t, registryModuleProvider, rm.Provider)
		assert.Equal(t, rm.VCSRepo.Branch, "")
		assert.Equal(t, rm.VCSRepo.DisplayIdentifier, githubIdentifier)
		assert.Equal(t, rm.VCSRepo.Identifier, githubIdentifier)
		assert.Equal(t, rm.VCSRepo.IngressSubmodules, true)
		assert.Equal(t, rm.VCSRepo.OAuthTokenID, oauthTokenTest.ID)
		assert.Equal(t, rm.VCSRepo.RepositoryHTTPURL, fmt.Sprintf("https://github.com/%s", githubIdentifier))
		assert.Equal(t, rm.VCSRepo.ServiceProvider, string(ServiceProviderGithub))
		assert.Regexp(t, fmt.Sprintf("^%s/webhooks/vcs/[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12}$", regexp.QuoteMeta(DefaultConfig().Address)), rm.VCSRepo.WebhookURL)

		t.Run("permissions are properly decoded", func(t *testing.T) {
			assert.True(t, rm.Permissions.CanDelete)
			assert.True(t, rm.Permissions.CanResync)
			assert.True(t, rm.Permissions.CanRetry)
		})

		t.Run("relationships are properly decoded", func(t *testing.T) {
			assert.Equal(t, orgTest.Name, rm.Organization.Name)
		})

		t.Run("timestamps are properly decoded", func(t *testing.T) {
			assert.NotEmpty(t, rm.CreatedAt)
			assert.NotEmpty(t, rm.UpdatedAt)
		})
	})

	t.Run("with invalid options", func(t *testing.T) {
		t.Run("without an identifier", func(t *testing.T) {
			options := RegistryModuleCreateWithVCSConnectionOptions{
				VCSRepo: &RegistryModuleVCSRepoOptions{
					Identifier:        String(""),
					OAuthTokenID:      String(oauthTokenTest.ID),
					DisplayIdentifier: String(githubIdentifier),
				},
			}
			rm, err := client.RegistryModules.CreateWithVCSConnection(ctx, options)
			assert.Nil(t, rm)
			assert.Equal(t, err, ErrRequiredIdentifier)
		})

		t.Run("without an oauth token ID", func(t *testing.T) {
			options := RegistryModuleCreateWithVCSConnectionOptions{
				VCSRepo: &RegistryModuleVCSRepoOptions{
					Identifier:        String(githubIdentifier),
					OAuthTokenID:      String(""),
					DisplayIdentifier: String(githubIdentifier),
				},
			}
			rm, err := client.RegistryModules.CreateWithVCSConnection(ctx, options)
			assert.Nil(t, rm)
			assert.Equal(t, err, ErrRequiredOauthTokenID)
		})

		t.Run("without a display identifier", func(t *testing.T) {
			options := RegistryModuleCreateWithVCSConnectionOptions{
				VCSRepo: &RegistryModuleVCSRepoOptions{
					Identifier:        String(githubIdentifier),
					OAuthTokenID:      String(oauthTokenTest.ID),
					DisplayIdentifier: String(""),
				},
			}
			rm, err := client.RegistryModules.CreateWithVCSConnection(ctx, options)
			assert.Nil(t, rm)
			assert.Equal(t, err, ErrRequiredDisplayIdentifier)
		})
	})

	t.Run("without options", func(t *testing.T) {
		options := RegistryModuleCreateWithVCSConnectionOptions{}
		rm, err := client.RegistryModules.CreateWithVCSConnection(ctx, options)
		assert.Nil(t, rm)
		assert.Equal(t, err, ErrRequiredVCSRepo)
	})

}

func TestRegistryModulesRead(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	defer orgTestCleanup()

	registryModuleTest, registryModuleTestCleanup := createRegistryModule(t, client, orgTest, PrivateRegistry)
	defer registryModuleTestCleanup()

	publicRegistryModuleTest, publicRegistryModuleTestCleanup := createRegistryModule(t, client, orgTest, PublicRegistry)
	defer publicRegistryModuleTestCleanup()

	t.Run("with valid name and provider", func(t *testing.T) {
		rm, err := client.RegistryModules.Read(ctx, RegistryModuleID{
			Organization: orgTest.Name,
			Name:         registryModuleTest.Name,
			Provider:     registryModuleTest.Provider,
		})
		require.NoError(t, err)
		assert.Equal(t, registryModuleTest.ID, rm.ID)

		t.Run("permissions are properly decoded", func(t *testing.T) {
			assert.True(t, rm.Permissions.CanDelete)
			assert.True(t, rm.Permissions.CanResync)
			assert.True(t, rm.Permissions.CanRetry)
		})

		t.Run("timestamps are properly decoded", func(t *testing.T) {
			assert.NotEmpty(t, rm.CreatedAt)
			assert.NotEmpty(t, rm.UpdatedAt)
		})
	})

	t.Run("with complete registry module ID fields for private module", func(t *testing.T) {
		rm, err := client.RegistryModules.Read(ctx, RegistryModuleID{
			Organization: orgTest.Name,
			Name:         registryModuleTest.Name,
			Provider:     registryModuleTest.Provider,
			Namespace:    orgTest.Name,
			RegistryName: PrivateRegistry,
		})
		require.NoError(t, err)
		require.NotEmpty(t, rm)
		assert.Equal(t, registryModuleTest.ID, rm.ID)

		t.Run("permissions are properly decoded", func(t *testing.T) {
			require.NotEmpty(t, rm.Permissions)
			assert.True(t, rm.Permissions.CanDelete)
			assert.True(t, rm.Permissions.CanResync)
			assert.True(t, rm.Permissions.CanRetry)
		})

		t.Run("timestamps are properly decoded", func(t *testing.T) {
			assert.NotEmpty(t, rm.CreatedAt)
			assert.NotEmpty(t, rm.UpdatedAt)
		})
	})

	t.Run("with complete registry module ID fields for public module", func(t *testing.T) {
		rm, err := client.RegistryModules.Read(ctx, RegistryModuleID{
			Organization: orgTest.Name,
			Name:         publicRegistryModuleTest.Name,
			Provider:     publicRegistryModuleTest.Provider,
			Namespace:    publicRegistryModuleTest.Namespace,
			RegistryName: PublicRegistry,
		})
		require.NoError(t, err)
		require.NotEmpty(t, rm)
		assert.Equal(t, publicRegistryModuleTest.ID, rm.ID)

		t.Run("permissions are properly decoded", func(t *testing.T) {
			require.NotEmpty(t, rm.Permissions)
			assert.True(t, rm.Permissions.CanDelete)
			assert.True(t, rm.Permissions.CanResync)
			assert.True(t, rm.Permissions.CanRetry)
		})

		t.Run("timestamps are properly decoded", func(t *testing.T) {
			assert.NotEmpty(t, rm.CreatedAt)
			assert.NotEmpty(t, rm.UpdatedAt)
		})
	})

	t.Run("without a name", func(t *testing.T) {
		rm, err := client.RegistryModules.Read(ctx, RegistryModuleID{
			Organization: orgTest.Name,
			Name:         "",
			Provider:     registryModuleTest.Provider,
		})
		assert.Nil(t, rm)
		assert.EqualError(t, err, ErrRequiredName.Error())
	})

	t.Run("with an invalid name", func(t *testing.T) {
		rm, err := client.RegistryModules.Read(ctx, RegistryModuleID{
			Organization: orgTest.Name,
			Name:         badIdentifier,
			Provider:     registryModuleTest.Provider,
		})
		assert.Nil(t, rm)
		assert.EqualError(t, err, ErrInvalidName.Error())
	})

	t.Run("without a provider", func(t *testing.T) {
		rm, err := client.RegistryModules.Read(ctx, RegistryModuleID{
			Organization: orgTest.Name,
			Name:         registryModuleTest.Name,
			Provider:     "",
		})
		assert.Nil(t, rm)
		assert.Equal(t, err, ErrRequiredProvider)
	})

	t.Run("with an invalid provider", func(t *testing.T) {
		rm, err := client.RegistryModules.Read(ctx, RegistryModuleID{
			Organization: orgTest.Name,
			Name:         registryModuleTest.Name,
			Provider:     badIdentifier,
		})
		assert.Nil(t, rm)
		assert.Equal(t, err, ErrInvalidProvider)
	})

	t.Run("with an invalid registry name", func(t *testing.T) {
		rm, err := client.RegistryModules.Read(ctx, RegistryModuleID{
			Organization: orgTest.Name,
			Name:         registryModuleTest.Name,
			Provider:     registryModuleTest.Provider,
			Namespace:    orgTest.Name,
			RegistryName: "PRIVATE",
		})
		assert.Nil(t, rm)
		assert.Equal(t, err, ErrInvalidRegistryName)
	})

	t.Run("without a valid organization", func(t *testing.T) {
		rm, err := client.RegistryModules.Read(ctx, RegistryModuleID{
			Organization: badIdentifier,
			Name:         registryModuleTest.Name,
			Provider:     registryModuleTest.Provider,
		})
		assert.Nil(t, rm)
		assert.EqualError(t, err, ErrInvalidOrg.Error())
	})

	t.Run("without a valid namespace for public registry module", func(t *testing.T) {
		rm, err := client.RegistryModules.Read(ctx, RegistryModuleID{
			Organization: orgTest.Name,
			Name:         publicRegistryModuleTest.Name,
			Provider:     publicRegistryModuleTest.Provider,
			RegistryName: PublicRegistry,
		})
		assert.Nil(t, rm)
		assert.EqualError(t, err, ErrRequiredNamespace.Error())
	})

	t.Run("when the registry module does not exist", func(t *testing.T) {
		rm, err := client.RegistryModules.Read(ctx, RegistryModuleID{
			Organization: orgTest.Name,
			Name:         "nonexisting",
			Provider:     "nonexisting",
		})
		assert.Nil(t, rm)
		assert.Error(t, err)
	})
}

func TestRegistryModulesDelete(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	defer orgTestCleanup()

	registryModuleTest, _ := createRegistryModule(t, client, orgTest, PrivateRegistry)

	t.Run("with valid name", func(t *testing.T) {
		err := client.RegistryModules.Delete(ctx, orgTest.Name, registryModuleTest.Name)
		require.NoError(t, err)

		rm, err := client.RegistryModules.Read(ctx, RegistryModuleID{
			Organization: orgTest.Name,
			Name:         registryModuleTest.Name,
			Provider:     registryModuleTest.Provider,
		})
		assert.Nil(t, rm)
		assert.Error(t, err)
	})

	t.Run("without a name", func(t *testing.T) {
		err := client.RegistryModules.Delete(ctx, orgTest.Name, "")
		assert.EqualError(t, err, ErrRequiredName.Error())
	})

	t.Run("with an invalid name", func(t *testing.T) {
		err := client.RegistryModules.Delete(ctx, orgTest.Name, badIdentifier)
		assert.EqualError(t, err, ErrInvalidName.Error())
	})

	t.Run("without a valid organization", func(t *testing.T) {
		err := client.RegistryModules.Delete(ctx, badIdentifier, registryModuleTest.Name)
		assert.EqualError(t, err, ErrInvalidOrg.Error())
	})

	t.Run("when the registry module does not exist", func(t *testing.T) {
		err := client.RegistryModules.Delete(ctx, orgTest.Name, "nonexisting")
		assert.Error(t, err)
		assert.Equal(t, ErrResourceNotFound, err)
	})
}

func TestRegistryModulesDeleteProvider(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	defer orgTestCleanup()

	registryModuleTest, _ := createRegistryModule(t, client, orgTest, PrivateRegistry)

	t.Run("with valid name and provider", func(t *testing.T) {
		err := client.RegistryModules.DeleteProvider(ctx, RegistryModuleID{
			Organization: orgTest.Name,
			Name:         registryModuleTest.Name,
			Provider:     registryModuleTest.Provider,
		})
		require.NoError(t, err)

		rm, err := client.RegistryModules.Read(ctx, RegistryModuleID{
			Organization: orgTest.Name,
			Name:         registryModuleTest.Name,
			Provider:     registryModuleTest.Provider,
		})
		assert.Nil(t, rm)
		assert.Error(t, err)
	})

	t.Run("without a name", func(t *testing.T) {
		err := client.RegistryModules.DeleteProvider(ctx, RegistryModuleID{
			Organization: orgTest.Name,
			Name:         "",
			Provider:     registryModuleTest.Provider,
		})
		assert.EqualError(t, err, ErrRequiredName.Error())
	})

	t.Run("with an invalid name", func(t *testing.T) {
		err := client.RegistryModules.DeleteProvider(ctx, RegistryModuleID{
			Organization: orgTest.Name,
			Name:         badIdentifier,
			Provider:     registryModuleTest.Provider,
		})
		assert.EqualError(t, err, ErrInvalidName.Error())
	})

	t.Run("without a provider", func(t *testing.T) {
		err := client.RegistryModules.DeleteProvider(ctx, RegistryModuleID{
			Organization: orgTest.Name,
			Name:         registryModuleTest.Name,
			Provider:     "",
		})
		assert.Equal(t, err, ErrRequiredProvider)
	})

	t.Run("with an invalid provider", func(t *testing.T) {
		err := client.RegistryModules.DeleteProvider(ctx, RegistryModuleID{
			Organization: orgTest.Name,
			Name:         registryModuleTest.Name,
			Provider:     badIdentifier,
		})
		assert.Equal(t, err, ErrInvalidProvider)
	})

	t.Run("without a valid organization", func(t *testing.T) {
		err := client.RegistryModules.DeleteProvider(ctx, RegistryModuleID{
			Organization: badIdentifier,
			Name:         registryModuleTest.Name,
			Provider:     registryModuleTest.Provider,
		})
		assert.EqualError(t, err, ErrInvalidOrg.Error())
	})

	t.Run("when the registry module name and provider do not exist", func(t *testing.T) {
		err := client.RegistryModules.DeleteProvider(ctx, RegistryModuleID{
			Organization: orgTest.Name,
			Name:         "nonexisting",
			Provider:     "nonexisting",
		})
		assert.Error(t, err)
		assert.Equal(t, ErrResourceNotFound, err)
	})
}

func TestRegistryModulesDeleteVersion(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	defer orgTestCleanup()

	registryModuleTest, registryModuleTestCleanup := createRegistryModuleWithVersion(t, client, orgTest)
	defer registryModuleTestCleanup()

	t.Run("with valid name and provider", func(t *testing.T) {
		options := RegistryModuleCreateVersionOptions{
			Version: String("1.2.3"),
		}
		rmv, err := client.RegistryModules.CreateVersion(ctx, RegistryModuleID{
			Organization: orgTest.Name,
			Name:         registryModuleTest.Name,
			Provider:     registryModuleTest.Provider,
		}, options)
		require.NoError(t, err)
		require.NotEmpty(t, rmv.Version)

		rm, err := client.RegistryModules.Read(ctx, RegistryModuleID{
			Organization: orgTest.Name,
			Name:         registryModuleTest.Name,
			Provider:     registryModuleTest.Provider,
		})
		require.NoError(t, err)
		require.NotEmpty(t, rm.VersionStatuses)
		require.Equal(t, 2, len(rm.VersionStatuses))

		err = client.RegistryModules.DeleteVersion(ctx, RegistryModuleID{
			Organization: orgTest.Name,
			Name:         registryModuleTest.Name,
			Provider:     registryModuleTest.Provider,
		}, rmv.Version)
		require.NoError(t, err)

		rm, err = client.RegistryModules.Read(ctx, RegistryModuleID{
			Organization: orgTest.Name,
			Name:         registryModuleTest.Name,
			Provider:     registryModuleTest.Provider,
		})
		require.NoError(t, err)
		assert.NotEmpty(t, rm.VersionStatuses)
		assert.Equal(t, 1, len(rm.VersionStatuses))
		assert.NotEqual(t, registryModuleTest.VersionStatuses[0].Version, rmv.Version)
		assert.Equal(t, registryModuleTest.VersionStatuses, rm.VersionStatuses)
	})

	t.Run("without a name", func(t *testing.T) {
		err := client.RegistryModules.DeleteVersion(ctx, RegistryModuleID{
			Organization: orgTest.Name,
			Name:         "",
			Provider:     registryModuleTest.Provider,
		}, registryModuleTest.VersionStatuses[0].Version)
		assert.EqualError(t, err, ErrRequiredName.Error())
	})

	t.Run("with an invalid name", func(t *testing.T) {
		err := client.RegistryModules.DeleteVersion(ctx, RegistryModuleID{
			Organization: orgTest.Name,
			Name:         badIdentifier,
			Provider:     registryModuleTest.Provider,
		}, registryModuleTest.VersionStatuses[0].Version)
		assert.EqualError(t, err, ErrInvalidName.Error())
	})

	t.Run("without a provider", func(t *testing.T) {
		err := client.RegistryModules.DeleteVersion(ctx, RegistryModuleID{
			Organization: orgTest.Name,
			Name:         registryModuleTest.Name,
			Provider:     "",
		}, registryModuleTest.VersionStatuses[0].Version)
		assert.Equal(t, err, ErrRequiredProvider)
	})

	t.Run("with an invalid provider", func(t *testing.T) {
		err := client.RegistryModules.DeleteVersion(ctx, RegistryModuleID{
			Organization: orgTest.Name,
			Name:         registryModuleTest.Name,
			Provider:     badIdentifier,
		}, registryModuleTest.VersionStatuses[0].Version)
		assert.Equal(t, err, ErrInvalidProvider)
	})

	t.Run("without a version", func(t *testing.T) {
		err := client.RegistryModules.DeleteVersion(ctx, RegistryModuleID{
			Organization: orgTest.Name,
			Name:         registryModuleTest.Name,
			Provider:     registryModuleTest.Provider,
		}, "")
		assert.Equal(t, err, ErrRequiredVersion)
	})

	t.Run("with an invalid version", func(t *testing.T) {
		err := client.RegistryModules.DeleteVersion(ctx, RegistryModuleID{
			Organization: orgTest.Name,
			Name:         registryModuleTest.Name,
			Provider:     registryModuleTest.Provider,
		}, badIdentifier)
		assert.Equal(t, err, ErrInvalidVersion)
	})

	t.Run("without a valid organization", func(t *testing.T) {
		err := client.RegistryModules.DeleteVersion(ctx, RegistryModuleID{
			Organization: badIdentifier,
			Name:         registryModuleTest.Name,
			Provider:     registryModuleTest.Provider,
		}, registryModuleTest.VersionStatuses[0].Version)
		assert.EqualError(t, err, ErrInvalidOrg.Error())
	})

	t.Run("when the registry module name, provider, and version do not exist", func(t *testing.T) {
		err := client.RegistryModules.DeleteVersion(ctx, RegistryModuleID{
			Organization: orgTest.Name,
			Name:         "nonexisting",
			Provider:     "nonexisting",
		}, "2.0.0")
		assert.Error(t, err)
	})
}

func TestRegistryModulesUpload(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	defer orgTestCleanup()

	rm, _ := createRegistryModule(t, client, orgTest, PrivateRegistry)

	optionsModuleVersion := RegistryModuleCreateVersionOptions{
		Version: String("1.0.0"),
	}
	rmv, err := client.RegistryModules.CreateVersion(ctx, RegistryModuleID{
		Organization: orgTest.Name,
		Name:         rm.Name,
		Provider:     rm.Provider,
	}, optionsModuleVersion)
	if err != nil {
		t.Fatal(err)
	}

	t.Run("with valid upload URL", func(t *testing.T) {
		err = client.RegistryModules.Upload(
			ctx,
			*rmv,
			"test-fixtures/config-version",
		)
		require.NoError(t, err)
	})

	t.Run("with missing upload URL", func(t *testing.T) {
		delete(rmv.Links, "upload")

		err = client.RegistryModules.Upload(
			ctx,
			*rmv,
			"test-fixtures/config-version",
		)
		assert.EqualError(t, err, "provided RegistryModuleVersion does not contain an upload link")
	})
}

func TestRegistryModule_Unmarshal(t *testing.T) {
	data := map[string]interface{}{
		"data": map[string]interface{}{
			"type": "registry-modules",
			"id":   "1",
			"attributes": map[string]interface{}{
				"name":          "module",
				"provider":      "tfe",
				"namespace":     "org-abc",
				"registry-name": "private",
				"permissions": map[string]interface{}{
					"can-delete": true,
					"can-resync": true,
					"can-retry":  true,
				},
				"status": RegistryModuleStatusPending,
				"vcs-repo": map[string]interface{}{
					"branch":              "main",
					"display-identifier":  "display",
					"identifier":          "identifier",
					"ingress-submodules":  true,
					"oauth-token-id":      "token",
					"repository-http-url": "github.com",
					"service-provider":    "github",
					"webhook-url":         "https://app.terraform.io/webhooks/vcs/aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee",
				},
				"version-statuses": []interface{}{
					map[string]interface{}{
						"version": "1.1.1",
						"status":  RegistryModuleVersionStatusPending,
						"error":   "no error",
					},
				},
			},
		},
	}

	byteData, err := json.Marshal(data)
	require.NoError(t, err)

	responseBody := bytes.NewReader(byteData)
	rm := &RegistryModule{}
	err = unmarshalResponse(responseBody, rm)
	require.NoError(t, err)

	assert.Equal(t, rm.ID, "1")
	assert.Equal(t, rm.Name, "module")
	assert.Equal(t, rm.Provider, "tfe")
	assert.Equal(t, rm.Namespace, "org-abc")
	assert.Equal(t, rm.RegistryName, PrivateRegistry)
	assert.Equal(t, rm.Permissions.CanDelete, true)
	assert.Equal(t, rm.Permissions.CanRetry, true)
	assert.Equal(t, rm.Status, RegistryModuleStatusPending)
	assert.Equal(t, rm.VCSRepo.Branch, "main")
	assert.Equal(t, rm.VCSRepo.DisplayIdentifier, "display")
	assert.Equal(t, rm.VCSRepo.Identifier, "identifier")
	assert.Equal(t, rm.VCSRepo.IngressSubmodules, true)
	assert.Equal(t, rm.VCSRepo.OAuthTokenID, "token")
	assert.Equal(t, rm.VCSRepo.RepositoryHTTPURL, "github.com")
	assert.Equal(t, rm.VCSRepo.ServiceProvider, "github")
	assert.Equal(t, rm.VCSRepo.WebhookURL, "https://app.terraform.io/webhooks/vcs/aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee")
	assert.Equal(t, rm.Status, RegistryModuleStatusPending)
	assert.Equal(t, rm.VersionStatuses[0].Version, "1.1.1")
	assert.Equal(t, rm.VersionStatuses[0].Status, RegistryModuleVersionStatusPending)
	assert.Equal(t, rm.VersionStatuses[0].Error, "no error")
}

func TestRegistryCreateWithVCSOptions_Marshal(t *testing.T) {
	// https://www.terraform.io/docs/cloud/api/modules.html#sample-payload
	opts := RegistryModuleCreateWithVCSConnectionOptions{
		VCSRepo: &RegistryModuleVCSRepoOptions{
			Identifier:        String("id"),
			OAuthTokenID:      String("token"),
			DisplayIdentifier: String("display-id"),
		},
	}

	reqBody, err := serializeRequestBody(&opts)
	require.NoError(t, err)
	req, err := retryablehttp.NewRequest("POST", "url", reqBody)
	require.NoError(t, err)
	bodyBytes, err := req.BodyBytes()
	require.NoError(t, err)

	expectedBody := `{"data":{"type":"registry-modules","attributes":{"vcs-repo":{"identifier":"id","oauth-token-id":"token","display-identifier":"display-id"}}}}
`
	assert.Equal(t, expectedBody, string(bodyBytes))
}
