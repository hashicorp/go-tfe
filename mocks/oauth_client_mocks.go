// Code generated by MockGen. DO NOT EDIT.
// Source: oauth_client.go

// Package mocks is a generated GoMock package.
package mocks

import (
	context "context"
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"
	tfe "github.com/hashicorp/go-tfe"
)

// MockOAuthClients is a mock of OAuthClients interface.
type MockOAuthClients struct {
	ctrl     *gomock.Controller
	recorder *MockOAuthClientsMockRecorder
}

// MockOAuthClientsMockRecorder is the mock recorder for MockOAuthClients.
type MockOAuthClientsMockRecorder struct {
	mock *MockOAuthClients
}

// NewMockOAuthClients creates a new mock instance.
func NewMockOAuthClients(ctrl *gomock.Controller) *MockOAuthClients {
	mock := &MockOAuthClients{ctrl: ctrl}
	mock.recorder = &MockOAuthClientsMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockOAuthClients) EXPECT() *MockOAuthClientsMockRecorder {
	return m.recorder
}

// Create mocks base method.
func (m *MockOAuthClients) Create(ctx context.Context, organization string, options tfe.OAuthClientCreateOptions) (*tfe.OAuthClient, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Create", ctx, organization, options)
	ret0, _ := ret[0].(*tfe.OAuthClient)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Create indicates an expected call of Create.
func (mr *MockOAuthClientsMockRecorder) Create(ctx, organization, options interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Create", reflect.TypeOf((*MockOAuthClients)(nil).Create), ctx, organization, options)
}

// Delete mocks base method.
func (m *MockOAuthClients) Delete(ctx context.Context, oAuthClientID string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Delete", ctx, oAuthClientID)
	ret0, _ := ret[0].(error)
	return ret0
}

// Delete indicates an expected call of Delete.
func (mr *MockOAuthClientsMockRecorder) Delete(ctx, oAuthClientID interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Delete", reflect.TypeOf((*MockOAuthClients)(nil).Delete), ctx, oAuthClientID)
}

// List mocks base method.
func (m *MockOAuthClients) List(ctx context.Context, organization string, options tfe.OAuthClientListOptions) (*tfe.OAuthClientList, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "List", ctx, organization, options)
	ret0, _ := ret[0].(*tfe.OAuthClientList)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// List indicates an expected call of List.
func (mr *MockOAuthClientsMockRecorder) List(ctx, organization, options interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "List", reflect.TypeOf((*MockOAuthClients)(nil).List), ctx, organization, options)
}

// Read mocks base method.
func (m *MockOAuthClients) Read(ctx context.Context, oAuthClientID string) (*tfe.OAuthClient, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Read", ctx, oAuthClientID)
	ret0, _ := ret[0].(*tfe.OAuthClient)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Read indicates an expected call of Read.
func (mr *MockOAuthClientsMockRecorder) Read(ctx, oAuthClientID interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Read", reflect.TypeOf((*MockOAuthClients)(nil).Read), ctx, oAuthClientID)
}
