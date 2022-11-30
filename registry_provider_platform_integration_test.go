package tfe

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRegistryProviderPlatformsCreate(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	provider, providerTestCleanup := createRegistryProvider(t, client, nil, PrivateRegistry)
	defer providerTestCleanup()

	version, versionCleanup := createRegistryProviderVersion(t, client, provider)
	defer versionCleanup()

	versionID := RegistryProviderVersionID{
		RegistryProviderID: RegistryProviderID{
			OrganizationName: provider.Organization.Name,
			RegistryName:     provider.RegistryName,
			Namespace:        provider.Namespace,
			Name:             provider.Name,
		},
		Version: version.Version,
	}

	t.Run("with valid options", func(t *testing.T) {
		options := RegistryProviderPlatformCreateOptions{
			OS:       "foo",
			Arch:     "scrimbles",
			Shasum:   "shasum",
			Filename: "filename",
		}

		rpp, err := client.RegistryProviderPlatforms.Create(ctx, versionID, options)
		require.NoError(t, err)
		assert.NotEmpty(t, rpp.ID)
		assert.Equal(t, options.OS, rpp.OS)
		assert.Equal(t, options.Arch, rpp.Arch)
		assert.Equal(t, options.Shasum, rpp.Shasum)
		assert.Equal(t, options.Filename, rpp.Filename)

		t.Run("relationships are properly decoded", func(t *testing.T) {
			assert.Equal(t, version.ID, rpp.RegistryProviderVersion.ID)
		})

		t.Run("attributes are properly decoded", func(t *testing.T) {
			assert.NotEmpty(t, rpp.Arch)
			assert.NotEmpty(t, rpp.OS)
			assert.NotEmpty(t, rpp.Shasum)
			assert.NotEmpty(t, rpp.Filename)
		})
	})

	t.Run("with invalid options", func(t *testing.T) {
		t.Run("without an OS", func(t *testing.T) {
			options := RegistryProviderPlatformCreateOptions{
				OS:       "",
				Arch:     "scrimbles",
				Shasum:   "shasum",
				Filename: "filename",
			}

			sadRpp, err := client.RegistryProviderPlatforms.Create(ctx, versionID, options)

			assert.Nil(t, sadRpp)
			assert.EqualError(t, err, ErrRequiredOS.Error())
		})

		t.Run("without an arch", func(t *testing.T) {
			options := RegistryProviderPlatformCreateOptions{
				OS:       "os",
				Arch:     "",
				Shasum:   "shasum",
				Filename: "filename",
			}

			sadRpp, err := client.RegistryProviderPlatforms.Create(ctx, versionID, options)

			assert.Nil(t, sadRpp)
			assert.EqualError(t, err, ErrRequiredArch.Error())
		})

		t.Run("without a shasum", func(t *testing.T) {
			options := RegistryProviderPlatformCreateOptions{
				OS:       "os",
				Arch:     "scrimbles",
				Shasum:   "",
				Filename: "filename",
			}

			sadRpp, err := client.RegistryProviderPlatforms.Create(ctx, versionID, options)

			assert.Nil(t, sadRpp)
			assert.EqualError(t, err, ErrRequiredShasum.Error())
		})

		t.Run("without a filename", func(t *testing.T) {
			options := RegistryProviderPlatformCreateOptions{
				OS:       "os",
				Arch:     "scrimbles",
				Shasum:   "shasum",
				Filename: "",
			}

			sadRpp, err := client.RegistryProviderPlatforms.Create(ctx, versionID, options)

			assert.Nil(t, sadRpp)
			assert.EqualError(t, err, ErrRequiredFilename.Error())
		})

		t.Run("with a public provider", func(t *testing.T) {
			options := RegistryProviderPlatformCreateOptions{
				OS:       "os",
				Arch:     "scrimbles",
				Shasum:   "shasum",
				Filename: "filename",
			}

			versionID = RegistryProviderVersionID{
				RegistryProviderID: RegistryProviderID{
					OrganizationName: provider.Organization.Name,
					RegistryName:     PublicRegistry,
					Namespace:        provider.Namespace,
					Name:             provider.Name,
				},
				Version: version.Version,
			}

			rm, err := client.RegistryProviderPlatforms.Create(ctx, versionID, options)
			assert.Nil(t, rm)
			assert.EqualError(t, err, ErrRequiredPrivateRegistry.Error())
		})

		t.Run("without a valid registry provider version id", func(t *testing.T) {
			options := RegistryProviderPlatformCreateOptions{
				OS:       "os",
				Arch:     "scrimbles",
				Shasum:   "shasum",
				Filename: "filename",
			}

			versionID = RegistryProviderVersionID{
				RegistryProviderID: RegistryProviderID{
					OrganizationName: badIdentifier,
					RegistryName:     provider.RegistryName,
					Namespace:        provider.Namespace,
					Name:             provider.Name,
				},
				Version: version.Version,
			}

			rm, err := client.RegistryProviderPlatforms.Create(ctx, versionID, options)
			assert.Nil(t, rm)
			assert.EqualError(t, err, ErrInvalidOrg.Error())
		})
	})
}

