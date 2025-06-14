// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package tfe

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRegistryNoCodeModulesCreate(t *testing.T) {
	skipUnlessBeta(t)
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	defer orgTestCleanup()

	t.Run("with valid options", func(t *testing.T) {
		t.Run("with no version given", func(t *testing.T) {
			registryModuleTest, registryModuleTestCleanup := createRegistryModule(t, client, orgTest, PrivateRegistry)
			defer registryModuleTestCleanup()

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

			ncOptions := RegistryNoCodeModuleCreateOptions{
				RegistryModule: registryModuleTest,
			}

			noCodeModule, err := client.RegistryNoCodeModules.Create(ctx, orgTest.Name, ncOptions)
			require.NoError(t, err)
			assert.NotEmpty(t, noCodeModule.ID)
			require.NotEmpty(t, noCodeModule.Organization)
			assert.True(t, noCodeModule.Enabled)
			require.NotEmpty(t, noCodeModule.RegistryModule)
			assert.Equal(t, orgTest.Name, noCodeModule.Organization.Name)
			assert.Equal(t, registryModuleTest.ID, noCodeModule.RegistryModule.ID)
		})
		t.Run("with version pin given", func(t *testing.T) {
			registryModuleTest, _ := createRegistryModule(t, client, orgTest, PrivateRegistry)

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

			ncOptions := RegistryNoCodeModuleCreateOptions{
				VersionPin:     "1.2.3",
				RegistryModule: registryModuleTest,
			}

			noCodeModule, err := client.RegistryNoCodeModules.Create(ctx, orgTest.Name, ncOptions)
			require.NoError(t, err)
			assert.NotEmpty(t, noCodeModule.ID)
			require.NotEmpty(t, noCodeModule.Organization)
			require.NotEmpty(t, noCodeModule.RegistryModule)
			assert.True(t, noCodeModule.Enabled)
			assert.Equal(t, ncOptions.VersionPin, noCodeModule.VersionPin)
			assert.Equal(t, orgTest.Name, noCodeModule.Organization.Name)
			assert.Equal(t, registryModuleTest.ID, noCodeModule.RegistryModule.ID)
		})
		t.Run("with enabled set to false", func(t *testing.T) {
			registryModuleTest, _ := createRegistryModuleWithVersion(t, client, orgTest)

			ncOptions := RegistryNoCodeModuleCreateOptions{
				RegistryModule: registryModuleTest,
				Enabled:        Bool(false),
			}

			noCodeModule, err := client.RegistryNoCodeModules.Create(ctx, orgTest.Name, ncOptions)
			require.NoError(t, err)
			assert.NotEmpty(t, noCodeModule.ID)
			require.NotEmpty(t, noCodeModule.Organization)
			require.NotEmpty(t, noCodeModule.RegistryModule)
			assert.False(t, noCodeModule.Enabled)
			assert.Equal(t, ncOptions.VersionPin, noCodeModule.VersionPin)
			assert.Equal(t, orgTest.Name, noCodeModule.Organization.Name)
			assert.Equal(t, registryModuleTest.ID, noCodeModule.RegistryModule.ID)
		})
	})

	t.Run("with invalid options", func(t *testing.T) {
		registryModuleTest, registryModuleTestCleanup := createRegistryModule(t, client, orgTest, PrivateRegistry)
		defer registryModuleTestCleanup()

		t.Run("with version pinned to one that does not exist", func(t *testing.T) {
			options := RegistryNoCodeModuleCreateOptions{
				VersionPin:     "1.2.5",
				RegistryModule: registryModuleTest,
			}

			noCodeModule, err := client.RegistryNoCodeModules.Create(ctx, orgTest.Name, options)
			require.Error(t, err)
			require.Nil(t, noCodeModule)
		})
	})

	t.Run("with variable options", func(t *testing.T) {
		registryModuleTest, registryModuleTestCleanup := createRegistryModule(t, client, orgTest, PrivateRegistry)
		defer registryModuleTestCleanup()

		options := RegistryNoCodeModuleCreateOptions{
			RegistryModule: registryModuleTest,
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

		noCodeModule, err := client.RegistryNoCodeModules.Create(ctx, orgTest.Name, options)
		require.NoError(t, err)
		assert.NotEmpty(t, noCodeModule.ID)
		require.NotEmpty(t, noCodeModule.Organization)
		require.NotEmpty(t, noCodeModule.RegistryModule)
		require.True(t, noCodeModule.Enabled)
		assert.Equal(t, orgTest.Name, noCodeModule.Organization.Name)
		assert.Equal(t, registryModuleTest.ID, noCodeModule.RegistryModule.ID)
		assert.Equal(t, len(options.VariableOptions), len(noCodeModule.VariableOptions))
	})
}

func TestRegistryNoCodeModulesRead(t *testing.T) {
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

		ncm, err := client.RegistryNoCodeModules.Read(ctx, noCodeModule.ID, nil)
		require.NoError(t, err)
		assert.Equal(t, noCodeModule.ID, ncm.ID)
		assert.True(t, noCodeModule.Enabled)
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

		ncm, err := client.RegistryNoCodeModules.Read(ctx, noCodeModule.ID, &RegistryNoCodeModuleReadOptions{
			Include: []RegistryNoCodeModuleIncludeOpt{RegistryNoCodeIncludeVariableOptions},
		})
		require.NoError(t, err)
		assert.Equal(t, noCodeModule.ID, ncm.ID)
		assert.True(t, noCodeModule.Enabled)
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
		ncm, err := client.RegistryNoCodeModules.Read(ctx, "non-existing", nil)
		assert.Nil(t, ncm)
		assert.Equal(t, err, ErrResourceNotFound)
	})
}

