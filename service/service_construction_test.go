package service

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"math/big"
	"testing"

	"github.com/ava-labs/avalanche-rosetta/mapper"
	mocks "github.com/ava-labs/avalanche-rosetta/mocks/client"

	"github.com/coinbase/rosetta-sdk-go/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"
)

func TestConstructionMetadata(t *testing.T) {
	service := ConstructionService{
		config: &Config{Mode: ModeOnline},
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
		assert.Nil(t, resp)
		assert.Equal(t, errUnavailableOffline.Code, err.Code)
	})

	t.Run("requires from address", func(t *testing.T) {
		resp, err := service.ConstructionMetadata(
			context.Background(),
			&types.ConstructionMetadataRequest{},
		)
		assert.Nil(t, resp)
		assert.Equal(t, errInvalidInput.Code, err.Code)
		assert.Equal(t, "from address is not provided", err.Details["error"])
	})
}

func TestContructionHash(t *testing.T) {
	service := ConstructionService{}

	t.Run("no transaction", func(t *testing.T) {
		resp, err := service.ConstructionHash(
			context.Background(),
			&types.ConstructionHashRequest{},
		)
		assert.Nil(t, resp)
		assert.Equal(t, errInvalidInput.Code, err.Code)
		assert.Equal(t, "signed transaction value is not provided", err.Details["error"])
	})

	t.Run("invalid transaction", func(t *testing.T) {
		resp, err := service.ConstructionHash(context.Background(), &types.ConstructionHashRequest{
			SignedTransaction: "{}",
		})
		assert.Nil(t, resp)
		assert.Equal(t, errInvalidInput.Code, err.Code)
	})

	t.Run("valid transaction", func(t *testing.T) {
		signed := `{"nonce":"0x6","gasPrice":"0x6d6e2edc00","gas":"0x5208","to":"0x85ad9d1fcf50b72255e4288dca0ad29f5f509409","value":"0xde0b6b3a7640000","input":"0x","v":"0x150f6","r":"0x64d46cc17cbdbcf73b204a6979172eb3148237ecd369181b105e92b0d7fa49a7","s":"0x285063de57245f532a14b13f605bed047a9d20ebfd0db28e01bc8cc9eaac40ee","hash":"0x92ea9280c1653aa9042c7a4d3a608c2149db45064609c18b270c7c73738e2a46"}` //nolint:lll

		resp, err := service.ConstructionHash(context.Background(), &types.ConstructionHashRequest{
			SignedTransaction: signed,
		})
		assert.Nil(t, err)
		assert.Equal(
			t,
			"0x92ea9280c1653aa9042c7a4d3a608c2149db45064609c18b270c7c73738e2a46",
			resp.TransactionIdentifier.Hash,
		)
	})
}

func TestConstructionDerive(t *testing.T) {
	service := ConstructionService{}

	t.Run("no public key", func(t *testing.T) {
		resp, err := service.ConstructionDerive(
			context.Background(),
			&types.ConstructionDeriveRequest{},
		)
		assert.Nil(t, resp)
		assert.Equal(t, errInvalidInput.Code, err.Code)
		assert.Equal(t, "public key is not provided", err.Details["error"])
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
		assert.Nil(t, resp)
		assert.Equal(t, errInvalidInput.Code, err.Code)
		assert.Equal(t, "invalid public key", err.Details["error"])
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
		assert.Nil(t, err)
		assert.Equal(
			t,
			"0x156daFC6e9A1304fD5C9AB686acB4B3c802FE3f7",
			resp.AccountIdentifier.Address,
		)
	})
}

func forceMarshalMap(t *testing.T, i interface{}) map[string]interface{} {
	m, err := marshalJSONMap(i)
	if err != nil {
		t.Fatalf("could not marshal map %s", types.PrintStruct(i))
	}

	return m
}

