// Code generated by MockGen. DO NOT EDIT.
// Source: agent.go
//
// Generated by this command:
//
//	mockgen -source=agent.go -destination=mocks/agents.go -package=mocks
//

// Package mocks is a generated GoMock package.
package mocks

import (
	context "context"
	reflect "reflect"

	tfe "github.com/hashicorp/go-tfe"
	gomock "go.uber.org/mock/gomock"
)

// MockAgents is a mock of Agents interface.
type MockAgents struct {
	ctrl     *gomock.Controller
	recorder *MockAgentsMockRecorder
}

// MockAgentsMockRecorder is the mock recorder for MockAgents.
type MockAgentsMockRecorder struct {
	mock *MockAgents
}

// NewMockAgents creates a new mock instance.
func NewMockAgents(ctrl *gomock.Controller) *MockAgents {
	mock := &MockAgents{ctrl: ctrl}
	mock.recorder = &MockAgentsMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockAgents) EXPECT() *MockAgentsMockRecorder {
	return m.recorder
}

// List mocks base method.
func (m *MockAgents) List(ctx context.Context, agentPoolID string, options *tfe.AgentListOptions) (*tfe.AgentList, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "List", ctx, agentPoolID, options)
	ret0, _ := ret[0].(*tfe.AgentList)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// List indicates an expected call of List.
func (mr *MockAgentsMockRecorder) List(ctx, agentPoolID, options any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "List", reflect.TypeOf((*MockAgents)(nil).List), ctx, agentPoolID, options)
}

// Read mocks base method.
func (m *MockAgents) Read(ctx context.Context, agentID string) (*tfe.Agent, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Read", ctx, agentID)
	ret0, _ := ret[0].(*tfe.Agent)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Read indicates an expected call of Read.
func (mr *MockAgentsMockRecorder) Read(ctx, agentID any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Read", reflect.TypeOf((*MockAgents)(nil).Read), ctx, agentID)
}
