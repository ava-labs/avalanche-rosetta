package service

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"math/big"
	"testing"

	"github.com/ava-labs/coreth/interfaces"
	"github.com/coinbase/rosetta-sdk-go/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/ava-labs/avalanche-rosetta/client"
	"github.com/ava-labs/avalanche-rosetta/mapper"

	rosConst "github.com/ava-labs/avalanche-rosetta/constants"
)

const (
	defaultSymbol          = "TEST"
	defaultDecimals        = 18
	defaultContractAddress = "0x30e5449b6712Adf4156c8c474250F6eA4400eB82"
	defaultFromAddress     = "0xe3a5B4d7f79d64088C8d4ef153A7DDe2B2d47309"
	defaultToAddress       = "0x57B414a0332B5CaB885a451c2a28a07d1e9b8a8d"
)

func TestConstructionMetadata(t *testing.T) {
	ctrl := gomock.NewController(t)
	client := client.NewMockClient(ctrl)
	ctx := context.Background()
	skippedBackend := NewMockConstructionBackend(ctrl)
	skippedBackend.EXPECT().ShouldHandleRequest(gomock.Any()).Return(false).AnyTimes()

	service := ConstructionService{
		config:                &Config{Mode: ModeOnline},
		client:                client,
		pChainBackend:         skippedBackend,
		cChainAtomicTxBackend: skippedBackend,
	}

	t.Run("unavailable in offline mode", func(t *testing.T) {
		service := ConstructionService{
			config: &Config{
				Mode: ModeOffline,
			},
		}

		resp, err := service.ConstructionMetadata(
			context.Background(),
			&types.ConstructionMetadataRequest{},
		)
		require.Nil(t, resp)
		require.Equal(t, ErrUnavailableOffline.Code, err.Code)
	})

	t.Run("requires from address", func(t *testing.T) {
		resp, err := service.ConstructionMetadata(
			context.Background(),
			&types.ConstructionMetadataRequest{},
		)
		require.Nil(t, resp)
		require.Equal(t, ErrInvalidInput.Code, err.Code)
		require.Equal(t, "from address is not provided", err.Details["error"])
	})

	t.Run("basic native transfer", func(t *testing.T) {
		to := common.HexToAddress(defaultToAddress)
		client.EXPECT().NonceAt(
			ctx,
			common.HexToAddress("0xe3a5B4d7f79d64088C8d4ef153A7DDe2B2d47309"),
			(*big.Int)(nil),
		).Return(
			uint64(0),
			nil,
		)
		client.EXPECT().SuggestGasPrice(
			ctx,
		).Return(
			big.NewInt(1000000000),
			nil,
		)
		client.EXPECT().EstimateGas(
			ctx,
			interfaces.CallMsg{
				From:  common.HexToAddress("0xe3a5B4d7f79d64088C8d4ef153A7DDe2B2d47309"),
				To:    &to,
				Value: big.NewInt(42894881044106498),
			},
		).Return(
			uint64(21001),
			nil,
		)
		input := map[string]interface{}{
			"from":  "0xe3a5B4d7f79d64088C8d4ef153A7DDe2B2d47309",
			"to":    "0x57B414a0332B5CaB885a451c2a28a07d1e9b8a8d",
			"value": "0x9864aac3510d02",
		}
		resp, err := service.ConstructionMetadata(
			ctx,
			&types.ConstructionMetadataRequest{
				Options: input,
			},
		)
		require.Nil(t, err)
		metadata := &metadata{
			GasPrice: big.NewInt(1000000000),
			GasLimit: 21_001,
			Nonce:    0,
		}
		require.Equal(t, &types.ConstructionMetadataResponse{
			Metadata: forceMarshalMap(t, metadata),
			SuggestedFee: []*types.Amount{
				{
					Value:    "21001000000000",
					Currency: mapper.AvaxCurrency,
				},
			},
		}, resp)
	})
	t.Run("basic unwrap transfer", func(t *testing.T) {
		contractAddress := common.HexToAddress(defaultContractAddress)
		client.EXPECT().NonceAt(
			ctx,
			common.HexToAddress(defaultFromAddress),
			(*big.Int)(nil),
		).Return(
			uint64(0),
			nil,
		)
		client.EXPECT().SuggestGasPrice(
			ctx,
		).Return(
			big.NewInt(1000000000),
			nil,
		)
		client.EXPECT().EstimateGas(
			ctx,
			interfaces.CallMsg{
				From: common.HexToAddress(defaultFromAddress),
				To:   &contractAddress,
				Data: common.Hex2Bytes(
					"6e28667100000000000000000000000000000000000000000000000000000000b4d360e30000000000000000000000000000000000000000000000000000000000000000",
				),
			},
		).Return(
			uint64(21001),
			nil,
		)
		currencyMetadata := map[string]interface{}{
			"contractAddress": defaultContractAddress,
		}
		currency := map[string]interface{}{
			"symbol":   defaultSymbol,
			"decimals": defaultDecimals,
			"metadata": currencyMetadata,
		}
		inputMetadata := map[string]interface{}{
			"bridge_unwrap": true,
		}
		input := map[string]interface{}{
			"from":     defaultFromAddress,
			"to":       "0x920eb8ca79f07eb3bfc39c324c8113948ed3104c",
			"value":    "0xb4d360e3",
			"currency": currency,
			"metadata": inputMetadata,
		}
		resp, err := service.ConstructionMetadata(
			ctx,
			&types.ConstructionMetadataRequest{
				Options: input,
			},
		)
		require.Nil(t, err)
		metadata := &metadata{
			GasPrice:       big.NewInt(1000000000),
			GasLimit:       21_001,
			Nonce:          0,
			UnwrapBridgeTx: true,
		}
		require.Equal(t, &types.ConstructionMetadataResponse{
			Metadata: forceMarshalMap(t, metadata),
			SuggestedFee: []*types.Amount{
				{
					Value:    "21001000000000",
					Currency: mapper.AvaxCurrency,
				},
			},
		}, resp)
	})
	t.Run("basic erc20 transfer", func(t *testing.T) {
		contractAddress := common.HexToAddress(defaultContractAddress)
		client.EXPECT().NonceAt(
			ctx,
			common.HexToAddress(defaultFromAddress),
			(*big.Int)(nil),
		).Return(
			uint64(0),
			nil,
		)
		client.EXPECT().SuggestGasPrice(
			ctx,
		).Return(
			big.NewInt(1000000000),
			nil,
		)
		client.EXPECT().EstimateGas(
			ctx,
			interfaces.CallMsg{
				From: common.HexToAddress(defaultFromAddress),
				To:   &contractAddress,
				Data: common.Hex2Bytes(
					"a9059cbb000000000000000000000000920eb8ca79f07eb3bfc39c324c8113948ed3104c00000000000000000000000000000000000000000000000000000000b4d360e3",
				),
			},
		).Return(
			uint64(21001),
			nil,
		)
		currencyMetadata := map[string]interface{}{
			"contractAddress": defaultContractAddress,
		}
		currency := map[string]interface{}{
			"symbol":   defaultSymbol,
			"decimals": defaultDecimals,
			"metadata": currencyMetadata,
		}
		input := map[string]interface{}{
			"from":     defaultFromAddress,
			"to":       "0x920eb8ca79f07eb3bfc39c324c8113948ed3104c",
			"value":    "0xb4d360e3",
			"currency": currency,
		}
		resp, err := service.ConstructionMetadata(
			ctx,
			&types.ConstructionMetadataRequest{
				Options: input,
			},
		)
		require.Nil(t, err)
		metadata := &metadata{
			GasPrice: big.NewInt(1000000000),
			GasLimit: 21_001,
			Nonce:    0,
		}
		require.Equal(t, &types.ConstructionMetadataResponse{
			Metadata: forceMarshalMap(t, metadata),
			SuggestedFee: []*types.Amount{
				{
					Value:    "21001000000000",
					Currency: mapper.AvaxCurrency,
				},
			},
		}, resp)
	})
}

