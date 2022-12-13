package service

import (
	"context"

	"github.com/ava-labs/avalanche-rosetta/client"
	cmapper "github.com/ava-labs/avalanche-rosetta/mapper/cchain"
	"github.com/coinbase/rosetta-sdk-go/server"
	"github.com/coinbase/rosetta-sdk-go/types"
)

// MempoolService implements the /mempool/* endpoints
type MempoolService struct {
	config *Config
	client client.Client
}

// NewMempoolService returns a new mempool servicer
func NewMempoolService(config *Config, client client.Client) server.MempoolAPIServicer {
	return &MempoolService{
		config: config,
		client: client,
	}
}

// Mempool implements the /mempool endpoint
func (s MempoolService) Mempool(
	ctx context.Context,
	req *types.NetworkRequest,
) (*types.MempoolResponse, *types.Error) {
	if s.config.IsOfflineMode() {
		return nil, ErrUnavailableOffline
	}

	content, err := s.client.TxPoolContent(ctx)
	if err != nil {
		return nil, WrapError(ErrClientError, err)
	}

	return &types.MempoolResponse{
		TransactionIdentifiers: append(
			cmapper.MempoolTransactionsIDs(content.Pending),
			cmapper.MempoolTransactionsIDs(content.Queued)...,
		),
	}, nil
}

// MempoolTransaction implements the /mempool/transaction endpoint
func (s MempoolService) MempoolTransaction(
	ctx context.Context,
	req *types.MempoolTransactionRequest,
) (*types.MempoolTransactionResponse, *types.Error) {
	return nil, ErrNotImplemented
}
