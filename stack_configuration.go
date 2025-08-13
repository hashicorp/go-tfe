// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package tfe

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/url"
	"time"
)

// StackConfigurations describes all the stacks configurations-related methods that the
// HCP Terraform API supports.
// NOTE WELL: This is a beta feature and is subject to change until noted otherwise in the
// release notes.
type StackConfigurations interface {
	// CreateAndUpload packages and uploads the specified Terraform Stacks
	// configuration files in association with a Stack.
	CreateAndUpload(ctx context.Context, stackID string, path string, opts *CreateStackConfigurationOptions) (*StackConfiguration, error)

	// ReadConfiguration returns a stack configuration by its ID.
	Read(ctx context.Context, id string) (*StackConfiguration, error)

	// ListStackConfigurations returns a list of stack configurations for a stack.
	List(ctx context.Context, stackID string, options *StackConfigurationListOptions) (*StackConfigurationList, error)

	// JSONSchemas returns a byte slice of the JSON schema for the stack configuration.
	JSONSchemas(ctx context.Context, stackConfigurationID string) ([]byte, error)

	// AwaitCompleted generates a channel that will receive the status of the
	// stack configuration as it progresses, until that status is "converged",
	// "converging", "errored", "canceled".
	AwaitCompleted(ctx context.Context, stackConfigurationID string) <-chan WaitForStatusResult

	// AwaitPrepared generates a channel that will receive the status of the
	// stack configuration as it progresses, until that status is "<status>",
	// "errored", "canceled".
	AwaitStatus(ctx context.Context, stackConfigurationID string, status StackConfigurationStatus) <-chan WaitForStatusResult
}

type StackConfigurationStatus string

const (
	StackConfigurationStatusPending    StackConfigurationStatus = "pending"
	StackConfigurationStatusQueued     StackConfigurationStatus = "queued"
	StackConfigurationStatusPreparing  StackConfigurationStatus = "preparing"
	StackConfigurationStatusEnqueueing StackConfigurationStatus = "enqueueing"
	StackConfigurationStatusConverged  StackConfigurationStatus = "converged"
	StackConfigurationStatusConverging StackConfigurationStatus = "converging"
	StackConfigurationStatusErrored    StackConfigurationStatus = "errored"
	StackConfigurationStatusCanceled   StackConfigurationStatus = "canceled"
	StackConfigurationStatusCompleted  StackConfigurationStatus = "completed"
)

func (s StackConfigurationStatus) String() string {
	return string(s)
}

type stackConfigurations struct {
	client *Client
}

var _ StackConfigurations = &stackConfigurations{}

func (s stackConfigurations) Read(ctx context.Context, id string) (*StackConfiguration, error) {
	req, err := s.client.NewRequest("GET", fmt.Sprintf("stack-configurations/%s", url.PathEscape(id)), nil)
	if err != nil {
		return nil, err
	}

	stackConfiguration := &StackConfiguration{}
	err = req.Do(ctx, stackConfiguration)
	if err != nil {
		return nil, err
	}

	return stackConfiguration, nil
}

/**
* Returns the JSON schema for the stack configuration as a byte slice.
* The return value needs to be unmarshalled into a struct to be useful.
* It is meant to be unmarshalled with terraform/internal/command/jsonproivder.Providers.
 */
