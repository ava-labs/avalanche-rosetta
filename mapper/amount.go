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
	contractAddress common.Address,
	contractSymbol string,
	contractDecimal uint8,
	isSender bool) *types.Amount {
	value := common.BytesToHash(data)
	decimalValue := value.Big()

	if isSender {
		decimalValue = new(big.Int).Neg(decimalValue)
	}

	currency := Erc20Currency(contractSymbol, int32(contractDecimal), contractAddress.String())
	return &types.Amount{
		Value:    decimalValue.String(),
		Currency: currency,
	}
}

func Erc20Currency(symbol string, decimals int32, contractAddress string) *types.Currency {
	metadata := make(map[string]interface{})
	metadata[TokenTypeMetadata] = "ERC20"
	metadata[ContractAddressMetadata] = contractAddress

	return &types.Currency{
		Symbol:   symbol,
		Decimals: decimals,
		Metadata: metadata,
	}
}
