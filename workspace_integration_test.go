package tfe

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"
	"testing"
	"time"

	retryablehttp "github.com/hashicorp/go-retryablehttp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type WorkspaceTableOptions struct {
	createOptions *WorkspaceCreateOptions
	updateOptions *WorkspaceUpdateOptions
}

type WorkspaceTableTest struct {
	scenario  string
	options   *WorkspaceTableOptions
	setup     func(options *WorkspaceTableOptions) (w *Workspace, cleanup func())
	assertion func(w *Workspace, options *WorkspaceTableOptions, err error)
}

func TestWorkspacesList(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	t.Cleanup(orgTestCleanup)

	wTest1, wTest1Cleanup := createWorkspace(t, client, orgTest)
	t.Cleanup(wTest1Cleanup)
	wTest2, wTest2Cleanup := createWorkspace(t, client, orgTest)
	t.Cleanup(wTest2Cleanup)

	t.Run("without list options", func(t *testing.T) {
		wl, err := client.Workspaces.List(ctx, orgTest.Name, nil)
		require.NoError(t, err)
		assert.Contains(t, wl.Items, wTest1)
		assert.Contains(t, wl.Items, wTest2)
		assert.Equal(t, 1, wl.CurrentPage)
		assert.Equal(t, 2, wl.TotalCount)
	})

	t.Run("with list options", func(t *testing.T) {
		// Request a page number which is out of range. The result should
		// be successful, but return no results if the paging options are
		// properly passed along.
		wl, err := client.Workspaces.List(ctx, orgTest.Name, &WorkspaceListOptions{
			ListOptions: ListOptions{
				PageNumber: 999,
				PageSize:   100,
			},
		})
		require.NoError(t, err)
		assert.Empty(t, wl.Items)
		assert.Equal(t, 999, wl.CurrentPage)
		assert.Equal(t, 2, wl.TotalCount)
	})

	t.Run("when searching a known workspace", func(t *testing.T) {
		// Use a known workspace prefix as search attribute. The result
		// should be successful and only contain the matching workspace.
		wl, err := client.Workspaces.List(ctx, orgTest.Name, &WorkspaceListOptions{
			Search: wTest1.Name[:len(wTest1.Name)-5],
		})
		require.NoError(t, err)
		assert.Contains(t, wl.Items, wTest1)
		assert.NotContains(t, wl.Items, wTest2)
		assert.Equal(t, 1, wl.CurrentPage)
		assert.Equal(t, 1, wl.TotalCount)
	})

	t.Run("when searching using a tag", func(t *testing.T) {
		tagName := "tagtest"

		// Add the tag to the first workspace for searching.
		err := client.Workspaces.AddTags(ctx, wTest1.ID, WorkspaceAddTagsOptions{
			Tags: []*Tag{
				{
					Name: tagName,
				},
			},
		})
		require.NoError(t, err)

		// The result should be successful and only contain the workspace with the
		// new tag.
		wl, err := client.Workspaces.List(ctx, orgTest.Name, &WorkspaceListOptions{
			Tags: tagName,
		})
		require.NoError(t, err)
		assert.Equal(t, wl.Items[0].ID, wTest1.ID)
		assert.Equal(t, 1, wl.CurrentPage)
		assert.Equal(t, 1, wl.TotalCount)
	})

	t.Run("when searching using exclude-tags", func(t *testing.T) {
		for wsID, tag := range map[string]string{wTest1.ID: "foo", wTest2.ID: "bar"} {
			err := client.Workspaces.AddTags(ctx, wsID, WorkspaceAddTagsOptions{
				Tags: []*Tag{
					{
						Name: tag,
					},
				},
			})
			require.NoError(t, err)
		}

		wl, err := client.Workspaces.List(ctx, orgTest.Name, &WorkspaceListOptions{
			ExcludeTags: "foo",
		})

		require.NoError(t, err)
		assert.Contains(t, wl.Items[0].ID, wTest2.ID)
		assert.Equal(t, 1, wl.CurrentPage)
		assert.Equal(t, 1, wl.TotalCount)
	})

	t.Run("when searching an unknown workspace", func(t *testing.T) {
		// Use a nonexisting workspace name as search attribute. The result
		// should be successful, but return no results.
		wl, err := client.Workspaces.List(ctx, orgTest.Name, &WorkspaceListOptions{
			Search: "nonexisting",
		})
		require.NoError(t, err)
		assert.Empty(t, wl.Items)
		assert.Equal(t, 1, wl.CurrentPage)
		assert.Equal(t, 0, wl.TotalCount)
	})

	t.Run("without a valid organization", func(t *testing.T) {
		wl, err := client.Workspaces.List(ctx, badIdentifier, nil)
		assert.Nil(t, wl)
		assert.EqualError(t, err, ErrInvalidOrg.Error())
	})

	t.Run("with organization included", func(t *testing.T) {
		wl, err := client.Workspaces.List(ctx, orgTest.Name, &WorkspaceListOptions{
			Include: []WSIncludeOpt{WSOrganization},
		})

		require.NoError(t, err)
		require.NotEmpty(t, wl.Items)
		require.NotNil(t, wl.Items[0].Organization)
		assert.NotEmpty(t, wl.Items[0].Organization.Email)
	})

	t.Run("with current-state-version,current-run included", func(t *testing.T) {
		_, rCleanup := createRunApply(t, client, wTest1)
		t.Cleanup(rCleanup)

		wl, err := client.Workspaces.List(ctx, orgTest.Name, &WorkspaceListOptions{
			Include: []WSIncludeOpt{WSCurrentStateVer, WSCurrentRun},
		})

		require.NoError(t, err)
		require.NotEmpty(t, wl.Items)

		foundWTest1 := false
		for _, ws := range wl.Items {
			if ws.ID == wTest1.ID {
				foundWTest1 = true
				require.NotNil(t, wl.Items[0].CurrentStateVersion)
				assert.NotEmpty(t, wl.Items[0].CurrentStateVersion.DownloadURL)

				require.NotNil(t, wl.Items[0].CurrentRun)
				assert.NotEmpty(t, wl.Items[0].CurrentRun.Message)
			}
		}

		assert.True(t, foundWTest1)
	})

	t.Run("when searching a known substring", func(t *testing.T) {
		wildcardSearch := "*-prod"
		// should be successful, and return 1 result
		wTest, wTestCleanup := createWorkspaceWithOptions(t, client, orgTest, WorkspaceCreateOptions{
			Name: String("hashicorp-prod"),
		})
		t.Cleanup(wTestCleanup)

		wl, err := client.Workspaces.List(ctx, orgTest.Name, &WorkspaceListOptions{
			WildcardName: wildcardSearch,
		})

		require.NoError(t, err)
		assert.NotEmpty(t, wTest.ID)
		assert.Equal(t, 1, wl.TotalCount)
	})

	t.Run("when wildcard match does not exist", func(t *testing.T) {
		wildcardSearch := "*-dev"
		// should be successful, but return no results
		wTest, wTestCleanup := createWorkspaceWithOptions(t, client, orgTest, WorkspaceCreateOptions{
			Name: String("hashicorp-staging"),
		})
		t.Cleanup(wTestCleanup)

		wl, err := client.Workspaces.List(ctx, orgTest.Name, &WorkspaceListOptions{
			WildcardName: wildcardSearch,
		})

		require.NoError(t, err)
		assert.NotEmpty(t, wTest.ID)
		assert.Equal(t, 0, wl.TotalCount)
	})
}

func TestWorkspacesCreateTableDriven(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	t.Cleanup(orgTestCleanup)

	workspaceTableTests := []WorkspaceTableTest{
		{
			scenario: "when options include tags-regex",
			options: &WorkspaceTableOptions{
				createOptions: &WorkspaceCreateOptions{
					Name:                String("foobar"),
					FileTriggersEnabled: Bool(false),
					VCSRepo: &VCSRepoOptions{
						TagsRegex: String("barfoo")},
				},
			},
			setup: func(options *WorkspaceTableOptions) (w *Workspace, cleanup func()) {
				// Remove the below organization creation and use the one from the outer scope once the feature flag is removed
				orgTest, orgTestCleanup := createOrganizationWithOptions(t, client, OrganizationCreateOptions{
					Name:  String("tst-" + randomString(t)[0:20]),
					Email: String(fmt.Sprintf("%s@tfe.local", randomString(t))),
				})

				w, wTestCleanup := createWorkspaceWithVCS(t, client, orgTest, *options.createOptions)

				return w, func() {
					t.Cleanup(orgTestCleanup)
					t.Cleanup(wTestCleanup)
				}
			},
			assertion: func(w *Workspace, options *WorkspaceTableOptions, err error) {
				assert.Equal(t, *options.createOptions.VCSRepo.TagsRegex, w.VCSRepo.TagsRegex)

				// Get a refreshed view from the API.
				refreshed, readErr := client.Workspaces.Read(ctx, w.Organization.Name, *options.createOptions.Name)
				require.NoError(t, readErr)

				for _, item := range []*Workspace{
					w,
					refreshed,
				} {
					assert.Equal(t, *options.createOptions.VCSRepo.TagsRegex, item.VCSRepo.TagsRegex)
				}
			},
		},
		{
			scenario: "when options include both non-empty tags-regex and trigger-patterns error is returned",
			options: &WorkspaceTableOptions{
				createOptions: &WorkspaceCreateOptions{
					Name:                String("foobar"),
					FileTriggersEnabled: Bool(false),
					VCSRepo:             &VCSRepoOptions{TagsRegex: String("foobar")},
					TriggerPatterns:     []string{"/module-1/**/*", "/**/networking/*"},
				},
			},
			assertion: func(w *Workspace, options *WorkspaceTableOptions, err error) {
				assert.Nil(t, w)
				assert.EqualError(t, err, ErrUnsupportedBothTagsRegexAndTriggerPatterns.Error())
			},
		},
		{
			scenario: "when options include both non-empty tags-regex and trigger-prefixes error is returned",
			options: &WorkspaceTableOptions{
				createOptions: &WorkspaceCreateOptions{
					Name:                String("foobar"),
					FileTriggersEnabled: Bool(false),
					VCSRepo:             &VCSRepoOptions{TagsRegex: String("foobar")},
					TriggerPrefixes:     []string{"/module-1", "/module-2"},
				},
			},
			assertion: func(w *Workspace, options *WorkspaceTableOptions, err error) {
				assert.Nil(t, w)
				assert.EqualError(t, err, ErrUnsupportedBothTagsRegexAndTriggerPrefixes.Error())
			},
		},
		{
			scenario: "when options include both non-empty tags-regex and file-triggers-enabled as true an error is returned",
			options: &WorkspaceTableOptions{
				createOptions: &WorkspaceCreateOptions{
					Name:                String("foobar"),
					FileTriggersEnabled: Bool(true),
					VCSRepo:             &VCSRepoOptions{TagsRegex: String("foobar")},
				},
			},
			assertion: func(w *Workspace, options *WorkspaceTableOptions, err error) {
				assert.Nil(t, w)
				assert.EqualError(t, err, ErrUnsupportedBothTagsRegexAndFileTriggersEnabled.Error())
			},
		},
		{
			scenario: "when options include both non-empty tags-regex and file-triggers-enabled as false an error is not returned",
			options: &WorkspaceTableOptions{
				createOptions: &WorkspaceCreateOptions{
					Name:                String("foobar"),
					FileTriggersEnabled: Bool(false),
					VCSRepo:             &VCSRepoOptions{TagsRegex: String("foobar")},
				},
			},
			setup: func(options *WorkspaceTableOptions) (w *Workspace, cleanup func()) {
				w, wTestCleanup := createWorkspaceWithVCS(t, client, orgTest, *options.createOptions)

				return w, func() {
					t.Cleanup(wTestCleanup)
				}
			},
			assertion: func(w *Workspace, options *WorkspaceTableOptions, err error) {
				require.NotNil(t, w)
				require.NoError(t, err)
			},
		},
	}

	for _, tableTest := range workspaceTableTests {
		t.Run(tableTest.scenario, func(t *testing.T) {
			var workspace *Workspace
			var cleanup func()
			var err error
			if tableTest.setup != nil {
				workspace, cleanup = tableTest.setup(tableTest.options)
				defer cleanup()
			} else {
				workspace, err = client.Workspaces.Create(ctx, orgTest.Name, *tableTest.options.createOptions)
			}
			tableTest.assertion(workspace, tableTest.options, err)
		})
	}
}