func TestPreprocessMetadata(t *testing.T) {
	ctx := context.Background()
	client := &mocks.Client{}
	networkIdentifier := &types.NetworkIdentifier{
		Network:    "Fuji",
		Blockchain: "Avalanche",
	}
	service := ConstructionService{
		config: &Config{Mode: ModeOnline},
		client: client,
	}

	t.Run("basic flow", func(t *testing.T) {
		intent := `[{"operation_identifier":{"index":0},"type":"CALL","account":{"address":"0xe3a5B4d7f79d64088C8d4ef153A7DDe2B2d47309"},"amount":{"value":"-42894881044106498","currency":{"symbol":"AVAX","decimals":18}}},{"operation_identifier":{"index":1},"type":"CALL","account":{"address":"0x57B414a0332B5CaB885a451c2a28a07d1e9b8a8d"},"amount":{"value":"42894881044106498","currency":{"symbol":"AVAX","decimals":18}}}]` // nolint
		var ops []*types.Operation
		assert.NoError(t, json.Unmarshal([]byte(intent), &ops))
		preprocessResponse, err := service.ConstructionPreprocess(
			ctx,
			&types.ConstructionPreprocessRequest{
				NetworkIdentifier: networkIdentifier,
				Operations:        ops,
			},
		)
		assert.Nil(t, err)
		optionsRaw := `{"from":"0xe3a5B4d7f79d64088C8d4ef153A7DDe2B2d47309"}`
		var opt options
		assert.NoError(t, json.Unmarshal([]byte(optionsRaw), &opt))
		assert.Equal(t, &types.ConstructionPreprocessResponse{
			Options: forceMarshalMap(t, opt),
		}, preprocessResponse)

		metadata := &metadata{
			GasPrice: big.NewInt(1000000000),
			Nonce:    0,
		}

		client.On(
			"SuggestGasPrice",
			ctx,
		).Return(
			big.NewInt(1000000000),
			nil,
		).Once()
		client.On(
			"NonceAt",
			ctx,
			common.HexToAddress("0xe3a5B4d7f79d64088C8d4ef153A7DDe2B2d47309"),
			(*big.Int)(nil),
		).Return(
			uint64(0),
			nil,
		).Once()
		metadataResponse, err := service.ConstructionMetadata(ctx, &types.ConstructionMetadataRequest{
			NetworkIdentifier: networkIdentifier,
			Options:           forceMarshalMap(t, opt),
		})
		assert.Nil(t, err)
		assert.Equal(t, &types.ConstructionMetadataResponse{
			Metadata: forceMarshalMap(t, metadata),
			SuggestedFee: []*types.Amount{
				{
					Value:    "21000000000000",
					Currency: mapper.AvaxCurrency,
				},
			},
		}, metadataResponse)
	})
}

func TestConstructionService(t *testing.T) {
	// 	ctx := context.Background()
	// 	client := &mocks.Client{}
	// 	networkIdentifier := &types.NetworkIdentifier{
	// 		Network:    "Fuji",
	// 		Blockchain: "Avalanche",
	// 	}
	// 	service := ConstructionService{
	// 		config: &Config{Mode: ModeOnline},
	// 		client: client,
	// 	}
	//
	// 	t.Run("basic flow", func(t *testing.T) {
	// 		intent := `[{"operation_identifier":{"index":0},"type":"CALL","account":{"address":"0xe3a5B4d7f79d64088C8d4ef153A7DDe2B2d47309"},"amount":{"value":"-42894881044106498","currency":{"symbol":"AVAX","decimals":18}}},{"operation_identifier":{"index":1},"type":"CALL","account":{"address":"0x57B414a0332B5CaB885a451c2a28a07d1e9b8a8d"},"amount":{"value":"42894881044106498","currency":{"symbol":"AVAX","decimals":18}}}]` // nolint
	// 		var ops []*types.Operation
	// 		assert.NoError(t, json.Unmarshal([]byte(intent), &ops))
	// 		preprocessResponse, err := service.ConstructionPreprocess(
	// 			ctx,
	// 			&types.ConstructionPreprocessRequest{
	// 				NetworkIdentifier: networkIdentifier,
	// 				Operations:        ops,
	// 			},
	// 		)
	// 		assert.Nil(t, err)
	// 		optionsRaw := `{"from":"0xe3a5B4d7f79d64088C8d4ef153A7DDe2B2d47309"}`
	// 		var options txOptions
	// 		assert.NoError(t, json.Unmarshal([]byte(optionsRaw), &options))
	// 		assert.Equal(t, &types.ConstructionPreprocessResponse{
	// 			Options: forceMarshalMap(t, options),
	// 		}, preprocessResponse)
	// 	})
}
