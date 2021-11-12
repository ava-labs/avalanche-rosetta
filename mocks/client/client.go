// Code generated by mockery 2.9.0. DO NOT EDIT.

package client

import (
	big "math/big"

	client "github.com/ava-labs/avalanche-rosetta/client"
	common "github.com/ethereum/go-ethereum/common"

	context "context"

	interfaces "github.com/ava-labs/coreth/interfaces"

	mock "github.com/stretchr/testify/mock"

	types "github.com/ava-labs/coreth/core/types"
)

// Client is an autogenerated mock type for the Client type
type Client struct {
	mock.Mock
}

// BalanceAt provides a mock function with given fields: _a0, _a1, _a2
func (_m *Client) BalanceAt(_a0 context.Context, _a1 common.Address, _a2 *big.Int) (*big.Int, error) {
	ret := _m.Called(_a0, _a1, _a2)

	var r0 *big.Int
	if rf, ok := ret.Get(0).(func(context.Context, common.Address, *big.Int) *big.Int); ok {
		r0 = rf(_a0, _a1, _a2)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*big.Int)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, common.Address, *big.Int) error); ok {
		r1 = rf(_a0, _a1, _a2)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// BlockByHash provides a mock function with given fields: _a0, _a1
func (_m *Client) BlockByHash(_a0 context.Context, _a1 common.Hash) (*types.Block, error) {
	ret := _m.Called(_a0, _a1)

	var r0 *types.Block
	if rf, ok := ret.Get(0).(func(context.Context, common.Hash) *types.Block); ok {
		r0 = rf(_a0, _a1)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*types.Block)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, common.Hash) error); ok {
		r1 = rf(_a0, _a1)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// BlockByNumber provides a mock function with given fields: _a0, _a1
func (_m *Client) BlockByNumber(_a0 context.Context, _a1 *big.Int) (*types.Block, error) {
	ret := _m.Called(_a0, _a1)

	var r0 *types.Block
	if rf, ok := ret.Get(0).(func(context.Context, *big.Int) *types.Block); ok {
		r0 = rf(_a0, _a1)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*types.Block)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, *big.Int) error); ok {
		r1 = rf(_a0, _a1)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// ChainID provides a mock function with given fields: _a0
func (_m *Client) ChainID(_a0 context.Context) (*big.Int, error) {
	ret := _m.Called(_a0)

	var r0 *big.Int
	if rf, ok := ret.Get(0).(func(context.Context) *big.Int); ok {
		r0 = rf(_a0)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*big.Int)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context) error); ok {
		r1 = rf(_a0)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// EstimateGas provides a mock function with given fields: ctx, msg
func (_m *Client) EstimateGas(ctx context.Context, msg interfaces.CallMsg) (uint64, error) {
	ret := _m.Called(ctx, msg)

	var r0 uint64
	if rf, ok := ret.Get(0).(func(context.Context, interfaces.CallMsg) uint64); ok {
		r0 = rf(ctx, msg)
	} else {
		r0 = ret.Get(0).(uint64)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, interfaces.CallMsg) error); ok {
		r1 = rf(ctx, msg)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// HeaderByHash provides a mock function with given fields: _a0, _a1
func (_m *Client) HeaderByHash(_a0 context.Context, _a1 common.Hash) (*types.Header, error) {
	ret := _m.Called(_a0, _a1)

	var r0 *types.Header
	if rf, ok := ret.Get(0).(func(context.Context, common.Hash) *types.Header); ok {
		r0 = rf(_a0, _a1)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*types.Header)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, common.Hash) error); ok {
		r1 = rf(_a0, _a1)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// HeaderByNumber provides a mock function with given fields: _a0, _a1
func (_m *Client) HeaderByNumber(_a0 context.Context, _a1 *big.Int) (*types.Header, error) {
	ret := _m.Called(_a0, _a1)

	var r0 *types.Header
	if rf, ok := ret.Get(0).(func(context.Context, *big.Int) *types.Header); ok {
		r0 = rf(_a0, _a1)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*types.Header)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, *big.Int) error); ok {
		r1 = rf(_a0, _a1)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// IsBootstrapped provides a mock function with given fields: _a0, _a1
func (_m *Client) IsBootstrapped(_a0 context.Context, _a1 string) (bool, error) {
	ret := _m.Called(_a0, _a1)

	var r0 bool
	if rf, ok := ret.Get(0).(func(context.Context, string) bool); ok {
		r0 = rf(_a0, _a1)
	} else {
		r0 = ret.Get(0).(bool)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, string) error); ok {
		r1 = rf(_a0, _a1)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// NetworkName provides a mock function with given fields: _a0
func (_m *Client) NetworkName(_a0 context.Context) (string, error) {
	ret := _m.Called(_a0)

	var r0 string
	if rf, ok := ret.Get(0).(func(context.Context) string); ok {
		r0 = rf(_a0)
	} else {
		r0 = ret.Get(0).(string)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context) error); ok {
		r1 = rf(_a0)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// NodeVersion provides a mock function with given fields: _a0
func (_m *Client) NodeVersion(_a0 context.Context) (string, error) {
	ret := _m.Called(_a0)

	var r0 string
	if rf, ok := ret.Get(0).(func(context.Context) string); ok {
		r0 = rf(_a0)
	} else {
		r0 = ret.Get(0).(string)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context) error); ok {
		r1 = rf(_a0)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// NonceAt provides a mock function with given fields: _a0, _a1, _a2
func (_m *Client) NonceAt(_a0 context.Context, _a1 common.Address, _a2 *big.Int) (uint64, error) {
	ret := _m.Called(_a0, _a1, _a2)

	var r0 uint64
	if rf, ok := ret.Get(0).(func(context.Context, common.Address, *big.Int) uint64); ok {
		r0 = rf(_a0, _a1, _a2)
	} else {
		r0 = ret.Get(0).(uint64)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, common.Address, *big.Int) error); ok {
		r1 = rf(_a0, _a1, _a2)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// Peers provides a mock function with given fields: _a0
func (_m *Client) Peers(_a0 context.Context) ([]client.Peer, error) {
	ret := _m.Called(_a0)

	var r0 []client.Peer
	if rf, ok := ret.Get(0).(func(context.Context) []client.Peer); ok {
		r0 = rf(_a0)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]client.Peer)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context) error); ok {
		r1 = rf(_a0)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// SendTransaction provides a mock function with given fields: _a0, _a1
func (_m *Client) SendTransaction(_a0 context.Context, _a1 *types.Transaction) error {
	ret := _m.Called(_a0, _a1)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, *types.Transaction) error); ok {
		r0 = rf(_a0, _a1)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// SuggestGasPrice provides a mock function with given fields: _a0
func (_m *Client) SuggestGasPrice(_a0 context.Context) (*big.Int, error) {
	ret := _m.Called(_a0)

	var r0 *big.Int
	if rf, ok := ret.Get(0).(func(context.Context) *big.Int); ok {
		r0 = rf(_a0)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*big.Int)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context) error); ok {
		r1 = rf(_a0)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// TraceTransaction provides a mock function with given fields: _a0, _a1
func (_m *Client) TraceTransaction(_a0 context.Context, _a1 string) (*client.Call, []*client.FlatCall, error) {
	ret := _m.Called(_a0, _a1)

	var r0 *client.Call
	if rf, ok := ret.Get(0).(func(context.Context, string) *client.Call); ok {
		r0 = rf(_a0, _a1)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*client.Call)
		}
	}

	var r1 []*client.FlatCall
	if rf, ok := ret.Get(1).(func(context.Context, string) []*client.FlatCall); ok {
		r1 = rf(_a0, _a1)
	} else {
		if ret.Get(1) != nil {
			r1 = ret.Get(1).([]*client.FlatCall)
		}
	}

	var r2 error
	if rf, ok := ret.Get(2).(func(context.Context, string) error); ok {
		r2 = rf(_a0, _a1)
	} else {
		r2 = ret.Error(2)
	}

	return r0, r1, r2
}

// TransactionByHash provides a mock function with given fields: _a0, _a1
func (_m *Client) TransactionByHash(_a0 context.Context, _a1 common.Hash) (*types.Transaction, bool, error) {
	ret := _m.Called(_a0, _a1)

	var r0 *types.Transaction
	if rf, ok := ret.Get(0).(func(context.Context, common.Hash) *types.Transaction); ok {
		r0 = rf(_a0, _a1)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*types.Transaction)
		}
	}

	var r1 bool
	if rf, ok := ret.Get(1).(func(context.Context, common.Hash) bool); ok {
		r1 = rf(_a0, _a1)
	} else {
		r1 = ret.Get(1).(bool)
	}

	var r2 error
	if rf, ok := ret.Get(2).(func(context.Context, common.Hash) error); ok {
		r2 = rf(_a0, _a1)
	} else {
		r2 = ret.Error(2)
	}

	return r0, r1, r2
}

// TransactionReceipt provides a mock function with given fields: _a0, _a1
func (_m *Client) TransactionReceipt(_a0 context.Context, _a1 common.Hash) (*types.Receipt, error) {
	ret := _m.Called(_a0, _a1)

	var r0 *types.Receipt
	if rf, ok := ret.Get(0).(func(context.Context, common.Hash) *types.Receipt); ok {
		r0 = rf(_a0, _a1)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*types.Receipt)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, common.Hash) error); ok {
		r1 = rf(_a0, _a1)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// TxPoolContent provides a mock function with given fields: _a0
func (_m *Client) TxPoolContent(_a0 context.Context) (*client.TxPoolContent, error) {
	ret := _m.Called(_a0)

	var r0 *client.TxPoolContent
	if rf, ok := ret.Get(0).(func(context.Context) *client.TxPoolContent); ok {
		r0 = rf(_a0)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*client.TxPoolContent)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context) error); ok {
		r1 = rf(_a0)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// TxPoolStatus provides a mock function with given fields: _a0
func (_m *Client) TxPoolStatus(_a0 context.Context) (*client.TxPoolStatus, error) {
	ret := _m.Called(_a0)

	var r0 *client.TxPoolStatus
	if rf, ok := ret.Get(0).(func(context.Context) *client.TxPoolStatus); ok {
		r0 = rf(_a0)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*client.TxPoolStatus)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context) error); ok {
		r1 = rf(_a0)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
func (_m *Client) ContractInfo(_a0 common.Address, _a1 bool) (*client.ContractInfo, error) {
	ret := _m.Called(_a0, _a1)

	var r0 *client.ContractInfo
	if rf, ok := ret.Get(0).(func(common.Address, bool) *client.ContractInfo); ok {
		r0 = rf(_a0, _a1)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*client.ContractInfo)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(common.Address, bool) error); ok {
		r1 = rf(_a0, _a1)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

func (_m *Client) EvmTransferLogs(_a0 context.Context, _a1 common.Hash, _a2 common.Hash) ([]types.Log, error) {
	ret := _m.Called(_a0, _a1, _a2)

	var r0 []types.Log
	if rf, ok := ret.Get(0).(func(context.Context, common.Hash, common.Hash) []types.Log); ok {
		r0 = rf(_a0, _a1, _a2)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]types.Log)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, common.Hash, common.Hash) error); ok {
		r1 = rf(_a0, _a1, _a2)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

func (_m *Client) CallContract(_a0 context.Context, _a1 interfaces.CallMsg, _a2 *big.Int) ([]byte, error) {
	ret := _m.Called(_a0, _a1, _a2)

	var r0 []byte
	if rf, ok := ret.Get(0).(func(context.Context, interfaces.CallMsg, *big.Int) []byte); ok {
		r0 = rf(_a0, _a1, _a2)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]byte)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, interfaces.CallMsg, *big.Int) error); ok {
		r1 = rf(_a0, _a1, _a2)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
