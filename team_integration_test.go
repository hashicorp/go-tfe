package tfe

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"

	retryablehttp "github.com/hashicorp/go-retryablehttp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTeamsList(t *testing.T) {
	skipIfFreeOnly(t)

	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	defer orgTestCleanup()

	tmTest1, tmTest1Cleanup := createTeam(t, client, orgTest)
	defer tmTest1Cleanup()
	tmTest2, tmTest2Cleanup := createTeam(t, client, orgTest)
	defer tmTest2Cleanup()
	tmTest3, tmTest3Cleanup := createTeam(t, client, orgTest)
	defer tmTest3Cleanup()

	t.Run("without list options", func(t *testing.T) {
		tl, err := client.Teams.List(ctx, orgTest.Name, nil)
		require.NoError(t, err)
		assert.Contains(t, tl.Items, tmTest1)
		assert.Contains(t, tl.Items, tmTest2)

		t.Skip("paging not supported yet in API")
		assert.Equal(t, 1, tl.CurrentPage)
		assert.Equal(t, 2, tl.TotalCount)
	})

	t.Run("with list options", func(t *testing.T) {
		t.Skip("paging not supported yet in API")
		// Request a page number which is out of range. The result should
		// be successful, but return no results if the paging options are
		// properly passed along.
		tl, err := client.Teams.List(ctx, orgTest.Name, &TeamListOptions{
			ListOptions: ListOptions{
				PageNumber: 999,
				PageSize:   100,
			},
		})
		require.NoError(t, err)
		assert.Empty(t, tl.Items)
		assert.Equal(t, 999, tl.CurrentPage)
		assert.Equal(t, 2, tl.TotalCount)

		tl, err = client.Teams.List(ctx, orgTest.Name, &TeamListOptions{
			Names: []string{tmTest2.Name, tmTest3.Name},
		})

		assert.Equal(t, tl.Items, 2)
		assert.Contains(t, tl.Items, tmTest2)
		assert.Contains(t, tl.Items, tmTest3)

		t.Run("with invalid names query param", func(t *testing.T) {
			// should return an error because we've included an empty string
			tl, err = client.Teams.List(ctx, orgTest.Name, &TeamListOptions{
				Names: []string{tmTest2.Name, ""},
			})
			assert.Equal(t, err, ErrEmptyTeamName)
		})
	})

	t.Run("without a valid organization", func(t *testing.T) {
		tl, err := client.Teams.List(ctx, badIdentifier, nil)
		assert.Nil(t, tl)
		assert.EqualError(t, err, ErrInvalidOrg.Error())
	})
}

func TestTeamsCreate(t *testing.T) {
	skipIfFreeOnly(t)

	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	defer orgTestCleanup()

	t.Run("with valid options", func(t *testing.T) {
		options := TeamCreateOptions{
			Name: String("foo"),
		}

		tm, err := client.Teams.Create(ctx, orgTest.Name, options)
		require.NoError(t, err)

		// Get a refreshed view from the API.
		refreshed, err := client.Teams.Read(ctx, tm.ID)
		require.NoError(t, err)

		for _, item := range []*Team{
			tm,
			refreshed,
		} {
			assert.NotEmpty(t, item.ID)
			assert.Equal(t, *options.Name, item.Name)
		}
	})

	t.Run("with sso-team-id", func(t *testing.T) {
		options := TeamCreateOptions{
			Name:      String("rockettes"),
			SSOTeamID: String("7dddb675-73e0-4858-a8ad-0e597064301b"),
		}
		team, err := client.Teams.Create(ctx, orgTest.Name, options)

		assert.Nil(t, err)
		assert.Equal(t, *options.Name, team.Name)

		assert.NotNil(t, team.SSOTeamID)
		assert.Equal(t, *options.SSOTeamID, team.SSOTeamID)
	})

	t.Run("when options is missing name", func(t *testing.T) {
		tm, err := client.Teams.Create(ctx, "foo", TeamCreateOptions{})
		assert.Nil(t, tm)
		assert.EqualError(t, err, ErrRequiredName.Error())
	})

	t.Run("when options has an invalid organization", func(t *testing.T) {
		tm, err := client.Teams.Create(ctx, badIdentifier, TeamCreateOptions{
			Name: String("foo"),
		})
		assert.Nil(t, tm)
		assert.EqualError(t, err, ErrInvalidOrg.Error())
	})
}

