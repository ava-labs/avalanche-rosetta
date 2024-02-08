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

	backend, err := NewBackend(service.ModeOnline, mockPClient, mockIndexerParser, avaxAssetID, networkIdentifier, networkID)
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
