// Code generated by MockGen. DO NOT EDIT.
// Source: variable.go
//
// Generated by this command:
//
//	mockgen -source=variable.go -destination=mocks/variable_mocks.go -package=mocks
//

// Package mocks is a generated GoMock package.
package mocks

import (
	context "context"
	reflect "reflect"

	tfe "github.com/optable/go-tfe"
	gomock "go.uber.org/mock/gomock"
)

// MockVariables is a mock of Variables interface.
type MockVariables struct {
	ctrl     *gomock.Controller
	recorder *MockVariablesMockRecorder
}

// MockVariablesMockRecorder is the mock recorder for MockVariables.
type MockVariablesMockRecorder struct {
	mock *MockVariables
}

// NewMockVariables creates a new mock instance.
func NewMockVariables(ctrl *gomock.Controller) *MockVariables {
	mock := &MockVariables{ctrl: ctrl}
	mock.recorder = &MockVariablesMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockVariables) EXPECT() *MockVariablesMockRecorder {
	return m.recorder
}

// Create mocks base method.
func (m *MockVariables) Create(ctx context.Context, workspaceID string, options tfe.VariableCreateOptions) (*tfe.Variable, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Create", ctx, workspaceID, options)
	ret0, _ := ret[0].(*tfe.Variable)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Create indicates an expected call of Create.
func (mr *MockVariablesMockRecorder) Create(ctx, workspaceID, options any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Create", reflect.TypeOf((*MockVariables)(nil).Create), ctx, workspaceID, options)
}

// Delete mocks base method.
func (m *MockVariables) Delete(ctx context.Context, workspaceID, variableID string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Delete", ctx, workspaceID, variableID)
	ret0, _ := ret[0].(error)
	return ret0
}

// Delete indicates an expected call of Delete.
func (mr *MockVariablesMockRecorder) Delete(ctx, workspaceID, variableID any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Delete", reflect.TypeOf((*MockVariables)(nil).Delete), ctx, workspaceID, variableID)
}

// List mocks base method.
func (m *MockVariables) List(ctx context.Context, workspaceID string, options *tfe.VariableListOptions) (*tfe.VariableList, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "List", ctx, workspaceID, options)
	ret0, _ := ret[0].(*tfe.VariableList)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// List indicates an expected call of List.
func (mr *MockVariablesMockRecorder) List(ctx, workspaceID, options any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "List", reflect.TypeOf((*MockVariables)(nil).List), ctx, workspaceID, options)
}

// Read mocks base method.
func (m *MockVariables) Read(ctx context.Context, workspaceID, variableID string) (*tfe.Variable, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Read", ctx, workspaceID, variableID)
	ret0, _ := ret[0].(*tfe.Variable)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Read indicates an expected call of Read.
func (mr *MockVariablesMockRecorder) Read(ctx, workspaceID, variableID any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Read", reflect.TypeOf((*MockVariables)(nil).Read), ctx, workspaceID, variableID)
}

// Update mocks base method.
func (m *MockVariables) Update(ctx context.Context, workspaceID, variableID string, options tfe.VariableUpdateOptions) (*tfe.Variable, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Update", ctx, workspaceID, variableID, options)
	ret0, _ := ret[0].(*tfe.Variable)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Update indicates an expected call of Update.
func (mr *MockVariablesMockRecorder) Update(ctx, workspaceID, variableID, options any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Update", reflect.TypeOf((*MockVariables)(nil).Update), ctx, workspaceID, variableID, options)
}
