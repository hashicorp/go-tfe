// Copyright IBM Corp. 2018, 2026
// SPDX-License-Identifier: MPL-2.0

package tfe

import (
	"context"
	"fmt"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// scimGroupMappingDelay throttles SCIM group-mapping Create/Update/Delete calls to avoid 429s
// 1.8s was chosen empirically (trial-and-error) as an optimal stable value.
const scimGroupMappingDelay = 1800 * time.Millisecond

func TestAdminSCIMGroupMappings_Create(t *testing.T) {
	skipUnlessEnterprise(t)
	client := testClient(t)
	ctx := context.Background()

	enableSCIM(ctx, t, client, true)
	t.Cleanup(func() { enableSCIM(ctx, t, client, false) })

	scimClient, scimGroups := setupSCIMGroups(ctx, t, client)

	testcases := []struct {
		name string
		// groupFor returns the SCIM group to link for the team at index i.
		groupFor func(i int) AdminSCIMGroup
	}{
		{
			name:     "one team to one group",
			groupFor: func(i int) AdminSCIMGroup { return scimGroups[i] },
		},
		{
			name:     "multiple teams to one group",
			groupFor: func(_ int) AdminSCIMGroup { return scimGroups[0] },
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			for i, team := range createTeams(t, client, 2) {
				group := tc.groupFor(i)
				linkSCIMGroupMapping(ctx, t, scimClient, team.ID, group.ID)
				scimAttr := getScimAttributeValues(ctx, t, client, team.ID)
				assert.True(t, scimAttr.SCIMLinked, "Expected SCIMLinked to be true after creating mapping")
				assert.Equal(t, group.Name, scimAttr.SCIMGroupName, "Expected SCIMGroupName to match the linked group")
				assert.False(t, scimAttr.SCIMSyncPaused, "Expected SCIMSyncPaused to be false after creating mapping")
			}
		})
	}

	t.Run("re-link after delete on same team", func(t *testing.T) {
		teamID := createSingleTeam(t, client)
		require.NoError(t, createSCIMGroupMapping(ctx, scimClient, teamID, scimGroups[0].ID))
		require.NoError(t, deleteSCIMGroupMapping(ctx, scimClient, teamID))
		linkSCIMGroupMapping(ctx, t, scimClient, teamID, scimGroups[1].ID)

		scimAttr := getScimAttributeValues(ctx, t, client, teamID)
		assert.True(t, scimAttr.SCIMLinked, "Expected SCIMLinked to be true after re-linking")
		assert.Equal(t, scimGroups[1].Name, scimAttr.SCIMGroupName, "Expected SCIMGroupName to match the re-linked group")
	})

	errorCases := []struct {
		name string
		// setup returns the teamID and scimGroupID to use for the Create call.
		setup func(t *testing.T) (teamID, scimGroupID string)
		// nilOptions, when true, calls Create directly with nil options instead of
		// going through the createSCIMGroupMapping wrapper.
		nilOptions  bool
		expectedErr error
	}{
		{
			name: "team already mapped",
			setup: func(t *testing.T) (string, string) {
				teamID := createSingleTeam(t, client)
				linkSCIMGroupMapping(ctx, t, scimClient, teamID, scimGroups[0].ID)
				return teamID, scimGroups[1].ID
			},
			expectedErr: ErrSCIMTeamAlreadyMapped,
		},
		{
			name: "link same team to same group twice",
			setup: func(t *testing.T) (string, string) {
				teamID := createSingleTeam(t, client)
				linkSCIMGroupMapping(ctx, t, scimClient, teamID, scimGroups[0].ID)
				return teamID, scimGroups[0].ID
			},
			expectedErr: ErrSCIMTeamAlreadyMapped,
		},
		{
			name: "non-existent SCIM group",
			setup: func(t *testing.T) (string, string) {
				return createSingleTeam(t, client), "this-scim-group-does-not-exist"
			},
			expectedErr: ErrResourceNotFound,
		},
		{
			name: "non-existent team",
			setup: func(_ *testing.T) (string, string) {
				return "this-team-does-not-exist", scimGroups[0].ID
			},
			expectedErr: ErrResourceNotFound,
		},
		{
			name: "owners team not allowed",
			setup: func(t *testing.T) (string, string) {
				org, orgCleanup := createOrganization(t, client)
				t.Cleanup(orgCleanup)
				teamList, err := client.Teams.List(ctx, org.Name, &TeamListOptions{Query: "owners"})
				require.NoError(t, err)
				return teamList.Items[0].ID, scimGroups[0].ID
			},
			expectedErr: ErrSCIMGroupMappingOwnersTeam,
		},
		{
			name: "site admin group not allowed",
			setup: func(t *testing.T) (string, string) {
				siteAdminGroupID := scimGroups[0].ID
				teamID := createSingleTeam(t, client)
				_, err := scimClient.Update(ctx, AdminSCIMSettingUpdateOptions{SiteAdminGroupSCIMID: &siteAdminGroupID})
				require.NoError(t, err, "Failed to set site admin group")
				t.Cleanup(func() {
					_, err := scimClient.Update(ctx, AdminSCIMSettingUpdateOptions{SiteAdminGroupSCIMID: nil})
					require.NoError(t, err, "Failed to clear site admin group")
				})
				return teamID, siteAdminGroupID
			},
			expectedErr: ErrSCIMGroupMappingSiteAdminGroup,
		},
		{
			name: "invalid team ID",
			setup: func(_ *testing.T) (string, string) {
				return "this is an invalid team id", scimGroups[0].ID
			},
			expectedErr: ErrInvalidTeamID,
		},
		{
			name: "invalid SCIM group ID",
			setup: func(t *testing.T) (string, string) {
				return createSingleTeam(t, client), "this is an invalid scim group id"
			},
			expectedErr: ErrInvalidSCIMGroupID,
		},
		{
			name: "nil options",
			setup: func(t *testing.T) (string, string) {
				return createSingleTeam(t, client), ""
			},
			nilOptions:  true,
			expectedErr: ErrRequiredSCIMGroupMappingCreateOps,
		},
	}

	for _, tc := range errorCases {
		t.Run(tc.name, func(t *testing.T) {
			teamID, scimGroupID := tc.setup(t)
			var err error
			if tc.nilOptions {
				err = scimClient.SCIMGroupMappings.Create(ctx, teamID, nil)
			} else {
				err = createSCIMGroupMapping(ctx, scimClient, teamID, scimGroupID)
			}
			require.EqualError(t, err, tc.expectedErr.Error())
		})
	}
}

