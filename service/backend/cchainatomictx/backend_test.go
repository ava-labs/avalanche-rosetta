package cchainatomictx

import (
	"testing"

	"github.com/coinbase/rosetta-sdk-go/types"
	"github.com/stretchr/testify/assert"

	"github.com/ava-labs/avalanche-rosetta/mapper"
	"github.com/ava-labs/avalanche-rosetta/service"
)

func TestShouldHandleRequest(t *testing.T) {
	networkIdentifier := &types.NetworkIdentifier{
		Blockchain: service.BlockchainName,
		Network:    mapper.FujiNetwork,
	}

	cBech32Address := "C-avax18xxz9e8323836t5wtpqh6fmrsjnksd6mka3gh7"
	cEvmAddress := "0x8cBE7BdCd93FD767349074CBdD6CB69127eb0950"

	backend := NewBackend(nil)

	t.Run("should handle c-chain bech32 request", func(t *testing.T) {
		assert.True(t, backend.ShouldHandleRequest(
			&types.ConstructionDeriveRequest{
				NetworkIdentifier: networkIdentifier,
				Metadata: map[string]interface{}{
					mapper.MetaAddressFormat: mapper.AddressFormatBech32,
				},
			},
		))
		assert.True(t, backend.ShouldHandleRequest(
			&types.AccountBalanceRequest{
				NetworkIdentifier: networkIdentifier,
				AccountIdentifier: &types.AccountIdentifier{Address: cBech32Address},
			},
		))
		assert.True(t, backend.ShouldHandleRequest(
			&types.AccountCoinsRequest{
				NetworkIdentifier: networkIdentifier,
				AccountIdentifier: &types.AccountIdentifier{Address: cBech32Address},
			},
		))
	})

	t.Run("should not handle regular c-chain request", func(t *testing.T) {
		assert.False(t, backend.ShouldHandleRequest(
			&types.ConstructionDeriveRequest{
				NetworkIdentifier: networkIdentifier,
			},
		))
		assert.False(t, backend.ShouldHandleRequest(
			&types.AccountBalanceRequest{
				NetworkIdentifier: networkIdentifier,
				AccountIdentifier: &types.AccountIdentifier{Address: cEvmAddress},
			},
		))
		assert.False(t, backend.ShouldHandleRequest(
			&types.AccountCoinsRequest{
				NetworkIdentifier: networkIdentifier,
				AccountIdentifier: &types.AccountIdentifier{Address: cEvmAddress},
			},
		))
	})
}
