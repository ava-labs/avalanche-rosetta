package pchain

import (
	"context"

	"github.com/ava-labs/avalanchego/codec"
	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/avalanchego/utils/crypto"
	"github.com/ava-labs/avalanchego/vms/platformvm/blocks"
	"github.com/coinbase/rosetta-sdk-go/types"

	"github.com/ava-labs/avalanche-rosetta/backend/pchain/indexer"
	"github.com/ava-labs/avalanche-rosetta/client"
	"github.com/ava-labs/avalanche-rosetta/constants"
	"github.com/ava-labs/avalanche-rosetta/mapper"
	"github.com/ava-labs/avalanche-rosetta/service"

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
	fac              *crypto.FactorySECP256K1R
	networkID        *types.NetworkIdentifier
	networkHRP       string
	pClient          client.PChainClient
	indexerParser    indexer.Parser
	getUTXOsPageSize uint32
	codec            codec.Manager
	codecVersion     uint16
	chainIDs         map[ids.ID]constants.ChainIDAlias
	avaxAssetID      ids.ID
	txParserCfg      pmapper.TxParserConfig
}

// NewBackend creates a P-chain service backend
func NewBackend(
	nodeMode constants.NodeMode,
	pClient client.PChainClient,
	indexerParser indexer.Parser,
	assetID ids.ID,
	networkIdentifier *types.NetworkIdentifier,
) (*Backend, error) {
	genHandler, err := newGenesisHandler(indexerParser)
	if err != nil {
		return nil, err
	}

	backEnd := &Backend{
		genesisHandler:   genHandler,
		fac:              &crypto.FactorySECP256K1R{},
		networkID:        networkIdentifier,
		pClient:          pClient,
		getUTXOsPageSize: 1024,
		codec:            blocks.Codec,
		codecVersion:     blocks.Version,
		indexerParser:    indexerParser,
		chainIDs:         map[ids.ID]constants.ChainIDAlias{},
		avaxAssetID:      assetID,
	}

	if nodeMode == constants.Online {
		if err := backEnd.initChainIDs(); err != nil {
			return nil, err
		}
	}

	backEnd.networkHRP, err = mapper.GetHRP(networkIdentifier)
	if err != nil {
		return nil, err
	}

	backEnd.txParserCfg = pmapper.TxParserConfig{
		IsConstruction: false,
		Hrp:            backEnd.networkHRP,
		ChainIDs:       backEnd.chainIDs,
		AvaxAssetID:    backEnd.avaxAssetID,
		PChainClient:   backEnd.pClient,
	}
	return backEnd, err
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

func (b *Backend) initChainIDs() error {
	ctx := context.Background()
	b.chainIDs = map[ids.ID]constants.ChainIDAlias{
		ids.Empty: constants.PChain,
	}

	cChainID, err := b.pClient.GetBlockchainID(ctx, constants.CChain.String())
	if err != nil {
		return err
	}
	b.chainIDs[cChainID] = constants.CChain

	xChainID, err := b.pClient.GetBlockchainID(ctx, constants.XChain.String())
	if err != nil {
		return err
	}
	b.chainIDs[xChainID] = constants.XChain

	return nil
}

// isPChain checks network identifier to make sure sub-network identifier set to "P"
func (b *Backend) isPChain(reqNetworkID *types.NetworkIdentifier) bool {
	return reqNetworkID != nil &&
		reqNetworkID.SubNetworkIdentifier != nil &&
		reqNetworkID.SubNetworkIdentifier.Network == constants.PChain.String()
}
