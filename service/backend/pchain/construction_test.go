package pchain

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"testing"
	"time"

	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/avalanchego/upgrade"
	"github.com/ava-labs/avalanchego/vms/components/gas"
	"github.com/coinbase/rosetta-sdk-go/types"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/ava-labs/avalanche-rosetta/client"
	"github.com/ava-labs/avalanche-rosetta/constants"
	"github.com/ava-labs/avalanche-rosetta/mapper"
	"github.com/ava-labs/avalanche-rosetta/service"
	"github.com/ava-labs/avalanche-rosetta/service/backend/common"
	"github.com/ava-labs/avalanche-rosetta/service/backend/pchain/indexer"

	pmapper "github.com/ava-labs/avalanche-rosetta/mapper/pchain"
	avaconstants "github.com/ava-labs/avalanchego/utils/constants"
)

var (
	pChainNetworkIdentifier = &types.NetworkIdentifier{
		Blockchain: service.BlockchainName,
		Network:    constants.FujiNetwork,
		SubNetworkIdentifier: &types.SubNetworkIdentifier{
			Network: constants.PChain.String(),
		},
	}

	cAccountIdentifier = &types.AccountIdentifier{Address: "C-fuji123zu6qwhtd9qdd45ryu3j0qtr325gjgddys6u8"}
	pAccountIdentifier = &types.AccountIdentifier{Address: "P-fuji123zu6qwhtd9qdd45ryu3j0qtr325gjgddys6u8"}
	stakeRewardAccount = &types.AccountIdentifier{Address: "P-fuji1ea7dxk8zazpyf8tgc8yg3xyfatey0deqvg9pv2"}

	cChainID, _ = ids.FromString("yH8D7ThNJkxmtkuv2jgBa4P1Rn3Qpr4pPr7QYNfcdoS6k6HWp")
	pChainID    = ids.Empty

	nodeID = "NodeID-Bvsx89JttQqhqdgwtizAPoVSNW74Xcr2S"

	avalancheNetworkID = avaconstants.FujiID

	avaxAssetID, _ = ids.FromString("U8iRqJoiJm8xZHAacmvYyZVwqQx6uDNtQeP3CQ6fcgQk3JqnK")

	txFee       = 1_000_000
	txFeeString = "4099000"

	coinID1 = "2ryRVCwNSjEinTViuvDkzX41uQzx3g4babXxZMD46ZV1a9X4Eg:0"

	gasPrice        = gas.Price(1000)
	feeStateWeights = gas.Dimensions{
		gas.Bandwidth: 1,
		gas.DBRead:    1,
		gas.DBWrite:   1,
		gas.Compute:   1,
	}
)

func getTxFee(timestamp time.Time) string {
	upgradeConfig := upgrade.GetConfig(avalancheNetworkID)
	if upgradeConfig.IsEtnaActivated(timestamp) {
		return "4099000"
	}
	return "1000000"
}

func shouldMockGetFeeState(clientMock *client.MockPChainClient) {
	upgradeConfig := upgrade.GetConfig(avalancheNetworkID)
	if upgradeConfig.IsEtnaActivated(time.Now()) {
		clientMock.EXPECT().GetFeeState(context.Background()).Return(gas.State{}, gasPrice, time.Time{}, nil)
	}
}

func buildRosettaSignerJSON(coinIdentifiers []string, signers []*types.AccountIdentifier) string {
	importSigners := []*common.Signer{}
	for i, s := range signers {
		importSigners = append(importSigners, &common.Signer{
			CoinIdentifier:    coinIdentifiers[i],
			AccountIdentifier: s,
		})
	}
	bytes, _ := json.Marshal(importSigners)
	return string(bytes)
}

func TestConstructionDerive(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	pChainMock := client.NewMockPChainClient(ctrl)
	parserMock := indexer.NewMockParser(ctrl)
	parserMock.EXPECT().GetGenesisBlock(ctx).Return(dummyGenesis, nil)
	backend, err := NewBackend(
		pChainMock,
		parserMock,
		avaxAssetID,
		pChainNetworkIdentifier,
		avalancheNetworkID,
	)
	require.NoError(t, err)

	t.Run("p-chain address", func(t *testing.T) {
		require := require.New(t)

		src := "02e0d4392cfa224d4be19db416b3cf62e90fb2b7015e7b62a95c8cb490514943f6"
		b, err := hex.DecodeString(src)
		require.NoError(err)

		resp, terr := backend.ConstructionDerive(
			context.Background(),
			&types.ConstructionDeriveRequest{
				NetworkIdentifier: pChainNetworkIdentifier,
				PublicKey: &types.PublicKey{
					Bytes:     b,
					CurveType: types.Secp256k1,
				},
			},
		)
		require.Nil(terr)
		require.Equal(
			"P-fuji15f9g0h5xkr5cp47n6u3qxj6yjtzzzrdr23a3tl",
			resp.AccountIdentifier.Address,
		)
	})
}

