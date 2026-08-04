// Copyright IBM Corp. 2018, 2026
// SPDX-License-Identifier: MPL-2.0

package middleware

import (
	"net/http"

	khttp "github.com/microsoft/kiota-http-go"
)

const contentTypeJSONAPI = "application/vnd.api+json"

// ContentTypeMiddleware ensures that POST, PATCH, and DELETE requests always
// carry a Content-Type: application/vnd.api+json header, even when the Kiota
// request builder did not set one because the request has no body.
//
// The Atlas API requires this header on all mutating requests regardless of
// whether a body is present; omitting it results in a 415 Unsupported Media
// Type response.  The go-tfe v1 client set this header unconditionally for
// POST, PATCH, and DELETE; this middleware restores that behavior for the v2
// Kiota-based client.
type ContentTypeMiddleware struct{}

// Intercept implements the khttp.Middleware interface.
func (m *ContentTypeMiddleware) Intercept(pipeline khttp.Pipeline, middlewareIndex int, req *http.Request) (*http.Response, error) {
	switch req.Method {
	case http.MethodPost, http.MethodPatch, http.MethodDelete:
		if req.Header.Get("Content-Type") == "" {
			req.Header.Set("Content-Type", contentTypeJSONAPI)
		}
	}
	return pipeline.Next(req, middlewareIndex)
}
