package tfe

import (
	"context"
)

// Compile-time proof of interface implementation.
var _ AdminRuns = (*adminRuns)(nil)

// AdminRuns describes all the admin run related methods that the Terraform Enterprise
// API supports.
//
// TFE API docs:
// https://www.terraform.io/docs/enterprise/api/admin/runs.html
type AdminRuns interface {
	// List all runs in the Terraform Enterprise installation.
	List(ctx context.Context, options AdminRunListOptions) (*RunList, error)
}

type adminRuns struct {
	client *Client
}

//AdminRunListOptions represents the options for listing runs.
type AdminRunListOptions struct {
	ListOptions
	Q      *string   `url:"q,omitempty"`
	Status RunStatus `url:"filter[status],omitempty"`
}

// List all runs in the Terraform Enterprise installation.
func (s *adminRuns) List(ctx context.Context, options AdminRunListOptions) (*RunList, error) {

	u := "admin/runs"

	req, err := s.client.newRequest("GET", u, &options)
	if err != nil {
		return nil, err
	}

	rl := &RunList{}
	err = s.client.do(ctx, req, rl)
	if err != nil {
		return nil, err
	}

	return rl, nil
}
