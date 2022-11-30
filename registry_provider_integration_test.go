package tfe

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRegistryProvidersList(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	t.Run("with providers", func(t *testing.T) {
		orgTest, orgTestCleanup := createOrganization(t, client)
		defer orgTestCleanup()

		createN := 10
		providers := make([]*RegistryProvider, 0)
		// These providers will be destroyed when the org is cleaned up
		for i := 0; i < createN; i++ {
			// Create public providers
			providerTest, _ := createRegistryProvider(t, client, orgTest, PublicRegistry)
			providers = append(providers, providerTest)
		}
		for i := 0; i < createN; i++ {
			// Create private providers
			providerTest, _ := createRegistryProvider(t, client, orgTest, PrivateRegistry)
			providers = append(providers, providerTest)
		}
		providerN := len(providers)
		publicProviderN := createN

		t.Run("returns all providers", func(t *testing.T) {
			returnedProviders, err := client.RegistryProviders.List(ctx, orgTest.Name, &RegistryProviderListOptions{
				ListOptions: ListOptions{
					PageNumber: 0,
					PageSize:   providerN,
				},
			})
			require.NoError(t, err)
			assert.NotEmpty(t, returnedProviders.Items)
			assert.Equal(t, providerN, returnedProviders.TotalCount)
			assert.Equal(t, 1, returnedProviders.TotalPages)
		})

		t.Run("with list options", func(t *testing.T) {
			// Request a page number which is out of range. The result should
			// be successful, but return no results if the paging options are
			// properly passed along.
			rpl, err := client.RegistryProviders.List(ctx, orgTest.Name, &RegistryProviderListOptions{
				ListOptions: ListOptions{
					PageNumber: 999,
					PageSize:   100,
				},
			})
			require.NoError(t, err)
			assert.Empty(t, rpl.Items)
			assert.Equal(t, 999, rpl.CurrentPage)
			assert.Equal(t, 20, rpl.TotalCount)
		})

		t.Run("filters on registry name", func(t *testing.T) {
			returnedProviders, err := client.RegistryProviders.List(ctx, orgTest.Name, &RegistryProviderListOptions{
				RegistryName: PublicRegistry,
				ListOptions: ListOptions{
					PageNumber: 0,
					PageSize:   providerN,
				},
			})
			require.NoError(t, err)
			assert.NotEmpty(t, returnedProviders.Items)
			assert.Equal(t, publicProviderN, returnedProviders.TotalCount)
			assert.Equal(t, 1, returnedProviders.TotalPages)
			for _, rp := range returnedProviders.Items {
				foundProvider := false
				for _, p := range providers {
					if rp.ID == p.ID {
						foundProvider = true
						break
					}
				}
				assert.Equal(t, PublicRegistry, rp.RegistryName)
				assert.True(t, foundProvider, "Expected to find provider %s but did not:\nexpected:\n%v\nreturned\n%v", rp.ID, providers, returnedProviders)
			}
		})

		t.Run("searches", func(t *testing.T) {
			expectedProvider := providers[0]
			returnedProviders, err := client.RegistryProviders.List(ctx, orgTest.Name, &RegistryProviderListOptions{
				Search: expectedProvider.Name,
			})

			require.NoError(t, err)

			assert.NotEmpty(t, returnedProviders.Items)
			assert.Equal(t, 1, returnedProviders.TotalCount)
			assert.Equal(t, 1, returnedProviders.TotalPages)

			foundProvider := returnedProviders.Items[0]
			assert.Equal(t, foundProvider.ID, expectedProvider.ID, "Expected to find provider %s but did not:\nexpected:\n%v\nreturned\n%v", expectedProvider.ID, providers, returnedProviders)
		})
	})

	t.Run("without providers", func(t *testing.T) {
		orgTest, orgTestCleanup := createOrganization(t, client)
		defer orgTestCleanup()

		providers, err := client.RegistryProviders.List(ctx, orgTest.Name, nil)
		require.NoError(t, err)
		assert.Empty(t, providers.Items)
		assert.Equal(t, 0, providers.TotalCount)
		assert.Equal(t, 0, providers.TotalPages)
	})

	t.Run("with include provider versions", func(t *testing.T) {
		version1, version1Cleanup := createRegistryProviderVersion(t, client, nil)
		defer version1Cleanup()

		provider := version1.RegistryProvider

		version2, version2Cleanup := createRegistryProviderVersion(t, client, provider)
		defer version2Cleanup()

		versions := []*RegistryProviderVersion{version1, version2}

		options := RegistryProviderListOptions{
			Include: &[]RegistryProviderIncludeOps{
				RegistryProviderVersionsInclude,
			},
		}

		providersRead, err := client.RegistryProviders.List(ctx, provider.Organization.Name, &options)
		require.NoError(t, err)

		require.NotEmpty(t, providersRead.Items)
		providerRead := providersRead.Items[0]
		assert.Equal(t, providerRead.ID, provider.ID)
		assert.Equal(t, len(versions), len(providerRead.RegistryProviderVersions))
		foundVersion := &RegistryProviderVersion{}
		for _, v := range providerRead.RegistryProviderVersions {
			for i := 0; i < len(versions); i++ {
				if v.ID == versions[i].ID {
					foundVersion = versions[i]
					break
				}
			}
			assert.True(t, foundVersion.ID != "", "Expected to find versions: %v but did not", versions)
			assert.Equal(t, v.Version, foundVersion.Version)
		}
	})
}

