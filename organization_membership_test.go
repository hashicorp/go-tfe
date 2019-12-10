package tfe

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOrganizationMembershipsList(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	defer orgTestCleanup()

	memTest1, memTest1Cleanup := createOrganizationMembership(t, client, orgTest)
	defer memTest1Cleanup()
	memTest2, memTest2Cleanup := createOrganizationMembership(t, client, orgTest)
	defer memTest2Cleanup()

	t.Run("without list options", func(t *testing.T) {
		ml, err := client.OrganizationMemberships.List(ctx, orgTest.Name, OrganizationMembershipListOptions{})
		require.NoError(t, err)

		assert.Contains(t, ml.Items, memTest1)
		assert.Contains(t, ml.Items, memTest2)
	})

	t.Run("with list options", func(t *testing.T) {
		ml, err := client.OrganizationMemberships.List(ctx, orgTest.Name, OrganizationMembershipListOptions{
			ListOptions: ListOptions{
				PageNumber: 999,
				PageSize:   100,
			},
		})

		require.NoError(t, err)
		assert.Empty(t, ml.Items)
		assert.Equal(t, 999, ml.CurrentPage)

		// Three because the creator of the organizaiton is a member, in addition to the two we added to setup the test.
		assert.Equal(t, 3, ml.TotalCount)
	})

	t.Run("without a valid organization", func(t *testing.T) {
		ml, err := client.OrganizationMemberships.List(ctx, badIdentifier, OrganizationMembershipListOptions{})
		assert.Nil(t, ml)
		assert.EqualError(t, err, "invalid value for organization")
	})
}

func TestOrganizationMembershipsCreate(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	defer orgTestCleanup()

	t.Run("with valid options", func(t *testing.T) {
		options := OrganizationMembershipCreateOptions{
			Email: String(fmt.Sprintf("%s@tfe.local", randomString(t))),
		}

		mem, err := client.OrganizationMemberships.Create(ctx, orgTest.Name, options)
		require.NoError(t, err)

		// Get a refreshed view from the API.
		refreshed, err := client.OrganizationMemberships.List(ctx, orgTest.Name, OrganizationMembershipListOptions{})
		require.NoError(t, err)
		assert.Contains(t, refreshed.Items, mem)
	})

	t.Run("when options is missing email", func(t *testing.T) {
		mem, err := client.OrganizationMemberships.Create(ctx, orgTest.Name, OrganizationMembershipCreateOptions{})

		assert.Nil(t, mem)
		assert.EqualError(t, err, "email is required")
	})

	t.Run("with an invalid organization", func(t *testing.T) {
		mem, err := client.OrganizationMemberships.Create(ctx, badIdentifier, OrganizationMembershipCreateOptions{
			Email: String(fmt.Sprintf("%s@tfe.local", randomString(t))),
		})

		assert.Nil(t, mem)
		assert.EqualError(t, err, "invalid value for organization")
	})

	t.Run("when an error is returned from the api", func(t *testing.T) {
		mem, err := client.OrganizationMemberships.Create(ctx, orgTest.Name, OrganizationMembershipCreateOptions{
			Email: String("not-an-email-address"),
		})

		assert.Nil(t, mem)
		assert.Error(t, err)
	})
}

func TestOrganizationMembershipsDelete(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	defer orgTestCleanup()

	mem, _ := createOrganizationMembership(t, client, orgTest)

	t.Run("with valid options", func(t *testing.T) {
		err := client.OrganizationMemberships.Delete(ctx, mem.ID)
		require.NoError(t, err)

		// Get a refreshed view from the API.
		refreshed, err := client.OrganizationMemberships.List(ctx, orgTest.Name, OrganizationMembershipListOptions{})
		require.NoError(t, err)
		assert.NotContains(t, refreshed.Items, mem)
	})

	t.Run("when membership is invalid", func(t *testing.T) {
		err := client.OrganizationMemberships.Delete(ctx, badIdentifier)

		assert.EqualError(t, err, "invalid value for membership")
	})

	t.Run("when an error is returned from the api", func(t *testing.T) {
		err := client.OrganizationMemberships.Delete(ctx, "not-an-identifier")

		assert.Error(t, err)
	})
}
