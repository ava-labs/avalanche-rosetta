package client

import (
	"context"
	"fmt"
	"strings"

	"github.com/ava-labs/avalanchego/cache"
	"github.com/ava-labs/coreth/core/types"
	"github.com/ava-labs/coreth/ethclient"
	"github.com/ava-labs/coreth/interfaces"
	"github.com/ethereum/go-ethereum/common"
)

const (
	logCacheSize       = 100
	transferMethodHash = "0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef"
)

// EvmLogsClient is a client for requesting evm logs
type EvmLogsClient struct {
	ethClient *ethclient.Client
	cache     *cache.LRU
}

// NewEvmLogsClient returns a new EVM Logs client
func NewEvmLogsClient(endpoint string) (*EvmLogsClient, error) {
	endpoint = strings.TrimSuffix(endpoint, "/")

	c, err := ethclient.Dial(fmt.Sprintf("%s%s", endpoint, prefixEth))
	if err != nil {
		return nil, err
	}

	cache := &cache.LRU{Size: logCacheSize}

	return &EvmLogsClient{
		ethClient: c,
		cache:     cache,
	}, nil
}

// EvmLogs returns a set of evm logs based on the requested block hash and transaction hash
func (c *EvmLogsClient) EvmLogs(
	ctx context.Context,
	blockHash common.Hash,
	transactionHash common.Hash,
) ([]types.Log, error) {
	blockLogs, isCached := c.cache.Get(blockHash.String())
	if !isCached {
		var err error
		var topics [][]common.Hash = [][]common.Hash{{common.HexToHash(transferMethodHash)}}

		var filter interfaces.FilterQuery = interfaces.FilterQuery{BlockHash: &blockHash, Topics: topics}
		blockLogs, err = c.ethClient.FilterLogs(ctx, filter)

		if err != nil {
			return nil, err
		}
		c.cache.Put(blockHash.String(), blockLogs)
	}

	var filteredLogs []types.Log

	for _, log := range blockLogs.([]types.Log) {
		if log.TxHash == transactionHash {
			filteredLogs = append(filteredLogs, log)
		}
	}

	return filteredLogs, nil
}
