package service

import (
	"encoding/json"
	"math/big"
	"strconv"

	"github.com/ethereum/go-ethereum/common/hexutil"
)

const (
	seconds2milliseconds = 1000
)

type options struct {
	From                   string   `json:"from"`
	To                     string   `json:"to"`
	Value                  *big.Int `json:"value"`
	SuggestedFeeMultiplier *float64 `json:"suggested_fee_multiplier,omitempty"`
	GasPrice               *big.Int `json:"gas_price,omitempty"`
	GasLimit               *big.Int `json:"gas_limit,omitempty"`
	Nonce                  *big.Int `json:"nonce,omitempty"`
}

type optionsWire struct {
	From                   string   `json:"from"`
	To                     string   `json:"to"`
	Value                  string   `json:"value"`
	SuggestedFeeMultiplier *float64 `json:"suggested_fee_multiplier,omitempty"`
	GasPrice               string   `json:"gas_price,omitempty"`
	GasLimit               string   `json:"gas_limit,omitempty"`
	Nonce                  string   `json:"nonce,omitempty"`
}

func (o *options) MarshalJSON() ([]byte, error) {
	ow := &optionsWire{
		From:                   o.From,
		To:                     o.To,
		SuggestedFeeMultiplier: o.SuggestedFeeMultiplier,
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

type parseMetadata struct {
	Nonce    uint64   `json:"nonce"`
	GasPrice *big.Int `json:"gas_price"`
	GasLimit uint64   `json:"gas_limit"`
	ChainID  *big.Int `json:"chain_id"`
}

type parseMetadataWire struct {
	Nonce    string `json:"nonce"`
	GasPrice string `json:"gas_price"`
	GasLimit string `json:"gas_limit"`
	ChainID  string `json:"chain_id"`
}

func (p *parseMetadata) MarshalJSON() ([]byte, error) {
	pmw := &parseMetadataWire{
		Nonce:    hexutil.Uint64(p.Nonce).String(),
		GasPrice: hexutil.EncodeBig(p.GasPrice),
		GasLimit: hexutil.Uint64(p.GasLimit).String(),
		ChainID:  hexutil.EncodeBig(p.ChainID),
	}

	return json.Marshal(pmw)
}

type transaction struct {
	From     string   `json:"from"`
	To       string   `json:"to"`
	Value    *big.Int `json:"value"`
	Data     []byte   `json:"data"`
	Nonce    uint64   `json:"nonce"`
	GasPrice *big.Int `json:"gas_price"`
	GasLimit uint64   `json:"gas"`
	ChainID  *big.Int `json:"chain_id"`
}

type transactionWire struct {
	From     string `json:"from"`
	To       string `json:"to"`
	Value    string `json:"value"`
	Data     string `json:"data"`
	Nonce    string `json:"nonce"`
	GasPrice string `json:"gas_price"`
	GasLimit string `json:"gas"`
	ChainID  string `json:"chain_id"`
}

func (t *transaction) MarshalJSON() ([]byte, error) {
	tw := &transactionWire{
		From:     t.From,
		To:       t.To,
		Value:    hexutil.EncodeBig(t.Value),
		Data:     hexutil.Encode(t.Data),
		Nonce:    hexutil.EncodeUint64(t.Nonce),
		GasPrice: hexutil.EncodeBig(t.GasPrice),
		GasLimit: hexutil.EncodeUint64(t.GasLimit),
		ChainID:  hexutil.EncodeBig(t.ChainID),
	}

	return json.Marshal(tw)
}

func (t *transaction) UnmarshalJSON(data []byte) error {
	var tw transactionWire
	if err := json.Unmarshal(data, &tw); err != nil {
		return err
	}

	value, err := hexutil.DecodeBig(tw.Value)
	if err != nil {
		return err
	}

	twData, err := hexutil.Decode(tw.Data)
	if err != nil {
		return err
	}

	nonce, err := hexutil.DecodeUint64(tw.Nonce)
	if err != nil {
		return err
	}

	gasPrice, err := hexutil.DecodeBig(tw.GasPrice)
	if err != nil {
		return err
	}

	gasLimit, err := hexutil.DecodeUint64(tw.GasLimit)
	if err != nil {
		return err
	}

	chainID, err := hexutil.DecodeBig(tw.ChainID)
	if err != nil {
		return err
	}

	t.From = tw.From
	t.To = tw.To
	t.Value = value
	t.Data = twData
	t.Nonce = nonce
	t.GasPrice = gasPrice
	t.GasLimit = gasLimit
	t.ChainID = chainID
	t.GasPrice = gasPrice
	return nil
}

type accountMetadata struct {
	Nonce uint64 `json:"nonce"`
}

type accountMetadataWire struct {
	Nonce string `json:"nonce"`
}

func (m *accountMetadata) MarshalJSON() ([]byte, error) {
	mw := &accountMetadataWire{
		Nonce: strconv.FormatUint(m.Nonce, 10), //nolint:gomnd
	}

	return json.Marshal(mw)
}

func (m *accountMetadata) UnmarshalJSON(data []byte) error {
	var mw accountMetadataWire
	if err := json.Unmarshal(data, &mw); err != nil {
		return err
	}

	nonce, err := strconv.ParseUint(mw.Nonce, 10, 64) //nolint:gomnd
	if err != nil {
		return err
	}

	m.Nonce = nonce
	return nil
}
