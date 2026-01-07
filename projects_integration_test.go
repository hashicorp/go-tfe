// Copyright IBM Corp. 2018, 2025
// SPDX-License-Identifier: MPL-2.0

package tfe

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/stretchr/testify/require"

	"github.com/hashicorp/jsonapi"
)

func TestProjectsList(t *testing.T) {
	t.Parallel()
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	t.Cleanup(orgTestCleanup)

	pTest1, pTestCleanup := createProject(t, client, orgTest)
	t.Cleanup(pTestCleanup)

	pTest2, pTestCleanup := createProject(t, client, orgTest)
	t.Cleanup(pTestCleanup)

	t.Run("without list options", func(t *testing.T) {
		pl, err := client.Projects.List(ctx, orgTest.Name, nil)
		require.NoError(t, err)
		assert.Contains(t, pl.Items, pTest1)

		assert.Equal(t, 1, pl.CurrentPage)
		assert.Equal(t, 3, pl.TotalCount)
	})

	t.Run("with pagination list options", func(t *testing.T) {
		pl, err := client.Projects.List(ctx, orgTest.Name, &ProjectListOptions{
			ListOptions: ListOptions{
				PageNumber: 1,
				PageSize:   100,
			},
		})
		require.NoError(t, err)
		assert.Contains(t, pl.Items, pTest1)
		assert.Contains(t, pl.Items, pTest2)
		assert.Equal(t, true, containsProject(pl.Items, "Default Project"))
		assert.Equal(t, 3, len(pl.Items))
	})

	t.Run("with query list option", func(t *testing.T) {
		pl, err := client.Projects.List(ctx, orgTest.Name, &ProjectListOptions{
			Query: "Default",
		})
		require.NoError(t, err)
		assert.Equal(t, true, containsProject(pl.Items, "Default Project"))
		assert.Equal(t, 1, len(pl.Items))
	})

	t.Run("without a valid organization", func(t *testing.T) {
		pl, err := client.Projects.List(ctx, badIdentifier, nil)
		assert.Nil(t, pl)
		assert.EqualError(t, err, ErrInvalidOrg.Error())
	})

	t.Run("when using a tags filter", func(t *testing.T) {
		skipUnlessBeta(t)

		p1, pTestCleanup1 := createProjectWithOptions(t, client, orgTest, ProjectCreateOptions{
			Name: randomStringWithoutSpecialChar(t),
			TagBindings: []*TagBinding{
				{Key: "key1", Value: "value1"},
				{Key: "key2", Value: "value2a"},
			},
		})
		p2, pTestCleanup2 := createProjectWithOptions(t, client, orgTest, ProjectCreateOptions{
			Name: randomStringWithoutSpecialChar(t),
			TagBindings: []*TagBinding{
				{Key: "key2", Value: "value2b"},
				{Key: "key3", Value: "value3"},
			},
		})
		t.Cleanup(pTestCleanup1)
		t.Cleanup(pTestCleanup2)

		// List all the workspaces under the given tag
		pl, err := client.Projects.List(ctx, orgTest.Name, &ProjectListOptions{
			TagBindings: []*TagBinding{
				{Key: "key1"},
			},
		})
		assert.NoError(t, err)
		assert.Len(t, pl.Items, 1)
		assert.Contains(t, pl.Items, p1)

		pl2, err := client.Projects.List(ctx, orgTest.Name, &ProjectListOptions{
			TagBindings: []*TagBinding{
				{Key: "key2"},
			},
		})
		assert.NoError(t, err)
		assert.Len(t, pl2.Items, 2)
		assert.Contains(t, pl2.Items, p1, p2)

		pl3, err := client.Projects.List(ctx, orgTest.Name, &ProjectListOptions{
			TagBindings: []*TagBinding{
				{Key: "key2", Value: "value2b"},
			},
		})
		assert.NoError(t, err)
		assert.Len(t, pl3.Items, 1)
		assert.Contains(t, pl3.Items, p2)
	})

	t.Run("when including effective tags relationship", func(t *testing.T) {
		skipUnlessBeta(t)

		orgTest2, orgTest2Cleanup := createOrganization(t, client)
		t.Cleanup(orgTest2Cleanup)

		_, pTestCleanup1 := createProjectWithOptions(t, client, orgTest2, ProjectCreateOptions{
			Name: randomStringWithoutSpecialChar(t),
			TagBindings: []*TagBinding{
				{Key: "key1", Value: "value1"},
				{Key: "key2", Value: "value2a"},
			},
		})
		t.Cleanup(pTestCleanup1)

		pl, err := client.Projects.List(ctx, orgTest2.Name, &ProjectListOptions{
			Include: []ProjectIncludeOpt{ProjectEffectiveTagBindings},
		})
		require.NoError(t, err)
		require.Len(t, pl.Items, 2)
		require.Len(t, pl.Items[0].EffectiveTagBindings, 2)
		assert.NotEmpty(t, pl.Items[0].EffectiveTagBindings[0].Key)
		assert.NotEmpty(t, pl.Items[0].EffectiveTagBindings[0].Value)
		assert.NotEmpty(t, pl.Items[0].EffectiveTagBindings[1].Key)
		assert.NotEmpty(t, pl.Items[0].EffectiveTagBindings[1].Value)
	})
}

