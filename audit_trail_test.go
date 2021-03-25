package tfe

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuditTrailsList(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	t.Run("with no time limit", func(t *testing.T) {
		a, err := client.AuditTrails.List(ctx, AuditTrailListOptions{})
		require.NoError(t, err)
		assert.NotEmpty(t, a.Data)
		assert.NotEmpty(t, a.Pagination)
	})

	t.Run("with time limit", func(t *testing.T) {
		sinceTime := time.Now()
		a, err := client.AuditTrails.List(ctx, AuditTrailListOptions{Since: &sinceTime})
		require.NoError(t, err)
		assert.NotEmpty(t, a.Pagination)
	})
}
