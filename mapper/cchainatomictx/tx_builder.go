package cchainatomictx

import (
	"errors"
	"fmt"
	"slices"

	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/avalanchego/utils"
	"github.com/ava-labs/avalanchego/utils/formatting/address"
	"github.com/ava-labs/avalanchego/vms/components/avax"
	"github.com/ava-labs/avalanchego/vms/saevm/cchain/tx"
	"github.com/ava-labs/avalanchego/vms/secp256k1fx"
	"github.com/ava-labs/libevm/common"
	"github.com/coinbase/rosetta-sdk-go/parser"
	"github.com/coinbase/rosetta-sdk-go/types"

	"github.com/ava-labs/avalanche-rosetta/mapper"
)

var errMissingCoinIdentifier = errors.New("input operation does not have coin identifier")

// BuildTx constructs an evm tx based on the provided operation type, Rosetta matches and metadata
// This method is only used during construction.
func BuildTx(opType string, matches []*parser.Match, metadata Metadata, avaxAssetID ids.ID) (*tx.Tx, []*types.AccountIdentifier, error) {
	switch opType {
	case mapper.OpExport:
		return buildExportTx(matches, metadata, avaxAssetID)
	case mapper.OpImport:
		return buildImportTx(matches, metadata, avaxAssetID)
	default:
		return nil, nil, fmt.Errorf("unsupported atomic operation type %s", opType)
	}
}

// [buildExportTx] returns a duly initialized tx if it does not err
func buildExportTx(
	matches []*parser.Match,
	metadata Metadata,
	avaxAssetID ids.ID,
) (*tx.Tx, []*types.AccountIdentifier, error) {
	ins, signers := buildIns(matches, metadata, avaxAssetID)

	exportedOutputs, err := buildExportedOutputs(matches, avaxAssetID)
	if err != nil {
		return nil, nil, err
	}

	return &tx.Tx{Unsigned: &tx.Export{
		NetworkID:        metadata.NetworkID,
		BlockchainID:     metadata.CChainID,
		DestinationChain: *metadata.DestinationChainID,
		Ins:              ins,
		ExportedOutputs:  exportedOutputs,
	}}, signers, nil
}

// [buildImportTx] returns a duly initialized tx if it does not err
func buildImportTx(
	matches []*parser.Match,
	metadata Metadata,
	avaxAssetID ids.ID,
) (*tx.Tx, []*types.AccountIdentifier, error) {
	importedInputs, signers, err := buildImportedInputs(matches, avaxAssetID)
	if err != nil {
		return nil, nil, err
	}

	outs := buildOuts(matches, avaxAssetID)

	return &tx.Tx{Unsigned: &tx.Import{
		NetworkID:      metadata.NetworkID,
		BlockchainID:   metadata.CChainID,
		SourceChain:    *metadata.SourceChainID,
		ImportedInputs: importedInputs,
		Outs:           outs,
	}}, signers, nil
}

func buildIns(matches []*parser.Match, metadata Metadata, avaxAssetID ids.ID) ([]tx.Input, []*types.AccountIdentifier) {
	inputMatch := matches[0]

	ins := make([]tx.Input, 0, len(inputMatch.Operations))
	signers := make([]*types.AccountIdentifier, 0, len(inputMatch.Operations))
	for i, op := range inputMatch.Operations {
		ins = append(ins, tx.Input{
			Address: common.HexToAddress(op.Account.Address),
			Amount:  inputMatch.Amounts[i].Uint64(),
			AssetID: avaxAssetID,
			Nonce:   metadata.Nonce,
		})
		signers = append(signers, op.Account)
	}

	// Signing is performed externally to Rosetta, so no signers are sorted alongside the inputs.
	slices.SortFunc(ins, tx.Input.Compare)

	return ins, signers
}

func buildImportedInputs(matches []*parser.Match, avaxAssetID ids.ID) ([]*avax.TransferableInput, []*types.AccountIdentifier, error) {
	inputMatch := matches[0]

	importedInputs := []*avax.TransferableInput{}
	signers := []*types.AccountIdentifier{}
	for i, op := range inputMatch.Operations {
		if op.CoinChange == nil || op.CoinChange.CoinIdentifier == nil {
			return nil, nil, errMissingCoinIdentifier
		}
		utxoID, err := mapper.DecodeUTXOID(op.CoinChange.CoinIdentifier.Identifier)
		if err != nil {
			return nil, nil, err
		}

		importedInputs = append(importedInputs, &avax.TransferableInput{
			UTXOID: *utxoID,
			Asset:  avax.Asset{ID: avaxAssetID},
			In: &secp256k1fx.TransferInput{
				Amt: inputMatch.Amounts[i].Uint64(),
				Input: secp256k1fx.Input{
					SigIndices: []uint32{0},
				},
			},
		})
		signers = append(signers, op.Account)
	}
	utils.Sort(importedInputs)

	return importedInputs, signers, nil
}

func buildOuts(matches []*parser.Match, avaxAssetID ids.ID) []tx.Output {
	outputMatch := matches[1]

	outs := make([]tx.Output, 0, len(outputMatch.Operations))
	for i, op := range outputMatch.Operations {
		outs = append(outs, tx.Output{
			Address: common.HexToAddress(op.Account.Address),
			Amount:  outputMatch.Amounts[i].Uint64(),
			AssetID: avaxAssetID,
		})
	}
	slices.SortFunc(outs, tx.Output.Compare)

	return outs
}

func buildExportedOutputs(matches []*parser.Match, avaxAssetID ids.ID) ([]*avax.TransferableOutput, error) {
	outputMatch := matches[1]

	outs := []*avax.TransferableOutput{}
	for i, op := range outputMatch.Operations {
		destinationAddress, err := address.ParseToID(op.Account.Address)
		if err != nil {
			return nil, err
		}

		outs = append(outs, &avax.TransferableOutput{
			Asset: avax.Asset{ID: avaxAssetID},
			Out: &secp256k1fx.TransferOutput{
				Amt: outputMatch.Amounts[i].Uint64(),
				OutputOwners: secp256k1fx.OutputOwners{
					Locktime:  0,
					Threshold: 1,
					Addrs:     []ids.ShortID{destinationAddress},
				},
			},
		})
	}

	tx.SortExportedOutputs(outs)

	return outs, nil
}
