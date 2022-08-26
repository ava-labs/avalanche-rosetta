package pchain

import (
	"context"
	"errors"

	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/avalanchego/vms/platformvm"
	"github.com/ava-labs/avalanchego/vms/platformvm/stakeable"
	"github.com/ava-labs/avalanchego/vms/secp256k1fx"
	"github.com/coinbase/rosetta-sdk-go/types"

	"github.com/ava-labs/avalanche-rosetta/service"
	"github.com/ava-labs/avalanche-rosetta/service/backend/pchain/indexer"

	"github.com/ava-labs/avalanchego/api"
	"github.com/ava-labs/avalanchego/utils/formatting"
	"github.com/ava-labs/avalanchego/vms/components/avax"
	"golang.org/x/sync/errgroup"

	"github.com/ava-labs/avalanche-rosetta/mapper"
	pmapper "github.com/ava-labs/avalanche-rosetta/mapper/pchain"
	"github.com/ava-labs/avalanche-rosetta/service/backend/common"
)

var (
	errMissingBlockIndexHash = errors.New("a positive block index, a block hash or both must be specified")
	errMismatchedHeight      = errors.New("provided block height does not match height of the block with given hash")
	errTxInitialize          = errors.New("tx initialize error")
)

// Block implements the /block endpoint
func (b *Backend) Block(ctx context.Context, request *types.BlockRequest) (*types.BlockResponse, *types.Error) {
	var blockIndex int64
	var hash string

	if request.BlockIdentifier.Index != nil {
		blockIndex = *request.BlockIdentifier.Index
	}

	if request.BlockIdentifier.Hash != nil {
		hash = *request.BlockIdentifier.Hash
	}

	isGenesisBlockRequest, err := b.isGenesisBlockRequest(ctx, blockIndex, hash)
	if err != nil {
		return nil, service.WrapError(service.ErrClientError, err)
	}

	if isGenesisBlockRequest {
		genesisBlock, transactions, err := b.getGenesisBlockAndTransactions(ctx, request.NetworkIdentifier)
		if err != nil {
			return nil, service.WrapError(service.ErrClientError, err)
		}

		return &types.BlockResponse{
			Block: &types.Block{
				BlockIdentifier: b.genesisBlockIdentifier,
				// Parent block identifier of genesis block is set to itself instead of the hash of the genesis state
				// This is done as the genesis state hash cannot be used as a transaction id for the /block apis
				// and the operations found in the genesis state are returned as operations of the genesis block.
				ParentBlockIdentifier: b.genesisBlockIdentifier,
				Transactions:          transactions,
				Timestamp:             genesisBlock.Timestamp,
				Metadata: map[string]interface{}{
					pmapper.MetadataMessage: genesisBlock.Message,
				},
			},
		}, nil
	}

	block, err := b.getBlockDetails(ctx, blockIndex, hash)
	if err != nil {
		return nil, service.WrapError(service.ErrClientError, err)
	}
	blockIndex = int64(block.Height)

	transactions, err := b.parseTransactions(ctx, request.NetworkIdentifier, block.Txs)
	if err != nil {
		return nil, service.WrapError(service.ErrInternalError, err)
	}

	resp := &types.BlockResponse{
		Block: &types.Block{
			BlockIdentifier: &types.BlockIdentifier{
				Index: blockIndex,
				Hash:  block.BlockID.String(),
			},
			ParentBlockIdentifier: &types.BlockIdentifier{
				Index: blockIndex - 1,
				Hash:  block.ParentID.String(),
			},
			Timestamp:    block.Timestamp,
			Transactions: transactions,
		},
	}

	return resp, nil
}