func TestRegistryProviderPlatformsDelete(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	provider, providerCleanup := createRegistryProvider(t, client, nil, PrivateRegistry)
	defer providerCleanup()

	version, versionCleanup := createRegistryProviderVersion(t, client, provider)
	defer versionCleanup()

	versionID := RegistryProviderVersionID{
		RegistryProviderID: RegistryProviderID{
			OrganizationName: provider.Organization.Name,
			RegistryName:     provider.RegistryName,
			Namespace:        provider.Namespace,
			Name:             provider.Name,
		},
		Version: version.Version,
	}

	t.Run("with a valid version", func(t *testing.T) {
		platform, _ := createRegistryProviderPlatform(t, client, provider, version)

		platformID := RegistryProviderPlatformID{
			RegistryProviderVersionID: versionID,
			OS:                        platform.OS,
			Arch:                      platform.Arch,
		}

		err := client.RegistryProviderPlatforms.Delete(ctx, platformID)
		require.NoError(t, err)
	})

	t.Run("with a non-existent version", func(t *testing.T) {
		platformID := RegistryProviderPlatformID{
			RegistryProviderVersionID: versionID,
			OS:                        "nope",
			Arch:                      "no",
		}

		err := client.RegistryProviderPlatforms.Delete(ctx, platformID)
		assert.Error(t, err)
	})
}

func TestRegistryProviderPlatformsRead(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	provider, providerCleanup := createRegistryProvider(t, client, nil, PrivateRegistry)
	defer providerCleanup()

	providerID := RegistryProviderID{
		OrganizationName: provider.Organization.Name,
		Namespace:        provider.Namespace,
		Name:             provider.Name,
		RegistryName:     provider.RegistryName,
	}

	version, versionCleanup := createRegistryProviderVersion(t, client, provider)
	defer versionCleanup()

	versionID := RegistryProviderVersionID{
		RegistryProviderID: providerID,
		Version:            version.Version,
	}

	platform, platformCleanup := createRegistryProviderPlatform(t, client, provider, version)
	defer platformCleanup()

	t.Run("with valid platform", func(t *testing.T) {
		platformID := RegistryProviderPlatformID{
			RegistryProviderVersionID: versionID,
			OS:                        platform.OS,
			Arch:                      platform.Arch,
		}

		readPlatform, err := client.RegistryProviderPlatforms.Read(ctx, platformID)
		require.NoError(t, err)

		assert.Equal(t, platformID.OS, readPlatform.OS)
		assert.Equal(t, platformID.Arch, readPlatform.Arch)
		assert.Equal(t, platform.Filename, readPlatform.Filename)
		assert.Equal(t, platform.Shasum, readPlatform.Shasum)

		t.Run("relationships are properly decoded", func(t *testing.T) {
			assert.Equal(t, platform.RegistryProviderVersion.ID, readPlatform.RegistryProviderVersion.ID)
		})

		t.Run("includes provider binary upload link", func(t *testing.T) {
			expectedLinks := []string{
				"provider-binary-upload",
			}
			for _, l := range expectedLinks {
				_, ok := readPlatform.Links[l].(string)
				assert.True(t, ok, "Expect upload link: %s", l)
			}
		})
	})

	t.Run("with non-existent os", func(t *testing.T) {
		platformID := RegistryProviderPlatformID{
			RegistryProviderVersionID: versionID,
			OS:                        "DoesNotExist",
			Arch:                      platform.Arch,
		}

		_, err := client.RegistryProviderPlatforms.Read(ctx, platformID)
		assert.Error(t, err)
	})

	t.Run("with non-existent arch", func(t *testing.T) {
		platformID := RegistryProviderPlatformID{
			RegistryProviderVersionID: versionID,
			OS:                        platform.OS,
			Arch:                      "DoesNotExist",
		}

		_, err := client.RegistryProviderPlatforms.Read(ctx, platformID)
		assert.Error(t, err)
	})
}

