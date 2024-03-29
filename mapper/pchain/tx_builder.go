package pchain

import (
	"errors"
	"fmt"

	"github.com/ava-labs/avalanchego/codec"
	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/avalanchego/utils"
	"github.com/ava-labs/avalanchego/utils/constants"
	"github.com/ava-labs/avalanchego/utils/formatting"
	"github.com/ava-labs/avalanchego/utils/formatting/address"
	"github.com/ava-labs/avalanchego/vms/components/avax"
	"github.com/ava-labs/avalanchego/vms/platformvm/signer"
	"github.com/ava-labs/avalanchego/vms/platformvm/txs"
	"github.com/ava-labs/avalanchego/vms/secp256k1fx"
	"github.com/coinbase/rosetta-sdk-go/parser"
	"github.com/coinbase/rosetta-sdk-go/types"
	"github.com/ethereum/go-ethereum/common/math"

	"github.com/ava-labs/avalanche-rosetta/mapper"
)

var (
	errInvalidMetadata      = errors.New("invalid metadata")
	errOutputAmountOverflow = errors.New("sum of output amounts caused overflow")
)

// BuildTx constructs a P-chain Tx based on the provided operation type, Rosetta matches and metadata
// This method is only used during construction.
func BuildTx(
	opType string,
	matches []*parser.Match,
	payloadMetadata Metadata,
	codec codec.Manager,
	avaxAssetID ids.ID,
) (*txs.Tx, []*types.AccountIdentifier, error) {
	switch opType {
	case OpImportAvax:
		return buildImportTx(matches, payloadMetadata, codec, avaxAssetID)
	case OpExportAvax:
		return buildExportTx(matches, payloadMetadata, codec, avaxAssetID)
	case OpAddPermissionlessValidator:
		return buildAddPermissionlessValidatorTx(matches, payloadMetadata, codec, avaxAssetID)
	case OpAddPermissionlessDelegator:
		return buildAddPermissionlessDelegatorTx(matches, payloadMetadata, codec, avaxAssetID)
	case OpAddValidator:
		// TODO: Remove Post-Durango
		return buildAddValidatorTx(matches, payloadMetadata, codec, avaxAssetID)
	case OpAddDelegator:
		// TODO: Remove Post-Durango
		return buildAddDelegatorTx(matches, payloadMetadata, codec, avaxAssetID)
	default:
		return nil, nil, fmt.Errorf("invalid tx type: %s", opType)
	}
}

// [buildImportTx] returns a duly initialized tx if it does not err
func buildImportTx(
	matches []*parser.Match,
	metadata Metadata,
	codec codec.Manager,
	avaxAssetID ids.ID,
) (*txs.Tx, []*types.AccountIdentifier, error) {
	blockchainID := metadata.BlockchainID
	sourceChainID := metadata.SourceChainID

	ins, imported, signers, err := buildInputs(matches[0].Operations, avaxAssetID)
	if err != nil {
		return nil, nil, fmt.Errorf("parse inputs failed: %w", err)
	}

	outs, _, _, err := buildOutputs(matches[1].Operations, codec, avaxAssetID)
	if err != nil {
		return nil, nil, fmt.Errorf("parse outputs failed: %w", err)
	}

	tx := &txs.Tx{Unsigned: &txs.ImportTx{
		BaseTx: txs.BaseTx{BaseTx: avax.BaseTx{
			NetworkID:    metadata.NetworkID,
			BlockchainID: blockchainID,
			Outs:         outs,
			Ins:          ins,
		}},
		ImportedInputs: imported,
		SourceChain:    sourceChainID,
	}}

	return tx, signers, tx.Sign(codec, nil)
}

