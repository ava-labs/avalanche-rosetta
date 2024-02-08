package pchain

import (
	"math/big"
	"testing"

	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/avalanchego/utils/constants"
	"github.com/ava-labs/avalanchego/vms/components/avax"
	"github.com/coinbase/rosetta-sdk-go/types"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/ava-labs/avalanche-rosetta/client"
	rosConst "github.com/ava-labs/avalanche-rosetta/constants"
	"github.com/ava-labs/avalanche-rosetta/mapper"
)

var (
	avaxAssetID, _ = ids.FromString("U8iRqJoiJm8xZHAacmvYyZVwqQx6uDNtQeP3CQ6fcgQk3JqnK")
	cChainID, _    = ids.FromString("yH8D7ThNJkxmtkuv2jgBa4P1Rn3Qpr4pPr7QYNfcdoS6k6HWp")
	chainIDs       = map[ids.ID]rosConst.ChainIDAlias{
		ids.Empty: rosConst.PChain,
		cChainID:  rosConst.CChain,
	}
)

func TestMapInOperation(t *testing.T) {
	require := require.New(t)

	_, addValidatorTx, inputAccounts := buildValidatorTx()

	require.Len(addValidatorTx.Ins, 2)
	require.Empty(addValidatorTx.Outs)

	ctrl := gomock.NewController(t)
	pchainClient := client.NewMockPChainClient(ctrl)
	parserCfg := TxParserConfig{
		IsConstruction: false,
		Hrp:            constants.FujiHRP,
		ChainIDs:       chainIDs,
		AvaxAssetID:    avaxAssetID,
		PChainClient:   pchainClient,
	}
	parser, err := NewTxParser(parserCfg, inputAccounts, nil)
	require.NoError(err)
	inOps := newTxOps(false)
	require.NoError(parser.insToOperations(inOps, OpAddValidator, addValidatorTx.Ins, OpTypeInput))

	rosettaInOp := inOps.Ins

	// first input checks
	in := addValidatorTx.Ins[0]
	rosettaOp := rosettaInOp[0]
	require.Equal(int64(0), rosettaOp.OperationIdentifier.Index)
	require.Equal(OpAddValidator, rosettaOp.Type)
	require.Equal(in.UTXOID.String(), rosettaOp.CoinChange.CoinIdentifier.Identifier)
	require.Equal(types.CoinSpent, rosettaOp.CoinChange.CoinAction)
	require.Equal(OpTypeInput, rosettaOp.Metadata["type"])
	require.Equal(OpAddValidator, rosettaOp.Type)
	require.Equal(types.String(mapper.StatusSuccess), rosettaOp.Status)
	require.Equal(float64(0), rosettaOp.Metadata["locktime"])
	require.Nil(rosettaOp.Metadata["threshold"])
	require.NotNil(rosettaOp.Metadata["sig_indices"])

	// second input checks
	in = addValidatorTx.Ins[1]
	rosettaOp = rosettaInOp[1]
	require.Equal(int64(1), rosettaOp.OperationIdentifier.Index)
	require.Equal(OpAddValidator, rosettaOp.Type)
	require.Equal(in.UTXOID.String(), rosettaOp.CoinChange.CoinIdentifier.Identifier)
	require.Equal(types.CoinSpent, rosettaOp.CoinChange.CoinAction)
	require.Equal(OpTypeInput, rosettaOp.Metadata["type"])
	require.Equal(OpAddValidator, rosettaOp.Type)
	require.Equal(types.String(mapper.StatusSuccess), rosettaOp.Status)
	require.Equal(float64(1666781236), rosettaOp.Metadata["locktime"])
	require.Nil(rosettaOp.Metadata["threshold"])
	require.NotNil(rosettaOp.Metadata["sig_indices"])
}

func TestMapNonAvaxTransactionInConstruction(t *testing.T) {
	require := require.New(t)

	_, importTx, inputAccounts := buildImport()

	avaxIn := importTx.ImportedInputs[0]

	ctrl := gomock.NewController(t)
	pchainClient := client.NewMockPChainClient(ctrl)
	parserCfg := TxParserConfig{
		IsConstruction: true,
		Hrp:            constants.FujiHRP,
		ChainIDs:       chainIDs,

		// passing empty as AVAX id, so that
		// actual avax id in import transaction will not match with AVAX transaction
		AvaxAssetID:  ids.Empty,
		PChainClient: pchainClient,
	}
	parser, err := NewTxParser(parserCfg, inputAccounts, nil)
	require.NoError(err)
	inOps := newTxOps(true)
	err = parser.insToOperations(inOps, OpImportAvax, []*avax.TransferableInput{avaxIn}, OpTypeInput)
	require.ErrorIs(errUnsupportedAssetInConstruction, err)
}

