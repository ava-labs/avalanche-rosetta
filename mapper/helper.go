package mapper

import (
	"errors"
	"strings"

	"github.com/ava-labs/avalanchego/utils/constants"
	"github.com/coinbase/rosetta-sdk-go/types"
)

var errUnsupportedChain = errors.New("unsupported chain")
var errUnsupportedNetwork = errors.New("unsupported network")

// EqualFoldContains checks if the array contains the string regardless of casing
func EqualFoldContains(arr []string, str string) bool {
	for _, a := range arr {
		if strings.EqualFold(a, str) {
			return true
		}
	}
	return false
}

// IsPChain checks network identifier to make sure sub-network identifier set to "P"
func IsPChain(networkIdentifier *types.NetworkIdentifier) bool {
	return networkIdentifier != nil &&
		networkIdentifier.SubNetworkIdentifier != nil &&
		networkIdentifier.SubNetworkIdentifier.Network == PChainNetworkIdentifier
}

// GetAliasAndHRP fetches chain id alias and hrp for address formatting.
// Right now only P chain id alias is supported
func GetAliasAndHRP(networkIdentifier *types.NetworkIdentifier) (string, string, error) {
	if !IsPChain(networkIdentifier) {
		return "", "", errUnsupportedChain
	}

	var hrp string
	switch networkIdentifier.Network {
	case FujiNetwork:
		hrp = constants.GetHRP(constants.FujiID)
	case MainnetNetwork:
		hrp = constants.GetHRP(constants.MainnetID)
	default:
		return "", "", errUnsupportedNetwork
	}

	return PChainIDAlias, hrp, nil
}
