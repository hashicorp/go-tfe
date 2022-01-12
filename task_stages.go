package tfe

import (
	"context"
	"fmt"
	"time"
)

type TaskStages interface {
	Read(ctx context.Context, taskStageID string, options *TaskStageReadOptions) (*TaskStage, error)

	List(ctx context.Context, runID string, options *TaskStageListOptions) (*TaskStageList, error)
}

type taskStages struct {
	client *Client
}

type Stage string

const (
	PreApply Stage = "pre_apply"
	PostPlan Stage = "post_plan"
)

type TaskStage struct {
	ID               string                    `jsonapi:"primary,task-stages"`
	Stage            Stage                     `jsonapi:"attr,stage"`
	StatusTimestamps TaskStageStatusTimestamps `jsonapi:"attr,status-timestamps"`
	CreatedAt        time.Time                 `jsonapi:"attr,created-at,iso8601"`
	UpdatedAt        time.Time                 `jsonapi:"attr,updated-at,iso8601"`

	Run         *Run          `jsonapi:"relation,run"`
	TaskResults []*TaskResult `jsonapi:"relation,task-results"`
}

type TaskStageList struct {
	*Pagination
	Items []*TaskStage
}

type TaskStageStatusTimestamps struct {
	ErroredAt  time.Time `jsonapi:"attr,errored-at,rfc3339"`
	RunningAt  time.Time `jsonapi:"attr,running-at,rfc3339"`
	CanceledAt time.Time `jsonapi:"attr,canceled-at,rfc3339"`
	FailedAt   time.Time `jsonapi:"attr,failed-at,rfc3339"`
	PassedAt   time.Time `jsonapi:"attr,passed-at,rfc3339"`
}

type TaskStageReadOptions struct {
	Include string `url:"include"`
}

func (s *taskStages) Read(ctx context.Context, taskStageID string, options *TaskStageReadOptions) (*TaskStage, error) {
	if !validStringID(&taskStageID) {
		return nil, ErrInvalidTaskStageID
	}

	u := fmt.Sprintf("task-stages/%s", taskStageID)
	req, err := s.client.newRequest("GET", u, &options)
	if err != nil {
		return nil, err
	}

	t := &TaskStage{}
	err = s.client.do(ctx, req, t)
	if err != nil {
		return nil, err
	}

	return t, nil
}

type TaskStageListOptions struct {
	ListOptions
}

func (s *taskStages) List(ctx context.Context, runID string, options *TaskStageListOptions) (*TaskStageList, error) {
	if !validStringID(&runID) {
		return nil, ErrInvalidRunID
	}

	u := fmt.Sprintf("runs/%s/task-stages", runID)
	req, err := s.client.newRequest("GET", u, &options)
	if err != nil {
		return nil, err
	}

	tlist := &TaskStageList{}

	err = s.client.do(ctx, req, tlist)
	if err != nil {
		return nil, err
	}

	return tlist, nil
}
