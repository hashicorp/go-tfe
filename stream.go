// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package tfe

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"strings"

	abs "github.com/microsoft/kiota-abstractions-go"
)

// GetStream sends a GET request using the client's configured middleware and
// authentication, returning the raw *http.Response without buffering the body.
// The caller is responsible for closing the response body. This method is useful
// for API endpoints that return large responses, such as log files or streaming data.
//
// Usually, calling an endpoint using the API interface will return a deserialized struct
// or buffered []byte slice. However, for endpoints that return large responses, you can
// choose to use GetStream to get the raw *http.Response and stream the body directly without
// buffering it in memory.
func (c *Client) GetStream(ctx context.Context, uriOrPath string) (*http.Response, error) {
	req := abs.NewRequestInformation()
	u, err := url.Parse(uriOrPath)
	if err != nil {
		return nil, fmt.Errorf("failed to parse URL: %w", err)
	}

	if !strings.HasPrefix(u.Path, c.baseURL.Path) {
		u.Path = path.Join(c.baseURL.Path, u.Path)
	}

	if u.Host == "" || u.Scheme == "" {
		u.Host = c.baseURL.Host
		u.Scheme = c.baseURL.Scheme
	}

	req.SetUri(*u)

	nativeRequest, err := c.adapter.ConvertToNativeRequest(ctx, req)
	if err != nil {
		return nil, err
	}

	httpReq, ok := nativeRequest.(*http.Request)
	if !ok {
		return nil, fmt.Errorf("unexpected native request type %T", nativeRequest)
	}

	httpResp, err := c.adapter.Client.Do(httpReq)
	if err != nil {
		var urlErr *url.Error
		if errors.As(err, &urlErr) {
			return nil, urlErr.Err
		}
		return nil, err
	}

	return httpResp, nil
}
