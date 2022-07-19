package pchain

import (
	"github.com/coinbase/rosetta-sdk-go/types"

	pmapper "github.com/ava-labs/avalanche-rosetta/mapper/pchain"
	"github.com/ava-labs/avalanche-rosetta/service"
)

var (
	_ service.ConstructionBackend = &Backend{}
	_ service.NetworkBackend      = &Backend{}
	_ service.AccountBackend      = &Backend{}
	_ service.BlockBackend        = &Backend{}
)

type Backend struct {
	networkIdentifier *types.NetworkIdentifier
}

func NewBackend(networkIdentifier *types.NetworkIdentifier) (*Backend, error) {
	return &Backend{networkIdentifier: networkIdentifier}, nil
}

func (b *Backend) ShouldHandleRequest(req interface{}) bool {
	switch r := req.(type) {
	case *types.AccountBalanceRequest:
		return pmapper.IsPChain(r.NetworkIdentifier)
	case *types.AccountCoinsRequest:
		return pmapper.IsPChain(r.NetworkIdentifier)
	case *types.BlockRequest:
		return pmapper.IsPChain(r.NetworkIdentifier)
	case *types.BlockTransactionRequest:
		return pmapper.IsPChain(r.NetworkIdentifier)
	case *types.ConstructionDeriveRequest:
		return pmapper.IsPChain(r.NetworkIdentifier)
	case *types.ConstructionMetadataRequest:
		return pmapper.IsPChain(r.NetworkIdentifier)
	case *types.ConstructionPreprocessRequest:
		return pmapper.IsPChain(r.NetworkIdentifier)
	case *types.ConstructionPayloadsRequest:
		return pmapper.IsPChain(r.NetworkIdentifier)
	case *types.ConstructionParseRequest:
		return pmapper.IsPChain(r.NetworkIdentifier)
	case *types.ConstructionCombineRequest:
		return pmapper.IsPChain(r.NetworkIdentifier)
	case *types.ConstructionHashRequest:
		return pmapper.IsPChain(r.NetworkIdentifier)
	case *types.ConstructionSubmitRequest:
		return pmapper.IsPChain(r.NetworkIdentifier)
	case *types.NetworkRequest:
		return pmapper.IsPChain(r.NetworkIdentifier)
	}

	return false
}
