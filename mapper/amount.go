package mapper

import (
	"math/big"

	cconstants "github.com/ava-labs/avalanche-rosetta/constants/cchain"
	pconstants "github.com/ava-labs/avalanche-rosetta/constants/pchain"
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
	return Amount(value, cconstants.AvaxCurrency)
}

// AtomicAvaxAmount creates a Rosetta Amount representing AVAX amount in nAVAXs with given quantity
func AtomicAvaxAmount(value *big.Int) *types.Amount {
	return Amount(value, pconstants.AtomicAvaxCurrency)
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