func TestTeamsRead(t *testing.T) {
	skipIfFreeOnly(t)

	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	defer orgTestCleanup()

	tmTest, tmTestCleanup := createTeam(t, client, orgTest)
	defer tmTestCleanup()

	opts := TeamCreateOptions{
		Name:      String(randomString(t)),
		SSOTeamID: String(randomString(t)),
		OrganizationAccess: &OrganizationAccessOptions{
			ManagePolicies: Bool(true),
		},
	}
	ssoTeam, err := client.Teams.Create(ctx, orgTest.Name, opts)
	require.NoError(t, err)
	defer func() {
		err := client.Teams.Delete(ctx, ssoTeam.ID)
		require.NoError(t, err)
	}()

	t.Run("when the team exists", func(t *testing.T) {
		tm, err := client.Teams.Read(ctx, tmTest.ID)
		require.NoError(t, err)
		assert.Equal(t, tmTest, tm)

		t.Run("visibility is returned", func(t *testing.T) {
			assert.Equal(t, "secret", tm.Visibility)
		})

		t.Run("permissions are properly decoded", func(t *testing.T) {
			assert.True(t, tm.Permissions.CanDestroy)
		})

		t.Run("organization access is properly decoded", func(t *testing.T) {
			assert.Equal(t, tm.OrganizationAccess.ManagePolicies, *opts.OrganizationAccess.ManagePolicies)
		})

		t.Run("SSO team id is returned", func(t *testing.T) {
			assert.NotNil(t, ssoTeam.SSOTeamID)
			assert.Equal(t, *opts.SSOTeamID, ssoTeam.SSOTeamID)
		})
	})

	t.Run("when the team does not exist", func(t *testing.T) {
		tm, err := client.Teams.Read(ctx, "nonexisting")
		assert.Nil(t, tm)
		assert.Equal(t, err, ErrResourceNotFound)
	})

	t.Run("without a valid team ID", func(t *testing.T) {
		tm, err := client.Teams.Read(ctx, badIdentifier)
		assert.Nil(t, tm)
		assert.Equal(t, err, ErrInvalidTeamID)
	})
}

func TestTeamsUpdate(t *testing.T) {
	skipIfFreeOnly(t)

	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	defer orgTestCleanup()

	tmTest, tmTestCleanup := createTeam(t, client, orgTest)
	defer tmTestCleanup()

	t.Run("with valid options", func(t *testing.T) {
		options := TeamUpdateOptions{
			Name: String("foo bar"),
			OrganizationAccess: &OrganizationAccessOptions{
				ManagePolicies:        Bool(false),
				ManageVCSSettings:     Bool(true),
				ManagePolicyOverrides: Bool(true),
				ManageProviders:       Bool(true),
				ManageModules:         Bool(false),
			},
			Visibility: String("organization"),
		}

		tm, err := client.Teams.Update(ctx, tmTest.ID, options)
		require.NoError(t, err)

		refreshed, err := client.Teams.Read(ctx, tmTest.ID)
		require.NoError(t, err)

		for _, item := range []*Team{
			tm,
			refreshed,
		} {
			assert.Equal(t, *options.Name, item.Name)
			assert.Equal(t,
				*options.Visibility,
				item.Visibility,
			)
			assert.Equal(t,
				*options.OrganizationAccess.ManagePolicies,
				item.OrganizationAccess.ManagePolicies,
			)
			assert.Equal(t,
				*options.OrganizationAccess.ManageVCSSettings,
				item.OrganizationAccess.ManageVCSSettings,
			)
			assert.Equal(t,
				*options.OrganizationAccess.ManagePolicyOverrides,
				item.OrganizationAccess.ManagePolicyOverrides,
			)
			assert.Equal(t,
				*options.OrganizationAccess.ManageProviders,
				item.OrganizationAccess.ManageProviders,
			)
			assert.Equal(t,
				*options.OrganizationAccess.ManageModules,
				item.OrganizationAccess.ManageModules,
			)
		}
	})

	t.Run("when the team does not exist", func(t *testing.T) {
		tm, err := client.Teams.Update(ctx, "nonexisting", TeamUpdateOptions{
			Name: String("foo bar"),
		})
		assert.Nil(t, tm)
		assert.Equal(t, err, ErrResourceNotFound)
	})

	t.Run("without a valid team ID", func(t *testing.T) {
		tm, err := client.Teams.Update(ctx, badIdentifier, TeamUpdateOptions{})
		assert.Nil(t, tm)
		assert.Equal(t, err, ErrInvalidTeamID)
	})
}

func TestTeamsDelete(t *testing.T) {
	skipIfFreeOnly(t)

	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	defer orgTestCleanup()

	tmTest, _ := createTeam(t, client, orgTest)

	t.Run("with valid options", func(t *testing.T) {
		err := client.Teams.Delete(ctx, tmTest.ID)
		require.NoError(t, err)

		// Try loading the workspace - it should fail.
		_, err = client.Teams.Read(ctx, tmTest.ID)
		assert.Equal(t, err, ErrResourceNotFound)
	})

	t.Run("without valid team ID", func(t *testing.T) {
		err := client.Teams.Delete(ctx, badIdentifier)
		assert.Equal(t, err, ErrInvalidTeamID)
	})
}

