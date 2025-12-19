// Copyright IBM Corp. 2018, 2025
// SPDX-License-Identifier: MPL-2.0

package tfe

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOrganizationsList(t *testing.T) {
	t.Parallel()
	client := testClient(t)
	ctx := context.Background()

	orgTest1, orgTest1Cleanup := createOrganization(t, client)
	t.Cleanup(orgTest1Cleanup)
	orgTest2, orgTest2Cleanup := createOrganization(t, client)
	t.Cleanup(orgTest2Cleanup)

	t.Run("with no list options", func(t *testing.T) {
		orgl, err := client.Organizations.List(ctx, nil)
		require.NoError(t, err)
		assert.Contains(t, orgl.Items, orgTest1)
		assert.Contains(t, orgl.Items, orgTest2)

		t.Skip("paging not supported yet in API")
		assert.Equal(t, 1, orgl.CurrentPage)
		assert.Equal(t, 2, orgl.TotalCount)
	})

	t.Run("with list options", func(t *testing.T) {
		t.Skip("paging not supported yet in API")
		// Request a page number which is out of range. The result should
		// be successful, but return no results if the paging options are
		// properly passed along.
		orgl, err := client.Organizations.List(ctx, &OrganizationListOptions{
			ListOptions: ListOptions{
				PageNumber: 999,
				PageSize:   100,
			},
		})
		require.NoError(t, err)
		assert.Empty(t, orgl)
		assert.Equal(t, 999, orgl.CurrentPage)
		assert.Equal(t, 2, orgl.TotalCount)
	})

	t.Run("when querying on a valid org name", func(t *testing.T) {
		org, orgTestCleanup := createOrganization(t, client)
		t.Cleanup(orgTestCleanup)

		orgList, err := client.Organizations.List(ctx, &OrganizationListOptions{
			Query: org.Name,
		})

		require.NoError(t, err)
		assert.Equal(t, true, orgItemsContainsName(orgList.Items, org.Name))
	})

	t.Run("when querying on a valid email", func(t *testing.T) {
		org, orgTestCleanup := createOrganization(t, client)
		t.Cleanup(orgTestCleanup)

		orgList, err := client.Organizations.List(ctx, &OrganizationListOptions{
			Query: org.Email,
		})

		require.NoError(t, err)
		assert.Equal(t, true, orgItemsContainsEmail(orgList.Items, org.Email))
	})

	t.Run("with invalid query name", func(t *testing.T) {
		org, orgTestCleanup := createOrganization(t, client)
		t.Cleanup(orgTestCleanup)

		orgList, err := client.Organizations.List(ctx, &OrganizationListOptions{
			Query: org.Name,
		})

		require.NoError(t, err)
		assert.NotEqual(t, orgList.Items, orgTest1.Name)
	})

	t.Run("with invalid query email", func(t *testing.T) {
		org, orgTestCleanup := createOrganization(t, client)
		t.Cleanup(orgTestCleanup)

		orgList, err := client.Organizations.List(ctx, &OrganizationListOptions{
			Query: org.Email,
		})

		require.NoError(t, err)
		assert.NotEqual(t, orgList.Items, orgTest1.Email)
	})
}

func TestOrganizationsCreate(t *testing.T) {
	t.Parallel()
	client := testClient(t)
	ctx := context.Background()

	t.Run("with valid options", func(t *testing.T) {
		options := OrganizationCreateOptions{
			Name:  String(randomString(t)),
			Email: String(randomString(t) + "@tfe.local"),
		}

		org, err := client.Organizations.Create(ctx, options)
		require.NoError(t, err)

		t.Cleanup(func() {
			err := client.Organizations.Delete(ctx, org.Name)
			if err != nil {
				t.Logf("error deleting organization (%s): %s", org.Name, err)
			}
		})

		assert.Equal(t, *options.Name, org.Name)
		assert.Equal(t, *options.Email, org.Email)
		assert.Equal(t, "remote", org.DefaultExecutionMode)
		assert.Nil(t, org.DefaultAgentPool)
	})

	t.Run("when no email is provided", func(t *testing.T) {
		org, err := client.Organizations.Create(ctx, OrganizationCreateOptions{
			Name: String("foo"),
		})
		assert.Nil(t, org)
		assert.Equal(t, err, ErrRequiredEmail)
	})

	t.Run("when no name is provided", func(t *testing.T) {
		_, err := client.Organizations.Create(ctx, OrganizationCreateOptions{
			Email: String("foo@bar.com"),
		})
		assert.EqualError(t, err, ErrRequiredName.Error())
	})

	t.Run("with invalid name", func(t *testing.T) {
		org, err := client.Organizations.Create(ctx, OrganizationCreateOptions{
			Name:  String(badIdentifier),
			Email: String("foo@bar.com"),
		})
		assert.Nil(t, org)
		assert.EqualError(t, err, ErrInvalidName.Error())
	})
}

