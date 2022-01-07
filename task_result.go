package tfe

import (
	"context"
	"fmt"
	"time"
)

var _ TaskResults = (*taskResults)(nil)

type TaskResults interface {
	Read(ctx context.Context, taskResultID string) (*TaskResult, error)
}

type taskResults struct {
	client *Client
}

type TaskResultStatus string
type TaskEnforcementLevel string

const (
	TaskPassed      TaskResultStatus     = "passed"
	TaskFailed      TaskResultStatus     = "failed"
	TaskRunning     TaskResultStatus     = "running"
	TaskPending     TaskResultStatus     = "pending"
	TaskUnreachable TaskResultStatus     = "unreachable"
	Advisory        TaskEnforcementLevel = "advisory"
	Mandatory       TaskEnforcementLevel = "mandatory"
)

type TaskResult struct {
	ID                            string                  `jsonapi:"primary,task-results"`
	Status                        TaskResultStatus        `jsonapi:"attr,status"`
	Message                       string                  `jsonapi:"attr,message"`
	StatusTimestamps              RunTaskStatusTimestamps `jsonapi:"attr,status-timestamps"`
	URL                           string                  `jsonapi:"attr,url"`
	CreatedAt                     time.Time               `jsonapi:"attr,created-at,iso8601"`
	UpdatedAt                     time.Time               `jsonapi:"attr,updated-at,iso8601"`
	TaskID                        string                  `jsonapi:"attr,task-id"`
	TaskName                      string                  `jsonapi:"attr,task-name"`
	TaskURL                       string                  `jsonapi:"attr,task-url"`
	WorkspaceTaskID               string                  `jsonapi:"attr,workspace-task-id"`
	WorkspaceTaskEnforcementLevel TaskEnforcementLevel    `jsonapi:"attr,workspace-task-enforcement-level"`

	TaskStage *TaskStage `jsonapi:"relation,task_stage"`
}

func (t *taskResults) Read(ctx context.Context, taskResultID string) (*TaskResult, error) {
	if !validStringID(&taskResultID) {
		return nil, ErrInvalidTaskResultID
	}

	u := fmt.Sprintf("task-results/%s", taskResultID)
	req, err := t.client.newRequest("GET", u, nil)
	if err != nil {
		return nil, err
	}

	r := &TaskResult{}
	err = t.client.do(ctx, req, r)
	if err != nil {
		return nil, err
	}

	return r, nil
}
