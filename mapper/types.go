package mapper

import (
	"github.com/coinbase/rosetta-sdk-go/types"
)

const (
	OpFee            = "fee"
	OpTransfer       = "transfer"
	OpTokenTransfer  = "token_transfer"
	OpContractCreate = "contract_create"
	OpContractCall   = "contract_call"

	TxStatusSuccess = "success"
	TxStatusFailure = "failure"
)

var (
	AvaxCurrency = &types.Currency{
		Symbol:   "AVAX",
		Decimals: 18,
	}

	OperationStatuses = []*types.OperationStatus{
		{
			Status:     TxStatusSuccess,
			Successful: true,
		},
		{
			Status:     TxStatusFailure,
			Successful: false,
		},
	}

	OperationTypes = []string{
		OpFee,
		OpTransfer,
		OpTokenTransfer,
		OpContractCreate,
		OpContractCall,
	}
)
