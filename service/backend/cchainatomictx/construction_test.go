package cchainatomictx

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"testing"

	"github.com/ava-labs/avalanchego/ids"
	avaConst "github.com/ava-labs/avalanchego/utils/constants"
	"github.com/ava-labs/avalanchego/utils/formatting"
	"github.com/coinbase/rosetta-sdk-go/types"
	ethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"

	"github.com/ava-labs/avalanche-rosetta/constants"
	"github.com/ava-labs/avalanche-rosetta/mapper"
	mocks "github.com/ava-labs/avalanche-rosetta/mocks/client"
	"github.com/ava-labs/avalanche-rosetta/service"
	"github.com/ava-labs/avalanche-rosetta/service/backend/common"
)

var (
	networkIdentifier = &types.NetworkIdentifier{
		Blockchain: service.BlockchainName,
		Network:    constants.FujiNetwork,
	}

	cAccountIdentifier       = &types.AccountIdentifier{Address: "0x3158e80abD5A1e1aa716003C9Db096792C379621"}
	cAccountBech32Identifier = &types.AccountIdentifier{Address: "C-fuji1wmd9dfrqpud6daq0cde47u0r7pkrr46ep60399"}
	pAccountIdentifier       = &types.AccountIdentifier{Address: "P-fuji1wmd9dfrqpud6daq0cde47u0r7pkrr46ep60399"}

	cChainID, _ = ids.FromString("yH8D7ThNJkxmtkuv2jgBa4P1Rn3Qpr4pPr7QYNfcdoS6k6HWp")
	pChainID    = ids.Empty

	avalancheNetworkID = avaConst.FujiID

	avaxAssetID, _ = ids.FromString("U8iRqJoiJm8xZHAacmvYyZVwqQx6uDNtQeP3CQ6fcgQk3JqnK")
)

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
	backend := NewBackend(nil, ids.Empty, avalancheNetworkID)

	t.Run("c-chain address", func(t *testing.T) {
		src := "02e0d4392cfa224d4be19db416b3cf62e90fb2b7015e7b62a95c8cb490514943f6"
		b, _ := hex.DecodeString(src)

		resp, err := backend.ConstructionDerive(
			context.Background(),
			&types.ConstructionDeriveRequest{
				NetworkIdentifier: networkIdentifier,
				PublicKey: &types.PublicKey{
					Bytes:     b,
					CurveType: types.Secp256k1,
				},
			},
		)
		assert.Nil(t, err)
		assert.Equal(
			t,
			"C-fuji15f9g0h5xkr5cp47n6u3qxj6yjtzzzrdr23a3tl",
			resp.AccountIdentifier.Address,
		)
	})
}

