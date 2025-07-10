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

		assert.Equal(t, 0, ncl.CurrentPage)
		assert.Equal(t, 0, ncl.TotalCount)

		assert.NotNil(t, ncl.Items[0].Subscribable)
		assert.NotEmpty(t, ncl.Items[0].Subscribable)
		assert.NotNil(t, ncl.Items[0].SubscribableChoice.Workspace)
		assert.NotEmpty(t, ncl.Items[0].SubscribableChoice.Workspace)
	})

	t.Run("with list options", func(t *testing.T) {
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

func TestNotificationConfigurationList_forTeams(t *testing.T) {
	skipUnlessBeta(t)
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	t.Cleanup(orgTestCleanup)

	newSubscriptionUpdater(orgTest).WithPlusEntitlementPlan().Update(t)

	tmTest, tmTestCleanup := createTeam(t, client, orgTest)
	t.Cleanup(tmTestCleanup)
	require.NotNil(t, tmTest)

	ncTest1, ncTestCleanup1 := createTeamNotificationConfiguration(t, client, tmTest, nil)
	t.Cleanup(ncTestCleanup1)
	ncTest2, ncTestCleanup2 := createTeamNotificationConfiguration(t, client, tmTest, nil)
	t.Cleanup(ncTestCleanup2)

	t.Run("with a valid team", func(t *testing.T) {
		ncl, err := client.NotificationConfigurations.List(
			ctx,
			tmTest.ID,
			&NotificationConfigurationListOptions{
				SubscribableChoice: &NotificationConfigurationSubscribableChoice{
					Team: tmTest,
				},
			},
		)
		require.NoError(t, err)
		assert.Contains(t, ncl.Items, ncTest1)
		assert.Contains(t, ncl.Items, ncTest2)
	})

	t.Run("without a valid team", func(t *testing.T) {
		ncl, err := client.NotificationConfigurations.List(
			ctx,
			badIdentifier,
			&NotificationConfigurationListOptions{
				SubscribableChoice: &NotificationConfigurationSubscribableChoice{
					Team: tmTest,
				},
			},
		)
		assert.Nil(t, ncl)
		assert.EqualError(t, err, ErrInvalidTeamID.Error())
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
			Triggers:        []NotificationTriggerType{NotificationTriggerCreated},
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
			Triggers:        []NotificationTriggerType{NotificationTriggerCreated},
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
			Triggers:        []NotificationTriggerType{NotificationTriggerCreated},
		}

		nc, err := client.NotificationConfigurations.Create(ctx, wTest.ID, options)
		assert.Nil(t, nc)
		assert.Equal(t, err, ErrRequiredURL)
	})

	t.Run("without a required value URL when destination type is slack", func(t *testing.T) {
		options := NotificationConfigurationCreateOptions{
			DestinationType: NotificationDestination(NotificationDestinationTypeSlack),
			Enabled:         Bool(false),
			Name:            String(randomString(t)),
			Triggers:        []NotificationTriggerType{NotificationTriggerCreated},
		}

		nc, err := client.NotificationConfigurations.Create(ctx, wTest.ID, options)
		assert.Nil(t, nc)
		assert.Equal(t, err, ErrRequiredURL)
	})

	t.Run("without a required value URL when destination type is MS Teams", func(t *testing.T) {
		options := NotificationConfigurationCreateOptions{
			DestinationType: NotificationDestination(NotificationDestinationTypeMicrosoftTeams),
			Enabled:         Bool(false),
			Name:            String(randomString(t)),
			Triggers:        []NotificationTriggerType{NotificationTriggerCreated},
		}

		nc, err := client.NotificationConfigurations.Create(ctx, wTest.ID, options)
		assert.Nil(t, nc)
		assert.Equal(t, err, ErrRequiredURL)
	})

	t.Run("without a valid workspace", func(t *testing.T) {
		nc, err := client.NotificationConfigurations.Create(ctx, badIdentifier, NotificationConfigurationCreateOptions{})
		assert.Nil(t, nc)
		assert.EqualError(t, err, ErrInvalidWorkspaceID.Error())
	})

	t.Run("with an invalid notification trigger", func(t *testing.T) {
		options := NotificationConfigurationCreateOptions{
			DestinationType: NotificationDestination(NotificationDestinationTypeGeneric),
			Enabled:         Bool(false),
			Name:            String(randomString(t)),
			Token:           String(randomString(t)),
			URL:             String("http://example.com"),
			Triggers:        []NotificationTriggerType{"the beacons of gondor are lit"},
		}

		nc, err := client.NotificationConfigurations.Create(ctx, wTest.ID, options)
		assert.Nil(t, nc)
		assert.EqualError(t, err, ErrInvalidNotificationTrigger.Error())
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

func TestNotificationConfigurationsCreate_byType(t *testing.T) {
	t.Parallel()

	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	t.Cleanup(orgTestCleanup)

	upgradeOrganizationSubscription(t, client, orgTest)

	wTest, wTestCleanup := createWorkspace(t, client, orgTest)
	t.Cleanup(wTestCleanup)

	// Create user to use when testing email destination type
	orgMemberTest, orgMemberTestCleanup := createOrganizationMembership(t, client, orgTest)
	t.Cleanup(orgMemberTestCleanup)

	orgMemberTest.User = &User{ID: orgMemberTest.User.ID}

	testCases := []NotificationTriggerType{
		NotificationTriggerCreated,
		NotificationTriggerPlanning,
		NotificationTriggerNeedsAttention,
		NotificationTriggerApplying,
		NotificationTriggerCompleted,
		NotificationTriggerErrored,
		NotificationTriggerAssessmentDrifted,
		NotificationTriggerAssessmentFailed,
		NotificationTriggerAssessmentCheckFailed,
		NotificationTriggerWorkspaceAutoDestroyReminder,
		NotificationTriggerWorkspaceAutoDestroyRunResults,
	}

	for _, trigger := range testCases {
		message := fmt.Sprintf("with trigger %s and all required values", trigger)

		t.Run(message, func(t *testing.T) {
			t.Parallel()
			options := NotificationConfigurationCreateOptions{
				DestinationType: NotificationDestination(NotificationDestinationTypeGeneric),
				Enabled:         Bool(false),
				Name:            String(randomString(t)),
				Token:           String(randomString(t)),
				URL:             String("http://example.com"),
				Triggers:        []NotificationTriggerType{trigger},
			}

			_, err := client.NotificationConfigurations.Create(ctx, wTest.ID, options)
			require.NoError(t, err)
		})
	}
}

func TestNotificationConfigurationCreate_forTeams(t *testing.T) {
	skipUnlessBeta(t)
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	t.Cleanup(orgTestCleanup)

	newSubscriptionUpdater(orgTest).WithPlusEntitlementPlan().Update(t)

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
		options := NotificationConfigurationCreateOptions{
			DestinationType:    NotificationDestination(NotificationDestinationTypeGeneric),
			Enabled:            Bool(false),
			Name:               String(randomString(t)),
			Token:              String(randomString(t)),
			URL:                String("http://example.com"),
			Triggers:           []NotificationTriggerType{NotificationTriggerChangeRequestCreated},
			SubscribableChoice: &NotificationConfigurationSubscribableChoice{Team: tmTest},
		}
		nc, err := client.NotificationConfigurations.Create(ctx, tmTest.ID, options)

		require.NoError(t, err)
		require.NotNil(t, nc)
	})

	t.Run("without a required value", func(t *testing.T) {
		options := NotificationConfigurationCreateOptions{
			DestinationType:    NotificationDestination(NotificationDestinationTypeGeneric),
			Enabled:            Bool(false),
			Token:              String(randomString(t)),
			URL:                String("http://example.com"),
			Triggers:           []NotificationTriggerType{NotificationTriggerChangeRequestCreated},
			SubscribableChoice: &NotificationConfigurationSubscribableChoice{Team: tmTest},
		}
		nc, err := client.NotificationConfigurations.Create(ctx, tmTest.ID, options)

		assert.Nil(t, nc)
		assert.EqualError(t, err, ErrRequiredName.Error())
	})

	t.Run("without a required value URL when destination type is generic", func(t *testing.T) {
		options := NotificationConfigurationCreateOptions{
			DestinationType:    NotificationDestination(NotificationDestinationTypeGeneric),
			Enabled:            Bool(false),
			Name:               String(randomString(t)),
			Token:              String(randomString(t)),
			Triggers:           []NotificationTriggerType{NotificationTriggerChangeRequestCreated},
			SubscribableChoice: &NotificationConfigurationSubscribableChoice{Team: tmTest},
		}

		nc, err := client.NotificationConfigurations.Create(ctx, tmTest.ID, options)
		assert.Nil(t, nc)
		assert.Equal(t, err, ErrRequiredURL)
	})

	t.Run("without a required value URL when destination type is slack", func(t *testing.T) {
		options := NotificationConfigurationCreateOptions{
			DestinationType:    NotificationDestination(NotificationDestinationTypeSlack),
			Enabled:            Bool(false),
			Name:               String(randomString(t)),
			Triggers:           []NotificationTriggerType{NotificationTriggerChangeRequestCreated},
			SubscribableChoice: &NotificationConfigurationSubscribableChoice{Team: tmTest},
		}

		nc, err := client.NotificationConfigurations.Create(ctx, tmTest.ID, options)
		assert.Nil(t, nc)
		assert.Equal(t, err, ErrRequiredURL)
	})

	t.Run("without a required value URL when destination type is MS Teams", func(t *testing.T) {
		options := NotificationConfigurationCreateOptions{
			DestinationType:    NotificationDestination(NotificationDestinationTypeMicrosoftTeams),
			Enabled:            Bool(false),
			Name:               String(randomString(t)),
			Triggers:           []NotificationTriggerType{NotificationTriggerChangeRequestCreated},
			SubscribableChoice: &NotificationConfigurationSubscribableChoice{Team: tmTest},
		}

		nc, err := client.NotificationConfigurations.Create(ctx, tmTest.ID, options)
		assert.Nil(t, nc)
		assert.Equal(t, err, ErrRequiredURL)
	})

	t.Run("without a valid team", func(t *testing.T) {
		nc, err := client.NotificationConfigurations.Create(ctx, badIdentifier, NotificationConfigurationCreateOptions{
			SubscribableChoice: &NotificationConfigurationSubscribableChoice{
				Team: tmTest,
			},
		})
		assert.Nil(t, nc)
		assert.EqualError(t, err, ErrInvalidTeamID.Error())
	})

	t.Run("with an invalid notification trigger", func(t *testing.T) {
		options := NotificationConfigurationCreateOptions{
			DestinationType:    NotificationDestination(NotificationDestinationTypeGeneric),
			Enabled:            Bool(false),
			Name:               String(randomString(t)),
			Token:              String(randomString(t)),
			URL:                String("http://example.com"),
			Triggers:           []NotificationTriggerType{"the beacons of gondor are lit"},
			SubscribableChoice: &NotificationConfigurationSubscribableChoice{Team: tmTest},
		}

		nc, err := client.NotificationConfigurations.Create(ctx, tmTest.ID, options)
		assert.Nil(t, nc)
		assert.EqualError(t, err, ErrInvalidNotificationTrigger.Error())
	})

	t.Run("with email users when destination type is email", func(t *testing.T) {
		options := NotificationConfigurationCreateOptions{
			DestinationType:    NotificationDestination(NotificationDestinationTypeEmail),
			Enabled:            Bool(false),
			Name:               String(randomString(t)),
			EmailUsers:         []*User{orgMemberTest.User},
			SubscribableChoice: &NotificationConfigurationSubscribableChoice{Team: tmTest},
		}

		_, err := client.NotificationConfigurations.Create(ctx, tmTest.ID, options)
		require.NoError(t, err)
	})

	t.Run("without email users when destination type is email", func(t *testing.T) {
		options := NotificationConfigurationCreateOptions{
			DestinationType:    NotificationDestination(NotificationDestinationTypeEmail),
			Enabled:            Bool(false),
			Name:               String(randomString(t)),
			SubscribableChoice: &NotificationConfigurationSubscribableChoice{Team: tmTest},
		}

		_, err := client.NotificationConfigurations.Create(ctx, tmTest.ID, options)
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
		assert.Equal(t, err, ErrInvalidNotificationConfigID)
	})
}

func TestNotificationConfigurationRead_forTeams(t *testing.T) {
	skipUnlessBeta(t)
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	t.Cleanup(orgTestCleanup)

	newSubscriptionUpdater(orgTest).WithPlusEntitlementPlan().Update(t)

	tmTest, tmTestCleanup := createTeam(t, client, orgTest)
	t.Cleanup(tmTestCleanup)

	ncTest, ncTestCleanup := createTeamNotificationConfiguration(t, client, tmTest, nil)
	t.Cleanup(ncTestCleanup)

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
		assert.Equal(t, err, ErrInvalidNotificationConfigID)
	})
}

