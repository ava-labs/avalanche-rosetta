package mapper

import (
	"testing"

	clientTypes "github.com/ava-labs/avalanche-rosetta/client"
	ethtypes "github.com/ava-labs/coreth/core/types"
	"github.com/coinbase/rosetta-sdk-go/types"
	ethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"
)

var WAVAX = &types.Currency{
	Symbol:   "WAVAX",
	Decimals: 18,
	Metadata: map[string]interface{}{
		"contractAddress": "0xB31f66AA3C1e785363F0875A1B74E27b85FD66c7",
	},
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

		currency := clientTypes.ContractCurrency{
			Symbol:   WAVAX.Symbol,
			Decimals: WAVAX.Decimals,
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
		}, erc20Ops(log, &currency, 1))
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

		currency := clientTypes.ContractCurrency{
			Symbol:   WAVAX.Symbol,
			Decimals: WAVAX.Decimals,
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
		}, erc20Ops(log, &currency, 1))
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

		currency := clientTypes.ContractCurrency{
			Symbol:   WAVAX.Symbol,
			Decimals: WAVAX.Decimals,
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
		}, erc20Ops(log, &currency, 1))
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
