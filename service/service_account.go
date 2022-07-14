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

	"github.com/ava-labs/coreth/interfaces"

	"github.com/ava-labs/avalanche-rosetta/client"
	"github.com/ava-labs/avalanche-rosetta/mapper"
)

type AccountBackend interface {
	ShouldHandleRequest(req interface{}) bool
	AccountBalance(ctx context.Context, req *types.AccountBalanceRequest) (*types.AccountBalanceResponse, *types.Error)
	AccountCoins(ctx context.Context, req *types.AccountCoinsRequest) (*types.AccountCoinsResponse, *types.Error)
}

// AccountService implements the /account/* endpoints
type AccountService struct {
	config                *Config
	client                client.Client
	cChainAtomicTxBackend AccountBackend
	pChainBackend         AccountBackend
}

// NewAccountService returns a new network servicer
func NewAccountService(
	config *Config,
	client client.Client,
	pChainBackend AccountBackend,
	cChainAtomicTxBackend AccountBackend,
) server.AccountAPIServicer {
	return &AccountService{
		config:                config,
		client:                client,
		cChainAtomicTxBackend: cChainAtomicTxBackend,
		pChainBackend:         pChainBackend,
	}
}

// AccountBalance implements the /account/balance endpoint
func (s AccountService) AccountBalance(
	ctx context.Context,
	req *types.AccountBalanceRequest,
) (*types.AccountBalanceResponse, *types.Error) {
	if s.config.IsOfflineMode() {
		return nil, ErrUnavailableOffline
	}

	if s.pChainBackend.ShouldHandleRequest(req) {
		return s.pChainBackend.AccountBalance(ctx, req)
	}

	if req.AccountIdentifier == nil {
		return nil, WrapError(ErrInvalidInput, "account identifier is not provided")
	}

	// If the address is in Bech32 format, we check the atomic balance
	if s.cChainAtomicTxBackend.ShouldHandleRequest(req) {
		return s.cChainAtomicTxBackend.AccountBalance(ctx, req)
	}

	header, terr := blockHeaderFromInput(ctx, s.client, req.BlockIdentifier)
	if terr != nil {
		return nil, terr
	}

	address := ethcommon.HexToAddress(req.AccountIdentifier.Address)

	nonce, err := s.client.NonceAt(ctx, address, header.Number)
	if err != nil {
		return nil, WrapError(ErrClientError, err)
	}

	metadata := &accountMetadata{
		Nonce: nonce,
	}

	metadataMap, err := marshalJSONMap(metadata)
	if err != nil {
		return nil, WrapError(ErrInternalError, err)
	}

	avaxBalance, err := s.client.BalanceAt(ctx, address, header.Number)
	if err != nil {
		return nil, WrapError(ErrClientError, err)
	}

	balances := []*types.Amount{}
	if len(req.Currencies) == 0 {
		balances = append(balances, mapper.AvaxAmount(avaxBalance))
	}

	for _, currency := range req.Currencies {
		value, ok := currency.Metadata[mapper.ContractAddressMetadata]
		if !ok {
			if utils.Equal(currency, mapper.AvaxCurrency) {
				balances = append(balances, mapper.AvaxAmount(avaxBalance))
				continue
			}
			return nil, WrapError(ErrCallInvalidParams, errors.New("non-avax currencies must specify contractAddress in metadata"))
		}

		identifierAddress := req.AccountIdentifier.Address
		if has0xPrefix(identifierAddress) {
			identifierAddress = identifierAddress[2:42]
		}

		data, err := hexutil.Decode(BalanceOfMethodPrefix + identifierAddress)
		if err != nil {
			return nil, WrapError(ErrCallInvalidParams, fmt.Errorf("%w: marshalling balanceOf call msg data failed", err))
		}

		contractAddress := ethcommon.HexToAddress(value.(string))
		callMsg := interfaces.CallMsg{To: &contractAddress, Data: data}
		response, err := s.client.CallContract(ctx, callMsg, header.Number)
		if err != nil {
			return nil, WrapError(ErrInternalError, err)
		}

		amount := mapper.Erc20Amount(response, currency, false)

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
		return nil, ErrUnavailableOffline
	}

	if s.pChainBackend.ShouldHandleRequest(req) {
		return s.pChainBackend.AccountCoins(ctx, req)
	}

	if s.cChainAtomicTxBackend.ShouldHandleRequest(req) {
		return s.cChainAtomicTxBackend.AccountCoins(ctx, req)
	}

	return nil, ErrNotImplemented
}
