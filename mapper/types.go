package mapper

import (
	"github.com/coinbase/rosetta-sdk-go/types"
)

const (
	MainnetChainID = 43114
	MainnetAssetID = "FvwEAhmxKfeiG8SnEvq42hc6whRyY3EFYAvebMqDNDGCgxN5Z"

	FujiChainID = 43113
	FujiAssetID = "U8iRqJoiJm8xZHAacmvYyZVwqQx6uDNtQeP3CQ6fcgQk3JqnK"

	OpCall         = "CALL"
	OpFee          = "FEE"
	OpCreate       = "CREATE"
	OpCreate2      = "CREATE2"
	OpSelfDestruct = "SELFDESTRUCT"
	OpCallCode     = "CALLCODE"
	OpDelegateCall = "DELEGATECALL"
	OpStaticCall   = "STATICCALL"
	OpDestruct     = "DESTRUCT"
	OpImport       = "IMPORT"
	OpExport       = "EXPORT"

	StatusSuccess = "SUCCESS"
	StatusFailure = "FAILURE"
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
		Decimals: 18, //nolint:gomnd
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

	OperationTypes = []string{
		OpFee,
		OpCall,
		OpCreate,
		OpCreate2,
		OpSelfDestruct,
		OpCallCode,
		OpDelegateCall,
		OpStaticCall,
		OpDestruct,
		OpImport,
		OpExport,
	}

	CallMethods = []string{
		"eth_getTransactionReceipt",
	}
)

func CallType(t string) bool {
	callTypes := []string{
		OpCall,
		OpCallCode,
		OpDelegateCall,
		OpStaticCall,
	}

	for _, callType := range callTypes {
		if callType == t {
			return true
		}
	}

	return false
}

func CreateType(t string) bool {
	createTypes := []string{
		OpCreate,
		OpCreate2,
	}

	for _, createType := range createTypes {
		if createType == t {
			return true
		}
	}

	return false
}
