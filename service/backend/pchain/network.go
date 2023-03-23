package pchain

import (
	"context"

	"github.com/coinbase/rosetta-sdk-go/types"

	"github.com/ava-labs/avalanche-rosetta/constants"
	"github.com/ava-labs/avalanche-rosetta/mapper"
	pmapper "github.com/ava-labs/avalanche-rosetta/mapper/pchain"
	"github.com/ava-labs/avalanche-rosetta/service"
)

// NetworkIdentifier returns P-chain network identifier
// used by /network/list endpoint to list available networks
func (b *Backend) NetworkIdentifier() *types.NetworkIdentifier {
	return b.networkID
}

// NetworkStatus implements /network/status endpoint for P-chain
func (b *Backend) NetworkStatus(ctx context.Context, _ *types.NetworkRequest) (*types.NetworkStatusResponse, *types.Error) {
	// Fetch peers
	infoPeers, err := b.pClient.Peers(ctx)
	if err != nil {
		return nil, service.WrapError(service.ErrClientError, err)
	}
	peers := mapper.Peers(infoPeers)

	// Check if network is bootstrapped
	ready, err := b.pClient.IsBootstrapped(ctx, constants.PChain.String())
	if err != nil {
		return nil, service.WrapError(service.ErrClientError, err)
	}

	if !ready {
		genesisBlock := b.getGenesisBlock()
		return &types.NetworkStatusResponse{
			CurrentBlockIdentifier: b.getGenesisIdentifier(),
			CurrentBlockTimestamp:  genesisBlock.Timestamp,
			GenesisBlockIdentifier: b.getGenesisIdentifier(),
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
		GenesisBlockIdentifier: b.getGenesisIdentifier(),
		SyncStatus:             mapper.StageSynced,
		Peers:                  peers,
	}, nil
}

// NetworkOptions implements /network/options endpoint for P-chain
func (b *Backend) NetworkOptions(_ context.Context, _ *types.NetworkRequest) (*types.NetworkOptionsResponse, *types.Error) {
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
