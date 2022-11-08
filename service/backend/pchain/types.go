package pchain

import (
	"errors"

	"github.com/ava-labs/avalanchego/codec"
	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/avalanchego/utils/hashing"
	"github.com/ava-labs/avalanchego/vms/platformvm/txs"
	"github.com/coinbase/rosetta-sdk-go/types"

	"github.com/ava-labs/avalanche-rosetta/constants"
	"github.com/ava-labs/avalanche-rosetta/mapper"
	pmapper "github.com/ava-labs/avalanche-rosetta/mapper/pchain"
	"github.com/ava-labs/avalanche-rosetta/service"
	"github.com/ava-labs/avalanche-rosetta/service/backend/common"
)

var (
	_ common.AvaxTx    = &pTx{}
	_ common.TxBuilder = &pTxBuilder{}
	_ common.TxParser  = &pTxParser{}

	errInvalidTransaction = errors.New("invalid transaction")
)

// AccountBalance contains P-chain account balances
type AccountBalance struct {
	Total              uint64
	Unlocked           uint64
	Staked             uint64
	LockedStakeable    uint64
	LockedNotStakeable uint64
}

type pTx struct {
	Tx           *txs.Tx
	Codec        codec.Manager
	CodecVersion uint16
}

func (p *pTx) Initialize() error {
	if p.Tx == nil {
		return common.ErrNoTxGiven
	}
	return p.Tx.Sign(p.Codec, nil)
}

func (p *pTx) Marshal() ([]byte, error) {
	return p.Codec.Marshal(p.CodecVersion, p.Tx)
}

func (p *pTx) Unmarshal(bytes []byte) error {
	tx := txs.Tx{}
	_, err := p.Codec.Unmarshal(bytes, &tx)
	if err != nil {
		return err
	}
	if err := tx.Sign(p.Codec, nil); err != nil {
		return err
	}
	p.Tx = &tx

	return p.Initialize()
}

func (p *pTx) SigningPayload() []byte {
	return hashing.ComputeHash256(p.Tx.Unsigned.Bytes())
}

func (p *pTx) Hash() ids.ID {
	return p.Tx.ID()
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
	hrp         string
	chainIDs    map[ids.ID]constants.ChainIDAlias
	avaxAssetID ids.ID
}

func (p pTxParser) ParseTx(tx *common.RosettaTx, inputAddresses map[string]*types.AccountIdentifier) ([]*types.Operation, error) {
	pTx, ok := tx.Tx.(*pTx)
	if !ok {
		return nil, errInvalidTransaction
	}

	parserCfg := pmapper.TxParserConfig{
		IsConstruction: true,
		Hrp:            p.hrp,
		ChainIDs:       p.chainIDs,
		AvaxAssetID:    p.avaxAssetID,
		PChainClient:   nil,
	}
	parser, err := pmapper.NewTxParser(parserCfg, inputAddresses, nil)
	if err != nil {
		return nil, err
	}

	transactions, err := parser.Parse(pTx.Tx)
	if err != nil {
		return nil, err
	}

	return transactions.Operations, nil
}
