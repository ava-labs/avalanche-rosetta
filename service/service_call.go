package service

import (
	"context"

	"github.com/coinbase/rosetta-sdk-go/server"
	"github.com/coinbase/rosetta-sdk-go/types"

	"github.com/figment-networks/avalanche-rosetta/client"
)

// ConstructionService implements /call/* endpoints
type CallService struct {
	config *Config
	evm    *client.EvmClient
}

// NewCallService returns a new call servicer
func NewCallService(config *Config, evmClient *client.EvmClient) server.CallAPIServicer {
	return &CallService{
		config: config,
		evm:    evmClient,
	}
}

// Call implements the /call endpoint.
func (s CallService) Call(ctx context.Context, req *types.CallRequest) (*types.CallResponse, *types.Error) {
	if s.config.IsOfflineMode() {
		return nil, errUnavailableOffline
	}
	return nil, errNotImplemented
}
