// Copyright IBM Corp. 2018, 2026

// Package middleware contains the custom middleware used by the go-tfe SDK, as well as options
// for configuring the default middlewares.
package middleware

import (
	"fmt"
	nethttp "net/http"

	khttp "github.com/microsoft/kiota-http-go"
)

func nilErrorFactory(_ *nethttp.Response, _ error) error {
	return nil
}

// GetForKiota uses the provided options to configure the default middlewares used by kiota
// as well as the custom middleware supplied by the SDK.
//
// This function replaces Kiota's built-in RetryHandler with a custom RetryMiddleware.
// Kiota's RetryHandler has a hardcoded isRetriableErrorCode gate that only covers 429, 503, 504
// and short-circuits before calling the ShouldRetry callback. Our custom middleware calls
// ShouldRetry unconditionally, allowing retries on 429, 425, and all 5xx (when RetryServerErrors
// is enabled).
func GetForKiota(tfeSDKVersion string, options ...MiddlewareOption) ([]khttp.Middleware, error) {
	var errFactory APIErrorFactory = nilErrorFactory
	var retryOpts = RetryOptions{
		Enabled:    false,
		Hook:       func(retryCount int, response *nethttp.Response) {},
		MaxRetries: 5,
	}
	var headerOpts nethttp.Header
	for _, option := range options {
		switch option.key {
		case "Headers":
			var ok bool
			headerOpts, ok = option.value.(nethttp.Header)
			if !ok {
				return nil, fmt.Errorf("invalid type for Headers option: expected net/http.Header, got %T", option.value)
			}
		case "RetryOptions":
			opts, ok := option.value.(RetryOptions)
			if !ok {
				continue
			}
			retryOpts.Enabled = opts.Enabled
			retryOpts.RetryServerErrors = opts.RetryServerErrors
			retryOpts.MaxRetries = opts.MaxRetries
			if opts.Hook != nil {
				retryOpts.Hook = opts.Hook
			}
		case "ErrorInterceptor":
			factory, ok := option.value.(APIErrorFactory)
			if !ok {
				continue
			}
			errFactory = factory
		}
	}

	// Build the custom retry middleware that bypasses Kiota's isRetriableErrorCode gate.
	// The ShouldRetry callback is the sole decider of whether to retry — no pre-filtering.
	retryMiddleware := NewRetryMiddleware(RetryMiddlewareOptions{
		MaxRetries:   retryOpts.MaxRetries,
		DelaySeconds: 1,
		ShouldRetry: func(executionCount int, request *nethttp.Request, response *nethttp.Response) bool {
			// Retry on 429 (rate limited) and 425 (too early) when rate limit retries are enabled
			if retryOpts.Enabled && (response.StatusCode == 429 || response.StatusCode == 425) {
				retryOpts.Hook(executionCount, response)
				return true
			}
			// Retry on all 5xx if RetryServerErrors is enabled (independent of rate limit setting)
			if retryOpts.RetryServerErrors && response.StatusCode >= 500 {
				retryOpts.Hook(executionCount, response)
				return true
			}
			return false
		},
	})

	headersMiddleware := NewHeadersMiddleware(headerOpts)

	// Build the middleware pipeline explicitly rather than using
	// khttp.GetDefaultMiddlewaresWithOptions (which always injects Kiota's
	// RetryHandler that we can't fully control).
	return []khttp.Middleware{
		NewErrorMiddleware(errFactory),
		retryMiddleware,
		NewRateLimitMiddleware(),
		khttp.NewRedirectHandlerWithOptions(khttp.RedirectHandlerOptions{
			MaxRedirects: 5,
			ShouldRedirect: func(req *nethttp.Request, res *nethttp.Response) bool {
				return true
			},
		}),
		khttp.NewCompressionHandlerWithOptions(*khttp.NewCompressionOptionsReference(false)),
		headersMiddleware,
		khttp.NewUserAgentHandlerWithOptions(&khttp.UserAgentHandlerOptions{
			Enabled:        headerOpts.Get("User-Agent") == "",
			ProductName:    "go-tfe",
			ProductVersion: tfeSDKVersion,
		}),
		khttp.NewHeadersInspectionHandlerWithOptions(func() khttp.HeadersInspectionOptions {
			opts := *khttp.NewHeadersInspectionOptions()
			opts.InspectRequestHeaders = false
			opts.InspectResponseHeaders = true
			return opts
		}()),
	}, nil
}
