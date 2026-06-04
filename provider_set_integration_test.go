// Copyright IBM Corp. 2018, 2026
// SPDX-License-Identifier: MPL-2.0

package tfe

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProviderSetsRead(t *testing.T) {
	skipUnlessBeta(t)
	t.Parallel()
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	defer orgTestCleanup()

	newSubscriptionUpdater(orgTest)

	project := orgTest.DefaultProject
	workspace, workspaceCleanup := createWorkspace(t, client, orgTest)

	// likely a NOOP but added for consistency and future-proofing
	defer workspaceCleanup()

	createOptions := providerSetTestCreateOptionsWithRelationships(
		t,
		[]*Project{project},
		[]*Workspace{workspace},
	)
	ps, err := client.ProviderSets.Create(ctx, orgTest.Name, createOptions)
	require.NoError(t, err)

	t.Run("ReadById", func(t *testing.T) {
		t.Run("with valid provider set id", func(t *testing.T) {
			psRead, err := client.ProviderSets.Read(ctx, ps.ID)
			require.NoError(t, err)

			assert.Equal(t, ps.ID, psRead.ID)
			assert.Equal(t, ps.Name, psRead.Name)
			assert.Equal(t, ps.Description, psRead.Description)
			assert.Equal(t, ps.ProviderSource, psRead.ProviderSource)
			assert.Equal(t, ps.ConfigurationHcl, psRead.ConfigurationHcl)

			require.Len(t, psRead.Projects, 1)
			require.Len(t, psRead.Workspaces, 1)
			assert.Equal(t, psRead.Projects[0].ID, createOptions.Projects[0].ID)
			assert.Equal(t, psRead.Workspaces[0].ID, createOptions.Workspaces[0].ID)
		})

		t.Run("with invalid provider set ID", func(t *testing.T) {
			psRead, err := client.ProviderSets.Read(ctx, "invalid/id")
			assert.EqualError(t, err, ErrInvalidProviderSetID.Error())
			assert.Nil(t, psRead)
		})

		t.Run("with unexisting provider set ID", func(t *testing.T) {
			psRead, err := client.ProviderSets.Read(ctx, "unexisting-id")
			assert.EqualError(t, err, ErrResourceNotFound.Error())
			assert.Nil(t, psRead)
		})

		t.Run("with empty provider set id", func(t *testing.T) {
			ps, err := client.ProviderSets.Read(ctx, "")
			assert.EqualError(t, err, ErrRequiredProviderSetID.Error())
			assert.Nil(t, ps)
		})
	})

	t.Run("ReadByName", func(t *testing.T) {
		t.Run("with valid provider set name", func(t *testing.T) {
			psRead, err := client.ProviderSets.ReadByName(ctx, orgTest.Name, createOptions.Name)
			require.NoError(t, err)

			assert.Equal(t, ps.ID, psRead.ID)
			assert.Equal(t, ps.Name, psRead.Name)
			assert.Equal(t, ps.Description, psRead.Description)
			assert.Equal(t, ps.ProviderSource, psRead.ProviderSource)
			assert.Equal(t, ps.ConfigurationHcl, psRead.ConfigurationHcl)

			require.Len(t, psRead.Projects, 1)
			require.Len(t, psRead.Workspaces, 1)
			assert.Equal(t, psRead.Projects[0].ID, createOptions.Projects[0].ID)
			assert.Equal(t, psRead.Workspaces[0].ID, createOptions.Workspaces[0].ID)
		})

		t.Run("with unexisting org ID", func(t *testing.T) {
			psRead, err := client.ProviderSets.ReadByName(ctx, "unexisting-id", createOptions.Name)
			assert.EqualError(t, err, ErrResourceNotFound.Error())
			assert.Nil(t, psRead)
		})

		t.Run("with unexisting provider set", func(t *testing.T) {
			psRead, err := client.ProviderSets.ReadByName(ctx, orgTest.Name, "unexisting-provider-set")
			assert.EqualError(t, err, ErrResourceNotFound.Error())
			assert.Nil(t, psRead)
		})

		t.Run("with invalid org ID", func(t *testing.T) {
			ps, err := client.ProviderSets.ReadByName(ctx, "invalid/org", createOptions.Name)
			assert.EqualError(t, err, ErrInvalidOrg.Error())
			assert.Nil(t, ps)
		})

		t.Run("with invalid provider set name", func(t *testing.T) {
			ps, err := client.ProviderSets.ReadByName(ctx, orgTest.Name, "invalid/name")
			assert.EqualError(t, err, ErrInvalidName.Error())
			assert.Nil(t, ps)
		})

		t.Run("with empty provider set name", func(t *testing.T) {
			ps, err := client.ProviderSets.ReadByName(ctx, orgTest.Name, "")
			assert.EqualError(t, err, ErrRequiredName.Error())
			assert.Nil(t, ps)
		})
	})
}