func TestRegistryProvidersCreate(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	defer orgTestCleanup()

	t.Run("with valid options", func(t *testing.T) {
		publicProviderOptions := RegistryProviderCreateOptions{
			Name:         "provider_name",
			Namespace:    "public_namespace",
			RegistryName: PublicRegistry,
		}
		privateProviderOptions := RegistryProviderCreateOptions{
			Name:         "provider_name",
			Namespace:    orgTest.Name,
			RegistryName: PrivateRegistry,
		}

		registryOptions := []RegistryProviderCreateOptions{publicProviderOptions, privateProviderOptions}

		for _, options := range registryOptions {
			testName := fmt.Sprintf("with %s provider", options.RegistryName)
			t.Run(testName, func(t *testing.T) {
				prv, err := client.RegistryProviders.Create(ctx, orgTest.Name, options)
				require.NoError(t, err)
				assert.NotEmpty(t, prv.ID)
				assert.Equal(t, options.Name, prv.Name)
				assert.Equal(t, options.Namespace, prv.Namespace)
				assert.Equal(t, options.RegistryName, prv.RegistryName)

				t.Run(testPermissionsProperlyDecoded, func(t *testing.T) {
					assert.True(t, prv.Permissions.CanDelete)
				})

				t.Run(testRelationshipsProperlyDecoded, func(t *testing.T) {
					assert.Equal(t, orgTest.Name, prv.Organization.Name)
				})

				t.Run(testTimestampsProperlyDecoded, func(t *testing.T) {
					assert.NotEmpty(t, prv.CreatedAt)
					assert.NotEmpty(t, prv.UpdatedAt)
				})
			})
		}
	})

	t.Run(testWithInvalidOptions, func(t *testing.T) {
		t.Run("without a name", func(t *testing.T) {
			options := RegistryProviderCreateOptions{
				Namespace:    "namespace",
				RegistryName: PublicRegistry,
			}
			rm, err := client.RegistryProviders.Create(ctx, orgTest.Name, options)
			assert.Nil(t, rm)
			assert.EqualError(t, err, ErrInvalidName.Error())
		})

		t.Run(testWithInvalidName, func(t *testing.T) {
			options := RegistryProviderCreateOptions{
				Name:         "invalid name",
				Namespace:    "namespace",
				RegistryName: PublicRegistry,
			}
			rm, err := client.RegistryProviders.Create(ctx, orgTest.Name, options)
			assert.Nil(t, rm)
			assert.EqualError(t, err, ErrInvalidName.Error())
		})

		t.Run("without a namespace", func(t *testing.T) {
			options := RegistryProviderCreateOptions{
				Name:         "name",
				RegistryName: PublicRegistry,
			}
			rm, err := client.RegistryProviders.Create(ctx, orgTest.Name, options)
			assert.Nil(t, rm)
			assert.EqualError(t, err, ErrInvalidNamespace.Error())
		})

		t.Run("with an invalid namespace", func(t *testing.T) {
			options := RegistryProviderCreateOptions{
				Name:         "name",
				Namespace:    "invalid namespace",
				RegistryName: PublicRegistry,
			}
			rm, err := client.RegistryProviders.Create(ctx, orgTest.Name, options)
			assert.Nil(t, rm)
			assert.EqualError(t, err, ErrInvalidNamespace.Error())
		})

		t.Run("without a registry-name", func(t *testing.T) {
			options := RegistryProviderCreateOptions{
				Name:      "name",
				Namespace: "namespace",
			}
			rm, err := client.RegistryProviders.Create(ctx, orgTest.Name, options)
			assert.Nil(t, rm)
			// This error is returned by the API
			assert.EqualError(t, err, "invalid attribute\n\nRegistry name can't be blank\ninvalid attribute\n\nRegistry name is not included in the list")
		})

		t.Run("with an invalid registry-name", func(t *testing.T) {
			options := RegistryProviderCreateOptions{
				Name:         "name",
				Namespace:    "namespace",
				RegistryName: "invalid",
			}
			rm, err := client.RegistryProviders.Create(ctx, orgTest.Name, options)
			assert.Nil(t, rm)
			// This error is returned by the API
			assert.EqualError(t, err, "invalid attribute\n\nRegistry name is not included in the list")
		})
	})

	t.Run(testWithoutValidOrganization, func(t *testing.T) {
		options := RegistryProviderCreateOptions{
			Name:         "name",
			Namespace:    "namespace",
			RegistryName: PublicRegistry,
		}
		rm, err := client.RegistryProviders.Create(ctx, badIdentifier, options)
		assert.Nil(t, rm)
		assert.EqualError(t, err, ErrInvalidOrg.Error())
	})
}

