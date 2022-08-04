package mapper

import (
	"math/big"

	"github.com/coinbase/rosetta-sdk-go/types"
	"github.com/ethereum/go-ethereum/common"
)

func Amount(value *big.Int, currency *types.Currency) *types.Amount {
	if value == nil {
		return nil
	}

	return &types.Amount{
		Value:    value.String(),
		Currency: currency,
	}
}

func AvaxAmount(value *big.Int) *types.Amount {
	return Amount(value, AvaxCurrency)
}

func AtomicAvaxAmount(value *big.Int) *types.Amount {
	return Amount(value, AtomicAvaxCurrency)
}

func Erc20Amount(
	bytes []byte,
	currency *types.Currency,
	sender bool,
) *types.Amount {
	value := common.BytesToHash(bytes).Big()

	if sender {
		value = new(big.Int).Neg(value)
	}

	return Amount(value, currency)
}
