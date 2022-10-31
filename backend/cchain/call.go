package cchain

import (
	"context"
	"encoding/json"

	"github.com/ava-labs/avalanche-rosetta/backend"
	"github.com/coinbase/rosetta-sdk-go/types"
	"github.com/ethereum/go-ethereum/common"
)

// GetTransactionReceiptInput is the input to the call
// method "eth_getTransactionReceipt".
type GetTransactionReceiptInput struct {
	TxHash string `json:"tx_hash"`
}

// Call implements the /call endpoint.
func (b *Backend) Call(
	ctx context.Context,
	req *types.CallRequest,
) (*types.CallResponse, *types.Error) {
	if b.config.IsOfflineMode() {
		return nil, backend.ErrUnavailableOffline
	}

	switch req.Method {
	case "eth_getTransactionReceipt":
		return b.callGetTransactionReceipt(ctx, req)
	default:
		return nil, backend.ErrCallInvalidMethod
	}
}

func (b *Backend) callGetTransactionReceipt(
	ctx context.Context,
	req *types.CallRequest,
) (*types.CallResponse, *types.Error) {
	var input GetTransactionReceiptInput
	if err := types.UnmarshalMap(req.Parameters, &input); err != nil {
		return nil, backend.WrapError(backend.ErrCallInvalidParams, err)
	}

	if len(input.TxHash) == 0 {
		return nil, backend.WrapError(backend.ErrCallInvalidParams, "tx_hash missing from params")
	}

	receipt, err := b.client.TransactionReceipt(ctx, common.HexToHash(input.TxHash))
	if err != nil {
		return nil, backend.WrapError(backend.ErrClientError, err)
	}

	jsonOutput, err := receipt.MarshalJSON()
	if err != nil {
		return nil, backend.WrapError(backend.ErrInternalError, err)
	}

	var receiptMap map[string]interface{}
	if err := json.Unmarshal(jsonOutput, &receiptMap); err != nil {
		return nil, backend.WrapError(backend.ErrInternalError, err)
	}

	return &types.CallResponse{Result: receiptMap}, nil
}