func TestWorkspacesCreate(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	t.Cleanup(orgTestCleanup)

	t.Run("with valid options", func(t *testing.T) {
		options := WorkspaceCreateOptions{
			Name:                       String("foo"),
			AllowDestroyPlan:           Bool(false),
			AutoApply:                  Bool(true),
			Description:                String("qux"),
			AssessmentsEnabled:         Bool(false),
			FileTriggersEnabled:        Bool(true),
			Operations:                 Bool(true),
			QueueAllRuns:               Bool(true),
			SpeculativeEnabled:         Bool(true),
			SourceName:                 String("my-app"),
			SourceURL:                  String("http://my-app-hostname.io"),
			StructuredRunOutputEnabled: Bool(true),
			TerraformVersion:           String("0.11.0"),
			TriggerPrefixes:            []string{"/modules", "/shared"},
			WorkingDirectory:           String("bar/"),
			Tags: []*Tag{
				{
					Name: "tag1",
				},
				{
					Name: "tag2",
				},
			},
		}

		w, err := client.Workspaces.Create(ctx, orgTest.Name, options)
		require.NoError(t, err)

		// Get a refreshed view from the API.
		refreshed, err := client.Workspaces.Read(ctx, orgTest.Name, *options.Name)
		require.NoError(t, err)

		for _, item := range []*Workspace{
			w,
			refreshed,
		} {
			assert.NotEmpty(t, item.ID)
			assert.Equal(t, *options.Name, item.Name)
			assert.Equal(t, *options.Description, item.Description)
			assert.Equal(t, *options.AllowDestroyPlan, item.AllowDestroyPlan)
			assert.Equal(t, *options.AutoApply, item.AutoApply)
			assert.Equal(t, *options.AssessmentsEnabled, item.AssessmentsEnabled)
			assert.Equal(t, *options.FileTriggersEnabled, item.FileTriggersEnabled)
			assert.Equal(t, *options.Operations, item.Operations)
			assert.Equal(t, *options.QueueAllRuns, item.QueueAllRuns)
			assert.Equal(t, *options.SpeculativeEnabled, item.SpeculativeEnabled)
			assert.Equal(t, *options.SourceName, item.SourceName)
			assert.Equal(t, *options.SourceURL, item.SourceURL)
			assert.Equal(t, *options.StructuredRunOutputEnabled, item.StructuredRunOutputEnabled)
			assert.Equal(t, options.Tags[0].Name, item.TagNames[0])
			assert.Equal(t, options.Tags[1].Name, item.TagNames[1])
			assert.Equal(t, *options.TerraformVersion, item.TerraformVersion)
			assert.Equal(t, options.TriggerPrefixes, item.TriggerPrefixes)
			assert.Equal(t, *options.WorkingDirectory, item.WorkingDirectory)
		}
	})

	t.Run("when options is missing name", func(t *testing.T) {
		w, err := client.Workspaces.Create(ctx, "foo", WorkspaceCreateOptions{})
		assert.Nil(t, w)
		assert.EqualError(t, err, ErrRequiredName.Error())
	})

	t.Run("when options has an invalid name", func(t *testing.T) {
		w, err := client.Workspaces.Create(ctx, "foo", WorkspaceCreateOptions{
			Name: String(badIdentifier),
		})
		assert.Nil(t, w)
		assert.EqualError(t, err, ErrInvalidName.Error())
	})

	t.Run("when options has an invalid organization", func(t *testing.T) {
		w, err := client.Workspaces.Create(ctx, badIdentifier, WorkspaceCreateOptions{
			Name: String("foo"),
		})
		assert.Nil(t, w)
		assert.EqualError(t, err, ErrInvalidOrg.Error())
	})

	t.Run("when options includes both an operations value and an enforcement mode value", func(t *testing.T) {
		options := WorkspaceCreateOptions{
			Name:          String(fmt.Sprintf("foo-%s", randomString(t))),
			ExecutionMode: String("remote"),
			Operations:    Bool(true),
		}

		w, err := client.Workspaces.Create(ctx, orgTest.Name, options)
		assert.Nil(t, w)
		assert.Equal(t, err, ErrUnsupportedOperations)
	})

	t.Run("when an agent pool ID is specified without 'agent' execution mode", func(t *testing.T) {
		options := WorkspaceCreateOptions{
			Name:        String(fmt.Sprintf("foo-%s", randomString(t))),
			AgentPoolID: String("apool-xxxxx"),
		}

		w, err := client.Workspaces.Create(ctx, orgTest.Name, options)
		assert.Nil(t, w)
		assert.Equal(t, err, ErrRequiredAgentMode)
	})

	t.Run("when 'agent' execution mode is specified without an an agent pool ID", func(t *testing.T) {
		options := WorkspaceCreateOptions{
			Name:          String(fmt.Sprintf("foo-%s", randomString(t))),
			ExecutionMode: String("agent"),
		}

		w, err := client.Workspaces.Create(ctx, orgTest.Name, options)
		assert.Nil(t, w)
		assert.Equal(t, err, ErrRequiredAgentPoolID)
	})

	t.Run("when an error is returned from the API", func(t *testing.T) {
		w, err := client.Workspaces.Create(ctx, "bar", WorkspaceCreateOptions{
			Name:             String(fmt.Sprintf("bar-%s", randomString(t))),
			TerraformVersion: String("nonexisting"),
		})
		assert.Nil(t, w)
		assert.Error(t, err)
	})

	t.Run("when options include trigger-patterns (behind a feature flag)", func(t *testing.T) {
		// Remove the below organization creation and use the one from the outer scope once the feature flag is removed
		orgTest, orgTestCleanup := createOrganizationWithOptions(t, client, OrganizationCreateOptions{
			Name:  String("tst-" + randomString(t)[0:20] + "-ff-on"),
			Email: String(fmt.Sprintf("%s@tfe.local", randomString(t))),
		})
		t.Cleanup(orgTestCleanup)

		options := WorkspaceCreateOptions{
			Name:                String("foobar"),
			FileTriggersEnabled: Bool(true),
			TriggerPatterns:     []string{"/module-1/**/*", "/**/networking/*"},
		}
		w, err := client.Workspaces.Create(ctx, orgTest.Name, options)

		require.NoError(t, err)
		assert.Equal(t, options.TriggerPatterns, w.TriggerPatterns)

		// Get a refreshed view from the API.
		refreshed, err := client.Workspaces.Read(ctx, orgTest.Name, *options.Name)
		require.NoError(t, err)

		for _, item := range []*Workspace{
			w,
			refreshed,
		} {
			assert.Equal(t, options.TriggerPatterns, item.TriggerPatterns)
		}
	})

	t.Run("when options include both non-empty trigger-patterns and trigger-paths error is returned", func(t *testing.T) {
		options := WorkspaceCreateOptions{
			Name:                String(fmt.Sprintf("foobar-%s", randomString(t))),
			FileTriggersEnabled: Bool(true),
			TriggerPrefixes:     []string{"/module-1", "/module-2"},
			TriggerPatterns:     []string{"/module-1/**/*", "/**/networking/*"},
		}
		w, err := client.Workspaces.Create(ctx, orgTest.Name, options)

		assert.Nil(t, w)
		assert.EqualError(t, err, ErrUnsupportedBothTriggerPatternsAndPrefixes.Error())
	})

	t.Run("when options include trigger-patterns populated and empty trigger-paths workspace is created", func(t *testing.T) {
		// Remove the below organization creation and use the one from the outer scope once the feature flag is removed
		orgTest, orgTestCleanup := createOrganizationWithOptions(t, client, OrganizationCreateOptions{
			Name:  String("tst-" + randomString(t)[0:20] + "-ff-on"),
			Email: String(fmt.Sprintf("%s@tfe.local", randomString(t))),
		})
		t.Cleanup(orgTestCleanup)

		options := WorkspaceCreateOptions{
			Name:                String(fmt.Sprintf("foobar-%s", randomString(t))),
			FileTriggersEnabled: Bool(true),
			TriggerPrefixes:     []string{},
			TriggerPatterns:     []string{"/module-1/**/*", "/**/networking/*"},
		}
		w, err := client.Workspaces.Create(ctx, orgTest.Name, options)

		require.NoError(t, err)
		assert.Equal(t, options.TriggerPatterns, w.TriggerPatterns)
	})
}

