package service

import (
	"context"
	"testing"

	"github.com/coinbase/rosetta-sdk-go/types"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	"github.com/ava-labs/avalanche-rosetta/constants"
)

func TestAccountBalance(t *testing.T) {
	ctrl := gomock.NewController(t)
	pBackendMock := NewMockAccountBackend(ctrl)
	cBackendMock := NewMockAccountBackend(ctrl)

	service := AccountService{
		config:                &Config{Mode: ModeOnline},
		pChainBackend:         pBackendMock,
		cChainAtomicTxBackend: cBackendMock,
	}
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
		pBackendMock.EXPECT().ShouldHandleRequest(req).Return(true)
		pBackendMock.EXPECT().AccountBalance(gomock.Any(), req).Return(expectedResp, nil)

		resp, err := service.AccountBalance(context.Background(), req)

		assert.Nil(t, err)
		assert.Equal(t, expectedResp, resp)
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
		pBackendMock.EXPECT().ShouldHandleRequest(req).Return(false)
		cBackendMock.EXPECT().ShouldHandleRequest(req).Return(true)
		cBackendMock.EXPECT().AccountBalance(gomock.Any(), req).Return(expectedResp, nil)

		resp, err := service.AccountBalance(context.Background(), req)

		assert.Nil(t, err)
		assert.Equal(t, expectedResp, resp)
	})
}

func TestAccountCoins(t *testing.T) {
	ctrl := gomock.NewController(t)
	pBackendMock := NewMockAccountBackend(ctrl)
	cBackendMock := NewMockAccountBackend(ctrl)

	service := AccountService{
		config:                &Config{Mode: ModeOnline},
		pChainBackend:         pBackendMock,
		cChainAtomicTxBackend: cBackendMock,
	}
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

		pBackendMock.EXPECT().ShouldHandleRequest(req).Return(true)
		pBackendMock.EXPECT().AccountCoins(gomock.Any(), req).Return(expectedResp, nil)

		resp, err := service.AccountCoins(context.Background(), req)

		assert.Nil(t, err)
		assert.Equal(t, expectedResp, resp)
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

		pBackendMock.EXPECT().ShouldHandleRequest(req).Return(false)
		cBackendMock.EXPECT().ShouldHandleRequest(req).Return(true)
		cBackendMock.EXPECT().AccountCoins(gomock.Any(), req).Return(expectedResp, nil)

		resp, err := service.AccountCoins(context.Background(), req)

		assert.Nil(t, err)
		assert.Equal(t, expectedResp, resp)
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

		pBackendMock.EXPECT().ShouldHandleRequest(req).Return(false)
		cBackendMock.EXPECT().ShouldHandleRequest(req).Return(false)

		resp, err := service.AccountCoins(context.Background(), req)

		assert.Equal(t, ErrNotImplemented, err)
		assert.Nil(t, resp)
	})
}
