package mapper

import (
	"fmt"
	"log"
	"math/big"
	"strings"

	ethtypes "github.com/ava-labs/coreth/core/types"
	"github.com/ava-labs/coreth/plugin/evm"
	"github.com/coinbase/rosetta-sdk-go/types"
	"github.com/ethereum/go-ethereum/common"

	clientTypes "github.com/ava-labs/avalanche-rosetta/client"
)

const (
	topicsInErc721Transfer = 4
	topicsInErc20Transfer  = 3

	transferMethodHash = "0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef"
)

var (
	X2crate     = big.NewInt(1000000000)
	zeroAddress = common.Address{}
)

func Transaction(
	header *ethtypes.Header,
	tx *ethtypes.Transaction,
	msg *ethtypes.Message,
	receipt *ethtypes.Receipt,
	trace *clientTypes.Call,
	flattenedTrace []*clientTypes.FlatCall,
	client clientTypes.Client,
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
	for _, log := range receipt.Logs {
		// Only check transfer logs
		if len(log.Topics) == 0 || log.Topics[0].String() != transferMethodHash {
			continue
		}

		// If in standard mode, token address must be whitelisted
		if !isAnalyticsMode && !EqualFoldContains(standardModeWhiteList, log.Address.String()) {
			continue
		}

		switch len(log.Topics) {
		case topicsInErc721Transfer:
			symbol, _, err := client.GetContractInfo(log.Address, false)
			if err != nil {
				return nil, err
			}

			if symbol == clientTypes.UnknownERC721Symbol && !includeUnknownTokens {
				continue
			}

			erc721Ops := erc721Ops(log, int64(len(ops)))
			ops = append(ops, erc721Ops...)
		case topicsInErc20Transfer:
			symbol, decimals, err := client.GetContractInfo(log.Address, true)
			if err != nil {
				return nil, err
			}

			if symbol == clientTypes.UnknownERC20Symbol && !includeUnknownTokens {
				continue
			}

			erc20Ops := erc20Ops(log, ToCurrency(symbol, decimals, log.Address), int64(len(ops)))
			ops = append(ops, erc20Ops...)
		default:
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
	if err := tx.Sign(evm.Codec, nil); err != nil {
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
					Value:    new(big.Int).Mul(new(big.Int).SetUint64(out.Amount), X2crate).String(),
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
					Value:    new(big.Int).Mul(new(big.Int).SetUint64(in.Amount), new(big.Int).Neg(X2crate)).String(),
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

	atomicTxs, err := evm.ExtractAtomicTxs(extra, block.Time() >= ap5Activation, evm.Codec)
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
func MempoolTransactionsIDs(accountMap clientTypes.TxAccountMap) []*types.TransactionIdentifier {
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

func traceOps(trace []*clientTypes.FlatCall, startIndex int) []*types.Operation {
	ops := []*types.Operation{}
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

func erc20Ops(transferLog *ethtypes.Log, currency *types.Currency, opsLen int64) []*types.Operation {
	fromAddress := common.BytesToAddress(transferLog.Topics[1].Bytes())
	toAddress := common.BytesToAddress(transferLog.Topics[2].Bytes())

	// Mint
	if fromAddress == zeroAddress {
		return []*types.Operation{{
			OperationIdentifier: &types.OperationIdentifier{
				Index: opsLen,
			},
			Status:  types.String(StatusSuccess),
			Type:    OpErc20Mint,
			Amount:  Erc20Amount(transferLog.Data, currency, false),
			Account: Account(&toAddress),
		}}
	}

	// Burn
	if toAddress == zeroAddress {
		return []*types.Operation{{
			OperationIdentifier: &types.OperationIdentifier{
				Index: opsLen,
			},
			Status:  types.String(StatusSuccess),
			Type:    OpErc20Burn,
			Amount:  Erc20Amount(transferLog.Data, currency, true),
			Account: Account(&fromAddress),
		}}
	}

	return []*types.Operation{{
		// Send
		OperationIdentifier: &types.OperationIdentifier{
			Index: opsLen,
		},
		Status:  types.String(StatusSuccess),
		Type:    OpErc20Transfer,
		Amount:  Erc20Amount(transferLog.Data, currency, true),
		Account: Account(&fromAddress),
	}, {
		// Receive
		OperationIdentifier: &types.OperationIdentifier{
			Index: opsLen + 1,
		},
		Status:  types.String(StatusSuccess),
		Type:    OpErc20Transfer,
		Amount:  Erc20Amount(transferLog.Data, currency, false),
		Account: Account(&toAddress),
		RelatedOperations: []*types.OperationIdentifier{
			{
				Index: opsLen,
			},
		},
	}}
}

func erc721Ops(transferLog *ethtypes.Log, opsLen int64) []*types.Operation {
	fromAddress := common.BytesToAddress(transferLog.Topics[1].Bytes())
	toAddress := common.BytesToAddress(transferLog.Topics[2].Bytes())
	metadata := map[string]interface{}{
		ContractAddressMetadata:  transferLog.Address.String(),
		IndexTransferredMetadata: transferLog.Topics[3].String(),
	}

	// Mint
	if fromAddress == zeroAddress {
		return []*types.Operation{{
			OperationIdentifier: &types.OperationIdentifier{
				Index: opsLen,
			},
			Status:   types.String(StatusSuccess),
			Type:     OpErc721Mint,
			Account:  Account(&toAddress),
			Metadata: metadata,
		}}
	}

	// Burn
	if toAddress == zeroAddress {
		return []*types.Operation{{
			OperationIdentifier: &types.OperationIdentifier{
				Index: opsLen,
			},
			Status:   types.String(StatusSuccess),
			Type:     OpErc721Burn,
			Account:  Account(&fromAddress),
			Metadata: metadata,
		}}
	}

	return []*types.Operation{{
		// Send
		OperationIdentifier: &types.OperationIdentifier{
			Index: opsLen,
		},
		Status:   types.String(StatusSuccess),
		Type:     OpErc721TransferSender,
		Account:  Account(&fromAddress),
		Metadata: metadata,
	}, {
		// Receive
		OperationIdentifier: &types.OperationIdentifier{
			Index: opsLen + 1,
		},
		Status:   types.String(StatusSuccess),
		Type:     OpErc721TransferReceive,
		Account:  Account(&toAddress),
		Metadata: metadata,
		RelatedOperations: []*types.OperationIdentifier{
			{
				Index: opsLen,
			},
		},
	}}
}
