// Code generated by mockery v2.20.2. DO NOT EDIT.

package chain

import (
	common "github.com/ava-labs/avalanche-rosetta/service/backend/common"
	mock "github.com/stretchr/testify/mock"

	types "github.com/coinbase/rosetta-sdk-go/types"
)

// TxBuilder is an autogenerated mock type for the TxBuilder type
type TxBuilder struct {
	mock.Mock
}

// BuildTx provides a mock function with given fields: matches, rawMetadata
func (_m *TxBuilder) BuildTx(matches []*types.Operation, rawMetadata map[string]interface{}) (common.AvaxTx, []*types.AccountIdentifier, *types.Error) {
	ret := _m.Called(matches, rawMetadata)

	var r0 common.AvaxTx
	var r1 []*types.AccountIdentifier
	var r2 *types.Error
	if rf, ok := ret.Get(0).(func([]*types.Operation, map[string]interface{}) (common.AvaxTx, []*types.AccountIdentifier, *types.Error)); ok {
		return rf(matches, rawMetadata)
	}
	if rf, ok := ret.Get(0).(func([]*types.Operation, map[string]interface{}) common.AvaxTx); ok {
		r0 = rf(matches, rawMetadata)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(common.AvaxTx)
		}
	}

	if rf, ok := ret.Get(1).(func([]*types.Operation, map[string]interface{}) []*types.AccountIdentifier); ok {
		r1 = rf(matches, rawMetadata)
	} else {
		if ret.Get(1) != nil {
			r1 = ret.Get(1).([]*types.AccountIdentifier)
		}
	}

	if rf, ok := ret.Get(2).(func([]*types.Operation, map[string]interface{}) *types.Error); ok {
		r2 = rf(matches, rawMetadata)
	} else {
		if ret.Get(2) != nil {
			r2 = ret.Get(2).(*types.Error)
		}
	}

	return r0, r1, r2
}

type mockConstructorTestingTNewTxBuilder interface {
	mock.TestingT
	Cleanup(func())
}

// NewTxBuilder creates a new instance of TxBuilder. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func NewTxBuilder(t mockConstructorTestingTNewTxBuilder) *TxBuilder {
	mock := &TxBuilder{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
