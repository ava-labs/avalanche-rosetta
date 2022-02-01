package client

import (
	"github.com/ava-labs/avalanchego/cache"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
)

const (
	contractCacheSize       = 1024
	contractAddressMetadata = "contractAddress"
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

// GetContractCurrency returns the currency for a specific address
func (c *ContractClient) GetContractCurrency(addr common.Address, erc20 bool) (*ContractCurrency, error) {
	if currency, cached := c.cache.Get(addr); cached {
		return currency.(*ContractCurrency), nil
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

	currency := &ContractCurrency{
		Symbol:   symbol,
		Decimals: int32(decimals),
	}

	// Cache defaults for contract address to avoid unnecessary lookups
	c.cache.Put(addr, currency)
	return currency, nil
}
