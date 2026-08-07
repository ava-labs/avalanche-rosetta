package pchain

import (
	"context"
	"testing"

	"github.com/ava-labs/avalanchego/api"
	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/avalanchego/utils/formatting"
	"github.com/ava-labs/avalanchego/vms/components/avax"
	"github.com/ava-labs/avalanchego/vms/components/verify"
	"github.com/ava-labs/avalanchego/vms/platformvm/block"
	"github.com/ava-labs/avalanchego/vms/platformvm/signer"
	"github.com/ava-labs/avalanchego/vms/platformvm/txs"
	"github.com/ava-labs/avalanchego/vms/secp256k1fx"
	"github.com/coinbase/rosetta-sdk-go/types"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/ava-labs/avalanche-rosetta/client"
	"github.com/ava-labs/avalanche-rosetta/constants"
	"github.com/ava-labs/avalanche-rosetta/service"
	"github.com/ava-labs/avalanche-rosetta/service/backend/pchain/indexer"

	avaconstants "github.com/ava-labs/avalanchego/utils/constants"
	avatypes "github.com/ava-labs/avalanchego/vms/types"
)

func TestFetchBlkDependencies(t *testing.T) {
	dummyGenesis = &indexer.ParsedGenesisBlock{}

	ctrl := gomock.NewController(t)
	mockPClient := client.NewMockPChainClient(ctrl)
	mockIndexerParser := indexer.NewMockParser(ctrl)

	ctx := context.Background()

	networkID := avaconstants.MainnetID
	networkIdentifier := &types.NetworkIdentifier{
		Blockchain: service.BlockchainName,
		Network:    constants.MainnetNetwork,
		SubNetworkIdentifier: &types.SubNetworkIdentifier{
			Network: constants.PChain.String(),
		},
	}

	signedImportTx, err := makeImportTx(t, networkID)
	require.NoError(t, err)

	genesisTxID := ids.Empty
	nonGenesisTxID := signedImportTx.ID()

	tx := &txs.Tx{
		Unsigned: &txs.ExportTx{
			BaseTx: txs.BaseTx{
				BaseTx: avax.BaseTx{
					NetworkID:    avalancheNetworkID,
					BlockchainID: pChainID,
					Ins: []*avax.TransferableInput{
						{
							UTXOID: avax.UTXOID{
								// Genesis allocation input
								TxID:        genesisTxID,
								OutputIndex: 1234,
							},
							Asset: avax.Asset{
								ID: avaxAssetID,
							},
							In: &secp256k1fx.TransferInput{
								Amt:   1000,
								Input: secp256k1fx.Input{},
							},
						},
						{
							UTXOID: avax.UTXOID{
								TxID:        nonGenesisTxID,
								OutputIndex: 1,
							},
							Asset: avax.Asset{
								ID: avaxAssetID,
							},
							In: &secp256k1fx.TransferInput{
								Amt:   2000,
								Input: secp256k1fx.Input{},
							},
						},
					},
				},
			},
			DestinationChain: cChainID,
			ExportedOutputs:  nil,
		},
	}

	mockIndexerParser.EXPECT().GetGenesisBlock(ctx).Return(dummyGenesis, nil)
	mockPClient.EXPECT().GetTx(gomock.Any(), nonGenesisTxID).Return(signedImportTx.Bytes(), nil)

	mockPClient.EXPECT().GetRewardUTXOs(gomock.Any(), &api.GetTxArgs{
		TxID:     nonGenesisTxID,
		Encoding: formatting.Hex,
	}).Return(nil, nil)

	backend, err := NewBackend(mockPClient, mockIndexerParser, avaxAssetID, networkIdentifier, networkID)
	require.NoError(t, err)

	deps, err := backend.fetchBlkDependencies(ctx, []*txs.Tx{tx})
	require.NoError(t, err)

	require.Len(t, deps, 2)
	require.Equal(t, ids.Empty, deps[genesisTxID].Tx.ID())
	require.NotEqual(t, ids.Empty, deps[nonGenesisTxID].Tx.ID())
	require.Equal(t, signedImportTx, deps[nonGenesisTxID].Tx)
}

