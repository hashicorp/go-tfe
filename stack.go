// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package tfe

import (
	"context"
	"fmt"
	"net/url"
	"time"
)

// Stacks describes all the stacks-related methods that the HCP Terraform API supports.
// NOTE WELL: This is a beta feature and is subject to change until noted otherwise in the
// release notes.
type Stacks interface {
	// List returns a list of stacks, optionally filtered by project.
	List(ctx context.Context, organization string, options *StackListOptions) (*StackList, error)

	// Read returns a stack by its ID.
	Read(ctx context.Context, stackID string) (*Stack, error)

	// Create creates a new stack.
	Create(ctx context.Context, options StackCreateOptions) (*Stack, error)

	// Update updates a stack.
	Update(ctx context.Context, stackID string, options StackUpdateOptions) (*Stack, error)

	// Delete deletes a stack.
	Delete(ctx context.Context, stackID string) error

	// ForceDelete deletes a stack.
	ForceDelete(ctx context.Context, stackID string) error

	// FetchConfiguration updates the configuration of a stack, triggering stack preparation.
	FetchConfiguration(ctx context.Context, stackID string) (*Stack, error)
}

// stacks implements Stacks.
type stacks struct {
	client *Client
}

var _ Stacks = &stacks{}

// StackSortColumn represents a string that can be used to sort items when using
// the List method.
type StackSortColumn string

const (
	// StackSortByName sorts by the name attribute.
	StackSortByName StackSortColumn = "name"

	// StackSortByUpdatedAt sorts by the updated-at attribute.
	StackSortByUpdatedAt StackSortColumn = "updated-at"

	// StackSortByNameDesc sorts by the name attribute in descending order.
	StackSortByNameDesc StackSortColumn = "-name"

	// StackSortByUpdatedAtDesc sorts by the updated-at attribute in descending order.
	StackSortByUpdatedAtDesc StackSortColumn = "-updated-at"
)

// StackList represents a list of stacks.
type StackList struct {
	*Pagination
	Items []*Stack
}

// StackVCSRepo represents the version control system repository for a stack.
type StackVCSRepo struct {
	Identifier        string `jsonapi:"attr,identifier"`
	Branch            string `jsonapi:"attr,branch,omitempty"`
	GHAInstallationID string `jsonapi:"attr,github-app-installation-id,omitempty"`
	OAuthTokenID      string `jsonapi:"attr,oauth-token-id,omitempty"`
}

// StackVCSRepoOptions
type StackVCSRepoOptions struct {
	Identifier        string `json:"identifier"`
	Branch            string `json:"branch,omitempty"`
	GHAInstallationID string `json:"github-app-installation-id,omitempty"`
	OAuthTokenID      string `json:"oauth-token-id,omitempty"`
}

type LinkedStackConnections struct {
	UpstreamCount   int `jsonapi:"attr,upstream-count"`
	DownstreamCount int `jsonapi:"attr,downstream-count"`
	InputsCount     int `jsonapi:"attr,inputs-count"`
	OutputsCount    int `jsonapi:"attr,outputs-count"`
}

// Stack represents a stack.
type Stack struct {
	ID                     string                  `jsonapi:"primary,stacks"`
	Name                   string                  `jsonapi:"attr,name"`
	Description            string                  `jsonapi:"attr,description"`
	VCSRepo                *StackVCSRepo           `jsonapi:"attr,vcs-repo"`
	SpeculativeEnabled     bool                    `jsonapi:"attr,speculative-enabled"`
	CreatedAt              time.Time               `jsonapi:"attr,created-at,iso8601"`
	UpdatedAt              time.Time               `jsonapi:"attr,updated-at,iso8601"`
	LinkedStackConnections *LinkedStackConnections `jsonapi:"attr,linked-stack-connections"`

	// Relationships
	Project                  *Project            `jsonapi:"relation,project"`
	AgentPool                *AgentPool          `jsonapi:"relation,agent-pool"`
	LatestStackConfiguration *StackConfiguration `jsonapi:"relation,latest-stack-configuration"`
}