func TestWorkspacesRead(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	t.Cleanup(orgTestCleanup)

	wTest, wTestCleanup := createWorkspace(t, client, orgTest)
	t.Cleanup(wTestCleanup)

	t.Run("when the workspace exists", func(t *testing.T) {
		w, err := client.Workspaces.Read(ctx, orgTest.Name, wTest.Name)
		require.NoError(t, err)
		assert.Equal(t, wTest, w)

		assert.True(t, w.Permissions.CanDestroy)
		assert.NotEmpty(t, w.Actions)
		assert.Equal(t, orgTest.Name, w.Organization.Name)
		assert.NotEmpty(t, w.CreatedAt)
	})

	t.Run("when the workspace does not exist", func(t *testing.T) {
		w, err := client.Workspaces.Read(ctx, orgTest.Name, "nonexisting")
		assert.Nil(t, w)
		assert.Error(t, err)
	})

	t.Run("when the organization does not exist", func(t *testing.T) {
		w, err := client.Workspaces.Read(ctx, "nonexisting", "nonexisting")
		assert.Nil(t, w)
		assert.Error(t, err)
	})

	t.Run("without a valid organization", func(t *testing.T) {
		w, err := client.Workspaces.Read(ctx, badIdentifier, wTest.Name)
		assert.Nil(t, w)
		assert.EqualError(t, err, ErrInvalidOrg.Error())
	})

	t.Run("without a valid workspace", func(t *testing.T) {
		w, err := client.Workspaces.Read(ctx, orgTest.Name, badIdentifier)
		assert.Nil(t, w)
		assert.EqualError(t, err, ErrInvalidWorkspaceValue.Error())
	})
}

func TestWorkspacesReadWithOptions(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	t.Cleanup(orgTestCleanup)

	wTest, wTestCleanup := createWorkspace(t, client, orgTest)
	t.Cleanup(wTestCleanup)

	svTest, svTestCleanup := createStateVersion(t, client, 0, wTest)
	t.Cleanup(svTestCleanup)

	// give TFC some time to process the statefile and extract the outputs.
	waitForSVOutputs(t, client, svTest.ID)

	t.Run("when options to include resource", func(t *testing.T) {
		opts := &WorkspaceReadOptions{
			Include: []WSIncludeOpt{WSOutputs},
		}
		w, err := client.Workspaces.ReadWithOptions(ctx, orgTest.Name, wTest.Name, opts)
		require.NoError(t, err)

		assert.Equal(t, wTest.ID, w.ID)
		assert.NotEmpty(t, w.Outputs)

		svOutputs, err := client.StateVersions.ListOutputs(ctx, svTest.ID, nil)
		require.NoError(t, err)

		assert.Len(t, w.Outputs, len(svOutputs.Items))

		wsOutputsSensitive := map[string]bool{}
		wsOutputsTypes := map[string]string{}
		for _, op := range w.Outputs {
			wsOutputsSensitive[op.Name] = op.Sensitive
			wsOutputsTypes[op.Name] = op.Type
		}
		for _, svop := range svOutputs.Items {
			valSensitive, ok := wsOutputsSensitive[svop.Name]
			assert.True(t, ok)
			assert.Equal(t, svop.Sensitive, valSensitive)

			valType, ok := wsOutputsTypes[svop.Name]
			assert.True(t, ok)
			assert.Equal(t, svop.Type, valType)
		}
	})
}

func TestWorkspacesReadWithHistory(t *testing.T) {
	client := testClient(t)

	orgTest, orgTestCleanup := createOrganization(t, client)
	t.Cleanup(orgTestCleanup)

	wTest, wTestCleanup := createWorkspace(t, client, orgTest)
	t.Cleanup(wTestCleanup)

	_, rCleanup := createRunApply(t, client, wTest)
	t.Cleanup(rCleanup)

	_, err := retry(func() (interface{}, error) {
		w, err := client.Workspaces.Read(context.Background(), orgTest.Name, wTest.Name)
		require.NoError(t, err)

		if w.RunsCount != 1 {
			return nil, fmt.Errorf("expected %d runs but found %d", 1, w.RunsCount)
		}

		if w.ResourceCount != 1 {
			return nil, fmt.Errorf("expected %d resources but found %d", 1, w.ResourceCount)
		}

		return w, nil
	})

	if err != nil {
		t.Error(err)
	}
}

func TestWorkspacesReadReadme(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	t.Cleanup(orgTestCleanup)

	wTest, wTestCleanup := createWorkspaceWithVCS(t, client, orgTest, WorkspaceCreateOptions{})
	t.Cleanup(wTestCleanup)

	_, rCleanup := createRunApply(t, client, wTest)
	t.Cleanup(rCleanup)

	t.Run("when the readme exists", func(t *testing.T) {
		w, err := client.Workspaces.Readme(ctx, wTest.ID)
		require.NoError(t, err)
		require.NotNil(t, w)

		readme, err := io.ReadAll(w)
		require.NoError(t, err)
		require.True(
			t,
			strings.HasPrefix(string(readme), `This is a simple test`),
			"got: %s", readme,
		)
	})

	t.Run("when the readme does not exist", func(t *testing.T) {
		w, err := client.Workspaces.Readme(ctx, "nonexisting")
		assert.Nil(t, w)
		assert.Error(t, err)
	})

	t.Run("without a valid workspace ID", func(t *testing.T) {
		w, err := client.Workspaces.Readme(ctx, badIdentifier)
		assert.Nil(t, w)
		assert.EqualError(t, err, ErrInvalidWorkspaceID.Error())
	})
}

func TestWorkspacesReadByID(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	t.Cleanup(orgTestCleanup)

	wTest, wTestCleanup := createWorkspace(t, client, orgTest)
	t.Cleanup(wTestCleanup)

	t.Run("when the workspace exists", func(t *testing.T) {
		w, err := client.Workspaces.ReadByID(ctx, wTest.ID)
		require.NoError(t, err)
		assert.Equal(t, wTest, w)

		assert.True(t, w.Permissions.CanDestroy)
		assert.Equal(t, orgTest.Name, w.Organization.Name)
		assert.NotEmpty(t, w.CreatedAt)
		assert.NotEmpty(t, w.Actions)
	})

	t.Run("when the workspace does not exist", func(t *testing.T) {
		w, err := client.Workspaces.ReadByID(ctx, "nonexisting")
		assert.Nil(t, w)
		assert.Error(t, err)
	})

	t.Run("without a valid workspace ID", func(t *testing.T) {
		w, err := client.Workspaces.ReadByID(ctx, badIdentifier)
		assert.Nil(t, w)
		assert.EqualError(t, err, ErrInvalidWorkspaceID.Error())
	})
}

