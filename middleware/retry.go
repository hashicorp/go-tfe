// Package middleware contains the custom middleware used by the go-tfe SDK.
package middleware

import (
	"io"
	"math"
	nethttp "net/http"
	"strconv"
	"time"

	khttp "github.com/microsoft/kiota-http-go"
)

const (
	retryAttemptHeader = "Retry-Attempt"
	retryAfterHeader   = "Retry-After"

	defaultMaxRetries        = 3
	absoluteMaxRetries       = 10
	defaultDelaySeconds      = 3
	absoluteMaxDelaySeconds  = 180
)

// ShouldRetryFunc determines whether a request should be retried based on the response.
type ShouldRetryFunc func(executionCount int, request *nethttp.Request, response *nethttp.Response) bool

// RetryMiddleware is a custom retry middleware that replaces Kiota's built-in RetryHandler.
// Unlike Kiota's implementation, it does NOT gate retries behind a hardcoded isRetriableErrorCode
// check (which only covers 429, 503, 504). Instead, it delegates the retry decision entirely
// to the provided ShouldRetry callback, allowing retries on any status code (e.g. 425, 500, 502).
type RetryMiddleware struct {
	maxRetries   int
	delaySeconds int
	shouldRetry  ShouldRetryFunc
}

// RetryMiddlewareOptions configures the custom retry middleware.
type RetryMiddlewareOptions struct {
	// MaxRetries is the maximum number of retry attempts. Capped at 10.
	MaxRetries int
	// DelaySeconds is the base delay between retries (used for exponential backoff).
	DelaySeconds int
	// ShouldRetry determines whether a given response warrants a retry.
	// This is called unconditionally for every non-nil response — there is no
	// status code pre-filter.
	ShouldRetry ShouldRetryFunc
}

// NewRetryMiddleware creates a custom retry middleware with the given options.
func NewRetryMiddleware(opts RetryMiddlewareOptions) khttp.Middleware {
	maxRetries := opts.MaxRetries
	if maxRetries < 1 {
		maxRetries = defaultMaxRetries
	} else if maxRetries > absoluteMaxRetries {
		maxRetries = absoluteMaxRetries
	}

	delaySeconds := opts.DelaySeconds
	if delaySeconds < 1 {
		delaySeconds = defaultDelaySeconds
	} else if delaySeconds > absoluteMaxDelaySeconds {
		delaySeconds = absoluteMaxDelaySeconds
	}

	shouldRetry := opts.ShouldRetry
	if shouldRetry == nil {
		shouldRetry = func(_ int, _ *nethttp.Request, _ *nethttp.Response) bool {
			return false
		}
	}

	return &RetryMiddleware{
		maxRetries:   maxRetries,
		delaySeconds: delaySeconds,
		shouldRetry:  shouldRetry,
	}
}

// Intercept implements the khttp.Middleware interface.
func (m *RetryMiddleware) Intercept(pipeline khttp.Pipeline, middlewareIndex int, req *nethttp.Request) (*nethttp.Response, error) {
	response, err := pipeline.Next(req, middlewareIndex)
	if err != nil {
		return response, err
	}
	return m.retryRequest(pipeline, middlewareIndex, req, response, 0, 0)
}

func (m *RetryMiddleware) retryRequest(
	pipeline khttp.Pipeline,
	middlewareIndex int,
	req *nethttp.Request,
	resp *nethttp.Response,
	executionCount int,
	cumulativeDelay time.Duration,
) (*nethttp.Response, error) {
	// Check all retry conditions. Unlike Kiota's RetryHandler, we do NOT pre-filter
	// by status code. The ShouldRetry callback is the sole arbiter of whether to retry.
	if !m.isRetriableRequest(req) {
		return resp, nil
	}
	if executionCount >= m.maxRetries {
		return resp, nil
	}
	if cumulativeDelay >= time.Duration(absoluteMaxDelaySeconds)*time.Second {
		return resp, nil
	}
	if !m.shouldRetry(executionCount, req, resp) {
		return resp, nil
	}

	// Proceed with retry
	executionCount++
	delay := m.getRetryDelay(resp, executionCount)
	cumulativeDelay += delay

	req.Header.Set(retryAttemptHeader, strconv.Itoa(executionCount))

	// Reset body for retry if possible
	if req.Body != nil {
		if seeker, ok := req.Body.(io.Seeker); ok {
			seeker.Seek(0, io.SeekStart)
		}
	}

	// Wait for the delay, respecting context cancellation
	ctx := req.Context()
	t := time.NewTimer(delay)
	select {
	case <-ctx.Done():
		t.Stop()
		return nil, ctx.Err()
	case <-t.C:
	}

	response, err := pipeline.Next(req, middlewareIndex)
	if err != nil {
		return response, err
	}
	return m.retryRequest(pipeline, middlewareIndex, req, response, executionCount, cumulativeDelay)
}

// isRetriableRequest checks whether the request type supports retrying.
// POST/PUT/PATCH with streaming bodies (ContentLength == -1) cannot be retried.
func (m *RetryMiddleware) isRetriableRequest(req *nethttp.Request) bool {
	isBodiedMethod := req.Method == "POST" || req.Method == "PUT" || req.Method == "PATCH"
	if isBodiedMethod && req.Body != nil {
		return req.ContentLength != -1
	}
	return true
}

// getRetryDelay calculates the delay before the next retry attempt.
// It respects the Retry-After header if present, otherwise uses exponential backoff.
func (m *RetryMiddleware) getRetryDelay(resp *nethttp.Response, executionCount int) time.Duration {
	if resp != nil {
		retryAfter := resp.Header.Get(retryAfterHeader)
		if retryAfter != "" {
			// Try parsing as seconds (float)
			if seconds, err := strconv.ParseFloat(retryAfter, 64); err == nil && seconds > 0 {
				return time.Duration(seconds * float64(time.Second))
			}
			// Try parsing as HTTP-date
			if t, err := time.Parse(time.RFC1123, retryAfter); err == nil {
				if d := time.Until(t); d > 0 {
					return d
				}
			}
		}
	}

	// Exponential backoff: delaySeconds^executionCount
	return time.Duration(math.Pow(float64(m.delaySeconds), float64(executionCount))) * time.Second
}
