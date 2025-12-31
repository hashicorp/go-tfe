package middleware

import (
	"time"

	nethttp "net/http"

	khttp "github.com/microsoft/kiota-http-go"
)

func GetForKiota(tfeSDKVersion string, options ...MiddlewareOption) ([]khttp.Middleware, error) {
	retryServerErrors := false
	for _, option := range options {
		switch option.key {
		case "RetryServerErrors":
			retryServerErrors = option.value.(bool)
		}
	}

	retryOptions := khttp.RetryHandlerOptions{
		MaxRetries:   5,
		DelaySeconds: 1,
		ShouldRetry: func(delay time.Duration, executionCount int, request *nethttp.Request, response *nethttp.Response) bool {
			// Retry on 425, 429, and 5XX if the option is enabled
			return (response.StatusCode == 429 || response.StatusCode == 425) || (retryServerErrors && response.StatusCode >= 500)
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

	return khttp.GetDefaultMiddlewaresWithOptions(
		&retryOptions,
		&redirectHandlerOptions,
		compressionOptions,
		&userAgentHandlerOptions,
		headersOptions,
	)
}
