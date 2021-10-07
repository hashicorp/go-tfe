package tfe

import (
	"context"
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"
)

// MockResources is a mock of Resources interface.
type MockResources struct {
	ctrl     *gomock.Controller
	recorder *MockResourcesMockRecorder
}

// MockWorkspacesMockRecorder is the mock recorder for MockWorkspaces.
type MockResourcesMockRecorder struct {
	mock *MockResources
}

// NewMockResources creates a new mock instance.
func NewMockResources(ctrl *gomock.Controller) *MockResources {
	mock := &MockResources{ctrl: ctrl}
	mock.recorder = &MockResourcesMockRecorder{mock}
	return mock
}

// List mocks base method.
func (m *MockResources) List(ctx context.Context, workspaceID string, options ResourceListOptions) (*ResourceList, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "List", ctx, workspaceID, options)
	ret0, _ := ret[0].(*ResourceList)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// List indicates an expected call of List.
func (mr *MockResourcesMockRecorder) List(ctx, organization, options interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "List", reflect.TypeOf((*MockResources)(nil).List), ctx, organization, options)
}
