// Code generated by mockery v1.0.0
package automock

import mock "github.com/stretchr/testify/mock"
import types "github.com/kyma-project/kyma/components/console-backend-service/internal/domain/k8s/types"

// resourceSvc is an autogenerated mock type for the resourceSvc type
type resourceSvc struct {
	mock.Mock
}

// Create provides a mock function with given fields: namespace, resource
func (_m *resourceSvc) Create(namespace string, resource types.Resource) (*types.Resource, error) {
	ret := _m.Called(namespace, resource)

	var r0 *types.Resource
	if rf, ok := ret.Get(0).(func(string, types.Resource) *types.Resource); ok {
		r0 = rf(namespace, resource)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*types.Resource)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(string, types.Resource) error); ok {
		r1 = rf(namespace, resource)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
