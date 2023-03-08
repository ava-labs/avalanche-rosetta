package pchain

import (
	"context"
	"fmt"
	"testing"

	"github.com/coinbase/rosetta-sdk-go/types"
	"github.com/stretchr/testify/assert"

	"github.com/ava-labs/avalanche-rosetta/constants"
	mocks "github.com/ava-labs/avalanche-rosetta/mocks/client"
	idxmocks "github.com/ava-labs/avalanche-rosetta/mocks/service/backend/pchain/indexer"
	"github.com/ava-labs/avalanche-rosetta/service"
	"github.com/ava-labs/avalanchego/ids"
)

func TestShouldHandleRequest(t *testing.T) {
	pChainNetworkIdentifier := &types.NetworkIdentifier{
		Blockchain: service.BlockchainName,
		Network:    constants.FujiNetwork,
		SubNetworkIdentifier: &types.SubNetworkIdentifier{
			Network: constants.PChain.String(),
		},
	}

	cChainNetworkIdentifier := &types.NetworkIdentifier{
		Blockchain: service.BlockchainName,
		Network:    constants.FujiNetwork,
	}

	ctx := context.Background()
	clientMock := &mocks.PChainClient{}
	clientMock.Mock.On("GetBlockchainID", ctx, constants.CChain.String()).Return(ids.ID{'C'}, nil)
	clientMock.Mock.On("GetBlockchainID", ctx, constants.XChain.String()).Return(ids.ID{'X'}, nil)
	parserMock := &idxmocks.Parser{}
	parserMock.Mock.On("GetGenesisBlock", ctx).Return(dummyGenesis, nil)
	backend, err := NewBackend(
		service.ModeOnline,
		clientMock,
		parserMock,
		avaxAssetID,
		pChainNetworkIdentifier,
		avalancheNetworkID,
	)
	assert.Nil(t, err)

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