func TestTeam_Unmarshal(t *testing.T) {
	data := map[string]interface{}{
		"data": map[string]interface{}{
			"type": "teams",
			"id":   "1",
			"attributes": map[string]interface{}{
				"name": "team hashi",
				"organization-access": map[string]interface{}{
					"manage-policies":     true,
					"manage-workspaces":   true,
					"manage-vcs-settings": true,
					"manage-projects":     true,
				},
				"permissions": map[string]interface{}{
					"can-destroy":           true,
					"can-update-membership": true,
				},
			},
		},
	}
	byteData, err := json.Marshal(data)
	require.NoError(t, err)

	responseBody := bytes.NewReader(byteData)
	team := &Team{}
	err = unmarshalResponse(responseBody, team)
	require.NoError(t, err)

	assert.Equal(t, team.ID, "1")
	assert.Equal(t, team.Name, "team hashi")
	assert.Empty(t, team.SSOTeamID)
	assert.Equal(t, team.OrganizationAccess.ManageWorkspaces, true)
	assert.Equal(t, team.OrganizationAccess.ManageVCSSettings, true)
	assert.Equal(t, team.OrganizationAccess.ManagePolicies, true)
	assert.Equal(t, team.OrganizationAccess.ManageProjects, true)
	assert.Equal(t, team.Permissions.CanDestroy, true)
	assert.Equal(t, team.Permissions.CanUpdateMembership, true)
}

func TestTeamCreateOptions_Marshal(t *testing.T) {
	opts := TeamCreateOptions{
		Name:       String("team name"),
		Visibility: String("organization"),
		OrganizationAccess: &OrganizationAccessOptions{
			ManagePolicies: Bool(true),
		},
	}

	reqBody, err := serializeRequestBody(&opts)
	require.NoError(t, err)
	req, err := retryablehttp.NewRequest("POST", "url", reqBody)
	require.NoError(t, err)
	bodyBytes, err := req.BodyBytes()
	require.NoError(t, err)

	expectedBody := `{"data":{"type":"teams","attributes":{"name":"team name","organization-access":{"manage-policies":true},"visibility":"organization"}}}
`
	assert.Equal(t, expectedBody, string(bodyBytes))
}

func TestTeamsUpdateRunTasks(t *testing.T) {
	skipIfFreeOnly(t)
	skipUnlessBeta(t)
	skipIfEnterprise(t)

	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	defer orgTestCleanup()

	tmTest, tmTestCleanup := createTeam(t, client, orgTest)
	defer tmTestCleanup()

	t.Run("with valid options", func(t *testing.T) {
		options := TeamUpdateOptions{
			Name: String("foo bar"),
			OrganizationAccess: &OrganizationAccessOptions{
				ManageRunTasks: Bool(true),
			},
			Visibility: String("organization"),
		}

		tm, err := client.Teams.Update(ctx, tmTest.ID, options)
		require.NoError(t, err)

		refreshed, err := client.Teams.Read(ctx, tmTest.ID)
		require.NoError(t, err)

		for _, item := range []*Team{
			tm,
			refreshed,
		} {
			assert.Equal(t, *options.Name, item.Name)
			assert.Equal(t,
				*options.OrganizationAccess.ManageRunTasks,
				item.OrganizationAccess.ManageRunTasks,
			)
		}
	})
}

func TestTeamsUpdateManageProjects(t *testing.T) {
	skipIfFreeOnly(t)
	skipUnlessBeta(t)
	skipIfEnterprise(t)

	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	defer orgTestCleanup()

	tmTest, tmTestCleanup := createTeam(t, client, orgTest)
	defer tmTestCleanup()

	t.Run("with valid options", func(t *testing.T) {
		options := TeamUpdateOptions{
			Name: String("foo bar"),
			OrganizationAccess: &OrganizationAccessOptions{
				// **Note: ManageProjects requires ManageWorkspaces field to be set and subject to change later.**
				ManageWorkspaces: Bool(true),
				ManageProjects:   Bool(true),
			},
			Visibility: String("organization"),
		}

		tm, err := client.Teams.Update(ctx, tmTest.ID, options)
		require.NoError(t, err)

		refreshed, err := client.Teams.Read(ctx, tmTest.ID)
		require.NoError(t, err)

		for _, item := range []*Team{
			tm,
			refreshed,
		} {
			assert.Equal(t, *options.Name, item.Name)
			assert.Equal(t,
				*options.OrganizationAccess.ManageProjects,
				item.OrganizationAccess.ManageProjects,
			)
		}
	})
}