func TestAdminSCIMGroupMappings_Update(t *testing.T) {
	skipUnlessEnterprise(t)
	client := testClient(t)
	ctx := context.Background()

	enableSCIM(ctx, t, client, true)
	t.Cleanup(func() { enableSCIM(ctx, t, client, false) })

	scimClient, scimGroups := setupSCIMGroups(ctx, t, client)

	// linkedTeamSetup creates a team linked to scim group and returns its ID.
	linkedTeamSetup := func(t *testing.T) string {
		teamID := createSingleTeam(t, client)
		linkSCIMGroupMapping(ctx, t, scimClient, teamID, scimGroups[0].ID)
		return teamID
	}

	testcases := []struct {
		name string
		// setup returns the teamID to update.
		setup       func(t *testing.T) string
		options     *AdminSCIMGroupMappingUpdateOptions
		shouldError bool
		expectedErr error
		// assertAfter runs only when shouldError is false, to verify post-update state.
		assertAfter func(t *testing.T, teamID string)
	}{
		{
			name:    "pause sync",
			setup:   linkedTeamSetup,
			options: &AdminSCIMGroupMappingUpdateOptions{SCIMSyncPaused: Bool(true)},
			assertAfter: func(t *testing.T, teamID string) {
				scimAttr := getScimAttributeValues(ctx, t, client, teamID)
				assert.True(t, scimAttr.SCIMSyncPaused, "Expected SCIMSyncPaused to be true after pausing sync")
			},
		},
		{
			name: "unpause sync",
			setup: func(t *testing.T) string {
				teamID := linkedTeamSetup(t)
				require.NoError(t, updateSCIMGroupMapping(ctx, scimClient, teamID, &AdminSCIMGroupMappingUpdateOptions{SCIMSyncPaused: Bool(true)}))
				return teamID
			},
			options: &AdminSCIMGroupMappingUpdateOptions{SCIMSyncPaused: Bool(false)},
			assertAfter: func(t *testing.T, teamID string) {
				scimAttr := getScimAttributeValues(ctx, t, client, teamID)
				assert.False(t, scimAttr.SCIMSyncPaused, "Expected SCIMSyncPaused to be false after unpausing sync")
			},
		},
		{
			name:        "team not linked",
			setup:       func(t *testing.T) string { return createSingleTeam(t, client) },
			options:     &AdminSCIMGroupMappingUpdateOptions{SCIMSyncPaused: Bool(true)},
			shouldError: true,
			expectedErr: ErrSCIMGroupMappingTeamNotLinked,
		},
		{
			name:        "invalid team ID",
			setup:       func(_ *testing.T) string { return "this is an invalid team id" },
			options:     &AdminSCIMGroupMappingUpdateOptions{SCIMSyncPaused: Bool(true)},
			shouldError: true,
			expectedErr: ErrInvalidTeamID,
		},
		{
			name:        "nil SCIMSyncPaused",
			setup:       func(t *testing.T) string { return createSingleTeam(t, client) },
			options:     &AdminSCIMGroupMappingUpdateOptions{SCIMSyncPaused: nil},
			shouldError: true,
			expectedErr: ErrSCIMSyncPausedNil,
		},
		{
			name:        "non-existent team",
			setup:       func(_ *testing.T) string { return "this-team-does-not-exist" },
			options:     &AdminSCIMGroupMappingUpdateOptions{SCIMSyncPaused: Bool(true)},
			shouldError: true,
			expectedErr: ErrResourceNotFound,
		},
		{
			name: "idempotent pause - already paused",
			setup: func(t *testing.T) string {
				teamID := linkedTeamSetup(t)
				require.NoError(t, updateSCIMGroupMapping(ctx, scimClient, teamID, &AdminSCIMGroupMappingUpdateOptions{SCIMSyncPaused: Bool(true)}))
				return teamID
			},
			options: &AdminSCIMGroupMappingUpdateOptions{SCIMSyncPaused: Bool(true)},
			assertAfter: func(t *testing.T, teamID string) {
				scimAttr := getScimAttributeValues(ctx, t, client, teamID)
				assert.True(t, scimAttr.SCIMSyncPaused, "Expected SCIMSyncPaused to remain true after re-pausing")
			},
		},
		{
			name:    "idempotent unpause - not paused",
			setup:   linkedTeamSetup,
			options: &AdminSCIMGroupMappingUpdateOptions{SCIMSyncPaused: Bool(false)},
			assertAfter: func(t *testing.T, teamID string) {
				scimAttr := getScimAttributeValues(ctx, t, client, teamID)
				assert.False(t, scimAttr.SCIMSyncPaused, "Expected SCIMSyncPaused to remain false after unpausing")
			},
		},
		{
			name: "team linked then unlinked",
			setup: func(t *testing.T) string {
				teamID := createSingleTeam(t, client)
				require.NoError(t, createSCIMGroupMapping(ctx, scimClient, teamID, scimGroups[0].ID))
				require.NoError(t, deleteSCIMGroupMapping(ctx, scimClient, teamID))
				return teamID
			},
			options:     &AdminSCIMGroupMappingUpdateOptions{SCIMSyncPaused: Bool(true)},
			shouldError: true,
			expectedErr: ErrSCIMGroupMappingTeamNotLinked,
		},
		{
			name:        "nil options",
			setup:       func(t *testing.T) string { return createSingleTeam(t, client) },
			options:     nil,
			shouldError: true,
			expectedErr: ErrRequiredSCIMGroupMappingUpdateOps,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			teamID := tc.setup(t)
			err := updateSCIMGroupMapping(ctx, scimClient, teamID, tc.options)
			if tc.shouldError {
				require.EqualError(t, err, tc.expectedErr.Error())
				return
			}
			require.NoError(t, err)
			if tc.assertAfter != nil {
				tc.assertAfter(t, teamID)
			}
		})
	}
}