func TestContructionHash(t *testing.T) {
	ctrl := gomock.NewController(t)
	skippedBackend := NewMockConstructionBackend(ctrl)
	skippedBackend.EXPECT().ShouldHandleRequest(gomock.Any()).Return(false).AnyTimes()

	service := ConstructionService{
		pChainBackend:         skippedBackend,
		cChainAtomicTxBackend: skippedBackend,
	}

	t.Run("no transaction", func(t *testing.T) {
		resp, err := service.ConstructionHash(
			context.Background(),
			&types.ConstructionHashRequest{},
		)
		require.Nil(t, resp)
		require.Equal(t, ErrInvalidInput.Code, err.Code)
		require.Equal(t, "signed transaction value is not provided", err.Details["error"])
	})

	t.Run("invalid transaction", func(t *testing.T) {
		resp, err := service.ConstructionHash(context.Background(), &types.ConstructionHashRequest{
			SignedTransaction: "{}",
		})
		require.Nil(t, resp)
		require.Equal(t, ErrInvalidInput.Code, err.Code)
	})

	t.Run("valid transaction", func(t *testing.T) {
		signed := `{"nonce":"0x6","gasPrice":"0x6d6e2edc00","gas":"0x5208","to":"0x85ad9d1fcf50b72255e4288dca0ad29f5f509409","value":"0xde0b6b3a7640000","input":"0x","v":"0x150f6","r":"0x64d46cc17cbdbcf73b204a6979172eb3148237ecd369181b105e92b0d7fa49a7","s":"0x285063de57245f532a14b13f605bed047a9d20ebfd0db28e01bc8cc9eaac40ee","hash":"0x92ea9280c1653aa9042c7a4d3a608c2149db45064609c18b270c7c73738e2a46"}`
		request := signedTransactionWrapper{SignedTransaction: []byte(signed), Currency: nil}

		json, err := json.Marshal(request)
		require.NoError(t, err)

		resp, terr := service.ConstructionHash(context.Background(), &types.ConstructionHashRequest{
			SignedTransaction: string(json),
		})
		require.Nil(t, terr)
		require.Equal(
			t,
			"0x92ea9280c1653aa9042c7a4d3a608c2149db45064609c18b270c7c73738e2a46",
			resp.TransactionIdentifier.Hash,
		)
	})

	t.Run("legacy transaction success", func(t *testing.T) {
		signed := `{"nonce":"0x6","gasPrice":"0x6d6e2edc00","gas":"0x5208","to":"0x85ad9d1fcf50b72255e4288dca0ad29f5f509409","value":"0xde0b6b3a7640000","input":"0x","v":"0x150f6","r":"0x64d46cc17cbdbcf73b204a6979172eb3148237ecd369181b105e92b0d7fa49a7","s":"0x285063de57245f532a14b13f605bed047a9d20ebfd0db28e01bc8cc9eaac40ee","hash":"0x92ea9280c1653aa9042c7a4d3a608c2149db45064609c18b270c7c73738e2a46"}` //nolint:lll

		resp, err := service.ConstructionHash(context.Background(), &types.ConstructionHashRequest{
			SignedTransaction: signed,
		})
		require.Nil(t, err)
		require.Equal(
			t,
			"0x92ea9280c1653aa9042c7a4d3a608c2149db45064609c18b270c7c73738e2a46",
			resp.TransactionIdentifier.Hash,
		)
	})

	t.Run("legacy transaction failure", func(t *testing.T) {
		signed := `{"gasPrice":"0x6d6e2edc00","gas":"0x5208","to":"0x85ad9d1fcf50b72255e4288dca0ad29f5f509409","value":"0xde0b6b3a7640000","input":"0x","v":"0x150f6","r":"0x64d46cc17cbdbcf73b204a6979172eb3148237ecd369181b105e92b0d7fa49a7","s":"0x285063de57245f532a14b13f605bed047a9d20ebfd0db28e01bc8cc9eaac40ee","hash":"0x92ea9280c1653aa9042c7a4d3a608c2149db45064609c18b270c7c73738e2a46"}` //nolint:lll

		resp, err := service.ConstructionHash(context.Background(), &types.ConstructionHashRequest{
			SignedTransaction: signed,
		})
		require.Contains(t, err.Details["error"].(string), "nonce")
		require.Nil(t, resp)
	})
}

