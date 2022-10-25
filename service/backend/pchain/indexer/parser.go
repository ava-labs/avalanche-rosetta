package indexer

import (
	"context"
	"fmt"
	"time"

	"github.com/ava-labs/avalanchego/codec"
	"github.com/ava-labs/avalanchego/genesis"
	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/avalanchego/indexer"
	"github.com/ava-labs/avalanchego/snow"
	"github.com/ava-labs/avalanchego/utils/constants"
	"github.com/ava-labs/avalanchego/utils/hashing"

	pBlocks "github.com/ava-labs/avalanchego/vms/platformvm/blocks"
	pGenesis "github.com/ava-labs/avalanchego/vms/platformvm/genesis"
	"github.com/ava-labs/avalanchego/vms/platformvm/txs"
	proposerBlk "github.com/ava-labs/avalanchego/vms/proposervm/block"

	"github.com/ava-labs/avalanche-rosetta/client"
	"github.com/ava-labs/avalanche-rosetta/mapper"
)

var (
	_ Parser = &parser{}

	genesisTimestamp = time.Date(2020, time.September, 10, 0, 0, 0, 0, time.UTC)
)

// Parser defines the interface for a P-chain indexer parser
type Parser interface {
	// GetGenesisBlock parses and returns the Genesis block
	GetGenesisBlock(ctx context.Context) (*ParsedGenesisBlock, error)
	// GetPlatformHeight returns the current block height of P-chain
	GetPlatformHeight(ctx context.Context) (uint64, error)
	// ParseCurrentBlock parses and returns the current tip of P-chain
	ParseCurrentBlock(ctx context.Context) (*ParsedBlock, error)
	// ParseBlockAtHeight parses and returns the block at the specified index
	ParseBlockAtHeight(ctx context.Context, height uint64) (*ParsedBlock, error)
	// ParseBlockWithHash parses and returns the block with the specified hash
	ParseBlockWithHash(ctx context.Context, hash string) (*ParsedBlock, error)
}

type parser struct {
	// ideally parser should rely only on pchain indexer apis.
	// The full PChainClient is currently needed just to retrieve
	// NetworkID.
	// TODO: reduce pChainClient to indexer methods only
	// TODO: consider introducing a cache for parsed blocks
	pChainClient client.PChainClient

	codec        codec.Manager
	codecVersion uint16

	networkID uint32
	aliaser   ids.Aliaser
}

// NewParser creates a new P-chain indexer parser
func NewParser(pChainClient client.PChainClient) (Parser, error) {
	aliaser := ids.NewAliaser()
	err := aliaser.Alias(constants.PlatformChainID, mapper.PChainNetworkIdentifier)
	if err != nil {
		return nil, err
	}

	networkID, err := pChainClient.GetNetworkID(context.Background())
	if err != nil {
		return nil, err
	}

	return &parser{
		pChainClient: pChainClient,
		codec:        pBlocks.Codec,
		codecVersion: pBlocks.Version,
		networkID:    networkID,
		aliaser:      aliaser,
	}, nil
}

func (p *parser) GetPlatformHeight(ctx context.Context) (uint64, error) {
	container, err := p.pChainClient.GetLastAccepted(ctx)
	if err != nil {
		return 0, err
	}
	blk, err := p.parseContainer(container)
	if err != nil {
		return 0, err
	}
	return blk.Height, nil
}

