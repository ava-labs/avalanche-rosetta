package service

import (
	"context"

	"github.com/ava-labs/coreth/ethclient"
	"github.com/coinbase/rosetta-sdk-go/server"
	"github.com/coinbase/rosetta-sdk-go/types"

	ethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"

	"github.com/figment-networks/avalanche-rosetta/mapper"
)

// AccountService implements the /account/* endpoints
type AccountService struct {
	config *Config
	client *ethclient.Client
}

// NewAccountService returns a new network servicer
func NewAccountService(config *Config, client *ethclient.Client) server.AccountAPIServicer {
	return &AccountService{
		config: config,
		client: client,
	}
}

// AccountBalance implements the /account/balance endpoint
func (s AccountService) AccountBalance(ctx context.Context, req *types.AccountBalanceRequest) (*types.AccountBalanceResponse, *types.Error) {
	if s.config.IsOfflineMode() {
		return nil, errUnavailableOffline
	}

	if req.AccountIdentifier == nil {
		return nil, wrapError(errInvalidInput, "account identifier is not provided")
	}

	header, err := blockHeaderFromInput(s.client, req.BlockIdentifier)
	if err != nil {
		log.Error("block header fetch failed:", err)
		return nil, errInternalError
	}

	address := ethcommon.HexToAddress(req.AccountIdentifier.Address)
	balance, balanceErr := s.client.BalanceAt(context.Background(), address, header.Number)
	if err != nil {
		log.Error("balance fetch failed:", balanceErr)
		return nil, errInternalError
	}

	resp := &types.AccountBalanceResponse{
		BlockIdentifier: &types.BlockIdentifier{
			Index: header.Number.Int64(),
			Hash:  header.Hash().String(),
		},
		Balances: []*types.Amount{
			mapper.AvaxAmount(balance),
		},
	}

	return resp, nil
}

// AccountCoins implements the /account/coins endpoint
func (s AccountService) AccountCoins(ctx context.Context, req *types.AccountCoinsRequest) (*types.AccountCoinsResponse, *types.Error) {
	if s.config.IsOfflineMode() {
		return nil, errUnavailableOffline
	}
	return nil, errNotImplemented
}
