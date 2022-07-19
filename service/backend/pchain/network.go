package pchain

import (
	"context"

	"github.com/coinbase/rosetta-sdk-go/types"

	"github.com/ava-labs/avalanche-rosetta/service"
)

func (b *Backend) NetworkIdentifier() *types.NetworkIdentifier {
	return b.networkIdentifier
}

func (b *Backend) NetworkStatus(ctx context.Context, request *types.NetworkRequest) (*types.NetworkStatusResponse, *types.Error) {
	return nil, service.ErrNotImplemented
}

func (b *Backend) NetworkOptions(ctx context.Context, request *types.NetworkRequest) (*types.NetworkOptionsResponse, *types.Error) {
	return nil, service.ErrNotImplemented
}
