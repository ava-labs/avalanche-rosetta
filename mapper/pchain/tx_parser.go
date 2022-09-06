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
	pChainValidator "github.com/ava-labs/avalanchego/vms/platformvm/validator"
	"github.com/ava-labs/avalanchego/vms/secp256k1fx"
	"github.com/coinbase/rosetta-sdk-go/types"

	"github.com/ava-labs/avalanche-rosetta/mapper"
)

var (
	errUnknownDestinationChain  = errors.New("unknown destination chain")
	errNoDependencyTxs          = errors.New("no dependency txs provided")
	errNoMatchingRewardOutputs  = errors.New("no matching reward outputs")
	errNoMatchingInputAddresses = errors.New("no matching input addresses")
	errNoOutputAddresses        = errors.New("no output addresses")
	errFailedToGetUTXOAddresses = errors.New("failed to get utxo addresses")
	errFailedToCheckMultisig    = errors.New("failed to check utxo for multisig")
	errOutputTypeAssertion      = errors.New("output type assertion failed")
)

type OperationFilter func(operation *types.Operation) (bool, error)

type TxParser struct {
	isConstruction  bool
	hrp             string
	chainIDs        map[string]string
	dependencyTxs   map[string]*DependencyTx
	inputTxAccounts map[string]*types.AccountIdentifier
}

func NewTxParser(
	isConstruction bool,
	hrp string,
	chainIDs map[string]string,
	inputTxAccounts map[string]*types.AccountIdentifier,
	dependencyTxs map[string]*DependencyTx,
) *TxParser {
	if inputTxAccounts == nil {
		inputTxAccounts = make(map[string]*types.AccountIdentifier)
	}

	return &TxParser{
		isConstruction:  isConstruction,
		hrp:             hrp,
		chainIDs:        chainIDs,
		inputTxAccounts: inputTxAccounts,
		dependencyTxs:   dependencyTxs,
	}
}

func (t *TxParser) Parse(txID ids.ID, tx txs.UnsignedTx) (*types.Transaction, error) {
	var ops []*types.Operation
	var txType string
	var err error
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
	case *txs.AdvanceTimeTx:
		txType = OpAdvanceTime
		// no op tx
	default:
		log.Printf("unknown type %T", unsignedTx)
	}
	if err != nil {
		return nil, err
	}

	return &types.Transaction{
		TransactionIdentifier: &types.TransactionIdentifier{
			Hash: txID.String(),
		},
		Operations: ops,
		Metadata: map[string]interface{}{
			MetadataTxType: txType,
		},
	}, nil
}

func (t *TxParser) parseExportTx(txID ids.ID, tx *txs.ExportTx) ([]*types.Operation, error) {
	ops, err := t.baseTxToCombinedOperations(txID, &tx.BaseTx, OpExportAvax)
	if err != nil {
		return nil, err
	}

	chainID := tx.DestinationChain.String()
	chainIDAlias, ok := t.chainIDs[chainID]
	if !ok {
		return nil, errUnknownDestinationChain
	}

	exportedOuts, err := t.outsToOperations(len(ops), len(tx.Outs), OpExportAvax, txID, tx.ExportedOutputs, OpTypeExport, chainIDAlias)
	if err != nil {
		return nil, err
	}
	ops = append(ops, exportedOuts...)

	return ops, nil
}

func (t *TxParser) parseImportTx(txID ids.ID, tx *txs.ImportTx) ([]*types.Operation, error) {
	ops := []*types.Operation{}

	ins, err := t.insToOperations(0, OpImportAvax, tx.Ins, OpTypeInput)
	if err != nil {
		return nil, err
	}

	ops = append(ops, ins...)
	importedIns, err := t.insToOperations(len(ops), OpImportAvax, tx.ImportedInputs, OpTypeImport)
	if err != nil {
		return nil, err
	}

	ops = append(ops, importedIns...)
	outs, err := t.outsToOperations(len(ops), 0, OpImportAvax, txID, tx.Outs, OpTypeOutput, mapper.PChainNetworkIdentifier)
	if err != nil {
		return nil, err
	}
	ops = append(ops, outs...)

	return ops, nil
}

