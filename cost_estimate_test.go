package tfe

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCostEstimatesRead(t *testing.T) {
	skipIfEnterprise(t)

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
	rTest, _ := createCostEstimatedRun(t, client, wTest)

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