func TestProviderSetsDelete(t *testing.T) {
	skipUnlessBeta(t)
	t.Parallel()
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	defer orgTestCleanup()

	newSubscriptionUpdater(orgTest)

	t.Run("with valid provider id", func(t *testing.T) {
		ps, err := client.ProviderSets.Create(ctx, orgTest.Name, providerSetTestCreateOptions(t))
		require.NoError(t, err)

		err = client.ProviderSets.Delete(ctx, ps.ID)
		assert.NoError(t, err)

		psRead, err := client.ProviderSets.Read(ctx, ps.ID)
		assert.EqualError(t, err, ErrResourceNotFound.Error())
		assert.Nil(t, psRead)
	})

	t.Run("with invalid provider set ID", func(t *testing.T) {
		err := client.ProviderSets.Delete(ctx, "invalid/id")
		assert.EqualError(t, err, ErrInvalidProviderSetID.Error())
	})

	t.Run("with unexisting provider set ID", func(t *testing.T) {
		err := client.ProviderSets.Delete(ctx, "unexisting-id")
		assert.EqualError(t, err, ErrResourceNotFound.Error())
	})
}

func TestProviderSetsCreate(t *testing.T) {
	skipUnlessBeta(t)
	t.Parallel()
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	defer orgTestCleanup()

	newSubscriptionUpdater(orgTest)

	project := orgTest.DefaultProject
	workspace, workspaceCleanup := createWorkspace(t, client, orgTest)

	// likely a NOOP but added for consistency and future-proofing
	defer workspaceCleanup()

	t.Run("with valid attributes", func(t *testing.T) {
		options := providerSetTestCreateOptions(t)

		ps, err := client.ProviderSets.Create(ctx, orgTest.Name, options)
		require.NoError(t, err)

		assert.Equal(t, ps.Name, options.Name)
		assert.Equal(t, ps.Description, *options.Description)
		assert.False(t, ps.Global)
		assert.Equal(t, ps.ProviderSource, options.ProviderSource)
		assert.Equal(t, ps.ConfigurationHcl, options.ConfigurationHcl)
	})

	t.Run("with global valid attributes", func(t *testing.T) {
		options := providerSetTestCreateOptions(t)
		options.Global = Bool(true)

		ps, err := client.ProviderSets.Create(ctx, orgTest.Name, options)
		require.NoError(t, err)

		assert.Equal(t, ps.Name, options.Name)
		assert.Equal(t, ps.Description, *options.Description)
		assert.True(t, ps.Global)
		assert.Equal(t, ps.ProviderSource, options.ProviderSource)
		assert.Equal(t, ps.ConfigurationHcl, options.ConfigurationHcl)
	})

	t.Run("with invalid org ID", func(t *testing.T) {
		ps, err := client.ProviderSets.Create(ctx, "invalid/org", providerSetTestCreateOptions(t))
		assert.EqualError(t, err, ErrInvalidOrg.Error())
		assert.Nil(t, ps)
	})

	t.Run("with non existing org", func(t *testing.T) {
		ps, err := client.ProviderSets.Create(ctx, "some-none-existing-org", providerSetTestCreateOptions(t))
		assert.EqualError(t, err, ErrResourceNotFound.Error())
		assert.Nil(t, ps)
	})

	t.Run("with invalid name", func(t *testing.T) {
		options := providerSetTestCreateOptions(t)
		options.Name = "invalid/name"

		ps, err := client.ProviderSets.Create(ctx, orgTest.Name, options)
		assert.EqualError(t, err, ErrInvalidName.Error())
		assert.Nil(t, ps)
	})

	t.Run("with missing name", func(t *testing.T) {
		options := providerSetTestCreateOptions(t)
		options.Name = ""

		ps, err := client.ProviderSets.Create(ctx, orgTest.Name, options)
		assert.EqualError(t, err, ErrRequiredName.Error())
		assert.Nil(t, ps)
	})

	t.Run("with missing provider source", func(t *testing.T) {
		options := providerSetTestCreateOptions(t)
		options.ProviderSource = ""

		ps, err := client.ProviderSets.Create(ctx, orgTest.Name, options)
		assert.EqualError(t, err, ErrRequiredProviderSource.Error())
		assert.Nil(t, ps)
	})

	t.Run("with missing configuration hcl", func(t *testing.T) {
		options := providerSetTestCreateOptions(t)
		options.ConfigurationHcl = ""

		ps, err := client.ProviderSets.Create(ctx, orgTest.Name, options)
		assert.EqualError(t, err, ErrRequiredConfigurationHcl.Error())
		assert.Nil(t, ps)
	})

	t.Run("with relationships", func(t *testing.T) {
		options := providerSetTestCreateOptionsWithRelationships(
			t,
			[]*Project{project},
			[]*Workspace{workspace},
		)

		ps, err := client.ProviderSets.Create(ctx, orgTest.Name, options)
		require.NoError(t, err)

		assert.Equal(t, ps.Name, options.Name)
		require.Len(t, ps.Projects, 1)
		require.Len(t, ps.Workspaces, 1)
		assert.Equal(t, ps.Projects[0].ID, options.Projects[0].ID)
		assert.Equal(t, ps.Workspaces[0].ID, options.Workspaces[0].ID)
	})

	t.Run("with relationships on a global provider set", func(t *testing.T) {
		options := providerSetTestCreateOptionsWithRelationships(
			t,
			[]*Project{project},
			[]*Workspace{workspace},
		)
		options.Global = Bool(true)

		ps, err := client.ProviderSets.Create(ctx, orgTest.Name, options)
		assert.EqualError(t, err, ErrProviderSetGlobalRelationships.Error())
		assert.Nil(t, ps)
	})

	t.Run("with missing project ID", func(t *testing.T) {
		options := providerSetTestCreateOptionsWithRelationships(
			t,
			[]*Project{{ID: ""}},
			nil,
		)

		ps, err := client.ProviderSets.Create(ctx, orgTest.Name, options)
		assert.EqualError(t, err, ErrRequiredProjectID.Error())
		assert.Nil(t, ps)
	})

	t.Run("with invalid project ID", func(t *testing.T) {
		options := providerSetTestCreateOptionsWithRelationships(
			t,
			[]*Project{{ID: "invalid id"}},
			nil,
		)

		ps, err := client.ProviderSets.Create(ctx, orgTest.Name, options)
		assert.EqualError(t, err, ErrInvalidProjectID.Error())
		assert.Nil(t, ps)
	})

	t.Run("with unexisting project ID", func(t *testing.T) {
		options := providerSetTestCreateOptionsWithRelationships(
			t,
			[]*Project{{ID: "unexisting-id"}},
			nil,
		)

		ps, err := client.ProviderSets.Create(ctx, orgTest.Name, options)
		assert.Contains(t, err.Error(), "Invalid Projects")
		assert.Nil(t, ps)
	})

	t.Run("with missing workspace ID", func(t *testing.T) {
		options := providerSetTestCreateOptionsWithRelationships(
			t,
			nil,
			[]*Workspace{{ID: ""}},
		)

		ps, err := client.ProviderSets.Create(ctx, orgTest.Name, options)
		assert.EqualError(t, err, ErrRequiredWorkspaceID.Error())
		assert.Nil(t, ps)
	})

	t.Run("with invalid workspace ID", func(t *testing.T) {
		options := providerSetTestCreateOptionsWithRelationships(
			t,
			nil,
			[]*Workspace{{ID: "invalid id"}},
		)

		ps, err := client.ProviderSets.Create(ctx, orgTest.Name, options)
		assert.EqualError(t, err, ErrInvalidWorkspaceID.Error())
		assert.Nil(t, ps)
	})

	t.Run("with unexisting workspace ID", func(t *testing.T) {
		options := providerSetTestCreateOptionsWithRelationships(
			t,
			nil,
			[]*Workspace{{ID: "unexisting-id"}},
		)

		ps, err := client.ProviderSets.Create(ctx, orgTest.Name, options)
		assert.Contains(t, err.Error(), "Invalid Workspaces")
		assert.Nil(t, ps)
	})
}

