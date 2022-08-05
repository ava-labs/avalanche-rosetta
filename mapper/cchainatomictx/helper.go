package cchainatomictx

import (
	"github.com/ava-labs/avalanchego/utils/formatting/address"
	"github.com/coinbase/rosetta-sdk-go/types"

	"github.com/ava-labs/avalanche-rosetta/mapper"
)

func IsCChainBech32Address(accountIdentifier *types.AccountIdentifier) bool {
	if chainID, _, _, err := address.Parse(accountIdentifier.Address); err == nil {
		return chainID == mapper.CChainNetworkIdentifier
	}
	return false
}

func IsAtomicOpType(t string) bool {
	return t == mapper.OpExport || t == mapper.OpImport
}
