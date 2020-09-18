package service

import (
	"context"
	"math/big"

	"github.com/coinbase/rosetta-sdk-go/server"
	"github.com/coinbase/rosetta-sdk-go/types"

	"github.com/figment-networks/avalanche-rosetta/client"
	"github.com/figment-networks/avalanche-rosetta/mapper"
)

// NetworkService implements all /network endpoints
type NetworkService struct {
	network *types.NetworkIdentifier
	info    *client.InfoClient
	evm     *client.EvmClient
}

// NewNetworkService returns a new network servicer
func NewNetworkService(network *types.NetworkIdentifier, evmClient *client.EvmClient, infoClient *client.InfoClient) server.NetworkAPIServicer {
	return &NetworkService{
		network: network,
		evm:     evmClient,
		info:    infoClient,
	}
}

// NetworkList implements the /network/list endpoint
func (s *NetworkService) NetworkList(ctx context.Context, request *types.MetadataRequest) (*types.NetworkListResponse, *types.Error) {
	return &types.NetworkListResponse{
		NetworkIdentifiers: []*types.NetworkIdentifier{
			s.network,
		},
	}, nil
}

// NetworkStatus implements the /network/status endpoint
func (s *NetworkService) NetworkStatus(ctx context.Context, request *types.NetworkRequest) (*types.NetworkStatusResponse, *types.Error) {
	// Fetch the latest block
	blockHeader, err := s.evm.HeaderByNumber(context.Background(), nil)
	if err != nil {
		return nil, errStatusBlockFetchFailed
	}
	if blockHeader == nil {
		return nil, errStatusBlockNotFound
	}

	// Fetch the genesis block
	genesisHeader, err := s.evm.HeaderByNumber(context.Background(), big.NewInt(0))
	if err != nil {
		return nil, errStatusBlockFetchFailed
	}
	if genesisHeader == nil {
		return nil, errStatusBlockNotFound
	}

	// Fetch all node's peers
	infoPeers, err := s.info.Peers()
	if err != nil {
		return nil, errStatusPeersFailed
	}
	peers := mapper.Peers(infoPeers)

	return &types.NetworkStatusResponse{
		CurrentBlockTimestamp: int64(blockHeader.Time * 1000),
		CurrentBlockIdentifier: &types.BlockIdentifier{
			Index: blockHeader.Number.Int64(),
			Hash:  blockHeader.Hash().String(),
		},
		// TODO: include oldest block
		GenesisBlockIdentifier: &types.BlockIdentifier{
			Index: genesisHeader.Number.Int64(),
			Hash:  genesisHeader.Hash().String(),
		},
		Peers: peers,
	}, nil
}

// NetworkOptions implements the /network/options endpoint
func (s *NetworkService) NetworkOptions(ctx context.Context, request *types.NetworkRequest) (*types.NetworkOptionsResponse, *types.Error) {
	nodeVersion, err := s.info.NodeVersion()
	if err != nil {
		return nil, errStatusNodeVersionFailed
	}

	middlewareVersion := MiddlewareVersion

	resp := &types.NetworkOptionsResponse{
		Version: &types.Version{
			RosettaVersion:    RosettaVersion,
			MiddlewareVersion: &middlewareVersion,
			NodeVersion:       nodeVersion,
		},
		Allow: &types.Allow{
			OperationStatuses: mapper.OperationStatuses,
			OperationTypes:    mapper.OperationTypes,
			Errors:            allErrors(),
		},
	}

	return resp, nil
}
