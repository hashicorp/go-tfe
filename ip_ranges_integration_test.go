// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package tfe

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIPRangesRead(t *testing.T) {
	skipUnlessAfterDate(t, time.Date(2023, 5, 1, 0, 0, 0, 0, time.Local))
	skipIfEnterprise(t)

	client := testClient(t)
	ctx := context.Background()

	t.Run("without modifiedSince", func(t *testing.T) {
		r, err := client.Meta.IPRanges.Read(ctx, "")
		require.NoError(t, err)
		assert.NotEmpty(t, r.API)
		assert.NotEmpty(t, r.Notifications)
		assert.NotEmpty(t, r.Sentinel)
		assert.NotEmpty(t, r.VCS)
	})

	t.Run("with future modifiedSince", func(t *testing.T) {
		ts := time.Now().Add(48 * time.Hour)
		modifiedSince := ts.Format("Mon, 02 Jan 2006 00:00:00 GMT")
		r, err := client.Meta.IPRanges.Read(ctx, modifiedSince)
		require.NoError(t, err)
		assert.Empty(t, r.API)
		assert.Empty(t, r.Notifications)
		assert.Empty(t, r.Sentinel)
		assert.Empty(t, r.VCS)
	})
}
