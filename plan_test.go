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

func TestPlansRead(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	rTest, rTestCleanup := createPlannedRun(t, client, nil)
	defer rTestCleanup()

	t.Run("when the plan exists", func(t *testing.T) {
		p, err := client.Plans.Read(ctx, rTest.Plan.ID)
		require.NoError(t, err)
		assert.True(t, p.HasChanges)
		assert.NotEmpty(t, p.LogReadURL)
		assert.Equal(t, p.Status, PlanFinished)
		assert.NotEmpty(t, p.StatusTimestamps)
		assert.NotNil(t, p.StatusTimestamps.StartedAt)
	})

	t.Run("when the plan does not exist", func(t *testing.T) {
		p, err := client.Plans.Read(ctx, "nonexisting")
		assert.Nil(t, p)
		assert.Equal(t, err, ErrResourceNotFound)
	})

	t.Run("with invalid plan ID", func(t *testing.T) {
		p, err := client.Plans.Read(ctx, badIdentifier)
		assert.Nil(t, p)
		assert.EqualError(t, err, "invalid value for plan ID")
	})
}

func TestPlansLogs(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	rTest, rTestCleanup := createPlannedRun(t, client, nil)
	defer rTestCleanup()

	t.Run("when the log exists", func(t *testing.T) {
		p, err := client.Plans.Read(ctx, rTest.Plan.ID)
		require.NoError(t, err)

		logReader, err := client.Plans.Logs(ctx, p.ID)
		require.NoError(t, err)

		logs, err := ioutil.ReadAll(logReader)
		require.NoError(t, err)

		assert.Contains(t, string(logs), "1 to add, 0 to change, 0 to destroy")
	})

	t.Run("when the log does not exist", func(t *testing.T) {
		logs, err := client.Plans.Logs(ctx, "nonexisting")
		assert.Nil(t, logs)
		assert.Error(t, err)
	})
}

func TestPlan_Unmarshal(t *testing.T) {
	data := map[string]interface{}{
		"data": map[string]interface{}{
			"type": "plans",
			"id":   "1",
			"attributes": map[string]interface{}{
				"has-changes":           true,
				"log-read-url":          "hashicorp.com",
				"resource-additions":    1,
				"resource-changes":      1,
				"resource-destructions": 1,
				"status":                PlanCanceled,
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
	plan := &Plan{}
	err = unmarshalResponse(responseBody, plan)
	require.NoError(t, err)

	queuedParsedTime, err := time.Parse(time.RFC3339, "2020-03-16T23:15:59+00:00")
	require.NoError(t, err)
	erroredParsedTime, err := time.Parse(time.RFC3339, "2019-03-16T23:23:59+00:00")
	require.NoError(t, err)

	assert.Equal(t, plan.HasChanges, true)
	assert.Equal(t, plan.LogReadURL, "hashicorp.com")
	assert.Equal(t, plan.ResourceAdditions, 1)
	assert.Equal(t, plan.ResourceChanges, 1)
	assert.Equal(t, plan.ResourceDestructions, 1)
	assert.Equal(t, plan.Status, PlanCanceled)
	assert.NotEmpty(t, plan.StatusTimestamps)
	assert.Equal(t, plan.StatusTimestamps.QueuedAt, queuedParsedTime)
	assert.Equal(t, plan.StatusTimestamps.ErroredAt, erroredParsedTime)
}
