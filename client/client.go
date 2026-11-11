package client

import (
	"context"
	"fmt"
	"math/big"
	"strings"

	"github.com/ava-labs/avalanchego/api/info"
	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/avalanchego/utils/rpc"
	"github.com/ava-labs/avalanchego/vms/components/avax"
	"github.com/ava-labs/avalanchego/vms/saevm/cchain"
	"github.com/ava-labs/avalanchego/vms/saevm/cchain/tx"
	ethereum "github.com/ava-labs/libevm"
	"github.com/ava-labs/libevm/common"
	"github.com/ava-labs/libevm/core/types"
)

// Interface compliance
var _ Client = &client{}

type Client interface {
	// info.Client methods
	InfoClient

	ChainID(context.Context) (*big.Int, error)
	BlockByHash(context.Context, common.Hash) (*types.Block, error)
	BlockByNumber(context.Context, *big.Int) (*types.Block, error)
	HeaderByHash(context.Context, common.Hash) (*types.Header, error)
	HeaderByNumber(context.Context, *big.Int) (*types.Header, error)
	TransactionByHash(context.Context, common.Hash) (*types.Transaction, bool, error)
	TransactionReceipt(context.Context, common.Hash) (*types.Receipt, error)
	TraceTransaction(context.Context, string) (*Call, []*FlatCall, error)
	TraceBlockByHash(context.Context, string) ([]*Call, [][]*FlatCall, error)
	SendTransaction(context.Context, *types.Transaction) error
	BalanceAt(context.Context, common.Address, *big.Int) (*big.Int, error)
	NonceAt(context.Context, common.Address, *big.Int) (uint64, error)
	SuggestGasPrice(context.Context) (*big.Int, error)
	EstimateGas(context.Context, ethereum.CallMsg) (uint64, error)
	TxPoolContent(context.Context) (*TxPoolContent, error)
	GetContractInfo(common.Address, bool) (string, uint8, error)
	CallContract(context.Context, ethereum.CallMsg, *big.Int) ([]byte, error)
	IssueTx(ctx context.Context, txBytes []byte, options ...rpc.Option) (ids.ID, error)
	GetAtomicUTXOs(ctx context.Context, addrs []ids.ShortID, sourceChain string, limit uint32, startAddress ids.ShortID, startUTXOID ids.ID, options ...rpc.Option) ([]*avax.UTXO, ids.ShortID, ids.ID, error)
	EstimateBaseFee(ctx context.Context) (*big.Int, error)
}

// EvmClient calls the C-Chain's `avax` API.
type EvmClient struct{ *cchain.Client }

// IssueTx issues the serialized C-Chain atomic transaction `txBytes` and returns its ID.
func (c EvmClient) IssueTx(ctx context.Context, txBytes []byte, options ...rpc.Option) (ids.ID, error) {
	t, err := tx.Parse(txBytes)
	if err != nil {
		return ids.Empty, fmt.Errorf("parsing atomic tx: %w", err)
	}
	if err := c.Client.IssueTx(ctx, t, options...); err != nil {
		return ids.Empty, err
	}
	return t.ID(), nil
}

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
		Client:         *info.NewClient(endpoint),
		EvmClient:      EvmClient{cchain.NewClient(endpoint)},
		EthClient:      eth,
		ContractClient: NewContractClient(eth.Client.Eth),
	}, nil
}

// GetAtomicUTXOs returns the UTXOs controlled by `addrs` that were exported to the C-Chain from
// the chain with alias `sourceChain`.
func (c *client) GetAtomicUTXOs(
	ctx context.Context,
	addrs []ids.ShortID,
	sourceChain string,
	limit uint32,
	startAddress ids.ShortID,
	startUTXOID ids.ID,
	options ...rpc.Option,
) ([]*avax.UTXO, ids.ShortID, ids.ID, error) {
	chainID, err := c.Client.GetBlockchainID(ctx, sourceChain, options...)
	if err != nil {
		return nil, ids.ShortID{}, ids.Empty, fmt.Errorf("resolving chain alias %q: %w", sourceChain, err)
	}
	return c.EvmClient.Client.GetUTXOs(ctx, addrs, chainID, limit, startAddress, startUTXOID, options...)
}
