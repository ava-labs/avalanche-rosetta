package pchain

import (
	"context"
	"encoding/hex"
	"math/big"
	"testing"
	"time"

	"github.com/ava-labs/avalanchego/api/info"
	"github.com/ava-labs/avalanchego/ids"
	ajson "github.com/ava-labs/avalanchego/utils/json"
	"github.com/coinbase/rosetta-sdk-go/types"
	"github.com/stretchr/testify/assert"

	"github.com/ava-labs/avalanche-rosetta/mapper"
	mocks "github.com/ava-labs/avalanche-rosetta/mocks/client"
	"github.com/ava-labs/avalanche-rosetta/service"
)

var (
	pChainNetworkIdentifier = &types.NetworkIdentifier{
		Blockchain: service.BlockchainName,
		Network:    mapper.FujiNetwork,
		SubNetworkIdentifier: &types.SubNetworkIdentifier{
			Network: mapper.PChainNetworkIdentifier,
		},
	}

	cAccountIdentifier = &types.AccountIdentifier{Address: "C-fuji123zu6qwhtd9qdd45ryu3j0qtr325gjgddys6u8"}
	pAccountIdentifier = &types.AccountIdentifier{Address: "P-fuji123zu6qwhtd9qdd45ryu3j0qtr325gjgddys6u8"}
	stakeRewardAccount = &types.AccountIdentifier{Address: "P-fuji1ea7dxk8zazpyf8tgc8yg3xyfatey0deqvg9pv2"}

	cChainID, _ = ids.FromString("yH8D7ThNJkxmtkuv2jgBa4P1Rn3Qpr4pPr7QYNfcdoS6k6HWp")
	pChainID    = ids.Empty

	nodeID = "NodeID-A68C3atsKqTNMR9Qra8FbqpFkMgHV5Dtu"

	networkID = 5

	opTypeInput  = "INPUT"
	opTypeImport = "IMPORT"
	opTypeExport = "EXPORT"
	opTypeOutput = "OUTPUT"
	opTypeStake  = "STAKE"

	txFee = 1_000_000
)

func TestConstructionDerive(t *testing.T) {
	backend := NewBackend(nil, nil, pChainNetworkIdentifier)

	t.Run("p-chain address", func(t *testing.T) {
		src := "02e0d4392cfa224d4be19db416b3cf62e90fb2b7015e7b62a95c8cb490514943f6"
		b, _ := hex.DecodeString(src)

		resp, err := backend.ConstructionDerive(
			context.Background(),
			&types.ConstructionDeriveRequest{
				NetworkIdentifier: &types.NetworkIdentifier{
					Network: mapper.FujiNetwork,
					SubNetworkIdentifier: &types.SubNetworkIdentifier{
						Network: mapper.PChainNetworkIdentifier,
					},
				},
				PublicKey: &types.PublicKey{
					Bytes:     b,
					CurveType: types.Secp256k1,
				},
			},
		)
		assert.Nil(t, err)
		assert.Equal(
			t,
			"P-fuji15f9g0h5xkr5cp47n6u3qxj6yjtzzzrdr23a3tl",
			resp.AccountIdentifier.Address,
		)
	})
}

