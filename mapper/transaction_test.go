package mapper

import (
	"encoding/hex"
	"math/big"
	"testing"

	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/avalanchego/utils/formatting"
	"github.com/ava-labs/avalanchego/vms/components/avax"
	ethtypes "github.com/ava-labs/coreth/core/types"
	"github.com/ava-labs/coreth/plugin/evm"
	"github.com/coinbase/rosetta-sdk-go/types"
	ethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"

	"github.com/ava-labs/avalanche-rosetta/constants"
)

var WAVAX = &types.Currency{
	Symbol:   "WAVAX",
	Decimals: 18,
	Metadata: map[string]interface{}{
		"contractAddress": "0xB31f66AA3C1e785363F0875A1B74E27b85FD66c7",
	},
}

func TestZeroAddress(t *testing.T) {
	t.Run("correct address", func(t *testing.T) {
		assert.Equal(t, ethcommon.HexToAddress("0x0000000000000000000000000000000000000000"), zeroAddress)
	})
}

func TestERC20Ops(t *testing.T) {
	t.Run("transfer op", func(t *testing.T) {
		log := &ethtypes.Log{
			Address: ethcommon.HexToAddress("0xB31f66AA3C1e785363F0875A1B74E27b85FD66c7"),
			Topics: []ethcommon.Hash{
				ethcommon.HexToHash("0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef"),
				ethcommon.HexToHash("0x000000000000000000000000f1b77573a8525acfa116a785092d1ba90d96bf37"),
				ethcommon.HexToHash("0x0000000000000000000000005d95ae932d42e53bb9da4de65e9b7263a4fa8564"),
			},
			Data: ethcommon.FromHex("0x0000000000000000000000000000000000000000000009513ea9de0243800000"),
		}

		assert.Equal(t, []*types.Operation{
			{
				OperationIdentifier: &types.OperationIdentifier{
					Index: 1,
				},
				Type:   OpErc20Transfer,
				Status: types.String(StatusSuccess),
				Account: &types.AccountIdentifier{
					Address: "0xf1B77573A8525aCfa116a785092d1Ba90D96BF37",
				},
				Amount: &types.Amount{
					Value:    "-44000000000000000000000",
					Currency: WAVAX,
				},
			},
			{
				OperationIdentifier: &types.OperationIdentifier{
					Index: 2,
				},
				RelatedOperations: []*types.OperationIdentifier{
					{
						Index: 1,
					},
				},
				Type:   OpErc20Transfer,
				Status: types.String(StatusSuccess),
				Account: &types.AccountIdentifier{
					Address: "0x5d95ae932D42E53Bb9DA4DE65E9b7263A4fA8564",
				},
				Amount: &types.Amount{
					Value:    "44000000000000000000000",
					Currency: WAVAX,
				},
			},
		}, erc20Ops(log, WAVAX, 1))
	})

	t.Run("burn op", func(t *testing.T) {
		log := &ethtypes.Log{
			Address: ethcommon.HexToAddress("0xB31f66AA3C1e785363F0875A1B74E27b85FD66c7"),
			Topics: []ethcommon.Hash{
				ethcommon.HexToHash("0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef"),
				ethcommon.HexToHash("0x000000000000000000000000f1b77573a8525acfa116a785092d1ba90d96bf37"),
				ethcommon.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000000"),
			},
			Data: ethcommon.FromHex("0x0000000000000000000000000000000000000000000009513ea9de0243800000"),
		}

		assert.Equal(t, []*types.Operation{
			{
				OperationIdentifier: &types.OperationIdentifier{
					Index: 1,
				},
				Type:   OpErc20Burn,
				Status: types.String(StatusSuccess),
				Account: &types.AccountIdentifier{
					Address: "0xf1B77573A8525aCfa116a785092d1Ba90D96BF37",
				},
				Amount: &types.Amount{
					Value:    "-44000000000000000000000",
					Currency: WAVAX,
				},
			},
		}, erc20Ops(log, WAVAX, 1))
	})

	t.Run("mint op", func(t *testing.T) {
		log := &ethtypes.Log{
			Address: ethcommon.HexToAddress("0xB31f66AA3C1e785363F0875A1B74E27b85FD66c7"),
			Topics: []ethcommon.Hash{
				ethcommon.HexToHash("0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef"),
				ethcommon.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000000"),
				ethcommon.HexToHash("0x000000000000000000000000f1b77573a8525acfa116a785092d1ba90d96bf37"),
			},
			Data: ethcommon.FromHex("0x0000000000000000000000000000000000000000000009513ea9de0243800000"),
		}

		assert.Equal(t, []*types.Operation{
			{
				OperationIdentifier: &types.OperationIdentifier{
					Index: 1,
				},
				Type:   OpErc20Mint,
				Status: types.String(StatusSuccess),
				Account: &types.AccountIdentifier{
					Address: "0xf1B77573A8525aCfa116a785092d1Ba90D96BF37",
				},
				Amount: &types.Amount{
					Value:    "44000000000000000000000",
					Currency: WAVAX,
				},
			},
		}, erc20Ops(log, WAVAX, 1))
	})
}

