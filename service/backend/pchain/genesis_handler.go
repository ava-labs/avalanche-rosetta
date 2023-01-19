package pchain

import (
	"context"
	"errors"

	"github.com/ava-labs/avalanchego/codec"
	"github.com/ava-labs/avalanchego/vms/components/avax"
	"github.com/ava-labs/avalanchego/vms/platformvm/blocks"
	"github.com/ava-labs/avalanchego/vms/platformvm/stakeable"
	"github.com/ava-labs/avalanchego/vms/platformvm/txs"
	"github.com/ava-labs/avalanchego/vms/secp256k1fx"
	"github.com/coinbase/rosetta-sdk-go/types"

	"github.com/ava-labs/avalanche-rosetta/service"
	"github.com/ava-labs/avalanche-rosetta/service/backend/pchain/indexer"
)

var _ genesisHandler = &gHandler{}

type genesisHandler interface {
	isGenesisBlockRequest(index int64, hash string) (bool, error)
	getGenesisBlock() *indexer.ParsedGenesisBlock
	getGenesisIdentifier() *types.BlockIdentifier

	// [getFullGenesisTxs] returns proper genesis txs + genesis allocation tx
	getFullGenesisTxs() ([]*txs.Tx, error)
	buildGenesisAllocationTx() (*txs.Tx, error)
}

func newGenesisHandler(nodeMode string, indexerParser indexer.Parser) genesisHandler {
	gh := &gHandler{
		indexerParser: indexerParser,

		// Note: since genesis block and transactions can be considerably larger
		// than any other block generated during the blockchain lifetime
		// a special codec is used to parse genesis-related objects
		genesisCodec: blocks.GenesisCodec,
	}

	// Initializing genesis block from indexer only in online mode
	if nodeMode == service.ModeOnline {
		_ = gh.lazyLoadGenesisBlk()
	}

	return gh
}

type gHandler struct {
	indexerParser indexer.Parser
	genesisCodec  codec.Manager

	// genesisBlk is lazily initialized, as soon as
	// pChainClient is ready to serve requests
	genesisBlkFetched bool
	genesisBlk        *indexer.ParsedGenesisBlock
	genesisIdentifier *types.BlockIdentifier

	allocationTx *txs.Tx
}

func (gh *gHandler) lazyLoadGenesisBlk() bool {
	if gh.genesisBlkFetched {
		return true // genesis block loaded
	}

	genesisBlk, err := gh.indexerParser.GetGenesisBlock(context.Background())
	if err == nil {
		gh.genesisBlkFetched = true
		gh.genesisBlk = genesisBlk
		gh.genesisIdentifier = &types.BlockIdentifier{
			Index: int64(genesisBlk.Height),
			Hash:  genesisBlk.BlockID.String(),
		}
		return true // genesis block loaded
	}

	return false // genesis block not loaded
}

func (gh *gHandler) isGenesisBlockRequest(index int64, hash string) (bool, error) {
	if loaded := gh.lazyLoadGenesisBlk(); !loaded {
		return false, errors.New("could not load genesis data")
	}

	// if hash is provided, make sure it matches genesis block hash
	if hash != "" {
		return hash == gh.genesisBlk.BlockID.String(), nil
	}

	// if hash is omitted, check if the height matches the genesis block height
	return index == int64(gh.genesisBlk.Height), nil
}

// getGenesisBlock is a simple getter for genesisBlk. It does not check
// whether genesisBlk has been duly initialized. Check is up to caller
func (gh *gHandler) getGenesisBlock() *indexer.ParsedGenesisBlock {
	return gh.genesisBlk
}

// getGenesisIdentifier is a simple getter for genesisIdentifier. It does not check
// whether genesisIdentifier has been duly initialized. Check is up to caller
func (gh *gHandler) getGenesisIdentifier() *types.BlockIdentifier {
	return gh.genesisIdentifier
}

// getFullGenesisTxs does not check whether genesis
// has been duly initialized. Check is up to caller
func (gh *gHandler) getFullGenesisTxs() ([]*txs.Tx, error) {
	res := gh.genesisBlk.Txs
	allocationTx, err := gh.buildGenesisAllocationTx()
	if err != nil {
		return nil, err
	}
	res = append(res, allocationTx)
	return res, nil
}

// Genesis allocation UTXOs are not part of a real transaction.
// For convenience and compatibility with the rest of the parsing functionality
// they are treated as outputs of an import tx with no inputs and id ids.Empty
func (gh *gHandler) buildGenesisAllocationTx() (*txs.Tx, error) {
	if gh.allocationTx != nil {
		return gh.allocationTx, nil
	}

	outs := []*avax.TransferableOutput{}
	for _, utxo := range gh.genesisBlk.UTXOs {
		outIntf := utxo.Out
		if lockedOut, ok := outIntf.(*stakeable.LockOut); ok {
			outIntf = lockedOut.TransferableOut
		}

		out, ok := outIntf.(*secp256k1fx.TransferOutput)

		if !ok {
			return nil, errUnableToParseUTXO
		}

		outs = append(outs, &avax.TransferableOutput{
			Asset: avax.Asset{
				ID: utxo.AssetID(),
			},
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

	// TODO: this is probably not the right way to build this tx
	// Some fields are missing that we populate in tx Builder
	allocationTx := &txs.ImportTx{}
	allocationTx.Outs = outs
	tx := &txs.Tx{
		Unsigned: allocationTx,
	}
	if err := tx.Sign(gh.genesisCodec, nil); err != nil {
		return nil, err
	}
	gh.allocationTx = tx
	return gh.allocationTx, nil
}
