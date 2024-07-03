// Code generated by MockGen. DO NOT EDIT.
// Source: run_task.go
//
// Generated by this command:
//
//	mockgen -source=run_task.go -destination=mocks/run_tasks_mocks.go -package=mocks
//

// Package mocks is a generated GoMock package.
package mocks

import (
	context "context"
	reflect "reflect"

	tfe "github.com/optable/go-tfe"
	gomock "go.uber.org/mock/gomock"
)

// MockRunTasks is a mock of RunTasks interface.
type MockRunTasks struct {
	ctrl     *gomock.Controller
	recorder *MockRunTasksMockRecorder
}

// MockRunTasksMockRecorder is the mock recorder for MockRunTasks.
type MockRunTasksMockRecorder struct {
	mock *MockRunTasks
}

// NewMockRunTasks creates a new mock instance.
func NewMockRunTasks(ctrl *gomock.Controller) *MockRunTasks {
	mock := &MockRunTasks{ctrl: ctrl}
	mock.recorder = &MockRunTasksMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockRunTasks) EXPECT() *MockRunTasksMockRecorder {
	return m.recorder
}

// AttachToWorkspace mocks base method.
func (m *MockRunTasks) AttachToWorkspace(ctx context.Context, workspaceID, runTaskID string, enforcementLevel tfe.TaskEnforcementLevel) (*tfe.WorkspaceRunTask, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "AttachToWorkspace", ctx, workspaceID, runTaskID, enforcementLevel)
	ret0, _ := ret[0].(*tfe.WorkspaceRunTask)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// AttachToWorkspace indicates an expected call of AttachToWorkspace.
func (mr *MockRunTasksMockRecorder) AttachToWorkspace(ctx, workspaceID, runTaskID, enforcementLevel any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "AttachToWorkspace", reflect.TypeOf((*MockRunTasks)(nil).AttachToWorkspace), ctx, workspaceID, runTaskID, enforcementLevel)
}

// Create mocks base method.
func (m *MockRunTasks) Create(ctx context.Context, organization string, options tfe.RunTaskCreateOptions) (*tfe.RunTask, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Create", ctx, organization, options)
	ret0, _ := ret[0].(*tfe.RunTask)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Create indicates an expected call of Create.
func (mr *MockRunTasksMockRecorder) Create(ctx, organization, options any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Create", reflect.TypeOf((*MockRunTasks)(nil).Create), ctx, organization, options)
}

// Delete mocks base method.
func (m *MockRunTasks) Delete(ctx context.Context, runTaskID string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Delete", ctx, runTaskID)
	ret0, _ := ret[0].(error)
	return ret0
}

// Delete indicates an expected call of Delete.
func (mr *MockRunTasksMockRecorder) Delete(ctx, runTaskID any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Delete", reflect.TypeOf((*MockRunTasks)(nil).Delete), ctx, runTaskID)
}

// List mocks base method.
func (m *MockRunTasks) List(ctx context.Context, organization string, options *tfe.RunTaskListOptions) (*tfe.RunTaskList, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "List", ctx, organization, options)
	ret0, _ := ret[0].(*tfe.RunTaskList)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// List indicates an expected call of List.
func (mr *MockRunTasksMockRecorder) List(ctx, organization, options any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "List", reflect.TypeOf((*MockRunTasks)(nil).List), ctx, organization, options)
}

// Read mocks base method.
func (m *MockRunTasks) Read(ctx context.Context, runTaskID string) (*tfe.RunTask, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Read", ctx, runTaskID)
	ret0, _ := ret[0].(*tfe.RunTask)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Read indicates an expected call of Read.
func (mr *MockRunTasksMockRecorder) Read(ctx, runTaskID any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Read", reflect.TypeOf((*MockRunTasks)(nil).Read), ctx, runTaskID)
}

// ReadWithOptions mocks base method.
func (m *MockRunTasks) ReadWithOptions(ctx context.Context, runTaskID string, options *tfe.RunTaskReadOptions) (*tfe.RunTask, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ReadWithOptions", ctx, runTaskID, options)
	ret0, _ := ret[0].(*tfe.RunTask)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ReadWithOptions indicates an expected call of ReadWithOptions.
func (mr *MockRunTasksMockRecorder) ReadWithOptions(ctx, runTaskID, options any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ReadWithOptions", reflect.TypeOf((*MockRunTasks)(nil).ReadWithOptions), ctx, runTaskID, options)
}

// Update mocks base method.
func (m *MockRunTasks) Update(ctx context.Context, runTaskID string, options tfe.RunTaskUpdateOptions) (*tfe.RunTask, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Update", ctx, runTaskID, options)
	ret0, _ := ret[0].(*tfe.RunTask)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Update indicates an expected call of Update.
func (mr *MockRunTasksMockRecorder) Update(ctx, runTaskID, options any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Update", reflect.TypeOf((*MockRunTasks)(nil).Update), ctx, runTaskID, options)
}
