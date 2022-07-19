package pchain

import (
	"context"

	"github.com/ava-labs/avalanche-rosetta/service"
	"github.com/coinbase/rosetta-sdk-go/types"
)

func (b *Backend) Block(ctx context.Context, request *types.BlockRequest) (*types.BlockResponse, *types.Error) {
	return nil, service.ErrNotImplemented
}

func (b *Backend) BlockTransaction(ctx context.Context, request *types.BlockTransactionRequest) (*types.BlockTransactionResponse, *types.Error) {
	return nil, service.ErrNotImplemented
}
