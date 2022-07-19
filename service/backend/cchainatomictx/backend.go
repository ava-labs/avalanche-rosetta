package cchainatomictx

import (
	"github.com/ava-labs/avalanchego/utils/crypto"
	"github.com/coinbase/rosetta-sdk-go/types"

	"github.com/ava-labs/avalanche-rosetta/mapper"
	"github.com/ava-labs/avalanche-rosetta/service"
)

var (
	_ service.ConstructionBackend = &Backend{}
	_ service.AccountBackend      = &Backend{}
)

type Backend struct {
	fac *crypto.FactorySECP256K1R
}

func NewBackend() (*Backend, error) {
	return &Backend{
		fac: &crypto.FactorySECP256K1R{},
	}, nil
}

func (b *Backend) ShouldHandleRequest(req interface{}) bool {
	switch r := req.(type) {
	case *types.AccountBalanceRequest:
		return false
	case *types.AccountCoinsRequest:
		return false
	case *types.ConstructionDeriveRequest:
		return r.Metadata[mapper.MetaAddressFormat] == mapper.AddressFormatBech32
	case *types.ConstructionMetadataRequest:
		return false
	case *types.ConstructionPreprocessRequest:
		return false
	case *types.ConstructionPayloadsRequest:
		return false
	case *types.ConstructionParseRequest:
		return false
	case *types.ConstructionCombineRequest:
		return false
	case *types.ConstructionHashRequest:
		return false
	case *types.ConstructionSubmitRequest:
		return false
	}

	return false
}
