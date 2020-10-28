package mapper

import (
	"github.com/coinbase/rosetta-sdk-go/types"
)

const (
	OpFee            = "FEE"
	OpTransfer       = "TRANSFER"
	OpTokenTransfer  = "TOKEN_TRANSFER"
	OpContractCreate = "CONTRACT_CREATE"
	OpContractCall   = "CONTRACT_CALL"

	TxStatusSuccess = "SUCCESS"
	TxStatusFailure = "FAILURE"
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
