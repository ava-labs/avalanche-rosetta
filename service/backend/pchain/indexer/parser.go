package indexer

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ava-labs/avalanchego/codec"
	"github.com/ava-labs/avalanchego/genesis"
	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/avalanchego/snow"
	"github.com/ava-labs/avalanchego/utils/constants"
	"github.com/ava-labs/avalanchego/utils/hashing"

	pBlocks "github.com/ava-labs/avalanchego/vms/platformvm/blocks"
	pGenesis "github.com/ava-labs/avalanchego/vms/platformvm/genesis"
	"github.com/ava-labs/avalanchego/vms/platformvm/txs"
	proposerBlk "github.com/ava-labs/avalanchego/vms/proposervm/block"

	"github.com/ava-labs/avalanche-rosetta/client"
	rosConst "github.com/ava-labs/avalanche-rosetta/constants"
)

var (
	_ Parser = &parser{}

	genesisTimestamp         = time.Date(2020, time.September, 10, 0, 0, 0, 0, time.UTC)
	noProposerTime           = time.Time{}
	errMissingBlockIndexHash = errors.New("a positive block index, a block hash or both must be specified")
)

// Parser defines the interface for a P-chain indexer parser
// Note: we use indexer just because platformVM does not currently offer a way to retrieve
// blocks by height. However we do NOT want to use the indexer to retrieve blocks by ID; instead
// we'll use platformvm.GetBlock api for that. The reason is that we want to use
// platformVM-level block ID as P-chain blocks identifier rather than proposerVM-level.
type Parser interface {
	// GetGenesisBlock parses and returns the Genesis block
	GetGenesisBlock(ctx context.Context) (*ParsedGenesisBlock, error)
	// ParseNonGenesisBlock returns the block with provided hash or height
	ParseNonGenesisBlock(ctx context.Context, hash string, height uint64) (*ParsedBlock, error)
	// GetPlatformHeight returns the current block height of P-chain
	GetPlatformHeight(ctx context.Context) (uint64, error)
	// ParseCurrentBlock parses and returns the current tip of P-chain
	ParseCurrentBlock(ctx context.Context) (*ParsedBlock, error)
}

type parser struct {
	// The full PChainClient is currently needed just to retrieve
	// NetworkID.
	// TODO: consider introducing a cache for parsed blocks
	pChainClient client.PChainClient

	codec        codec.Manager
	codecVersion uint16

	networkID uint32

	aliaser ids.Aliaser
}

// NewParser creates a new P-chain indexer parser
// Note: NewParser should not contain calls to pChainClient as we
// cannot assume client is ready to serve requests immediately
func NewParser(pChainClient client.PChainClient, avalancheNetworkID uint32) (Parser, error) {
	aliaser := ids.NewAliaser()
	err := aliaser.Alias(constants.PlatformChainID, rosConst.PChain.String())
	if err != nil {
		return nil, err
	}

	return &parser{
		pChainClient: pChainClient,
		codec:        pBlocks.Codec,
		codecVersion: pBlocks.Version,
		networkID:    avalancheNetworkID,
		aliaser:      aliaser,
	}, nil
}

func (p *parser) GetPlatformHeight(ctx context.Context) (uint64, error) {
	container, _, err := p.pChainClient.GetLastAccepted(ctx)
	if err != nil {
		return 0, err
	}
	blk, err := p.parseProposerBlock(container.Bytes)
	if err != nil {
		return 0, err
	}
	return blk.Height, nil
}

// GetGenesisBlock is called to initialize P-chain genesis information upon startup.
// GetGenesisBlock should not call the indexer, to ensure backward compatibility with
// previous installations which do no host a block indexer.
func (p *parser) GetGenesisBlock(_ context.Context) (*ParsedGenesisBlock, error) {
	bytes, _, err := genesis.FromConfig(genesis.GetConfig(p.networkID))
	if err != nil {
		return nil, err
	}

	genesisState, err := pGenesis.Parse(bytes)
	if err != nil {
		return nil, err
	}

	genesisTimestamp := time.Unix(int64(genesisState.Timestamp), 0)

	var genesisTxs []*txs.Tx
	genesisTxs = append(genesisTxs, genesisState.Validators...)
	genesisTxs = append(genesisTxs, genesisState.Chains...)

	// Build genesis ID and ParentID as it's done in platformVM'State,
	// without polling indexer (for backward compatibility).
	genesisParentID := hashing.ComputeHash256Array(bytes)
	genesisBlock, err := pBlocks.NewApricotCommitBlock(genesisParentID, 0)
	if err != nil {
		return nil, err
	}
	genesisBlockID := genesisBlock.ID()

	// genesis gets its own context to unlock caching
	genesisCtx := &snow.Context{
		BCLookup:  p.aliaser,
		NetworkID: p.networkID,
	}
	for _, utxo := range genesisState.UTXOs {
		utxo.UTXO.Out.InitCtx(genesisCtx)
	}

	return &ParsedGenesisBlock{
		ParsedBlock: ParsedBlock{
			ParentID:  genesisParentID,
			Height:    0,
			BlockID:   genesisBlockID,
			BlockType: "GenesisBlock",
			Timestamp: genesisTimestamp.UnixMilli(),
			Txs:       genesisTxs,
		},
		GenesisBlockData: GenesisBlockData{
			Message:       genesisState.Message,
			InitialSupply: genesisState.InitialSupply,
			UTXOs:         genesisState.UTXOs,
		},
	}, nil
}