func TestConstructionDerive(t *testing.T) {
	ctrl := gomock.NewController(t)
	skippedBackend := NewMockConstructionBackend(ctrl)
	skippedBackend.EXPECT().ShouldHandleRequest(gomock.Any()).Return(false).AnyTimes()
	service := ConstructionService{
		pChainBackend:         skippedBackend,
		cChainAtomicTxBackend: skippedBackend,
	}

	t.Run("no public key", func(t *testing.T) {
		resp, err := service.ConstructionDerive(
			context.Background(),
			&types.ConstructionDeriveRequest{},
		)
		require.Nil(t, resp)
		require.Equal(t, ErrInvalidInput.Code, err.Code)
		require.Equal(t, "public key is not provided", err.Details["error"])
	})

	t.Run("invalid public key", func(t *testing.T) {
		resp, err := service.ConstructionDerive(
			context.Background(),
			&types.ConstructionDeriveRequest{
				PublicKey: &types.PublicKey{
					Bytes:     []byte("invaliddata"),
					CurveType: types.Secp256k1,
				},
			},
		)
		require.Nil(t, resp)
		require.Equal(t, ErrInvalidInput.Code, err.Code)
		require.Equal(t, "invalid public key", err.Details["error"])
	})

	t.Run("valid public key", func(t *testing.T) {
		src := "03d0156cec2e01eff9c66e5dbc3c70f98214ec90a25eb43320ebcddc1a94b677f0"
		b, _ := hex.DecodeString(src)

		resp, err := service.ConstructionDerive(
			context.Background(),
			&types.ConstructionDeriveRequest{
				PublicKey: &types.PublicKey{
					Bytes:     b,
					CurveType: types.Secp256k1,
				},
			},
		)
		require.Nil(t, err)
		require.Equal(
			t,
			"0x156daFC6e9A1304fD5C9AB686acB4B3c802FE3f7",
			resp.AccountIdentifier.Address,
		)
	})
}

func forceMarshalMap(t *testing.T, i interface{}) map[string]interface{} {
	m, err := mapper.MarshalJSONMap(i)
	require.NoError(t, err)
	return m
}

