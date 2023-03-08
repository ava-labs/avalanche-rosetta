package pchain

import (
	"math/big"
	"testing"

	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/avalanchego/utils/constants"
	"github.com/ava-labs/avalanchego/vms/components/avax"
	"github.com/coinbase/rosetta-sdk-go/types"
	"github.com/stretchr/testify/assert"

	rosConst "github.com/ava-labs/avalanche-rosetta/constants"
	"github.com/ava-labs/avalanche-rosetta/mapper"
	mocks "github.com/ava-labs/avalanche-rosetta/mocks/client"
)

var (
	avaxAssetID, _ = ids.FromString("U8iRqJoiJm8xZHAacmvYyZVwqQx6uDNtQeP3CQ6fcgQk3JqnK")
	cChainID, _    = ids.FromString("yH8D7ThNJkxmtkuv2jgBa4P1Rn3Qpr4pPr7QYNfcdoS6k6HWp")
	chainIDs       = map[ids.ID]rosConst.ChainIDAlias{
		ids.Empty: rosConst.PChain,
		cChainID:  rosConst.CChain,
	}

	pchainClient = &mocks.PChainClient{}
)

func TestMapInOperation(t *testing.T) {
	_, addValidatorTx, inputAccounts := buildValidatorTx()

	assert.Equal(t, 2, len(addValidatorTx.Ins))
	assert.Equal(t, 0, len(addValidatorTx.Outs))

	parserCfg := TxParserConfig{
		IsConstruction: false,
		Hrp:            constants.FujiHRP,
		ChainIDs:       chainIDs,
		AvaxAssetID:    avaxAssetID,
		PChainClient:   pchainClient,
	}
	parser, _ := NewTxParser(parserCfg, inputAccounts, nil)
	inOps := newTxOps(false)
	err := parser.insToOperations(inOps, OpAddValidator, addValidatorTx.Ins, OpTypeInput)
	assert.Nil(t, err)

	rosettaInOp := inOps.Ins

	// first input checks
	in := addValidatorTx.Ins[0]
	rosettaOp := rosettaInOp[0]
	assert.Equal(t, int64(0), rosettaOp.OperationIdentifier.Index)
	assert.Equal(t, OpAddValidator, rosettaOp.Type)
	assert.Equal(t, in.UTXOID.String(), rosettaOp.CoinChange.CoinIdentifier.Identifier)
	assert.Equal(t, types.CoinSpent, rosettaOp.CoinChange.CoinAction)
	assert.Equal(t, OpTypeInput, rosettaOp.Metadata["type"])
	assert.Equal(t, OpAddValidator, rosettaOp.Type)
	assert.Equal(t, types.String(mapper.StatusSuccess), rosettaOp.Status)
	assert.Equal(t, float64(0), rosettaOp.Metadata["locktime"])
	assert.Nil(t, rosettaOp.Metadata["threshold"])
	assert.NotNil(t, rosettaOp.Metadata["sig_indices"])

	// second input checks
	in = addValidatorTx.Ins[1]
	rosettaOp = rosettaInOp[1]
	assert.Equal(t, int64(1), rosettaOp.OperationIdentifier.Index)
	assert.Equal(t, OpAddValidator, rosettaOp.Type)
	assert.Equal(t, in.UTXOID.String(), rosettaOp.CoinChange.CoinIdentifier.Identifier)
	assert.Equal(t, types.CoinSpent, rosettaOp.CoinChange.CoinAction)
	assert.Equal(t, OpTypeInput, rosettaOp.Metadata["type"])
	assert.Equal(t, OpAddValidator, rosettaOp.Type)
	assert.Equal(t, types.String(mapper.StatusSuccess), rosettaOp.Status)
	assert.Equal(t, float64(1666781236), rosettaOp.Metadata["locktime"])
	assert.Nil(t, rosettaOp.Metadata["threshold"])
	assert.NotNil(t, rosettaOp.Metadata["sig_indices"])
}

