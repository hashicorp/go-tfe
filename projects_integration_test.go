//go:build integration
// +build integration

package tfe

import (
	"context"
	"github.com/stretchr/testify/assert"
	"testing"

	"github.com/stretchr/testify/require"
)

//func TestProjectsList(t *testing.T) {
//	skipIfNotCINode(t)
//
//	client := testClient(t)
//	ctx := context.Background()
//
//	// Create your test helper resources here
//	t.Run("test not yet implemented", func(t *testing.T) {
//		require.NotNil(t, nil)
//	})
//}

func TestProjectsRead(t *testing.T) {
	skipIfNotCINode(t)

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
	skipIfNotCINode(t)

	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	defer orgTestCleanup()

	// Create your test helper resources here
	t.Run("with valid options", func(t *testing.T) {
		options := ProjectCreateOptions{
			Name: String("foo"),
		}

		w, err := client.Projects.Create(ctx, orgTest.Name, options)
		require.NoError(t, err)

		// Get a refreshed view from the API.
		refreshed, err := client.Projects.Read(ctx, w.ID)
		require.NoError(t, err)

		for _, item := range []*Project{
			w,
			refreshed,
		} {
			assert.NotEmpty(t, item.ID)
			assert.Equal(t, *options.Name, item.Name)
		}
	})

	t.Run("when options is missing name", func(t *testing.T) {
		w, err := client.Projects.Create(ctx, "foo", ProjectCreateOptions{})
		assert.Nil(t, w)
		assert.EqualError(t, err, ErrRequiredName.Error())
	})

	t.Run("when options has an invalid name", func(t *testing.T) {
		w, err := client.Projects.Create(ctx, "foo", ProjectCreateOptions{
			Name: String(badIdentifier),
		})
		assert.Nil(t, w)
		assert.EqualError(t, err, ErrInvalidName.Error())
	})

	t.Run("when options has an invalid organization", func(t *testing.T) {
		w, err := client.Projects.Create(ctx, badIdentifier, ProjectCreateOptions{
			Name: String("foo"),
		})
		assert.Nil(t, w)
		assert.EqualError(t, err, ErrInvalidOrg.Error())
	})
}

func TestProjectsUpdate(t *testing.T) {
	skipIfNotCINode(t)

	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	defer orgTestCleanup()

	t.Run("with valid options", func(t *testing.T) {
		kBefore, kTestCleanup := createProject(t, client, orgTest)
		defer kTestCleanup()

		kAfter, err := client.Projects.Update(ctx, kBefore.ID, ProjectUpdateOptions{
			Name: String("new_project_name"),
		})
		require.NoError(t, err)

		assert.Equal(t, kBefore.ID, kAfter.ID)
		assert.NotEqual(t, kBefore.Name, kAfter.Name)
	})

	t.Run("when updating with invalid name", func(t *testing.T) {
		kBefore, kTestCleanup := createProject(t, client, orgTest)
		defer kTestCleanup()

		kAfter, err := client.Projects.Update(ctx, kBefore.ID, ProjectUpdateOptions{
			Name: String(badIdentifier),
		})
		assert.Nil(t, kAfter)
		assert.EqualError(t, err, ErrInvalidName.Error())
	})

	t.Run("without a valid projects ID", func(t *testing.T) {
		w, err := client.Projects.Update(ctx, badIdentifier, ProjectUpdateOptions{})
		assert.Nil(t, w)
		assert.EqualError(t, err, ErrInvalidProjectID.Error())
	})
}
