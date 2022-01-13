package mapper

import (
	"fmt"
	"log"
	"math/big"
	"strings"

	ethtypes "github.com/ava-labs/coreth/core/types"
	"github.com/ava-labs/coreth/plugin/evm"
	"github.com/coinbase/rosetta-sdk-go/types"

	"github.com/ava-labs/avalanche-rosetta/client"
	clientTypes "github.com/ava-labs/avalanche-rosetta/client"
)

const (
	topicsInErc721Transfer = 4
	topicsInErc20Transfer  = 3
	zeroAddress            = "0x0000000000000000000000000000000000000000000000000000000000000000"
)

var (
	x2crate = big.NewInt(1000000000) //nolint:gomnd
)

func Transaction(
	header *ethtypes.Header,
	tx *ethtypes.Transaction,
	msg *ethtypes.Message,
	receipt *ethtypes.Receipt,
	trace *client.Call,
	flattenedTrace []*client.FlatCall,
	transferLogs []ethtypes.Log,
	client client.Client,
	isAnalyticsMode bool,
	standardModeWhiteList []string,
	includeUnknownTokens bool,
) (*types.Transaction, error) {
	ops := []*types.Operation{}
	sender := msg.From()
	feeReceiver := &header.Coinbase

	txFee := new(big.Int).SetUint64(receipt.GasUsed)
	txFee = txFee.Mul(txFee, msg.GasPrice())

	feeOps := []*types.Operation{
		{
			OperationIdentifier: &types.OperationIdentifier{
				Index: 0,
			},
			Type:    OpFee,
			Status:  types.String(StatusSuccess),
			Account: Account(&sender),
			Amount:  AvaxAmount(new(big.Int).Neg(txFee)),
		},
		{
			OperationIdentifier: &types.OperationIdentifier{
				Index: 1,
			},
			RelatedOperations: []*types.OperationIdentifier{
				{
					Index: 0,
				},
			},
			Type:    OpFee,
			Status:  types.String(StatusSuccess),
			Account: Account(feeReceiver),
			Amount:  AvaxAmount(txFee),
		},
	}

	ops = append(ops, feeOps...)

	traceOps := traceOps(flattenedTrace, len(feeOps))
	ops = append(ops, traceOps...)
	// Logs will be empty if in standard mode and token whitelist is empty
	for _, transferLog := range transferLogs {
		// If in standard mode, token address must be whitelisted
		if !isAnalyticsMode && !EqualFoldContains(standardModeWhiteList, transferLog.Address.String()) {
			continue
		}

		// ERC721 index the value in the transfer event.  ERC20's do not
		if len(transferLog.Topics) == topicsInErc721Transfer {
			contractInfo, err := client.ContractInfo(transferLog.Address, false)
			if err != nil {
				return nil, err
			}

			// Don't include default tokens if setting is not enabled
			if !includeUnknownTokens && contractInfo.Symbol == clientTypes.UnknownERC721Symbol {
				continue
			}

			erc721txs := parseErc721Txs(transferLog, int64(len(ops)))
			ops = append(ops, erc721txs...)
		} else {
			contractInfo, err := client.ContractInfo(transferLog.Address, true)
			if err != nil {
				return nil, err
			}

			// Don't include default tokens if setting is not enabled
			if (!includeUnknownTokens && contractInfo.Symbol == clientTypes.UnknownERC20Symbol) ||
				(len(transferLog.Topics) != topicsInErc20Transfer) {
				continue
			}

			erc20txs := parseErc20Txs(transferLog, contractInfo, int64(len(ops)))
			ops = append(ops, erc20txs...)
		}
	}
	return &types.Transaction{
		TransactionIdentifier: &types.TransactionIdentifier{
			Hash: tx.Hash().String(),
		},
		Operations: ops,
		Metadata: map[string]interface{}{
			"gas":       tx.Gas(),
			"gas_price": tx.GasPrice().String(),
			"receipt":   receipt,
			"trace":     trace,
			"type":      tx.Type(),
		},
	}, nil
}

