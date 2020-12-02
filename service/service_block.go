package service

import (
	"context"
	"math/big"
	"strings"

	"github.com/ava-labs/coreth/ethclient"
	"github.com/coinbase/rosetta-sdk-go/server"
	"github.com/coinbase/rosetta-sdk-go/types"
	ethcommon "github.com/ethereum/go-ethereum/common"

	corethTypes "github.com/ava-labs/coreth/core/types"

	"github.com/figment-networks/avalanche-rosetta/client"
	"github.com/figment-networks/avalanche-rosetta/mapper"
)

// BlockService implements the /block/* endpoints
type BlockService struct {
	config *Config
	debug  *client.DebugClient
	evm    *ethclient.Client
}

// NewBlockService returns a new block servicer
func NewBlockService(config *Config, evmClient *ethclient.Client, debugClient *client.DebugClient) server.BlockAPIServicer {
	return &BlockService{
		config: config,
		evm:    evmClient,
		debug:  debugClient,
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

	var (
		blockIdentifier       *types.BlockIdentifier
		parentBlockIdentifier *types.BlockIdentifier
		block                 *corethTypes.Block
		err                   error
	)

	if hash := request.BlockIdentifier.Hash; hash != nil {
		block, err = s.evm.BlockByHash(context.Background(), ethcommon.HexToHash(*hash))
	} else {
		if index := request.BlockIdentifier.Index; block == nil && index != nil {
			block, err = s.evm.BlockByNumber(context.Background(), big.NewInt(*index))
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

	if block.ParentHash().String() != genesisBlockHash {
		parentBlock, err := s.evm.HeaderByHash(context.Background(), block.ParentHash())
		if err == nil {
			parentBlockIdentifier = &types.BlockIdentifier{
				Index: parentBlock.Number.Int64(),
				Hash:  parentBlock.Hash().String(),
			}
		} else {
			return nil, wrapError(errClientError, err)
		}
	} else {
		parentBlockIdentifier = &types.BlockIdentifier{
			Index: 0,
			Hash:  genesisBlockHash,
		}
	}

	crossTransactions, err := mapper.CrossChainTransactions(block)
	if err != nil {
		return nil, wrapError(errInternalError, err)
	}

	transactions, terr := s.fetchTransactions(ctx, block)
	if err != nil {
		return nil, terr
	}

	return &types.BlockResponse{
		Block: &types.Block{
			BlockIdentifier:       blockIdentifier,
			ParentBlockIdentifier: parentBlockIdentifier,
			Timestamp:             int64(block.Time() * 1000),
			Transactions:          append(crossTransactions, transactions...),
			Metadata: map[string]interface{}{
				"gas_limit":  block.GasLimit(),
				"gas_used":   block.GasUsed(),
				"difficulty": block.Difficulty(),
				"nonce":      block.Nonce(),
				"size":       block.Size().String(),
			},
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

	header, err := s.evm.HeaderByHash(ctx, ethcommon.HexToHash(request.BlockIdentifier.Hash))
	if err != nil {
		return nil, wrapError(errClientError, err)
	}

	hash := ethcommon.HexToHash(request.TransactionIdentifier.Hash)
	tx, pending, err := s.evm.TransactionByHash(context.Background(), hash)
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

	receipt, err := s.evm.TransactionReceipt(context.Background(), tx.Hash())
	if err != nil {
		return nil, wrapError(errClientError, err)
	}

	trace, err := s.debug.TraceTransaction(tx.Hash().String())
	if err != nil {
		return nil, wrapError(errClientError, err)
	}

	transaction, err := mapper.Transaction(header, tx, &msg, receipt, trace)
	if err != nil {
		return nil, wrapError(errInternalError, err)
	}

	return transaction, nil
}
