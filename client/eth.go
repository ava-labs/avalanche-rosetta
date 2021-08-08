package client

import (
	"context"
	"fmt"
	"strings"

	"github.com/ava-labs/coreth/ethclient"
	"github.com/ethereum/go-ethereum/eth/tracers"
)

var (
	tracerTimeout = "180s"
)

// EthClient provides access to Coreth API
type EthClient struct {
	*ethclient.Client
	rpc         *RPC
	traceConfig *tracers.TraceConfig
}

// NewEthClient returns a new EVM client
func NewEthClient(endpoint string) (*EthClient, error) {
	endpoint = strings.TrimSuffix(endpoint, "/")

	c, err := ethclient.Dial(fmt.Sprintf("%s%s", endpoint, prefixEth))
	if err != nil {
		return nil, err
	}
	raw := Dial(fmt.Sprintf("%s%s", endpoint, prefixEth))

	return &EthClient{
		Client: c,
		rpc:    raw,
		traceConfig: &tracers.TraceConfig{
			Timeout: &tracerTimeout,
			Tracer:  &jsTracer,
		},
	}, nil
}

// TxPoolStatus return the current tx pool status
func (c *EthClient) TxPoolStatus(ctx context.Context) (*TxPoolStatus, error) {
	status := &TxPoolStatus{}
	err := c.rpc.Call(ctx, "txpool_status", nil, status)
	if err != nil {
		status = nil
	}
	return status, err
}

// TxPoolContent returns the tx pool content
func (c *EthClient) TxPoolContent(ctx context.Context) (*TxPoolContent, error) {
	content := &TxPoolContent{}
	err := c.rpc.Call(ctx, "txpool_inspect", nil, content)
	if err != nil {
		content = nil
	}
	return content, err
}

// TraceTransaction returns a transaction trace
func (c *EthClient) TraceTransaction(ctx context.Context, hash string) (*Call, []*FlatCall, error) {
	result := &Call{}
	args := []interface{}{hash, c.traceConfig}

	err := c.rpc.Call(context.Background(), "debug_traceTransaction", args, &result)
	if err != nil {
		return nil, nil, err
	}
	flattened := result.init()
	return result, flattened, nil
}
