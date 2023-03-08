package pchain

import (
	"github.com/ava-labs/avalanchego/codec"
	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/avalanchego/utils/crypto/secp256k1"
	"github.com/ava-labs/avalanchego/vms/platformvm/blocks"
	"github.com/coinbase/rosetta-sdk-go/types"

	"github.com/ava-labs/avalanche-rosetta/client"
	"github.com/ava-labs/avalanche-rosetta/constants"
	"github.com/ava-labs/avalanche-rosetta/mapper"
	"github.com/ava-labs/avalanche-rosetta/service"
	"github.com/ava-labs/avalanche-rosetta/service/backend/pchain/indexer"

	pmapper "github.com/ava-labs/avalanche-rosetta/mapper/pchain"
)

var (
	_ service.ConstructionBackend = &Backend{}
	_ service.NetworkBackend      = &Backend{}
	_ service.AccountBackend      = &Backend{}
	_ service.BlockBackend        = &Backend{}
)

type Backend struct {
	genesisHandler
	fac                *secp256k1.Factory
	networkID          *types.NetworkIdentifier
	networkHRP         string
	avalancheNetworkID uint32
	pClient            client.PChainClient
	indexerParser      indexer.Parser
	getUTXOsPageSize   uint32
	codec              codec.Manager
	codecVersion       uint16
	avaxAssetID        ids.ID
	txParserCfg        pmapper.TxParserConfig
}

// NewBackend creates a P-chain service backend
func NewBackend(
	nodeMode string,
	pClient client.PChainClient,
	indexerParser indexer.Parser,
	assetID ids.ID,
	networkIdentifier *types.NetworkIdentifier,
	avalancheNetworkID uint32,
) (*Backend, error) {
	genHandler, err := newGenesisHandler(indexerParser)
	if err != nil {
		return nil, err
	}

	b := &Backend{
		genesisHandler:     genHandler,
		fac:                &secp256k1.Factory{},
		networkID:          networkIdentifier,
		pClient:            pClient,
		getUTXOsPageSize:   1024,
		codec:              blocks.Codec,
		codecVersion:       blocks.Version,
		indexerParser:      indexerParser,
		avaxAssetID:        assetID,
		avalancheNetworkID: avalancheNetworkID,
	}

	if nodeMode == service.ModeOnline {
		var err error
		if b.networkHRP, err = mapper.GetHRP(b.networkID); err != nil {
			return nil, err
		}
	}

	b.txParserCfg = pmapper.TxParserConfig{
		IsConstruction: false,
		Hrp:            b.networkHRP,
		ChainIDs:       nil,
		AvaxAssetID:    b.avaxAssetID,
		PChainClient:   b.pClient,
	}

	return b, nil
}

// ShouldHandleRequest returns whether a given request should be handled by this backend
func (b *Backend) ShouldHandleRequest(req interface{}) bool {
	switch r := req.(type) {
	case *types.AccountBalanceRequest:
		return b.isPChain(r.NetworkIdentifier)
	case *types.AccountCoinsRequest:
		return b.isPChain(r.NetworkIdentifier)
	case *types.BlockRequest:
		return b.isPChain(r.NetworkIdentifier)
	case *types.BlockTransactionRequest:
		return b.isPChain(r.NetworkIdentifier)
	case *types.ConstructionDeriveRequest:
		return b.isPChain(r.NetworkIdentifier)
	case *types.ConstructionMetadataRequest:
		return b.isPChain(r.NetworkIdentifier)
	case *types.ConstructionPreprocessRequest:
		return b.isPChain(r.NetworkIdentifier)
	case *types.ConstructionPayloadsRequest:
		return b.isPChain(r.NetworkIdentifier)
	case *types.ConstructionParseRequest:
		return b.isPChain(r.NetworkIdentifier)
	case *types.ConstructionCombineRequest:
		return b.isPChain(r.NetworkIdentifier)
	case *types.ConstructionHashRequest:
		return b.isPChain(r.NetworkIdentifier)
	case *types.ConstructionSubmitRequest:
		return b.isPChain(r.NetworkIdentifier)
	case *types.NetworkRequest:
		return b.isPChain(r.NetworkIdentifier)
	}

	return false
}

// isPChain checks network identifier to make sure sub-network identifier set to "P"
func (b *Backend) isPChain(reqNetworkID *types.NetworkIdentifier) bool {
	return reqNetworkID != nil &&
		reqNetworkID.SubNetworkIdentifier != nil &&
		reqNetworkID.SubNetworkIdentifier.Network == constants.PChain.String()
}
