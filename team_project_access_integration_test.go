// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package tfe

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/stretchr/testify/require"
)

func TestTeamProjectAccessesList(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	defer orgTestCleanup()

	pTest, pTestCleanup := createProject(t, client, orgTest)
	defer pTestCleanup()

	tmTest1, tmTest1Cleanup := createTeam(t, client, orgTest)
	defer tmTest1Cleanup()
	tmTest2, tmTest2Cleanup := createTeam(t, client, orgTest)
	defer tmTest2Cleanup()

	tpaTest1, tpaTest1Cleanup := createTeamProjectAccess(t, client, tmTest1, pTest, orgTest)
	defer tpaTest1Cleanup()
	tpaTest2, tpaTest2Cleanup := createTeamProjectAccess(t, client, tmTest2, pTest, orgTest)
	defer tpaTest2Cleanup()

	t.Run("with valid options", func(t *testing.T) {
		tpal, err := client.TeamProjectAccess.List(ctx, TeamProjectAccessListOptions{
			ProjectID: pTest.ID,
		})
		require.NoError(t, err)
		assert.Contains(t, tpal.Items, tpaTest1)
		assert.Contains(t, tpal.Items, tpaTest2)
	})

	t.Run("with list options", func(t *testing.T) {
		// Request a page number which is out of range. The result should
		// be successful, but return no results if the paging options are
		// properly passed along.
		tpal, err := client.TeamProjectAccess.List(ctx, TeamProjectAccessListOptions{
			ProjectID: pTest.ID,
			ListOptions: ListOptions{
				PageNumber: 999,
				PageSize:   100,
			},
		})
		require.NoError(t, err)
		assert.Empty(t, tpal.Items)
		assert.Equal(t, 999, tpal.CurrentPage)
		assert.Equal(t, 2, tpal.TotalCount)
	})

	t.Run("without projectID options", func(t *testing.T) {
		tpal, err := client.TeamProjectAccess.List(ctx, TeamProjectAccessListOptions{
			ListOptions: ListOptions{
				PageNumber: 2,
				PageSize:   25,
			},
		})
		assert.Nil(t, tpal)
		assert.Equal(t, err, ErrInvalidProjectID)
	})

	t.Run("without a valid projectID", func(t *testing.T) {
		tpal, err := client.TeamProjectAccess.List(ctx, TeamProjectAccessListOptions{
			ProjectID: badIdentifier,
		})
		assert.Nil(t, tpal)
		assert.EqualError(t, err, ErrInvalidProjectID.Error())
	})
}

func TestTeamProjectAccessesRead(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	defer orgTestCleanup()

	pTest, pTestCleanup := createProject(t, client, orgTest)
	defer pTestCleanup()

	tmTest, tmTestCleanup := createTeam(t, client, orgTest)
	defer tmTestCleanup()

	tpaTest, tpaTestCleanup := createTeamProjectAccess(t, client, tmTest, pTest, orgTest)
	defer tpaTestCleanup()

	t.Run("when the team access exists", func(t *testing.T) {
		tpa, err := client.TeamProjectAccess.Read(ctx, tpaTest.ID)
		require.NoError(t, err)

		assert.Equal(t, TeamProjectAccessAdmin, tpa.Access)

		t.Run("team relationship is decoded", func(t *testing.T) {
			assert.NotEmpty(t, tpa.Team)
		})

		t.Run("project relationship is decoded", func(t *testing.T) {
			assert.NotEmpty(t, tpa.Project)
		})
	})

	t.Run("when the team access does not exist", func(t *testing.T) {
		tpa, err := client.TeamProjectAccess.Read(ctx, "nonexisting")
		assert.Nil(t, tpa)
		assert.Equal(t, err, ErrResourceNotFound)
	})

	t.Run("without a valid team access ID", func(t *testing.T) {
		tpa, err := client.TeamProjectAccess.Read(ctx, badIdentifier)
		assert.Nil(t, tpa)
		assert.Equal(t, err, ErrInvalidTeamProjectAccessID)
	})
}

