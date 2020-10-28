package service

import (
	"math/big"

	"github.com/coinbase/rosetta-sdk-go/types"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
)

type Config struct {
	Mode      string
	ChainID   *big.Int
	NetworkID *types.NetworkIdentifier
}

const (
	ModeOffline = "offline"
	ModeOnline  = "online"
)

func (c Config) IsOfflineMode() bool {
	return c.Mode == ModeOffline
}

func (c Config) IsOnlineMode() bool {
	return c.Mode == ModeOnline
}

func (c Config) Signer() ethtypes.EIP155Signer {
	return ethtypes.NewEIP155Signer(c.ChainID)
}
