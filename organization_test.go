package tfe

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOrganizationsList(t *testing.T) {
	client := testClient(t)

	orgTest1, orgTest1Cleanup := createOrganization(t, client)
	defer orgTest1Cleanup()
	orgTest2, orgTest2Cleanup := createOrganization(t, client)
	defer orgTest2Cleanup()

	t.Run("with no list options", func(t *testing.T) {
		orgs, err := client.Organizations.List(ListOrganizationsOptions{})
		require.Nil(t, err)

		assert.Contains(t, orgs, orgTest1)
		assert.Contains(t, orgs, orgTest2)
	})

	t.Run("with list options", func(t *testing.T) {
		t.Skip("paging not supported yet in API")

		// Request a page number which is out of range. The result should
		// be successful, but return no results if the paging options are
		// properly passed along.
		orgs, err := client.Organizations.List(ListOrganizationsOptions{
			ListOptions: ListOptions{
				PageNumber: 999,
				PageSize:   100,
			},
		})
		require.Nil(t, err)

		assert.Equal(t, 0, len(orgs))
	})
}

func TestOrganizationsCreate(t *testing.T) {
	client := testClient(t)

	t.Run("with valid options", func(t *testing.T) {
		options := CreateOrganizationOptions{
			Name:  String(randomString(t)),
			Email: String(randomString(t) + "@tfe.local"),
		}

		org, err := client.Organizations.Create(options)
		require.Nil(t, err)
		defer client.Organizations.Delete(org.Name)

		assert.Equal(t, *options.Name, org.Name)
		assert.Equal(t, *options.Email, org.Email)
	})

	t.Run("without valid options", func(t *testing.T) {
		_, err := client.Organizations.Create(CreateOrganizationOptions{})
		require.NotNil(t, err)
	})

	t.Run("with invalid name", func(t *testing.T) {
		org, err := client.Organizations.Create(CreateOrganizationOptions{
			Name:  String(badIdentifier),
			Email: String("foo@bar.com"),
		})
		assert.Nil(t, org)
		assert.EqualError(t, err, "Invalid value for name")
	})

	t.Run("when no email is provided", func(t *testing.T) {
		org, err := client.Organizations.Create(CreateOrganizationOptions{
			Name: String("foo"),
		})
		assert.Nil(t, org)
		assert.EqualError(t, err, "Email is required")
	})
}

func TestOrganizationsRetrieve(t *testing.T) {
	client := testClient(t)

	orgTest, orgTestCleanup := createOrganization(t, client)
	defer orgTestCleanup()

	t.Run("when the org exists", func(t *testing.T) {
		org, err := client.Organizations.Retrieve(orgTest.Name)
		require.Nil(t, err)
		assert.Equal(t, orgTest, org)

		t.Run("permissions are properly decoded", func(t *testing.T) {
			assert.True(t, org.Permissions.CanDestroy)
		})

		t.Run("timestamps are populated", func(t *testing.T) {
			assert.False(t, org.CreatedAt.IsZero())
			assert.False(t, org.TrialExpiresAt.IsZero())
		})
	})

	t.Run("with invalid name", func(t *testing.T) {
		org, err := client.Organizations.Retrieve(badIdentifier)
		assert.Nil(t, org)
		assert.EqualError(t, err, "Invalid value for name")
	})

	t.Run("when the org does not exist", func(t *testing.T) {
		_, err := client.Organizations.Retrieve(randomString(t))
		assert.NotNil(t, err)
	})
}

func TestOrganizationsUpdate(t *testing.T) {
	client := testClient(t)

	t.Run("with valid options", func(t *testing.T) {
		orgTest, orgTestCleanup := createOrganization(t, client)

		options := UpdateOrganizationOptions{
			Name:  String(randomString(t)),
			Email: String(randomString(t) + "@tfe.local"),
		}

		org, err := client.Organizations.Update(orgTest.Name, options)
		if err != nil {
			orgTestCleanup()
		}
		require.Nil(t, err)

		// Make sure we clean up the renamed org.
		defer client.Organizations.Delete(org.Name)

		// Also get a fresh result from the API to ensure we get the
		// expected values back.
		refreshed, err := client.Organizations.Retrieve(*options.Name)
		require.Nil(t, err)

		for _, item := range []*Organization{
			org,
			refreshed,
		} {
			assert.Equal(t, *options.Name, item.Name)
			assert.Equal(t, *options.Email, item.Email)
		}
	})

	t.Run("with invalid name", func(t *testing.T) {
		org, err := client.Organizations.Update(badIdentifier, UpdateOrganizationOptions{})
		assert.Nil(t, org)
		assert.EqualError(t, err, "Invalid value for name")
	})

	t.Run("when only updating a subset of fields", func(t *testing.T) {
		orgTest, orgTestCleanup := createOrganization(t, client)
		defer orgTestCleanup()

		org, err := client.Organizations.Update(orgTest.Name, UpdateOrganizationOptions{})
		require.Nil(t, err)

		assert.Equal(t, orgTest.Name, org.Name)
		assert.Equal(t, orgTest.Email, org.Email)
	})
}

func TestOrganizationsDelete(t *testing.T) {
	client := testClient(t)

	t.Run("with valid options", func(t *testing.T) {
		orgTest, _ := createOrganization(t, client)

		err := client.Organizations.Delete(orgTest.Name)
		require.Nil(t, err)

		// Try fetching the org again - it should error.
		_, err = client.Organizations.Retrieve(orgTest.Name)
		assert.EqualError(t, err, "Resource not found")
	})

	t.Run("with invalid name", func(t *testing.T) {
		err := client.Organizations.Delete(badIdentifier)
		assert.EqualError(t, err, "Invalid value for name")
	})
}