func TestRegistryProvidersRead(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	defer orgTestCleanup()

	type ProviderContext struct {
		RegistryName RegistryName
	}

	providerContexts := []ProviderContext{
		{
			RegistryName: PublicRegistry,
		},
		{
			RegistryName: PrivateRegistry,
		},
	}

	for _, prvCtx := range providerContexts {
		testName := fmt.Sprintf("with %s provider", prvCtx.RegistryName)
		t.Run(testName, func(t *testing.T) {
			t.Run("with valid provider", func(t *testing.T) {
				registryProviderTest, providerTestCleanup := createRegistryProvider(t, client, orgTest, prvCtx.RegistryName)
				defer providerTestCleanup()

				id := RegistryProviderID{
					OrganizationName: orgTest.Name,
					RegistryName:     registryProviderTest.RegistryName,
					Namespace:        registryProviderTest.Namespace,
					Name:             registryProviderTest.Name,
				}

				prv, err := client.RegistryProviders.Read(ctx, id, nil)
				require.NoError(t, err)

				assert.NotEmpty(t, prv.ID)
				assert.Equal(t, registryProviderTest.Name, prv.Name)
				assert.Equal(t, registryProviderTest.Namespace, prv.Namespace)
				assert.Equal(t, registryProviderTest.RegistryName, prv.RegistryName)

				t.Run(testPermissionsProperlyDecoded, func(t *testing.T) {
					assert.True(t, prv.Permissions.CanDelete)
				})

				t.Run(testRelationshipsProperlyDecoded, func(t *testing.T) {
					assert.Equal(t, orgTest.Name, prv.Organization.Name)
				})

				t.Run(testTimestampsProperlyDecoded, func(t *testing.T) {
					assert.NotEmpty(t, prv.CreatedAt)
					assert.NotEmpty(t, prv.UpdatedAt)
				})
			})

			t.Run("when the registry provider does not exist", func(t *testing.T) {
				id := RegistryProviderID{
					OrganizationName: orgTest.Name,
					RegistryName:     prvCtx.RegistryName,
					Namespace:        "nonexistent",
					Name:             "nonexistent",
				}
				_, err := client.RegistryProviders.Read(ctx, id, nil)
				assert.Error(t, err)
				// Local TFC/E will return a forbidden here when TFC/E is in development mode
				// In non development mode this returns a 404
				assert.Equal(t, ErrResourceNotFound, err)
			})
		})
	}

	t.Run("populates version relationships", func(t *testing.T) {
		version1, version1Cleanup := createRegistryProviderVersion(t, client, nil)
		defer version1Cleanup()

		provider := version1.RegistryProvider

		version2, version2Cleanup := createRegistryProviderVersion(t, client, provider)
		defer version2Cleanup()

		versions := []*RegistryProviderVersion{version1, version2}

		id := RegistryProviderID{
			OrganizationName: provider.Organization.Name,
			RegistryName:     provider.RegistryName,
			Namespace:        provider.Namespace,
			Name:             provider.Name,
		}

		options := RegistryProviderReadOptions{
			Include: []RegistryProviderIncludeOps{
				RegistryProviderVersionsInclude,
			},
		}

		providerRead, err := client.RegistryProviders.Read(ctx, id, &options)
		require.NoError(t, err)
		require.NotEmpty(t, providerRead.RegistryProviderVersions)

		assert.Equal(t, providerRead.ID, provider.ID)
		assert.Equal(t, len(versions), len(providerRead.RegistryProviderVersions))
		foundVersion := &RegistryProviderVersion{}
		for _, v := range providerRead.RegistryProviderVersions {
			for i := 0; i < len(versions); i++ {
				if v.ID == versions[i].ID {
					foundVersion = versions[i]
					break
				}
			}
			assert.True(t, foundVersion.ID != "", "Expected to find versions: %v but did not", versions)
			assert.Equal(t, v.Version, foundVersion.Version)
		}
	})
}

