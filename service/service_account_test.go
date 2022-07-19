package service

import (
	"context"
	"testing"

	"github.com/coinbase/rosetta-sdk-go/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/ava-labs/avalanche-rosetta/mapper"
	mocks "github.com/ava-labs/avalanche-rosetta/mocks/service"
)

func TestAccountBalance(t *testing.T) {
	pBackendMock := &mocks.AccountBackend{}
	cBackendMock := &mocks.AccountBackend{}
	service := AccountService{
		config:                &Config{Mode: ModeOnline},
		pChainBackend:         pBackendMock,
		cChainAtomicTxBackend: cBackendMock,
	}
	t.Run("p-chain request is delegated to p-chain backend", func(t *testing.T) {
		req := &types.AccountBalanceRequest{
			NetworkIdentifier: &types.NetworkIdentifier{
				Network: mapper.FujiNetwork,
				SubNetworkIdentifier: &types.SubNetworkIdentifier{
					Network: mapper.PChainNetworkIdentifier,
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
				Network: mapper.FujiNetwork,
			},
			AccountIdentifier: &types.AccountIdentifier{
				Address: "C-fuji15f9g0h5xkr5cp47n6u3qxj6yjtzzzrdr23a3tl",
			},
		}

		expectedResp := &types.AccountBalanceResponse{}
		pBackendMock.On("ShouldHandleRequest", req).Return(false)
		cBackendMock.On("ShouldHandleRequest", req).Return(true)
		cBackendMock.On("AccountBalance", mock.Anything, req).Return(expectedResp, nil)

		resp, err := service.AccountBalance(context.Background(), req)

		assert.Nil(t, err)
		assert.Equal(t, expectedResp, resp)
		cBackendMock.AssertExpectations(t)
	})
}

func TestAccountCoins(t *testing.T) {
	pBackendMock := &mocks.AccountBackend{}
	cBackendMock := &mocks.AccountBackend{}

	service := AccountService{
		config:                &Config{Mode: ModeOnline},
		pChainBackend:         pBackendMock,
		cChainAtomicTxBackend: cBackendMock,
	}
	t.Run("p-chain request is delegated to p-chain backend", func(t *testing.T) {
		req := &types.AccountCoinsRequest{
			NetworkIdentifier: &types.NetworkIdentifier{
				Network: mapper.FujiNetwork,
				SubNetworkIdentifier: &types.SubNetworkIdentifier{
					Network: mapper.PChainNetworkIdentifier,
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
				Network: mapper.FujiNetwork,
			},
			AccountIdentifier: &types.AccountIdentifier{
				Address: "C-fuji15f9g0h5xkr5cp47n6u3qxj6yjtzzzrdr23a3tl",
			},
		}

		expectedResp := &types.AccountCoinsResponse{}

		pBackendMock.On("ShouldHandleRequest", req).Return(false)
		cBackendMock.On("ShouldHandleRequest", req).Return(true)
		cBackendMock.On("AccountCoins", mock.Anything, req).Return(expectedResp, nil)

		resp, err := service.AccountCoins(context.Background(), req)

		assert.Nil(t, err)
		assert.Equal(t, expectedResp, resp)
		cBackendMock.AssertExpectations(t)
	})

	t.Run("c-chain regular request is not supported", func(t *testing.T) {
		req := &types.AccountCoinsRequest{
			NetworkIdentifier: &types.NetworkIdentifier{
				Network: mapper.FujiNetwork,
			},
			AccountIdentifier: &types.AccountIdentifier{
				Address: "0x197E90f9FAD81970bA7976f33CbD77088E5D7cf7",
			},
		}

		pBackendMock.On("ShouldHandleRequest", req).Return(false)
		cBackendMock.On("ShouldHandleRequest", req).Return(false)

		resp, err := service.AccountCoins(context.Background(), req)

		assert.Equal(t, ErrNotImplemented, err)
		assert.Nil(t, resp)
		cBackendMock.AssertExpectations(t)
	})
}