// TestRegistryNoCodeModuleReadVariables tests the ReadVariables method of the
// RegistryNoCodeModules service.
//
// This test requires that the environment variable "GITHUB_REGISTRY_NO_CODE_MODULE_IDENTIFIER" is set
// with the ID of an existing no-code module that has variables.
func TestRegistryNoCodeModulesReadVariables(t *testing.T) {
	skipUnlessBeta(t)
	client := testClient(t)
	ctx := context.Background()
	r := require.New(t)

	ncmID := os.Getenv("GITHUB_REGISTRY_NO_CODE_MODULE_IDENTIFIER")
	if ncmID == "" {
		t.Skip("Export a valid GITHUB_REGISTRY_NO_CODE_MODULE_IDENTIFIER before running this test")
	}

	ncm, err := client.RegistryNoCodeModules.Read(ctx, ncmID, nil)
	r.NoError(err)
	r.NotNil(ncm)

	t.Run("happy path", func(t *testing.T) {
		vars, err := client.RegistryNoCodeModules.ReadVariables(ctx, ncm.ID, ncm.VersionPin, &RegistryNoCodeModuleReadVariablesOptions{})
		r.NoError(err)
		r.NotNil(vars)
		r.NotEmpty(vars)
	})
}

func TestRegistryNoCodeModulesUpdate(t *testing.T) {
	skipUnlessBeta(t)
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	defer orgTestCleanup()

	registryModuleTest, registryModuleTestCleanup := createRegistryModule(t, client, orgTest, PrivateRegistry)
	defer registryModuleTestCleanup()

	t.Run("update no-code registry module", func(t *testing.T) {
		noCodeModule, noCodeModuleCleanup := createNoCodeRegistryModule(t, client, orgTest.Name, registryModuleTest, nil)
		defer noCodeModuleCleanup()

		assert.True(t, noCodeModule.Enabled)

		options := RegistryNoCodeModuleUpdateOptions{
			RegistryModule: &RegistryModule{ID: registryModuleTest.ID},
			Enabled:        Bool(false),
		}
		updated, err := client.RegistryNoCodeModules.Update(ctx, noCodeModule.ID, options)
		require.NoError(t, err)
		assert.False(t, updated.Enabled)
	})
	t.Run("no changes when no options are set", func(t *testing.T) {
		noCodeModule, noCodeModuleCleanup := createNoCodeRegistryModule(t, client, orgTest.Name, registryModuleTest, nil)
		defer noCodeModuleCleanup()

		options := RegistryNoCodeModuleUpdateOptions{
			RegistryModule: &RegistryModule{ID: registryModuleTest.ID},
		}
		updated, err := client.RegistryNoCodeModules.Update(ctx, noCodeModule.ID, options)
		require.NoError(t, err)
		assert.Equal(t, *noCodeModule, *updated)
	})
}

