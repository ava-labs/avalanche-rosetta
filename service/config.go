package service

import (
	"math/big"

	ethtypes "github.com/ava-labs/coreth/core/types"
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
	TokenAddresses     []string
	EnableErc20        bool
	EnableErc721       bool
	IndexDefaultTokens bool

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

// TokenListContains returns true if the token addresses list contain the provided address
func (c Config) TokenListContains(contractAddress string) bool {
	return Contains(c.TokenAddresses, contractAddress)
}

// IsTokenListEmpty returns true if the token addresses list is empty
func (c Config) IsTokenListEmpty() bool {
	return len(c.TokenAddresses) == 0
}

// Signer returns an eth signer object for a given chain
func (c Config) Signer() ethtypes.Signer {
	return ethtypes.LatestSignerForChainID(c.ChainID)
}
