package tfe

import (
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOrganizations(t *testing.T) {
	client := testClient(t)

	org1, cleanupOrg1 := createOrganization(t, client)
	defer cleanupOrg1()
	org2, cleanupOrg2 := createOrganization(t, client)
	defer cleanupOrg2()

	// Get an initial list of the organizations for comparison.
	orgs, err := client.Organizations()
	require.Nil(t, err)

	expect := []*Organization{org1, org2}

	// Sort to ensure we are comparing in the right order
	sort.Stable(OrganizationNameSort(expect))
	sort.Stable(OrganizationNameSort(orgs))

	assert.Equal(t, expect, orgs)
}

func TestOrganization(t *testing.T) {
	client := testClient(t)

	org, cleanup := createOrganization(t, client)
	defer cleanup()

	t.Run("when the org exists", func(t *testing.T) {
		result, err := client.Organization(*org.Name)
		require.Nil(t, err)
		assert.Equal(t, org, result)
	})

	t.Run("when the org does not exist", func(t *testing.T) {
		_, err := client.Organization(randomString(t))
		assert.NotNil(t, err)
	})

	t.Run("permissions are properly decoded", func(t *testing.T) {
		if !org.Permissions.Can("destroy") {
			t.Fatal("should be able to destroy")
		}
	})
}

func TestCreateOrganization(t *testing.T) {
	client := testClient(t)

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
}

func TestModifyOrganization(t *testing.T) {
	client := testClient(t)

	t.Run("with valid parameters", func(t *testing.T) {
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

	t.Run("with invalid parameters", func(t *testing.T) {
		org, cleanup := createOrganization(t, client)
		defer cleanup()

		input := &ModifyOrganizationInput{
			Name:  org.Name,
			Email: String("nope"),
		}

		_, err := client.ModifyOrganization(input)
		assert.NotNil(t, err)
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