func TestRegistryProviderPlatformsList(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	t.Run("with platforms", func(t *testing.T) {
		provider, providerCleanup := createRegistryProvider(t, client, nil, PrivateRegistry)
		defer providerCleanup()

		version, versionCleanup := createRegistryProviderVersion(t, client, provider)
		defer versionCleanup()

		numToCreate := 10
		platforms := make([]*RegistryProviderPlatform, 0)
		for i := 0; i < numToCreate; i++ {
			platform, _ := createRegistryProviderPlatform(t, client, provider, version)
			platforms = append(platforms, platform)
		}
		numPlatforms := len(platforms)

		providerID := RegistryProviderID{
			OrganizationName: provider.Organization.Name,
			Namespace:        provider.Namespace,
			Name:             provider.Name,
			RegistryName:     provider.RegistryName,
		}
		versionID := RegistryProviderVersionID{
			RegistryProviderID: providerID,
			Version:            version.Version,
		}

		t.Run("returns all platforms", func(t *testing.T) {
			returnedPlatforms, err := client.RegistryProviderPlatforms.List(ctx, versionID, &RegistryProviderPlatformListOptions{
				ListOptions: ListOptions{
					PageNumber: 0,
					PageSize:   numPlatforms,
				},
			})
			require.NoError(t, err)
			assert.NotEmpty(t, returnedPlatforms.Items)
			assert.Equal(t, numPlatforms, returnedPlatforms.TotalCount)
			assert.Equal(t, 1, returnedPlatforms.TotalPages)
			for _, rp := range returnedPlatforms.Items {
				foundPlatform := false
				for _, p := range platforms {
					if rp.ID == p.ID {
						foundPlatform = true
						break
					}
				}
				assert.True(t, foundPlatform, "Expected to find platform %s but did not:\nexpected:\n%v\nreturned\n%v", rp.ID, platforms, returnedPlatforms)
			}
		})

		t.Run("returns pages of platforms", func(t *testing.T) {
			numPages := 2
			pageSize := numPlatforms / numPages

			for page := 0; page < numPages; page++ {
				testName := fmt.Sprintf("returns page %d of platforms", page)
				t.Run(testName, func(t *testing.T) {
					returnedPlatforms, err := client.RegistryProviderPlatforms.List(ctx, versionID, &RegistryProviderPlatformListOptions{
						ListOptions: ListOptions{
							PageNumber: page,
							PageSize:   pageSize,
						},
					})
					require.NoError(t, err)
					assert.NotEmpty(t, returnedPlatforms.Items)
					assert.Equal(t, numPlatforms, returnedPlatforms.TotalCount)
					assert.Equal(t, numPages, returnedPlatforms.TotalPages)
					assert.Equal(t, pageSize, len(returnedPlatforms.Items))
					for _, rp := range returnedPlatforms.Items {
						foundPlatform := false
						for _, p := range platforms {
							if rp.ID == p.ID {
								foundPlatform = true
								break
							}
						}
						assert.True(t, foundPlatform, "Expected to find platform %s but did not:\nexpected:\n%v\nreturned\n%v", rp.ID, platforms, returnedPlatforms)
					}
				})
			}
		})
	})

	t.Run("without platforms", func(t *testing.T) {
		provider, providerCleanup := createRegistryProvider(t, client, nil, PrivateRegistry)
		defer providerCleanup()

		version, versionCleanup := createRegistryProviderVersion(t, client, provider)
		defer versionCleanup()

		versionID := RegistryProviderVersionID{
			RegistryProviderID: RegistryProviderID{
				OrganizationName: provider.Organization.Name,
				Namespace:        provider.Namespace,
				Name:             provider.Name,
				RegistryName:     provider.RegistryName,
			},
			Version: version.Version,
		}
		platforms, err := client.RegistryProviderPlatforms.List(ctx, versionID, nil)
		require.NoError(t, err)
		assert.Empty(t, platforms.Items)
		assert.Equal(t, 0, platforms.TotalCount)
		assert.Equal(t, 0, platforms.TotalPages)
	})
}