// [buildExportTx] returns a duly initialized tx if it does not err
func buildExportTx(
	matches []*parser.Match,
	metadata Metadata,
	codec codec.Manager,
	avaxAssetID ids.ID,
) (*txs.Tx, []*types.AccountIdentifier, error) {
	if metadata.ExportMetadata == nil {
		return nil, nil, errInvalidMetadata
	}
	blockchainID := metadata.BlockchainID
	destinationChainID := metadata.DestinationChainID

	ins, _, signers, err := buildInputs(matches[0].Operations, avaxAssetID)
	if err != nil {
		return nil, nil, fmt.Errorf("parse inputs failed: %w", err)
	}

	outs, _, exported, err := buildOutputs(matches[1].Operations, codec, avaxAssetID)
	if err != nil {
		return nil, nil, fmt.Errorf("parse outputs failed: %w", err)
	}

	tx := &txs.Tx{Unsigned: &txs.ExportTx{
		BaseTx: txs.BaseTx{BaseTx: avax.BaseTx{
			NetworkID:    metadata.NetworkID,
			BlockchainID: blockchainID,
			Outs:         outs,
			Ins:          ins,
		}},
		DestinationChain: destinationChainID,
		ExportedOutputs:  exported,
	}}

	return tx, signers, tx.Sign(codec, nil)
}

// TODO: Remove Post-Durango
// [buildAddValidatorTx] returns a duly initialized tx if it does not err
func buildAddValidatorTx(
	matches []*parser.Match,
	metadata Metadata,
	codec codec.Manager,
	avaxAssetID ids.ID,
) (*txs.Tx, []*types.AccountIdentifier, error) {
	if metadata.StakingMetadata == nil {
		return nil, nil, errInvalidMetadata
	}

	blockchainID := metadata.BlockchainID

	nodeID, err := ids.NodeIDFromString(metadata.NodeID)
	if err != nil {
		return nil, nil, err
	}

	rewardsOwner, err := buildOutputOwner(
		metadata.ValidationRewardsOwners,
		metadata.Locktime,
		metadata.Threshold,
	)
	if err != nil {
		return nil, nil, err
	}

	ins, _, signers, err := buildInputs(matches[0].Operations, avaxAssetID)
	if err != nil {
		return nil, nil, fmt.Errorf("parse inputs failed: %w", err)
	}

	outs, stakeOutputs, _, err := buildOutputs(matches[1].Operations, codec, avaxAssetID)
	if err != nil {
		return nil, nil, fmt.Errorf("parse outputs failed: %w", err)
	}

	weight, err := sumOutputAmounts(stakeOutputs)
	if err != nil {
		return nil, nil, err
	}

	tx := &txs.Tx{Unsigned: &txs.AddValidatorTx{
		BaseTx: txs.BaseTx{BaseTx: avax.BaseTx{
			NetworkID:    metadata.NetworkID,
			BlockchainID: blockchainID,
			Outs:         outs,
			Ins:          ins,
		}},
		Validator: txs.Validator{
			NodeID: nodeID,
			Start:  metadata.Start,
			End:    metadata.End,
			Wght:   weight,
		},
		StakeOuts:        stakeOutputs,
		RewardsOwner:     rewardsOwner,
		DelegationShares: metadata.Shares,
	}}

	return tx, signers, tx.Sign(codec, nil)
}

