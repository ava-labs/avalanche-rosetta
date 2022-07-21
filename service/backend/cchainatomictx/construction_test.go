package cchainatomictx

import (
	"context"
	"encoding/hex"
	"testing"

	"github.com/coinbase/rosetta-sdk-go/types"
	"github.com/stretchr/testify/assert"

	"github.com/ava-labs/avalanche-rosetta/mapper"
	"github.com/ava-labs/avalanche-rosetta/service"
)

var networkIdentifier = &types.NetworkIdentifier{
	Blockchain: service.BlockchainName,
	Network:    mapper.FujiNetwork,
}

func TestConstructionDerive(t *testing.T) {
	backend, _ := NewBackend()

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