func TestWorkspacesUpdate(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	t.Cleanup(orgTestCleanup)

	upgradeOrganizationSubscription(t, client, orgTest)

	wTest, wCleanup := createWorkspace(t, client, orgTest)
	t.Cleanup(wCleanup)

	t.Run("when updating a subset of values", func(t *testing.T) {
		options := WorkspaceUpdateOptions{
			Name:               String(wTest.Name),
			AllowDestroyPlan:   Bool(false),
			AutoApply:          Bool(true),
			Operations:         Bool(true),
			QueueAllRuns:       Bool(true),
			AssessmentsEnabled: Bool(true),
			TerraformVersion:   String("0.15.4"),
		}

		wAfter, err := client.Workspaces.Update(ctx, orgTest.Name, wTest.Name, options)
		require.NoError(t, err)

		assert.Equal(t, wTest.Name, wAfter.Name)
		assert.NotEqual(t, wTest.AllowDestroyPlan, wAfter.AllowDestroyPlan)
		assert.NotEqual(t, wTest.AutoApply, wAfter.AutoApply)
		assert.NotEqual(t, wTest.QueueAllRuns, wAfter.QueueAllRuns)
		assert.NotEqual(t, wTest.AssessmentsEnabled, wAfter.AssessmentsEnabled)
		assert.NotEqual(t, wTest.TerraformVersion, wAfter.TerraformVersion)
		assert.Equal(t, wTest.WorkingDirectory, wAfter.WorkingDirectory)
	})

	t.Run("with valid options", func(t *testing.T) {
		options := WorkspaceUpdateOptions{
			Name:                       String(randomString(t)),
			AllowDestroyPlan:           Bool(true),
			AutoApply:                  Bool(false),
			FileTriggersEnabled:        Bool(true),
			Operations:                 Bool(false),
			QueueAllRuns:               Bool(false),
			SpeculativeEnabled:         Bool(true),
			Description:                String("updated description"),
			StructuredRunOutputEnabled: Bool(true),
			TerraformVersion:           String("0.11.1"),
			TriggerPrefixes:            []string{"/modules", "/shared"},
			WorkingDirectory:           String("baz/"),
		}

		w, err := client.Workspaces.Update(ctx, orgTest.Name, wTest.Name, options)
		require.NoError(t, err)

		// Get a refreshed view of the workspace from the API
		refreshed, err := client.Workspaces.Read(ctx, orgTest.Name, *options.Name)
		require.NoError(t, err)

		for _, item := range []*Workspace{
			w,
			refreshed,
		} {
			assert.Equal(t, *options.Name, item.Name)
			assert.Equal(t, *options.AllowDestroyPlan, item.AllowDestroyPlan)
			assert.Equal(t, *options.AutoApply, item.AutoApply)
			assert.Equal(t, *options.FileTriggersEnabled, item.FileTriggersEnabled)
			assert.Equal(t, *options.Description, item.Description)
			assert.Equal(t, *options.Operations, item.Operations)
			assert.Equal(t, *options.QueueAllRuns, item.QueueAllRuns)
			assert.Equal(t, *options.SpeculativeEnabled, item.SpeculativeEnabled)
			assert.Equal(t, *options.StructuredRunOutputEnabled, item.StructuredRunOutputEnabled)
			assert.Equal(t, *options.TerraformVersion, item.TerraformVersion)
			assert.Equal(t, options.TriggerPrefixes, item.TriggerPrefixes)
			assert.Equal(t, *options.WorkingDirectory, item.WorkingDirectory)
		}
	})

	t.Run("when options includes both an operations value and an enforcement mode value", func(t *testing.T) {
		options := WorkspaceUpdateOptions{
			ExecutionMode: String("remote"),
			Operations:    Bool(true),
		}

		wAfter, err := client.Workspaces.Update(ctx, orgTest.Name, wTest.Name, options)
		assert.Nil(t, wAfter)
		assert.Equal(t, err, ErrUnsupportedOperations)
	})

	t.Run("when 'agent' execution mode is specified without an agent pool ID", func(t *testing.T) {
		options := WorkspaceUpdateOptions{
			ExecutionMode: String("agent"),
		}

		wAfter, err := client.Workspaces.Update(ctx, orgTest.Name, wTest.Name, options)
		assert.Nil(t, wAfter)
		assert.Equal(t, err, ErrRequiredAgentPoolID)
	})

	t.Run("when an error is returned from the api", func(t *testing.T) {
		w, err := client.Workspaces.Update(ctx, orgTest.Name, wTest.Name, WorkspaceUpdateOptions{
			TerraformVersion: String("nonexisting"),
		})
		assert.Nil(t, w)
		assert.Error(t, err)
	})

	t.Run("when options has an invalid name", func(t *testing.T) {
		w, err := client.Workspaces.Update(ctx, orgTest.Name, badIdentifier, WorkspaceUpdateOptions{})
		assert.Nil(t, w)
		assert.EqualError(t, err, ErrInvalidWorkspaceValue.Error())
	})

	t.Run("when options has an invalid organization", func(t *testing.T) {
		w, err := client.Workspaces.Update(ctx, badIdentifier, wTest.Name, WorkspaceUpdateOptions{})
		assert.Nil(t, w)
		assert.EqualError(t, err, ErrInvalidOrg.Error())
	})

	t.Run("when options include trigger-patterns (behind a feature flag)", func(t *testing.T) {
		// Remove the below organization and workspace creation and use the one from the outer scope once the feature flag is removed
		orgTest, orgTestCleanup := createOrganizationWithOptions(t, client, OrganizationCreateOptions{
			Name:  String("tst-" + randomString(t)[0:20] + "-ff-on"),
			Email: String(fmt.Sprintf("%s@tfe.local", randomString(t))),
		})
		t.Cleanup(orgTestCleanup)

		wTest, wCleanup := createWorkspaceWithOptions(t, client, orgTest, WorkspaceCreateOptions{
			Name:            String(randomString(t)),
			TriggerPrefixes: []string{"/prefix-1/", "/prefix-2/"},
		})
		t.Cleanup(wCleanup)
		assert.Equal(t, wTest.TriggerPrefixes, []string{"/prefix-1/", "/prefix-2/"}) // Sanity test

		options := WorkspaceUpdateOptions{
			Name:                String("foobar"),
			FileTriggersEnabled: Bool(true),
			TriggerPatterns:     []string{"/module-1/**/*", "/**/networking/*"},
		}
		w, err := client.Workspaces.Update(ctx, orgTest.Name, wTest.Name, options)
		require.NoError(t, err)

		// Get a refreshed view from the API.
		refreshed, err := client.Workspaces.Read(ctx, orgTest.Name, *options.Name)
		require.NoError(t, err)

		for _, item := range []*Workspace{
			w,
			refreshed,
		} {
			assert.Empty(t, options.TriggerPrefixes)
			assert.Equal(t, options.TriggerPatterns, item.TriggerPatterns)
		}
	})

	t.Run("when options include both trigger-patterns and trigger-paths error is returned", func(t *testing.T) {
		options := WorkspaceUpdateOptions{
			Name:                String("foobar"),
			FileTriggersEnabled: Bool(true),
			TriggerPrefixes:     []string{"/module-1", "/module-2"},
			TriggerPatterns:     []string{"/module-1/**/*", "/**/networking/*"},
		}
		w, err := client.Workspaces.Update(ctx, orgTest.Name, wTest.Name, options)

		assert.Nil(t, w)
		assert.EqualError(t, err, ErrUnsupportedBothTriggerPatternsAndPrefixes.Error())
	})

	t.Run("when options include trigger-patterns populated and empty trigger-paths workspace is updated", func(t *testing.T) {
		// Remove the below organization creation and use the one from the outer scope once the feature flag is removed
		orgTest, orgTestCleanup := createOrganizationWithOptions(t, client, OrganizationCreateOptions{
			Name:  String("tst-" + randomString(t)[0:20] + "-ff-on"),
			Email: String(fmt.Sprintf("%s@tfe.local", randomString(t))),
		})
		t.Cleanup(orgTestCleanup)

		wTest, wCleanup := createWorkspaceWithOptions(t, client, orgTest, WorkspaceCreateOptions{
			Name:            String(randomString(t)),
			TriggerPatterns: []string{"/pattern-1/**/*", "/pattern-2/**/*"},
		})
		t.Cleanup(wCleanup)
		assert.Equal(t, wTest.TriggerPatterns, []string{"/pattern-1/**/*", "/pattern-2/**/*"}) // Sanity test

		options := WorkspaceUpdateOptions{
			Name:                String("foobar"),
			FileTriggersEnabled: Bool(true),
			TriggerPrefixes:     []string{},
			TriggerPatterns:     []string{"/module-1/**/*", "/**/networking/*"},
		}
		w, err := client.Workspaces.Update(ctx, orgTest.Name, wTest.Name, options)
		require.NoError(t, err)

		// Get a refreshed view from the API.
		refreshed, err := client.Workspaces.Read(ctx, orgTest.Name, *options.Name)
		require.NoError(t, err)

		for _, item := range []*Workspace{
			w,
			refreshed,
		} {
			assert.Empty(t, options.TriggerPrefixes)
			assert.Equal(t, options.TriggerPatterns, item.TriggerPatterns)
		}
	})
}

func TestWorkspacesUpdateTableDriven(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	t.Cleanup(orgTestCleanup)

	wTest, wCleanup := createWorkspace(t, client, orgTest)
	t.Cleanup(wCleanup)

	workspaceTableTests := []WorkspaceTableTest{
		{
			scenario: "when options include VCSRepo tags-regex",
			options: &WorkspaceTableOptions{
				createOptions: &WorkspaceCreateOptions{
					Name:                String("foobar"),
					FileTriggersEnabled: Bool(false),
					VCSRepo: &VCSRepoOptions{
						TagsRegex: String("barfoo")},
				},
				updateOptions: &WorkspaceUpdateOptions{
					Name:                String("foobar"),
					FileTriggersEnabled: Bool(false),
					VCSRepo:             &VCSRepoOptions{TagsRegex: String("foobar")},
				},
			},
			setup: func(options *WorkspaceTableOptions) (w *Workspace, cleanup func()) {
				orgTest, orgTestCleanup := createOrganizationWithOptions(t, client, OrganizationCreateOptions{
					Name:  String("tst-" + randomString(t)[0:20]),
					Email: String(fmt.Sprintf("%s@tfe.local", randomString(t))),
				})

				wTest, wTestCleanup := createWorkspaceWithVCS(t, client, orgTest, *options.createOptions)
				return wTest, func() {
					t.Cleanup(orgTestCleanup)
					t.Cleanup(wTestCleanup)
				}
			},
			assertion: func(workspace *Workspace, options *WorkspaceTableOptions, _ error) {
				assert.Equal(t, *options.createOptions.VCSRepo.TagsRegex, workspace.VCSRepo.TagsRegex)
				assert.Equal(t, workspace.VCSRepo.TagsRegex, *String("barfoo")) // Sanity test

				w, err := client.Workspaces.Update(ctx, workspace.Organization.Name, workspace.Name, *options.updateOptions)
				require.NoError(t, err)

				assert.Equal(t, w.VCSRepo.TagsRegex, *String("foobar")) // Sanity test

				// Get a refreshed view from the API.
				refreshed, err := client.Workspaces.Read(ctx, workspace.Organization.Name, *options.updateOptions.Name)
				require.NoError(t, err)

				for _, item := range []*Workspace{
					w,
					refreshed,
				} {
					assert.Empty(t, options.updateOptions.TriggerPrefixes)
					assert.Empty(t, options.updateOptions.TriggerPatterns, item.TriggerPatterns)
				}
			},
		},
		{
			scenario: "when options include tags-regex and file-triggers-enabled is true an error is returned",
			options: &WorkspaceTableOptions{
				updateOptions: &WorkspaceUpdateOptions{
					Name:                String("foobar"),
					FileTriggersEnabled: Bool(true),
					VCSRepo:             &VCSRepoOptions{TagsRegex: String("foobar")},
				},
			},
			assertion: func(w *Workspace, options *WorkspaceTableOptions, err error) {
				assert.Nil(t, w)
				assert.EqualError(t, err, ErrUnsupportedBothTagsRegexAndFileTriggersEnabled.Error())
			},
		},
		{
			scenario: "when options include both non-empty tags-regex and file-triggers-enabled an error is returned",
			options: &WorkspaceTableOptions{
				updateOptions: &WorkspaceUpdateOptions{
					Name:                String("foobar"),
					FileTriggersEnabled: Bool(true),
					VCSRepo:             &VCSRepoOptions{TagsRegex: String("foobar")},
				},
			},
			assertion: func(w *Workspace, options *WorkspaceTableOptions, err error) {
				assert.Nil(t, w)
				assert.EqualError(t, err, ErrUnsupportedBothTagsRegexAndFileTriggersEnabled.Error())
			},
		},
		{
			scenario: "when options include both tags-regex and trigger-prefixes an error is returned",
			options: &WorkspaceTableOptions{
				updateOptions: &WorkspaceUpdateOptions{
					Name:                String("foobar"),
					FileTriggersEnabled: Bool(false),
					TriggerPrefixes:     []string{"/module-1", "/module-2"},
					VCSRepo:             &VCSRepoOptions{TagsRegex: String("foobar")},
				},
			},
			assertion: func(w *Workspace, options *WorkspaceTableOptions, err error) {
				assert.Nil(t, w)
				assert.EqualError(t, err, ErrUnsupportedBothTagsRegexAndTriggerPrefixes.Error())
			},
		},
		{
			scenario: "when options include both tags-regex and trigger-patterns error is returned",
			options: &WorkspaceTableOptions{
				updateOptions: &WorkspaceUpdateOptions{
					Name:                String("foobar"),
					FileTriggersEnabled: Bool(false),
					TriggerPatterns:     []string{"/module-1/**/*", "/**/networking/*"},
					VCSRepo:             &VCSRepoOptions{TagsRegex: String("foobar")},
				},
			},
			assertion: func(w *Workspace, options *WorkspaceTableOptions, err error) {
				assert.Nil(t, w)
				assert.EqualError(t, err, ErrUnsupportedBothTagsRegexAndTriggerPatterns.Error())
			},
		},
	}

	for _, tableTest := range workspaceTableTests {
		t.Run(tableTest.scenario, func(t *testing.T) {
			var workspace *Workspace
			var cleanup func()
			var err error
			if tableTest.setup != nil {
				workspace, cleanup = tableTest.setup(tableTest.options)
				defer cleanup()
			} else {
				workspace, err = client.Workspaces.Update(ctx, orgTest.Name, wTest.Name, *tableTest.options.updateOptions)
			}
			tableTest.assertion(workspace, tableTest.options, err)
		})
	}
}