func TestAdminSCIMGroupMappings_Delete(t *testing.T) {
	skipUnlessEnterprise(t)
	client := testClient(t)
	ctx := context.Background()

	enableSCIM(ctx, t, client, true)
	t.Cleanup(func() { enableSCIM(ctx, t, client, false) })

	scimClient, scimGroups := setupSCIMGroups(ctx, t, client)

	testcases := []struct {
		name string
		// setup returns the teamID to delete the mapping for.
		setup       func(t *testing.T) string
		shouldError bool
		expectedErr error
		// assertAfter runs only when shouldError is false, to verify post-delete state.
		assertAfter func(t *testing.T, teamID string)
	}{
		{
			name: "unlink mapped team",
			setup: func(t *testing.T) string {
				teamID := createSingleTeam(t, client)
				require.NoError(t, createSCIMGroupMapping(ctx, scimClient, teamID, scimGroups[0].ID))
				return teamID
			},
			assertAfter: func(t *testing.T, teamID string) {
				scimAttr := getScimAttributeValues(ctx, t, client, teamID)
				assert.False(t, scimAttr.SCIMLinked, "Expected SCIMLinked to be false after deleting mapping")
				assert.Empty(t, scimAttr.SCIMGroupName, "Expected SCIMGroupName to be empty after deleting mapping")
			},
		},
		{
			name:        "team not linked",
			setup:       func(t *testing.T) string { return createSingleTeam(t, client) },
			shouldError: true,
			expectedErr: ErrSCIMGroupMappingTeamNotLinked,
		},
		{
			name:        "invalid team ID",
			setup:       func(_ *testing.T) string { return "this is an invalid team id" },
			shouldError: true,
			expectedErr: ErrInvalidTeamID,
		},
		{
			name:        "non-existent team",
			setup:       func(_ *testing.T) string { return "this-team-does-not-exist" },
			shouldError: true,
			expectedErr: ErrResourceNotFound,
		},
		{
			name: "delete after pause",
			setup: func(t *testing.T) string {
				teamID := createSingleTeam(t, client)
				require.NoError(t, createSCIMGroupMapping(ctx, scimClient, teamID, scimGroups[0].ID))
				require.NoError(t, updateSCIMGroupMapping(ctx, scimClient, teamID, &AdminSCIMGroupMappingUpdateOptions{SCIMSyncPaused: Bool(true)}))
				return teamID
			},
			assertAfter: func(t *testing.T, teamID string) {
				scimAttr := getScimAttributeValues(ctx, t, client, teamID)
				assert.False(t, scimAttr.SCIMLinked, "Expected SCIMLinked to be false after deleting paused mapping")
				assert.Empty(t, scimAttr.SCIMGroupName, "Expected SCIMGroupName to be empty after deleting paused mapping")
			},
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			teamID := tc.setup(t)
			err := deleteSCIMGroupMapping(ctx, scimClient, teamID)
			if tc.shouldError {
				require.EqualError(t, err, tc.expectedErr.Error())
				return
			}
			require.NoError(t, err)
			if tc.assertAfter != nil {
				tc.assertAfter(t, teamID)
			}
		})
	}
}

