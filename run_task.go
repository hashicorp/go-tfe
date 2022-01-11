package tfe

import (
	"context"
	"fmt"
	"net/url"
)

var _ RunTasks = (*runTasks)(nil)

type RunTasks interface {
	Create(ctx context.Context, organization string, options RunTaskCreateOptions) (*RunTask, error)

	List(ctx context.Context, organization string, options *RunTaskListOptions) (*RunTaskList, error)

	Read(ctx context.Context, runTaskID string) (*RunTask, error)

	ReadWithOptions(ctx context.Context, runTaskID string, options *RunTaskReadOptions) (*RunTask, error)

	Update(ctx context.Context, runTaskID string, options RunTaskUpdateOptions) (*RunTask, error)

	Delete(ctx context.Context, runTaskID string) error
}

type runTasks struct {
	client *Client
}

type RunTask struct {
	ID       string  `jsonapi:"primary,tasks"`
	Name     string  `jsonapi:"attr,name"`
	URL      string  `jsonapi:"attr,url"`
	Category string  `jsonapi:"attr,category"`
	HmacKey  *string `jsonapi:"attr,hmac-key,omitempty"`

	Organization      *Organization       `jsonapi:"relation,organization"`
	WorkspaceRunTasks []*WorkspaceRunTask `jsonapi:"relation,workspace-tasks"`
}

type RunTaskList struct {
	*Pagination
	Items []*RunTask
}

type RunTaskCreateOptions struct {
	Type     string  `jsonapi:"primary,tasks"`
	Name     string  `jsonapi:"attr,name"`
	URL      string  `jsonapi:"attr,url"`
	Category string  `jsonapi:"attr,category"`
	HmacKey  *string `jsonapi:"attr,hmac-key,omitempty"`
}

func (o *RunTaskCreateOptions) valid() error {
	if !validString(&o.Name) {
		return ErrRequiredName
	}

	if !validString(&o.URL) {
		return ErrInvalidRunTaskURL
	}

	if o.Category != "task" {
		return ErrInvalidRunTaskCategory
	}

	return nil
}

func (s *runTasks) Create(ctx context.Context, organization string, options RunTaskCreateOptions) (*RunTask, error) {
	if !validStringID(&organization) {
		return nil, ErrInvalidOrg
	}

	if err := options.valid(); err != nil {
		return nil, err
	}

	u := fmt.Sprintf("organizations/%s/tasks", url.QueryEscape(organization))
	req, err := s.client.newRequest("POST", u, &options)
	if err != nil {
		return nil, err
	}

	r := &RunTask{}
	err = s.client.do(ctx, req, r)
	if err != nil {
		return nil, err
	}

	return r, nil
}

type RunTaskListOptions struct {
	Include string `url:"include"`
	ListOptions
}

func (s *runTasks) List(ctx context.Context, organization string, options *RunTaskListOptions) (*RunTaskList, error) {
	if !validStringID(&organization) {
		return nil, ErrInvalidOrg
	}

	u := fmt.Sprintf("organizations/%s/tasks", url.QueryEscape(organization))
	req, err := s.client.newRequest("GET", u, &options)
	if err != nil {
		return nil, err
	}

	rl := &RunTaskList{}
	err = s.client.do(ctx, req, rl)
	if err != nil {
		return nil, err
	}

	return rl, nil
}

func (s *runTasks) Read(ctx context.Context, runTaskID string) (*RunTask, error) {
	return s.ReadWithOptions(ctx, runTaskID, nil)
}

type RunTaskReadOptions struct {
	Include string `url:"include"`
}

func (s *runTasks) ReadWithOptions(ctx context.Context, runTaskID string, options *RunTaskReadOptions) (*RunTask, error) {
	if !validStringID(&runTaskID) {
		return nil, ErrInvalidRunTaskID
	}

	u := fmt.Sprintf("tasks/%s", url.QueryEscape(runTaskID))
	req, err := s.client.newRequest("GET", u, options)
	if err != nil {
		return nil, err
	}

	r := &RunTask{}
	err = s.client.do(ctx, req, r)
	if err != nil {
		return nil, err
	}

	return r, nil
}

type RunTaskUpdateOptions struct {
	Type     string  `jsonapi:"primary,tasks"`
	Name     *string `jsonapi:"attr,name,omitempty"`
	URL      *string `jsonapi:"attr,url,omitempty"`
	Category *string `jsonapi:"attr,category,omitempty"`
	HmacKey  *string `jsonapi:"attr,hmac-key,omitempty"`
}

func (o *RunTaskUpdateOptions) valid() error {
	if o.Name != nil && !validString(o.Name) {
		return ErrRequiredName
	}

	if o.URL != nil && !validString(o.URL) {
		return ErrInvalidRunTaskURL
	}

	if o.Category != nil && *o.Category != "task" {
		return ErrInvalidRunTaskCategory
	}

	return nil
}

func (s *runTasks) Update(ctx context.Context, runTaskID string, options RunTaskUpdateOptions) (*RunTask, error) {
	if !validStringID(&runTaskID) {
		return nil, ErrInvalidRunTaskID
	}

	if err := options.valid(); err != nil {
		return nil, err
	}

	u := fmt.Sprintf("tasks/%s", url.QueryEscape(runTaskID))
	req, err := s.client.newRequest("PATCH", u, &options)
	if err != nil {
		return nil, err
	}

	r := &RunTask{}
	err = s.client.do(ctx, req, r)
	if err != nil {
		return nil, err
	}

	return r, nil
}

func (s *runTasks) Delete(ctx context.Context, runTaskID string) error {
	if !validStringID(&runTaskID) {
		return ErrInvalidRunTaskID
	}

	u := fmt.Sprintf("tasks/%s", runTaskID)
	req, err := s.client.newRequest("DELETE", u, nil)
	if err != nil {
		return err
	}

	return s.client.do(ctx, req, nil)
}
