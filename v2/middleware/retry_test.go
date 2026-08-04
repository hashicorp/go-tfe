// Copyright IBM Corp. 2018, 2026

package middleware

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	khttp "github.com/microsoft/kiota-http-go"
)

// testPipeline implements khttp.Pipeline for testing. It routes requests to the
// underlying test server transport.
type testPipeline struct {
	transport http.RoundTripper
}

func (p *testPipeline) Next(req *http.Request, _ int) (*http.Response, error) {
	return p.transport.RoundTrip(req)
}

func newTestPipeline(handler http.Handler) (*testPipeline, *httptest.Server) {
	server := httptest.NewServer(handler)
	return &testPipeline{transport: http.DefaultTransport}, server
}

func TestRetryMiddleware_RetriesOn500(t *testing.T) {
	var attempts int32

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count := atomic.AddInt32(&attempts, 1)
		if count <= 2 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	})

	pipeline, server := newTestPipeline(handler)
	defer server.Close()

	middleware := NewRetryMiddleware(RetryMiddlewareOptions{
		MaxRetries:   5,
		DelaySeconds: 1,
		ShouldRetry: func(_ int, _ *http.Request, resp *http.Response, _ error) bool {
			return resp.StatusCode >= 500
		},
	})

	req, _ := http.NewRequest("GET", server.URL+"/test", nil)
	resp, err := middleware.Intercept(pipeline, 0, req)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	if atomic.LoadInt32(&attempts) != 3 {
		t.Fatalf("expected 3 attempts, got %d", atomic.LoadInt32(&attempts))
	}
}

func TestRetryMiddleware_RetriesOn502(t *testing.T) {
	var attempts int32

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count := atomic.AddInt32(&attempts, 1)
		if count == 1 {
			w.WriteHeader(http.StatusBadGateway)
			return
		}
		w.WriteHeader(http.StatusOK)
	})

	pipeline, server := newTestPipeline(handler)
	defer server.Close()

	middleware := NewRetryMiddleware(RetryMiddlewareOptions{
		MaxRetries:   3,
		DelaySeconds: 1,
		ShouldRetry: func(_ int, _ *http.Request, resp *http.Response, _ error) bool {
			return resp.StatusCode >= 500
		},
	})

	req, _ := http.NewRequest("GET", server.URL+"/test", nil)
	resp, err := middleware.Intercept(pipeline, 0, req)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	if atomic.LoadInt32(&attempts) != 2 {
		t.Fatalf("expected 2 attempts, got %d", atomic.LoadInt32(&attempts))
	}
}

func TestRetryMiddleware_RetriesOn429(t *testing.T) {
	var attempts int32

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count := atomic.AddInt32(&attempts, 1)
		if count == 1 {
			w.Header().Set("Retry-After", "0.01")
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		w.WriteHeader(http.StatusOK)
	})

	pipeline, server := newTestPipeline(handler)
	defer server.Close()

	middleware := NewRetryMiddleware(RetryMiddlewareOptions{
		MaxRetries:   3,
		DelaySeconds: 1,
		ShouldRetry: func(_ int, _ *http.Request, resp *http.Response, _ error) bool {
			return resp.StatusCode == 429 || resp.StatusCode == 425
		},
	})

	req, _ := http.NewRequest("GET", server.URL+"/test", nil)
	resp, err := middleware.Intercept(pipeline, 0, req)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	if atomic.LoadInt32(&attempts) != 2 {
		t.Fatalf("expected 2 attempts, got %d", atomic.LoadInt32(&attempts))
	}
}

