package tfe

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	retryablehttp "github.com/hashicorp/go-retryablehttp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fixtureBody struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Method string `json:"method"`
}

func newTestRequest(r *retryablehttp.Request) ClientRequest {
	header := make(http.Header)
	header.Add("TestHeader", "test-header-value")

	return ClientRequest{
		retryableRequest: r,
		http:             retryablehttp.NewClient(),
		Header:           header,
	}
}

func TestClientRequest_DoJSON(t *testing.T) {
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fakeBody := map[string]any{
			"id":     "example",
			"name":   "fixture",
			"method": r.Method,
		}
		fakeBodyRaw, err := json.Marshal(fakeBody)
		require.NoError(t, err)

		if strings.HasSuffix(r.URL.String(), "/ok_request") {
			w.Header().Set("content-type", "application/json")
			w.Header().Set("content-length", strconv.FormatInt(int64(len(fakeBodyRaw)), 10))
			w.WriteHeader(http.StatusOK)
			_, err = w.Write(fakeBodyRaw)
			require.NoError(t, err)
		} else if strings.HasSuffix(r.URL.String(), "/bad_request") {
			w.WriteHeader(http.StatusBadRequest)
		} else if strings.HasSuffix(r.URL.String(), "/created_request") {
			w.WriteHeader(http.StatusCreated)
		} else if strings.HasSuffix(r.URL.String(), "/not_modified_request") {
			w.WriteHeader(http.StatusNotModified)
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(func() {
		testServer.Close()
	})

	t.Run("Success 200 responses", func(t *testing.T) {
		r, err := retryablehttp.NewRequest("PUT", fmt.Sprintf("%s/ok_request", testServer.URL), nil)
		require.NoError(t, err)

		ctx := context.Background()

		request := newTestRequest(r)
		putResponseBody := &fixtureBody{}
		err = request.DoJSON(ctx, putResponseBody)
		require.NoError(t, err)

		assert.Equal(t, "example", putResponseBody.ID)
		assert.Equal(t, "fixture", putResponseBody.Name)
		assert.Equal(t, "PUT", putResponseBody.Method)
	})

	t.Run("Success response with no body", func(t *testing.T) {
		r, err := retryablehttp.NewRequest("POST", fmt.Sprintf("%s/created_request", testServer.URL), nil)
		require.NoError(t, err)

		ctx := context.Background()

		request := newTestRequest(r)
		err = request.DoJSON(ctx, nil)
		require.NoError(t, err)
	})

	t.Run("Not Modified response", func(t *testing.T) {
		r, err := retryablehttp.NewRequest("POST", fmt.Sprintf("%s/not_modified_request", testServer.URL), nil)
		require.NoError(t, err)

		ctx := context.Background()

		request := newTestRequest(r)
		postResponseBody := &fixtureBody{}
		err = request.DoJSON(ctx, postResponseBody)
		require.NoError(t, err)

		assert.Empty(t, postResponseBody.Method)
		assert.Empty(t, postResponseBody.ID)
	})

	t.Run("Bad 400 responses", func(t *testing.T) {
		r, err := retryablehttp.NewRequest("POST", fmt.Sprintf("%s/bad_request", testServer.URL), nil)
		require.NoError(t, err)

		ctx := context.Background()

		request := newTestRequest(r)
		postResponseBody := &fixtureBody{}
		err = request.DoJSON(ctx, postResponseBody)

		// body is empty (no response)
		assert.Empty(t, postResponseBody.Method)
		assert.Empty(t, postResponseBody.ID)

		assert.EqualError(t, err, "error HTTP response: 400")
	})
}
