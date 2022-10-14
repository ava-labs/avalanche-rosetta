package pchain

import (
	"errors"
	"log"
	"math/big"

	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/avalanchego/utils/formatting/address"
	"github.com/ava-labs/avalanchego/vms/components/avax"
	"github.com/ava-labs/avalanchego/vms/platformvm/stakeable"
	"github.com/ava-labs/avalanchego/vms/platformvm/txs"
	"github.com/ava-labs/avalanchego/vms/platformvm/validator"
	"github.com/ava-labs/avalanchego/vms/secp256k1fx"
	"github.com/coinbase/rosetta-sdk-go/types"

	"github.com/ava-labs/avalanche-rosetta/mapper"
)

var (
	errNilChainIDs                    = errors.New("chain ids cannot be nil")
	errNilInputTxAccounts             = errors.New("input tx accounts cannot be nil")
	errUnknownDestinationChain        = errors.New("unknown destination chain")
	errNoDependencyTxs                = errors.New("no dependency txs provided")
	errNoMatchingRewardOutputs        = errors.New("no matching reward outputs")
	errNoMatchingInputAddresses       = errors.New("no matching input addresses")
	errNoOutputAddresses              = errors.New("no output addresses")
	errFailedToGetUTXOAddresses       = errors.New("failed to get utxo addresses")
	errFailedToCheckMultisig          = errors.New("failed to check utxo for multisig")
	errOutputTypeAssertion            = errors.New("output type assertion failed")
	errUnknownRewardSourceTransaction = errors.New("unknown source tx type for reward tx")
)

// TxParser parses P-chain transactions and generate corresponding Rosetta operations
type TxParser struct {
	// isConstruction indicates if parsing is done as part of construction or /block endpoints
	isConstruction bool
	// hrp used for address formatting
	hrp string
	// chainIDs contain chain id to chain id alias mappings
	chainIDs map[string]string
	// dependencyTxs contain transaction id to dependence transaction mapping
	dependencyTxs map[string]*DependencyTx
	// inputTxAccounts contain utxo id to account identifier mappings
	inputTxAccounts map[string]*types.AccountIdentifier
}

// NewTxParser returns a new transaction parser
func NewTxParser(
	isConstruction bool,
	hrp string,
	chainIDs map[string]string,
	inputTxAccounts map[string]*types.AccountIdentifier,
	dependencyTxs map[string]*DependencyTx,
) (*TxParser, error) {
	if chainIDs == nil {
		return nil, errNilChainIDs
	}

	if inputTxAccounts == nil {
		return nil, errNilInputTxAccounts
	}

	return &TxParser{
		isConstruction:  isConstruction,
		hrp:             hrp,
		chainIDs:        chainIDs,
		inputTxAccounts: inputTxAccounts,
		dependencyTxs:   dependencyTxs,
	}, nil
}

