package client

import (
	"context"

	"github.com/ava-labs/avalanche-rosetta/cache"
	"github.com/ava-labs/coreth/core/types"
	"github.com/ava-labs/coreth/ethclient"
	"github.com/ava-labs/coreth/interfaces"
	"github.com/ethereum/go-ethereum/common"
)

// InfoClient is a client for the Info API
type EvmLogsClient struct {
	ethClient *ethclient.Client
	cache     *cache.LRU
}

// NewEthClient returns a new EVM client
func (c *EvmLogsClient) GetEvmLogs(ctx context.Context, blockHash *common.Hash, transactionHash *common.Hash) ([]types.Log, error) {
	blockLogs, isCached := c.cache.Get(blockHash.String())

	if !isCached {
		var err error
		var filter interfaces.FilterQuery = interfaces.FilterQuery{BlockHash: blockHash}
		blockLogs, err = c.ethClient.FilterLogs(ctx, filter)

		if err != nil {
			return nil, err
		}
	}

	var filteredLogs []types.Log

	for _, log := range blockLogs.([]types.Log) {
		if log.TxHash == *transactionHash {
			filteredLogs = append(filteredLogs, log)
		}
	}

	return filteredLogs, nil
}
