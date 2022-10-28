package cchain

import (
	"github.com/ava-labs/avalanche-rosetta/client"
	"github.com/ava-labs/avalanche-rosetta/service"
	"github.com/coinbase/rosetta-sdk-go/types"
)

var (
	// _ service.ConstructionBackend = &Backend{}
	// _ service.NetworkBackend      = &Backend{}
	// _ service.AccountBackend      = &Backend{}
	_ service.BlockBackend = &Backend{}
)

type Backend struct {
	config *Config
	client client.Client

	genesisBlock *types.Block
}