func (p *parser) ParseCurrentBlock(ctx context.Context) (*ParsedBlock, error) {
	height, err := p.GetPlatformHeight(ctx)
	if err != nil {
		return nil, err
	}

	return p.parseBlockAtHeight(ctx, height)
}

func (p *parser) ParseNonGenesisBlock(ctx context.Context, hash string, height uint64) (*ParsedBlock, error) {
	if height <= 0 && hash == "" {
		return nil, errMissingBlockIndexHash
	}

	if hash != "" {
		return p.parseBlockWithHash(ctx, hash)
	}

	return p.parseBlockAtHeight(ctx, height)
}

func (p *parser) parseBlockAtHeight(ctx context.Context, height uint64) (*ParsedBlock, error) {
	// P-chain indexer does not include genesis and store block at height 1 with index 0.
	// Therefore containers are looked up with index = height - 1.
	// Note that genesis does not cause a problem here as it is handled in a separate code path
	container, err := p.pChainClient.GetContainerByIndex(ctx, height-1)
	if err != nil {
		return nil, err
	}

	return p.parseProposerBlock(container.Bytes)
}

func (p *parser) parseBlockWithHash(ctx context.Context, hash string) (*ParsedBlock, error) {
	hashID, err := ids.FromString(hash)
	if err != nil {
		return nil, err
	}

	// hashID is P-Chain block ID (not proposerVM block ID). Hence we try pulling
	// block from P-Chain API (not the indexer, which tracks proposerVM blocks)
	blkBytes, err := p.pChainClient.GetBlock(ctx, hashID)
	if err != nil {
		return nil, err
	}

	return p.parsePChainBlock(blkBytes, noProposerTime)
}

// [parseProposerBlock] parses blocks are retrieved from index api.
// [parseProposerBlock] tries to parse block as ProposerVM block first.
// In case of failure, it tries to parse it as a pre-proposerVM block.
func (p *parser) parseProposerBlock(blkBytes []byte) (*ParsedBlock, error) {
	pChainBlkBytes := blkBytes
	proposerTime := noProposerTime

	proBlk, err := proposerBlk.Parse(blkBytes)
	if err == nil {
		// inner proposerVM bytes, to be parsed as P-chain block
		pChainBlkBytes = proBlk.Block()

		// retrieve relevant proposer data
		if b, ok := proBlk.(proposerBlk.SignedBlock); ok {
			proposerTime = b.Timestamp()
		}
	}

	return p.parsePChainBlock(pChainBlkBytes, proposerTime)
}

func (p *parser) parsePChainBlock(pChainBlkBytes []byte, proposerTime time.Time) (*ParsedBlock, error) {
	blk, err := pBlocks.Parse(p.codec, pChainBlkBytes)
	if err != nil {
		return nil, fmt.Errorf("unmarshaling block bytes errored with %w", err)
	}
	txes := blk.Txs()
	if txes == nil {
		txes = []*txs.Tx{}
	}

	// We retrieve timestamps from the block to have a deployment-independent timestamp.
	// This blkTime is not guarateed to be monotonic before Banff blocks, whose Mainnet
	// activation happened on Tuesday, 2022 October 18 at 12 p.m. EDT.
	blkTime, err := retrieveTime(blk, proposerTime)
	if err != nil {
		return nil, fmt.Errorf("failed retrieving block time: %w", err)
	}

	return &ParsedBlock{
		BlockID:   blk.ID(),
		BlockType: fmt.Sprintf("%T", blk),
		ParentID:  blk.Parent(),
		Timestamp: blkTime.UnixMilli(),

		Height: blk.Height(),
		Txs:    txes,
	}, nil
}

func retrieveTime(pchainBlk pBlocks.Block, proposerTime time.Time) (time.Time, error) {
	switch b := pchainBlk.(type) {
	// Banff blocks serialize pchain time
	case *pBlocks.BanffProposalBlock:
		return b.Timestamp(), nil
	case *pBlocks.BanffStandardBlock:
		return b.Timestamp(), nil
	case *pBlocks.BanffAbortBlock:
		return b.Timestamp(), nil
	case *pBlocks.BanffCommitBlock:
		return b.Timestamp(), nil

	// Apricot Proposal blocks may contain an advance time tx
	// setting pchain time
	case *pBlocks.ApricotProposalBlock:
		if t, ok := b.Tx.Unsigned.(*txs.AdvanceTimeTx); ok {
			return t.Timestamp(), nil
		}

	case *pBlocks.ApricotAtomicBlock,
		*pBlocks.ApricotStandardBlock,
		*pBlocks.ApricotAbortBlock,
		*pBlocks.ApricotCommitBlock:
		// no relevant time information in these blocks

	default:
		return time.Time{}, fmt.Errorf("unknown block type %T", b)
	}

	// No timestamp was found in given pre-Banff block.
	// Fallback to proposer timestamp as used by Snowman++
	// if available. While proposer timestamp should be close
	// to pchain time at block creation, time monotonicity is
	// not guaranteed.
	if !proposerTime.IsZero() {
		return proposerTime, nil
	}

	// Fallback to the genesis timestamp. We cannot simply
	// return time.Time{} as Rosetta expects timestamps to be available
	// after 1/1/2000.
	return genesisTimestamp, nil
}
