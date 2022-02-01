package client

import (
	"github.com/ava-labs/avalanchego/cache"
	"github.com/coinbase/rosetta-sdk-go/types"
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

// ContractCurrency returns the currency for a specific address
func (c *ContractClient) ContractCurrency(addr common.Address, erc20 bool) (*types.Currency, error) {
	if currency, cached := c.cache.Get(addr); cached {
		return currency.(*types.Currency), nil
	}

	token, err := NewContractInfoToken(addr, c.ethClient)
	if err != nil {
		return nil, err
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

	currency := &types.Currency{
		Symbol:   symbol,
		Decimals: int32(decimals),
	}

	// Cache defaults for contract address to avoid unnecessary lookups
	c.cache.Put(addr, currency)
	return currency, nil
}