func TestERC721Ops(t *testing.T) {
	t.Run("transfer op", func(t *testing.T) {
		log := &ethtypes.Log{
			Address: ethcommon.HexToAddress("0xB31f66AA3C1e785363F0875A1B74E27b85FD66c7"),
			Topics: []ethcommon.Hash{
				ethcommon.HexToHash("0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef"),
				ethcommon.HexToHash("0x000000000000000000000000f1b77573a8525acfa116a785092d1ba90d96bf37"),
				ethcommon.HexToHash("0x0000000000000000000000005d95ae932d42e53bb9da4de65e9b7263a4fa8564"),
				ethcommon.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000051"),
			},
		}

		assert.Equal(t, []*types.Operation{
			{
				OperationIdentifier: &types.OperationIdentifier{
					Index: 1,
				},
				Type:   OpErc721TransferSender,
				Status: types.String(StatusSuccess),
				Account: &types.AccountIdentifier{
					Address: "0xf1B77573A8525aCfa116a785092d1Ba90D96BF37",
				},
				Metadata: map[string]interface{}{
					ContractAddressMetadata:  "0xB31f66AA3C1e785363F0875A1B74E27b85FD66c7",
					IndexTransferredMetadata: "0x0000000000000000000000000000000000000000000000000000000000000051",
				},
			},
			{
				OperationIdentifier: &types.OperationIdentifier{
					Index: 2,
				},
				RelatedOperations: []*types.OperationIdentifier{
					{
						Index: 1,
					},
				},
				Type:   OpErc721TransferReceive,
				Status: types.String(StatusSuccess),
				Account: &types.AccountIdentifier{
					Address: "0x5d95ae932D42E53Bb9DA4DE65E9b7263A4fA8564",
				},
				Metadata: map[string]interface{}{
					ContractAddressMetadata:  "0xB31f66AA3C1e785363F0875A1B74E27b85FD66c7",
					IndexTransferredMetadata: "0x0000000000000000000000000000000000000000000000000000000000000051",
				},
			},
		}, erc721Ops(log, 1))
	})

	t.Run("burn op", func(t *testing.T) {
		log := &ethtypes.Log{
			Address: ethcommon.HexToAddress("0xB31f66AA3C1e785363F0875A1B74E27b85FD66c7"),
			Topics: []ethcommon.Hash{
				ethcommon.HexToHash("0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef"),
				ethcommon.HexToHash("0x000000000000000000000000f1b77573a8525acfa116a785092d1ba90d96bf37"),
				ethcommon.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000000"),
				ethcommon.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000051"),
			},
		}

		assert.Equal(t, []*types.Operation{
			{
				OperationIdentifier: &types.OperationIdentifier{
					Index: 1,
				},
				Type:   OpErc721Burn,
				Status: types.String(StatusSuccess),
				Account: &types.AccountIdentifier{
					Address: "0xf1B77573A8525aCfa116a785092d1Ba90D96BF37",
				},
				Metadata: map[string]interface{}{
					ContractAddressMetadata:  "0xB31f66AA3C1e785363F0875A1B74E27b85FD66c7",
					IndexTransferredMetadata: "0x0000000000000000000000000000000000000000000000000000000000000051",
				},
			},
		}, erc721Ops(log, 1))
	})

	t.Run("mint op", func(t *testing.T) {
		log := &ethtypes.Log{
			Address: ethcommon.HexToAddress("0xB31f66AA3C1e785363F0875A1B74E27b85FD66c7"),
			Topics: []ethcommon.Hash{
				ethcommon.HexToHash("0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef"),
				ethcommon.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000000"),
				ethcommon.HexToHash("0x000000000000000000000000f1b77573a8525acfa116a785092d1ba90d96bf37"),
				ethcommon.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000051"),
			},
		}

		assert.Equal(t, []*types.Operation{
			{
				OperationIdentifier: &types.OperationIdentifier{
					Index: 1,
				},
				Type:   OpErc721Mint,
				Status: types.String(StatusSuccess),
				Account: &types.AccountIdentifier{
					Address: "0xf1B77573A8525aCfa116a785092d1Ba90D96BF37",
				},
				Metadata: map[string]interface{}{
					ContractAddressMetadata:  "0xB31f66AA3C1e785363F0875A1B74E27b85FD66c7",
					IndexTransferredMetadata: "0x0000000000000000000000000000000000000000000000000000000000000051",
				},
			},
		}, erc721Ops(log, 1))
	})
}

