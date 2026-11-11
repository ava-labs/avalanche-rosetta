package cchainatomictx

import (
	"errors"

	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/avalanchego/utils/hashing"
	"github.com/ava-labs/avalanchego/vms/saevm/cchain/tx"
	"github.com/coinbase/rosetta-sdk-go/types"

	"github.com/ava-labs/avalanche-rosetta/mapper"
	"github.com/ava-labs/avalanche-rosetta/service"
	"github.com/ava-labs/avalanche-rosetta/service/backend/common"

	cmapper "github.com/ava-labs/avalanche-rosetta/mapper/cchainatomictx"
)

var (
	_ common.AvaxTx    = &cAtomicTx{}
	_ common.TxBuilder = &cAtomicTxBuilder{}
	_ common.TxParser  = &cAtomicTxParser{}
)

type cAtomicTx struct {
	Tx *tx.Tx

	// unsignedBytes is the canonical encoding of the unsigned transaction, which is what signers
	// sign. It is set by Initialize.
	unsignedBytes []byte
}

// Initialize caches the unsigned encoding of the transaction.
func (c *cAtomicTx) Initialize() error {
	if c.Tx == nil {
		return common.ErrNoTxGiven
	}
	unsignedBytes, err := tx.UnsignedBytes(c.Tx.Unsigned)
	if err != nil {
		return err
	}
	c.unsignedBytes = unsignedBytes
	return nil
}

func (c *cAtomicTx) Marshal() ([]byte, error) {
	return c.Tx.Bytes()
}

func (c *cAtomicTx) Unmarshal(bytes []byte) error {
	t, err := tx.Parse(bytes)
	if err != nil {
		return err
	}
	c.Tx = t

	return c.Initialize()
}

func (c *cAtomicTx) SigningPayload() []byte {
	return hashing.ComputeHash256(c.unsignedBytes)
}

func (c *cAtomicTx) Hash() ids.ID {
	return c.Tx.ID()
}

type cAtomicTxBuilder struct {
	avaxAssetID ids.ID
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
	atomicTx, signers, err := cmapper.BuildTx(opType, matches, cMetadata, c.avaxAssetID)
	if err != nil {
		return nil, nil, service.WrapError(service.ErrInternalError, err)
	}

	cTx := &cAtomicTx{Tx: atomicTx}
	if err := cTx.Initialize(); err != nil {
		return nil, nil, service.WrapError(service.ErrInternalError, err)
	}

	return cTx, signers, nil
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
	return parser.Parse(cTx.Tx)
}