func TestRegistryProvidersDelete(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	defer orgTestCleanup()

	type ProviderContext struct {
		RegistryName RegistryName
	}

	providerContexts := []ProviderContext{
		{
			RegistryName: PublicRegistry,
		},
		{
			RegistryName: PrivateRegistry,
		},
	}

	for _, prvCtx := range providerContexts {
		testName := fmt.Sprintf("with %s provider", prvCtx.RegistryName)
		t.Run(testName, func(t *testing.T) {
			t.Run("with valid provider", func(t *testing.T) {
				registryProviderTest, _ := createRegistryProvider(t, client, orgTest, prvCtx.RegistryName)

				id := RegistryProviderID{
					OrganizationName: orgTest.Name,
					RegistryName:     registryProviderTest.RegistryName,
					Namespace:        registryProviderTest.Namespace,
					Name:             registryProviderTest.Name,
				}

				err := client.RegistryProviders.Delete(ctx, id)
				require.NoError(t, err)

				prv, err := client.RegistryProviders.Read(ctx, id, nil)
				assert.Nil(t, prv)
				assert.Error(t, err)
			})

			t.Run("when the registry provider does not exist", func(t *testing.T) {
				id := RegistryProviderID{
					OrganizationName: orgTest.Name,
					RegistryName:     prvCtx.RegistryName,
					Namespace:        "nonexistent",
					Name:             "nonexistent",
				}
				err := client.RegistryProviders.Delete(ctx, id)
				assert.Error(t, err)
				// Local TFC/E will return a forbidden here when TFC/E is in development mode
				// In non development mode this returns a 404
				assert.Equal(t, ErrResourceNotFound, err)
			})
		})
	}
}

func TestRegistryProvidersIDValidation(t *testing.T) {
	orgName := "orgName"
	registryName := PublicRegistry

	t.Run("valid", func(t *testing.T) {
		id := RegistryProviderID{
			OrganizationName: orgName,
			RegistryName:     registryName,
			Namespace:        "namespace",
			Name:             "name",
		}
		require.NoError(t, id.valid())
	})

	t.Run("without a name", func(t *testing.T) {
		id := RegistryProviderID{
			OrganizationName: orgName,
			RegistryName:     registryName,
			Namespace:        "namespace",
			Name:             "",
		}
		assert.EqualError(t, id.valid(), ErrInvalidName.Error())
	})

	t.Run(testWithInvalidName, func(t *testing.T) {
		id := RegistryProviderID{
			OrganizationName: orgName,
			RegistryName:     registryName,
			Namespace:        "namespace",
			Name:             badIdentifier,
		}
		assert.EqualError(t, id.valid(), ErrInvalidName.Error())
	})

	t.Run("without a namespace", func(t *testing.T) {
		id := RegistryProviderID{
			OrganizationName: orgName,
			RegistryName:     registryName,
			Namespace:        "",
			Name:             "name",
		}
		assert.EqualError(t, id.valid(), ErrInvalidNamespace.Error())
	})

	t.Run("with an invalid namespace", func(t *testing.T) {
		id := RegistryProviderID{
			OrganizationName: orgName,
			RegistryName:     registryName,
			Namespace:        badIdentifier,
			Name:             "name",
		}
		assert.EqualError(t, id.valid(), ErrInvalidNamespace.Error())
	})

	t.Run("without a registry-name", func(t *testing.T) {
		id := RegistryProviderID{
			OrganizationName: orgName,
			RegistryName:     "",
			Namespace:        "namespace",
			Name:             "name",
		}
		assert.EqualError(t, id.valid(), ErrInvalidRegistryName.Error())
	})

	t.Run(testWithoutValidOrganization, func(t *testing.T) {
		id := RegistryProviderID{
			OrganizationName: badIdentifier,
			RegistryName:     registryName,
			Namespace:        "namespace",
			Name:             "name",
		}
		assert.EqualError(t, id.valid(), ErrInvalidOrg.Error())
	})
}