func TestExportTxConstruction(t *testing.T) {
	exportOperations := []*types.Operation{
		{
			OperationIdentifier: &types.OperationIdentifier{Index: 0},
			RelatedOperations:   nil,
			Type:                pmapper.OpExportAvax,
			Account:             pAccountIdentifier,
			Amount:              mapper.AtomicAvaxAmount(big.NewInt(-1_000_000_000)),
			CoinChange: &types.CoinChange{
				CoinIdentifier: &types.CoinIdentifier{Identifier: coinID1},
				CoinAction:     types.CoinSpent,
			},
			Metadata: map[string]interface{}{
				"type":        pmapper.OpTypeInput,
				"sig_indices": []interface{}{0.0},
				"locktime":    0.0,
			},
		},
		{
			OperationIdentifier: &types.OperationIdentifier{Index: 1},
			Type:                pmapper.OpExportAvax,
			Account:             cAccountIdentifier,
			Amount:              mapper.AtomicAvaxAmount(big.NewInt(999_000_000)),
			Metadata: map[string]interface{}{
				"type":      pmapper.OpTypeExport,
				"threshold": 1.0,
				"locktime":  0.0,
			},
		},
	}

	preprocessMetadata := map[string]interface{}{
		"destination_chain": constants.CChain.String(),
	}

	metadataOptions := map[string]interface{}{
		"destination_chain": constants.CChain.String(),
		"base_fee":          getTxFee(time.Now()),
		"type":              pmapper.OpExportAvax,
	}

	payloadsMetadata := map[string]interface{}{
		"network_id":           float64(avalancheNetworkID),
		"destination_chain":    constants.CChain.String(),
		"destination_chain_id": cChainID.String(),
		"blockchain_id":        pChainID.String(),
	}

	signers := []*types.AccountIdentifier{pAccountIdentifier}
	exportSigners := buildRosettaSignerJSON([]string{coinID1}, signers)

	unsignedExportTx := "0x0000000000120000000500000000000000000000000000000000000000000000000000000000000000000000000000000001f52a5a6dd8f1b3fe05204bdab4f6bcb5a7059f88d0443c636f6c158f838dd1a8000000003d9bdac0ed1d761330cf680efdeb1a42159eb387d6d2950c96f7d28f61bbe2aa00000005000000003b9aca000000000100000000000000007fc93d85c6d62c5b2ac0b519c87010ea5294012d1e407030d6acd0021cac10d5000000013d9bdac0ed1d761330cf680efdeb1a42159eb387d6d2950c96f7d28f61bbe2aa00000007000000003b8b87c0000000000000000000000001000000015445cd01d75b4a06b6b41939193c0b1c5544490d0000000065e8045f"
	unsignedExportTxHash, err := hex.DecodeString("44d579f5cb3c83f4137223a0368721734b622ec392007760eed97f3f1a40c595")
	require.NoError(t, err)

	signingPayloads := []*types.SigningPayload{
		{
			AccountIdentifier: pAccountIdentifier,
			Bytes:             unsignedExportTxHash,
			SignatureType:     types.EcdsaRecovery,
		},
	}

	signedExportTx := "0x0000000000120000000500000000000000000000000000000000000000000000000000000000000000000000000000000001f52a5a6dd8f1b3fe05204bdab4f6bcb5a7059f88d0443c636f6c158f838dd1a8000000003d9bdac0ed1d761330cf680efdeb1a42159eb387d6d2950c96f7d28f61bbe2aa00000005000000003b9aca000000000100000000000000007fc93d85c6d62c5b2ac0b519c87010ea5294012d1e407030d6acd0021cac10d5000000013d9bdac0ed1d761330cf680efdeb1a42159eb387d6d2950c96f7d28f61bbe2aa00000007000000003b8b87c0000000000000000000000001000000015445cd01d75b4a06b6b41939193c0b1c5544490d0000000100000009000000017403e32bb967e71902a988b7da635b4bca2475eedbfd23176610a88162f3a92f20b61f2185825b04b7f8ee8c76427c8dc80eb6091f9e594ef259a59856e5401b0137dc0dc4"
	signedExportTxSignature, err := hex.DecodeString("7403e32bb967e71902a988b7da635b4bca2475eedbfd23176610a88162f3a92f20b61f2185825b04b7f8ee8c76427c8dc80eb6091f9e594ef259a59856e5401b01")
	require.NoError(t, err)
	signedExportTxHash := "bG7jzw16x495XSFdhEavHWR836Ya5teoB1YxRC1inN3HEtqbs"

	wrappedTxFormat := `{"tx":"%s","signers":%s,"destination_chain":"%s","destination_chain_id":"%s"}`
	wrappedUnsignedExportTx := fmt.Sprintf(wrappedTxFormat, unsignedExportTx, exportSigners, constants.CChain.String(), cChainID.String())
	wrappedSignedExportTx := fmt.Sprintf(wrappedTxFormat, signedExportTx, exportSigners, constants.CChain.String(), cChainID.String())

	signatures := []*types.Signature{{
		SigningPayload: &types.SigningPayload{
			AccountIdentifier: pAccountIdentifier,
			Bytes:             unsignedExportTxHash,
			SignatureType:     types.EcdsaRecovery,
		},
		SignatureType: types.EcdsaRecovery,
		Bytes:         signedExportTxSignature,
	}}

	ctx := context.Background()
	ctrl := gomock.NewController(t)
	clientMock := client.NewMockPChainClient(ctrl)
	parserMock := indexer.NewMockParser(ctrl)
	parserMock.EXPECT().GetGenesisBlock(ctx).Return(dummyGenesis, nil)
	backend, err := NewBackend(
		clientMock,
		parserMock,
		avaxAssetID,
		pChainNetworkIdentifier,
		avalancheNetworkID,
	)
	require.NoError(t, err)

	t.Run("preprocess endpoint", func(t *testing.T) {
		shouldMockGetFeeState(clientMock)
		resp, err := backend.ConstructionPreprocess(
			ctx,
			&types.ConstructionPreprocessRequest{
				NetworkIdentifier: pChainNetworkIdentifier,
				Operations:        exportOperations,
				Metadata:          preprocessMetadata,
			},
		)
		require.Nil(t, err)
		require.Equal(t, metadataOptions, resp.Options)
	})

	t.Run("metadata endpoint", func(t *testing.T) {
		clientMock.EXPECT().GetBlockchainID(ctx, constants.PChain.String()).Return(pChainID, nil)
		clientMock.EXPECT().GetBlockchainID(ctx, constants.CChain.String()).Return(cChainID, nil)

		resp, err := backend.ConstructionMetadata(
			ctx,
			&types.ConstructionMetadataRequest{
				NetworkIdentifier: pChainNetworkIdentifier,
				Options:           metadataOptions,
			},
		)
		require.Nil(t, err)
		require.Equal(t, payloadsMetadata, resp.Metadata)
	})

	t.Run("payloads endpoint", func(t *testing.T) {
		resp, err := backend.ConstructionPayloads(
			ctx,
			&types.ConstructionPayloadsRequest{
				NetworkIdentifier: pChainNetworkIdentifier,
				Operations:        exportOperations,
				Metadata:          payloadsMetadata,
			},
		)
		require.Nil(t, err)
		require.Equal(t, wrappedUnsignedExportTx, resp.UnsignedTransaction)
		require.Equal(t, signingPayloads, resp.Payloads,
			"signing payloads mismatch: %s %s",
			marshalSigningPayloads(signingPayloads),
			marshalSigningPayloads(resp.Payloads))
	})

	t.Run("parse endpoint (unsigned)", func(t *testing.T) {
		resp, err := backend.ConstructionParse(
			ctx,
			&types.ConstructionParseRequest{
				NetworkIdentifier: pChainNetworkIdentifier,
				Transaction:       wrappedUnsignedExportTx,
				Signed:            false,
			},
		)
		require.Nil(t, err)
		require.Nil(t, resp.AccountIdentifierSigners)
		require.Equal(t, exportOperations, resp.Operations)
	})

	t.Run("combine endpoint", func(t *testing.T) {
		resp, err := backend.ConstructionCombine(
			ctx,
			&types.ConstructionCombineRequest{
				NetworkIdentifier:   pChainNetworkIdentifier,
				UnsignedTransaction: wrappedUnsignedExportTx,
				Signatures:          signatures,
			},
		)

		require.Nil(t, err)
		require.Equal(t, wrappedSignedExportTx, resp.SignedTransaction)
	})

	t.Run("parse endpoint (signed)", func(t *testing.T) {
		resp, err := backend.ConstructionParse(
			ctx,
			&types.ConstructionParseRequest{
				NetworkIdentifier: pChainNetworkIdentifier,
				Transaction:       wrappedSignedExportTx,
				Signed:            true,
			},
		)
		require.Nil(t, err)
		require.Equal(t, signers, resp.AccountIdentifierSigners)
		require.Equal(t, exportOperations, resp.Operations)
	})

	t.Run("hash endpoint", func(t *testing.T) {
		resp, err := backend.ConstructionHash(ctx, &types.ConstructionHashRequest{
			NetworkIdentifier: pChainNetworkIdentifier,
			SignedTransaction: wrappedSignedExportTx,
		})
		require.Nil(t, err)
		require.Equal(t, signedExportTxHash, resp.TransactionIdentifier.Hash)
	})

	t.Run("submit endpoint", func(t *testing.T) {
		require := require.New(t)

		signedTxBytes, err := mapper.DecodeToBytes(signedExportTx)
		require.NoError(err)
		txID, err := ids.FromString(signedExportTxHash)
		require.NoError(err)

		clientMock.EXPECT().IssueTx(ctx, signedTxBytes).Return(txID, nil)

		resp, terr := backend.ConstructionSubmit(ctx, &types.ConstructionSubmitRequest{
			NetworkIdentifier: pChainNetworkIdentifier,
			SignedTransaction: wrappedSignedExportTx,
		})

		require.Nil(terr)
		require.Equal(signedExportTxHash, resp.TransactionIdentifier.Hash)
	})
}