// SCIM group mapping API wrappers. Sleep only for calls that are expected
// to reach the API, so validation-only failures don't incur unnecessary
// delay in test cases while still throttling real Create/Update/Delete
// requests to avoid 429s. The delay is an empirically chosen stable value,
// not a precise encoding of a specific requests-per-minute limit.

// createSCIMGroupMapping links teamID to groupID.
func createSCIMGroupMapping(ctx context.Context, scim *SCIMResource, teamID, groupID string) error {
	if validStringID(&teamID) && validStringID(&groupID) {
		time.Sleep(scimGroupMappingDelay)
	}
	return scim.SCIMGroupMappings.Create(ctx, teamID, &AdminSCIMGroupMappingCreateOptions{SCIMGroupID: groupID})
}

// updateSCIMGroupMapping updates the mapping for teamID with the provided options.
func updateSCIMGroupMapping(ctx context.Context, scim *SCIMResource, teamID string, opts *AdminSCIMGroupMappingUpdateOptions) error {
	if validStringID(&teamID) && opts != nil && opts.SCIMSyncPaused != nil {
		time.Sleep(scimGroupMappingDelay)
	}
	return scim.SCIMGroupMappings.Update(ctx, teamID, opts)
}

// deleteSCIMGroupMapping unlinks any SCIM group mapping for teamID.
func deleteSCIMGroupMapping(ctx context.Context, scim *SCIMResource, teamID string) error {
	if validStringID(&teamID) {
		time.Sleep(scimGroupMappingDelay)
	}
	return scim.SCIMGroupMappings.Delete(ctx, teamID)
}