func crossChainTransaction(
	rawIdx int,
	avaxAssetID string,
	tx *evm.Tx,
) ([]*types.Operation, error) {
	var (
		ops = []*types.Operation{}
		idx = int64(rawIdx)
	)

	// Prepare transaction for ID calcuation
	if err := tx.Sign(codecManager, nil); err != nil {
		return nil, err
	}

	switch t := tx.UnsignedAtomicTx.(type) {
	case *evm.UnsignedImportTx:
		// Create de-duplicated list of input
		// transaction IDs
		mTxIDs := map[string]struct{}{}
		for _, in := range t.ImportedInputs {
			mTxIDs[in.TxID.String()] = struct{}{}
		}
		i := 0
		txIDs := make([]string, len(mTxIDs))
		for txID := range mTxIDs {
			txIDs[i] = txID
			i++
		}

		for _, out := range t.Outs {
			if out.AssetID.String() != avaxAssetID {
				continue
			}

			op := &types.Operation{
				OperationIdentifier: &types.OperationIdentifier{
					Index: idx,
				},
				Type:   OpImport,
				Status: types.String(StatusSuccess),
				Account: &types.AccountIdentifier{
					Address: out.Address.Hex(),
				},
				Amount: &types.Amount{
					Value:    new(big.Int).Mul(new(big.Int).SetUint64(out.Amount), x2crate).String(),
					Currency: AvaxCurrency,
				},
				Metadata: map[string]interface{}{
					"tx":            t.ID().String(),
					"tx_ids":        txIDs,
					"blockchain_id": t.BlockchainID.String(),
					"network_id":    t.NetworkID,
					"source_chain":  t.SourceChain.String(),
					"meta":          t.Metadata,
					"asset_id":      out.AssetID.String(),
				},
			}
			ops = append(ops, op)
			idx++
		}
	case *evm.UnsignedExportTx:
		for _, in := range t.Ins {
			if in.AssetID.String() != avaxAssetID {
				continue
			}

			op := &types.Operation{
				OperationIdentifier: &types.OperationIdentifier{
					Index: idx,
				},
				Type:   OpExport,
				Status: types.String(StatusSuccess),
				Account: &types.AccountIdentifier{
					Address: in.Address.Hex(),
				},
				Amount: &types.Amount{
					Value:    new(big.Int).Mul(new(big.Int).SetUint64(in.Amount), new(big.Int).Neg(x2crate)).String(),
					Currency: AvaxCurrency,
				},
				Metadata: map[string]interface{}{
					"tx":                t.ID().String(),
					"blockchain_id":     t.BlockchainID.String(),
					"network_id":        t.NetworkID,
					"destination_chain": t.DestinationChain.String(),
					"meta":              t.Metadata,
					"asset_id":          in.AssetID.String(),
				},
			}
			ops = append(ops, op)
			idx++
		}
	default:
		return nil, fmt.Errorf("unsupported transaction: %T", t)
	}
	return ops, nil
}

func CrossChainTransactions(
	avaxAssetID string,
	block *ethtypes.Block,
	ap5Activation uint64,
) ([]*types.Transaction, error) {
	transactions := []*types.Transaction{}

	extra := block.ExtData()
	if len(extra) == 0 {
		return transactions, nil
	}

	atomicTxs, err := evm.ExtractAtomicTxs(extra, block.Time() >= ap5Activation, codecManager)
	if err != nil {
		return nil, err
	}

	ops := []*types.Operation{}
	for _, tx := range atomicTxs {
		txOps, err := crossChainTransaction(len(ops), avaxAssetID, tx)
		if err != nil {
			return nil, err
		}
		ops = append(ops, txOps...)
	}

	// TODO: migrate to using atomic transaction ID instead of marking as a block
	// transaction
	//
	// NOTE: We need to be very careful about this because it will require
	// integrators to re-index the chain to get the new result.
	transactions = append(transactions, &types.Transaction{
		TransactionIdentifier: &types.TransactionIdentifier{
			Hash: block.Hash().String(),
		},
		Operations: ops,
	})

	return transactions, nil
}

// MempoolTransactionsIDs returns a list of transction IDs in the mempool
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