// StackConfigurationStatusTimestamps represents the timestamps for a stack configuration
type StackConfigurationStatusTimestamps struct {
	QueuedAt     *time.Time `jsonapi:"attr,queued-at,omitempty,rfc3339"`
	CompletedAt  *time.Time `jsonapi:"attr,completed-at,omitempty,rfc3339"`
	PreparingAt  *time.Time `jsonapi:"attr,preparing-at,omitempty,rfc3339"`
	EnqueueingAt *time.Time `jsonapi:"attr,enqueueing-at,omitempty,rfc3339"`
	CanceledAt   *time.Time `jsonapi:"attr,canceled-at,omitempty,rfc3339"`
	ErroredAt    *time.Time `jsonapi:"attr,errored-at,omitempty,rfc3339"`
}

// StackComponent represents a stack component, specified by configuration
type StackComponent struct {
	Name       string `json:"name"`
	Correlator string `json:"correlator"`
	Expanded   bool   `json:"expanded"`
	Removed    bool   `json:"removed"`
}

// StackConfiguration represents a stack configuration snapshot
type StackConfiguration struct {
	// Attributes
	ID                      string            `jsonapi:"primary,stack-configurations"`
	Status                  string            `jsonapi:"attr,status"`
	SequenceNumber          int               `jsonapi:"attr,sequence-number"`
	Components              []*StackComponent `jsonapi:"attr,components"`
	PreparingEventStreamURL string            `jsonapi:"attr,preparing-event-stream-url"`
	CreatedAt               time.Time         `jsonapi:"attr,created-at,iso8601"`
	UpdatedAt               time.Time         `jsonapi:"attr,updated-at,iso8601"`
	Speculative             bool              `jsonapi:"attr,speculative"`

	// Relationships
	Stack             *Stack             `jsonapi:"relation,stack"`
	IngressAttributes *IngressAttributes `jsonapi:"relation,ingress-attributes"`
}

// StackState represents a stack state
type StackState struct {
	// Attributes
	ID                    string `jsonapi:"primary,stack-states"`
	Description           string `jsonapi:"attr,description"`
	Generation            int    `jsonapi:"attr,generation"`
	Status                string `jsonapi:"attr,status"`
	Deployment            string `jsonapi:"attr,deployment"`
	Components            string `jsonapi:"attr,components"`
	IsCurrent             bool   `jsonapi:"attr,is-current"`
	ResourceInstanceCount int    `jsonapi:"attr,resource-instance-count"`

	// Relationships
	Stack              *Stack              `jsonapi:"relation,stack"`
	StackDeploymentRun *StackDeploymentRun `jsonapi:"relation,stack-deployment-run"`
}

// StackListOptions represents the options for listing stacks.
type StackListOptions struct {
	ListOptions
	ProjectID    string          `url:"filter[project[id]],omitempty"`
	Sort         StackSortColumn `url:"sort,omitempty"`
	SearchByName string          `url:"search[name],omitempty"`
}

// StackCreateOptions represents the options for creating a stack. The project
// relation is required.
type StackCreateOptions struct {
	Type        string               `jsonapi:"primary,stacks"`
	Name        string               `jsonapi:"attr,name"`
	Description *string              `jsonapi:"attr,description,omitempty"`
	VCSRepo     *StackVCSRepoOptions `jsonapi:"attr,vcs-repo"`
	Project     *Project             `jsonapi:"relation,project"`
	AgentPool   *AgentPool           `jsonapi:"relation,agent-pool"`
}

// StackUpdateOptions represents the options for updating a stack.
type StackUpdateOptions struct {
	Name        *string              `jsonapi:"attr,name,omitempty"`
	Description *string              `jsonapi:"attr,description,omitempty"`
	VCSRepo     *StackVCSRepoOptions `jsonapi:"attr,vcs-repo"`
	AgentPool   *AgentPool           `jsonapi:"relation,agent-pool"`
}

// WaitForStatusResult is the data structure that is sent over the channel
// returned by various status polling functions. For each result, either the
// Error or the Status will be set, but not both. If the Quit field is set,
// the channel will be closed. If the Quit field is set and the Error is
// nil, the Status field will be set to a specified quit status.
type WaitForStatusResult struct {
	ID           string
	Status       string
	ReadAttempts int
	Error        error
	Quit         bool
}

const minimumPollingIntervalMs = 3000
const maximumPollingIntervalMs = 5000

// FetchConfiguration fetches the latest configuration of a stack from VCS, triggering stack operations
func (s *stacks) FetchConfiguration(ctx context.Context, stackID string) (*Stack, error) {
	req, err := s.client.NewRequest("POST", fmt.Sprintf("stacks/%s/fetch-latest-from-vcs", url.PathEscape(stackID)), nil)
	if err != nil {
		return nil, err
	}

	stack := &Stack{}
	err = req.Do(ctx, stack)
	if err != nil {
		return nil, err
	}

	return stack, nil
}

