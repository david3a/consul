// Code generated by mockery v2.15.0. DO NOT EDIT.

package pbdns

import (
	context "context"

	mock "github.com/stretchr/testify/mock"
)

// MockDNSServiceServer is an autogenerated mock type for the DNSServiceServer type
type MockDNSServiceServer struct {
	mock.Mock
}

// Query provides a mock function with given fields: _a0, _a1
func (_m *MockDNSServiceServer) Query(_a0 context.Context, _a1 *QueryRequest) (*QueryResponse, error) {
	ret := _m.Called(_a0, _a1)

	var r0 *QueryResponse
	if rf, ok := ret.Get(0).(func(context.Context, *QueryRequest) *QueryResponse); ok {
		r0 = rf(_a0, _a1)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*QueryResponse)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, *QueryRequest) error); ok {
		r1 = rf(_a0, _a1)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

type mockConstructorTestingTNewMockDNSServiceServer interface {
	mock.TestingT
	Cleanup(func())
}

// NewMockDNSServiceServer creates a new instance of MockDNSServiceServer. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func NewMockDNSServiceServer(t mockConstructorTestingTNewMockDNSServiceServer) *MockDNSServiceServer {
	mock := &MockDNSServiceServer{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}