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
	return &InfoClient{
		info.NewClient(endpoint, apiTimeout),
	}, nil
}

func (i *InfoClient) IsBootstrapped(ctx context.Context, chain string) (bool, error) {
	return i.Client.IsBootstrapped(chain)
}

func (i *InfoClient) NetworkName(ctx context.Context) (string, error) {
	return i.Client.GetNetworkName()
}

func (i *InfoClient) Peers(ctx context.Context) ([]network.PeerInfo, error) {
	return i.Client.Peers()
}
