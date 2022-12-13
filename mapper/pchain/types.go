package pchain

import (
	"github.com/ava-labs/avalanchego/ids"
)

const (
	MetadataOpType           = "type"
	MetadataTxType           = "tx_type"
	MetadataStakingTxID      = "staking_tx_id"
	MetadataValidatorNodeID  = "validator_node_id"
	MetadataStakingStartTime = "staking_start_time"
	MetadataStakingEndTime   = "staking_end_time"
	MetadataMessage          = "message"
	MetadataSigner           = "signer"

	SubAccountTypeSharedMemory       = "shared_memory"
	SubAccountTypeUnlocked           = "unlocked"
	SubAccountTypeLockedStakeable    = "locked_stakeable"
	SubAccountTypeLockedNotStakeable = "locked_not_stakeable"
	SubAccountTypeStaked             = "staked"

	metadataImportedInputs = "imported_inputs"
)

var CallMethods = []string{}

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
	NodeID          string   `json:"node_id"`
	Start           uint64   `json:"start"`
	End             uint64   `json:"end"`
	Shares          uint32   `json:"shares"`
	Memo            string   `json:"memo"`
	Locktime        uint64   `json:"locktime"`
	Threshold       uint32   `json:"threshold"`
	RewardAddresses []string `json:"reward_addresses"`
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
	NodeID          string   `json:"node_id"`
	RewardAddresses []string `json:"reward_addresses"`
	Start           uint64   `json:"start"`
	End             uint64   `json:"end"`
	Shares          uint32   `json:"shares"`
	Locktime        uint64   `json:"locktime"`
	Threshold       uint32   `json:"threshold"`
	Memo            string   `json:"memo"`
}
