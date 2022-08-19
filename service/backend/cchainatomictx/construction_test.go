package cchainatomictx

import (
	"context"
	"encoding/hex"
	"math/big"
	"testing"

	"github.com/ava-labs/avalanchego/ids"
	"github.com/coinbase/rosetta-sdk-go/types"
	ethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"

	"github.com/ava-labs/avalanche-rosetta/mapper"
	mocks "github.com/ava-labs/avalanche-rosetta/mocks/client"
	"github.com/ava-labs/avalanche-rosetta/service"
)

var (
	networkIdentifier = &types.NetworkIdentifier{
		Blockchain: service.BlockchainName,
		Network:    mapper.FujiNetwork,
	}

	cAccountIdentifier       = &types.AccountIdentifier{Address: "0x3158e80abD5A1e1aa716003C9Db096792C379621"}
	cAccountBech32Identifier = &types.AccountIdentifier{Address: "C-fuji1wmd9dfrqpud6daq0cde47u0r7pkrr46ep60399"}
	pAccountIdentifier       = &types.AccountIdentifier{Address: "P-fuji1wmd9dfrqpud6daq0cde47u0r7pkrr46ep60399"}

	cChainID, _ = ids.FromString("yH8D7ThNJkxmtkuv2jgBa4P1Rn3Qpr4pPr7QYNfcdoS6k6HWp")
	pChainID    = ids.Empty

	networkID = 5

	avaxAssetID, _ = ids.FromString("U8iRqJoiJm8xZHAacmvYyZVwqQx6uDNtQeP3CQ6fcgQk3JqnK")
)

func TestConstructionDerive(t *testing.T) {
	backend := NewBackend(nil, ids.Empty)

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
		"destination_chain": "P",
	}

	suggestedFeeValue := "280750"

	payloadsMetadata := map[string]interface{}{
		"network_id":           float64(networkID),
		"c_chain_id":           cChainID.String(),
		"destination_chain":    "P",
		"destination_chain_id": pChainID.String(),
		"nonce":                float64(nonce),
	}

	ctx := context.Background()
	clientMock := &mocks.Client{}
	backend := NewBackend(clientMock, avaxAssetID)

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

		clientMock.On("GetNetworkID", ctx).Return(uint32(networkID), nil)
		clientMock.On("GetBlockchainID", ctx, "C").Return(cChainID, nil)
		clientMock.On("GetBlockchainID", ctx, "P").Return(pChainID, nil)
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

	t.Run("payloads endpoint", func(t *testing.T) {})

	t.Run("parse endpoint (unsigned)", func(t *testing.T) {})

	t.Run("combine endpoint", func(t *testing.T) {})

	t.Run("parse (signed) endpoint", func(t *testing.T) {})

	t.Run("hash endpoint", func(t *testing.T) {})

	t.Run("submit endpoint", func(t *testing.T) {})
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
		"source_chain": "P",
	}

	metadataOptions := map[string]interface{}{
		"atomic_tx_gas": 12318.,
		"source_chain":  "P",
	}

	suggestedFeeValue := "307950"

	payloadsMetadata := map[string]interface{}{
		"nonce":           0.,
		"network_id":      float64(networkID),
		"c_chain_id":      cChainID.String(),
		"source_chain_id": pChainID.String(),
	}

	ctx := context.Background()
	clientMock := &mocks.Client{}
	backend := NewBackend(clientMock, avaxAssetID)

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

		clientMock.On("GetNetworkID", ctx).Return(uint32(networkID), nil)
		clientMock.On("GetBlockchainID", ctx, "C").Return(cChainID, nil)
		clientMock.On("GetBlockchainID", ctx, "P").Return(pChainID, nil)
		clientMock.On("EstimateBaseFee", ctx).Return(big.NewInt(25_000_000_000), nil)

		resp, apiErr := backend.ConstructionMetadata(ctx, req)

		assert.Nil(t, apiErr)
		assert.Equal(t, payloadsMetadata, resp.Metadata)
		assert.Equal(t, suggestedFeeValue, resp.SuggestedFee[0].Value)

		clientMock.AssertExpectations(t)
	})

	t.Run("payloads endpoint", func(t *testing.T) {})

	t.Run("parse endpoint (unsigned)", func(t *testing.T) {})

	t.Run("combine endpoint", func(t *testing.T) {})

	t.Run("parse (signed) endpoint", func(t *testing.T) {})

	t.Run("hash endpoint", func(t *testing.T) {})

	t.Run("submit endpoint", func(t *testing.T) {})
}
