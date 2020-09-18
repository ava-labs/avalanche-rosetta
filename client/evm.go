package client

import (
	"fmt"

	"github.com/ethereum/go-ethereum/ethclient"
)

// EvmClient implements the EMV API spec (Ethereum Virtual Machine)
// https://docs.avax.network/v1.0/en/api/evm/
type EvmClient struct {
	*ethclient.Client
}

func NewEvmClient(endpoint string) *EvmClient {
	ethclient, err := ethclient.Dial(fmt.Sprintf("%s%s", endpoint, PrefixEVM))
	if err != nil {
		// TODO: this should not panic at all
		panic(err)
	}
	return &EvmClient{ethclient}
}
