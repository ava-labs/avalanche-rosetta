package pchain

import (
	"context"

	"github.com/coinbase/rosetta-sdk-go/types"

	"github.com/ava-labs/avalanche-rosetta/backend"
	"github.com/ava-labs/avalanche-rosetta/constants"
	pconstants "github.com/ava-labs/avalanche-rosetta/constants/pchain"
	"github.com/ava-labs/avalanche-rosetta/mapper"
	pmapper "github.com/ava-labs/avalanche-rosetta/mapper/pchain"
)

// NetworkIdentifier returns P-chain network identifier
// used by /network/list endpoint to list available networks
func (b *Backend) NetworkIdentifier() *types.NetworkIdentifier {
	return b.networkID
}

// NetworkStatus implements /network/status endpoint for P-chain
func (b *Backend) NetworkStatus(ctx context.Context, req *types.NetworkRequest) (*types.NetworkStatusResponse, *types.Error) {
	// Fetch peers
	infoPeers, err := b.pClient.Peers(ctx)
	if err != nil {
		return nil, backend.WrapError(backend.ErrClientError, err)
	}
	peers := mapper.Peers(infoPeers)

	// Check if network is bootstrapped
	ready, err := b.pClient.IsBootstrapped(ctx, constants.PChain.String())
	if err != nil {
		return nil, backend.WrapError(backend.ErrClientError, err)
	}

	if !ready {
		genesisBlock := b.getGenesisBlock()
		return &types.NetworkStatusResponse{
			CurrentBlockIdentifier: b.getGenesisIdentifier(),
			CurrentBlockTimestamp:  genesisBlock.Timestamp,
			GenesisBlockIdentifier: b.getGenesisIdentifier(),
			SyncStatus:             constants.StageBootstrap,
			Peers:                  peers,
		}, nil
	}

	// Current block height
	currentBlock, err := b.indexerParser.ParseCurrentBlock(ctx)
	if err != nil {
		return nil, backend.WrapError(backend.ErrClientError, err)
	}

	return &types.NetworkStatusResponse{
		CurrentBlockIdentifier: &types.BlockIdentifier{
			Index: int64(currentBlock.Height),
			Hash:  currentBlock.BlockID.String(),
		},
		CurrentBlockTimestamp:  currentBlock.Timestamp,
		GenesisBlockIdentifier: b.getGenesisIdentifier(),
		SyncStatus:             constants.StageSynced,
		Peers:                  peers,
	}, nil
}

// NetworkOptions implements /network/options endpoint for P-chain
func (b *Backend) NetworkOptions(ctx context.Context, request *types.NetworkRequest) (*types.NetworkOptionsResponse, *types.Error) {
	return &types.NetworkOptionsResponse{
		Version: &types.Version{
			RosettaVersion:    types.RosettaAPIVersion,
			NodeVersion:       constants.NodeVersion,
			MiddlewareVersion: types.String(constants.MiddlewareVersion),
		},
		Allow: &types.Allow{
			OperationStatuses:       constants.OperationStatuses,
			OperationTypes:          pconstants.TxTypes(),
			CallMethods:             pmapper.CallMethods,
			Errors:                  backend.Errors,
			HistoricalBalanceLookup: false,
		},
	}, nil
}
