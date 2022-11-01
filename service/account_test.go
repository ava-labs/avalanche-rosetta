package service

import (
	"context"
	"testing"

	"github.com/coinbase/rosetta-sdk-go/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/ava-labs/avalanche-rosetta/backend"
	cBackend "github.com/ava-labs/avalanche-rosetta/backend/cchain"
	"github.com/ava-labs/avalanche-rosetta/constants"
	cltmocks "github.com/ava-labs/avalanche-rosetta/mocks/client"
	mocks "github.com/ava-labs/avalanche-rosetta/mocks/service"
)

func TestAccountBalance(t *testing.T) {
	cChainBackend := cBackend.NewBackend(
		&cBackend.Config{
			Mode: constants.Online,
		},
		&cltmocks.Client{},
	)
	pBackendMock := &mocks.AccountBackend{}
	atomicBackendMock := &mocks.AccountBackend{}
	service := NewAccountService(constants.Online, cChainBackend, pBackendMock, atomicBackendMock)

	t.Run("p-chain request is delegated to p-chain backend", func(t *testing.T) {
		req := &types.AccountBalanceRequest{
			NetworkIdentifier: &types.NetworkIdentifier{
				Network: constants.FujiNetwork,
				SubNetworkIdentifier: &types.SubNetworkIdentifier{
					Network: constants.PChain.String(),
				},
			},
			AccountIdentifier: &types.AccountIdentifier{
				Address: "P-fuji15f9g0h5xkr5cp47n6u3qxj6yjtzzzrdr23a3tl",
			},
		}

		expectedResp := &types.AccountBalanceResponse{}
		pBackendMock.On("ShouldHandleRequest", req).Return(true)
		pBackendMock.On("AccountBalance", mock.Anything, req).Return(expectedResp, nil)

		resp, err := service.AccountBalance(context.Background(), req)

		assert.Nil(t, err)
		assert.Equal(t, expectedResp, resp)
		pBackendMock.AssertExpectations(t)
	})

	t.Run("c-chain atomic request is delegated to c-chain atomic tx backend", func(t *testing.T) {
		req := &types.AccountBalanceRequest{
			NetworkIdentifier: &types.NetworkIdentifier{
				Network: constants.FujiNetwork,
			},
			AccountIdentifier: &types.AccountIdentifier{
				Address: "C-fuji15f9g0h5xkr5cp47n6u3qxj6yjtzzzrdr23a3tl",
			},
		}

		expectedResp := &types.AccountBalanceResponse{}
		pBackendMock.On("ShouldHandleRequest", req).Return(false)
		atomicBackendMock.On("ShouldHandleRequest", req).Return(true)
		atomicBackendMock.On("AccountBalance", mock.Anything, req).Return(expectedResp, nil)

		resp, err := service.AccountBalance(context.Background(), req)

		assert.Nil(t, err)
		assert.Equal(t, expectedResp, resp)
		atomicBackendMock.AssertExpectations(t)
	})
}

func TestAccountCoins(t *testing.T) {
	cChainBackend := cBackend.NewBackend(
		&cBackend.Config{
			Mode: constants.Online,
		},
		&cltmocks.Client{},
	)
	pBackendMock := &mocks.AccountBackend{}
	atomicBackendMock := &mocks.AccountBackend{}
	service := NewAccountService(constants.Online, cChainBackend, pBackendMock, atomicBackendMock)

	t.Run("p-chain request is delegated to p-chain backend", func(t *testing.T) {
		req := &types.AccountCoinsRequest{
			NetworkIdentifier: &types.NetworkIdentifier{
				Network: constants.FujiNetwork,
				SubNetworkIdentifier: &types.SubNetworkIdentifier{
					Network: constants.PChain.String(),
				},
			},
			AccountIdentifier: &types.AccountIdentifier{
				Address: "P-fuji15f9g0h5xkr5cp47n6u3qxj6yjtzzzrdr23a3tl",
			},
		}

		expectedResp := &types.AccountCoinsResponse{}

		pBackendMock.On("ShouldHandleRequest", req).Return(true)
		pBackendMock.On("AccountCoins", mock.Anything, req).Return(expectedResp, nil)

		resp, err := service.AccountCoins(context.Background(), req)

		assert.Nil(t, err)
		assert.Equal(t, expectedResp, resp)
		pBackendMock.AssertExpectations(t)
	})

	t.Run("c-chain atomic request is delegated to c-chain atomic tx backend", func(t *testing.T) {
		req := &types.AccountCoinsRequest{
			NetworkIdentifier: &types.NetworkIdentifier{
				Network: constants.FujiNetwork,
			},
			AccountIdentifier: &types.AccountIdentifier{
				Address: "C-fuji15f9g0h5xkr5cp47n6u3qxj6yjtzzzrdr23a3tl",
			},
		}

		expectedResp := &types.AccountCoinsResponse{}

		pBackendMock.On("ShouldHandleRequest", req).Return(false)
		atomicBackendMock.On("ShouldHandleRequest", req).Return(true)
		atomicBackendMock.On("AccountCoins", mock.Anything, req).Return(expectedResp, nil)

		resp, err := service.AccountCoins(context.Background(), req)

		assert.Nil(t, err)
		assert.Equal(t, expectedResp, resp)
		atomicBackendMock.AssertExpectations(t)
	})

	t.Run("c-chain regular request is not supported", func(t *testing.T) {
		req := &types.AccountCoinsRequest{
			NetworkIdentifier: &types.NetworkIdentifier{
				Network: constants.FujiNetwork,
			},
			AccountIdentifier: &types.AccountIdentifier{
				Address: "0x197E90f9FAD81970bA7976f33CbD77088E5D7cf7",
			},
		}

		pBackendMock.On("ShouldHandleRequest", req).Return(false)
		atomicBackendMock.On("ShouldHandleRequest", req).Return(false)

		resp, err := service.AccountCoins(context.Background(), req)

		assert.Equal(t, backend.ErrNotImplemented, err)
		assert.Nil(t, resp)
		atomicBackendMock.AssertExpectations(t)
	})
}
