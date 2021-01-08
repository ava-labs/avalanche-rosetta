package client

import (
	"context"
	"fmt"
)

const (
	methodAvmAssetDescription = "avm.getAssetDescription"
)

// AvmClient is a client for the AVM API
type AvmClient struct {
	rpc *RPC
}

// NewAvmClient returns a new client to Info API
func NewAvmClient(endpoint string) (*AvmClient, error) {
	c := Dial(fmt.Sprintf("%s%s", endpoint, prefixAvm))
	return &AvmClient{rpc: c}, nil
}

// AssetDescription returns the asset description for a given ID
func (c *AvmClient) AssetDescription(ctx context.Context, assetID string) (*Asset, error) {
	result := &Asset{}

	return result, c.rpc.Call(
		ctx,
		methodAvmAssetDescription,
		map[string]string{"assetID": assetID},
		result,
	)
}
