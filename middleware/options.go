package middleware

import nethttp "net/http"

type MiddlewareOption struct {
	key   string
	value any
}

type RetryHookCallback func(retryCount int, response *nethttp.Response)

func WithRetryServerErrorsOption(option bool) MiddlewareOption {
	return MiddlewareOption{key: "RetryServerErrors", value: option}
}

func WithRetryHookOption(hook RetryHookCallback) MiddlewareOption {
	return MiddlewareOption{key: "RetryHook", value: hook}
}