func TestImportTxConstruction(t *testing.T) {
	importOperations := []*types.Operation{
		{
			OperationIdentifier: &types.OperationIdentifier{Index: 0},
			RelatedOperations:   nil,
			Type:                pmapper.OpImportAvax,
			Account:             cAccountIdentifier,
			Amount:              mapper.AtomicAvaxAmount(big.NewInt(-1_000_000_000)),
			CoinChange: &types.CoinChange{
				CoinIdentifier: &types.CoinIdentifier{Identifier: coinID1},
				CoinAction:     types.CoinSpent,
			},
			Metadata: map[string]interface{}{
				"type":        pmapper.OpTypeImport,
				"sig_indices": []interface{}{0.0},
				"locktime":    0.0,
			},
		},
		{
			OperationIdentifier: &types.OperationIdentifier{Index: 1},
			Type:                pmapper.OpImportAvax,
			Account:             pAccountIdentifier,
			Amount:              mapper.AtomicAvaxAmount(big.NewInt(999_000_000)),
			Metadata: map[string]interface{}{
				"type":      pmapper.OpTypeOutput,
				"threshold": 1.0,
				"locktime":  0.0,
			},
		},
	}

	preprocessMetadata := map[string]interface{}{
		"source_chain": constants.CChain.String(),
	}

	metadataOptions := map[string]interface{}{
		"source_chain": constants.CChain.String(),
		"base_fee":     getTxFee(time.Now()),
		"type":         pmapper.OpImportAvax,
	}

	payloadsMetadata := map[string]interface{}{
		"network_id":      float64(avalancheNetworkID),
		"source_chain_id": cChainID.String(),
		"blockchain_id":   pChainID.String(),
	}

	signers := []*types.AccountIdentifier{cAccountIdentifier}
	importSigners := buildRosettaSignerJSON([]string{coinID1}, signers)

	unsignedImportTx := "0x000000000011000000050000000000000000000000000000000000000000000000000000000000000000000000013d9bdac0ed1d761330cf680efdeb1a42159eb387d6d2950c96f7d28f61bbe2aa00000007000000003b8b87c0000000000000000000000001000000015445cd01d75b4a06b6b41939193c0b1c5544490d00000000000000007fc93d85c6d62c5b2ac0b519c87010ea5294012d1e407030d6acd0021cac10d500000001f52a5a6dd8f1b3fe05204bdab4f6bcb5a7059f88d0443c636f6c158f838dd1a8000000003d9bdac0ed1d761330cf680efdeb1a42159eb387d6d2950c96f7d28f61bbe2aa00000005000000003b9aca000000000100000000000000004ce8b27d"
	unsignedImportTxHash, err := hex.DecodeString("e9114ae12065d1f8631bc40729c806a3a4793de714001bfee66482f520dc1865")
	require.NoError(t, err)
	wrappedUnsignedImportTx := `{"tx":"` + unsignedImportTx + `","signers":` + importSigners + `}` //nolint:goconst

	signingPayloads := []*types.SigningPayload{
		{
			AccountIdentifier: cAccountIdentifier,
			Bytes:             unsignedImportTxHash,
			SignatureType:     types.EcdsaRecovery,
		},
	}

	signedImportTx := "0x000000000011000000050000000000000000000000000000000000000000000000000000000000000000000000013d9bdac0ed1d761330cf680efdeb1a42159eb387d6d2950c96f7d28f61bbe2aa00000007000000003b8b87c0000000000000000000000001000000015445cd01d75b4a06b6b41939193c0b1c5544490d00000000000000007fc93d85c6d62c5b2ac0b519c87010ea5294012d1e407030d6acd0021cac10d500000001f52a5a6dd8f1b3fe05204bdab4f6bcb5a7059f88d0443c636f6c158f838dd1a8000000003d9bdac0ed1d761330cf680efdeb1a42159eb387d6d2950c96f7d28f61bbe2aa00000005000000003b9aca0000000001000000000000000100000009000000017403e32bb967e71902a988b7da635b4bca2475eedbfd23176610a88162f3a92f20b61f2185825b04b7f8ee8c76427c8dc80eb6091f9e594ef259a59856e5401b018ac25b4e"
	signedImportTxSignature, err := hex.DecodeString("7403e32bb967e71902a988b7da635b4bca2475eedbfd23176610a88162f3a92f20b61f2185825b04b7f8ee8c76427c8dc80eb6091f9e594ef259a59856e5401b01")
	require.NoError(t, err)
	signedImportTxHash := "byyEVU6RL7PQNSVT8qEnybWGV5BbBfJwFV6bEDV5mkymXRz62"

	wrappedSignedImportTx := `{"tx":"` + signedImportTx + `","signers":` + importSigners + `}`

	signatures := []*types.Signature{{
		SigningPayload: &types.SigningPayload{
			AccountIdentifier: cAccountIdentifier,
			Bytes:             unsignedImportTxHash,
			SignatureType:     types.EcdsaRecovery,
		},
		SignatureType: types.EcdsaRecovery,
		Bytes:         signedImportTxSignature,
	}}

	ctx := context.Background()
	ctrl := gomock.NewController(t)
	clientMock := client.NewMockPChainClient(ctrl)
	parserMock := indexer.NewMockParser(ctrl)
	parserMock.EXPECT().GetGenesisBlock(ctx).Return(dummyGenesis, nil)
	backend, err := NewBackend(
		clientMock,
		parserMock,
		avaxAssetID,
		pChainNetworkIdentifier,
		avalancheNetworkID,
	)
	require.NoError(t, err)

	t.Run("preprocess endpoint", func(t *testing.T) {
		shouldMockGetFeeState(clientMock)
		resp, err := backend.ConstructionPreprocess(
			ctx,
			&types.ConstructionPreprocessRequest{
				NetworkIdentifier: pChainNetworkIdentifier,
				Operations:        importOperations,
				Metadata:          preprocessMetadata,
			},
		)
		require.Nil(t, err)
		require.Equal(t, metadataOptions, resp.Options)
	})

	t.Run("metadata endpoint", func(t *testing.T) {
		clientMock.EXPECT().GetBlockchainID(ctx, constants.PChain.String()).Return(pChainID, nil)
		clientMock.EXPECT().GetBlockchainID(ctx, constants.CChain.String()).Return(cChainID, nil)

		resp, err := backend.ConstructionMetadata(
			ctx,
			&types.ConstructionMetadataRequest{
				NetworkIdentifier: pChainNetworkIdentifier,
				Options:           metadataOptions,
			},
		)
		require.Nil(t, err)
		require.Equal(t, payloadsMetadata, resp.Metadata)
	})

	t.Run("payloads endpoint", func(t *testing.T) {
		resp, err := backend.ConstructionPayloads(
			ctx,
			&types.ConstructionPayloadsRequest{
				NetworkIdentifier: pChainNetworkIdentifier,
				Operations:        importOperations,
				Metadata:          payloadsMetadata,
			},
		)
		require.Nil(t, err)
		require.Equal(t, wrappedUnsignedImportTx, resp.UnsignedTransaction)
		require.Equal(t, signingPayloads, resp.Payloads,
			"signing payloads mismatch: %s %s",
			marshalSigningPayloads(signingPayloads),
			marshalSigningPayloads(resp.Payloads))
	})

	t.Run("parse endpoint (unsigned)", func(t *testing.T) {
		resp, err := backend.ConstructionParse(
			ctx,
			&types.ConstructionParseRequest{
				NetworkIdentifier: pChainNetworkIdentifier,
				Transaction:       wrappedUnsignedImportTx,
				Signed:            false,
			},
		)
		require.Nil(t, err)
		require.Nil(t, resp.AccountIdentifierSigners)
		require.Equal(t, importOperations, resp.Operations)
	})

	t.Run("combine endpoint", func(t *testing.T) {
		resp, err := backend.ConstructionCombine(
			ctx,
			&types.ConstructionCombineRequest{
				NetworkIdentifier:   pChainNetworkIdentifier,
				UnsignedTransaction: wrappedUnsignedImportTx,
				Signatures:          signatures,
			},
		)

		require.Nil(t, err)
		require.Equal(t, wrappedSignedImportTx, resp.SignedTransaction)
	})

	t.Run("parse endpoint (signed)", func(t *testing.T) {
		resp, err := backend.ConstructionParse(
			ctx,
			&types.ConstructionParseRequest{
				NetworkIdentifier: pChainNetworkIdentifier,
				Transaction:       wrappedSignedImportTx,
				Signed:            true,
			},
		)
		require.Nil(t, err)
		require.Equal(t, signers, resp.AccountIdentifierSigners)
		require.Equal(t, importOperations, resp.Operations)
	})

	t.Run("hash endpoint", func(t *testing.T) {
		resp, err := backend.ConstructionHash(ctx, &types.ConstructionHashRequest{
			NetworkIdentifier: pChainNetworkIdentifier,
			SignedTransaction: wrappedSignedImportTx,
		})
		require.Nil(t, err)
		require.Equal(t, signedImportTxHash, resp.TransactionIdentifier.Hash)
	})

	t.Run("submit endpoint", func(t *testing.T) {
		require := require.New(t)

		signedTxBytes, err := mapper.DecodeToBytes(signedImportTx)
		require.NoError(err)
		txID, err := ids.FromString(signedImportTxHash)
		require.NoError(err)

		clientMock.EXPECT().IssueTx(ctx, signedTxBytes).Return(txID, nil)

		resp, terr := backend.ConstructionSubmit(ctx, &types.ConstructionSubmitRequest{
			NetworkIdentifier: pChainNetworkIdentifier,
			SignedTransaction: wrappedSignedImportTx,
		})

		require.Nil(terr)
		require.Equal(signedImportTxHash, resp.TransactionIdentifier.Hash)
	})
}