func TestTeamProjectAccessesAdd(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	defer orgTestCleanup()

	pTest, pTestCleanup := createProject(t, client, orgTest)
	defer pTestCleanup()

	tmTest, tmTestCleanup := createTeam(t, client, orgTest)
	defer tmTestCleanup()

	t.Run("with valid options", func(t *testing.T) {
		options := TeamProjectAccessAddOptions{
			Access:  *ProjectAccess(TeamProjectAccessAdmin),
			Team:    tmTest,
			Project: pTest,
		}

		tpa, err := client.TeamProjectAccess.Add(ctx, options)
		defer func() {
			err := client.TeamProjectAccess.Remove(ctx, tpa.ID)
			if err != nil {
				t.Logf("error removing team access (%s): %s", tpa.ID, err)
			}
		}()

		require.NoError(t, err)

		// Get a refreshed view from the API.
		refreshed, err := client.TeamProjectAccess.Read(ctx, tpa.ID)
		require.NoError(t, err)

		for _, item := range []*TeamProjectAccess{
			tpa,
			refreshed,
		} {
			assert.NotEmpty(t, item.ID)
			assert.Equal(t, options.Access, item.Access)
		}
	})

	t.Run("with valid options for all custom TeamProject permissions", func(t *testing.T) {
		options := TeamProjectAccessAddOptions{
			Access:  *ProjectAccess(TeamProjectAccessCustom),
			Team:    tmTest,
			Project: pTest,
			ProjectAccess: &TeamProjectAccessProjectPermissionsOptions{
				Settings:     ProjectSettingsPermission(ProjectSettingsPermissionUpdate),
				Teams:        ProjectTeamsPermission(ProjectTeamsPermissionManage),
				VariableSets: ProjectVariableSetsPermission(ProjectVariableSetsPermissionWrite),
			},
			WorkspaceAccess: &TeamProjectAccessWorkspacePermissionsOptions{
				Runs:          WorkspaceRunsPermission(WorkspaceRunsPermissionApply),
				SentinelMocks: WorkspaceSentinelMocksPermission(WorkspaceSentinelMocksPermissionRead),
				StateVersions: WorkspaceStateVersionsPermission(WorkspaceStateVersionsPermissionWrite),
				Variables:     WorkspaceVariablesPermission(WorkspaceVariablesPermissionWrite),
				Create:        Bool(true),
				Locking:       Bool(true),
				Move:          Bool(true),
				Delete:        Bool(false),
				RunTasks:      Bool(false),
			},
		}

		tpa, err := client.TeamProjectAccess.Add(ctx, options)
		defer func() {
			err := client.TeamProjectAccess.Remove(ctx, tpa.ID)
			if err != nil {
				t.Logf("error removing team access (%s): %s", tpa.ID, err)
			}
		}()

		require.NoError(t, err)

		// Get a refreshed view from the API.
		refreshed, err := client.TeamProjectAccess.Read(ctx, tpa.ID)
		require.NoError(t, err)

		for _, item := range []*TeamProjectAccess{
			tpa,
			refreshed,
		} {
			assert.NotEmpty(t, item.ID)
			assert.Equal(t, options.Access, item.Access)
			assert.Equal(t, *options.ProjectAccess.Settings, item.ProjectAccess.ProjectSettingsPermission)
			assert.Equal(t, *options.ProjectAccess.Teams, item.ProjectAccess.ProjectTeamsPermission)
			assert.Equal(t, *options.ProjectAccess.VariableSets, item.ProjectAccess.ProjectVariableSetsPermission)
			assert.Equal(t, *options.WorkspaceAccess.Runs, item.WorkspaceAccess.WorkspaceRunsPermission)
			assert.Equal(t, *options.WorkspaceAccess.SentinelMocks, item.WorkspaceAccess.WorkspaceSentinelMocksPermission)
			assert.Equal(t, *options.WorkspaceAccess.StateVersions, item.WorkspaceAccess.WorkspaceStateVersionsPermission)
			assert.Equal(t, *options.WorkspaceAccess.Variables, item.WorkspaceAccess.WorkspaceVariablesPermission)
			assert.Equal(t, item.WorkspaceAccess.WorkspaceCreatePermission, true)
			assert.Equal(t, item.WorkspaceAccess.WorkspaceLockingPermission, true)
			assert.Equal(t, item.WorkspaceAccess.WorkspaceMovePermission, true)
			assert.Equal(t, item.WorkspaceAccess.WorkspaceDeletePermission, false)
			assert.Equal(t, item.WorkspaceAccess.WorkspaceRunTasksPermission, false)
		}
	})

	t.Run("with valid options for some custom TeamProject permissions", func(t *testing.T) {
		options := TeamProjectAccessAddOptions{
			Access:  *ProjectAccess(TeamProjectAccessCustom),
			Team:    tmTest,
			Project: pTest,
			ProjectAccess: &TeamProjectAccessProjectPermissionsOptions{
				Settings: ProjectSettingsPermission(ProjectSettingsPermissionUpdate),
			},
			WorkspaceAccess: &TeamProjectAccessWorkspacePermissionsOptions{
				Runs: WorkspaceRunsPermission(WorkspaceRunsPermissionApply),
			},
		}

		tpa, err := client.TeamProjectAccess.Add(ctx, options)
		t.Cleanup(func() {
			err := client.TeamProjectAccess.Remove(ctx, tpa.ID)
			if err != nil {
				t.Logf("error removing team access (%s): %s", tpa.ID, err)
			}
		})

		require.NoError(t, err)

		// Get a refreshed view from the API.
		refreshed, err := client.TeamProjectAccess.Read(ctx, tpa.ID)
		require.NoError(t, err)

		for _, item := range []*TeamProjectAccess{
			tpa,
			refreshed,
		} {
			assert.NotEmpty(t, item.ID)
			assert.Equal(t, options.Access, item.Access)
			assert.Equal(t, *options.ProjectAccess.Settings, item.ProjectAccess.ProjectSettingsPermission)
			assert.Equal(t, *options.WorkspaceAccess.Runs, item.WorkspaceAccess.WorkspaceRunsPermission)
		}
	})

	t.Run("when the team already has access to the project", func(t *testing.T) {
		_, tpaTestCleanup := createTeamProjectAccess(t, client, tmTest, pTest, nil)
		defer tpaTestCleanup()

		options := TeamProjectAccessAddOptions{
			Access:  *ProjectAccess(TeamProjectAccessAdmin),
			Team:    tmTest,
			Project: pTest,
		}

		_, err := client.TeamProjectAccess.Add(ctx, options)
		assert.Error(t, err)
	})

	t.Run("when options is missing access", func(t *testing.T) {
		tpa, err := client.TeamProjectAccess.Add(ctx, TeamProjectAccessAddOptions{
			Team:    tmTest,
			Project: pTest,
		})
		assert.Nil(t, tpa)
		assert.Equal(t, err, ErrInvalidTeamProjectAccessType)
	})

	t.Run("when options is missing team", func(t *testing.T) {
		tpa, err := client.TeamProjectAccess.Add(ctx, TeamProjectAccessAddOptions{
			Access:  *ProjectAccess(TeamProjectAccessAdmin),
			Project: pTest,
		})
		assert.Nil(t, tpa)
		assert.Equal(t, err, ErrRequiredTeam)
	})

	t.Run("when options is missing project", func(t *testing.T) {
		tpa, err := client.TeamProjectAccess.Add(ctx, TeamProjectAccessAddOptions{
			Access: *ProjectAccess(TeamProjectAccessAdmin),
			Team:   tmTest,
		})
		assert.Nil(t, tpa)
		assert.Equal(t, err, ErrRequiredProject)
	})

	t.Run("when invalid custom project permission is provided in options", func(t *testing.T) {
		tpa, err := client.TeamProjectAccess.Add(ctx, TeamProjectAccessAddOptions{
			Access:  *ProjectAccess(TeamProjectAccessCustom),
			Team:    tmTest,
			Project: pTest,
			ProjectAccess: &TeamProjectAccessProjectPermissionsOptions{
				Teams: ProjectTeamsPermission(badIdentifier),
			},
		})
		assert.Nil(t, tpa)
		assert.Error(t, err)
	})

	t.Run("when invalid access is provided in options", func(t *testing.T) {
		tpa, err := client.TeamProjectAccess.Add(ctx, TeamProjectAccessAddOptions{
			Access:  badIdentifier,
			Team:    tmTest,
			Project: pTest,
		})
		assert.Nil(t, tpa)
		assert.Equal(t, err, ErrInvalidTeamProjectAccessType)
	})
}

