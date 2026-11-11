package client

import (
	"context"
	"math/big"

	"github.com/ava-labs/avalanchego/vms/evm/evmclient"
	"github.com/ava-labs/libevm/eth/tracers"
)

var (
	tracer        = "callTracer"
	tracerTimeout = "180s"
	prefixEth     = "/ext/bc/C/rpc"
)

// EthClient provides access to Coreth API
type EthClient struct {
	*evmclient.Client
	traceConfig *tracers.TraceConfig
}

// NewEthClient returns a new EVM client
func NewEthClient(ctx context.Context, endpoint string) (*EthClient, error) {
	endpointURL := endpoint + prefixEth

	c, err := evmclient.Dial(ctx, endpointURL)
	if err != nil {
		return nil, err
	}

	return &EthClient{
		Client: c,
		traceConfig: &tracers.TraceConfig{
			Timeout: &tracerTimeout,
			Tracer:  &tracer,
		},
	}, nil
}

// TxPoolContent returns the tx pool content
func (c *EthClient) TxPoolContent(ctx context.Context) (*TxPoolContent, error) {
	var content TxPoolContent

	err := c.RPC.CallContext(ctx, &content, "txpool_inspect")

	return &content, err
}

// TraceTransaction returns a transaction trace
func (c *EthClient) TraceTransaction(ctx context.Context, hash string) (*Call, []*FlatCall, error) {
	var result Call

	err := c.RPC.CallContext(ctx, &result, "debug_traceTransaction", hash, c.traceConfig)
	if err != nil {
		return nil, nil, err
	}

	flattened := result.init()

	return &result, flattened, nil
}

// TraceBlockByHash returns the transaction traces of all transactions in the block
func (c *EthClient) TraceBlockByHash(ctx context.Context, hash string) ([]*Call, [][]*FlatCall, error) {
	var raw []struct {
		*Call `json:"result"`
	}

	err := c.RPC.CallContext(ctx, &raw, "debug_traceBlockByHash", hash, c.traceConfig)
	if err != nil {
		return nil, nil, err
	}

	result := make([]*Call, len(raw))
	flattened := make([][]*FlatCall, len(raw))
	for i, tx := range raw {
		result[i] = tx.Call
		flattened[i] = tx.Call.init()
	}

	return result, flattened, nil
}

// EstimateBaseFee returns the node's estimate of the base fee of the next block.
func (c *EthClient) EstimateBaseFee(ctx context.Context) (*big.Int, error) {
	baseFee, err := c.Client.EstimateBaseFee(ctx)
	if err != nil {
		return nil, err
	}
	return new(big.Int).SetUint64(uint64(baseFee)), nil
}
