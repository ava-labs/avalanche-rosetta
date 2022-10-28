package cchain

import (
	"context"
	"math/big"

	"github.com/ava-labs/avalanche-rosetta/client"
	"github.com/ava-labs/avalanche-rosetta/constants"
	"github.com/ava-labs/avalanche-rosetta/mapper"
	"github.com/coinbase/rosetta-sdk-go/types"
	"github.com/coinbase/rosetta-sdk-go/utils"

	cconstants "github.com/ava-labs/avalanche-rosetta/constants/cchain"
)

// this is common across all chains. TODO: make one
func (b *Backend) NetworkIdentifier() *types.NetworkIdentifier {
	return b.config.NetworkID
}

// NetworkStatus implements the /network/status endpoint
func (b *Backend) NetworkStatus(
	ctx context.Context,
	request *types.NetworkRequest,
) (*types.NetworkStatusResponse, *types.Error) {
	if b.config.IsOfflineMode() {
		return nil, ErrUnavailableOffline
	}

	// Fetch peers
	infoPeers, err := b.client.Peers(ctx)
	if err != nil {
		return nil, WrapError(ErrClientError, err)
	}
	peers := mapper.Peers(infoPeers)

	// Check if all C/X chains are ready
	if err := checkBootstrapStatus(ctx, b.client); err != nil {
		if err.Code == ErrNotReady.Code {
			return &types.NetworkStatusResponse{
				CurrentBlockTimestamp:  b.genesisBlock.Timestamp,
				CurrentBlockIdentifier: b.genesisBlock.BlockIdentifier,
				GenesisBlockIdentifier: b.genesisBlock.BlockIdentifier,
				SyncStatus:             constants.StageBootstrap,
				Peers:                  peers,
			}, nil
		}
		return nil, err
	}

	// Fetch the latest block
	blockHeader, err := b.client.HeaderByNumber(ctx, nil)
	if err != nil {
		return nil, WrapError(ErrClientError, err)
	}
	if blockHeader == nil {
		return nil, WrapError(ErrClientError, "latest block not found")
	}

	// Fetch the genesis block
	genesisHeader, err := b.client.HeaderByNumber(ctx, big.NewInt(0))
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
		SyncStatus: constants.StageSynced,
		Peers:      peers,
	}, nil
}

// NetworkOptions implements the /network/options endpoint
func (b *Backend) NetworkOptions(
	ctx context.Context,
	request *types.NetworkRequest,
) (*types.NetworkOptionsResponse, *types.Error) {
	return &types.NetworkOptionsResponse{
		Version: &types.Version{
			RosettaVersion:    types.RosettaAPIVersion,
			NodeVersion:       NodeVersion,
			MiddlewareVersion: types.String(MiddlewareVersion),
		},
		Allow: &types.Allow{
			OperationStatuses:       constants.OperationStatuses,
			OperationTypes:          cconstants.CChainOps(),
			CallMethods:             cconstants.CChainCallMethods(),
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
