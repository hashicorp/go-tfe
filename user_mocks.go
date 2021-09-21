// Code generated by MockGen. DO NOT EDIT.
// Source: user.go

// Package tfe is a generated GoMock package.
package tfe

import (
	context "context"
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"
)

// MockUsers is a mock of Users interface.
type MockUsers struct {
	ctrl     *gomock.Controller
	recorder *MockUsersMockRecorder
}

// MockUsersMockRecorder is the mock recorder for MockUsers.
type MockUsersMockRecorder struct {
	mock *MockUsers
}

// NewMockUsers creates a new mock instance.
func NewMockUsers(ctrl *gomock.Controller) *MockUsers {
	mock := &MockUsers{ctrl: ctrl}
	mock.recorder = &MockUsersMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockUsers) EXPECT() *MockUsersMockRecorder {
	return m.recorder
}

// ReadCurrent mocks base method.
func (m *MockUsers) ReadCurrent(ctx context.Context) (*User, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ReadCurrent", ctx)
	ret0, _ := ret[0].(*User)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ReadCurrent indicates an expected call of ReadCurrent.
func (mr *MockUsersMockRecorder) ReadCurrent(ctx interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ReadCurrent", reflect.TypeOf((*MockUsers)(nil).ReadCurrent), ctx)
}

// Update mocks base method.
func (m *MockUsers) Update(ctx context.Context, options UserUpdateOptions) (*User, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Update", ctx, options)
	ret0, _ := ret[0].(*User)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Update indicates an expected call of Update.
func (mr *MockUsersMockRecorder) Update(ctx, options interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Update", reflect.TypeOf((*MockUsers)(nil).Update), ctx, options)
}
