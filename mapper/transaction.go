package mapper

import (
	"math/big"
	"strings"

	"github.com/coinbase/rosetta-sdk-go/types"
	ethcommon "github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"

	"github.com/figment-networks/avalanche-rosetta/client"
)

func Transaction(header *ethtypes.Header, tx *ethtypes.Transaction, msg *ethtypes.Message, receipt *ethtypes.Receipt) (*types.Transaction, error) {
	var (
		status    string
		sender    *ethcommon.Address
		recipient *ethcommon.Address
	)

	if receipt.Status == ethtypes.ReceiptStatusSuccessful {
		status = TxStatusSuccess
	} else {
		status = TxStatusFailure
	}

	senderAddr := msg.From()
	sender = &senderAddr
	recipient = msg.To()

	txFee := new(big.Int).SetUint64(receipt.GasUsed)
	txFee = txFee.Mul(txFee, msg.GasPrice())

	ops := []*types.Operation{
		// Sender of the tx amount
		{
			OperationIdentifier: &types.OperationIdentifier{
				Index: 0,
			},
			Type:    OpTransfer,
			Status:  &status,
			Account: Account(sender),
			Amount:  Amount(new(big.Int).Neg(tx.Value()), AvaxCurrency),
		},
		// Recipient of the tx amount
		{
			OperationIdentifier: &types.OperationIdentifier{
				Index: 1,
			},
			Type:    OpTransfer,
			Status:  &status,
			Account: Account(recipient),
			Amount:  Amount(tx.Value(), AvaxCurrency),
		},
		// Fees receiver
		{
			OperationIdentifier: &types.OperationIdentifier{
				Index: 2,
			},
			Type:    OpFee,
			Status:  &status,
			Account: Account(&header.Coinbase),
			Amount:  Amount(txFee, AvaxCurrency),
		},
	}

	return &types.Transaction{
		TransactionIdentifier: &types.TransactionIdentifier{
			Hash: tx.Hash().String(),
		},
		Operations: ops,
		Metadata: map[string]interface{}{
			"gas":       tx.Gas(),
			"gas_price": tx.GasPrice().String(),
		},
	}, nil
}

func MempoolTransactionsIDs(accountMap client.TxAccountMap) []*types.TransactionIdentifier {
	result := []*types.TransactionIdentifier{}

	for _, txNonceMap := range accountMap {
		for _, tx := range txNonceMap {
			// todo: this should be a parsed out struct from the eth client
			chunks := strings.Split(tx, ":")

			result = append(result, &types.TransactionIdentifier{
				Hash: chunks[0],
			})
		}
	}

	return result
}