func TestMapNonAvaxTransactionInConstruction(t *testing.T) {
	_, importTx, inputAccounts := buildImport()

	avaxIn := importTx.ImportedInputs[0]

	parserCfg := TxParserConfig{
		IsConstruction: true,
		Hrp:            constants.FujiHRP,
		ChainIDs:       chainIDs,

		// passing empty as AVAX id, so that
		// actual avax id in import transaction will not match with AVAX transaction
		AvaxAssetID:  ids.Empty,
		PChainClient: pchainClient,
	}
	parser, _ := NewTxParser(parserCfg, inputAccounts, nil)
	inOps := newTxOps(true)
	err := parser.insToOperations(inOps, OpImportAvax, []*avax.TransferableInput{avaxIn}, OpTypeInput)
	assert.ErrorIs(t, errUnsupportedAssetInConstruction, err)
}

func TestMapOutOperation(t *testing.T) {
	_, addDelegatorTx, inputAccounts := buildAddDelegator()

	assert.Equal(t, 1, len(addDelegatorTx.Ins))
	assert.Equal(t, 1, len(addDelegatorTx.Outs))

	avaxOut := addDelegatorTx.Outs[0]

	parserCfg := TxParserConfig{
		IsConstruction: true,
		Hrp:            constants.FujiHRP,
		ChainIDs:       chainIDs,
		AvaxAssetID:    avaxAssetID,
		PChainClient:   pchainClient,
	}
	parser, _ := NewTxParser(parserCfg, inputAccounts, nil)
	outOps := newTxOps(false)
	err := parser.outsToOperations(outOps, OpAddDelegator, ids.Empty, []*avax.TransferableOutput{avaxOut}, OpTypeOutput, rosConst.PChain)
	assert.Nil(t, err)

	rosettaOutOp := outOps.Outs

	assert.Equal(t, int64(0), rosettaOutOp[0].OperationIdentifier.Index)
	assert.Equal(t, "P-fuji1gdkq8g208e3j4epyjmx65jglsw7vauh86l47ac", rosettaOutOp[0].Account.Address)
	assert.Equal(t, mapper.AtomicAvaxCurrency, rosettaOutOp[0].Amount.Currency)
	assert.Equal(t, "996649063", rosettaOutOp[0].Amount.Value)
	assert.Equal(t, OpTypeOutput, rosettaOutOp[0].Metadata["type"])
	assert.Nil(t, rosettaOutOp[0].Status)
	assert.Equal(t, OpAddDelegator, rosettaOutOp[0].Type)

	assert.NotNil(t, rosettaOutOp[0].Metadata["threshold"])
	assert.NotNil(t, rosettaOutOp[0].Metadata["locktime"])
	assert.Nil(t, rosettaOutOp[0].Metadata["sig_indices"])
}

func TestMapAddValidatorTx(t *testing.T) {
	signedTx, addValidatorTx, inputAccounts := buildValidatorTx()

	assert.Equal(t, 2, len(addValidatorTx.Ins))
	assert.Equal(t, 0, len(addValidatorTx.Outs))

	parserCfg := TxParserConfig{
		IsConstruction: true,
		Hrp:            constants.FujiHRP,
		ChainIDs:       chainIDs,
		AvaxAssetID:    avaxAssetID,
		PChainClient:   pchainClient,
	}
	parser, _ := NewTxParser(parserCfg, inputAccounts, nil)
	rosettaTransaction, err := parser.Parse(signedTx)
	assert.Nil(t, err)

	total := len(addValidatorTx.Ins) + len(addValidatorTx.Outs) + len(addValidatorTx.StakeOuts)
	assert.Equal(t, total, len(rosettaTransaction.Operations))

	cntTxType, cntInputMeta, cntOutputMeta, cntMetaType := verifyRosettaTransaction(rosettaTransaction.Operations, OpAddValidator, OpTypeStakeOutput)

	assert.Equal(t, 3, cntTxType)
	assert.Equal(t, 2, cntInputMeta)
	assert.Equal(t, 0, cntOutputMeta)
	assert.Equal(t, 1, cntMetaType)
}

