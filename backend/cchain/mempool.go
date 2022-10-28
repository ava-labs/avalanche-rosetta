package cchain

import (
	"context"

	cmapper "github.com/ava-labs/avalanche-rosetta/mapper/cchain"
	"github.com/coinbase/rosetta-sdk-go/types"
)

// Mempool implements the /mempool endpoint
func (b *Backend) Mempool(
	ctx context.Context,
	req *types.NetworkRequest,
) (*types.MempoolResponse, *types.Error) {
	if b.config.IsOfflineMode() {
		return nil, ErrUnavailableOffline
	}

	content, err := b.client.TxPoolContent(ctx)
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
func (b *Backend) MempoolTransaction(
	ctx context.Context,
	req *types.MempoolTransactionRequest,
) (*types.MempoolTransactionResponse, *types.Error) {
	return nil, ErrNotImplemented
}
