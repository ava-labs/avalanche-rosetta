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
	"github.com/ava-labs/avalanchego/utils/wrappers"
	"github.com/ava-labs/avalanchego/vms/platformvm"
	"github.com/ava-labs/avalanchego/vms/proposervm/block"

	"github.com/ava-labs/avalanche-rosetta/client"
	"github.com/ava-labs/avalanche-rosetta/mapper"
	"github.com/ava-labs/avalanche-rosetta/service/backend/common"
)

var errParserUninitialized = errors.New("uninitialized parser")

const genesisTimestamp = 1599696000

type Parser interface {
	Initialize(ctx context.Context) (*ParsedGenesisBlock, error)
	GetPlatformHeight(ctx context.Context) (uint64, error)
	ParseCurrentBlock(ctx context.Context) (*ParsedBlock, error)
	ParseBlockAtIndex(ctx context.Context, index uint64) (*ParsedBlock, error)
	ParseBlockWithHash(ctx context.Context, hash string) (*ParsedBlock, error)
}

// Interface compliance
var _ Parser = &parser{}

type parser struct {
	networkID   uint32
	avaxAssetID ids.ID
	aliaser     ids.Aliaser

	codec codec.Manager

	ctx *snow.Context

	pChainClient client.PChainClient

	genesisTimestamp time.Time
}

func NewParser(pChainClient client.PChainClient) (Parser, error) {
	errs := wrappers.Errs{}

	aliaser := ids.NewAliaser()
	errs.Add(aliaser.Alias(constants.PlatformChainID, mapper.PChainNetworkIdentifier))

	return &parser{
		codec:            platformvm.Codec,
		pChainClient:     pChainClient,
		aliaser:          aliaser,
		genesisTimestamp: time.Unix(genesisTimestamp, 0),
	}, errs.Err
}

func (p *parser) initCtx(ctx context.Context) error {
	if p.ctx == nil {
		networkID, err := p.pChainClient.GetNetworkID(ctx)
		if err != nil {
			return err
		}

		p.networkID = networkID
		p.ctx = &snow.Context{
			BCLookup:  p.aliaser,
			NetworkID: networkID,
		}
	}

	return nil
}

func (p *parser) GetPlatformHeight(ctx context.Context) (uint64, error) {
	err := p.initCtx(ctx)
	if err != nil {
		return 0, err
	}

	return p.pChainClient.GetHeight(ctx)
}

func (p *parser) Initialize(ctx context.Context) (*ParsedGenesisBlock, error) {
	err := p.initCtx(ctx)
	if err != nil {
		return nil, err
	}

	errs := wrappers.Errs{}

	bytes, avaxAssetID, err := genesis.FromConfig(genesis.GetConfig(p.networkID))
	errs.Add(err)
	p.avaxAssetID = avaxAssetID

	genesis := &platformvm.Genesis{}
	_, err = platformvm.GenesisCodec.Unmarshal(bytes, genesis)
	errs.Add(err)
	errs.Add(genesis.Initialize())

	blockTime := new(time.Time)
	p.genesisTimestamp = time.Unix(int64(genesis.Timestamp), 0)
	*blockTime = p.genesisTimestamp

	var genesisTxs []*platformvm.Tx
	genesisTxs = append(genesisTxs, genesis.Validators...)
	genesisTxs = append(genesisTxs, genesis.Chains...)

	// Genesis commit block's parent ID is the hash of genesis state
	var genesisParentID ids.ID = hashing.ComputeHash256Array(bytes)

	// Genesis Block is not indexed by the indexer, but its block ID can be accessed from block 0's parent id
	genesisChildBlock, err := p.ParseBlockAtIndex(ctx, 1)
	if err != nil {
		return nil, err
	}

	for _, utxo := range genesis.UTXOs {
		utxo.UTXO.Out.InitCtx(p.ctx)
	}

	genesisBlockID := genesisChildBlock.ParentID

	return &ParsedGenesisBlock{
		ParsedBlock: ParsedBlock{
			ParentID:  genesisParentID,
			Height:    0,
			BlockID:   genesisBlockID,
			BlockType: "GenesisBlock",
			Timestamp: blockTime.UnixMilli(),
			Txs:       genesisTxs,
			Proposer:  Proposer{},
		},
		GenesisBlockData: GenesisBlockData{
			Message:       genesis.Message,
			InitialSupply: genesis.InitialSupply,
			UTXOs:         genesis.UTXOs,
		},
	}, errs.Err
}

func (p *parser) ParseCurrentBlock(ctx context.Context) (*ParsedBlock, error) {
	err := p.initCtx(ctx)
	if err != nil {
		return nil, err
	}

	height, err := p.GetPlatformHeight(ctx)
	if err != nil {
		return nil, err
	}

	return p.ParseBlockAtIndex(ctx, height)
}

