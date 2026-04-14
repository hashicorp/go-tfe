package middleware

import (
	"fmt"
	"net/http"
	"strconv"

	khttp "github.com/microsoft/kiota-http-go"
)

const rateLimitResetHeader = "X-RateLimit-Reset"

// RateLimitMiddleware translates custom rate limit headers to Retry-After,
// which is handled by Kiota's built-in retry middleware.
type RateLimitMiddleware struct{}

func NewRateLimitMiddleware() khttp.Middleware {
	return &RateLimitMiddleware{}
}

func (m *RateLimitMiddleware) Intercept(pipeline khttp.Pipeline, index int, req *http.Request) (*http.Response, error) {
	resp, err := pipeline.Next(req, index)
	if err != nil {
		return resp, err
	}

	// if rate limited, translate the custom header into Retry-After, which Kiota will use internally
	resetHeader := resp.Header.Get(rateLimitResetHeader)
	if resetHeader != "" {
		val, _ := strconv.ParseFloat(resetHeader, 64)

		if val > 0 {
			// Inject the standard header that Kiota's RetryHandler understands
			resp.Header.Set("Retry-After", fmt.Sprintf("%.3f", val))
		}
	}

	return resp, nil
}
