package client

import (
	"errors"
	"fmt"
)

type DebugClient struct {
	rpc RPC
}

func NewDebugClient(endpoint string) *DebugClient {
	return &DebugClient{
		rpc: NewRPCClient(fmt.Sprintf("%s%s", endpoint, PrefixEVM)),
	}
}

func (c DebugClient) TraceTransaction(hash string) error {
	return errors.New("debug rpc trace call is not supported yet")
}
