package tfe

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestNewClient(t *testing.T) {
	t.Run("uses env vars if values are missing", func(t *testing.T) {
		defer setupEnvVars("abcd1234", "https://mytfe.local")()

		client, err := NewClient(nil)
		if err != nil {
			t.Fatal(err)
		}
		if client.token != "abcd1234" {
			t.Fatalf("unexpected token: %q", client.token)
		}
		if client.baseURL.String() != "https://mytfe.local"+apiVersionPath {
			t.Fatalf("unexpected address: %q", client.baseURL.String())
		}
	})

	t.Run("fails if token is empty", func(t *testing.T) {
		defer setupEnvVars("", "")()

		_, err := NewClient(&Config{})
		if err == nil || err.Error() != "Missing API token" {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("makes a new client with good settings", func(t *testing.T) {
		httpClient := &http.Client{}

		config := &Config{
			Address:    "http://tfe.foo",
			Token:      "abcd1234",
			HTTPClient: httpClient,
		}

		client, err := NewClient(config)
		if err != nil {
			t.Fatal(err)
		}

		if config.Address+apiVersionPath != client.baseURL.String() {
			t.Fatalf("unexpected client address %q", client.baseURL.String())
		}
		if config.Token != client.token {
			t.Fatalf("unexpected client token %q", client.token)
		}
		if httpClient != client.http {
			t.Fatal("unexpected HTTP client value")
		}
	})

	t.Run("creates a default http client", func(t *testing.T) {
		defer setupEnvVars("", "")()

		client, err := NewClient(&Config{
			Token: "abcd1234",
		})
		if err != nil {
			t.Fatal(err)
		}

		if client.http == nil {
			t.Fatal("expected default http client, got nil")
		}
	})
}

func TestDefaultConfig(t *testing.T) {
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

func TestClientUpload(t *testing.T) {
	client := testClient(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/":
			// Redirect to ensure we can handle Archivist redirects during PUT.
			http.Redirect(w, r, "/redirected", 307)
			return
		case "/redirected":
			body, err := ioutil.ReadAll(r.Body)
			if err != nil {
				http.Error(w, "failed reading body", 500)
				return
			}

			if string(body) != "hello" {
				http.Error(w, "unexpected body", 400)
				return
			}
		case "/error":
			http.Error(w, "oops", 400)
			return
		}
	}))
	defer ts.Close()

	t.Run("with a valid request", func(t *testing.T) {
		err := client.upload(ts.URL, bytes.NewBufferString("hello"))
		if err != nil {
			t.Fatal(err)
		}
	})

	t.Run("when the url returns a bad status code", func(t *testing.T) {
		err := client.upload(ts.URL+"/error", &bytes.Buffer{})
		if err == nil {
			t.Fatalf("Expect status code error, got nil")
		}
	})
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
