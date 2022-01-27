package client

import (
	"context"
	"fmt"

	"github.com/ava-labs/avalanchego/utils/rpc"
	"github.com/ava-labs/coreth/eth/tracers"
	"github.com/ava-labs/coreth/ethclient"
)

var (
	tracer    = "callTracer"
	prefixEth = "/ext/bc/C/rpc"
)

// EthClient provides access to Coreth API
type EthClient struct {
	ethclient.Client
	rpc         rpc.Requester
	traceConfig *tracers.TraceConfig
}

// NewEthClient returns a new EVM client
func NewEthClient(endpoint string) (*EthClient, error) {
	endpointURL := fmt.Sprintf("%s%s", endpoint, prefixEth)

	c, err := ethclient.Dial(endpointURL)
	if err != nil {
		return nil, err
	}

	return &EthClient{
		Client: c,
		rpc:    rpc.NewRPCRequester(endpoint),
		traceConfig: &tracers.TraceConfig{
			Timeout: &tracerTimeout,
			Tracer:  &tracer,
		},
	}, nil
}

// TxPoolContent returns the tx pool content
func (c *EthClient) TxPoolContent(ctx context.Context) (*TxPoolContent, error) {
	content := &TxPoolContent{}
	err := c.rpc.SendJSONRPCRequest(ctx, prefixEth, "txpool_inspect", nil, content)
	if err != nil {
		content = nil
	}
	return content, err
}

// TraceTransaction returns a transaction trace
func (c *EthClient) TraceTransaction(ctx context.Context, hash string) (*Call, []*FlatCall, error) {
	var result Call
	args := []interface{}{hash, c.traceConfig}

	if err := c.rpc.SendJSONRPCRequest(ctx, prefixEth, "debug_traceTransaction", args, &result); err != nil {
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

	args := []interface{}{hash, c.traceConfig}
	if err := c.rpc.SendJSONRPCRequest(ctx, prefixEth, "debug_traceBlockByHash", args, &raw); err != nil {
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
