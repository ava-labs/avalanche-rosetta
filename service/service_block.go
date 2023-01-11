package service

import (
	"context"
	"math/big"
	"strings"

	"github.com/coinbase/rosetta-sdk-go/server"
	"github.com/coinbase/rosetta-sdk-go/types"
	"github.com/coinbase/rosetta-sdk-go/utils"

	corethTypes "github.com/chain4travel/caminoethvm/core/types"
	ethcommon "github.com/ethereum/go-ethereum/common"

	"github.com/chain4travel/camino-rosetta/client"
	"github.com/chain4travel/camino-rosetta/mapper"
)

// BlockService implements the /block/* endpoints
type BlockService struct {
	config *Config
	client client.Client

	genesisBlock *types.Block
}

// NewBlockService returns a new block servicer
func NewBlockService(config *Config, c client.Client) server.BlockAPIServicer {
	return &BlockService{
		config:       config,
		client:       c,
		genesisBlock: makeGenesisBlock(config.GenesisBlockHash),
	}
}

// Block implements the /block endpoint
func (s *BlockService) Block(
	ctx context.Context,
	request *types.BlockRequest,
) (*types.BlockResponse, *types.Error) {
	if s.config.IsOfflineMode() {
		return nil, errUnavailableOffline
	}

	if request.BlockIdentifier == nil {
		return nil, errBlockInvalidInput
	}
	if request.BlockIdentifier.Hash == nil && request.BlockIdentifier.Index == nil {
		return nil, errBlockInvalidInput
	}

	if s.isGenesisBlockRequest(request.BlockIdentifier) {
		return &types.BlockResponse{
			Block: s.genesisBlock,
		}, nil
	}

	var (
		blockIdentifier       *types.BlockIdentifier
		parentBlockIdentifier *types.BlockIdentifier
		block                 *corethTypes.Block
		err                   error
	)

	if hash := request.BlockIdentifier.Hash; hash != nil {
		block, err = s.client.BlockByHash(ctx, ethcommon.HexToHash(*hash))
	} else if index := request.BlockIdentifier.Index; block == nil && index != nil {
		block, err = s.client.BlockByNumber(ctx, big.NewInt(*index))
	}
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return nil, errBlockNotFound
		}
		return nil, wrapError(errClientError, err)
	}

	blockIdentifier = &types.BlockIdentifier{
		Index: block.Number().Int64(),
		Hash:  block.Hash().String(),
	}

	if block.ParentHash().String() != s.config.GenesisBlockHash {
		parentBlock, err := s.client.HeaderByHash(ctx, block.ParentHash())
		if err != nil {
			return nil, wrapError(errClientError, err)
		}

		parentBlockIdentifier = &types.BlockIdentifier{
			Index: parentBlock.Number.Int64(),
			Hash:  parentBlock.Hash().String(),
		}
	} else {
		parentBlockIdentifier = s.genesisBlock.BlockIdentifier
	}

	transactions, terr := s.fetchTransactions(ctx, block)
	if terr != nil {
		return nil, terr
	}

	crosstx, terr := s.parseCrossChainTransactions(block)
	if terr != nil {
		return nil, terr
	}

	return &types.BlockResponse{
		Block: &types.Block{
			BlockIdentifier:       blockIdentifier,
			ParentBlockIdentifier: parentBlockIdentifier,
			Timestamp:             int64(block.Time() * utils.MillisecondsInSecond),
			Transactions:          append(transactions, crosstx...),
			Metadata:              mapper.BlockMetadata(block),
		},
	}, nil
}

// BlockTransaction implements the /block/transaction endpoint.
func (s *BlockService) BlockTransaction(
	ctx context.Context,
	request *types.BlockTransactionRequest,
) (*types.BlockTransactionResponse, *types.Error) {
	if s.config.IsOfflineMode() {
		return nil, errUnavailableOffline
	}

	if request.BlockIdentifier == nil {
		return nil, wrapError(errInvalidInput, "block identifier is not provided")
	}

	header, err := s.client.HeaderByHash(ctx, ethcommon.HexToHash(request.BlockIdentifier.Hash))
	if err != nil {
		return nil, wrapError(errClientError, err)
	}

	hash := ethcommon.HexToHash(request.TransactionIdentifier.Hash)
	tx, pending, err := s.client.TransactionByHash(ctx, hash)
	if err != nil {
		return nil, wrapError(errClientError, err)
	}
	if pending {
		return nil, nil
	}

	trace, flattened, err := s.client.TraceTransaction(ctx, tx.Hash().String())
	if err != nil {
		return nil, wrapError(errClientError, err)
	}

	transaction, terr := s.fetchTransaction(ctx, tx, header, trace, flattened)
	if terr != nil {
		return nil, terr
	}

	return &types.BlockTransactionResponse{
		Transaction: transaction,
	}, nil
}

func (s *BlockService) fetchTransactions(
	ctx context.Context,
	block *corethTypes.Block,
) ([]*types.Transaction, *types.Error) {
	transactions := []*types.Transaction{}

	trace, flattened, err := s.client.TraceBlockByHash(ctx, block.Hash().String())
	if err != nil {
		return nil, wrapError(errClientError, err)
	}

	for i, tx := range block.Transactions() {
		transaction, terr := s.fetchTransaction(ctx, tx, block.Header(), trace[i], flattened[i])
		if terr != nil {
			return nil, terr
		}

		transactions = append(transactions, transaction)
	}

	return transactions, nil
}

func (s *BlockService) fetchTransaction(
	ctx context.Context,
	tx *corethTypes.Transaction,
	header *corethTypes.Header,
	trace *client.Call,
	flattened []*client.FlatCall,
) (*types.Transaction, *types.Error) {
	msg, err := tx.AsMessage(s.config.Signer(), header.BaseFee)
	if err != nil {
		return nil, wrapError(errClientError, err)
	}

	receipt, err := s.client.TransactionReceipt(ctx, tx.Hash())
	if err != nil {
		return nil, wrapError(errClientError, err)
	}

	transaction, err := mapper.Transaction(header, tx, &msg, receipt, trace, flattened, s.client, s.config.IsAnalyticsMode(), s.config.TokenWhiteList, s.config.IndexUnknownTokens)
	if err != nil {
		return nil, wrapError(errInternalError, err)
	}

	return transaction, nil
}

func (s *BlockService) parseCrossChainTransactions(
	block *corethTypes.Block,
) ([]*types.Transaction, *types.Error) {
	result := []*types.Transaction{}

	crossTxs, err := mapper.CrossChainTransactions(s.config.CamAssetID, block, s.config.AP5Activation)
	if err != nil {
		return nil, wrapError(errInternalError, err)
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

func (s *BlockService) isGenesisBlockRequest(id *types.PartialBlockIdentifier) bool {
	if number := id.Index; number != nil {
		return *number == s.genesisBlock.BlockIdentifier.Index
	}
	if hash := id.Hash; hash != nil {
		return *hash == s.genesisBlock.BlockIdentifier.Hash
	}
	return false
}
