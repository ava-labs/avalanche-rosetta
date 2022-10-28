package cchain

import (
	"github.com/coinbase/rosetta-sdk-go/types"
	"github.com/ethereum/go-ethereum/common"
)

const (
	ContractAddressMetadata  = "contractAddress"
	indexTransferredMetadata = "indexTransferred"
)

func toCurrency(symbol string, decimals uint8, contractAddress common.Address) *types.Currency {
	return &types.Currency{
		Symbol:   symbol,
		Decimals: int32(decimals),
		Metadata: map[string]interface{}{
			ContractAddressMetadata: contractAddress.Hex(),
		},
	}
}
