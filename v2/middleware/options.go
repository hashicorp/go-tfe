// Copyright IBM Corp. 2018, 2026

package middleware

import (
	nethttp "net/http"
)

// MiddlewareOption is a functional option for configuring the middleware pipeline.
type MiddlewareOption struct {
	key   string
	value any
}

// RetryHookCallback is called before each retry attempt with the attempt number and response.
type RetryHookCallback func(retryCount int, response *nethttp.Response)

// RetryOptions configures the retry behavior for the middleware pipeline.
type RetryOptions struct {
	Enabled           bool
	MaxRetries        int
	RetryServerErrors bool
	Hook              RetryHookCallback
}

// WithRetryOptions creates a middleware option that configures retry behavior.
func WithRetryOptions(enabled, enabledForServerErrors bool, maxRetries int, hook RetryHookCallback) MiddlewareOption {
	return MiddlewareOption{key: "RetryOptions", value: RetryOptions{
		Enabled:           enabled,
		RetryServerErrors: enabledForServerErrors,
		Hook:              hook,
		MaxRetries:        maxRetries,
	}}
}

// WithErrorInterceptorOption creates a middleware option that configures error interception.
func WithErrorInterceptorOption(errorFactory APIErrorFactory) MiddlewareOption {
	return MiddlewareOption{key: "ErrorInterceptor", value: errorFactory}
}

// WithHeaders creates a middleware option that adds the provided headers to each request.
func WithHeaders(headers nethttp.Header) MiddlewareOption {
	return MiddlewareOption{key: "Headers", value: headers}
}