func TestOrganizationsReadWithBusiness(t *testing.T) {
	t.Parallel()
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	t.Cleanup(orgTestCleanup)
	// With Business
	newSubscriptionUpdater(orgTest).WithBusinessPlan().Update(t)

	t.Run("when the org exists", func(t *testing.T) {
		org, err := client.Organizations.Read(ctx, orgTest.Name)
		require.NoError(t, err)
		assert.Equal(t, orgTest.Name, org.Name)
		assert.Equal(t, orgTest.ExternalID, org.ExternalID)
		assert.NotEmpty(t, org.Permissions)

		t.Run("permissions are properly decoded", func(t *testing.T) {
			assert.True(t, org.Permissions.CanDestroy)
			assert.True(t, org.Permissions.CanDeployNoCodeModules)
			assert.True(t, org.Permissions.CanManageNoCodeModules)
		})
	})
}

func TestOrganizationsRead(t *testing.T) {
	t.Parallel()
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	t.Cleanup(orgTestCleanup)

	t.Run("when the org exists", func(t *testing.T) {
		org, err := client.Organizations.Read(ctx, orgTest.Name)
		require.NoError(t, err)
		assert.Equal(t, orgTest, org)
		assert.NotEmpty(t, org.Permissions)

		t.Run("permissions are properly decoded", func(t *testing.T) {
			assert.True(t, org.Permissions.CanDestroy)
		})

		t.Run("timestamps are populated", func(t *testing.T) {
			assert.NotEmpty(t, org.CreatedAt)
			// By default accounts are in the free tier and are not in a trial
			assert.Empty(t, org.TrialExpiresAt)
			assert.Greater(t, org.RemainingTestableCount, 1)
		})
	})

	t.Run("with invalid name", func(t *testing.T) {
		org, err := client.Organizations.Read(ctx, badIdentifier)
		assert.Nil(t, org)
		assert.EqualError(t, err, ErrInvalidOrg.Error())
	})

	t.Run("when the org does not exist", func(t *testing.T) {
		_, err := client.Organizations.Read(ctx, randomString(t))
		assert.Error(t, err)
	})

	t.Run("reads default project", func(t *testing.T) {
		org, err := client.Organizations.ReadWithOptions(ctx, orgTest.Name, OrganizationReadOptions{Include: []OrganizationIncludeOpt{OrganizationDefaultProject}})
		require.NoError(t, err)
		assert.Equal(t, orgTest.Name, org.Name)

		require.NotNil(t, org.DefaultProject)
		assert.NotNil(t, org.DefaultProject.Name)
	})

	t.Run("with default execution mode of 'agent'", func(t *testing.T) {
		orgAgentTest, orgAgentTestCleanup := createOrganizationWithDefaultAgentPool(t, client)
		org, err := client.Organizations.Read(ctx, orgAgentTest.Name)

		t.Cleanup(orgAgentTestCleanup)
		require.NoError(t, err)

		t.Run("execution mode and agent pool are properly decoded", func(t *testing.T) {
			assert.Equal(t, "agent", org.DefaultExecutionMode)
			assert.NotNil(t, org.DefaultAgentPool)
			assert.Equal(t, org.DefaultAgentPool.ID, orgAgentTest.DefaultAgentPool.ID)
		})
	})

	t.Run("read primary hyok configuration of an organization", func(t *testing.T) {
		skipHYOKIntegrationTests(t)

		// replace the environment variable with a valid organization name that has primary hyok configuration
		hyokOrganizationName := os.Getenv("HYOK_ORGANIZATION_NAME")
		if hyokOrganizationName == "" {
			t.Fatal("Export a valid HYOK_ORGANIZATION_NAME before running this test!")
		}

		org, err := client.Organizations.Read(ctx, hyokOrganizationName)
		require.NoError(t, err)
		assert.NotEmpty(t, org.PrimaryHYOKConfiguration)
	})

	t.Run("read enforce hyok of an organization", func(t *testing.T) {
		skipHYOKIntegrationTests(t)

		// replace the environment variable with a valid organization name that has enforce hyok set to true or false
		hyokOrganizationName := os.Getenv("HYOK_ORGANIZATION_NAME")
		if hyokOrganizationName == "" {
			t.Fatal("Export a valid HYOK_ORGANIZATION_NAME before running this test!")
		}

		org, err := client.Organizations.Read(ctx, hyokOrganizationName)
		require.NoError(t, err)
		assert.True(t, org.EnforceHYOK || !org.EnforceHYOK)
	})
}

