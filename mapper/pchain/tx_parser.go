package pchain

import (
	"context"
	"errors"
	"fmt"
	"log"
	"math/big"

	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/avalanchego/utils/formatting/address"
	"github.com/ava-labs/avalanchego/vms/components/avax"
	"github.com/ava-labs/avalanchego/vms/platformvm/stakeable"
	"github.com/ava-labs/avalanchego/vms/platformvm/txs"
	"github.com/ava-labs/avalanchego/vms/secp256k1fx"
	"github.com/coinbase/rosetta-sdk-go/types"

	"github.com/ava-labs/avalanche-rosetta/client"
	"github.com/ava-labs/avalanche-rosetta/constants"
	"github.com/ava-labs/avalanche-rosetta/mapper"
)

var (
	errNilPChainClient                = errors.New("pchain client can only be nil during construction")
	errNilInputTxAccounts             = errors.New("input tx accounts cannot be nil")
	errUnknownDestinationChain        = errors.New("unknown destination chain")
	errNoDependencyTxs                = errors.New("no dependency txs provided")
	errNoMatchingRewardOutputs        = errors.New("no matching reward outputs")
	errNoMatchingInputAddresses       = errors.New("no matching input addresses")
	errNoOutputAddresses              = errors.New("no output addresses")
	errFailedToGetUTXOAddresses       = errors.New("failed to get utxo addresses")
	errFailedToCheckMultisig          = errors.New("failed to check utxo for multisig")
	errUnknownOutputType              = errors.New("unknown output type")
	errUnknownInputType               = errors.New("unknown input type")
	errUnknownRewardSourceTransaction = errors.New("unknown source tx type for reward tx")
	errUnsupportedAssetInConstruction = errors.New("unsupported asset passed during construction")
)

type TxParserConfig struct {
	// IsConstruction indicates if parsing is done as part of construction or /block endpoints
	IsConstruction bool
	// Hrp used for address formatting
	Hrp string

	// ChainIDs maps chain id to chain id alias mappings
	// ChainIDs may provided by TxParser called or lazily initialized,
	// as soon as pChainClient is ready to serve requests
	ChainIDs map[ids.ID]constants.ChainIDAlias

	// AvaxAssetID contains asset id for AVAX currency
	AvaxAssetID ids.ID
	// PChainClient holds a P-chain client, used to lookup asset descriptions for non-AVAX assets
	PChainClient client.PChainClient
}

func (cfg *TxParserConfig) lazyInitChainIDs() error {
	if cfg.ChainIDs != nil {
		return nil // mapping provided by caller
	}

	cfg.ChainIDs = map[ids.ID]constants.ChainIDAlias{
		ids.Empty: constants.PChain,
	}

	ctx := context.Background()
	cChainID, err := cfg.PChainClient.GetBlockchainID(ctx, constants.CChain.String())
	if err != nil {
		return err
	}
	cfg.ChainIDs[cChainID] = constants.CChain

	xChainID, err := cfg.PChainClient.GetBlockchainID(ctx, constants.XChain.String())
	if err != nil {
		return err
	}
	cfg.ChainIDs[xChainID] = constants.XChain
	return nil
}

// TxParser parses P-chain transactions and generate corresponding Rosetta operations
type TxParser struct {
	cfg TxParserConfig

	// dependencyTxs maps transaction id to dependence transaction mapping
	dependencyTxs BlockTxDependencies
	// inputTxAccounts contain utxo id to account identifier mappings
	inputTxAccounts map[string]*types.AccountIdentifier
}

// NewTxParser returns a new transaction parser
func NewTxParser(
	cfg TxParserConfig,
	inputTxAccounts map[string]*types.AccountIdentifier,
	dependencyTxs BlockTxDependencies,
) (*TxParser, error) {
	if err := cfg.lazyInitChainIDs(); err != nil {
		return nil, err
	}

	if inputTxAccounts == nil {
		return nil, errNilInputTxAccounts
	}

	if !cfg.IsConstruction && cfg.PChainClient == nil {
		return nil, errNilPChainClient
	}

	return &TxParser{
		cfg:             cfg,
		inputTxAccounts: inputTxAccounts,
		dependencyTxs:   dependencyTxs,
	}, nil
}

