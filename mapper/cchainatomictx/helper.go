package cchainatomictx

import (
	"math/big"

	"github.com/ava-labs/avalanchego/utils/formatting/address"
	"github.com/coinbase/rosetta-sdk-go/types"

	"github.com/ava-labs/avalanche-rosetta/constants"
	pconstants "github.com/ava-labs/avalanche-rosetta/constants/pchain"
	"github.com/ava-labs/avalanche-rosetta/mapper"
)

// IsCChainBech32Address checks whether a given account identifier contains a C-chain Bech32 type address
func IsCChainBech32Address(accountIdentifier *types.AccountIdentifier) bool {
	if chainID, _, _, err := address.Parse(accountIdentifier.Address); err == nil {
		return chainID == constants.CChain.String()
	}
	return false
}

// AtomicAvaxAmount creates a Rosetta Amount representing AVAX amount in nAVAXs with given quantity
func AtomicAvaxAmount(value *big.Int) *types.Amount {
	return mapper.Amount(value, pconstants.AtomicAvaxCurrency)
}
