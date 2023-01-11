package service

import (
	"context"

	"github.com/coinbase/rosetta-sdk-go/server"
	"github.com/coinbase/rosetta-sdk-go/types"

	"github.com/chain4travel/camino-rosetta/client"
	"github.com/chain4travel/camino-rosetta/mapper"
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
		return nil, errUnavailableOffline
	}

	content, err := s.client.TxPoolContent(ctx)
	if err != nil {
		return nil, wrapError(errClientError, err)
	}

	return &types.MempoolResponse{
		TransactionIdentifiers: append(
			mapper.MempoolTransactionsIDs(content.Pending),
			mapper.MempoolTransactionsIDs(content.Queued)...,
		),
	}, nil
}

// MempoolTransaction implements the /mempool/transaction endpoint
func (s MempoolService) MempoolTransaction(
	ctx context.Context,
	req *types.MempoolTransactionRequest,
) (*types.MempoolTransactionResponse, *types.Error) {
	return nil, errNotImplemented
}