func TestOrganizationsUpdate(t *testing.T) {
	t.Parallel()
	client := testClient(t)
	ctx := context.Background()

	t.Run("with HCP Terraform-only options", func(t *testing.T) {
		skipIfEnterprise(t)

		orgTest, orgTestCleanup := createOrganization(t, client)
		t.Cleanup(orgTestCleanup)

		options := OrganizationUpdateOptions{
			SendPassingStatusesForUntriggeredSpeculativePlans: Bool(false),
		}

		org, err := client.Organizations.Update(ctx, orgTest.Name, options)
		require.NoError(t, err)

		assert.Equal(t, false, org.SendPassingStatusesForUntriggeredSpeculativePlans)
	})

	t.Run("with new AggregatedCommitStatusEnabled option", func(t *testing.T) {
		skipIfEnterprise(t)

		for _, testCase := range []bool{true, false} {
			orgTest, orgTestCleanup := createOrganization(t, client)
			t.Cleanup(orgTestCleanup)

			options := OrganizationUpdateOptions{
				AggregatedCommitStatusEnabled: Bool(testCase),
			}

			org, err := client.Organizations.Update(ctx, orgTest.Name, options)
			require.NoError(t, err)

			assert.Equal(t, testCase, org.AggregatedCommitStatusEnabled)
		}
	})

	t.Run("with new SpeculativePlanManagementEnabled option", func(t *testing.T) {
		skipIfEnterprise(t)

		for _, testCase := range []bool{true, false} {
			orgTest, orgTestCleanup := createOrganization(t, client)
			t.Cleanup(orgTestCleanup)

			options := OrganizationUpdateOptions{
				SpeculativePlanManagementEnabled: Bool(testCase),
			}

			org, err := client.Organizations.Update(ctx, orgTest.Name, options)
			require.NoError(t, err)

			assert.Equal(t, testCase, org.SpeculativePlanManagementEnabled)
		}
	})

	t.Run("with new UserTokensEnabled option", func(t *testing.T) {
		orgTest, orgTestCleanup := createOrganization(t, client)
		t.Cleanup(orgTestCleanup)

		assert.True(t, *orgTest.UserTokensEnabled, "user tokens enabled by default")

		// we need to switch to an owner's team token, otherwise the client (which auths with a user token)
		// wont be able to delete the org after we disable user tokens
		teamList, err := client.Teams.List(ctx, orgTest.Name, &TeamListOptions{
			Names: []string{"owners"},
		})
		require.NoError(t, err)

		// it should be the only team, we just created the org...
		require.Len(t, teamList.Items, 1)
		ownersTeam := teamList.Items[0]

		ownerToken, ownerTokenCleanup := createTeamToken(t, client, ownersTeam)
		t.Cleanup(ownerTokenCleanup)

		ownerClient := testClient(t)
		ownerClient.token = ownerToken.Token

		// disable user tokens for the organization
		options := OrganizationUpdateOptions{
			UserTokensEnabled: Bool(false),
		}

		org, err := ownerClient.Organizations.Update(ctx, orgTest.Name, options)
		require.NoError(t, err)
		assert.False(t, *org.UserTokensEnabled, "user tokens disabled")

		// try reading something with the user token client and verify that it fails, where the team token client
		// succeeds
		_, err = client.Organizations.Read(ctx, orgTest.Name)
		assert.Error(t, err)
		assert.ErrorContains(t, err, "unauthorized")

		org, err = ownerClient.Organizations.Read(ctx, orgTest.Name)
		assert.NoError(t, err)
		assert.Equal(t, orgTest.Name, org.Name)
		assert.False(t, *org.UserTokensEnabled, "user tokens disabled")

		// re-enable user tokens
		options = OrganizationUpdateOptions{
			UserTokensEnabled: Bool(true),
		}
		org, err = ownerClient.Organizations.Update(ctx, orgTest.Name, options)
		require.NoError(t, err)
		assert.True(t, *org.UserTokensEnabled, "user tokens re-enabled")

		// try reading with the user token again and verify that it works
		org, err = client.Organizations.Read(ctx, orgTest.Name)
		assert.NoError(t, err)
		assert.Equal(t, orgTest.Name, org.Name)
		assert.True(t, *org.UserTokensEnabled, "user tokens re-enabled")
	})

	t.Run("with valid options", func(t *testing.T) {
		orgTest, orgTestCleanup := createOrganization(t, client)

		options := OrganizationUpdateOptions{
			Name:                 String(randomString(t)),
			Email:                String(randomString(t) + "@tfe.local"),
			SessionTimeout:       Int(3600),
			SessionRemember:      Int(3600),
			DefaultExecutionMode: String("local"),
		}

		org, err := client.Organizations.Update(ctx, orgTest.Name, options)
		if err != nil {
			orgTestCleanup()
		}
		require.NoError(t, err)

		// Make sure we clean up the renamed org.
		defer func() {
			err := client.Organizations.Delete(ctx, org.Name)
			if err != nil {
				t.Logf("Error deleting organization (%s): %s", org.Name, err)
			}
		}()

		// Also get a fresh result from the API to ensure we get the
		// expected values back.
		refreshed, err := client.Organizations.Read(ctx, *options.Name)
		require.NoError(t, err)

		for _, item := range []*Organization{
			org,
			refreshed,
		} {
			assert.Equal(t, *options.Name, item.Name)
			assert.Equal(t, *options.Email, item.Email)
			assert.Equal(t, *options.SessionTimeout, item.SessionTimeout)
			assert.Equal(t, *options.SessionRemember, item.SessionRemember)
			assert.Equal(t, *options.DefaultExecutionMode, item.DefaultExecutionMode)
		}
	})

	t.Run("with invalid name", func(t *testing.T) {
		org, err := client.Organizations.Update(ctx, badIdentifier, OrganizationUpdateOptions{})
		assert.Nil(t, org)
		assert.EqualError(t, err, ErrInvalidOrg.Error())
	})

	t.Run("with agent pool provided, but remote execution mode", func(t *testing.T) {
		orgTest, orgTestCleanup := createOrganization(t, client)
		t.Cleanup(orgTestCleanup)

		pool, agentPoolCleanup := createAgentPool(t, client, orgTest)
		t.Cleanup(agentPoolCleanup)

		org, err := client.Organizations.Update(ctx, orgTest.Name, OrganizationUpdateOptions{
			DefaultAgentPool: pool,
		})
		assert.Nil(t, org)
		assert.ErrorContains(t, err, "Default agent pool must not be specified unless using 'agent' execution mode")
	})

	t.Run("when only updating a subset of fields", func(t *testing.T) {
		orgTest, orgTestCleanup := createOrganization(t, client)
		t.Cleanup(orgTestCleanup)

		org, err := client.Organizations.Update(ctx, orgTest.Name, OrganizationUpdateOptions{})
		require.NoError(t, err)
		assert.Equal(t, orgTest.Name, org.Name)
		assert.Equal(t, orgTest.Email, org.Email)
	})

	t.Run("with different default execution modes", func(t *testing.T) {
		// this helper creates an organization and then updates it to use a default agent pool, so it implicitly asserts
		// that the organization's execution mode can be updated from 'remote' -> 'agent'
		org, orgAgentTestCleanup := createOrganizationWithDefaultAgentPool(t, client)
		assert.Equal(t, "agent", org.DefaultExecutionMode)
		assert.NotNil(t, org.DefaultAgentPool)

		// assert that organization's execution mode can be updated from 'agent' -> 'remote'
		org, err := client.Organizations.Update(ctx, org.Name, OrganizationUpdateOptions{
			DefaultExecutionMode: String("remote"),
		})
		require.NoError(t, err)
		assert.Equal(t, "remote", org.DefaultExecutionMode)
		assert.Nil(t, org.DefaultAgentPool)

		// assert that organization's execution mode can be updated from 'remote' -> 'local'
		org, err = client.Organizations.Update(ctx, org.Name, OrganizationUpdateOptions{
			DefaultExecutionMode: String("local"),
		})
		require.NoError(t, err)
		assert.Equal(t, "local", org.DefaultExecutionMode)
		assert.Nil(t, org.DefaultAgentPool)

		t.Cleanup(orgAgentTestCleanup)
	})

	t.Run("update enforce hyok of an organization to true", func(t *testing.T) {
		skipHYOKIntegrationTests(t)

		// replace the environment variable with a valid organization name with hyok permissions
		hyokOrganizationName := os.Getenv("HYOK_ORGANIZATION_NAME")
		if hyokOrganizationName == "" {
			t.Fatal("Export a valid HYOK_ORGANIZATION_NAME before running this test!")
		}

		org, err := client.Organizations.Update(ctx, hyokOrganizationName, OrganizationUpdateOptions{
			EnforceHYOK: Bool(true),
		})
		require.NoError(t, err)
		assert.True(t, org.EnforceHYOK)
	})

	t.Run("update enforce hyok of an organization to false", func(t *testing.T) {
		skipHYOKIntegrationTests(t)

		// replace the environment variable with a valid organization name with hyok permissions
		hyokOrganizationName := os.Getenv("HYOK_ORGANIZATION_NAME")
		if hyokOrganizationName == "" {
			t.Fatal("Export a valid HYOK_ORGANIZATION_NAME before running this test!")
		}

		org, err := client.Organizations.Update(ctx, hyokOrganizationName, OrganizationUpdateOptions{
			EnforceHYOK: Bool(false),
		})
		require.NoError(t, err)
		assert.False(t, org.EnforceHYOK)
	})
}