func TestAddValidatorTxConstruction(t *testing.T) {
	startTime := uint64(1659592163)
	endTime := startTime + 14*86400
	shares := uint32(200000)

	operations := []*types.Operation{
		{
			OperationIdentifier: &types.OperationIdentifier{Index: 0},
			RelatedOperations:   nil,
			Type:                pmapper.OpAddValidator,
			Account:             pAccountIdentifier,
			Amount:              mapper.AtomicAvaxAmount(big.NewInt(-2_000_000_000_000)),
			CoinChange: &types.CoinChange{
				CoinIdentifier: &types.CoinIdentifier{Identifier: coinID1},
				CoinAction:     "coin_spent",
			},
			Metadata: map[string]interface{}{
				"type":        pmapper.OpTypeInput,
				"sig_indices": []interface{}{0.0},
				"locktime":    0.0,
			},
		},
		{
			OperationIdentifier: &types.OperationIdentifier{Index: 1},
			Type:                pmapper.OpAddValidator,
			Account:             pAccountIdentifier,
			Amount:              mapper.AtomicAvaxAmount(big.NewInt(2_000_000_000_000)),
			Metadata: map[string]interface{}{
				"type":      pmapper.OpTypeStakeOutput,
				"locktime":  0.0,
				"threshold": 1.0,
				// the following are ignored by payloads endpoint but generated by parse
				// added here so that we can simply compare with parse outputs
				"staking_start_time":       startTime,
				"staking_end_time":         endTime,
				"validator_node_id":        nodeID,
				"subnet_id":                pChainID.String(),
				"delegation_rewards_owner": []string{stakeRewardAccount.Address},
				"validator_rewards_owner":  []string{stakeRewardAccount.Address},
			},
		},
	}

	preprocessMetadata := map[string]interface{}{
		"node_id":          nodeID,
		"start":            startTime,
		"end":              endTime,
		"shares":           shares,
		"reward_addresses": []string{stakeRewardAccount.Address},
	}

	metadataOptions := map[string]interface{}{
		"type":             pmapper.OpAddValidator,
		"base_fee":         getTxFee(time.Now()),
		"node_id":          nodeID,
		"start":            startTime,
		"end":              endTime,
		"shares":           shares,
		"reward_addresses": []string{stakeRewardAccount.Address},
	}

	payloadsMetadata := map[string]interface{}{
		"network_id":                 float64(avalancheNetworkID),
		"blockchain_id":              pChainID.String(),
		"node_id":                    nodeID,
		"start":                      float64(startTime),
		"end":                        float64(endTime),
		"shares":                     float64(shares),
		"locktime":                   0.0,
		"subnet":                     "",
		"threshold":                  1.0,
		"reward_addresses":           []interface{}{stakeRewardAccount.Address},
		"delegator_reward_addresses": nil,
		"bls_proof_of_possession":    "",
		"bls_public_key":             "",
	}

	signers := []*types.AccountIdentifier{pAccountIdentifier}
	stakeSigners := buildRosettaSignerJSON([]string{coinID1}, signers)

	unsignedTx := "0x00000000000c0000000500000000000000000000000000000000000000000000000000000000000000000000000000000001f52a5a6dd8f1b3fe05204bdab4f6bcb5a7059f88d0443c636f6c158f838dd1a8000000003d9bdac0ed1d761330cf680efdeb1a42159eb387d6d2950c96f7d28f61bbe2aa00000005000001d1a94a200000000001000000000000000077e1d5c6c289c49976f744749d54369d2129d7500000000062eb5de30000000062fdd2e3000001d1a94a2000000000013d9bdac0ed1d761330cf680efdeb1a42159eb387d6d2950c96f7d28f61bbe2aa00000007000001d1a94a2000000000000000000000000001000000015445cd01d75b4a06b6b41939193c0b1c5544490d0000000b00000000000000000000000100000001cf7cd358e2e882449d68c1c8889889eaf247b72000030d4000000000482f5298"
	unsignedTxHash, err := hex.DecodeString("00c9e13de9f32b5808e54c024b15bdeee5925cbb918d90a272def046cedae800")
	require.NoError(t, err)
	wrappedUnsignedTx := `{"tx":"` + unsignedTx + `","signers":` + stakeSigners + `}`

	signingPayloads := []*types.SigningPayload{
		{
			AccountIdentifier: pAccountIdentifier,
			Bytes:             unsignedTxHash,
			SignatureType:     types.EcdsaRecovery,
		},
	}

	signedTx := "0x00000000000c0000000500000000000000000000000000000000000000000000000000000000000000000000000000000001f52a5a6dd8f1b3fe05204bdab4f6bcb5a7059f88d0443c636f6c158f838dd1a8000000003d9bdac0ed1d761330cf680efdeb1a42159eb387d6d2950c96f7d28f61bbe2aa00000005000001d1a94a200000000001000000000000000077e1d5c6c289c49976f744749d54369d2129d7500000000062eb5de30000000062fdd2e3000001d1a94a2000000000013d9bdac0ed1d761330cf680efdeb1a42159eb387d6d2950c96f7d28f61bbe2aa00000007000001d1a94a2000000000000000000000000001000000015445cd01d75b4a06b6b41939193c0b1c5544490d0000000b00000000000000000000000100000001cf7cd358e2e882449d68c1c8889889eaf247b72000030d400000000100000009000000017403e32bb967e71902a988b7da635b4bca2475eedbfd23176610a88162f3a92f20b61f2185825b04b7f8ee8c76427c8dc80eb6091f9e594ef259a59856e5401b01e228f820"
	signedTxSignature, err := hex.DecodeString("7403e32bb967e71902a988b7da635b4bca2475eedbfd23176610a88162f3a92f20b61f2185825b04b7f8ee8c76427c8dc80eb6091f9e594ef259a59856e5401b01")
	require.NoError(t, err)
	signedTxHash := "2Exfhp6qjdNz8HvECFH2sQxvUxJsaygZjWriY8xh3BvBXWh7Nb"

	wrappedSignedTx := `{"tx":"` + signedTx + `","signers":` + stakeSigners + `}`

	signatures := []*types.Signature{{
		SigningPayload: &types.SigningPayload{
			AccountIdentifier: cAccountIdentifier,
			Bytes:             unsignedTxHash,
			SignatureType:     types.EcdsaRecovery,
		},
		SignatureType: types.EcdsaRecovery,
		Bytes:         signedTxSignature,
	}}

	ctx := context.Background()
	ctrl := gomock.NewController(t)
	clientMock := client.NewMockPChainClient(ctrl)
	parserMock := indexer.NewMockParser(ctrl)
	parserMock.EXPECT().GetGenesisBlock(ctx).Return(dummyGenesis, nil)
	backend, err := NewBackend(
		clientMock,
		parserMock,
		avaxAssetID,
		pChainNetworkIdentifier,
		avalancheNetworkID,
	)
	require.NoError(t, err)

	t.Run("preprocess endpoint", func(t *testing.T) {
		shouldMockGetFeeState(clientMock)
		resp, err := backend.ConstructionPreprocess(
			ctx,
			&types.ConstructionPreprocessRequest{
				NetworkIdentifier: pChainNetworkIdentifier,
				Operations:        operations,
				Metadata:          preprocessMetadata,
			},
		)
		require.Nil(t, err)
		require.Equal(t, metadataOptions, resp.Options)
	})

	t.Run("metadata endpoint", func(t *testing.T) {
		shouldMockGetFeeState(clientMock)
		clientMock.EXPECT().GetBlockchainID(ctx, constants.PChain.String()).Return(pChainID, nil)

		resp, err := backend.ConstructionMetadata(
			ctx,
			&types.ConstructionMetadataRequest{
				NetworkIdentifier: pChainNetworkIdentifier,
				Options:           metadataOptions,
			},
		)
		require.Nil(t, err)
		require.Equal(t, payloadsMetadata, resp.Metadata)
	})

	t.Run("payloads endpoint", func(t *testing.T) {
		resp, err := backend.ConstructionPayloads(
			ctx,
			&types.ConstructionPayloadsRequest{
				NetworkIdentifier: pChainNetworkIdentifier,
				Operations:        operations,
				Metadata:          payloadsMetadata,
			},
		)
		require.Nil(t, err)
		require.Equal(t, wrappedUnsignedTx, resp.UnsignedTransaction)
		require.Equal(t, signingPayloads, resp.Payloads,
			"signing payloads mismatch: %s %s",
			marshalSigningPayloads(signingPayloads),
			marshalSigningPayloads(resp.Payloads))
	})

	t.Run("parse endpoint (unsigned)", func(t *testing.T) {
		resp, err := backend.ConstructionParse(
			ctx,
			&types.ConstructionParseRequest{
				NetworkIdentifier: pChainNetworkIdentifier,
				Transaction:       wrappedUnsignedTx,
				Signed:            false,
			},
		)
		require.Nil(t, err)
		require.Nil(t, resp.AccountIdentifierSigners)
		require.Equal(t, operations, resp.Operations)
	})

	t.Run("combine endpoint", func(t *testing.T) {
		resp, err := backend.ConstructionCombine(
			ctx,
			&types.ConstructionCombineRequest{
				NetworkIdentifier:   pChainNetworkIdentifier,
				UnsignedTransaction: wrappedUnsignedTx,
				Signatures:          signatures,
			},
		)

		require.Nil(t, err)
		require.Equal(t, wrappedSignedTx, resp.SignedTransaction)
	})

	t.Run("parse endpoint (signed)", func(t *testing.T) {
		resp, err := backend.ConstructionParse(
			ctx,
			&types.ConstructionParseRequest{
				NetworkIdentifier: pChainNetworkIdentifier,
				Transaction:       wrappedSignedTx,
				Signed:            true,
			},
		)
		require.Nil(t, err)
		require.Equal(t, signers, resp.AccountIdentifierSigners)
		require.Equal(t, operations, resp.Operations)
	})

	t.Run("hash endpoint", func(t *testing.T) {
		resp, err := backend.ConstructionHash(ctx, &types.ConstructionHashRequest{
			NetworkIdentifier: pChainNetworkIdentifier,
			SignedTransaction: wrappedSignedTx,
		})
		require.Nil(t, err)
		require.Equal(t, signedTxHash, resp.TransactionIdentifier.Hash)
	})

	t.Run("submit endpoint", func(t *testing.T) {
		require := require.New(t)

		signedTxBytes, err := mapper.DecodeToBytes(signedTx)
		require.NoError(err)
		txID, err := ids.FromString(signedTxHash)
		require.NoError(err)

		clientMock.EXPECT().IssueTx(ctx, signedTxBytes).Return(txID, nil)

		resp, terr := backend.ConstructionSubmit(ctx, &types.ConstructionSubmitRequest{
			NetworkIdentifier: pChainNetworkIdentifier,
			SignedTransaction: wrappedSignedTx,
		})

		require.Nil(terr)
		require.Equal(signedTxHash, resp.TransactionIdentifier.Hash)
	})
}