func TestMapAddDelegatorTx(t *testing.T) {
	signedTx, addDelegatorTx, inputAccounts := buildAddDelegator()

	assert.Equal(t, 1, len(addDelegatorTx.Ins))
	assert.Equal(t, 1, len(addDelegatorTx.Outs))
	assert.Equal(t, 1, len(addDelegatorTx.StakeOuts))

	parserCfg := TxParserConfig{
		IsConstruction: true,
		Hrp:            constants.FujiHRP,
		ChainIDs:       chainIDs,
		AvaxAssetID:    avaxAssetID,
		PChainClient:   pchainClient,
	}
	parser, _ := NewTxParser(parserCfg, inputAccounts, nil)
	rosettaTransaction, err := parser.Parse(signedTx)
	assert.Nil(t, err)

	total := len(addDelegatorTx.Ins) + len(addDelegatorTx.Outs) + len(addDelegatorTx.StakeOuts)
	assert.Equal(t, total, len(rosettaTransaction.Operations))

	cntTxType, cntInputMeta, cntOutputMeta, cntMetaType := verifyRosettaTransaction(rosettaTransaction.Operations, OpAddDelegator, OpTypeStakeOutput)

	assert.Equal(t, 3, cntTxType)
	assert.Equal(t, 1, cntInputMeta)
	assert.Equal(t, 1, cntOutputMeta)
	assert.Equal(t, 1, cntMetaType)

	assert.Equal(t, types.CoinSpent, rosettaTransaction.Operations[0].CoinChange.CoinAction)
	assert.Nil(t, rosettaTransaction.Operations[1].CoinChange)
	assert.Nil(t, rosettaTransaction.Operations[2].CoinChange)

	assert.Equal(t, addDelegatorTx.Ins[0].UTXOID.String(), rosettaTransaction.Operations[0].CoinChange.CoinIdentifier.Identifier)

	assert.Equal(t, int64(0), rosettaTransaction.Operations[0].OperationIdentifier.Index)
	assert.Equal(t, int64(1), rosettaTransaction.Operations[1].OperationIdentifier.Index)
	assert.Equal(t, int64(2), rosettaTransaction.Operations[2].OperationIdentifier.Index)

	assert.Equal(t, OpAddDelegator, rosettaTransaction.Operations[0].Type)
	assert.Equal(t, OpAddDelegator, rosettaTransaction.Operations[1].Type)
	assert.Equal(t, OpAddDelegator, rosettaTransaction.Operations[2].Type)

	assert.Equal(t, OpTypeInput, rosettaTransaction.Operations[0].Metadata["type"])
	assert.Equal(t, OpTypeOutput, rosettaTransaction.Operations[1].Metadata["type"])
	assert.Equal(t, OpTypeStakeOutput, rosettaTransaction.Operations[2].Metadata["type"])
}

func TestMapImportTx(t *testing.T) {
	signedTx, importTx, inputAccounts := buildImport()

	assert.Equal(t, 0, len(importTx.Ins))
	assert.Equal(t, 3, len(importTx.Outs))
	assert.Equal(t, 1, len(importTx.ImportedInputs))

	parserCfg := TxParserConfig{
		IsConstruction: true,
		Hrp:            constants.FujiHRP,
		ChainIDs:       chainIDs,
		AvaxAssetID:    avaxAssetID,
		PChainClient:   pchainClient,
	}
	parser, _ := NewTxParser(parserCfg, inputAccounts, nil)
	rosettaTransaction, err := parser.Parse(signedTx)
	assert.Nil(t, err)

	total := len(importTx.Ins) + len(importTx.Outs) + len(importTx.ImportedInputs) - 2 // - 1 for the multisig output
	assert.Equal(t, total, len(rosettaTransaction.Operations))

	cntTxType, cntInputMeta, cntOutputMeta, cntMetaType := verifyRosettaTransaction(rosettaTransaction.Operations, OpImportAvax, OpTypeImport)

	assert.Equal(t, 2, cntTxType)
	assert.Equal(t, 0, cntInputMeta)
	assert.Equal(t, 1, cntOutputMeta)
	assert.Equal(t, 1, cntMetaType)

	assert.Equal(t, types.CoinSpent, rosettaTransaction.Operations[0].CoinChange.CoinAction)
	assert.Nil(t, rosettaTransaction.Operations[1].CoinChange)
}

