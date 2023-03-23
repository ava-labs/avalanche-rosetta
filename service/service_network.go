package service

import (
	"context"
	"math/big"

	"github.com/coinbase/rosetta-sdk-go/server"
	"github.com/coinbase/rosetta-sdk-go/types"
	"github.com/coinbase/rosetta-sdk-go/utils"

	"github.com/ava-labs/avalanche-rosetta/client"
	"github.com/ava-labs/avalanche-rosetta/constants"
	"github.com/ava-labs/avalanche-rosetta/mapper"
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
	config        *Config
	client        client.Client
	pChainBackend NetworkBackend
	genesisBlock  *types.Block
}

// NewNetworkService returns a new network servicer
func NewNetworkService(
	config *Config,
	client client.Client,
	pChainBackend NetworkBackend,
) server.NetworkAPIServicer {
	genesisBlock := makeGenesisBlock(config.GenesisBlockHash)

	return &NetworkService{
		config:        config,
		client:        client,
		pChainBackend: pChainBackend,
		genesisBlock:  genesisBlock,
	}
}

// NetworkList implements the /network/list endpoint
func (s *NetworkService) NetworkList(
	_ context.Context,
	_ *types.MetadataRequest,
) (*types.NetworkListResponse, *types.Error) {
	return &types.NetworkListResponse{
		NetworkIdentifiers: []*types.NetworkIdentifier{
			s.config.NetworkID,
			s.pChainBackend.NetworkIdentifier(),
		},
	}, nil
}

// NetworkStatus implements the /network/status endpoint
func (s *NetworkService) NetworkStatus(
	ctx context.Context,
	request *types.NetworkRequest,
) (*types.NetworkStatusResponse, *types.Error) {
	if s.config.IsOfflineMode() {
		return nil, ErrUnavailableOffline
	}

	if s.pChainBackend.ShouldHandleRequest(request) {
		return s.pChainBackend.NetworkStatus(ctx, request)
	}

	// Fetch peers
	infoPeers, err := s.client.Peers(ctx)
	if err != nil {
		return nil, WrapError(ErrClientError, err)
	}
	peers := mapper.Peers(infoPeers)

	// Check if all C/X chains are ready
	if err := checkBootstrapStatus(ctx, s.client); err != nil {
		if err.Code == ErrNotReady.Code {
			return &types.NetworkStatusResponse{
				CurrentBlockTimestamp:  s.genesisBlock.Timestamp,
				CurrentBlockIdentifier: s.genesisBlock.BlockIdentifier,
				GenesisBlockIdentifier: s.genesisBlock.BlockIdentifier,
				SyncStatus:             mapper.StageBootstrap,
				Peers:                  peers,
			}, nil
		}
		return nil, err
	}

	// Fetch the latest block
	blockHeader, err := s.client.HeaderByNumber(ctx, nil)
	if err != nil {
		return nil, WrapError(ErrClientError, err)
	}
	if blockHeader == nil {
		return nil, WrapError(ErrClientError, "latest block not found")
	}

	// Fetch the genesis block
	genesisHeader, err := s.client.HeaderByNumber(ctx, big.NewInt(0))
	if err != nil {
		return nil, WrapError(ErrClientError, err)
	}
	if genesisHeader == nil {
		return nil, WrapError(ErrClientError, "genesis block not found")
	}

	return &types.NetworkStatusResponse{
		CurrentBlockTimestamp: int64(blockHeader.Time * utils.MillisecondsInSecond),
		CurrentBlockIdentifier: &types.BlockIdentifier{
			Index: blockHeader.Number.Int64(),
			Hash:  blockHeader.Hash().String(),
		},
		GenesisBlockIdentifier: &types.BlockIdentifier{
			Index: genesisHeader.Number.Int64(),
			Hash:  genesisHeader.Hash().String(),
		},
		SyncStatus: mapper.StageSynced,
		Peers:      peers,
	}, nil
}

// NetworkOptions implements the /network/options endpoint
func (s *NetworkService) NetworkOptions(
	ctx context.Context,
	request *types.NetworkRequest,
) (*types.NetworkOptionsResponse, *types.Error) {
	if s.pChainBackend.ShouldHandleRequest(request) {
		return s.pChainBackend.NetworkOptions(ctx, request)
	}

	return &types.NetworkOptionsResponse{
		Version: &types.Version{
			RosettaVersion:    types.RosettaAPIVersion,
			NodeVersion:       NodeVersion,
			MiddlewareVersion: types.String(MiddlewareVersion),
		},
		Allow: &types.Allow{
			OperationStatuses:       mapper.OperationStatuses,
			OperationTypes:          mapper.OperationTypes,
			CallMethods:             mapper.CallMethods,
			Errors:                  Errors,
			HistoricalBalanceLookup: true,
		},
	}, nil
}

func checkBootstrapStatus(ctx context.Context, client client.Client) *types.Error {
	cReady, err := client.IsBootstrapped(ctx, constants.CChain.String())
	if err != nil {
		return WrapError(ErrClientError, err)
	}

	xReady, err := client.IsBootstrapped(ctx, constants.XChain.String())
	if err != nil {
		return WrapError(ErrClientError, err)
	}

	if !cReady {
		return WrapError(ErrNotReady, "C-Chain is not ready")
	}

	if !xReady {
		return WrapError(ErrNotReady, "X-Chain is not ready")
	}

	return nil
}