func TestMapOutOperation(t *testing.T) {
	require := require.New(t)

	_, addDelegatorTx, inputAccounts := buildAddDelegator()

	require.Len(addDelegatorTx.Ins, 1)
	require.Len(addDelegatorTx.Outs, 1)

	avaxOut := addDelegatorTx.Outs[0]

	ctrl := gomock.NewController(t)
	pchainClient := client.NewMockPChainClient(ctrl)
	parserCfg := TxParserConfig{
		IsConstruction: true,
		Hrp:            constants.FujiHRP,
		ChainIDs:       chainIDs,
		AvaxAssetID:    avaxAssetID,
		PChainClient:   pchainClient,
	}
	parser, err := NewTxParser(parserCfg, inputAccounts, nil)
	require.NoError(err)
	outOps := newTxOps(false)
	require.NoError(parser.outsToOperations(outOps, OpAddDelegator, ids.Empty, []*avax.TransferableOutput{avaxOut}, OpTypeOutput, rosConst.PChain))

	rosettaOutOp := outOps.Outs

	require.Equal(int64(0), rosettaOutOp[0].OperationIdentifier.Index)
	require.Equal("P-fuji1gdkq8g208e3j4epyjmx65jglsw7vauh86l47ac", rosettaOutOp[0].Account.Address)
	require.Equal(mapper.AtomicAvaxCurrency, rosettaOutOp[0].Amount.Currency)
	require.Equal("996649063", rosettaOutOp[0].Amount.Value)
	require.Equal(OpTypeOutput, rosettaOutOp[0].Metadata["type"])
	require.Nil(rosettaOutOp[0].Status)
	require.Equal(OpAddDelegator, rosettaOutOp[0].Type)

	require.NotNil(rosettaOutOp[0].Metadata["threshold"])
	require.NotNil(rosettaOutOp[0].Metadata["locktime"])
	require.Nil(rosettaOutOp[0].Metadata["sig_indices"])
}

func TestMapAddValidatorTx(t *testing.T) {
	require := require.New(t)

	signedTx, addValidatorTx, inputAccounts := buildValidatorTx()

	require.Len(addValidatorTx.Ins, 2)
	require.Empty(addValidatorTx.Outs)

	ctrl := gomock.NewController(t)
	pchainClient := client.NewMockPChainClient(ctrl)
	parserCfg := TxParserConfig{
		IsConstruction: true,
		Hrp:            constants.FujiHRP,
		ChainIDs:       chainIDs,
		AvaxAssetID:    avaxAssetID,
		PChainClient:   pchainClient,
	}
	parser, err := NewTxParser(parserCfg, inputAccounts, nil)
	require.NoError(err)
	rosettaTransaction, err := parser.Parse(signedTx)
	require.NoError(err)

	total := len(addValidatorTx.Ins) + len(addValidatorTx.Outs) + len(addValidatorTx.StakeOuts)
	require.Len(rosettaTransaction.Operations, total)

	cntTxType, cntInputMeta, cntOutputMeta, cntMetaType := verifyRosettaTransaction(rosettaTransaction.Operations, OpAddValidator, OpTypeStakeOutput)

	require.Equal(3, cntTxType)
	require.Equal(2, cntInputMeta)
	require.Equal(0, cntOutputMeta)
	require.Equal(1, cntMetaType)
}

