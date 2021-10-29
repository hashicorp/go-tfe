// Code generated by MockGen. DO NOT EDIT.
// Source: admin_organization.go

// Package mocks is a generated GoMock package.
package mocks

import (
	context "context"
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"
)

// MockAdminOrganizations is a mock of AdminOrganizations interface.
type MockAdminOrganizations struct {
	ctrl     *gomock.Controller
	recorder *MockAdminOrganizationsMockRecorder
}

// MockAdminOrganizationsMockRecorder is the mock recorder for MockAdminOrganizations.
type MockAdminOrganizationsMockRecorder struct {
	mock *MockAdminOrganizations
}

// NewMockAdminOrganizations creates a new mock instance.
func NewMockAdminOrganizations(ctrl *gomock.Controller) *MockAdminOrganizations {
	mock := &MockAdminOrganizations{ctrl: ctrl}
	mock.recorder = &MockAdminOrganizationsMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockAdminOrganizations) EXPECT() *MockAdminOrganizationsMockRecorder {
	return m.recorder
}

// Delete mocks base method.
func (m *MockAdminOrganizations) Delete(ctx context.Context, organization string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Delete", ctx, organization)
	ret0, _ := ret[0].(error)
	return ret0
}

// Delete indicates an expected call of Delete.
func (mr *MockAdminOrganizationsMockRecorder) Delete(ctx, organization interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Delete", reflect.TypeOf((*MockAdminOrganizations)(nil).Delete), ctx, organization)
}

// List mocks base method.
func (m *MockAdminOrganizations) List(ctx context.Context, options AdminOrganizationListOptions) (*AdminOrganizationList, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "List", ctx, options)
	ret0, _ := ret[0].(*AdminOrganizationList)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// List indicates an expected call of List.
func (mr *MockAdminOrganizationsMockRecorder) List(ctx, options interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "List", reflect.TypeOf((*MockAdminOrganizations)(nil).List), ctx, options)
}

// Read mocks base method.
func (m *MockAdminOrganizations) Read(ctx context.Context, organization string) (*AdminOrganization, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Read", ctx, organization)
	ret0, _ := ret[0].(*AdminOrganization)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Read indicates an expected call of Read.
func (mr *MockAdminOrganizationsMockRecorder) Read(ctx, organization interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Read", reflect.TypeOf((*MockAdminOrganizations)(nil).Read), ctx, organization)
}

// Update mocks base method.
func (m *MockAdminOrganizations) Update(ctx context.Context, organization string, options AdminOrganizationUpdateOptions) (*AdminOrganization, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Update", ctx, organization, options)
	ret0, _ := ret[0].(*AdminOrganization)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Update indicates an expected call of Update.
func (mr *MockAdminOrganizationsMockRecorder) Update(ctx, organization, options interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Update", reflect.TypeOf((*MockAdminOrganizations)(nil).Update), ctx, organization, options)
}