func TestRetryMiddleware_RetriesOn425(t *testing.T) {
	var attempts int32

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count := atomic.AddInt32(&attempts, 1)
		if count == 1 {
			w.WriteHeader(425) // Too Early
			return
		}
		w.WriteHeader(http.StatusOK)
	})

	pipeline, server := newTestPipeline(handler)
	defer server.Close()

	middleware := NewRetryMiddleware(RetryMiddlewareOptions{
		MaxRetries:   3,
		DelaySeconds: 1,
		ShouldRetry: func(_ int, _ *http.Request, resp *http.Response, _ error) bool {
			return resp.StatusCode == 429 || resp.StatusCode == 425
		},
	})

	req, _ := http.NewRequest("GET", server.URL+"/test", nil)
	resp, err := middleware.Intercept(pipeline, 0, req)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	if atomic.LoadInt32(&attempts) != 2 {
		t.Fatalf("expected 2 attempts, got %d", atomic.LoadInt32(&attempts))
	}
}

func TestRetryMiddleware_DoesNotRetryWhenShouldRetryReturnsFalse(t *testing.T) {
	var attempts int32

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&attempts, 1)
		w.WriteHeader(http.StatusBadRequest) // 400 - not retriable
	})

	pipeline, server := newTestPipeline(handler)
	defer server.Close()

	middleware := NewRetryMiddleware(RetryMiddlewareOptions{
		MaxRetries:   3,
		DelaySeconds: 1,
		ShouldRetry: func(_ int, _ *http.Request, resp *http.Response, _ error) bool {
			// Only retry on 429, 425, 5xx
			return resp.StatusCode == 429 || resp.StatusCode == 425 || resp.StatusCode >= 500
		},
	})

	req, _ := http.NewRequest("GET", server.URL+"/test", nil)
	resp, err := middleware.Intercept(pipeline, 0, req)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
	if atomic.LoadInt32(&attempts) != 1 {
		t.Fatalf("expected 1 attempt (no retry), got %d", atomic.LoadInt32(&attempts))
	}
}

func TestRetryMiddleware_RespectsMaxRetries(t *testing.T) {
	var attempts int32

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&attempts, 1)
		w.WriteHeader(http.StatusInternalServerError)
	})

	pipeline, server := newTestPipeline(handler)
	defer server.Close()

	middleware := NewRetryMiddleware(RetryMiddlewareOptions{
		MaxRetries:   2,
		DelaySeconds: 1,
		ShouldRetry: func(_ int, _ *http.Request, resp *http.Response, _ error) bool {
			return resp.StatusCode >= 500
		},
	})

	req, _ := http.NewRequest("GET", server.URL+"/test", nil)
	resp, err := middleware.Intercept(pipeline, 0, req)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", resp.StatusCode)
	}
	// 1 initial + 2 retries = 3 total
	if atomic.LoadInt32(&attempts) != 3 {
		t.Fatalf("expected 3 total attempts (1 + 2 retries), got %d", atomic.LoadInt32(&attempts))
	}
}

func TestRetryMiddleware_DoesNotRetryStreamingBody(t *testing.T) {
	var attempts int32

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&attempts, 1)
		w.WriteHeader(http.StatusInternalServerError)
	})

	pipeline, server := newTestPipeline(handler)
	defer server.Close()

	middleware := NewRetryMiddleware(RetryMiddlewareOptions{
		MaxRetries:   3,
		DelaySeconds: 1,
		ShouldRetry: func(_ int, _ *http.Request, resp *http.Response, _ error) bool {
			return resp.StatusCode >= 500
		},
	})

	// Create a POST with ContentLength=-1 (streaming body, not seekable)
	req, _ := http.NewRequest("POST", server.URL+"/test", bytes.NewReader([]byte("body")))
	req.ContentLength = -1 // Simulate streaming body
	resp, err := middleware.Intercept(pipeline, 0, req)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", resp.StatusCode)
	}
	// Should NOT retry because ContentLength is -1 (streaming)
	if atomic.LoadInt32(&attempts) != 1 {
		t.Fatalf("expected 1 attempt (no retry for streaming body), got %d", atomic.LoadInt32(&attempts))
	}
}

