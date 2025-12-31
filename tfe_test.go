// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package tfe

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hashicorp/go-tfe/api/models"
	"github.com/stretchr/testify/assert"
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
		Address: ts.URL,
		Token:   "test-token",
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
			Address: ts.URL,
		}

		_, err := NewClient(cfg)
		if err == nil || err.Error() != "missing API token" {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("makes a new client with good settings", func(t *testing.T) {
		config := &Config{
			Address: ts.URL,
			Token:   "abcd1234",
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
		Address: ts.URL,
		Token:   "abcd1234",
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
	})
}
