// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package tfe

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIPRangesRead(t *testing.T) {
	server, client := testServerWithClient(t, map[string]http.HandlerFunc{
		"/api/meta/ip-ranges": func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Last-Modified", time.Now().Format(http.TimeFormat))
			w.WriteHeader(200)
			w.Write([]byte(`{
				"api": [
					"192.168.1.10"
				],
				"notifications": [
					"192.168.1.11"
				],
				"sentinel": [
					"192.168.1.12"
				],
				"vcs": [
					"192.168.1.13"
				]
			}`))
		},
	})
	defer server.Close()
	ctx := context.Background()

	t.Run("without modifiedSince", func(t *testing.T) {
		r, err := client.Meta.IPRanges.Read(ctx, "")
		require.NoError(t, err)
		assert.NotEmpty(t, r.API)
		assert.NotEmpty(t, r.Notifications)
		assert.NotEmpty(t, r.Sentinel)
		assert.NotEmpty(t, r.VCS)
	})
}