func TestCrossChainImportedInputs(t *testing.T) {
	t.Run("Cross chain imported input in metadata", func(t *testing.T) {
		var (
			rawIdx      = 0
			avaxAssetID = "U8iRqJoiJm8xZHAacmvYyZVwqQx6uDNtQeP3CQ6fcgQk3JqnK"
			hexTx       = "0x000000000000000000057fc93d85c6d62c5b2ac0b519c87010ea5294012d1e407030d6acd0021cac10d5000000000000000000000000000000000000000000000000000000000000000000000001d4e3812503247f042cc9bbb3395ceca49e28687726489b0ea2d4dd259fadb8b6000000003d9bdac0ed1d761330cf680efdeb1a42159eb387d6d2950c96f7d28f61bbe2aa0000000500000000772651c00000000100000000000000013158e80abd5a1e1aa716003c9db096792c37962100000000772209123d9bdac0ed1d761330cf680efdeb1a42159eb387d6d2950c96f7d28f61bbe2aa000000010000000900000001309767786d3c3548f34373c858f2bab210c6bd6c837d069e314930273d32acc361e86a05bc5bd251e4cb5809bbca7680361ed3263ff5ba2aa467294bfa000aef00cfb71da5"
			decodeTx, _ = formatting.Decode(formatting.Hex, hexTx)
			tx          = &evm.Tx{}

			networkIdentifier = &types.NetworkIdentifier{
				Network: constants.FujiNetwork,
			}
			chainIDToAliasMapping = map[ids.ID]constants.ChainIDAlias{
				ids.Empty: constants.PChain,
			}
		)

		_, err := evm.Codec.Unmarshal(decodeTx, tx)
		assert.Nil(t, err)
		ops, metadata, err := crossChainTransaction(networkIdentifier, chainIDToAliasMapping, rawIdx, avaxAssetID, tx)
		assert.Nil(t, err)
		assert.Nil(t, metadata[MetadataExportedOutputs])
		assert.Equal(t, AtomicAvaxAmount(big.NewInt(280750)), metadata[MetadataTxFee])

		unsignedImportTx := tx.UnsignedAtomicTx.(*evm.UnsignedImportTx)
		assert.Equal(t, []*types.Operation{
			{
				OperationIdentifier: &types.OperationIdentifier{
					Index: 0,
				},
				Type:   OpImport,
				Status: types.String(StatusSuccess),
				Account: &types.AccountIdentifier{
					Address: "0x3158e80abD5A1e1aa716003C9Db096792C379621",
				},
				Amount: &types.Amount{
					Value:    "1998719250000000000",
					Currency: AvaxCurrency,
				},
				Metadata: map[string]interface{}{
					"asset_id":      "U8iRqJoiJm8xZHAacmvYyZVwqQx6uDNtQeP3CQ6fcgQk3JqnK",
					"blockchain_id": "yH8D7ThNJkxmtkuv2jgBa4P1Rn3Qpr4pPr7QYNfcdoS6k6HWp",
					"meta":          unsignedImportTx.Metadata,
					"network_id":    uint32(5),
					"source_chain":  "11111111111111111111111111111111LpoYY",
					"tx":            "G8BG8APvViEryMzeEMb4YM49TtZeozLPsQ8aR44aZf7waGWJR",
					"tx_ids":        []string{"2ckxSTK6TbZuC2kwoTD5QWmwiGCq4EXMGQpZWcdJK2ye9v4dSE"},
				},
			},
		}, ops)
	})
}

