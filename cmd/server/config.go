package main

import (
	"encoding/json"
	"errors"
	"io/ioutil"
)

type config struct {
	RPCEndpoint string `json:"rpc_endpoint"`
	ListenAddr  string `json:"listen_addr"`
	NetworkName string `json:"network_name"`
	ChainID     int64  `json:"chain_id"`
}

func readConfig(path string) (*config, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	cfg := &config{}
	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

func (c *config) Validate() error {
	if c.RPCEndpoint == "" {
		return errors.New("avalanche rpc endpoint is not provided")
	}
	if c.ListenAddr == "" {
		c.ListenAddr = "0.0.0.0:8080"
	}
	return nil
}
