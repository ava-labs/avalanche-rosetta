package cchain

import (
	"encoding/json"
	"math/big"

	cconstants "github.com/ava-labs/avalanche-rosetta/constants/cchain"
	"github.com/coinbase/rosetta-sdk-go/types"
)

const BalanceOfMethodPrefix = "0x70a08231000000000000000000000000"

type options struct {
	From                   string          `json:"from"`
	To                     string          `json:"to"`
	Value                  *big.Int        `json:"value"`
	SuggestedFeeMultiplier *float64        `json:"suggested_fee_multiplier,omitempty"`
	GasPrice               *big.Int        `json:"gas_price,omitempty"`
	GasLimit               *big.Int        `json:"gas_limit,omitempty"`
	Nonce                  *big.Int        `json:"nonce,omitempty"`
	Currency               *types.Currency `json:"currency,omitempty"`
}

type accountMetadata struct {
	Nonce uint64 `json:"nonce"`
}

type transaction struct {
	From     string          `json:"from"`
	To       string          `json:"to"`
	Value    *big.Int        `json:"value"`
	Data     []byte          `json:"data"`
	Nonce    uint64          `json:"nonce"`
	GasPrice *big.Int        `json:"gas_price"`
	GasLimit uint64          `json:"gas"`
	ChainID  *big.Int        `json:"chain_id"`
	Currency *types.Currency `json:"currency,omitempty"`
}

type metadata struct {
	Nonce    uint64   `json:"nonce"`
	GasPrice *big.Int `json:"gas_price"`
	GasLimit uint64   `json:"gas_limit"`
}

type parseMetadata struct {
	Nonce    uint64   `json:"nonce"`
	GasPrice *big.Int `json:"gas_price"`
	GasLimit uint64   `json:"gas_limit"`
	ChainID  *big.Int `json:"chain_id"`
}

// has0xPrefix validates str begins with '0x' or '0X'.
// Copied from the go-ethereum hextuil.go library
func has0xPrefix(str string) bool {
	return len(str) >= 2 && str[0] == '0' && (str[1] == 'x' || str[1] == 'X')
}

type signedTransactionWrapper struct {
	SignedTransaction []byte          `json:"signed_tx"`
	Currency          *types.Currency `json:"currency,omitempty"`
}

func (t *signedTransactionWrapper) UnmarshalJSON(data []byte) error {
	// We need to re-define the signedTransactionWrapper struct to avoid
	// infinite recursion while unmarshaling.
	//
	// We don't define another struct because this is never used outside of this
	// function.
	tw := struct {
		SignedTransaction []byte          `json:"signed_tx"`
		Currency          *types.Currency `json:"currency,omitempty"`
	}{}
	if err := json.Unmarshal(data, &tw); err != nil {
		return err
	}

	// Exit early if SignedTransaction is populated
	if len(tw.SignedTransaction) > 0 {
		t.SignedTransaction = tw.SignedTransaction
		t.Currency = tw.Currency
		return nil
	}

	// Handle legacy format (will error during processing if invalid)
	t.SignedTransaction = data
	t.Currency = cconstants.AvaxCurrency
	return nil
}
