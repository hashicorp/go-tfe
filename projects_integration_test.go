// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package tfe

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/stretchr/testify/require"
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

		p1, wTestCleanup1 := createProjectWithOptions(t, client, orgTest, ProjectCreateOptions{
			Name: randomStringWithoutSpecialChar(t),
			TagBindings: []*TagBinding{
				{Key: "key1", Value: "value1"},
				{Key: "key2", Value: "value2a"},
			},
		})
		p2, wTestCleanup2 := createProjectWithOptions(t, client, orgTest, ProjectCreateOptions{
			Name: randomStringWithoutSpecialChar(t),
			TagBindings: []*TagBinding{
				{Key: "key2", Value: "value2b"},
				{Key: "key3", Value: "value3"},
			},
		})
		t.Cleanup(wTestCleanup1)
		t.Cleanup(wTestCleanup2)

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
}

func TestProjectsUpdate(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	defer orgTestCleanup()

	t.Run("with valid options", func(t *testing.T) {
		kBefore, kTestCleanup := createProject(t, client, orgTest)
		defer kTestCleanup()

		kAfter, err := client.Projects.Update(ctx, kBefore.ID, ProjectUpdateOptions{
			Name:        String("new project name"),
			Description: String("updated description"),
			TagBindings: []*TagBinding{
				{Key: "foo", Value: "bar"},
			},
		})
		require.NoError(t, err)

		assert.Equal(t, kBefore.ID, kAfter.ID)
		assert.NotEqual(t, kBefore.Name, kAfter.Name)
		assert.NotEqual(t, kBefore.Description, kAfter.Description)

		if betaFeaturesEnabled() {
			bindings, err := client.Projects.ListTagBindings(ctx, kAfter.ID)
			require.NoError(t, err)

			assert.Len(t, bindings, 1)
			assert.Equal(t, "foo", bindings[0].Key)
			assert.Equal(t, "bar", bindings[0].Value)
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
