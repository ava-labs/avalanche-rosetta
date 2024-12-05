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
	"github.com/ava-labs/avalanchego/utils/formatting"
	"github.com/ava-labs/avalanchego/vms/components/gas"
	"github.com/ava-labs/avalanchego/vms/platformvm/signer"
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

	// ewok account identifiers
	// Public Key (hex): 0327448e78ffa8cdb24cf19be0204ad954b1bdb4db8c51183534c1eecf2ebd094e
	// Private Key: PrivateKey-ewoqjP7PxY4yr3iLTpLisriqt94hdyDFNgchSxGGztUrTXtNN
	ewoqAccountP = &types.AccountIdentifier{Address: "P-fuji18jma8ppw3nhx5r4ap8clazz0dps7rv5u6wmu4t"}

	cChainID, _ = ids.FromString("yH8D7ThNJkxmtkuv2jgBa4P1Rn3Qpr4pPr7QYNfcdoS6k6HWp")
	pChainID    = ids.Empty

	nodeID = "NodeID-Bvsx89JttQqhqdgwtizAPoVSNW74Xcr2S"

	avalancheNetworkID = avaconstants.FujiID

	avaxAssetID, _ = ids.FromString("U8iRqJoiJm8xZHAacmvYyZVwqQx6uDNtQeP3CQ6fcgQk3JqnK")

	coinID1 = "2ryRVCwNSjEinTViuvDkzX41uQzx3g4babXxZMD46ZV1a9X4Eg:0"

	gasPrice = gas.Price(1000)

	sampleBlsPublicKey      = "0x90f8c7a0425d6b433fb1bd95b41ef76221cdd05d922247356912abfb14db1642ae410bd627052b023e1ea7fb49a909ff"
	sampleProofOfPossession = "0x8df4c907d6d41db47fbe12834789c10572805e61d32d09411ccc3a1d10ea3a978496d6449eb802bc721fed1571ef251300e83bfd41573947b63d4ce631c8724d98b087d0f10ae05426f52e13656be277531fc459f82400e66822d514073bcb7b"
)

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

	matches, err := common.MatchOperations(exportOperations)
	require.NoError(t, err)

	preprocessMetadata := map[string]interface{}{
		"destination_chain": constants.CChain.String(),
	}

	metadataOptions := map[string]interface{}{
		"destination_chain": constants.CChain.String(),
		"type":              pmapper.OpExportAvax,
		"matches":           matches,
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

	matches, err := common.MatchOperations(importOperations)
	require.NoError(t, err)

	preprocessMetadata := map[string]interface{}{
		"source_chain": constants.CChain.String(),
	}

	metadataOptions := map[string]interface{}{
		"source_chain": constants.CChain.String(),
		"type":         pmapper.OpImportAvax,
		"matches":      matches,
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
	pop, err := parsePoP(sampleBlsPublicKey, sampleProofOfPossession)
	require.NoError(t, err)

	operations := []*types.Operation{
		{
			OperationIdentifier: &types.OperationIdentifier{Index: 0},
			RelatedOperations:   nil,
			Type:                pmapper.OpAddPermissionlessValidator,
			Account:             ewoqAccountP,
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
			Type:                pmapper.OpAddPermissionlessValidator,
			Account:             ewoqAccountP,
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
				"delegation_rewards_owner": []string{ewoqAccountP.Address},
				"validator_rewards_owner":  []string{ewoqAccountP.Address},
				"signer":                   pop,
			},
		},
	}

	matches, err := common.MatchOperations(operations)
	require.NoError(t, err)

	preprocessMetadata := map[string]interface{}{
		"node_id":                 nodeID,
		"start":                   startTime,
		"end":                     endTime,
		"shares":                  shares,
		"reward_addresses":        []string{ewoqAccountP.Address},
		"bls_public_key":          sampleBlsPublicKey,
		"bls_proof_of_possession": sampleProofOfPossession,
	}

	metadataOptions := map[string]interface{}{
		"type":                    pmapper.OpAddPermissionlessValidator,
		"node_id":                 nodeID,
		"start":                   startTime,
		"end":                     endTime,
		"shares":                  shares,
		"reward_addresses":        []string{ewoqAccountP.Address},
		"bls_public_key":          sampleBlsPublicKey,
		"bls_proof_of_possession": sampleProofOfPossession,
		"matches":                 matches,
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
		"reward_addresses":           []interface{}{ewoqAccountP.Address},
		"delegator_reward_addresses": nil,
		"bls_proof_of_possession":    sampleProofOfPossession,
		"bls_public_key":             sampleBlsPublicKey,
	}

	signers := []*types.AccountIdentifier{ewoqAccountP}
	stakeSigners := buildRosettaSignerJSON([]string{coinID1}, signers)

	unsignedTx := "0x0000000000190000000500000000000000000000000000000000000000000000000000000000000000000000000000000001f52a5a6dd8f1b3fe05204bdab4f6bcb5a7059f88d0443c636f6c158f838dd1a8000000003d9bdac0ed1d761330cf680efdeb1a42159eb387d6d2950c96f7d28f61bbe2aa00000005000001d1a94a200000000001000000000000000077e1d5c6c289c49976f744749d54369d2129d7500000000062eb5de30000000062fdd2e3000001d1a94a200000000000000000000000000000000000000000000000000000000000000000000000001c90f8c7a0425d6b433fb1bd95b41ef76221cdd05d922247356912abfb14db1642ae410bd627052b023e1ea7fb49a909ff8df4c907d6d41db47fbe12834789c10572805e61d32d09411ccc3a1d10ea3a978496d6449eb802bc721fed1571ef251300e83bfd41573947b63d4ce631c8724d98b087d0f10ae05426f52e13656be277531fc459f82400e66822d514073bcb7b000000013d9bdac0ed1d761330cf680efdeb1a42159eb387d6d2950c96f7d28f61bbe2aa00000007000001d1a94a2000000000000000000000000001000000013cb7d3842e8cee6a0ebd09f1fe884f6861e1b29c0000000b000000000000000000000001000000013cb7d3842e8cee6a0ebd09f1fe884f6861e1b29c0000000b000000000000000000000001000000013cb7d3842e8cee6a0ebd09f1fe884f6861e1b29c00030d4000000000354ff86b"
	unsignedTxHash, err := hex.DecodeString("eb8d33435e683be27907d990c374e64680db91310f3c66e0220f404b5d2433a3")
	require.NoError(t, err)
	wrappedUnsignedTx := `{"tx":"` + unsignedTx + `","signers":` + stakeSigners + `}`

	signingPayloads := []*types.SigningPayload{
		{
			AccountIdentifier: ewoqAccountP,
			Bytes:             unsignedTxHash,
			SignatureType:     types.EcdsaRecovery,
		},
	}

	signedTx := "0x0000000000190000000500000000000000000000000000000000000000000000000000000000000000000000000000000001f52a5a6dd8f1b3fe05204bdab4f6bcb5a7059f88d0443c636f6c158f838dd1a8000000003d9bdac0ed1d761330cf680efdeb1a42159eb387d6d2950c96f7d28f61bbe2aa00000005000001d1a94a200000000001000000000000000077e1d5c6c289c49976f744749d54369d2129d7500000000062eb5de30000000062fdd2e3000001d1a94a200000000000000000000000000000000000000000000000000000000000000000000000001c90f8c7a0425d6b433fb1bd95b41ef76221cdd05d922247356912abfb14db1642ae410bd627052b023e1ea7fb49a909ff8df4c907d6d41db47fbe12834789c10572805e61d32d09411ccc3a1d10ea3a978496d6449eb802bc721fed1571ef251300e83bfd41573947b63d4ce631c8724d98b087d0f10ae05426f52e13656be277531fc459f82400e66822d514073bcb7b000000013d9bdac0ed1d761330cf680efdeb1a42159eb387d6d2950c96f7d28f61bbe2aa00000007000001d1a94a2000000000000000000000000001000000013cb7d3842e8cee6a0ebd09f1fe884f6861e1b29c0000000b000000000000000000000001000000013cb7d3842e8cee6a0ebd09f1fe884f6861e1b29c0000000b000000000000000000000001000000013cb7d3842e8cee6a0ebd09f1fe884f6861e1b29c00030d4000000001000000090000000137d7f04741b3492cb9ad8dbeda64513e569a049a5e5fb5d9c53e27bdfa428acb5b526a45fb9de09b38f1d2f4bb7c0a35bfa753e46c5cc339c0d9a0d9ed0aa415019f1a796e"
	signedTxSignature, err := hex.DecodeString("37d7f04741b3492cb9ad8dbeda64513e569a049a5e5fb5d9c53e27bdfa428acb5b526a45fb9de09b38f1d2f4bb7c0a35bfa753e46c5cc339c0d9a0d9ed0aa41501")
	require.NoError(t, err)
	signedTxHash := "7MgvKBj4pWRFoUosicfdzcJf3ejFnnE4LkSPNNfZDCzJ5SVwE"

	wrappedSignedTx := `{"tx":"` + signedTx + `","signers":` + stakeSigners + `}`

	signatures := []*types.Signature{{
		SigningPayload: &types.SigningPayload{
			AccountIdentifier: ewoqAccountP,
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
			Type:                pmapper.OpAddPermissionlessDelegator,
			Account:             ewoqAccountP,
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
			Type:                pmapper.OpAddPermissionlessDelegator,
			Account:             ewoqAccountP,
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
				"delegator_rewards_owner": []string{ewoqAccountP.Address},
			},
		},
	}

	matches, err := common.MatchOperations(operations)
	require.NoError(t, err)

	preprocessMetadata := map[string]interface{}{
		"node_id":          nodeID,
		"start":            startTime,
		"end":              endTime,
		"reward_addresses": []string{ewoqAccountP.Address},
	}

	metadataOptions := map[string]interface{}{
		"type":             pmapper.OpAddPermissionlessDelegator,
		"node_id":          nodeID,
		"start":            startTime,
		"end":              endTime,
		"reward_addresses": []string{ewoqAccountP.Address},
		"matches":          matches,
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
		"reward_addresses":           []interface{}{ewoqAccountP.Address},
		"bls_proof_of_possession":    "",
		"bls_public_key":             "",
		"delegator_reward_addresses": nil,
		"subnet":                     "",
	}

	signers := []*types.AccountIdentifier{ewoqAccountP}
	stakeSigners := buildRosettaSignerJSON([]string{coinID1}, signers)

	unsignedTx := "0x00000000001a0000000500000000000000000000000000000000000000000000000000000000000000000000000000000001f52a5a6dd8f1b3fe05204bdab4f6bcb5a7059f88d0443c636f6c158f838dd1a8000000003d9bdac0ed1d761330cf680efdeb1a42159eb387d6d2950c96f7d28f61bbe2aa0000000500000005d21dba0000000001000000000000000077e1d5c6c289c49976f744749d54369d2129d7500000000062eb5de30000000062fdd2e300000005d21dba000000000000000000000000000000000000000000000000000000000000000000000000013d9bdac0ed1d761330cf680efdeb1a42159eb387d6d2950c96f7d28f61bbe2aa0000000700000005d21dba00000000000000000000000001000000013cb7d3842e8cee6a0ebd09f1fe884f6861e1b29c0000000b000000000000000000000001000000013cb7d3842e8cee6a0ebd09f1fe884f6861e1b29c00000000ebba4acb"
	unsignedTxHash, err := hex.DecodeString("b7c169b5608c1b625e9bb0d487ae350f7f4fccb8e9a7ce39ba472aff69c9e175")
	require.NoError(t, err)
	wrappedUnsignedTx := `{"tx":"` + unsignedTx + `","signers":` + stakeSigners + `}`

	signingPayloads := []*types.SigningPayload{
		{
			AccountIdentifier: ewoqAccountP,
			Bytes:             unsignedTxHash,
			SignatureType:     types.EcdsaRecovery,
		},
	}

	signedTx := "0x00000000001a0000000500000000000000000000000000000000000000000000000000000000000000000000000000000001f52a5a6dd8f1b3fe05204bdab4f6bcb5a7059f88d0443c636f6c158f838dd1a8000000003d9bdac0ed1d761330cf680efdeb1a42159eb387d6d2950c96f7d28f61bbe2aa0000000500000005d21dba0000000001000000000000000077e1d5c6c289c49976f744749d54369d2129d7500000000062eb5de30000000062fdd2e300000005d21dba000000000000000000000000000000000000000000000000000000000000000000000000013d9bdac0ed1d761330cf680efdeb1a42159eb387d6d2950c96f7d28f61bbe2aa0000000700000005d21dba00000000000000000000000001000000013cb7d3842e8cee6a0ebd09f1fe884f6861e1b29c0000000b000000000000000000000001000000013cb7d3842e8cee6a0ebd09f1fe884f6861e1b29c000000010000000900000001ec8550b25b2a82364ae4944b52ed16c645fc51fa3f239fd9430912760a70f6795381ea7d76a59e22443934e015f695a418c30666415773206e1a8e96bc0ff9bb01a538423f"
	signedTxSignature, err := hex.DecodeString("ec8550b25b2a82364ae4944b52ed16c645fc51fa3f239fd9430912760a70f6795381ea7d76a59e22443934e015f695a418c30666415773206e1a8e96bc0ff9bb01")
	require.NoError(t, err)
	signedTxHash := "2qiPWNr3TgsWbiZLWMyVRQQqgLiq4cmvg7e7gsUjgmVZZwtgsD"

	wrappedSignedTx := `{"tx":"` + signedTx + `","signers":` + stakeSigners + `}`

	signatures := []*types.Signature{{
		SigningPayload: &types.SigningPayload{
			AccountIdentifier: ewoqAccountP,
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

func parsePoP(blsPublicKey, blsProofOfPossession string) (*signer.ProofOfPossession, error) {
	publicKeyBytes, err := formatting.Decode(formatting.HexNC, blsPublicKey)
	if err != nil {
		return nil, err
	}
	popBytes, err := formatting.Decode(formatting.HexNC, blsProofOfPossession)
	if err != nil {
		return nil, err
	}
	pop := &signer.ProofOfPossession{}
	copy(pop.PublicKey[:], publicKeyBytes)
	copy(pop.ProofOfPossession[:], popBytes)
	return pop, nil
}
