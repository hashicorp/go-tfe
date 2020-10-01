package tfe

import (
	"context"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestIPRangesRead(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	t.Run("without modifiedSince", func(t *testing.T) {
		r, err := client.IPRanges.Read(ctx, "")
		require.NoError(t, err)
		assert.NotEmpty(t, r.API)
		assert.NotEmpty(t, r.Notifications)
		assert.NotEmpty(t, r.Sentinel)
		assert.NotEmpty(t, r.VCS)
	})

	t.Run("with future modifiedSince", func(t *testing.T) {
		ts := time.Now().Add(48 * time.Hour)
		modifiedSince := ts.Format("Mon, 02 Jan 2006 00:00:00 GMT")
		r, err := client.IPRanges.Read(ctx, modifiedSince)
		require.NoError(t, err)
		assert.Empty(t, r.API)
		assert.Empty(t, r.Notifications)
		assert.Empty(t, r.Sentinel)
		assert.Empty(t, r.VCS)
	})
}
