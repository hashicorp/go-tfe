package tfe

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPlanExportsCreate(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	rTest, rTestCleanup := createPlannedRun(t, client, nil)
	defer rTestCleanup()

	pTest, err := client.Plans.Read(ctx, rTest.Plan.ID)
	assert.NoError(t, err)

	t.Run("with valid options", func(t *testing.T) {
		options := PlanExportCreateOptions{
			Plan:     pTest,
			DataType: PlanExportType(PlanExportSentinelMockBundleV0),
		}

		pe, err := client.PlanExports.Create(ctx, options)
		assert.NoError(t, err)
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
		assert.EqualError(t, err, "plan is required")
	})

	t.Run("without a data type", func(t *testing.T) {
		options := PlanExportCreateOptions{
			Plan:     pTest,
			DataType: nil,
		}

		pe, err := client.PlanExports.Create(ctx, options)
		assert.Nil(t, pe)
		assert.EqualError(t, err, "data type is required")
	})
}

func TestPlanExportsRead(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	peTest, peTestCleanup := createPlanExport(t, client, nil)
	defer peTestCleanup()

	t.Run("without a valid ID", func(t *testing.T) {
		_, err := client.PlanExports.Read(ctx, badIdentifier)
		assert.EqualError(t, err, "invalid value for plan export ID")
	})

	t.Run("with a valid ID", func(t *testing.T) {
		pe, err := client.PlanExports.Read(ctx, peTest.ID)
		assert.NoError(t, err)
		assert.Equal(t, peTest.ID, pe.ID)
		assert.Equal(t, peTest.DataType, pe.DataType)
	})
}

func TestPlanExportsDownload(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	t.Run("without a valid ID", func(t *testing.T) {
		_, err := client.PlanExports.Download(ctx, badIdentifier)
		assert.EqualError(t, err, "invalid value for plan export ID")
	})
}

func TestPlanExportsDelete(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	peTest, peTestCleanup := createPlanExport(t, client, nil)
	defer peTestCleanup()

	t.Run("when the export does not exist", func(t *testing.T) {
		err := client.Policies.Delete(ctx, "pe-doesntexist")
		assert.Equal(t, err, ErrResourceNotFound)
	})

	t.Run("without a valid ID", func(t *testing.T) {
		err := client.PlanExports.Delete(ctx, badIdentifier)
		assert.EqualError(t, err, "invalid value for plan export ID")
	})

	t.Run("with a valid ID", func(t *testing.T) {
		err := client.PlanExports.Delete(ctx, peTest.ID)
		assert.NoError(t, err)
	})
}