func TestMapNonConstructionImportTx(t *testing.T) {
	signedTx, importTx, inputAccounts := buildImport()

	assert.Equal(t, 0, len(importTx.Ins))
	assert.Equal(t, 3, len(importTx.Outs))
	assert.Equal(t, 1, len(importTx.ImportedInputs))

	parserCfg := TxParserConfig{
		IsConstruction: false,
		Hrp:            constants.FujiHRP,
		ChainIDs:       chainIDs,
		AvaxAssetID:    avaxAssetID,
		PChainClient:   pchainClient,
	}
	parser, _ := NewTxParser(parserCfg, inputAccounts, nil)
	rosettaTransaction, err := parser.Parse(signedTx)
	assert.Nil(t, err)

	total := len(importTx.Ins) + len(importTx.Outs) + len(importTx.ImportedInputs) - 3 // - 1 for the multisig output
	assert.Equal(t, total, len(rosettaTransaction.Operations))

	cntTxType, cntInputMeta, cntOutputMeta, cntMetaType := verifyRosettaTransaction(rosettaTransaction.Operations, OpImportAvax, OpTypeImport)

	assert.Equal(t, 1, cntTxType)
	assert.Equal(t, 0, cntInputMeta)
	assert.Equal(t, 1, cntOutputMeta)
	assert.Equal(t, 0, cntMetaType)

	assert.Equal(t, types.CoinCreated, rosettaTransaction.Operations[0].CoinChange.CoinAction)

	// Verify that export output are properly generated
	importInputs, ok := rosettaTransaction.Metadata[mapper.MetadataImportedInputs].([]*types.Operation)
	assert.True(t, ok)

	importedInput := importTx.ImportedInputs[0]
	expectedImportedInputs := []*types.Operation{{
		OperationIdentifier: &types.OperationIdentifier{Index: 1},
		Type:                OpImportAvax,
		Status:              types.String(mapper.StatusSuccess),
		Account:             nil,
		Amount:              mapper.AtomicAvaxAmount(big.NewInt(-int64(importedInput.Input().Amount()))),
		CoinChange: &types.CoinChange{
			CoinIdentifier: &types.CoinIdentifier{Identifier: importedInput.UTXOID.String()},
			CoinAction:     types.CoinSpent,
		},
		Metadata: map[string]interface{}{
			"type":     OpTypeImport,
			"locktime": 0.0,
		},
	}}

	assert.Equal(t, expectedImportedInputs, importInputs)
}

