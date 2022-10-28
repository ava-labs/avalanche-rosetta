package cchain

import (
	"math/big"

	cconstants "github.com/ava-labs/avalanche-rosetta/constants/cchain"
	"github.com/ava-labs/avalanche-rosetta/mapper"
	"github.com/coinbase/rosetta-sdk-go/types"
	"github.com/ethereum/go-ethereum/common"
)

func AvaxAmount(value *big.Int) *types.Amount {
	return mapper.Amount(value, cconstants.AvaxCurrency)
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

	return mapper.Amount(value, currency)
}
