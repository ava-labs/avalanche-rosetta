package pchain

import (
	"github.com/ava-labs/avalanchego/codec"
	"github.com/ava-labs/avalanchego/utils/crypto"
	"github.com/ava-labs/avalanchego/vms/platformvm"
	"github.com/coinbase/rosetta-sdk-go/types"

	"github.com/ava-labs/avalanche-rosetta/client"
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
	fac               *crypto.FactorySECP256K1R
	networkIdentifier *types.NetworkIdentifier
	pClient           client.PChainClient
	getUTXOsPageSize  uint32
	codec             codec.Manager
	codecVersion      uint16
}

func NewBackend(
	pClient client.PChainClient,
	networkIdentifier *types.NetworkIdentifier,
) *Backend {
	return &Backend{
		fac:               &crypto.FactorySECP256K1R{},
		networkIdentifier: networkIdentifier,
		pClient:           pClient,
		getUTXOsPageSize:  1024,
		codec:             platformvm.Codec,
		codecVersion:      platformvm.CodecVersion,
	}
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
