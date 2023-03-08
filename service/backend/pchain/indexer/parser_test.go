package indexer

import (
	"context"
	stdjson "encoding/json"
	"fmt"
	"os"
	"testing"

	"github.com/ava-labs/avalanchego/snow"
	"github.com/ava-labs/avalanchego/utils/constants"
	pGenesis "github.com/ava-labs/avalanchego/vms/platformvm/genesis"
	"github.com/ava-labs/avalanchego/vms/platformvm/txs"

	rosConst "github.com/ava-labs/avalanche-rosetta/constants"
	mocks "github.com/ava-labs/avalanche-rosetta/mocks/client"

	"github.com/ava-labs/avalanchego/api"
	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/avalanchego/indexer"
	"github.com/ava-labs/avalanchego/utils/formatting"
	"github.com/stretchr/testify/assert"

	"github.com/stretchr/testify/mock"
)

var (
	p Parser
	g *ParsedGenesisBlock
)

// idxs of the containers we test against
var idxs = []uint64{
	0,
	1,
	2,
	8,
	48,
	173,
	382,
	911,
	1603,
	5981,
	131475,
	211277,
	211333,
	806002,
	810424,
	1000000,
	1000001,
	1000002,
	1000004,
}

func readFixture(path string, sprintfArgs ...interface{}) []byte {
	relpath := fmt.Sprintf(path, sprintfArgs...)
	ret, err := os.ReadFile(fmt.Sprintf("testdata/%s", relpath))
	if err != nil {
		panic(err)
	}

	return ret
}

func TestMain(m *testing.M) {
	ctx := context.Background()
	pchainClient := &mocks.PChainClient{}

	for _, idx := range idxs {
		ret := readFixture("ins/%v.json", idx)

		var container indexer.Container
		err := stdjson.Unmarshal(ret, &container)
		if err != nil {
			panic(err)
		}

		pchainClient.On("GetContainerByIndex", ctx, idx).Return(container, nil).Once()
	}

	txID, err := ids.FromString("jWgE5KiiCejNYbYGDzhu9WAXrAdgwav9EXuycNVdB62rSU4tH")
	if err != nil {
		panic(err)
	}
	arg := &api.GetTxArgs{
		TxID:     txID,
		Encoding: formatting.Hex,
	}
	bytes := [][]byte{{0, 0, 96, 135, 38, 30, 158, 122, 109, 66, 126, 42, 192, 155, 20, 141, 194, 137, 85, 161, 188, 115, 215, 227, 44, 148, 7, 201, 191, 227, 25, 222, 126, 28, 0, 0, 0, 7, 33, 230, 115, 23, 203, 196, 190, 42, 235, 0, 103, 122, 214, 70, 39, 120, 168, 245, 34, 116, 185, 214, 5, 223, 37, 145, 178, 48, 39, 168, 125, 255, 0, 0, 0, 7, 0, 0, 0, 4, 238, 10, 47, 173, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 0, 0, 0, 1, 237, 104, 212, 116, 123, 119, 22, 41, 162, 163, 85, 62, 170, 126, 105, 250, 197, 149, 192, 120}} //nolint:lll
	pchainClient.On("GetRewardUTXOs", ctx, arg).Return(bytes, nil).Once()

	pchainClient.On("GetHeight", ctx, mock.Anything).Return(uint64(1000000), nil)

	p, err = NewParser(pchainClient, constants.MainnetID)
	if err != nil {
		panic(err)
	}

	g, err = p.GetGenesisBlock(ctx)
	if err != nil {
		panic(err)
	}

	os.Exit(m.Run())
}

func TestGenesisBlockCreateChainTxs(t *testing.T) {
	a := assert.New(t)

	g.Txs = g.Txs[(len(g.Txs) - 2):]
	for _, tx := range g.Txs {
		castTx := tx.Unsigned.(*txs.CreateChainTx)
		castTx.GenesisData = []byte{}
	}

	g.UTXOs = []*pGenesis.UTXO{}

	j, err := stdjson.Marshal(g)
	if err != nil {
		panic(err)
	}

	ret := readFixture("outs/genesis.json")
	a.JSONEq(string(ret), string(j))
}

func TestGenesisBlockParseTxs(t *testing.T) {
	a := assert.New(t)
	pchainClient := &mocks.PChainClient{}

	p, err := NewParser(pchainClient, constants.FujiID)
	if err != nil {
		panic(err)
	}

	ctx := context.Background()
	g, err := p.GetGenesisBlock(ctx)
	if err != nil {
		panic(err)
	}

	initializeTxCtx(g.Txs, constants.FujiID)
	j, err := stdjson.MarshalIndent(g, "", "  ")
	if err != nil {
		panic(err)
	}

	ret := readFixture("outs/genesis_fuji_txs.json")
	a.JSONEq(string(ret), string(j))
}

func TestFixtures(t *testing.T) {
	ctx := context.Background()
	a := assert.New(t)

	for _, idx := range idxs {
		// +1 because we do -1 inside parseBlockAtIndex
		// and ins/outs are based on container ids
		// instead of block ids
		block, err := p.ParseNonGenesisBlock(ctx, "", idx+1)
		if err != nil {
			panic(err)
		}

		initializeTxCtx(block.Txs, constants.MainnetID)
		j, err := stdjson.Marshal(block)
		if err != nil {
			panic(err)
		}

		ret := readFixture("outs/%v.json", idx)
		a.JSONEq(string(ret), string(j))
	}
}

func initializeTxCtx(txs []*txs.Tx, networkID uint32) {
	aliaser := ids.NewAliaser()
	_ = aliaser.Alias(constants.PlatformChainID, rosConst.PChain.String())
	ctx := &snow.Context{
		BCLookup:  aliaser,
		NetworkID: networkID,
	}
	for _, tx := range txs {
		tx.Unsigned.InitCtx(ctx)
	}
}
