package service

import (
	"math/big"

	"github.com/ava-labs/avalanchego/graft/coreth/params"
	"github.com/coinbase/rosetta-sdk-go/types"

	ethtypes "github.com/ava-labs/libevm/core/types"
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
		if c.ChainID.Cmp(avalancheMainnetChainID) == 0 {
			return ethtypes.LatestSigner(GetChainConfig(avalancheMainnetChainID))
		}
		if c.ChainID.Cmp(avalancheFujiChainID) == 0 {
			return ethtypes.LatestSigner(GetChainConfig(avalancheFujiChainID))
		}
		if c.ChainID.Cmp(avalancheLocalChainID) == 0 {
			return ethtypes.LatestSigner(GetChainConfig(avalancheLocalChainID))
		}
	}
	return ethtypes.LatestSignerForChainID(c.ChainID)
}

// C-Chain EVM chain IDs.
var (
	avalancheMainnetChainID = big.NewInt(43114)
	avalancheFujiChainID    = big.NewInt(43113)
	avalancheLocalChainID   = big.NewInt(43112)
)

// GetChainConfig returns the EVM chain configuration used to select the transaction signer for
// `chainID`.
func GetChainConfig(chainID *big.Int) *params.ChainConfig {
	c := &params.ChainConfig{
		ChainID:             chainID,
		HomesteadBlock:      big.NewInt(0),
		DAOForkBlock:        big.NewInt(0),
		DAOForkSupport:      true,
		EIP150Block:         big.NewInt(0),
		EIP155Block:         big.NewInt(0),
		EIP158Block:         big.NewInt(0),
		ByzantiumBlock:      big.NewInt(0),
		ConstantinopleBlock: big.NewInt(0),
		PetersburgBlock:     big.NewInt(0),
		IstanbulBlock:       big.NewInt(0),
		MuirGlacierBlock:    big.NewInt(0),
	}

	// Berlin and London activated on the live networks at the blocks of the Apricot Phase 2 and 3
	// upgrades, not at genesis.
	switch {
	case chainID.Cmp(avalancheFujiChainID) == 0:
		c.BerlinBlock = big.NewInt(184985) // https://testnet.snowtrace.io/block/184985?chainid=43113
		c.LondonBlock = big.NewInt(805078) // https://testnet.snowtrace.io/block/805078?chainid=43113
	case chainID.Cmp(avalancheMainnetChainID) == 0:
		c.BerlinBlock = big.NewInt(1640340) // https://snowtrace.io/block/1640340?chainid=43114
		c.LondonBlock = big.NewInt(3308552) // https://snowtrace.io/block/3308552?chainid=43114
	}
	return c
}
