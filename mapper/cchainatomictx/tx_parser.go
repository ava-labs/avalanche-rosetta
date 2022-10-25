package cchainatomictx

import (
	"errors"
	"fmt"
	"math/big"
	"strconv"

	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/avalanchego/utils/formatting/address"
	"github.com/ava-labs/avalanchego/vms/components/avax"
	"github.com/ava-labs/avalanchego/vms/secp256k1fx"
	"github.com/ava-labs/coreth/plugin/evm"
	"github.com/coinbase/rosetta-sdk-go/types"

	"github.com/ava-labs/avalanche-rosetta/mapper"
)

var (
	errUnknownDestinationChain  = errors.New("unknown destination chain")
	errNoMatchingInputAddresses = errors.New("no matching input addresses")
)

// TxParser parses C-chain atomic transactions and generate corresponding Rosetta operations
type TxParser struct {
	// hrp used for address formatting
	hrp string
	// chainIDs maps chain ids to chain id aliases
	chainIDs map[ids.ID]string
	// inputTxAccounts contain utxo id to account identifier mappings
	inputTxAccounts map[string]*types.AccountIdentifier
}

// NewTxParser returns a new transaction parser
func NewTxParser(hrp string, chainIDs map[ids.ID]string, inputTxAccounts map[string]*types.AccountIdentifier) *TxParser {
	return &TxParser{hrp: hrp, chainIDs: chainIDs, inputTxAccounts: inputTxAccounts}
}

// Parse converts the given atomic evm tx to corresponding Rosetta operations
// This method is only used during construction.
func (t *TxParser) Parse(tx evm.Tx) ([]*types.Operation, error) {
	switch unsignedTx := tx.UnsignedAtomicTx.(type) {
	case *evm.UnsignedExportTx:
		return t.parseExportTx(unsignedTx)
	case *evm.UnsignedImportTx:
		return t.parseImportTx(unsignedTx)
	default:
		return nil, fmt.Errorf("unsupported tx type")
	}
}

func (t *TxParser) parseExportTx(exportTx *evm.UnsignedExportTx) ([]*types.Operation, error) {
	operations := []*types.Operation{}
	ins := t.insToOperations(0, mapper.OpExport, exportTx.Ins)

	destinationChainID := exportTx.DestinationChain
	chainAlias, ok := t.chainIDs[destinationChainID]
	if !ok {
		return nil, errUnknownDestinationChain
	}

	operations = append(operations, ins...)
	outs, err := t.exportedOutputsToOperations(len(ins), mapper.OpExport, chainAlias, exportTx.ExportedOutputs)
	if err != nil {
		return nil, err
	}
	operations = append(operations, outs...)

	return operations, nil
}

func (t *TxParser) parseImportTx(importTx *evm.UnsignedImportTx) ([]*types.Operation, error) {
	operations := []*types.Operation{}
	ins, err := t.importedInToOperations(0, mapper.OpImport, importTx.ImportedInputs)
	if err != nil {
		return nil, err
	}

	operations = append(operations, ins...)
	outs := t.outsToOperations(len(ins), mapper.OpImport, importTx.Outs)
	operations = append(operations, outs...)

	return operations, nil
}

func (t *TxParser) insToOperations(startIdx int64, opType string, ins []evm.EVMInput) []*types.Operation {
	idx := startIdx
	operations := []*types.Operation{}
	for _, in := range ins {
		inputAmount := new(big.Int).SetUint64(in.Amount)
		operations = append(operations, &types.Operation{
			OperationIdentifier: &types.OperationIdentifier{
				Index: idx,
			},
			Type:    opType,
			Account: &types.AccountIdentifier{Address: in.Address.Hex()},
			// Negating input amount
			Amount: mapper.AtomicAvaxAmount(new(big.Int).Neg(inputAmount)),
		})
		idx++
	}
	return operations
}

func (t *TxParser) importedInToOperations(startIdx int64, opType string, ins []*avax.TransferableInput) ([]*types.Operation, error) {
	idx := startIdx
	operations := []*types.Operation{}
	for _, in := range ins {
		inputAmount := new(big.Int).SetUint64(in.In.Amount())

		utxoID := in.UTXOID.String()
		account, ok := t.inputTxAccounts[utxoID]
		if !ok {
			return nil, errNoMatchingInputAddresses
		}

		operations = append(operations, &types.Operation{
			OperationIdentifier: &types.OperationIdentifier{
				Index: idx,
			},
			Type:    opType,
			Account: account,
			// Negating input amount
			Amount: mapper.AtomicAvaxAmount(new(big.Int).Neg(inputAmount)),
			CoinChange: &types.CoinChange{
				CoinIdentifier: &types.CoinIdentifier{Identifier: utxoID},
				CoinAction:     types.CoinSpent,
			},
		})
		idx++
	}
	return operations, nil
}

func (t *TxParser) outsToOperations(startIdx int, opType string, outs []evm.EVMOutput) []*types.Operation {
	idx := startIdx
	operations := []*types.Operation{}
	for _, out := range outs {
		operations = append(operations, &types.Operation{
			OperationIdentifier: &types.OperationIdentifier{
				Index: int64(idx),
			},
			Account:           &types.AccountIdentifier{Address: out.Address.Hex()},
			RelatedOperations: buildRelatedOperations(startIdx),
			Type:              opType,
			Amount: &types.Amount{
				Value:    strconv.FormatUint(out.Amount, 10),
				Currency: mapper.AtomicAvaxCurrency,
			},
		})
		idx++
	}
	return operations
}

func (t *TxParser) exportedOutputsToOperations(
	startIdx int,
	opType string,
	chainAlias string,
	outs []*avax.TransferableOutput,
) ([]*types.Operation, error) {
	idx := startIdx
	operations := []*types.Operation{}
	for _, out := range outs {
		var addr string
		transferOutput := out.Output().(*secp256k1fx.TransferOutput)
		if transferOutput != nil && len(transferOutput.Addrs) > 0 {
			var err error
			addr, err = address.Format(chainAlias, t.hrp, transferOutput.Addrs[0][:])
			if err != nil {
				return nil, err
			}
		}

		operations = append(operations, &types.Operation{
			OperationIdentifier: &types.OperationIdentifier{
				Index: int64(idx),
			},
			Account:           &types.AccountIdentifier{Address: addr},
			RelatedOperations: buildRelatedOperations(startIdx),
			Type:              opType,
			Amount: &types.Amount{
				Value:    strconv.FormatUint(out.Out.Amount(), 10),
				Currency: mapper.AtomicAvaxCurrency,
			},
		})
		idx++
	}
	return operations, nil
}

func buildRelatedOperations(idx int) []*types.OperationIdentifier {
	var identifiers []*types.OperationIdentifier
	for i := 0; i < idx; i++ {
		identifiers = append(identifiers, &types.OperationIdentifier{
			Index: int64(i),
		})
	}
	return identifiers
}