func TestRetryMiddleware_RetriesPostWithKnownContentLength(t *testing.T) {
	var attempts int32

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count := atomic.AddInt32(&attempts, 1)
		if count == 1 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	})

	pipeline, server := newTestPipeline(handler)
	defer server.Close()

	middleware := NewRetryMiddleware(RetryMiddlewareOptions{
		MaxRetries:   3,
		DelaySeconds: 1,
		ShouldRetry: func(_ int, _ *http.Request, resp *http.Response, _ error) bool {
			return resp.StatusCode >= 500
		},
	})

	body := []byte("request body")
	req, _ := http.NewRequest("POST", server.URL+"/test", bytes.NewReader(body))
	req.ContentLength = int64(len(body))
	req.GetBody = func() (io.ReadCloser, error) {
		return io.NopCloser(bytes.NewReader(body)), nil
	}
	resp, err := middleware.Intercept(pipeline, 0, req)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	if atomic.LoadInt32(&attempts) != 2 {
		t.Fatalf("expected 2 attempts, got %d", atomic.LoadInt32(&attempts))
	}
}

func TestRetryMiddleware_HookCalledOnRetry(t *testing.T) {
	var attempts int32
	var hookCalls int32

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count := atomic.AddInt32(&attempts, 1)
		if count <= 2 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	})

	pipeline, server := newTestPipeline(handler)
	defer server.Close()

	middleware := NewRetryMiddleware(RetryMiddlewareOptions{
		MaxRetries:   5,
		DelaySeconds: 1,
		ShouldRetry: func(executionCount int, _ *http.Request, resp *http.Response, _ error) bool {
			if resp.StatusCode >= 500 {
				atomic.AddInt32(&hookCalls, 1)
				return true
			}
			return false
		},
	})

	req, _ := http.NewRequest("GET", server.URL+"/test", nil)
	resp, err := middleware.Intercept(pipeline, 0, req)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	if atomic.LoadInt32(&hookCalls) != 2 {
		t.Fatalf("expected hook called 2 times, got %d", atomic.LoadInt32(&hookCalls))
	}
}

func TestRetryMiddleware_SetsRetryAttemptHeader(t *testing.T) {
	var lastRetryAttempt string

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		lastRetryAttempt = r.Header.Get("Retry-Attempt")
		if lastRetryAttempt == "" {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	})

	pipeline, server := newTestPipeline(handler)
	defer server.Close()

	middleware := NewRetryMiddleware(RetryMiddlewareOptions{
		MaxRetries:   3,
		DelaySeconds: 1,
		ShouldRetry: func(_ int, _ *http.Request, resp *http.Response, _ error) bool {
			return resp.StatusCode >= 500
		},
	})

	req, _ := http.NewRequest("GET", server.URL+"/test", nil)
	resp, err := middleware.Intercept(pipeline, 0, req)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	if lastRetryAttempt != "1" {
		t.Fatalf("expected Retry-Attempt header '1', got '%s'", lastRetryAttempt)
	}
}

func TestGetForKiota_NoKiotaRetryHandler(t *testing.T) {
	middlewares, err := GetForKiota("1.0.0",
		WithRetryOptions(true, true, 3, nil),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for _, mw := range middlewares {
		if _, isKiotaRetry := mw.(*khttp.RetryHandler); isKiotaRetry {
			t.Fatal("expected Kiota's RetryHandler to be removed from the pipeline")
		}
	}

	// Verify our custom RetryMiddleware is present
	found := false
	for _, mw := range middlewares {
		if _, ok := mw.(*RetryMiddleware); ok {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected custom RetryMiddleware to be in the pipeline")
	}
}

func TestGetForKiota_RetryDisabled(t *testing.T) {
	var attempts int32

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&attempts, 1)
		w.WriteHeader(http.StatusInternalServerError)
	})

	pipeline, server := newTestPipeline(handler)
	defer server.Close()

	middlewares, err := GetForKiota("1.0.0",
		WithRetryOptions(false, false, 3, nil),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Find our retry middleware
	var retryMW khttp.Middleware
	for _, mw := range middlewares {
		if _, ok := mw.(*RetryMiddleware); ok {
			retryMW = mw
			break
		}
	}
	if retryMW == nil {
		t.Fatal("RetryMiddleware not found in pipeline")
	}

	req, _ := http.NewRequest("GET", server.URL+"/test", nil)
	resp, err := retryMW.Intercept(pipeline, 0, req)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", resp.StatusCode)
	}
	// Should not retry when disabled
	if atomic.LoadInt32(&attempts) != 1 {
		t.Fatalf("expected 1 attempt (retry disabled), got %d", atomic.LoadInt32(&attempts))
	}
}

