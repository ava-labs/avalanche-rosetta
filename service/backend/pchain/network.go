package pchain

import (
	"context"

	"github.com/coinbase/rosetta-sdk-go/types"

	"github.com/ava-labs/avalanche-rosetta/mapper"
	pmapper "github.com/ava-labs/avalanche-rosetta/mapper/pchain"
	"github.com/ava-labs/avalanche-rosetta/service"
)

func (b *Backend) NetworkIdentifier() *types.NetworkIdentifier {
	return b.networkIdentifier
}

func (b *Backend) NetworkStatus(ctx context.Context, req *types.NetworkRequest) (*types.NetworkStatusResponse, *types.Error) {
	// Fetch peers
	infoPeers, err := b.pClient.Peers(ctx)
	if err != nil {
		return nil, service.WrapError(service.ErrClientError, err)
	}
	peers := mapper.Peers(infoPeers)

	// Check if network is bootstrapped
	ready, err := b.pClient.IsBootstrapped(ctx, mapper.PChainNetworkIdentifier)
	if err != nil {
		return nil, service.WrapError(service.ErrClientError, err)
	}

	genesisBlock, err := b.getGenesisBlock(ctx)
	if err != nil {
		return nil, service.WrapError(service.ErrClientError, err)
	}

	if !ready {
		return &types.NetworkStatusResponse{
			CurrentBlockIdentifier: b.genesisBlockIdentifier,
			CurrentBlockTimestamp:  genesisBlock.Timestamp,
			GenesisBlockIdentifier: b.genesisBlockIdentifier,
			SyncStatus:             mapper.StageBootstrap,
			Peers:                  peers,
		}, nil
	}

	// Current block height
	currentBlock, err := b.indexerParser.ParseCurrentBlock(ctx)
	if err != nil {
		return nil, service.WrapError(service.ErrClientError, err)
	}

	return &types.NetworkStatusResponse{
		CurrentBlockIdentifier: &types.BlockIdentifier{
			Index: int64(currentBlock.Height),
			Hash:  currentBlock.BlockID.String(),
		},
		CurrentBlockTimestamp:  currentBlock.Timestamp,
		GenesisBlockIdentifier: b.genesisBlockIdentifier,
		SyncStatus:             mapper.StageSynced,
		Peers:                  peers,
	}, nil
}

func (b *Backend) NetworkOptions(ctx context.Context, request *types.NetworkRequest) (*types.NetworkOptionsResponse, *types.Error) {
	return &types.NetworkOptionsResponse{
		Version: &types.Version{
			RosettaVersion:    types.RosettaAPIVersion,
			NodeVersion:       service.NodeVersion,
			MiddlewareVersion: types.String(service.MiddlewareVersion),
		},
		Allow: &types.Allow{
			OperationStatuses:       mapper.OperationStatuses,
			OperationTypes:          pmapper.OperationTypes,
			CallMethods:             pmapper.CallMethods,
			Errors:                  service.Errors,
			HistoricalBalanceLookup: false,
		},
	}, nil
}