func TestWorkspacesUpdateByID(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	t.Cleanup(orgTestCleanup)

	wTest, wCleanup := createWorkspace(t, client, orgTest)
	t.Cleanup(wCleanup)

	t.Run("when updating a subset of values", func(t *testing.T) {
		options := WorkspaceUpdateOptions{
			Name:             String(wTest.Name),
			AllowDestroyPlan: Bool(false),
			AutoApply:        Bool(true),
			Operations:       Bool(true),
			QueueAllRuns:     Bool(true),
			TerraformVersion: String("0.10.0"),
		}

		wAfter, err := client.Workspaces.UpdateByID(ctx, wTest.ID, options)
		require.NoError(t, err)

		assert.Equal(t, wTest.Name, wAfter.Name)
		assert.NotEqual(t, wTest.AllowDestroyPlan, wAfter.AllowDestroyPlan)
		assert.NotEqual(t, wTest.AutoApply, wAfter.AutoApply)
		assert.NotEqual(t, wTest.QueueAllRuns, wAfter.QueueAllRuns)
		assert.NotEqual(t, wTest.TerraformVersion, wAfter.TerraformVersion)
		assert.Equal(t, wTest.WorkingDirectory, wAfter.WorkingDirectory)
	})

	t.Run("with valid options", func(t *testing.T) {
		options := WorkspaceUpdateOptions{
			Name:                       String(randomString(t)),
			AllowDestroyPlan:           Bool(true),
			AutoApply:                  Bool(false),
			FileTriggersEnabled:        Bool(true),
			Operations:                 Bool(false),
			QueueAllRuns:               Bool(false),
			SpeculativeEnabled:         Bool(true),
			StructuredRunOutputEnabled: Bool(true),
			TerraformVersion:           String("0.11.1"),
			TriggerPrefixes:            []string{"/modules", "/shared"},
			WorkingDirectory:           String("baz/"),
		}

		w, err := client.Workspaces.UpdateByID(ctx, wTest.ID, options)
		require.NoError(t, err)

		// Get a refreshed view of the workspace from the API
		refreshed, err := client.Workspaces.Read(ctx, orgTest.Name, *options.Name)
		require.NoError(t, err)

		for _, item := range []*Workspace{
			w,
			refreshed,
		} {
			assert.Equal(t, *options.Name, item.Name)
			assert.Equal(t, *options.AllowDestroyPlan, item.AllowDestroyPlan)
			assert.Equal(t, *options.AutoApply, item.AutoApply)
			assert.Equal(t, *options.FileTriggersEnabled, item.FileTriggersEnabled)
			assert.Equal(t, *options.Operations, item.Operations)
			assert.Equal(t, *options.QueueAllRuns, item.QueueAllRuns)
			assert.Equal(t, *options.SpeculativeEnabled, item.SpeculativeEnabled)
			assert.Equal(t, *options.StructuredRunOutputEnabled, item.StructuredRunOutputEnabled)
			assert.Equal(t, *options.TerraformVersion, item.TerraformVersion)
			assert.Equal(t, options.TriggerPrefixes, item.TriggerPrefixes)
			assert.Equal(t, *options.WorkingDirectory, item.WorkingDirectory)
		}
	})

	t.Run("when an error is returned from the api", func(t *testing.T) {
		w, err := client.Workspaces.UpdateByID(ctx, wTest.ID, WorkspaceUpdateOptions{
			TerraformVersion: String("nonexisting"),
		})
		assert.Nil(t, w)
		assert.Error(t, err)
	})

	t.Run("without a valid workspace ID", func(t *testing.T) {
		w, err := client.Workspaces.UpdateByID(ctx, badIdentifier, WorkspaceUpdateOptions{})
		assert.Nil(t, w)
		assert.EqualError(t, err, ErrInvalidWorkspaceID.Error())
	})
}

func TestWorkspacesDelete(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	t.Cleanup(orgTestCleanup)

	// ignore workspace cleanup b/c it will be destroyed during tests
	wTest, _ := createWorkspace(t, client, orgTest)

	t.Run("with valid options", func(t *testing.T) {
		err := client.Workspaces.Delete(ctx, orgTest.Name, wTest.Name)
		require.NoError(t, err)

		// Try loading the workspace - it should fail.
		_, err = client.Workspaces.Read(ctx, orgTest.Name, wTest.Name)
		assert.Equal(t, ErrResourceNotFound, err)
	})

	t.Run("when organization is invalid", func(t *testing.T) {
		err := client.Workspaces.Delete(ctx, badIdentifier, wTest.Name)
		assert.EqualError(t, err, ErrInvalidOrg.Error())
	})

	t.Run("when workspace is invalid", func(t *testing.T) {
		err := client.Workspaces.Delete(ctx, orgTest.Name, badIdentifier)
		assert.EqualError(t, err, ErrInvalidWorkspaceValue.Error())
	})
}

func TestWorkspacesDeleteByID(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	t.Cleanup(orgTestCleanup)

	// ignore workspace cleanup b/c it will be destroyed during tests
	wTest, _ := createWorkspace(t, client, orgTest)

	t.Run("with valid options", func(t *testing.T) {
		err := client.Workspaces.DeleteByID(ctx, wTest.ID)
		require.NoError(t, err)

		// Try loading the workspace - it should fail.
		_, err = client.Workspaces.ReadByID(ctx, wTest.ID)
		assert.Equal(t, ErrResourceNotFound, err)
	})

	t.Run("without a valid workspace ID", func(t *testing.T) {
		err := client.Workspaces.DeleteByID(ctx, badIdentifier)
		assert.EqualError(t, err, ErrInvalidWorkspaceID.Error())
	})
}

func TestCanForceDeletePermission(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	t.Cleanup(orgTestCleanup)

	wTest, wCleanup := createWorkspace(t, client, orgTest)
	t.Cleanup(wCleanup)

	t.Run("workspace permission set includes can-force-delete", func(t *testing.T) {
		w, err := client.Workspaces.ReadByID(ctx, wTest.ID)
		require.NoError(t, err)
		assert.Equal(t, wTest, w)
		require.NotNil(t, w.Permissions)
		require.NotNil(t, w.Permissions.CanForceDelete)
		assert.True(t, *w.Permissions.CanForceDelete)
	})
}

func TestWorkspacesSafeDelete(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	t.Cleanup(orgTestCleanup)

	// ignore workspace cleanup b/c it will be destroyed during tests
	wTest, _ := createWorkspace(t, client, orgTest)

	t.Run("with valid options", func(t *testing.T) {
		err := client.Workspaces.SafeDelete(ctx, orgTest.Name, wTest.Name)
		require.NoError(t, err)

		// Try loading the workspace - it should fail.
		_, err = client.Workspaces.Read(ctx, orgTest.Name, wTest.Name)
		assert.Equal(t, ErrResourceNotFound, err)
	})

	t.Run("when organization is invalid", func(t *testing.T) {
		err := client.Workspaces.SafeDelete(ctx, badIdentifier, wTest.Name)
		assert.EqualError(t, err, ErrInvalidOrg.Error())
	})

	t.Run("when workspace is invalid", func(t *testing.T) {
		err := client.Workspaces.SafeDelete(ctx, orgTest.Name, badIdentifier)
		assert.EqualError(t, err, ErrInvalidWorkspaceValue.Error())
	})

	t.Run("when workspace is locked", func(t *testing.T) {
		wTest, workspaceCleanup := createWorkspace(t, client, orgTest)
		t.Cleanup(workspaceCleanup)
		w, err := client.Workspaces.Lock(ctx, wTest.ID, WorkspaceLockOptions{})
		require.NoError(t, err)
		require.True(t, w.Locked)

		err = client.Workspaces.SafeDelete(ctx, orgTest.Name, wTest.Name)
		assert.Contains(t, err.Error(), "conflict")
		assert.Contains(t, err.Error(), "currently locked")
	})

	t.Run("when workspace has resources under management", func(t *testing.T) {
		wTest, workspaceCleanup := createWorkspace(t, client, orgTest)
		t.Cleanup(workspaceCleanup)
		_, svTestCleanup := createStateVersion(t, client, 0, wTest)
		t.Cleanup(svTestCleanup)

		err := client.Workspaces.SafeDelete(ctx, orgTest.Name, wTest.Name)
		// cant verify the exact error here because it is timing dependent on the backend
		// based on whether the state version has been processed yet
		assert.Contains(t, err.Error(), "conflict")
	})
}

