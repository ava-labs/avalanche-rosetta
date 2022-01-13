package client

import (
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

func (i *InfoClient) IsBootstrapped(chain string) (bool, error) {
	return i.Client.IsBootstrapped(chain)
}

func (i *InfoClient) NetworkName() (string, error) {
	return i.Client.GetNetworkName()
}

func (i *InfoClient) Peers() ([]network.PeerInfo, error) {
	return i.Client.Peers()
}
