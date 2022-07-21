package pchain

import (
	"fmt"
	"testing"

	"github.com/coinbase/rosetta-sdk-go/types"
	"github.com/stretchr/testify/assert"

	"github.com/ava-labs/avalanche-rosetta/mapper"
	"github.com/ava-labs/avalanche-rosetta/service"
)

func TestShouldHandleRequest(t *testing.T) {
	pChainNetworkIdentifier := &types.NetworkIdentifier{
		Blockchain: service.BlockchainName,
		Network:    mapper.FujiNetwork,
		SubNetworkIdentifier: &types.SubNetworkIdentifier{
			Network: mapper.PChainNetworkIdentifier,
		},
	}

	cChainNetworkIdentifier := &types.NetworkIdentifier{
		Blockchain: service.BlockchainName,
		Network:    mapper.FujiNetwork,
	}

	backend := NewBackend(nil, pChainNetworkIdentifier)

	testData := []struct {
		name              string
		networkIdentifier *types.NetworkIdentifier
		expected          bool
	}{
		{"p-chain", pChainNetworkIdentifier, true},
		{"c-chain", cChainNetworkIdentifier, false},
	}

	for _, tc := range testData {
		t.Run(fmt.Sprintf("should handle request for %s should return %t", tc.name, tc.expected), func(t *testing.T) {
			requests := []interface{}{
				&types.ConstructionDeriveRequest{NetworkIdentifier: tc.networkIdentifier},
				&types.ConstructionPreprocessRequest{NetworkIdentifier: tc.networkIdentifier},
				&types.ConstructionMetadataRequest{NetworkIdentifier: tc.networkIdentifier},
				&types.ConstructionPayloadsRequest{NetworkIdentifier: tc.networkIdentifier},
				&types.ConstructionCombineRequest{NetworkIdentifier: tc.networkIdentifier},
				&types.ConstructionHashRequest{NetworkIdentifier: tc.networkIdentifier},
				&types.ConstructionSubmitRequest{NetworkIdentifier: tc.networkIdentifier},
				&types.AccountBalanceRequest{NetworkIdentifier: tc.networkIdentifier},
				&types.AccountCoinsRequest{NetworkIdentifier: tc.networkIdentifier},
				&types.AccountBalanceRequest{NetworkIdentifier: tc.networkIdentifier},
				&types.AccountCoinsRequest{NetworkIdentifier: tc.networkIdentifier},
				&types.BlockRequest{NetworkIdentifier: tc.networkIdentifier},
				&types.BlockTransactionRequest{NetworkIdentifier: tc.networkIdentifier},
				&types.NetworkRequest{NetworkIdentifier: tc.networkIdentifier},
			}
			for _, r := range requests {
				assert.Equal(t, tc.expected, backend.ShouldHandleRequest(r))
			}
		})
	}
}