func TestWorkspacesSafeDeleteByID(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	t.Cleanup(orgTestCleanup)

	// ignore workspace cleanup b/c it will be destroyed during tests
	wTest, _ := createWorkspace(t, client, orgTest)

	t.Run("with valid options", func(t *testing.T) {
		err := client.Workspaces.SafeDeleteByID(ctx, wTest.ID)
		require.NoError(t, err)

		// Try loading the workspace - it should fail.
		_, err = client.Workspaces.ReadByID(ctx, wTest.ID)
		assert.Equal(t, ErrResourceNotFound, err)
	})

	t.Run("without a valid workspace ID", func(t *testing.T) {
		err := client.Workspaces.SafeDeleteByID(ctx, badIdentifier)
		assert.EqualError(t, err, ErrInvalidWorkspaceID.Error())
	})

	t.Run("when workspace is locked", func(t *testing.T) {
		wTest, workspaceCleanup := createWorkspace(t, client, orgTest)
		t.Cleanup(workspaceCleanup)
		w, err := client.Workspaces.Lock(ctx, wTest.ID, WorkspaceLockOptions{})
		require.NoError(t, err)
		require.True(t, w.Locked)

		err = client.Workspaces.SafeDeleteByID(ctx, wTest.ID)
		assert.Contains(t, err.Error(), "conflict")
		assert.Contains(t, err.Error(), "currently locked")
	})

	t.Run("when workspace has resources under management", func(t *testing.T) {
		wTest, workspaceCleanup := createWorkspace(t, client, orgTest)
		t.Cleanup(workspaceCleanup)
		_, svTestCleanup := createStateVersion(t, client, 0, wTest)
		t.Cleanup(svTestCleanup)

		err := client.Workspaces.SafeDeleteByID(ctx, wTest.ID)
		// cant verify the exact error here because it is timing dependent on the backend
		// based on whether the state version has been processed yet
		assert.Contains(t, err.Error(), "conflict")
	})
}

func TestWorkspacesRemoveVCSConnection(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	t.Cleanup(orgTestCleanup)

	wTest, wTestCleanup := createWorkspaceWithVCS(t, client, orgTest, WorkspaceCreateOptions{})
	t.Cleanup(wTestCleanup)

	t.Run("remove vcs integration", func(t *testing.T) {
		w, err := client.Workspaces.RemoveVCSConnection(ctx, orgTest.Name, wTest.Name)
		require.NoError(t, err)
		assert.Equal(t, (*VCSRepo)(nil), w.VCSRepo)
	})
}

func TestWorkspacesRemoveVCSConnectionByID(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	t.Cleanup(orgTestCleanup)

	wTest, wTestCleanup := createWorkspaceWithVCS(t, client, orgTest, WorkspaceCreateOptions{})
	t.Cleanup(wTestCleanup)

	t.Run("remove vcs integration", func(t *testing.T) {
		w, err := client.Workspaces.RemoveVCSConnectionByID(ctx, wTest.ID)
		require.NoError(t, err)
		assert.Equal(t, (*VCSRepo)(nil), w.VCSRepo)
	})
}

func TestWorkspacesLock(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	t.Cleanup(orgTestCleanup)

	wTest, wTestCleanup := createWorkspace(t, client, orgTest)
	t.Cleanup(wTestCleanup)

	t.Run("with valid options", func(t *testing.T) {
		w, err := client.Workspaces.Lock(ctx, wTest.ID, WorkspaceLockOptions{})
		require.NoError(t, err)
		assert.True(t, w.Locked)
	})

	t.Run("when workspace is already locked", func(t *testing.T) {
		_, err := client.Workspaces.Lock(ctx, wTest.ID, WorkspaceLockOptions{})
		assert.Equal(t, ErrWorkspaceLocked, err)
	})

	t.Run("without a valid workspace ID", func(t *testing.T) {
		w, err := client.Workspaces.Lock(ctx, badIdentifier, WorkspaceLockOptions{})
		assert.Nil(t, w)
		assert.EqualError(t, err, ErrInvalidWorkspaceID.Error())
	})
}

func TestWorkspacesUnlock(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	t.Cleanup(orgTestCleanup)

	wTest, wTestCleanup := createWorkspace(t, client, orgTest)
	t.Cleanup(wTestCleanup)

	w, err := client.Workspaces.Lock(ctx, wTest.ID, WorkspaceLockOptions{})
	if err != nil {
		orgTestCleanup()
	}
	require.NoError(t, err)
	require.True(t, w.Locked)

	t.Run("with valid options", func(t *testing.T) {
		w, err := client.Workspaces.Unlock(ctx, wTest.ID)
		require.NoError(t, err)
		assert.False(t, w.Locked)
	})

	t.Run("when workspace is already unlocked", func(t *testing.T) {
		_, err := client.Workspaces.Unlock(ctx, wTest.ID)
		assert.Equal(t, ErrWorkspaceNotLocked, err)
	})

	t.Run("when a workspace is locked by a run", func(t *testing.T) {
		wTest2, wTest2Cleanup := createWorkspace(t, client, orgTest)
		t.Cleanup(wTest2Cleanup)

		_, rTestCleanup := createRun(t, client, wTest2)
		t.Cleanup(rTestCleanup)

		// Wait for wTest2 to be locked by a run
		waitForRunLock(t, client, wTest2.ID)

		_, err = client.Workspaces.Unlock(ctx, wTest2.ID)
		assert.Equal(t, ErrWorkspaceLockedByRun, err)
	})

	t.Run("without a valid workspace ID", func(t *testing.T) {
		w, err := client.Workspaces.Unlock(ctx, badIdentifier)
		assert.Nil(t, w)
		assert.EqualError(t, err, ErrInvalidWorkspaceID.Error())
	})
}

func TestWorkspacesForceUnlock(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	t.Cleanup(orgTestCleanup)

	wTest, wTestCleanup := createWorkspace(t, client, orgTest)
	t.Cleanup(wTestCleanup)

	w, err := client.Workspaces.Lock(ctx, wTest.ID, WorkspaceLockOptions{})
	if err != nil {
		orgTestCleanup()
	}
	require.NoError(t, err)
	require.True(t, w.Locked)

	t.Run("with valid options", func(t *testing.T) {
		w, err := client.Workspaces.ForceUnlock(ctx, wTest.ID)
		require.NoError(t, err)
		assert.False(t, w.Locked)
	})

	t.Run("when workspace is already unlocked", func(t *testing.T) {
		_, err := client.Workspaces.ForceUnlock(ctx, wTest.ID)
		assert.Equal(t, ErrWorkspaceNotLocked, err)
	})

	t.Run("without a valid workspace ID", func(t *testing.T) {
		w, err := client.Workspaces.ForceUnlock(ctx, badIdentifier)
		assert.Nil(t, w)
		assert.EqualError(t, err, ErrInvalidWorkspaceID.Error())
	})
}

func TestWorkspacesAssignSSHKey(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	t.Cleanup(orgTestCleanup)

	wTest, wTestCleanup := createWorkspace(t, client, orgTest)
	t.Cleanup(wTestCleanup)

	sshKeyTest, sshKeyTestCleanup := createSSHKey(t, client, orgTest)
	t.Cleanup(sshKeyTestCleanup)

	t.Run("with valid options", func(t *testing.T) {
		w, err := client.Workspaces.AssignSSHKey(ctx, wTest.ID, WorkspaceAssignSSHKeyOptions{
			SSHKeyID: String(sshKeyTest.ID),
		})
		require.NoError(t, err)
		require.NotNil(t, w.SSHKey)
		assert.Equal(t, w.SSHKey.ID, sshKeyTest.ID)
	})

	t.Run("without an SSH key ID", func(t *testing.T) {
		w, err := client.Workspaces.AssignSSHKey(ctx, wTest.ID, WorkspaceAssignSSHKeyOptions{})
		assert.Nil(t, w)
		assert.Equal(t, err, ErrRequiredSHHKeyID)
	})

	t.Run("without a valid SSH key ID", func(t *testing.T) {
		w, err := client.Workspaces.AssignSSHKey(ctx, wTest.ID, WorkspaceAssignSSHKeyOptions{
			SSHKeyID: String(badIdentifier),
		})
		assert.Nil(t, w)
		assert.Equal(t, err, ErrInvalidSHHKeyID)
	})

	t.Run("without a valid workspace ID", func(t *testing.T) {
		w, err := client.Workspaces.AssignSSHKey(ctx, badIdentifier, WorkspaceAssignSSHKeyOptions{
			SSHKeyID: String(sshKeyTest.ID),
		})
		assert.Nil(t, w)
		assert.EqualError(t, err, ErrInvalidWorkspaceID.Error())
	})
}

func TestWorkspacesUnassignSSHKey(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	t.Cleanup(orgTestCleanup)

	wTest, wTestCleanup := createWorkspace(t, client, orgTest)
	t.Cleanup(wTestCleanup)

	sshKeyTest, sshKeyTestCleanup := createSSHKey(t, client, orgTest)
	t.Cleanup(sshKeyTestCleanup)

	w, err := client.Workspaces.AssignSSHKey(ctx, wTest.ID, WorkspaceAssignSSHKeyOptions{
		SSHKeyID: String(sshKeyTest.ID),
	})
	if err != nil {
		orgTestCleanup()
	}
	require.NoError(t, err)
	require.NotNil(t, w.SSHKey)
	require.Equal(t, w.SSHKey.ID, sshKeyTest.ID)

	t.Run("with valid options", func(t *testing.T) {
		w, err := client.Workspaces.UnassignSSHKey(ctx, wTest.ID)
		assert.Nil(t, err)
		assert.Nil(t, w.SSHKey)
	})

	t.Run("without a valid workspace ID", func(t *testing.T) {
		w, err := client.Workspaces.UnassignSSHKey(ctx, badIdentifier)
		assert.Nil(t, w)
		assert.EqualError(t, err, ErrInvalidWorkspaceID.Error())
	})
}

