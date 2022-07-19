package cchainatomictx

import (
	"context"
	"github.com/ava-labs/avalanche-rosetta/service"
	"github.com/coinbase/rosetta-sdk-go/types"
)

func (b *Backend) AccountBalance(ctx context.Context, req *types.AccountBalanceRequest) (*types.AccountBalanceResponse, *types.Error) {
	return nil, service.ErrNotImplemented
}

func (b *Backend) AccountCoins(ctx context.Context, req *types.AccountCoinsRequest) (*types.AccountCoinsResponse, *types.Error) {
	return nil, service.ErrNotImplemented
}
