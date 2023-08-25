package cchainatomictx

import (
	"context"
	"errors"
	"fmt"
	"math/big"

	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/avalanchego/utils/formatting/address"
	"github.com/ava-labs/avalanchego/utils/math"
	"github.com/ava-labs/avalanchego/vms/components/avax"
	"github.com/coinbase/rosetta-sdk-go/types"

	"github.com/ava-labs/avalanche-rosetta/constants"
	"github.com/ava-labs/avalanche-rosetta/mapper"
	"github.com/ava-labs/avalanche-rosetta/service"
	"github.com/ava-labs/avalanche-rosetta/service/backend/common"
)

var (
	errUnableToParseUTXO     = errors.New("unable to parse UTXO")
	errUnableToGetUTXOOutput = errors.New("unable to get UTXO output")
)

// AccountBalance implements /account/balance endpoint for C-chain atomic transaction balance
//
// This endpoint is called if the account is in Bech32 format for a C-chain request
func (b *Backend) AccountBalance(ctx context.Context, req *types.AccountBalanceRequest) (*types.AccountBalanceResponse, *types.Error) {
	if req.AccountIdentifier == nil {
		return nil, service.WrapError(service.ErrInvalidInput, "account identifier is not provided")
	}
	blockIdentifier, coins, wrappedErr := b.getAccountCoins(ctx, req.AccountIdentifier.Address)
	if wrappedErr != nil {
		return nil, wrappedErr
	}

	var balanceValue uint64

	for _, coin := range coins {
		amountValue, err := types.AmountValue(coin.Amount)
		if err != nil {
			return nil, service.WrapError(service.ErrInternalError, "unable to extract amount from UTXO")
		}

		balanceValue, err = math.Add64(balanceValue, amountValue.Uint64())
		if err != nil {
			return nil, service.WrapError(service.ErrInternalError, "overflow while calculating balance")
		}
	}

	balance := mapper.AtomicAvaxAmount(new(big.Int).SetUint64(balanceValue))
	return &types.AccountBalanceResponse{
		BlockIdentifier: blockIdentifier,
		Balances:        []*types.Amount{balance},
	}, nil
}

// AccountCoins implements /account/coins endpoint for C-chain atomic transaction UTXOs
//
// This endpoint is called if the account is in Bech32 format for a C-chain request
func (b *Backend) AccountCoins(ctx context.Context, req *types.AccountCoinsRequest) (*types.AccountCoinsResponse, *types.Error) {
	if req.AccountIdentifier == nil {
		return nil, service.WrapError(service.ErrInvalidInput, "account identifier is not provided")
	}
	blockIdentifier, coins, wrappedErr := b.getAccountCoins(ctx, req.AccountIdentifier.Address)
	if wrappedErr != nil {
		return nil, wrappedErr
	}

	return &types.AccountCoinsResponse{
		BlockIdentifier: blockIdentifier,
		Coins:           common.SortUnique(coins),
	}, nil
}

func (b *Backend) getAccountCoins(ctx context.Context, address string) (*types.BlockIdentifier, []*types.Coin, *types.Error) {
	var coins []*types.Coin
	sourceChains := []constants.ChainIDAlias{
		constants.PChain,
		constants.XChain,
	}

	preHeader, err := b.cClient.HeaderByNumber(ctx, nil)
	if err != nil {
		return nil, nil, service.WrapError(service.ErrInternalError, err)
	}

	for _, chain := range sourceChains {
		chainCoins, wrappedErr := b.fetchCoinsFromChain(ctx, address, chain)
		if wrappedErr != nil {
			return nil, nil, wrappedErr
		}
		coins = append(coins, chainCoins...)
	}

	postHeader, err := b.cClient.HeaderByNumber(ctx, nil)
	if err != nil {
		return nil, nil, service.WrapError(service.ErrInternalError, err)
	}

	// Since there is no API to return coins and block height at the time of query, we lookup block height before and after
	// and fail the request if theyd differ since it means we don't know which block the coins are at
	if preHeader.Number.Cmp(postHeader.Number) != 0 {
		return nil, nil, service.WrapError(service.ErrInternalError, "new block received while fetching coins")
	}

	blockIdentifier := &types.BlockIdentifier{
		Index: postHeader.Number.Int64(),
		Hash:  postHeader.Hash().String(),
	}

	return blockIdentifier, coins, nil
}

func (b *Backend) fetchCoinsFromChain(ctx context.Context, prefixAddress string, sourceChain constants.ChainIDAlias) ([]*types.Coin, *types.Error) {
	var (
		coins []*types.Coin

		// Used for pagination
		lastUtxoAddress ids.ShortID
		lastUtxoID      ids.ID
	)

	// Parse [prefixAddress]
	chainIDAlias, _, addressBytes, err := address.Parse(prefixAddress)
	if err != nil {
		return nil, service.WrapError(service.ErrInternalError, err)
	}
	if chainIDAlias != constants.CChain.String() {
		return nil, service.WrapError(
			service.ErrInternalError,
			fmt.Errorf("invalid ChainID alias wanted=%s have=%s", constants.CChain.String(), chainIDAlias),
		)
	}
	addr, err := ids.ToShortID(addressBytes)
	if err != nil {
		return nil, service.WrapError(service.ErrInternalError, err)
	}

	// GetUTXOs controlled by [addr]
	for {
		utxos, newUtxoAddress, newUtxoID, err := b.cClient.GetAtomicUTXOs(ctx, []ids.ShortID{addr}, sourceChain.String(), b.getUTXOsPageSize, lastUtxoAddress, lastUtxoID)
		if err != nil {
			return nil, service.WrapError(service.ErrInternalError, "unable to get UTXOs")
		}

		// convert raw UTXO bytes to Rosetta Coins
		coinsPage, err := b.processUtxos(sourceChain, utxos)
		if err != nil {
			return nil, service.WrapError(service.ErrInternalError, err)
		}

		coins = append(coins, coinsPage...)

		// Fetch next page only if there may be more UTXOs
		if len(utxos) < int(b.getUTXOsPageSize) {
			break
		}

		lastUtxoAddress = newUtxoAddress
		lastUtxoID = newUtxoID
	}

	return coins, nil
}

func (b *Backend) processUtxos(sourceChain constants.ChainIDAlias, utxos [][]byte) ([]*types.Coin, error) {
	coins := make([]*types.Coin, 0)
	for _, utxoBytes := range utxos {
		utxo := avax.UTXO{}
		_, err := b.codec.Unmarshal(utxoBytes, &utxo)
		if err != nil {
			return nil, errUnableToParseUTXO
		}

		transferableOut, ok := utxo.Out.(avax.TransferableOut)
		if !ok {
			return nil, errUnableToGetUTXOOutput
		}

		amount := mapper.AtomicAvaxAmount(new(big.Int).SetUint64(transferableOut.Amount()))
		amount.Metadata = map[string]interface{}{
			"source_chain": sourceChain.String(),
		}

		coin := &types.Coin{
			CoinIdentifier: &types.CoinIdentifier{Identifier: utxo.UTXOID.String()},
			Amount:         amount,
		}
		coins = append(coins, coin)
	}
	return coins, nil
}
