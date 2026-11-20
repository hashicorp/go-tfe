// Copyright IBM Corp. 2018, 2026
// SPDX-License-Identifier: MPL-2.0

package tfe

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/hashicorp/go-tfe/v2/api/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUploadConfigReplayAndIsolation(t *testing.T) {
	pack := testPackedConfig(t)
	data, err := pack.archiveBytes()
	require.NoError(t, err)
	var requests atomic.Int32
	const targetPath = "/outside-api/a%2Fb?sig=dummy%2Bvalue&part=one&part=two"
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPut, r.Method)
		assert.Equal(t, targetPath, r.RequestURI)
		assert.Equal(t, "application/octet-stream", r.Header.Get("Content-Type"))
		assert.Equal(t, int64(len(data)), r.ContentLength)
		assert.Empty(t, r.TransferEncoding)
		for _, header := range []string{"Authorization", "Cookie", "X-API-Private"} {
			assert.Empty(t, r.Header.Values(header))
		}
		body, err := io.ReadAll(r.Body)
		assert.NoError(t, err)
		assert.Equal(t, data, body)
		if requests.Add(1) == 1 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	t.Cleanup(srv.Close)
	var apiCalls, apiRetries atomic.Int32
	client, err := NewClient(&Config{
		Address: srv.URL,
		Token:   "api-private-token",
		Headers: http.Header{"Cookie": {"session=private"}, "X-Api-Private": {"private"}},
		HTTPTransport: configRoundTripper(func(*http.Request) (*http.Response, error) {
			apiCalls.Add(1)
			return nil, errors.New("unexpected API request")
		}),
		RetryServerErrors: true,
		RetryHook:         func(int, *http.Response) { apiRetries.Add(1) },
	})
	require.NoError(t, err)
	useLocalUploadTLS(client, srv)
	require.NoError(t, client.UploadConfig(t.Context(), configUploadVersion(srv.URL+targetPath), pack))
	require.EqualValues(t, 2, requests.Load())
	require.Zero(t, apiCalls.Load())
	require.Zero(t, apiRetries.Load())
	_, err = pack.archiveBytes()
	require.NoError(t, err, "UploadConfig must not close caller-owned storage")
}

func TestUploadConfigResponses(t *testing.T) {
	for _, test := range []struct {
		status   int
		attempts int32
		want     string
	}{
		{http.StatusTemporaryRedirect, 1, "redirect refused"},
		{http.StatusForbidden, 1, "HTTP 403"},
		{http.StatusBadGateway, 3, "HTTP 502"},
	} {
		t.Run(test.want, func(t *testing.T) {
			const secret = "dummy-capability-secret"
			var attempts, redirected atomic.Int32
			srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/redirect" {
					redirected.Add(1)
					w.WriteHeader(http.StatusNoContent)
					return
				}
				attempts.Add(1)
				w.Header().Set("Location", "/redirect?sig="+secret)
				w.WriteHeader(test.status)
				_, _ = io.WriteString(w, secret)
			}))
			t.Cleanup(srv.Close)
			client := newTestConfigClient(t)
			useLocalUploadTLS(client, srv)
			err := client.UploadConfig(t.Context(), configUploadVersion(srv.URL+"/object?sig="+secret), testPackedConfig(t))
			require.ErrorContains(t, err, test.want)
			require.NotContains(t, err.Error(), secret)
			require.Equal(t, test.attempts, attempts.Load())
			require.Zero(t, redirected.Load())
		})
	}
}

func TestUploadConfigValidation(t *testing.T) {
	client := newTestConfigClient(t)
	var requests atomic.Int32
	client.configUploadClient = &http.Client{Transport: configRoundTripper(func(*http.Request) (*http.Response, error) {
		requests.Add(1)
		return nil, errors.New("unexpected request")
	})}
	pack := testPackedConfig(t)
	const secret = "dummy-capability-secret"
	for _, target := range []string{
		"/object?sig=" + secret,
		"http://example.test/" + secret,
		"https://" + secret + "@example.test/object",
		"https://example.test/object#" + secret,
		"https://example.test/%zz?sig=" + secret,
		"https://:443/object?sig=" + secret,
	} {
		err := client.UploadConfig(t.Context(), configUploadVersion(target), pack)
		require.ErrorContains(t, err, "invalid HTTPS upload URL")
		require.NotContains(t, err.Error(), secret)
	}
	var nilVersion *models.ConfigurationVersions
	var nilAttributes *models.ConfigurationVersions_attributes
	versionWithoutAttributes := models.NewConfigurationVersions()
	versionWithoutAttributes.SetAttributes(nilAttributes)
	for _, version := range []models.ConfigurationVersionsable{nil, nilVersion, versionWithoutAttributes, configUploadVersion("")} {
		require.ErrorContains(t, client.UploadConfig(t.Context(), version, pack), "has no upload URL")
	}
	target := configUploadVersion("https://example.test/object")
	require.ErrorContains(t, client.UploadConfig(t.Context(), target, nil), "a packed config is required")
	require.NoError(t, pack.Close())
	require.ErrorContains(t, client.UploadConfig(t.Context(), target, pack), "closed or uninitialized")
	require.Zero(t, requests.Load())
}

func TestUploadConfigTransportErrorAndCancellation(t *testing.T) {
	t.Run("transport errors are sanitized", func(t *testing.T) {
		const secret = "dummy-capability-secret"
		client := newTestConfigClient(t)
		client.configUploadClient = &http.Client{Transport: configRoundTripper(func(r *http.Request) (*http.Response, error) {
			return nil, errors.New(r.URL.String())
		})}
		err := client.UploadConfig(t.Context(), configUploadVersion("https://example.test/object?sig="+secret), testPackedConfig(t))
		require.ErrorContains(t, err, "network request failed")
		require.NotContains(t, err.Error(), secret)
	})
	t.Run("deadline interrupts backoff", func(t *testing.T) {
		client := newTestConfigClient(t)
		var attempts atomic.Int32
		client.configUploadClient = &http.Client{Transport: configRoundTripper(func(*http.Request) (*http.Response, error) {
			attempts.Add(1)
			return nil, errors.New("retryable")
		})}
		pack := testPackedConfig(t)
		ctx, cancel := context.WithTimeout(t.Context(), 100*time.Millisecond)
		defer cancel()
		err := client.UploadConfig(ctx, configUploadVersion("https://example.test/object"), pack)
		require.ErrorIs(t, err, context.DeadlineExceeded)
		require.EqualValues(t, 1, attempts.Load())
	})
}

func useLocalUploadTLS(client *Client, server *httptest.Server) {
	client.configUploadClient = &http.Client{
		Transport:     server.Client().Transport,
		CheckRedirect: defaultConfigUploadClient.CheckRedirect,
	}
}

func newTestConfigClient(t *testing.T) *Client {
	t.Helper()
	client, err := NewClient(&Config{Token: "not-a-real-token"})
	require.NoError(t, err)
	return client
}

func testPackedConfig(t *testing.T) *PackedConfig {
	t.Helper()
	dir := t.TempDir()
	writeConfigFixture(t, dir, "main.tf", "test config")
	pack, err := PackConfig(t.Context(), dir)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, pack.Close()) })
	return pack
}

func configUploadVersion(target string) *models.ConfigurationVersions {
	version := models.NewConfigurationVersions()
	attrs := models.NewConfigurationVersions_attributes()
	attrs.SetUploadUrl(&target)
	version.SetAttributes(attrs)
	return version
}

type configRoundTripper func(*http.Request) (*http.Response, error)

func (f configRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}
