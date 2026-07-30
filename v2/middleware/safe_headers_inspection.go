// Copyright IBM Corp. 2018, 2026

package middleware

import (
	nethttp "net/http"

	abs "github.com/microsoft/kiota-abstractions-go"
	khttp "github.com/microsoft/kiota-http-go"
)

// headersInspectionReqKey is the context key used by kiota-http-go to store
// per-request HeadersInspectionOptions.
var headersInspectionReqKey = abs.RequestOptionKey{Key: "nethttplibrary.HeadersInspectionOptions"}

type headersInspectionOptions interface {
	abs.RequestOption
	GetInspectRequestHeaders() bool
	GetInspectResponseHeaders() bool
	GetRequestHeaders() *abs.RequestHeaders
	GetResponseHeaders() *abs.ResponseHeaders
}

// headersInspectionHandler implements khttp.Middleware. It retrieves
// HeadersInspectionOptions from the request context when present, or creates
// fresh options with the configured defaults. This ensures each concurrent
// request writes into its own headers map.
type headersInspectionHandler struct {
	inspectRequestHeaders  bool
	inspectResponseHeaders bool
}

// NewHeadersInspectionHandler returns a headersInspectionHandler configured
// with the given defaults.
func NewHeadersInspectionHandler(inspectRequest, inspectResponse bool) *headersInspectionHandler {
	return &headersInspectionHandler{
		inspectRequestHeaders:  inspectRequest,
		inspectResponseHeaders: inspectResponse,
	}
}

// Intercept implements khttp.Middleware.
func (h *headersInspectionHandler) Intercept(pipeline khttp.Pipeline, middlewareIndex int, req *nethttp.Request) (*nethttp.Response, error) {
	reqOption, ok := req.Context().Value(headersInspectionReqKey).(headersInspectionOptions)
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
