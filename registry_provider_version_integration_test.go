//go:build integration
// +build integration

package tfe

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRegistryProviderVersionsIDValidation(t *testing.T) {
	version := "1.0.0"
	validRegistryProviderId := RegistryProviderID{
		OrganizationName: "orgName",
		RegistryName:     PrivateRegistry,
		Namespace:        "namespace",
		Name:             "name",
	}
	invalidRegistryProviderId := RegistryProviderID{
		OrganizationName: badIdentifier,
		RegistryName:     PrivateRegistry,
		Namespace:        "namespace",
		Name:             "name",
	}
	publicRegistryProviderId := RegistryProviderID{
		OrganizationName: "orgName",
		RegistryName:     PublicRegistry,
		Namespace:        "namespace",
		Name:             "name",
	}

	t.Run("valid", func(t *testing.T) {
		id := RegistryProviderVersionID{
			Version:            version,
			RegistryProviderID: validRegistryProviderId,
		}
		require.NoError(t, id.valid())
	})

	t.Run("without a version", func(t *testing.T) {
		id := RegistryProviderVersionID{
			Version:            "",
			RegistryProviderID: validRegistryProviderId,
		}
		assert.EqualError(t, id.valid(), ErrInvalidVersion.Error())
	})

	t.Run("without a key-id", func(t *testing.T) {
		id := RegistryProviderVersionID{
			Version:            "",
			RegistryProviderID: validRegistryProviderId,
		}
		assert.EqualError(t, id.valid(), ErrInvalidVersion.Error())
	})

	t.Run("invalid version", func(t *testing.T) {
		t.Skip("This is skipped as we don't actually validate version is a valid semver - the registry does this validation")
		id := RegistryProviderVersionID{
			Version:            "foo",
			RegistryProviderID: validRegistryProviderId,
		}
		assert.EqualError(t, id.valid(), ErrInvalidVersion.Error())
	})

	t.Run("invalid registry for parent provider", func(t *testing.T) {
		id := RegistryProviderVersionID{
			Version:            version,
			RegistryProviderID: publicRegistryProviderId,
		}
		assert.EqualError(t, id.valid(), ErrRequiredPrivateRegistry.Error())
	})

	t.Run("without a valid registry provider id", func(t *testing.T) {
		// this is a proxy for all permutations of an invalid registry provider id
		// it is assumed that validity of the registry provider id is delegated to its own valid method
		id := RegistryProviderVersionID{
			Version:            version,
			RegistryProviderID: invalidRegistryProviderId,
		}
		assert.EqualError(t, id.valid(), ErrInvalidOrg.Error())
	})
}

