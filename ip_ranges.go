// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package tfe

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"time"
)

// Compile-time proof of interface implementation.
var _ IPRanges = (*ipRanges)(nil)

// IP Ranges provides a list of HCP Terraform or Terraform Enterprise's IP ranges.
//
// TFE API docs: https://developer.hashicorp.com/terraform/cloud-docs/api-docs/ip-ranges
type IPRanges interface {
	// Retrieve HCP Terraform IP ranges. If `modifiedSince` is not nil
	// then it will only return the IP ranges changes since that date.
	// The format for `modifiedSince` can be found here:
	// https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/If-Modified-Since
	Read(ctx context.Context, modifiedSince *time.Time) (*IPRange, error)
}

// ipRanges implements IPRanges interface.
type ipRanges struct {
	client *Client
}

// IPRange represents a list of HCP Terraform's IP ranges
type IPRange struct {
	// List of IP ranges in CIDR notation used for connections from user site to HCP Terraform APIs
	API []string `json:"api,omitempty"`
	// List of IP ranges in CIDR notation used for notifications
	Notifications []string `json:"notifications,omitempty"`
	// List of IP ranges in CIDR notation used for outbound requests from Sentinel policies
	Sentinel []string `json:"sentinel,omitempty"`
	// List of IP ranges in CIDR notation used for connecting to VCS providers
	VCS []string `json:"vcs,omitempty"`
}

// Read an IPRange that was not modified since the specified date.
func (i *ipRanges) Read(ctx context.Context, modifiedSince *time.Time) (*IPRange, error) {

	reqHeaders := http.Header{}
	reqHeaders.Add("Accept", "application/json, */*")

	if modifiedSince != nil {
		reqHeaders.Add("If-Modified-Since", modifiedSince.Format(http.TimeFormat))
	}

	resp, err := i.client.GetStream(ctx, "/api/meta/ip-ranges", reqHeaders)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode == http.StatusNotModified {
		return nil, nil
	}

	ipRanges, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var ir IPRange
	err = json.Unmarshal(ipRanges, &ir)
	if err != nil {
		return nil, err
	}

	return &ir, nil
}
