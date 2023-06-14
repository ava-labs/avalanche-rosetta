package pchain

import (
	"context"
	"testing"

	"github.com/ava-labs/avalanchego/api"
	"github.com/ava-labs/avalanchego/ids"
	avaConst "github.com/ava-labs/avalanchego/utils/constants"
	"github.com/ava-labs/avalanchego/utils/formatting"
	"github.com/ava-labs/avalanchego/vms/components/avax"
	"github.com/ava-labs/avalanchego/vms/components/verify"
	"github.com/ava-labs/avalanchego/vms/platformvm/blocks"
	"github.com/ava-labs/avalanchego/vms/platformvm/txs"
	"github.com/ava-labs/avalanchego/vms/secp256k1fx"
	avaTypes "github.com/ava-labs/avalanchego/vms/types"
	"github.com/coinbase/rosetta-sdk-go/types"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/ava-labs/avalanche-rosetta/constants"
	mocks "github.com/ava-labs/avalanche-rosetta/mocks/client"
	indexerMocks "github.com/ava-labs/avalanche-rosetta/mocks/service/backend/pchain/indexer"
	"github.com/ava-labs/avalanche-rosetta/service"
	"github.com/ava-labs/avalanche-rosetta/service/backend/pchain/indexer"
)

func TestFetchBlkDependencies(t *testing.T) {
	dummyGenesis = &indexer.ParsedGenesisBlock{}

	mockPClient := mocks.NewPChainClient(t)
	mockIndexerParser := indexerMocks.NewParser(t)

	ctx := context.Background()

	networkID := avaConst.MainnetID
	networkIdentifier := &types.NetworkIdentifier{
		Blockchain: service.BlockchainName,
		Network:    constants.MainnetNetwork,
		SubNetworkIdentifier: &types.SubNetworkIdentifier{
			Network: constants.PChain.String(),
		},
	}

	signedImportTx, err := makeImportTx(t, networkID)
	require.Nil(t, err)

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

	mockIndexerParser.Mock.On("GetGenesisBlock", ctx).Return(dummyGenesis, nil)
	mockPClient.Mock.On("GetTx", mock.Anything, nonGenesisTxID).Return(signedImportTx.Bytes(), nil)

	mockPClient.Mock.On("GetRewardUTXOs", mock.Anything, &api.GetTxArgs{
		TxID:     nonGenesisTxID,
		Encoding: formatting.Hex,
	}).Return(nil, nil)

	backend, err := NewBackend(service.ModeOnline, mockPClient, mockIndexerParser, avaxAssetID, networkIdentifier, networkID)
	require.Nil(t, err)

	deps, err := backend.fetchBlkDependencies(ctx, []*txs.Tx{tx})
	require.Nil(t, err)

	mockPClient.AssertExpectations(t)
	mockIndexerParser.AssertExpectations(t)

	require.Equal(t, 2, len(deps))
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
				Memo: avaTypes.JSONByteSlice{},
			},
			SyntacticallyVerified: false,
		},
		SourceChain:    cChainID,
		ImportedInputs: []*avax.TransferableInput{},
	}
	signedImportTx, err := txs.NewSigned(importTx, blocks.Codec, nil)
	require.Nil(t, err)
	signedImportTx.Creds = []verify.Verifiable{}
	return signedImportTx, err
}