func TestRegistryProviderVersionsCreate(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	providerTest, providerTestCleanup := createRegistryProvider(t, client, nil, PrivateRegistry)
	defer providerTestCleanup()

	providerId := RegistryProviderID{
		OrganizationName: providerTest.Organization.Name,
		RegistryName:     providerTest.RegistryName,
		Namespace:        providerTest.Namespace,
		Name:             providerTest.Name,
	}

	t.Run("with valid options", func(t *testing.T) {
		options := RegistryProviderVersionCreateOptions{
			Version: "1.0.0",
			KeyID:   "abcdefg",
		}
		prvv, err := client.RegistryProviderVersions.Create(ctx, providerId, options)
		require.NoError(t, err)
		assert.NotEmpty(t, prvv.ID)
		assert.Equal(t, options.Version, prvv.Version)
		assert.Equal(t, options.KeyID, prvv.KeyID)

		t.Run("relationships are properly decoded", func(t *testing.T) {
			assert.Equal(t, providerTest.ID, prvv.RegistryProvider.ID)
		})

		t.Run("timestamps are properly decoded", func(t *testing.T) {
			assert.NotEmpty(t, prvv.CreatedAt)
			assert.NotEmpty(t, prvv.UpdatedAt)
		})

		t.Run("includes upload links", func(t *testing.T) {
			_, err := prvv.ShasumsUploadURL()
			require.NoError(t, err)
			_, err = prvv.ShasumsSigUploadURL()
			require.NoError(t, err)
			expectedLinks := []string{
				"shasums-upload",
				"shasums-sig-upload",
			}
			for _, l := range expectedLinks {
				_, ok := prvv.Links[l].(string)
				assert.True(t, ok, "Expect upload link: %s", l)
			}
		})

		t.Run("doesn't include download links", func(t *testing.T) {
			_, err := prvv.ShasumsDownloadURL()
			assert.Error(t, err)
			_, err = prvv.ShasumsSigDownloadURL()
			assert.Error(t, err)
		})
	})

	t.Run("with invalid options", func(t *testing.T) {
		t.Run("without a version", func(t *testing.T) {
			options := RegistryProviderVersionCreateOptions{
				Version: "",
				KeyID:   "abcdefg",
			}
			rm, err := client.RegistryProviderVersions.Create(ctx, providerId, options)
			assert.Nil(t, rm)
			assert.EqualError(t, err, ErrInvalidVersion.Error())
		})

		t.Run("without a key-id", func(t *testing.T) {
			options := RegistryProviderVersionCreateOptions{
				Version: "1.0.0",
				KeyID:   "",
			}
			rm, err := client.RegistryProviderVersions.Create(ctx, providerId, options)
			assert.Nil(t, rm)
			assert.EqualError(t, err, ErrInvalidKeyID.Error())
		})

		t.Run("with a public provider", func(t *testing.T) {
			options := RegistryProviderVersionCreateOptions{
				Version: "1.0.0",
				KeyID:   "abcdefg",
			}
			providerId := RegistryProviderID{
				OrganizationName: providerTest.Organization.Name,
				RegistryName:     PublicRegistry,
				Namespace:        providerTest.Namespace,
				Name:             providerTest.Name,
			}
			rm, err := client.RegistryProviderVersions.Create(ctx, providerId, options)
			assert.Nil(t, rm)
			assert.EqualError(t, err, ErrRequiredPrivateRegistry.Error())
		})

		t.Run("without a valid provider id", func(t *testing.T) {
			options := RegistryProviderVersionCreateOptions{
				Version: "1.0.0",
				KeyID:   "abcdefg",
			}
			providerId := RegistryProviderID{
				OrganizationName: badIdentifier,
				RegistryName:     providerTest.RegistryName,
				Namespace:        providerTest.Namespace,
				Name:             providerTest.Name,
			}
			rm, err := client.RegistryProviderVersions.Create(ctx, providerId, options)
			assert.Nil(t, rm)
			assert.EqualError(t, err, ErrInvalidOrg.Error())
		})
	})
}

func TestRegistryProviderVersionsList(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	t.Run("with versions", func(t *testing.T) {
		provider, providerCleanup := createRegistryProvider(t, client, nil, PrivateRegistry)
		defer providerCleanup()

		createN := 10
		versions := make([]*RegistryProviderVersion, 0)
		// these providers will be destroyed when the org is cleaned up
		for i := 0; i < createN; i++ {
			version, _ := createRegistryProviderVersion(t, client, provider)
			versions = append(versions, version)
		}
		versionN := len(versions)

		providerID := RegistryProviderID{
			OrganizationName: provider.Organization.Name,
			Namespace:        provider.Namespace,
			Name:             provider.Name,
			RegistryName:     provider.RegistryName,
		}

		t.Run("returns all versions", func(t *testing.T) {
			returnedVersions, err := client.RegistryProviderVersions.List(ctx, providerID, &RegistryProviderVersionListOptions{
				ListOptions: ListOptions{
					PageNumber: 0,
					PageSize:   versionN,
				},
			})
			require.NoError(t, err)
			require.NotEmpty(t, returnedVersions.Items)
			assert.Equal(t, versionN, returnedVersions.TotalCount)
			assert.Equal(t, 1, returnedVersions.TotalPages)
			for _, rv := range returnedVersions.Items {
				foundVersion := false
				for _, v := range versions {
					if rv.ID == v.ID {
						foundVersion = true
						break
					}
				}
				assert.True(t, foundVersion, "Expected to find version %s but did not:\nexpected:\n%v\nreturned\n%v", rv.ID, versions, returnedVersions)
			}
		})

		t.Run("returns pages", func(t *testing.T) {
			pageN := 2
			pageSize := versionN / pageN

			for page := 0; page < pageN; page++ {
				testName := fmt.Sprintf("returns page %d of versions", page)
				t.Run(testName, func(t *testing.T) {
					returnedVersions, err := client.RegistryProviderVersions.List(ctx, providerID, &RegistryProviderVersionListOptions{
						ListOptions: ListOptions{
							PageNumber: page,
							PageSize:   pageSize,
						},
					})
					require.NoError(t, err)
					require.NotEmpty(t, returnedVersions.Items)

					assert.Equal(t, versionN, returnedVersions.TotalCount)
					assert.Equal(t, pageN, returnedVersions.TotalPages)
					assert.Equal(t, pageSize, len(returnedVersions.Items))
					for _, rv := range returnedVersions.Items {
						foundVersion := false
						for _, v := range versions {
							if rv.ID == v.ID {
								foundVersion = true
								break
							}
						}
						assert.True(t, foundVersion, "Expected to find version %s but did not:\nexpected:\n%v\nreturned\n%v", rv.ID, versions, returnedVersions)
					}
				})
			}
		})
	})

	t.Run("without versions", func(t *testing.T) {
		provider, providerCleanup := createRegistryProvider(t, client, nil, PrivateRegistry)
		defer providerCleanup()

		providerID := RegistryProviderID{
			OrganizationName: provider.Organization.Name,
			Namespace:        provider.Namespace,
			Name:             provider.Name,
			RegistryName:     provider.RegistryName,
		}

		versions, err := client.RegistryProviderVersions.List(ctx, providerID, nil)
		require.NoError(t, err)
		assert.Empty(t, versions.Items)
		assert.Equal(t, 0, versions.TotalCount)
		assert.Equal(t, 0, versions.TotalPages)
	})

	// TODO
	t.Run("with include provider platforms", func(t *testing.T) {
	})
}

