//go:build integration
// +build integration

package tfe

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOrganizationsList(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	orgTest1, orgTest1Cleanup := createOrganization(t, client)
	defer orgTest1Cleanup()
	orgTest2, orgTest2Cleanup := createOrganization(t, client)
	defer orgTest2Cleanup()

	t.Run("with no list options", func(t *testing.T) {
		orgl, err := client.Organizations.List(ctx, OrganizationListOptions{})
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
		orgl, err := client.Organizations.List(ctx, OrganizationListOptions{
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
}

func TestOrganizationsCreate(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	t.Run("with valid options", func(t *testing.T) {
		options := OrganizationCreateOptions{
			Name:  String(randomString(t)),
			Email: String(randomString(t) + "@tfe.local"),
		}

		org, err := client.Organizations.Create(ctx, options)
		require.NoError(t, err)

		// Make sure we clean up the created org.
		defer func() {
			err := client.Organizations.Delete(ctx, org.Name)
			if err != nil {
				t.Logf("error deleting organization (%s): %s", org.Name, err)
			}
		}()

		assert.Equal(t, *options.Name, org.Name)
		assert.Equal(t, *options.Email, org.Email)
	})

	t.Run("when no email is provided", func(t *testing.T) {
		org, err := client.Organizations.Create(ctx, OrganizationCreateOptions{
			Name: String("foo"),
		})
		assert.Nil(t, org)
		assert.EqualError(t, err, "email is required")
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

func TestOrganizationsRead(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	defer orgTestCleanup()

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
}

func TestOrganizationsUpdate(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	t.Run("with valid options", func(t *testing.T) {
		orgTest, orgTestCleanup := createOrganization(t, client)

		options := OrganizationUpdateOptions{
			Name:            String(randomString(t)),
			Email:           String(randomString(t) + "@tfe.local"),
			SessionTimeout:  Int(3600),
			SessionRemember: Int(3600),
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
		}
	})

	t.Run("with invalid name", func(t *testing.T) {
		org, err := client.Organizations.Update(ctx, badIdentifier, OrganizationUpdateOptions{})
		assert.Nil(t, org)
		assert.EqualError(t, err, ErrInvalidOrg.Error())
	})

	t.Run("when only updating a subset of fields", func(t *testing.T) {
		orgTest, orgTestCleanup := createOrganization(t, client)
		defer orgTestCleanup()

		org, err := client.Organizations.Update(ctx, orgTest.Name, OrganizationUpdateOptions{})
		require.NoError(t, err)
		assert.Equal(t, orgTest.Name, org.Name)
		assert.Equal(t, orgTest.Email, org.Email)
	})
}

func TestOrganizationsDelete(t *testing.T) {
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

func TestOrganizationsCapacity(t *testing.T) {
	t.Skip("Capacity queues are not available in the API")
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	defer orgTestCleanup()

	wTest1, wTestCleanup1 := createWorkspace(t, client, orgTest)
	defer wTestCleanup1()
	wTest2, wTestCleanup2 := createWorkspace(t, client, orgTest)
	defer wTestCleanup2()
	wTest3, wTestCleanup3 := createWorkspace(t, client, orgTest)
	defer wTestCleanup3()
	wTest4, wTestCleanup4 := createWorkspace(t, client, orgTest)
	defer wTestCleanup4()

	t.Run("without queued runs", func(t *testing.T) {
		c, err := client.Organizations.Capacity(ctx, orgTest.Name)
		require.NoError(t, err)
		assert.Equal(t, 0, c.Pending)
		assert.Equal(t, 0, c.Running)
	})

	// For this test FRQ should be enabled and have a
	// limit of 2 concurrent runs per organization.
	t.Run("with queued runs", func(t *testing.T) {
		_, runCleanup1 := createRun(t, client, wTest1)
		defer runCleanup1()
		_, runCleanup2 := createRun(t, client, wTest2)
		defer runCleanup2()
		_, runCleanup3 := createRun(t, client, wTest3)
		defer runCleanup3()
		_, runCleanup4 := createRun(t, client, wTest4)
		defer runCleanup4()

		c, err := client.Organizations.Capacity(ctx, orgTest.Name)
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

func TestOrganizationsEntitlements(t *testing.T) {
	skipIfEnterprise(t)
	skipIfFreeOnly(t)

	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	defer orgTestCleanup()

	t.Run("when the org exists", func(t *testing.T) {
		entitlements, err := client.Organizations.Entitlements(ctx, orgTest.Name)
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
	})

	t.Run("with invalid name", func(t *testing.T) {
		entitlements, err := client.Organizations.Entitlements(ctx, badIdentifier)
		assert.Nil(t, entitlements)
		assert.EqualError(t, err, ErrInvalidOrg.Error())
	})

	t.Run("when the org does not exist", func(t *testing.T) {
		_, err := client.Organizations.Entitlements(ctx, randomString(t))
		assert.Equal(t, ErrResourceNotFound, err)
	})
}

func TestOrganizationsRunQueue(t *testing.T) {
	t.Skip("Capacity queues are not available in the API")
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	defer orgTestCleanup()

	wTest1, wTestCleanup1 := createWorkspace(t, client, orgTest)
	defer wTestCleanup1()
	wTest2, wTestCleanup2 := createWorkspace(t, client, orgTest)
	defer wTestCleanup2()
	wTest3, wTestCleanup3 := createWorkspace(t, client, orgTest)
	defer wTestCleanup3()
	wTest4, wTestCleanup4 := createWorkspace(t, client, orgTest)
	defer wTestCleanup4()

	t.Run("without queued runs", func(t *testing.T) {
		rq, err := client.Organizations.RunQueue(ctx, orgTest.Name, RunQueueOptions{})
		require.NoError(t, err)
		assert.Equal(t, 0, len(rq.Items))
	})

	// Create a couple or runs to fill the queue.
	rTest1, rTestCleanup1 := createRun(t, client, wTest1)
	defer rTestCleanup1()
	rTest2, rTestCleanup2 := createRun(t, client, wTest2)
	defer rTestCleanup2()
	rTest3, rTestCleanup3 := createRun(t, client, wTest3)
	defer rTestCleanup3()
	rTest4, rTestCleanup4 := createRun(t, client, wTest4)
	defer rTestCleanup4()

	// For this test FRQ should be enabled and have a
	// limit of 2 concurrent runs per organization.
	t.Run("with queued runs", func(t *testing.T) {
		rq, err := client.Organizations.RunQueue(ctx, orgTest.Name, RunQueueOptions{})
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
		rq, err := client.Organizations.RunQueue(ctx, orgTest.Name, RunQueueOptions{})
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
		rq, err := client.Organizations.RunQueue(ctx, orgTest.Name, RunQueueOptions{
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
	data := map[string]interface{}{
		"data": map[string]interface{}{
			"type": "organizations",
			"id":   "org-name",
			"attributes": map[string]interface{}{
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
	assert.Equal(t, org.CreatedAt, parsedTime)
	assert.Equal(t, org.CollaboratorAuthPolicy, AuthPolicyPassword)
	assert.Equal(t, org.CostEstimationEnabled, true)
	assert.Equal(t, org.Email, "test@hashicorp.com")
	assert.NotEmpty(t, org.Permissions)
	assert.Equal(t, org.Permissions.CanCreateTeam, true)
}
