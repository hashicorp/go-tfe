package tfe

import (
	"context"
	"fmt"
	"net/url"
)

var _ WorkspaceRunTasks = (*workspaceRunTasks)(nil)

type WorkspaceRunTasks interface {
	Create(ctx context.Context, workspaceID string, options WorkspaceRunTaskCreateOptions) (*WorkspaceRunTask, error)

	List(ctx context.Context, workspaceID string, options *WorkspaceRunTaskListOptions) (*WorkspaceRunTaskList, error)

	Read(ctx context.Context, workspaceID string, workspaceTaskID string) (*WorkspaceRunTask, error)

	Update(ctx context.Context, workspaceID string, workspaceTaskID string, options WorkspaceRunTaskUpdateOptions) (*WorkspaceRunTask, error)

	Delete(ctx context.Context, workspaceID string, workspaceTaskID string) error
}

type workspaceRunTasks struct {
	client *Client
}

type WorkspaceRunTask struct {
	ID               string               `jsonapi:"primary,workspace-tasks"`
	EnforcementLevel TaskEnforcementLevel `jsonapi:"attr,enforcement-level"`

	RunTask   *RunTask   `jsonapi:"relation,task"`
	Workspace *Workspace `jsonapi:"relation,workspace"`
}

type WorkspaceRunTaskList struct {
	*Pagination
	Items []*WorkspaceRunTask
}

type WorkspaceRunTaskListOptions struct {
	ListOptions
}

func (s *workspaceRunTasks) List(ctx context.Context, workspaceID string, options *WorkspaceRunTaskListOptions) (*WorkspaceRunTaskList, error) {
	if !validStringID(&workspaceID) {
		return nil, ErrInvalidWorkspaceID
	}

	u := fmt.Sprintf("workspaces/%s/tasks", url.QueryEscape(workspaceID))
	req, err := s.client.newRequest("GET", u, options)
	if err != nil {
		return nil, err
	}

	rl := &WorkspaceRunTaskList{}
	err = s.client.do(ctx, req, rl)
	if err != nil {
		return nil, err
	}

	return rl, nil
}

func (s *workspaceRunTasks) Read(ctx context.Context, workspaceID string, workspaceTaskID string) (*WorkspaceRunTask, error) {
	if !validStringID(&workspaceID) {
		return nil, ErrInvalidWorkspaceID
	}

	if !validStringID(&workspaceTaskID) {
		return nil, ErrInvalidWorkspaceRunTaskID
	}

	u := fmt.Sprintf(
		"workspaces/%s/tasks/%s",
		url.QueryEscape(workspaceID),
		url.QueryEscape(workspaceTaskID),
	)
	req, err := s.client.newRequest("GET", u, nil)
	if err != nil {
		return nil, err
	}

	wr := &WorkspaceRunTask{}
	err = s.client.do(ctx, req, wr)
	if err != nil {
		return nil, err
	}

	return wr, nil
}

type WorkspaceRunTaskCreateOptions struct {
	Type             string               `jsonapi:"primary,workspace-tasks"`
	EnforcementLevel TaskEnforcementLevel `jsonapi:"attr,enforcement-level"`
	RunTask          *RunTask             `jsonapi:"relation,task"`
}

func (o *WorkspaceRunTaskCreateOptions) valid() error {
	if o.RunTask.ID == "" {
		return ErrInvalidRunTaskID
	}

	return nil
}

func (s *workspaceRunTasks) Create(ctx context.Context, workspaceID string, options WorkspaceRunTaskCreateOptions) (*WorkspaceRunTask, error) {
	if !validStringID(&workspaceID) {
		return nil, ErrInvalidWorkspaceID
	}

	if err := options.valid(); err != nil {
		return nil, err
	}

	u := fmt.Sprintf("workspaces/%s/tasks", workspaceID)
	req, err := s.client.newRequest("POST", u, &options)
	if err != nil {
		return nil, err
	}

	wr := &WorkspaceRunTask{}
	err = s.client.do(ctx, req, wr)
	if err != nil {
		return nil, err
	}

	return wr, nil
}

type WorkspaceRunTaskUpdateOptions struct {
	Type             string               `jsonapi:"primary,workspace-tasks"`
	EnforcementLevel TaskEnforcementLevel `jsonapi:"attr,enforcement-level,omitempty"`
}

func (s *workspaceRunTasks) Update(ctx context.Context, workspaceID string, workspaceTaskID string, options WorkspaceRunTaskUpdateOptions) (*WorkspaceRunTask, error) {
	if !validStringID(&workspaceID) {
		return nil, ErrInvalidWorkspaceID
	}

	if !validStringID(&workspaceTaskID) {
		return nil, ErrInvalidWorkspaceRunTaskID
	}

	u := fmt.Sprintf(
		"workspaces/%s/tasks/%s",
		url.QueryEscape(workspaceID),
		url.QueryEscape(workspaceTaskID),
	)
	req, err := s.client.newRequest("PATCH", u, &options)
	if err != nil {
		return nil, err
	}

	wr := &WorkspaceRunTask{}
	err = s.client.do(ctx, req, wr)
	if err != nil {
		return nil, err
	}

	return wr, nil
}

func (s *workspaceRunTasks) Delete(ctx context.Context, workspaceID string, workspaceTaskID string) error {
	if !validStringID(&workspaceID) {
		return ErrInvalidWorkspaceID
	}

	if !validStringID(&workspaceTaskID) {
		return ErrInvalidWorkspaceRunTaskType
	}

	u := fmt.Sprintf(
		"workspaces/%s/tasks/%s",
		url.QueryEscape(workspaceID),
		url.QueryEscape(workspaceTaskID),
	)
	req, err := s.client.newRequest("DELETE", u, nil)
	if err != nil {
		return err
	}

	return s.client.do(ctx, req, nil)
}
