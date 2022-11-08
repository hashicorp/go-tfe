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

func TestCostEstimatesRead(t *testing.T) {
	skipIfEnterprise(t)
	skipIfFreeOnly(t)

	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	defer orgTestCleanup()

	// Enable cost estimation for the test organization.
	orgTest, err := client.Organizations.Update(
		ctx,
		orgTest.Name,
		OrganizationUpdateOptions{
			CostEstimationEnabled: Bool(true),
		},
	)
	require.NoError(t, err)

	wTest, wTestCleanup := createWorkspace(t, client, orgTest)
	defer wTestCleanup()
	rTest, rTestCleanup := createCostEstimatedRun(t, client, wTest)
	defer rTestCleanup()

	t.Run("when the costEstimate exists", func(t *testing.T) {
		ce, err := client.CostEstimates.Read(ctx, rTest.CostEstimate.ID)
		require.NoError(t, err)
		assert.Equal(t, ce.Status, CostEstimateFinished)
		assert.NotEmpty(t, ce.StatusTimestamps)
	})

	t.Run("when the costEstimate does not exist", func(t *testing.T) {
		ce, err := client.CostEstimates.Read(ctx, "nonexisting")
		assert.Nil(t, ce)
		assert.Equal(t, ErrResourceNotFound, err)
	})

	t.Run("with invalid costEstimate ID", func(t *testing.T) {
		ce, err := client.CostEstimates.Read(ctx, badIdentifier)
		assert.Nil(t, ce)
		assert.EqualError(t, err, ErrInvalidCostEstimateID.Error())
	})
}

func TestCostEsimate_Unmarshal(t *testing.T) {
	data := map[string]interface{}{
		"data": map[string]interface{}{
			"type": "cost-estimates",
			"id":   "ce-ntv3HbhJqvFzamy7",
			"attributes": map[string]interface{}{
				"delta-monthly-cost":      "100",
				"error-message":           "message",
				"matched-resources-count": 1,
				"prior-monthly-cost":      "100",
				"proposed-monthly-cost":   "100",
				"resources-count":         1,
				"status":                  CostEstimateCanceled,
				"status-timestamps": map[string]string{
					"queued-at":  "2020-03-16T23:15:59+00:00",
					"errored-at": "2019-03-16T23:23:59+00:00",
				},
			},
		},
	}
	byteData, err := json.Marshal(data)
	require.NoError(t, err)

	queuedParsedTime, err := time.Parse(time.RFC3339, "2020-03-16T23:15:59+00:00")
	require.NoError(t, err)
	erroredParsedTime, err := time.Parse(time.RFC3339, "2019-03-16T23:23:59+00:00")
	require.NoError(t, err)

	responseBody := bytes.NewReader(byteData)
	ce := &CostEstimate{}
	err = unmarshalResponse(responseBody, ce)
	require.NoError(t, err)

	assert.Equal(t, ce.ID, "ce-ntv3HbhJqvFzamy7")
	assert.Equal(t, ce.DeltaMonthlyCost, "100")
	assert.Equal(t, ce.ErrorMessage, "message")
	assert.Equal(t, ce.MatchedResourcesCount, 1)
	assert.Equal(t, ce.PriorMonthlyCost, "100")
	assert.Equal(t, ce.ProposedMonthlyCost, "100")
	assert.Equal(t, ce.ResourcesCount, 1)
	assert.Equal(t, ce.Status, CostEstimateCanceled)
	assert.Equal(t, ce.StatusTimestamps.QueuedAt, queuedParsedTime)
	assert.Equal(t, ce.StatusTimestamps.ErroredAt, erroredParsedTime)
}