func makeImportTx(t *testing.T, networkID uint32) (*txs.Tx, error) {
	importTx := &txs.ImportTx{
		BaseTx: txs.BaseTx{
			BaseTx: avax.BaseTx{
				NetworkID: networkID,
				Outs: []*avax.TransferableOutput{
					{
						Asset: avax.Asset{
							ID: avaxAssetID,
						},
						Out: &secp256k1fx.TransferOutput{
							Amt:          2000,
							OutputOwners: secp256k1fx.OutputOwners{Addrs: []ids.ShortID{}},
						},
					},
				},
				Ins:  []*avax.TransferableInput{},
				Memo: avatypes.JSONByteSlice{},
			},
			SyntacticallyVerified: false,
		},
		SourceChain:    cChainID,
		ImportedInputs: []*avax.TransferableInput{},
	}
	signedImportTx, err := txs.NewSigned(importTx, block.Codec, nil)
	require.NoError(t, err)
	signedImportTx.Creds = []verify.Verifiable{}
	return signedImportTx, err
}

// TestFetchBlkDependenciesRewardAutoRenewedValidator verifies that when a block
// contains a RewardAutoRenewedValidatorTx, GetRewardUTXOs is queried with the
// reward tx's own ID (not the staking tx ID referenced by TxID).
// AvalancheGo stores reward UTXOs under the reward tx's own ID for this tx type.
func TestFetchBlkDependenciesRewardAutoRenewedValidator(t *testing.T) {
	dummyGenesis = &indexer.ParsedGenesisBlock{}

	ctrl := gomock.NewController(t)
	mockPClient := client.NewMockPChainClient(ctrl)
	mockIndexerParser := indexer.NewMockParser(ctrl)

	ctx := context.Background()

	networkID := avaconstants.MainnetID
	networkIdentifier := &types.NetworkIdentifier{
		Blockchain: service.BlockchainName,
		Network:    constants.MainnetNetwork,
		SubNetworkIdentifier: &types.SubNetworkIdentifier{
			Network: constants.PChain.String(),
		},
	}

	// Build a minimal AddAutoRenewedValidatorTx to serve as the staking tx.
	rewardsOwner := &secp256k1fx.OutputOwners{
		Threshold: 1,
		Addrs:     []ids.ShortID{ids.GenerateTestShortID()},
	}
	validatorNodeID := ids.GenerateTestNodeID()
	stakingTx, err := txs.NewSigned(&txs.AddAutoRenewedValidatorTx{
		BaseTx:                txs.BaseTx{BaseTx: avax.BaseTx{NetworkID: networkID}},
		ValidatorNodeID:       validatorNodeID[:],
		Signer:                &signer.Empty{},
		ValidatorRewardsOwner: rewardsOwner,
		DelegatorRewardsOwner: rewardsOwner,
		ValidatorAuthority:    rewardsOwner,
	}, txs.Codec, nil)
	require.NoError(t, err)
	stakingTxID := stakingTx.ID()

	// Build the RewardAutoRenewedValidatorTx referencing the staking tx.
	rewardTx, err := txs.NewSigned(&txs.RewardAutoRenewedValidatorTx{
		TxID:      stakingTxID,
		Timestamp: 1,
	}, txs.Codec, nil)
	require.NoError(t, err)
	rewardTxID := rewardTx.ID()

	mockIndexerParser.EXPECT().GetGenesisBlock(ctx).Return(dummyGenesis, nil)

	// The staking tx is fetched by its own ID.
	mockPClient.EXPECT().GetTx(gomock.Any(), stakingTxID).Return(stakingTx.Bytes(), nil)

	// GetRewardUTXOs must be called with the reward tx's own ID, not the staking tx ID.
	// The gomock expectation enforces this: calling with stakingTxID would not match.
	mockPClient.EXPECT().GetRewardUTXOs(gomock.Any(), &api.GetTxArgs{
		TxID:     rewardTxID,
		Encoding: formatting.Hex,
	}).Return(nil, nil)

	backend, err := NewBackend(mockPClient, mockIndexerParser, avaxAssetID, networkIdentifier, networkID)
	require.NoError(t, err)

	deps, err := backend.fetchBlkDependencies(ctx, []*txs.Tx{rewardTx})
	require.NoError(t, err)

	// Dep is keyed by both the staking tx ID (for parser lookup via TxID) and the
	// reward tx ID (so isMultisig can resolve reward UTXOs when they are spent later).
	require.Len(t, deps, 2)
	require.Contains(t, deps, stakingTxID)
	require.NotNil(t, deps[stakingTxID].Tx)
	require.Contains(t, deps, rewardTxID)
	require.Same(t, deps[stakingTxID], deps[rewardTxID])
}