func TestProjectsReadWithOptions(t *testing.T) {
	t.Parallel()
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	t.Cleanup(orgTestCleanup)

	pTest, pTestCleanup := createProjectWithOptions(t, client, orgTest, ProjectCreateOptions{
		Name: "project-with-tags",
		TagBindings: []*TagBinding{
			{Key: "foo", Value: "bar"},
		},
	})
	t.Cleanup(pTestCleanup)

	t.Run("when the project exists", func(t *testing.T) {
		p, err := client.Projects.ReadWithOptions(ctx, pTest.ID, ProjectReadOptions{
			Include: []ProjectIncludeOpt{ProjectEffectiveTagBindings},
		})
		require.NoError(t, err)
		assert.Equal(t, orgTest.Name, p.Organization.Name)

		// Tag data is included
		require.Len(t, p.EffectiveTagBindings, 1)
		assert.Equal(t, "foo", p.EffectiveTagBindings[0].Key)
		assert.Equal(t, "bar", p.EffectiveTagBindings[0].Value)
	})
}

func TestProjectsRead(t *testing.T) {
	t.Parallel()
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	t.Cleanup(orgTestCleanup)

	pTest, pTestCleanup := createProject(t, client, orgTest)
	t.Cleanup(pTestCleanup)

	t.Run("when the project exists", func(t *testing.T) {
		w, err := client.Projects.Read(ctx, pTest.ID)
		require.NoError(t, err)
		assert.Equal(t, pTest, w)
		assert.Equal(t, orgTest.Name, w.Organization.Name)
	})

	t.Run("when the project does not exist", func(t *testing.T) {
		w, err := client.Projects.Read(ctx, "nonexisting")
		assert.Nil(t, w)
		assert.Error(t, err)
	})

	t.Run("without a valid project ID", func(t *testing.T) {
		w, err := client.Projects.Read(ctx, badIdentifier)
		assert.Nil(t, w)
		assert.EqualError(t, err, ErrInvalidProjectID.Error())
	})

	t.Run("with default execution mode of 'agent'", func(t *testing.T) {
		agentPoolTest, agentPoolTestCleanup := createAgentPool(t, client, orgTest)
		t.Cleanup(agentPoolTestCleanup)

		proj, projCleanup := createProjectWithOptions(t, client, orgTest, ProjectCreateOptions{
			Name:                 "project-with-agent-pool",
			DefaultExecutionMode: String("agent"),
			DefaultAgentPoolID:   String(agentPoolTest.ID),
		})
		t.Cleanup(projCleanup)

		t.Run("execution mode and agent pool are properly decoded", func(t *testing.T) {
			assert.Equal(t, "agent", proj.DefaultExecutionMode)
			assert.NotNil(t, proj.DefaultAgentPool)
			assert.Equal(t, proj.DefaultAgentPool.ID, agentPoolTest.ID)
		})
	})

	t.Run("when project is inheriting the default execution mode", func(t *testing.T) {
		defaultExecutionOrgTest, defaultExecutionOrgTestCleanup := createOrganizationWithDefaultAgentPool(t, client)
		t.Cleanup(defaultExecutionOrgTestCleanup)

		options := ProjectCreateOptions{
			Name: fmt.Sprintf("tst-%s", randomString(t)[0:20]),
			SettingOverwrites: &ProjectSettingOverwrites{
				ExecutionMode: Bool(false),
				AgentPool:     Bool(false),
			},
		}

		pDefaultTest, pDefaultTestCleanup := createProjectWithOptions(t, client, defaultExecutionOrgTest, options)
		t.Cleanup(pDefaultTestCleanup)

		t.Run("and project execution mode is default", func(t *testing.T) {
			p, err := client.Projects.Read(ctx, pDefaultTest.ID)
			assert.NoError(t, err)
			assert.NotEmpty(t, p)

			assert.Equal(t, defaultExecutionOrgTest.DefaultExecutionMode, p.DefaultExecutionMode)
			require.NotNil(t, p.SettingOverwrites)
			assert.Equal(t, false, *p.SettingOverwrites.ExecutionMode)
			assert.Equal(t, false, *p.SettingOverwrites.ExecutionMode)
		})
	})
}

