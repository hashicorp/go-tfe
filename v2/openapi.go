// Copyright IBM Corp. 2018, 2026
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
	Read(ctx context.Context, prerelease bool, modifiedSince *time.Time) (*OpenAPIResponse, error)
}

// openAPI implements OpenAPI interface.
type openAPI struct {
	client *Client
}

// WithLastModified represents a response that contains a last modified time.
type WithLastModified struct {
	LastModified *time.Time
}

// OpenAPIResponse represents the response from reading the OpenAPI specification, including the
// raw bytes and the last modified time.
type OpenAPIResponse struct {
	WithLastModified
	Bytes []byte
}

// IsNotModified returns true if the response does not contain data or a last modified time.
func (o *WithLastModified) IsNotModified() bool {
	return o.LastModified == nil
}

// Read the OpenAPI specification that was not modified since the specified date. If prerelease is
// true, the public beta version of the OpenAPI specification will be returned.
func (i *openAPI) Read(ctx context.Context, prerelease bool, modifiedSince *time.Time) (*OpenAPIResponse, error) {
	reqHeaders := http.Header{}
	reqHeaders.Add("Accept", "application/json, */*")

	if modifiedSince != nil {
		reqHeaders.Add("If-Modified-Since", modifiedSince.UTC().Format(http.TimeFormat))
	}

	url := "/openapi"
	if prerelease {
		url += "/prerelease.json"
	} else {
		url += "/stable.json"
	}

	resp, err := i.client.GetStream(ctx, url, reqHeaders)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode == http.StatusNotModified {
		return &OpenAPIResponse{}, nil
	}

	openAPIData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close() //nolint:errcheck

	var lastModified *time.Time = nil
	if lm := resp.Header.Get("Last-Modified"); lm != "" {
		if t, err := time.Parse(http.TimeFormat, lm); err == nil {
			lastModified = &t
		}
	}

	return &OpenAPIResponse{
		Bytes:            openAPIData,
		WithLastModified: WithLastModified{LastModified: lastModified},
	}, nil
}
