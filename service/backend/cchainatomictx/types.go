package cchainatomictx

import (
	"errors"

	"github.com/ava-labs/avalanchego/codec"
	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/avalanchego/utils/hashing"
	"github.com/ava-labs/coreth/plugin/evm"
	"github.com/coinbase/rosetta-sdk-go/types"

	"github.com/ava-labs/avalanche-rosetta/mapper"
	cmapper "github.com/ava-labs/avalanche-rosetta/mapper/cchainatomictx"
	"github.com/ava-labs/avalanche-rosetta/service"
	"github.com/ava-labs/avalanche-rosetta/service/backend/common"
)

var (
	_ common.AvaxTx    = &cAtomicTx{}
	_ common.TxBuilder = &cAtomicTxBuilder{}
	_ common.TxParser  = &cAtomicTxParser{}
)

type cAtomicTx struct {
	Tx           *evm.Tx
	Codec        codec.Manager
	CodecVersion uint16
}

func (c *cAtomicTx) Initialize() error {
	if c.Tx == nil {
		return common.ErrNoTxGiven
	}
	return c.Tx.Sign(c.Codec, nil)
}

func (c *cAtomicTx) Marshal() ([]byte, error) {
	return c.Codec.Marshal(c.CodecVersion, c.Tx)
}

func (c *cAtomicTx) Unmarshal(bytes []byte) error {
	tx := evm.Tx{}
	_, err := c.Codec.Unmarshal(bytes, &tx)
	if err != nil {
		return err
	}
	if err := tx.Sign(c.Codec, nil); err != nil {
		return err
	}
	c.Tx = &tx

	return c.Initialize()
}

func (c *cAtomicTx) SigningPayload() []byte {
	return hashing.ComputeHash256(c.Tx.Bytes())
}

func (c *cAtomicTx) Hash() ids.ID {
	return c.Tx.ID()
}

type cAtomicTxBuilder struct {
	avaxAssetID  ids.ID
	codec        codec.Manager
	codecVersion uint16
}

func (c cAtomicTxBuilder) BuildTx(operations []*types.Operation, metadata map[string]interface{}) (common.AvaxTx, []*types.AccountIdentifier, *types.Error) {
	cMetadata := cmapper.Metadata{}
	err := mapper.UnmarshalJSONMap(metadata, &cMetadata)
	if err != nil {
		return nil, nil, service.WrapError(service.ErrInvalidInput, err)
	}

	matches, err := common.MatchOperations(operations)
	if err != nil {
		return nil, nil, service.WrapError(service.ErrInvalidInput, err)
	}

	opType := matches[0].Operations[0].Type
	tx, signers, err := cmapper.BuildTx(opType, matches, cMetadata, c.codec, c.avaxAssetID)
	if err != nil {
		return nil, nil, service.WrapError(service.ErrInternalError, err)
	}

	return &cAtomicTx{
		Tx:           tx,
		Codec:        c.codec,
		CodecVersion: c.codecVersion,
	}, signers, nil
}

type cAtomicTxParser struct {
	hrp      string
	chainIDs map[ids.ID]string
}

func (c cAtomicTxParser) ParseTx(tx *common.RosettaTx, inputAddresses map[string]*types.AccountIdentifier) ([]*types.Operation, error) {
	cTx, ok := tx.Tx.(*cAtomicTx)
	if !ok {
		return nil, errors.New("invalid transaction")
	}
	parser := cmapper.NewTxParser(c.hrp, c.chainIDs, inputAddresses)
	return parser.Parse(*cTx.Tx)
}
