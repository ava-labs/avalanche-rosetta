package mapper

import (
	"math/big"

	"github.com/coinbase/rosetta-sdk-go/types"
)

const MetadataExportedOutputs = "exported_outputs"

func Amount(value *big.Int, currency *types.Currency) *types.Amount {
	if value == nil {
		return nil
	}

	return &types.Amount{
		Value:    value.String(),
		Currency: currency,
	}
}