func TestExportTxConstruction(t *testing.T) {
	nonce := uint64(48)

	opExport := "EXPORT"

	exportOperations := []*types.Operation{
		{
			OperationIdentifier: &types.OperationIdentifier{Index: 0},
			RelatedOperations:   nil,
			Type:                opExport,
			Account:             cAccountIdentifier,
			Amount:              mapper.AtomicAvaxAmount(big.NewInt(-10_000_000)),
		},
		{
			OperationIdentifier: &types.OperationIdentifier{Index: 1},
			RelatedOperations: []*types.OperationIdentifier{
				{Index: 0},
			},
			Type:    opExport,
			Account: pAccountIdentifier,
			Amount:  mapper.AtomicAvaxAmount(big.NewInt(9_719_250)),
		},
	}

	metadataOptions := map[string]interface{}{
		"atomic_tx_gas":     11230.,
		"from":              cAccountIdentifier.Address,
		"destination_chain": constants.PChain.String(),
	}

	suggestedFeeValue := "280750"

	payloadsMetadata := map[string]interface{}{
		"network_id":           float64(avalancheNetworkID),
		"c_chain_id":           cChainID.String(),
		"destination_chain":    constants.PChain.String(),
		"destination_chain_id": pChainID.String(),
		"nonce":                float64(nonce),
	}

	signers := []*types.AccountIdentifier{cAccountIdentifier}
	exportSigners := buildRosettaSignerJSON([]string{""}, signers)

	unsignedExportTx := "0x000000000001000000057fc93d85c6d62c5b2ac0b519c87010ea5294012d1e407030d6acd0021cac10d50000000000000000000000000000000000000000000000000000000000000000000000013158e80abd5a1e1aa716003c9db096792c37962100000000009896803d9bdac0ed1d761330cf680efdeb1a42159eb387d6d2950c96f7d28f61bbe2aa0000000000000030000000013d9bdac0ed1d761330cf680efdeb1a42159eb387d6d2950c96f7d28f61bbe2aa000000070000000000944dd20000000000000000000000010000000176da56a4600f1ba6f40fc3735f71e3f06c31d7590000000024739402"
	unsignedExportTxHash, _ := hex.DecodeString("75afdcba5bf36457ba9edd65b07f40dcd3111d3c98a53550025af931b7500a7b")

	signingPayloads := []*types.SigningPayload{
		{
			AccountIdentifier: cAccountIdentifier,
			Bytes:             unsignedExportTxHash,
			SignatureType:     types.EcdsaRecovery,
		},
	}

	wrappedTxFormat := `{"tx":"%s","signers":%s,"destination_chain":"%s","destination_chain_id":"%s"}`
	wrappedUnsignedExportTx := fmt.Sprintf(wrappedTxFormat, unsignedExportTx, exportSigners, constants.PChain.String(), pChainID.String())

	signedExportTxSignature, _ := hex.DecodeString("2acfc2cedd3c42978728518b13cc84a64f23784af591516e8dfe0cce544bc63c370ca6d64b2550f12f56a800b8a73ff8573131bf54e584de38c91fc14dd7336801")
	signedExportTx := "0x000000000001000000057fc93d85c6d62c5b2ac0b519c87010ea5294012d1e407030d6acd0021cac10d50000000000000000000000000000000000000000000000000000000000000000000000013158e80abd5a1e1aa716003c9db096792c37962100000000009896803d9bdac0ed1d761330cf680efdeb1a42159eb387d6d2950c96f7d28f61bbe2aa0000000000000030000000013d9bdac0ed1d761330cf680efdeb1a42159eb387d6d2950c96f7d28f61bbe2aa000000070000000000944dd20000000000000000000000010000000176da56a4600f1ba6f40fc3735f71e3f06c31d7590000000100000009000000012acfc2cedd3c42978728518b13cc84a64f23784af591516e8dfe0cce544bc63c370ca6d64b2550f12f56a800b8a73ff8573131bf54e584de38c91fc14dd733680149056c11"
	signedExportTxHash := "pkSEF4YvVo6YjirHfmWvt9j2zrWdzEAGckDKrbyq1WbPtWdAX"

	signatures := []*types.Signature{
		{
			SigningPayload: &types.SigningPayload{
				AccountIdentifier: cAccountIdentifier,
				Bytes:             unsignedExportTxHash,
				SignatureType:     types.EcdsaRecovery,
			},
			SignatureType: types.EcdsaRecovery,
			Bytes:         signedExportTxSignature,
		},
	}

	wrappedSignedExportTx := fmt.Sprintf(wrappedTxFormat, signedExportTx, exportSigners, constants.PChain.String(), pChainID.String())

	ctx := context.Background()
	clientMock := &mocks.Client{}
	backend := NewBackend(clientMock, avaxAssetID, avalancheNetworkID)

	t.Run("preprocess endpoint", func(t *testing.T) {
		req := &types.ConstructionPreprocessRequest{
			NetworkIdentifier: networkIdentifier,
			Operations:        exportOperations,
		}

		resp, apiErr := backend.ConstructionPreprocess(ctx, req)

		assert.Nil(t, apiErr)
		assert.Equal(t, metadataOptions, resp.Options)

		clientMock.AssertExpectations(t)
	})

	t.Run("metadata endpoint", func(t *testing.T) {
		req := &types.ConstructionMetadataRequest{
			NetworkIdentifier: networkIdentifier,
			Options:           metadataOptions,
		}

		clientMock.On("GetBlockchainID", ctx, constants.CChain.String()).Return(cChainID, nil)
		clientMock.On("GetBlockchainID", ctx, constants.PChain.String()).Return(pChainID, nil)
		clientMock.
			On("NonceAt", ctx, ethcommon.HexToAddress(cAccountIdentifier.Address), (*big.Int)(nil)).
			Return(nonce, nil)
		clientMock.On("EstimateBaseFee", ctx).Return(big.NewInt(25_000_000_000), nil)

		resp, apiErr := backend.ConstructionMetadata(ctx, req)

		assert.Nil(t, apiErr)
		assert.Equal(t, payloadsMetadata, resp.Metadata)
		assert.Equal(t, suggestedFeeValue, resp.SuggestedFee[0].Value)

		clientMock.AssertExpectations(t)
	})

	t.Run("payloads endpoint", func(t *testing.T) {
		req := &types.ConstructionPayloadsRequest{
			NetworkIdentifier: networkIdentifier,
			Metadata:          payloadsMetadata,
			Operations:        exportOperations,
		}

		resp, apiErr := backend.ConstructionPayloads(ctx, req)

		assert.Nil(t, apiErr)
		assert.Equal(t, wrappedUnsignedExportTx, resp.UnsignedTransaction)
		assert.Equal(t, signingPayloads, resp.Payloads)

		clientMock.AssertExpectations(t)
	})

	t.Run("parse endpoint (unsigned)", func(t *testing.T) {
		req := &types.ConstructionParseRequest{
			NetworkIdentifier: networkIdentifier,
			Transaction:       wrappedUnsignedExportTx,
			Signed:            false,
		}

		resp, apiErr := backend.ConstructionParse(ctx, req)

		assert.Nil(t, apiErr)
		assert.Nil(t, resp.AccountIdentifierSigners)
		assert.Equal(t, exportOperations, resp.Operations)

		clientMock.AssertExpectations(t)
	})

	t.Run("combine endpoint", func(t *testing.T) {
		req := &types.ConstructionCombineRequest{
			NetworkIdentifier:   networkIdentifier,
			UnsignedTransaction: wrappedUnsignedExportTx,
			Signatures:          signatures,
		}

		resp, apiErr := backend.ConstructionCombine(ctx, req)

		assert.Nil(t, apiErr)
		assert.Equal(t, wrappedSignedExportTx, resp.SignedTransaction)

		clientMock.AssertExpectations(t)
	})

	t.Run("parse (signed) endpoint", func(t *testing.T) {
		req := &types.ConstructionParseRequest{
			NetworkIdentifier: networkIdentifier,
			Transaction:       wrappedSignedExportTx,
			Signed:            true,
		}

		resp, apiErr := backend.ConstructionParse(ctx, req)

		assert.Nil(t, apiErr)
		assert.Equal(t, signers, resp.AccountIdentifierSigners)
		assert.Equal(t, exportOperations, resp.Operations)

		clientMock.AssertExpectations(t)
	})

	t.Run("hash endpoint", func(t *testing.T) {
		resp, err := backend.ConstructionHash(ctx, &types.ConstructionHashRequest{
			NetworkIdentifier: networkIdentifier,
			SignedTransaction: wrappedSignedExportTx,
		})
		assert.Nil(t, err)
		assert.Equal(t, signedExportTxHash, resp.TransactionIdentifier.Hash)

		clientMock.AssertExpectations(t)
	})

	t.Run("submit endpoint", func(t *testing.T) {
		signedTxBytes, _ := formatting.Decode(formatting.Hex, signedExportTx)
		txID, _ := ids.FromString(signedExportTxHash)

		clientMock.On("IssueTx", ctx, signedTxBytes).Return(txID, nil)

		resp, apiErr := backend.ConstructionSubmit(ctx, &types.ConstructionSubmitRequest{
			NetworkIdentifier: networkIdentifier,
			SignedTransaction: wrappedSignedExportTx,
		})

		assert.Nil(t, apiErr)
		assert.Equal(t, signedExportTxHash, resp.TransactionIdentifier.Hash)

		clientMock.AssertExpectations(t)
	})
}