func TestRegistryProviderVersionsDelete(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	provider, providerCleanup := createRegistryProvider(t, client, nil, PrivateRegistry)
	defer providerCleanup()

	t.Run("with valid version", func(t *testing.T) {
		version, _ := createRegistryProviderVersion(t, client, provider)

		versionID := RegistryProviderVersionID{
			RegistryProviderID: RegistryProviderID{
				OrganizationName: version.RegistryProvider.Organization.Name,
				RegistryName:     version.RegistryProvider.RegistryName,
				Namespace:        version.RegistryProvider.Namespace,
				Name:             version.RegistryProvider.Name,
			},
			Version: version.Version,
		}

		err := client.RegistryProviderVersions.Delete(ctx, versionID)
		require.NoError(t, err)
	})

	t.Run("with non existing version", func(t *testing.T) {
		versionID := RegistryProviderVersionID{
			RegistryProviderID: RegistryProviderID{
				OrganizationName: provider.Organization.Name,
				RegistryName:     provider.RegistryName,
				Namespace:        provider.Namespace,
				Name:             provider.Name,
			},
			Version: "1.0.0",
		}

		err := client.RegistryProviderVersions.Delete(ctx, versionID)
		assert.Error(t, err)
	})
}

func TestRegistryProviderVersionsRead(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	t.Run("with valid version", func(t *testing.T) {
		version, versionCleanup := createRegistryProviderVersion(t, client, nil)
		defer versionCleanup()

		versionID := RegistryProviderVersionID{
			RegistryProviderID: RegistryProviderID{
				OrganizationName: version.RegistryProvider.Organization.Name,
				RegistryName:     version.RegistryProvider.RegistryName,
				Namespace:        version.RegistryProvider.Namespace,
				Name:             version.RegistryProvider.Name,
			},
			Version: version.Version,
		}

		readVersion, err := client.RegistryProviderVersions.Read(ctx, versionID)
		require.NoError(t, err)
		assert.Equal(t, version.ID, readVersion.ID)
		assert.Equal(t, version.Version, readVersion.Version)
		assert.Equal(t, version.KeyID, readVersion.KeyID)

		t.Run("relationships are properly decoded", func(t *testing.T) {
			assert.Equal(t, version.RegistryProvider.ID, readVersion.RegistryProvider.ID)
		})

		t.Run("timestamps are properly decoded", func(t *testing.T) {
			assert.NotEmpty(t, readVersion.CreatedAt)
			assert.NotEmpty(t, readVersion.UpdatedAt)
		})

		t.Run("includes upload links", func(t *testing.T) {
			expectedLinks := []string{
				"shasums-upload",
				"shasums-sig-upload",
			}
			for _, l := range expectedLinks {
				_, ok := readVersion.Links[l].(string)
				assert.True(t, ok, "Expect upload link: %s", l)
			}
		})
	})

	t.Run("with non existing version", func(t *testing.T) {
		provider, providerCleanup := createRegistryProvider(t, client, nil, PrivateRegistry)
		defer providerCleanup()

		versionID := RegistryProviderVersionID{
			RegistryProviderID: RegistryProviderID{
				OrganizationName: provider.Organization.Name,
				RegistryName:     provider.RegistryName,
				Namespace:        provider.Namespace,
				Name:             provider.Name,
			},
			Version: "1.0.0",
		}

		_, err := client.RegistryProviderVersions.Read(ctx, versionID)
		assert.Error(t, err)
	})

}
