package main

import (
	"encoding/json"
	"errors"
	"os"

	"github.com/ava-labs/avalanche-rosetta/client"
	"github.com/ava-labs/avalanche-rosetta/service"
	ethcommon "github.com/ethereum/go-ethereum/common"
)

var (
	errMissingRPC           = errors.New("avalanche rpc endpoint is not provided")
	errInvalidMode          = errors.New("invalid rosetta mode")
	errGenesisBlockRequired = errors.New("genesis block hash is not provided")
	errInvalidTokenAddress  = errors.New("invalid token address provided")
	errInvalidErc20Address  = errors.New("not all token addresses provided are valid erc20s")
	errInvalidIngestionMode = errors.New("invalid rosetta ingestion mode")
)

type config struct {
	Mode             string `json:"mode"`
	RPCEndpoint      string `json:"rpc_endpoint"`
	ListenAddr       string `json:"listen_addr"`
	NetworkName      string `json:"network_name"`
	ChainID          int64  `json:"chain_id"`
	LogRequests      bool   `json:"log_requests"`
	GenesisBlockHash string `json:"genesis_block_hash"`

	IngestionMode      string   `json:"ingestion_mode"`
	TokenWhiteList     []string `json:"token_whitelist"`
	IndexUnknownTokens bool     `json:"index_unknown_tokens"`
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
		c.Mode = service.ModeOnline
	}

	if c.IngestionMode == "" {
		c.IngestionMode = service.StandardIngestion
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

	if len(c.TokenWhiteList) != 0 {
		for _, token := range c.TokenWhiteList {
			if !ethcommon.IsHexAddress(token) {
				return errInvalidTokenAddress
			}
		}
	}

	if !(c.IngestionMode == service.AnalyticsIngestion || c.IngestionMode == service.StandardIngestion) {
		return errInvalidIngestionMode
	}
	return nil
}

func (c *config) ValidateWhitelistOnlyValidErc20s(cli client.Client) error {
	for _, token := range c.TokenWhiteList {
		ethAddress := ethcommon.HexToAddress(token)
		currency, err := cli.ContractCurrency(ethAddress, true)
		if err != nil {
			return err
		}
		if currency.Decimals == client.UnknownERC20Decimals && currency.Symbol == client.UnknownERC20Symbol {
			return errInvalidErc20Address
		}
	}
	return nil
}
