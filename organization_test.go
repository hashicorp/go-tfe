package tfe

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOrganizationsList(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	orgTest1, orgTest1Cleanup := createOrganization(t, client)
	defer orgTest1Cleanup()
	orgTest2, orgTest2Cleanup := createOrganization(t, client)
	defer orgTest2Cleanup()

	t.Run("with no list options", func(t *testing.T) {
		orgs, err := client.Organizations.List(ctx, OrganizationListOptions{})
		require.NoError(t, err)
		assert.Contains(t, orgs, orgTest1)
		assert.Contains(t, orgs, orgTest2)
	})

	t.Run("with list options", func(t *testing.T) {
		t.Skip("paging not supported yet in API")
		// Request a page number which is out of range. The result should
		// be successful, but return no results if the paging options are
		// properly passed along.
		orgs, err := client.Organizations.List(ctx, OrganizationListOptions{
			ListOptions: ListOptions{
				PageNumber: 999,
				PageSize:   100,
			},
		})
		require.NoError(t, err)
		assert.Empty(t, orgs)
	})
}

func TestOrganizationsCreate(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	t.Run("with valid options", func(t *testing.T) {
		options := OrganizationCreateOptions{
			Name:  String(randomString(t)),
			Email: String(randomString(t) + "@tfe.local"),
		}

		org, err := client.Organizations.Create(ctx, options)
		require.NoError(t, err)

		// Make sure we clean up the created org.
		defer client.Organizations.Delete(ctx, org.Name)

		assert.Equal(t, *options.Name, org.Name)
		assert.Equal(t, *options.Email, org.Email)
	})

	t.Run("when no email is provided", func(t *testing.T) {
		org, err := client.Organizations.Create(ctx, OrganizationCreateOptions{
			Name: String("foo"),
		})
		assert.Nil(t, org)
		assert.EqualError(t, err, "Email is required")
	})

	t.Run("when no name is provided", func(t *testing.T) {
		_, err := client.Organizations.Create(ctx, OrganizationCreateOptions{
			Email: String("foo@bar.com"),
		})
		assert.EqualError(t, err, "Name is required")
	})

	t.Run("with invalid name", func(t *testing.T) {
		org, err := client.Organizations.Create(ctx, OrganizationCreateOptions{
			Name:  String(badIdentifier),
			Email: String("foo@bar.com"),
		})
		assert.Nil(t, org)
		assert.EqualError(t, err, "Invalid value for name")
	})
}

func TestOrganizationsRead(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	defer orgTestCleanup()

	t.Run("when the org exists", func(t *testing.T) {
		org, err := client.Organizations.Read(ctx, orgTest.Name)
		require.NoError(t, err)
		assert.Equal(t, orgTest, org)

		t.Run("permissions are properly decoded", func(t *testing.T) {
			assert.True(t, org.Permissions.CanDestroy)
		})

		t.Run("timestamps are populated", func(t *testing.T) {
			assert.NotEmpty(t, org.CreatedAt)
			assert.NotEmpty(t, org.TrialExpiresAt)
		})
	})

	t.Run("with invalid name", func(t *testing.T) {
		org, err := client.Organizations.Read(ctx, badIdentifier)
		assert.Nil(t, org)
		assert.EqualError(t, err, "Invalid value for organization")
	})

	t.Run("when the org does not exist", func(t *testing.T) {
		_, err := client.Organizations.Read(ctx, randomString(t))
		assert.Error(t, err)
	})
}

func TestOrganizationsUpdate(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	t.Run("with valid options", func(t *testing.T) {
		orgTest, orgTestCleanup := createOrganization(t, client)

		options := OrganizationUpdateOptions{
			Name:            String(randomString(t)),
			Email:           String(randomString(t) + "@tfe.local"),
			SessionTimeout:  Int(3600),
			SessionRemember: Int(3600),
		}

		org, err := client.Organizations.Update(ctx, orgTest.Name, options)
		if err != nil {
			orgTestCleanup()
		}
		require.NoError(t, err)

		// Make sure we clean up the renamed org.
		defer client.Organizations.Delete(ctx, org.Name)

		// Also get a fresh result from the API to ensure we get the
		// expected values back.
		refreshed, err := client.Organizations.Read(ctx, *options.Name)
		require.NoError(t, err)

		for _, item := range []*Organization{
			org,
			refreshed,
		} {
			assert.Equal(t, *options.Name, item.Name)
			assert.Equal(t, *options.Email, item.Email)
			assert.Equal(t, *options.SessionTimeout, item.SessionTimeout)
			assert.Equal(t, *options.SessionRemember, item.SessionRemember)
		}
	})

	t.Run("with invalid name", func(t *testing.T) {
		org, err := client.Organizations.Update(ctx, badIdentifier, OrganizationUpdateOptions{})
		assert.Nil(t, org)
		assert.EqualError(t, err, "Invalid value for organization")
	})

	t.Run("when only updating a subset of fields", func(t *testing.T) {
		orgTest, orgTestCleanup := createOrganization(t, client)
		defer orgTestCleanup()

		org, err := client.Organizations.Update(ctx, orgTest.Name, OrganizationUpdateOptions{})
		require.NoError(t, err)
		assert.Equal(t, orgTest.Name, org.Name)
		assert.Equal(t, orgTest.Email, org.Email)
	})
}

func TestOrganizationsDelete(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	t.Run("with valid options", func(t *testing.T) {
		orgTest, _ := createOrganization(t, client)

		err := client.Organizations.Delete(ctx, orgTest.Name)
		require.NoError(t, err)

		// Try fetching the org again - it should error.
		_, err = client.Organizations.Read(ctx, orgTest.Name)
		assert.Equal(t, err, ErrResourceNotFound)
	})

	t.Run("with invalid name", func(t *testing.T) {
		err := client.Organizations.Delete(ctx, badIdentifier)
		assert.EqualError(t, err, "Invalid value for organization")
	})
}
