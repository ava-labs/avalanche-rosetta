package service

import (
	"encoding/json"
	"math/big"

	cconstants "github.com/ava-labs/avalanche-rosetta/constants/cchain"
	"github.com/coinbase/rosetta-sdk-go/types"
	"github.com/ethereum/go-ethereum/common/hexutil"
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

type optionsWire struct {
	From                   string          `json:"from"`
	To                     string          `json:"to"`
	Value                  string          `json:"value"`
	SuggestedFeeMultiplier *float64        `json:"suggested_fee_multiplier,omitempty"`
	GasPrice               string          `json:"gas_price,omitempty"`
	GasLimit               string          `json:"gas_limit,omitempty"`
	Nonce                  string          `json:"nonce,omitempty"`
	Currency               *types.Currency `json:"currency,omitempty"`
}

func (o *options) MarshalJSON() ([]byte, error) {
	ow := &optionsWire{
		From:                   o.From,
		To:                     o.To,
		SuggestedFeeMultiplier: o.SuggestedFeeMultiplier,
		Currency:               o.Currency,
	}
	if o.Value != nil {
		ow.Value = hexutil.EncodeBig(o.Value)
	}
	if o.GasPrice != nil {
		ow.GasPrice = hexutil.EncodeBig(o.GasPrice)
	}
	if o.GasLimit != nil {
		ow.GasLimit = hexutil.EncodeBig(o.GasLimit)
	}
	if o.Nonce != nil {
		ow.Nonce = hexutil.EncodeBig(o.Nonce)
	}

	return json.Marshal(ow)
}

func (o *options) UnmarshalJSON(data []byte) error {
	var ow optionsWire
	if err := json.Unmarshal(data, &ow); err != nil {
		return err
	}
	o.From = ow.From
	o.To = ow.To
	o.SuggestedFeeMultiplier = ow.SuggestedFeeMultiplier
	o.Currency = ow.Currency

	if len(ow.Value) > 0 {
		value, err := hexutil.DecodeBig(ow.Value)
		if err != nil {
			return err
		}
		o.Value = value
	}

	if len(ow.GasPrice) > 0 {
		gasPrice, err := hexutil.DecodeBig(ow.GasPrice)
		if err != nil {
			return err
		}
		o.GasPrice = gasPrice
	}

	if len(ow.GasLimit) > 0 {
		gasLimit, err := hexutil.DecodeBig(ow.GasLimit)
		if err != nil {
			return err
		}
		o.GasLimit = gasLimit
	}

	if len(ow.Nonce) > 0 {
		nonce, err := hexutil.DecodeBig(ow.Nonce)
		if err != nil {
			return err
		}
		o.Nonce = nonce
	}

	return nil
}

type metadata struct {
	Nonce    uint64   `json:"nonce"`
	GasPrice *big.Int `json:"gas_price"`
	GasLimit uint64   `json:"gas_limit"`
}

type metadataWire struct {
	Nonce    string `json:"nonce"`
	GasPrice string `json:"gas_price"`
	GasLimit string `json:"gas_limit"`
}

func (m *metadata) MarshalJSON() ([]byte, error) {
	mw := &metadataWire{
		Nonce:    hexutil.Uint64(m.Nonce).String(),
		GasPrice: hexutil.EncodeBig(m.GasPrice),
		GasLimit: hexutil.Uint64(m.GasLimit).String(),
	}

	return json.Marshal(mw)
}

func (m *metadata) UnmarshalJSON(data []byte) error {
	var mw metadataWire
	if err := json.Unmarshal(data, &mw); err != nil {
		return err
	}

	gasPrice, err := hexutil.DecodeBig(mw.GasPrice)
	if err != nil {
		return err
	}
	m.GasPrice = gasPrice

	gasLimit, err := hexutil.DecodeUint64(mw.GasLimit)
	if err != nil {
		return err
	}
	m.GasLimit = gasLimit

	nonce, err := hexutil.DecodeUint64(mw.Nonce)
	if err != nil {
		return err
	}
	m.Nonce = nonce

	return nil
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