func TestProjectsCreate(t *testing.T) {
	t.Parallel()
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	t.Cleanup(orgTestCleanup)

	newSubscriptionUpdater(orgTest).WithBusinessPlan().Update(t)

	t.Run("with valid options", func(t *testing.T) {
		options := ProjectCreateOptions{
			Name:        "foo",
			Description: String("qux"),
		}

		w, err := client.Projects.Create(ctx, orgTest.Name, options)
		require.NoError(t, err)

		refreshed, err := client.Projects.Read(ctx, w.ID)
		require.NoError(t, err)

		for _, item := range []*Project{
			w,
			refreshed,
		} {
			assert.NotEmpty(t, item.ID)
			assert.Equal(t, options.Name, item.Name)
			assert.Equal(t, *options.Description, item.Description)
		}
	})

	t.Run("when options is missing name", func(t *testing.T) {
		w, err := client.Projects.Create(ctx, orgTest.Name, ProjectCreateOptions{})
		assert.Nil(t, w)
		assert.EqualError(t, err, ErrRequiredName.Error())
	})

	t.Run("when options has an invalid name", func(t *testing.T) {
		w, err := client.Projects.Create(ctx, orgTest.Name, ProjectCreateOptions{
			Name: badIdentifier,
		})
		assert.Nil(t, w)
		assert.Contains(t, err.Error(), "invalid attribute\n\nName may only contain")
	})

	t.Run("when options has an invalid organization", func(t *testing.T) {
		w, err := client.Projects.Create(ctx, badIdentifier, ProjectCreateOptions{
			Name: "foo",
		})
		assert.Nil(t, w)
		assert.EqualError(t, err, ErrInvalidOrg.Error())
	})

	t.Run("when options has an invalid auto destroy activity duration", func(t *testing.T) {
		skipUnlessBeta(t)

		w, err := client.Projects.Create(ctx, orgTest.Name, ProjectCreateOptions{
			Name:                        "foo",
			AutoDestroyActivityDuration: jsonapi.NewNullableAttrWithValue("20m"),
		})
		assert.Nil(t, w)
		assert.Contains(t, err.Error(), "invalid attribute\n\nAuto destroy activity duration has an incorrect format, we expect up to 4 numeric digits and 1 unit ('d' or 'h')")
	})

	t.Run("when a default agent pool ID is specified without 'agent' execution mode", func(t *testing.T) {
		agentPoolTest, agentPoolTestCleanup := createAgentPool(t, client, orgTest)
		t.Cleanup(agentPoolTestCleanup)

		p, err := client.Projects.Create(ctx, orgTest.Name, ProjectCreateOptions{
			Name:                 fmt.Sprintf("foo-%s", randomString(t)),
			DefaultExecutionMode: String("remote"),
			DefaultAgentPoolID:   String(agentPoolTest.ID),
		})

		assert.Nil(t, p)
		assert.Contains(t, err.Error(), "unprocessable entity\n\nAgent pool must not be specified unless using 'agent' execution mode")
	})

	t.Run("when 'agent' execution mode is specified without an a default agent pool ID", func(t *testing.T) {
		p, err := client.Projects.Create(ctx, orgTest.Name, ProjectCreateOptions{
			Name:                 fmt.Sprintf("foo-%s", randomString(t)),
			DefaultExecutionMode: String("agent"),
		})

		assert.Nil(t, p)
		assert.Contains(t, err.Error(), "invalid attribute\n\nDefault agent pool must be specified when using 'agent' execution mode")
	})

	t.Run("when no execution mode is specified, in an organization with local as default execution mode", func(t *testing.T) {
		orgTest, orgTestCleanup := createOrganizationWithOptions(t, client, OrganizationCreateOptions{
			Name:                 String("tst-" + randomString(t)[0:20]),
			Email:                String(fmt.Sprintf("%s@tfe.local", randomString(t))),
			DefaultExecutionMode: String("local"),
		})
		t.Cleanup(orgTestCleanup)

		options := ProjectCreateOptions{
			Name: fmt.Sprintf("foo-%s", randomString(t)),
			SettingOverwrites: &ProjectSettingOverwrites{
				ExecutionMode: Bool(false),
				AgentPool:     Bool(false),
			},
		}

		p, err := client.Projects.Create(ctx, orgTest.Name, options)
		require.NoError(t, err)

		// Get a refreshed view from the API.
		refreshed, err := client.Projects.Read(ctx, p.ID)
		require.NoError(t, err)

		assert.Equal(t, "local", refreshed.DefaultExecutionMode)
	})

	t.Run("when agent pool and execution mode setting overwrites do not match", func(t *testing.T) {
		agentPoolTest, agentPoolTestCleanup := createAgentPool(t, client, orgTest)
		t.Cleanup(agentPoolTestCleanup)

		p, err := client.Projects.Create(ctx, orgTest.Name, ProjectCreateOptions{
			Name:                 fmt.Sprintf("foo-%s", randomString(t)),
			DefaultExecutionMode: String("agent"),
			DefaultAgentPoolID:   String(agentPoolTest.ID),
			SettingOverwrites: &ProjectSettingOverwrites{
				AgentPool:     Bool(false),
				ExecutionMode: Bool(true),
			},
		})

		assert.Nil(t, p)
		assert.Contains(t, err.Error(), "If agent-pool and execution-mode are both included in setting-overwrites, their values must be the same.")
	})

	t.Run("when organization has a default execution mode", func(t *testing.T) {
		defaultExecutionOrgTest, defaultExecutionOrgTestCleanup := createOrganizationWithDefaultAgentPool(t, client)
		t.Cleanup(defaultExecutionOrgTestCleanup)

		t.Run("with setting overwrites set to false, project inherits the default execution mode", func(t *testing.T) {
			options := ProjectCreateOptions{
				Name: fmt.Sprintf("tst-proj-%s", randomString(t)[0:20]),
				SettingOverwrites: &ProjectSettingOverwrites{
					ExecutionMode: Bool(false),
					AgentPool:     Bool(false),
				},
			}
			p, err := client.Projects.Create(ctx, defaultExecutionOrgTest.Name, options)

			require.NoError(t, err)
			assert.Equal(t, "agent", p.DefaultExecutionMode)
		})

		t.Run("with setting overwrites set to true, project ignores the default execution mode", func(t *testing.T) {
			options := ProjectCreateOptions{
				Name:                 fmt.Sprintf("tst-proj-%s", randomString(t)[0:20]),
				DefaultExecutionMode: String("local"),
				SettingOverwrites: &ProjectSettingOverwrites{
					ExecutionMode: Bool(true),
					AgentPool:     Bool(true),
				},
			}
			p, err := client.Projects.Create(ctx, defaultExecutionOrgTest.Name, options)

			require.NoError(t, err)
			assert.Equal(t, "local", p.DefaultExecutionMode)
		})

		t.Run("when explicitly setting default execution mode, project ignores the org default execution mode", func(t *testing.T) {
			options := ProjectCreateOptions{
				Name:                 fmt.Sprintf("tst-proj-%s", randomString(t)[0:20]),
				DefaultExecutionMode: String("remote"),
			}
			p, err := client.Projects.Create(ctx, defaultExecutionOrgTest.Name, options)

			require.NoError(t, err)
			assert.Equal(t, "remote", p.DefaultExecutionMode)
		})
	})
}