// Parse converts the given unsigned P-chain tx to corresponding Rosetta Transaction
func (t *TxParser) Parse(signedTx *txs.Tx) (*types.Transaction, error) {
	var (
		ops    *txOps
		txType string
		err    error
	)

	txID := signedTx.ID()
	switch unsignedTx := signedTx.Unsigned.(type) {
	case *txs.ExportTx:
		txType = OpExportAvax
		ops, err = t.parseExportTx(txID, unsignedTx)
	case *txs.ImportTx:
		txType = OpImportAvax
		ops, err = t.parseImportTx(txID, unsignedTx)
	case *txs.AddValidatorTx:
		txType = OpAddValidator
		ops, err = t.parseAddValidatorTx(txID, unsignedTx)
	case *txs.AddDelegatorTx:
		txType = OpAddDelegator
		ops, err = t.parseAddDelegatorTx(txID, unsignedTx)
	case *txs.RewardValidatorTx:
		txType = OpRewardValidator
		ops, err = t.parseRewardValidatorTx(unsignedTx)
	case *txs.CreateSubnetTx:
		txType = OpCreateSubnet
		ops, err = t.parseCreateSubnetTx(txID, unsignedTx)
	case *txs.CreateChainTx:
		txType = OpCreateChain
		ops, err = t.parseCreateChainTx(txID, unsignedTx)
	case *txs.AddSubnetValidatorTx:
		txType = OpAddSubnetValidator
		ops, err = t.parseAddSubnetValidatorTx(txID, unsignedTx)
	case *txs.AddPermissionlessValidatorTx:
		txType = OpAddPermissionlessValidator
		ops, err = t.parseAddPermissionlessValidatorTx(txID, unsignedTx)
	case *txs.AddPermissionlessDelegatorTx:
		txType = OpAddPermissionlessDelegator
		ops, err = t.parseAddPermissionlessDelegatorTx(txID, unsignedTx)
	case *txs.RemoveSubnetValidatorTx:
		txType = OpRemoveSubnetValidator
		ops, err = t.parseRemoveSubnetValidatorTx(txID, unsignedTx)
	case *txs.TransformSubnetTx:
		txType = OpTransformSubnetValidator
		ops, err = t.parseTransformSubnetTx(txID, unsignedTx)
	case *txs.AdvanceTimeTx:
		txType = OpAdvanceTime
		// no op tx
	default:
		log.Printf("unknown type %T", unsignedTx)
	}
	if err != nil {
		return nil, err
	}

	txMetadata := map[string]interface{}{
		MetadataTxType: txType,
	}

	var operations []*types.Operation
	if ops != nil {
		operations = ops.IncludedOperations()
		idx := len(operations)
		if ops.ImportIns != nil {
			importedInputs := addOperationIdentifiers(ops.ImportIns, idx)
			idx += len(importedInputs)
			txMetadata[mapper.MetadataImportedInputs] = importedInputs
		}

		if ops.ExportOuts != nil {
			exportedOutputs := addOperationIdentifiers(ops.ExportOuts, idx)
			txMetadata[mapper.MetadataExportedOutputs] = exportedOutputs
		}
	}

	return &types.Transaction{
		TransactionIdentifier: &types.TransactionIdentifier{
			Hash: txID.String(),
		},
		Operations: operations,
		Metadata:   txMetadata,
	}, nil
}

func addOperationIdentifiers(operations []*types.Operation, startIdx int) []*types.Operation {
	result := make([]*types.Operation, 0, len(operations))
	for idx, operation := range operations {
		operation := operation
		operation.OperationIdentifier = &types.OperationIdentifier{Index: int64(startIdx + idx)}
		result = append(result, operation)
	}

	return result
}

