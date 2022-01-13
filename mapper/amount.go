package mapper

import (
	"math/big"
	"strconv"

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

func FeeAmount(value int64) *types.Amount {
	return &types.Amount{
		Value:    strconv.FormatInt(value, 10), //nolint:gomnd
		Currency: AvaxCurrency,
	}
}

func AvaxAmount(value *big.Int) *types.Amount {
	return Amount(value, AvaxCurrency)
}

func Erc20Amount(
	data []byte,
	addr common.Address,
	symbol string,
	decimals uint8,
	sender bool) *types.Amount {
	value := common.BytesToHash(data).Big()

	if sender {
		value = new(big.Int).Neg(value)
	}

	return &types.Amount{
		Value: value.String(),
		Currency: &types.Currency{
			Symbol:   symbol,
			Decimals: int32(decimals),
			Metadata: map[string]interface{}{
				ContractAddressMetadata: addr.String(),
			},
		},
	}
}