func TestMapExportTx(t *testing.T) {
	signedTx, exportTx, inputAccounts := buildExport()

	assert.Equal(t, 1, len(exportTx.Ins))
	assert.Equal(t, 1, len(exportTx.Outs))
	assert.Equal(t, 1, len(exportTx.ExportedOutputs))

	parserCfg := TxParserConfig{
		IsConstruction: true,
		Hrp:            constants.FujiHRP,
		ChainIDs:       chainIDs,
		AvaxAssetID:    avaxAssetID,
		PChainClient:   pchainClient,
	}
	parser, _ := NewTxParser(parserCfg, inputAccounts, nil)
	rosettaTransaction, err := parser.Parse(signedTx)
	assert.Nil(t, err)

	total := len(exportTx.Ins) + len(exportTx.Outs) + len(exportTx.ExportedOutputs)
	assert.Equal(t, total, len(rosettaTransaction.Operations))

	cntTxType, cntInputMeta, cntOutputMeta, cntMetaType := verifyRosettaTransaction(rosettaTransaction.Operations, OpExportAvax, OpTypeExport)

	assert.Equal(t, 3, cntTxType)
	assert.Equal(t, 1, cntInputMeta)
	assert.Equal(t, 1, cntOutputMeta)
	assert.Equal(t, 1, cntMetaType)
}

func TestMapNonConstructionExportTx(t *testing.T) {
	signedTx, exportTx, inputAccounts := buildExport()

	assert.Equal(t, 1, len(exportTx.Ins))
	assert.Equal(t, 1, len(exportTx.Outs))
	assert.Equal(t, 1, len(exportTx.ExportedOutputs))

	parserCfg := TxParserConfig{
		IsConstruction: false,
		Hrp:            constants.FujiHRP,
		ChainIDs:       chainIDs,
		AvaxAssetID:    avaxAssetID,
		PChainClient:   pchainClient,
	}
	parser, _ := NewTxParser(parserCfg, inputAccounts, nil)
	rosettaTransaction, err := parser.Parse(signedTx)
	assert.Nil(t, err)

	total := len(exportTx.Ins) + len(exportTx.Outs)
	assert.Equal(t, total, len(rosettaTransaction.Operations))

	cntTxType, cntInputMeta, cntOutputMeta, cntMetaType := verifyRosettaTransaction(rosettaTransaction.Operations, OpExportAvax, OpTypeExport)

	assert.Equal(t, 2, cntTxType)
	assert.Equal(t, 1, cntInputMeta)
	assert.Equal(t, 1, cntOutputMeta)
	assert.Equal(t, 0, cntMetaType)

	txType, ok := rosettaTransaction.Metadata[MetadataTxType].(string)
	assert.True(t, ok)
	assert.Equal(t, OpExportAvax, txType)

	// Verify that export output are properly generated
	exportOutputs, ok := rosettaTransaction.Metadata[mapper.MetadataExportedOutputs].([]*types.Operation)
	assert.True(t, ok)

	// setting isConstruction to true in order to include exported output in the operations
	parserCfg = TxParserConfig{
		IsConstruction: true,
		Hrp:            constants.FujiHRP,
		ChainIDs:       chainIDs,
		AvaxAssetID:    avaxAssetID,
		PChainClient:   pchainClient,
	}
	parser, _ = NewTxParser(parserCfg, inputAccounts, nil)
	rosettaTransactionWithExportOperations, err := parser.Parse(signedTx)
	assert.Nil(t, err)

	out := rosettaTransactionWithExportOperations.Operations[2]
	out.Status = types.String(mapper.StatusSuccess)
	out.CoinChange = exportOutputs[0].CoinChange
	assert.Equal(t, []*types.Operation{out}, exportOutputs)
}

func verifyRosettaTransaction(operations []*types.Operation, txType string, metaType string) (int, int, int, int) {
	cntOpInputMeta := 0
	cntOpOutputMeta := 0
	cntTxType := 0
	cntMetaType := 0

	for _, v := range operations {
		if v.Type == txType {
			cntTxType++
		}

		meta := &OperationMetadata{}
		_ = mapper.UnmarshalJSONMap(v.Metadata, meta)

		if meta.Type == OpTypeInput {
			cntOpInputMeta++
			continue
		}
		if meta.Type == OpTypeOutput {
			cntOpOutputMeta++
			continue
		}
		if meta.Type == metaType {
			cntMetaType++
			continue
		}
	}

	return cntTxType, cntOpInputMeta, cntOpOutputMeta, cntMetaType
}
