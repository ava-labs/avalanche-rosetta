package client

import (
	"context"

	"github.com/ava-labs/avalanchego/cache"
	"github.com/ava-labs/coreth/core/types"
	"github.com/ava-labs/coreth/ethclient"
	"github.com/ava-labs/coreth/interfaces"
	"github.com/ethereum/go-ethereum/common"
)

const (
	logCacheSize       = 1024
	transferMethodHash = "0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef"
)

// EvmLogsClient is a client for requesting evm logs
type EvmLogsClient struct {
	ethClient ethclient.Client
	cache     *cache.LRU
}

// NewEvmLogsClient returns a new EVM Logs client
func NewEvmLogsClient(c ethclient.Client) *EvmLogsClient {
	return &EvmLogsClient{
		ethClient: c,
		cache:     &cache.LRU{Size: logCacheSize},
	}
}

// EvmTransferLogs returns a set of evm logs based on the requested block hash and transaction hash
func (c *EvmLogsClient) EvmTransferLogs(
	ctx context.Context,
	blockHash common.Hash,
	transactionHash common.Hash,
) ([]types.Log, error) {
	logs, cached := c.cache.Get(blockHash.String())
	if !cached {
		var err error

		topics := [][]common.Hash{{common.HexToHash(transferMethodHash)}}
		filter := interfaces.FilterQuery{BlockHash: &blockHash, Topics: topics}
		logs, err = c.ethClient.FilterLogs(ctx, filter)
		if err != nil {
			return nil, err
		}

		c.cache.Put(blockHash.String(), logs)
	}

	var filteredLogs []types.Log

	for _, log := range logs.([]types.Log) {
		if log.TxHash == transactionHash {
			filteredLogs = append(filteredLogs, log)
		}
	}

	return filteredLogs, nil
}