func TestProviderSetsUpdate(t *testing.T) {
	skipUnlessBeta(t)
	t.Parallel()
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	defer orgTestCleanup()

	newSubscriptionUpdater(orgTest)

	project := orgTest.DefaultProject
	workspace, workspaceCleanup := createWorkspace(t, client, orgTest)

	// likely a NOOP but added for consistency and future-proofing
	defer workspaceCleanup()

	type tc struct {
		initial ProviderSetCreateOptions
		input   ProviderSetUpdateOptions
		ps      *ProviderSet
		err     error
	}

	testCases := map[string]struct {
		initial       func(*testing.T) ProviderSetCreateOptions
		providerSetID *string
		input         ProviderSetUpdateOptions
		test          func(*testing.T, tc)
	}{
		"with valid full attributes": {
			input: ProviderSetUpdateOptions{
				Name:           String(randomString(t)),
				Global:         Bool(true),
				Description:    String("updated description"),
				ProviderSource: String("registry.terraform.io/hashicorp/aws"),
				ConfigurationHcl: String(`
				provider "aws" {
					region = "us-west-2"
				}
			`),
			},
			test: func(t *testing.T, tc tc) {
				require.NoError(t, tc.err)
				assert.Equal(t, *tc.input.Name, tc.ps.Name)
				assert.Equal(t, *tc.input.Description, tc.ps.Description)
				assert.Equal(t, *tc.input.ProviderSource, tc.ps.ProviderSource)
				assert.Equal(t, *tc.input.ConfigurationHcl, tc.ps.ConfigurationHcl)
				assert.True(t, tc.ps.Global)
			},
		},
		"with only name": {
			input: ProviderSetUpdateOptions{
				Name: String(randomString(t)),
			},
			test: func(t *testing.T, tc tc) {
				require.NoError(t, tc.err)
				assert.Equal(t, *tc.input.Name, tc.ps.Name)
				assert.Equal(t, *tc.initial.Description, tc.ps.Description)
				assert.Equal(t, tc.initial.ProviderSource, tc.ps.ProviderSource)
				assert.Equal(t, tc.initial.ConfigurationHcl, tc.ps.ConfigurationHcl)
				assert.False(t, tc.ps.Global)
			},
		},
		"with only description": {
			input: ProviderSetUpdateOptions{
				Description: String("updated description"),
			},
			test: func(t *testing.T, tc tc) {
				require.NoError(t, tc.err)
				assert.Equal(t, *tc.input.Description, tc.ps.Description)
				assert.Equal(t, tc.initial.Name, tc.ps.Name)
				assert.Equal(t, tc.initial.ProviderSource, tc.ps.ProviderSource)
				assert.Equal(t, tc.initial.ConfigurationHcl, tc.ps.ConfigurationHcl)
				assert.False(t, tc.ps.Global)
			},
		},
		"with only provider source": {
			input: ProviderSetUpdateOptions{
				ProviderSource: String("registry.terraform.io/hashicorp/azurerm"),
			},
			test: func(t *testing.T, tc tc) {
				require.NoError(t, tc.err)
				assert.Equal(t, *tc.input.ProviderSource, tc.ps.ProviderSource)
				assert.Equal(t, tc.initial.Name, tc.ps.Name)
				assert.Equal(t, *tc.initial.Description, tc.ps.Description)
				assert.Equal(t, tc.initial.ConfigurationHcl, tc.ps.ConfigurationHcl)
				assert.False(t, tc.ps.Global)
			},
		},
		"with only configuration hcl": {
			input: ProviderSetUpdateOptions{
				ConfigurationHcl: String(`
			provider "azurerm" {
				resource_provider_registrations = "none"
				features {}
			}
		`),
			},
			test: func(t *testing.T, tc tc) {
				require.NoError(t, tc.err)
				assert.Equal(t, *tc.input.ConfigurationHcl, tc.ps.ConfigurationHcl)
				assert.Equal(t, tc.initial.Name, tc.ps.Name)
				assert.Equal(t, *tc.initial.Description, tc.ps.Description)
				assert.Equal(t, tc.initial.ProviderSource, tc.ps.ProviderSource)
				assert.False(t, tc.ps.Global)
			},
		},
		"with only global true": {
			input: ProviderSetUpdateOptions{
				Global: Bool(true),
			},
			test: func(t *testing.T, tc tc) {
				require.NoError(t, tc.err)
				assert.True(t, tc.ps.Global)
				assert.Equal(t, tc.initial.Name, tc.ps.Name)
				assert.Equal(t, *tc.initial.Description, tc.ps.Description)
				assert.Equal(t, tc.initial.ProviderSource, tc.ps.ProviderSource)
				assert.Equal(t, tc.initial.ConfigurationHcl, tc.ps.ConfigurationHcl)
			},
		},
		"with invalid name": {
			input: ProviderSetUpdateOptions{
				Name: String("invalid/name"),
			},
			test: func(t *testing.T, tc tc) {
				assert.EqualError(t, tc.err, ErrInvalidName.Error())
				assert.Nil(t, tc.ps)
			},
		},
		"with invalid providerSetID": {
			providerSetID: String("invalid/id"),
			test: func(t *testing.T, tc tc) {
				assert.EqualError(t, tc.err, ErrInvalidProviderSetID.Error())
				assert.Nil(t, tc.ps)
			},
		},
		"with unexisting provider set ID": {
			providerSetID: String("unexisting-id"),
			input:         ProviderSetUpdateOptions{},
			test: func(t *testing.T, tc tc) {
				assert.EqualError(t, tc.err, ErrResourceNotFound.Error())
				assert.Nil(t, tc.ps)
			},
		},
		"with relationships": {
			input: ProviderSetUpdateOptions{
				Projects:   []*Project{{ID: project.ID}},
				Workspaces: []*Workspace{{ID: workspace.ID}},
			},
			test: func(t *testing.T, tc tc) {
				require.NoError(t, tc.err)
				require.Len(t, tc.ps.Projects, 1)
				require.Len(t, tc.ps.Workspaces, 1)
				assert.Equal(t, tc.ps.Projects[0].ID, project.ID)
				assert.Equal(t, tc.ps.Workspaces[0].ID, workspace.ID)
			},
		},
		"with empty relationships when has initial relationships": {
			initial: func(t *testing.T) ProviderSetCreateOptions {
				return providerSetTestCreateOptionsWithRelationships(
					t,
					[]*Project{{ID: project.ID}},
					[]*Workspace{{ID: workspace.ID}},
				)
			},
			input: ProviderSetUpdateOptions{
				Projects:   []*Project{},
				Workspaces: []*Workspace{},
			},
			test: func(t *testing.T, tc tc) {
				require.NoError(t, tc.err)
				require.Len(t, tc.ps.Projects, 0)
				require.Len(t, tc.ps.Workspaces, 0)
			},
		},
		"with nil relationships when has initial relationships": {
			initial: func(t *testing.T) ProviderSetCreateOptions {
				return providerSetTestCreateOptionsWithRelationships(
					t,
					[]*Project{{ID: project.ID}},
					[]*Workspace{{ID: workspace.ID}},
				)
			},
			input: ProviderSetUpdateOptions{
				Projects:   nil,
				Workspaces: nil,
			},
			test: func(t *testing.T, tc tc) {
				require.NoError(t, tc.err)
				require.Len(t, tc.ps.Projects, 1)
				require.Len(t, tc.ps.Workspaces, 1)
				assert.Equal(t, tc.ps.Projects[0].ID, project.ID)
				assert.Equal(t, tc.ps.Workspaces[0].ID, workspace.ID)
			},
		},
		"with missing project ID": {
			input: ProviderSetUpdateOptions{
				Projects: []*Project{{ID: ""}},
			},
			test: func(t *testing.T, tc tc) {
				assert.EqualError(t, tc.err, ErrRequiredProjectID.Error())
				assert.Nil(t, tc.ps)
			},
		},
		"with invalid project ID": {
			input: ProviderSetUpdateOptions{
				Projects: []*Project{{ID: "invalid id"}},
			},
			test: func(t *testing.T, tc tc) {
				assert.EqualError(t, tc.err, ErrInvalidProjectID.Error())
				assert.Nil(t, tc.ps)
			},
		},
		"with unexisting project ID": {
			input: ProviderSetUpdateOptions{
				Projects: []*Project{{ID: "unexisting-id"}},
			},
			test: func(t *testing.T, tc tc) {
				assert.Contains(t, tc.err.Error(), "Invalid Projects")
				assert.Nil(t, tc.ps)
			},
		},
		"with missing workspace ID": {
			input: ProviderSetUpdateOptions{
				Workspaces: []*Workspace{{ID: ""}},
			},
			test: func(t *testing.T, tc tc) {
				assert.EqualError(t, tc.err, ErrRequiredWorkspaceID.Error())
				assert.Nil(t, tc.ps)
			},
		},
		"with invalid workspace ID": {
			input: ProviderSetUpdateOptions{
				Workspaces: []*Workspace{{ID: "invalid id"}},
			},
			test: func(t *testing.T, tc tc) {
				assert.EqualError(t, tc.err, ErrInvalidWorkspaceID.Error())
				assert.Nil(t, tc.ps)
			},
		},
		"with unexisting workspace ID": {
			input: ProviderSetUpdateOptions{
				Workspaces: []*Workspace{{ID: "unexisting-id"}},
			},
			test: func(t *testing.T, tc tc) {
				assert.Contains(t, tc.err.Error(), "Invalid Workspaces")
				assert.Nil(t, tc.ps)
			},
		},
	}

	for desc, testCase := range testCases {
		t.Run(desc, func(t *testing.T) {
			initial := providerSetTestCreateOptions(t)
			if testCase.initial != nil {
				initial = testCase.initial(t)
			}
			ps, err := client.ProviderSets.Create(ctx, orgTest.Name, initial)
			require.NoError(t, err)

			id := ps.ID
			if testCase.providerSetID != nil {
				id = *testCase.providerSetID
			}

			psUpdated, err := client.ProviderSets.Update(ctx, id, testCase.input)

			tc := tc{
				input:   testCase.input,
				initial: initial,
				ps:      psUpdated,
				err:     err,
			}
			testCase.test(t, tc)
		})
	}
}

func providerSetTestCreateOptions(t *testing.T) ProviderSetCreateOptions {
	return providerSetTestCreateOptionsWithRelationships(t, nil, nil)
}

func providerSetTestCreateOptionsWithRelationships(
	t *testing.T,
	projects []*Project,
	workspaces []*Workspace,
) ProviderSetCreateOptions {
	return ProviderSetCreateOptions{
		Name:           randomString(t),
		Description:    String("some test provider set"),
		Global:         Bool(false),
		ProviderSource: "registry.terraform.io/hashicorp/aws",
		Projects:       projects,
		Workspaces:     workspaces,

		ConfigurationHcl: `
		provider "aws" {
			region = "us-east-1"
		}
		`,
	}
}
