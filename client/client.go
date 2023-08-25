package client

import (
	"context"
	"math/big"
	"strings"

	"github.com/ava-labs/avalanche-rosetta/constants"
	"github.com/ava-labs/avalanchego/api/info"
	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/avalanchego/utils/rpc"
	ethtypes "github.com/ava-labs/coreth/core/types"
	"github.com/ava-labs/coreth/interfaces"
	"github.com/ava-labs/coreth/plugin/evm"
	ethcommon "github.com/ethereum/go-ethereum/common"
)

// Interface compliance
var _ Client = &client{}

type Client interface {
	// info.Client methods
	InfoClient

	ChainID(context.Context) (*big.Int, error)
	BlockByHash(context.Context, ethcommon.Hash) (*ethtypes.Block, error)
	BlockByNumber(context.Context, *big.Int) (*ethtypes.Block, error)
	HeaderByHash(context.Context, ethcommon.Hash) (*ethtypes.Header, error)
	HeaderByNumber(context.Context, *big.Int) (*ethtypes.Header, error)
	TransactionByHash(context.Context, ethcommon.Hash) (*ethtypes.Transaction, bool, error)
	TransactionReceipt(context.Context, ethcommon.Hash) (*ethtypes.Receipt, error)
	TraceTransaction(context.Context, string) (*Call, []*FlatCall, error)
	TraceBlockByHash(context.Context, string) ([]*Call, [][]*FlatCall, error)
	SendTransaction(context.Context, *ethtypes.Transaction) error
	BalanceAt(context.Context, ethcommon.Address, *big.Int) (*big.Int, error)
	NonceAt(context.Context, ethcommon.Address, *big.Int) (uint64, error)
	SuggestGasPrice(context.Context) (*big.Int, error)
	EstimateGas(context.Context, interfaces.CallMsg) (uint64, error)
	TxPoolContent(context.Context) (*TxPoolContent, error)
	GetContractInfo(ethcommon.Address, bool) (string, uint8, error)
	CallContract(context.Context, interfaces.CallMsg, *big.Int) ([]byte, error)
	IssueTx(ctx context.Context, txBytes []byte, options ...rpc.Option) (ids.ID, error)
	GetAtomicUTXOs(ctx context.Context, addrs []ids.ShortID, sourceChain string, limit uint32, startAddress ids.ShortID, startUTXOID ids.ID, options ...rpc.Option) ([][]byte, ids.ShortID, ids.ID, error)
	EstimateBaseFee(ctx context.Context) (*big.Int, error)
}

type EvmClient evm.Client

type client struct {
	info.Client
	EvmClient
	*EthClient
	*ContractClient
}

// NewClient returns a new client for Avalanche APIs
func NewClient(ctx context.Context, endpoint string) (Client, error) {
	endpoint = strings.TrimSuffix(endpoint, "/")

	eth, err := NewEthClient(ctx, endpoint)
	if err != nil {
		return nil, err
	}

	return &client{
		Client:         info.NewClient(endpoint),
		EvmClient:      evm.NewClient(endpoint, constants.CChain.String()),
		EthClient:      eth,
		ContractClient: NewContractClient(eth.Client),
	}, nil
}