func TestNotificationConfigurationUpdate_forTeams(t *testing.T) {
	skipUnlessBeta(t)
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	t.Cleanup(orgTestCleanup)

	newSubscriptionUpdater(orgTest).WithPlusEntitlementPlan().Update(t)

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

	options := &NotificationConfigurationCreateOptions{
		DestinationType:    NotificationDestination(NotificationDestinationTypeEmail),
		Enabled:            Bool(false),
		Name:               String(randomString(t)),
		EmailUsers:         []*User{orgMemberTest1.User},
		SubscribableChoice: &NotificationConfigurationSubscribableChoice{Team: tmTest},
	}
	ncEmailTest, ncEmailTestCleanup := createTeamNotificationConfiguration(t, client, tmTest, options)
	t.Cleanup(ncEmailTestCleanup)

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

	t.Run("with invalid notification trigger", func(t *testing.T) {
		options := NotificationConfigurationUpdateOptions{
			Triggers: []NotificationTriggerType{"fly you fools!"},
		}

		nc, err := client.NotificationConfigurations.Update(ctx, ncTest.ID, options)
		assert.Nil(t, nc)
		assert.EqualError(t, err, ErrInvalidNotificationTrigger.Error())
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
		assert.Equal(t, err, ErrInvalidNotificationConfigID)
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

	t.Run("with invalid notification trigger", func(t *testing.T) {
		options := NotificationConfigurationUpdateOptions{
			Triggers: []NotificationTriggerType{"fly you fools!"},
		}

		nc, err := client.NotificationConfigurations.Update(ctx, ncTest.ID, options)
		assert.Nil(t, nc)
		assert.EqualError(t, err, ErrInvalidNotificationTrigger.Error())
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
		assert.Equal(t, err, ErrInvalidNotificationConfigID)
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
		assert.Equal(t, err, ErrInvalidNotificationConfigID)
	})
}

func TestNotificationConfigurationDelete_forTeams(t *testing.T) {
	skipUnlessBeta(t)
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	t.Cleanup(orgTestCleanup)

	newSubscriptionUpdater(orgTest).WithPlusEntitlementPlan().Update(t)

	tmTest, tmTestCleanup := createTeam(t, client, orgTest)
	t.Cleanup(tmTestCleanup)

	ncTest, _ := createTeamNotificationConfiguration(t, client, tmTest, nil)

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
		assert.Equal(t, err, ErrInvalidNotificationConfigID)
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
		assert.Equal(t, err, ErrInvalidNotificationConfigID)
	})
}

func TestNotificationConfigurationVerify_forTeams(t *testing.T) {
	skipUnlessBeta(t)
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	t.Cleanup(orgTestCleanup)

	newSubscriptionUpdater(orgTest).WithPlusEntitlementPlan().Update(t)

	tmTest, tmTestCleanup := createTeam(t, client, orgTest)
	t.Cleanup(tmTestCleanup)

	ncTest, ncTestCleanup := createTeamNotificationConfiguration(t, client, tmTest, nil)
	t.Cleanup(ncTestCleanup)

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
		assert.Equal(t, err, ErrInvalidNotificationConfigID)
	})
}
