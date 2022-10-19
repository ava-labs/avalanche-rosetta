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

// Interface compliance
var _ Parser = &parser{}

type parser struct {
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
	return p.pChainClient.GetHeight(ctx)
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
// In case of failure, it tries to parsing it as a pre-proposerVM block.
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

	return &ParsedBlock{
		BlockID:   blk.ID(),
		BlockType: fmt.Sprintf("%T", blk),
		ParentID:  blk.Parent(),
		Timestamp: container.Timestamp,
		Height:    blk.Height(),
		Txs:       blk.Txs(),
		Proposer:  proBlkData,
	}, nil
}