func TestImportTxConstruction(t *testing.T) {
	opImport := "IMPORT"

	coinID1 := "23CLURk1Czf1aLui1VdcuWSiDeFskfp3Sn8TQG7t6NKfeQRYDj:2"
	coinID2 := "2QmMXKS6rKQMnEh2XYZ4ZWCJmy8RpD3LyVZWxBG25t4N1JJqxY:1"

	importOperations := []*types.Operation{
		{
			OperationIdentifier: &types.OperationIdentifier{Index: 0},
			Type:                opImport,
			Account:             cAccountBech32Identifier,
			Amount:              mapper.AtomicAvaxAmount(big.NewInt(-15_000_000)),
			CoinChange: &types.CoinChange{
				CoinIdentifier: &types.CoinIdentifier{Identifier: coinID1},
				CoinAction:     types.CoinSpent,
			},
		},
		{
			OperationIdentifier: &types.OperationIdentifier{Index: 1},
			Type:                opImport,
			Account:             cAccountBech32Identifier,
			Amount:              mapper.AtomicAvaxAmount(big.NewInt(-5_000_000)),
			CoinChange: &types.CoinChange{
				CoinIdentifier: &types.CoinIdentifier{Identifier: coinID2},
				CoinAction:     types.CoinSpent,
			},
		},
		{
			OperationIdentifier: &types.OperationIdentifier{Index: 2},
			RelatedOperations: []*types.OperationIdentifier{
				{Index: 0},
				{Index: 1},
			},
			Type:    opImport,
			Account: cAccountIdentifier,
			Amount:  mapper.AtomicAvaxAmount(big.NewInt(19_692_050)),
		},
	}

	preprocessMetadata := map[string]interface{}{
		"source_chain": constants.PChain.String(),
	}

	metadataOptions := map[string]interface{}{
		"atomic_tx_gas": 12318.,
		"source_chain":  constants.PChain.String(),
	}

	suggestedFeeValue := "307950"

	payloadsMetadata := map[string]interface{}{
		"nonce":           0.,
		"network_id":      float64(avalancheNetworkID),
		"c_chain_id":      cChainID.String(),
		"source_chain_id": pChainID.String(),
	}

	unsignedImportTx := "0x000000000000000000057fc93d85c6d62c5b2ac0b519c87010ea5294012d1e407030d6acd0021cac10d500000000000000000000000000000000000000000000000000000000000000000000000288ae5dd070e6d74f16c26358cd4a8f43746d4d338b5b75b668741c6d95816af5000000023d9bdac0ed1d761330cf680efdeb1a42159eb387d6d2950c96f7d28f61bbe2aa000000050000000000e4e1c00000000100000000b9a824340e1b94f27500cdfcbf8eaa9d4ee5e57b2823cb8b158de17689916c74000000013d9bdac0ed1d761330cf680efdeb1a42159eb387d6d2950c96f7d28f61bbe2aa0000000500000000004c4b400000000100000000000000013158e80abd5a1e1aa716003c9db096792c37962100000000012c7a123d9bdac0ed1d761330cf680efdeb1a42159eb387d6d2950c96f7d28f61bbe2aa00000000c67b0534"

	unsignedImportTxHash, _ := hex.DecodeString("33f98143f7f061e262e0fabca57b7f0dc110a79073ed263fc900ebdd0c1fe6fc")

	signingPayloads := []*types.SigningPayload{
		{
			AccountIdentifier: cAccountBech32Identifier,
			Bytes:             unsignedImportTxHash,
			SignatureType:     types.EcdsaRecovery,
		},
		{
			AccountIdentifier: cAccountBech32Identifier,
			Bytes:             unsignedImportTxHash,
			SignatureType:     types.EcdsaRecovery,
		},
	}

	signers := []*types.AccountIdentifier{cAccountBech32Identifier, cAccountBech32Identifier}
	importSigners := buildRosettaSignerJSON([]string{coinID1, coinID2}, signers)

	wrappedUnsignedImportTx := `{"tx":"` + unsignedImportTx + `","signers":` + importSigners + `}`

	signedImportTxSignature, _ := hex.DecodeString("a06d20d1d175b1e1d2b6e647ab5321717967de7e9367c28df8c0e20634ec7827019fe38e8d4f123f8e5286f3236db8dbb419e264628e2f17330a6c8da60d342401")
	signedImportTx := "0x000000000000000000057fc93d85c6d62c5b2ac0b519c87010ea5294012d1e407030d6acd0021cac10d500000000000000000000000000000000000000000000000000000000000000000000000288ae5dd070e6d74f16c26358cd4a8f43746d4d338b5b75b668741c6d95816af5000000023d9bdac0ed1d761330cf680efdeb1a42159eb387d6d2950c96f7d28f61bbe2aa000000050000000000e4e1c00000000100000000b9a824340e1b94f27500cdfcbf8eaa9d4ee5e57b2823cb8b158de17689916c74000000013d9bdac0ed1d761330cf680efdeb1a42159eb387d6d2950c96f7d28f61bbe2aa0000000500000000004c4b400000000100000000000000013158e80abd5a1e1aa716003c9db096792c37962100000000012c7a123d9bdac0ed1d761330cf680efdeb1a42159eb387d6d2950c96f7d28f61bbe2aa000000020000000900000001a06d20d1d175b1e1d2b6e647ab5321717967de7e9367c28df8c0e20634ec7827019fe38e8d4f123f8e5286f3236db8dbb419e264628e2f17330a6c8da60d3424010000000900000001a06d20d1d175b1e1d2b6e647ab5321717967de7e9367c28df8c0e20634ec7827019fe38e8d4f123f8e5286f3236db8dbb419e264628e2f17330a6c8da60d342401dc68b1fc"
	signedImportTxHash := "2Rz6T1gteozqm5sCG52hDHk6m4iMY65R1LWfBCuPo3f595yrT7"

	wrappedSignedImportTx := `{"tx":"` + signedImportTx + `","signers":` + importSigners + `}`

	signature := &types.Signature{
		SigningPayload: &types.SigningPayload{
			AccountIdentifier: pAccountIdentifier,
			Bytes:             unsignedImportTxHash,
			SignatureType:     types.EcdsaRecovery,
		},
		SignatureType: types.EcdsaRecovery,
		Bytes:         signedImportTxSignature,
	}

	// two signatures, one for each input
	signatures := []*types.Signature{signature, signature}

	ctx := context.Background()
	clientMock := &mocks.Client{}
	backend := NewBackend(clientMock, avaxAssetID, avalancheNetworkID)

	t.Run("preprocess endpoint", func(t *testing.T) {
		req := &types.ConstructionPreprocessRequest{
			NetworkIdentifier: networkIdentifier,
			Operations:        importOperations,
			Metadata:          preprocessMetadata,
		}

		resp, apiErr := backend.ConstructionPreprocess(ctx, req)

		assert.Nil(t, apiErr)
		assert.Equal(t, metadataOptions, resp.Options)

		clientMock.AssertExpectations(t)
	})

	t.Run("metadata endpoint", func(t *testing.T) {
		req := &types.ConstructionMetadataRequest{
			NetworkIdentifier: networkIdentifier,
			Options:           metadataOptions,
		}

		clientMock.On("GetBlockchainID", ctx, constants.CChain.String()).Return(cChainID, nil)
		clientMock.On("GetBlockchainID", ctx, constants.PChain.String()).Return(pChainID, nil)
		clientMock.On("EstimateBaseFee", ctx).Return(big.NewInt(25_000_000_000), nil)

		resp, apiErr := backend.ConstructionMetadata(ctx, req)

		assert.Nil(t, apiErr)
		assert.Equal(t, payloadsMetadata, resp.Metadata)
		assert.Equal(t, suggestedFeeValue, resp.SuggestedFee[0].Value)

		clientMock.AssertExpectations(t)
	})

	t.Run("payloads endpoint", func(t *testing.T) {
		req := &types.ConstructionPayloadsRequest{
			NetworkIdentifier: networkIdentifier,
			Metadata:          payloadsMetadata,
			Operations:        importOperations,
		}

		resp, apiErr := backend.ConstructionPayloads(ctx, req)

		assert.Nil(t, apiErr)
		assert.Equal(t, wrappedUnsignedImportTx, resp.UnsignedTransaction)
		assert.Equal(t, signingPayloads, resp.Payloads)

		clientMock.AssertExpectations(t)
	})

	t.Run("parse endpoint (unsigned)", func(t *testing.T) {
		req := &types.ConstructionParseRequest{
			NetworkIdentifier: networkIdentifier,
			Transaction:       wrappedUnsignedImportTx,
			Signed:            false,
		}

		resp, apiErr := backend.ConstructionParse(ctx, req)

		assert.Nil(t, apiErr)
		assert.Nil(t, resp.AccountIdentifierSigners)
		assert.Equal(t, importOperations, resp.Operations)
	})

	t.Run("combine endpoint", func(t *testing.T) {
		req := &types.ConstructionCombineRequest{
			NetworkIdentifier:   networkIdentifier,
			UnsignedTransaction: wrappedUnsignedImportTx,
			Signatures:          signatures,
		}

		resp, apiErr := backend.ConstructionCombine(ctx, req)

		assert.Nil(t, apiErr)
		assert.Equal(t, wrappedSignedImportTx, resp.SignedTransaction)

		clientMock.AssertExpectations(t)
	})

	t.Run("parse (signed) endpoint", func(t *testing.T) {
		req := &types.ConstructionParseRequest{
			NetworkIdentifier: networkIdentifier,
			Transaction:       wrappedSignedImportTx,
			Signed:            true,
		}

		resp, apiErr := backend.ConstructionParse(ctx, req)

		assert.Nil(t, apiErr)
		assert.Equal(t, signers, resp.AccountIdentifierSigners)
		assert.Equal(t, importOperations, resp.Operations)

		clientMock.AssertExpectations(t)
	})

	t.Run("hash endpoint", func(t *testing.T) {
		resp, err := backend.ConstructionHash(ctx, &types.ConstructionHashRequest{
			NetworkIdentifier: networkIdentifier,
			SignedTransaction: wrappedSignedImportTx,
		})
		assert.Nil(t, err)
		assert.Equal(t, signedImportTxHash, resp.TransactionIdentifier.Hash)

		clientMock.AssertExpectations(t)
	})

	t.Run("submit endpoint", func(t *testing.T) {
		signedTxBytes, _ := formatting.Decode(formatting.Hex, signedImportTx)
		txID, _ := ids.FromString(signedImportTxHash)

		clientMock.On("IssueTx", ctx, signedTxBytes).Return(txID, nil)

		resp, apiErr := backend.ConstructionSubmit(ctx, &types.ConstructionSubmitRequest{
			NetworkIdentifier: networkIdentifier,
			SignedTransaction: wrappedSignedImportTx,
		})

		assert.Nil(t, apiErr)
		assert.Equal(t, signedImportTxHash, resp.TransactionIdentifier.Hash)

		clientMock.AssertExpectations(t)
	})
}