func TestWorkspaces_AddRemoteStateConsumers(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	t.Cleanup(orgTestCleanup)

	wTest, wTestCleanup := createWorkspace(t, client, orgTest)
	t.Cleanup(wTestCleanup)

	// Update workspace to not allow global remote state
	options := WorkspaceUpdateOptions{
		GlobalRemoteState: Bool(false),
	}
	wTest, err := client.Workspaces.Update(ctx, orgTest.Name, wTest.Name, options)
	require.NoError(t, err)

	t.Run("successfully adds a remote state consumer", func(t *testing.T) {
		wTestConsumer1, wTestCleanupConsumer1 := createWorkspace(t, client, orgTest)
		t.Cleanup(wTestCleanupConsumer1)
		wTestConsumer2, wTestCleanupConsumer2 := createWorkspace(t, client, orgTest)
		t.Cleanup(wTestCleanupConsumer2)

		err := client.Workspaces.AddRemoteStateConsumers(ctx, wTest.ID, WorkspaceAddRemoteStateConsumersOptions{
			Workspaces: []*Workspace{wTestConsumer1, wTestConsumer2},
		})
		require.NoError(t, err)

		_, err = client.Workspaces.Read(ctx, orgTest.Name, wTest.Name)
		require.NoError(t, err)

		rsc, err := client.Workspaces.ListRemoteStateConsumers(ctx, wTest.ID, nil)
		require.NoError(t, err)
		assert.Equal(t, 2, len(rsc.Items))
		assert.Contains(t, rsc.Items, wTestConsumer1)
		assert.Contains(t, rsc.Items, wTestConsumer2)
	})

	t.Run("with invalid options", func(t *testing.T) {
		err := client.Workspaces.AddRemoteStateConsumers(ctx, wTest.ID, WorkspaceAddRemoteStateConsumersOptions{})
		require.Error(t, err)
		assert.EqualError(t, err, ErrWorkspacesRequired.Error())

		err = client.Workspaces.AddRemoteStateConsumers(ctx, wTest.ID, WorkspaceAddRemoteStateConsumersOptions{
			Workspaces: []*Workspace{},
		})
		require.Error(t, err)
		assert.EqualError(t, err, ErrWorkspaceMinLimit.Error())
	})

	t.Run("without a valid workspace ID", func(t *testing.T) {
		err := client.Workspaces.AddRemoteStateConsumers(ctx, badIdentifier, WorkspaceAddRemoteStateConsumersOptions{})
		require.Error(t, err)
		assert.EqualError(t, err, ErrInvalidWorkspaceID.Error())
	})
}

func TestWorkspaces_RemoveRemoteStateConsumers(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	t.Cleanup(orgTestCleanup)

	wTest, wTestCleanup := createWorkspace(t, client, orgTest)
	t.Cleanup(wTestCleanup)

	// Update workspace to not allow global remote state
	options := WorkspaceUpdateOptions{
		GlobalRemoteState: Bool(false),
	}
	wTest, err := client.Workspaces.Update(ctx, orgTest.Name, wTest.Name, options)
	require.NoError(t, err)

	t.Run("successfully removes a remote state consumer", func(t *testing.T) {
		wTestConsumer1, wTestCleanupConsumer1 := createWorkspace(t, client, orgTest)
		t.Cleanup(wTestCleanupConsumer1)
		wTestConsumer2, wTestCleanupConsumer2 := createWorkspace(t, client, orgTest)
		t.Cleanup(wTestCleanupConsumer2)

		err := client.Workspaces.AddRemoteStateConsumers(ctx, wTest.ID, WorkspaceAddRemoteStateConsumersOptions{
			Workspaces: []*Workspace{wTestConsumer1, wTestConsumer2},
		})
		require.NoError(t, err)

		rsc, err := client.Workspaces.ListRemoteStateConsumers(ctx, wTest.ID, nil)
		require.NoError(t, err)
		assert.Equal(t, 2, len(rsc.Items))
		assert.Contains(t, rsc.Items, wTestConsumer1)
		assert.Contains(t, rsc.Items, wTestConsumer2)

		err = client.Workspaces.RemoveRemoteStateConsumers(ctx, wTest.ID, WorkspaceRemoveRemoteStateConsumersOptions{
			Workspaces: []*Workspace{wTestConsumer1},
		})
		require.NoError(t, err)

		_, err = client.Workspaces.Read(ctx, orgTest.Name, wTest.Name)
		require.NoError(t, err)

		rsc, err = client.Workspaces.ListRemoteStateConsumers(ctx, wTest.ID, nil)
		require.NoError(t, err)
		assert.Contains(t, rsc.Items, wTestConsumer2)
		assert.Equal(t, 1, len(rsc.Items))

		err = client.Workspaces.RemoveRemoteStateConsumers(ctx, wTest.ID, WorkspaceRemoveRemoteStateConsumersOptions{
			Workspaces: []*Workspace{wTestConsumer2},
		})
		require.NoError(t, err)

		rsc, err = client.Workspaces.ListRemoteStateConsumers(ctx, wTest.ID, nil)
		require.NoError(t, err)
		assert.Empty(t, len(rsc.Items))
	})

	t.Run("with invalid options", func(t *testing.T) {
		err := client.Workspaces.RemoveRemoteStateConsumers(ctx, wTest.ID, WorkspaceRemoveRemoteStateConsumersOptions{})
		require.Error(t, err)
		assert.EqualError(t, err, ErrWorkspacesRequired.Error())

		err = client.Workspaces.RemoveRemoteStateConsumers(ctx, wTest.ID, WorkspaceRemoveRemoteStateConsumersOptions{
			Workspaces: []*Workspace{},
		})
		require.Error(t, err)
		assert.EqualError(t, err, ErrWorkspaceMinLimit.Error())
	})

	t.Run("without a valid workspace ID", func(t *testing.T) {
		err := client.Workspaces.RemoveRemoteStateConsumers(ctx, badIdentifier, WorkspaceRemoveRemoteStateConsumersOptions{})
		require.Error(t, err)
		assert.EqualError(t, err, ErrInvalidWorkspaceID.Error())
	})
}

func TestWorkspaces_UpdateRemoteStateConsumers(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	t.Cleanup(orgTestCleanup)

	wTest, wTestCleanup := createWorkspace(t, client, orgTest)
	t.Cleanup(wTestCleanup)

	// Update workspace to not allow global remote state
	options := WorkspaceUpdateOptions{
		GlobalRemoteState: Bool(false),
	}
	wTest, err := client.Workspaces.Update(ctx, orgTest.Name, wTest.Name, options)
	require.NoError(t, err)

	t.Run("successfully updates a remote state consumer", func(t *testing.T) {
		wTestConsumer1, wTestCleanupConsumer1 := createWorkspace(t, client, orgTest)
		t.Cleanup(wTestCleanupConsumer1)
		wTestConsumer2, wTestCleanupConsumer2 := createWorkspace(t, client, orgTest)
		t.Cleanup(wTestCleanupConsumer2)

		err := client.Workspaces.AddRemoteStateConsumers(ctx, wTest.ID, WorkspaceAddRemoteStateConsumersOptions{
			Workspaces: []*Workspace{wTestConsumer1},
		})
		require.NoError(t, err)

		rsc, err := client.Workspaces.ListRemoteStateConsumers(ctx, wTest.ID, nil)
		require.NoError(t, err)
		assert.Equal(t, 1, len(rsc.Items))
		assert.Contains(t, rsc.Items, wTestConsumer1)

		err = client.Workspaces.UpdateRemoteStateConsumers(ctx, wTest.ID, WorkspaceUpdateRemoteStateConsumersOptions{
			Workspaces: []*Workspace{wTestConsumer2},
		})
		require.NoError(t, err)

		rsc, err = client.Workspaces.ListRemoteStateConsumers(ctx, wTest.ID, nil)
		require.NoError(t, err)
		assert.Equal(t, 1, len(rsc.Items))
		assert.Contains(t, rsc.Items, wTestConsumer2)

	})

	t.Run("with invalid options", func(t *testing.T) {
		err := client.Workspaces.UpdateRemoteStateConsumers(ctx, wTest.ID, WorkspaceUpdateRemoteStateConsumersOptions{})
		require.Error(t, err)
		assert.EqualError(t, err, ErrWorkspacesRequired.Error())

		err = client.Workspaces.UpdateRemoteStateConsumers(ctx, wTest.ID, WorkspaceUpdateRemoteStateConsumersOptions{
			Workspaces: []*Workspace{},
		})
		require.Error(t, err)
		assert.EqualError(t, err, ErrWorkspaceMinLimit.Error())
	})

	t.Run("without a valid workspace ID", func(t *testing.T) {
		err := client.Workspaces.UpdateRemoteStateConsumers(ctx, badIdentifier, WorkspaceUpdateRemoteStateConsumersOptions{})
		require.Error(t, err)
		assert.EqualError(t, err, ErrInvalidWorkspaceID.Error())
	})
}