func (t *TxParser) parseExportTx(txID ids.ID, tx *txs.ExportTx) (*txOps, error) {
	ops, err := t.baseTxToCombinedOperations(txID, &tx.BaseTx, OpExportAvax)
	if err != nil {
		return nil, err
	}

	chainIDAlias, ok := t.cfg.ChainIDs[tx.DestinationChain]
	if !ok {
		return nil, errUnknownDestinationChain
	}

	err = t.outsToOperations(ops, OpExportAvax, txID, tx.ExportedOutputs, OpTypeExport, chainIDAlias)
	if err != nil {
		return nil, err
	}

	return ops, nil
}

func (t *TxParser) parseImportTx(txID ids.ID, tx *txs.ImportTx) (*txOps, error) {
	ops := newTxOps(t.cfg.IsConstruction)

	err := t.insToOperations(ops, OpImportAvax, tx.Ins, OpTypeInput)
	if err != nil {
		return nil, err
	}

	err = t.insToOperations(ops, OpImportAvax, tx.ImportedInputs, OpTypeImport)
	if err != nil {
		return nil, err
	}

	err = t.outsToOperations(ops, OpImportAvax, txID, tx.Outs, OpTypeOutput, constants.PChain)
	if err != nil {
		return nil, err
	}

	return ops, nil
}

func (t *TxParser) parseAddValidatorTx(txID ids.ID, tx *txs.AddValidatorTx) (*txOps, error) {
	ops, err := t.baseTxToCombinedOperations(txID, &tx.BaseTx, OpAddValidator)
	if err != nil {
		return nil, err
	}

	err = t.outsToOperations(ops, OpAddValidator, txID, tx.Stake(), OpTypeStakeOutput, constants.PChain)
	if err != nil {
		return nil, err
	}
	addMetadataToStakeOuts(ops, &tx.Validator)

	return ops, nil
}

func (t *TxParser) parseAddPermissionlessValidatorTx(txID ids.ID, tx *txs.AddPermissionlessValidatorTx) (*txOps, error) {
	ops, err := t.baseTxToCombinedOperations(txID, &tx.BaseTx, OpAddValidator)
	if err != nil {
		return nil, err
	}

	err = t.outsToOperations(ops, OpAddPermissionlessValidator, txID, tx.Stake(), OpTypeStakeOutput, constants.PChain)
	if err != nil {
		return nil, err
	}
	addMetadataToStakeOuts(ops, &tx.Validator)

	if tx.Signer != nil {
		for _, out := range ops.StakeOuts {
			out.Metadata[MetadataSigner] = tx.Signer
		}
	}

	return ops, nil
}

func (t *TxParser) parseAddDelegatorTx(txID ids.ID, tx *txs.AddDelegatorTx) (*txOps, error) {
	ops, err := t.baseTxToCombinedOperations(txID, &tx.BaseTx, OpAddDelegator)
	if err != nil {
		return nil, err
	}

	err = t.outsToOperations(ops, OpAddDelegator, txID, tx.Stake(), OpTypeStakeOutput, constants.PChain)
	if err != nil {
		return nil, err
	}
	addMetadataToStakeOuts(ops, &tx.Validator)

	return ops, nil
}

func (t *TxParser) parseAddPermissionlessDelegatorTx(txID ids.ID, tx *txs.AddPermissionlessDelegatorTx) (*txOps, error) {
	ops, err := t.baseTxToCombinedOperations(txID, &tx.BaseTx, OpAddDelegator)
	if err != nil {
		return nil, err
	}

	err = t.outsToOperations(ops, OpAddPermissionlessDelegator, txID, tx.Stake(), OpTypeStakeOutput, constants.PChain)
	if err != nil {
		return nil, err
	}
	addMetadataToStakeOuts(ops, &tx.Validator)

	return ops, nil
}

