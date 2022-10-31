package service

import (
	"context"

	"github.com/coinbase/rosetta-sdk-go/server"
	"github.com/coinbase/rosetta-sdk-go/types"

	"github.com/ava-labs/avalanche-rosetta/backend"
	cBackend "github.com/ava-labs/avalanche-rosetta/backend/cchain"
	"github.com/ava-labs/avalanche-rosetta/constants"
)

// NetworkBackend represents a backend that implements /block family of apis for a subset of requests
// Endpoint handlers in this file delegates requests to corresponding backends based on the request.
// Each backend implements a ShouldHandleRequest method to determine whether that backend should handle the given request.
//
// P-chain support is implemented in pchain.Backend which implements this interface.
// Eventually, the C-chain block logic implemented in this file should be extracted to its own backend as well.
type NetworkBackend interface {
	// ShouldHandleRequest returns whether a given request should be handled by this backend
	ShouldHandleRequest(req interface{}) bool
	// NetworkIdentifier returns the identifier of the network it supports
	NetworkIdentifier() *types.NetworkIdentifier
	// NetworkStatus implements /network/status endpoint for this backend
	NetworkStatus(ctx context.Context, request *types.NetworkRequest) (*types.NetworkStatusResponse, *types.Error)
	// NetworkOptions implements /network/options endpoint for this backend
	NetworkOptions(ctx context.Context, request *types.NetworkRequest) (*types.NetworkOptionsResponse, *types.Error)
}

// NetworkService implements all /network endpoints
type NetworkService struct {
	mode          string
	cChainBackend *cBackend.Backend
	pChainBackend NetworkBackend
}

// NewNetworkService returns a new network servicer
func NewNetworkService(
	mode string,
	cChainBackend *cBackend.Backend,
	pChainBackend NetworkBackend,
) server.NetworkAPIServicer {
	return &NetworkService{
		mode:          mode,
		cChainBackend: cChainBackend,
		pChainBackend: pChainBackend,
	}
}

// NetworkList implements the /network/list endpoint
func (s *NetworkService) NetworkList(
	ctx context.Context,
	request *types.MetadataRequest,
) (*types.NetworkListResponse, *types.Error) {
	return &types.NetworkListResponse{
		NetworkIdentifiers: []*types.NetworkIdentifier{
			s.cChainBackend.NetworkIdentifier(),
			s.pChainBackend.NetworkIdentifier(),
		},
	}, nil
}

// NetworkStatus implements the /network/status endpoint
func (s *NetworkService) NetworkStatus(
	ctx context.Context,
	request *types.NetworkRequest,
) (*types.NetworkStatusResponse, *types.Error) {
	if s.mode == constants.ModeOffline {
		return nil, backend.ErrUnavailableOffline
	}

	if s.pChainBackend.ShouldHandleRequest(request) {
		return s.pChainBackend.NetworkStatus(ctx, request)
	}

	return s.cChainBackend.NetworkStatus(ctx, request)
}

// NetworkOptions implements the /network/options endpoint
func (s *NetworkService) NetworkOptions(
	ctx context.Context,
	request *types.NetworkRequest,
) (*types.NetworkOptionsResponse, *types.Error) {
	if s.pChainBackend.ShouldHandleRequest(request) {
		return s.pChainBackend.NetworkOptions(ctx, request)
	}

	return s.cChainBackend.NetworkOptions(ctx, request)
}
