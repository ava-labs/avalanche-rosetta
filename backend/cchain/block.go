package cchain

import (
	"context"
	"math/big"
	"strings"

	"github.com/coinbase/rosetta-sdk-go/types"
	"github.com/coinbase/rosetta-sdk-go/utils"

	"github.com/ava-labs/avalanchego/ids"
	corethTypes "github.com/ava-labs/coreth/core/types"
	ethcommon "github.com/ethereum/go-ethereum/common"

	"github.com/ava-labs/avalanche-rosetta/backend"
	"github.com/ava-labs/avalanche-rosetta/client"
	"github.com/ava-labs/avalanche-rosetta/constants"
	cmapper "github.com/ava-labs/avalanche-rosetta/mapper/cchain"
)

// ShouldHandleRequest returns whether a given request should be handled by this backend
func (b *Backend) ShouldHandleRequest(req interface{}) bool {
	// currently this is the last backend polled hence ShouldHandleRequest is always true
	// TODO: cleanup
	return true
}

// Block implements the /block endpoint
func (b *Backend) Block(
	ctx context.Context,
	request *types.BlockRequest,
) (*types.BlockResponse, *types.Error) {
	if b.config.IsOfflineMode() {
		return nil, backend.ErrUnavailableOffline
	}

	if request.BlockIdentifier == nil {
		return nil, backend.ErrBlockInvalidInput
	}
	if request.BlockIdentifier.Hash == nil && request.BlockIdentifier.Index == nil {
		return nil, backend.ErrBlockInvalidInput
	}

	if b.isGenesisBlockRequest(request.BlockIdentifier) {
		return &types.BlockResponse{
			Block: b.genesisBlock,
		}, nil
	}

	var (
		blockIdentifier       *types.BlockIdentifier
		parentBlockIdentifier *types.BlockIdentifier
		block                 *corethTypes.Block
		err                   error
	)

	if hash := request.BlockIdentifier.Hash; hash != nil {
		block, err = b.client.BlockByHash(ctx, ethcommon.HexToHash(*hash))
	} else if index := request.BlockIdentifier.Index; block == nil && index != nil {
		block, err = b.client.BlockByNumber(ctx, big.NewInt(*index))
	}
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return nil, backend.ErrBlockNotFound
		}
		return nil, backend.WrapError(backend.ErrClientError, err)
	}

	blockIdentifier = &types.BlockIdentifier{
		Index: block.Number().Int64(),
		Hash:  block.Hash().String(),
	}

	if block.ParentHash().String() != b.config.GenesisBlockHash {
		parentBlock, err := b.client.HeaderByHash(ctx, block.ParentHash())
		if err != nil {
			return nil, backend.WrapError(backend.ErrClientError, err)
		}

		parentBlockIdentifier = &types.BlockIdentifier{
			Index: parentBlock.Number.Int64(),
			Hash:  parentBlock.Hash().String(),
		}
	} else {
		parentBlockIdentifier = b.genesisBlock.BlockIdentifier
	}

	transactions, terr := b.fetchTransactions(ctx, block)
	if terr != nil {
		return nil, terr
	}

	crosstx, terr := b.parseCrossChainTransactions(request.NetworkIdentifier, block)
	if terr != nil {
		return nil, terr
	}

	return &types.BlockResponse{
		Block: &types.Block{
			BlockIdentifier:       blockIdentifier,
			ParentBlockIdentifier: parentBlockIdentifier,
			Timestamp:             int64(block.Time() * utils.MillisecondsInSecond),
			Transactions:          append(transactions, crosstx...),
			Metadata:              cmapper.BlockMetadata(block),
		},
	}, nil
}

