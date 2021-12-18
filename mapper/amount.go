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
	metadata := make(map[string]interface{})
	metadata[ContractAddressMetadata] = contractAddress.String()

	return &types.Amount{
		Value: decimalValue.String(),
		Currency: &types.Currency{
			Symbol:   contractSymbol,
			Decimals: int32(contractDecimal),
			Metadata: metadata,
		},
	}
}