func (t *TxParser) parseAddValidatorTx(txID ids.ID, tx *txs.AddValidatorTx) ([]*types.Operation, error) {
	ops, err := t.baseTxToCombinedOperations(txID, &tx.BaseTx, OpAddValidator)
	if err != nil {
		return nil, err
	}

	stakeOuts, err := t.outsToOperations(len(ops), len(tx.Outs), OpAddValidator, txID, tx.Stake(), OpTypeStakeOutput, mapper.PChainNetworkIdentifier)
	if err != nil {
		return nil, err
	}
	addStakingMetadataToOperations(stakeOuts, &tx.Validator)

	ops = append(ops, stakeOuts...)

	return ops, nil
}

func (t *TxParser) parseAddDelegatorTx(txID ids.ID, tx *txs.AddDelegatorTx) ([]*types.Operation, error) {
	ops, err := t.baseTxToCombinedOperations(txID, &tx.BaseTx, OpAddDelegator)
	if err != nil {
		return nil, err
	}

	stakeOuts, err := t.outsToOperations(len(ops), len(tx.Outs), OpAddDelegator, txID, tx.Stake(), OpTypeStakeOutput, mapper.PChainNetworkIdentifier)
	if err != nil {
		return nil, err
	}
	addStakingMetadataToOperations(stakeOuts, &tx.Validator)

	ops = append(ops, stakeOuts...)

	return ops, nil
}

func (t *TxParser) parseRewardValidatorTx(tx *txs.RewardValidatorTx) ([]*types.Operation, error) {
	ops := []*types.Operation{}
	stakingTxID := tx.TxID

	if t.dependencyTxs == nil {
		return nil, errNoDependencyTxs
	}
	rewardOuts := t.dependencyTxs[stakingTxID.String()]
	if rewardOuts == nil {
		return nil, errNoMatchingRewardOutputs
	}
	outs, err := t.utxosToOperations(0, OpRewardValidator, rewardOuts.RewardUTXOs, OpTypeReward, mapper.PChainNetworkIdentifier)
	if err != nil {
		return nil, err
	}

	var validator *pChainValidator.Validator
	switch utx := rewardOuts.Tx.Unsigned.(type) {
	case *txs.AddValidatorTx:
		validator = &utx.Validator
	case *txs.AddDelegatorTx:
		validator = &utx.Validator
	}

	addStakingMetadataToOperations(outs, validator)

	ops = append(ops, outs...)

	return ops, nil
}

func addStakingMetadataToOperations(outs []*types.Operation, validator *pChainValidator.Validator) {
	if validator != nil {
		for _, out := range outs {
			out.Metadata[MetadataValidatorNodeID] = validator.NodeID.String()
			out.Metadata[MetadataStakingStartTime] = validator.Start
			out.Metadata[MetadataStakingEndTime] = validator.End
		}
	}
}

func (t *TxParser) parseCreateSubnetTx(txID ids.ID, tx *txs.CreateSubnetTx) ([]*types.Operation, error) {
	return t.baseTxToCombinedOperations(txID, &tx.BaseTx, OpCreateSubnet)
}

func (t *TxParser) parseAddSubnetValidatorTx(txID ids.ID, tx *txs.AddSubnetValidatorTx) ([]*types.Operation, error) {
	return t.baseTxToCombinedOperations(txID, &tx.BaseTx, OpAddSubnetValidator)
}

func (t *TxParser) parseCreateChainTx(txID ids.ID, tx *txs.CreateChainTx) ([]*types.Operation, error) {
	return t.baseTxToCombinedOperations(txID, &tx.BaseTx, OpCreateChain)
}

func (t *TxParser) baseTxToCombinedOperations(txID ids.ID, tx *txs.BaseTx, txType string) ([]*types.Operation, error) {
	ops := []*types.Operation{}

	ins, err := t.insToOperations(0, txType, tx.Ins, OpTypeInput)
	if err != nil {
		return nil, err
	}

	outs, err := t.outsToOperations(len(ins), 0, txType, txID, tx.Outs, OpTypeOutput, mapper.PChainNetworkIdentifier)
	if err != nil {
		return nil, err
	}

	ops = append(ops, ins...)
	ops = append(ops, outs...)

	return ops, nil
}

func (t *TxParser) shouldSkipOperation(metaType string) bool {
	// Do not skip any operation for construction parse
	if t.isConstruction {
		return false
	}

	switch metaType {
	case OpTypeImport,
		OpTypeExport,
		OpTypeCreateChain:
		// ignore import, export and create-chain operations
		return true
	default:
		return false
	}
}

