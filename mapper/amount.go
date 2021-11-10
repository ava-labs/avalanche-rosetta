package mapper

import (
	"math/big"
	"strconv"

	"github.com/coinbase/rosetta-sdk-go/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
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

func Erc721Amount(
	indexHash common.Hash,
	contractAddress common.Address,
	contractSymbol string,
	contractDecimal uint8,
	isSender bool) *types.Amount {
	index := indexHash.Big()
	metadata := make(map[string]interface{})
	metadata["tokenType"] = "ERC721"
	metadata["contractAddress"] = contractAddress.String()
	metadata["indexTransfered"] = index.String()

	return &types.Amount{
		Currency: &types.Currency{
			Symbol:   contractSymbol,
			Decimals: int32(contractDecimal),
			Metadata: metadata,
		},
	}
}

func Erc20Amount(
	data []byte,
	contractAddress common.Address,
	contractSymbol string,
	contractDecimal uint8,
	isSender bool) *types.Amount {
	value := common.Bytes2Hex(data)
	decimalValue := hexutil.MustDecodeBig(value)

	if isSender {
		decimalValue = new(big.Int).Neg(decimalValue)
	}
	metadata := make(map[string]interface{})
	metadata["tokenType"] = "ERC20"
	metadata["contractAddress"] = contractAddress.String()

	return &types.Amount{
		Value: decimalValue.String(),
		Currency: &types.Currency{
			Symbol:   contractSymbol,
			Decimals: int32(contractDecimal),
			Metadata: metadata,
		},
	}
}
