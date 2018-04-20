package tfe

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewClient(t *testing.T) {
	t.Run("fails if config is nil", func(t *testing.T) {
		_, err := NewClient(nil)
		assert.EqualError(t, err, "Missing client config")
	})

	t.Run("fails if token is empty", func(t *testing.T) {
		_, err := NewClient(&Config{})
		assert.EqualError(t, err, "Missing client token")
	})

	t.Run("makes a new client with good settings", func(t *testing.T) {
		httpClient := &http.Client{}

		config := &Config{
			Address:    "http://tfe.foo",
			Token:      "abcd1234",
			HTTPClient: httpClient,
		}

		client, err := NewClient(config)
		assert.Nil(t, err)

		assert.Equal(t, config.Address, client.config.Address)
		assert.Equal(t, config.Token, client.config.Token)
		assert.Equal(t, httpClient, client.http)
	})

	t.Run("creates a default http client", func(t *testing.T) {
		client, err := NewClient(&Config{
			Token: "abcd1234",
		})
		assert.Nil(t, err)

		assert.NotNil(t, client.http)
	})
}

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	assert.Equal(t, DefaultAddress, config.Address)
	assert.Equal(t, "", config.Token)
	assert.Nil(t, config.HTTPClient)
}

func TestRequest(t *testing.T) {
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
		if string(body) != "hello from the client" {
			http.Error(w, "Unexpected request body", 500)
			return
		}

		w.WriteHeader(200)
		w.Write([]byte("hello from the server"))
	}))
	defer ts.Close()

	client := testClient(t, func(c *Config) {
		c.Address = ts.URL
		c.Token = "FOOBARBAZ"
	})

	resp, err := client.do(&request{
		method: "PUT",
		path:   "/hello",
		query:  url.Values{"extra_query": {"yes"}},
		header: http.Header{"ExtraHeader": {"yes"}},
		body:   bytes.NewBufferString("hello from the client"),
		lopt: &ListOptions{
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
	if v := string(body); v != "hello from the server" {
		t.Fatalf("Unexpected response body: %q", v)
	}
}
