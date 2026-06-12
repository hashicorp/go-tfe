// Copyright IBM Corp. 2018, 2026

package middleware

import (
	"fmt"
	"net/http"
	"strconv"

	khttp "github.com/microsoft/kiota-http-go"
)

const rateLimitResetHeader = "X-RateLimit-Reset"

// RateLimitMiddleware translates custom rate limit headers to Retry-After,
// which is used by the retry middleware for backoff timing.
type RateLimitMiddleware struct{}

// NewRateLimitMiddleware creates a middleware that translates X-RateLimit-Reset
// headers into standard Retry-After headers for use by the retry middleware.
func NewRateLimitMiddleware() khttp.Middleware {
	return &RateLimitMiddleware{}
}

// Intercept implements the khttp.Middleware interface.
func (m *RateLimitMiddleware) Intercept(pipeline khttp.Pipeline, index int, req *http.Request) (*http.Response, error) {
	resp, err := pipeline.Next(req, index)
	if err != nil {
		return resp, err
	}

	// If rate limited, translate the custom header into Retry-After for the retry middleware.
	resetHeader := resp.Header.Get(rateLimitResetHeader)
	if resetHeader != "" {
		val, err := strconv.ParseFloat(resetHeader, 64)
		if err == nil && val > 0 {
			resp.Header.Set("Retry-After", fmt.Sprintf("%.3f", val))
		}
	}

	return resp, nil
}
