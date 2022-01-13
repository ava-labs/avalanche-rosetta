package client

import (
	"context"

	"github.com/ava-labs/avalanchego/api/info"
	"github.com/ava-labs/avalanchego/network"
)

type InfoClient struct {
	info.Client
}

func NewInfoClient(endpoint string) (*InfoClient, error) {
	c := info.NewClient(endpoint, apiTimeout)
	return &InfoClient{c}, nil
}

func (i *InfoClient) IsBootstrapped(_ context.Context, chain string) (bool, error) {
	return i.Client.IsBootstrapped(chain)
}

func (i *InfoClient) NetworkName(_ context.Context) (string, error) {
	return i.Client.GetNetworkName()
}

func (i *InfoClient) Peers(_ context.Context) ([]network.PeerInfo, error) {
	return i.Client.Peers()
}
