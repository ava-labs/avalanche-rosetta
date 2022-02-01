package client

import (
	"github.com/ava-labs/avalanchego/cache"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
)

const (
	contractCacheSize = 1024
)

// ContractClient is a client for the calling contract information
type ContractClient struct {
	ethClient *ethclient.Client
	cache     *cache.LRU
}

// NewContractClient returns a new ContractInfo client
func NewContractClient(endpointURL string) (*ContractClient, error) {
	c, err := ethclient.Dial(endpointURL)
	if err != nil {
		return nil, err
	}

	cache := &cache.LRU{Size: contractCacheSize}

	return &ContractClient{
		ethClient: c,
		cache:     cache,
	}, nil
}

// ContractInfo returns the ContractInfo for a specific address
func (c *ContractClient) ContractInfo(contractAddress common.Address, isErc20 bool) (*ContractInfo, error) {
	cachedInfo, isCached := c.cache.Get(contractAddress)

	if isCached {
		castCachedInfo := cachedInfo.(*ContractInfo)
		return castCachedInfo, nil
	}

	token, err := NewContractInfoToken(contractAddress, c.ethClient)
	if err != nil {
		return nil, err
	}
	symbol, symbolErr := token.Symbol(nil)
	decimals, decimalErr := token.Decimals(nil)

	// Any of these indicate a failure to get complete information from contract
	if symbolErr != nil || decimalErr != nil || symbol == "" || decimals == 0 {
		if isErc20 {
			symbol = UnknownERC20Symbol
			decimals = UnknownERC20Decimals
		} else {
			symbol = UnknownERC721Symbol
			decimals = UnknownERC721Decimals
		}
	}
	contractInfo := &ContractInfo{Symbol: symbol, Decimals: decimals}

	// Cache defaults for contract address to avoid unnecessary lookups
	c.cache.Put(contractAddress, contractInfo)
	return contractInfo, nil
}