func TestExportTxConstruction(t *testing.T) {
	opExportAvax := "EXPORT_AVAX"

	operations := []*types.Operation{
		{
			OperationIdentifier: &types.OperationIdentifier{Index: 0},
			RelatedOperations:   nil,
			Type:                opExportAvax,
			Account:             pAccountIdentifier,
			Amount:              mapper.AtomicAvaxAmount(big.NewInt(-1_000_000_000)),
			CoinChange: &types.CoinChange{
				CoinIdentifier: &types.CoinIdentifier{Identifier: "2ryRVCwNSjEinTViuvDkzX41uQzx3g4babXxZMD46ZV1a9X4Eg:0"},
				CoinAction:     "coin_spent",
			},
			Metadata: map[string]interface{}{
				"type": opTypeInput,
			},
		},
		{
			OperationIdentifier: &types.OperationIdentifier{Index: 1},
			Type:                opExportAvax,
			Account:             cAccountIdentifier,
			Amount:              mapper.AtomicAvaxAmount(big.NewInt(999_000_000)),
			Metadata: map[string]interface{}{
				"type": opTypeExport,
			},
		},
	}

	preprocessMetadata := map[string]interface{}{
		"destination_chain": "C",
	}

	metadataOptions := map[string]interface{}{
		"destination_chain": "C",
		"type":              opExportAvax,
	}

	payloadsMetadata := map[string]interface{}{
		"network_id":           float64(networkID),
		"destination_chain":    "C",
		"destination_chain_id": cChainID.String(),
		"blockchain_id":        pChainID.String(),
	}

	ctx := context.Background()
	clientMock := &mocks.PChainClient{}
	backend := NewBackend(clientMock, nil, pChainNetworkIdentifier)

	t.Run("preprocess endpoint", func(t *testing.T) {
		resp, err := backend.ConstructionPreprocess(
			ctx,
			&types.ConstructionPreprocessRequest{
				NetworkIdentifier: pChainNetworkIdentifier,
				Operations:        operations,
				Metadata:          preprocessMetadata,
			},
		)
		assert.Nil(t, err)
		assert.Equal(t, metadataOptions, resp.Options)

		clientMock.AssertExpectations(t)
	})

	t.Run("metadata endpoint", func(t *testing.T) {
		clientMock.On("GetNetworkID", ctx).Return(uint32(networkID), nil)
		clientMock.On("GetTxFee", ctx).Return(&info.GetTxFeeResponse{TxFee: ajson.Uint64(txFee)}, nil)
		clientMock.On("GetBlockchainID", ctx, mapper.PChainNetworkIdentifier).Return(pChainID, nil)
		clientMock.On("GetBlockchainID", ctx, mapper.CChainNetworkIdentifier).Return(cChainID, nil)

		resp, err := backend.ConstructionMetadata(
			ctx,
			&types.ConstructionMetadataRequest{
				NetworkIdentifier: pChainNetworkIdentifier,
				Options:           metadataOptions,
			},
		)
		assert.Nil(t, err)
		assert.Equal(t, payloadsMetadata, resp.Metadata)

		clientMock.AssertExpectations(t)
	})

	t.Run("payloads endpoint", func(t *testing.T) {})

	t.Run("parse endpoint (unsigned)", func(t *testing.T) {})

	t.Run("combine endpoint", func(t *testing.T) {})

	t.Run("parse endpoint (signed)", func(t *testing.T) {})

	t.Run("hash endpoint", func(t *testing.T) {})

	t.Run("submit endpoint", func(t *testing.T) {})
}

func TestImportTxConstruction(t *testing.T) {
	opImportAvax := "IMPORT_AVAX"

	operations := []*types.Operation{
		{
			OperationIdentifier: &types.OperationIdentifier{Index: 0},
			RelatedOperations:   nil,
			Type:                opImportAvax,
			Account:             cAccountIdentifier,
			Amount:              mapper.AtomicAvaxAmount(big.NewInt(-1_000_000_000)),
			CoinChange: &types.CoinChange{
				CoinIdentifier: &types.CoinIdentifier{Identifier: "2ryRVCwNSjEinTViuvDkzX41uQzx3g4babXxZMD46ZV1a9X4Eg:0"},
				CoinAction:     "coin_spent",
			},
			Metadata: map[string]interface{}{
				"type": opTypeImport,
			},
		},
		{
			OperationIdentifier: &types.OperationIdentifier{Index: 1},
			Type:                opImportAvax,
			Account:             pAccountIdentifier,
			Amount:              mapper.AtomicAvaxAmount(big.NewInt(999_000_000)),
			Metadata: map[string]interface{}{
				"type": opTypeOutput,
			},
		},
	}

	preprocessMetadata := map[string]interface{}{
		"source_chain": "C",
	}

	metadataOptions := map[string]interface{}{
		"source_chain": "C",
		"type":         opImportAvax,
	}

	payloadsMetadata := map[string]interface{}{
		"network_id":      float64(networkID),
		"source_chain_id": cChainID.String(),
		"blockchain_id":   pChainID.String(),
	}

	ctx := context.Background()
	clientMock := &mocks.PChainClient{}
	backend := NewBackend(clientMock, nil, pChainNetworkIdentifier)

	t.Run("preprocess endpoint", func(t *testing.T) {
		resp, err := backend.ConstructionPreprocess(
			ctx,
			&types.ConstructionPreprocessRequest{
				NetworkIdentifier: pChainNetworkIdentifier,
				Operations:        operations,
				Metadata:          preprocessMetadata,
			},
		)
		assert.Nil(t, err)
		assert.Equal(t, metadataOptions, resp.Options)

		clientMock.AssertExpectations(t)
	})

	t.Run("metadata endpoint", func(t *testing.T) {
		clientMock.On("GetNetworkID", ctx).Return(uint32(networkID), nil)
		clientMock.On("GetTxFee", ctx).Return(&info.GetTxFeeResponse{TxFee: ajson.Uint64(txFee)}, nil)
		clientMock.On("GetBlockchainID", ctx, mapper.PChainNetworkIdentifier).Return(pChainID, nil)
		clientMock.On("GetBlockchainID", ctx, mapper.CChainNetworkIdentifier).Return(cChainID, nil)

		resp, err := backend.ConstructionMetadata(
			ctx,
			&types.ConstructionMetadataRequest{
				NetworkIdentifier: pChainNetworkIdentifier,
				Options:           metadataOptions,
			},
		)
		assert.Nil(t, err)
		assert.Equal(t, payloadsMetadata, resp.Metadata)

		clientMock.AssertExpectations(t)
	})

	t.Run("payloads endpoint", func(t *testing.T) {})

	t.Run("parse endpoint (unsigned)", func(t *testing.T) {})

	t.Run("combine endpoint", func(t *testing.T) {})

	t.Run("parse endpoint (signed)", func(t *testing.T) {})

	t.Run("hash endpoint", func(t *testing.T) {})

	t.Run("submit endpoint", func(t *testing.T) {})
}

