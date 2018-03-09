package tfe

import (
	"net/http"
	"testing"
)

func TestNewClient(t *testing.T) {
	t.Run("fails if config is nil", func(t *testing.T) {
		_, err := NewClient(nil)
		if err == nil || err.Error() != "Missing client config" {
			t.Fatalf("expect missing client error, got %v", err)
		}
	})

	t.Run("fails if token is empty", func(t *testing.T) {
		_, err := NewClient(&Config{})
		if err == nil || err.Error() != "Missing client token" {
			t.Fatalf("expect missing token error, got %v", err)
		}
	})

	t.Run("makes a new client with good settings", func(t *testing.T) {
		httpClient := &http.Client{}

		client, err := NewClient(&Config{
			Address:    "http://tfe.foo",
			Token:      "abcd1234",
			HTTPClient: httpClient,
		})
		if err != nil {
			t.Fatal(err)
		}

		if v := client.config.Address; v != "http://tfe.foo" {
			t.Fatalf("unexpected address: %q", v)
		}
		if v := client.config.Token; v != "abcd1234" {
			t.Fatalf("unexpected token: %q", v)
		}
		if v := client.http; v != httpClient {
			t.Fatalf("unexpected http client: %v", v)
		}
	})

	t.Run("creates a default http client", func(t *testing.T) {
		client, err := NewClient(&Config{
			Token: "abcd1234",
		})
		if err != nil {
			t.Fatal(err)
		}
		if v := client.http; v == nil {
			t.Fatal("expected a default http client, got nil")
		}
	})
}

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()
	if v := config.Address; v != DefaultAddress {
		t.Fatalf("unexpected default address: %q", v)
	}
	if v := config.Token; v != "" {
		t.Fatalf("expect token to be empty, got %q", v)
	}
	if v := config.HTTPClient; v != nil {
		t.Fatalf("expect http client to be nil, got %v", v)
	}
}