func (p *parser) ParseBlockAtIndex(ctx context.Context, index uint64) (*ParsedBlock, error) {
	err := p.initCtx(ctx)
	if err != nil {
		return nil, err
	}

	container, err := p.pChainClient.GetContainerByIndex(ctx, index-1)
	if err != nil {
		return nil, err
	}

	return p.parseBlockBytes(container.Bytes)
}

func (p *parser) ParseBlockWithHash(ctx context.Context, hash string) (*ParsedBlock, error) {
	err := p.initCtx(ctx)
	if err != nil {
		return nil, err
	}

	hashID, err := ids.FromString(hash)
	if err != nil {
		return nil, err
	}

	container, err := p.pChainClient.GetContainerByID(ctx, hashID)
	if err != nil {
		return nil, err
	}

	return p.parseBlockBytes(container.Bytes)
}

func (p *parser) parseBlockBytes(proposerBytes []byte) (*ParsedBlock, error) {
	errs := wrappers.Errs{}

	proposer, bytes, err := getProposerFromBytes(proposerBytes)
	if err != nil {
		return nil, fmt.Errorf("fetching proposer from block bytes errored with %w", err)
	}

	if p.genesisTimestamp.IsZero() {
		return nil, errParserUninitialized
	}

	// Default block time to proposer timestamp if exists, otherwise to genesis block
	blockTimestamp := new(time.Time)

	if proposer.Timestamp > genesisTimestamp {
		*blockTimestamp = time.Unix(proposer.Timestamp, 0)
	} else {
		*blockTimestamp = time.Unix(genesisTimestamp, 0)
	}

	var blk platformvm.Block
	ver, err := p.codec.Unmarshal(bytes, &blk)
	if err != nil {
		return nil, fmt.Errorf("unmarshaling block bytes errored with %w", err)
	}
	blkID := ids.ID(hashing.ComputeHash256Array(bytes))

	parsedBlock := ParsedBlock{
		Height:    blk.Height(),
		BlockID:   blkID,
		BlockType: fmt.Sprintf("%T", blk),
		Proposer:  proposer,
	}

	switch castBlk := blk.(type) {
	case *platformvm.ProposalBlock:
		errs.Add(common.InitializeTx(ver, p.codec, castBlk.Tx))

		parsedBlock.ParentID = castBlk.PrntID
		parsedBlock.Txs = []*platformvm.Tx{&castBlk.Tx}
	case *platformvm.AtomicBlock:
		errs.Add(common.InitializeTx(ver, p.codec, castBlk.Tx))

		parsedBlock.ParentID = castBlk.PrntID
		parsedBlock.Txs = []*platformvm.Tx{&castBlk.Tx}
	case *platformvm.StandardBlock:
		var txs []*platformvm.Tx

		for i, tx := range castBlk.Txs {
			errs.Add(common.InitializeTx(ver, p.codec, *tx))
			txs = append(txs, castBlk.Txs[i])
		}

		parsedBlock.ParentID = castBlk.PrntID
		parsedBlock.Txs = txs
	case *platformvm.AbortBlock:
		parsedBlock.ParentID = castBlk.PrntID
		parsedBlock.Txs = []*platformvm.Tx{}
	case *platformvm.CommitBlock:
		parsedBlock.ParentID = castBlk.PrntID
		parsedBlock.Txs = []*platformvm.Tx{}
	default:
		errs.Add(fmt.Errorf("no handler exists for block type %T", castBlk))
	}

	// If the block has an advance time tx, use its timestamp as the block timestamp
	for _, tx := range parsedBlock.Txs {
		if att, ok := tx.UnsignedTx.(*platformvm.UnsignedAdvanceTimeTx); ok {
			*blockTimestamp = att.Timestamp()
			break
		}
	}

	parsedBlock.Timestamp = blockTimestamp.UnixMilli()

	return &parsedBlock, errs.Err
}

func getProposerFromBytes(bytes []byte) (Proposer, []byte, error) {
	proposer, err := block.Parse(bytes)
	if err != nil || proposer == nil {
		return Proposer{}, bytes, nil
	}

	switch castBlock := proposer.(type) {
	case block.SignedBlock:
		return Proposer{
			ID:           castBlock.ID(),
			NodeID:       castBlock.Proposer(),
			PChainHeight: castBlock.PChainHeight(),
			Timestamp:    castBlock.Timestamp().Unix(),
			ParentID:     castBlock.ParentID(),
		}, castBlock.Block(), nil
	case block.Block:
		return Proposer{}, castBlock.Block(), nil
	default:
		return Proposer{}, bytes, fmt.Errorf("no handler exists for proposer block type %T", castBlock)
	}
}