func TestTeamProjectAccessesUpdate(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	defer orgTestCleanup()

	pTest, pTestCleanup := createProject(t, client, orgTest)
	defer pTestCleanup()

	tmTest, tmTestCleanup := createTeam(t, client, orgTest)
	defer tmTestCleanup()

	tpaTest, tpaTestCleanup := createTeamProjectAccess(t, client, tmTest, pTest, orgTest)
	defer tpaTestCleanup()

	t.Run("with valid attributes", func(t *testing.T) {
		options := TeamProjectAccessUpdateOptions{
			Access: ProjectAccess(TeamProjectAccessRead),
		}

		tpa, err := client.TeamProjectAccess.Update(ctx, tpaTest.ID, options)
		require.NoError(t, err)

		assert.Equal(t, tpa.Access, TeamProjectAccessRead)
	})

	t.Run("with valid custom permissions attributes for all permissions", func(t *testing.T) {
		options := TeamProjectAccessUpdateOptions{
			Access: ProjectAccess(TeamProjectAccessCustom),
			ProjectAccess: &TeamProjectAccessProjectPermissionsOptions{
				Settings: ProjectSettingsPermission(ProjectSettingsPermissionUpdate),
				Teams:    ProjectTeamsPermission(ProjectTeamsPermissionManage),
			},
			WorkspaceAccess: &TeamProjectAccessWorkspacePermissionsOptions{
				Runs:          WorkspaceRunsPermission(WorkspaceRunsPermissionPlan),
				SentinelMocks: WorkspaceSentinelMocksPermission(WorkspaceSentinelMocksPermissionNone),
				StateVersions: WorkspaceStateVersionsPermission(WorkspaceStateVersionsPermissionReadOutputs),
				Variables:     WorkspaceVariablesPermission(WorkspaceVariablesPermissionRead),
				Create:        Bool(false),
				Locking:       Bool(false),
				Move:          Bool(false),
				Delete:        Bool(true),
				RunTasks:      Bool(true),
			},
		}

		tpa, err := client.TeamProjectAccess.Update(ctx, tpaTest.ID, options)
		require.NoError(t, err)
		require.NotNil(t, options.ProjectAccess)
		require.NotNil(t, options.WorkspaceAccess)
		assert.Equal(t, tpa.Access, TeamProjectAccessCustom)
		assert.Equal(t, *options.ProjectAccess.Teams, tpa.ProjectAccess.ProjectTeamsPermission)
		assert.Equal(t, *options.ProjectAccess.Settings, tpa.ProjectAccess.ProjectSettingsPermission)
		assert.Equal(t, *options.WorkspaceAccess.Runs, tpa.WorkspaceAccess.WorkspaceRunsPermission)
		assert.Equal(t, *options.WorkspaceAccess.SentinelMocks, tpa.WorkspaceAccess.WorkspaceSentinelMocksPermission)
		assert.Equal(t, *options.WorkspaceAccess.StateVersions, tpa.WorkspaceAccess.WorkspaceStateVersionsPermission)
		assert.Equal(t, *options.WorkspaceAccess.Variables, tpa.WorkspaceAccess.WorkspaceVariablesPermission)
		assert.Equal(t, false, tpa.WorkspaceAccess.WorkspaceCreatePermission)
		assert.Equal(t, false, tpa.WorkspaceAccess.WorkspaceLockingPermission)
		assert.Equal(t, false, tpa.WorkspaceAccess.WorkspaceMovePermission)
		assert.Equal(t, true, tpa.WorkspaceAccess.WorkspaceDeletePermission)
		assert.Equal(t, true, tpa.WorkspaceAccess.WorkspaceRunTasksPermission)
	})

	t.Run("with valid custom permissions attributes for some permissions", func(t *testing.T) {
		// create tpaCustomTest to verify unupdated attributes stay the same for custom permissions
		// because going from admin to read to custom changes the values of all custom permissions
		tm2Test, tm2TestCleanup := createTeam(t, client, orgTest)
		defer tm2TestCleanup()

		TpaOptions := TeamProjectAccessAddOptions{
			Access:  *ProjectAccess(TeamProjectAccessCustom),
			Team:    tm2Test,
			Project: pTest,
		}

		tpaCustomTest, err := client.TeamProjectAccess.Add(ctx, TpaOptions)
		require.NoError(t, err)

		options := TeamProjectAccessUpdateOptions{
			Access: ProjectAccess(TeamProjectAccessCustom),
			ProjectAccess: &TeamProjectAccessProjectPermissionsOptions{
				Teams: ProjectTeamsPermission(ProjectTeamsPermissionManage),
			},
			WorkspaceAccess: &TeamProjectAccessWorkspacePermissionsOptions{
				Create: Bool(false),
			},
		}

		tpa, err := client.TeamProjectAccess.Update(ctx, tpaCustomTest.ID, options)
		require.NoError(t, err)
		require.NotNil(t, options.ProjectAccess)
		require.NotNil(t, options.WorkspaceAccess)
		assert.Equal(t, *options.ProjectAccess.Teams, tpa.ProjectAccess.ProjectTeamsPermission)
		assert.Equal(t, false, tpa.WorkspaceAccess.WorkspaceCreatePermission)
		// assert that other attributes remain the same
		assert.Equal(t, tpaCustomTest.ProjectAccess.ProjectSettingsPermission, tpa.ProjectAccess.ProjectSettingsPermission)
		assert.Equal(t, tpaCustomTest.WorkspaceAccess.WorkspaceLockingPermission, tpa.WorkspaceAccess.WorkspaceLockingPermission)
		assert.Equal(t, tpaCustomTest.WorkspaceAccess.WorkspaceMovePermission, tpa.WorkspaceAccess.WorkspaceMovePermission)
		assert.Equal(t, tpaCustomTest.WorkspaceAccess.WorkspaceDeletePermission, tpa.WorkspaceAccess.WorkspaceDeletePermission)
		assert.Equal(t, tpaCustomTest.WorkspaceAccess.WorkspaceRunsPermission, tpa.WorkspaceAccess.WorkspaceRunsPermission)
		assert.Equal(t, tpaCustomTest.WorkspaceAccess.WorkspaceSentinelMocksPermission, tpa.WorkspaceAccess.WorkspaceSentinelMocksPermission)
		assert.Equal(t, tpaCustomTest.WorkspaceAccess.WorkspaceStateVersionsPermission, tpa.WorkspaceAccess.WorkspaceStateVersionsPermission)
	})
	t.Run("with invalid custom permissions attributes", func(t *testing.T) {
		options := TeamProjectAccessUpdateOptions{
			Access: ProjectAccess(TeamProjectAccessCustom),
			ProjectAccess: &TeamProjectAccessProjectPermissionsOptions{
				Teams: ProjectTeamsPermission(badIdentifier),
			},
		}

		tpa, err := client.TeamProjectAccess.Update(ctx, tpaTest.ID, options)

		assert.Nil(t, tpa)
		assert.Error(t, err)
	})
}

func TestTeamProjectAccessesRemove(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	defer orgTestCleanup()

	pTest, pTestCleanup := createProject(t, client, orgTest)
	defer pTestCleanup()

	tmTest, tmTestCleanup := createTeam(t, client, orgTest)
	defer tmTestCleanup()

	tpaTest, _ := createTeamProjectAccess(t, client, tmTest, pTest, orgTest)

	t.Run("with valid options", func(t *testing.T) {
		err := client.TeamProjectAccess.Remove(ctx, tpaTest.ID)
		require.NoError(t, err)

		// Try loading the project - it should fail.
		_, err = client.TeamProjectAccess.Read(ctx, tpaTest.ID)
		assert.Equal(t, err, ErrResourceNotFound)
	})

	t.Run("when the team access does not exist", func(t *testing.T) {
		err := client.TeamProjectAccess.Remove(ctx, tpaTest.ID)
		assert.Equal(t, err, ErrResourceNotFound)
	})

	t.Run("when the team access ID is invalid", func(t *testing.T) {
		err := client.TeamProjectAccess.Remove(ctx, badIdentifier)
		assert.Equal(t, err, ErrInvalidTeamProjectAccessID)
	})
}
