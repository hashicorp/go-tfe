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

	ts, client := testServerWithClient(t, map[string]http.HandlerFunc{
		"GET /api/v2/policy-checks/{id}/output": func(w http.ResponseWriter, r *http.Request) {
			// Verify auth middleware applied the Bearer token
			assert.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))

			// Verify User-Agent header contains the default client user agent
			assert.Contains(t, r.Header.Get("User-Agent"), DefaultUserAgent)

			w.Header().Set("Content-Type", "application/octet-stream")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(expectedOutput))
		},
	})

	cases := []struct {
		name string
		uri  string
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
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			resp, err := client.GetStream(context.Background(), tc.uri)
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, http.StatusOK, resp.StatusCode)

			body, err := io.ReadAll(resp.Body)
			require.NoError(t, err)
			assert.Equal(t, expectedOutput, string(body))
		})
	}
}