func (s stackConfigurations) JSONSchemas(ctx context.Context, stackConfigurationID string) ([]byte, error) {
	req, err := s.client.NewRequest("GET", fmt.Sprintf("stack-configurations/%s/json-schemas", url.PathEscape(stackConfigurationID)), nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Accept", "application/json")

	var raw bytes.Buffer
	err = req.Do(ctx, &raw)
	if err != nil {
		return nil, err
	}

	return raw.Bytes(), nil
}

// AwaitCompleted generates a channel that will receive the status of the stack configuration as it progresses.
// The channel will be closed when the stack configuration reaches a status indicating that or an error occurs. The
// read will be retried dependending on the configuration of the client. When the channel is closed,
// the last value will either be a completed status or an error.
func (s stackConfigurations) AwaitCompleted(ctx context.Context, stackConfigurationID string) <-chan WaitForStatusResult {
	return awaitPoll(ctx, stackConfigurationID, func(ctx context.Context) (string, error) {
		stackConfiguration, err := s.Read(ctx, stackConfigurationID)
		if err != nil {
			return "", err
		}

		return stackConfiguration.Status, nil
	}, []string{StackConfigurationStatusConverged.String(), StackConfigurationStatusConverging.String(), StackConfigurationStatusCompleted.String(), StackConfigurationStatusErrored.String(), StackConfigurationStatusCanceled.String()})
}

// AwaitStatus generates a channel that will receive the status of the stack configuration as it progresses.
// The channel will be closed when the stack configuration reaches a status indicating that or an error occurs. The
// read will be retried dependending on the configuration of the client. When the channel is closed,
// the last value will either be the specified status, "errored" status, or "canceled" status, or an error.
func (s stackConfigurations) AwaitStatus(ctx context.Context, stackConfigurationID string, status StackConfigurationStatus) <-chan WaitForStatusResult {
	return awaitPoll(ctx, stackConfigurationID, func(ctx context.Context) (string, error) {
		stackConfiguration, err := s.Read(ctx, stackConfigurationID)
		if err != nil {
			return "", err
		}

		return stackConfiguration.Status, nil
	}, []string{status.String(), StackConfigurationStatusErrored.String(), StackConfigurationStatusCanceled.String()})
}

// StackConfigurationList represents a paginated list of stack configurations.
type StackConfigurationList struct {
	Pagination *Pagination
	Items      []*StackConfiguration
}

// StackConfigurationListOptions represents the options for listing stack configurations.
type StackConfigurationListOptions struct {
	ListOptions
}

func (s stackConfigurations) List(ctx context.Context, stackID string, options *StackConfigurationListOptions) (*StackConfigurationList, error) {
	if options == nil {
		options = &StackConfigurationListOptions{}
	}

	req, err := s.client.NewRequest("GET", fmt.Sprintf("stacks/%s/stack-configurations", url.PathEscape(stackID)), options)
	if err != nil {
		return nil, err
	}

	result := &StackConfigurationList{}
	err = req.Do(ctx, result)
	if err != nil {
		return nil, err
	}

	return result, nil
}

type CreateStackConfigurationOptions struct {
	SelectedDeployments []string `jsonapi:"attr,selected-deployments,omitempty"`
	SpeculativeEnabled  *bool    `jsonapi:"attr,speculative,omitempty"`
}

// CreateAndUpload packages and uploads the specified Terraform Stacks
// configuration files in association with a Stack.
func (s stackConfigurations) CreateAndUpload(ctx context.Context, stackID, path string, opts *CreateStackConfigurationOptions) (*StackConfiguration, error) {
	if opts == nil {
		opts = &CreateStackConfigurationOptions{}
	}
	u := fmt.Sprintf("stacks/%s/stack-configurations", url.PathEscape(stackID))
	req, err := s.client.NewRequest("POST", u, opts)
	if err != nil {
		return nil, fmt.Errorf("error creating stack configuration request for stack %q: %w", stackID, err)
	}

	sc := &StackConfiguration{}
	err = req.Do(ctx, sc)
	if err != nil {
		return nil, fmt.Errorf("error creating stack configuration for stack %q: %w", stackID, err)
	}

	uploadURL, err := s.pollForUploadURL(ctx, sc.ID)
	if err != nil {
		return nil, fmt.Errorf("error retrieving upload URL for stack configuration %q: %w", sc.ID, err)
	}

	body, err := packContents(path)
	if err != nil {
		return nil, err
	}

	err = s.UploadTarGzip(ctx, uploadURL, body)
	if err != nil {
		return nil, err
	}

	return sc, nil
}

// PollForUploadURL polls for the upload URL of a stack configuration until it becomes available.
// It makes a request every 2 seconds until the upload URL is present in the response.
// It will timeout after 10 seconds.
func (s stackConfigurations) pollForUploadURL(ctx context.Context, stackConfigurationID string) (string, error) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	timeout := time.NewTimer(15 * time.Second)
	defer timeout.Stop()

	for {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-timeout.C:
			return "", fmt.Errorf("timeout waiting for upload URL for stack configuration %q", stackConfigurationID)
		case <-ticker.C:
			urlReq, err := s.client.NewRequest("GET", fmt.Sprintf("stack-configurations/%s/upload-url", stackConfigurationID), nil)
			if err != nil {
				return "", fmt.Errorf("error creating upload URL request for stack configuration %q: %w", stackConfigurationID, err)
			}

			type UploadURLResponse struct {
				Data struct {
					SourceUploadURL *string `json:"source-upload-url"`
				} `json:"data"`
			}

			uploadResp := &UploadURLResponse{}
			err = urlReq.DoJSON(ctx, uploadResp)
			if err != nil {
				return "", fmt.Errorf("error getting upload URL for stack configuration %q: %w", stackConfigurationID, err)
			}

			if uploadResp.Data.SourceUploadURL != nil {
				return *uploadResp.Data.SourceUploadURL, nil
			}
		}
	}
}

// UploadTarGzip is used to upload Terraform configuration files contained a tar gzip archive.
// Any stream implementing io.Reader can be passed into this method. This method is also
// particularly useful for tar streams created by non-default go-slug configurations.
//
// **Note**: This method does not validate the content being uploaded and is therefore the caller's
// responsibility to ensure the raw content is a valid Terraform configuration.
func (s stackConfigurations) UploadTarGzip(ctx context.Context, uploadURL string, archive io.Reader) error {
	return s.client.doForeignPUTRequest(ctx, uploadURL, archive)
}
