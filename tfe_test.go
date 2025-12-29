// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package tfe

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/hashicorp/go-tfe/api/models"
	"github.com/stretchr/testify/assert"
	"golang.org/x/time/rate"
)

func setDefaultServerHeaders(w http.ResponseWriter) {
	w.Header().Set("Content-Type", ContentTypeJSONAPI)
	w.Header().Set("X-RateLimit-Limit", "30")
	w.Header().Set("TFP-API-Version", "34.21.9")
	w.Header().Set("X-TFE-Version", "202205-1")
	w.Header().Set("TFP-AppName", "HCP Terraform")
}

func testServer(t *testing.T, handlers map[string]http.HandlerFunc) *httptest.Server {
	t.Helper()

	mux := http.NewServeMux()

	for pattern, fn := range handlers {
		mux.HandleFunc(pattern, fn)
	}

	ts := httptest.NewServer(mux)
	t.Cleanup(ts.Close)
	return ts
}

func testServerWithClient(t *testing.T, handlers map[string]http.HandlerFunc) (*httptest.Server, *Client) {
	ts := testServer(t, handlers)

	client, err := NewClient(&Config{
		HTTPClient: ts.Client(),
		Token:      "test-token",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	return ts, client
}

func Test_NewClient(t *testing.T) {
	ts := testServer(t, map[string]http.HandlerFunc{
		"/": func(w http.ResponseWriter, r *http.Request) {
			setDefaultServerHeaders(w)
			w.WriteHeader(204)
		}})

	t.Run("fails if token is empty", func(t *testing.T) {
		cfg := &Config{
			HTTPClient: ts.Client(),
		}

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

func TestClient_API(t *testing.T) {
	ts := testServer(t, map[string]http.HandlerFunc{
		"/api/v2/account/details": func(w http.ResponseWriter, r *http.Request) {
			setDefaultServerHeaders(w)

			w.WriteHeader(200)
			w.Write([]byte(`{
	"data": {
		"id": "usr-1234",
		"type": "users",
		"attributes": {
			"email": "test@hashicorp.com"
		}
	}
}`))
		},
		"/": func(w http.ResponseWriter, r *http.Request) {
			setDefaultServerHeaders(w)
			w.WriteHeader(404)
			w.Write([]byte(`{
	"errors": [
		{
			"status": "404",
			"title": "resource not found"
		}
	]
}`))
		}})

	cfg := &Config{
		HTTPClient: ts.Client(),
		Address:    ts.URL,
		Token:      "abcd1234",
	}

	t.Run("basic success", func(t *testing.T) {
		client, err := NewClient(cfg)
		if err != nil {
			t.Fatal(err)
		}

		response, err := client.API.Account().Details().Get(context.Background(), nil)
		if err != nil {
			t.Fatalf("Failed to fetch Account Details: %s", err)
		}

		expected := "test@hashicorp.com"
		if actual := *response.GetData().GetAttributes().GetEmail(); actual != expected {
			t.Errorf("expected account details data attribute email to be %q, got %q", expected, actual)
		}

		expected = "usr-1234"
		if actual := *response.GetData().GetId(); actual != expected {
			t.Errorf("expected account details id to be %q, got %q", expected, actual)
		}
	})

	t.Run("basic not found", func(t *testing.T) {
		client, err := NewClient(cfg)
		if err != nil {
			t.Fatal(err)
		}

		response, err := client.API.Organizations().ByOrganization_name("hashicorp").GetAsWithOrganization_nameGetResponse(context.Background(), nil)
		merr, ok := err.(*models.Errors)
		if !ok {
			t.Fatalf("expected *models.Errors, got %T", err)
		}

		if merr.ResponseStatusCode != 404 {
			t.Errorf("expected status code %d, got %d", 404, merr.ResponseStatusCode)
		}

		if len(merr.GetErrors()) != 1 {
			t.Fatalf("expected %d errors, got %d", 1, len(merr.GetErrors()))
		}

		for _, msg := range merr.GetErrors() {
			expected := "404"
			if actual := *msg.GetStatus(); actual != expected {
				t.Fatalf("expected error status %q, got %q", expected, actual)
			}

			expected = "resource not found"
			if actual := *msg.GetTitle(); actual != expected {
				t.Fatalf("expected error title %q, got %q", expected, actual)
			}
		}

		if response != nil {
			t.Fatalf("expected nil organization response, got %v", response)
		}
	})
}

func TestClient_defaultConfig(t *testing.T) {
	t.Run("with no environment variables", func(t *testing.T) {
		config := DefaultConfig()

		assert.Equal(t, config.Address, DefaultAddress)
		assert.Equal(t, config.Token, "")
		assert.NotNil(t, config.HTTPClient)
	})
}

func TestClient_configureLimiter(t *testing.T) {
	t.SkipNow()

	rateLimit := ""
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", ContentTypeJSONAPI)
		w.Header().Set("X-RateLimit-Limit", rateLimit)
		w.WriteHeader(204) // We query the configured ping URL which should return a 204.
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

func TestClient_retryHTTPCheck(t *testing.T) {
	ts := testServer(t, map[string]http.HandlerFunc{
		"/": func(w http.ResponseWriter, r *http.Request) {
			setDefaultServerHeaders(w)
			w.WriteHeader(204)
		},
	})

	cfg := &Config{
		Address:    ts.URL,
		Token:      "dummy-token",
		HTTPClient: ts.Client(),
	}

	connErr := errors.New("connection error")

	cases := map[string]struct {
		resp              *http.Response
		err               error
		retryServerErrors bool
		checkOK           bool
		checkErr          error
	}{
		"429-no-server-errors": {
			resp:     &http.Response{StatusCode: 429},
			err:      nil,
			checkOK:  true,
			checkErr: nil,
		},
		"429-with-server-errors": {
			resp:              &http.Response{StatusCode: 429},
			err:               nil,
			retryServerErrors: true,
			checkOK:           true,
			checkErr:          nil,
		},
		"500-no-server-errors": {
			resp:     &http.Response{StatusCode: 500},
			err:      nil,
			checkOK:  false,
			checkErr: nil,
		},
		"500-with-server-errors": {
			resp:              &http.Response{StatusCode: 500},
			err:               nil,
			retryServerErrors: true,
			checkOK:           true,
			checkErr:          nil,
		},
		"err-no-server-errors": {
			err:      connErr,
			checkOK:  false,
			checkErr: connErr,
		},
		"err-with-server-errors": {
			err:               connErr,
			retryServerErrors: true,
			checkOK:           true,
			checkErr:          connErr,
		},
	}

	ctx := context.Background()

	for name, tc := range cases {
		client, err := NewClient(cfg)
		if err != nil {
			t.Fatal(err)
		}

		client.RetryServerErrors(tc.retryServerErrors)

		checkOK, checkErr := client.retryHTTPCheck(ctx, tc.resp, tc.err)
		if checkOK != tc.checkOK {
			t.Fatalf("test %s expected checkOK %t, got: %t", name, tc.checkOK, checkOK)
		}
		if checkErr != tc.checkErr {
			t.Fatalf("test %s expected checkErr %v, got: %v", name, tc.checkErr, checkErr)
		}
	}
}

func TestClient_retryHTTPBackoff(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", ContentTypeJSONAPI)
		w.Header().Set("X-RateLimit-Limit", "30")
		w.WriteHeader(204) // We query the configured ping URL which should return a 204.
	}))
	defer ts.Close()

	var attempts int
	retryLogHook := func(attemptNum int, resp *http.Response) {
		attempts++
	}

	cfg := &Config{
		Address:      ts.URL,
		Token:        "dummy-token",
		HTTPClient:   ts.Client(),
		RetryLogHook: retryLogHook,
	}

	client, err := NewClient(cfg)
	if err != nil {
		t.Fatal(err)
	}

	retries := 3
	resp := &http.Response{StatusCode: 500}

	for i := 0; i < retries; i++ {
		client.retryHTTPBackoff(time.Second, time.Second, i, resp)
	}

	if attempts != retries {
		t.Fatalf("expected %d log hook callbacks, got: %d callbacks", retries, attempts)
	}
}
