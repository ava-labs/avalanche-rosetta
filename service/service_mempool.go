package service

import (
	"context"

	"github.com/coinbase/rosetta-sdk-go/server"
	"github.com/coinbase/rosetta-sdk-go/types"

	"github.com/figment-networks/avalanche-rosetta/client"
	"github.com/figment-networks/avalanche-rosetta/mapper"
)

// MempoolService implements the /mempool/* endpoints
type MempoolService struct {
	config *Config
	evm    *client.EvmClient
	txpool *client.TxPoolClient
}

// NewMempoolService returns a new mempool servicer
func NewMempoolService(config *Config, evmClient *client.EvmClient, txpoolClient *client.TxPoolClient) server.MempoolAPIServicer {
	return &MempoolService{
		config: config,
		evm:    evmClient,
		txpool: txpoolClient,
	}
}

// Mempool implements the /mempool endpoint
func (s MempoolService) Mempool(ctx context.Context, req *types.NetworkRequest) (*types.MempoolResponse, *types.Error) {
	if s.config.IsOfflineMode() {
		return nil, errUnavailableOffline
	}

	content, err := s.txpool.Content()
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
func (s MempoolService) MempoolTransaction(ctx context.Context, req *types.MempoolTransactionRequest) (*types.MempoolTransactionResponse, *types.Error) {
	return nil, errNotImplemented
}
