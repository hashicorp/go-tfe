// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package tfe

import (
	"context"
	"io"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOpenAPIRead(t *testing.T) {
	server, client := testServerWithClient(t, "/", map[string]http.HandlerFunc{
		"/openapi/stable.json": func(w http.ResponseWriter, r *http.Request) {
			if r.Header.Get("If-Modified-Since") != "" {
				w.WriteHeader(http.StatusNotModified)
				return
			}

			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Last-Modified", time.Now().Format(http.TimeFormat))
			w.WriteHeader(200)

			file, err := os.OpenFile("openapi/spec.json", os.O_RDONLY, 0o644)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			defer file.Close()

			_, err = io.Copy(w, file)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
		},
	})
	defer server.Close()
	ctx := context.Background()

	t.Run("without modifiedSince", func(t *testing.T) {
		r, err := client.Meta.OpenAPI.Read(ctx, false, nil)
		require.NoError(t, err)
		assert.NotEmpty(t, r)
	})

	t.Run("with modifiedSince", func(t *testing.T) {
		modifiedSince := time.Now().Add(-1 * time.Hour)
		r, err := client.Meta.OpenAPI.Read(ctx, false, &modifiedSince)
		require.NoError(t, err)
		assert.Nil(t, r)
	})
}