func TestPreprocessMetadata(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	client := client.NewMockClient(ctrl)
	networkIdentifier := &types.NetworkIdentifier{
		Network:    rosConst.FujiNetwork,
		Blockchain: "Avalanche",
	}
	skippedBackend := NewMockConstructionBackend(ctrl)
	skippedBackend.EXPECT().ShouldHandleRequest(gomock.Any()).Return(false).AnyTimes()
	service := ConstructionService{
		config:                &Config{Mode: ModeOnline},
		client:                client,
		pChainBackend:         skippedBackend,
		cChainAtomicTxBackend: skippedBackend,
	}
	intent := `[{"operation_identifier":{"index":0},"type":"CALL","account":{"address":"0xe3a5B4d7f79d64088C8d4ef153A7DDe2B2d47309"},"amount":{"value":"-42894881044106498","currency":{"symbol":"AVAX","decimals":18}}},{"operation_identifier":{"index":1},"type":"CALL","account":{"address":"0x57B414a0332B5CaB885a451c2a28a07d1e9b8a8d"},"amount":{"value":"42894881044106498","currency":{"symbol":"AVAX","decimals":18}}}]`
	t.Run("currency info doesn't match between the operations", func(t *testing.T) {
		unclearIntent := `[{"operation_identifier":{"index":0},"type":"CALL","account":{"address":"0xe3a5B4d7f79d64088C8d4ef153A7DDe2B2d47309"},"amount":{"value":"-42894881044106498","currency":{"symbol":"AVAX","decimals":18}}},{"operation_identifier":{"index":1},"type":"CALL","account":{"address":"0x57B414a0332B5CaB885a451c2a28a07d1e9b8a8d"},"amount":{"value":"42894881044106498","currency":{"symbol":"NOAX","decimals":18}}}]`

		var ops []*types.Operation
		require.NoError(t, json.Unmarshal([]byte(unclearIntent), &ops))
		preprocessResponse, err := service.ConstructionPreprocess(
			ctx,
			&types.ConstructionPreprocessRequest{
				NetworkIdentifier: networkIdentifier,
				Operations:        ops,
			},
		)
		require.Nil(t, preprocessResponse)
		require.Equal(t, "currency info doesn't match between the operations", err.Details["error"])
	})
	t.Run("basic flow", func(t *testing.T) {
		var ops []*types.Operation
		require.NoError(t, json.Unmarshal([]byte(intent), &ops))
		preprocessResponse, err := service.ConstructionPreprocess(
			ctx,
			&types.ConstructionPreprocessRequest{
				NetworkIdentifier: networkIdentifier,
				Operations:        ops,
			},
		)
		require.Nil(t, err)
		optionsRaw := `{"from":"0xe3a5B4d7f79d64088C8d4ef153A7DDe2B2d47309","to":"0x57B414a0332B5CaB885a451c2a28a07d1e9b8a8d","value":"0x9864aac3510d02", "currency":{"symbol":"AVAX","decimals":18}}`
		var opt options
		require.NoError(t, json.Unmarshal([]byte(optionsRaw), &opt))
		require.Equal(t, &types.ConstructionPreprocessResponse{
			Options: forceMarshalMap(t, &opt),
		}, preprocessResponse)

		metadata := &metadata{
			GasPrice: big.NewInt(1000000000),
			GasLimit: 21_001,
			Nonce:    0,
		}

		client.EXPECT().SuggestGasPrice(
			ctx,
		).Return(
			big.NewInt(1000000000),
			nil,
		)
		to := common.HexToAddress("0x57B414a0332B5CaB885a451c2a28a07d1e9b8a8d")
		client.EXPECT().EstimateGas(
			ctx,
			interfaces.CallMsg{
				From:  common.HexToAddress("0xe3a5B4d7f79d64088C8d4ef153A7DDe2B2d47309"),
				To:    &to,
				Value: big.NewInt(42894881044106498),
			},
		).Return(
			uint64(21001),
			nil,
		)
		client.EXPECT().NonceAt(
			ctx,
			common.HexToAddress("0xe3a5B4d7f79d64088C8d4ef153A7DDe2B2d47309"),
			(*big.Int)(nil),
		).Return(
			uint64(0),
			nil,
		)
		metadataResponse, err := service.ConstructionMetadata(
			ctx,
			&types.ConstructionMetadataRequest{
				NetworkIdentifier: networkIdentifier,
				Options:           forceMarshalMap(t, &opt),
			},
		)
		require.Nil(t, err)
		require.Equal(t, &types.ConstructionMetadataResponse{
			Metadata: forceMarshalMap(t, metadata),
			SuggestedFee: []*types.Amount{
				{
					Value:    "21001000000000",
					Currency: mapper.AvaxCurrency,
				},
			},
		}, metadataResponse)
	})

	t.Run("basic flow (backwards compatible)", func(t *testing.T) {
		var ops []*types.Operation
		require.NoError(t, json.Unmarshal([]byte(intent), &ops))

		optionsRaw := `{"from":"0xe3a5B4d7f79d64088C8d4ef153A7DDe2B2d47309"}`
		var opt options
		require.NoError(t, json.Unmarshal([]byte(optionsRaw), &opt))

		metadata := &metadata{
			GasPrice: big.NewInt(1000000000),
			GasLimit: 21_000,
			Nonce:    0,
		}

		client.EXPECT().SuggestGasPrice(
			ctx,
		).Return(
			big.NewInt(1000000000),
			nil,
		)
		client.EXPECT().NonceAt(
			ctx,
			common.HexToAddress("0xe3a5B4d7f79d64088C8d4ef153A7DDe2B2d47309"),
			(*big.Int)(nil),
		).Return(
			uint64(0),
			nil,
		)
		metadataResponse, err := service.ConstructionMetadata(
			ctx,
			&types.ConstructionMetadataRequest{
				NetworkIdentifier: networkIdentifier,
				Options:           forceMarshalMap(t, &opt),
			},
		)
		require.Nil(t, err)
		require.Equal(t, &types.ConstructionMetadataResponse{
			Metadata: forceMarshalMap(t, metadata),
			SuggestedFee: []*types.Amount{
				{
					Value:    "21000000000000",
					Currency: mapper.AvaxCurrency,
				},
			},
		}, metadataResponse)
	})

	t.Run("custom gas price flow", func(t *testing.T) {
		var ops []*types.Operation
		require.NoError(t, json.Unmarshal([]byte(intent), &ops))
		preprocessResponse, err := service.ConstructionPreprocess(
			ctx,
			&types.ConstructionPreprocessRequest{
				NetworkIdentifier: networkIdentifier,
				Operations:        ops,
				Metadata: map[string]interface{}{
					"gas_price": "1100000000",
				},
			},
		)
		require.Nil(t, err)
		optionsRaw := `{"from":"0xe3a5B4d7f79d64088C8d4ef153A7DDe2B2d47309","to":"0x57B414a0332B5CaB885a451c2a28a07d1e9b8a8d","value":"0x9864aac3510d02","gas_price":"0x4190ab00", "currency":{"decimals":18, "symbol":"AVAX"}}`
		var opt options
		require.NoError(t, json.Unmarshal([]byte(optionsRaw), &opt))
		require.Equal(t, &types.ConstructionPreprocessResponse{
			Options: forceMarshalMap(t, &opt),
		}, preprocessResponse)

		metadata := &metadata{
			GasPrice: big.NewInt(1100000000),
			GasLimit: 21_000,
			Nonce:    0,
		}

		to := common.HexToAddress("0x57B414a0332B5CaB885a451c2a28a07d1e9b8a8d")
		client.EXPECT().EstimateGas(
			ctx,
			interfaces.CallMsg{
				From:  common.HexToAddress("0xe3a5B4d7f79d64088C8d4ef153A7DDe2B2d47309"),
				To:    &to,
				Value: big.NewInt(42894881044106498),
			},
		).Return(
			uint64(21000),
			nil,
		)
		client.EXPECT().NonceAt(
			ctx,
			common.HexToAddress("0xe3a5B4d7f79d64088C8d4ef153A7DDe2B2d47309"),
			(*big.Int)(nil),
		).Return(
			uint64(0),
			nil,
		)
		metadataResponse, err := service.ConstructionMetadata(
			ctx,
			&types.ConstructionMetadataRequest{
				NetworkIdentifier: networkIdentifier,
				Options:           forceMarshalMap(t, &opt),
			},
		)
		require.Nil(t, err)
		require.Equal(t, &types.ConstructionMetadataResponse{
			Metadata: forceMarshalMap(t, metadata),
			SuggestedFee: []*types.Amount{
				{
					Value:    "23100000000000",
					Currency: mapper.AvaxCurrency,
				},
			},
		}, metadataResponse)
	})

	t.Run("custom gas price flow (ignore multiplier)", func(t *testing.T) {
		var ops []*types.Operation
		require.NoError(t, json.Unmarshal([]byte(intent), &ops))
		multiplier := float64(1.1)
		preprocessResponse, err := service.ConstructionPreprocess(
			ctx,
			&types.ConstructionPreprocessRequest{
				NetworkIdentifier:      networkIdentifier,
				Operations:             ops,
				SuggestedFeeMultiplier: &multiplier,
				Metadata: map[string]interface{}{
					"gas_price": "1100000000",
				},
			},
		)
		require.Nil(t, err)
		optionsRaw := `{"from":"0xe3a5B4d7f79d64088C8d4ef153A7DDe2B2d47309","to":"0x57B414a0332B5CaB885a451c2a28a07d1e9b8a8d","value":"0x9864aac3510d02","gas_price":"0x4190ab00","suggested_fee_multiplier":1.1, "currency":{"decimals":18, "symbol":"AVAX"}}`
		var opt options
		require.NoError(t, json.Unmarshal([]byte(optionsRaw), &opt))
		require.Equal(t, &types.ConstructionPreprocessResponse{
			Options: forceMarshalMap(t, &opt),
		}, preprocessResponse)

		metadata := &metadata{
			GasPrice: big.NewInt(1100000000),
			GasLimit: 21_000,
			Nonce:    0,
		}

		to := common.HexToAddress("0x57B414a0332B5CaB885a451c2a28a07d1e9b8a8d")
		client.EXPECT().EstimateGas(
			ctx,
			interfaces.CallMsg{
				From:  common.HexToAddress("0xe3a5B4d7f79d64088C8d4ef153A7DDe2B2d47309"),
				To:    &to,
				Value: big.NewInt(42894881044106498),
			},
		).Return(
			uint64(21000),
			nil,
		)
		client.EXPECT().NonceAt(
			ctx,
			common.HexToAddress("0xe3a5B4d7f79d64088C8d4ef153A7DDe2B2d47309"),
			(*big.Int)(nil),
		).Return(
			uint64(0),
			nil,
		)
		metadataResponse, err := service.ConstructionMetadata(
			ctx,
			&types.ConstructionMetadataRequest{
				NetworkIdentifier: networkIdentifier,
				Options:           forceMarshalMap(t, &opt),
			},
		)
		require.Nil(t, err)
		require.Equal(t, &types.ConstructionMetadataResponse{
			Metadata: forceMarshalMap(t, metadata),
			SuggestedFee: []*types.Amount{
				{
					Value:    "23100000000000",
					Currency: mapper.AvaxCurrency,
				},
			},
		}, metadataResponse)
	})

	t.Run("fee multiplier", func(t *testing.T) {
		var ops []*types.Operation
		require.NoError(t, json.Unmarshal([]byte(intent), &ops))
		multiplier := float64(1.1)
		preprocessResponse, err := service.ConstructionPreprocess(
			ctx,
			&types.ConstructionPreprocessRequest{
				NetworkIdentifier:      networkIdentifier,
				Operations:             ops,
				SuggestedFeeMultiplier: &multiplier,
			},
		)
		require.Nil(t, err)
		optionsRaw := `{"from":"0xe3a5B4d7f79d64088C8d4ef153A7DDe2B2d47309","to":"0x57B414a0332B5CaB885a451c2a28a07d1e9b8a8d","value":"0x9864aac3510d02","suggested_fee_multiplier":1.1, "currency":{"decimals":18, "symbol":"AVAX"}}`
		var opt options
		require.NoError(t, json.Unmarshal([]byte(optionsRaw), &opt))
		require.Equal(t, &types.ConstructionPreprocessResponse{
			Options: forceMarshalMap(t, &opt),
		}, preprocessResponse)

		metadata := &metadata{
			GasPrice: big.NewInt(1100000000),
			GasLimit: 21_000,
			Nonce:    0,
		}

		to := common.HexToAddress("0x57B414a0332B5CaB885a451c2a28a07d1e9b8a8d")
		client.EXPECT().EstimateGas(
			ctx,
			interfaces.CallMsg{
				From:  common.HexToAddress("0xe3a5B4d7f79d64088C8d4ef153A7DDe2B2d47309"),
				To:    &to,
				Value: big.NewInt(42894881044106498),
			},
		).Return(
			uint64(21000),
			nil,
		)
		client.EXPECT().SuggestGasPrice(
			ctx,
		).Return(
			big.NewInt(1000000000),
			nil,
		)
		client.EXPECT().NonceAt(
			ctx,
			common.HexToAddress("0xe3a5B4d7f79d64088C8d4ef153A7DDe2B2d47309"),
			(*big.Int)(nil),
		).Return(
			uint64(0),
			nil,
		)
		metadataResponse, err := service.ConstructionMetadata(
			ctx,
			&types.ConstructionMetadataRequest{
				NetworkIdentifier: networkIdentifier,
				Options:           forceMarshalMap(t, &opt),
			},
		)
		require.Nil(t, err)
		require.Equal(t, &types.ConstructionMetadataResponse{
			Metadata: forceMarshalMap(t, metadata),
			SuggestedFee: []*types.Amount{
				{
					Value:    "23100000000000",
					Currency: mapper.AvaxCurrency,
				},
			},
		}, metadataResponse)
	})

	t.Run("custom nonce", func(t *testing.T) {
		var ops []*types.Operation
		require.NoError(t, json.Unmarshal([]byte(intent), &ops))
		multiplier := float64(1.1)
		preprocessResponse, err := service.ConstructionPreprocess(
			ctx,
			&types.ConstructionPreprocessRequest{
				NetworkIdentifier:      networkIdentifier,
				Operations:             ops,
				SuggestedFeeMultiplier: &multiplier,
				Metadata: map[string]interface{}{
					"nonce": "1",
				},
			},
		)
		require.Nil(t, err)
		optionsRaw := `{"from":"0xe3a5B4d7f79d64088C8d4ef153A7DDe2B2d47309","to":"0x57B414a0332B5CaB885a451c2a28a07d1e9b8a8d","value":"0x9864aac3510d02","suggested_fee_multiplier":1.1, "nonce":"0x1", "currency":{"decimals":18, "symbol":"AVAX"}}`
		var opt options
		require.NoError(t, json.Unmarshal([]byte(optionsRaw), &opt))
		require.Equal(t, &types.ConstructionPreprocessResponse{
			Options: forceMarshalMap(t, &opt),
		}, preprocessResponse)

		metadata := &metadata{
			GasPrice: big.NewInt(1100000000),
			GasLimit: 21_000,
			Nonce:    1,
		}

		to := common.HexToAddress("0x57B414a0332B5CaB885a451c2a28a07d1e9b8a8d")
		client.EXPECT().EstimateGas(
			ctx,
			interfaces.CallMsg{
				From:  common.HexToAddress("0xe3a5B4d7f79d64088C8d4ef153A7DDe2B2d47309"),
				To:    &to,
				Value: big.NewInt(42894881044106498),
			},
		).Return(
			uint64(21000),
			nil,
		)
		client.EXPECT().SuggestGasPrice(
			ctx,
		).Return(
			big.NewInt(1000000000),
			nil,
		)
		metadataResponse, err := service.ConstructionMetadata(
			ctx,
			&types.ConstructionMetadataRequest{
				NetworkIdentifier: networkIdentifier,
				Options:           forceMarshalMap(t, &opt),
			},
		)
		require.Nil(t, err)
		require.Equal(t, &types.ConstructionMetadataResponse{
			Metadata: forceMarshalMap(t, metadata),
			SuggestedFee: []*types.Amount{
				{
					Value:    "23100000000000",
					Currency: mapper.AvaxCurrency,
				},
			},
		}, metadataResponse)
	})

	t.Run("custom gas limit", func(t *testing.T) {
		var ops []*types.Operation
		require.NoError(t, json.Unmarshal([]byte(intent), &ops))
		multiplier := float64(1.1)
		preprocessResponse, err := service.ConstructionPreprocess(
			ctx,
			&types.ConstructionPreprocessRequest{
				NetworkIdentifier:      networkIdentifier,
				Operations:             ops,
				SuggestedFeeMultiplier: &multiplier,
				Metadata: map[string]interface{}{
					"gas_limit": "40000",
				},
			},
		)
		require.Nil(t, err)
		optionsRaw := `{"from":"0xe3a5B4d7f79d64088C8d4ef153A7DDe2B2d47309","to":"0x57B414a0332B5CaB885a451c2a28a07d1e9b8a8d","value":"0x9864aac3510d02","suggested_fee_multiplier":1.1,"gas_limit":"0x9c40", "currency":{"decimals":18, "symbol":"AVAX"}}`
		var opt options
		require.NoError(t, json.Unmarshal([]byte(optionsRaw), &opt))
		require.Equal(t, &types.ConstructionPreprocessResponse{
			Options: forceMarshalMap(t, &opt),
		}, preprocessResponse)

		metadata := &metadata{
			GasPrice: big.NewInt(1100000000),
			GasLimit: 40_000,
			Nonce:    0,
		}

		client.EXPECT().SuggestGasPrice(
			ctx,
		).Return(
			big.NewInt(1000000000),
			nil,
		)
		client.EXPECT().NonceAt(
			ctx,
			common.HexToAddress("0xe3a5B4d7f79d64088C8d4ef153A7DDe2B2d47309"),
			(*big.Int)(nil),
		).Return(
			uint64(0),
			nil,
		)
		metadataResponse, err := service.ConstructionMetadata(
			ctx,
			&types.ConstructionMetadataRequest{
				NetworkIdentifier: networkIdentifier,
				Options:           forceMarshalMap(t, &opt),
			},
		)
		require.Nil(t, err)
		require.Equal(t, &types.ConstructionMetadataResponse{
			Metadata: forceMarshalMap(t, metadata),
			SuggestedFee: []*types.Amount{
				{
					Value:    "44000000000000",
					Currency: mapper.AvaxCurrency,
				},
			},
		}, metadataResponse)
	})

	t.Run("basic erc20 flow", func(t *testing.T) {
		erc20Intent := `[{"operation_identifier":{"index":0},"type":"ERC20_TRANSFER","account":{"address":"0xe3a5B4d7f79d64088C8d4ef153A7DDe2B2d47309"},"amount":{"value":"-42894881044106498","currency":{"symbol":"TEST","decimals":18, "metadata": {"contractAddress": "0x30e5449b6712Adf4156c8c474250F6eA4400eB82"}}}},{"operation_identifier":{"index":1},"type":"ERC20_TRANSFER","account":{"address":"0x57B414a0332B5CaB885a451c2a28a07d1e9b8a8d"},"amount":{"value":"42894881044106498","currency":{"symbol":"TEST","decimals":18, "metadata": {"contractAddress": "0x30e5449b6712Adf4156c8c474250F6eA4400eB82"}}}}]`
		tokenList := []string{defaultContractAddress}

		service := ConstructionService{
			config:                &Config{Mode: ModeOnline, TokenWhiteList: tokenList},
			client:                client,
			pChainBackend:         skippedBackend,
			cChainAtomicTxBackend: skippedBackend,
		}
		var ops []*types.Operation
		require.NoError(t, json.Unmarshal([]byte(erc20Intent), &ops))
		preprocessResponse, err := service.ConstructionPreprocess(
			ctx,
			&types.ConstructionPreprocessRequest{
				NetworkIdentifier: networkIdentifier,
				Operations:        ops,
			},
		)
		require.Nil(t, err)
		optionsRaw := `{"from":"0xe3a5B4d7f79d64088C8d4ef153A7DDe2B2d47309","to":"0x57B414a0332B5CaB885a451c2a28a07d1e9b8a8d","value":"0x9864aac3510d02", "currency":{"symbol":"TEST","decimals":18, "metadata": {"contractAddress": "0x30e5449b6712Adf4156c8c474250F6eA4400eB82"}}}`
		var opt options
		require.NoError(t, json.Unmarshal([]byte(optionsRaw), &opt))
		require.Equal(t, &types.ConstructionPreprocessResponse{
			Options: forceMarshalMap(t, &opt),
		}, preprocessResponse)

		metadata := &metadata{
			GasPrice: big.NewInt(1000000000),
			GasLimit: 21_001,
			Nonce:    0,
		}

		client.EXPECT().SuggestGasPrice(
			ctx,
		).Return(
			big.NewInt(1000000000),
			nil,
		)
		contractAddress := common.HexToAddress(defaultContractAddress)
		client.EXPECT().EstimateGas(
			ctx,
			interfaces.CallMsg{
				From: common.HexToAddress("0xe3a5B4d7f79d64088C8d4ef153A7DDe2B2d47309"),
				To:   &contractAddress,
				Data: common.Hex2Bytes(
					"a9059cbb00000000000000000000000057B414a0332B5CaB885a451c2a28a07d1e9b8a8d000000000000000000000000000000000000000000000000009864aac3510d02",
				),
			},
		).Return(
			uint64(21001),
			nil,
		)
		client.EXPECT().NonceAt(
			ctx,
			common.HexToAddress("0xe3a5B4d7f79d64088C8d4ef153A7DDe2B2d47309"),
			(*big.Int)(nil),
		).Return(
			uint64(0),
			nil,
		)

		metadataResponse, err := service.ConstructionMetadata(
			ctx,
			&types.ConstructionMetadataRequest{
				NetworkIdentifier: networkIdentifier,
				Options:           forceMarshalMap(t, &opt),
			},
		)
		require.Nil(t, err)
		require.Equal(t, &types.ConstructionMetadataResponse{
			Metadata: forceMarshalMap(t, metadata),
			SuggestedFee: []*types.Amount{
				{
					Value:    "21001000000000",
					Currency: mapper.AvaxCurrency,
				},
			},
		}, metadataResponse)
	})

	t.Run("basic unwrap flow", func(t *testing.T) {
		unwrapIntent := `[{"operation_identifier":{"index":0},"type":"ERC20_BURN","account":{"address":"0xe3a5B4d7f79d64088C8d4ef153A7DDe2B2d47309"},"amount":{"value":"-42894881044106498","currency":{"symbol":"TEST","decimals":18, "metadata": {"contractAddress": "0x30e5449b6712Adf4156c8c474250F6eA4400eB82"}}}}]`
		bridgeTokenList := []string{defaultContractAddress}
		skippedBackend := NewMockConstructionBackend(ctrl)
		skippedBackend.EXPECT().ShouldHandleRequest(gomock.Any()).Return(false).AnyTimes()

		service := ConstructionService{
			config: &Config{
				Mode:            ModeOnline,
				BridgeTokenList: bridgeTokenList,
			},
			client:                client,
			pChainBackend:         skippedBackend,
			cChainAtomicTxBackend: skippedBackend,
		}
		var ops []*types.Operation
		require.NoError(t, json.Unmarshal([]byte(unwrapIntent), &ops))

		requestMetadata := map[string]interface{}{
			"bridge_unwrap": true,
		}
		preprocessResponse, err := service.ConstructionPreprocess(
			ctx,
			&types.ConstructionPreprocessRequest{
				NetworkIdentifier: networkIdentifier,
				Operations:        ops,
				Metadata:          requestMetadata,
			},
		)
		require.Nil(t, err)
		optionsRaw := `{"metadata": {"bridge_unwrap":true}, "from":"0xe3a5B4d7f79d64088C8d4ef153A7DDe2B2d47309","value":"0x9864aac3510d02", "currency":{"symbol":"TEST","decimals":18, "metadata": {"contractAddress": "0x30e5449b6712Adf4156c8c474250F6eA4400eB82"}}}`
		var opt options
		require.NoError(t, json.Unmarshal([]byte(optionsRaw), &opt))
		require.Equal(t, &types.ConstructionPreprocessResponse{
			Options: forceMarshalMap(t, &opt),
		}, preprocessResponse)

		metadata := &metadata{
			GasPrice:       big.NewInt(1000000000),
			GasLimit:       21_001,
			Nonce:          0,
			UnwrapBridgeTx: true,
		}

		client.EXPECT().SuggestGasPrice(
			ctx,
		).Return(
			big.NewInt(1000000000),
			nil,
		)
		contractAddress := common.HexToAddress(defaultContractAddress)
		client.EXPECT().EstimateGas(
			ctx,
			interfaces.CallMsg{
				From: common.HexToAddress("0xe3a5B4d7f79d64088C8d4ef153A7DDe2B2d47309"),
				To:   &contractAddress,
				Data: common.Hex2Bytes(
					"6e286671000000000000000000000000000000000000000000000000009864aac3510d020000000000000000000000000000000000000000000000000000000000000000",
				),
			},
		).Return(
			uint64(21001),
			nil,
		)
		client.EXPECT().NonceAt(
			ctx,
			common.HexToAddress("0xe3a5B4d7f79d64088C8d4ef153A7DDe2B2d47309"),
			(*big.Int)(nil),
		).Return(
			uint64(0),
			nil,
		)

		metadataResponse, err := service.ConstructionMetadata(
			ctx,
			&types.ConstructionMetadataRequest{
				NetworkIdentifier: networkIdentifier,
				Options:           forceMarshalMap(t, &opt),
			},
		)
		require.Nil(t, err)
		require.Equal(t, &types.ConstructionMetadataResponse{
			Metadata: forceMarshalMap(t, metadata),
			SuggestedFee: []*types.Amount{
				{
					Value:    "21001000000000",
					Currency: mapper.AvaxCurrency,
				},
			},
		}, metadataResponse)
	})

	t.Run("arbitrary contract call flow", func(t *testing.T) {
		contractCallIntent := `[{"operation_identifier":{"index":0},"type":"CALL","account":{"address":"0xe3a5B4d7f79d64088C8d4ef153A7DDe2B2d47309"},"amount":{"value":"0","currency":{"symbol":"TEST","decimals":18}}},{"operation_identifier":{"index":1},"type":"CALL","account":{"address":"0x57B414a0332B5CaB885a451c2a28a07d1e9b8a8d"},"amount":{"value":"0","currency":{"symbol":"TEST","decimals":18}}}]`
		service := ConstructionService{
			config:                &Config{Mode: ModeOnline},
			client:                client,
			pChainBackend:         skippedBackend,
			cChainAtomicTxBackend: skippedBackend,
		}

		var ops []*types.Operation
		require.NoError(t, json.Unmarshal([]byte(contractCallIntent), &ops))

		requestMetadata := map[string]interface{}{
			"bridge_unwrap":    false,
			"method_signature": `deploy(bytes32,address,address,address,address)`,
			"method_args":      []string{"0x3100000000000000000000000000000000000000000000000000000000000000", "0x323e3ab04a3795ad79cc92378fcdb0a0aec51ba5", "0x14e37c2e9cd255404bd35b4542fd9ccaa070aed6", "0x323e3ab04a3795ad79cc92378fcdb0a0aec51ba5", "0x14e37c2e9cd255404bd35b4542fd9ccaa070aed6"},
		}

		preprocessResponse, err := service.ConstructionPreprocess(
			ctx,
			&types.ConstructionPreprocessRequest{
				NetworkIdentifier: networkIdentifier,
				Operations:        ops,
				Metadata:          requestMetadata,
			},
		)
		require.Nil(t, err)

		optionsRaw := `{"from":"0xe3a5B4d7f79d64088C8d4ef153A7DDe2B2d47309","to":"0x57B414a0332B5CaB885a451c2a28a07d1e9b8a8d","value":"0x0", "currency":{"symbol":"TEST","decimals":18}, "contract_address": "0x57B414a0332B5CaB885a451c2a28a07d1e9b8a8d", "method_signature": "deploy(bytes32,address,address,address,address)", "data": "0xb0d78b753100000000000000000000000000000000000000000000000000000000000000000000000000000000000000323e3ab04a3795ad79cc92378fcdb0a0aec51ba500000000000000000000000014e37c2e9cd255404bd35b4542fd9ccaa070aed6000000000000000000000000323e3ab04a3795ad79cc92378fcdb0a0aec51ba500000000000000000000000014e37c2e9cd255404bd35b4542fd9ccaa070aed6", "method_args":["0x3100000000000000000000000000000000000000000000000000000000000000","0x323e3ab04a3795ad79cc92378fcdb0a0aec51ba5","0x14e37c2e9cd255404bd35b4542fd9ccaa070aed6","0x323e3ab04a3795ad79cc92378fcdb0a0aec51ba5","0x14e37c2e9cd255404bd35b4542fd9ccaa070aed6"] }`
		var opt options
		require.NoError(t, json.Unmarshal([]byte(optionsRaw), &opt))
		require.Equal(t, &types.ConstructionPreprocessResponse{
			Options: forceMarshalMap(t, &opt),
		}, preprocessResponse)

		// call metadata API
		metadata := &metadata{
			GasPrice:        big.NewInt(1000000000),
			GasLimit:        21_001,
			Nonce:           0,
			UnwrapBridgeTx:  false,
			ContractData:    "0xb0d78b753100000000000000000000000000000000000000000000000000000000000000000000000000000000000000323e3ab04a3795ad79cc92378fcdb0a0aec51ba500000000000000000000000014e37c2e9cd255404bd35b4542fd9ccaa070aed6000000000000000000000000323e3ab04a3795ad79cc92378fcdb0a0aec51ba500000000000000000000000014e37c2e9cd255404bd35b4542fd9ccaa070aed6",
			MethodSignature: "deploy(bytes32,address,address,address,address)",
			MethodArgs:      []string{"0x3100000000000000000000000000000000000000000000000000000000000000", "0x323e3ab04a3795ad79cc92378fcdb0a0aec51ba5", "0x14e37c2e9cd255404bd35b4542fd9ccaa070aed6", "0x323e3ab04a3795ad79cc92378fcdb0a0aec51ba5", "0x14e37c2e9cd255404bd35b4542fd9ccaa070aed6"},
		}

		client.EXPECT().SuggestGasPrice(
			ctx,
		).Return(
			big.NewInt(1000000000),
			nil,
		)
		contractAddress := common.HexToAddress(defaultToAddress)
		data, _ := hexutil.Decode("0xb0d78b753100000000000000000000000000000000000000000000000000000000000000000000000000000000000000323e3ab04a3795ad79cc92378fcdb0a0aec51ba500000000000000000000000014e37c2e9cd255404bd35b4542fd9ccaa070aed6000000000000000000000000323e3ab04a3795ad79cc92378fcdb0a0aec51ba500000000000000000000000014e37c2e9cd255404bd35b4542fd9ccaa070aed6")
		client.EXPECT().EstimateGas(
			ctx,
			interfaces.CallMsg{
				From: common.HexToAddress("0xe3a5B4d7f79d64088C8d4ef153A7DDe2B2d47309"),
				To:   &contractAddress,
				Data: data,
			},
		).Return(
			uint64(21001),
			nil,
		)
		client.EXPECT().NonceAt(
			ctx,
			common.HexToAddress("0xe3a5B4d7f79d64088C8d4ef153A7DDe2B2d47309"),
			(*big.Int)(nil),
		).Return(
			uint64(0),
			nil,
		)

		metadataResponse, err := service.ConstructionMetadata(
			ctx,
			&types.ConstructionMetadataRequest{
				NetworkIdentifier: networkIdentifier,
				Options:           forceMarshalMap(t, &opt),
			},
		)
		require.Nil(t, err)
		require.Equal(t, &types.ConstructionMetadataResponse{
			Metadata: forceMarshalMap(t, metadata),
			SuggestedFee: []*types.Amount{
				{
					Value:    "21001000000000",
					Currency: mapper.AvaxCurrency,
				},
			},
		}, metadataResponse)
	})
}