// List returns a list of stacks, optionally filtered by additional paameters.
func (s stacks) List(ctx context.Context, organization string, options *StackListOptions) (*StackList, error) {
	if err := options.valid(); err != nil {
		return nil, err
	}

	req, err := s.client.NewRequest("GET", fmt.Sprintf("organizations/%s/stacks", organization), options)
	if err != nil {
		return nil, err
	}

	sl := &StackList{}
	err = req.Do(ctx, sl)
	if err != nil {
		return nil, err
	}

	return sl, nil
}

// Read returns a stack by its ID.
func (s stacks) Read(ctx context.Context, stackID string) (*Stack, error) {
	req, err := s.client.NewRequest("GET", fmt.Sprintf("stacks/%s", url.PathEscape(stackID)), nil)
	if err != nil {
		return nil, err
	}

	stack := &Stack{}
	err = req.Do(ctx, stack)
	if err != nil {
		return nil, err
	}

	return stack, nil
}

// Create creates a new stack.
func (s stacks) Create(ctx context.Context, options StackCreateOptions) (*Stack, error) {
	if err := options.valid(); err != nil {
		return nil, err
	}

	req, err := s.client.NewRequest("POST", "stacks", &options)
	if err != nil {
		return nil, err
	}

	stack := &Stack{}
	err = req.Do(ctx, stack)
	if err != nil {
		return nil, err
	}

	return stack, nil
}

// Update updates a stack.
func (s stacks) Update(ctx context.Context, stackID string, options StackUpdateOptions) (*Stack, error) {
	req, err := s.client.NewRequest("PATCH", fmt.Sprintf("stacks/%s", url.PathEscape(stackID)), &options)
	if err != nil {
		return nil, err
	}

	stack := &Stack{}
	err = req.Do(ctx, stack)
	if err != nil {
		return nil, err
	}

	return stack, nil
}

// Delete deletes a stack.
func (s stacks) Delete(ctx context.Context, stackID string) error {
	req, err := s.client.NewRequest("DELETE", fmt.Sprintf("stacks/%s", url.PathEscape(stackID)), nil)
	if err != nil {
		return err
	}

	return req.Do(ctx, nil)
}

// ForceDelete deletes a stack that still has deployments.
func (s stacks) ForceDelete(ctx context.Context, stackID string) error {
	req, err := s.client.NewRequest("DELETE", fmt.Sprintf("stacks/%s?force=true", url.PathEscape(stackID)), nil)
	if err != nil {
		return err
	}

	return req.Do(ctx, nil)
}

func (s *StackListOptions) valid() error {
	return nil
}

func (s StackCreateOptions) valid() error {
	if s.Name == "" {
		return ErrRequiredName
	}

	if s.Project.ID == "" {
		return ErrRequiredProject
	}

	return nil
}

// awaitPoll is a helper function that uses a callback to read a status, then
// waits for a terminal status or an error. The callback should return the
// current status, or an error. For each time the status changes, the channel
// emits a new result. The id parameter should be the ID of the resource being
// polled, which is used in the result to help identify the resource being polled.
func awaitPoll(ctx context.Context, id string, reader func(ctx context.Context) (string, error), quitStatus []string) <-chan WaitForStatusResult {
	resultCh := make(chan WaitForStatusResult)

	mapStatus := make(map[string]struct{}, len(quitStatus))
	for _, status := range quitStatus {
		mapStatus[status] = struct{}{}
	}

	go func() {
		defer close(resultCh)

		reads := 0
		lastStatus := ""
		for {
			select {
			case <-ctx.Done():
				resultCh <- WaitForStatusResult{ID: id, Error: fmt.Errorf("context canceled: %w", ctx.Err())}
				return
			case <-time.After(backoff(minimumPollingIntervalMs, maximumPollingIntervalMs, reads)):
				status, err := reader(ctx)
				if err != nil {
					resultCh <- WaitForStatusResult{ID: id, Error: err, Quit: true}
					return
				}

				_, terminal := mapStatus[status]

				if status != lastStatus {
					resultCh <- WaitForStatusResult{
						ID:           id,
						Status:       status,
						ReadAttempts: reads + 1,
						Quit:         terminal,
					}
				}

				lastStatus = status

				if terminal {
					return
				}

				reads += 1
			}
		}
	}()

	return resultCh
}
