package pchain

import (
	"context"

	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/avalanchego/vms/platformvm/txs"
	"github.com/coinbase/rosetta-sdk-go/types"

	"github.com/ava-labs/avalanche-rosetta/service"

	"github.com/ava-labs/avalanchego/api"
	"github.com/ava-labs/avalanchego/utils/formatting"
	"github.com/ava-labs/avalanchego/vms/components/avax"
	"golang.org/x/sync/errgroup"

	pmapper "github.com/ava-labs/avalanche-rosetta/mapper/pchain"
)

// Block implements the /block endpoint
func (b *Backend) Block(ctx context.Context, request *types.BlockRequest) (*types.BlockResponse, *types.Error) {
	var blockIndex int64
	if request.BlockIdentifier.Index != nil {
		blockIndex = *request.BlockIdentifier.Index
	}

	var hash string
	if request.BlockIdentifier.Hash != nil {
		hash = *request.BlockIdentifier.Hash
	}

	var (
		blkIdentifier       *types.BlockIdentifier
		parentBlkIdentifier *types.BlockIdentifier
		blkTime             int64
		rTxs                []*types.Transaction
		metadata            map[string]interface{}
	)

	isGenesisReq, err := b.isGenesisBlockRequest(blockIndex, hash)
	switch {
	case err != nil:
		// avalanchego node may be not ready or reachable
		return nil, service.WrapError(service.ErrClientError, err)

	case isGenesisReq:
		genesisTxs, err := b.getFullGenesisTxs()
		if err != nil {
			return nil, service.WrapError(service.ErrClientError, err)
		}
		rosettaTxs, err := pmapper.ParseRosettaTxs(b.txParserCfg, genesisTxs, nil)
		if err != nil {
			return nil, service.WrapError(service.ErrClientError, err)
		}
		genesisBlock := b.getGenesisBlock()

		blkIdentifier = b.getGenesisIdentifier()

		// Parent block identifier of genesis block is set to itself instead of the hash of the genesis state
		// This is done as the genesis state hash cannot be used as a transaction id for the /block apis
		// and the operations found in the genesis state are returned as operations of the genesis block.
		parentBlkIdentifier = b.getGenesisIdentifier()
		blkTime = genesisBlock.Timestamp
		rTxs = rosettaTxs
		metadata = map[string]interface{}{
			pmapper.MetadataMessage: genesisBlock.Message,
		}

	default:
		block, err := b.indexerParser.ParseNonGenesisBlock(ctx, hash, uint64(blockIndex))
		if err != nil {
			return nil, service.WrapError(service.ErrClientError, err)
		}
		blockIndex = int64(block.Height)

		blkDeps, err := b.fetchBlkDependencies(ctx, block.Txs)
		if err != nil {
			return nil, service.WrapError(service.ErrInternalError, err)
		}

		rosettaTxs, err := pmapper.ParseRosettaTxs(b.txParserCfg, block.Txs, blkDeps)
		if err != nil {
			return nil, service.WrapError(service.ErrInternalError, err)
		}

		blkIdentifier = &types.BlockIdentifier{
			Index: blockIndex,
			Hash:  block.BlockID.String(),
		}
		parentBlkIdentifier = &types.BlockIdentifier{
			Index: blockIndex - 1,
			Hash:  block.ParentID.String(),
		}
		blkTime = block.Timestamp
		rTxs = rosettaTxs
		metadata = nil
	}

	resp := &types.BlockResponse{
		Block: &types.Block{
			BlockIdentifier:       blkIdentifier,
			ParentBlockIdentifier: parentBlkIdentifier,
			Timestamp:             blkTime,
			Transactions:          rTxs,
			Metadata:              metadata,
		},
	}
	return resp, nil
}

