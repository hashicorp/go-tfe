// Copyright IBM Corp. 2018, 2026
// SPDX-License-Identifier: MPL-2.0

package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// terminalPipeline is a minimal khttp.Pipeline that returns the given response
// and lets the test inspect the final request headers.
type terminalPipeline struct {
	capturedReq *http.Request
	resp        *http.Response
}

func (p *terminalPipeline) Next(req *http.Request, _ int) (*http.Response, error) {
	p.capturedReq = req
	return p.resp, nil
}

func TestContentTypeMiddleware_setsMissingHeader(t *testing.T) {
	tests := []struct {
		method      string
		wantHeader  bool
		presetValue string
	}{
		{method: http.MethodPost, wantHeader: true},
		{method: http.MethodPatch, wantHeader: true},
		{method: http.MethodDelete, wantHeader: true},
		{method: http.MethodGet, wantHeader: false},
		{method: http.MethodPut, wantHeader: false},
		// Pre-existing Content-Type must not be overwritten.
		{method: http.MethodPost, wantHeader: true, presetValue: "application/octet-stream"},
	}

	for _, tc := range tests {
		t.Run(tc.method+"_preset="+tc.presetValue, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, "https://example.com/api/v2/test", nil)
			if tc.presetValue != "" {
				req.Header.Set("Content-Type", tc.presetValue)
			}

			resp := &http.Response{StatusCode: http.StatusOK}
			pipeline := &terminalPipeline{resp: resp}

			m := &ContentTypeMiddleware{}
			_, err := m.Intercept(pipeline, 0, req)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			got := pipeline.capturedReq.Header.Get("Content-Type")

			if tc.presetValue != "" {
				// Pre-existing header must be preserved.
				if got != tc.presetValue {
					t.Errorf("Content-Type modified: got %q, want %q", got, tc.presetValue)
				}
				return
			}

			if tc.wantHeader {
				if got != contentTypeJSONAPI {
					t.Errorf("Content-Type = %q, want %q", got, contentTypeJSONAPI)
				}
			} else {
				if got != "" {
					t.Errorf("Content-Type should not be set for %s, got %q", tc.method, got)
				}
			}
		})
	}
}
