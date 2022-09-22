// Code generated by mockery v2.12.3. DO NOT EDIT.

package chain

import (
	context "context"

	indexer "github.com/ava-labs/avalanche-rosetta/service/backend/pchain/indexer"
	mock "github.com/stretchr/testify/mock"
)

// Parser is an autogenerated mock type for the Parser type
type Parser struct {
	mock.Mock
}

// GetGenesisBlock provides a mock function with given fields: ctx
func (_m *Parser) GetGenesisBlock(ctx context.Context) (*indexer.ParsedGenesisBlock, error) {
	ret := _m.Called(ctx)

	var r0 *indexer.ParsedGenesisBlock
	if rf, ok := ret.Get(0).(func(context.Context) *indexer.ParsedGenesisBlock); ok {
		r0 = rf(ctx)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*indexer.ParsedGenesisBlock)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context) error); ok {
		r1 = rf(ctx)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetPlatformHeight provides a mock function with given fields: ctx
func (_m *Parser) GetPlatformHeight(ctx context.Context) (uint64, error) {
	ret := _m.Called(ctx)

	var r0 uint64
	if rf, ok := ret.Get(0).(func(context.Context) uint64); ok {
		r0 = rf(ctx)
	} else {
		r0 = ret.Get(0).(uint64)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context) error); ok {
		r1 = rf(ctx)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// ParseBlockAtIndex provides a mock function with given fields: ctx, index
func (_m *Parser) ParseBlockAtIndex(ctx context.Context, index uint64) (*indexer.ParsedBlock, error) {
	ret := _m.Called(ctx, index)

	var r0 *indexer.ParsedBlock
	if rf, ok := ret.Get(0).(func(context.Context, uint64) *indexer.ParsedBlock); ok {
		r0 = rf(ctx, index)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*indexer.ParsedBlock)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, uint64) error); ok {
		r1 = rf(ctx, index)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// ParseBlockWithHash provides a mock function with given fields: ctx, hash
func (_m *Parser) ParseBlockWithHash(ctx context.Context, hash string) (*indexer.ParsedBlock, error) {
	ret := _m.Called(ctx, hash)

	var r0 *indexer.ParsedBlock
	if rf, ok := ret.Get(0).(func(context.Context, string) *indexer.ParsedBlock); ok {
		r0 = rf(ctx, hash)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*indexer.ParsedBlock)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, string) error); ok {
		r1 = rf(ctx, hash)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// ParseCurrentBlock provides a mock function with given fields: ctx
func (_m *Parser) ParseCurrentBlock(ctx context.Context) (*indexer.ParsedBlock, error) {
	ret := _m.Called(ctx)

	var r0 *indexer.ParsedBlock
	if rf, ok := ret.Get(0).(func(context.Context) *indexer.ParsedBlock); ok {
		r0 = rf(ctx)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*indexer.ParsedBlock)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context) error); ok {
		r1 = rf(ctx)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

type NewParserT interface {
	mock.TestingT
	Cleanup(func())
}

// NewParser creates a new instance of Parser. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func NewParser(t NewParserT) *Parser {
	mock := &Parser{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
