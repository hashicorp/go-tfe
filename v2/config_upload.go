// Copyright IBM Corp. 2018, 2026
// SPDX-License-Identifier: MPL-2.0

package tfe

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/hashicorp/go-tfe/v2/api/models"
)

const (
	configUploadAttempts = 3
	configUploadTimeout  = 10 * time.Minute
	configUploadDrain    = 8 << 10
)

// Keep credential-bearing uploads out of the API middleware and off transports
// that callers can configure or replace.
var defaultConfigUploadClient = &http.Client{
	Transport: &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: time.Second,
	},
	CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
}

// UploadConfig sends config to the upload URL returned when the configuration
// version was created. It does not create the version or wait for it to be ready.
// The caller is responsible for closing config.
//
// UploadConfig requires HTTPS and does not follow redirects. It uses system
// TLS trust and environment proxies, not [Config.HTTPTransport] or API headers.
// The upload URL is a credential; do not log the response that contains it.
func (c *Client) UploadConfig(ctx context.Context, configVersion models.ConfigurationVersionsable, config *PackedConfig) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if c == nil || c.configUploadClient == nil {
		return errors.New("upload config: client is uninitialized")
	}
	data, err := config.archiveBytes()
	if err != nil {
		return err
	}
	target, err := configUploadTarget(configVersion)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(ctx, configUploadTimeout)
	defer cancel()
	for attempt := 1; ; attempt++ {
		retry, err := c.putConfig(ctx, target, data)
		if !retry || attempt == configUploadAttempts {
			return err
		}
		timer := time.NewTimer(time.Duration(attempt) * 200 * time.Millisecond)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
	}
}

func configUploadTarget(version models.ConfigurationVersionsable) (string, error) {
	missing := errors.New("upload config: configuration version has no upload URL")
	if model, ok := version.(*models.ConfigurationVersions); version == nil || (ok && model == nil) {
		return "", missing
	}
	attrs := version.GetAttributes()
	if model, ok := attrs.(*models.ConfigurationVersions_attributes); attrs == nil || (ok && model == nil) {
		return "", missing
	}
	raw := attrs.GetUploadUrl()
	if raw == nil || *raw == "" {
		return "", missing
	}
	u, err := url.Parse(*raw)
	if err != nil || u.Scheme != "https" || u.Opaque != "" || u.Hostname() == "" || u.User != nil || strings.Contains(*raw, "#") {
		return "", errors.New("upload config: invalid HTTPS upload URL")
	}
	return *raw, nil
}

func (c *Client) putConfig(ctx context.Context, target string, data []byte) (bool, error) {
	// A bytes.Reader lets net/http infer the content length.
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, target, bytes.NewReader(data))
	if err != nil {
		return false, errors.New("upload config: cannot create request")
	}
	req.Header.Set("Content-Type", "application/octet-stream")
	resp, err := c.configUploadClient.Do(req)
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return false, ctxErr
		}
		// Do not return or wrap transport errors, which can contain the URL.
		return true, errors.New("upload config: network request failed")
	}
	// Limit the drain for connection reuse; never interpret server response data.
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, configUploadDrain)) //nolint:errcheck
	_ = resp.Body.Close()                                                    //nolint:errcheck
	if err := ctx.Err(); err != nil {
		return false, err
	}
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return false, nil
	}
	if resp.StatusCode >= 300 && resp.StatusCode < 400 {
		return false, fmt.Errorf("upload config: redirect refused (HTTP %d)", resp.StatusCode)
	}
	statusErr := fmt.Errorf("upload config: server returned HTTP %d", resp.StatusCode)
	switch resp.StatusCode {
	case http.StatusInternalServerError, http.StatusBadGateway, http.StatusServiceUnavailable, http.StatusGatewayTimeout:
		return true, statusErr
	default:
		return false, statusErr
	}
}
