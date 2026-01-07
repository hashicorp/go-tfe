// Copyright IBM Corp. 2018, 2025
// SPDX-License-Identifier: MPL-2.0

package tfe

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type tfeAPI struct {
	ID                string                   `jsonapi:"primary,tfe"`
	Name              string                   `jsonapi:"attr,name"`
	CreatedAt         time.Time                `jsonapi:"attr,created-at,iso8601"`
	Enabled           bool                     `jsonapi:"attr,enabled"`
	Emails            []string                 `jsonapi:"attr,emails"`
	Status            tfeAPIStatus             `jsonapi:"attr,status"`
	StatusTimestamps  tfeAPITimestamps         `jsonapi:"attr,status-timestamps"`
	DeliveryResponses []tfeAPIDeliveryResponse `jsonapi:"attr,delivery-responses"`
}

type tfeAPIDeliveryResponse struct {
	Body string `jsonapi:"attr,body"`
	Code int    `jsonapi:"attr,code"`
}

type tfeAPIStatus string

type tfeAPITimestamps struct {
	QueuedAt time.Time `jsonapi:"attr,queued-at,rfc3339"`
}

const (
	tfeAPIStatusNormal tfeAPIStatus = "normal"
)

func Test_unmarshalResponse(t *testing.T) {
	t.Parallel()
	t.Run("unmarshal properly formatted json", func(t *testing.T) {
		// This structure is intended to include multiple possible fields and
		// formats that are valid for JSON:API
		data := map[string]interface{}{
			"data": map[string]interface{}{
				"type": "tfe",
				"id":   "1",
				"attributes": map[string]interface{}{
					"name":       "terraform",
					"created-at": "2016-08-17T08:27:12Z",
					"enabled":    true,
					"status":     tfeAPIStatusNormal,
					"emails":     []string{"test@hashicorp.com"},
					"delivery-responses": []interface{}{
						map[string]interface{}{
							"body": "<html>",
							"code": 200,
						},
						map[string]interface{}{
							"body": "<body>",
							"code": 300,
						},
					},
					"status-timestamps": map[string]string{
						"queued-at": "2020-03-16T23:15:59+00:00",
					},
				},
			},
		}
		byteData, errMarshal := json.Marshal(data)
		require.NoError(t, errMarshal)
		responseBody := bytes.NewReader(byteData)

		unmarshalledRequestBody := tfeAPI{}
		err := unmarshalResponse(responseBody, &unmarshalledRequestBody)
		require.NoError(t, err)
		queuedParsedTime, err := time.Parse(time.RFC3339, "2020-03-16T23:15:59+00:00")
		require.NoError(t, err)

		assert.Equal(t, unmarshalledRequestBody.ID, "1")
		assert.Equal(t, unmarshalledRequestBody.Name, "terraform")
		assert.Equal(t, unmarshalledRequestBody.Status, tfeAPIStatusNormal)
		assert.Equal(t, len(unmarshalledRequestBody.Emails), 1)
		assert.Equal(t, unmarshalledRequestBody.Emails[0], "test@hashicorp.com")
		assert.Equal(t, unmarshalledRequestBody.StatusTimestamps.QueuedAt, queuedParsedTime)
		assert.NotEmpty(t, unmarshalledRequestBody.DeliveryResponses)
		assert.Equal(t, len(unmarshalledRequestBody.DeliveryResponses), 2)
		assert.Equal(t, unmarshalledRequestBody.DeliveryResponses[0].Body, "<html>")
		assert.Equal(t, unmarshalledRequestBody.DeliveryResponses[0].Code, 200)
		assert.Equal(t, unmarshalledRequestBody.DeliveryResponses[1].Body, "<body>")
		assert.Equal(t, unmarshalledRequestBody.DeliveryResponses[1].Code, 300)
		assert.Equal(t, unmarshalledRequestBody.Enabled, true)
	})

	t.Run("can only unmarshal Items that are slices", func(t *testing.T) {
		responseBody := bytes.NewReader([]byte(""))
		malformattedItemStruct := struct {
			*Pagination
			Items int
		}{
			Items: 1,
		}
		err := unmarshalResponse(responseBody, &malformattedItemStruct)
		require.Error(t, err)
		assert.Equal(t, err, ErrItemsMustBeSlice)
	})

	t.Run("can only unmarshal a struct", func(t *testing.T) {
		payload := "random"
		responseBody := bytes.NewReader([]byte(payload))

		notStruct := "not a struct"
		err := unmarshalResponse(responseBody, notStruct)
		assert.Error(t, err)
		assert.EqualError(t, err, fmt.Sprintf("%v must be a struct or an io.Writer", notStruct))
	})
}

func Test_BaseURL(t *testing.T) {
	t.Parallel()
	client, err := NewClient(&Config{
		Address:  "https://example.com",
		BasePath: "api/v99",
	})

	require.NoError(t, err)

	url := client.BaseURL()
	assert.Equal(t, "https://example.com/api/v99/", url.String())
}

func Test_DefaultBaseURL(t *testing.T) {
	t.Parallel()
	client, err := NewClient(&Config{
		Address: "https://example.com",
	})

	require.NoError(t, err)

	url := client.BaseURL()
	assert.Equal(t, "https://example.com/api/v2/", url.String())
}

func Test_DefaultRegistryBaseURL(t *testing.T) {
	t.Parallel()
	client, err := NewClient(&Config{
		Address: "https://example.com",
	})

	require.NoError(t, err)

	url := client.BaseRegistryURL()
	assert.Equal(t, "https://example.com/api/registry/", url.String())
}

