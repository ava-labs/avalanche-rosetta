package indexer

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"testing"

	"github.com/ava-labs/avalanchego/api"
	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/avalanchego/indexer"
	"github.com/ava-labs/avalanchego/snow"
	"github.com/ava-labs/avalanchego/utils/formatting"
	"github.com/ava-labs/avalanchego/vms/platformvm/genesis"
	"github.com/ava-labs/avalanchego/vms/platformvm/txs"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/ava-labs/avalanche-rosetta/client"
	"github.com/ava-labs/avalanche-rosetta/constants"

	avaconstants "github.com/ava-labs/avalanchego/utils/constants"
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
	ret, err := os.ReadFile("testdata/" + relpath)
	if err != nil {
		panic(err)
	}

	return ret
}

func TestMain(m *testing.M) {
	ctx := context.Background()
	pchainClient := client.NewMockPChainClient(gomock.NewController(nil))

	for _, idx := range idxs {
		ret := readFixture("ins/%v.json", idx)

		var container indexer.Container
		err := json.Unmarshal(ret, &container)
		if err != nil {
			panic(err)
		}

		pchainClient.EXPECT().GetContainerByIndex(ctx, idx).Return(container, nil)
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
	pchainClient.EXPECT().GetRewardUTXOs(ctx, arg).Return(bytes, nil)

	pchainClient.EXPECT().GetHeight(ctx, gomock.Any()).Return(uint64(1000000), nil)

	p, err = NewParser(pchainClient, avaconstants.MainnetID)
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
	require := require.New(t)

	g.Txs = g.Txs[(len(g.Txs) - 2):]
	for _, tx := range g.Txs {
		castTx := tx.Unsigned.(*txs.CreateChainTx)
		castTx.GenesisData = []byte{}
	}

	g.UTXOs = []*genesis.UTXO{}

	j, err := json.Marshal(g)
	require.NoError(err)

	ret := readFixture("outs/genesis.json")
	require.JSONEq(string(ret), string(j))
}

func TestGenesisBlockParseTxs(t *testing.T) {
	require := require.New(t)
	ctrl := gomock.NewController(t)
	pchainClient := client.NewMockPChainClient(ctrl)

	p, err := NewParser(pchainClient, avaconstants.FujiID)
	require.NoError(err)

	ctx := context.Background()
	g, err := p.GetGenesisBlock(ctx)
	require.NoError(err)

	initializeTxCtx(g.Txs, avaconstants.FujiID)
	j, err := json.MarshalIndent(g, "", "  ")
	require.NoError(err)

	ret := readFixture("outs/genesis_fuji_txs.json")
	require.JSONEq(string(ret), string(j))
}

func TestFixtures(t *testing.T) {
	require := require.New(t)
	ctx := context.Background()

	for _, idx := range idxs {
		// +1 because we do -1 inside parseBlockAtIndex
		// and ins/outs are based on container ids
		// instead of block ids
		block, err := p.ParseNonGenesisBlock(ctx, "", idx+1)
		require.NoError(err)

		initializeTxCtx(block.Txs, avaconstants.MainnetID)
		j, err := json.Marshal(block)
		require.NoError(err)

		ret := readFixture("outs/%v.json", idx)
		require.JSONEq(string(ret), string(j))
	}
}

func initializeTxCtx(txs []*txs.Tx, networkID uint32) {
	aliaser := ids.NewAliaser()
	_ = aliaser.Alias(avaconstants.PlatformChainID, constants.PChain.String())
	ctx := &snow.Context{
		BCLookup:  aliaser,
		NetworkID: networkID,
	}
	for _, tx := range txs {
		tx.Unsigned.InitCtx(ctx)
	}
}
