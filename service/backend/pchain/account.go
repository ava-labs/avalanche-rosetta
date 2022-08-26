package pchain

import (
	"context"
	"errors"
	"strconv"
	"time"

	"github.com/ava-labs/avalanchego/vms/platformvm/stakeable"
	"github.com/ava-labs/avalanchego/vms/secp256k1fx"

	pmapper "github.com/ava-labs/avalanche-rosetta/mapper/pchain"
	"github.com/ava-labs/avalanche-rosetta/service/backend/common"

	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/avalanchego/utils/formatting/address"
	"github.com/ava-labs/avalanchego/utils/math"
	"github.com/ava-labs/avalanchego/vms/components/avax"
	"github.com/coinbase/rosetta-sdk-go/types"

	"github.com/ava-labs/avalanche-rosetta/mapper"
	"github.com/ava-labs/avalanche-rosetta/service"
)

var (
	errUnableToGetUTXOs           = errors.New("unable to get UTXOs")
	errUnableToParseUTXO          = errors.New("unable to parse UTXO")
	errUnableToGetUTXOOut         = errors.New("unable to get UTXO output")
	errTotalOverflow              = errors.New("overflow while calculating total balance")
	errUnlockedOverflow           = errors.New("overflow while calculating unlocked balance")
	errLockedOverflow             = errors.New("overflow while calculating locked balance")
	errNotStakeableOverflow       = errors.New("overflow while calculating locked not stakeable balance")
	errLockedNotStakeableOverflow = errors.New("overflow while calculating locked not stakeable balance")
	errUnlockedStakeableOverflow  = errors.New("overflow while calculating unlocked stakeable balance")
)

func (b *Backend) AccountBalance(ctx context.Context, req *types.AccountBalanceRequest) (*types.AccountBalanceResponse, *types.Error) {
	if req.AccountIdentifier == nil {
		return nil, service.WrapError(service.ErrInvalidInput, "account identifier is not provided")
	}
	if req.BlockIdentifier != nil {
		return nil, service.WrapError(service.ErrNotSupported, "historical balance lookups are not supported")
	}

	var balanceType string
	if req.AccountIdentifier.SubAccount != nil {
		balanceType = req.AccountIdentifier.SubAccount.Address
	}
	fetchImportable := balanceType == pmapper.SubAccountTypeSharedMemory

	height, balance, typedErr := b.fetchBalance(ctx, req.AccountIdentifier.Address, fetchImportable)
	if typedErr != nil {
		return nil, typedErr
	}

	var balanceValue uint64
	switch balanceType {
	case pmapper.SubAccountTypeUnlocked:
		balanceValue = balance.Unlocked
	case pmapper.SubAccountTypeLockedStakeable:
		balanceValue = balance.LockedStakeable
	case pmapper.SubAccountTypeLockedNotStakeable:
		balanceValue = balance.LockedNotStakeable
	case pmapper.SubAccountTypeStaked:
		balanceValue = balance.Staked
	case pmapper.SubAccountTypeSharedMemory:
		balanceValue = balance.Total
	case "": // Defaults to total balance
		balanceValue = balance.Total
	default:
		return nil, service.WrapError(service.ErrInvalidInput, "unknown account type "+balanceType)
	}

	block, err := b.getBlockDetails(ctx, int64(height), "")
	if err != nil {
		return nil, service.WrapError(service.ErrInvalidInput, "unable to get height")
	}

	return &types.AccountBalanceResponse{
		BlockIdentifier: &types.BlockIdentifier{
			Index: int64(height),
			Hash:  block.BlockID.String(),
		},
		Balances: []*types.Amount{
			{
				Value:    strconv.FormatUint(balanceValue, 10),
				Currency: mapper.AtomicAvaxCurrency,
			},
		},
	}, nil
}

