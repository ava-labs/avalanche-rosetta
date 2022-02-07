package client

import (
	"context"
	"fmt"

	"github.com/ava-labs/avalanchego/utils/rpc"
	"github.com/ava-labs/coreth/ethclient"
	"github.com/ethereum/go-ethereum/eth/tracers"
)

var (
	tracerTimeout = "180s"
	prefixEth     = "/ext/bc/C/rpc"
)

type EthClient struct {
	ethclient.Client
	rpc         rpc.Requester
	traceConfig *tracers.TraceConfig
}

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
			Tracer:  &jsTracer,
		},
	}, nil
}

func (c *EthClient) TxPoolContent(ctx context.Context) (*TxPoolContent, error) {
	content := &TxPoolContent{}
	err := c.rpc.SendJSONRPCRequest(ctx, prefixEth, "txpool_inspect", nil, content)
	if err != nil {
		content = nil
	}
	return content, err
}

func (c *EthClient) TraceTransaction(ctx context.Context, hash string) (*Call, []*FlatCall, error) {
	var result *Call
	args := []interface{}{hash, c.traceConfig}

	if err := c.rpc.SendJSONRPCRequest(ctx, prefixEth, "debug_traceTransaction", args, &result); err != nil {
		return nil, nil, err
	}

	flattened := result.init()

	return result, flattened, nil
}

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
