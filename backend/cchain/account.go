package cchain

import (
	"context"
	"errors"
	"fmt"

	"github.com/ava-labs/avalanche-rosetta/mapper"
	"github.com/ava-labs/coreth/interfaces"
	"github.com/coinbase/rosetta-sdk-go/types"
	"github.com/coinbase/rosetta-sdk-go/utils"
	"github.com/ethereum/go-ethereum/common/hexutil"

	cconstants "github.com/ava-labs/avalanche-rosetta/constants/cchain"
	cmapper "github.com/ava-labs/avalanche-rosetta/mapper/cchain"
	ethcommon "github.com/ethereum/go-ethereum/common"
)

// AccountBalance implements the /account/balance endpoint
func (b *Backend) AccountBalance(
	ctx context.Context,
	req *types.AccountBalanceRequest,
) (*types.AccountBalanceResponse, *types.Error) {
	if b.config.IsOfflineMode() {
		return nil, ErrUnavailableOffline
	}

	if req.AccountIdentifier == nil {
		return nil, WrapError(ErrInvalidInput, "account identifier is not provided")
	}

	header, terr := blockHeaderFromInput(ctx, b.client, req.BlockIdentifier)
	if terr != nil {
		return nil, terr
	}

	address := ethcommon.HexToAddress(req.AccountIdentifier.Address)

	nonce, err := b.client.NonceAt(ctx, address, header.Number)
	if err != nil {
		return nil, WrapError(ErrClientError, err)
	}

	metadata := &accountMetadata{
		Nonce: nonce,
	}

	metadataMap, err := mapper.MarshalJSONMap(metadata)
	if err != nil {
		return nil, WrapError(ErrInternalError, err)
	}

	avaxBalance, err := b.client.BalanceAt(ctx, address, header.Number)
	if err != nil {
		return nil, WrapError(ErrClientError, err)
	}

	balances := []*types.Amount{}
	if len(req.Currencies) == 0 {
		balances = append(balances, cmapper.AvaxAmount(avaxBalance))
	}

	for _, currency := range req.Currencies {
		value, ok := currency.Metadata[cmapper.ContractAddressMetadata]
		if !ok {
			if utils.Equal(currency, cconstants.AvaxCurrency) {
				balances = append(balances, cmapper.AvaxAmount(avaxBalance))
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
		response, err := b.client.CallContract(ctx, callMsg, header.Number)
		if err != nil {
			return nil, WrapError(ErrInternalError, err)
		}

		amount := cmapper.Erc20Amount(response, currency, false)

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
func (b Backend) AccountCoins(
	ctx context.Context,
	req *types.AccountCoinsRequest,
) (*types.AccountCoinsResponse, *types.Error) {
	if b.config.IsOfflineMode() {
		return nil, ErrUnavailableOffline
	}
	return nil, ErrNotImplemented
}
