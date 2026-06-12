// Copyright IBM Corp. 2018, 2026

package middleware

import (
	"net/http"

	khttp "github.com/microsoft/kiota-http-go"
)

// NewHeadersMiddleware creates a new instance of HeadersMiddleware with the provided headers.
func NewHeadersMiddleware(headers http.Header) khttp.Middleware {
	return &HeadersMiddleware{
		headers: headers,
	}
}

// HeadersMiddleware is a custom middleware that adds the configured headers to each request.
type HeadersMiddleware struct {
	headers http.Header
}

// Intercept implements the khttp.Middleware interface.
func (m *HeadersMiddleware) Intercept(pipeline khttp.Pipeline, middlewareindex int, req *http.Request) (*http.Response, error) {
	for key, values := range m.headers {
		for _, value := range values {
			req.Header.Add(key, value)
		}
	}
	return pipeline.Next(req, middlewareindex)
}