// TODO: Remove Post-Durango
// [buildAddDelegatorTx] returns a duly initialized tx if it does not err
func buildAddDelegatorTx(
	matches []*parser.Match,
	metadata Metadata,
	codec codec.Manager,
	avaxAssetID ids.ID,
) (*txs.Tx, []*types.AccountIdentifier, error) {
	if metadata.StakingMetadata == nil {
		return nil, nil, errInvalidMetadata
	}

	blockchainID := metadata.BlockchainID

	nodeID, err := ids.NodeIDFromString(metadata.NodeID)
	if err != nil {
		return nil, nil, err
	}

	rewardsOwner, err := buildOutputOwner(
		metadata.ValidationRewardsOwners,
		metadata.Locktime,
		metadata.Threshold,
	)
	if err != nil {
		return nil, nil, err
	}

	ins, _, signers, err := buildInputs(matches[0].Operations, avaxAssetID)
	if err != nil {
		return nil, nil, fmt.Errorf("parse inputs failed: %w", err)
	}

	outs, stakeOutputs, _, err := buildOutputs(matches[1].Operations, codec, avaxAssetID)
	if err != nil {
		return nil, nil, fmt.Errorf("parse outputs failed: %w", err)
	}

	weight, err := sumOutputAmounts(stakeOutputs)
	if err != nil {
		return nil, nil, err
	}

	tx := &txs.Tx{Unsigned: &txs.AddDelegatorTx{
		BaseTx: txs.BaseTx{BaseTx: avax.BaseTx{
			NetworkID:    metadata.NetworkID,
			BlockchainID: blockchainID,
			Outs:         outs,
			Ins:          ins,
		}},
		Validator: txs.Validator{
			NodeID: nodeID,
			Start:  metadata.Start,
			End:    metadata.End,
			Wght:   weight,
		},
		StakeOuts:              stakeOutputs,
		DelegationRewardsOwner: rewardsOwner,
	}}

	return tx, signers, tx.Sign(codec, nil)
}

// [buildAddPermissionlessValidatorTx] returns a duly initialized tx if it does not err
func buildAddPermissionlessValidatorTx(
	matches []*parser.Match,
	metadata Metadata,
	codec codec.Manager,
	avaxAssetID ids.ID,
) (*txs.Tx, []*types.AccountIdentifier, error) {
	if metadata.StakingMetadata == nil {
		return nil, nil, errInvalidMetadata
	}

	blockchainID := metadata.BlockchainID

	nodeID, err := ids.NodeIDFromString(metadata.NodeID)
	if err != nil {
		return nil, nil, err
	}

	subnetID := constants.PrimaryNetworkID
	if metadata.Subnet != "" {
		subnetID, err = ids.FromString(metadata.Subnet)
		if err != nil {
			return nil, nil, fmt.Errorf("%w: invalid subnet id", errInvalidMetadata)
		}
	}

	publicKeyBytes, err := formatting.Decode(formatting.HexNC, metadata.BLSPublicKey)
	if err != nil {
		return nil, nil, err
	}
	popBytes, err := formatting.Decode(formatting.HexNC, metadata.BLSProofOfPossession)
	if err != nil {
		return nil, nil, err
	}
	pop := &signer.ProofOfPossession{}
	copy(pop.PublicKey[:], publicKeyBytes)
	copy(pop.ProofOfPossession[:], popBytes)
	if err = pop.Verify(); err != nil {
		return nil, nil, err
	}

	validationRewardsOwner, err := buildOutputOwner(
		metadata.ValidationRewardsOwners,
		metadata.Locktime,
		metadata.Threshold,
	)
	if err != nil {
		return nil, nil, err
	}

	var delegationRewardsOwner *secp256k1fx.OutputOwners
	if len(metadata.DelegationRewardsOwners) == 0 {
		delegationRewardsOwner = validationRewardsOwner
	} else {
		delegationRewardsOwner, err = buildOutputOwner(
			metadata.DelegationRewardsOwners,
			metadata.Locktime,
			metadata.Threshold,
		)
		if err != nil {
			return nil, nil, err
		}
	}

	ins, _, signers, err := buildInputs(matches[0].Operations, avaxAssetID)
	if err != nil {
		return nil, nil, fmt.Errorf("parse inputs failed: %w", err)
	}

	outs, stakeOutputs, _, err := buildOutputs(matches[1].Operations, codec, avaxAssetID)
	if err != nil {
		return nil, nil, fmt.Errorf("parse outputs failed: %w", err)
	}

	weight, err := sumOutputAmounts(stakeOutputs)
	if err != nil {
		return nil, nil, err
	}

	tx := &txs.Tx{Unsigned: &txs.AddPermissionlessValidatorTx{
		BaseTx: txs.BaseTx{BaseTx: avax.BaseTx{
			NetworkID:    metadata.NetworkID,
			BlockchainID: blockchainID,
			Outs:         outs,
			Ins:          ins,
		}},
		Validator: txs.Validator{
			NodeID: nodeID,
			Start:  metadata.Start,
			End:    metadata.End,
			Wght:   weight,
		},
		Subnet:                subnetID,
		Signer:                pop,
		StakeOuts:             stakeOutputs,
		ValidatorRewardsOwner: validationRewardsOwner,
		DelegatorRewardsOwner: delegationRewardsOwner,
		DelegationShares:      metadata.Shares,
	}}

	return tx, signers, tx.Sign(codec, nil)
}