func TestOrganizationsDelete(t *testing.T) {
	t.Parallel()
	client := testClient(t)
	ctx := context.Background()

	t.Run("with valid options", func(t *testing.T) {
		orgTest, _ := createOrganization(t, client)

		err := client.Organizations.Delete(ctx, orgTest.Name)
		require.NoError(t, err)

		// Try fetching the org again - it should error.
		_, err = client.Organizations.Read(ctx, orgTest.Name)
		assert.Equal(t, err, ErrResourceNotFound)
	})

	t.Run("with invalid name", func(t *testing.T) {
		err := client.Organizations.Delete(ctx, badIdentifier)
		assert.EqualError(t, err, ErrInvalidOrg.Error())
	})
}

func TestOrganizationsReadCapacity_RunDependent(t *testing.T) {
	t.Parallel()
	t.Skip("Capacity queues are not available in the API")
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	t.Cleanup(orgTestCleanup)

	wTest1, wTestCleanup1 := createWorkspace(t, client, orgTest)
	t.Cleanup(wTestCleanup1)
	wTest2, wTestCleanup2 := createWorkspace(t, client, orgTest)
	t.Cleanup(wTestCleanup2)
	wTest3, wTestCleanup3 := createWorkspace(t, client, orgTest)
	t.Cleanup(wTestCleanup3)
	wTest4, wTestCleanup4 := createWorkspace(t, client, orgTest)
	t.Cleanup(wTestCleanup4)

	t.Run("without queued runs", func(t *testing.T) {
		c, err := client.Organizations.ReadCapacity(ctx, orgTest.Name)
		require.NoError(t, err)
		assert.Equal(t, 0, c.Pending)
		assert.Equal(t, 0, c.Running)
	})

	// For this test FRQ should be enabled and have a
	// limit of 2 concurrent runs per organization.
	t.Run("with queued runs", func(t *testing.T) {
		_, runCleanup1 := createRun(t, client, wTest1)
		t.Cleanup(runCleanup1)
		_, runCleanup2 := createRun(t, client, wTest2)
		t.Cleanup(runCleanup2)
		_, runCleanup3 := createRun(t, client, wTest3)
		t.Cleanup(runCleanup3)
		_, runCleanup4 := createRun(t, client, wTest4)
		t.Cleanup(runCleanup4)

		c, err := client.Organizations.ReadCapacity(ctx, orgTest.Name)
		require.NoError(t, err)
		assert.Equal(t, 2, c.Pending)
		assert.Equal(t, 2, c.Running)
	})

	t.Run("with invalid name", func(t *testing.T) {
		org, err := client.Organizations.Read(ctx, badIdentifier)
		assert.Nil(t, org)
		assert.EqualError(t, err, ErrInvalidOrg.Error())
	})

	t.Run("when the org does not exist", func(t *testing.T) {
		_, err := client.Organizations.Read(ctx, randomString(t))
		assert.Error(t, err)
	})
}

