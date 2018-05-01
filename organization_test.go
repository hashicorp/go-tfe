package tfe

import (
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListOrganizations(t *testing.T) {
	client := testClient(t)

	org1, cleanupOrg1 := createOrganization(t, client)
	defer cleanupOrg1()
	org2, cleanupOrg2 := createOrganization(t, client)
	defer cleanupOrg2()

	t.Run("with no list options", func(t *testing.T) {
		orgs, err := client.ListOrganizations(&ListOrganizationsInput{})
		require.Nil(t, err)

		expect := []*Organization{org1, org2}

		// Sort to ensure we are comparing in the right order
		sort.Stable(OrganizationNameSort(expect))
		sort.Stable(OrganizationNameSort(orgs))

		assert.Equal(t, expect, orgs)
	})

	t.Run("with list options", func(t *testing.T) {
		t.Skip("paging not supported yet in API")

		// Request a page number which is out of range. The result should
		// be successful, but return no results if the paging options are
		// properly passed along.
		orgs, err := client.ListOrganizations(&ListOrganizationsInput{
			ListOptions: ListOptions{
				PageNumber: 999,
				PageSize:   100,
			},
		})
		require.Nil(t, err)

		assert.Equal(t, 0, len(orgs))
	})
}

func TestOrganization(t *testing.T) {
	client := testClient(t)

	org, cleanup := createOrganization(t, client)
	defer cleanup()

	t.Run("when the org exists", func(t *testing.T) {
		result, err := client.Organization(*org.Name)
		require.Nil(t, err)
		assert.Equal(t, org, result)

		t.Run("permissions are properly decoded", func(t *testing.T) {
			if !result.Permissions.Can("destroy") {
				t.Fatal("should be able to destroy")
			}
		})

		t.Run("timestamps are populated", func(t *testing.T) {
			assert.False(t, result.CreatedAt.IsZero())
			assert.False(t, result.TrialExpiresAt.IsZero())
		})
	})

	t.Run("when the org does not exist", func(t *testing.T) {
		_, err := client.Organization(randomString(t))
		assert.NotNil(t, err)
	})
}

func TestCreateOrganization(t *testing.T) {
	client := testClient(t)

	t.Run("with valid input", func(t *testing.T) {
		input := &CreateOrganizationInput{
			Name:  String(randomString(t)),
			Email: String(randomString(t) + "@tfe.local"),
		}

		result, err := client.CreateOrganization(input)
		require.Nil(t, err)
		defer client.DeleteOrganization(&DeleteOrganizationInput{
			Name: input.Name,
		})

		assert.Equal(t, input.Name, result.Organization.Name)
		assert.Equal(t, input.Email, result.Organization.Email)
	})

	t.Run("with invalid name", func(t *testing.T) {
		result, err := client.CreateOrganization(&CreateOrganizationInput{
			Name:  String("! / nope"),
			Email: String("foo@bar.com"),
		})
		assert.Nil(t, result)
		assert.EqualError(t, err, "Invalid value for Name")
	})

	t.Run("when no email is provided", func(t *testing.T) {
		result, err := client.CreateOrganization(&CreateOrganizationInput{
			Name: String("foo"),
		})
		assert.Nil(t, result)
		assert.EqualError(t, err, "Email is required")
	})
}

func TestModifyOrganization(t *testing.T) {
	client := testClient(t)

	t.Run("with valid input", func(t *testing.T) {
		org, cleanup := createOrganization(t, client)
		defer cleanup()

		input := &ModifyOrganizationInput{
			Name:   org.Name,
			Rename: String(randomString(t)),
			Email:  String(randomString(t) + "@tfe.local"),
		}

		output, err := client.ModifyOrganization(input)
		require.Nil(t, err)

		// Make sure we clean up the renamed org.
		defer client.DeleteOrganization(&DeleteOrganizationInput{
			Name: output.Organization.Name,
		})

		// Also get a fresh result from the API to ensure we get the
		// expected values back.
		refreshedOrg, err := client.Organization(*input.Rename)
		require.Nil(t, err)

		for _, resultOrg := range []*Organization{
			output.Organization,
			refreshedOrg,
		} {
			assert.Equal(t, input.Rename, resultOrg.Name)
			assert.Equal(t, input.Email, resultOrg.Email)
		}
	})

	t.Run("with invalid name", func(t *testing.T) {
		result, err := client.ModifyOrganization(&ModifyOrganizationInput{
			Name: String("! / nope"),
		})
		assert.Nil(t, result)
		assert.EqualError(t, err, "Invalid value for Name")
	})

	t.Run("when only updating a subset of fields", func(t *testing.T) {
		org, cleanup := createOrganization(t, client)
		defer cleanup()

		input := &ModifyOrganizationInput{
			Name: org.Name,
		}

		output, err := client.ModifyOrganization(input)
		require.Nil(t, err)

		result := output.Organization
		assert.Equal(t, input.Name, result.Name)
		assert.Equal(t, org.Email, result.Email)
	})
}

func TestDeleteOrganization(t *testing.T) {
	client := testClient(t)

	t.Run("with valid input", func(t *testing.T) {
		org, cleanup := createOrganization(t, client)
		defer cleanup()

		output, err := client.DeleteOrganization(&DeleteOrganizationInput{
			Name: org.Name,
		})
		require.Nil(t, err)

		require.Equal(t, &DeleteOrganizationOutput{}, output)

		// Try fetching the org again - it should error.
		_, err = client.Organization(*org.Name)
		assert.EqualError(t, err, "Resource not found")
	})

	t.Run("with invalid name", func(t *testing.T) {
		result, err := client.DeleteOrganization(&DeleteOrganizationInput{
			Name: String("! / nope"),
		})
		assert.Nil(t, result)
		assert.EqualError(t, err, "Invalid value for Name")
	})
}