// TestFetchBlkDependenciesRewardAndSpendSameStakingTx guards against a dependency
// dedup regression: when a block holds both a RewardAutoRenewedValidatorTx (referencing
// staking tx X) and another tx spending an output of X, two requests for X are produced —
// one with the reward-UTXO override and one without. The staking tx must be fetched once
// and its reward UTXOs queried under the reward tx ID; the strict gomock expectations below
// (GetTx once, GetRewardUTXOs only with rewardTxID) fail if the bare duplicate request wins.
func TestFetchBlkDependenciesRewardAndSpendSameStakingTx(t *testing.T) {
	dummyGenesis = &indexer.ParsedGenesisBlock{}

	ctrl := gomock.NewController(t)
	mockPClient := client.NewMockPChainClient(ctrl)
	mockIndexerParser := indexer.NewMockParser(ctrl)

	ctx := context.Background()

	networkID := avaconstants.MainnetID
	networkIdentifier := &types.NetworkIdentifier{
		Blockchain: service.BlockchainName,
		Network:    constants.MainnetNetwork,
		SubNetworkIdentifier: &types.SubNetworkIdentifier{
			Network: constants.PChain.String(),
		},
	}

	rewardsOwner := &secp256k1fx.OutputOwners{
		Threshold: 1,
		Addrs:     []ids.ShortID{ids.GenerateTestShortID()},
	}
	validatorNodeID := ids.GenerateTestNodeID()
	stakingTx, err := txs.NewSigned(&txs.AddAutoRenewedValidatorTx{
		BaseTx:                txs.BaseTx{BaseTx: avax.BaseTx{NetworkID: networkID}},
		ValidatorNodeID:       validatorNodeID[:],
		Signer:                &signer.Empty{},
		ValidatorRewardsOwner: rewardsOwner,
		DelegatorRewardsOwner: rewardsOwner,
		ValidatorAuthority:    rewardsOwner,
	}, txs.Codec, nil)
	require.NoError(t, err)
	stakingTxID := stakingTx.ID()

	rewardTx, err := txs.NewSigned(&txs.RewardAutoRenewedValidatorTx{
		TxID:      stakingTxID,
		Timestamp: 1,
	}, txs.Codec, nil)
	require.NoError(t, err)
	rewardTxID := rewardTx.ID()

	// A second tx in the same block that spends an output of the staking tx X,
	// producing a bare {X, ids.Empty} dependency request alongside the reward tx's
	// {X, rewardTxID} request.
	spendingTx := &txs.Tx{
		Unsigned: &txs.ExportTx{
			BaseTx: txs.BaseTx{BaseTx: avax.BaseTx{
				NetworkID:    networkID,
				BlockchainID: pChainID,
				Ins: []*avax.TransferableInput{
					{
						UTXOID: avax.UTXOID{TxID: stakingTxID, OutputIndex: 0},
						Asset:  avax.Asset{ID: avaxAssetID},
						In:     &secp256k1fx.TransferInput{Amt: 1000, Input: secp256k1fx.Input{}},
					},
				},
			}},
			DestinationChain: cChainID,
		},
	}

	mockIndexerParser.EXPECT().GetGenesisBlock(ctx).Return(dummyGenesis, nil)
	// Fetched exactly once despite two requests referencing it.
	mockPClient.EXPECT().GetTx(gomock.Any(), stakingTxID).Return(stakingTx.Bytes(), nil)
	// Reward UTXOs must be queried under the reward tx ID, never the staking tx ID.
	mockPClient.EXPECT().GetRewardUTXOs(gomock.Any(), &api.GetTxArgs{
		TxID:     rewardTxID,
		Encoding: formatting.Hex,
	}).Return(nil, nil)

	backend, err := NewBackend(mockPClient, mockIndexerParser, avaxAssetID, networkIdentifier, networkID)
	require.NoError(t, err)

	deps, err := backend.fetchBlkDependencies(ctx, []*txs.Tx{spendingTx, rewardTx})
	require.NoError(t, err)

	require.Contains(t, deps, stakingTxID)
	require.Contains(t, deps, rewardTxID)
	require.Same(t, deps[stakingTxID], deps[rewardTxID])
}
