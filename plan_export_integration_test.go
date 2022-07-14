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

func TestPlanExportsCreate(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	rTest, rTestCleanup := createPlannedRun(t, client, nil)
	defer rTestCleanup()

	pTest, err := client.Plans.Read(ctx, rTest.Plan.ID)
	require.NoError(t, err)

	t.Run("with valid options", func(t *testing.T) {
		options := PlanExportCreateOptions{
			Plan:     pTest,
			DataType: PlanExportType(PlanExportSentinelMockBundleV0),
		}

		pe, err := client.PlanExports.Create(ctx, options)
		require.NoError(t, err)
		assert.NotEmpty(t, pe.ID)
		assert.Equal(t, PlanExportSentinelMockBundleV0, pe.DataType)
	})

	t.Run("without a plan", func(t *testing.T) {
		options := PlanExportCreateOptions{
			Plan:     nil,
			DataType: PlanExportType(PlanExportSentinelMockBundleV0),
		}

		pe, err := client.PlanExports.Create(ctx, options)
		assert.Nil(t, pe)
		assert.Equal(t, err, ErrRequiredPlan)
	})

	t.Run("without a data type", func(t *testing.T) {
		options := PlanExportCreateOptions{
			Plan:     pTest,
			DataType: nil,
		}

		pe, err := client.PlanExports.Create(ctx, options)
		assert.Nil(t, pe)
		assert.Equal(t, err, ErrRequiredDataType)
	})
}

func TestPlanExportsRead(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	peTest, peTestCleanup := createPlanExport(t, client, nil)
	defer peTestCleanup()

	t.Run("with a valid ID", func(t *testing.T) {
		pe, err := client.PlanExports.Read(ctx, peTest.ID)
		require.NoError(t, err)
		assert.Equal(t, peTest.ID, pe.ID)
		assert.Equal(t, peTest.DataType, pe.DataType)
		assert.NotEmpty(t, pe.StatusTimestamps)
		assert.NotNil(t, pe.StatusTimestamps.QueuedAt)
	})

	t.Run("without a valid ID", func(t *testing.T) {
		pe, err := client.PlanExports.Read(ctx, badIdentifier)
		assert.Nil(t, pe)
		assert.Equal(t, err, ErrInvalidPlanExportID)
	})
}

func TestPlanExportsDelete(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	peTest, peTestCleanup := createPlanExport(t, client, nil)
	defer peTestCleanup()

	t.Run("with a valid ID", func(t *testing.T) {
		err := client.PlanExports.Delete(ctx, peTest.ID)
		require.NoError(t, err)
	})

	t.Run("when the export does not exist", func(t *testing.T) {
		err := client.Policies.Delete(ctx, "pe-doesntexist")
		assert.Equal(t, err, ErrResourceNotFound)
	})

	t.Run("without a valid ID", func(t *testing.T) {
		err := client.PlanExports.Delete(ctx, badIdentifier)
		assert.Equal(t, err, ErrInvalidPlanExportID)
	})
}

func TestPlanExportsDownload(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	peTest, peCleanup := createPlanExport(t, client, nil)
	defer peCleanup()

	t.Run("with a valid ID", func(t *testing.T) {
		pe, err := client.PlanExports.Download(ctx, peTest.ID)
		assert.NotNil(t, pe)
		require.NoError(t, err)
	})

	t.Run("without a valid ID", func(t *testing.T) {
		pe, err := client.PlanExports.Download(ctx, badIdentifier)
		assert.Nil(t, pe)
		assert.Equal(t, err, ErrInvalidPlanExportID)
	})
}

func TestPlanExport_Unmarshal(t *testing.T) {
	data := map[string]interface{}{
		"data": map[string]interface{}{
			"type": "plan-exports",
			"id":   "1",
			"attributes": map[string]interface{}{
				"data-type": PlanExportSentinelMockBundleV0,
				"status":    PlanExportCanceled,
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
	pe := &PlanExport{}
	err = unmarshalResponse(responseBody, pe)
	require.NoError(t, err)

	queuedParsedTime, err := time.Parse(time.RFC3339, "2020-03-16T23:15:59+00:00")
	require.NoError(t, err)
	erroredParsedTime, err := time.Parse(time.RFC3339, "2019-03-16T23:23:59+00:00")
	require.NoError(t, err)

	assert.Equal(t, pe.DataType, PlanExportSentinelMockBundleV0)
	assert.Equal(t, pe.Status, PlanExportCanceled)
	assert.NotEmpty(t, pe.StatusTimestamps)
	assert.Equal(t, pe.StatusTimestamps.QueuedAt, queuedParsedTime)
	assert.Equal(t, pe.StatusTimestamps.ErroredAt, erroredParsedTime)
}