// BlockTransaction implements the /block/transaction endpoint.
func (b *Backend) BlockTransaction(ctx context.Context, request *types.BlockTransactionRequest) (*types.BlockTransactionResponse, *types.Error) {
	var (
		targetTxs     []*txs.Tx
		dependencyTxs pmapper.BlockTxDependencies
	)

	isGenesisReq, err := b.isGenesisBlockRequest(request.BlockIdentifier.Index, request.BlockIdentifier.Hash)
	switch {
	case err != nil:
		// avalanchego node may be not ready or reachable
		return nil, service.WrapError(service.ErrClientError, err)

	case isGenesisReq:
		genesisTxs, err := b.getFullGenesisTxs()
		if err != nil {
			return nil, service.WrapError(service.ErrClientError, err)
		}

		targetTxs = genesisTxs
		dependencyTxs = nil

	default:
		block, err := b.indexerParser.ParseNonGenesisBlock(ctx, request.BlockIdentifier.Hash, uint64(request.BlockIdentifier.Index))
		if err != nil {
			return nil, service.WrapError(service.ErrClientError, err)
		}
		deps, err := b.fetchBlkDependencies(ctx, block.Txs)
		if err != nil {
			return nil, service.WrapError(service.ErrInternalError, err)
		}

		targetTxs = block.Txs
		dependencyTxs = deps
	}

	rosettaTxs, err := pmapper.ParseRosettaTxs(b.txParserCfg, targetTxs, dependencyTxs)
	if err != nil {
		return nil, service.WrapError(service.ErrInternalError, err)
	}
	for _, rTx := range rosettaTxs {
		if rTx.TransactionIdentifier.Hash == request.TransactionIdentifier.Hash {
			return &types.BlockTransactionResponse{
				Transaction: rTx,
			}, nil
		}
	}

	return nil, service.ErrTransactionNotFound
}

func (b *Backend) fetchBlkDependencies(ctx context.Context, txs []*txs.Tx) (pmapper.BlockTxDependencies, error) {
	blockDeps := make(pmapper.BlockTxDependencies)
	depsTxIDs := []ids.ID{}
	for _, tx := range txs {
		inputTxsIds, err := blockDeps.GetTxDependenciesIDs(tx.Unsigned)
		if err != nil {
			return nil, err
		}
		depsTxIDs = append(depsTxIDs, inputTxsIds...)
	}

	dependencyTxChan := make(chan *pmapper.SingleTxDependency, len(depsTxIDs))
	eg, ctx := errgroup.WithContext(ctx)

	for _, txID := range depsTxIDs {
		txID := txID
		eg.Go(func() error {
			return b.fetchDependencyTx(ctx, txID, dependencyTxChan)
		})
	}
	if err := eg.Wait(); err != nil {
		return nil, err
	}
	close(dependencyTxChan)

	for dTx := range dependencyTxChan {
		blockDeps[dTx.Tx.ID()] = dTx
	}

	return blockDeps, nil
}

func (b *Backend) fetchDependencyTx(ctx context.Context, txID ids.ID, out chan *pmapper.SingleTxDependency) error {
	// Genesis state contains initial allocation UTXOs. These are not technically part of a transaction.
	// As a result, their UTXO id uses zero value transaction id. In that case, return genesis allocation data
	if txID == ids.Empty {
		allocationTx, err := b.buildGenesisAllocationTx()
		if allocationTx != nil {
			out <- &pmapper.SingleTxDependency{
				Tx: allocationTx,
			}
		}
		return err
	}

	txBytes, err := b.pClient.GetTx(ctx, txID)
	if err != nil {
		return err
	}

	tx, err := txs.Parse(txs.Codec, txBytes)
	if err != nil {
		return err
	}

	utxoBytes, err := b.pClient.GetRewardUTXOs(ctx, &api.GetTxArgs{
		TxID:     txID,
		Encoding: formatting.Hex,
	})
	if err != nil {
		return err
	}

	utxos := []*avax.UTXO{}
	for _, bytes := range utxoBytes {
		utxo := avax.UTXO{}
		_, err = b.codec.Unmarshal(bytes, &utxo)
		if err != nil {
			return err
		}
		utxos = append(utxos, &utxo)
	}
	out <- &pmapper.SingleTxDependency{
		Tx:          tx,
		RewardUTXOs: utxos,
	}

	return nil
}
