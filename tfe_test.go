package tfe

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"golang.org/x/time/rate"
)

func TestClient_newClient(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/vnd.api+json")
		w.Header().Set("X-RateLimit-Limit", "30")
		w.WriteHeader(404) // We query the configured base URL which should return a 404.
	}))
	defer ts.Close()

	cfg := &Config{
		HTTPClient: ts.Client(),
	}

	t.Run("uses env vars if values are missing", func(t *testing.T) {
		defer setupEnvVars("abcd1234", ts.URL)()

		client, err := NewClient(cfg)
		if err != nil {
			t.Fatal(err)
		}
		if client.token != "abcd1234" {
			t.Fatalf("unexpected token: %q", client.token)
		}
		if client.baseURL.String() != ts.URL+DefaultBasePath {
			t.Fatalf("unexpected address: %q", client.baseURL.String())
		}
	})

	t.Run("fails if token is empty", func(t *testing.T) {
		defer setupEnvVars("", "")()

		_, err := NewClient(cfg)
		if err == nil || err.Error() != "missing API token" {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("makes a new client with good settings", func(t *testing.T) {
		config := &Config{
			Address:    ts.URL,
			Token:      "abcd1234",
			HTTPClient: ts.Client(),
		}

		client, err := NewClient(config)
		if err != nil {
			t.Fatal(err)
		}

		if config.Address+DefaultBasePath != client.baseURL.String() {
			t.Fatalf("unexpected client address %q", client.baseURL.String())
		}
		if config.Token != client.token {
			t.Fatalf("unexpected client token %q", client.token)
		}
		if ts.Client() != client.http.HTTPClient {
			t.Fatal("unexpected HTTP client value")
		}
	})
}

func TestClient_defaultConfig(t *testing.T) {
	t.Run("with no environment variables", func(t *testing.T) {
		defer setupEnvVars("", "")()

		config := DefaultConfig()

		if config.Address != DefaultAddress {
			t.Fatalf("expected %q, got %q", DefaultAddress, config.Address)
		}
		if config.Token != "" {
			t.Fatalf("expected empty token, got %q", config.Token)
		}
		if config.HTTPClient == nil {
			t.Fatalf("expected default http client, got %v", config.HTTPClient)
		}
	})

	t.Run("with environment variables", func(t *testing.T) {
		defer setupEnvVars("abcd1234", "https://mytfe.local")()
	})
}

func TestClient_headers(t *testing.T) {
	testedCalls := 0
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		testedCalls++

		if testedCalls == 1 {
			w.Header().Set("Content-Type", "application/vnd.api+json")
			w.Header().Set("X-RateLimit-Limit", "30")
			w.WriteHeader(404) // We query the configured base URL which should return a 404.
			return
		}

		if r.Header.Get("Accept") != "application/vnd.api+json" {
			t.Fatalf("unexpected accept header: %q", r.Header.Get("Accept"))
		}
		if r.Header.Get("Authorization") != "Bearer dummy-token" {
			t.Fatalf("unexpected authorization header: %q", r.Header.Get("Authorization"))
		}
		if r.Header.Get("My-Custom-Header") != "foobar" {
			t.Fatalf("unexpected custom header: %q", r.Header.Get("My-Custom-Header"))
		}
		if r.Header.Get("Terraform-Version") != "0.11.9" {
			t.Fatalf("unexpected Terraform version header: %q", r.Header.Get("Terraform-Version"))
		}
		if r.Header.Get("User-Agent") != "go-tfe" {
			t.Fatalf("unexpected user agent header: %q", r.Header.Get("User-Agent"))
		}
	}))
	defer ts.Close()

	cfg := &Config{
		Address:    ts.URL,
		Token:      "dummy-token",
		Headers:    make(http.Header),
		HTTPClient: ts.Client(),
	}

	// Set some custom header.
	cfg.Headers.Set("My-Custom-Header", "foobar")
	cfg.Headers.Set("Terraform-Version", "0.11.9")

	// This one should be overridden!
	cfg.Headers.Set("Authorization", "bad-token")

	client, err := NewClient(cfg)
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()

	// Make a few calls so we can check they all send the expected headers.
	_, _ = client.Organizations.List(ctx, OrganizationListOptions{})
	_, _ = client.Plans.Logs(ctx, "plan-123456789")
	_ = client.Runs.Apply(ctx, "run-123456789", RunApplyOptions{})
	_, _ = client.Workspaces.Lock(ctx, "ws-123456789", WorkspaceLockOptions{})
	_, _ = client.Workspaces.Read(ctx, "organization", "workspace")

	if testedCalls != 6 {
		t.Fatalf("expected 6 tested calls, got: %d", testedCalls)
	}
}

