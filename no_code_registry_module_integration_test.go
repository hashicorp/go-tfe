// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package tfe

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNoCodeRegistryModulesCreate(t *testing.T) {
	skipUnlessBeta(t)
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	defer orgTestCleanup()

	registryModuleTest, registryModuleTestCleanup := createRegistryModule(t, client, orgTest, PrivateRegistry)
	defer registryModuleTestCleanup()

	t.Run("with valid options", func(t *testing.T) {
		t.Run("with follow-latest-version and enabled", func(t *testing.T) {
			options := RegistryNoCodeModuleCreateOptions{
				FollowLatestVersion: Bool(true),
				Enabled:             Bool(true),
				RegistryModule:      registryModuleTest,
			}

			noCodeModule, err := client.NoCodeRegistryModules.Create(ctx, orgTest.Name, options)
			require.NoError(t, err)
			assert.NotEmpty(t, noCodeModule.ID)
			require.NotEmpty(t, noCodeModule.Organization)
			require.NotEmpty(t, noCodeModule.RegistryModule)
			assert.Equal(t, *options.FollowLatestVersion, noCodeModule.FollowLatestVersion)
			assert.Equal(t, *options.Enabled, noCodeModule.Enabled)
			assert.Equal(t, orgTest.Name, noCodeModule.Organization.Name)
			assert.Equal(t, noCodeModule.RegistryModule.ID, noCodeModule.RegistryModule.ID)
		})
	})

	t.Run("with invalid options", func(t *testing.T) {
		t.Run("with enabled not present", func(t *testing.T) {
			options := RegistryNoCodeModuleCreateOptions{
				FollowLatestVersion: Bool(true),
				RegistryModule:      registryModuleTest,
			}

			noCodeModule, err := client.NoCodeRegistryModules.Create(ctx, orgTest.Name, options)
			require.Error(t, err)
			require.Nil(t, noCodeModule)
		})
		t.Run("with follow_latest_version not present", func(t *testing.T) {
			options := RegistryNoCodeModuleCreateOptions{
				Enabled:        Bool(true),
				RegistryModule: registryModuleTest,
			}

			noCodeModule, err := client.NoCodeRegistryModules.Create(ctx, orgTest.Name, options)
			require.Error(t, err)
			require.Nil(t, noCodeModule)
		})
		t.Run("with registry module not present", func(t *testing.T) {
			options := RegistryNoCodeModuleCreateOptions{
				Enabled:             Bool(true),
				FollowLatestVersion: Bool(true),
			}

			noCodeModule, err := client.NoCodeRegistryModules.Create(ctx, orgTest.Name, options)
			require.Error(t, err)
			require.Nil(t, noCodeModule)
		})

		t.Run("with invalid registry module", func(t *testing.T) {
			options := RegistryNoCodeModuleCreateOptions{
				FollowLatestVersion: Bool(true),
				Enabled:             Bool(true),
				RegistryModule:      &RegistryModule{ID: "invalid"},
			}

			noCodeModule, err := client.NoCodeRegistryModules.Create(ctx, orgTest.Name, options)
			assert.Error(t, err)
			assert.Nil(t, noCodeModule)
		})
	})

	t.Run("with variable options", func(t *testing.T) {
		options := RegistryNoCodeModuleCreateOptions{
			FollowLatestVersion: Bool(true),
			Enabled:             Bool(true),
			RegistryModule:      registryModuleTest,
			VariableOptions: []*NoCodeVariableOption{
				{
					VariableName: "var1",
					VariableType: "string",
					Options:      []string{"option1", "option2"},
				},
				{
					VariableName: "my_var",
					VariableType: "string",
					Options:      []string{"my_option1", "my_option2"},
				},
			},
		}

		noCodeModule, err := client.NoCodeRegistryModules.Create(ctx, orgTest.Name, options)
		require.NoError(t, err)
		assert.NotEmpty(t, noCodeModule.ID)
		require.NotEmpty(t, noCodeModule.Organization)
		require.NotEmpty(t, noCodeModule.RegistryModule)
		assert.Equal(t, *options.FollowLatestVersion, noCodeModule.FollowLatestVersion)
		assert.Equal(t, *options.Enabled, noCodeModule.Enabled)
		assert.Equal(t, orgTest.Name, noCodeModule.Organization.Name)
		assert.Equal(t, noCodeModule.RegistryModule.ID, noCodeModule.RegistryModule.ID)
		assert.Equal(t, len(options.VariableOptions), len(noCodeModule.VariableOptions))
	})
}
