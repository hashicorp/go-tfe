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

	w, wCleanup := createWorkspace(t, client, nil)
	defer wCleanup()

	ncTest1, _ := createNotificationConfiguration(t, client, w)
	ncTest2, _ := createNotificationConfiguration(t, client, w)

	t.Run("without a valid workspace", func(t *testing.T) {
		ncl, err := client.NotificationConfigurations.List(ctx, badIdentifier)
		assert.Nil(t, ncl)
		assert.EqualError(t, err, "invalid value for workspace ID")
	})

	t.Run("with a valid workspace", func(t *testing.T) {
		ncl, err := client.NotificationConfigurations.List(ctx, w.ID)
		require.NoError(t, err)

		found := []string{}
		for _, nc := range ncl.Items {
			found = append(found, nc.ID)
		}

		assert.Contains(t, found, ncTest1.ID)
		assert.Contains(t, found, ncTest2.ID)
		assert.Equal(t, 2, len(ncl.Items))
	})
}

func TestNotificationConfigurationCreate(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	w, wCleanup := createWorkspace(t, client, nil)
	defer wCleanup()

	t.Run("without a valid workspace", func(t *testing.T) {
		nc, err := client.NotificationConfigurations.Create(ctx, badIdentifier, NotificationConfigurationCreateOptions{})
		assert.Nil(t, nc)
		assert.EqualError(t, err, "invalid value for workspace ID")
	})

	t.Run("without a required value", func(t *testing.T) {
		options := NotificationConfigurationCreateOptions{
			DestinationType: NotificationDestination(NotificationDestinationTypeGeneric),
			Enabled:         Bool(false),
			Token:           String(randomString(t)),
			URL:             String("http://example.com"),
			Triggers:        []string{NotificationTriggerCreated},
		}

		nc, err := client.NotificationConfigurations.Create(ctx, w.ID, options)
		assert.Nil(t, nc)
		assert.EqualError(t, err, "name is required")
	})

	t.Run("with all required values", func(t *testing.T) {
		options := NotificationConfigurationCreateOptions{
			DestinationType: NotificationDestination(NotificationDestinationTypeGeneric),
			Enabled:         Bool(false),
			Name:            String(randomString(t)),
			Token:           String(randomString(t)),
			URL:             String("http://example.com"),
			Triggers:        []string{NotificationTriggerCreated},
		}

		_, err := client.NotificationConfigurations.Create(ctx, w.ID, options)
		require.NoError(t, err)
	})
}

func TestNotificationConfigurationRead(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	ncTest, ncCleanup := createNotificationConfiguration(t, client, nil)
	defer ncCleanup()

	t.Run("with a valid ID", func(t *testing.T) {
		nc, err := client.NotificationConfigurations.Read(ctx, ncTest.ID)
		require.NoError(t, err)
		assert.Equal(t, ncTest.ID, nc.ID)
	})

	t.Run("when the notification configuration does not exist", func(t *testing.T) {
		_, err := client.NotificationConfigurations.Read(ctx, "nc-notreal")
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

	ncTest, ncCleanup := createNotificationConfiguration(t, client, nil)
	defer ncCleanup()

	t.Run("without options", func(t *testing.T) {
		nc, err := client.NotificationConfigurations.Update(ctx, ncTest.ID, NotificationConfigurationUpdateOptions{})
		require.NoError(t, err)
		assert.Equal(t, ncTest.Enabled, nc.Enabled)
	})

	t.Run("with options", func(t *testing.T) {
		options := NotificationConfigurationUpdateOptions{
			Enabled: Bool(true),
			Name:    String("newName"),
		}

		nc, err := client.NotificationConfigurations.Update(ctx, ncTest.ID, options)
		require.NoError(t, err)
		assert.NotEqual(t, ncTest.Enabled, nc.Enabled)
		assert.NotEqual(t, ncTest.Name, nc.Name)
	})

	t.Run("when the notification configuration does not exist", func(t *testing.T) {
		_, err := client.NotificationConfigurations.Update(ctx, "nc-notreal", NotificationConfigurationUpdateOptions{})
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

	ncTest, _ := createNotificationConfiguration(t, client, nil)

	t.Run("with a valid ID", func(t *testing.T) {
		err := client.NotificationConfigurations.Delete(ctx, ncTest.ID)
		require.NoError(t, err)

		_, err = client.NotificationConfigurations.Read(ctx, ncTest.ID)
		assert.Equal(t, err, ErrResourceNotFound)
	})

	t.Run("when the notification configuration does not exist", func(t *testing.T) {
		err := client.NotificationConfigurations.Delete(ctx, "nc-notreal")
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

	ncTest, _ := createNotificationConfiguration(t, client, nil)

	t.Run("when the notification configuration ID is invalid", func(t *testing.T) {
		_, err := client.NotificationConfigurations.Verify(ctx, badIdentifier)
		assert.EqualError(t, err, "invalid value for notification configuration ID")
	})

	t.Run("with a valid ID", func(t *testing.T) {
		_, err := client.NotificationConfigurations.Verify(ctx, ncTest.ID)
		require.NoError(t, err)
	})
}
