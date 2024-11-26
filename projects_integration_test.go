// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package tfe

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/stretchr/testify/require"

	"github.com/hashicorp/jsonapi"
)

func TestProjectsList(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	defer orgTestCleanup()

	pTest1, pTestCleanup := createProject(t, client, orgTest)
	defer pTestCleanup()

	pTest2, pTestCleanup := createProject(t, client, orgTest)
	defer pTestCleanup()

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
}

func TestProjectsRead(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	defer orgTestCleanup()

	pTest, pTestCleanup := createProject(t, client, orgTest)
	defer pTestCleanup()

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
}

func TestProjectsCreate(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	defer orgTestCleanup()

	upgradeOrganizationSubscription(t, client, orgTest)

	t.Run("with valid options", func(t *testing.T) {
		options := ProjectCreateOptions{
			Name:                        "foo",
			Description:                 String("qux"),
			AutoDestroyActivityDuration: jsonapi.NewNullableAttrWithValue("3d"),
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
			assert.Equal(t, options.AutoDestroyActivityDuration, item.AutoDestroyActivityDuration)
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
		w, err := client.Projects.Create(ctx, orgTest.Name, ProjectCreateOptions{
			Name:                        "foo",
			AutoDestroyActivityDuration: jsonapi.NewNullableAttrWithValue("20m"),
		})
		assert.Nil(t, w)
		assert.Contains(t, err.Error(), "invalid attribute\n\nAuto destroy activity duration has an incorrect format, we expect up to 4 numeric digits and 1 unit ('d' or 'h')")
	})
}

func TestProjectsUpdate(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	defer orgTestCleanup()

	t.Run("with valid options", func(t *testing.T) {
		kBefore, kTestCleanup := createProject(t, client, orgTest)
		t.Cleanup(kTestCleanup)

		kAfter, err := client.Projects.Update(ctx, kBefore.ID, ProjectUpdateOptions{
			Name:        String("new project name"),
			Description: String("updated description"),
			TagBindings: []*TagBinding{
				{Key: "foo", Value: "bar"},
			},
			AutoDestroyActivityDuration: jsonapi.NewNullableAttrWithValue("3d"),
		})
		require.NoError(t, err)

		assert.Equal(t, kBefore.ID, kAfter.ID)
		assert.NotEqual(t, kBefore.Name, kAfter.Name)
		assert.NotEqual(t, kBefore.Description, kAfter.Description)
		assert.NotEqual(t, kBefore.AutoDestroyActivityDuration, kAfter.AutoDestroyActivityDuration)

		if betaFeaturesEnabled() {
			bindings, err := client.Projects.ListTagBindings(ctx, kAfter.ID)
			require.NoError(t, err)

			assert.Len(t, bindings, 1)
			assert.Equal(t, "foo", bindings[0].Key)
			assert.Equal(t, "bar", bindings[0].Value)

			effectiveBindings, err := client.Projects.ListEffectiveTagBindings(ctx, kAfter.ID)
			require.NoError(t, err)

			assert.Len(t, effectiveBindings, 1)
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
				if b.Key == "foo" {
					assert.Equal(t, "bar", b.Value)
				} else if b.Key == "baz" {
					assert.Equal(t, "qux", b.Value)
				} else {
					assert.Fail(t, "unexpected tag binding %q", b.Key)
				}
			}
		}
	})

	t.Run("when updating with invalid name", func(t *testing.T) {
		kBefore, kTestCleanup := createProject(t, client, orgTest)
		defer kTestCleanup()

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
		upgradeOrganizationSubscription(t, client, orgTest)

		kBefore, kTestCleanup := createProject(t, client, orgTest)
		defer kTestCleanup()

		w, err := client.Projects.Update(ctx, kBefore.ID, ProjectUpdateOptions{
			AutoDestroyActivityDuration: jsonapi.NewNullableAttrWithValue("bar"),
		})
		assert.Nil(t, w)
		assert.Contains(t, err.Error(), "invalid attribute\n\nAuto destroy activity duration has an incorrect format, we expect up to 4 numeric digits and 1 unit ('d' or 'h')")
	})
}

func TestProjectsAddTagBindings(t *testing.T) {
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

		_, err := client.Workspaces.AddTagBindings(ctx, pTest.ID, WorkspaceAddTagBindingsOptions{
			TagBindings: tagBindings,
		})
		require.Error(t, err, "cannot exceed 10 bindings per resource")
	})
}

func TestProjectsDelete(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	defer orgTestCleanup()

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
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	defer orgTestCleanup()

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
