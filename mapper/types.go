package mapper

import (
	"github.com/ethereum/go-ethereum/common"

	"github.com/coinbase/rosetta-sdk-go/types"
)

const (
	ContractAddressMetadata  = "contractAddress"
	IndexTransferredMetadata = "indexTransferred"

	StatusSuccess = "SUCCESS"
	StatusFailure = "FAILURE"

	MetadataImportedInputs  = "imported_inputs"
	MetadataExportedOutputs = "exported_outputs"
	MetadataAddressFormat   = "address_format"
	AddressFormatBech32     = "bech32"
)

var (
	StageBootstrap = &types.SyncStatus{
		Synced: types.Bool(false),
		Stage:  types.String("BOOTSTRAP"),
	}

	StageSynced = &types.SyncStatus{
		Synced: types.Bool(true),
		Stage:  types.String("SYNCED"),
	}

	AvaxCurrency = &types.Currency{
		Symbol:   "AVAX",
		Decimals: 18,
	}

	AtomicAvaxCurrency = &types.Currency{
		Symbol:   "AVAX",
		Decimals: 9,
	}

	OperationStatuses = []*types.OperationStatus{
		{
			Status:     StatusSuccess,
			Successful: true,
		},
		{
			Status:     StatusFailure,
			Successful: false,
		},
	}

	CallMethods = []string{
		"eth_getTransactionReceipt",
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
