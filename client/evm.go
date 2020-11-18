package client

import (
	"fmt"

	"github.com/ethereum/go-ethereum/ethclient"
)

// EvmClient implements the EMV API spec (Ethereum Virtual Machine)
type EvmClient struct {
	*ethclient.Client
}

func NewEvmClient(endpoint string) (*EvmClient, error) {
	ethclient, err := ethclient.Dial(fmt.Sprintf("%s%s", endpoint, PrefixEVM))
	if err != nil {
		return nil, err
	}
	return &EvmClient{ethclient}, nil
}