// linkSCIMGroupMapping links teamID to groupID, and registers a
// cleanup function that deletes the mapping after the test finishes.
func linkSCIMGroupMapping(ctx context.Context, t *testing.T, scim *SCIMResource, teamID, groupID string) {
	t.Helper()
	require.NoError(t, createSCIMGroupMapping(ctx, scim, teamID, groupID))
	t.Cleanup(func() {
		err := deleteSCIMGroupMapping(ctx, scim, teamID)
		require.NoError(t, err)
	})
}

// createSingleTeam returns the ID of one freshly created team.
func createSingleTeam(t *testing.T, client *Client) string {
	return createTeams(t, client, 1)[0].ID
}

// createTeams creates n teams and returns their details.
// It also registers cleanup functions to delete the teams after the test finishes.
func createTeams(t *testing.T, client *Client, n int) []*Team {
	if n <= 0 {
		return nil
	}

	var testTeams []*Team
	var teamCleanupFuncs []func()

	org, orgCleanup := createOrganization(t, client)

	for range n {
		testTeam, teamCleanup := createTeam(t, client, org)
		teamCleanupFuncs = append(teamCleanupFuncs, teamCleanup)
		testTeams = append(testTeams, testTeam)
	}

	t.Cleanup(func() {
		for _, teamCleanupFunc := range teamCleanupFuncs {
			teamCleanupFunc()
		}
		orgCleanup()
	})

	return testTeams
}

// setupSCIMGroups creates a SCIM token, creates two SCIM groups, and returns the SCIM resource and the created groups.
func setupSCIMGroups(ctx context.Context, t *testing.T, client *Client) (*SCIMResource, []AdminSCIMGroup) {
	var scimGroups []AdminSCIMGroup
	var createdGroupIDs []string

	scimToken, err := client.Admin.Settings.SCIM.Tokens.Create(ctx, "integration-test-token")
	require.NoError(t, err)

	t.Cleanup(func() {
		for i := len(createdGroupIDs) - 1; i >= 0; i-- {
			time.Sleep(scimGroupMappingDelay)
			deleteSCIMGroup(ctx, t, client, createdGroupIDs[i], scimToken.Token)
		}
	})

	for range 2 {
		randomGroupName := randomStringWithoutSpecialChar(t)
		scimGroupID := createSCIMGroup(ctx, t, client, randomGroupName, scimToken.Token)
		createdGroupIDs = append(createdGroupIDs, scimGroupID)
		scimGroups = append(scimGroups, AdminSCIMGroup{ID: scimGroupID, Name: randomGroupName})
	}
	return client.Admin.Settings.SCIM, scimGroups
}

// teamSCIMAttributes is a temporary helper used until the Team struct exposes
// SCIM attributes natively via the Team Read API. At that point this helper
// can be removed
// TODO(TF-35675): Expose SCIM attributes natively via the Team Read API so
// this helper can be removed. https://hashicorp.atlassian.net/browse/TF-35675
type teamSCIMAttributes struct {
	SCIMGroupName  string `jsonapi:"attr,scim-group-name"`
	SCIMLinked     bool   `jsonapi:"attr,scim-linked"`
	SCIMSyncPaused bool   `jsonapi:"attr,scim-sync-paused"`
}

// getScimAttributeValues retrieves the SCIM-related attributes for the team with teamID.
func getScimAttributeValues(ctx context.Context, t *testing.T, client *Client, teamID string) teamSCIMAttributes {
	req, err := client.NewRequest("GET", fmt.Sprintf("teams/%s", url.PathEscape(teamID)), nil)
	require.NoError(t, err)

	var attrs teamSCIMAttributes
	err = req.Do(ctx, &attrs)
	require.NoError(t, err)

	return attrs
}
