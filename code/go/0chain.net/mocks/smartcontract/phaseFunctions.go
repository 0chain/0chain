// Code generated by mockery v0.0.0-dev. DO NOT EDIT.

package mocks

import (
	minersc "0chain.net/smartcontract/minersc"
	mock "github.com/stretchr/testify/mock"

	state "0chain.net/chaincore/chain/state"
)

// phaseFunctions is an autogenerated mock type for the phaseFunctions type
type phaseFunctions struct {
	mock.Mock
}

// Execute provides a mock function with given fields: balances, gn
func (_m *phaseFunctions) Execute(balances state.StateContextI, gn *minersc.GlobalNode) error {
	ret := _m.Called(balances, gn)

	var r0 error
	if rf, ok := ret.Get(0).(func(state.StateContextI, *minersc.GlobalNode) error); ok {
		r0 = rf(balances, gn)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
