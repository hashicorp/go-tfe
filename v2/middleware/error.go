package middleware

import (
	"net/http"

	khttp "github.com/microsoft/kiota-http-go"
)

// APIErrorFactory is a function type that takes an HTTP response and a middleware error,
// and returns a new error. It is used to return custom error types based on the response
// from the API.
type APIErrorFactory func(resp *http.Response, pipelineErr error) error

// ErrorMiddleware is a custom middleware that uses an APIErrorFactory to convert HTTP responses
// into custom error types when the response status code indicates an error (400 or above).
type ErrorMiddleware struct {
	errFactory APIErrorFactory
}

// NewErrorMiddleware creates a new kiota middleware that uses the provided factory to convert
// HTTP responses into custom error types when the response status code indicates an error (400 or above).
func NewErrorMiddleware(errFactory APIErrorFactory) khttp.Middleware {
	return &ErrorMiddleware{
		errFactory: errFactory,
	}
}

// Intercept implements the khttp.Middleware interface.
func (m *ErrorMiddleware) Intercept(pipeline khttp.Pipeline, middlewareindex int, req *http.Request) (*http.Response, error) {
	response, err := pipeline.Next(req, middlewareindex)
	if err != nil {
		if response != nil && response.StatusCode >= 400 {
			if apiErr := m.errFactory(response, err); apiErr != nil {
				return nil, apiErr
			}
		}
		return response, err
	}

	if response.StatusCode >= 400 {
		if apiErr := m.errFactory(response, err); apiErr != nil {
			return nil, apiErr
		}
	}

	return response, nil
}