func TestOrganizationsReadEntitlements(t *testing.T) {
	t.Parallel()
	skipIfEnterprise(t)

	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	t.Cleanup(orgTestCleanup)

	newSubscriptionUpdater(orgTest).WithStandardEntitlementPlan().Update(t)

	t.Run("when the org exists", func(t *testing.T) {
		entitlements, err := client.Organizations.ReadEntitlements(ctx, orgTest.Name)
		require.NoError(t, err)

		assert.NotEmpty(t, entitlements.ID)
		assert.True(t, entitlements.Agents)
		assert.True(t, entitlements.AuditLogging)
		assert.True(t, entitlements.CostEstimation)
		assert.True(t, entitlements.Operations)
		assert.True(t, entitlements.PrivateModuleRegistry)
		assert.True(t, entitlements.SSO)
		assert.True(t, entitlements.Sentinel)
		assert.True(t, entitlements.StateStorage)
		assert.True(t, entitlements.Teams)
		assert.True(t, entitlements.VCSIntegrations)
		assert.False(t, entitlements.WaypointActions)
		assert.True(t, entitlements.WaypointTemplatesAndAddons)
	})

	t.Run("with invalid name", func(t *testing.T) {
		entitlements, err := client.Organizations.ReadEntitlements(ctx, badIdentifier)
		assert.Nil(t, entitlements)
		assert.EqualError(t, err, ErrInvalidOrg.Error())
	})

	t.Run("when the org does not exist", func(t *testing.T) {
		_, err := client.Organizations.ReadEntitlements(ctx, randomString(t))
		assert.Equal(t, ErrResourceNotFound, err)
	})
}

