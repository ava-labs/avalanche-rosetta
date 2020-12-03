package mapper

import (
	"github.com/coinbase/rosetta-sdk-go/types"
)

const (
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
	AvaxCurrency = &types.Currency{
		Symbol:   "AVAX",
		Decimals: 18,
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
