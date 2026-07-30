// Copyright IBM Corp. 2018, 2026

package middleware

import (
	nethttp "net/http"

	abs "github.com/microsoft/kiota-abstractions-go"
	khttp "github.com/microsoft/kiota-http-go"
)

// headersInspectionReqKey mirrors the unexported headersInspectionKeyValue in
// kiota-http-go (headers_inspection_handler.go) so that context lookups are
// wire-compatible with any per-request HeadersInspectionOptions that a caller
// may have stored via khttp.NewHeadersInspectionOptions().GetKey().
var headersInspectionReqKey = abs.RequestOptionKey{Key: "nethttplibrary.HeadersInspectionOptions"}

// headersInspectionOptsIface mirrors the unexported headersInspectionOptionsInt
// interface in kiota-http-go so that we can type-assert per-request options
// stored in the context without importing an internal type.
type headersInspectionOptsIface interface {
	abs.RequestOption
	GetInspectRequestHeaders() bool
	GetInspectResponseHeaders() bool
	GetRequestHeaders() *abs.RequestHeaders
	GetResponseHeaders() *abs.ResponseHeaders
}

// safeHeadersInspectionHandler is a concurrent-safe replacement for
// khttp.HeadersInspectionHandler.
//
// The upstream handler (kiota-http-go v1.5.6) contains a data race: when a
// request does not carry a per-request HeadersInspectionOptions in its
// context, Intercept falls back to &middleware.options — a pointer into the
// handler struct itself. All concurrent goroutines then share that single
// HeadersInspectionOptions value and race writing response headers into the
// embedded ResponseHeaders map, producing:
//
//	fatal error: concurrent map read and map write
//	github.com/microsoft/kiota-abstractions-go.(*header).Add
//	github.com/microsoft/kiota-http-go.HeadersInspectionHandler.Intercept
//
// This implementation allocates a fresh HeadersInspectionOptions for every
// request that does not already carry one, so each goroutine writes into its
// own private map. Per-request options provided by callers (the intended
// consumption pattern) are honoured unchanged.
type safeHeadersInspectionHandler struct {
	inspectRequestHeaders  bool
	inspectResponseHeaders bool
}

func newSafeHeadersInspectionHandler(inspectRequest, inspectResponse bool) *safeHeadersInspectionHandler {
	return &safeHeadersInspectionHandler{
		inspectRequestHeaders:  inspectRequest,
		inspectResponseHeaders: inspectResponse,
	}
}

// Intercept implements khttp.Middleware.
func (h *safeHeadersInspectionHandler) Intercept(pipeline khttp.Pipeline, middlewareIndex int, req *nethttp.Request) (*nethttp.Response, error) {
	// Prefer a per-request option already stored in the context (set by the
	// caller to observe specific request/response headers). If none is present,
	// allocate a fresh one for this request only — never touch shared state.
	reqOption, ok := req.Context().Value(headersInspectionReqKey).(headersInspectionOptsIface)
	if !ok {
		freshOpts := khttp.NewHeadersInspectionOptions()
		freshOpts.InspectRequestHeaders = h.inspectRequestHeaders
		freshOpts.InspectResponseHeaders = h.inspectResponseHeaders
		reqOption = freshOpts
	}

	if reqOption.GetInspectRequestHeaders() {
		for k, v := range req.Header {
			if len(v) == 1 {
				reqOption.GetRequestHeaders().Add(k, v[0])
			} else {
				reqOption.GetRequestHeaders().Add(k, v[0], v[1:]...)
			}
		}
	}

	response, err := pipeline.Next(req, middlewareIndex)
	if err != nil {
		return response, err
	}

	if reqOption.GetInspectResponseHeaders() {
		for k, v := range response.Header {
			if len(v) == 1 {
				reqOption.GetResponseHeaders().Add(k, v[0])
			} else {
				reqOption.GetResponseHeaders().Add(k, v[0], v[1:]...)
			}
		}
	}

	return response, err
}