func (b *Backend) AccountCoins(ctx context.Context, req *types.AccountCoinsRequest) (*types.AccountCoinsResponse, *types.Error) {
	if req.AccountIdentifier == nil {
		return nil, service.WrapError(service.ErrInvalidInput, "account identifier is not provided")
	}
	addr, err := address.ParseToID(req.AccountIdentifier.Address)
	if err != nil {
		return nil, service.WrapError(service.ErrInvalidInput, "unable to convert address")
	}

	currencyAssetIDs, wrappedErr := b.buildCurrencyAssetIDs(ctx, req)
	if err != nil {
		return nil, wrappedErr
	}

	var subAccountAddress string
	if req.AccountIdentifier.SubAccount != nil {
		subAccountAddress = req.AccountIdentifier.SubAccount.Address
	}
	fetchSharedMemory := subAccountAddress == pmapper.SubAccountTypeSharedMemory

	height, utxos, _, typedErr := b.fetchUTXOsAndStakedOutputs(ctx, addr, false, fetchSharedMemory)
	if typedErr != nil {
		return nil, typedErr
	}

	// convert raw UTXO bytes to Rosetta Coins
	coins, err := b.processUtxos(currencyAssetIDs, utxos)
	if err != nil {
		return nil, service.WrapError(service.ErrInternalError, err)
	}

	block, err := b.getBlockDetails(ctx, int64(height), "")
	if err != nil {
		return nil, service.WrapError(service.ErrInvalidInput, "unable to get height")
	}

	return &types.AccountCoinsResponse{
		BlockIdentifier: &types.BlockIdentifier{
			Index: int64(height),
			Hash:  block.BlockID.String(),
		},
		Coins: common.SortUnique(coins),
	}, nil
}

func (b *Backend) fetchBalance(ctx context.Context, addrString string, fetchImportable bool) (uint64, *AccountBalance, *types.Error) {
	addr, err := address.ParseToID(addrString)
	if err != nil {
		return 0, nil, service.WrapError(service.ErrInvalidInput, "unable to convert address")
	}

	height, utxos, stakedUTXOBytes, typedErr := b.fetchUTXOsAndStakedOutputs(ctx, addr, !fetchImportable, fetchImportable)
	if typedErr != nil {
		return 0, nil, typedErr
	}

	balance, err := b.getBalancesWithoutMultisig(utxos)
	if err != nil {
		return 0, nil, service.WrapError(service.ErrInternalError, err)
	}

	// parse staked UTXO bytes to UTXO structs
	stakedAmount, err := b.calculateStakedAmount(stakedUTXOBytes)
	if err != nil {
		return 0, nil, service.WrapError(service.ErrInternalError, err)
	}

	balance.Staked = stakedAmount
	balance.Total += stakedAmount

	return height, balance, nil
}

// Copy of the platformvm service's GetBalance implementation.
// This is needed as multisig UTXOs are cleaned in parseUTXOs and its output must be used for the calculations. Ref:
// https://github.com/ava-labs/avalanchego/blob/0950acab667e0c16a55e9a9bb72bcbe25c3b88cf/vms/platformvm/service.go#L184
func (b *Backend) getBalancesWithoutMultisig(utxos []avax.UTXO) (*AccountBalance, error) {
	currentTime := uint64(time.Now().Unix())

	accountBalance := &AccountBalance{
		Total:              0,
		Staked:             0,
		Unlocked:           0,
		LockedStakeable:    0,
		LockedNotStakeable: 0,
	}

utxoFor:
	for _, utxo := range utxos {
		switch out := utxo.Out.(type) {
		case *secp256k1fx.TransferOutput:
			if out.Locktime <= currentTime {
				newBalance, err := math.Add64(accountBalance.Unlocked, out.Amount())
				if err != nil {
					return nil, errUnlockedOverflow
				}
				accountBalance.Unlocked = newBalance
			} else {
				newBalance, err := math.Add64(accountBalance.LockedNotStakeable, out.Amount())
				if err != nil {
					return nil, errNotStakeableOverflow
				}
				accountBalance.LockedNotStakeable = newBalance
			}
		case *stakeable.LockOut:
			innerOut, ok := out.TransferableOut.(*secp256k1fx.TransferOutput)
			switch {
			case !ok:
				continue utxoFor
			case innerOut.Locktime > currentTime:
				newBalance, err := math.Add64(accountBalance.LockedNotStakeable, out.Amount())
				if err != nil {
					return nil, errLockedNotStakeableOverflow
				}
				accountBalance.LockedNotStakeable = newBalance
			case out.Locktime <= currentTime:
				newBalance, err := math.Add64(accountBalance.Unlocked, out.Amount())
				if err != nil {
					return nil, errUnlockedOverflow
				}
				accountBalance.Unlocked = newBalance
			default:
				newBalance, err := math.Add64(accountBalance.LockedStakeable, out.Amount())
				if err != nil {
					return nil, errUnlockedStakeableOverflow
				}
				accountBalance.LockedStakeable = newBalance
			}
		default:
			continue utxoFor
		}
	}

	lockedBalance, err := math.Add64(accountBalance.LockedStakeable, accountBalance.LockedNotStakeable)
	if err != nil {
		return nil, errLockedOverflow
	}

	totalBalance, err := math.Add64(accountBalance.Unlocked, lockedBalance)
	if err != nil {
		return nil, errTotalOverflow
	}

	accountBalance.Total = totalBalance

	return accountBalance, nil
}

