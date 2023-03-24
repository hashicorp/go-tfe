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

func TestNoCodeRegistryModulesRead(t *testing.T) {
	skipUnlessBeta(t)
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	defer orgTestCleanup()

	registryModuleTest, registryModuleTestCleanup := createRegistryModule(t, client, orgTest, PrivateRegistry)
	defer registryModuleTestCleanup()

	t.Run("with valid ID", func(t *testing.T) {
		noCodeModule, noCodeModuleCleanup := createNoCodeRegistryModule(t, client, orgTest.Name, registryModuleTest, nil)
		defer noCodeModuleCleanup()

		ncm, err := client.NoCodeRegistryModules.Read(ctx, RegistryNoCodeModuleReadOptions{
			ID: noCodeModule.ID,
		})
		require.NoError(t, err)
		assert.Equal(t, noCodeModule.ID, ncm.ID)
		assert.Equal(t, noCodeModule.FollowLatestVersion, ncm.FollowLatestVersion)
		assert.Equal(t, noCodeModule.Enabled, ncm.Enabled)
		assert.Equal(t, noCodeModule.Organization.Name, ncm.Organization.Name)
		assert.Equal(t, noCodeModule.RegistryModule.ID, ncm.RegistryModule.ID)
	})

	t.Run("when the variable-options is included in the params", func(t *testing.T) {
		varOpts := []*NoCodeVariableOption{
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
		}
		noCodeModule, noCodeModuleCleanup := createNoCodeRegistryModule(t, client, orgTest.Name, registryModuleTest, varOpts)
		defer noCodeModuleCleanup()

		ncm, err := client.NoCodeRegistryModules.Read(ctx, RegistryNoCodeModuleReadOptions{
			ID:      noCodeModule.ID,
			Include: []NoCodeReadIncludeOpt{NoCodeIncludeVariableOptions},
		})
		require.NoError(t, err)
		assert.Equal(t, noCodeModule.ID, ncm.ID)
		assert.Equal(t, noCodeModule.FollowLatestVersion, ncm.FollowLatestVersion)
		assert.Equal(t, noCodeModule.Enabled, ncm.Enabled)
		assert.Equal(t, noCodeModule.Organization.Name, ncm.Organization.Name)
		assert.Equal(t, noCodeModule.RegistryModule.ID, ncm.RegistryModule.ID)

		assert.Equal(t, len(varOpts), len(ncm.VariableOptions))
		for i, opt := range varOpts {
			assert.Equal(t, opt.VariableName, ncm.VariableOptions[i].VariableName)
			assert.Equal(t, opt.VariableType, ncm.VariableOptions[i].VariableType)
			assert.Equal(t, opt.Options, ncm.VariableOptions[i].Options)
		}
	})

	t.Run("when the id does not exist", func(t *testing.T) {
		ncm, err := client.NoCodeRegistryModules.Read(ctx, RegistryNoCodeModuleReadOptions{
			ID: "non-existing",
		})
		assert.Nil(t, ncm)
		assert.Equal(t, err, ErrResourceNotFound)
	})
}

func createNoCodeRegistryModule(t *testing.T, client *Client, orgName string, rm *RegistryModule, variables []*NoCodeVariableOption) (*RegistryNoCodeModule, func()) {
	options := RegistryNoCodeModuleCreateOptions{
		FollowLatestVersion: Bool(true),
		Enabled:             Bool(true),
		RegistryModule:      rm,
		VariableOptions:     variables,
	}

	ctx := context.Background()

	ncm, err := client.NoCodeRegistryModules.Create(ctx, orgName, options)
	require.NoError(t, err)
	require.NotEmpty(t, ncm)
	return ncm, func() {
		//TODO
	}
}
