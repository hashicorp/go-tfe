// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package tfe

import (
	"context"
	"io"
	"net/http"
	"time"
)

// Compile-time proof of interface implementation.
var _ OpenAPI = (*openAPI)(nil)

// OpenAPI provides access to the OpenAPI specification for HCP Terraform or Terraform Enterprise.
type OpenAPI interface {
	Read(ctx context.Context, prerelease bool, modifiedSince *time.Time) ([]byte, error)
}

// openAPI implements OpenAPI interface.
type openAPI struct {
	client *Client
}

// Read the OpenAPI specification that was not modified since the specified date. If prerelease is
// true, the public beta version of the OpenAPI specification will be returned.
func (i *openAPI) Read(ctx context.Context, prerelease bool, modifiedSince *time.Time) ([]byte, error) {
	reqHeaders := http.Header{}
	reqHeaders.Add("Accept", "application/json, */*")

	if modifiedSince != nil {
		reqHeaders.Add("If-Modified-Since", modifiedSince.Format(http.TimeFormat))
	}

	url := "/api/meta/openapi"
	if prerelease {
		url += "?version=prerelease"
	}

	resp, err := i.client.GetStream(ctx, url, reqHeaders)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode == http.StatusNotModified {
		return nil, nil
	}

	openAPIData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return openAPIData, nil
}