// [buildAddPermissionlessDelegatorTx] returns a duly initialized tx if it does not err
func buildAddPermissionlessDelegatorTx(
	matches []*parser.Match,
	metadata Metadata,
	codec codec.Manager,
	avaxAssetID ids.ID,
) (*txs.Tx, []*types.AccountIdentifier, error) {
	if metadata.StakingMetadata == nil {
		return nil, nil, errInvalidMetadata
	}

	blockchainID := metadata.BlockchainID

	nodeID, err := ids.NodeIDFromString(metadata.NodeID)
	if err != nil {
		return nil, nil, err
	}
	rewardsOwner, err := buildOutputOwner(metadata.ValidationRewardsOwners, metadata.Locktime, metadata.Threshold)
	if err != nil {
		return nil, nil, err
	}

	ins, _, signers, err := buildInputs(matches[0].Operations, avaxAssetID)
	if err != nil {
		return nil, nil, fmt.Errorf("parse inputs failed: %w", err)
	}

	outs, stakeOutputs, _, err := buildOutputs(matches[1].Operations, codec, avaxAssetID)
	if err != nil {
		return nil, nil, fmt.Errorf("parse outputs failed: %w", err)
	}

	weight, err := sumOutputAmounts(stakeOutputs)
	if err != nil {
		return nil, nil, err
	}

	subnetID := constants.PrimaryNetworkID
	if metadata.Subnet != "" {
		subnetID, err = ids.FromString(metadata.Subnet)
		if err != nil {
			return nil, nil, fmt.Errorf("%w: invalid subnet id", errInvalidMetadata)
		}
	}

	tx := &txs.Tx{Unsigned: &txs.AddPermissionlessDelegatorTx{
		BaseTx: txs.BaseTx{BaseTx: avax.BaseTx{
			NetworkID:    metadata.NetworkID,
			BlockchainID: blockchainID,
			Outs:         outs,
			Ins:          ins,
		}},
		Validator: txs.Validator{
			NodeID: nodeID,
			Start:  metadata.Start,
			End:    metadata.End,
			Wght:   weight,
		},
		Subnet:                 subnetID,
		StakeOuts:              stakeOutputs,
		DelegationRewardsOwner: rewardsOwner,
	}}

	return tx, signers, tx.Sign(codec, nil)
}

func buildOutputOwner(
	addrs []string,
	locktime uint64,
	threshold uint32,
) (*secp256k1fx.OutputOwners, error) {
	rewardAddrs := make([]ids.ShortID, len(addrs))
	for i, addr := range addrs {
		addrID, err := address.ParseToID(addr)
		if err != nil {
			return nil, err
		}
		rewardAddrs[i] = addrID
	}
	utils.Sort(rewardAddrs)

	return &secp256k1fx.OutputOwners{
		Locktime:  locktime,
		Threshold: threshold,
		Addrs:     rewardAddrs,
	}, nil
}