func TestBackendDelegations(t *testing.T) {
	ctrl := gomock.NewController(t)

	testCases := []string{
		"p-chain",
		"c-chain-atomic-tx",
	}

	makeBackends := func(currentBackend int) []*MockConstructionBackend {
		backends := make([]*MockConstructionBackend, len(testCases))
		for i := range backends {
			backends[i] = NewMockConstructionBackend(ctrl)

			if i == currentBackend {
				backends[i].EXPECT().ShouldHandleRequest(gomock.Any()).Return(true).Times(8)
				break
			}

			backends[i].EXPECT().ShouldHandleRequest(gomock.Any()).Return(false).Times(8)
		}
		return backends
	}

	for idx, backendName := range testCases {
		backends := makeBackends(idx)

		offlineService := ConstructionService{
			config:                &Config{Mode: ModeOffline},
			pChainBackend:         backends[0],
			cChainAtomicTxBackend: backends[1],
		}

		onlineService := ConstructionService{
			config:                &Config{Mode: ModeOnline},
			pChainBackend:         backends[0],
			cChainAtomicTxBackend: backends[1],
		}

		t.Run("Derive request is delegated to "+backendName, func(t *testing.T) {
			req := &types.ConstructionDeriveRequest{
				PublicKey: &types.PublicKey{},
			}

			expectedResp := &types.ConstructionDeriveResponse{
				AccountIdentifier: &types.AccountIdentifier{
					Address: "P-fuji15f9g0h5xkr5cp47n6u3qxj6yjtzzzrdr23a3tl",
				},
			}

			backends[idx].EXPECT().ConstructionDerive(gomock.Any(), req).Return(expectedResp, nil)
			resp, err := offlineService.ConstructionDerive(context.Background(), req)

			require.Nil(t, err)
			require.Equal(t, expectedResp, resp)
		})

		t.Run("Preprocess request is delegated to "+backendName, func(t *testing.T) {
			req := &types.ConstructionPreprocessRequest{}

			expectedResp := &types.ConstructionPreprocessResponse{
				Options: map[string]interface{}{"key": "value"},
			}

			backends[idx].EXPECT().ConstructionPreprocess(gomock.Any(), req).Return(expectedResp, nil)
			resp, err := offlineService.ConstructionPreprocess(context.Background(), req)

			require.Nil(t, err)
			require.Equal(t, expectedResp, resp)
		})

		t.Run("Metadata request is delegated to "+backendName, func(t *testing.T) {
			req := &types.ConstructionMetadataRequest{}

			expectedResp := &types.ConstructionMetadataResponse{
				Metadata: map[string]interface{}{"key": "value"},
			}

			backends[idx].EXPECT().ConstructionMetadata(gomock.Any(), req).Return(expectedResp, nil)
			resp, err := onlineService.ConstructionMetadata(context.Background(), req)

			require.Nil(t, err)
			require.Equal(t, expectedResp, resp)
		})

		t.Run("Payloads request is delegated to "+backendName, func(t *testing.T) {
			req := &types.ConstructionPayloadsRequest{}

			expectedResp := &types.ConstructionPayloadsResponse{UnsignedTransaction: "unsignedtxn"}

			backends[idx].EXPECT().ConstructionPayloads(gomock.Any(), req).Return(expectedResp, nil)
			resp, err := offlineService.ConstructionPayloads(context.Background(), req)

			require.Nil(t, err)
			require.Equal(t, expectedResp, resp)
		})

		t.Run("Combine request is delegated to "+backendName, func(t *testing.T) {
			req := &types.ConstructionCombineRequest{
				UnsignedTransaction: "unsignedtxn",
				Signatures:          []*types.Signature{{}},
			}

			expectedResp := &types.ConstructionCombineResponse{SignedTransaction: "unsignedtxn"}

			backends[idx].EXPECT().ConstructionCombine(gomock.Any(), req).Return(expectedResp, nil)
			resp, err := offlineService.ConstructionCombine(context.Background(), req)

			require.Nil(t, err)
			require.Equal(t, expectedResp, resp)
		})

		t.Run("Parse request is delegated to "+backendName, func(t *testing.T) {
			req := &types.ConstructionParseRequest{}
			expectedResp := &types.ConstructionParseResponse{}

			backends[idx].EXPECT().ConstructionParse(gomock.Any(), req).Return(expectedResp, nil)
			resp, err := offlineService.ConstructionParse(context.Background(), req)

			require.Nil(t, err)
			require.Equal(t, expectedResp, resp)
		})

		t.Run("Hash request is delegated to "+backendName, func(t *testing.T) {
			req := &types.ConstructionHashRequest{SignedTransaction: "signedtxn"}
			expectedResp := &types.TransactionIdentifierResponse{
				TransactionIdentifier: &types.TransactionIdentifier{Hash: "txn hash"},
			}

			backends[idx].EXPECT().ConstructionHash(gomock.Any(), req).Return(expectedResp, nil)
			resp, err := offlineService.ConstructionHash(context.Background(), req)

			require.Nil(t, err)
			require.Equal(t, expectedResp, resp)
		})

		t.Run("Submit request is delegated to "+backendName, func(t *testing.T) {
			req := &types.ConstructionSubmitRequest{SignedTransaction: "signedtxn"}
			expectedResp := &types.TransactionIdentifierResponse{
				TransactionIdentifier: &types.TransactionIdentifier{Hash: "txn hash"},
			}

			backends[idx].EXPECT().ConstructionSubmit(gomock.Any(), req).Return(expectedResp, nil)
			resp, err := onlineService.ConstructionSubmit(context.Background(), req)

			require.Nil(t, err)
			require.Equal(t, expectedResp, resp)
		})
	}
}
