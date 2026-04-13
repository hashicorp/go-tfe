// Package middleware contains the custom middleware used by the go-tfe SDK, as well as options
// for configuring the default middlewares.
package middleware

import (
	"time"

	nethttp "net/http"

	khttp "github.com/microsoft/kiota-http-go"
)

func nilErrorFactory(_ *nethttp.Response, _ error) error {
	return nil
}

// GetForKiota uses the provided options to configure the default middlewares used by kiota
// as well as the custom middleware supplied by the SDK.
func GetForKiota(tfeSDKVersion string, options ...MiddlewareOption) ([]khttp.Middleware, error) {
	retryServerErrors := false
	var retryHook RetryHookCallback = func(int, *nethttp.Response) {}
	var errFactory APIErrorFactory = nilErrorFactory

	for _, option := range options {
		switch option.key {
		case "RetryServerErrors":
			retryServerErrors = option.value.(bool)
		case "RetryHook":
			retryHook = option.value.(RetryHookCallback)
		case "ErrorInterceptor":
			errFactory = option.value.(APIErrorFactory)
		}
	}

	retryOptions := khttp.RetryHandlerOptions{
		MaxRetries:   5,
		DelaySeconds: 1,
		ShouldRetry: func(delay time.Duration, executionCount int, request *nethttp.Request, response *nethttp.Response) bool {
			// Retry on 425, 429, and 5XX if the option is enabled
			if (response.StatusCode == 429 || response.StatusCode == 425) || (retryServerErrors && response.StatusCode >= 500) {
				retryHook(executionCount, response)
				return true
			}
			return false
		},
	}
	redirectHandlerOptions := khttp.RedirectHandlerOptions{
		MaxRedirects: 5,
		ShouldRedirect: func(req *nethttp.Request, res *nethttp.Response) bool {
			return true
		},
	}
	compressionOptions := khttp.NewCompressionOptionsReference(false)
	userAgentHandlerOptions := khttp.UserAgentHandlerOptions{
		Enabled:        true,
		ProductName:    "go-tfe",
		ProductVersion: tfeSDKVersion,
	}

	headersOptions := khttp.NewHeadersInspectionOptions()
	headersOptions.InspectRequestHeaders = false
	headersOptions.InspectResponseHeaders = true

	defaultMiddleware, err := khttp.GetDefaultMiddlewaresWithOptions(
		&retryOptions,
		&redirectHandlerOptions,
		compressionOptions,
		&userAgentHandlerOptions,
		headersOptions,
	)
	if err != nil {
		return nil, err
	}

	return append(defaultMiddleware, NewErrorMiddleware(errFactory)), nil
}