func (p *parser) GetGenesisBlock(ctx context.Context) (*ParsedGenesisBlock, error) {
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

	// Genesis commit block's parent ID is the hash of genesis state
	var genesisParentID ids.ID = hashing.ComputeHash256Array(bytes)

	// Genesis Block is not indexed by the indexer, but its block ID can be accessed from block 0's parent id
	genesisChildBlock, err := p.ParseBlockAtHeight(ctx, 1)
	if err != nil {
		return nil, err
	}

	// genesis gets its own context to unlock caching
	genesisCtx := &snow.Context{
		BCLookup:  p.aliaser,
		NetworkID: p.networkID,
	}
	for _, utxo := range genesisState.UTXOs {
		utxo.UTXO.Out.InitCtx(genesisCtx)
	}

	genesisBlockID := genesisChildBlock.ParentID

	return &ParsedGenesisBlock{
		ParsedBlock: ParsedBlock{
			ParentID:  genesisParentID,
			Height:    0,
			BlockID:   genesisBlockID,
			BlockType: "GenesisBlock",
			Timestamp: genesisTimestamp.UnixMilli(),
			Txs:       genesisTxs,
			Proposer:  Proposer{},
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

	return p.ParseBlockAtHeight(ctx, height)
}

func (p *parser) ParseBlockAtHeight(ctx context.Context, height uint64) (*ParsedBlock, error) {
	// P-chain indexer does not include genesis and store block at height 1 with index 0.
	// Therefore containers are looked up with index = height - 1.
	// Note that genesis does not cause a problem here as it is handled in a separate code path
	container, err := p.pChainClient.GetContainerByIndex(ctx, height-1)
	if err != nil {
		return nil, err
	}

	return p.parseContainer(container)
}

func (p *parser) ParseBlockWithHash(ctx context.Context, hash string) (*ParsedBlock, error) {
	hashID, err := ids.FromString(hash)
	if err != nil {
		return nil, err
	}

	container, err := p.pChainClient.GetContainerByID(ctx, hashID)
	if err != nil {
		return nil, err
	}

	return p.parseContainer(container)
}

// [parseContainer] parses blocks are retrieved from index api.
// [parseContainer] tries to parse container asProposerVM block first.
// In case of failure, it tries to parse it as a pre-proposerVM block.
func (p *parser) parseContainer(container indexer.Container) (*ParsedBlock, error) {
	blkBytes := container.Bytes
	pChainBlkBytes := blkBytes
	proBlkData := Proposer{}

	proBlk, _, err := proposerBlk.Parse(blkBytes)
	if err == nil {
		// inner proposerVM bytes, to be parsed as P-chain block
		pChainBlkBytes = proBlk.Block()

		// retrieve relevant proposer data
		if b, ok := proBlk.(proposerBlk.SignedBlock); ok {
			proBlkData = Proposer{
				ID:           b.ID(),
				ParentID:     b.ParentID(),
				NodeID:       b.Proposer(),
				PChainHeight: b.PChainHeight(),
				Timestamp:    b.Timestamp().Unix(),
			}
		} else {
			proBlkData = Proposer{
				ID:       proBlk.ID(),
				ParentID: proBlk.ParentID(),
			}
		}
	}

	blk, err := pBlocks.Parse(p.codec, pChainBlkBytes)
	if err != nil {
		return nil, fmt.Errorf("unmarshaling block bytes errored with %w", err)
	}
	txes := blk.Txs()
	if txes == nil {
		txes = []*txs.Tx{}
	}

	// container.Timestamp is the time the indexer node discovered the requested block.
	// As such, its value depends on the specific indexer deployment. Instead we retrieve
	// timestamps from the block to have a deployment-independent timestamp. This blkTime
	// is not guarateed to be monotonic before Banff blocks, whose Mainnet activation happened
	// on Tuesday, 2022 October 18 at 12 p.m. EDT.
	blkTime, err := retrieveTime(blk, proBlk)
	if err != nil {
		return nil, fmt.Errorf("failed retrieving block time: %w", err)
	}

	return &ParsedBlock{
		BlockID:   blk.ID(),
		BlockType: fmt.Sprintf("%T", blk),
		ParentID:  blk.Parent(),
		Timestamp: blkTime.UnixMilli(),

		Height:   blk.Height(),
		Txs:      txes,
		Proposer: proBlkData,
	}, nil
}

func retrieveTime(pchainBlk pBlocks.Block, proBlk proposerBlk.Block) (time.Time, error) {
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
	if proBlk != nil {
		if signedProBlk, ok := proBlk.(proposerBlk.SignedBlock); ok {
			return signedProBlk.Timestamp(), nil
		}
	}

	// Fallback to the genesis timestamp. We cannot simply
	// return time.Time{} as Rosetta expects timestamps to be available
	// after 1/1/2000.
	return genesisTimestamp, nil
}