// Parse converts the given unsigned P-chain tx to corresponding Rosetta Transaction
func (t *TxParser) Parse(txID ids.ID, tx txs.UnsignedTx) (*types.Transaction, error) {
	var (
		ops    *txOps
		txType string
		err    error
	)

	switch unsignedTx := tx.(type) {
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

	chainID := tx.DestinationChain.String()
	chainIDAlias, ok := t.chainIDs[chainID]
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
	ops := newTxOps(t.isConstruction)

	err := t.insToOperations(ops, OpImportAvax, tx.Ins, OpTypeInput)
	if err != nil {
		return nil, err
	}

	err = t.insToOperations(ops, OpImportAvax, tx.ImportedInputs, OpTypeImport)
	if err != nil {
		return nil, err
	}

	err = t.outsToOperations(ops, OpImportAvax, txID, tx.Outs, OpTypeOutput, mapper.PChainNetworkIdentifier)
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

	err = t.outsToOperations(ops, OpAddValidator, txID, tx.Stake(), OpTypeStakeOutput, mapper.PChainNetworkIdentifier)
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

	err = t.outsToOperations(ops, OpAddPermissionlessValidator, txID, tx.Stake(), OpTypeStakeOutput, mapper.PChainNetworkIdentifier)
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

	err = t.outsToOperations(ops, OpAddDelegator, txID, tx.Stake(), OpTypeStakeOutput, mapper.PChainNetworkIdentifier)
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

	err = t.outsToOperations(ops, OpAddPermissionlessDelegator, txID, tx.Stake(), OpTypeStakeOutput, mapper.PChainNetworkIdentifier)
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
	rewardOuts := t.dependencyTxs[stakingTxID.String()]
	if rewardOuts == nil {
		return nil, errNoMatchingRewardOutputs
	}
	ops := newTxOps(t.isConstruction)
	err := t.utxosToOperations(ops, OpRewardValidator, rewardOuts.RewardUTXOs, OpTypeReward, mapper.PChainNetworkIdentifier)
	if err != nil {
		return nil, err
	}

	var v *validator.Validator
	switch utx := rewardOuts.Tx.Unsigned.(type) {
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

func addMetadataToStakeOuts(ops *txOps, validator *validator.Validator) {
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
	ops := newTxOps(t.isConstruction)

	err := t.insToOperations(ops, txType, tx.Ins, OpTypeInput)
	if err != nil {
		return nil, err
	}

	err = t.outsToOperations(ops, txType, txID, tx.Outs, OpTypeOutput, mapper.PChainNetworkIdentifier)
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
	if t.isConstruction {
		status = nil
	}

	for _, in := range txIns {
		metadata := &OperationMetadata{
			Type: metaType,
		}

		if transferInput, ok := in.In.(*secp256k1fx.TransferInput); ok {
			metadata.SigIndices = transferInput.SigIndices
		}

		opMetadata, err := mapper.MarshalJSONMap(metadata)
		if err != nil {
			return err
		}

		utxoID := in.UTXOID.String()

		var account *types.AccountIdentifier

		// Check if the dependency is not multisig and extract account id from it
		// for non-imported inputs or when tx is being constructed
		if t.isConstruction || metaType != OpTypeImport {
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
			account, ok = t.inputTxAccounts[utxoID]
			if !ok {
				return errNoMatchingInputAddresses
			}
		}

		inputAmount := new(big.Int).SetUint64(in.In.Amount())
		inOp := &types.Operation{
			OperationIdentifier: &types.OperationIdentifier{
				Index: int64(inOps.Len()),
			},
			Type:    opType,
			Status:  status,
			Account: account,
			// Negating input amount
			Amount: mapper.AtomicAvaxAmount(new(big.Int).Neg(inputAmount)),
			CoinChange: &types.CoinChange{
				CoinIdentifier: &types.CoinIdentifier{
					Identifier: utxoID,
				},
				CoinAction: types.CoinSpent,
			},
			Metadata: opMetadata,
		}

		inOps.Append(inOp, metaType)
	}
	return nil
}

func (t *TxParser) outsToOperations(
	outOps *txOps,
	opType string,
	txID ids.ID,
	txOut []*avax.TransferableOutput,
	metaType string,
	chainIDAlias string,
) error {
	outIndexOffset := outOps.OutputLen()
	status := types.String(mapper.StatusSuccess)
	if t.isConstruction {
		status = nil
	}

	for outIndex, out := range txOut {
		transferOut := out.Out

		if lockOut, ok := transferOut.(*stakeable.LockOut); ok {
			transferOut = lockOut.TransferableOut
		}

		transferOutput, ok := transferOut.(*secp256k1fx.TransferOutput)
		if !ok {
			return errOutputTypeAssertion
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
	chainIDAlias string,
) error {
	status := types.String(mapper.StatusSuccess)
	if t.isConstruction {
		status = nil
	}

	for _, utxo := range utxos {
		outIntf := utxo.Out
		if lockedOut, ok := outIntf.(*stakeable.LockOut); ok {
			outIntf = lockedOut.TransferableOut
		}

		out, ok := outIntf.(*secp256k1fx.TransferOutput)

		if !ok {
			return errOutputTypeAssertion
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
	status *string,
	startIndex int,
	txID ids.ID,
	outIndex uint32,
	opType, metaType, chainIDAlias string,
) (*types.Operation, error) {
	if len(out.Addrs) == 0 {
		return nil, errNoOutputAddresses
	}

	outAddrID := out.Addrs[0]
	outAddrFormat, err := address.Format(chainIDAlias, t.hrp, outAddrID[:])
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
	if !t.isConstruction {
		coinChange = &types.CoinChange{
			CoinIdentifier: &types.CoinIdentifier{Identifier: utxoID.String()},
			CoinAction:     types.CoinCreated,
		}
	}

	return &types.Operation{
		Type: opType,
		OperationIdentifier: &types.OperationIdentifier{
			Index: int64(startIndex),
		},
		CoinChange: coinChange,
		Status:     status,
		Account:    &types.AccountIdentifier{Address: outAddrFormat},
		Amount:     mapper.AtomicAvaxAmount(outBigAmount),
		Metadata:   opMetadata,
	}, nil
}

func (t *TxParser) isMultisig(utxoid avax.UTXOID) (bool, error) {
	dependencyTx, ok := t.dependencyTxs[utxoid.TxID.String()]
	if !ok {
		return false, errFailedToCheckMultisig
	}

	utxoMap := getUTXOMap(dependencyTx)
	utxo, ok := utxoMap[utxoid.OutputIndex]
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

// GetAccountsFromUTXOs extracts destination accounts from given dependency transactions
func GetAccountsFromUTXOs(hrp string, dependencyTxs map[string]*DependencyTx) (map[string]*types.AccountIdentifier, error) {
	addresses := make(map[string]*types.AccountIdentifier)
	for _, dependencyTx := range dependencyTxs {
		utxoMap := getUTXOMap(dependencyTx)

		for _, utxo := range utxoMap {
			addressable, ok := utxo.Out.(avax.Addressable)
			if !ok {
				return nil, errFailedToGetUTXOAddresses
			}

			addrs := addressable.Addresses()

			if len(addrs) != 1 {
				continue
			}

			addr, err := address.Format(mapper.PChainNetworkIdentifier, hrp, addrs[0])
			addresses[utxo.UTXOID.String()] = &types.AccountIdentifier{Address: addr}
			if err != nil {
				return nil, err
			}
		}
	}

	return addresses, nil
}

// GetDependencyTxIDs generates the list of transaction ids used in the inputs to given unsigned transaction
// this list is then used to fetch the dependency transactions in order to extract source addresses
// as this information is not part of the transaction objects on chain.
func GetDependencyTxIDs(tx txs.UnsignedTx) ([]ids.ID, error) {
	var txIds []ids.ID
	switch unsignedTx := tx.(type) {
	case *txs.ExportTx:
		txIds = append(txIds, getUniqueTxIds(unsignedTx.Ins)...)
	case *txs.ImportTx:
		txIds = append(txIds, getUniqueTxIds(unsignedTx.Ins)...)
	case *txs.AddValidatorTx:
		txIds = append(txIds, getUniqueTxIds(unsignedTx.Ins)...)
	case *txs.AddPermissionlessValidatorTx:
		txIds = append(txIds, getUniqueTxIds(unsignedTx.Ins)...)
	case *txs.AddDelegatorTx:
		txIds = append(txIds, getUniqueTxIds(unsignedTx.Ins)...)
	case *txs.AddPermissionlessDelegatorTx:
		txIds = append(txIds, getUniqueTxIds(unsignedTx.Ins)...)
	case *txs.CreateSubnetTx:
		txIds = append(txIds, getUniqueTxIds(unsignedTx.Ins)...)
	case *txs.CreateChainTx:
		txIds = append(txIds, getUniqueTxIds(unsignedTx.Ins)...)
	case *txs.AddSubnetValidatorTx:
		txIds = append(txIds, getUniqueTxIds(unsignedTx.Ins)...)
	case *txs.TransformSubnetTx:
		txIds = append(txIds, getUniqueTxIds(unsignedTx.Ins)...)
	case *txs.RemoveSubnetValidatorTx:
		txIds = append(txIds, getUniqueTxIds(unsignedTx.Ins)...)
	case *txs.RewardValidatorTx:
		txIds = append(txIds, unsignedTx.TxID)
	case *txs.AdvanceTimeTx:
		// advance time txs do not have inputs
	default:
		log.Printf("unknown type %T", unsignedTx)
	}

	ids.SortIDs(txIds)

	return txIds, nil
}

func getUniqueTxIds(ins []*avax.TransferableInput) []ids.ID {
	txnIDs := make(map[string]ids.ID)
	for _, in := range ins {
		txnIDs[in.UTXOID.TxID.String()] = in.UTXOID.TxID
	}

	uniqueTxnIDs := make([]ids.ID, 0, len(txnIDs))
	for _, txnID := range txnIDs {
		uniqueTxnIDs = append(uniqueTxnIDs, txnID)
	}
	return uniqueTxnIDs
}

func getUTXOMap(d *DependencyTx) map[uint32]*avax.UTXO {
	utxos := make(map[uint32]*avax.UTXO)

	if d.Tx != nil {
		// Generate UTXOs from outputs
		switch unsignedTx := d.Tx.Unsigned.(type) {
		case *txs.ExportTx:
			mapUTXOs(d.Tx.ID(), unsignedTx.Outs, utxos)
		case *txs.ImportTx:
			mapUTXOs(d.Tx.ID(), unsignedTx.Outs, utxos)
		case *txs.AddValidatorTx:
			mapUTXOs(d.Tx.ID(), unsignedTx.Outs, utxos)
			mapUTXOs(d.Tx.ID(), unsignedTx.Stake(), utxos)
		case *txs.AddPermissionlessValidatorTx:
			mapUTXOs(d.Tx.ID(), unsignedTx.Outs, utxos)
			mapUTXOs(d.Tx.ID(), unsignedTx.Stake(), utxos)
		case *txs.AddDelegatorTx:
			mapUTXOs(d.Tx.ID(), unsignedTx.Outs, utxos)
			mapUTXOs(d.Tx.ID(), unsignedTx.Stake(), utxos)
		case *txs.AddPermissionlessDelegatorTx:
			mapUTXOs(d.Tx.ID(), unsignedTx.Outs, utxos)
			mapUTXOs(d.Tx.ID(), unsignedTx.Stake(), utxos)
		case *txs.CreateSubnetTx:
			mapUTXOs(d.Tx.ID(), unsignedTx.Outs, utxos)
		case *txs.AddSubnetValidatorTx:
			mapUTXOs(d.Tx.ID(), unsignedTx.Outs, utxos)
		case *txs.TransformSubnetTx:
			mapUTXOs(d.Tx.ID(), unsignedTx.Outs, utxos)
		case *txs.RemoveSubnetValidatorTx:
			mapUTXOs(d.Tx.ID(), unsignedTx.Outs, utxos)
		case *txs.CreateChainTx:
			mapUTXOs(d.Tx.ID(), unsignedTx.Outs, utxos)
		default:
			// no utxos extracted from unsupported transaction types
		}
	}

	// Add reward UTXOs
	for _, utxo := range d.RewardUTXOs {
		utxos[utxo.OutputIndex] = utxo
	}

	return utxos
}

func mapUTXOs(txID ids.ID, outs []*avax.TransferableOutput, utxos map[uint32]*avax.UTXO) {
	outIndexOffset := uint32(len(utxos))
	for i, out := range outs {
		outIndex := outIndexOffset + uint32(i)
		utxos[outIndex] = &avax.UTXO{
			UTXOID: avax.UTXOID{
				TxID:        txID,
				OutputIndex: outIndex,
			},
			Asset: out.Asset,
			Out:   out.Out,
		}
	}
}

type txOps struct {
	isConstruction bool
	Ins            []*types.Operation
	Outs           []*types.Operation
	StakeOuts      []*types.Operation
	ImportIns      []*types.Operation
	ExportOuts     []*types.Operation
}

func newTxOps(isConstruction bool) *txOps {
	return &txOps{isConstruction: isConstruction}
}

func (t *txOps) IncludedOperations() []*types.Operation {
	ops := []*types.Operation{}
	ops = append(ops, t.Ins...)
	ops = append(ops, t.Outs...)
	ops = append(ops, t.StakeOuts...)
	return ops
}

// Used to populate operation identifier
func (t *txOps) Len() int {
	return len(t.Ins) + len(t.Outs) + len(t.StakeOuts)
}

// Used to populate coin identifier
func (t *txOps) OutputLen() int {
	return len(t.Outs) + len(t.StakeOuts)
}

func (t *txOps) Append(op *types.Operation, metaType string) {
	switch metaType {
	case OpTypeImport:
		if t.isConstruction {
			t.Ins = append(t.Ins, op)
		} else {
			// removing operation identifier as these will be skipped in the final operations list
			op.OperationIdentifier = nil
			t.ImportIns = append(t.ImportIns, op)
		}
	case OpTypeExport:
		if t.isConstruction {
			t.Outs = append(t.Outs, op)
		} else {
			// removing operation identifier as these will be skipped in the final operations list
			op.OperationIdentifier = nil
			t.ExportOuts = append(t.ExportOuts, op)
		}
	case OpTypeStakeOutput, OpTypeReward:
		t.StakeOuts = append(t.StakeOuts, op)
	case OpTypeOutput:
		t.Outs = append(t.Outs, op)
	case OpTypeInput:
		t.Ins = append(t.Ins, op)
	}
}