// nolint:gocognit
func traceOps(trace []*client.FlatCall, startIndex int) []*types.Operation {
	var ops []*types.Operation
	if len(trace) == 0 {
		return ops
	}

	destroyedAccounts := map[string]*big.Int{}
	for _, call := range trace {
		// Handle partial transaction success
		metadata := map[string]interface{}{}
		opStatus := StatusSuccess
		if call.Revert {
			opStatus = StatusFailure
			metadata["error"] = call.Error
		}

		var zeroValue bool
		if call.Value.Sign() == 0 {
			zeroValue = true
		}

		// Skip all 0 value CallType operations (TODO: make optional to include)
		//
		// We can't continue here because we may need to adjust our destroyed
		// accounts map if a CallTYpe operation resurrects an account.
		shouldAdd := true
		if zeroValue && CallType(call.Type) {
			shouldAdd = false
		}

		// Checksum addresses
		from := call.From.String()
		to := call.To.String()

		if shouldAdd {
			fromOp := &types.Operation{
				OperationIdentifier: &types.OperationIdentifier{
					Index: int64(len(ops) + startIndex),
				},
				Type:   call.Type,
				Status: types.String(opStatus),
				Account: &types.AccountIdentifier{
					Address: from,
				},
				Amount: &types.Amount{
					Value:    new(big.Int).Neg(call.Value).String(),
					Currency: AvaxCurrency,
				},
				Metadata: metadata,
			}
			if zeroValue {
				fromOp.Amount = nil
			} else {
				_, destroyed := destroyedAccounts[from]
				if destroyed && opStatus == StatusSuccess {
					destroyedAccounts[from] = new(big.Int).Sub(destroyedAccounts[from], call.Value)
				}
			}

			ops = append(ops, fromOp)
		}

		// Add to destroyed accounts if SELFDESTRUCT
		// and overwrite existing balance.
		if call.Type == OpSelfDestruct {
			destroyedAccounts[from] = new(big.Int)

			// If destination of of SELFDESTRUCT is self,
			// we should skip. In the EVM, the balance is reset
			// after the balance is increased on the destination
			// so this is a no-op.
			if from == to {
				continue
			}
		}

		// Skip empty to addresses (this may not
		// actually occur but leaving it as a
		// sanity check)
		if len(call.To.String()) == 0 {
			continue
		}

		// If the account is resurrected, we remove it from
		// the destroyed accounts map.
		if CreateType(call.Type) {
			delete(destroyedAccounts, to)
		}

		if shouldAdd {
			lastOpIndex := ops[len(ops)-1].OperationIdentifier.Index
			toOp := &types.Operation{
				OperationIdentifier: &types.OperationIdentifier{
					Index: lastOpIndex + 1,
				},
				RelatedOperations: []*types.OperationIdentifier{
					{
						Index: lastOpIndex,
					},
				},
				Type:   call.Type,
				Status: types.String(opStatus),
				Account: &types.AccountIdentifier{
					Address: to,
				},
				Amount: &types.Amount{
					Value:    call.Value.String(),
					Currency: AvaxCurrency,
				},
				Metadata: metadata,
			}
			if zeroValue {
				toOp.Amount = nil
			} else {
				_, destroyed := destroyedAccounts[to]
				if destroyed && opStatus == StatusSuccess {
					destroyedAccounts[to] = new(big.Int).Add(destroyedAccounts[to], call.Value)
				}
			}

			ops = append(ops, toOp)
		}
	}

	// Zero-out all destroyed accounts that are removed
	// during transaction finalization.
	for acct, val := range destroyedAccounts {
		if val.Sign() == 0 {
			continue
		}

		if val.Sign() < 0 {
			log.Fatalf("negative balance for suicided account %s: %s\n", acct, val.String())
		}

		ops = append(ops, &types.Operation{
			OperationIdentifier: &types.OperationIdentifier{
				Index: ops[len(ops)-1].OperationIdentifier.Index + 1,
			},
			Type:   OpDestruct,
			Status: types.String(StatusSuccess),
			Account: &types.AccountIdentifier{
				Address: acct,
			},
			Amount: &types.Amount{
				Value:    new(big.Int).Neg(val).String(),
				Currency: AvaxCurrency,
			},
		})
	}

	return ops
}

