package pchain

import (
	"context"

	"github.com/ava-labs/avalanchego/api"
	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/avalanchego/utils/formatting"
	"github.com/ava-labs/avalanchego/vms/components/avax"
	"github.com/ava-labs/avalanchego/vms/platformvm/txs"
	"github.com/coinbase/rosetta-sdk-go/types"
	"golang.org/x/sync/errgroup"

	"github.com/ava-labs/avalanche-rosetta/service"

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

// depRequest pairs a dependency tx ID with the tx ID used to query reward UTXOs.
// For most tx types rewardUTXOTxID is ids.Empty, meaning the dep tx ID is used.
// For RewardAutoRenewedValidatorTx, AvalancheGo stores reward UTXOs under the
// reward tx's own ID (not the referenced staking tx ID), so rewardUTXOTxID is
// set to the reward tx's ID.
type depRequest struct {
	depTxID        ids.ID
	rewardUTXOTxID ids.ID
}

func (b *Backend) fetchBlkDependencies(ctx context.Context, blkTxs []*txs.Tx) (pmapper.BlockTxDependencies, error) {
	blockDeps := make(pmapper.BlockTxDependencies)

	var depReqs []depRequest
	// rewardTxIDs maps stakingTxID → rewardTxID for RewardAutoRenewedValidatorTx deps.
	// Used after fetching to also index each dep under its reward tx ID so that
	// isMultisig can resolve reward UTXOs when they are spent in a later block.
	rewardTxIDs := make(map[ids.ID]ids.ID)
	for _, tx := range blkTxs {
		if utx, ok := tx.Unsigned.(*txs.RewardAutoRenewedValidatorTx); ok {
			// Reward UTXOs are stored under the reward tx's own ID in AvalancheGo,
			// not under the staking tx ID (utx.TxID). Fetch the staking tx for
			// validator metadata, but query reward UTXOs with the reward tx ID.
			// RewardAutoRenewedValidatorTx has no BaseTx inputs (InputIDs returns nil),
			// so bypassing GetTxDependenciesIDs is safe for the current protocol.
			rewardTxID := tx.ID()
			rewardTxIDs[utx.TxID] = rewardTxID
			depReqs = append(depReqs, depRequest{
				depTxID:        utx.TxID,
				rewardUTXOTxID: rewardTxID,
			})
			continue
		}
		inputTxIDs, err := pmapper.GetTxDependenciesIDs(tx.Unsigned)
		if err != nil {
			return nil, err
		}
		for _, id := range inputTxIDs {
			depReqs = append(depReqs, depRequest{depTxID: id})
		}
	}

	dependencyTxChan := make(chan *pmapper.SingleTxDependency, len(depReqs))
	eg, ctx := errgroup.WithContext(ctx)

	for _, req := range depReqs {
		req := req
		eg.Go(func() error {
			return b.fetchDependencyTx(ctx, req.depTxID, req.rewardUTXOTxID, dependencyTxChan)
		})
	}
	if err := eg.Wait(); err != nil {
		return nil, err
	}
	close(dependencyTxChan)

	for dTx := range dependencyTxChan {
		blockDeps[dTx.Tx.ID()] = dTx
	}

	// Index reward deps under their reward tx ID as well, so that isMultisig
	// can look up the dep when a downstream tx spends a reward UTXO (whose
	// UTXOID.TxID is the reward tx ID, not the staking tx ID).
	for stakingTxID, rewardTxID := range rewardTxIDs {
		if dep, ok := blockDeps[stakingTxID]; ok {
			blockDeps[rewardTxID] = dep
		}
	}

	return blockDeps, nil
}

// fetchDependencyTx fetches a dependency tx and its reward UTXOs.
// rewardUTXOTxID, when non-zero, overrides the tx ID used to query GetRewardUTXOs.
func (b *Backend) fetchDependencyTx(ctx context.Context, txID ids.ID, rewardUTXOTxID ids.ID, out chan *pmapper.SingleTxDependency) error {
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

	utxoQueryID := txID
	if rewardUTXOTxID != ids.Empty {
		utxoQueryID = rewardUTXOTxID
	}

	utxoBytes, err := b.pClient.GetRewardUTXOs(ctx, &api.GetTxArgs{
		TxID:     utxoQueryID,
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