// BlockTransaction implements the /block/transaction endpoint.
func (b *Backend) BlockTransaction(ctx context.Context, request *types.BlockTransactionRequest) (*types.BlockTransactionResponse, *types.Error) {
	isGenesisBlockRequest, err := b.isGenesisBlockRequest(ctx, request.BlockIdentifier.Index, request.BlockIdentifier.Hash)
	if err != nil {
		return nil, service.WrapError(service.ErrClientError, err)
	}

	var transactions []*types.Transaction

	if isGenesisBlockRequest {
		_, transactions, err = b.getGenesisBlockAndTransactions(ctx, request.NetworkIdentifier)
		if err != nil {
			return nil, service.WrapError(service.ErrInternalError, err)
		}
	} else {
		block, err := b.getBlockDetails(ctx, request.BlockIdentifier.Index, request.BlockIdentifier.Hash)
		if err != nil {
			return nil, service.WrapError(service.ErrClientError, err)
		}

		transactions, err = b.parseTransactions(ctx, request.NetworkIdentifier, block.Txs)
		if err != nil {
			return nil, service.WrapError(service.ErrInternalError, err)
		}
	}

	for i := range transactions {
		transaction := transactions[i]
		if transaction.TransactionIdentifier.Hash == request.TransactionIdentifier.Hash {
			return &types.BlockTransactionResponse{
				Transaction: transaction,
			}, nil
		}
	}

	return nil, service.ErrTransactionNotFound
}

func (b *Backend) parseTransactions(
	ctx context.Context,
	networkIdentifier *types.NetworkIdentifier,
	txs []*platformvm.Tx,
) ([]*types.Transaction, error) {
	dependencyTxs, err := b.fetchDependencyTxs(ctx, txs)
	if err != nil {
		return nil, err
	}

	parser, err := b.newTxParser(ctx, networkIdentifier, dependencyTxs)
	if err != nil {
		return nil, err
	}

	transactions := []*types.Transaction{}
	for _, tx := range txs {
		err = common.InitializeTx(b.codecVersion, b.codec, *tx)
		if err != nil {
			return nil, errTxInitialize
		}

		t, err := parser.Parse(tx.UnsignedTx)
		if err != nil {
			return nil, err
		}

		transactions = append(transactions, t)
	}
	return transactions, nil
}

func (b *Backend) fetchDependencyTxs(ctx context.Context, txs []*platformvm.Tx) (map[string]*pmapper.DependencyTx, error) {
	dependencyTxIDs := []ids.ID{}

	for _, tx := range txs {
		inputTxsIds, err := pmapper.GetDependencyTxIDs(tx.UnsignedTx)
		if err != nil {
			return nil, err
		}
		dependencyTxIDs = append(dependencyTxIDs, inputTxsIds...)
	}

	dependencyTxChan := make(chan *pmapper.DependencyTx, len(dependencyTxIDs))
	eg, ctx := errgroup.WithContext(ctx)

	dependencyTxs := make(map[string]*pmapper.DependencyTx)
	for i := range dependencyTxIDs {
		txID := dependencyTxIDs[i]
		eg.Go(func() error {
			return b.fetchDependencyTx(ctx, txID, dependencyTxChan)
		})
	}
	if err := eg.Wait(); err != nil {
		return nil, err
	}
	close(dependencyTxChan)

	for dTx := range dependencyTxChan {
		dependencyTxs[dTx.ID.String()] = dTx
	}

	return dependencyTxs, nil
}

func (b *Backend) fetchDependencyTx(ctx context.Context, txID ids.ID, out chan *pmapper.DependencyTx) error {
	// Genesis state contains initial allocation UTXOs. These are not technically part of a transaction.
	// As a result, their UTXO id uses zero value transaction id. In that case, return genesis allocation data
	if txID == ids.Empty {
		allocationTx, err := b.buildGenesisAllocationTx()
		if allocationTx != nil {
			out <- &pmapper.DependencyTx{
				ID: ids.Empty,
				Tx: allocationTx,
			}
		}
		return err
	}

	txBytes, err := b.pClient.GetTx(ctx, txID)
	if err != nil {
		return err
	}

	var tx platformvm.Tx
	_, err = b.codec.Unmarshal(txBytes, &tx)
	if err != nil {
		return err
	}

	err = common.InitializeTx(0, platformvm.Codec, tx)
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
	out <- &pmapper.DependencyTx{
		ID:          txID,
		Tx:          &tx,
		RewardUTXOs: utxos,
	}

	return nil
}

