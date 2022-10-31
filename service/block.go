package service

import (
	"context"

	"github.com/coinbase/rosetta-sdk-go/server"
	"github.com/coinbase/rosetta-sdk-go/types"

	"github.com/ava-labs/avalanche-rosetta/backend"
	cBackend "github.com/ava-labs/avalanche-rosetta/backend/cchain"
	"github.com/ava-labs/avalanche-rosetta/constants"
)

// BlockBackend represents a backend that implements /block family of apis for a subset of requests
// Endpoint handlers in this file delegates requests to corresponding backends based on the request.
// Each backend implements a ShouldHandleRequest method to determine whether that backend should handle the given request.
//
// P-chain support is implemented in pchain.Backend which implements this interface.
// Eventually, the C-chain block logic implemented in this file should be extracted to its own backend as well.
type BlockBackend interface {
	// ShouldHandleRequest returns whether a given request should be handled by this backend
	ShouldHandleRequest(req interface{}) bool
	// Block implements /block endpoint for this backend
	Block(ctx context.Context, request *types.BlockRequest) (*types.BlockResponse, *types.Error)
	// BlockTransaction implements /block/transaction endpoint for this backend
	BlockTransaction(ctx context.Context, request *types.BlockTransactionRequest) (*types.BlockTransactionResponse, *types.Error)
}

// BlockService implements the /block/* endpoints
type BlockService struct {
	mode          string
	cChainBackend *cBackend.Backend
	pChainBackend BlockBackend
}

// NewBlockService returns a new block servicer
func NewBlockService(
	mode string,
	cChainBackend *cBackend.Backend,
	pChainBackend BlockBackend,
) server.BlockAPIServicer {
	return &BlockService{
		mode:          mode,
		cChainBackend: cChainBackend,
		pChainBackend: pChainBackend,
	}
}

// Block implements the /block endpoint
func (s *BlockService) Block(
	ctx context.Context,
	request *types.BlockRequest,
) (*types.BlockResponse, *types.Error) {
	if s.mode == constants.ModeOffline {
		return nil, backend.ErrUnavailableOffline
	}

	if request.BlockIdentifier == nil {
		return nil, backend.ErrBlockInvalidInput
	}
	if request.BlockIdentifier.Hash == nil && request.BlockIdentifier.Index == nil {
		return nil, backend.ErrBlockInvalidInput
	}

	if s.pChainBackend.ShouldHandleRequest(request) {
		return s.pChainBackend.Block(ctx, request)
	}

	// TODO ABENEGIA: replace with ShouldHandleRequest
	// and return error if it's not even CChain block
	return s.cChainBackend.Block(ctx, request)
}

// BlockTransaction implements the /block/transaction endpoint.
func (s *BlockService) BlockTransaction(
	ctx context.Context,
	request *types.BlockTransactionRequest,
) (*types.BlockTransactionResponse, *types.Error) {
	if s.mode == constants.ModeOffline {
		return nil, backend.ErrUnavailableOffline
	}

	if request.BlockIdentifier == nil {
		return nil, backend.WrapError(backend.ErrInvalidInput, "block identifier is not provided")
	}

	if s.pChainBackend.ShouldHandleRequest(request) {
		return s.pChainBackend.BlockTransaction(ctx, request)
	}

	// TODO ABENEGIA: replace with ShouldHandleRequest
	// and return error if it's not even CChain block
	return s.cChainBackend.BlockTransaction(ctx, request)
}
