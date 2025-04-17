// Code generated by MockGen. DO NOT EDIT.
// Source: project.go
//
// Generated by this command:
//
//	mockgen -source=project.go -destination=mocks/project_mocks.go -package=mocks
//

// Package mocks is a generated GoMock package.
package mocks

import (
	context "context"
	reflect "reflect"

	tfe "github.com/hashicorp/go-tfe"
	gomock "go.uber.org/mock/gomock"
)

// MockProjects is a mock of Projects interface.
type MockProjects struct {
	ctrl     *gomock.Controller
	recorder *MockProjectsMockRecorder
}

// MockProjectsMockRecorder is the mock recorder for MockProjects.
type MockProjectsMockRecorder struct {
	mock *MockProjects
}

// NewMockProjects creates a new mock instance.
func NewMockProjects(ctrl *gomock.Controller) *MockProjects {
	mock := &MockProjects{ctrl: ctrl}
	mock.recorder = &MockProjectsMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockProjects) EXPECT() *MockProjectsMockRecorder {
	return m.recorder
}

// AddTagBindings mocks base method.
func (m *MockProjects) AddTagBindings(ctx context.Context, projectID string, options tfe.ProjectAddTagBindingsOptions) ([]*tfe.TagBinding, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "AddTagBindings", ctx, projectID, options)
	ret0, _ := ret[0].([]*tfe.TagBinding)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// AddTagBindings indicates an expected call of AddTagBindings.
func (mr *MockProjectsMockRecorder) AddTagBindings(ctx, projectID, options any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "AddTagBindings", reflect.TypeOf((*MockProjects)(nil).AddTagBindings), ctx, projectID, options)
}

// Create mocks base method.
func (m *MockProjects) Create(ctx context.Context, organization string, options tfe.ProjectCreateOptions) (*tfe.Project, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Create", ctx, organization, options)
	ret0, _ := ret[0].(*tfe.Project)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Create indicates an expected call of Create.
func (mr *MockProjectsMockRecorder) Create(ctx, organization, options any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Create", reflect.TypeOf((*MockProjects)(nil).Create), ctx, organization, options)
}

// Delete mocks base method.
func (m *MockProjects) Delete(ctx context.Context, projectID string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Delete", ctx, projectID)
	ret0, _ := ret[0].(error)
	return ret0
}

// Delete indicates an expected call of Delete.
func (mr *MockProjectsMockRecorder) Delete(ctx, projectID any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Delete", reflect.TypeOf((*MockProjects)(nil).Delete), ctx, projectID)
}

// DeleteAllTagBindings mocks base method.
func (m *MockProjects) DeleteAllTagBindings(ctx context.Context, projectID string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteAllTagBindings", ctx, projectID)
	ret0, _ := ret[0].(error)
	return ret0
}

// DeleteAllTagBindings indicates an expected call of DeleteAllTagBindings.
func (mr *MockProjectsMockRecorder) DeleteAllTagBindings(ctx, projectID any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteAllTagBindings", reflect.TypeOf((*MockProjects)(nil).DeleteAllTagBindings), ctx, projectID)
}

// List mocks base method.
func (m *MockProjects) List(ctx context.Context, organization string, options *tfe.ProjectListOptions) (*tfe.ProjectList, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "List", ctx, organization, options)
	ret0, _ := ret[0].(*tfe.ProjectList)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// List indicates an expected call of List.
func (mr *MockProjectsMockRecorder) List(ctx, organization, options any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "List", reflect.TypeOf((*MockProjects)(nil).List), ctx, organization, options)
}

// ListEffectiveTagBindings mocks base method.
func (m *MockProjects) ListEffectiveTagBindings(ctx context.Context, workspaceID string) ([]*tfe.EffectiveTagBinding, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListEffectiveTagBindings", ctx, workspaceID)
	ret0, _ := ret[0].([]*tfe.EffectiveTagBinding)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListEffectiveTagBindings indicates an expected call of ListEffectiveTagBindings.
func (mr *MockProjectsMockRecorder) ListEffectiveTagBindings(ctx, workspaceID any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListEffectiveTagBindings", reflect.TypeOf((*MockProjects)(nil).ListEffectiveTagBindings), ctx, workspaceID)
}

// ListTagBindings mocks base method.
func (m *MockProjects) ListTagBindings(ctx context.Context, projectID string) ([]*tfe.TagBinding, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListTagBindings", ctx, projectID)
	ret0, _ := ret[0].([]*tfe.TagBinding)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListTagBindings indicates an expected call of ListTagBindings.
func (mr *MockProjectsMockRecorder) ListTagBindings(ctx, projectID any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListTagBindings", reflect.TypeOf((*MockProjects)(nil).ListTagBindings), ctx, projectID)
}

// Read mocks base method.
func (m *MockProjects) Read(ctx context.Context, projectID string) (*tfe.Project, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Read", ctx, projectID)
	ret0, _ := ret[0].(*tfe.Project)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Read indicates an expected call of Read.
func (mr *MockProjectsMockRecorder) Read(ctx, projectID any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Read", reflect.TypeOf((*MockProjects)(nil).Read), ctx, projectID)
}

// ReadWithOptions mocks base method.
func (m *MockProjects) ReadWithOptions(ctx context.Context, projectID string, options tfe.ProjectReadOptions) (*tfe.Project, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ReadWithOptions", ctx, projectID, options)
	ret0, _ := ret[0].(*tfe.Project)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ReadWithOptions indicates an expected call of ReadWithOptions.
func (mr *MockProjectsMockRecorder) ReadWithOptions(ctx, projectID, options any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ReadWithOptions", reflect.TypeOf((*MockProjects)(nil).ReadWithOptions), ctx, projectID, options)
}

// Update mocks base method.
func (m *MockProjects) Update(ctx context.Context, projectID string, options tfe.ProjectUpdateOptions) (*tfe.Project, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Update", ctx, projectID, options)
	ret0, _ := ret[0].(*tfe.Project)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Update indicates an expected call of Update.
func (mr *MockProjectsMockRecorder) Update(ctx, projectID, options any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Update", reflect.TypeOf((*MockProjects)(nil).Update), ctx, projectID, options)
}
