package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/coinbase/rosetta-sdk-go/server"
	"github.com/coinbase/rosetta-sdk-go/types"
	"github.com/coinbase/rosetta-sdk-go/utils"

	ethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"

	"github.com/ava-labs/avalanche-rosetta/client"
	"github.com/ava-labs/avalanche-rosetta/mapper"
	"github.com/ava-labs/coreth/interfaces"
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

	header, terr := blockHeaderFromInput(ctx, s.client, req.BlockIdentifier)
	if terr != nil {
		return nil, terr
	}

	address := ethcommon.HexToAddress(req.AccountIdentifier.Address)

	nonce, err := s.client.NonceAt(ctx, address, header.Number)
	if err != nil {
		return nil, wrapError(errClientError, err)
	}

	metadata := &accountMetadata{
		Nonce: nonce,
	}

	metadataMap, err := marshalJSONMap(metadata)
	if err != nil {
		return nil, wrapError(errInternalError, err)
	}

	avaxBalance, err := s.client.BalanceAt(ctx, address, header.Number)
	if err != nil {
		return nil, wrapError(errClientError, err)
	}

	balances := []*types.Amount{}
	if len(req.Currencies) == 0 {
		balances = append(balances, mapper.AvaxAmount(avaxBalance))
	}

	for _, currency := range req.Currencies {
		value, ok := currency.Metadata[client.ContractAddressMetadata]
		if !ok {
			if utils.Equal(currency, mapper.AvaxCurrency) {
				balances = append(balances, mapper.AvaxAmount(avaxBalance))
				continue
			}
			return nil, wrapError(errCallInvalidParams, errors.New("non-avax currencies must specify contractAddress in metadata"))
		}

		identifierAddress := req.AccountIdentifier.Address
		if has0xPrefix(identifierAddress) {
			identifierAddress = identifierAddress[2:42]
		}

		data, err := hexutil.Decode(BalanceOfMethodPrefix + identifierAddress)
		if err != nil {
			return nil, wrapError(errCallInvalidParams, fmt.Errorf("%w: marshalling balanceOf call msg data failed", err))
		}

		contractAddress := ethcommon.HexToAddress(value.(string))
		callMsg := interfaces.CallMsg{To: &contractAddress, Data: data}
		response, err := s.client.CallContract(ctx, callMsg, header.Number)
		if err != nil {
			return nil, wrapError(errInternalError, err)
		}

		amount := mapper.Erc20Amount(response, false, currency)

		balances = append(balances, amount)
	}

	return &types.AccountBalanceResponse{
		BlockIdentifier: &types.BlockIdentifier{
			Index: header.Number.Int64(),
			Hash:  header.Hash().String(),
		},
		Balances: balances,
		Metadata: metadataMap,
	}, nil
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
