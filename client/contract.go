package client

import (
	"fmt"
	"strings"

	"github.com/ava-labs/avalanche-rosetta/cache"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
)

// InfoClient is a client for the Info API
type ContractClient struct {
	ethClient *ethclient.Client
	cache     *cache.LRU
}

// NewEthClient returns a new EVM client
func NewContractClient(endpoint string) (*ContractClient, error) {
	endpoint = strings.TrimSuffix(endpoint, "/")

	c, err := ethclient.Dial(fmt.Sprintf("%s%s", endpoint, prefixEth))
	if err != nil {
		return nil, err
	}

	cache := &cache.LRU{Size: 100}

	return &ContractClient{
		ethClient: c,
		cache:     cache,
	}, nil
}

// NewEthClient returns a new EVM client
func (c *ContractClient) ContractInfo(contractAddress common.Address, isErc20 bool) (*ContractInfo, error) {
	cachedInfo, isCached := c.cache.Get(contractAddress)

	if isCached {
		return cachedInfo.(*ContractInfo), nil
	}

	token, err := NewContractInfoToken(contractAddress, c.ethClient)
	if err != nil {
		return nil, err
	}
	symbol, symbolErr := token.Symbol(nil)
	decimals, decimalErr := token.Decimals(nil)

	//Any of these indicate a failure to get complete information from contract
	if symbolErr != nil || decimalErr != nil || symbol == "" || decimals == 0 {
		if isErc20 {
			symbol = ERC20DefaultSymbol
			decimals = ERC20DefaultDecimals
		} else {
			symbol = ERC721DefaultSymbol
			decimals = ERC721DefaultDecimals
		}
	}
	contractInfo := ContractInfo{Symbol: symbol, Decimals: decimals}

	//Cache defaults for contract address to avoid unnecessary lookups
	c.cache.Put(contractAddress, contractInfo)
	return &contractInfo, nil
}