func TestAddValidatorTxConstruction(t *testing.T) {
	opAddValidator := "ADD_VALIDATOR"
	startTime := uint64(time.Now().Unix())
	endTime := uint64(time.Now().Add(14 * 24 * time.Hour).Unix())
	weight := uint64(2_000_000_000_000)
	shares := uint32(10000)

	operations := []*types.Operation{
		{
			OperationIdentifier: &types.OperationIdentifier{Index: 0},
			RelatedOperations:   nil,
			Type:                opAddValidator,
			Account:             cAccountIdentifier,
			Amount:              mapper.AtomicAvaxAmount(big.NewInt(-2_000_000_000_000)),
			CoinChange: &types.CoinChange{
				CoinIdentifier: &types.CoinIdentifier{Identifier: "2ryRVCwNSjEinTViuvDkzX41uQzx3g4babXxZMD46ZV1a9X4Eg:0"},
				CoinAction:     "coin_spent",
			},
			Metadata: map[string]interface{}{
				"type": opTypeInput,
			},
		},
		{
			OperationIdentifier: &types.OperationIdentifier{Index: 1},
			Type:                opAddValidator,
			Account:             pAccountIdentifier,
			Amount:              mapper.AtomicAvaxAmount(big.NewInt(2_000_000_000_000)),
			Metadata: map[string]interface{}{
				"type": opTypeStake,
			},
		},
	}

	preprocessMetadata := map[string]interface{}{
		"node_id":          nodeID,
		"start":            startTime,
		"end":              endTime,
		"weight":           weight,
		"shares":           shares,
		"reward_addresses": []string{stakeRewardAccount.Address},
	}

	metadataOptions := map[string]interface{}{
		"type":             opAddValidator,
		"node_id":          nodeID,
		"start":            startTime,
		"end":              endTime,
		"weight":           weight,
		"shares":           shares,
		"reward_addresses": []string{stakeRewardAccount.Address},
	}

	payloadsMetadata := map[string]interface{}{
		"network_id":       float64(networkID),
		"blockchain_id":    pChainID.String(),
		"node_id":          nodeID,
		"start":            float64(startTime),
		"end":              float64(endTime),
		"weight":           float64(weight),
		"shares":           float64(shares),
		"locktime":         0.0,
		"threshold":        0.0,
		"memo":             "",
		"reward_addresses": []interface{}{stakeRewardAccount.Address},
	}

	ctx := context.Background()
	clientMock := &mocks.PChainClient{}
	backend := NewBackend(clientMock, nil, pChainNetworkIdentifier)

	t.Run("preprocess endpoint", func(t *testing.T) {
		resp, err := backend.ConstructionPreprocess(
			ctx,
			&types.ConstructionPreprocessRequest{
				NetworkIdentifier: pChainNetworkIdentifier,
				Operations:        operations,
				Metadata:          preprocessMetadata,
			},
		)
		assert.Nil(t, err)
		assert.Equal(t, metadataOptions, resp.Options)

		clientMock.AssertExpectations(t)
	})

	t.Run("metadata endpoint", func(t *testing.T) {
		clientMock.On("GetNetworkID", ctx).Return(uint32(networkID), nil)
		clientMock.On("GetBlockchainID", ctx, mapper.PChainNetworkIdentifier).Return(pChainID, nil)

		resp, err := backend.ConstructionMetadata(
			ctx,
			&types.ConstructionMetadataRequest{
				NetworkIdentifier: pChainNetworkIdentifier,
				Options:           metadataOptions,
			},
		)
		assert.Nil(t, err)
		assert.Equal(t, payloadsMetadata, resp.Metadata)

		clientMock.AssertExpectations(t)
	})

	t.Run("payloads endpoint", func(t *testing.T) {})

	t.Run("parse endpoint (unsigned)", func(t *testing.T) {})

	t.Run("combine endpoint", func(t *testing.T) {})

	t.Run("parse endpoint (signed)", func(t *testing.T) {})

	t.Run("hash endpoint", func(t *testing.T) {})

	t.Run("submit endpoint", func(t *testing.T) {})
}

