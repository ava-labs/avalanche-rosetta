package client

import (
	"github.com/ava-labs/avalanchego/cache"
	"github.com/ava-labs/coreth/ethclient"
	"github.com/coinbase/rosetta-sdk-go/types"
	"github.com/ethereum/go-ethereum/common"
)

const (
	contractCacheSize       = 1024
	ContractAddressMetadata = "contractAddress"
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

// GetContractCurrency returns the currency for a specific address
func (c *ContractClient) GetContractCurrency(addr common.Address, erc20 bool) (*types.Currency, error) {
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

	currency := ContractCurrency(symbol, int32(decimals), addr.String())

	// Cache defaults for contract address to avoid unnecessary lookups
	c.cache.Put(addr, currency)
	return currency, nil
}

func ContractCurrency(symbol string, decimals int32, addr string) *types.Currency {
	return &types.Currency{
		Symbol:   symbol,
		Decimals: decimals,
		Metadata: map[string]interface{}{
			ContractAddressMetadata: addr,
		},
	}
}
