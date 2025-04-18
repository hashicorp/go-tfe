// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package tfe

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"testing"
	"time"

	retryablehttp "github.com/hashicorp/go-retryablehttp"
	"github.com/hashicorp/jsonapi"
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
	setup     func(t *testing.T, options *WorkspaceTableOptions) (w *Workspace, cleanup func())
	assertion func(t *testing.T, w *Workspace, options *WorkspaceTableOptions, err error)
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
	wTest3, wTest3Cleanup := createWorkspace(t, client, orgTest)
	t.Cleanup(wTest3Cleanup)

	t.Run("without list options", func(t *testing.T) {
		wl, err := client.Workspaces.List(ctx, orgTest.Name, nil)
		require.NoError(t, err)
		assert.Contains(t, wl.Items, wTest1)
		assert.Contains(t, wl.Items, wTest2)
		assert.Equal(t, 1, wl.CurrentPage)
		assert.Equal(t, 3, wl.TotalCount)
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
		assert.Equal(t, 3, wl.TotalCount)
	})

	t.Run("when sorting by workspace names", func(t *testing.T) {
		wl, err := client.Workspaces.List(ctx, orgTest.Name, &WorkspaceListOptions{
			Sort: "name",
		})
		require.NoError(t, err)
		require.NotEmpty(t, wl.Items)
		require.GreaterOrEqual(t, len(wl.Items), 2)
		assert.Equal(t, wl.Items[0].Name < wl.Items[1].Name, true)
	})

	t.Run("when sorting workspaces on current-run.created-at", func(t *testing.T) {
		_, unappliedCleanup1 := createRunUnapplied(t, client, wTest2)
		t.Cleanup(unappliedCleanup1)

		_, unappliedCleanup2 := createRunUnapplied(t, client, wTest3)
		t.Cleanup(unappliedCleanup2)

		wl, err := client.Workspaces.List(ctx, orgTest.Name, &WorkspaceListOptions{
			Include: []WSIncludeOpt{WSCurrentRun},
			Sort:    "current-run.created-at",
		})

		require.NoError(t, err)
		require.NotEmpty(t, wl.Items)
		require.GreaterOrEqual(t, len(wl.Items), 2)
		assert.True(t, wl.Items[1].CurrentRun.CreatedAt.After(wl.Items[0].CurrentRun.CreatedAt))
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
		for wsID, tag := range map[string]string{wTest1.ID: "foo", wTest2.ID: "bar", wTest3.ID: "foo"} {
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
			if ws.ID != wTest1.ID {
				continue
			}
			foundWTest1 = true
			require.NotNil(t, wl.Items[0].CurrentStateVersion)
			assert.NotEmpty(t, wl.Items[0].CurrentStateVersion.DownloadURL)

			require.NotNil(t, wl.Items[0].CurrentRun)
			assert.NotEmpty(t, wl.Items[0].CurrentRun.Message)
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

	t.Run("when using a tags filter", func(t *testing.T) {
		skipUnlessBeta(t)

		w1, wTestCleanup1 := createWorkspaceWithOptions(t, client, orgTest, WorkspaceCreateOptions{
			Name: String(randomString(t)),
			TagBindings: []*TagBinding{
				{Key: "key1", Value: "value1"},
				{Key: "key2", Value: "value2a"},
			},
		})
		w2, wTestCleanup2 := createWorkspaceWithOptions(t, client, orgTest, WorkspaceCreateOptions{
			Name: String(randomString(t)),
			TagBindings: []*TagBinding{
				{Key: "key2", Value: "value2b"},
				{Key: "key3", Value: "value3"},
			},
		})
		t.Cleanup(wTestCleanup1)
		t.Cleanup(wTestCleanup2)

		// List all the workspaces under the given tag
		wl, err := client.Workspaces.List(ctx, orgTest.Name, &WorkspaceListOptions{
			TagBindings: []*TagBinding{
				{Key: "key1"},
			},
		})
		assert.NoError(t, err)
		assert.Len(t, wl.Items, 1)
		assert.Contains(t, wl.Items, w1)

		wl2, err := client.Workspaces.List(ctx, orgTest.Name, &WorkspaceListOptions{
			TagBindings: []*TagBinding{
				{Key: "key2"},
			},
		})
		assert.NoError(t, err)
		assert.Len(t, wl2.Items, 2)
		assert.Contains(t, wl2.Items, w1, w2)

		wl3, err := client.Workspaces.List(ctx, orgTest.Name, &WorkspaceListOptions{
			TagBindings: []*TagBinding{
				{Key: "key2", Value: "value2b"},
			},
		})
		assert.NoError(t, err)
		assert.Len(t, wl3.Items, 1)
		assert.Contains(t, wl3.Items, w2)
	})

	t.Run("when including effective tag bindings", func(t *testing.T) {
		skipUnlessBeta(t)

		orgTest2, orgTest2Cleanup := createOrganization(t, client)
		t.Cleanup(orgTest2Cleanup)

		prj, pTestCleanup1 := createProjectWithOptions(t, client, orgTest2, ProjectCreateOptions{
			Name: randomStringWithoutSpecialChar(t),
			TagBindings: []*TagBinding{
				{Key: "key3", Value: "value3"},
			},
		})
		t.Cleanup(pTestCleanup1)

		_, wTestCleanup1 := createWorkspaceWithOptions(t, client, orgTest2, WorkspaceCreateOptions{
			Name:    String(randomString(t)),
			Project: prj,
			TagBindings: []*TagBinding{
				{Key: "key1", Value: "value1"},
				{Key: "key2", Value: "value2a"},
			},
		})
		t.Cleanup(wTestCleanup1)

		wl, err := client.Workspaces.List(ctx, orgTest2.Name, &WorkspaceListOptions{
			Include: []WSIncludeOpt{WSEffectiveTagBindings},
		})
		require.NoError(t, err)
		require.Len(t, wl.Items, 1)
		require.Len(t, wl.Items[0].EffectiveTagBindings, 3)
		assert.NotEmpty(t, wl.Items[0].EffectiveTagBindings[0].Key)
		assert.NotEmpty(t, wl.Items[0].EffectiveTagBindings[0].Value)
		assert.NotEmpty(t, wl.Items[0].EffectiveTagBindings[1].Key)
		assert.NotEmpty(t, wl.Items[0].EffectiveTagBindings[1].Value)
		assert.NotEmpty(t, wl.Items[0].EffectiveTagBindings[2].Key)
		assert.NotEmpty(t, wl.Items[0].EffectiveTagBindings[2].Value)

		inheritedTagsFound := 0
		for _, tag := range wl.Items[0].EffectiveTagBindings {
			if tag.Links["inherited-from"] != nil {
				inheritedTagsFound += 1
			}
		}

		if inheritedTagsFound != 1 {
			t.Fatalf("Expected 1 inherited tag, got %d", inheritedTagsFound)
		}
	})

	t.Run("when using project id filter and project contains workspaces", func(t *testing.T) {
		// create a project in the orgTest
		p, pTestCleanup := createProject(t, client, orgTest)
		defer pTestCleanup()
		// create a workspace with project
		w, wTestCleanup := createWorkspaceWithOptions(t, client, orgTest, WorkspaceCreateOptions{
			Name:    String(randomString(t)),
			Project: p,
		})
		defer wTestCleanup()

		// List all the workspaces under the given ProjectID
		wl, err := client.Workspaces.List(ctx, orgTest.Name, &WorkspaceListOptions{
			ProjectID: p.ID,
		})
		require.NoError(t, err)
		assert.Contains(t, wl.Items, w)
	})

	t.Run("when using project id filter but project contains no workspaces", func(t *testing.T) {
		// create a project in the orgTest
		p, pTestCleanup := createProject(t, client, orgTest)
		defer pTestCleanup()

		// List all the workspaces under the given ProjectID
		wl, err := client.Workspaces.List(ctx, orgTest.Name, &WorkspaceListOptions{
			ProjectID: p.ID,
		})
		require.NoError(t, err)
		assert.Empty(t, wl.Items)
	})

	t.Run("when filter workspaces by current run status", func(t *testing.T) {
		wTest, wTestCleanup := createWorkspace(t, client, orgTest)
		t.Cleanup(wTestCleanup)

		rn, appliedCleanup := createRunApply(t, client, wTest)
		t.Cleanup(appliedCleanup)

		wl, err := client.Workspaces.List(ctx, orgTest.Name, &WorkspaceListOptions{
			CurrentRunStatus: string(RunApplied),
		})

		require.NoError(t, err)
		require.NotEmpty(t, wl.Items)
		require.GreaterOrEqual(t, len(wl.Items), 1)

		found := false
		for _, ws := range wl.Items {
			if ws.ID != wTest.ID {
				continue
			}
			assert.Equal(t, ws.CurrentRun.ID, rn.ID)
			found = true
		}

		assert.True(t, found)
	})
}

func TestWorkspacesCreateTableDriven(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	t.Cleanup(orgTestCleanup)

	oc, oaCleanup := createOAuthToken(t, client, orgTest)
	t.Cleanup(oaCleanup)

	workspaceTableTests := []WorkspaceTableTest{
		{
			scenario: "when options include vcs-repo",
			options: &WorkspaceTableOptions{
				createOptions: &WorkspaceCreateOptions{
					Name: String("foobar"),
					VCSRepo: &VCSRepoOptions{
						Identifier:   String("hashicorp/terraform-random-module"),
						OAuthTokenID: &oc.ID,
						Branch:       String("main"),
					},
				},
			},
			assertion: func(t *testing.T, w *Workspace, options *WorkspaceTableOptions, err error) {
				require.NoError(t, err)
				require.NotNil(t, w)
				require.NotEmpty(t, w.VCSRepo.Identifier)
				require.NotEmpty(t, w.VCSRepo.OAuthTokenID)
				require.NotEmpty(t, w.VCSRepo.Branch)

				wRead, err := client.Workspaces.ReadByID(ctx, w.ID)
				require.NoError(t, err)
				require.Equal(t, w.VCSRepo.Identifier, wRead.VCSRepo.Identifier)
				require.Equal(t, w.VCSRepo.OAuthTokenID, wRead.VCSRepo.OAuthTokenID)
				require.Equal(t, w.VCSRepo.Branch, wRead.VCSRepo.Branch)
			},
		},
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
			setup: func(t *testing.T, options *WorkspaceTableOptions) (w *Workspace, cleanup func()) {
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
			assertion: func(t *testing.T, w *Workspace, options *WorkspaceTableOptions, err error) {
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
			assertion: func(t *testing.T, w *Workspace, options *WorkspaceTableOptions, err error) {
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
			assertion: func(t *testing.T, w *Workspace, options *WorkspaceTableOptions, err error) {
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
			assertion: func(t *testing.T, w *Workspace, options *WorkspaceTableOptions, err error) {
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
			setup: func(t *testing.T, options *WorkspaceTableOptions) (w *Workspace, cleanup func()) {
				w, wTestCleanup := createWorkspaceWithVCS(t, client, orgTest, *options.createOptions)

				return w, func() {
					t.Cleanup(wTestCleanup)
				}
			},
			assertion: func(t *testing.T, w *Workspace, options *WorkspaceTableOptions, err error) {
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
				workspace, cleanup = tableTest.setup(t, tableTest.options)
				defer cleanup()
			} else {
				workspace, err = client.Workspaces.Create(ctx, orgTest.Name, *tableTest.options.createOptions)
			}
			tableTest.assertion(t, workspace, tableTest.options, err)
		})
	}
}

func TestWorkspacesCreateTableDrivenWithGithubApp(t *testing.T) {
	gHAInstallationID := os.Getenv("GITHUB_APP_INSTALLATION_ID")

	if gHAInstallationID == "" {
		t.Skip("Export a valid GITHUB_APP_INSTALLATION_ID before running this test!")
	}
	client := testClient(t)
	ctx := context.Background()

	orgTest1, orgTestCleanup := createOrganization(t, client)
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
			setup: func(t *testing.T, options *WorkspaceTableOptions) (w *Workspace, cleanup func()) {
				// Remove the below organization creation and use the one from the outer scope once the feature flag is removed
				orgTest, orgTestCleanup := createOrganizationWithOptions(t, client, OrganizationCreateOptions{
					Name:  String("tst-" + randomString(t)[0:20]),
					Email: String(fmt.Sprintf("%s@tfe.local", randomString(t))),
				})

				w, wTestCleanup := createWorkspaceWithGithubApp(t, client, orgTest, *options.createOptions)

				return w, func() {
					t.Cleanup(orgTestCleanup)
					t.Cleanup(wTestCleanup)
				}
			},
			assertion: func(t *testing.T, w *Workspace, options *WorkspaceTableOptions, err error) {
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
	}
	for _, tableTest := range workspaceTableTests {
		t.Run(tableTest.scenario, func(t *testing.T) {
			var workspace *Workspace
			var cleanup func()
			var err error
			if tableTest.setup != nil {
				workspace, cleanup = tableTest.setup(t, tableTest.options)
				defer cleanup()
			} else {
				workspace, err = client.Workspaces.Create(ctx, orgTest1.Name, *tableTest.options.createOptions)
			}
			tableTest.assertion(t, workspace, tableTest.options, err)
		})
	}
}

func TestWorkspacesCreate(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	t.Cleanup(orgTestCleanup)

	t.Run("with valid project option", func(t *testing.T) {
		skipUnlessBeta(t)

		options := WorkspaceCreateOptions{
			Name:                       String(fmt.Sprintf("foo-%s", randomString(t))),
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
			Project:                    orgTest.DefaultProject,
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
			assert.Equal(t, options.Project.ID, item.Project.ID)
		}
	})

	t.Run("with valid auto-apply-run-trigger option", func(t *testing.T) {
		skipIfEnterprise(t)
		// FEATURE FLAG: auto-apply-run-trigger
		// Once un-flagged, delete this test and add an AutoApplyRunTrigger field
		// to the basic "with valid options" test below.

		options := WorkspaceCreateOptions{
			Name:                String(fmt.Sprintf("foo-%s", randomString(t))),
			AutoApplyRunTrigger: Bool(true),
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
			assert.Equal(t, *options.AutoApplyRunTrigger, item.AutoApplyRunTrigger)
		}
	})

	t.Run("with valid options", func(t *testing.T) {
		options := WorkspaceCreateOptions{
			Name:                       String(fmt.Sprintf("foo-%s", randomString(t))),
			AllowDestroyPlan:           Bool(true),
			AutoApply:                  Bool(true),
			AutoDestroyAt:              NullableTime(time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)),
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
			assert.Equal(t, options.AutoDestroyAt, item.AutoDestroyAt)
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
			Name: String(fmt.Sprintf("foo-%s", randomString(t))),
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

	t.Run("when no execution mode is specified, in an organization with local as default execution mode", func(t *testing.T) {
		// Remove the below organization creation and use the one from the outer scope once the feature flag is removed
		orgTest, orgTestCleanup := createOrganizationWithOptions(t, client, OrganizationCreateOptions{
			Name:                 String("tst-" + randomString(t)[0:20] + "-ff-on"),
			Email:                String(fmt.Sprintf("%s@tfe.local", randomString(t))),
			DefaultExecutionMode: String("local"),
		})
		t.Cleanup(orgTestCleanup)

		options := WorkspaceCreateOptions{
			Name: String(fmt.Sprintf("foo-%s", randomString(t))),
			SettingOverwrites: &WorkspaceSettingOverwritesOptions{
				ExecutionMode: Bool(false),
			},
		}

		_, err := client.Workspaces.Create(ctx, orgTest.Name, options)
		require.NoError(t, err)

		// Get a refreshed view from the API.
		refreshed, err := client.Workspaces.Read(ctx, orgTest.Name, *options.Name)
		require.NoError(t, err)

		assert.Equal(t, "local", refreshed.ExecutionMode)
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

	t.Run("when organization has a default execution mode", func(t *testing.T) {
		defaultExecutionOrgTest, defaultExecutionOrgTestCleanup := createOrganizationWithDefaultAgentPool(t, client)
		t.Cleanup(defaultExecutionOrgTestCleanup)

		t.Run("with setting overwrites set to false, workspace inherits the default execution mode", func(t *testing.T) {
			options := WorkspaceCreateOptions{
				Name: String(fmt.Sprintf("tst-agent-cody-banks-%s", randomString(t))),
				SettingOverwrites: &WorkspaceSettingOverwritesOptions{
					ExecutionMode: Bool(false),
					AgentPool:     Bool(false),
				},
			}
			w, err := client.Workspaces.Create(ctx, defaultExecutionOrgTest.Name, options)

			require.NoError(t, err)
			assert.Equal(t, "agent", w.ExecutionMode)
		})

		t.Run("with setting overwrites set to true, workspace ignores the default execution mode", func(t *testing.T) {
			options := WorkspaceCreateOptions{
				Name:          String(fmt.Sprintf("tst-agent-tony-tanks-%s", randomString(t))),
				ExecutionMode: String("local"),
				SettingOverwrites: &WorkspaceSettingOverwritesOptions{
					ExecutionMode: Bool(true),
					AgentPool:     Bool(true),
				},
			}
			w, err := client.Workspaces.Create(ctx, defaultExecutionOrgTest.Name, options)

			require.NoError(t, err)
			assert.Equal(t, "local", w.ExecutionMode)
		})

		t.Run("when explicitly setting execution mode, workspace ignores the default execution mode", func(t *testing.T) {
			options := WorkspaceCreateOptions{
				Name:          String(fmt.Sprintf("tst-remotely-interesting-workspace-%s", randomString(t))),
				ExecutionMode: String("remote"),
			}
			w, err := client.Workspaces.Create(ctx, defaultExecutionOrgTest.Name, options)

			require.NoError(t, err)
			assert.Equal(t, "remote", w.ExecutionMode)
		})
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
		assert.NotEmpty(t, wTest.SettingOverwrites)
	})

	t.Run("links are properly decoded", func(t *testing.T) {
		w, err := client.Workspaces.Read(ctx, orgTest.Name, wTest.Name)
		require.NoError(t, err)

		assert.NotEmpty(t, w.Links["self-html"])
		assert.Contains(t, w.Links["self-html"], fmt.Sprintf("/app/%s/workspaces/%s", orgTest.Name, wTest.Name))

		assert.NotEmpty(t, w.Links["self"])
		assert.Contains(t, w.Links["self"], fmt.Sprintf("/api/v2/organizations/%s/workspaces/%s", orgTest.Name, wTest.Name))
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

	t.Run("when workspace is inheriting the default execution mode", func(t *testing.T) {
		defaultExecutionOrgTest, defaultExecutionOrgTestCleanup := createOrganizationWithDefaultAgentPool(t, client)
		t.Cleanup(defaultExecutionOrgTestCleanup)

		options := WorkspaceCreateOptions{
			Name: String(fmt.Sprintf("tst-agent-cody-banks-%s", randomString(t))),
			SettingOverwrites: &WorkspaceSettingOverwritesOptions{
				ExecutionMode: Bool(false),
				AgentPool:     Bool(false),
			},
		}

		wDefaultTest, wDefaultTestCleanup := createWorkspaceWithOptions(t, client, defaultExecutionOrgTest, options)
		t.Cleanup(wDefaultTestCleanup)

		t.Run("and workspace execution mode is default", func(t *testing.T) {
			w, err := client.Workspaces.Read(ctx, defaultExecutionOrgTest.Name, wDefaultTest.Name)
			assert.NoError(t, err)
			assert.NotEmpty(t, w)

			assert.Equal(t, defaultExecutionOrgTest.DefaultExecutionMode, w.ExecutionMode)
			assert.NotEmpty(t, w.SettingOverwrites)
			assert.Equal(t, false, *w.SettingOverwrites.ExecutionMode)
			assert.Equal(t, false, *w.SettingOverwrites.ExecutionMode)
		})
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

	// give HCP Terraform some time to process the statefile and extract the outputs.
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

// If you've set your own GITHUB_POLICY_SET_IDENTIFIER, make sure the readme
// starts with the string: This is a simple test
// Otherwise the test will not pass
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

func TestWorkspacesAddTagBindings(t *testing.T) {
	skipUnlessBeta(t)

	client := testClient(t)
	ctx := context.Background()

	wTest, wCleanup := createWorkspace(t, client, nil)
	t.Cleanup(wCleanup)

	t.Run("when adding tag bindings to a workspace", func(t *testing.T) {
		tagBindings := []*TagBinding{
			{Key: "foo", Value: "bar"},
			{Key: "baz", Value: "qux"},
		}

		bindings, err := client.Workspaces.AddTagBindings(ctx, wTest.ID, WorkspaceAddTagBindingsOptions{
			TagBindings: tagBindings,
		})
		require.NoError(t, err)

		assert.Len(t, bindings, 2)
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

		_, err := client.Workspaces.AddTagBindings(ctx, wTest.ID, WorkspaceAddTagBindingsOptions{
			TagBindings: tagBindings,
		})
		require.Error(t, err, "cannot exceed 10 bindings per resource")
	})
}

func TestWorkspaces_DeleteAllTagBindings(t *testing.T) {
	skipUnlessBeta(t)

	client := testClient(t)
	ctx := context.Background()

	wTest, wCleanup := createWorkspace(t, client, nil)
	t.Cleanup(wCleanup)

	tagBindings := []*TagBinding{
		{Key: "foo", Value: "bar"},
		{Key: "baz", Value: "qux"},
	}

	_, err := client.Workspaces.AddTagBindings(ctx, wTest.ID, WorkspaceAddTagBindingsOptions{
		TagBindings: tagBindings,
	})
	require.NoError(t, err)

	err = client.Workspaces.DeleteAllTagBindings(ctx, wTest.ID)
	require.NoError(t, err)

	bindings, err := client.Workspaces.ListTagBindings(ctx, wTest.ID)
	require.NoError(t, err)
	require.Empty(t, bindings)
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

	t.Run("when updating auto-apply-run-trigger", func(t *testing.T) {
		skipIfEnterprise(t)
		// Feature flag: auto-apply-run-trigger. Once flag is removed, delete
		// this test and add the attribute to one generic update test.
		options := WorkspaceUpdateOptions{
			AutoApplyRunTrigger: Bool(true),
		}

		wAfter, err := client.Workspaces.Update(ctx, orgTest.Name, wTest.Name, options)
		require.NoError(t, err)

		assert.Equal(t, wTest.Name, wAfter.Name)
		assert.NotEqual(t, wTest.AutoApplyRunTrigger, wAfter.AutoApplyRunTrigger)
	})

	t.Run("when updating project", func(t *testing.T) {
		skipUnlessBeta(t)

		kBefore, kTestCleanup := createProject(t, client, orgTest)
		defer kTestCleanup()

		wBefore, wBeforeCleanup := createWorkspaceWithOptions(t, client, orgTest, WorkspaceCreateOptions{
			Name:    String(randomString(t)),
			Project: kBefore,
		})
		defer wBeforeCleanup()

		options := WorkspaceUpdateOptions{
			Name:               String(wBefore.Name),
			AllowDestroyPlan:   Bool(false),
			AutoApply:          Bool(true),
			Operations:         Bool(true),
			QueueAllRuns:       Bool(true),
			AssessmentsEnabled: Bool(true),
			TerraformVersion:   String("0.15.4"),
			Project:            orgTest.DefaultProject,
		}

		wAfter, err := client.Workspaces.Update(ctx, orgTest.Name, wBefore.Name, options)
		require.NoError(t, err)

		require.NotNil(t, wAfter.Project)
		require.NotNil(t, orgTest.DefaultProject)

		assert.Equal(t, wBefore.Name, wAfter.Name)
		assert.Equal(t, wAfter.Project.ID, orgTest.DefaultProject.ID)
	})

	t.Run("with valid options", func(t *testing.T) {
		options := WorkspaceUpdateOptions{
			Name:                       String(randomString(t)),
			AllowDestroyPlan:           Bool(true),
			AutoApply:                  Bool(false),
			AutoDestroyAt:              NullableTime(time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)),
			FileTriggersEnabled:        Bool(true),
			Operations:                 Bool(false),
			QueueAllRuns:               Bool(false),
			SpeculativeEnabled:         Bool(true),
			Description:                String("updated description"),
			StructuredRunOutputEnabled: Bool(true),
			TerraformVersion:           String("0.11.1"),
			TriggerPrefixes:            []string{"/modules", "/shared"},
			WorkingDirectory:           String("baz/"),
			TagBindings: []*TagBinding{
				{Key: "foo", Value: "bar"},
			},
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
			assert.Equal(t, options.AutoDestroyAt, item.AutoDestroyAt)
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

		if betaFeaturesEnabled() {
			bindings, err := client.Workspaces.ListTagBindings(ctx, wTest.ID)
			require.NoError(t, err)

			assert.Len(t, bindings, 1)
			assert.Equal(t, "foo", bindings[0].Key)
			assert.Equal(t, "bar", bindings[0].Value)

			effectiveBindings, err := client.Workspaces.ListEffectiveTagBindings(ctx, wTest.ID)
			require.NoError(t, err)

			assert.Len(t, effectiveBindings, 1)
			assert.Equal(t, "foo", effectiveBindings[0].Key)
			assert.Equal(t, "bar", effectiveBindings[0].Value)
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
			setup: func(t *testing.T, options *WorkspaceTableOptions) (w *Workspace, cleanup func()) {
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
			assertion: func(t *testing.T, workspace *Workspace, options *WorkspaceTableOptions, _ error) {
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
			assertion: func(t *testing.T, w *Workspace, options *WorkspaceTableOptions, err error) {
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
			assertion: func(t *testing.T, w *Workspace, options *WorkspaceTableOptions, err error) {
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
			assertion: func(t *testing.T, w *Workspace, options *WorkspaceTableOptions, err error) {
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
			assertion: func(t *testing.T, w *Workspace, options *WorkspaceTableOptions, err error) {
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
				workspace, cleanup = tableTest.setup(t, tableTest.options)
				defer cleanup()
			} else {
				workspace, err = client.Workspaces.Update(ctx, orgTest.Name, wTest.Name, *tableTest.options.updateOptions)
			}
			tableTest.assertion(t, workspace, tableTest.options, err)
		})
	}
}

func TestWorkspacesUpdateTableDrivenWithGithubApp(t *testing.T) {
	gHAInstallationID := os.Getenv("GITHUB_APP_INSTALLATION_ID")

	if gHAInstallationID == "" {
		t.Skip("Export a valid GITHUB_APP_INSTALLATION_ID before running this test!")
	}
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
					VCSRepo: &VCSRepoOptions{
						TagsRegex: String("foobar"),
					},
				},
			},
			setup: func(t *testing.T, options *WorkspaceTableOptions) (w *Workspace, cleanup func()) {
				orgTest, orgTestCleanup := createOrganizationWithOptions(t, client, OrganizationCreateOptions{
					Name:  String("tst-" + randomString(t)[0:20]),
					Email: String(fmt.Sprintf("%s@tfe.local", randomString(t))),
				})

				wTest, wTestCleanup := createWorkspaceWithGithubApp(t, client, orgTest, *options.createOptions)
				return wTest, func() {
					t.Cleanup(orgTestCleanup)
					t.Cleanup(wTestCleanup)
				}
			},
			assertion: func(t *testing.T, workspace *Workspace, options *WorkspaceTableOptions, _ error) {
				assert.Equal(t, *options.createOptions.VCSRepo.TagsRegex, workspace.VCSRepo.TagsRegex)
				assert.Equal(t, workspace.VCSRepo.TagsRegex, *String("barfoo")) // Sanity test

				w, err := client.Workspaces.Update(ctx, workspace.Organization.Name, workspace.Name, *options.updateOptions)
				require.NoError(t, err)
				assert.Equal(t, w.VCSRepo.TagsRegex, *String("foobar")) // Sanity test
			},
		},
	}

	for _, tableTest := range workspaceTableTests {
		t.Run(tableTest.scenario, func(t *testing.T) {
			var workspace *Workspace
			var cleanup func()
			var err error
			if tableTest.setup != nil {
				workspace, cleanup = tableTest.setup(t, tableTest.options)
				defer cleanup()
			} else {
				workspace, err = client.Workspaces.Update(ctx, orgTest.Name, wTest.Name, *tableTest.options.updateOptions)
			}
			tableTest.assertion(t, workspace, tableTest.options, err)
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

func TestWorkspacesUpdateWithDefaultExecutionMode(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	defaultExecutionOrgTest, defaultExecutionOrgTestCleanup := createOrganizationWithDefaultAgentPool(t, client)
	t.Cleanup(defaultExecutionOrgTestCleanup)

	wTest, wCleanup := createWorkspace(t, client, defaultExecutionOrgTest)
	t.Cleanup(wCleanup)

	t.Run("when explicitly setting execution mode, workspace ignores the default execution mode", func(t *testing.T) {
		options := WorkspaceUpdateOptions{
			ExecutionMode: String("remote"),
		}
		w, err := client.Workspaces.Update(ctx, defaultExecutionOrgTest.Name, wTest.Name, options)

		require.NoError(t, err)
		assert.Equal(t, "remote", w.ExecutionMode)
	})

	t.Run("with setting overwrites set to true, workspace ignores the default execution mode", func(t *testing.T) {
		options := WorkspaceUpdateOptions{
			ExecutionMode: String("local"),
			SettingOverwrites: &WorkspaceSettingOverwritesOptions{
				ExecutionMode: Bool(true),
				AgentPool:     Bool(true),
			},
		}
		w, err := client.Workspaces.Update(ctx, defaultExecutionOrgTest.Name, wTest.Name, options)

		require.NoError(t, err)
		assert.Equal(t, "local", w.ExecutionMode)
	})

	t.Run("with setting overwrites set to false, workspace inherits the default execution mode", func(t *testing.T) {
		options := WorkspaceUpdateOptions{
			SettingOverwrites: &WorkspaceSettingOverwritesOptions{
				ExecutionMode: Bool(false),
				AgentPool:     Bool(false),
			},
		}
		w, err := client.Workspaces.Update(ctx, defaultExecutionOrgTest.Name, wTest.Name, options)

		require.NoError(t, err)
		assert.Equal(t, "agent", w.ExecutionMode)
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
		assert.True(t, errors.Is(err, ErrWorkspaceLockedCannotDelete))
	})

	t.Run("when workspace has resources under management", func(t *testing.T) {
		wTest, workspaceCleanup := createWorkspace(t, client, orgTest)
		t.Cleanup(workspaceCleanup)
		_, svTestCleanup := createStateVersion(t, client, 0, wTest)
		t.Cleanup(svTestCleanup)

		_, err := retry(func() (interface{}, error) {
			err := client.Workspaces.SafeDelete(ctx, orgTest.Name, wTest.Name)
			if errors.Is(err, ErrWorkspaceStillProcessing) {
				return nil, err
			}

			return nil, nil
		})

		if err != nil {
			t.Fatalf("Workspace still processing after retrying: %s", err)
		}

		err = client.Workspaces.SafeDelete(ctx, orgTest.Name, wTest.Name)
		assert.True(t, errors.Is(err, ErrWorkspaceNotSafeToDelete))
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
		assert.True(t, errors.Is(err, ErrWorkspaceLockedCannotDelete))
	})

	t.Run("when workspace has resources under management", func(t *testing.T) {
		wTest, workspaceCleanup := createWorkspace(t, client, orgTest)
		t.Cleanup(workspaceCleanup)
		_, svTestCleanup := createStateVersion(t, client, 0, wTest)
		t.Cleanup(svTestCleanup)

		_, err := retry(func() (interface{}, error) {
			err := client.Workspaces.SafeDeleteByID(ctx, wTest.ID)
			if errors.Is(err, ErrWorkspaceStillProcessing) {
				return nil, err
			}

			return nil, nil
		})

		if err != nil {
			t.Fatalf("Workspace still processing after retrying: %s", err)
		}

		err = client.Workspaces.SafeDeleteByID(ctx, wTest.ID)
		assert.True(t, errors.Is(err, ErrWorkspaceNotSafeToDelete))
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
		require.Empty(t, wTest.LockedBy)

		w, err := client.Workspaces.Lock(ctx, wTest.ID, WorkspaceLockOptions{})
		require.NoError(t, err)
		assert.True(t, w.Locked)

		require.NoError(t, err)
		require.NotEmpty(t, w.LockedBy)
		requireExactlyOneNotEmpty(t, w.LockedBy.Run, w.LockedBy.Team, w.LockedBy.User)
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

	t.Run("when a workspace is locked by a team", func(t *testing.T) {
		wTest2, wTest2Cleanup := createWorkspace(t, client, orgTest)
		t.Cleanup(wTest2Cleanup)

		// Create a new team to lock the workspace
		tmTest, tmTestCleanup := createTeam(t, client, orgTest)
		defer tmTestCleanup()
		ta, err := client.TeamAccess.Add(ctx, TeamAccessAddOptions{
			Access:    Access(AccessAdmin),
			Team:      tmTest,
			Workspace: wTest2,
		})
		assert.Nil(t, err)
		defer func() {
			err := client.TeamAccess.Remove(ctx, ta.ID)
			if err != nil {
				t.Logf("error removing team access (%s): %s", ta.ID, err)
			}
		}()
		tt, ttTestCleanup := createTeamToken(t, client, tmTest)
		defer ttTestCleanup()

		// Create a new client with the team token
		teamClient := testClient(t)
		teamClient.token = tt.Token

		// Lock the workspace with the team client
		_, err = teamClient.Workspaces.Lock(ctx, wTest2.ID, WorkspaceLockOptions{})
		assert.Nil(t, err)

		// Attempt to unlock the workspace with the original client
		_, err = client.Workspaces.Unlock(ctx, wTest2.ID)
		assert.Equal(t, ErrWorkspaceLockedByTeam, err)
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
	parsedTime, err := time.Parse(iso8601TimeFormat, "2020-07-15T23:38:43.821Z")
	assert.NoError(t, err)

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
	skipUnlessBeta(t)

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

func TestWorkspacesProjects(t *testing.T) {
	skipUnlessBeta(t)
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	defer orgTestCleanup()

	wTest, wTestCleanup := createWorkspace(t, client, orgTest)
	defer wTestCleanup()

	t.Run("created workspace includes default organization project", func(t *testing.T) {
		require.NotNil(t, orgTest.DefaultProject)
		require.NotNil(t, wTest.Project)
		assert.Equal(t, wTest.Project.ID, orgTest.DefaultProject.ID)
	})

	t.Run("created workspace includes project ID", func(t *testing.T) {
		assert.NotNil(t, wTest.Project.ID)
	})

	t.Run("read workspace includes project ID", func(t *testing.T) {
		workspace, err := client.Workspaces.ReadByID(ctx, wTest.ID)
		assert.NoError(t, err)
		assert.NotNil(t, workspace.Project.ID)
	})

	t.Run("list workspace includes project ID", func(t *testing.T) {
		workspaces, err := client.Workspaces.List(ctx, orgTest.Name, &WorkspaceListOptions{})
		assert.NoError(t, err)
		for idx, item := range workspaces.Items {
			assert.NotNil(t, item.Project.ID, "No project ID set on workspace %s at idx %d", item.ID, idx)
		}
	})
}

func TestWorkspace_DataRetentionPolicy(t *testing.T) {
	skipUnlessEnterprise(t)

	client := testClient(t)
	ctx := context.Background()

	wTest, wTestCleanup := createWorkspace(t, client, nil)
	defer wTestCleanup()

	dataRetentionPolicy, err := client.Workspaces.ReadDataRetentionPolicyChoice(ctx, wTest.ID)
	assert.Nil(t, err)
	require.Nil(t, dataRetentionPolicy)

	workspace, err := client.Workspaces.ReadByID(ctx, wTest.ID)
	require.NoError(t, err)
	require.Nil(t, workspace.DataRetentionPolicy)
	require.Nil(t, workspace.DataRetentionPolicyChoice)

	t.Run("set and update data retention policy to delete older", func(t *testing.T) {
		createdDataRetentionPolicy, err := client.Workspaces.SetDataRetentionPolicyDeleteOlder(ctx, wTest.ID, DataRetentionPolicyDeleteOlderSetOptions{DeleteOlderThanNDays: 33})
		require.NoError(t, err)
		require.Equal(t, 33, createdDataRetentionPolicy.DeleteOlderThanNDays)
		require.Contains(t, createdDataRetentionPolicy.ID, "drp-")

		dataRetentionPolicy, err = client.Workspaces.ReadDataRetentionPolicyChoice(ctx, wTest.ID)
		require.NoError(t, err)
		require.NotNil(t, dataRetentionPolicy.DataRetentionPolicyDeleteOlder)

		require.Equal(t, 33, dataRetentionPolicy.DataRetentionPolicyDeleteOlder.DeleteOlderThanNDays)
		require.Equal(t, createdDataRetentionPolicy.ID, dataRetentionPolicy.DataRetentionPolicyDeleteOlder.ID)
		require.Contains(t, dataRetentionPolicy.DataRetentionPolicyDeleteOlder.ID, "drp-")

		workspace, err := client.Workspaces.ReadByID(ctx, wTest.ID)
		require.NoError(t, err)
		require.Equal(t, dataRetentionPolicy.DataRetentionPolicyDeleteOlder.ID, workspace.DataRetentionPolicyChoice.DataRetentionPolicyDeleteOlder.ID)

		// deprecated DataRetentionPolicy field should also have been populated
		require.NotNil(t, workspace.DataRetentionPolicy)
		require.Equal(t, workspace.DataRetentionPolicy.ID, dataRetentionPolicy.DataRetentionPolicyDeleteOlder.ID)

		// try updating the number of days
		createdDataRetentionPolicy, err = client.Workspaces.SetDataRetentionPolicyDeleteOlder(ctx, wTest.ID, DataRetentionPolicyDeleteOlderSetOptions{DeleteOlderThanNDays: 1})
		require.NoError(t, err)
		require.Equal(t, 1, createdDataRetentionPolicy.DeleteOlderThanNDays)

		dataRetentionPolicy, err = client.Workspaces.ReadDataRetentionPolicyChoice(ctx, wTest.ID)
		require.NoError(t, err)
		require.NotNil(t, dataRetentionPolicy.DataRetentionPolicyDeleteOlder)
		require.Equal(t, 1, dataRetentionPolicy.DataRetentionPolicyDeleteOlder.DeleteOlderThanNDays)
		require.Equal(t, createdDataRetentionPolicy.ID, dataRetentionPolicy.DataRetentionPolicyDeleteOlder.ID)
	})

	t.Run("set data retention policy to not delete", func(t *testing.T) {
		createdDataRetentionPolicy, err := client.Workspaces.SetDataRetentionPolicyDontDelete(ctx, wTest.ID, DataRetentionPolicyDontDeleteSetOptions{})
		require.NoError(t, err)
		require.Contains(t, createdDataRetentionPolicy.ID, "drp-")

		dataRetentionPolicy, err = client.Workspaces.ReadDataRetentionPolicyChoice(ctx, wTest.ID)
		require.NoError(t, err)
		require.NotNil(t, dataRetentionPolicy.DataRetentionPolicyDontDelete)
		require.Equal(t, createdDataRetentionPolicy.ID, dataRetentionPolicy.DataRetentionPolicyDontDelete.ID)

		// dont delete policies should leave the legacy DataRetentionPolicy field on workspaces empty
		workspace, err := client.Workspaces.ReadByID(ctx, wTest.ID)
		require.NoError(t, err)
		require.Nil(t, workspace.DataRetentionPolicy)
	})

	t.Run("change data retention policy type", func(t *testing.T) {
		_, err = client.Workspaces.SetDataRetentionPolicyDeleteOlder(ctx, wTest.ID, DataRetentionPolicyDeleteOlderSetOptions{DeleteOlderThanNDays: 45})
		require.NoError(t, err)

		dataRetentionPolicy, err = client.Workspaces.ReadDataRetentionPolicyChoice(ctx, wTest.ID)
		require.NoError(t, err)
		require.NotNil(t, dataRetentionPolicy.DataRetentionPolicyDeleteOlder)
		require.Equal(t, 45, dataRetentionPolicy.DataRetentionPolicyDeleteOlder.DeleteOlderThanNDays)
		require.Nil(t, dataRetentionPolicy.DataRetentionPolicyDontDelete)

		_, err = client.Workspaces.SetDataRetentionPolicyDontDelete(ctx, wTest.ID, DataRetentionPolicyDontDeleteSetOptions{})
		require.NoError(t, err)
		dataRetentionPolicy, err = client.Workspaces.ReadDataRetentionPolicyChoice(ctx, wTest.ID)
		require.NoError(t, err)
		require.Nil(t, dataRetentionPolicy.DataRetentionPolicyDeleteOlder)
		require.NotNil(t, dataRetentionPolicy.DataRetentionPolicyDontDelete)

		_, err = client.Workspaces.SetDataRetentionPolicyDeleteOlder(ctx, wTest.ID, DataRetentionPolicyDeleteOlderSetOptions{DeleteOlderThanNDays: 20})
		require.NoError(t, err)

		dataRetentionPolicy, err = client.Workspaces.ReadDataRetentionPolicyChoice(ctx, wTest.ID)
		require.NoError(t, err)
		require.NotNil(t, dataRetentionPolicy.DataRetentionPolicyDeleteOlder)
		require.Equal(t, 20, dataRetentionPolicy.DataRetentionPolicyDeleteOlder.DeleteOlderThanNDays)
		require.Nil(t, dataRetentionPolicy.DataRetentionPolicyDontDelete)
	})

	t.Run("delete data retention policy", func(t *testing.T) {
		err = client.Workspaces.DeleteDataRetentionPolicy(ctx, wTest.ID)
		require.NoError(t, err)

		dataRetentionPolicy, err = client.Workspaces.ReadDataRetentionPolicyChoice(ctx, wTest.ID)
		assert.Nil(t, err)
		require.Nil(t, dataRetentionPolicy)
	})
}

func TestWorkspacesAutoDestroy(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	t.Cleanup(orgTestCleanup)

	newSubscriptionUpdater(orgTest).WithBusinessPlan().Update(t)

	autoDestroyAt := NullableTime(time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC))
	wTest, wCleanup := createWorkspaceWithOptions(t, client, orgTest, WorkspaceCreateOptions{
		Name:          String(randomString(t)),
		AutoDestroyAt: autoDestroyAt,
	})
	t.Cleanup(wCleanup)

	require.Equal(t, autoDestroyAt, wTest.AutoDestroyAt)

	// respect default omitempty
	w, err := client.Workspaces.Update(ctx, orgTest.Name, wTest.Name, WorkspaceUpdateOptions{
		AutoDestroyAt: nil,
	})

	require.NoError(t, err)
	require.NotNil(t, w.AutoDestroyAt)

	// explicitly update the value of auto_destroy_at
	w, err = client.Workspaces.Update(ctx, orgTest.Name, wTest.Name, WorkspaceUpdateOptions{
		AutoDestroyAt: NullableTime(time.Date(2025, 1, 2, 0, 0, 0, 0, time.UTC)),
	})

	require.NoError(t, err)
	require.NotNil(t, w.AutoDestroyAt)
	require.NotEqual(t, autoDestroyAt, w.AutoDestroyAt)

	// disable auto destroy
	w, err = client.Workspaces.Update(ctx, orgTest.Name, wTest.Name, WorkspaceUpdateOptions{
		AutoDestroyAt: NullTime(),
	})

	require.NoError(t, err)
	require.Nil(t, w.AutoDestroyAt)
}

func TestWorkspacesAutoDestroyDuration(t *testing.T) {
	skipUnlessBeta(t)

	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	t.Cleanup(orgTestCleanup)

	newSubscriptionUpdater(orgTest).WithBusinessPlan().Update(t)

	t.Run("when creating a new workspace with standalone auto destroy settings", func(t *testing.T) {
		duration := jsonapi.NewNullableAttrWithValue("14d")
		nilDuration := jsonapi.NewNullNullableAttr[string]()
		nilAutoDestroy := jsonapi.NewNullNullableAttr[time.Time]()
		wTest, wCleanup := createWorkspaceWithOptions(t, client, orgTest, WorkspaceCreateOptions{
			Name:                        String(randomString(t)),
			AutoDestroyActivityDuration: duration,
			InheritsProjectAutoDestroy:  Bool(false),
		})
		t.Cleanup(wCleanup)

		require.Equal(t, duration, wTest.AutoDestroyActivityDuration)
		require.NotEqual(t, nilAutoDestroy, wTest.AutoDestroyAt)
		require.Equal(t, wTest.InheritsProjectAutoDestroy, false)

		w, err := client.Workspaces.Update(ctx, orgTest.Name, wTest.Name, WorkspaceUpdateOptions{
			AutoDestroyActivityDuration: nilDuration,
			InheritsProjectAutoDestroy:  Bool(false),
		})

		require.NoError(t, err)
		require.False(t, w.AutoDestroyActivityDuration.IsSpecified())
		require.False(t, w.AutoDestroyAt.IsSpecified())
		require.Equal(t, wTest.InheritsProjectAutoDestroy, false)
	})
}

func TestWorkspaces_effectiveTagBindingsInheritedFrom(t *testing.T) {
	skipUnlessBeta(t)

	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	t.Cleanup(orgTestCleanup)

	projTest, projTestCleanup := createProject(t, client, orgTest)
	t.Cleanup(projTestCleanup)

	ws, wsCleanup := createWorkspaceWithOptions(t, client, orgTest, WorkspaceCreateOptions{
		Name:    String("mycoolworkspace"),
		Project: projTest,
	})
	t.Cleanup(wsCleanup)

	_, err := client.Workspaces.AddTagBindings(ctx, ws.ID, WorkspaceAddTagBindingsOptions{
		TagBindings: []*TagBinding{
			{
				Key:   "a",
				Value: "1",
			},
			{
				Key:   "b",
				Value: "2",
			},
		},
	})
	require.NoError(t, err)

	t.Run("when no tags are inherited from the project", func(t *testing.T) {
		effectiveBindings, err := client.Workspaces.ListEffectiveTagBindings(ctx, ws.ID)
		require.NoError(t, err)

		for _, binding := range effectiveBindings {
			require.Nil(t, binding.Links)
		}
	})

	t.Run("when tags are inherited from the project", func(t *testing.T) {
		_, err := client.Projects.AddTagBindings(ctx, projTest.ID, ProjectAddTagBindingsOptions{
			TagBindings: []*TagBinding{
				{
					Key:   "inherited",
					Value: "foo",
				},
			},
		})
		require.NoError(t, err)

		effectiveBindings, err := client.Workspaces.ListEffectiveTagBindings(ctx, ws.ID)
		require.NoError(t, err)

		for _, binding := range effectiveBindings {
			if binding.Key == "inherited" {
				require.NotNil(t, binding.Links)
				require.NotNil(t, binding.Links["inherited-from"])
			} else {
				require.Nil(t, binding.Links)
			}
		}
	})
}
