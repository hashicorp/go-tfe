// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package tfe

import (
	"context"
	"fmt"

	abstractions "github.com/microsoft/kiota-abstractions-go"
	"github.com/microsoft/kiota-abstractions-go/serialization"
)

// Compile-time proof of interface implementation.
var _ IPRanges = (*ipRanges)(nil)

// IP Ranges provides a list of HCP Terraform or Terraform Enterprise's IP ranges.
//
// TFE API docs: https://developer.hashicorp.com/terraform/cloud-docs/api-docs/ip-ranges
type IPRanges interface {
	// Retrieve HCP Terraform IP ranges. If `modifiedSince` is not an empty string
	// then it will only return the IP ranges changes since that date.
	// The format for `modifiedSince` can be found here:
	// https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/If-Modified-Since
	Read(ctx context.Context, modifiedSince string) (*IPRange, error)
}

// ipRanges implements IPRanges interface.
type ipRanges struct {
	client *Client
}

// IPRange represents a list of HCP Terraform's IP ranges
type IPRange struct {
	// List of IP ranges in CIDR notation used for connections from user site to HCP Terraform APIs
	API []string
	// List of IP ranges in CIDR notation used for notifications
	Notifications []string
	// List of IP ranges in CIDR notation used for outbound requests from Sentinel policies
	Sentinel []string
	// List of IP ranges in CIDR notation used for connecting to VCS providers
	VCS []string
}

func deserializeStringArray(n serialization.ParseNode) ([]string, error) {
	val, err := n.GetCollectionOfPrimitiveValues("string")
	if err != nil {
		return nil, err
	}
	if val != nil {
		result := make([]string, len(val))
		for i, v := range val {
			result[i] = *v.(*string)
		}
		return result, nil
	}
	return nil, nil
}

func (m *IPRange) GetFieldDeserializers() map[string]func(serialization.ParseNode) error {
	return map[string]func(serialization.ParseNode) error{
		"api": func(n serialization.ParseNode) error {
			val, err := deserializeStringArray(n)
			if err != nil {
				return err
			}
			m.API = val
			return nil
		},
		"notifications": func(n serialization.ParseNode) error {
			val, err := deserializeStringArray(n)
			if err != nil {
				return err
			}
			m.Notifications = val
			return nil
		},
		"sentinel": func(n serialization.ParseNode) error {
			val, err := deserializeStringArray(n)
			if err != nil {
				return err
			}
			m.Sentinel = val
			return nil
		},
		"vcs": func(n serialization.ParseNode) error {
			val, err := deserializeStringArray(n)
			if err != nil {
				return err
			}
			m.VCS = val
			return nil
		},
	}
}

func firstError(todo ...(func() error)) error {
	for _, fn := range todo {
		if err := fn(); err != nil {
			return err
		}
	}
	return nil
}

func (m *IPRange) Serialize(writer serialization.SerializationWriter) error {
	return firstError([]func() error{
		func() error {
			return writer.WriteCollectionOfStringValues("api", m.API)
		},
		func() error {
			return writer.WriteCollectionOfStringValues("notifications", m.Notifications)
		},
		func() error {
			return writer.WriteCollectionOfStringValues("sentinel", m.Sentinel)
		},
		func() error {
			return writer.WriteCollectionOfStringValues("vcs", m.VCS)
		},
	}...)
}

func factory(parseNode serialization.ParseNode) (serialization.Parsable, error) {
	return &IPRange{}, nil
}

// Read an IPRange that was not modified since the specified date.
func (i *ipRanges) Read(ctx context.Context, modifiedSince string) (*IPRange, error) {
	reqHeaders := abstractions.NewRequestHeaders()
	reqHeaders.Add("Accept", "application/json, */*")
	reqHeaders.Add("Content-Type", "application/json")

	// Temporarily clear the base URL because this endpoint is not on that base.
	baseURL := i.client.baseURL
	i.client.adapter.SetBaseUrl("")
	defer i.client.adapter.SetBaseUrl(baseURL.String())

	req := abstractions.RequestInformation{
		Method:             abstractions.GET,
		UrlTemplate:        fmt.Sprintf("%s://%s/api/meta/ip-ranges", baseURL.Scheme, baseURL.Host),
		Headers:            reqHeaders,
		PathParameters:     make(map[string]string),
		QueryParameters:    make(map[string]string),
		QueryParametersAny: make(map[string]any),
	}

	if modifiedSince != "" {
		req.Headers.Add("If-Modified-Since", modifiedSince)
	}

	nullErrorFactory := abstractions.ErrorMappings{
		"XXX": func(parseNode serialization.ParseNode) (serialization.Parsable, error) {
			return nil, nil // Explicitly tell the parser to return nothing
		},
	}

	resp, err := i.client.adapter.Send(ctx, &req, factory, nullErrorFactory)
	if err != nil {
		return nil, err
	}
	ir := resp.(*IPRange)
	return ir, nil
}
