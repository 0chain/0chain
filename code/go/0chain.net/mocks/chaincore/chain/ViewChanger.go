// Code generated by mockery v0.0.0-dev. DO NOT EDIT.

package mocks

import (
	block "0chain.net/chaincore/block"

	context "context"

	mock "github.com/stretchr/testify/mock"
)

// ViewChanger is an autogenerated mock type for the ViewChanger type
type ViewChanger struct {
	mock.Mock
}

// ViewChange provides a mock function with given fields: ctx, lfb
func (_m *ViewChanger) ViewChange(ctx context.Context, lfb *block.Block) error {
	ret := _m.Called(ctx, lfb)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, *block.Block) error); ok {
		r0 = rf(ctx, lfb)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