func TestRegistryNoCodeModulesDelete(t *testing.T) {
	skipUnlessBeta(t)
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	defer orgTestCleanup()

	registryModuleTest, registryModuleTestCleanup := createRegistryModule(t, client, orgTest, PrivateRegistry)
	defer registryModuleTestCleanup()

	t.Run("with valid ID", func(t *testing.T) {
		noCodeModule, _ := createNoCodeRegistryModule(t, client, orgTest.Name, registryModuleTest, nil)

		err := client.RegistryNoCodeModules.Delete(ctx, noCodeModule.ID)
		require.NoError(t, err)

		rm, err := client.RegistryNoCodeModules.Read(ctx, noCodeModule.ID, nil)
		assert.Nil(t, rm)
		assert.Error(t, err)
	})

	t.Run("without an ID", func(t *testing.T) {
		err := client.RegistryNoCodeModules.Delete(ctx, "")
		assert.EqualError(t, err, ErrInvalidModuleID.Error())
	})

	t.Run("with an invalid ID", func(t *testing.T) {
		err := client.RegistryNoCodeModules.Delete(ctx, "invalid")
		assert.EqualError(t, err, ErrResourceNotFound.Error())
	})
}

func createNoCodeRegistryModule(t *testing.T, client *Client, orgName string, rm *RegistryModule, variables []*NoCodeVariableOption) (*RegistryNoCodeModule, func()) {
	options := RegistryNoCodeModuleCreateOptions{
		RegistryModule:  rm,
		VariableOptions: variables,
	}

	ctx := context.Background()

	ncm, err := client.RegistryNoCodeModules.Create(ctx, orgName, options)
	require.NoError(t, err)
	require.NotEmpty(t, ncm)
	return ncm, func() {
		if err := client.RegistryNoCodeModules.Delete(ctx, ncm.ID); err != nil {
			t.Errorf("Error destroying no-code registry module! WARNING: Dangling resources\n"+
				"may exist! The full error is shown below.\n\n"+
				"NoCode Module: %s\nError: %s", ncm.ID, err)
		}
	}
}

func TestRegistryNoCodeModulesCreateWorkspace(t *testing.T) {
	skipUnlessBeta(t)
	client := testClient(t)
	ctx := context.Background()
	r := require.New(t)

	// create an org that will be deleted later. the wskp will live here
	orgTest, orgTestCleanup := createOrganization(t, client)
	defer orgTestCleanup()

	org, err := client.Organizations.Read(ctx, orgTest.Name)
	r.NoError(err)
	r.NotNil(org)

	githubIdentifier := os.Getenv("GITHUB_REGISTRY_NO_CODE_MODULE_IDENTIFIER")
	if githubIdentifier == "" {
		t.Skip("Export a valid GITHUB_REGISTRY_NO_CODE_MODULE_IDENTIFIER before running this test")
	}

	token, cleanupToken := createOAuthToken(t, client, org)
	defer cleanupToken()

	rmOpts := RegistryModuleCreateWithVCSConnectionOptions{
		VCSRepo: &RegistryModuleVCSRepoOptions{
			OrganizationName:  String(org.Name),
			Identifier:        String(githubIdentifier),
			Tags:              Bool(true),
			OAuthTokenID:      String(token.ID),
			DisplayIdentifier: String(githubIdentifier),
		},
	}

	rm, err := client.RegistryModules.CreateWithVCSConnection(ctx, rmOpts)
	r.NoError(err)

	// 1. create the registry module
	// 2. create the no-code module, with the registry module
	// 3. use the ID to create the workspace
	ncm, err := client.RegistryNoCodeModules.Create(ctx, org.Name, RegistryNoCodeModuleCreateOptions{
		RegistryModule:  rm,
		Enabled:         Bool(true),
		VariableOptions: nil,
	})
	r.NoError(err)
	r.NotNil(ncm)

	// We sleep for 10 seconds to let the module finish getting ready
	time.Sleep(time.Second * 10)

	t.Run("test creating a workspace via a no-code module", func(t *testing.T) {
		wn := fmt.Sprintf("foo-%s", randomString(t))
		sn := "my-app"
		su := "http://my-app.com"
		w, err := client.RegistryNoCodeModules.CreateWorkspace(
			ctx,
			ncm.ID,
			&RegistryNoCodeModuleCreateWorkspaceOptions{
				Name:          wn,
				SourceName:    String(sn),
				SourceURL:     String(su),
				ExecutionMode: String("remote"),
			},
		)
		r.NoError(err)
		r.Equal(wn, w.Name)
		r.Equal(sn, w.SourceName)
		r.Equal(su, w.SourceURL)
		r.Equal("remote", w.ExecutionMode)
	})

	t.Run("fail to create a workspace with a bad module ID", func(t *testing.T) {
		wn := fmt.Sprintf("foo-%s", randomString(t))
		_, err = client.RegistryNoCodeModules.CreateWorkspace(
			ctx,
			"codeno-abc123XYZ",
			&RegistryNoCodeModuleCreateWorkspaceOptions{
				Name: wn,
			},
		)
		r.Error(err)
	})
}