func (t *TxParser) parseRewardValidatorTx(tx *txs.RewardValidatorTx) (*txOps, error) {
	stakingTxID := tx.TxID

	if t.dependencyTxs == nil {
		return nil, errNoDependencyTxs
	}
	dep := t.dependencyTxs[stakingTxID]
	if dep == nil {
		return nil, errNoMatchingRewardOutputs
	}
	ops := newTxOps(t.cfg.IsConstruction)
	err := t.utxosToOperations(ops, OpRewardValidator, dep.RewardUTXOs, OpTypeReward, constants.PChain)
	if err != nil {
		return nil, err
	}

	var v *txs.Validator
	switch utx := dep.Tx.Unsigned.(type) {
	case *txs.AddValidatorTx:
		v = &utx.Validator
	case *txs.AddDelegatorTx:
		v = &utx.Validator
	case *txs.AddPermissionlessValidatorTx:
		v = &utx.Validator
	case *txs.AddPermissionlessDelegatorTx:
		v = &utx.Validator
	default:
		return nil, errUnknownRewardSourceTransaction
	}
	addMetadataToStakeOuts(ops, v)

	return ops, nil
}

func addMetadataToStakeOuts(ops *txOps, validator *txs.Validator) {
	if validator == nil {
		return
	}

	for _, out := range ops.StakeOuts {
		out.Metadata[MetadataValidatorNodeID] = validator.NodeID.String()
		out.Metadata[MetadataStakingStartTime] = validator.Start
		out.Metadata[MetadataStakingEndTime] = validator.End
	}
}

func (t *TxParser) parseCreateSubnetTx(txID ids.ID, tx *txs.CreateSubnetTx) (*txOps, error) {
	return t.baseTxToCombinedOperations(txID, &tx.BaseTx, OpCreateSubnet)
}

func (t *TxParser) parseAddSubnetValidatorTx(txID ids.ID, tx *txs.AddSubnetValidatorTx) (*txOps, error) {
	return t.baseTxToCombinedOperations(txID, &tx.BaseTx, OpAddSubnetValidator)
}

func (t *TxParser) parseRemoveSubnetValidatorTx(txID ids.ID, tx *txs.RemoveSubnetValidatorTx) (*txOps, error) {
	return t.baseTxToCombinedOperations(txID, &tx.BaseTx, OpRemoveSubnetValidator)
}

func (t *TxParser) parseTransformSubnetTx(txID ids.ID, tx *txs.TransformSubnetTx) (*txOps, error) {
	return t.baseTxToCombinedOperations(txID, &tx.BaseTx, OpTransformSubnetValidator)
}

func (t *TxParser) parseCreateChainTx(txID ids.ID, tx *txs.CreateChainTx) (*txOps, error) {
	return t.baseTxToCombinedOperations(txID, &tx.BaseTx, OpCreateChain)
}

func (t *TxParser) baseTxToCombinedOperations(txID ids.ID, tx *txs.BaseTx, txType string) (*txOps, error) {
	ops := newTxOps(t.cfg.IsConstruction)

	err := t.insToOperations(ops, txType, tx.Ins, OpTypeInput)
	if err != nil {
		return nil, err
	}

	err = t.outsToOperations(ops, txType, txID, tx.Outs, OpTypeOutput, constants.PChain)
	if err != nil {
		return nil, err
	}

	return ops, nil
}

