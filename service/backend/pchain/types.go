package pchain

import (
	"errors"

	"github.com/ava-labs/avalanchego/codec"
	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/avalanchego/utils/hashing"
	"github.com/ava-labs/avalanchego/vms/platformvm"
	"github.com/coinbase/rosetta-sdk-go/types"

	"github.com/ava-labs/avalanche-rosetta/mapper"
	pmapper "github.com/ava-labs/avalanche-rosetta/mapper/pchain"
	"github.com/ava-labs/avalanche-rosetta/service"
	"github.com/ava-labs/avalanche-rosetta/service/backend/common"
)

var errInvalidTransaction = errors.New("invalid transaction")

type AccountBalance struct {
	Total              uint64
	Unlocked           uint64
	Staked             uint64
	LockedStakeable    uint64
	LockedNotStakeable uint64
}

type pTx struct {
	Tx           *platformvm.Tx
	Codec        codec.Manager
	CodecVersion uint16
}

func (p *pTx) Marshal() ([]byte, error) {
	return p.Codec.Marshal(p.CodecVersion, p.Tx)
}

func (p *pTx) Unmarshal(bytes []byte) error {
	tx := platformvm.Tx{}
	_, err := p.Codec.Unmarshal(bytes, &tx)
	if err != nil {
		return err
	}
	p.Tx = &tx
	return nil
}

func (p *pTx) SigningPayload() ([]byte, error) {
	unsignedAtomicBytes, err := p.Codec.Marshal(p.CodecVersion, &p.Tx.UnsignedTx)
	if err != nil {
		return nil, err
	}

	hash := hashing.ComputeHash256(unsignedAtomicBytes)
	return hash, nil
}

func (p *pTx) Hash() ([]byte, error) {
	bytes, err := p.Codec.Marshal(p.CodecVersion, &p.Tx)
	if err != nil {
		return nil, err
	}

	hash := hashing.ComputeHash256(bytes)
	return hash, nil
}

type pTxBuilder struct {
	avaxAssetID  ids.ID
	codec        codec.Manager
	codecVersion uint16
}

func (p pTxBuilder) BuildTx(operations []*types.Operation, metadataMap map[string]interface{}) (common.AvaxTx, []*types.AccountIdentifier, *types.Error) {
	var metadata pmapper.Metadata
	err := mapper.UnmarshalJSONMap(metadataMap, &metadata)
	if err != nil {
		return nil, nil, service.WrapError(service.ErrInvalidInput, err)
	}

	matches, err := common.MatchOperations(operations)
	if err != nil {
		return nil, nil, service.WrapError(service.ErrInvalidInput, err)
	}

	opType := matches[0].Operations[0].Type
	tx, signers, err := pmapper.BuildTx(opType, matches, metadata, p.codec, p.avaxAssetID)
	if err != nil {
		return nil, nil, service.WrapError(service.ErrInvalidInput, err)
	}

	return &pTx{
		Tx:           tx,
		Codec:        p.codec,
		CodecVersion: p.codecVersion,
	}, signers, nil
}

type pTxParser struct {
	hrp      string
	chainIDs map[string]string
}

func (p pTxParser) ParseTx(tx *common.RosettaTx, inputAddresses map[string]*types.AccountIdentifier) ([]*types.Operation, error) {
	pTx, ok := tx.Tx.(*pTx)
	if !ok {
		return nil, errInvalidTransaction
	}

	parser := pmapper.NewTxParser(true, p.hrp, p.chainIDs, inputAddresses, nil)
	transactions, err := parser.Parse(pTx.Tx.UnsignedTx)
	if err != nil {
		return nil, err
	}

	return transactions.Operations, nil
}
