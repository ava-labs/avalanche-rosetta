package service

import (
	"context"
	"encoding/json"
	"math/big"

	"github.com/ava-labs/avalanche-rosetta/client"
	"github.com/coinbase/rosetta-sdk-go/types"

	ethtypes "github.com/ava-labs/coreth/core/types"
	"github.com/ethereum/go-ethereum/common"
	ethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

const (
	transferGasLimit = uint64(21000) //nolint:gomnd
	genesisTimestamp = 946713601000  // min allowable timestamp
)

func makeGenesisBlock(hash string) *types.Block {
	return &types.Block{
		BlockIdentifier: &types.BlockIdentifier{
			Index: 0,
			Hash:  hash,
		},
		ParentBlockIdentifier: &types.BlockIdentifier{
			Index: 0,
			Hash:  hash,
		},
		Timestamp: genesisTimestamp,
	}
}

func blockHeaderFromInput(
	c client.Client,
	input *types.PartialBlockIdentifier,
) (*ethtypes.Header, *types.Error) {
	var (
		header *ethtypes.Header
		err    error
	)

	if input == nil {
		header, err = c.HeaderByNumber(context.Background(), nil)
	} else {
		if input.Hash == nil && input.Index == nil {
			return nil, errInvalidInput
		}

		if input.Index != nil {
			header, err = c.HeaderByNumber(context.Background(), big.NewInt(*input.Index))
		} else {
			header, err = c.HeaderByHash(context.Background(), ethcommon.HexToHash(*input.Hash))
		}
	}

	if err != nil {
		return nil, wrapError(errInternalError, err)
	}

	return header, nil
}

// unmarshalJSONMap converts map[string]interface{} into a interface{}.
func unmarshalJSONMap(m map[string]interface{}, i interface{}) error {
	b, err := json.Marshal(m)
	if err != nil {
		return err
	}

	return json.Unmarshal(b, i)
}

// marshalJSONMap converts an interface into a map[string]interface{}.
func marshalJSONMap(i interface{}) (map[string]interface{}, error) {
	b, err := json.Marshal(i)
	if err != nil {
		return nil, err
	}

	var m map[string]interface{}
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, err
	}

	return m, nil
}

type options struct {
	From                   string   `json:"from"`
	SuggestedFeeMultiplier *float64 `json:"suggested_fee_multiplier,omitempty"`
	GasPrice               *big.Int `json:"gas_price,omitempty"`
	Nonce                  *big.Int `json:"nonce,omitempty"`
}

type optionsWire struct {
	From                   string   `json:"from"`
	SuggestedFeeMultiplier *float64 `json:"suggested_fee_multiplier,omitempty"`
	GasPrice               string   `json:"gas_price,omitempty"`
	Nonce                  string   `json:"nonce,omitempty"`
}

func (o *options) MarshalJSON() ([]byte, error) {
	ow := &optionsWire{
		From:                   o.From,
		SuggestedFeeMultiplier: o.SuggestedFeeMultiplier,
	}
	if o.Nonce != nil {
		ow.Nonce = hexutil.EncodeBig(o.Nonce)
	}
	if o.GasPrice != nil {
		ow.GasPrice = hexutil.EncodeBig(o.GasPrice)
	}

	return json.Marshal(ow)
}

func (o *options) UnmarshalJSON(data []byte) error {
	var ow optionsWire
	if err := json.Unmarshal(data, &ow); err != nil {
		return err
	}

	if len(ow.Nonce) > 0 {
		nonce, err := hexutil.DecodeBig(ow.Nonce)
		if err != nil {
			return err
		}
		o.Nonce = nonce
	}

	if len(ow.GasPrice) > 0 {
		gasPrice, err := hexutil.DecodeBig(ow.GasPrice)
		if err != nil {
			return err
		}
		o.GasPrice = gasPrice
	}

	return nil
}

type metadata struct {
	Nonce    uint64   `json:"nonce"`
	GasPrice *big.Int `json:"gas_price"`
}

type metadataWire struct {
	Nonce    string `json:"nonce"`
	GasPrice string `json:"gas_price"`
}

func (m *metadata) MarshalJSON() ([]byte, error) {
	mw := &metadataWire{
		Nonce:    hexutil.Uint64(m.Nonce).String(),
		GasPrice: hexutil.EncodeBig(m.GasPrice),
	}

	return json.Marshal(mw)
}

func (m *metadata) UnmarshalJSON(data []byte) error {
	var mw metadataWire
	if err := json.Unmarshal(data, &mw); err != nil {
		return err
	}

	nonce, err := hexutil.DecodeUint64(mw.Nonce)
	if err != nil {
		return err
	}

	gasPrice, err := hexutil.DecodeBig(mw.GasPrice)
	if err != nil {
		return err
	}

	m.GasPrice = gasPrice
	m.Nonce = nonce
	return nil
}

type parseMetadata struct {
	Nonce    uint64   `json:"nonce"`
	GasPrice *big.Int `json:"gas_price"`
	ChainID  *big.Int `json:"chain_id"`
}

type parseMetadataWire struct {
	Nonce    string `json:"nonce"`
	GasPrice string `json:"gas_price"`
	ChainID  string `json:"chain_id"`
}

func (p *parseMetadata) MarshalJSON() ([]byte, error) {
	pmw := &parseMetadataWire{
		Nonce:    hexutil.Uint64(p.Nonce).String(),
		GasPrice: hexutil.EncodeBig(p.GasPrice),
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

// ChecksumAddress ensures an Ethereum hex address
// is in Checksum Format. If the address cannot be converted,
// it returns !ok.
func ChecksumAddress(address string) (string, bool) {
	addr, err := common.NewMixedcaseAddressFromString(address)
	if err != nil {
		return "", false
	}

	return addr.Address().Hex(), true
}
