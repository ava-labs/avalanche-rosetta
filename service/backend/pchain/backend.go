package pchain

import (
	"github.com/ava-labs/avalanchego/codec"
	"github.com/ava-labs/avalanchego/genesis"
	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/avalanchego/upgrade"
	"github.com/ava-labs/avalanchego/vms/platformvm/block"
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
	upgradeConfig      upgrade.Config
	feeConfig          genesis.TxFeeConfig
}

// NewBackend creates a P-chain service backend
func NewBackend(
	pClient client.PChainClient,
	indexerParser indexer.Parser,
	avaxAssetID ids.ID,
	networkIdentifier *types.NetworkIdentifier,
	avalancheNetworkID uint32,
) (*Backend, error) {
	genHandler, err := newGenesisHandler(indexerParser)
	if err != nil {
		return nil, err
	}

	networkHRP, err := mapper.GetHRP(networkIdentifier)
	if err != nil {
		return nil, err
	}

	upgradeConfig := upgrade.GetConfig(avalancheNetworkID)
	feeConfig := genesis.GetTxFeeConfig(avalancheNetworkID)

	return &Backend{
		genesisHandler:     genHandler,
		networkID:          networkIdentifier,
		networkHRP:         networkHRP,
		avalancheNetworkID: avalancheNetworkID,
		pClient:            pClient,
		indexerParser:      indexerParser,
		getUTXOsPageSize:   1024,
		codec:              block.Codec,
		codecVersion:       block.CodecVersion,
		avaxAssetID:        avaxAssetID,
		txParserCfg: pmapper.TxParserConfig{
			IsConstruction: false,
			Hrp:            networkHRP,
			ChainIDs:       nil,
			AvaxAssetID:    avaxAssetID,
			PChainClient:   pClient,
		},
		upgradeConfig: upgradeConfig,
		feeConfig:     feeConfig,
	}, nil
}

// ShouldHandleRequest returns whether a given request should be handled by this backend
func (*Backend) ShouldHandleRequest(req interface{}) bool {
	switch r := req.(type) {
	case *types.AccountBalanceRequest:
		return isPChain(r.NetworkIdentifier)
	case *types.AccountCoinsRequest:
		return isPChain(r.NetworkIdentifier)
	case *types.BlockRequest:
		return isPChain(r.NetworkIdentifier)
	case *types.BlockTransactionRequest:
		return isPChain(r.NetworkIdentifier)
	case *types.ConstructionDeriveRequest:
		return isPChain(r.NetworkIdentifier)
	case *types.ConstructionMetadataRequest:
		return isPChain(r.NetworkIdentifier)
	case *types.ConstructionPreprocessRequest:
		return isPChain(r.NetworkIdentifier)
	case *types.ConstructionPayloadsRequest:
		return isPChain(r.NetworkIdentifier)
	case *types.ConstructionParseRequest:
		return isPChain(r.NetworkIdentifier)
	case *types.ConstructionCombineRequest:
		return isPChain(r.NetworkIdentifier)
	case *types.ConstructionHashRequest:
		return isPChain(r.NetworkIdentifier)
	case *types.ConstructionSubmitRequest:
		return isPChain(r.NetworkIdentifier)
	case *types.NetworkRequest:
		return isPChain(r.NetworkIdentifier)
	}

	return false
}

// isPChain checks network identifier to make sure sub-network identifier set to "P"
func isPChain(reqNetworkID *types.NetworkIdentifier) bool {
	return reqNetworkID != nil &&
		reqNetworkID.SubNetworkIdentifier != nil &&
		reqNetworkID.SubNetworkIdentifier.Network == constants.PChain.String()
}
