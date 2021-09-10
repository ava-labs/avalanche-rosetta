package service

import (
	"context"

	"github.com/coinbase/rosetta-sdk-go/server"
	"github.com/coinbase/rosetta-sdk-go/types"

	ethcommon "github.com/ethereum/go-ethereum/common"

	"github.com/ava-labs/avalanche-rosetta/client"
	"github.com/ava-labs/avalanche-rosetta/mapper"
)

// AccountService implements the /account/* endpoints
type AccountService struct {
	config *Config
	client client.Client
}

// NewAccountService returns a new network servicer
func NewAccountService(config *Config, client client.Client) server.AccountAPIServicer {
	return &AccountService{
		config: config,
		client: client,
	}
}

// AccountBalance implements the /account/balance endpoint
func (s AccountService) AccountBalance(
	ctx context.Context,
	req *types.AccountBalanceRequest,
) (*types.AccountBalanceResponse, *types.Error) {
	if s.config.IsOfflineMode() {
		return nil, errUnavailableOffline
	}

	if req.AccountIdentifier == nil {
		return nil, wrapError(errInvalidInput, "account identifier is not provided")
	}

	header, err := blockHeaderFromInput(s.client, req.BlockIdentifier)
	if err != nil {
		return nil, wrapError(errInternalError, err)
	}

	address := ethcommon.HexToAddress(req.AccountIdentifier.Address)
	balance, balanceErr := s.client.BalanceAt(context.Background(), address, header.Number)
	if err != nil {
		return nil, wrapError(errInternalError, balanceErr)
	}

	nonce, nonceErr := s.client.NonceAt(ctx, address, header.Number)
	if nonceErr != nil {
		return nil, wrapError(errClientError, nonceErr)
	}

	metadata := &accountMetadata{
		Nonce: nonce,
	}

	metadataMap, metadataErr := marshalJSONMap(metadata)
	if err != nil {
		return nil, wrapError(errInternalError, metadataErr)
	}

	resp := &types.AccountBalanceResponse{
		BlockIdentifier: &types.BlockIdentifier{
			Index: header.Number.Int64(),
			Hash:  header.Hash().String(),
		},
		Balances: []*types.Amount{
			mapper.AvaxAmount(balance),
		},
		Metadata: metadataMap,
	}

	return resp, nil
}

// AccountCoins implements the /account/coins endpoint
func (s AccountService) AccountCoins(
	ctx context.Context,
	req *types.AccountCoinsRequest,
) (*types.AccountCoinsResponse, *types.Error) {
	if s.config.IsOfflineMode() {
		return nil, errUnavailableOffline
	}
	return nil, errNotImplemented
}
