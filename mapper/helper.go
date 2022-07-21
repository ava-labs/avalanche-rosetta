package mapper

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/avalanchego/utils/constants"
	"github.com/ava-labs/avalanchego/vms/components/avax"
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

// Parse string into avax.UTXOID
func DecodeUTXOID(s string) (*avax.UTXOID, error) {
	split := strings.Split(s, ":")
	if len(split) != 2 {
		return nil, fmt.Errorf("invalid utxo ID format")
	}

	txID, err := ids.FromString(split[0])
	if err != nil {
		return nil, fmt.Errorf("invalid tx ID: %w", err)
	}

	outputIdx, err := strconv.ParseUint(split[1], 10, 32)
	if err != nil {
		return nil, fmt.Errorf("invalid output index: %w", err)
	}

	return &avax.UTXOID{
		TxID:        txID,
		OutputIndex: uint32(outputIdx),
	}, nil
}
