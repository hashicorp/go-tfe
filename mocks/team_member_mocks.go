// Code generated by MockGen. DO NOT EDIT.
// Source: team_member.go
//
// Generated by this command:
//
//	mockgen -source=team_member.go -destination=mocks/team_member_mocks.go -package=mocks
//

// Package mocks is a generated GoMock package.
package mocks

import (
	context "context"
	reflect "reflect"

	tfe "github.com/optable/go-tfe"
	gomock "go.uber.org/mock/gomock"
)

// MockTeamMembers is a mock of TeamMembers interface.
type MockTeamMembers struct {
	ctrl     *gomock.Controller
	recorder *MockTeamMembersMockRecorder
}

// MockTeamMembersMockRecorder is the mock recorder for MockTeamMembers.
type MockTeamMembersMockRecorder struct {
	mock *MockTeamMembers
}

// NewMockTeamMembers creates a new mock instance.
func NewMockTeamMembers(ctrl *gomock.Controller) *MockTeamMembers {
	mock := &MockTeamMembers{ctrl: ctrl}
	mock.recorder = &MockTeamMembersMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockTeamMembers) EXPECT() *MockTeamMembersMockRecorder {
	return m.recorder
}

// Add mocks base method.
func (m *MockTeamMembers) Add(ctx context.Context, teamID string, options tfe.TeamMemberAddOptions) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Add", ctx, teamID, options)
	ret0, _ := ret[0].(error)
	return ret0
}

// Add indicates an expected call of Add.
func (mr *MockTeamMembersMockRecorder) Add(ctx, teamID, options any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Add", reflect.TypeOf((*MockTeamMembers)(nil).Add), ctx, teamID, options)
}

// List mocks base method.
func (m *MockTeamMembers) List(ctx context.Context, teamID string) ([]*tfe.User, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "List", ctx, teamID)
	ret0, _ := ret[0].([]*tfe.User)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// List indicates an expected call of List.
func (mr *MockTeamMembersMockRecorder) List(ctx, teamID any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "List", reflect.TypeOf((*MockTeamMembers)(nil).List), ctx, teamID)
}

// ListOrganizationMemberships mocks base method.
func (m *MockTeamMembers) ListOrganizationMemberships(ctx context.Context, teamID string) ([]*tfe.OrganizationMembership, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListOrganizationMemberships", ctx, teamID)
	ret0, _ := ret[0].([]*tfe.OrganizationMembership)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListOrganizationMemberships indicates an expected call of ListOrganizationMemberships.
func (mr *MockTeamMembersMockRecorder) ListOrganizationMemberships(ctx, teamID any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListOrganizationMemberships", reflect.TypeOf((*MockTeamMembers)(nil).ListOrganizationMemberships), ctx, teamID)
}

// ListUsers mocks base method.
func (m *MockTeamMembers) ListUsers(ctx context.Context, teamID string) ([]*tfe.User, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListUsers", ctx, teamID)
	ret0, _ := ret[0].([]*tfe.User)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListUsers indicates an expected call of ListUsers.
func (mr *MockTeamMembersMockRecorder) ListUsers(ctx, teamID any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListUsers", reflect.TypeOf((*MockTeamMembers)(nil).ListUsers), ctx, teamID)
}

// Remove mocks base method.
func (m *MockTeamMembers) Remove(ctx context.Context, teamID string, options tfe.TeamMemberRemoveOptions) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Remove", ctx, teamID, options)
	ret0, _ := ret[0].(error)
	return ret0
}

// Remove indicates an expected call of Remove.
func (mr *MockTeamMembersMockRecorder) Remove(ctx, teamID, options any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Remove", reflect.TypeOf((*MockTeamMembers)(nil).Remove), ctx, teamID, options)
}
