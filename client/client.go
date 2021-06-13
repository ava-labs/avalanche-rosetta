package client

import (
	"context"
	"math/big"

	ethtypes "github.com/ava-labs/coreth/core/types"
	ethcommon "github.com/ethereum/go-ethereum/common"
)

const (
	prefixInfo = "/ext/info"
	prefixEth  = "/ext/bc/C/rpc"
	prefixAvm  = "/ext/bc/X"
)

type Client interface {
	IsBootstrapped(context.Context, string) (bool, error)
	ChainID(context.Context) (*big.Int, error)
	BlockByHash(context.Context, ethcommon.Hash) (*ethtypes.Block, error)
	BlockByNumber(context.Context, *big.Int) (*ethtypes.Block, error)
	HeaderByHash(context.Context, ethcommon.Hash) (*ethtypes.Header, error)
	HeaderByNumber(context.Context, *big.Int) (*ethtypes.Header, error)
	TransactionByHash(context.Context, ethcommon.Hash) (*ethtypes.Transaction, bool, error)
	TransactionReceipt(context.Context, ethcommon.Hash) (*ethtypes.Receipt, error)
	TraceTransaction(context.Context, string) (*Call, error)
	SendTransaction(context.Context, *ethtypes.Transaction) error
	BalanceAt(context.Context, ethcommon.Address, *big.Int) (*big.Int, error)
	NonceAt(context.Context, ethcommon.Address, *big.Int) (uint64, error)
	SuggestGasPrice(context.Context) (*big.Int, error)
	TxPoolStatus(context.Context) (*TxPoolStatus, error)
	TxPoolContent(context.Context) (*TxPoolContent, error)
	NetworkName(context.Context) (string, error)
	Peers(context.Context) ([]Peer, error)
	NodeVersion(context.Context) (string, error)
}

type client struct {
	*EthClient
	*InfoClient
}

// NewClient returns a new client for Avalanche APIs
func NewClient(endpoint string) (Client, error) {
	eth, err := NewEthClient(endpoint)
	if err != nil {
		return nil, err
	}

	info, err := NewInfoClient(endpoint)
	if err != nil {
		return nil, err
	}

	return client{
		EthClient:  eth,
		InfoClient: info,
	}, nil
}
