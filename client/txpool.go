package client

import "fmt"

type TxPoolClient struct {
	rpc RPC
}

func NewTxPoolClient(endpoint string) *TxPoolClient {
	return &TxPoolClient{
		rpc: NewRPCClient(fmt.Sprintf("%s%s", endpoint, EvmPrefix)),
	}
}

func (c TxPoolClient) Status() (*TxPoolStatus, error) {
	status := &TxPoolStatus{}

	err := c.rpc.Call("txpool_status", nil, status)
	if err != nil {
		status = nil
	}

	return status, err
}

func (c TxPoolClient) Content() (*TxPoolContent, error) {
	content := &TxPoolContent{}

	err := c.rpc.Call("txpool_inspect", nil, content)
	if err != nil {
		content = nil
	}

	return content, err
}
