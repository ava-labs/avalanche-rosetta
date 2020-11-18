package client

import (
	"fmt"

	"github.com/ethereum/go-ethereum/eth"
)

var (
	tracerTimeout = "30s"
)

type DebugClient struct {
	rpc         RPC
	traceConfig *eth.TraceConfig
}

func NewDebugClient(endpoint string) *DebugClient {
	return &DebugClient{
		rpc: NewRPCClient(fmt.Sprintf("%s%s", endpoint, PrefixEVM)),
		traceConfig: &eth.TraceConfig{
			Timeout: &tracerTimeout,
			Tracer:  &jsTracer,
		},
	}
}

func (c DebugClient) TraceTransaction(hash string) (*Call, error) {
	result := &Call{}
	args := []interface{}{hash, c.traceConfig}

	err := c.rpc.Call("debug_traceTransaction", args, &result)
	if err != nil {
		return nil, err
	}

	return result, nil
}
