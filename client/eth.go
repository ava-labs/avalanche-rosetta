package client

import (
	"context"

	"github.com/ava-labs/avalanchego/utils/rpc"
	"github.com/ava-labs/coreth/ethclient"
	"github.com/ethereum/go-ethereum/eth/tracers"
)

var (
	tracerTimeout = "180s"
)

// EthClient provides access to Coreth API
type EthClient struct {
	ethclient.Client
	rpc         rpc.Requester
	traceConfig *tracers.TraceConfig
}

// NewEthClient returns a new EVM client
func NewEthClient(endpointURL string) (*EthClient, error) {
	c, err := ethclient.Dial(endpointURL)
	if err != nil {
		return nil, err
	}

	return &EthClient{
		Client: c,
		rpc:    rpc.NewRPCRequester(endpointURL),
		traceConfig: &tracers.TraceConfig{
			Timeout: &tracerTimeout,
			Tracer:  &jsTracer,
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
	result := &Call{}
	args := []interface{}{hash, c.traceConfig}

	err := c.rpc.SendJSONRPCRequest(ctx, prefixEth, "debug_traceTransaction", args, &result)
	if err != nil {
		return nil, nil, err
	}
	flattened := result.init()
	return result, flattened, nil
}
