package main

import (
	"encoding/json"
	"errors"
	"os"

	ethcommon "github.com/ethereum/go-ethereum/common"

	"github.com/ava-labs/avalanche-rosetta/client"
	"github.com/ava-labs/avalanche-rosetta/constants"
)

var (
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
	CChainID         int64  `json:"chain_id"`
	LogRequests      bool   `json:"log_requests"`
	GenesisBlockHash string `json:"genesis_block_hash"`

	IngestionMode          string   `json:"ingestion_mode"`
	TokenWhiteList         []string `json:"token_whitelist"`
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
		c.Mode = constants.Online.String()
	}

	if c.IngestionMode == "" {
		c.IngestionMode = constants.StandardIngestion.String()
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
	if _, err := constants.GetNodeMode(c.Mode); err != nil {
		return err
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

	if _, err := constants.GetNodeIngestion(c.Mode); err != nil {
		return err
	}

	if c.IngestionMode == constants.StandardIngestion.String() && c.IndexUnknownTokens {
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