func (b *Backend) newTxParser(
	ctx context.Context,
	networkIdentifier *types.NetworkIdentifier,
	dependencyTxs map[string]*pmapper.DependencyTx,
) (*pmapper.TxParser, error) {
	hrp, err := mapper.GetHRP(networkIdentifier)
	if err != nil {
		return nil, err
	}

	chainIDs, err := b.getChainIDs(ctx)
	if err != nil {
		return nil, err
	}

	inputAddresses, err := pmapper.GetAccountsFromUTXOs(hrp, dependencyTxs)
	if err != nil {
		return nil, err
	}

	return pmapper.NewTxParser(false, hrp, chainIDs, inputAddresses, dependencyTxs), nil
}

func (b *Backend) getChainIDs(ctx context.Context) (map[string]string, error) {
	if b.chainIDs == nil {
		b.chainIDs = map[string]string{
			ids.Empty.String(): mapper.PChainNetworkIdentifier,
		}

		cChainID, err := b.pClient.GetBlockchainID(ctx, mapper.CChainNetworkIdentifier)
		if err != nil {
			return nil, err
		}
		b.chainIDs[cChainID.String()] = mapper.CChainNetworkIdentifier

		xChainID, err := b.pClient.GetBlockchainID(ctx, mapper.XChainNetworkIdentifier)
		if err != nil {
			return nil, err
		}
		b.chainIDs[xChainID.String()] = mapper.XChainNetworkIdentifier
	}

	return b.chainIDs, nil
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

func (b *Backend) isGenesisBlockRequest(ctx context.Context, index int64, hash string) (bool, error) {
	genesisBlock, err := b.getGenesisBlock(ctx)
	if err != nil {
		return false, err
	}

	if hash != "" {
		return hash == genesisBlock.BlockID.String(), nil
	}

	return index == int64(genesisBlock.Height), nil
}

func (b *Backend) getGenesisBlockAndTransactions(
	ctx context.Context,
	networkIdentifier *types.NetworkIdentifier,
) (*indexer.ParsedGenesisBlock, []*types.Transaction, error) {
	genesisBlock, err := b.getGenesisBlock(ctx)
	if err != nil {
		return nil, nil, err
	}

	txs := []*platformvm.Tx{}
	txs = append(txs, genesisBlock.Txs...)

	allocationTx, err := b.buildGenesisAllocationTx()
	if err != nil {
		return nil, nil, err
	}
	txs = append(txs, allocationTx)

	parser, err := b.newTxParser(ctx, networkIdentifier, nil)
	if err != nil {
		return nil, nil, err
	}

	transactions := []*types.Transaction{}
	for _, tx := range txs {
		t, err := parser.Parse(tx.UnsignedTx)
		if err != nil {
			return nil, nil, err
		}

		transactions = append(transactions, t)
	}

	return genesisBlock, transactions, nil
}

// Genesis allocation UTXOs are not part of a real transaction.
// For convenience and compatibility with the rest of the parsing functionality
// they are treated as outputs of an import tx with no inputs and id ids.Empty
func (b *Backend) buildGenesisAllocationTx() (*platformvm.Tx, error) {
	outs := []*avax.TransferableOutput{}
	for _, utxo := range b.genesisBlock.UTXOs {
		outIntf := utxo.Out
		if lockedOut, ok := outIntf.(*stakeable.LockOut); ok {
			outIntf = lockedOut.TransferableOut
		}

		out, ok := outIntf.(*secp256k1fx.TransferOutput)

		if !ok {
			return nil, errUnableToParseUTXO
		}

		outs = append(outs, &avax.TransferableOutput{
			Out: &secp256k1fx.TransferOutput{
				Amt: out.Amount(),
				OutputOwners: secp256k1fx.OutputOwners{
					Addrs:     out.Addrs,
					Threshold: out.Threshold,
					Locktime:  out.Locktime,
				},
			},
		})
	}

	allocationTx := &platformvm.UnsignedImportTx{}
	allocationTx.Outs = outs
	tx := &platformvm.Tx{
		UnsignedTx: allocationTx,
	}
	return tx, nil
}