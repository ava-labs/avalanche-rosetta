package mapper

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/avalanchego/utils/constants"
	"github.com/ava-labs/avalanchego/utils/formatting"
	"github.com/ava-labs/avalanchego/vms/components/avax"
	"github.com/coinbase/rosetta-sdk-go/types"

	rosConst "github.com/ava-labs/avalanche-rosetta/constants"
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
	switch strings.ToLower(networkIdentifier.Network) {
	case rosConst.FujiNetwork:
		hrp = constants.GetHRP(constants.FujiID)
	case rosConst.MainnetNetwork:
		hrp = constants.GetHRP(constants.MainnetID)
	default:
		return "", errUnrecognizedNetwork
	}

	return hrp, nil
}

// UnmarshalJSONMap converts map[string]interface{} into a interface{}.
func UnmarshalJSONMap(m map[string]interface{}, i interface{}) error {
	b, err := json.Marshal(m)
	if err != nil {
		return err
	}

	return json.Unmarshal(b, i)
}

// MarshalJSONMap converts an interface into a map[string]interface{}.
func MarshalJSONMap(i interface{}) (map[string]interface{}, error) {
	b, err := json.Marshal(i)
	if err != nil {
		return nil, err
	}

	var m map[string]interface{}
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, err
	}

	return m, nil
}

// DecodeUTXOID decodes given string into avax.UTXOID
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

// EncodeBytes encodes given bytes to string
func EncodeBytes(bytes []byte) (string, error) {
	return formatting.Encode(formatting.Hex, bytes)
}

// DecodeToBytes decodes given string into bytes using the same encoding as EncodeBytes
func DecodeToBytes(binaryData string) ([]byte, error) {
	return formatting.Decode(formatting.Hex, binaryData)
}
