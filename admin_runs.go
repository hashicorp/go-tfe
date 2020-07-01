package tfe

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"
)

// Compile-time proof of interface implementation.
var _ AdminRuns = (*adminRuns)(nil)

// AdminRuns Users Admin API contains endpoints to help site administrators manage
// user accounts.
//
// TFE API docs: https://www.terraform.io/docs/cloud/api/admin runs.html
type AdminRuns interface {
	// List all the runs of the given installation.
	List(ctx context.Context, options AdminRunsListOptions) (*AdminRunsList, error)

	// Force-cancel a run by its ID.
	ForceCancel(ctx context.Context, runID string, options RunForceCancelOptions) error
}

// runs implements Users.
type adminRuns struct {
	client *Client
}

type adminRun struct {
	ID                     string               `jsonapi:"primary,runs"`
	Actions                *RunActions          `jsonapi:"attr,actions"`
	CreatedAt              time.Time            `jsonapi:"attr,created-at,iso8601"`
	ForceCancelAvailableAt time.Time            `jsonapi:"attr,force-cancel-available-at,iso8601"`
	HasChanges             bool                 `jsonapi:"attr,has-changes"`
	IsDestroy              bool                 `jsonapi:"attr,is-destroy"`
	Message                string               `jsonapi:"attr,message"`
	Permissions            *RunPermissions      `jsonapi:"attr,permissions"`
	PositionInQueue        int                  `jsonapi:"attr,position-in-queue"`
	Source                 RunSource            `jsonapi:"attr,source"`
	Status                 RunStatus            `jsonapi:"attr,status"`
	StatusTimestamps       *RunStatusTimestamps `jsonapi:"attr,status-timestamps"`

	// Relations
	Workspace    *Workspace    `jsonapi:"relation,workspace"`
	Organization *Organization `jsonapi:"relation,workspace.organization"`
}

// AdminRunsList represents a list of runs.
type AdminRunsList struct {
	*Pagination
	Items []*Run
}

// AdminRunsListOptions represents the options for listing runs.
type AdminRunsListOptions struct {
	ListOptions
	RunStatus *string `url:"filter[status],omitempty"` // sending an empty param returns zero results
	Query     *string `url:"q"`
	Include   string  `url:"include"`
}

func (o AdminRunsListOptions) valid() error {
	if o.RunStatus != nil && *o.RunStatus != "" {
		validRunStatus := []string{"pending", "plan_queued", "planning", "planned", "confirmed", "apply_queued", "applying", "applied", "discarded", "errored", "canceled", "cost_estimating", "cost_estimated", "policy_checking", "policy_override", "policy_soft_failed", "policy_checked", "planned_and_finished"}
		runStatus := strings.Split(*o.RunStatus, ",")

		// iterate over our statuses
		for _, status := range runStatus {

			// start with invalid
			valid := false
			for _, s := range validRunStatus {
				if status == s {
					// found a match, set to true and continue to the next status
					valid = true
					break
				}
			}

			if valid == false {
				return fmt.Errorf("invalid value %s for run status", status)
			}
		}
	}
	return nil
}

// List all the runs of the terraform enterprise installation.
func (s *adminRuns) List(ctx context.Context, options AdminRunsListOptions) (*AdminRunsList, error) {
	if err := options.valid(); err != nil {
		return nil, err
	}

	u := fmt.Sprintf("admin/runs")
	req, err := s.client.newRequest("GET", u, &options)
	if err != nil {
		return nil, err
	}

	rl := &AdminRunsList{}
	err = s.client.do(ctx, req, rl)
	if err != nil {
		return nil, err
	}

	return rl, nil
}

// ForceCancel is used to forcefully cancel a run by its ID.
func (s *adminRuns) ForceCancel(ctx context.Context, runID string, options RunForceCancelOptions) error {
	if !validStringID(&runID) {
		return errors.New("invalid value for run ID")
	}

	u := fmt.Sprintf("admin/runs/%s/actions/force-cancel", url.QueryEscape(runID))
	req, err := s.client.newRequest("POST", u, &options)
	if err != nil {
		return err
	}

	return s.client.do(ctx, req, nil)
}