func TestMapAddDelegatorTx(t *testing.T) {
	require := require.New(t)

	signedTx, addDelegatorTx, inputAccounts := buildAddDelegator()

	require.Len(addDelegatorTx.Ins, 1)
	require.Len(addDelegatorTx.Outs, 1)
	require.Len(addDelegatorTx.StakeOuts, 1)

	ctrl := gomock.NewController(t)
	pchainClient := client.NewMockPChainClient(ctrl)
	parserCfg := TxParserConfig{
		IsConstruction: true,
		Hrp:            constants.FujiHRP,
		ChainIDs:       chainIDs,
		AvaxAssetID:    avaxAssetID,
		PChainClient:   pchainClient,
	}
	parser, err := NewTxParser(parserCfg, inputAccounts, nil)
	require.NoError(err)
	rosettaTransaction, err := parser.Parse(signedTx)
	require.NoError(err)

	total := len(addDelegatorTx.Ins) + len(addDelegatorTx.Outs) + len(addDelegatorTx.StakeOuts)
	require.Len(rosettaTransaction.Operations, total)

	cntTxType, cntInputMeta, cntOutputMeta, cntMetaType := verifyRosettaTransaction(rosettaTransaction.Operations, OpAddDelegator, OpTypeStakeOutput)

	require.Equal(3, cntTxType)
	require.Equal(1, cntInputMeta)
	require.Equal(1, cntOutputMeta)
	require.Equal(1, cntMetaType)

	require.Equal(types.CoinSpent, rosettaTransaction.Operations[0].CoinChange.CoinAction)
	require.Nil(rosettaTransaction.Operations[1].CoinChange)
	require.Nil(rosettaTransaction.Operations[2].CoinChange)

	require.Equal(addDelegatorTx.Ins[0].UTXOID.String(), rosettaTransaction.Operations[0].CoinChange.CoinIdentifier.Identifier)

	require.Equal(int64(0), rosettaTransaction.Operations[0].OperationIdentifier.Index)
	require.Equal(int64(1), rosettaTransaction.Operations[1].OperationIdentifier.Index)
	require.Equal(int64(2), rosettaTransaction.Operations[2].OperationIdentifier.Index)

	require.Equal(OpAddDelegator, rosettaTransaction.Operations[0].Type)
	require.Equal(OpAddDelegator, rosettaTransaction.Operations[1].Type)
	require.Equal(OpAddDelegator, rosettaTransaction.Operations[2].Type)

	require.Equal(OpTypeInput, rosettaTransaction.Operations[0].Metadata["type"])
	require.Equal(OpTypeOutput, rosettaTransaction.Operations[1].Metadata["type"])
	require.Equal(OpTypeStakeOutput, rosettaTransaction.Operations[2].Metadata["type"])
}

func TestMapImportTx(t *testing.T) {
	require := require.New(t)
	signedTx, importTx, inputAccounts := buildImport()

	require.Empty(importTx.Ins)
	require.Len(importTx.Outs, 3)
	require.Len(importTx.ImportedInputs, 1)

	ctrl := gomock.NewController(t)
	pchainClient := client.NewMockPChainClient(ctrl)
	parserCfg := TxParserConfig{
		IsConstruction: true,
		Hrp:            constants.FujiHRP,
		ChainIDs:       chainIDs,
		AvaxAssetID:    avaxAssetID,
		PChainClient:   pchainClient,
	}
	parser, err := NewTxParser(parserCfg, inputAccounts, nil)
	require.NoError(err)
	rosettaTransaction, err := parser.Parse(signedTx)
	require.NoError(err)

	total := len(importTx.Ins) + len(importTx.Outs) + len(importTx.ImportedInputs) - 2 // - 1 for the multisig output
	require.Len(rosettaTransaction.Operations, total)

	cntTxType, cntInputMeta, cntOutputMeta, cntMetaType := verifyRosettaTransaction(rosettaTransaction.Operations, OpImportAvax, OpTypeImport)

	require.Equal(2, cntTxType)
	require.Zero(cntInputMeta)
	require.Equal(1, cntOutputMeta)
	require.Equal(1, cntMetaType)

	require.Equal(types.CoinSpent, rosettaTransaction.Operations[0].CoinChange.CoinAction)
	require.Nil(rosettaTransaction.Operations[1].CoinChange)
}

func TestMapNonConstructionImportTx(t *testing.T) {
	require := require.New(t)

	signedTx, importTx, inputAccounts := buildImport()

	require.Empty(importTx.Ins)
	require.Len(importTx.Outs, 3)
	require.Len(importTx.ImportedInputs, 1)

	ctrl := gomock.NewController(t)
	pchainClient := client.NewMockPChainClient(ctrl)
	parserCfg := TxParserConfig{
		IsConstruction: false,
		Hrp:            constants.FujiHRP,
		ChainIDs:       chainIDs,
		AvaxAssetID:    avaxAssetID,
		PChainClient:   pchainClient,
	}
	parser, err := NewTxParser(parserCfg, inputAccounts, nil)
	require.NoError(err)
	rosettaTransaction, err := parser.Parse(signedTx)
	require.NoError(err)

	total := len(importTx.Ins) + len(importTx.Outs) + len(importTx.ImportedInputs) - 3 // - 1 for the multisig output
	require.Len(rosettaTransaction.Operations, total)

	cntTxType, cntInputMeta, cntOutputMeta, cntMetaType := verifyRosettaTransaction(rosettaTransaction.Operations, OpImportAvax, OpTypeImport)

	require.Equal(1, cntTxType)
	require.Zero(cntInputMeta)
	require.Equal(1, cntOutputMeta)
	require.Zero(cntMetaType)

	require.Equal(types.CoinCreated, rosettaTransaction.Operations[0].CoinChange.CoinAction)

	// Verify that export output are properly generated
	importInputs, ok := rosettaTransaction.Metadata[mapper.MetadataImportedInputs].([]*types.Operation)
	require.True(ok)

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

	require.Equal(expectedImportedInputs, importInputs)
}

