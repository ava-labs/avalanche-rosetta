package cchainatomictx

import (
	"github.com/ava-labs/avalanchego/codec"
	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/avalanchego/utils/crypto"
	"github.com/ava-labs/coreth/plugin/evm"
	"github.com/coinbase/rosetta-sdk-go/types"

	"github.com/ava-labs/avalanche-rosetta/client"
	"github.com/ava-labs/avalanche-rosetta/mapper"
	cmapper "github.com/ava-labs/avalanche-rosetta/mapper/cchainatomictx"
	"github.com/ava-labs/avalanche-rosetta/service"
)

var (
	_ service.ConstructionBackend = &Backend{}
	_ service.AccountBackend      = &Backend{}
)

type Backend struct {
	fac              *crypto.FactorySECP256K1R
	cClient          client.Client
	getUTXOsPageSize uint32
	codec            codec.Manager
	codecVersion     uint16
	avaxAssetID      ids.ID
}

func NewBackend(cClient client.Client, avaxAssetID ids.ID) *Backend {
	return &Backend{
		fac:              &crypto.FactorySECP256K1R{},
		cClient:          cClient,
		getUTXOsPageSize: 1024,
		codec:            evm.Codec,
		codecVersion:     0,
		avaxAssetID:      avaxAssetID,
	}
}

func (b *Backend) ShouldHandleRequest(req interface{}) bool {
	switch r := req.(type) {
	case *types.AccountBalanceRequest:
		return cmapper.IsCChainBech32Address(r.AccountIdentifier)
	case *types.AccountCoinsRequest:
		return cmapper.IsCChainBech32Address(r.AccountIdentifier)
	case *types.ConstructionDeriveRequest:
		return r.Metadata[mapper.MetadataAddressFormat] == mapper.AddressFormatBech32
	case *types.ConstructionMetadataRequest:
		return r.Options[cmapper.MetadataAtomicTxGas] != nil
	case *types.ConstructionPreprocessRequest:
		return cmapper.IsAtomicOpType(r.Operations[0].Type)
	case *types.ConstructionPayloadsRequest:
		return cmapper.IsAtomicOpType(r.Operations[0].Type)
	case *types.ConstructionParseRequest:
		return b.isCchainAtomicTx(r.Transaction)
	case *types.ConstructionCombineRequest:
		return b.isCchainAtomicTx(r.UnsignedTransaction)
	case *types.ConstructionHashRequest:
		return b.isCchainAtomicTx(r.SignedTransaction)
	case *types.ConstructionSubmitRequest:
		return b.isCchainAtomicTx(r.SignedTransaction)
	}

	return false
}

func (b *Backend) isCchainAtomicTx(transaction string) bool {
	_, err := b.parsePayloadTxFromString(transaction)
	return err == nil
}
