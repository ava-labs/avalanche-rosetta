package pchain

import (
	"context"
	"errors"

	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/avalanchego/vms/platformvm"
	"github.com/coinbase/rosetta-sdk-go/types"

	"github.com/ava-labs/avalanche-rosetta/service"
	"github.com/ava-labs/avalanche-rosetta/service/backend/pchain/indexer"
)

var (
	errMissingBlockIndexHash = errors.New("a positive block index, a block hash or both must be specified")
	errMismatchedHeight      = errors.New("provided block height does not match height of the block with given hash")
)

func (b *Backend) Block(ctx context.Context, request *types.BlockRequest) (*types.BlockResponse, *types.Error) {
	return nil, service.ErrNotImplemented
}

func (b *Backend) BlockTransaction(ctx context.Context, request *types.BlockTransactionRequest) (*types.BlockTransactionResponse, *types.Error) {
	return nil, service.ErrNotImplemented
}

func (b *Backend) getBlockDetails(ctx context.Context, index int64, hash string) (*indexer.ParsedBlock, error) {
	if index <= 0 && hash == "" {
		return nil, errMissingBlockIndexHash
	}

	blockHeight := uint64(index)

	// Extract block id from hash parameter if it is non-empty, or from index if stated
	if hash != "" {
		height, err := b.getBlockHeight(ctx, hash)
		if err != nil {
			return nil, err
		}

		if blockHeight > 0 && height != blockHeight {
			return nil, errMismatchedHeight
		}
		blockHeight = height
	}

	return b.indexerParser.ParseBlockAtIndex(ctx, blockHeight)
}

func (b *Backend) getBlockHeight(ctx context.Context, hash string) (uint64, error) {
	blockID, err := ids.FromString(hash)
	if err != nil {
		return 0, err
	}

	blockBytes, err := b.pClient.GetBlock(ctx, blockID)
	if err != nil {
		return 0, err
	}

	var block platformvm.Block
	_, err = b.codec.Unmarshal(blockBytes, &block)
	if err != nil {
		return 0, err
	}

	return block.Height(), nil
}
