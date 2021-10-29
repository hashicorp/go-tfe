// Code generated by MockGen. DO NOT EDIT.
// Source: policy_check.go

// Package mocks is a generated GoMock package.
package mocks

import (
	context "context"
	io "io"
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"
)

// MockPolicyChecks is a mock of PolicyChecks interface.
type MockPolicyChecks struct {
	ctrl     *gomock.Controller
	recorder *MockPolicyChecksMockRecorder
}

// MockPolicyChecksMockRecorder is the mock recorder for MockPolicyChecks.
type MockPolicyChecksMockRecorder struct {
	mock *MockPolicyChecks
}

// NewMockPolicyChecks creates a new mock instance.
func NewMockPolicyChecks(ctrl *gomock.Controller) *MockPolicyChecks {
	mock := &MockPolicyChecks{ctrl: ctrl}
	mock.recorder = &MockPolicyChecksMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockPolicyChecks) EXPECT() *MockPolicyChecksMockRecorder {
	return m.recorder
}

// List mocks base method.
func (m *MockPolicyChecks) List(ctx context.Context, runID string, options PolicyCheckListOptions) (*PolicyCheckList, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "List", ctx, runID, options)
	ret0, _ := ret[0].(*PolicyCheckList)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// List indicates an expected call of List.
func (mr *MockPolicyChecksMockRecorder) List(ctx, runID, options interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "List", reflect.TypeOf((*MockPolicyChecks)(nil).List), ctx, runID, options)
}

// Logs mocks base method.
func (m *MockPolicyChecks) Logs(ctx context.Context, policyCheckID string) (io.Reader, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Logs", ctx, policyCheckID)
	ret0, _ := ret[0].(io.Reader)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Logs indicates an expected call of Logs.
func (mr *MockPolicyChecksMockRecorder) Logs(ctx, policyCheckID interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Logs", reflect.TypeOf((*MockPolicyChecks)(nil).Logs), ctx, policyCheckID)
}

// Override mocks base method.
func (m *MockPolicyChecks) Override(ctx context.Context, policyCheckID string) (*PolicyCheck, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Override", ctx, policyCheckID)
	ret0, _ := ret[0].(*PolicyCheck)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Override indicates an expected call of Override.
func (mr *MockPolicyChecksMockRecorder) Override(ctx, policyCheckID interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Override", reflect.TypeOf((*MockPolicyChecks)(nil).Override), ctx, policyCheckID)
}

// Read mocks base method.
func (m *MockPolicyChecks) Read(ctx context.Context, policyCheckID string) (*PolicyCheck, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Read", ctx, policyCheckID)
	ret0, _ := ret[0].(*PolicyCheck)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Read indicates an expected call of Read.
func (mr *MockPolicyChecksMockRecorder) Read(ctx, policyCheckID interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Read", reflect.TypeOf((*MockPolicyChecks)(nil).Read), ctx, policyCheckID)
}