func TestRegistryNoCodeModuleWorkspaceUpgrade(t *testing.T) {
	skipUnlessBeta(t)

	client := testClient(t)
	ctx := context.Background()
	r := require.New(t)

	orgTest, orgTestCleanup := createOrganization(t, client)
	defer orgTestCleanup()

	org, err := client.Organizations.Read(ctx, orgTest.Name)
	r.NoError(err)
	r.NotNil(org)

	githubIdentifier := os.Getenv("GITHUB_REGISTRY_NO_CODE_MODULE_IDENTIFIER")
	if githubIdentifier == "" {
		t.Skip("Export a valid GITHUB_REGISTRY_NO_CODE_MODULE_IDENTIFIER before running this test")
	}

	token, cleanupToken := createOAuthToken(t, client, org)
	defer cleanupToken()

	rmOpts := RegistryModuleCreateWithVCSConnectionOptions{
		VCSRepo: &RegistryModuleVCSRepoOptions{
			OrganizationName:  String(org.Name),
			Identifier:        String(githubIdentifier),
			Tags:              Bool(true),
			OAuthTokenID:      String(token.ID),
			DisplayIdentifier: String(githubIdentifier),
		},
		InitialVersion: String("1.0.0"),
	}

	// create the module
	rm, err := client.RegistryModules.CreateWithVCSConnection(ctx, rmOpts)
	r.NoError(err)

	// create the no-code module
	ncm, err := client.RegistryNoCodeModules.Create(ctx, org.Name, RegistryNoCodeModuleCreateOptions{
		RegistryModule:  rm,
		Enabled:         Bool(true),
		VariableOptions: nil,
	})
	r.NoError(err)
	r.NotNil(ncm)

	// We sleep for 10 seconds to let the module finish getting ready
	time.Sleep(time.Second * 10)

	// update the module's pinned version to be 1.0.0
	// NOTE: This is done here as an update instead of at create time, because
	// that results in the following error:
	// Validation failed: Provided version pin is not equal to latest or provided
	// string does not represent an existing version of the module.
	uncm, err := client.RegistryNoCodeModules.Update(ctx, ncm.ID, RegistryNoCodeModuleUpdateOptions{
		RegistryModule: rm,
		VersionPin:     "1.0.0",
	})
	r.NoError(err)
	r.NotNil(uncm)

	// create a workspace, which will be attempted to be updated during the test
	wn := fmt.Sprintf("foo-%s", randomString(t))
	sn := "my-app"
	su := "http://my-app.com"
	w, err := client.RegistryNoCodeModules.CreateWorkspace(
		ctx,
		uncm.ID,
		&RegistryNoCodeModuleCreateWorkspaceOptions{
			Name:       wn,
			SourceName: String(sn),
			SourceURL:  String(su),
		},
	)
	r.NoError(err)
	r.NotNil(w)

	// update the module's pinned version
	uncm, err = client.RegistryNoCodeModules.Update(ctx, ncm.ID, RegistryNoCodeModuleUpdateOptions{
		VersionPin: "1.0.1",
	})
	r.NoError(err)
	r.NotNil(uncm)

	t.Run("test upgrading a workspace via a no-code module", func(t *testing.T) {
		wu, err := client.RegistryNoCodeModules.UpgradeWorkspace(
			ctx,
			ncm.ID,
			w.ID,
			&RegistryNoCodeModuleUpgradeWorkspaceOptions{},
		)
		r.NoError(err)
		r.NotNil(wu)
		r.NotEmpty(wu.Status)
		r.NotEmpty(wu.PlanURL)
	})

	t.Run("fail to upgrade workspace with invalid no-code module", func(t *testing.T) {
		_, err = client.RegistryNoCodeModules.UpgradeWorkspace(
			ctx,
			ncm.ID+"-invalid",
			w.ID,
			&RegistryNoCodeModuleUpgradeWorkspaceOptions{},
		)
		r.Error(err)
	})

	t.Run("fail to upgrade workspace with invalid workspace ID", func(t *testing.T) {
		_, err = client.RegistryNoCodeModules.UpgradeWorkspace(
			ctx,
			ncm.ID,
			w.ID+"-invalid",
			&RegistryNoCodeModuleUpgradeWorkspaceOptions{},
		)
		r.Error(err)
	})
}
