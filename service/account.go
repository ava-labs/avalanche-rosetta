package service

import (
	"context"

	"github.com/coinbase/rosetta-sdk-go/server"
	"github.com/coinbase/rosetta-sdk-go/types"

	"github.com/ava-labs/avalanche-rosetta/backend"
	cBackend "github.com/ava-labs/avalanche-rosetta/backend/cchain"
	"github.com/ava-labs/avalanche-rosetta/constants"
)

// AccountBackend represents a backend that implements /account family of apis for a subset of requests.
// Endpoint handlers in this file delegates requests to corresponding backends based on the request.
// Each backend implements a ShouldHandleRequest method to determine whether that backend should handle the given request.
//
// P-chain and C-chain atomic transaction logic are implemented in pchain.Backend and cchainatomictx.Backend respectively.
// Eventually, the C-chain non-atomic transaction logic implemented in this file should be extracted to its own backend as well.
type AccountBackend interface {
	// ShouldHandleRequest returns whether a given request should be handled by this backend
	ShouldHandleRequest(req interface{}) bool
	// AccountBalance implements /account/balance endpoint for this backend
	AccountBalance(ctx context.Context, req *types.AccountBalanceRequest) (*types.AccountBalanceResponse, *types.Error)
	// AccountCoins implements /account/coins endpoint for this backend
	AccountCoins(ctx context.Context, req *types.AccountCoinsRequest) (*types.AccountCoinsResponse, *types.Error)
}

// AccountService implements the /account/* endpoints
type AccountService struct {
	mode                  string
	cChainBackend         *cBackend.Backend
	cChainAtomicTxBackend AccountBackend
	pChainBackend         AccountBackend
}

// NewAccountService returns a new network servicer
func NewAccountService(
	mode string,
	cChainBackend *cBackend.Backend,
	pChainBackend AccountBackend,
	cChainAtomicTxBackend AccountBackend,
) server.AccountAPIServicer {
	return &AccountService{
		mode:                  mode,
		cChainBackend:         cChainBackend,
		cChainAtomicTxBackend: cChainAtomicTxBackend,
		pChainBackend:         pChainBackend,
	}
}

// AccountBalance implements the /account/balance endpoint
func (s AccountService) AccountBalance(
	ctx context.Context,
	req *types.AccountBalanceRequest,
) (*types.AccountBalanceResponse, *types.Error) {
	if s.mode == constants.ModeOffline {
		return nil, backend.ErrUnavailableOffline
	}

	if req.AccountIdentifier == nil {
		return nil, backend.WrapError(backend.ErrInvalidInput, "account identifier is not provided")
	}

	if s.pChainBackend.ShouldHandleRequest(req) {
		return s.pChainBackend.AccountBalance(ctx, req)
	}

	// If the address is in Bech32 format, we check the atomic balance
	if s.cChainAtomicTxBackend.ShouldHandleRequest(req) {
		return s.cChainAtomicTxBackend.AccountBalance(ctx, req)
	}

	// TODO ABENEGIA: replace with ShouldHandleRequest
	// and return error if it's not even CChain block
	return s.cChainBackend.AccountBalance(ctx, req)
}

// AccountCoins implements the /account/coins endpoint
func (s AccountService) AccountCoins(
	ctx context.Context,
	req *types.AccountCoinsRequest,
) (*types.AccountCoinsResponse, *types.Error) {
	if s.mode == constants.ModeOffline {
		return nil, backend.ErrUnavailableOffline
	}

	if s.pChainBackend.ShouldHandleRequest(req) {
		return s.pChainBackend.AccountCoins(ctx, req)
	}

	if s.cChainAtomicTxBackend.ShouldHandleRequest(req) {
		return s.cChainAtomicTxBackend.AccountCoins(ctx, req)
	}

	return s.cChainBackend.AccountCoins(ctx, req)
}
