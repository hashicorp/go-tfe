// Code generated by MockGen. DO NOT EDIT.
// Source: agent_token.go

// Package mocks is a generated GoMock package.
package mocks

import (
	context "context"
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"
)

// MockAgentTokens is a mock of AgentTokens interface.
type MockAgentTokens struct {
	ctrl     *gomock.Controller
	recorder *MockAgentTokensMockRecorder
}

// MockAgentTokensMockRecorder is the mock recorder for MockAgentTokens.
type MockAgentTokensMockRecorder struct {
	mock *MockAgentTokens
}

// NewMockAgentTokens creates a new mock instance.
func NewMockAgentTokens(ctrl *gomock.Controller) *MockAgentTokens {
	mock := &MockAgentTokens{ctrl: ctrl}
	mock.recorder = &MockAgentTokensMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockAgentTokens) EXPECT() *MockAgentTokensMockRecorder {
	return m.recorder
}

// Delete mocks base method.
func (m *MockAgentTokens) Delete(ctx context.Context, agentTokenID string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Delete", ctx, agentTokenID)
	ret0, _ := ret[0].(error)
	return ret0
}

// Delete indicates an expected call of Delete.
func (mr *MockAgentTokensMockRecorder) Delete(ctx, agentTokenID interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Delete", reflect.TypeOf((*MockAgentTokens)(nil).Delete), ctx, agentTokenID)
}

// Generate mocks base method.
func (m *MockAgentTokens) Generate(ctx context.Context, agentPoolID string, options AgentTokenGenerateOptions) (*AgentToken, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Generate", ctx, agentPoolID, options)
	ret0, _ := ret[0].(*AgentToken)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Generate indicates an expected call of Generate.
func (mr *MockAgentTokensMockRecorder) Generate(ctx, agentPoolID, options interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Generate", reflect.TypeOf((*MockAgentTokens)(nil).Generate), ctx, agentPoolID, options)
}

// List mocks base method.
func (m *MockAgentTokens) List(ctx context.Context, agentPoolID string) (*AgentTokenList, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "List", ctx, agentPoolID)
	ret0, _ := ret[0].(*AgentTokenList)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// List indicates an expected call of List.
func (mr *MockAgentTokensMockRecorder) List(ctx, agentPoolID interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "List", reflect.TypeOf((*MockAgentTokens)(nil).List), ctx, agentPoolID)
}

// Read mocks base method.
func (m *MockAgentTokens) Read(ctx context.Context, agentTokenID string) (*AgentToken, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Read", ctx, agentTokenID)
	ret0, _ := ret[0].(*AgentToken)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Read indicates an expected call of Read.
func (mr *MockAgentTokensMockRecorder) Read(ctx, agentTokenID interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Read", reflect.TypeOf((*MockAgentTokens)(nil).Read), ctx, agentTokenID)
}
