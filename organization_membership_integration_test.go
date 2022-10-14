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
	skipIfNotCINode(t)

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
			Include: []OrgMembershipIncludeOpt{OrgMembershipUser},
		})
		require.NoError(t, err)

		assert.Contains(t, ml.Items, memTest1)
		assert.Contains(t, ml.Items, memTest2)
	})

	t.Run("with email filter option", func(t *testing.T) {
		_, memTest1Cleanup := createOrganizationMembership(t, client, orgTest)
		defer memTest1Cleanup()
		memTest2, memTest2Cleanup := createOrganizationMembership(t, client, orgTest)
		defer memTest2Cleanup()

		memTest3, memTest3Cleanup := createOrganizationMembership(t, client, orgTest)
		defer memTest3Cleanup()

		memTest2.User = &User{ID: memTest2.User.ID}
		memTest3.User = &User{ID: memTest3.User.ID}

		ml, err := client.OrganizationMemberships.List(ctx, orgTest.Name, &OrganizationMembershipListOptions{
			Emails: []string{memTest2.Email, memTest3.Email},
		})
		require.NoError(t, err)

		assert.Len(t, ml.Items, 2)
		assert.Contains(t, ml.Items, memTest2)
		assert.Contains(t, ml.Items, memTest3)

		t.Run("with invalid email", func(t *testing.T) {
			ml, err = client.OrganizationMemberships.List(ctx, orgTest.Name, &OrganizationMembershipListOptions{
				Emails: []string{"foobar"},
			})
			assert.Equal(t, err, ErrInvalidEmail)
		})
	})

	t.Run("with status filter option", func(t *testing.T) {
		_, memTest1Cleanup := createOrganizationMembership(t, client, orgTest)
		t.Cleanup(memTest1Cleanup)
		_, memTest2Cleanup := createOrganizationMembership(t, client, orgTest)
		t.Cleanup(memTest2Cleanup)

		ml, err := client.OrganizationMemberships.List(ctx, orgTest.Name, &OrganizationMembershipListOptions{
			Status: OrganizationMembershipInvited,
		})
		require.NoError(t, err)

		require.Len(t, ml.Items, 2)
		for _, member := range ml.Items {
			assert.Equal(t, member.Status, OrganizationMembershipInvited)
		}
	})

	t.Run("with search query string", func(t *testing.T) {
		memTest1, memTest1Cleanup := createOrganizationMembership(t, client, orgTest)
		t.Cleanup(memTest1Cleanup)
		_, memTest2Cleanup := createOrganizationMembership(t, client, orgTest)
		t.Cleanup(memTest2Cleanup)
		_, memTest3Cleanup := createOrganizationMembership(t, client, orgTest)
		t.Cleanup(memTest3Cleanup)

		t.Run("using an email", func(t *testing.T) {
			ml, err := client.OrganizationMemberships.List(ctx, orgTest.Name, &OrganizationMembershipListOptions{
				Query: memTest1.Email,
			})
			require.NoError(t, err)

			require.Len(t, ml.Items, 1)
			assert.Equal(t, ml.Items[0].Email, memTest1.Email)
		})

		t.Run("using a user name", func(t *testing.T) {
			t.Skip("Skipping, missing Account API support in order to set usernames")
		})
	})

	t.Run("without a valid organization", func(t *testing.T) {
		ml, err := client.OrganizationMemberships.List(ctx, badIdentifier, nil)
		assert.Nil(t, ml)
		assert.EqualError(t, err, ErrInvalidOrg.Error())
	})
}

func TestOrganizationMembershipsCreate(t *testing.T) {
	skipIfNotCINode(t)

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
			Include: []OrgMembershipIncludeOpt{OrgMembershipUser},
		})
		require.NoError(t, err)
		assert.Equal(t, refreshed, mem)
	})

	t.Run("when options is missing email", func(t *testing.T) {
		mem, err := client.OrganizationMemberships.Create(ctx, orgTest.Name, OrganizationMembershipCreateOptions{})

		assert.Nil(t, mem)
		assert.Equal(t, err, ErrRequiredEmail)
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
	skipIfNotCINode(t)

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
		assert.Equal(t, err, ErrInvalidMembership)
	})
}

func TestOrganizationMembershipsReadWithOptions(t *testing.T) {
	skipIfNotCINode(t)

	client := testClient(t)
	ctx := context.Background()

	memTest, memTestCleanup := createOrganizationMembership(t, client, nil)
	defer memTestCleanup()

	options := OrganizationMembershipReadOptions{
		Include: []OrgMembershipIncludeOpt{OrgMembershipUser},
	}

	t.Run("when the membership exists", func(t *testing.T) {
		mem, err := client.OrganizationMemberships.ReadWithOptions(ctx, memTest.ID, options)
		require.NoError(t, err)

		assert.Equal(t, memTest, mem)
	})

	t.Run("without options", func(t *testing.T) {
		_, err := client.OrganizationMemberships.ReadWithOptions(ctx, memTest.ID, OrganizationMembershipReadOptions{})
		require.NoError(t, err)
	})

	t.Run("without invalid include option", func(t *testing.T) {
		_, err := client.OrganizationMemberships.ReadWithOptions(ctx, memTest.ID, OrganizationMembershipReadOptions{
			Include: []OrgMembershipIncludeOpt{"users"},
		})
		assert.Equal(t, err, ErrInvalidIncludeValue)
	})

	t.Run("when the membership does not exist", func(t *testing.T) {
		mem, err := client.OrganizationMemberships.ReadWithOptions(ctx, "nonexisting", options)
		assert.Nil(t, mem)
		assert.Equal(t, err, ErrResourceNotFound)
	})

	t.Run("with invalid membership id", func(t *testing.T) {
		mem, err := client.OrganizationMemberships.ReadWithOptions(ctx, badIdentifier, options)
		assert.Nil(t, mem)
		assert.Equal(t, err, ErrInvalidMembership)
	})
}

func TestOrganizationMembershipsDelete(t *testing.T) {
	skipIfNotCINode(t)

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

		assert.Equal(t, err, ErrInvalidMembership)
	})

	t.Run("when an error is returned from the api", func(t *testing.T) {
		err := client.OrganizationMemberships.Delete(ctx, "not-an-identifier")

		assert.Error(t, err)
	})
}
