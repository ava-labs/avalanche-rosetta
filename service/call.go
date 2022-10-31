package service

import (
	"context"

	cBackend "github.com/ava-labs/avalanche-rosetta/backend/cchain"
	"github.com/coinbase/rosetta-sdk-go/server"
	"github.com/coinbase/rosetta-sdk-go/types"
)

// CallService implements /call/* endpoints
type CallService struct {
	cChainBackend *cBackend.Backend
}

// GetTransactionReceiptInput is the input to the call
// method "eth_getTransactionReceipt".
type GetTransactionReceiptInput struct {
	TxHash string `json:"tx_hash"`
}

// NewCallService returns a new call servicer
func NewCallService(cChainBackend *cBackend.Backend) server.CallAPIServicer {
	return &CallService{cChainBackend: cChainBackend}
}

// Call implements the /call endpoint.
func (s CallService) Call(
	ctx context.Context,
	req *types.CallRequest,
) (*types.CallResponse, *types.Error) {
	return s.cChainBackend.Call(ctx, req)
}