func TestAddDelegatorTxConstruction(t *testing.T) {
	opAddDelegator := "ADD_DELEGATOR"
	startTime := uint64(time.Now().Unix())
	endTime := uint64(time.Now().Add(14 * 24 * time.Hour).Unix())
	weight := uint64(25_000_000_000)

	operations := []*types.Operation{
		{
			OperationIdentifier: &types.OperationIdentifier{Index: 0},
			RelatedOperations:   nil,
			Type:                opAddDelegator,
			Account:             cAccountIdentifier,
			Amount:              mapper.AtomicAvaxAmount(big.NewInt(-25_000_000_000)),
			CoinChange: &types.CoinChange{
				CoinIdentifier: &types.CoinIdentifier{Identifier: "2ryRVCwNSjEinTViuvDkzX41uQzx3g4babXxZMD46ZV1a9X4Eg:0"},
				CoinAction:     "coin_spent",
			},
			Metadata: map[string]interface{}{
				"type": opTypeInput,
			},
		},
		{
			OperationIdentifier: &types.OperationIdentifier{Index: 1},
			Type:                opAddDelegator,
			Account:             pAccountIdentifier,
			Amount:              mapper.AtomicAvaxAmount(big.NewInt(25_000_000_000)),
			Metadata: map[string]interface{}{
				"type": opTypeStake,
			},
		},
	}

	preprocessMetadata := map[string]interface{}{
		"node_id":          nodeID,
		"start":            startTime,
		"end":              endTime,
		"weight":           weight,
		"reward_addresses": []string{stakeRewardAccount.Address},
	}

	metadataOptions := map[string]interface{}{
		"type":             opAddDelegator,
		"node_id":          nodeID,
		"start":            startTime,
		"end":              endTime,
		"weight":           weight,
		"reward_addresses": []string{stakeRewardAccount.Address},
	}

	payloadsMetadata := map[string]interface{}{
		"network_id":       float64(networkID),
		"blockchain_id":    pChainID.String(),
		"node_id":          nodeID,
		"start":            float64(startTime),
		"end":              float64(endTime),
		"weight":           float64(weight),
		"shares":           0.0,
		"locktime":         0.0,
		"threshold":        0.0,
		"memo":             "",
		"reward_addresses": []interface{}{stakeRewardAccount.Address},
	}

	ctx := context.Background()
	clientMock := &mocks.PChainClient{}
	backend := NewBackend(clientMock, nil, pChainNetworkIdentifier)

	t.Run("preprocess endpoint", func(t *testing.T) {
		resp, err := backend.ConstructionPreprocess(
			ctx,
			&types.ConstructionPreprocessRequest{
				NetworkIdentifier: pChainNetworkIdentifier,
				Operations:        operations,
				Metadata:          preprocessMetadata,
			},
		)
		assert.Nil(t, err)
		assert.Equal(t, metadataOptions, resp.Options)

		clientMock.AssertExpectations(t)
	})

	t.Run("metadata endpoint", func(t *testing.T) {
		clientMock.On("GetNetworkID", ctx).Return(uint32(networkID), nil)
		clientMock.On("GetBlockchainID", ctx, mapper.PChainNetworkIdentifier).Return(pChainID, nil)

		resp, err := backend.ConstructionMetadata(
			ctx,
			&types.ConstructionMetadataRequest{
				NetworkIdentifier: pChainNetworkIdentifier,
				Options:           metadataOptions,
			},
		)
		assert.Nil(t, err)
		assert.Equal(t, payloadsMetadata, resp.Metadata)

		clientMock.AssertExpectations(t)
	})

	t.Run("payloads endpoint", func(t *testing.T) {})

	t.Run("parse endpoint (unsigned)", func(t *testing.T) {})

	t.Run("combine endpoint", func(t *testing.T) {})

	t.Run("parse endpoint (signed)", func(t *testing.T) {})

	t.Run("hash endpoint", func(t *testing.T) {})

	t.Run("submit endpoint", func(t *testing.T) {})
}
