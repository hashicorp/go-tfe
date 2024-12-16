// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package tfe

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTeamNotificationConfigurationList(t *testing.T) {
	skipUnlessBeta(t)
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	t.Cleanup(orgTestCleanup)

	newSubscriptionUpdater(orgTest).WithPlusPlan().Update(t)

	tmTest, tmTestCleanup := createTeam(t, client, orgTest)
	t.Cleanup(tmTestCleanup)
	require.NotNil(t, tmTest)

	ncTest1, ncTestCleanup1 := createTeamNotificationConfiguration(t, client, tmTest, nil)
	t.Cleanup(ncTestCleanup1)
	ncTest2, ncTestCleanup2 := createTeamNotificationConfiguration(t, client, tmTest, nil)
	t.Cleanup(ncTestCleanup2)

	t.Run("with a valid team", func(t *testing.T) {
		ncl, err := client.TeamNotificationConfigurations.List(
			ctx,
			tmTest.ID,
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
		ncl, err := client.TeamNotificationConfigurations.List(
			ctx,
			tmTest.ID,
			&TeamNotificationConfigurationListOptions{
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

	t.Run("without a valid team", func(t *testing.T) {
		ncl, err := client.TeamNotificationConfigurations.List(
			ctx,
			badIdentifier,
			nil,
		)
		assert.Nil(t, ncl)
		assert.EqualError(t, err, ErrInvalidTeamID.Error())
	})
}

func TestTeamNotificationConfigurationCreate(t *testing.T) {
	skipUnlessBeta(t)
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	t.Cleanup(orgTestCleanup)

	newSubscriptionUpdater(orgTest).WithPlusPlan().Update(t)

	tmTest, tmTestCleanup := createTeam(t, client, orgTest)
	t.Cleanup(tmTestCleanup)

	// Create user to use when testing email destination type
	orgMemberTest, orgMemberTestCleanup := createOrganizationMembership(t, client, orgTest)
	t.Cleanup(orgMemberTestCleanup)

	// Add user to team
	options := TeamMemberAddOptions{
		OrganizationMembershipIDs: []string{orgMemberTest.ID},
	}
	err := client.TeamMembers.Add(ctx, tmTest.ID, options)
	require.NoError(t, err)

	t.Run("with all required values", func(t *testing.T) {
		options := TeamNotificationConfigurationCreateOptions{
			DestinationType: NotificationDestination(NotificationDestinationTypeGeneric),
			Enabled:         Bool(false),
			Name:            String(randomString(t)),
			Token:           String(randomString(t)),
			URL:             String("http://example.com"),
			Triggers:        []NotificationTriggerType{NotificationTriggerChangeRequestCreated},
		}

		_, err := client.TeamNotificationConfigurations.Create(ctx, tmTest.ID, options)
		require.NoError(t, err)
	})

	t.Run("without a required value", func(t *testing.T) {
		options := TeamNotificationConfigurationCreateOptions{
			DestinationType: NotificationDestination(NotificationDestinationTypeGeneric),
			Enabled:         Bool(false),
			Token:           String(randomString(t)),
			URL:             String("http://example.com"),
			Triggers:        []NotificationTriggerType{NotificationTriggerChangeRequestCreated},
		}

		nc, err := client.TeamNotificationConfigurations.Create(ctx, tmTest.ID, options)
		assert.Nil(t, nc)
		assert.EqualError(t, err, ErrRequiredName.Error())
	})

	t.Run("without a required value URL when destination type is generic", func(t *testing.T) {
		options := TeamNotificationConfigurationCreateOptions{
			DestinationType: NotificationDestination(NotificationDestinationTypeGeneric),
			Enabled:         Bool(false),
			Name:            String(randomString(t)),
			Token:           String(randomString(t)),
			Triggers:        []NotificationTriggerType{NotificationTriggerChangeRequestCreated},
		}

		nc, err := client.TeamNotificationConfigurations.Create(ctx, tmTest.ID, options)
		assert.Nil(t, nc)
		assert.Equal(t, err, ErrRequiredURL)
	})

	t.Run("without a required value URL when destination type is slack", func(t *testing.T) {
		options := TeamNotificationConfigurationCreateOptions{
			DestinationType: NotificationDestination(NotificationDestinationTypeSlack),
			Enabled:         Bool(false),
			Name:            String(randomString(t)),
			Triggers:        []NotificationTriggerType{NotificationTriggerChangeRequestCreated},
		}

		nc, err := client.TeamNotificationConfigurations.Create(ctx, tmTest.ID, options)
		assert.Nil(t, nc)
		assert.Equal(t, err, ErrRequiredURL)
	})

	t.Run("without a required value URL when destination type is MS Teams", func(t *testing.T) {
		options := TeamNotificationConfigurationCreateOptions{
			DestinationType: NotificationDestination(NotificationDestinationTypeMicrosoftTeams),
			Enabled:         Bool(false),
			Name:            String(randomString(t)),
			Triggers:        []NotificationTriggerType{NotificationTriggerChangeRequestCreated},
		}

		nc, err := client.TeamNotificationConfigurations.Create(ctx, tmTest.ID, options)
		assert.Nil(t, nc)
		assert.Equal(t, err, ErrRequiredURL)
	})

	t.Run("without a valid team", func(t *testing.T) {
		nc, err := client.TeamNotificationConfigurations.Create(ctx, badIdentifier, TeamNotificationConfigurationCreateOptions{})
		assert.Nil(t, nc)
		assert.EqualError(t, err, ErrInvalidTeamID.Error())
	})

	t.Run("with an invalid notification trigger", func(t *testing.T) {
		options := TeamNotificationConfigurationCreateOptions{
			DestinationType: NotificationDestination(NotificationDestinationTypeGeneric),
			Enabled:         Bool(false),
			Name:            String(randomString(t)),
			Token:           String(randomString(t)),
			URL:             String("http://example.com"),
			Triggers:        []NotificationTriggerType{"the beacons of gondor are lit"},
		}

		nc, err := client.TeamNotificationConfigurations.Create(ctx, tmTest.ID, options)
		assert.Nil(t, nc)
		assert.EqualError(t, err, ErrInvalidNotificationTrigger.Error())
	})

	t.Run("with email users when destination type is email", func(t *testing.T) {
		options := TeamNotificationConfigurationCreateOptions{
			DestinationType: NotificationDestination(NotificationDestinationTypeEmail),
			Enabled:         Bool(false),
			Name:            String(randomString(t)),
			EmailUsers:      []*User{orgMemberTest.User},
		}

		_, err := client.TeamNotificationConfigurations.Create(ctx, tmTest.ID, options)
		require.NoError(t, err)
	})

	t.Run("without email users when destination type is email", func(t *testing.T) {
		options := TeamNotificationConfigurationCreateOptions{
			DestinationType: NotificationDestination(NotificationDestinationTypeEmail),
			Enabled:         Bool(false),
			Name:            String(randomString(t)),
		}

		_, err := client.TeamNotificationConfigurations.Create(ctx, tmTest.ID, options)
		require.NoError(t, err)
	})
}

func TestTeamNotificationConfigurationsCreate_byType(t *testing.T) {
	skipUnlessBeta(t)
	t.Parallel()

	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	t.Cleanup(orgTestCleanup)

	newSubscriptionUpdater(orgTest).WithPlusPlan().Update(t)

	tmTest, tmTestCleanup := createTeam(t, client, orgTest)
	t.Cleanup(tmTestCleanup)

	testCases := []NotificationTriggerType{
		NotificationTriggerChangeRequestCreated,
	}

	for _, trigger := range testCases {
		trigger := trigger
		message := fmt.Sprintf("with trigger %s and all required values", trigger)

		t.Run(message, func(t *testing.T) {
			t.Parallel()
			options := TeamNotificationConfigurationCreateOptions{
				DestinationType: NotificationDestination(NotificationDestinationTypeGeneric),
				Enabled:         Bool(false),
				Name:            String(randomString(t)),
				Token:           String(randomString(t)),
				URL:             String("http://example.com"),
				Triggers:        []NotificationTriggerType{trigger},
			}

			_, err := client.TeamNotificationConfigurations.Create(ctx, tmTest.ID, options)
			require.NoError(t, err)
		})
	}
}

func TestTeamNotificationConfigurationRead(t *testing.T) {
	skipUnlessBeta(t)
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	t.Cleanup(orgTestCleanup)

	newSubscriptionUpdater(orgTest).WithPlusPlan().Update(t)

	tmTest, tmTestCleanup := createTeam(t, client, orgTest)
	t.Cleanup(tmTestCleanup)

	ncTest, ncTestCleanup := createTeamNotificationConfiguration(t, client, tmTest, nil)
	t.Cleanup(ncTestCleanup)

	t.Run("with a valid ID", func(t *testing.T) {
		nc, err := client.TeamNotificationConfigurations.Read(ctx, ncTest.ID)
		require.NoError(t, err)
		assert.Equal(t, ncTest.ID, nc.ID)
	})

	t.Run("when the notification configuration does not exist", func(t *testing.T) {
		_, err := client.TeamNotificationConfigurations.Read(ctx, "nonexisting")
		assert.Equal(t, err, ErrResourceNotFound)
	})

	t.Run("when the notification configuration ID is invalid", func(t *testing.T) {
		_, err := client.TeamNotificationConfigurations.Read(ctx, badIdentifier)
		assert.Equal(t, err, ErrInvalidNotificationConfigID)
	})
}

func TestTeamNotificationConfigurationUpdate(t *testing.T) {
	skipUnlessBeta(t)
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	t.Cleanup(orgTestCleanup)

	newSubscriptionUpdater(orgTest).WithPlusPlan().Update(t)

	tmTest, tmTestCleanup := createTeam(t, client, orgTest)
	t.Cleanup(tmTestCleanup)

	ncTest, ncTestCleanup := createTeamNotificationConfiguration(t, client, tmTest, nil)
	t.Cleanup(ncTestCleanup)

	// Create users to use when testing email destination type
	orgMemberTest1, orgMemberTest1Cleanup := createOrganizationMembership(t, client, orgTest)
	defer orgMemberTest1Cleanup()
	orgMemberTest2, orgMemberTest2Cleanup := createOrganizationMembership(t, client, orgTest)
	defer orgMemberTest2Cleanup()

	orgMemberTest1.User = &User{ID: orgMemberTest1.User.ID}
	orgMemberTest2.User = &User{ID: orgMemberTest2.User.ID}

	// Add users to team
	for _, orgMember := range []*OrganizationMembership{orgMemberTest1, orgMemberTest2} {
		options := TeamMemberAddOptions{
			OrganizationMembershipIDs: []string{orgMember.ID},
		}
		err := client.TeamMembers.Add(ctx, tmTest.ID, options)
		require.NoError(t, err)
	}

	options := &TeamNotificationConfigurationCreateOptions{
		DestinationType: NotificationDestination(NotificationDestinationTypeEmail),
		Enabled:         Bool(false),
		Name:            String(randomString(t)),
		EmailUsers:      []*User{orgMemberTest1.User},
	}
	ncEmailTest, ncEmailTestCleanup := createTeamNotificationConfiguration(t, client, tmTest, options)
	t.Cleanup(ncEmailTestCleanup)

	t.Run("with options", func(t *testing.T) {
		options := TeamNotificationConfigurationUpdateOptions{
			Enabled: Bool(true),
			Name:    String("newName"),
		}

		nc, err := client.TeamNotificationConfigurations.Update(ctx, ncTest.ID, options)
		require.NoError(t, err)
		assert.Equal(t, nc.Enabled, true)
		assert.Equal(t, nc.Name, "newName")
	})

	t.Run("with invalid notification trigger", func(t *testing.T) {
		options := TeamNotificationConfigurationUpdateOptions{
			Triggers: []NotificationTriggerType{"fly you fools!"},
		}

		nc, err := client.TeamNotificationConfigurations.Update(ctx, ncTest.ID, options)
		assert.Nil(t, nc)
		assert.EqualError(t, err, ErrInvalidNotificationTrigger.Error())
	})

	t.Run("with email users when destination type is email", func(t *testing.T) {
		options := TeamNotificationConfigurationUpdateOptions{
			Enabled:    Bool(true),
			Name:       String("newName"),
			EmailUsers: []*User{orgMemberTest1.User, orgMemberTest2.User},
		}

		nc, err := client.TeamNotificationConfigurations.Update(ctx, ncEmailTest.ID, options)
		require.NoError(t, err)
		assert.Equal(t, nc.Enabled, true)
		assert.Equal(t, nc.Name, "newName")
		assert.Contains(t, nc.EmailUsers, orgMemberTest1.User)
		assert.Contains(t, nc.EmailUsers, orgMemberTest2.User)
	})

	t.Run("without email users when destination type is email", func(t *testing.T) {
		options := TeamNotificationConfigurationUpdateOptions{
			Enabled: Bool(true),
			Name:    String("newName"),
		}

		nc, err := client.TeamNotificationConfigurations.Update(ctx, ncEmailTest.ID, options)
		require.NoError(t, err)
		assert.Equal(t, nc.Enabled, true)
		assert.Equal(t, nc.Name, "newName")
		assert.Empty(t, nc.EmailUsers)
	})

	t.Run("without options", func(t *testing.T) {
		_, err := client.TeamNotificationConfigurations.Update(ctx, ncTest.ID, TeamNotificationConfigurationUpdateOptions{})
		require.NoError(t, err)
	})

	t.Run("when the notification configuration does not exist", func(t *testing.T) {
		_, err := client.TeamNotificationConfigurations.Update(ctx, "nonexisting", TeamNotificationConfigurationUpdateOptions{})
		assert.Equal(t, err, ErrResourceNotFound)
	})

	t.Run("when the notification configuration ID is invalid", func(t *testing.T) {
		_, err := client.TeamNotificationConfigurations.Update(ctx, badIdentifier, TeamNotificationConfigurationUpdateOptions{})
		assert.Equal(t, err, ErrInvalidNotificationConfigID)
	})
}

func TestTeamNotificationConfigurationDelete(t *testing.T) {
	skipUnlessBeta(t)
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	t.Cleanup(orgTestCleanup)

	newSubscriptionUpdater(orgTest).WithPlusPlan().Update(t)

	tmTest, tmTestCleanup := createTeam(t, client, orgTest)
	t.Cleanup(tmTestCleanup)

	ncTest, _ := createTeamNotificationConfiguration(t, client, tmTest, nil)

	t.Run("with a valid ID", func(t *testing.T) {
		err := client.TeamNotificationConfigurations.Delete(ctx, ncTest.ID)
		require.NoError(t, err)

		_, err = client.TeamNotificationConfigurations.Read(ctx, ncTest.ID)
		assert.Equal(t, err, ErrResourceNotFound)
	})

	t.Run("when the notification configuration does not exist", func(t *testing.T) {
		err := client.TeamNotificationConfigurations.Delete(ctx, "nonexisting")
		assert.Equal(t, err, ErrResourceNotFound)
	})

	t.Run("when the notification configuration ID is invalid", func(t *testing.T) {
		err := client.TeamNotificationConfigurations.Delete(ctx, badIdentifier)
		assert.Equal(t, err, ErrInvalidNotificationConfigID)
	})
}

func TestTeamNotificationConfigurationVerify(t *testing.T) {
	skipUnlessBeta(t)
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	t.Cleanup(orgTestCleanup)

	newSubscriptionUpdater(orgTest).WithPlusPlan().Update(t)

	tmTest, tmTestCleanup := createTeam(t, client, orgTest)
	t.Cleanup(tmTestCleanup)

	ncTest, ncTestCleanup := createTeamNotificationConfiguration(t, client, tmTest, nil)
	t.Cleanup(ncTestCleanup)

	t.Run("with a valid ID", func(t *testing.T) {
		_, err := client.TeamNotificationConfigurations.Verify(ctx, ncTest.ID)
		require.NoError(t, err)
	})

	t.Run("when the notification configuration does not exists", func(t *testing.T) {
		_, err := client.TeamNotificationConfigurations.Verify(ctx, "nonexisting")
		assert.Equal(t, err, ErrResourceNotFound)
	})

	t.Run("when the notification configuration ID is invalid", func(t *testing.T) {
		_, err := client.TeamNotificationConfigurations.Verify(ctx, badIdentifier)
		assert.Equal(t, err, ErrInvalidNotificationConfigID)
	})
}