func TestProjectsUpdate(t *testing.T) {
	t.Parallel()
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	t.Cleanup(orgTestCleanup)

	agentPoolTest, agentPoolTestCleanup := createAgentPool(t, client, orgTest)
	t.Cleanup(agentPoolTestCleanup)

	t.Run("with valid options", func(t *testing.T) {
		kBefore, kTestCleanup := createProject(t, client, orgTest)
		t.Cleanup(kTestCleanup)

		kAfter, err := client.Projects.Update(ctx, kBefore.ID, ProjectUpdateOptions{
			Name:        String("new project name"),
			Description: String("updated description"),
			TagBindings: []*TagBinding{
				{Key: "foo", Value: "bar"},
			},
			DefaultExecutionMode: String("agent"),
			DefaultAgentPoolID:   String(agentPoolTest.ID),
		})
		require.NoError(t, err)

		assert.Equal(t, kBefore.ID, kAfter.ID)
		assert.NotEqual(t, kBefore.Name, kAfter.Name)
		assert.NotEqual(t, kBefore.Description, kAfter.Description)
		assert.NotEqual(t, kBefore.DefaultExecutionMode, kAfter.DefaultExecutionMode)
		assert.NotEqual(t, kBefore.DefaultAgentPool, kAfter.DefaultAgentPool)

		if betaFeaturesEnabled() {
			bindings, err := client.Projects.ListTagBindings(ctx, kAfter.ID)
			require.NoError(t, err)

			require.Len(t, bindings, 1)
			assert.Equal(t, "foo", bindings[0].Key)
			assert.Equal(t, "bar", bindings[0].Value)

			effectiveBindings, err := client.Projects.ListEffectiveTagBindings(ctx, kAfter.ID)
			require.NoError(t, err)

			require.Len(t, effectiveBindings, 1)
			assert.Equal(t, "foo", effectiveBindings[0].Key)
			assert.Equal(t, "bar", effectiveBindings[0].Value)

			ws, err := client.Workspaces.Create(ctx, orgTest.Name, WorkspaceCreateOptions{
				Name:    String("new-workspace-inherits-tags"),
				Project: kAfter,
				TagBindings: []*TagBinding{
					{Key: "baz", Value: "qux"},
				},
			})
			require.NoError(t, err)

			t.Cleanup(func() {
				err := client.Workspaces.DeleteByID(ctx, ws.ID)
				if err != nil {
					t.Errorf("Error destroying workspace! WARNING: Dangling resources\n"+
						"may exist! The full error is shown below.\n\n"+
						"Error: %s", err)
				}
			})

			wsEffectiveBindings, err := client.Workspaces.ListEffectiveTagBindings(ctx, ws.ID)
			require.NoError(t, err)

			assert.Len(t, wsEffectiveBindings, 2)
			for _, b := range wsEffectiveBindings {
				switch b.Key {
				case "foo":
					assert.Equal(t, "bar", b.Value)
				case "baz":
					assert.Equal(t, "qux", b.Value)
				default:
					assert.Fail(t, "unexpected tag binding %q", b.Key)
				}
			}
		}
	})

	t.Run("when updating with invalid name", func(t *testing.T) {
		kBefore, kTestCleanup := createProject(t, client, orgTest)
		t.Cleanup(kTestCleanup)

		kAfter, err := client.Projects.Update(ctx, kBefore.ID, ProjectUpdateOptions{
			Name: String(badIdentifier),
		})
		assert.Nil(t, kAfter)
		assert.Contains(t, err.Error(), "invalid attribute\n\nName may only contain")
	})

	t.Run("without a valid projects ID", func(t *testing.T) {
		w, err := client.Projects.Update(ctx, badIdentifier, ProjectUpdateOptions{})
		assert.Nil(t, w)
		assert.EqualError(t, err, ErrInvalidProjectID.Error())
	})

	t.Run("without a valid projects auto destroy activity duration", func(t *testing.T) {
		skipUnlessBeta(t)

		newSubscriptionUpdater(orgTest).WithBusinessPlan().Update(t)

		kBefore, kTestCleanup := createProject(t, client, orgTest)
		t.Cleanup(kTestCleanup)

		w, err := client.Projects.Update(ctx, kBefore.ID, ProjectUpdateOptions{
			AutoDestroyActivityDuration: jsonapi.NewNullableAttrWithValue("bar"),
		})
		assert.Nil(t, w)
		assert.Contains(t, err.Error(), "invalid attribute\n\nAuto destroy activity duration has an incorrect format, we expect up to 4 numeric digits and 1 unit ('d' or 'h')")
	})

	t.Run("with agent pool provided, but remote execution mode", func(t *testing.T) {
		kBefore, kTestCleanup := createProject(t, client, orgTest)
		t.Cleanup(kTestCleanup)

		pool, agentPoolCleanup := createAgentPool(t, client, orgTest)
		t.Cleanup(agentPoolCleanup)

		proj, err := client.Projects.Update(ctx, kBefore.ID, ProjectUpdateOptions{
			DefaultExecutionMode: String("local"),
			DefaultAgentPoolID:   String(pool.ID),
		})
		assert.Nil(t, proj)
		assert.ErrorContains(t, err, "Agent pool must not be specified unless using 'agent' execution mode")
	})

	t.Run("with different default execution modes", func(t *testing.T) {
		proj, projCleanup := createProject(t, client, orgTest)
		t.Cleanup(projCleanup)

		agentPool, agenPoolCleanup := createAgentPool(t, client, orgTest)
		t.Cleanup(agenPoolCleanup)

		assert.Equal(t, "remote", proj.DefaultExecutionMode)
		assert.Nil(t, proj.DefaultAgentPool)

		// assert that project's execution mode can be updated from 'remote' -> 'agent'
		proj, err := client.Projects.Update(ctx, proj.ID, ProjectUpdateOptions{
			DefaultExecutionMode: String("agent"),
			DefaultAgentPoolID:   String(agentPool.ID),
		})
		require.NoError(t, err)
		assert.Equal(t, "agent", proj.DefaultExecutionMode)
		assert.Equal(t, agentPool.ID, proj.DefaultAgentPool.ID)

		// assert that project's execution mode can be updated from 'agent' -> 'remote'
		proj, err = client.Projects.Update(ctx, proj.ID, ProjectUpdateOptions{
			DefaultExecutionMode: String("remote"),
		})
		require.NoError(t, err)
		assert.Equal(t, "remote", proj.DefaultExecutionMode)
		assert.Nil(t, proj.DefaultAgentPool)

		// assert that project's execution mode can be updated from 'remote' -> 'local'
		proj, err = client.Projects.Update(ctx, proj.ID, ProjectUpdateOptions{
			DefaultExecutionMode: String("local"),
		})
		require.NoError(t, err)
		assert.Equal(t, "local", proj.DefaultExecutionMode)
		assert.Nil(t, proj.DefaultAgentPool)
	})

	t.Run("with setting overwrites set to true, project ignores the default execution mode", func(t *testing.T) {
		defaultExecutionOrgTest, defaultExecutionOrgTestCleanup := createOrganizationWithDefaultAgentPool(t, client)
		t.Cleanup(defaultExecutionOrgTestCleanup)

		kBefore, kTestCleanup := createProject(t, client, defaultExecutionOrgTest)
		t.Cleanup(kTestCleanup)

		options := ProjectUpdateOptions{
			DefaultExecutionMode: String("local"),
			SettingOverwrites: &ProjectSettingOverwrites{
				ExecutionMode: Bool(true),
				AgentPool:     Bool(true),
			},
		}
		p, err := client.Projects.Update(ctx, kBefore.ID, options)

		require.NoError(t, err)
		assert.Equal(t, "local", p.DefaultExecutionMode)
	})

	t.Run("with setting overwrites set to false, project inherits the default execution mode", func(t *testing.T) {
		defaultExecutionOrgTest, defaultExecutionOrgTestCleanup := createOrganizationWithDefaultAgentPool(t, client)
		t.Cleanup(defaultExecutionOrgTestCleanup)

		kBefore, kTestCleanup := createProject(t, client, defaultExecutionOrgTest)
		t.Cleanup(kTestCleanup)

		options := ProjectUpdateOptions{
			SettingOverwrites: &ProjectSettingOverwrites{
				ExecutionMode: Bool(false),
				AgentPool:     Bool(false),
			},
		}
		p, err := client.Projects.Update(ctx, kBefore.ID, options)

		require.NoError(t, err)
		assert.Equal(t, "agent", p.DefaultExecutionMode)
	})
}