// flakyPipeline implements khttp.Pipeline for testing transport errors. It returns
// a transport error for the first `failures` attempts, then routes to the transport.
type flakyPipeline struct {
	transport http.RoundTripper
	failures  int32
	attempts  int32
	err       error
}

func (p *flakyPipeline) Next(req *http.Request, _ int) (*http.Response, error) {
	n := atomic.AddInt32(&p.attempts, 1)
	if n <= p.failures {
		return nil, p.err
	}
	return p.transport.RoundTrip(req)
}

func TestRetryMiddleware_RetriesTransportErrors(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	server := httptest.NewServer(handler)
	defer server.Close()

	pipeline := &flakyPipeline{
		transport: http.DefaultTransport,
		failures:  2,
		err:       errors.New("connection refused"),
	}

	middleware := NewRetryMiddleware(RetryMiddlewareOptions{
		MaxRetries:   5,
		DelaySeconds: 1,
		ShouldRetry: func(_ int, _ *http.Request, resp *http.Response, err error) bool {
			return err != nil || resp == nil
		},
	})

	req, _ := http.NewRequest("GET", server.URL+"/test", nil)
	resp, err := middleware.Intercept(pipeline, 0, req)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	// 2 failed attempts + 1 success
	if atomic.LoadInt32(&pipeline.attempts) != 3 {
		t.Fatalf("expected 3 attempts, got %d", atomic.LoadInt32(&pipeline.attempts))
	}
}

func TestRetryMiddleware_ReturnsErrorWhenTransportRetriesExhausted(t *testing.T) {
	transportErr := errors.New("connection reset by peer")
	pipeline := &flakyPipeline{
		transport: http.DefaultTransport,
		failures:  100, // always fail
		err:       transportErr,
	}

	middleware := NewRetryMiddleware(RetryMiddlewareOptions{
		MaxRetries:   2,
		DelaySeconds: 1,
		ShouldRetry: func(_ int, _ *http.Request, resp *http.Response, err error) bool {
			return err != nil
		},
	})

	req, _ := http.NewRequest("GET", "http://example.invalid/test", nil)
	resp, err := middleware.Intercept(pipeline, 0, req)
	if resp != nil {
		defer resp.Body.Close()
		t.Fatalf("expected nil response, got %v", resp)
	}

	if !errors.Is(err, transportErr) {
		t.Fatalf("expected transport error, got %v", err)
	}
	// 1 initial + 2 retries = 3 total
	if atomic.LoadInt32(&pipeline.attempts) != 3 {
		t.Fatalf("expected 3 total attempts (1 initial + 2 retries), got %d", atomic.LoadInt32(&pipeline.attempts))
	}
}

func TestRetryMiddleware_DoesNotRetryTransportErrorWhenShouldRetryReturnsFalse(t *testing.T) {
	transportErr := errors.New("connection refused")
	pipeline := &flakyPipeline{
		transport: http.DefaultTransport,
		failures:  100,
		err:       transportErr,
	}

	middleware := NewRetryMiddleware(RetryMiddlewareOptions{
		MaxRetries:   3,
		DelaySeconds: 1,
		ShouldRetry: func(_ int, _ *http.Request, resp *http.Response, err error) bool {
			// Only retry on responses, never on transport errors
			return resp != nil && resp.StatusCode >= 500
		},
	})

	req, _ := http.NewRequest("GET", "http://example.invalid/test", nil)
	resp, err := middleware.Intercept(pipeline, 0, req)
	if resp != nil {
		defer resp.Body.Close()
	}

	if !errors.Is(err, transportErr) {
		t.Fatalf("expected transport error, got %v", err)
	}
	if atomic.LoadInt32(&pipeline.attempts) != 1 {
		t.Fatalf("expected 1 attempt (no retry), got %d", atomic.LoadInt32(&pipeline.attempts))
	}
}