func TestOrganizationsReadRunQueue_RunDependent(t *testing.T) {
	t.Parallel()
	t.Skip("Capacity queues are not available in the API")
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	t.Cleanup(orgTestCleanup)

	wTest1, wTestCleanup1 := createWorkspace(t, client, orgTest)
	t.Cleanup(wTestCleanup1)
	wTest2, wTestCleanup2 := createWorkspace(t, client, orgTest)
	t.Cleanup(wTestCleanup2)
	wTest3, wTestCleanup3 := createWorkspace(t, client, orgTest)
	t.Cleanup(wTestCleanup3)
	wTest4, wTestCleanup4 := createWorkspace(t, client, orgTest)
	t.Cleanup(wTestCleanup4)

	t.Run("without queued runs", func(t *testing.T) {
		rq, err := client.Organizations.ReadRunQueue(ctx, orgTest.Name, ReadRunQueueOptions{})
		require.NoError(t, err)
		assert.Equal(t, 0, len(rq.Items))
	})

	// Create a couple or runs to fill the queue.
	rTest1, rTestCleanup1 := createRun(t, client, wTest1)
	t.Cleanup(rTestCleanup1)
	rTest2, rTestCleanup2 := createRun(t, client, wTest2)
	t.Cleanup(rTestCleanup2)
	rTest3, rTestCleanup3 := createRun(t, client, wTest3)
	t.Cleanup(rTestCleanup3)
	rTest4, rTestCleanup4 := createRun(t, client, wTest4)
	t.Cleanup(rTestCleanup4)

	// For this test FRQ should be enabled and have a
	// limit of 2 concurrent runs per organization.
	t.Run("with queued runs", func(t *testing.T) {
		rq, err := client.Organizations.ReadRunQueue(ctx, orgTest.Name, ReadRunQueueOptions{})
		require.NoError(t, err)

		found := []string{}
		for _, r := range rq.Items {
			found = append(found, r.ID)
		}

		assert.Contains(t, found, rTest1.ID)
		assert.Contains(t, found, rTest2.ID)
		assert.Contains(t, found, rTest3.ID)
		assert.Contains(t, found, rTest4.ID)
	})

	t.Run("without queue options", func(t *testing.T) {
		rq, err := client.Organizations.ReadRunQueue(ctx, orgTest.Name, ReadRunQueueOptions{})
		require.NoError(t, err)

		found := []string{}
		for _, r := range rq.Items {
			found = append(found, r.ID)
		}

		assert.Contains(t, found, rTest1.ID)
		assert.Contains(t, found, rTest2.ID)
		assert.Contains(t, found, rTest3.ID)
		assert.Contains(t, found, rTest4.ID)
		assert.Equal(t, 1, rq.CurrentPage)
		assert.Equal(t, 4, rq.TotalCount)
	})

	t.Run("with queue options", func(t *testing.T) {
		// Request a page number which is out of range. The result should
		// be successful, but return no results if the paging options are
		// properly passed along.
		rq, err := client.Organizations.ReadRunQueue(ctx, orgTest.Name, ReadRunQueueOptions{
			ListOptions: ListOptions{
				PageNumber: 999,
				PageSize:   100,
			},
		})
		require.NoError(t, err)

		assert.Empty(t, rq.Items)
		assert.Equal(t, 999, rq.CurrentPage)
		assert.Equal(t, 4, rq.TotalCount)
	})

	t.Run("with invalid name", func(t *testing.T) {
		org, err := client.Organizations.Read(ctx, badIdentifier)
		assert.Nil(t, org)
		assert.EqualError(t, err, ErrInvalidOrg.Error())
	})

	t.Run("when the org does not exist", func(t *testing.T) {
		_, err := client.Organizations.Read(ctx, randomString(t))
		assert.Error(t, err)
	})
}

func TestOrganization_Unmarshal(t *testing.T) {
	t.Parallel()
	data := map[string]interface{}{
		"data": map[string]interface{}{
			"type": "organizations",
			"id":   "org-name",
			"attributes": map[string]interface{}{
				"assessments-enforced":     true,
				"collaborator-auth-policy": AuthPolicyPassword,
				"cost-estimation-enabled":  true,
				"created-at":               "2018-03-02T23:42:06.651Z",
				"email":                    "test@hashicorp.com",
				"permissions": map[string]interface{}{
					"can-create-team": true,
				},
			},
		},
	}
	byteData, err := json.Marshal(data)
	require.NoError(t, err)

	responseBody := bytes.NewReader(byteData)
	org := &Organization{}
	err = unmarshalResponse(responseBody, org)
	require.NoError(t, err)

	iso8601TimeFormat := "2006-01-02T15:04:05Z"
	parsedTime, err := time.Parse(iso8601TimeFormat, "2018-03-02T23:42:06.651Z")
	require.NoError(t, err)
	assert.Equal(t, org.Name, "org-name")
	assert.Equal(t, org.AssessmentsEnforced, true)
	assert.Equal(t, org.CreatedAt, parsedTime)
	assert.Equal(t, org.CollaboratorAuthPolicy, AuthPolicyPassword)
	assert.Equal(t, org.CostEstimationEnabled, true)
	assert.Equal(t, org.Email, "test@hashicorp.com")
	assert.NotEmpty(t, org.Permissions)
	assert.Equal(t, org.Permissions.CanCreateTeam, true)
}

