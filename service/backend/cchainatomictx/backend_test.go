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

	backend, _ := NewBackend()

	t.Run("should handle c-chain bech32 request", func(t *testing.T) {
		assert.True(t, backend.ShouldHandleRequest(
			&types.ConstructionDeriveRequest{
				NetworkIdentifier: networkIdentifier,
				Metadata: map[string]interface{}{
					mapper.MetaAddressFormat: mapper.AddressFormatBech32,
				},
			},
		))
	})

	t.Run("should not handle regular c-chain request", func(t *testing.T) {
		assert.False(t, backend.ShouldHandleRequest(
			&types.ConstructionDeriveRequest{
				NetworkIdentifier: networkIdentifier,
			},
		))
	})
}
