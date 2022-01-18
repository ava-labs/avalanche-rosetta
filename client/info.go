package client

import (
	"context"
	"fmt"
	"strings"
)

const (
	methodGetBlockchainID = "info.getBlockchainID"
	methodGetNetworkID    = "info.getNetworkID"
	methodGetNetworkName  = "info.getNetworkName"
	methodGetNodeID       = "info.getNodeID"
	methodGetNodeVersion  = "info.getNodeVersion"
	methodGetPeers        = "info.peers"
	methodIsBootstrapped  = "info.isBootstrapped"
)

// InfoClient is a client for the Info API
type InfoClient struct {
	rpc *RPC
}

// NewInfoClient returns a new client to Info API
func NewInfoClient(endpoint string, token string) (*InfoClient, error) {
	endpoint = strings.TrimSuffix(endpoint, "/")
	endpointURL := fmt.Sprintf("%s%s", endpoint, prefixEth)
	if token != "" {
		endpointURL = fmt.Sprintf("%s%s%s%s", endpoint, prefixInfo, "?token=", token)
	}

	c := Dial(endpointURL)
	return &InfoClient{rpc: c}, nil
}

// IsBootstrapped returns true if a chans is bootstrapped
func (c *InfoClient) IsBootstrapped(ctx context.Context, chain string) (bool, error) {
	args := map[string]string{"chain": chain}
	resp := map[string]bool{}
	err := c.rpc.Call(ctx, methodIsBootstrapped, args, &resp)
	return resp["isBootstrapped"], err
}

// BlockchainID returns the current blockchain identifier
func (c *InfoClient) BlockchainID(ctx context.Context, alias string) (string, error) {
	data, err := c.rpc.CallRaw(ctx, methodGetBlockchainID, map[string]string{"alias": alias})
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// NetworkID returns the current network identifier
func (c *InfoClient) NetworkID(ctx context.Context) (string, error) {
	data, err := c.rpc.CallRaw(ctx, methodGetNetworkID, nil)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// NetworkName returns the current network name
func (c *InfoClient) NetworkName(ctx context.Context) (string, error) {
	resp := map[string]string{}
	err := c.rpc.Call(ctx, methodGetNetworkName, nil, &resp)
	return strings.Title(resp["networkName"]), err
}

// NodeID return the current node identifier
func (c *InfoClient) NodeID(ctx context.Context) (string, error) {
	data, err := c.rpc.CallRaw(ctx, methodGetNodeID, nil)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// NodeVersion returns the current node version
func (c *InfoClient) NodeVersion(ctx context.Context) (string, error) {
	resp := map[string]string{}
	if err := c.rpc.Call(ctx, methodGetNodeVersion, nil, &resp); err != nil {
		return "", err
	}
	return resp["version"], nil
}

// Peers returns the list of active peers
func (c *InfoClient) Peers(ctx context.Context) ([]Peer, error) {
	resp := infoPeersResponse{}
	if err := c.rpc.Call(ctx, methodGetPeers, nil, &resp); err != nil {
		return nil, err
	}
	return resp.Peers, nil
}