func (t *TxParser) insToOperations(
	inOps *txOps,
	opType string,
	txIns []*avax.TransferableInput,
	metaType string,
) error {
	status := types.String(mapper.StatusSuccess)
	if t.cfg.IsConstruction {
		status = nil
	}

	for _, in := range txIns {
		metadata := &OperationMetadata{
			Type: metaType,
		}

		input := in.In
		if stakeableIn, ok := input.(*stakeable.LockIn); ok {
			metadata.Locktime = stakeableIn.Locktime
			input = stakeableIn.TransferableIn
		}
		transferInput, ok := input.(*secp256k1fx.TransferInput)
		if !ok {
			return errUnknownInputType
		}
		metadata.SigIndices = transferInput.SigIndices

		opMetadata, err := mapper.MarshalJSONMap(metadata)
		if err != nil {
			return err
		}

		utxoIDStr := in.UTXOID.String()

		var account *types.AccountIdentifier

		// Check if the dependency is not multisig and extract account id from it
		// for non-imported inputs or when tx is being constructed
		if t.cfg.IsConstruction || metaType != OpTypeImport {
			// If dependency txs are provided, which is the case for /block endpoints
			// check whether the input UTXO is multisig. If so, skip it.
			if t.dependencyTxs != nil {
				isMultisig, err := t.isMultisig(in.UTXOID)
				if err != nil {
					return errFailedToCheckMultisig
				}
				if isMultisig {
					continue
				}
			}

			var ok bool
			account, ok = t.inputTxAccounts[utxoIDStr]
			if !ok {
				return errNoMatchingInputAddresses
			}
		}

		bigAmount := new(big.Int).SetUint64(in.In.Amount())
		// Negating input amount
		inputAmount := new(big.Int).Neg(bigAmount)

		amount, err := t.buildAmount(inputAmount, in.AssetID())
		if err != nil {
			return err
		}

		inOp := &types.Operation{
			OperationIdentifier: &types.OperationIdentifier{
				Index: int64(inOps.Len()),
			},
			Type:    opType,
			Status:  status,
			Account: account,
			Amount:  amount,
			CoinChange: &types.CoinChange{
				CoinIdentifier: &types.CoinIdentifier{
					Identifier: utxoIDStr,
				},
				CoinAction: types.CoinSpent,
			},
			Metadata: opMetadata,
		}

		inOps.Append(inOp, metaType)
	}
	return nil
}

func (t *TxParser) buildAmount(value *big.Int, assetID ids.ID) (*types.Amount, error) {
	if assetID == t.cfg.AvaxAssetID {
		return mapper.AtomicAvaxAmount(value), nil
	}

	if t.cfg.IsConstruction {
		return nil, errUnsupportedAssetInConstruction
	}

	currency, err := t.lookupCurrency(assetID)
	if err != nil {
		return nil, err
	}

	return mapper.Amount(value, currency), nil
}

func (t *TxParser) outsToOperations(
	outOps *txOps,
	opType string,
	txID ids.ID,
	txOut []*avax.TransferableOutput,
	metaType string,
	chainIDAlias constants.ChainIDAlias,
) error {
	outIndexOffset := outOps.OutputLen()
	status := types.String(mapper.StatusSuccess)
	if t.cfg.IsConstruction {
		status = nil
	}

	for outIndex, out := range txOut {
		transferOut := out.Out

		if lockOut, ok := transferOut.(*stakeable.LockOut); ok {
			transferOut = lockOut.TransferableOut
		}

		transferOutput, ok := transferOut.(*secp256k1fx.TransferOutput)
		if !ok {
			return errUnknownOutputType
		}

		// Rosetta cannot handle multisig at the moment. In order to pass data validation,
		// we treat multisig outputs like a burn and inputs line a mint and therefore
		// not include them in the operations
		//
		// Additionally, it is possible to have outputs without any addresses
		// (e.g. https://testnet.avascan.info/blockchain/p/block/81016)
		//
		// therefore we skip parsing operations unless there is exactly 1 address
		if len(transferOutput.Addrs) != 1 {
			continue
		}

		outOp, err := t.buildOutputOperation(
			transferOutput,
			out.AssetID(),
			status,
			outOps.Len(),
			txID,
			uint32(outIndexOffset+outIndex),
			opType,
			metaType,
			chainIDAlias,
		)
		if err != nil {
			return err
		}

		outOps.Append(outOp, metaType)
	}

	return nil
}