func TestClient_userAgent(t *testing.T) {
	testedCalls := 0
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		testedCalls++

		if testedCalls == 1 {
			w.Header().Set("Content-Type", "application/vnd.api+json")
			w.Header().Set("X-RateLimit-Limit", "30")
			w.WriteHeader(404) // We query the configured base URL which should return a 404.
			return
		}

		if r.Header.Get("User-Agent") != "hashicorp" {
			t.Fatalf("unexpected user agent header: %q", r.Header.Get("User-Agent"))
		}
	}))
	defer ts.Close()

	cfg := &Config{
		Address:    ts.URL,
		Token:      "dummy-token",
		Headers:    make(http.Header),
		HTTPClient: ts.Client(),
	}

	// Set a custom user agent.
	cfg.Headers.Set("User-Agent", "hashicorp")

	client, err := NewClient(cfg)
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()

	// Make a few calls so we can check they all send the expected headers.
	_, _ = client.Organizations.List(ctx, OrganizationListOptions{})
	_, _ = client.Plans.Logs(ctx, "plan-123456789")
	_ = client.Runs.Apply(ctx, "run-123456789", RunApplyOptions{})
	_, _ = client.Workspaces.Lock(ctx, "ws-123456789", WorkspaceLockOptions{})
	_, _ = client.Workspaces.Read(ctx, "organization", "workspace")

	if testedCalls != 6 {
		t.Fatalf("expected 6 tested calls, got: %d", testedCalls)
	}
}

func TestClient_configureLimiter(t *testing.T) {
	rateLimit := ""
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/vnd.api+json")
		w.Header().Set("X-RateLimit-Limit", rateLimit)
		w.WriteHeader(404) // We query the configured base URL which should return a 404.
	}))
	defer ts.Close()

	cfg := &Config{
		Address:    ts.URL,
		Token:      "dummy-token",
		HTTPClient: ts.Client(),
	}

	cases := map[string]struct {
		rate  string
		limit rate.Limit
		burst int
	}{
		"no-value": {
			rate:  "",
			limit: rate.Inf,
			burst: 0,
		},
		"limit-0": {
			rate:  "0",
			limit: rate.Inf,
			burst: 0,
		},
		"limit-30": {
			rate:  "30",
			limit: rate.Limit(19.8),
			burst: 9,
		},
		"limit-100": {
			rate:  "100",
			limit: rate.Limit(66),
			burst: 33,
		},
	}

	for name, tc := range cases {
		// First set the test rate limit.
		rateLimit = tc.rate

		client, err := NewClient(cfg)
		if err != nil {
			t.Fatal(err)
		}

		if client.limiter.Limit() != tc.limit {
			t.Fatalf("test %s expected limit %f, got: %f", name, tc.limit, client.limiter.Limit())
		}

		if client.limiter.Burst() != tc.burst {
			t.Fatalf("test %s expected burst %d, got: %d", name, tc.burst, client.limiter.Burst())
		}
	}
}

func setupEnvVars(token, address string) func() {
	origToken := os.Getenv("TFE_TOKEN")
	origAddress := os.Getenv("TFE_ADDRESS")

	os.Setenv("TFE_TOKEN", token)
	os.Setenv("TFE_ADDRESS", address)

	return func() {
		os.Setenv("TFE_TOKEN", origToken)
		os.Setenv("TFE_ADDRESS", origAddress)
	}
}
