package service

import (
	"context"

	cBackend "github.com/ava-labs/avalanche-rosetta/backend/cchain"
	"github.com/coinbase/rosetta-sdk-go/server"
	"github.com/coinbase/rosetta-sdk-go/types"
)

// MempoolService implements the /mempool/* endpoints
type MempoolService struct {
	mode          string
	cChainBackend *cBackend.Backend
}

// NewMempoolService returns a new mempool servicer
func NewMempoolService(mode string, cChainBackend *cBackend.Backend) server.MempoolAPIServicer {
	return &MempoolService{
		mode:          mode,
		cChainBackend: cChainBackend,
	}
}

// Mempool implements the /mempool endpoint
func (s MempoolService) Mempool(
	ctx context.Context,
	req *types.NetworkRequest,
) (*types.MempoolResponse, *types.Error) {
	if s.mode == ModeOffline {
		return nil, ErrUnavailableOffline
	}

	// TODO ABENEGIA: use ShouldHandleRequest for p, c and x chains
	// and return error if it's not even CChain block

	return s.cChainBackend.Mempool(ctx, req)
}

// MempoolTransaction implements the /mempool/transaction endpoint
func (s MempoolService) MempoolTransaction(
	ctx context.Context,
	req *types.MempoolTransactionRequest,
) (*types.MempoolTransactionResponse, *types.Error) {
	// TODO ABENEGIA: use ShouldHandleRequest for p, c and x chains
	// and return error if it's not even CChain block

	return s.cChainBackend.MempoolTransaction(ctx, req)
}
