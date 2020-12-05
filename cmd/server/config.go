package main

import (
	"encoding/json"
	"errors"
	"io/ioutil"
)

type config struct {
	Mode        string `json:"mode"`
	RPCEndpoint string `json:"rpc_endpoint"`
	ListenAddr  string `json:"listen_addr"`
	NetworkName string `json:"network_name"`
	ChainID     int64  `json:"chain_id"`
	LogRequests bool   `json:"log_requests"`
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
	if c.RPCEndpoint == "" {
		return errors.New("avalanche rpc endpoint is not provided")
	}

	c.ApplyDefaults()

	return nil
}
