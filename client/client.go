package client

import (
	"context"
	"math/big"

	"github.com/ava-labs/avalanchego/api/info"
	"github.com/ava-labs/avalanchego/network"
	"github.com/ava-labs/coreth/core/types"
	ethtypes "github.com/ava-labs/coreth/core/types"
	"github.com/ava-labs/coreth/interfaces"
	ethcommon "github.com/ethereum/go-ethereum/common"
)

// Interface compliance
var _ Client = &client{}

const prefixEth = "/ext/bc/C/rpc"

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
	EvmTransferLogs(ctx context.Context, blockHash ethcommon.Hash, transactionHash ethcommon.Hash) ([]types.Log, error)
	SendTransaction(context.Context, *ethtypes.Transaction) error
	BalanceAt(context.Context, ethcommon.Address, *big.Int) (*big.Int, error)
	NonceAt(context.Context, ethcommon.Address, *big.Int) (uint64, error)
	SuggestGasPrice(context.Context) (*big.Int, error)
	EstimateGas(ctx context.Context, msg interfaces.CallMsg) (uint64, error)
	TxPoolStatus(context.Context) (*TxPoolStatus, error)
	TxPoolContent(context.Context) (*TxPoolContent, error)
	GetNetworkName(context.Context) (string, error)
	Peers(context.Context) ([]network.PeerInfo, error)
	ContractInfo(contractAddress ethcommon.Address, isErc20 bool) (*ContractInfo, error)
	CallContract(ctx context.Context, msg interfaces.CallMsg, blockNumber *big.Int) ([]byte, error)
}

type client struct {
	info.Client
	*EthClient
	*EvmLogsClient
	*ContractClient
}

// NewClient returns a new client for Avalanche APIs
func NewClient(endpoint string) (Client, error) {
	eth, err := NewEthClient(endpoint)
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
		Client:         info.NewClient(endpoint),
		EthClient:      eth,
		EvmLogsClient:  evmlogs,
		ContractClient: contract,
	}, nil
}
