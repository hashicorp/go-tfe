// Copyright IBM Corp. 2018, 2026

package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	khttp "github.com/microsoft/kiota-http-go"
)

// TestSafeHeadersInspectionHandler_NoConcurrentMapWrite reproduces the race
// that was present when khttp.HeadersInspectionHandler was used directly in
// the middleware pipeline. With default pipeline settings no per-request
// HeadersInspectionOptions is placed in the context, so previously all
// goroutines shared &middleware.options and raced on the embedded
// ResponseHeaders map.
//
// Run with: go test -race ./middleware/ -run TestSafeHeadersInspectionHandler_NoConcurrentMapWrite
func TestSafeHeadersInspectionHandler_NoConcurrentMapWrite(t *testing.T) {
	// Server that always responds with a handful of headers.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("X-Request-Id", "abc123")
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-RateLimit-Limit", "100")
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	handler := newSafeHeadersInspectionHandler(false, true)

	// pipeline terminates at the test server.
	pipeline := &testPipeline{transport: http.DefaultTransport}

	const goroutines = 20
	var wg sync.WaitGroup
	wg.Add(goroutines)

	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, srv.URL, nil)
			if err != nil {
				t.Errorf("new request: %v", err)
				return
			}
			resp, err := handler.Intercept(pipeline, 0, req)
			if err != nil {
				t.Errorf("Intercept: %v", err)
				return
			}
			resp.Body.Close()
		}()
	}

	wg.Wait()
}

// TestSafeHeadersInspectionHandler_PerRequestOptionHonoured verifies that
// when a caller injects their own HeadersInspectionOptions into the request
// context, the handler populates that caller-provided instance rather than
// discarding it.
func TestSafeHeadersInspectionHandler_PerRequestOptionHonoured(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("X-Custom-Header", "hello")
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	handler := newSafeHeadersInspectionHandler(false, true)
	pipeline := &testPipeline{transport: http.DefaultTransport}

	inspectionOpts := khttp.NewHeadersInspectionOptions()
	inspectionOpts.InspectResponseHeaders = true

	ctx := context.WithValue(context.Background(), headersInspectionReqKey, inspectionOpts)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, srv.URL, nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}

	resp, err := handler.Intercept(pipeline, 0, req)
	if err != nil {
		t.Fatalf("Intercept: %v", err)
	}
	resp.Body.Close()

	got := inspectionOpts.GetResponseHeaders().Get("x-custom-header")
	if len(got) == 0 || got[0] != "hello" {
		t.Errorf("expected ResponseHeaders to contain X-Custom-Header=hello, got %v", got)
	}
}

// TestSafeHeadersInspectionHandler_RequestHeadersInspected verifies that when
// InspectRequestHeaders is true the outgoing request headers are captured into
// the RequestHeaders of the per-request option.
func TestSafeHeadersInspectionHandler_RequestHeadersInspected(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	handler := newSafeHeadersInspectionHandler(true, false)
	pipeline := &testPipeline{transport: http.DefaultTransport}

	inspectionOpts := khttp.NewHeadersInspectionOptions()
	inspectionOpts.InspectRequestHeaders = true

	ctx := context.WithValue(context.Background(), headersInspectionReqKey, inspectionOpts)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, srv.URL, nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("X-Trace-Id", "trace-42")

	resp, err := handler.Intercept(pipeline, 0, req)
	if err != nil {
		t.Fatalf("Intercept: %v", err)
	}
	resp.Body.Close()

	got := inspectionOpts.GetRequestHeaders().Get("x-trace-id")
	if len(got) == 0 || got[0] != "trace-42" {
		t.Errorf("expected RequestHeaders to contain X-Trace-Id=trace-42, got %v", got)
	}
}