func TestProjectsAddTagBindings(t *testing.T) {
	t.Parallel()
	skipUnlessBeta(t)

	client := testClient(t)
	ctx := context.Background()

	pTest, wCleanup := createProject(t, client, nil)
	t.Cleanup(wCleanup)

	t.Run("when adding tag bindings to a project", func(t *testing.T) {
		tagBindings := []*TagBinding{
			{Key: "foo", Value: "bar"},
			{Key: "baz", Value: "qux"},
		}

		bindings, err := client.Projects.AddTagBindings(ctx, pTest.ID, ProjectAddTagBindingsOptions{
			TagBindings: tagBindings,
		})
		require.NoError(t, err)

		require.Len(t, bindings, 2)
		assert.Equal(t, tagBindings[0].Key, bindings[0].Key)
		assert.Equal(t, tagBindings[0].Value, bindings[0].Value)
		assert.Equal(t, tagBindings[1].Key, bindings[1].Key)
		assert.Equal(t, tagBindings[1].Value, bindings[1].Value)
	})

	t.Run("when adding 26 tags", func(t *testing.T) {
		tagBindings := []*TagBinding{
			{Key: "alpha"},
			{Key: "bravo"},
			{Key: "charlie"},
			{Key: "delta"},
			{Key: "echo"},
			{Key: "foxtrot"},
			{Key: "golf"},
			{Key: "hotel"},
			{Key: "india"},
			{Key: "juliet"},
			{Key: "kilo"},
			{Key: "lima"},
			{Key: "mike"},
			{Key: "november"},
			{Key: "oscar"},
			{Key: "papa"},
			{Key: "quebec"},
			{Key: "romeo"},
			{Key: "sierra"},
			{Key: "tango"},
			{Key: "uniform"},
			{Key: "victor"},
			{Key: "whiskey"},
			{Key: "xray"},
			{Key: "yankee"},
			{Key: "zulu"},
		}

		_, err := client.Workspaces.AddTagBindings(ctx, pTest.ID, WorkspaceAddTagBindingsOptions{
			TagBindings: tagBindings,
		})
		require.Error(t, err, "cannot exceed 10 bindings per resource")
	})
}

