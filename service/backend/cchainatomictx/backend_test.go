package cchainatomictx

import (
	"testing"

	"github.com/ava-labs/coreth/plugin/evm"
	"github.com/coinbase/rosetta-sdk-go/types"
	"github.com/stretchr/testify/assert"

	"github.com/ava-labs/avalanche-rosetta/constants"
	"github.com/ava-labs/avalanche-rosetta/mapper"
	cmapper "github.com/ava-labs/avalanche-rosetta/mapper/cchainatomictx"
	"github.com/ava-labs/avalanche-rosetta/service"
)

func TestShouldHandleRequest(t *testing.T) {
	cChainNetworkIdentifier := &types.NetworkIdentifier{
		Blockchain: service.BlockchainName,
		Network:    constants.FujiNetwork,
	}

	bech32AccountIdentifier := &types.AccountIdentifier{Address: "C-avax1us3us4s4mv0g85vxjm8va04ewdl27wcwnqwejf"}
	evmAccountIdentifier := &types.AccountIdentifier{Address: "0x30cE0c38f953eE9CD5fbc247e63DE68D3263144b"}

	atomicOperations := []*types.Operation{
		{Type: mapper.OpImport},
	}

	evmOperations := []*types.Operation{
		{Type: mapper.OpCall},
	}

	backend := &Backend{
		codec:        evm.Codec,
		codecVersion: 0,
	}

	atomicTxString := `{"tx":"0x000000000000000000057fc93d85c6d62c5b2ac0b519c87010ea5294012d1e407030d6acd0021cac10d500000000000000000000000000000000000000000000000000000000000000000000000288ae5dd070e6d74f16c26358cd4a8f43746d4d338b5b75b668741c6d95816af5000000023d9bdac0ed1d761330cf680efdeb1a42159eb387d6d2950c96f7d28f61bbe2aa000000050000000000e4e1c00000000100000000b9a824340e1b94f27500cdfcbf8eaa9d4ee5e57b2823cb8b158de17689916c74000000013d9bdac0ed1d761330cf680efdeb1a42159eb387d6d2950c96f7d28f61bbe2aa0000000500000000004c4b400000000100000000000000013158e80abd5a1e1aa716003c9db096792c37962100000000012c7a123d9bdac0ed1d761330cf680efdeb1a42159eb387d6d2950c96f7d28f61bbe2aa000000020000000900000001a06d20d1d175b1e1d2b6e647ab5321717967de7e9367c28df8c0e20634ec7827019fe38e8d4f123f8e5286f3236db8dbb419e264628e2f17330a6c8da60d3424010000000900000001a06d20d1d175b1e1d2b6e647ab5321717967de7e9367c28df8c0e20634ec7827019fe38e8d4f123f8e5286f3236db8dbb419e264628e2f17330a6c8da60d342401dc68b1fc","signers":[{"coin_identifier":"23CLURk1Czf1aLui1VdcuWSiDeFskfp3Sn8TQG7t6NKfeQRYDj:2","account_identifier":{"address":"P-fuji1wmd9dfrqpud6daq0cde47u0r7pkrr46ep60399"}},{"coin_identifier":"2QmMXKS6rKQMnEh2XYZ4ZWCJmy8RpD3LyVZWxBG25t4N1JJqxY:1","account_identifier":{"address":"P-fuji1wmd9dfrqpud6daq0cde47u0r7pkrr46ep60399"}}]}`

	nonAtomicTxString := "evmtxstring"

	t.Run("return true for c-chain atomic tx requests", func(t *testing.T) {
		assert.True(t, backend.ShouldHandleRequest(&types.AccountBalanceRequest{
			NetworkIdentifier: cChainNetworkIdentifier,
			AccountIdentifier: bech32AccountIdentifier,
		}))
		assert.True(t, backend.ShouldHandleRequest(&types.AccountCoinsRequest{
			NetworkIdentifier: cChainNetworkIdentifier,
			AccountIdentifier: bech32AccountIdentifier,
		}))
		assert.True(t, backend.ShouldHandleRequest(&types.ConstructionDeriveRequest{
			NetworkIdentifier: cChainNetworkIdentifier,
			Metadata: map[string]interface{}{
				mapper.MetadataAddressFormat: mapper.AddressFormatBech32,
			},
		}))
		assert.True(t, backend.ShouldHandleRequest(&types.ConstructionMetadataRequest{
			NetworkIdentifier: cChainNetworkIdentifier,
			Options: map[string]interface{}{
				cmapper.MetadataAtomicTxGas: 123,
			},
		}))
		assert.True(t, backend.ShouldHandleRequest(&types.ConstructionPreprocessRequest{
			NetworkIdentifier: cChainNetworkIdentifier,
			Operations:        atomicOperations,
		}))
		assert.True(t, backend.ShouldHandleRequest(&types.ConstructionPayloadsRequest{
			NetworkIdentifier: cChainNetworkIdentifier,
			Operations:        atomicOperations,
		}))
		assert.True(t, backend.ShouldHandleRequest(&types.ConstructionParseRequest{
			NetworkIdentifier: cChainNetworkIdentifier,
			Transaction:       atomicTxString,
			Signed:            true,
		}))
		assert.True(t, backend.ShouldHandleRequest(&types.ConstructionCombineRequest{
			NetworkIdentifier:   cChainNetworkIdentifier,
			UnsignedTransaction: atomicTxString,
		}))
		assert.True(t, backend.ShouldHandleRequest(&types.ConstructionHashRequest{
			NetworkIdentifier: cChainNetworkIdentifier,
			SignedTransaction: atomicTxString,
		}))
		assert.True(t, backend.ShouldHandleRequest(&types.ConstructionSubmitRequest{
			NetworkIdentifier: cChainNetworkIdentifier,
			SignedTransaction: atomicTxString,
		}))
	})

	t.Run("return false for other c-chain requests", func(t *testing.T) {
		assert.False(t, backend.ShouldHandleRequest(&types.AccountBalanceRequest{
			NetworkIdentifier: cChainNetworkIdentifier,
			AccountIdentifier: evmAccountIdentifier,
		}))
		assert.False(t, backend.ShouldHandleRequest(&types.AccountCoinsRequest{
			NetworkIdentifier: cChainNetworkIdentifier,
			AccountIdentifier: evmAccountIdentifier,
		}))
		assert.False(t, backend.ShouldHandleRequest(&types.ConstructionDeriveRequest{
			NetworkIdentifier: cChainNetworkIdentifier,
		}))
		assert.False(t, backend.ShouldHandleRequest(&types.ConstructionMetadataRequest{
			NetworkIdentifier: cChainNetworkIdentifier,
		}))
		assert.False(t, backend.ShouldHandleRequest(&types.ConstructionPreprocessRequest{
			NetworkIdentifier: cChainNetworkIdentifier,
			Operations:        evmOperations,
		}))
		assert.False(t, backend.ShouldHandleRequest(&types.ConstructionPayloadsRequest{
			NetworkIdentifier: cChainNetworkIdentifier,
			Operations:        evmOperations,
		}))
		assert.False(t, backend.ShouldHandleRequest(&types.ConstructionParseRequest{
			NetworkIdentifier: cChainNetworkIdentifier,
			Transaction:       nonAtomicTxString,
		}))
		assert.False(t, backend.ShouldHandleRequest(&types.ConstructionCombineRequest{
			NetworkIdentifier:   cChainNetworkIdentifier,
			UnsignedTransaction: nonAtomicTxString,
		}))
		assert.False(t, backend.ShouldHandleRequest(&types.ConstructionHashRequest{
			NetworkIdentifier: cChainNetworkIdentifier,
			SignedTransaction: nonAtomicTxString,
		}))
		assert.False(t, backend.ShouldHandleRequest(&types.ConstructionSubmitRequest{
			NetworkIdentifier: cChainNetworkIdentifier,
			SignedTransaction: nonAtomicTxString,
		}))

		// Backend does not support /block and /network endpoints
		assert.False(t, backend.ShouldHandleRequest(&types.BlockRequest{
			NetworkIdentifier: cChainNetworkIdentifier,
		}))
		assert.False(t, backend.ShouldHandleRequest(&types.BlockTransactionRequest{
			NetworkIdentifier: cChainNetworkIdentifier,
		}))
		assert.False(t, backend.ShouldHandleRequest(&types.NetworkRequest{
			NetworkIdentifier: cChainNetworkIdentifier,
		}))
	})
}
