package tfe

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"reflect"
	"testing"

	"github.com/manyminds/api2go/jsonapi"
)

func TestNewClient(t *testing.T) {
	t.Run("uses env vars if values are missing", func(t *testing.T) {
		defer setupEnvVars("abcd1234", "https://mytfe.local")()

		client, err := NewClient(nil)
		if err != nil {
			t.Fatal(err)
		}
		if client.config.Token != "abcd1234" {
			t.Fatalf("unexpected token: %q", client.config.Token)
		}
		if client.config.Address != "https://mytfe.local" {
			t.Fatalf("unexpected address: %q", client.config.Address)
		}
	})

	t.Run("fails if token is empty", func(t *testing.T) {
		defer setupEnvVars("", "")()

		_, err := NewClient(&Config{})
		if err == nil || err.Error() != "Missing client token" {
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

		if config.Address != client.config.Address {
			t.Fatalf("unexpected client address %q", client.config.Address)
		}
		if config.Token != client.config.Token {
			t.Fatalf("unexpected client token %q", client.config.Token)
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
		if config.HTTPClient != nil {
			t.Fatal("expected http client to be nil")
		}
	})

	t.Run("with environment variables", func(t *testing.T) {
		defer setupEnvVars("abcd1234", "https://mytfe.local")()
	})
}

func TestClientRequest(t *testing.T) {
	var expectRequestBody string
	var responseBody string

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "PUT" {
			http.Error(w, "Bad HTTP method", 500)
			return
		}
		if r.URL.Path != "/hello" {
			http.Error(w, "Bad URL path", 500)
			return
		}
		if r.Header.Get("Authorization") != "Bearer FOOBARBAZ" {
			http.Error(w, "Bad Authorization header value", 500)
			return
		}
		if r.Header.Get("Content-Type") != "application/vnd.api+json" {
			http.Error(w, "Bad Content-Type header value", 500)
			return
		}
		if r.Header.Get("ExtraHeader") != "yes" {
			http.Error(w, "Extra header not preserved", 500)
			return
		}
		if r.URL.Query().Get("page[number]") != "3" {
			http.Error(w, "Bad page number value", 500)
			return
		}
		if r.URL.Query().Get("page[size]") != "50" {
			http.Error(w, "Bad page size value", 500)
			return
		}
		if r.URL.Query().Get("extra_query") != "yes" {
			http.Error(w, "Extra query param not preserved", 500)
			return
		}
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		if string(body) != expectRequestBody {
			http.Error(w, "Unexpected request body", 500)
			return
		}

		w.WriteHeader(200)
		w.Write([]byte(responseBody))
	}))
	defer ts.Close()

	client := testClient(t, func(c *Config) {
		c.Address = ts.URL
		c.Token = "FOOBARBAZ"
	})

	t.Run("raw input and output", func(t *testing.T) {
		expectRequestBody = "hello from client"
		responseBody = "hello from server"

		resp, err := client.do(&request{
			method: "PUT",
			path:   "/hello",
			query:  url.Values{"extra_query": {"yes"}},
			header: http.Header{"ExtraHeader": {"yes"}},
			body:   bytes.NewBufferString(expectRequestBody),
			lopt: ListOptions{
				PageNumber: 3,
				PageSize:   50,
			},
		})
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()

		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			t.Fatal(err)
		}
		if v := string(body); v != responseBody {
			t.Fatalf("Unexpected response body: %q", v)
		}
	})

	t.Run("input and output references", func(t *testing.T) {
		input := apiTestModel{Name: "yo"}

		serialized, err := jsonapi.Marshal(input)
		if err != nil {
			t.Fatal(err)
		}

		expectRequestBody = string(serialized)
		responseBody = string(serialized)

		var output apiTestModel

		if _, err := client.do(&request{
			method: "PUT",
			path:   "/hello",
			query:  url.Values{"extra_query": {"yes"}},
			header: http.Header{"ExtraHeader": {"yes"}},
			input:  input,
			output: &output,
			lopt: ListOptions{
				PageNumber: 3,
				PageSize:   50,
			},
		}); err != nil {
			t.Fatal(err)
		}

		if !reflect.DeepEqual(input, output) {
			t.Fatalf("\nExpect:\n%+v\n\nActual:\n%+v", input, output)
		}
	})
}

func TestClient_upload(t *testing.T) {
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

type apiTestModel struct {
	Name string `json:"name"`
}

func (apiTestModel) GetName() string    { return "api-test-model" }
func (apiTestModel) GetID() string      { return "" }
func (apiTestModel) SetID(string) error { return nil }

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