func TestAddDelegatorTxConstruction(t *testing.T) {
	startTime := uint64(1659592163)
	endTime := startTime + 14*86400

	operations := []*types.Operation{
		{
			OperationIdentifier: &types.OperationIdentifier{Index: 0},
			RelatedOperations:   nil,
			Type:                pmapper.OpAddDelegator,
			Account:             pAccountIdentifier,
			Amount:              mapper.AtomicAvaxAmount(big.NewInt(-25_000_000_000)),
			CoinChange: &types.CoinChange{
				CoinIdentifier: &types.CoinIdentifier{Identifier: coinID1},
				CoinAction:     "coin_spent",
			},
			Metadata: map[string]interface{}{
				"type":        pmapper.OpTypeInput,
				"sig_indices": []interface{}{0.0},
				"locktime":    0.0,
			},
		},
		{
			OperationIdentifier: &types.OperationIdentifier{Index: 1},
			Type:                pmapper.OpAddDelegator,
			Account:             pAccountIdentifier,
			Amount:              mapper.AtomicAvaxAmount(big.NewInt(25_000_000_000)),
			Metadata: map[string]interface{}{
				"type":      pmapper.OpTypeStakeOutput,
				"locktime":  0.0,
				"threshold": 1.0,
				// the following are ignored by payloads endpoint but generated by parse
				// added here so that we can simply compare with parse outputs
				"staking_start_time":      startTime,
				"staking_end_time":        endTime,
				"validator_node_id":       nodeID,
				"subnet_id":               pChainID.String(),
				"delegator_rewards_owner": []string{stakeRewardAccount.Address},
			},
		},
	}

	preprocessMetadata := map[string]interface{}{
		"node_id":          nodeID,
		"start":            startTime,
		"end":              endTime,
		"reward_addresses": []string{stakeRewardAccount.Address},
	}

	metadataOptions := map[string]interface{}{
		"type":             pmapper.OpAddDelegator,
		"base_fee":         getTxFee(time.Now()),
		"node_id":          nodeID,
		"start":            startTime,
		"end":              endTime,
		"reward_addresses": []string{stakeRewardAccount.Address},
	}

	payloadsMetadata := map[string]interface{}{
		"network_id":                 float64(avalancheNetworkID),
		"blockchain_id":              pChainID.String(),
		"node_id":                    nodeID,
		"start":                      float64(startTime),
		"end":                        float64(endTime),
		"shares":                     0.0,
		"locktime":                   0.0,
		"threshold":                  1.0,
		"reward_addresses":           []interface{}{stakeRewardAccount.Address},
		"bls_proof_of_possession":    "",
		"bls_public_key":             "",
		"delegator_reward_addresses": nil,
		"subnet":                     "",
	}

	signers := []*types.AccountIdentifier{pAccountIdentifier}
	stakeSigners := buildRosettaSignerJSON([]string{coinID1}, signers)

	unsignedTx := "0x00000000000e0000000500000000000000000000000000000000000000000000000000000000000000000000000000000001f52a5a6dd8f1b3fe05204bdab4f6bcb5a7059f88d0443c636f6c158f838dd1a8000000003d9bdac0ed1d761330cf680efdeb1a42159eb387d6d2950c96f7d28f61bbe2aa0000000500000005d21dba0000000001000000000000000077e1d5c6c289c49976f744749d54369d2129d7500000000062eb5de30000000062fdd2e300000005d21dba00000000013d9bdac0ed1d761330cf680efdeb1a42159eb387d6d2950c96f7d28f61bbe2aa0000000700000005d21dba00000000000000000000000001000000015445cd01d75b4a06b6b41939193c0b1c5544490d0000000b00000000000000000000000100000001cf7cd358e2e882449d68c1c8889889eaf247b72000000000eece5b91"
	unsignedTxHash, err := hex.DecodeString("832a55223ef63d8e39d85025df08c9ae82d0f185d4a0f60b14dec360d721b9f4")
	require.NoError(t, err)
	wrappedUnsignedTx := `{"tx":"` + unsignedTx + `","signers":` + stakeSigners + `}`

	signingPayloads := []*types.SigningPayload{
		{
			AccountIdentifier: pAccountIdentifier,
			Bytes:             unsignedTxHash,
			SignatureType:     types.EcdsaRecovery,
		},
	}

	signedTx := "0x00000000000e0000000500000000000000000000000000000000000000000000000000000000000000000000000000000001f52a5a6dd8f1b3fe05204bdab4f6bcb5a7059f88d0443c636f6c158f838dd1a8000000003d9bdac0ed1d761330cf680efdeb1a42159eb387d6d2950c96f7d28f61bbe2aa0000000500000005d21dba0000000001000000000000000077e1d5c6c289c49976f744749d54369d2129d7500000000062eb5de30000000062fdd2e300000005d21dba00000000013d9bdac0ed1d761330cf680efdeb1a42159eb387d6d2950c96f7d28f61bbe2aa0000000700000005d21dba00000000000000000000000001000000015445cd01d75b4a06b6b41939193c0b1c5544490d0000000b00000000000000000000000100000001cf7cd358e2e882449d68c1c8889889eaf247b7200000000100000009000000017403e32bb967e71902a988b7da635b4bca2475eedbfd23176610a88162f3a92f20b61f2185825b04b7f8ee8c76427c8dc80eb6091f9e594ef259a59856e5401b0143d545c4"
	signedTxSignature, err := hex.DecodeString("7403e32bb967e71902a988b7da635b4bca2475eedbfd23176610a88162f3a92f20b61f2185825b04b7f8ee8c76427c8dc80eb6091f9e594ef259a59856e5401b01")
	require.NoError(t, err)
	signedTxHash := "2eppnnog3TwkBTyQKMh44wz5bUy4geDETEeZVCz7m7uMnjGeCP"

	wrappedSignedTx := `{"tx":"` + signedTx + `","signers":` + stakeSigners + `}`

	signatures := []*types.Signature{{
		SigningPayload: &types.SigningPayload{
			AccountIdentifier: cAccountIdentifier,
			Bytes:             unsignedTxHash,
			SignatureType:     types.EcdsaRecovery,
		},
		SignatureType: types.EcdsaRecovery,
		Bytes:         signedTxSignature,
	}}

	ctx := context.Background()
	ctrl := gomock.NewController(t)
	clientMock := client.NewMockPChainClient(ctrl)
	parserMock := indexer.NewMockParser(ctrl)
	parserMock.EXPECT().GetGenesisBlock(ctx).Return(dummyGenesis, nil)
	backend, err := NewBackend(
		clientMock,
		parserMock,
		avaxAssetID,
		pChainNetworkIdentifier,
		avalancheNetworkID,
	)
	require.NoError(t, err)

	t.Run("preprocess endpoint", func(t *testing.T) {
		shouldMockGetFeeState(clientMock)
		resp, err := backend.ConstructionPreprocess(
			ctx,
			&types.ConstructionPreprocessRequest{
				NetworkIdentifier: pChainNetworkIdentifier,
				Operations:        operations,
				Metadata:          preprocessMetadata,
			},
		)
		require.Nil(t, err)
		require.Equal(t, metadataOptions, resp.Options)
	})

	t.Run("metadata endpoint", func(t *testing.T) {
		shouldMockGetFeeState(clientMock)
		clientMock.EXPECT().GetBlockchainID(ctx, constants.PChain.String()).Return(pChainID, nil)

		resp, err := backend.ConstructionMetadata(
			ctx,
			&types.ConstructionMetadataRequest{
				NetworkIdentifier: pChainNetworkIdentifier,
				Options:           metadataOptions,
			},
		)
		require.Nil(t, err)
		require.Equal(t, payloadsMetadata, resp.Metadata)
	})

	t.Run("payloads endpoint", func(t *testing.T) {
		resp, err := backend.ConstructionPayloads(
			ctx,
			&types.ConstructionPayloadsRequest{
				NetworkIdentifier: pChainNetworkIdentifier,
				Operations:        operations,
				Metadata:          payloadsMetadata,
			},
		)
		require.Nil(t, err)
		require.Equal(t, wrappedUnsignedTx, resp.UnsignedTransaction)
		require.Equal(t, signingPayloads, resp.Payloads,
			"signing payloads mismatch: %s %s",
			marshalSigningPayloads(signingPayloads),
			marshalSigningPayloads(resp.Payloads))
	})

	t.Run("parse endpoint (unsigned)", func(t *testing.T) {
		resp, err := backend.ConstructionParse(
			ctx,
			&types.ConstructionParseRequest{
				NetworkIdentifier: pChainNetworkIdentifier,
				Transaction:       wrappedUnsignedTx,
				Signed:            false,
			},
		)
		require.Nil(t, err)
		require.Nil(t, resp.AccountIdentifierSigners)
		require.Equal(t, operations, resp.Operations)
	})

	t.Run("combine endpoint", func(t *testing.T) {
		resp, err := backend.ConstructionCombine(
			ctx,
			&types.ConstructionCombineRequest{
				NetworkIdentifier:   pChainNetworkIdentifier,
				UnsignedTransaction: wrappedUnsignedTx,
				Signatures:          signatures,
			},
		)

		require.Nil(t, err)
		require.Equal(t, wrappedSignedTx, resp.SignedTransaction)
	})

	t.Run("parse endpoint (signed)", func(t *testing.T) {
		resp, err := backend.ConstructionParse(
			ctx,
			&types.ConstructionParseRequest{
				NetworkIdentifier: pChainNetworkIdentifier,
				Transaction:       wrappedSignedTx,
				Signed:            true,
			},
		)
		require.Nil(t, err)
		require.Equal(t, signers, resp.AccountIdentifierSigners)
		require.Equal(t, operations, resp.Operations)
	})

	t.Run("hash endpoint", func(t *testing.T) {
		resp, err := backend.ConstructionHash(ctx, &types.ConstructionHashRequest{
			NetworkIdentifier: pChainNetworkIdentifier,
			SignedTransaction: wrappedSignedTx,
		})
		require.Nil(t, err)
		require.Equal(t, signedTxHash, resp.TransactionIdentifier.Hash)
	})

	t.Run("submit endpoint", func(t *testing.T) {
		require := require.New(t)

		signedTxBytes, err := mapper.DecodeToBytes(signedTx)
		require.NoError(err)
		txID, err := ids.FromString(signedTxHash)
		require.NoError(err)

		clientMock.EXPECT().IssueTx(ctx, signedTxBytes).Return(txID, nil)

		resp, terr := backend.ConstructionSubmit(ctx, &types.ConstructionSubmitRequest{
			NetworkIdentifier: pChainNetworkIdentifier,
			SignedTransaction: wrappedSignedTx,
		})

		require.Nil(terr)
		require.Equal(signedTxHash, resp.TransactionIdentifier.Hash)
	})
}

func marshalSigningPayloads(payloads []*types.SigningPayload) string {
	bytes, err := json.Marshal(payloads)
	if err != nil {
		return "FAILED_TO_MARSHAL"
	}

	return string(bytes)
}
