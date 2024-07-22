// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package tfe

import (
	"context"
	"fmt"
	"io"
	"net/url"
)

// Stacks describes all the stacks-related methods that the HCP Terraform API supports.
// NOTE WELL: This is a beta feature and is subject to change until noted otherwise in the
// release notes.
type StackUncontrolledConfigSources interface {
	// Read returns a uncontrolled configuration source by its ID.
	Read(ctx context.Context, stackUncontrolledConfigSourceId string) (*StacksUncontrolledConfigSource, error)

	// CreateUncontrolled creates a new uncontrolled source
	Create(ctx context.Context, stackID string) (*StacksUncontrolledConfigSource, error)

	// Upload packages and uploads Terraform configuration files. It requires
	// the upload URL from an uncontrolled configuration and the full path to the
	// configuration files on disk.
	Upload(ctx context.Context, url string, path string) error

	// Upload a tar gzip archive to the specified configuration version upload URL.
	UploadTarGzip(ctx context.Context, url string, archive io.Reader) error
}

type stackUncontrolledConfigSources struct {
	client *Client
}

var _ StackUncontrolledConfigSources = &stackUncontrolledConfigSources{}

type CreateOptions struct {
	StackID string `jsonapi:"attr,stack-id"`
}

type StacksUncontrolledConfigSource struct {
	ID                 string              `jsonapi:"primary,stack-uncontrolled-configuration-sources"`
	SourceAddress      string              `jsonapi:"attr,source-address"`
	UploadUrl          string              `jsonapi:"attr,upload-url"`
	Stack              *Stack              `jsonapi:"relation,stack"`
	StackConfiguration *StackConfiguration `jsonapi:"relation,stack-configuration"`
}

type StackConfiguration struct {
	ID                        string `jsonapi:"primary,stack-configurations"`
	Status                    string `jsonapi:"attr,status"`
	SequenceNumber            int    `jsonapi:"attr,sequence-number"`
	StackConfigSourceAddress  string `jsonapi:"attr,stack-config-source-address"`
	TerraformCliVerion        string `jsonapi:"attr,terraform-cli-version"`
	TerraformCliConfigVersion string `jsonapi:"attr,terraform-cli-config-version"`
}

func (s stackUncontrolledConfigSources) Read(ctx context.Context, stackUncontrolledConfigSourceId string) (*StacksUncontrolledConfigSource, error) {
	req, err := s.client.NewRequest("GET", fmt.Sprintf("stack-uncontrolled-config-sources/%s", url.PathEscape(stackUncontrolledConfigSourceId)), nil)
	if err != nil {
		return nil, err
	}

	ucs := &StacksUncontrolledConfigSource{}
	err = req.Do(ctx, ucs)
	if err != nil {
		return nil, err
	}

	return ucs, nil
}

func (s stackUncontrolledConfigSources) Create(ctx context.Context, stackID string) (*StacksUncontrolledConfigSource, error) {
	req, err := s.client.NewRequest("POST", fmt.Sprintf("stacks/%s/uncontrolled-config-sources", url.PathEscape(stackID)), nil)
	if err != nil {
		return nil, err
	}

	ucs := &StacksUncontrolledConfigSource{}
	err = req.Do(ctx, ucs)
	if err != nil {
		return nil, err
	}

	return ucs, nil
}

// Upload packages and uploads Terraform configuration files. It requires the
// upload URL from a configuration version and the path to the configuration
// files on disk.
func (s stackUncontrolledConfigSources) Upload(ctx context.Context, uploadURL, path string) error {
	body, err := packContents(path)
	if err != nil {
		return err
	}

	return s.UploadTarGzip(ctx, uploadURL, body)
}

// UploadTarGzip is used to upload Terraform configuration files contained a tar gzip archive.
// Any stream implementing io.Reader can be passed into this method. This method is also
// particularly useful for tar streams created by non-default go-slug configurations.
//
// **Note**: This method does not validate the content being uploaded and is therefore the caller's
// responsibility to ensure the raw content is a valid Terraform configuration.
func (s stackUncontrolledConfigSources) UploadTarGzip(ctx context.Context, uploadURL string, archive io.Reader) error {
	return s.client.doForeignPUTRequest(ctx, uploadURL, archive)
}
