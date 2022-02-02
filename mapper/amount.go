package mapper

import (
	"math/big"

	"github.com/ava-labs/avalanche-rosetta/client"
	"github.com/coinbase/rosetta-sdk-go/types"
	"github.com/ethereum/go-ethereum/common"
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

func Erc20Amount(
	bytes []byte,
	addr common.Address,
	symbol string,
	decimals int32,
	sender bool) *types.Amount {
	value := common.BytesToHash(bytes).Big()

	if sender {
		value = new(big.Int).Neg(value)
	}

	currency := client.ContractCurrency(symbol, decimals, addr.String())
	return &types.Amount{
		Value:    value.String(),
		Currency: currency,
	}
}