func TestCrossChainExportedOuts(t *testing.T) {
	t.Run("Cross chain exported outputs in metadata", func(t *testing.T) {
		var (
			rawIdx      = 0
			avaxAssetID = "U8iRqJoiJm8xZHAacmvYyZVwqQx6uDNtQeP3CQ6fcgQk3JqnK"
			hexTx       = "000000000001000000057fc93d85c6d62c5b2ac0b519c87010ea5294012d1e407030d6acd0021cac10d50000000000000000000000000000000000000000000000000000000000000000000000013158e80abd5a1e1aa716003c9db096792c3796210000000000138aee3d9bdac0ed1d761330cf680efdeb1a42159eb387d6d2950c96f7d28f61bbe2aa000000000000003b000000013d9bdac0ed1d761330cf680efdeb1a42159eb387d6d2950c96f7d28f61bbe2aa0000000700000000000f424000000000000000000000000100000001c83ea4dc195a9275a349e4f616cbb45e23eab2fb00000001000000090000000167fb4fdaa15ce6804e680dc182f0e702259e6f9572a9f5fe0fc6053094951f612a3d9e8128d08be17ae5122d1790160ac8f2e6d21c4b7dde702624eb6219de7301"
			decodeTx, _ = hex.DecodeString(hexTx)
			tx          = &evm.Tx{}

			networkIdentifier = &types.NetworkIdentifier{
				Network: constants.FujiNetwork,
			}
			chainIDToAliasMapping = map[ids.ID]constants.ChainIDAlias{
				ids.Empty: constants.PChain,
			}
			metaBytes, _         = hex.DecodeString("000000000001000000057fc93d85c6d62c5b2ac0b519c87010ea5294012d1e407030d6acd0021cac10d50000000000000000000000000000000000000000000000000000000000000000000000013158e80abd5a1e1aa716003c9db096792c3796210000000000138aee3d9bdac0ed1d761330cf680efdeb1a42159eb387d6d2950c96f7d28f61bbe2aa000000000000003b000000013d9bdac0ed1d761330cf680efdeb1a42159eb387d6d2950c96f7d28f61bbe2aa0000000700000000000f424000000000000000000000000100000001c83ea4dc195a9275a349e4f616cbb45e23eab2fb00000001000000090000000167fb4fdaa15ce6804e680dc182f0e702259e6f9572a9f5fe0fc6053094951f612a3d9e8128d08be17ae5122d1790160ac8f2e6d21c4b7dde702624eb6219de7301")
			metaUnsignedBytes, _ = hex.DecodeString("000000000001000000057fc93d85c6d62c5b2ac0b519c87010ea5294012d1e407030d6acd0021cac10d50000000000000000000000000000000000000000000000000000000000000000000000013158e80abd5a1e1aa716003c9db096792c3796210000000000138aee3d9bdac0ed1d761330cf680efdeb1a42159eb387d6d2950c96f7d28f61bbe2aa000000000000003b000000013d9bdac0ed1d761330cf680efdeb1a42159eb387d6d2950c96f7d28f61bbe2aa0000000700000000000f424000000000000000000000000100000001c83ea4dc195a9275a349e4f616cbb45e23eab2fb")
			meta                 = &avax.Metadata{}
		)

		meta.Initialize(metaUnsignedBytes, metaBytes)
		_, err := evm.Codec.Unmarshal(decodeTx, tx)
		assert.Nil(t, err)
		ops, metadata, err := crossChainTransaction(networkIdentifier, chainIDToAliasMapping, rawIdx, avaxAssetID, tx)
		assert.Nil(t, err)
		assert.Equal(t, AtomicAvaxAmount(big.NewInt(280750)), metadata[MetadataTxFee])

		assert.Equal(t, []*types.Operation{
			{
				OperationIdentifier: &types.OperationIdentifier{
					Index: 0,
				},
				Type:   OpExport,
				Status: types.String(StatusSuccess),
				Account: &types.AccountIdentifier{
					Address: "0x3158e80abD5A1e1aa716003C9Db096792C379621",
				},
				Amount: &types.Amount{
					Value:    "-1280750000000000",
					Currency: AvaxCurrency,
				},
				Metadata: map[string]interface{}{
					"tx":                "7QUPqUAMdny53bVptZ2DgxLLN4qZ5X7MnBPseUKYnoh5C5v47",
					"blockchain_id":     "yH8D7ThNJkxmtkuv2jgBa4P1Rn3Qpr4pPr7QYNfcdoS6k6HWp",
					"network_id":        uint32(5),
					"destination_chain": "11111111111111111111111111111111LpoYY",
					"meta":              *meta,
					"asset_id":          "U8iRqJoiJm8xZHAacmvYyZVwqQx6uDNtQeP3CQ6fcgQk3JqnK",
				},
			},
		}, ops)

		assert.Equal(t, []*types.Operation{
			{
				OperationIdentifier: &types.OperationIdentifier{
					Index: 1,
				},
				Type:   OpExport,
				Status: types.String(StatusSuccess),
				Account: &types.AccountIdentifier{
					Address: "P-fuji1eql2fhqet2f8tg6funmpdja5tc374vhmdj2xz2",
				},
				Amount: &types.Amount{
					Value:    "1000000",
					Currency: AtomicAvaxCurrency,
				},
				CoinChange: &types.CoinChange{
					CoinIdentifier: &types.CoinIdentifier{
						Identifier: "7QUPqUAMdny53bVptZ2DgxLLN4qZ5X7MnBPseUKYnoh5C5v47:0",
					},
					CoinAction: types.CoinCreated,
				},
			},
		}, metadata[MetadataExportedOutputs])
	})
}
