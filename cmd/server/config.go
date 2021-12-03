package main

import (
	"encoding/json"
	"errors"
	"os"

	"github.com/ava-labs/avalanche-rosetta/service"
)

var (
	errMissingRPC           = errors.New("avalanche rpc endpoint is not provided")
	errInvalidMode          = errors.New("invalid rosetta mode")
	errGenesisBlockRequired = errors.New("genesis block hash is not provided")
)

type config struct {
	Mode               string `json:"mode"`
	RPCEndpoint        string `json:"rpc_endpoint"`
	ListenAddr         string `json:"listen_addr"`
	NetworkName        string `json:"network_name"`
	ChainID            int64  `json:"chain_id"`
	LogRequests        bool   `json:"log_requests"`
	GenesisBlockHash   string `json:"genesis_block_hash"`
	EnableErc20        bool   `json:"index_erc20"`
	EnableErc721       bool   `json:"enable_erc721"`
	IndexDefaultTokens bool   `json:"index_default_tokens"`
}

func readConfig(path string) (*config, error) {
	cfg := &config{}

	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	err = json.NewDecoder(f).Decode(cfg)
	return cfg, err
}

func (c *config) ApplyDefaults() {
	if c.Mode == "" {
		c.Mode = "online"
	}

	if c.RPCEndpoint == "" {
		c.RPCEndpoint = "http://localhost:9650"
	}

	if c.ListenAddr == "" {
		c.ListenAddr = "0.0.0.0:8080"
	}
}

func (c *config) Validate() error {
	c.ApplyDefaults()

	if c.RPCEndpoint == "" {
		return errMissingRPC
	}

	if !(c.Mode == service.ModeOffline || c.Mode == service.ModeOnline) {
		return errInvalidMode
	}

	if c.GenesisBlockHash == "" {
		return errGenesisBlockRequired
	}

	return nil
}
