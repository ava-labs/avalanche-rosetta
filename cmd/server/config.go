package main

import (
	"encoding/json"
	"errors"
	"os"

	ethcommon "github.com/ethereum/go-ethereum/common"

	"github.com/ava-labs/avalanche-rosetta/client"
	"github.com/ava-labs/avalanche-rosetta/mapper"
	"github.com/ava-labs/avalanche-rosetta/service"
	"github.com/ava-labs/avalanchego/utils/constants"
)

var (
	errInvalidMode             = errors.New("invalid rosetta mode")
	errGenesisBlockRequired    = errors.New("genesis block hash is not provided")
	errInvalidTokenAddress     = errors.New("invalid token address provided")
	errInvalidErc20Address     = errors.New("not all token addresses provided are valid erc20s")
	errInvalidIngestionMode    = errors.New("invalid rosetta ingestion mode")
	errInvalidUnknownTokenMode = errors.New("cannot index unknown tokens while in standard ingestion mode")
)

type config struct {
	Mode             string `json:"mode"`
	RPCBaseURL       string `json:"rpc_base_url"`
	IndexerBaseURL   string `json:"indexer_base_url"`
	ListenAddr       string `json:"listen_addr"`
	NetworkName      string `json:"network_name"`
	ChainID          int64  `json:"chain_id"`
	LogRequests      bool   `json:"log_requests"`
	GenesisBlockHash string `json:"genesis_block_hash"`

	IngestionMode          string   `json:"ingestion_mode"`
	TokenWhiteList         []string `json:"token_whitelist"`
	BridgeTokenList        []string `json:"bridge_tokens"`
	IndexUnknownTokens     bool     `json:"index_unknown_tokens"`
	ValidateERC20Whitelist bool     `json:"validate_erc20_whitelist"`
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

func (c *config) applyDefaults() {
	if c.Mode == "" {
		c.Mode = service.ModeOnline
	}

	if c.IngestionMode == "" {
		c.IngestionMode = service.StandardIngestion
	}

	if c.RPCBaseURL == "" {
		c.RPCBaseURL = "http://localhost:9650"
	}

	if c.IndexerBaseURL == "" {
		c.IndexerBaseURL = c.RPCBaseURL
	}

	if c.ListenAddr == "" {
		c.ListenAddr = "0.0.0.0:8080"
	}
}

func (c *config) validate() error {
	if !(c.Mode == service.ModeOffline || c.Mode == service.ModeOnline) {
		return errInvalidMode
	}

	if c.Mode == service.ModeOffline && c.ChainID == 0 {
		return errors.New("chainID must be configured when offline mode is selected")
	}

	if c.NetworkName == "" {
		return errors.New("network name not provided")
	}

	if _, err := constants.NetworkID(c.NetworkName); err != nil {
		return errors.New("network name not mapping to any known network ID")
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

	if len(c.BridgeTokenList) != 0 {
		for _, token := range c.BridgeTokenList {
			if !ethcommon.IsHexAddress(token) {
				return errInvalidTokenAddress
			}

			// include all bridge tokens within list of tokens whitelisted for indexing
			if !mapper.EqualFoldContains(c.TokenWhiteList, token) {
				c.TokenWhiteList = append(c.TokenWhiteList, c.BridgeTokenList...)
			}
		}
	}

	if !(c.IngestionMode == service.AnalyticsIngestion || c.IngestionMode == service.StandardIngestion) {
		return errInvalidIngestionMode
	}

	if c.IngestionMode == service.StandardIngestion && c.IndexUnknownTokens {
		return errInvalidUnknownTokenMode
	}
	return nil
}

func (c *config) validateWhitelistOnlyValidErc20s(cli client.Client) error {
	for _, token := range c.TokenWhiteList {
		ethAddress := ethcommon.HexToAddress(token)
		symbol, decimals, err := cli.GetContractInfo(ethAddress, true)
		if err != nil {
			return err
		}
		if decimals == 0 && symbol == client.UnknownERC20Symbol {
			return errInvalidErc20Address
		}
	}
	return nil
}

func (c *config) avalancheNetworkID() uint32 {
	// error checked in config.validate
	res, _ := constants.NetworkID(c.NetworkName)
	return res
}
