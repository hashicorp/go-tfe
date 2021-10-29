// Code generated by MockGen. DO NOT EDIT.
// Source: user_token.go

// Package mocks is a generated GoMock package.
package mocks

import (
	context "context"
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"
)

// MockUserTokens is a mock of UserTokens interface.
type MockUserTokens struct {
	ctrl     *gomock.Controller
	recorder *MockUserTokensMockRecorder
}

// MockUserTokensMockRecorder is the mock recorder for MockUserTokens.
type MockUserTokensMockRecorder struct {
	mock *MockUserTokens
}

// NewMockUserTokens creates a new mock instance.
func NewMockUserTokens(ctrl *gomock.Controller) *MockUserTokens {
	mock := &MockUserTokens{ctrl: ctrl}
	mock.recorder = &MockUserTokensMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockUserTokens) EXPECT() *MockUserTokensMockRecorder {
	return m.recorder
}

// Delete mocks base method.
func (m *MockUserTokens) Delete(ctx context.Context, tokenID string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Delete", ctx, tokenID)
	ret0, _ := ret[0].(error)
	return ret0
}

// Delete indicates an expected call of Delete.
func (mr *MockUserTokensMockRecorder) Delete(ctx, tokenID interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Delete", reflect.TypeOf((*MockUserTokens)(nil).Delete), ctx, tokenID)
}

// Generate mocks base method.
func (m *MockUserTokens) Generate(ctx context.Context, userID string, options UserTokenGenerateOptions) (*UserToken, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Generate", ctx, userID, options)
	ret0, _ := ret[0].(*UserToken)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Generate indicates an expected call of Generate.
func (mr *MockUserTokensMockRecorder) Generate(ctx, userID, options interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Generate", reflect.TypeOf((*MockUserTokens)(nil).Generate), ctx, userID, options)
}

// List mocks base method.
func (m *MockUserTokens) List(ctx context.Context, userID string) (*UserTokenList, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "List", ctx, userID)
	ret0, _ := ret[0].(*UserTokenList)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// List indicates an expected call of List.
func (mr *MockUserTokensMockRecorder) List(ctx, userID interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "List", reflect.TypeOf((*MockUserTokens)(nil).List), ctx, userID)
}

// Read mocks base method.
func (m *MockUserTokens) Read(ctx context.Context, tokenID string) (*UserToken, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Read", ctx, tokenID)
	ret0, _ := ret[0].(*UserToken)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Read indicates an expected call of Read.
func (mr *MockUserTokensMockRecorder) Read(ctx, tokenID interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Read", reflect.TypeOf((*MockUserTokens)(nil).Read), ctx, tokenID)
}
