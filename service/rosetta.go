package service

import (
	"math/big"

	ethtypes "github.com/ethereum/go-ethereum/core/types"
)

const (
	RosettaVersion    = "1.4.4"
	MiddlewareVersion = "0.1.0"
	BlockchainName    = "avalanche"
)

var (
	signer ethtypes.EIP155Signer
)

func SetChainID(val *big.Int) {
	if val == nil {
		panic("chain id value cant be nil")
	}
	signer = ethtypes.NewEIP155Signer(val)
}
