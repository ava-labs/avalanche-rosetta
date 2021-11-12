package service

import (
	"context"
	"fmt"

	"github.com/coinbase/rosetta-sdk-go/server"
	"github.com/coinbase/rosetta-sdk-go/types"

	ethcommon "github.com/ethereum/go-ethereum/common"

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

	header, err := blockHeaderFromInput(s.client, req.BlockIdentifier)
	if err != nil {
		return nil, wrapError(errInternalError, err)
	}

	address := ethcommon.HexToAddress(req.AccountIdentifier.Address)

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

	avaxBalance, balanceErr := s.client.BalanceAt(context.Background(), address, header.Number)
	if balanceErr != nil {
		return nil, wrapError(errInternalError, balanceErr)
	}

	balances := make([]*types.Amount, len(req.Currencies)+1)
	balances = append(balances, mapper.AvaxAmount(avaxBalance))

	for _, currency := range req.Currencies {
		value, ok := currency.Metadata[mapper.ContractAddressMetadata]
		if !ok {
			return nil, wrapError(errCallInvalidParams, fmt.Errorf("currencies must have contractAddress in metadata field"))
		}
		data := BalanceOfMethodPrefix + req.AccountIdentifier.Address
		contractAddress := ethcommon.HexToAddress(value.(string))
		callMsg := interfaces.CallMsg{To: &contractAddress, Data: []byte(data)}
		response, err := s.client.CallContract(ctx, callMsg, header.Number)
		if err != nil {
			return nil, wrapError(errInternalError, err)
		}

		contractInfo, err := s.client.ContractInfo(contractAddress, true)
		if err != nil {
			return nil, wrapError(errInternalError, err)
		} else if contractInfo.Symbol == client.UnknownERC20Symbol {
			return nil, wrapError(errCallInvalidParams,
				fmt.Errorf("unable to pull contract info for %s", contractAddress.String()))
		}

		amount := mapper.Erc20Amount(response, contractAddress, contractInfo.Symbol, contractInfo.Decimals, false)
		balances = append(balances, amount)

		if err != nil {
			return nil, wrapError(errInternalError, err)
		}
	}

	resp := &types.AccountBalanceResponse{
		BlockIdentifier: &types.BlockIdentifier{
			Index: header.Number.Int64(),
			Hash:  header.Hash().String(),
		},
		Balances: balances,
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
