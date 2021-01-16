package service

import (
	"context"
	"math/big"
	"strings"
	"sync"

	"github.com/coinbase/rosetta-sdk-go/server"
	"github.com/coinbase/rosetta-sdk-go/types"

	corethTypes "github.com/ava-labs/coreth/core/types"
	ethcommon "github.com/ethereum/go-ethereum/common"

	"github.com/figment-networks/avalanche-rosetta/client"
	"github.com/figment-networks/avalanche-rosetta/mapper"
)

// BlockService implements the /block/* endpoints
type BlockService struct {
	config *Config
	client client.Client

	genesisBlock *types.Block
	assets       map[string]*client.Asset
	assetsLock   sync.Mutex
}

// NewBlockService returns a new block servicer
func NewBlockService(config *Config, rcpClient client.Client) server.BlockAPIServicer {
	return &BlockService{
		config:       config,
		client:       rcpClient,
		assets:       map[string]*client.Asset{},
		assetsLock:   sync.Mutex{},
		genesisBlock: makeGenesisBlock(config.GenesisBlockHash),
	}
}

// Block implements the /block endpoint
func (s *BlockService) Block(ctx context.Context, request *types.BlockRequest) (*types.BlockResponse, *types.Error) {
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
		block, err = s.client.BlockByHash(context.Background(), ethcommon.HexToHash(*hash))
	} else {
		if index := request.BlockIdentifier.Index; block == nil && index != nil {
			block, err = s.client.BlockByNumber(context.Background(), big.NewInt(*index))
		}
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
		parentBlock, err := s.client.HeaderByHash(context.Background(), block.ParentHash())
		if err == nil {
			parentBlockIdentifier = &types.BlockIdentifier{
				Index: parentBlock.Number.Int64(),
				Hash:  parentBlock.Hash().String(),
			}
		} else {
			return nil, wrapError(errClientError, err)
		}
	} else {
		parentBlockIdentifier = s.genesisBlock.BlockIdentifier
	}

	transactions, terr := s.fetchTransactions(ctx, block)
	if terr != nil {
		return nil, terr
	}

	crosstx, terr := s.fetchCrossChainTransactions(ctx, block)
	if terr != nil {
		return nil, terr
	}

	return &types.BlockResponse{
		Block: &types.Block{
			BlockIdentifier:       blockIdentifier,
			ParentBlockIdentifier: parentBlockIdentifier,
			Timestamp:             int64(block.Time() * 1000),
			Transactions:          append(transactions, crosstx...),
			Metadata:              mapper.BlockMetadata(block),
		},
	}, nil
}

// BlockTransaction implements the /block/transaction endpoint.
func (s *BlockService) BlockTransaction(ctx context.Context, request *types.BlockTransactionRequest) (*types.BlockTransactionResponse, *types.Error) {
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
	tx, pending, err := s.client.TransactionByHash(context.Background(), hash)
	if err != nil {
		return nil, wrapError(errClientError, err)
	}
	if pending {
		return nil, nil
	}

	transaction, terr := s.fetchTransaction(ctx, tx, header)
	if err != nil {
		return nil, terr
	}

	return &types.BlockTransactionResponse{
		Transaction: transaction,
	}, nil
}

func (s *BlockService) fetchTransactions(ctx context.Context, block *corethTypes.Block) ([]*types.Transaction, *types.Error) {
	transactions := []*types.Transaction{}

	for _, tx := range block.Transactions() {
		transaction, err := s.fetchTransaction(ctx, tx, block.Header())
		if err != nil {
			return nil, err
		}
		transactions = append(transactions, transaction)
	}

	return transactions, nil
}

func (s *BlockService) fetchTransaction(ctx context.Context, tx *corethTypes.Transaction, header *corethTypes.Header) (*types.Transaction, *types.Error) {
	msg, err := tx.AsMessage(s.config.Signer())
	if err != nil {
		return nil, wrapError(errClientError, err)
	}

	receipt, err := s.client.TransactionReceipt(context.Background(), tx.Hash())
	if err != nil {
		return nil, wrapError(errClientError, err)
	}

	trace, err := s.client.TraceTransaction(ctx, tx.Hash().String())
	if err != nil {
		return nil, wrapError(errClientError, err)
	}

	transaction, err := mapper.Transaction(header, tx, &msg, receipt, trace)
	if err != nil {
		return nil, wrapError(errInternalError, err)
	}

	return transaction, nil
}

func (s *BlockService) fetchCrossChainTransactions(ctx context.Context, block *corethTypes.Block) ([]*types.Transaction, *types.Error) {
	result := []*types.Transaction{}

	crossTxs, err := mapper.CrossChainTransactions(block)
	if err != nil {
		return nil, wrapError(errInternalError, err)
	}

	for _, tx := range crossTxs {
		// Skip empty import/export transactions
		if len(tx.Operations) == 0 {
			continue
		}

		selectedOps := []*types.Operation{}
		for _, op := range tx.Operations {
			// Determine currency symbol from the tx asset ID
			asset, err := s.lookupAsset(ctx, op.Amount.Currency.Symbol)
			if err != nil {
				return nil, wrapError(errClientError, err)
			}

			// Select operations with AVAX currency
			if asset.Symbol == mapper.AvaxCurrency.Symbol {
				op.Amount.Currency = mapper.AvaxCurrency
				selectedOps = append(selectedOps, op)
			}
		}

		if len(selectedOps) > 0 {
			tx.Operations = selectedOps
			result = append(result, tx)
		}
	}

	return result, nil
}

func (s *BlockService) lookupAsset(ctx context.Context, id string) (*client.Asset, error) {
	if asset, ok := s.assets[id]; ok {
		return asset, nil
	}

	asset, err := s.client.AssetDescription(ctx, id)
	if err != nil {
		return nil, err
	}

	s.assetsLock.Lock()
	s.assets[id] = asset
	s.assetsLock.Unlock()

	return asset, nil
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
