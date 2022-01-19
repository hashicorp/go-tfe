//go:build integration
// +build integration

package tfe

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNotificationConfigurationList(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	wTest, wTestCleanup := createWorkspace(t, client, nil)
	defer wTestCleanup()

	ncTest1, ncTestCleanup1 := createNotificationConfiguration(t, client, wTest, nil)
	defer ncTestCleanup1()
	ncTest2, ncTestCleanup2 := createNotificationConfiguration(t, client, wTest, nil)
	defer ncTestCleanup2()

	t.Run("with a valid workspace", func(t *testing.T) {
		ncl, err := client.NotificationConfigurations.List(
			ctx,
			wTest.ID,
			nil,
		)
		require.NoError(t, err)
		assert.Contains(t, ncl.Items, ncTest1)
		assert.Contains(t, ncl.Items, ncTest2)

		t.Skip("paging not supported yet in API")
		assert.Equal(t, 1, ncl.CurrentPage)
		assert.Equal(t, 2, ncl.TotalCount)
	})

	t.Run("with list options", func(t *testing.T) {
		t.Skip("paging not supported yet in API")
		// Request a page number which is out of range. The result should
		// be successful, but return no results if the paging options are
		// properly passed along.
		ncl, err := client.NotificationConfigurations.List(
			ctx,
			wTest.ID,
			&NotificationConfigurationListOptions{
				ListOptions: ListOptions{
					PageNumber: 999,
					PageSize:   100,
				},
			},
		)
		require.NoError(t, err)
		assert.Empty(t, ncl.Items)
		assert.Equal(t, 999, ncl.CurrentPage)
		assert.Equal(t, 2, ncl.TotalCount)
	})

	t.Run("without a valid workspace", func(t *testing.T) {
		ncl, err := client.NotificationConfigurations.List(
			ctx,
			badIdentifier,
			nil,
		)
		assert.Nil(t, ncl)
		assert.EqualError(t, err, ErrInvalidWorkspaceID.Error())
	})
}

func TestNotificationConfigurationCreate(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	defer orgTestCleanup()

	wTest, wTestCleanup := createWorkspace(t, client, orgTest)
	defer wTestCleanup()

	// Create user to use when testing email destination type
	orgMemberTest, orgMemberTestCleanup := createOrganizationMembership(t, client, orgTest)
	defer orgMemberTestCleanup()

	orgMemberTest.User = &User{ID: orgMemberTest.User.ID}

	t.Run("with all required values", func(t *testing.T) {
		options := NotificationConfigurationCreateOptions{
			DestinationType: NotificationDestination(NotificationDestinationTypeGeneric),
			Enabled:         Bool(false),
			Name:            String(randomString(t)),
			Token:           String(randomString(t)),
			URL:             String("http://example.com"),
			Triggers:        []string{NotificationTriggerCreated},
		}

		_, err := client.NotificationConfigurations.Create(ctx, wTest.ID, options)
		require.NoError(t, err)
	})

	t.Run("without a required value", func(t *testing.T) {
		options := NotificationConfigurationCreateOptions{
			DestinationType: NotificationDestination(NotificationDestinationTypeGeneric),
			Enabled:         Bool(false),
			Token:           String(randomString(t)),
			URL:             String("http://example.com"),
			Triggers:        []string{NotificationTriggerCreated},
		}

		nc, err := client.NotificationConfigurations.Create(ctx, wTest.ID, options)
		assert.Nil(t, nc)
		assert.EqualError(t, err, ErrRequiredName.Error())
	})

	t.Run("without a required value URL when destination type is generic", func(t *testing.T) {
		options := NotificationConfigurationCreateOptions{
			DestinationType: NotificationDestination(NotificationDestinationTypeGeneric),
			Enabled:         Bool(false),
			Name:            String(randomString(t)),
			Token:           String(randomString(t)),
			Triggers:        []string{NotificationTriggerCreated},
		}

		nc, err := client.NotificationConfigurations.Create(ctx, wTest.ID, options)
		assert.Nil(t, nc)
		assert.EqualError(t, err, "url is required")
	})

	t.Run("without a valid workspace", func(t *testing.T) {
		nc, err := client.NotificationConfigurations.Create(ctx, badIdentifier, NotificationConfigurationCreateOptions{})
		assert.Nil(t, nc)
		assert.EqualError(t, err, ErrInvalidWorkspaceID.Error())
	})

	t.Run("with email users when destination type is email", func(t *testing.T) {
		options := NotificationConfigurationCreateOptions{
			DestinationType: NotificationDestination(NotificationDestinationTypeEmail),
			Enabled:         Bool(false),
			Name:            String(randomString(t)),
			EmailUsers:      []*User{orgMemberTest.User},
		}

		_, err := client.NotificationConfigurations.Create(ctx, wTest.ID, options)
		require.NoError(t, err)
	})

	t.Run("without email users when destination type is email", func(t *testing.T) {
		options := NotificationConfigurationCreateOptions{
			DestinationType: NotificationDestination(NotificationDestinationTypeEmail),
			Enabled:         Bool(false),
			Name:            String(randomString(t)),
		}

		_, err := client.NotificationConfigurations.Create(ctx, wTest.ID, options)
		require.NoError(t, err)
	})
}

func TestNotificationConfigurationRead(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	ncTest, ncTestCleanup := createNotificationConfiguration(t, client, nil, nil)
	defer ncTestCleanup()

	t.Run("with a valid ID", func(t *testing.T) {
		nc, err := client.NotificationConfigurations.Read(ctx, ncTest.ID)
		require.NoError(t, err)
		assert.Equal(t, ncTest.ID, nc.ID)
	})

	t.Run("when the notification configuration does not exist", func(t *testing.T) {
		_, err := client.NotificationConfigurations.Read(ctx, "nonexisting")
		assert.Equal(t, err, ErrResourceNotFound)
	})

	t.Run("when the notification configuration ID is invalid", func(t *testing.T) {
		_, err := client.NotificationConfigurations.Read(ctx, badIdentifier)
		assert.EqualError(t, err, "invalid value for notification configuration ID")
	})
}