func TestOrganizationsReadRunTasksPermission(t *testing.T) {
	t.Parallel()
	skipUnlessBeta(t)

	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	t.Cleanup(orgTestCleanup)

	t.Run("when the org exists", func(t *testing.T) {
		org, err := client.Organizations.Read(ctx, orgTest.Name)
		require.NoError(t, err)
		assert.Equal(t, orgTest, org)
		assert.NotEmpty(t, org.Permissions)

		t.Run("permissions are properly decoded", func(t *testing.T) {
			assert.True(t, org.Permissions.CanManageRunTasks)
		})
	})
}

func TestOrganizationsReadRunTasksEntitlement(t *testing.T) {
	t.Parallel()
	skipIfEnterprise(t)
	skipUnlessBeta(t)

	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	t.Cleanup(orgTestCleanup)

	t.Run("when the org exists", func(t *testing.T) {
		entitlements, err := client.Organizations.ReadEntitlements(ctx, orgTest.Name)
		require.NoError(t, err)

		assert.NotEmpty(t, entitlements.ID)
		assert.True(t, entitlements.RunTasks)
	})
}

func TestOrganizationsAllowForceDeleteSetting(t *testing.T) {
	t.Parallel()
	client := testClient(t)
	ctx := context.Background()

	t.Run("creates and updates allow force delete", func(t *testing.T) {
		options := OrganizationCreateOptions{
			Name:                       String(randomString(t)),
			Email:                      String(randomString(t) + "@tfe.local"),
			AllowForceDeleteWorkspaces: Bool(true),
		}

		org, err := client.Organizations.Create(ctx, options)
		require.NoError(t, err)

		t.Cleanup(func() {
			err := client.Organizations.Delete(ctx, org.Name)
			if err != nil {
				t.Errorf("error deleting organization (%s): %s", org.Name, err)
			}
		})

		assert.Equal(t, *options.Name, org.Name)
		assert.Equal(t, *options.Email, org.Email)
		assert.True(t, org.AllowForceDeleteWorkspaces)

		org, err = client.Organizations.Update(ctx, org.Name, OrganizationUpdateOptions{AllowForceDeleteWorkspaces: Bool(false)})
		require.NoError(t, err)
		assert.False(t, org.AllowForceDeleteWorkspaces)

		org, err = client.Organizations.Read(ctx, org.Name)
		require.NoError(t, err)
		assert.False(t, org.AllowForceDeleteWorkspaces)
	})
}