func TestMapExportTx(t *testing.T) {
	require := require.New(t)
	signedTx, exportTx, inputAccounts := buildExport()

	require.Len(exportTx.Ins, 1)
	require.Len(exportTx.Outs, 1)
	require.Len(exportTx.ExportedOutputs, 1)

	ctrl := gomock.NewController(t)
	pchainClient := client.NewMockPChainClient(ctrl)
	parserCfg := TxParserConfig{
		IsConstruction: true,
		Hrp:            constants.FujiHRP,
		ChainIDs:       chainIDs,
		AvaxAssetID:    avaxAssetID,
		PChainClient:   pchainClient,
	}
	parser, err := NewTxParser(parserCfg, inputAccounts, nil)
	require.NoError(err)
	rosettaTransaction, err := parser.Parse(signedTx)
	require.NoError(err)

	total := len(exportTx.Ins) + len(exportTx.Outs) + len(exportTx.ExportedOutputs)
	require.Len(rosettaTransaction.Operations, total)

	cntTxType, cntInputMeta, cntOutputMeta, cntMetaType := verifyRosettaTransaction(rosettaTransaction.Operations, OpExportAvax, OpTypeExport)

	require.Equal(3, cntTxType)
	require.Equal(1, cntInputMeta)
	require.Equal(1, cntOutputMeta)
	require.Equal(1, cntMetaType)
}

func TestMapNonConstructionExportTx(t *testing.T) {
	require := require.New(t)

	signedTx, exportTx, inputAccounts := buildExport()

	require.Len(exportTx.Ins, 1)
	require.Len(exportTx.Outs, 1)
	require.Len(exportTx.ExportedOutputs, 1)

	ctrl := gomock.NewController(t)
	pchainClient := client.NewMockPChainClient(ctrl)
	parserCfg := TxParserConfig{
		IsConstruction: false,
		Hrp:            constants.FujiHRP,
		ChainIDs:       chainIDs,
		AvaxAssetID:    avaxAssetID,
		PChainClient:   pchainClient,
	}
	parser, err := NewTxParser(parserCfg, inputAccounts, nil)
	require.NoError(err)
	rosettaTransaction, err := parser.Parse(signedTx)
	require.NoError(err)

	total := len(exportTx.Ins) + len(exportTx.Outs)
	require.Len(rosettaTransaction.Operations, total)

	cntTxType, cntInputMeta, cntOutputMeta, cntMetaType := verifyRosettaTransaction(rosettaTransaction.Operations, OpExportAvax, OpTypeExport)

	require.Equal(2, cntTxType)
	require.Equal(1, cntInputMeta)
	require.Equal(1, cntOutputMeta)
	require.Equal(0, cntMetaType)

	txType, ok := rosettaTransaction.Metadata[MetadataTxType].(string)
	require.True(ok)
	require.Equal(OpExportAvax, txType)

	// Verify that export output are properly generated
	exportOutputs, ok := rosettaTransaction.Metadata[mapper.MetadataExportedOutputs].([]*types.Operation)
	require.True(ok)

	// setting isConstruction to true in order to include exported output in the operations
	parserCfg = TxParserConfig{
		IsConstruction: true,
		Hrp:            constants.FujiHRP,
		ChainIDs:       chainIDs,
		AvaxAssetID:    avaxAssetID,
		PChainClient:   pchainClient,
	}
	parser, err = NewTxParser(parserCfg, inputAccounts, nil)
	require.NoError(err)
	rosettaTransactionWithExportOperations, err := parser.Parse(signedTx)
	require.NoError(err)

	out := rosettaTransactionWithExportOperations.Operations[2]
	out.Status = types.String(mapper.StatusSuccess)
	out.CoinChange = exportOutputs[0].CoinChange
	require.Equal([]*types.Operation{out}, exportOutputs)
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
