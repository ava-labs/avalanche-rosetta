package cchain

import (
	"github.com/ava-labs/avalanche-rosetta/client"
	"github.com/coinbase/rosetta-sdk-go/types"
)

// TODO ABENEGIA: commented out to avoid import cycle. To be readily solved
// var (
// 	_ service.ConstructionBackend = &Backend{}
// 	_ service.NetworkBackend      = &Backend{}
// 	_ service.AccountBackend      = &Backend{}
// 	_ service.BlockBackend        = &Backend{}
// )

func NewBackend(cfg *Config, client client.Client) *Backend {
	return &Backend{
		config:       cfg,
		client:       client,
		genesisBlock: makeGenesisBlock(cfg.GenesisBlockHash),
	}
}

type Backend struct {
	config *Config
	client client.Client

	genesisBlock *types.Block
}
