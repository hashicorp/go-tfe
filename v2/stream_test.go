// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package tfe

import (
	"context"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetStream(t *testing.T) {
	expectedOutput := "Sentinel policy check output: all policies passed."

	xCustomHeaderCount := 0

	ts, client := testServerWithClient(t, "/api/v2", map[string]http.HandlerFunc{
		"GET /api/v2/policy-checks/{id}/output": func(w http.ResponseWriter, r *http.Request) {
			// Verify auth middleware applied the Bearer token
			assert.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))

			// Verify User-Agent header contains the default client user agent
			assert.Contains(t, r.Header.Get("User-Agent"), DefaultUserAgent)

			if r.Header.Get("X-Custom-Header") != "" {
				xCustomHeaderCount++
				assert.Equal(t, "CustomValue", r.Header.Get("X-Custom-Header"))
			}

			w.Header().Set("Content-Type", "application/octet-stream")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(expectedOutput))
		},
	})

	cases := []struct {
		name    string
		uri     string
		headers http.Header
	}{
		{
			name: "With base URL",
			uri:  "/api/v2/policy-checks/polchk-123/output",
		},
		{
			name: "Without base URL",
			uri:  "/policy-checks/polchk-123/output",
		},
		{
			name: "With query parameters",
			uri:  "/api/v2/policy-checks/polchk-123/output?version=1",
		},
		{
			name: "With host and scheme",
			uri:  ts.URL + "/api/v2/policy-checks/polchk-123/output?version=1",
		},
		{
			name: "No leading slash, no base path",
			uri:  "policy-checks/polchk-123/output",
		},
		{
			name: "With headers",
			uri:  "/api/v2/policy-checks/polchk-123/output",
			headers: http.Header{
				"X-Custom-Header": []string{"CustomValue"},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			resp, err := client.GetStream(context.Background(), tc.uri, tc.headers)
			require.NoError(t, err)
			defer resp.Body.Close() //nolint:errcheck

			assert.Equal(t, http.StatusOK, resp.StatusCode)

			body, err := io.ReadAll(resp.Body)
			require.NoError(t, err)
			assert.Equal(t, expectedOutput, string(body))
		})
	}

	assert.Equal(t, 1, xCustomHeaderCount)
}
