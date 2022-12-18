package tfe

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAdminSettings_CostEstimation_Read(t *testing.T) {
	skipUnlessEnterprise(t)

	client := testClient(t)
	ctx := context.Background()

	costEstimationSettings, err := client.Admin.Settings.CostEstimation.Read(ctx)
	require.NoError(t, err)
	assert.Equal(t, "cost-estimation", costEstimationSettings.ID)
	assert.NotNil(t, costEstimationSettings.Enabled)
}

func TestAdminSettings_CostEstimation_Update(t *testing.T) {
	skipUnlessEnterprise(t)

	client := testClient(t)
	ctx := context.Background()

	_, err := client.Admin.Settings.CostEstimation.Read(ctx)
	require.NoError(t, err)

	costEnabled := false
	costEstimationSettings, err := client.Admin.Settings.CostEstimation.Update(ctx, AdminCostEstimationSettingOptions{
		Enabled: Bool(costEnabled),
	})
	require.NoError(t, err)
	assert.Equal(t, costEnabled, costEstimationSettings.Enabled)
}
