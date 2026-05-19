// Copyright IBM Corp. 2026
// SPDX-License-Identifier: MPL-2.0

package tfe

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFollowAPIRedirect_SingleRedirect(t *testing.T) {
	finalServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("hello from archivist")) //nolint:errcheck
	}))
	t.Cleanup(finalServer.Close)

	resp := &http.Response{
		StatusCode: http.StatusFound,
		Header:     http.Header{"Location": []string{finalServer.URL}},
		Body:       io.NopCloser(strings.NewReader("")),
	}

	client := &Client{}
	body, err := client.FollowAPIRedirect(context.Background(), resp)
	require.NoError(t, err)
	defer body.Close()

	data, err := io.ReadAll(body)
	require.NoError(t, err)
	assert.Equal(t, "hello from archivist", string(data))
}

func TestFollowAPIRedirect_TFEToArchivistToS3(t *testing.T) {
	s3Server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("plan export content from s3")) //nolint:errcheck
	}))
	t.Cleanup(s3Server.Close)

	archivistServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Location", s3Server.URL+"/bucket/object?X-Amz-Signature=abc123")
		w.WriteHeader(http.StatusTemporaryRedirect)
	}))
	t.Cleanup(archivistServer.Close)

	resp := &http.Response{
		StatusCode: http.StatusFound,
		Header:     http.Header{"Location": []string{archivistServer.URL + "/v1/object/abc"}},
		Body:       io.NopCloser(strings.NewReader("")),
	}

	client := &Client{}
	body, err := client.FollowAPIRedirect(context.Background(), resp)
	require.NoError(t, err)
	defer body.Close()

	data, err := io.ReadAll(body)
	require.NoError(t, err)
	assert.Equal(t, "plan export content from s3", string(data))
}

func TestFollowAPIRedirect_ArchivistNoRedirect(t *testing.T) {
	archivistServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("proxied content from archivist")) //nolint:errcheck
	}))
	t.Cleanup(archivistServer.Close)

	resp := &http.Response{
		StatusCode: http.StatusFound,
		Header:     http.Header{"Location": []string{archivistServer.URL + "/v1/object/abc"}},
		Body:       io.NopCloser(strings.NewReader("")),
	}

	client := &Client{}
	body, err := client.FollowAPIRedirect(context.Background(), resp)
	require.NoError(t, err)
	defer body.Close()

	data, err := io.ReadAll(body)
	require.NoError(t, err)
	assert.Equal(t, "proxied content from archivist", string(data))
}

func TestFollowAPIRedirect_RedirectLoop(t *testing.T) {
	var serverA, serverB *httptest.Server

	serverA = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Location", serverB.URL)
		w.WriteHeader(http.StatusTemporaryRedirect)
	}))
	t.Cleanup(serverA.Close)

	serverB = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Location", serverA.URL)
		w.WriteHeader(http.StatusTemporaryRedirect)
	}))
	t.Cleanup(serverB.Close)

	resp := &http.Response{
		StatusCode: http.StatusFound,
		Header:     http.Header{"Location": []string{serverA.URL}},
		Body:       io.NopCloser(strings.NewReader("")),
	}

	client := &Client{}
	_, err := client.FollowAPIRedirect(context.Background(), resp)
	assert.ErrorIs(t, err, ErrRedirectLoop)
}

func TestFollowAPIRedirect_Non200FinalResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	t.Cleanup(server.Close)

	resp := &http.Response{
		StatusCode: http.StatusFound,
		Header:     http.Header{"Location": []string{server.URL}},
		Body:       io.NopCloser(strings.NewReader("")),
	}

	client := &Client{}
	_, err := client.FollowAPIRedirect(context.Background(), resp)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unexpected response status: 403")
}

func TestFollowAPIRedirect_StreamingBody(t *testing.T) {
	largeContent := strings.Repeat("x", 1024*1024)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(largeContent)) //nolint:errcheck
	}))
	t.Cleanup(server.Close)

	resp := &http.Response{
		StatusCode: http.StatusFound,
		Header:     http.Header{"Location": []string{server.URL}},
		Body:       io.NopCloser(strings.NewReader("")),
	}

	client := &Client{}
	body, err := client.FollowAPIRedirect(context.Background(), resp)
	require.NoError(t, err)
	defer body.Close()

	buf := make([]byte, 10)
	n, err := body.Read(buf)
	require.NoError(t, err)
	assert.Equal(t, 10, n)
	assert.Equal(t, "xxxxxxxxxx", string(buf))
}

func TestFollowAPIRedirect_NoAuthHeaderSent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Empty(t, r.Header.Get("Authorization"))
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok")) //nolint:errcheck
	}))
	t.Cleanup(server.Close)

	resp := &http.Response{
		StatusCode: http.StatusFound,
		Header:     http.Header{"Location": []string{server.URL}},
		Body:       io.NopCloser(strings.NewReader("")),
	}

	client := &Client{token: "secret-token"}
	body, err := client.FollowAPIRedirect(context.Background(), resp)
	require.NoError(t, err)
	body.Close()
}
