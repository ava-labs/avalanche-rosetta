package service

import (
	"context"
	"log"
	"math/big"
	"strings"

	"github.com/coinbase/rosetta-sdk-go/server"
	"github.com/coinbase/rosetta-sdk-go/types"

	ethcommon "github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"

	"github.com/figment-networks/avalanche-rosetta/client"
	"github.com/figment-networks/avalanche-rosetta/mapper"
)

// BlockService implements the /block/* endpoints
type BlockService struct {
	config *Config
	evm    *client.EvmClient
}

// NewBlockService returns a new block servicer
func NewBlockService(config *Config, evmClient *client.EvmClient) server.BlockAPIServicer {
	return &BlockService{
		config: config,
		evm:    evmClient,
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
		block                 *ethtypes.Block
		transactions          []*types.Transaction
		err                   error
	)

	// Fetch block by hash
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
		return nil, errBlockFetchFailed
	}

	blockIdentifier = &types.BlockIdentifier{
		Index: block.Number().Int64(),
		Hash:  block.Hash().String(),
	}

	// Fetch the parent block since we dont have full info on the block itself
	parentBlock, err := s.evm.HeaderByHash(context.Background(), block.ParentHash())
	if err == nil {
		parentBlockIdentifier = &types.BlockIdentifier{
			Index: parentBlock.Number.Int64(),
			Hash:  parentBlock.Hash().String(),
		}
	} else {
		log.Println("parent block fetch failed:", err)
		return nil, errBlockFetchFailed
	}

	for _, tx := range block.Transactions() {
		msg, err := tx.AsMessage(s.config.Signer())
		if err != nil {
			log.Println("tx message error:", err)
			return nil, errInternalError
		}

		receipt, err := s.evm.TransactionReceipt(context.Background(), tx.Hash())
		if err != nil {
			if strings.Contains(err.Error(), "not found") {
				log.Println("cant find receipt for", tx.Hash())
				continue
			}

			log.Println("tx receipt fetch error:", err)
			return nil, errInternalError
		}

		transaction, err := mapper.Transaction(block.Header(), tx, &msg, receipt)
		if err != nil {
			log.Println("transaction mapper error:", err)
			return nil, errInternalError
		}

		transactions = append(transactions, transaction)
	}

	return &types.BlockResponse{
		Block: &types.Block{
			BlockIdentifier:       blockIdentifier,
			ParentBlockIdentifier: parentBlockIdentifier,
			Timestamp:             int64(block.Time() * 1000),
			Transactions:          transactions,
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

	hash := ethcommon.HexToHash(request.TransactionIdentifier.Hash)
	tx, pending, err := s.evm.TransactionByHash(context.Background(), hash)
	if err != nil {
		log.Println("tx fetch error:", err)
		return nil, errInternalError
	}
	if pending {
		log.Println("tx pending:", tx.Hash().String())
		return nil, errInternalError
	}

	msg, err := tx.AsMessage(s.config.Signer())
	if err != nil {
		return nil, errInternalError
	}

	receipt, err := s.evm.TransactionReceipt(context.Background(), tx.Hash())
	if err != nil {
		log.Println("tx receipt fetch error:", err)
		return nil, errInternalError
	}

	header, err := s.evm.HeaderByNumber(context.Background(), receipt.BlockNumber)
	if err != nil {
		log.Println("block header fetch error:", err)
		return nil, errInternalError
	}

	transaction, err := mapper.Transaction(header, tx, &msg, receipt)
	if err != nil {
		log.Println("tx mapper error:", err)
		return nil, errInternalError
	}

	return &types.BlockTransactionResponse{
		Transaction: transaction,
	}, nil
}
