// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package tfe

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPlansRead_RunDependent(t *testing.T) {
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
		assert.Equal(t, err, ErrInvalidPlanID)
	})

	t.Run("read hyok encrypted data key of a plan", func(t *testing.T) {
		skipHYOKIntegrationTests(t)

		// replace the environment variable with a valid plan ID that has a hyok encrypted data key
		hyokPlanID := os.Getenv("HYOK_PLAN_ID")
		if hyokPlanID == "" {
			t.Fatal("Export a valid HYOK_PLAN_ID before running this test!")
		}

		p, err := client.Plans.Read(ctx, hyokPlanID)
		require.NoError(t, err)
		assert.NotNil(t, p.HYOKEncryptedDataKey)
	})

	t.Run("read sanitized plan of a plan", func(t *testing.T) {
		skipHYOKIntegrationTests(t)

		// replace the environment variable with a valid plan ID that has a sanitized plan link
		hyokPlanID := os.Getenv("HYOK_PLAN_ID")
		if hyokPlanID == "" {
			t.Fatal("Export a valid HYOK_PLAN_ID before running this test!")
		}

		p, err := client.Plans.Read(ctx, hyokPlanID)
		require.NoError(t, err)
		assert.NotEmpty(t, p.Links["sanitized-plan"])
	})
}

func TestPlansLogs_RunDependent(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	rTest, rTestCleanup := createPlannedRun(t, client, nil)
	defer rTestCleanup()

	t.Run("when the log exists", func(t *testing.T) {
		p, err := client.Plans.Read(ctx, rTest.Plan.ID)
		require.NoError(t, err)

		logReader, err := client.Plans.Logs(ctx, p.ID)
		require.NoError(t, err)

		logs, err := io.ReadAll(logReader)
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

func TestPlansJSONOutput_RunDependent(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()
	rTest, rTestCleanup := createPlannedRun(t, client, nil)
	defer rTestCleanup()

	t.Run("when the JSON output exists", func(t *testing.T) {
		d, err := client.Plans.ReadJSONOutput(ctx, rTest.Plan.ID)
		require.NoError(t, err)
		var m map[string]interface{}
		err = json.Unmarshal(d, &m)
		require.NoError(t, err)
		assert.Contains(t, m, "planned_values")
		assert.Contains(t, m, "terraform_version")
	})

	t.Run("when the JSON output does not exist", func(t *testing.T) {
		d, err := client.Plans.ReadJSONOutput(ctx, "nonexisting")
		assert.Nil(t, d)
		assert.Error(t, err)
	})
}
