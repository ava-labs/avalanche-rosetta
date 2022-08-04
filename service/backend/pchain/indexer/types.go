package indexer

import (
	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/avalanchego/vms/platformvm"
)

type ParsedBlock struct {
	BlockID   ids.ID           `json:"id"`
	BlockType string           `json:"type"`
	ParentID  ids.ID           `json:"parent"`
	Timestamp int64            `json:"timestamp"`
	Height    uint64           `json:"height"`
	Txs       []*platformvm.Tx `json:"transactions"`
	Proposer  `json:"proposer"`
}

type GenesisBlockData struct {
	Message       string                    `json:"message"`
	InitialSupply uint64                    `json:"initialSupply"`
	UTXOs         []*platformvm.GenesisUTXO `json:"utxos"`
}

type ParsedGenesisBlock struct {
	ParsedBlock
	GenesisBlockData `json:"data"`
}

type Proposer struct {
	ID           ids.ID     `json:"id"`
	ParentID     ids.ID     `json:"parent"`
	NodeID       ids.NodeID `json:"nodeID"`
	PChainHeight uint64     `json:"pChainHeight"`
	Timestamp    int64      `json:"timestamp"`
}
