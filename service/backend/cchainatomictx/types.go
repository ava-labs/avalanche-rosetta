package cchainatomictx

import (
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

type cAtomicTx struct {
	Tx           *evm.Tx
	Codec        codec.Manager
	CodecVersion uint16
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
	c.Tx = &tx
	return nil
}

func (c *cAtomicTx) SigningPayload() ([]byte, error) {
	unsignedAtomicBytes, err := c.Codec.Marshal(c.CodecVersion, &c.Tx.UnsignedAtomicTx)
	if err != nil {
		return nil, err
	}

	hash := hashing.ComputeHash256(unsignedAtomicBytes)
	return hash, nil
}

func (c *cAtomicTx) Hash() ([]byte, error) {
	bytes, err := c.Codec.Marshal(c.CodecVersion, &c.Tx)
	if err != nil {
		return nil, err
	}

	hash := hashing.ComputeHash256(bytes)
	return hash, nil
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