func parseErc20Txs(transferLog ethtypes.Log, contractInfo *clientTypes.ContractInfo, opsLen int64) []*types.Operation {
	ops := []*types.Operation{}

	contractAddress := transferLog.Address
	addressFrom := transferLog.Topics[1]
	addressTo := transferLog.Topics[2]

	if addressFrom.Hex() == zeroAddress {
		mintOp := types.Operation{
			OperationIdentifier: &types.OperationIdentifier{
				Index: opsLen,
			},
			Status:  types.String(StatusSuccess),
			Type:    OpErc20Mint,
			Amount:  Erc20Amount(transferLog.Data, contractAddress, contractInfo.Symbol, contractInfo.Decimals, false),
			Account: Account(ConvertEVMTopicHashToAddress(&addressTo)),
		}
		ops = append(ops, &mintOp)
		return ops
	}

	if addressTo.Hex() == zeroAddress {
		burnOp := types.Operation{
			OperationIdentifier: &types.OperationIdentifier{
				Index: opsLen,
			},
			Status:  types.String(StatusSuccess),
			Type:    OpErc20Burn,
			Amount:  Erc20Amount(transferLog.Data, contractAddress, contractInfo.Symbol, contractInfo.Decimals, true),
			Account: Account(ConvertEVMTopicHashToAddress(&addressFrom)),
		}
		ops = append(ops, &burnOp)
		return ops
	}

	sendingOp := types.Operation{
		OperationIdentifier: &types.OperationIdentifier{
			Index: opsLen,
		},
		Status:  types.String(StatusSuccess),
		Type:    OpErc20Transfer,
		Amount:  Erc20Amount(transferLog.Data, contractAddress, contractInfo.Symbol, contractInfo.Decimals, true),
		Account: Account(ConvertEVMTopicHashToAddress(&addressFrom)),
	}
	receiptOp := types.Operation{
		OperationIdentifier: &types.OperationIdentifier{
			Index: opsLen + 1,
		},
		Status:  types.String(StatusSuccess),
		Type:    OpErc20Transfer,
		Amount:  Erc20Amount(transferLog.Data, contractAddress, contractInfo.Symbol, contractInfo.Decimals, false),
		Account: Account(ConvertEVMTopicHashToAddress(&addressTo)),
		RelatedOperations: []*types.OperationIdentifier{
			{
				Index: opsLen,
			},
		},
	}
	ops = append(ops, &sendingOp)
	ops = append(ops, &receiptOp)

	return ops
}

func parseErc721Txs(transferLog ethtypes.Log, opsLen int64) []*types.Operation {
	ops := []*types.Operation{}

	contractAddress := transferLog.Address
	addressFrom := transferLog.Topics[1]
	addressTo := transferLog.Topics[2]
	erc721Index := transferLog.Topics[3] // Erc721 4th topic is the index.  Data is empty
	metadata := map[string]interface{}{
		ContractAddressMetadata: contractAddress.String(),
		IndexTransferedMetadata: erc721Index.String(),
	}

	if addressFrom.Hex() == zeroAddress {
		mintOp := types.Operation{
			OperationIdentifier: &types.OperationIdentifier{
				Index: opsLen,
			},
			Status:   types.String(StatusSuccess),
			Type:     OpErc721Mint,
			Account:  Account(ConvertEVMTopicHashToAddress(&addressTo)),
			Metadata: metadata,
		}
		ops = append(ops, &mintOp)
		return ops
	}

	if addressTo.Hex() == zeroAddress {
		burnOp := types.Operation{
			OperationIdentifier: &types.OperationIdentifier{
				Index: opsLen,
			},
			Status:   types.String(StatusSuccess),
			Type:     OpErc721Burn,
			Account:  Account(ConvertEVMTopicHashToAddress(&addressFrom)),
			Metadata: metadata,
		}
		ops = append(ops, &burnOp)
		return ops
	}

	sendingOp := types.Operation{
		OperationIdentifier: &types.OperationIdentifier{
			Index: opsLen,
		},
		Status:   types.String(StatusSuccess),
		Type:     OpErc721TransferSender,
		Account:  Account(ConvertEVMTopicHashToAddress(&addressFrom)),
		Metadata: metadata,
	}
	receiptOp := types.Operation{
		OperationIdentifier: &types.OperationIdentifier{
			Index: opsLen + 1,
		},
		Status:   types.String(StatusSuccess),
		Type:     OpErc721TransferReceive,
		Account:  Account(ConvertEVMTopicHashToAddress(&addressTo)),
		Metadata: metadata,
		RelatedOperations: []*types.OperationIdentifier{
			{
				Index: opsLen,
			},
		},
	}
	ops = append(ops, &sendingOp)
	ops = append(ops, &receiptOp)
	return ops
}
