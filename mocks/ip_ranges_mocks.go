// Code generated by MockGen. DO NOT EDIT.
// Source: ip_ranges.go

// Package mocks is a generated GoMock package.
package mocks

import (
	context "context"
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"
)

// MockIPRanges is a mock of IPRanges interface.
type MockIPRanges struct {
	ctrl     *gomock.Controller
	recorder *MockIPRangesMockRecorder
}

// MockIPRangesMockRecorder is the mock recorder for MockIPRanges.
type MockIPRangesMockRecorder struct {
	mock *MockIPRanges
}

// NewMockIPRanges creates a new mock instance.
func NewMockIPRanges(ctrl *gomock.Controller) *MockIPRanges {
	mock := &MockIPRanges{ctrl: ctrl}
	mock.recorder = &MockIPRangesMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockIPRanges) EXPECT() *MockIPRangesMockRecorder {
	return m.recorder
}

// Read mocks base method.
func (m *MockIPRanges) Read(ctx context.Context, modifiedSince string) (*IPRange, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Read", ctx, modifiedSince)
	ret0, _ := ret[0].(*IPRange)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Read indicates an expected call of Read.
func (mr *MockIPRangesMockRecorder) Read(ctx, modifiedSince interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Read", reflect.TypeOf((*MockIPRanges)(nil).Read), ctx, modifiedSince)
}