func TestOrganization_DataRetentionPolicy(t *testing.T) {
	t.Parallel()
	skipUnlessEnterprise(t)

	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	t.Cleanup(orgTestCleanup)

	organization, err := client.Organizations.Read(ctx, orgTest.Name)
	require.NoError(t, err)
	require.Nil(t, organization.DataRetentionPolicy)
	require.Nil(t, organization.DataRetentionPolicyChoice)

	dataRetentionPolicy, err := client.Organizations.ReadDataRetentionPolicyChoice(ctx, orgTest.Name)
	require.NoError(t, err)
	require.Nil(t, dataRetentionPolicy)

	t.Run("set and update data retention policy to delete older", func(t *testing.T) {
		createdDataRetentionPolicy, err := client.Organizations.SetDataRetentionPolicyDeleteOlder(ctx, orgTest.Name, DataRetentionPolicyDeleteOlderSetOptions{DeleteOlderThanNDays: 33})
		require.NoError(t, err)
		require.Equal(t, 33, createdDataRetentionPolicy.DeleteOlderThanNDays)
		require.Contains(t, createdDataRetentionPolicy.ID, "drp-")

		dataRetentionPolicy, err = client.Organizations.ReadDataRetentionPolicyChoice(ctx, orgTest.Name)
		require.NoError(t, err)
		require.NotNil(t, dataRetentionPolicy.DataRetentionPolicyDeleteOlder)

		require.Equal(t, 33, dataRetentionPolicy.DataRetentionPolicyDeleteOlder.DeleteOlderThanNDays)
		require.Equal(t, createdDataRetentionPolicy.ID, dataRetentionPolicy.DataRetentionPolicyDeleteOlder.ID)
		require.Contains(t, dataRetentionPolicy.DataRetentionPolicyDeleteOlder.ID, "drp-")

		organization, err := client.Organizations.Read(ctx, orgTest.Name)
		require.NoError(t, err)
		require.Equal(t, dataRetentionPolicy.DataRetentionPolicyDeleteOlder.ID, organization.DataRetentionPolicyChoice.DataRetentionPolicyDeleteOlder.ID)

		// deprecated DataRetentionPolicy field should also have been populated
		require.NotNil(t, organization.DataRetentionPolicy)
		require.Equal(t, organization.DataRetentionPolicy.ID, dataRetentionPolicy.DataRetentionPolicyDeleteOlder.ID)

		// try updating the number of days
		createdDataRetentionPolicy, err = client.Organizations.SetDataRetentionPolicyDeleteOlder(ctx, orgTest.Name, DataRetentionPolicyDeleteOlderSetOptions{DeleteOlderThanNDays: 1})
		require.NoError(t, err)
		require.Equal(t, 1, createdDataRetentionPolicy.DeleteOlderThanNDays)

		dataRetentionPolicy, err = client.Organizations.ReadDataRetentionPolicyChoice(ctx, orgTest.Name)
		require.NoError(t, err)
		require.NotNil(t, dataRetentionPolicy.DataRetentionPolicyDeleteOlder)
		require.Equal(t, 1, dataRetentionPolicy.DataRetentionPolicyDeleteOlder.DeleteOlderThanNDays)
		require.Equal(t, createdDataRetentionPolicy.ID, dataRetentionPolicy.DataRetentionPolicyDeleteOlder.ID)
	})

	t.Run("set data retention policy to not delete", func(t *testing.T) {
		createdDataRetentionPolicy, err := client.Organizations.SetDataRetentionPolicyDontDelete(ctx, orgTest.Name, DataRetentionPolicyDontDeleteSetOptions{})
		require.NoError(t, err)
		require.Contains(t, createdDataRetentionPolicy.ID, "drp-")

		dataRetentionPolicy, err = client.Organizations.ReadDataRetentionPolicyChoice(ctx, orgTest.Name)
		require.NoError(t, err)
		require.NotNil(t, dataRetentionPolicy.DataRetentionPolicyDontDelete)
		require.Equal(t, createdDataRetentionPolicy.ID, dataRetentionPolicy.DataRetentionPolicyDontDelete.ID)

		// dont delete policies should leave the legacy DataRetentionPolicy field on organizations empty
		organization, err := client.Organizations.Read(ctx, orgTest.Name)
		require.NoError(t, err)
		require.Nil(t, organization.DataRetentionPolicy)
	})

	t.Run("change data retention policy type", func(t *testing.T) {
		_, err = client.Organizations.SetDataRetentionPolicyDeleteOlder(ctx, orgTest.Name, DataRetentionPolicyDeleteOlderSetOptions{DeleteOlderThanNDays: 45})
		require.NoError(t, err)

		dataRetentionPolicy, err = client.Organizations.ReadDataRetentionPolicyChoice(ctx, orgTest.Name)
		require.NoError(t, err)
		require.NotNil(t, dataRetentionPolicy.DataRetentionPolicyDeleteOlder)
		require.Equal(t, 45, dataRetentionPolicy.DataRetentionPolicyDeleteOlder.DeleteOlderThanNDays)
		require.Nil(t, dataRetentionPolicy.DataRetentionPolicyDontDelete)

		_, err = client.Organizations.SetDataRetentionPolicyDontDelete(ctx, orgTest.Name, DataRetentionPolicyDontDeleteSetOptions{})
		require.NoError(t, err)
		dataRetentionPolicy, err = client.Organizations.ReadDataRetentionPolicyChoice(ctx, orgTest.Name)
		require.NoError(t, err)
		require.Nil(t, dataRetentionPolicy.DataRetentionPolicyDeleteOlder)
		require.NotNil(t, dataRetentionPolicy.DataRetentionPolicyDontDelete)

		_, err = client.Organizations.SetDataRetentionPolicyDeleteOlder(ctx, orgTest.Name, DataRetentionPolicyDeleteOlderSetOptions{DeleteOlderThanNDays: 20})
		require.NoError(t, err)

		dataRetentionPolicy, err = client.Organizations.ReadDataRetentionPolicyChoice(ctx, orgTest.Name)
		require.NoError(t, err)
		require.NotNil(t, dataRetentionPolicy.DataRetentionPolicyDeleteOlder)
		require.Equal(t, 20, dataRetentionPolicy.DataRetentionPolicyDeleteOlder.DeleteOlderThanNDays)
		require.Nil(t, dataRetentionPolicy.DataRetentionPolicyDontDelete)
	})

	t.Run("delete data retention policy", func(t *testing.T) {
		err = client.Organizations.DeleteDataRetentionPolicy(ctx, orgTest.Name)
		require.NoError(t, err)

		dataRetentionPolicy, err = client.Organizations.ReadDataRetentionPolicyChoice(ctx, orgTest.Name)
		assert.Nil(t, err)
		require.Nil(t, dataRetentionPolicy)
	})
}

func orgItemsContainsName(items []*Organization, name string) bool {
	hasName := false
	for _, item := range items {
		if item.Name == name {
			hasName = true
			break
		}
	}

	return hasName
}

func orgItemsContainsEmail(items []*Organization, email string) bool {
	hasEmail := false
	for _, item := range items {
		if item.Email == email {
			hasEmail = true
			break
		}
	}

	return hasEmail
}