func Test_RegistryBaseURL(t *testing.T) {
	t.Parallel()
	client, err := NewClient(&Config{
		Address:          "https://example.com",
		RegistryBasePath: "/api/registry99",
	})

	require.NoError(t, err)

	url := client.BaseRegistryURL()
	assert.Equal(t, "https://example.com/api/registry99/", url.String())
}

func Test_EncodeQueryParams(t *testing.T) {
	t.Parallel()
	t.Run("with no listOptions and therefore no include field defined", func(t *testing.T) {
		urlVals := map[string][]string{
			"include": {},
		}
		requestURLquery := encodeQueryParams(urlVals)
		assert.Equal(t, requestURLquery, "")
	})
	t.Run("with listOptions setting multiple include options", func(t *testing.T) {
		urlVals := map[string][]string{
			"include": {"workspace", "cost_estimate"},
		}
		requestURLquery := encodeQueryParams(urlVals)
		assert.Equal(t, requestURLquery, "include=workspace%2Ccost_estimate")
	})
}

func Test_RegistryBasePath(t *testing.T) {
	t.Parallel()
	client, err := NewClient(&Config{
		Token: "foo",
	})
	require.NoError(t, err)

	t.Run("ensures client creates a request with registry base path", func(t *testing.T) {
		path := "/api/registry/some/path/to/resource"
		req, err := client.NewRequest("GET", path, nil)
		require.NoError(t, err)

		expected := os.Getenv("TFE_ADDRESS") + path
		assert.Equal(t, req.retryableRequest.URL.String(), expected)
	})
}

func Test_NewRequest(t *testing.T) {
	t.Parallel()
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/get_request_with_query_param":
			val := r.URL.Query().Get("include")
			if val != "workspace,cost_estimate" {
				t.Fatalf("unexpected include value: %q", val)
			}
			w.WriteHeader(http.StatusOK)
			return
		case "/api/v2/ping":
			w.WriteHeader(http.StatusOK)
			return
		default:
			t.Fatalf("unexpected request: %s", r.URL.String())
		}
	}))

	t.Cleanup(func() {
		testServer.Close()
	})

	client, err := NewClient(&Config{
		Address: testServer.URL,
	})
	require.NoError(t, err)

	t.Run("allows path to include query params", func(t *testing.T) {
		request, err := client.NewRequest("GET", "/get_request_with_query_param?include=workspace,cost_estimate", nil)
		require.NoError(t, err)

		ctx := context.Background()
		err = request.DoJSON(ctx, nil)
		require.NoError(t, err)
	})
}

func Test_NewRequestWithAdditionalQueryParams(t *testing.T) {
	t.Parallel()
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/get_request_include":
			val := r.URL.Query().Get("include")
			if val != "workspace,cost_estimate" {
				t.Fatalf("unexpected include value: %q", val)
			}
			w.WriteHeader(http.StatusOK)
			return
		case "/get_request_include_extra":
			val := r.URL.Query().Get("include")
			if val != "workspace,cost_estimate" {
				t.Fatalf("unexpected include value: expected %q, got %q", "extra,workspace,cost_estimate", val)
			}
			extra := r.URL.Query().Get("extra")
			if extra != "value" {
				t.Fatalf("unexpected extra value: expected %q, got %q", "value", extra)
			}
			w.WriteHeader(http.StatusOK)
			return
		case "/get_request_include_raw":
			extra := r.URL.Query().Get("Name")
			if extra != "yes" {
				t.Fatalf("unexpected query: %s", r.URL.RawQuery)
			}
			w.WriteHeader(http.StatusOK)
			return
		case "/delete_with_query":
			extra := r.URL.Query().Get("extra")
			if extra != "value" {
				t.Fatalf("unexpected query: expected %q, got %q", "value", extra)
			}
			w.WriteHeader(http.StatusOK)
			return
		case "/api/v2/ping":
			w.WriteHeader(http.StatusOK)
			return
		default:
			t.Fatalf("unexpected request: %s", r.URL.String())
		}
	}))
	t.Cleanup(func() {
		testServer.Close()
	})

	client, err := NewClient(&Config{
		Address: testServer.URL,
	})
	require.NoError(t, err)

	t.Run("with additional query parameters", func(t *testing.T) {
		request, err := client.NewRequestWithAdditionalQueryParams("GET", "/get_request_include", nil, map[string][]string{
			"include": {"workspace", "cost_estimate"},
		})
		require.NoError(t, err)

		ctx := context.Background()
		err = request.DoJSON(ctx, nil)
		require.NoError(t, err)
	})

	type extra struct {
		Extra string `url:"extra"`
	}

	// json-encoded structs use the field name as the query parameter name
	type raw struct {
		Name string `json:"extra"`
	}

	t.Run("GET request with req attr and additional request attributes", func(t *testing.T) {
		request, err := client.NewRequestWithAdditionalQueryParams("GET", "/get_request_include_extra", &extra{Extra: "value"}, map[string][]string{
			"include": {"workspace", "cost_estimate"},
		})
		require.NoError(t, err)

		ctx := context.Background()
		err = request.DoJSON(ctx, nil)
		require.NoError(t, err)
	})

	t.Run("DELETE request with additional request attributes", func(t *testing.T) {
		request, err := client.NewRequestWithAdditionalQueryParams("DELETE", "/delete_with_query", nil, map[string][]string{
			"extra": {"value"},
		})
		require.NoError(t, err)

		ctx := context.Background()
		err = request.DoJSON(ctx, nil)
		require.NoError(t, err)
	})

	t.Run("GET request with other kinds of annotations", func(t *testing.T) {
		request, err := client.NewRequestWithAdditionalQueryParams("GET", "/get_request_include_raw", &raw{Name: "yes"}, nil)
		require.NoError(t, err)

		ctx := context.Background()
		err = request.DoJSON(ctx, nil)
		require.NoError(t, err)
	})
}
