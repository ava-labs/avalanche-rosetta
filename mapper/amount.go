package mapper

import (
	"math/big"

	"github.com/coinbase/rosetta-sdk-go/types"
)

func Amount(value *big.Int, currency *types.Currency) *types.Amount {
	if value == nil {
		return nil
	}
	return &types.Amount{
		Value:    value.String(),
		Currency: AvaxCurrency,
	}
}

func AvaxAmount(value *big.Int) *types.Amount {
	return Amount(value, AvaxCurrency)
}
