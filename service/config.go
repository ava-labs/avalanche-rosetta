package service

import (
	"math/big"

	ethtypes "github.com/ava-labs/coreth/core/types"
	"github.com/ava-labs/coreth/params"
	"github.com/coinbase/rosetta-sdk-go/types"
)

// Config holds the service configuration
type Config struct {
	Mode               string
	ChainID            *big.Int
	NetworkID          *types.NetworkIdentifier
	GenesisBlockHash   string
	AvaxAssetID        string
	IngestionMode      string
	TokenWhiteList     []string
	BridgeTokenList    []string
	IndexUnknownTokens bool

	// Upgrade Times
	AP5Activation uint64
}

const (
	ModeOffline        = "offline"
	ModeOnline         = "online"
	StandardIngestion  = "standard"
	AnalyticsIngestion = "analytics"
)

// IsOfflineMode returns true if running in offline mode
func (c Config) IsOfflineMode() bool {
	return c.Mode == ModeOffline
}

// IsOnlineMode returns true if running in online mode
func (c Config) IsOnlineMode() bool {
	return c.Mode == ModeOnline
}

// IsAnalyticsMode returns true if running in analytics ingestion mode
func (c Config) IsAnalyticsMode() bool {
	return c.IngestionMode == AnalyticsIngestion
}

// IsStandardMode returns true if running in standard ingestion mode
func (c Config) IsStandardMode() bool {
	return c.IngestionMode == StandardIngestion
}

// IsTokenListEmpty returns true if the token addresses list is empty
func (c Config) IsTokenListEmpty() bool {
	return len(c.TokenWhiteList) == 0
}

// Signer returns an eth signer object for a given chain
func (c Config) Signer() ethtypes.Signer {
	if c.ChainID != nil {
		if c.ChainID.Cmp(params.AvalancheMainnetChainID) == 0 {
			return ethtypes.LatestSigner(params.AvalancheMainnetChainConfig)
		}
		if c.ChainID.Cmp(params.AvalancheFujiChainID) == 0 {
			return ethtypes.LatestSigner(params.AvalancheFujiChainConfig)
		}
		if c.ChainID.Cmp(params.AvalancheLocalChainID) == 0 {
			return ethtypes.LatestSigner(params.AvalancheLocalChainConfig)
		}
	}
	return ethtypes.LatestSignerForChainID(c.ChainID)
}