func (b *Backend) buildCurrencyAssetIDs(ctx context.Context, req *types.AccountCoinsRequest) (map[ids.ID]struct{}, *types.Error) {
	currencyAssetIDs := make(map[ids.ID]struct{})
	for _, reqCurrency := range req.Currencies {
		description, err := b.pClient.GetAssetDescription(ctx, reqCurrency.Symbol)
		if err != nil {
			return nil, service.WrapError(service.ErrInternalError, "unable to get asset description")
		}
		if int32(description.Denomination) != reqCurrency.Decimals {
			return nil, service.WrapError(service.ErrInvalidInput, "incorrect currency decimals")
		}
		currencyAssetIDs[description.AssetID] = struct{}{}
	}

	return currencyAssetIDs, nil
}

// Fetches UTXOs and staked outputs for the given account.
//
// Since these APIs don't return the corresponding block height or hash,
// which is needed for both /account/balance and /account/coins, chain height is checked before and after
// and if they differ, an error is returned.
func (b *Backend) fetchUTXOsAndStakedOutputs(
	ctx context.Context,
	addr ids.ShortID,
	fetchStaked bool,
	fetchSharedMemory bool,
) (uint64, []avax.UTXO, [][]byte, *types.Error) {
	// fetch preHeight before the balance fetch
	preHeight, err := b.pClient.GetHeight(ctx)
	if err != nil {
		return 0, nil, nil, service.WrapError(service.ErrInvalidInput, "unable to get postHeight")
	}

	var sourceChains []string
	if fetchSharedMemory {
		sourceChains = []string{
			mapper.CChainNetworkIdentifier,
			mapper.XChainNetworkIdentifier,
		}
	} else {
		sourceChains = []string{""}
	}

	var utxoBytes [][]byte

	for _, sc := range sourceChains {
		// fetch all UTXOs for addr
		chainUtxoBytes, err := b.getAccountUTXOs(ctx, addr, sc)
		if err != nil {
			return 0, nil, nil, service.WrapError(service.ErrInternalError, err)
		}
		utxoBytes = append(utxoBytes, chainUtxoBytes...)
	}

	if err != nil {
		return 0, nil, nil, service.WrapError(service.ErrInternalError, err)
	}

	var stakedUTXOBytes [][]byte
	if fetchStaked {
		// fetch staked outputs for addr
		_, stakedUTXOBytes, err = b.pClient.GetStake(ctx, []ids.ShortID{addr})
		if err != nil {
			return 0, nil, nil, service.WrapError(service.ErrInvalidInput, "unable to get stake")
		}
	}

	// fetch postHeight after the balance fetch and compare with preHeight
	postHeight, err := b.pClient.GetHeight(ctx)
	if err != nil {
		return 0, nil, nil, service.WrapError(service.ErrInvalidInput, "unable to get postHeight")
	}
	if postHeight != preHeight {
		return 0, nil, nil, service.WrapError(service.ErrInternalError, "new block added while fetching utxos")
	}

	// parse UTXO bytes to UTXO structs
	utxos, err := b.parseUTXOs(utxoBytes)
	if err != nil {
		return 0, nil, nil, service.WrapError(service.ErrInternalError, err)
	}

	return postHeight, utxos, stakedUTXOBytes, nil
}

