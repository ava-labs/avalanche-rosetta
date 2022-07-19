package pchain

import (
	"github.com/coinbase/rosetta-sdk-go/types"

	"github.com/ava-labs/avalanche-rosetta/mapper"
)

// IsPChain checks network identifier to make sure sub-network identifier set to "P"
func IsPChain(networkIdentifier *types.NetworkIdentifier) bool {
	if networkIdentifier != nil &&
		networkIdentifier.SubNetworkIdentifier != nil &&
		networkIdentifier.SubNetworkIdentifier.Network == mapper.PChainNetworkIdentifier {
		return true
	}

	return false
}