// BlockTransaction implements the /block/transaction endpoint.
func (b *Backend) BlockTransaction(
	ctx context.Context,
	request *types.BlockTransactionRequest,
) (*types.BlockTransactionResponse, *types.Error) {
	if b.config.IsOfflineMode() {
		return nil, backend.ErrUnavailableOffline
	}

	if request.BlockIdentifier == nil {
		return nil, backend.WrapError(backend.ErrInvalidInput, "block identifier is not provided")
	}

	header, err := b.client.HeaderByHash(ctx, ethcommon.HexToHash(request.BlockIdentifier.Hash))
	if err != nil {
		return nil, backend.WrapError(backend.ErrClientError, err)
	}

	hash := ethcommon.HexToHash(request.TransactionIdentifier.Hash)
	tx, pending, err := b.client.TransactionByHash(ctx, hash)
	if err != nil {
		return nil, backend.WrapError(backend.ErrClientError, err)
	}
	if pending {
		return nil, nil
	}

	trace, flattened, err := b.client.TraceTransaction(ctx, tx.Hash().String())
	if err != nil {
		return nil, backend.WrapError(backend.ErrClientError, err)
	}

	transaction, terr := b.fetchTransaction(ctx, tx, header, trace, flattened)
	if terr != nil {
		return nil, terr
	}

	return &types.BlockTransactionResponse{
		Transaction: transaction,
	}, nil
}

func (b *Backend) isGenesisBlockRequest(id *types.PartialBlockIdentifier) bool {
	if number := id.Index; number != nil {
		return *number == b.genesisBlock.BlockIdentifier.Index
	}
	if hash := id.Hash; hash != nil {
		return *hash == b.genesisBlock.BlockIdentifier.Hash
	}
	return false
}

func (b *Backend) fetchTransactions(
	ctx context.Context,
	block *corethTypes.Block,
) ([]*types.Transaction, *types.Error) {
	transactions := []*types.Transaction{}

	trace, flattened, err := b.client.TraceBlockByHash(ctx, block.Hash().String())
	if err != nil {
		return nil, backend.WrapError(backend.ErrClientError, err)
	}

	for i, tx := range block.Transactions() {
		transaction, terr := b.fetchTransaction(ctx, tx, block.Header(), trace[i], flattened[i])
		if terr != nil {
			return nil, terr
		}

		transactions = append(transactions, transaction)
	}

	return transactions, nil
}

func (b *Backend) fetchTransaction(
	ctx context.Context,
	tx *corethTypes.Transaction,
	header *corethTypes.Header,
	trace *client.Call,
	flattened []*client.FlatCall,
) (*types.Transaction, *types.Error) {
	msg, err := tx.AsMessage(b.config.Signer(), header.BaseFee)
	if err != nil {
		return nil, backend.WrapError(backend.ErrClientError, err)
	}

	receipt, err := b.client.TransactionReceipt(ctx, tx.Hash())
	if err != nil {
		return nil, backend.WrapError(backend.ErrClientError, err)
	}

	transaction, err := cmapper.Transaction(header, tx, &msg, receipt, trace, flattened, b.client, b.config.IsAnalyticsMode(), b.config.TokenWhiteList, b.config.IndexUnknownTokens)
	if err != nil {
		return nil, backend.WrapError(backend.ErrInternalError, err)
	}

	return transaction, nil
}

func (b *Backend) parseCrossChainTransactions(
	networkIdentifier *types.NetworkIdentifier,
	block *corethTypes.Block,
) ([]*types.Transaction, *types.Error) {
	result := []*types.Transaction{}

	// This map is used to create addresses for cross chain export outputs
	chainIDToAliasMapping := map[ids.ID]constants.ChainIDAlias{
		ids.Empty: constants.PChain,
	}
	crossTxs, err := cmapper.CrossChainTransactions(networkIdentifier, chainIDToAliasMapping, b.config.AvaxAssetID, block, b.config.AP5Activation)
	if err != nil {
		return nil, backend.WrapError(backend.ErrInternalError, err)
	}

	for _, tx := range crossTxs {
		// Skip empty import/export transactions
		if len(tx.Operations) == 0 {
			continue
		}

		result = append(result, tx)
	}

	return result, nil
}
