package mapper

import (
	"errors"
	"strings"

	"github.com/ava-labs/avalanchego/utils/constants"
	"github.com/coinbase/rosetta-sdk-go/types"
)

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
	if networkIdentifier != nil &&
		networkIdentifier.SubNetworkIdentifier != nil &&
		networkIdentifier.SubNetworkIdentifier.Network == PChainNetworkIdentifier {
		return true
	}

	return false
}

// GetAliasAndHRP fetches chain id alias and hrp for address formatting.
// Right now only P chain id alias is supported
func GetAliasAndHRP(networkIdentifier *types.NetworkIdentifier) (string, string, error) {
	var chainIDAlias, hrp string
	if !IsPChain(networkIdentifier) {
		return "", "", errors.New("only support P chain alias")
	}
	chainIDAlias = PChainIDAlias
	switch networkIdentifier.Network {
	case FujiNetwork:
		hrp = constants.GetHRP(constants.FujiID)
	case MainnetNetwork:
		hrp = constants.GetHRP(constants.MainnetID)
	default:
		return "", "", errors.New("can't recognize network")
	}

	return chainIDAlias, hrp, nil
}
