package cchain

import (
	"math/big"

	"github.com/ava-labs/avalanche-rosetta/constants"
	ethtypes "github.com/ava-labs/coreth/core/types"
	"github.com/coinbase/rosetta-sdk-go/types"
)

// Config holds the service configuration
type Config struct {
	Mode               constants.NodeMode
	ChainID            *big.Int
	NetworkID          *types.NetworkIdentifier
	GenesisBlockHash   string
	AvaxAssetID        string
	IngestionMode      constants.NodeIngestion
	TokenWhiteList     []string
	IndexUnknownTokens bool

	// Upgrade Times
	AP5Activation uint64
}

// IsOfflineMode returns true if running in offline mode
func (c Config) IsOfflineMode() bool {
	return c.Mode == constants.Offline
}

// IsOnlineMode returns true if running in online mode
func (c Config) IsOnlineMode() bool {
	return c.Mode == constants.Online
}

// IsAnalyticsMode returns true if running in analytics ingestion mode
func (c Config) IsAnalyticsMode() bool {
	return c.IngestionMode == constants.AnalyticsIngestion
}

// IsStandardMode returns true if running in standard ingestion mode
func (c Config) IsStandardMode() bool {
	return c.IngestionMode == constants.StandardIngestion
}

// IsTokenListEmpty returns true if the token addresses list is empty
func (c Config) IsTokenListEmpty() bool {
	return len(c.TokenWhiteList) == 0
}

// Signer returns an eth signer object for a given chain
func (c Config) Signer() ethtypes.Signer {
	return ethtypes.LatestSignerForChainID(c.ChainID)
}
