package mapper

import (
	"errors"
	"strings"

	"github.com/ava-labs/avalanchego/utils/constants"
	"github.com/coinbase/rosetta-sdk-go/types"
)

var errUnrecognizedNetwork = errors.New("can't recognize network")

// EqualFoldContains checks if the array contains the string regardless of casing
func EqualFoldContains(arr []string, str string) bool {
	for _, a := range arr {
		if strings.EqualFold(a, str) {
			return true
		}
	}
	return false
}

// GetHRP fetches hrp for address formatting.
func GetHRP(networkIdentifier *types.NetworkIdentifier) (string, error) {
	var hrp string
	switch networkIdentifier.Network {
	case FujiNetwork:
		hrp = constants.GetHRP(constants.FujiID)
	case MainnetNetwork:
		hrp = constants.GetHRP(constants.MainnetID)
	default:
		return "", errUnrecognizedNetwork
	}

	return hrp, nil
}
