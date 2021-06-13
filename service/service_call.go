package service

import (
	"context"
	"encoding/json"

	"github.com/coinbase/rosetta-sdk-go/server"
	"github.com/coinbase/rosetta-sdk-go/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/figment-networks/avalanche-rosetta/client"
)

// CallService implements /call/* endpoints
type CallService struct {
	config *Config
	client client.Client
}

// GetTransactionReceiptInput is the input to the call
// method "eth_getTransactionReceipt".
type GetTransactionReceiptInput struct {
	TxHash string `json:"tx_hash"`
}

// NewCallService returns a new call servicer
func NewCallService(config *Config, client client.Client) server.CallAPIServicer {
	return &CallService{
		config: config,
		client: client,
	}
}

// Call implements the /call endpoint.
func (s CallService) Call(
	ctx context.Context,
	req *types.CallRequest,
) (*types.CallResponse, *types.Error) {
	if s.config.IsOfflineMode() {
		return nil, errUnavailableOffline
	}

	switch req.Method {
	case "eth_getTransactionReceipt":
		return s.callGetTransactionReceipt(ctx, req)
	default:
		return nil, errCallInvalidMethod
	}
}

func (s CallService) callGetTransactionReceipt(
	ctx context.Context,
	req *types.CallRequest,
) (*types.CallResponse, *types.Error) {
	var input GetTransactionReceiptInput
	if err := types.UnmarshalMap(req.Parameters, &input); err != nil {
		return nil, wrapError(errCallInvalidParams, err)
	}

	if len(input.TxHash) == 0 {
		return nil, wrapError(errCallInvalidParams, "tx_hash missing from params")
	}

	receipt, err := s.client.TransactionReceipt(ctx, common.HexToHash(input.TxHash))
	if err != nil {
		return nil, wrapError(errClientError, err)
	}

	jsonOutput, err := receipt.MarshalJSON()
	if err != nil {
		return nil, wrapError(errInternalError, err)
	}

	var receiptMap map[string]interface{}
	if err := json.Unmarshal(jsonOutput, &receiptMap); err != nil {
		return nil, wrapError(errInternalError, err)
	}

	return &types.CallResponse{Result: receiptMap}, nil
}