func TestGetForKiota_TransportErrorRetriedWhenRetryServerErrorsEnabled(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	server := httptest.NewServer(handler)
	defer server.Close()

	pipeline := &flakyPipeline{
		transport: http.DefaultTransport,
		failures:  1,
		err:       errors.New("connection refused"),
	}

	var hookCalls int32
	hook := func(retryCount int, response *http.Response) {
		atomic.AddInt32(&hookCalls, 1)
		if response != nil {
			t.Errorf("expected nil response in hook for transport error, got %v", response)
		}
	}

	middlewares, err := GetForKiota("1.0.0",
		WithRetryOptions(false, true, 3, hook),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var retryMW khttp.Middleware
	for _, mw := range middlewares {
		if _, ok := mw.(*RetryMiddleware); ok {
			retryMW = mw
			break
		}
	}
	if retryMW == nil {
		t.Fatal("RetryMiddleware not found in pipeline")
	}

	req, _ := http.NewRequest("GET", server.URL+"/test", nil)
	resp, err := retryMW.Intercept(pipeline, 0, req)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	if atomic.LoadInt32(&pipeline.attempts) != 2 {
		t.Fatalf("expected 2 attempts, got %d", atomic.LoadInt32(&pipeline.attempts))
	}
	if atomic.LoadInt32(&hookCalls) != 1 {
		t.Fatalf("expected hook called 1 time, got %d", atomic.LoadInt32(&hookCalls))
	}
}

func TestGetForKiota_TransportErrorNotRetriedWhenRetryServerErrorsDisabled(t *testing.T) {
	transportErr := errors.New("connection refused")
	pipeline := &flakyPipeline{
		transport: http.DefaultTransport,
		failures:  100,
		err:       transportErr,
	}

	// RetryRateLimited=true but RetryServerErrors=false: transport errors must not retry
	middlewares, err := GetForKiota("1.0.0",
		WithRetryOptions(true, false, 3, nil),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var retryMW khttp.Middleware
	for _, mw := range middlewares {
		if _, ok := mw.(*RetryMiddleware); ok {
			retryMW = mw
			break
		}
	}
	if retryMW == nil {
		t.Fatal("RetryMiddleware not found in pipeline")
	}

	req, _ := http.NewRequest("GET", "http://example.invalid/test", nil)
	resp, err := retryMW.Intercept(pipeline, 0, req)
	if resp != nil {
		defer resp.Body.Close()
	}

	if !errors.Is(err, transportErr) {
		t.Fatalf("expected transport error, got %v", err)
	}
	if atomic.LoadInt32(&pipeline.attempts) != 1 {
		t.Fatalf("expected 1 attempt (no retry), got %d", atomic.LoadInt32(&pipeline.attempts))
	}
}

func TestGetForKiota_ServerErrorRetriesIndependentOfRateLimitFlag(t *testing.T) {
	var attempts int32

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count := atomic.AddInt32(&attempts, 1)
		if count == 1 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	})

	pipeline, server := newTestPipeline(handler)
	defer server.Close()

	// RetryRateLimited=false, RetryServerErrors=true
	// Server error retries should work even when rate limit retries are disabled
	middlewares, err := GetForKiota("1.0.0",
		WithRetryOptions(false, true, 3, nil),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var retryMW khttp.Middleware
	for _, mw := range middlewares {
		if _, ok := mw.(*RetryMiddleware); ok {
			retryMW = mw
			break
		}
	}
	if retryMW == nil {
		t.Fatal("RetryMiddleware not found in pipeline")
	}

	req, _ := http.NewRequest("GET", server.URL+"/test", nil)
	resp, err := retryMW.Intercept(pipeline, 0, req)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	if atomic.LoadInt32(&attempts) != 2 {
		t.Fatalf("expected 2 attempts (retry on 500), got %d", atomic.LoadInt32(&attempts))
	}
}