func TestProjects_DeleteAllTagBindings(t *testing.T) {
	t.Parallel()
	skipUnlessBeta(t)

	client := testClient(t)
	ctx := context.Background()

	pTest, wCleanup := createProject(t, client, nil)
	t.Cleanup(wCleanup)

	tagBindings := []*TagBinding{
		{Key: "foo", Value: "bar"},
		{Key: "baz", Value: "qux"},
	}

	_, err := client.Projects.AddTagBindings(ctx, pTest.ID, ProjectAddTagBindingsOptions{
		TagBindings: tagBindings,
	})
	require.NoError(t, err)

	err = client.Projects.DeleteAllTagBindings(ctx, pTest.ID)
	require.NoError(t, err)

	bindings, err := client.Projects.ListTagBindings(ctx, pTest.ID)
	require.NoError(t, err)
	require.Empty(t, bindings)
}

func TestProjectsDelete(t *testing.T) {
	t.Parallel()
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	t.Cleanup(orgTestCleanup)

	pTest, _ := createProject(t, client, orgTest)

	t.Run("with valid options", func(t *testing.T) {
		err := client.Projects.Delete(ctx, pTest.ID)
		require.NoError(t, err)

		// Try loading the project - it should fail.
		_, err = client.Projects.Read(ctx, pTest.ID)
		assert.Equal(t, err, ErrResourceNotFound)
	})

	t.Run("when the project does not exist", func(t *testing.T) {
		err := client.Projects.Delete(ctx, pTest.ID)
		assert.Equal(t, err, ErrResourceNotFound)
	})

	t.Run("when the project ID is invalid", func(t *testing.T) {
		err := client.Projects.Delete(ctx, badIdentifier)
		assert.EqualError(t, err, ErrInvalidProjectID.Error())
	})
}

func TestProjectsAutoDestroy(t *testing.T) {
	t.Parallel()
	skipUnlessBeta(t)
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	t.Cleanup(orgTestCleanup)

	newSubscriptionUpdater(orgTest).WithBusinessPlan().Update(t)

	t.Run("when creating workspace in project with autodestroy", func(t *testing.T) {
		options := ProjectCreateOptions{
			Name:                        "foo",
			Description:                 String("qux"),
			AutoDestroyActivityDuration: jsonapi.NewNullableAttrWithValue("3d"),
		}

		p, err := client.Projects.Create(ctx, orgTest.Name, options)
		require.NoError(t, err)

		w, _ := createWorkspaceWithOptions(t, client, orgTest, WorkspaceCreateOptions{
			Name:    String(randomString(t)),
			Project: p,
		})

		assert.Equal(t, p.AutoDestroyActivityDuration, w.AutoDestroyActivityDuration)
	})
}
