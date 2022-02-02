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

	currency := Erc20Currency(symbol, decimals, addr.String())
	return &types.Amount{
		Value:    value.String(),
		Currency: currency,
	}
}

func Erc20Currency(symbol string, decimals int32, contractAddress string) *types.Currency {
	return &types.Currency{
		Symbol:   symbol,
		Decimals: decimals,
		Metadata: map[string]interface{}{
			ContractAddressMetadata: contractAddress,
		},
	}
}