func TestNotificationConfigurationUpdate(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	defer orgTestCleanup()

	wTest, wTestCleanup := createWorkspace(t, client, orgTest)
	defer wTestCleanup()

	ncTest, ncTestCleanup := createNotificationConfiguration(t, client, wTest, nil)
	defer ncTestCleanup()

	// Create users to use when testing email destination type
	orgMemberTest1, orgMemberTest1Cleanup := createOrganizationMembership(t, client, orgTest)
	defer orgMemberTest1Cleanup()
	orgMemberTest2, orgMemberTest2Cleanup := createOrganizationMembership(t, client, orgTest)
	defer orgMemberTest2Cleanup()

	orgMemberTest1.User = &User{ID: orgMemberTest1.User.ID}
	orgMemberTest2.User = &User{ID: orgMemberTest2.User.ID}

	options := &NotificationConfigurationCreateOptions{
		DestinationType: NotificationDestination(NotificationDestinationTypeEmail),
		Enabled:         Bool(false),
		Name:            String(randomString(t)),
		EmailUsers:      []*User{orgMemberTest1.User},
	}
	ncEmailTest, ncEmailTestCleanup := createNotificationConfiguration(t, client, wTest, options)
	defer ncEmailTestCleanup()

	t.Run("with options", func(t *testing.T) {
		options := NotificationConfigurationUpdateOptions{
			Enabled: Bool(true),
			Name:    String("newName"),
		}

		nc, err := client.NotificationConfigurations.Update(ctx, ncTest.ID, options)
		require.NoError(t, err)
		assert.Equal(t, nc.Enabled, true)
		assert.Equal(t, nc.Name, "newName")
	})

	t.Run("with email users when destination type is email", func(t *testing.T) {
		options := NotificationConfigurationUpdateOptions{
			Enabled:    Bool(true),
			Name:       String("newName"),
			EmailUsers: []*User{orgMemberTest1.User, orgMemberTest2.User},
		}

		nc, err := client.NotificationConfigurations.Update(ctx, ncEmailTest.ID, options)
		require.NoError(t, err)
		assert.Equal(t, nc.Enabled, true)
		assert.Equal(t, nc.Name, "newName")
		assert.Contains(t, nc.EmailUsers, orgMemberTest1.User)
		assert.Contains(t, nc.EmailUsers, orgMemberTest2.User)
	})

	t.Run("without email users when destination type is email", func(t *testing.T) {
		options := NotificationConfigurationUpdateOptions{
			Enabled: Bool(true),
			Name:    String("newName"),
		}

		nc, err := client.NotificationConfigurations.Update(ctx, ncEmailTest.ID, options)
		require.NoError(t, err)
		assert.Equal(t, nc.Enabled, true)
		assert.Equal(t, nc.Name, "newName")
		assert.Empty(t, nc.EmailUsers)
	})

	t.Run("without options", func(t *testing.T) {
		_, err := client.NotificationConfigurations.Update(ctx, ncTest.ID, NotificationConfigurationUpdateOptions{})
		require.NoError(t, err)
	})

	t.Run("when the notification configuration does not exist", func(t *testing.T) {
		_, err := client.NotificationConfigurations.Update(ctx, "nonexisting", NotificationConfigurationUpdateOptions{})
		assert.Equal(t, err, ErrResourceNotFound)
	})

	t.Run("when the notification configuration ID is invalid", func(t *testing.T) {
		_, err := client.NotificationConfigurations.Update(ctx, badIdentifier, NotificationConfigurationUpdateOptions{})
		assert.EqualError(t, err, "invalid value for notification configuration ID")
	})
}

func TestNotificationConfigurationDelete(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	wTest, wTestCleanup := createWorkspace(t, client, nil)
	defer wTestCleanup()

	ncTest, _ := createNotificationConfiguration(t, client, wTest, nil)

	t.Run("with a valid ID", func(t *testing.T) {
		err := client.NotificationConfigurations.Delete(ctx, ncTest.ID)
		require.NoError(t, err)

		_, err = client.NotificationConfigurations.Read(ctx, ncTest.ID)
		assert.Equal(t, err, ErrResourceNotFound)
	})

	t.Run("when the notification configuration does not exist", func(t *testing.T) {
		err := client.NotificationConfigurations.Delete(ctx, "nonexisting")
		assert.Equal(t, err, ErrResourceNotFound)
	})

	t.Run("when the notification configuration ID is invalid", func(t *testing.T) {
		err := client.NotificationConfigurations.Delete(ctx, badIdentifier)
		assert.EqualError(t, err, "invalid value for notification configuration ID")
	})
}

func TestNotificationConfigurationVerify(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	ncTest, ncTestCleanup := createNotificationConfiguration(t, client, nil, nil)
	defer ncTestCleanup()

	t.Run("with a valid ID", func(t *testing.T) {
		_, err := client.NotificationConfigurations.Verify(ctx, ncTest.ID)
		require.NoError(t, err)
	})

	t.Run("when the notification configuration does not exists", func(t *testing.T) {
		_, err := client.NotificationConfigurations.Verify(ctx, "nonexisting")
		assert.Equal(t, err, ErrResourceNotFound)
	})

	t.Run("when the notification configuration ID is invalid", func(t *testing.T) {
		_, err := client.NotificationConfigurations.Verify(ctx, badIdentifier)
		assert.EqualError(t, err, "invalid value for notification configuration ID")
	})
}
