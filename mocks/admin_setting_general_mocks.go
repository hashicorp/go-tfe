// Code generated by MockGen. DO NOT EDIT.
// Source: admin_setting_general.go
//
// Generated by this command:
//
//	mockgen -source=admin_setting_general.go -destination=mocks/admin_setting_general_mocks.go -package=mocks
//

// Package mocks is a generated GoMock package.
package mocks

import (
	context "context"
	reflect "reflect"

	tfe "github.com/optable/go-tfe"
	gomock "go.uber.org/mock/gomock"
)

// MockGeneralSettings is a mock of GeneralSettings interface.
type MockGeneralSettings struct {
	ctrl     *gomock.Controller
	recorder *MockGeneralSettingsMockRecorder
}

// MockGeneralSettingsMockRecorder is the mock recorder for MockGeneralSettings.
type MockGeneralSettingsMockRecorder struct {
	mock *MockGeneralSettings
}

// NewMockGeneralSettings creates a new mock instance.
func NewMockGeneralSettings(ctrl *gomock.Controller) *MockGeneralSettings {
	mock := &MockGeneralSettings{ctrl: ctrl}
	mock.recorder = &MockGeneralSettingsMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockGeneralSettings) EXPECT() *MockGeneralSettingsMockRecorder {
	return m.recorder
}

// Read mocks base method.
func (m *MockGeneralSettings) Read(ctx context.Context) (*tfe.AdminGeneralSetting, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Read", ctx)
	ret0, _ := ret[0].(*tfe.AdminGeneralSetting)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Read indicates an expected call of Read.
func (mr *MockGeneralSettingsMockRecorder) Read(ctx any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Read", reflect.TypeOf((*MockGeneralSettings)(nil).Read), ctx)
}

// Update mocks base method.
func (m *MockGeneralSettings) Update(ctx context.Context, options tfe.AdminGeneralSettingsUpdateOptions) (*tfe.AdminGeneralSetting, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Update", ctx, options)
	ret0, _ := ret[0].(*tfe.AdminGeneralSetting)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Update indicates an expected call of Update.
func (mr *MockGeneralSettingsMockRecorder) Update(ctx, options any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Update", reflect.TypeOf((*MockGeneralSettings)(nil).Update), ctx, options)
}
