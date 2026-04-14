package middleware

import (
	nethttp "net/http"
)

type MiddlewareOption struct {
	key   string
	value any
}

type RetryHookCallback func(retryCount int, response *nethttp.Response)

type RetryOptions struct {
	Enabled           bool
	MaxRetries        int
	RetryServerErrors bool
	Hook              RetryHookCallback
}

func WithRetryOptions(enabled, enabledForServerErrors bool, maxRetries int, hook RetryHookCallback) MiddlewareOption {
	return MiddlewareOption{key: "RetryOptions", value: RetryOptions{
		Enabled:           enabled,
		RetryServerErrors: enabledForServerErrors,
		Hook:              hook,
		MaxRetries:        maxRetries,
	}}
}

func WithErrorInterceptorOption(errorFactory APIErrorFactory) MiddlewareOption {
	return MiddlewareOption{key: "ErrorInterceptor", value: errorFactory}
}
