package client

import (
	"context"
	"math/big"

	ethtypes "github.com/ava-labs/coreth/core/types"
	"github.com/ava-labs/coreth/interfaces"
	"github.com/coinbase/rosetta-sdk-go/types"
	ethcommon "github.com/ethereum/go-ethereum/common"
)

const (
	prefixInfo = "/ext/info"
	prefixEth  = "/ext/bc/C/rpc"
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
	TraceTransaction(context.Context, string) (*Call, []*FlatCall, error)
	EvmTransferLogs(context.Context, ethcommon.Hash, ethcommon.Hash) ([]ethtypes.Log, error)
	SendTransaction(context.Context, *ethtypes.Transaction) error
	BalanceAt(context.Context, ethcommon.Address, *big.Int) (*big.Int, error)
	NonceAt(context.Context, ethcommon.Address, *big.Int) (uint64, error)
	SuggestGasPrice(context.Context) (*big.Int, error)
	EstimateGas(context.Context, interfaces.CallMsg) (uint64, error)
	TxPoolStatus(context.Context) (*TxPoolStatus, error)
	TxPoolContent(context.Context) (*TxPoolContent, error)
	NetworkName(context.Context) (string, error)
	Peers(context.Context) ([]Peer, error)
	NodeVersion(context.Context) (string, error)
	ContractCurrency(ethcommon.Address, bool) (*types.Currency, error)
	CallContract(context.Context, interfaces.CallMsg, *big.Int) ([]byte, error)
}

type client struct {
	*EthClient
	*InfoClient
	*EvmLogsClient
	*ContractClient
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

	evmlogs, err := NewEvmLogsClient(endpoint)
	if err != nil {
		return nil, err
	}

	contract, err := NewContractClient(endpoint)
	if err != nil {
		return nil, err
	}

	return client{
		EthClient:      eth,
		InfoClient:     info,
		EvmLogsClient:  evmlogs,
		ContractClient: contract,
	}, nil
}