func TestWorkspaces_AddTags(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	t.Cleanup(orgTestCleanup)

	wTest, wTestCleanup := createWorkspace(t, client, orgTest)
	t.Cleanup(wTestCleanup)

	options := WorkspaceAddTagsOptions{
		Tags: []*Tag{
			{
				Name: "tag1",
			},
			{
				Name: "tag2",
			},
			{
				Name: "tag3",
			},
		},
	}

	t.Run("successfully adds tags", func(t *testing.T) {
		err := client.Workspaces.AddTags(ctx, wTest.ID, options)
		require.NoError(t, err)

		w, err := client.Workspaces.Read(ctx, orgTest.Name, wTest.Name)
		require.NoError(t, err)
		assert.Equal(t, 3, len(w.TagNames))
		assert.Equal(t, w.TagNames, []string{"tag1", "tag2", "tag3"})

		err = client.Workspaces.AddTags(ctx, wTest.ID, WorkspaceAddTagsOptions{
			Tags: []*Tag{
				{
					Name: "tag4",
				},
			},
		})
		require.NoError(t, err)

		w, err = client.Workspaces.Read(ctx, orgTest.Name, wTest.Name)
		require.NoError(t, err)
		assert.Equal(t, 4, len(w.TagNames))
		sort.Strings(w.TagNames)
		assert.EqualValues(t, w.TagNames, []string{"tag1", "tag2", "tag3", "tag4"})

		wt, err := client.Workspaces.ListTags(ctx, wTest.ID, nil)
		require.NoError(t, err)
		assert.Equal(t, 4, len(wt.Items))
		assert.Equal(t, wt.Items[3].Name, "tag4")
	})

	t.Run("successfully adds tags by id and name", func(t *testing.T) {
		wTest2, wTest2Cleanup := createWorkspace(t, client, orgTest)
		t.Cleanup(wTest2Cleanup)

		// add a tag to another workspace
		err := client.Workspaces.AddTags(ctx, wTest2.ID, WorkspaceAddTagsOptions{
			Tags: []*Tag{
				{
					Name: "tagbyid",
				},
			},
		})
		require.NoError(t, err)

		// get the id of the new tag
		tags, err := client.Workspaces.ListTags(ctx, wTest2.ID, nil)
		require.NoError(t, err)

		// add the tag to our workspace by id
		err = client.Workspaces.AddTags(ctx, wTest.ID, WorkspaceAddTagsOptions{
			Tags: []*Tag{
				{
					ID: tags.Items[0].ID,
				},
			},
		})
		require.NoError(t, err)

		// tag is now in the tag_names
		w, err := client.Workspaces.Read(ctx, orgTest.Name, wTest.Name)
		require.NoError(t, err)
		assert.Equal(t, 5, len(w.TagNames))
		sort.Strings(w.TagNames)
		assert.Equal(t, w.TagNames, []string{"tag1", "tag2", "tag3", "tag4", "tagbyid"})

		// tag is now in our tag list
		wt, err := client.Workspaces.ListTags(ctx, wTest.ID, nil)
		require.NoError(t, err)
		assert.Equal(t, 5, len(wt.Items))
		assert.Equal(t, wt.Items[4].ID, tags.Items[0].ID)
		assert.Equal(t, wt.Items[4].Name, "tagbyid")
	})

	t.Run("with invalid options", func(t *testing.T) {
		err := client.Workspaces.AddTags(ctx, wTest.ID, WorkspaceAddTagsOptions{
			Tags: []*Tag{},
		})
		require.Error(t, err)
		assert.EqualError(t, err, ErrMissingTagIdentifier.Error())
	})

	t.Run("without a valid workspace ID", func(t *testing.T) {
		err := client.Workspaces.AddTags(ctx, badIdentifier, WorkspaceAddTagsOptions{})
		require.Error(t, err)
		assert.EqualError(t, err, ErrInvalidWorkspaceID.Error())
	})
}

func TestWorkspaces_RemoveTags(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	t.Cleanup(orgTestCleanup)

	wTest, wTestCleanup := createWorkspace(t, client, orgTest)
	t.Cleanup(wTestCleanup)

	tags := []*Tag{
		{
			Name: "tag1",
		},
		{
			Name: "tag2",
		},
		{
			Name: "tag3",
		},
	}
	addOptions := WorkspaceAddTagsOptions{
		Tags: tags,
	}
	removeOptions := WorkspaceRemoveTagsOptions{
		Tags: tags[0:2],
	}

	t.Run("successfully removes tags", func(t *testing.T) {
		err := client.Workspaces.AddTags(ctx, wTest.ID, addOptions)
		require.NoError(t, err)

		w, err := client.Workspaces.Read(ctx, orgTest.Name, wTest.Name)
		require.NoError(t, err)
		assert.Equal(t, 3, len(w.TagNames))
		assert.Equal(t, w.TagNames, []string{"tag1", "tag2", "tag3"})

		err = client.Workspaces.RemoveTags(ctx, wTest.ID, removeOptions)
		require.NoError(t, err)

		w, err = client.Workspaces.Read(ctx, orgTest.Name, wTest.Name)
		require.NoError(t, err)
		assert.Equal(t, 1, len(w.TagNames))
		assert.Equal(t, w.TagNames, []string{"tag3"})

		wt, err := client.Workspaces.ListTags(ctx, wTest.ID, nil)
		require.NoError(t, err)
		assert.Equal(t, 1, len(wt.Items))
		assert.EqualValues(t, wt.Items[0].Name, "tag3")
	})

	t.Run("attempts to remove a tag that doesn't exist", func(t *testing.T) {
		err := client.Workspaces.RemoveTags(ctx, wTest.ID, WorkspaceRemoveTagsOptions{
			Tags: []*Tag{
				{
					Name: "NonExistentTag",
				},
			},
		})
		require.NoError(t, err)
	})

	t.Run("with invalid options", func(t *testing.T) {
		err := client.Workspaces.RemoveTags(ctx, wTest.ID, WorkspaceRemoveTagsOptions{
			Tags: []*Tag{},
		})
		require.Error(t, err)
		assert.EqualError(t, err, ErrMissingTagIdentifier.Error())
	})

	t.Run("without a valid workspace ID", func(t *testing.T) {
		err := client.Workspaces.RemoveTags(ctx, badIdentifier, WorkspaceRemoveTagsOptions{})
		require.Error(t, err)
		assert.EqualError(t, err, ErrInvalidWorkspaceID.Error())
	})
}

func TestWorkspace_Unmarshal(t *testing.T) {
	data := map[string]interface{}{
		"data": map[string]interface{}{
			"type": "workspaces",
			"id":   "ws-1234",
			"attributes": map[string]interface{}{
				"name":           "my-workspace",
				"auto-apply":     true,
				"created-at":     "2020-07-15T23:38:43.821Z",
				"resource-count": 2,
				"permissions": map[string]interface{}{
					"can-update": true,
					"can-lock":   true,
				},
				"vcs-repo": map[string]interface{}{
					"branch":              "main",
					"display-identifier":  "repo-name",
					"identifier":          "hashicorp/repo-name",
					"ingress-submodules":  true,
					"oauth-token-id":      "token",
					"repository-http-url": "github.com",
					"service-provider":    "github",
					"webhook-url":         "https://app.terraform.io/webhooks/vcs/aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee",
				},
				"actions": map[string]interface{}{
					"is-destroyable": true,
				},
				"trigger-prefixes": []string{"prefix-"},
				"trigger-patterns": []string{"pattern1/**/*", "pattern2/**/submodule/*"},
			},
		},
	}

	byteData, err := json.Marshal(data)
	require.NoError(t, err)

	responseBody := bytes.NewReader(byteData)
	ws := &Workspace{}
	err = unmarshalResponse(responseBody, ws)
	require.NoError(t, err)

	iso8601TimeFormat := "2006-01-02T15:04:05Z"
	parsedTime, _ := time.Parse(iso8601TimeFormat, "2020-07-15T23:38:43.821Z")

	assert.Equal(t, ws.ID, "ws-1234")
	assert.Equal(t, ws.Name, "my-workspace")
	assert.Equal(t, ws.AutoApply, true)
	assert.Equal(t, ws.CreatedAt, parsedTime)
	assert.Equal(t, ws.ResourceCount, 2)
	assert.Equal(t, ws.Permissions.CanUpdate, true)
	assert.Equal(t, ws.Permissions.CanLock, true)
	assert.Equal(t, ws.VCSRepo.Branch, "main")
	assert.Equal(t, ws.VCSRepo.DisplayIdentifier, "repo-name")
	assert.Equal(t, ws.VCSRepo.Identifier, "hashicorp/repo-name")
	assert.Equal(t, ws.VCSRepo.IngressSubmodules, true)
	assert.Equal(t, ws.VCSRepo.OAuthTokenID, "token")
	assert.Equal(t, ws.VCSRepo.RepositoryHTTPURL, "github.com")
	assert.Equal(t, ws.VCSRepo.ServiceProvider, "github")
	assert.Equal(t, ws.VCSRepo.WebhookURL, "https://app.terraform.io/webhooks/vcs/aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee")
	assert.Equal(t, ws.Actions.IsDestroyable, true)
	assert.Equal(t, ws.TriggerPrefixes, []string{"prefix-"})
	assert.Equal(t, ws.TriggerPatterns, []string{"pattern1/**/*", "pattern2/**/submodule/*"})
}

func TestWorkspaceCreateOptions_Marshal(t *testing.T) {
	opts := WorkspaceCreateOptions{
		AllowDestroyPlan: Bool(true),
		Name:             String("my-workspace"),
		TriggerPrefixes:  []string{"prefix-"},
		TriggerPatterns:  []string{"pattern1/**/*", "pattern2/**/*"},
		VCSRepo: &VCSRepoOptions{
			Identifier:   String("id"),
			OAuthTokenID: String("token"),
		},
	}

	reqBody, err := serializeRequestBody(&opts)
	require.NoError(t, err)
	req, err := retryablehttp.NewRequest("POST", "url", reqBody)
	require.NoError(t, err)
	bodyBytes, err := req.BodyBytes()
	require.NoError(t, err)

	expectedBody := `{"data":{"type":"workspaces","attributes":{"allow-destroy-plan":true,"name":"my-workspace","trigger-patterns":["pattern1/**/*","pattern2/**/*"],"trigger-prefixes":["prefix-"],"vcs-repo":{"identifier":"id","oauth-token-id":"token"}}}}
`
	assert.Equal(t, expectedBody, string(bodyBytes))
}

func TestWorkspacesRunTasksPermission(t *testing.T) {
	skipIfFreeOnly(t)
	skipIfBeta(t)

	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	t.Cleanup(orgTestCleanup)

	wTest, wTestCleanup := createWorkspace(t, client, orgTest)
	t.Cleanup(wTestCleanup)

	t.Run("when the workspace exists", func(t *testing.T) {
		w, err := client.Workspaces.Read(ctx, orgTest.Name, wTest.Name)
		require.NoError(t, err)
		assert.Equal(t, wTest, w)
		assert.True(t, w.Permissions.CanManageRunTasks)
	})
}