func (b *Backend) calculateStakedAmount(stakeUTXOs [][]byte) (uint64, error) {
	staked := uint64(0)

	for _, utxoBytes := range stakeUTXOs {
		utxo := avax.TransferableOutput{}

		_, err := b.codec.Unmarshal(utxoBytes, &utxo)
		if err != nil {
			return 0, errUnableToParseUTXO
		}

		outIntf := utxo.Out
		if lockedOut, ok := outIntf.(*stakeable.LockOut); ok {
			outIntf = lockedOut.TransferableOut
		}

		out, ok := outIntf.(*secp256k1fx.TransferOutput)
		if !ok {
			return 0, errUnableToParseUTXO
		}

		// ignore multisig
		if len(out.OutputOwners.Addrs) > 1 {
			continue
		}

		staked += out.Amt
	}

	return staked, nil
}

func (b *Backend) parseUTXOs(utxoBytes [][]byte) ([]avax.UTXO, error) {
	utxos := []avax.UTXO{}

	// when results are paginated, duplicate UTXOs may be provided. guarantee uniqueness
	utxoIDs := make(map[string]struct{})
	for _, bytes := range utxoBytes {
		utxo := avax.UTXO{}
		_, err := b.codec.Unmarshal(bytes, &utxo)
		if err != nil {
			return nil, errUnableToParseUTXO
		}

		if _, ok := utxoIDs[utxo.UTXOID.String()]; ok {
			continue
		}

		utxoIDs[utxo.UTXOID.String()] = struct{}{}

		// Skip multisig UTXOs
		addressable, ok := utxo.Out.(avax.Addressable)
		if !ok {
			return nil, errUnableToGetUTXOOut
		}
		if len(addressable.Addresses()) > 1 {
			continue
		}

		utxos = append(utxos, utxo)
	}

	return utxos, nil
}

func (b *Backend) getAccountUTXOs(ctx context.Context, addr ids.ShortID, sourceChain string) ([][]byte, error) {
	utxos := [][]byte{}

	// Used for pagination
	var startAddr ids.ShortID
	var startUTXOID ids.ID
	for {
		var utxoPage [][]byte
		var err error

		// GetUTXOs controlled by addr
		utxoPage, startAddr, startUTXOID, err = b.pClient.GetAtomicUTXOs(
			ctx,
			[]ids.ShortID{addr},
			sourceChain,
			b.getUTXOsPageSize,
			startAddr,
			startUTXOID,
		)
		if err != nil {
			return nil, errUnableToGetUTXOs
		}

		utxos = append(utxos, utxoPage...)

		// Fetch next page only if there may be more UTXOs
		if len(utxoPage) < int(b.getUTXOsPageSize) {
			break
		}
	}

	return utxos, nil
}

func (b *Backend) processUtxos(currencyAssetIDs map[ids.ID]struct{}, utxos []avax.UTXO) ([]*types.Coin, error) {
	coins := []*types.Coin{}

	for _, utxo := range utxos {
		// Skip UTXO if req.Currencies is specified but it doesn't contain the UTXOs asset
		if _, ok := currencyAssetIDs[utxo.AssetID()]; len(currencyAssetIDs) > 0 && !ok {
			continue
		}

		amounter, ok := utxo.Out.(avax.Amounter)
		if !ok {
			return nil, errUnableToGetUTXOOut
		}

		coin := &types.Coin{
			CoinIdentifier: &types.CoinIdentifier{Identifier: utxo.UTXOID.String()},
			Amount: &types.Amount{
				Value:    strconv.FormatUint(amounter.Amount(), 10),
				Currency: mapper.AtomicAvaxCurrency,
			},
		}
		coins = append(coins, coin)
	}
	return coins, nil
}