func buildInputs(
	operations []*types.Operation,
	avaxAssetID ids.ID,
) (
	ins []*avax.TransferableInput,
	imported []*avax.TransferableInput,
	signers []*types.AccountIdentifier,
	err error,
) {
	for _, op := range operations {
		utxoID, err := mapper.DecodeUTXOID(op.CoinChange.CoinIdentifier.Identifier)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("failed to decode UTXO ID: %w", err)
		}

		opMetadata, err := ParseOpMetadata(op.Metadata)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("parse input operation Metadata failed: %w", err)
		}

		val, err := types.AmountValue(op.Amount)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("parse operation amount failed: %w", err)
		}

		in := &avax.TransferableInput{
			UTXOID: *utxoID,
			Asset:  avax.Asset{ID: avaxAssetID},
			In: &secp256k1fx.TransferInput{
				Amt: val.Uint64(),
				Input: secp256k1fx.Input{
					SigIndices: opMetadata.SigIndices,
				},
			},
		}

		switch opMetadata.Type {
		case OpTypeImport:
			imported = append(imported, in)
		case OpTypeInput:
			ins = append(ins, in)
		default:
			return nil, nil, nil, fmt.Errorf("invalid option type: %s", op.Type)
		}
		signers = append(signers, op.Account)
	}

	utils.Sort(ins)
	utils.Sort(imported)

	return ins, imported, signers, nil
}

// ParseOpMetadata creates an OperationMetadata from given generic metadata map
func ParseOpMetadata(metadata map[string]interface{}) (*OperationMetadata, error) {
	var operationMetadata OperationMetadata
	if err := mapper.UnmarshalJSONMap(metadata, &operationMetadata); err != nil {
		return nil, err
	}

	// set threshold default to 1
	if operationMetadata.Threshold == 0 {
		operationMetadata.Threshold = 1
	}

	// set sig indices to a single signer if not provided
	if operationMetadata.SigIndices == nil {
		operationMetadata.SigIndices = []uint32{0}
	}

	return &operationMetadata, nil
}

func buildOutputs(
	operations []*types.Operation,
	codec codec.Manager,
	avaxAssetID ids.ID,
) (
	outs []*avax.TransferableOutput,
	stakeOutputs []*avax.TransferableOutput,
	exported []*avax.TransferableOutput,
	err error,
) {
	for _, op := range operations {
		opMetadata, err := ParseOpMetadata(op.Metadata)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("parse output operation Metadata failed: %w", err)
		}

		addrID, err := address.ParseToID(op.Account.Address)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("parse output address failed: %w", err)
		}

		outputOwners := &secp256k1fx.OutputOwners{
			Addrs:     []ids.ShortID{addrID},
			Locktime:  opMetadata.Locktime,
			Threshold: opMetadata.Threshold,
		}

		val, err := types.AmountValue(op.Amount)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("parse operation amount failed: %w", err)
		}

		out := &avax.TransferableOutput{
			Asset: avax.Asset{ID: avaxAssetID},
			Out: &secp256k1fx.TransferOutput{
				Amt:          val.Uint64(),
				OutputOwners: *outputOwners,
			},
		}

		switch opMetadata.Type {
		case OpTypeOutput:
			outs = append(outs, out)
		case OpTypeStakeOutput:
			stakeOutputs = append(stakeOutputs, out)
		case OpTypeExport:
			exported = append(exported, out)
		default:
			return nil, nil, nil, fmt.Errorf("invalid option type: %s", op.Type)
		}
	}

	avax.SortTransferableOutputs(outs, codec)
	avax.SortTransferableOutputs(stakeOutputs, codec)
	avax.SortTransferableOutputs(exported, codec)

	return outs, stakeOutputs, exported, nil
}

func sumOutputAmounts(stakeOutputs []*avax.TransferableOutput) (uint64, error) {
	var stakeOutputAmountSum uint64
	for _, out := range stakeOutputs {
		outAmount := out.Output().Amount()
		if outAmount > math.MaxUint64-stakeOutputAmountSum {
			return 0, errOutputAmountOverflow
		}
		stakeOutputAmountSum += outAmount
	}
	return stakeOutputAmountSum, nil
}