func (t *TxParser) utxosToOperations(
	outOps *txOps,
	opType string,
	utxos []*avax.UTXO,
	metaType string,
	chainIDAlias constants.ChainIDAlias,
) error {
	status := types.String(mapper.StatusSuccess)
	if t.cfg.IsConstruction {
		status = nil
	}

	for _, utxo := range utxos {
		outIntf := utxo.Out
		if lockedOut, ok := outIntf.(*stakeable.LockOut); ok {
			outIntf = lockedOut.TransferableOut
		}

		out, ok := outIntf.(*secp256k1fx.TransferOutput)

		if !ok {
			return errUnknownOutputType
		}

		// Rosetta cannot handle multisig at the moment. In order to pass data validation,
		// we treat multisig outputs like a burn and inputs line a mint and therefore
		// not include them in the operations
		//
		// Additionally, it is possible to have outputs without any addresses
		// (e.g. https://testnet.avascan.info/blockchain/p/block/81016)
		//
		// therefore we skip parsing operations unless there is exactly 1 address
		if len(out.Addrs) != 1 {
			continue
		}

		outOp, err := t.buildOutputOperation(
			out,
			utxo.AssetID(),
			status,
			outOps.Len(),
			utxo.TxID,
			utxo.OutputIndex,
			opType,
			metaType,
			chainIDAlias,
		)
		if err != nil {
			return err
		}

		outOps.Append(outOp, metaType)
	}

	return nil
}

func (t *TxParser) buildOutputOperation(
	out *secp256k1fx.TransferOutput,
	assetID ids.ID,
	status *string,
	startIndex int,
	txID ids.ID,
	outIndex uint32,
	opType, metaType string,
	chainIDAlias constants.ChainIDAlias,
) (*types.Operation, error) {
	if len(out.Addrs) == 0 {
		return nil, errNoOutputAddresses
	}

	outAddrID := out.Addrs[0]
	outAddrFormat, err := address.Format(chainIDAlias.String(), t.cfg.Hrp, outAddrID[:])
	if err != nil {
		return nil, err
	}

	metadata := &OperationMetadata{
		Type:      metaType,
		Threshold: out.OutputOwners.Threshold,
		Locktime:  out.OutputOwners.Locktime,
	}

	opMetadata, err := mapper.MarshalJSONMap(metadata)
	if err != nil {
		return nil, err
	}

	outBigAmount := big.NewInt(int64(out.Amount()))

	utxoID := avax.UTXOID{TxID: txID, OutputIndex: outIndex}

	// Do not add coin change during construction as txid is not yet generated
	// and therefore UTXO ids would be incorrect
	var coinChange *types.CoinChange
	if !t.cfg.IsConstruction {
		coinChange = &types.CoinChange{
			CoinIdentifier: &types.CoinIdentifier{Identifier: utxoID.String()},
			CoinAction:     types.CoinCreated,
		}
	}

	amount, err := t.buildAmount(outBigAmount, assetID)
	if err != nil {
		return nil, err
	}

	return &types.Operation{
		Type: opType,
		OperationIdentifier: &types.OperationIdentifier{
			Index: int64(startIndex),
		},
		CoinChange: coinChange,
		Status:     status,
		Account:    &types.AccountIdentifier{Address: outAddrFormat},
		Amount:     amount,
		Metadata:   opMetadata,
	}, nil
}

func (t *TxParser) isMultisig(utxoid avax.UTXOID) (bool, error) {
	dependencyTx, ok := t.dependencyTxs[utxoid.TxID]
	if !ok {
		return false, errFailedToCheckMultisig
	}

	utxoMap := dependencyTx.GetUtxos()
	utxo, ok := utxoMap[utxoid]
	if !ok {
		return false, errFailedToCheckMultisig
	}

	addressable, ok := utxo.Out.(avax.Addressable)
	if !ok {
		return false, errFailedToCheckMultisig
	}
	isMultisig := len(addressable.Addresses()) != 1

	return isMultisig, nil
}

func (t *TxParser) lookupCurrency(assetID ids.ID) (*types.Currency, error) {
	asset, err := t.cfg.PChainClient.GetAssetDescription(context.Background(), assetID.String())
	if err != nil {
		return nil, fmt.Errorf("error while looking up currency: %w", err)
	}

	return &types.Currency{
		Symbol:   asset.Symbol,
		Decimals: int32(asset.Denomination),
	}, nil
}
