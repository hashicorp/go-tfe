// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package tfe

import (
	"context"
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
			OS:       "linux",
			Arch:     "amd64",
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
		assert.False(t, rpp.ProviderBinaryUploaded)

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
				Arch:     "amd64",
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
				OS:       "linux",
				Arch:     "amd64",
				Shasum:   "",
				Filename: "filename",
			}

			sadRpp, err := client.RegistryProviderPlatforms.Create(ctx, versionID, options)

			assert.Nil(t, sadRpp)
			assert.EqualError(t, err, ErrRequiredShasum.Error())
		})

		t.Run("without a filename", func(t *testing.T) {
			options := RegistryProviderPlatformCreateOptions{
				OS:       "linux",
				Arch:     "amd64",
				Shasum:   "shasum",
				Filename: "",
			}

			sadRpp, err := client.RegistryProviderPlatforms.Create(ctx, versionID, options)

			assert.Nil(t, sadRpp)
			assert.EqualError(t, err, ErrRequiredFilename.Error())
		})

		t.Run("with a public provider", func(t *testing.T) {
			options := RegistryProviderPlatformCreateOptions{
				OS:       "linux",
				Arch:     "amd64",
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
				OS:       "linux",
				Arch:     "amd64",
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
		platform, _ := createRegistryProviderPlatform(t, client, provider, version, "", "")

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
			OS:                        "linux",
			Arch:                      "amd64",
		}

		err := client.RegistryProviderPlatforms.Delete(ctx, platformID)
		assert.Error(t, err)
	})
}

func TestRegistryProviderPlatformsRead(t *testing.T) {
	t.Skip()

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

	platform, platformCleanup := createRegistryProviderPlatform(t, client, provider, version, "", "")
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
		assert.Equal(t, platform.ProviderBinaryUploaded, readPlatform.ProviderBinaryUploaded)

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
	t.Skip()

	client := testClient(t)
	ctx := context.Background()

	t.Run("with platforms", func(t *testing.T) {
		provider, providerCleanup := createRegistryProvider(t, client, nil, PrivateRegistry)
		defer providerCleanup()

		version, versionCleanup := createRegistryProviderVersion(t, client, provider)
		defer versionCleanup()

		osl := []string{"linux", "darwin", "windows"}
		archl := []string{"amd64", "arm64", "amd64"}

		platforms := make([]*RegistryProviderPlatform, 0)
		for i, os := range osl {
			platform, _ := createRegistryProviderPlatform(t, client, provider, version, os, archl[i])
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

		t.Run("with no list options", func(t *testing.T) {
			returnedPlatforms, err := client.RegistryProviderPlatforms.List(ctx, versionID, nil)
			require.NoError(t, err)

			require.Len(t, returnedPlatforms.Items, numPlatforms)
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

		t.Run("with list options", func(t *testing.T) {
			returnedPlatforms, err := client.RegistryProviderPlatforms.List(ctx, versionID, &RegistryProviderPlatformListOptions{
				ListOptions: ListOptions{
					PageNumber: 999,
					PageSize:   100,
				},
			})
			require.NoError(t, err)

			require.Len(t, returnedPlatforms.Items, 0)
			assert.Equal(t, 999, returnedPlatforms.CurrentPage)
			assert.Equal(t, numPlatforms, returnedPlatforms.TotalCount)
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
