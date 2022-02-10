//go:build integration
// +build integration

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

	t.Run("without list options", func(t *testing.T) {
		memTest1, memTest1Cleanup := createOrganizationMembership(t, client, orgTest)
		defer memTest1Cleanup()
		memTest2, memTest2Cleanup := createOrganizationMembership(t, client, orgTest)
		defer memTest2Cleanup()

		// The create helper includes the related user, so we should remove it for our equality test
		memTest1.User = &User{ID: memTest1.User.ID}
		memTest2.User = &User{ID: memTest2.User.ID}

		ml, err := client.OrganizationMemberships.List(ctx, orgTest.Name, nil)
		require.NoError(t, err)

		assert.Contains(t, ml.Items, memTest1)
		assert.Contains(t, ml.Items, memTest2)
	})

	t.Run("with pagination options", func(t *testing.T) {
		_, memTest1Cleanup := createOrganizationMembership(t, client, orgTest)
		defer memTest1Cleanup()
		_, memTest2Cleanup := createOrganizationMembership(t, client, orgTest)
		defer memTest2Cleanup()

		ml, err := client.OrganizationMemberships.List(ctx, orgTest.Name, &OrganizationMembershipListOptions{
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

	t.Run("with include options", func(t *testing.T) {
		memTest1, memTest1Cleanup := createOrganizationMembership(t, client, orgTest)
		defer memTest1Cleanup()
		memTest2, memTest2Cleanup := createOrganizationMembership(t, client, orgTest)
		defer memTest2Cleanup()

		ml, err := client.OrganizationMemberships.List(ctx, orgTest.Name, &OrganizationMembershipListOptions{
			Include: []OrganizationMembershipIncludeOps{OrganizationMembershipUser},
		})
		require.NoError(t, err)

		assert.Contains(t, ml.Items, memTest1)
		assert.Contains(t, ml.Items, memTest2)
	})

	t.Run("without a valid organization", func(t *testing.T) {
		ml, err := client.OrganizationMemberships.List(ctx, badIdentifier, nil)
		assert.Nil(t, ml)
		assert.EqualError(t, err, ErrInvalidOrg.Error())
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
		refreshed, err := client.OrganizationMemberships.ReadWithOptions(ctx, mem.ID, OrganizationMembershipReadOptions{
			Include: []OrganizationMembershipIncludeOps{OrganizationMembershipUser},
		})
		require.NoError(t, err)
		assert.Equal(t, refreshed, mem)
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
		assert.EqualError(t, err, ErrInvalidOrg.Error())
	})

	t.Run("when an error is returned from the api", func(t *testing.T) {
		mem, err := client.OrganizationMemberships.Create(ctx, orgTest.Name, OrganizationMembershipCreateOptions{
			Email: String("not-an-email-address"),
		})

		assert.Nil(t, mem)
		assert.Error(t, err)
	})
}

func TestOrganizationMembershipsRead(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	memTest, memTestCleanup := createOrganizationMembership(t, client, nil)
	defer memTestCleanup()

	// The create API endpoint automatically includes the related user, so we should drop
	// the additional parts of the user which get deserialized.
	memTest.User = &User{
		ID: memTest.User.ID,
	}

	t.Run("when the membership exists", func(t *testing.T) {
		mem, err := client.OrganizationMemberships.Read(ctx, memTest.ID)
		require.NoError(t, err)

		assert.Equal(t, memTest, mem)
	})

	t.Run("when the membership does not exist", func(t *testing.T) {
		mem, err := client.OrganizationMemberships.Read(ctx, "nonexisting")
		assert.Nil(t, mem)
		assert.Equal(t, err, ErrResourceNotFound)
	})

	t.Run("with invalid membership id", func(t *testing.T) {
		mem, err := client.OrganizationMemberships.Read(ctx, badIdentifier)
		assert.Nil(t, mem)
		assert.EqualError(t, err, "invalid value for membership")
	})
}

func TestOrganizationMembershipsReadWithOptions(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	memTest, memTestCleanup := createOrganizationMembership(t, client, nil)
	defer memTestCleanup()

	options := OrganizationMembershipReadOptions{
		Include: []OrganizationMembershipIncludeOps{OrganizationMembershipUser},
	}

	t.Run("when the membership exists", func(t *testing.T) {
		mem, err := client.OrganizationMemberships.ReadWithOptions(ctx, memTest.ID, options)
		require.NoError(t, err)

		assert.Equal(t, memTest, mem)
	})

	t.Run("when the membership does not exist", func(t *testing.T) {
		mem, err := client.OrganizationMemberships.ReadWithOptions(ctx, "nonexisting", options)
		assert.Nil(t, mem)
		assert.Equal(t, err, ErrResourceNotFound)
	})

	t.Run("with invalid membership id", func(t *testing.T) {
		mem, err := client.OrganizationMemberships.ReadWithOptions(ctx, badIdentifier, options)
		assert.Nil(t, mem)
		assert.EqualError(t, err, "invalid value for membership")
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
		refreshed, err := client.OrganizationMemberships.List(ctx, orgTest.Name, nil)
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
