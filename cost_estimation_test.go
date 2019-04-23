package tfe

import (
	"context"
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCostEstimationsRead(t *testing.T) {
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

	wTest, _ := createWorkspace(t, client, orgTest)
	rTest, _ := createPlannedRun(t, client, wTest)

	t.Run("when the costEstimation exists", func(t *testing.T) {
		p, err := client.CostEstimations.Read(ctx, rTest.CostEstimation.ID)
		require.NoError(t, err)
		assert.NotEmpty(t, p.LogReadURL)
		assert.Equal(t, p.Status, CostEstimationFinished)
		assert.NotEmpty(t, p.StatusTimestamps)
	})

	t.Run("when the costEstimation does not exist", func(t *testing.T) {
		p, err := client.CostEstimations.Read(ctx, "nonexisting")
		assert.Nil(t, p)
		assert.Equal(t, err, ErrResourceNotFound)
	})

	t.Run("with invalid costEstimation ID", func(t *testing.T) {
		p, err := client.CostEstimations.Read(ctx, badIdentifier)
		assert.Nil(t, p)
		assert.EqualError(t, err, "invalid value for cost estimation ID")
	})
}

func TestCostEstimationsLogs(t *testing.T) {
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

	wTest, _ := createWorkspace(t, client, orgTest)
	rTest, _ := createPlannedRun(t, client, wTest)

	t.Run("when the log exists", func(t *testing.T) {
		p, err := client.CostEstimations.Read(ctx, rTest.CostEstimation.ID)
		require.NoError(t, err)

		logReader, err := client.CostEstimations.Logs(ctx, p.ID)
		require.NoError(t, err)

		_, err = ioutil.ReadAll(logReader)
		require.NoError(t, err)

		// skip testing log output since it's likely to change
	})

	t.Run("when the log does not exist", func(t *testing.T) {
		logs, err := client.CostEstimations.Logs(ctx, "nonexisting")
		assert.Nil(t, logs)
		assert.Error(t, err)
	})
}
