package client

import (
	"fmt"
)

// InfoClient implements Info API client spec
// https://docs.avax.network/v1.0/en/api/info
type InfoClient struct {
	rpc RPC
}

type infoPeersResponse struct {
	Peers []Peer `json:"peers"`
}

func NewInfoClient(endpoint string) *InfoClient {
	return &InfoClient{
		rpc: NewRPCClient(fmt.Sprintf("%s%s", endpoint, InfoPrefix)),
	}
}

func (c InfoClient) BlockchainID(alias string) (string, error) {
	data, err := c.rpc.CallRaw("info.getBlockchainID", map[string]string{"alias": alias})
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func (c InfoClient) NetworkID() (string, error) {
	data, err := c.rpc.CallRaw("info.getNetworkID", nil)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func (c InfoClient) NetworkName() (string, error) {
	resp := map[string]string{}
	err := c.rpc.Call("info.getNetworkName", nil, &resp)
	return resp["networkName"], err
}

func (c InfoClient) NodeID() (string, error) {
	data, err := c.rpc.CallRaw("info.getNodeID", nil)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func (c InfoClient) NodeVersion() (string, error) {
	resp := map[string]string{}
	if err := c.rpc.Call("info.getNodeVersion", nil, &resp); err != nil {
		return "", err
	}
	return resp["version"], nil
}

func (c InfoClient) Peers() ([]Peer, error) {
	resp := infoPeersResponse{}
	if err := c.rpc.Call("info.peers", nil, &resp); err != nil {
		return nil, err
	}
	return resp.Peers, nil
}
