// Code generated by MockGen. DO NOT EDIT.
// Source: admin_setting_smtp.go
//
// Generated by this command:
//
//	mockgen -source=admin_setting_smtp.go -destination=mocks/admin_setting_smtp_mocks.go -package=mocks
//

// Package mocks is a generated GoMock package.
package mocks

import (
	context "context"
	reflect "reflect"

	tfe "github.com/optable/go-tfe"
	gomock "go.uber.org/mock/gomock"
)

// MockSMTPSettings is a mock of SMTPSettings interface.
type MockSMTPSettings struct {
	ctrl     *gomock.Controller
	recorder *MockSMTPSettingsMockRecorder
}

// MockSMTPSettingsMockRecorder is the mock recorder for MockSMTPSettings.
type MockSMTPSettingsMockRecorder struct {
	mock *MockSMTPSettings
}

// NewMockSMTPSettings creates a new mock instance.
func NewMockSMTPSettings(ctrl *gomock.Controller) *MockSMTPSettings {
	mock := &MockSMTPSettings{ctrl: ctrl}
	mock.recorder = &MockSMTPSettingsMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockSMTPSettings) EXPECT() *MockSMTPSettingsMockRecorder {
	return m.recorder
}

// Read mocks base method.
func (m *MockSMTPSettings) Read(ctx context.Context) (*tfe.AdminSMTPSetting, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Read", ctx)
	ret0, _ := ret[0].(*tfe.AdminSMTPSetting)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Read indicates an expected call of Read.
func (mr *MockSMTPSettingsMockRecorder) Read(ctx any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Read", reflect.TypeOf((*MockSMTPSettings)(nil).Read), ctx)
}

// Update mocks base method.
func (m *MockSMTPSettings) Update(ctx context.Context, options tfe.AdminSMTPSettingsUpdateOptions) (*tfe.AdminSMTPSetting, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Update", ctx, options)
	ret0, _ := ret[0].(*tfe.AdminSMTPSetting)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Update indicates an expected call of Update.
func (mr *MockSMTPSettingsMockRecorder) Update(ctx, options any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Update", reflect.TypeOf((*MockSMTPSettings)(nil).Update), ctx, options)
}
