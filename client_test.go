package tfe

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"testing"

	"github.com/manyminds/api2go/jsonapi"
)

func TestNewClient(t *testing.T) {
	t.Run("fails if config is nil", func(t *testing.T) {
		_, err := NewClient(nil)
		if err == nil || err.Error() != "Missing client config" {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("fails if token is empty", func(t *testing.T) {
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

type apiTestModel struct {
	Name string `json:"name"`
}

func (apiTestModel) GetName() string    { return "api-test-model" }
func (apiTestModel) GetID() string      { return "" }
func (apiTestModel) SetID(string) error { return nil }
