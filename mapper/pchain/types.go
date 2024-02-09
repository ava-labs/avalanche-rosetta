package pchain

import (
	"github.com/ava-labs/avalanchego/ids"
)

const (
	OpImportAvax                 = "IMPORT_AVAX"
	OpExportAvax                 = "EXPORT_AVAX"
	OpAddValidator               = "ADD_VALIDATOR"
	OpAddPermissionlessValidator = "ADD_PERMISSIONLESS_VALIDATOR"
	OpAddDelegator               = "ADD_DELEGATOR"
	OpAddPermissionlessDelegator = "ADD_PERMISSIONLESS_DELEGATOR"
	OpRewardValidator            = "REWARD_VALIDATOR"
	OpCreateChain                = "CREATE_CHAIN"
	OpCreateSubnet               = "CREATE_SUBNET"
	OpAddSubnetValidator         = "ADD_SUBNET_VALIDATOR"
	OpRemoveSubnetValidator      = "REMOVE_SUBNET_VALIDATOR"
	OpTransformSubnetValidator   = "TRANSFORM_SUBNET_VALIDATOR"
	OpAdvanceTime                = "ADVANCE_TIME"
	OpBase                       = "BASE"
	OpTransferSubnetOwnership    = "TRANSFER_SUBNET_OWNERSHIP"

	OpTypeImport      = "IMPORT"
	OpTypeExport      = "EXPORT"
	OpTypeInput       = "INPUT"
	OpTypeOutput      = "OUTPUT"
	OpTypeStakeOutput = "STAKE"
	OpTypeReward      = "REWARD"

	MetadataOpType           = "type"
	MetadataTxType           = "tx_type"
	MetadataStakingTxID      = "staking_tx_id"
	MetadataValidatorNodeID  = "validator_node_id"
	MetadataStakingStartTime = "staking_start_time"
	MetadataStakingEndTime   = "staking_end_time"
	MetadataMessage          = "message"
	MetadataSigner           = "signer"

	MetadataValidatorRewards     = "validator_rewards"
	MetadataDelegationRewards    = "delegation_rewards"
	MetadataDelegationFeeRewards = "delegation_fee_rewards"

	SubAccountTypeSharedMemory       = "shared_memory"
	SubAccountTypeUnlocked           = "unlocked"
	SubAccountTypeLockedStakeable    = "locked_stakeable"
	SubAccountTypeLockedNotStakeable = "locked_not_stakeable"
	SubAccountTypeStaked             = "staked"
)

var (
	OperationTypes = []string{
		OpImportAvax,
		OpExportAvax,
		OpAddValidator,
		OpAddDelegator,
		OpRewardValidator,
		OpCreateChain,
		OpCreateSubnet,
		OpAddSubnetValidator,
		OpRemoveSubnetValidator,
		OpTransformSubnetValidator,
		OpAddPermissionlessValidator,
		OpAddPermissionlessDelegator,
	}
	CallMethods = []string{}
)

// OperationMetadata contains metadata fields specific to individual Rosetta operations as opposed to transactions
type OperationMetadata struct {
	Type       string   `json:"type"`
	SigIndices []uint32 `json:"sig_indices,omitempty"`
	Locktime   uint64   `json:"locktime"`
	Threshold  uint32   `json:"threshold,omitempty"`
}

// ImportExportOptions contain response fields returned by /construction/preprocess for P-chain Import/Export transactions
type ImportExportOptions struct {
	SourceChain      string `json:"source_chain"`
	DestinationChain string `json:"destination_chain"`
}

// StakingOptions contain response fields returned by /construction/preprocess for P-chain AddValidator/AddDelegator transactions
type StakingOptions struct {
	NodeID                  string   `json:"node_id"`
	BLSPublicKey            string   `json:"bls_public_key"`
	BLSProofOfPossession    string   `json:"bls_proof_of_possession"`
	ValidationRewardsOwners []string `json:"reward_addresses"`
	DelegationRewardsOwners []string `json:"delegator_reward_addresses"`
	Start                   uint64   `json:"start"` // TODO: Remove post-Durango
	End                     uint64   `json:"end"`
	Subnet                  string   `json:"subnet"`
	Shares                  uint32   `json:"shares"`
	Locktime                uint64   `json:"locktime"`
	Threshold               uint32   `json:"threshold"`
}

// Metadata contains metadata values returned by /construction/metadata for P-chain transactions
type Metadata struct {
	NetworkID    uint32 `json:"network_id"`
	BlockchainID ids.ID `json:"blockchain_id"`
	*ImportMetadata
	*ExportMetadata
	*StakingMetadata
}

// ImportMetadata contain response fields returned by /construction/metadata for P-chain Import transactions
type ImportMetadata struct {
	SourceChainID ids.ID `json:"source_chain_id"`
}

// ExportMetadata contain response fields returned by /construction/metadata for P-chain Export transactions
type ExportMetadata struct {
	DestinationChain   string `json:"destination_chain"`
	DestinationChainID ids.ID `json:"destination_chain_id"`
}

// StakingMetadata contain response fields returned by /construction/metadata for P-chain AddValidator/AddDelegator transactions
type StakingMetadata struct {
	NodeID                  string   `json:"node_id"`
	BLSPublicKey            string   `json:"bls_public_key"`
	BLSProofOfPossession    string   `json:"bls_proof_of_possession"`
	ValidationRewardsOwners []string `json:"reward_addresses"`
	DelegationRewardsOwners []string `json:"delegator_reward_addresses"`
	Start                   uint64   `json:"start"` // TODO: Remove Post-Durango Activation
	End                     uint64   `json:"end"`
	Subnet                  string   `json:"subnet"`
	Shares                  uint32   `json:"shares"`
	Locktime                uint64   `json:"locktime"`
	Threshold               uint32   `json:"threshold"`
}
