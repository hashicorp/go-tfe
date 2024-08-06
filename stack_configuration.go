// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package tfe

import (
	"bytes"
	"context"
	"fmt"
	"net/url"
)

// StackConfigurations describes all the stacks configurations-related methods that the
// HCP Terraform API supports.
// NOTE WELL: This is a beta feature and is subject to change until noted otherwise in the
// release notes.
type StackConfigurations interface {
	// ReadConfiguration returns a stack configuration by its ID.
	Read(ctx context.Context, id string) (*StackConfiguration, error)

	// JSONSchemas returns a byte slice of the JSON schema for the stack configuration.
	JSONSchemas(ctx context.Context, stackConfigurationID string) ([]byte, error)

	// AwaitPrepared generates a channel that will receive the status of the
	// stack configuration as it progresses, until that status is "converged",
	// "converging", "errored", "canceled".
	AwaitPrepared(ctx context.Context, stackConfigurationID string) <-chan WaitForStatusResult

	// AwaitConverged generates a channel that will receive the status of the
	// stack configuration as it progresses, until that status is "converged",
	// "canceled", or "errored".
	AwaitConverged(ctx context.Context, stackConfigurationID string) <-chan WaitForStatusResult
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

// AwaitPrepared generates a channel that will receive the status of the stack configuration as it progresses.
// The channel will be closed when the stack configuration reaches a status indicating that or an error occurs. The
// read will be retried dependending on the configuration of the client. When the channel is closed,
// the last value will either be a terminal status or an error.
func (s stackConfigurations) AwaitPrepared(ctx context.Context, stackConfigurationID string) <-chan WaitForStatusResult {
	return awaitPoll(ctx, stackConfigurationID, func(ctx context.Context) (string, error) {
		stackConfiguration, err := s.Read(ctx, stackConfigurationID)
		if err != nil {
			return "", err
		}

		return stackConfiguration.Status, nil
	}, []string{"converged", "converging", "errored", "canceled"})
}

// AwaitConverged generates a channel that will receive the status of the stack configuration as it progresses.
// The channel will be closed when the stack configuration reaches a status indicating that or an error occurs. The
// read will be retried dependending on the configuration of the client. When the channel is closed,
// the last value will either be a terminal status or an error.
func (s stackConfigurations) AwaitConverged(ctx context.Context, stackConfigurationID string) <-chan WaitForStatusResult {
	return awaitPoll(ctx, stackConfigurationID, func(ctx context.Context) (string, error) {
		stackConfiguration, err := s.Read(ctx, stackConfigurationID)
		if err != nil {
			return "", err
		}

		return stackConfiguration.Status, nil
	}, []string{"converged", "errored", "canceled"})
}