func (t *TxParser) insToOperations(
	startIndex int,
	opType string,
	txIns []*avax.TransferableInput,
	metaType string,
) ([]*types.Operation, error) {
	ins := make([]*types.Operation, 0)

	if t.shouldSkipOperation(metaType) {
		return ins, nil
	}

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
			return nil, err
		}

		// If dependency txs are provided, which is the case for /block endpoints
		// check whether the input UTXO is multisig. If so, skip it.
		if t.dependencyTxs != nil {
			isMultisig, err := t.isMultisig(in.UTXOID)
			if err != nil {
				return nil, errFailedToCheckMultisig
			}
			if isMultisig {
				continue
			}
		}

		utxoID := in.UTXOID.String()
		account, ok := t.inputTxAccounts[utxoID]
		if !ok {
			return nil, errNoMatchingInputAddresses
		}

		inputAmount := new(big.Int).SetUint64(in.In.Amount())
		inOp := &types.Operation{
			OperationIdentifier: &types.OperationIdentifier{
				Index: int64(startIndex),
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

		ins = append(ins, inOp)
		startIndex++
	}
	return ins, nil
}

func (t *TxParser) outsToOperations(
	startIndex int,
	outIndexOffset int,
	opType string,
	txID ids.ID,
	txOut []*avax.TransferableOutput,
	metaType string,
	chainIDAlias string,
) ([]*types.Operation, error) {
	outs := []*types.Operation{}

	if t.shouldSkipOperation(metaType) {
		return outs, nil
	}

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
			return nil, errOutputTypeAssertion
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
			startIndex,
			txID,
			uint32(outIndexOffset+outIndex),
			opType,
			metaType,
			chainIDAlias,
		)
		if err != nil {
			return nil, err
		}

		outs = append(outs, outOp)
		startIndex++
	}

	return outs, nil
}

func (t *TxParser) utxosToOperations(
	startIndex int,
	opType string,
	utxos []*avax.UTXO,
	metaType string,
	chainIDAlias string,
) ([]*types.Operation, error) {
	outs := []*types.Operation{}

	if t.shouldSkipOperation(metaType) {
		return outs, nil
	}

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
			return nil, errOutputTypeAssertion
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
			startIndex,
			utxo.TxID,
			utxo.OutputIndex,
			opType,
			metaType,
			chainIDAlias,
		)
		if err != nil {
			return nil, err
		}

		outs = append(outs, outOp)
		startIndex++
	}

	return outs, nil
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

func GetDependencyTxIDs(tx txs.UnsignedTx) ([]ids.ID, error) {
	var txIds []ids.ID
	switch unsignedTx := tx.(type) {
	case *txs.ExportTx:
		txIds = append(txIds, getUniqueTxIds(unsignedTx.Ins)...)
	case *txs.ImportTx:
		txIds = append(txIds, getUniqueTxIds(unsignedTx.Ins)...)
	case *txs.AddValidatorTx:
		txIds = append(txIds, getUniqueTxIds(unsignedTx.Ins)...)
	case *txs.AddDelegatorTx:
		txIds = append(txIds, getUniqueTxIds(unsignedTx.Ins)...)
	case *txs.CreateSubnetTx:
		txIds = append(txIds, getUniqueTxIds(unsignedTx.Ins)...)
	case *txs.CreateChainTx:
		txIds = append(txIds, getUniqueTxIds(unsignedTx.Ins)...)
	case *txs.AddSubnetValidatorTx:
		txIds = append(txIds, getUniqueTxIds(unsignedTx.Ins)...)
	case *txs.RewardValidatorTx:
		txIds = append(txIds, unsignedTx.TxID)
	}

	ids.SortIDs(txIds)

	return txIds, nil
}

func getUniqueTxIds(ins []*avax.TransferableInput) []ids.ID {
	txnIDs := make(map[string]ids.ID)
	for _, in := range ins {
		txnIDs[in.UTXOID.TxID.String()] = in.UTXOID.TxID
	}

	uniqueTxnIDs := []ids.ID{}
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
		case *txs.AddDelegatorTx:
			mapUTXOs(d.Tx.ID(), unsignedTx.Outs, utxos)
			mapUTXOs(d.Tx.ID(), unsignedTx.Stake(), utxos)
		case *txs.CreateSubnetTx:
			mapUTXOs(d.Tx.ID(), unsignedTx.Outs, utxos)
		case *txs.CreateChainTx:
			mapUTXOs(d.Tx.ID(), unsignedTx.Outs, utxos)
		case *txs.AddSubnetValidatorTx:
			mapUTXOs(d.Tx.ID(), unsignedTx.Outs, utxos)
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
