package client

import (
	"github.com/ava-labs/avalanchego/cache"
	"github.com/ava-labs/coreth/ethclient"
	"github.com/ethereum/go-ethereum/common"
)

const (
	contractCacheSize = 1024
)

// ContractClient is a client for the calling contract information
type ContractClient struct {
	ethClient ethclient.Client
	cache     *cache.LRU
}

// NewContractClient returns a new ContractInfo client
func NewContractClient(c ethclient.Client) *ContractClient {
	return &ContractClient{
		ethClient: c,
		cache:     &cache.LRU{Size: contractCacheSize},
	}
}

// GetContractInfo returns the symbol and decimals for [addr].
func (c *ContractClient) GetContractInfo(addr common.Address, erc20 bool) (string, uint8, error) {
	// We don't define another struct because this is never used outside of this
	// function.
	type ContractInfo struct {
		Symbol   string
		Decimals uint8
	}

	if currency, cached := c.cache.Get(addr); cached {
		cast := currency.(*ContractInfo)
		return cast.Symbol, cast.Decimals, nil
	}

	token, err := NewContractInfoToken(addr, c.ethClient)
	if err != nil {
		return "", 0, err
	}

	symbol, symbolErr := token.Symbol(nil)
	decimals, decimalErr := token.Decimals(nil)

	// Any of these indicate a failure to get complete information from contract
	if symbolErr != nil || decimalErr != nil || symbol == "" || decimals == 0 {
		if erc20 {
			symbol = UnknownERC20Symbol
			decimals = UnknownERC20Decimals
		} else {
			symbol = UnknownERC721Symbol
			decimals = UnknownERC721Decimals
		}
	}

	// Cache defaults for contract address to avoid unnecessary lookups
	c.cache.Put(addr, &ContractInfo{
		Symbol:   symbol,
		Decimals: decimals,
	})
	return symbol, decimals, nil
}
