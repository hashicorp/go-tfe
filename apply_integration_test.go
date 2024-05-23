// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package tfe

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAppliesRead(t *testing.T) {
	client := testClient(t)
	acquireRunMutex(t, client)

	ctx := context.Background()

	wTest, wTestCleanup := createWorkspace(t, client, nil)
	defer wTestCleanup()

	rTest, rTestCleanup := createRunApply(t, client, wTest)
	defer rTestCleanup()

	t.Run("when the plan exists", func(t *testing.T) {
		a, err := client.Applies.Read(ctx, rTest.Apply.ID)
		require.NoError(t, err)
		assert.NotEmpty(t, a.LogReadURL)
		assert.Equal(t, a.Status, ApplyFinished)
		assert.NotEmpty(t, a.StatusTimestamps)
	})

	t.Run("when the apply does not exist", func(t *testing.T) {
		a, err := client.Applies.Read(ctx, "nonexisting")
		assert.Nil(t, a)
		assert.Equal(t, err, ErrResourceNotFound)
	})

	t.Run("with invalid apply ID", func(t *testing.T) {
		a, err := client.Applies.Read(ctx, badIdentifier)
		assert.Nil(t, a)
		assert.EqualError(t, err, ErrInvalidApplyID.Error())
	})
}

func TestAppliesLogs(t *testing.T) {
	client := testClient(t)
	acquireRunMutex(t, client)

	ctx := context.Background()

	rTest, rTestCleanup := createRunApply(t, client, nil)
	defer rTestCleanup()

	t.Run("when the log exists", func(t *testing.T) {
		a, err := client.Applies.Read(ctx, rTest.Apply.ID)
		require.NoError(t, err)

		logReader, err := client.Applies.Logs(ctx, a.ID)
		require.NoError(t, err)

		logs, err := io.ReadAll(logReader)
		require.NoError(t, err)

		assert.Contains(t, string(logs), "1 added, 0 changed, 0 destroyed")
	})

	t.Run("when the log does not exist", func(t *testing.T) {
		logs, err := client.Applies.Logs(ctx, "nonexisting")
		assert.Nil(t, logs)
		assert.Error(t, err)
	})
}

func TestApplies_Unmarshal(t *testing.T) {
	data := map[string]interface{}{
		"data": map[string]interface{}{
			"type": "applies",
			"id":   "apply-47MBvjwzBG8YKc2v",
			"attributes": map[string]interface{}{
				"log-read-url":          "hashicorp.com",
				"resource-additions":    1,
				"resource-changes":      1,
				"resource-destructions": 1,
				"status":                ApplyCanceled,
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

	apply := &Apply{}
	err = unmarshalResponse(responseBody, apply)
	require.NoError(t, err)

	queuedParsedTime, err := time.Parse(time.RFC3339, "2020-03-16T23:15:59+00:00")
	require.NoError(t, err)
	erroredParsedTime, err := time.Parse(time.RFC3339, "2019-03-16T23:23:59+00:00")
	require.NoError(t, err)

	assert.Equal(t, apply.ID, "apply-47MBvjwzBG8YKc2v")
	assert.Equal(t, apply.ResourceAdditions, 1)
	assert.Equal(t, apply.ResourceChanges, 1)
	assert.Equal(t, apply.ResourceDestructions, 1)
	assert.Equal(t, apply.Status, ApplyCanceled)
	assert.Equal(t, apply.StatusTimestamps.QueuedAt, queuedParsedTime)
	assert.Equal(t, apply.StatusTimestamps.ErroredAt, erroredParsedTime)
}
