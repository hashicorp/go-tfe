package tfe

import (
	"bytes"
	"context"
	"encoding/json"
	"io/ioutil"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPolicyChecksList(t *testing.T) {
	skipIfFreeOnly(t)

	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	defer orgTestCleanup()

	pTest1, policyCleanup1 := createUploadedPolicy(t, client, true, orgTest)
	defer policyCleanup1()
	pTest2, policyCleanup2 := createUploadedPolicy(t, client, true, orgTest)
	defer policyCleanup2()
	wTest, wsCleanup := createWorkspace(t, client, orgTest)
	defer wsCleanup()
	createPolicySet(t, client, orgTest, []*Policy{pTest1, pTest2}, []*Workspace{wTest})

	rTest, runCleanup := createPlannedRun(t, client, wTest)
	defer runCleanup()

	t.Run("without list options", func(t *testing.T) {
		pcl, err := client.PolicyChecks.List(ctx, rTest.ID, PolicyCheckListOptions{})
		require.NoError(t, err)
		require.Equal(t, 1, len(pcl.Items))
		assert.NotEmpty(t, pcl.Items[0].Permissions)
		require.NotEmpty(t, pcl.Items[0].Result)
		assert.Equal(t, 2, pcl.Items[0].Result.Passed)
		assert.NotEmpty(t, pcl.Items[0].StatusTimestamps)
		assert.NotNil(t, pcl.Items[0].StatusTimestamps.QueuedAt)
	})

	t.Run("with list options", func(t *testing.T) {
		t.Skip("paging not supported yet in API")
		// Request a page number which is out of range. The result should
		// be successful, but return no results if the paging options are
		// properly passed along.
		pcl, err := client.PolicyChecks.List(ctx, rTest.ID, PolicyCheckListOptions{
			ListOptions: ListOptions{
				PageNumber: 999,
				PageSize:   100,
			},
		})
		require.NoError(t, err)
		assert.Empty(t, pcl.Items)
		assert.Equal(t, 999, pcl.CurrentPage)
		assert.Equal(t, 1, pcl.TotalCount)
	})

	t.Run("without a valid run ID", func(t *testing.T) {
		pcl, err := client.PolicyChecks.List(ctx, badIdentifier, PolicyCheckListOptions{})
		assert.Nil(t, pcl)
		assert.EqualError(t, err, ErrInvalidRunID.Error())
	})
}

func TestPolicyChecksRead(t *testing.T) {
	skipIfEnterprise(t)
	skipIfFreeOnly(t)

	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	defer orgTestCleanup()

	pTest, _ := createUploadedPolicy(t, client, true, orgTest)
	wTest, _ := createWorkspace(t, client, orgTest)
	createPolicySet(t, client, orgTest, []*Policy{pTest}, []*Workspace{wTest})

	rTest, _ := createPlannedRun(t, client, wTest)
	require.Equal(t, 1, len(rTest.PolicyChecks))

	t.Run("when the policy check exists", func(t *testing.T) {
		pc, err := client.PolicyChecks.Read(ctx, rTest.PolicyChecks[0].ID)
		require.NoError(t, err)

		require.NotEmpty(t, pc.Result)
		assert.NotEmpty(t, pc.Permissions)
		assert.Equal(t, PolicyScopeOrganization, pc.Scope)
		assert.Equal(t, PolicyPasses, pc.Status)
		assert.NotEmpty(t, pc.StatusTimestamps)
		assert.Equal(t, 1, pc.Result.Passed)
		assert.NotEmpty(t, pc.Run)
	})

	t.Run("when the policy check does not exist", func(t *testing.T) {
		pc, err := client.PolicyChecks.Read(ctx, "nonexisting")
		assert.Nil(t, pc)
		assert.Equal(t, ErrResourceNotFound, err)
	})

	t.Run("without a valid policy check ID", func(t *testing.T) {
		pc, err := client.PolicyChecks.Read(ctx, badIdentifier)
		assert.Nil(t, pc)
		assert.EqualError(t, err, "invalid value for policy check ID")
	})
}

func TestPolicyChecksOverride(t *testing.T) {
	skipIfFreeOnly(t)

	client := testClient(t)
	ctx := context.Background()

	t.Run("when the policy failed", func(t *testing.T) {
		orgTest, orgTestCleanup := createOrganization(t, client)
		defer orgTestCleanup()

		pTest, pTestCleanup := createUploadedPolicy(t, client, false, orgTest)
		defer pTestCleanup()

		wTest, wTestCleanup := createWorkspace(t, client, orgTest)
		defer wTestCleanup()
		createPolicySet(t, client, orgTest, []*Policy{pTest}, []*Workspace{wTest})
		rTest, tTestCleanup := createPlannedRun(t, client, wTest)
		defer tTestCleanup()

		pcl, err := client.PolicyChecks.List(ctx, rTest.ID, PolicyCheckListOptions{})
		require.NoError(t, err)
		require.Equal(t, 1, len(pcl.Items))
		require.Equal(t, PolicySoftFailed, pcl.Items[0].Status)

		pc, err := client.PolicyChecks.Override(ctx, pcl.Items[0].ID)
		require.NoError(t, err)

		assert.NotEmpty(t, pc.Result)
		assert.Equal(t, PolicyOverridden, pc.Status)
	})

	t.Run("when the policy passed", func(t *testing.T) {
		orgTest, orgTestCleanup := createOrganization(t, client)
		defer orgTestCleanup()

		pTest, pTestCleanup := createUploadedPolicy(t, client, true, orgTest)
		defer pTestCleanup()

		wTest, wTestCleanup := createWorkspace(t, client, orgTest)
		defer wTestCleanup()
		createPolicySet(t, client, orgTest, []*Policy{pTest}, []*Workspace{wTest})
		rTest, rTestCleanup := createPlannedRun(t, client, wTest)
		defer rTestCleanup()

		pcl, err := client.PolicyChecks.List(ctx, rTest.ID, PolicyCheckListOptions{})
		require.NoError(t, err)
		require.Equal(t, 1, len(pcl.Items))
		require.Equal(t, PolicyPasses, pcl.Items[0].Status)

		_, err = client.PolicyChecks.Override(ctx, pcl.Items[0].ID)
		assert.Error(t, err)
	})

	t.Run("without a valid policy check ID", func(t *testing.T) {
		p, err := client.PolicyChecks.Override(ctx, badIdentifier)
		assert.Nil(t, p)
		assert.EqualError(t, err, "invalid value for policy check ID")
	})
}

func TestPolicyChecksLogs(t *testing.T) {
	skipIfFreeOnly(t)

	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	defer orgTestCleanup()

	pTest, pTestCleanup := createUploadedPolicy(t, client, true, orgTest)
	defer pTestCleanup()
	wTest, wTestCleanup := createWorkspace(t, client, orgTest)
	defer wTestCleanup()
	createPolicySet(t, client, orgTest, []*Policy{pTest}, []*Workspace{wTest})

	rTest, rTestCleanup := createPlannedRun(t, client, wTest)
	defer rTestCleanup()
	require.Equal(t, 1, len(rTest.PolicyChecks))

	t.Run("when the log exists", func(t *testing.T) {
		pc, err := client.PolicyChecks.Read(ctx, rTest.PolicyChecks[0].ID)
		require.NoError(t, err)

		logReader, err := client.PolicyChecks.Logs(ctx, pc.ID)
		require.NoError(t, err)

		logs, err := ioutil.ReadAll(logReader)
		require.NoError(t, err)

		assert.Contains(t, string(logs), "1 policies evaluated")
	})

	t.Run("when the log does not exist", func(t *testing.T) {
		logs, err := client.PolicyChecks.Logs(ctx, "nonexisting")
		assert.Nil(t, logs)
		assert.Error(t, err)
	})
}

func TestPolicyCheck_Unmarshal(t *testing.T) {
	data := map[string]interface{}{
		"data": map[string]interface{}{
			"type": "policy-checks",
			"id":   "1",
			"attributes": map[string]interface{}{
				"actions": map[string]interface{}{
					"is-overridable": true,
				},
				"permissions": map[string]interface{}{
					"can-override": true,
				},
				"result": map[string]interface{}{
					"advisory-failed": 1,
					"duration":        1,
					"hard-failed":     1,
					"passed":          1,
					"result":          true,
					"soft-failed":     1,
					"total-failed":    1,
				},
				"scope":  PolicyScopeOrganization,
				"status": PolicyOverridden,
				"status-timestamps": map[string]string{
					"queued-at":  "2020-03-16T23:15:59+00:00",
					"errored-at": "2019-03-16T23:23:59+00:00",
				},
			},
		},
	}

	byteData, err := json.Marshal(data)
	require.NoError(t, err)

	responseBody := bytes.NewReader(byteData)
	pc := &PolicyCheck{}
	err = unmarshalResponse(responseBody, pc)
	require.NoError(t, err)

	queuedParsedTime, err := time.Parse(time.RFC3339, "2020-03-16T23:15:59+00:00")
	require.NoError(t, err)
	erroredParsedTime, err := time.Parse(time.RFC3339, "2019-03-16T23:23:59+00:00")
	require.NoError(t, err)

	assert.Equal(t, pc.ID, "1")
	assert.Equal(t, pc.Actions.IsOverridable, true)
	assert.Equal(t, pc.Permissions.CanOverride, true)
	assert.Equal(t, pc.Result.AdvisoryFailed, 1)
	assert.Equal(t, pc.Result.Duration, 1)
	assert.Equal(t, pc.Result.HardFailed, 1)
	assert.Equal(t, pc.Result.Passed, 1)
	assert.Equal(t, pc.Result.Result, true)
	assert.Equal(t, pc.Result.SoftFailed, 1)
	assert.Equal(t, pc.Result.TotalFailed, 1)
	assert.Equal(t, pc.Scope, PolicyScopeOrganization)
	assert.Equal(t, pc.Status, PolicyOverridden)
	assert.Equal(t, pc.StatusTimestamps.QueuedAt, queuedParsedTime)
	assert.Equal(t, pc.StatusTimestamps.ErroredAt, erroredParsedTime)
}
