package mapper

import (
	"github.com/ethereum/go-ethereum/common"

	"github.com/coinbase/rosetta-sdk-go/types"
)

const (
	ContractAddressMetadata  = "contractAddress"
	IndexTransferredMetadata = "indexTransferred"

	MetadataImportedInputs  = "imported_inputs"
	MetadataExportedOutputs = "exported_outputs"
	MetadataAddressFormat   = "address_format"
	AddressFormatBech32     = "bech32"
)

var (
	AvaxCurrency = &types.Currency{
		Symbol:   "AVAX",
		Decimals: 18,
	}

	AtomicAvaxCurrency = &types.Currency{
		Symbol:   "AVAX",
		Decimals: 9,
	}
)

func ToCurrency(symbol string, decimals uint8, contractAddress common.Address) *types.Currency {
	return &types.Currency{
		Symbol:   symbol,
		Decimals: int32(decimals),
		Metadata: map[string]interface{}{
			ContractAddressMetadata: contractAddress.Hex(),
		},
	}
}
