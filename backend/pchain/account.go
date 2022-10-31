package pchain

import (
	"context"
	"errors"
	"strconv"
	"time"

	"github.com/ava-labs/avalanchego/vms/platformvm/stakeable"
	"github.com/ava-labs/avalanchego/vms/secp256k1fx"

	"github.com/ava-labs/avalanche-rosetta/backend"
	"github.com/ava-labs/avalanche-rosetta/backend/common"
	"github.com/ava-labs/avalanche-rosetta/constants"
	pmapper "github.com/ava-labs/avalanche-rosetta/mapper/pchain"

	pconstants "github.com/ava-labs/avalanche-rosetta/constants/pchain"
	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/avalanchego/utils/formatting/address"
	"github.com/ava-labs/avalanchego/utils/math"
	"github.com/ava-labs/avalanchego/vms/components/avax"
	"github.com/coinbase/rosetta-sdk-go/types"
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

// AccountBalance implements /account/balance endpoint for P-chain
func (b *Backend) AccountBalance(ctx context.Context, req *types.AccountBalanceRequest) (*types.AccountBalanceResponse, *types.Error) {
	if req.AccountIdentifier == nil {
		return nil, backend.WrapError(backend.ErrInvalidInput, "account identifier is not provided")
	}
	if req.BlockIdentifier != nil {
		return nil, backend.WrapError(backend.ErrNotSupported, "historical balance lookups are not supported")
	}

	currencyAssetIDs, wrappedErr := b.buildCurrencyAssetIDs(ctx, req.Currencies)
	if wrappedErr != nil {
		return nil, wrappedErr
	}

	var balanceType string
	if req.AccountIdentifier.SubAccount != nil {
		balanceType = req.AccountIdentifier.SubAccount.Address
	}
	fetchImportable := balanceType == pmapper.SubAccountTypeSharedMemory

	height, balance, typedErr := b.fetchBalance(ctx, req.AccountIdentifier.Address, fetchImportable, currencyAssetIDs)
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
		return nil, backend.WrapError(backend.ErrInvalidInput, "unknown account type "+balanceType)
	}

	block, err := b.indexerParser.ParseNonGenesisBlock(ctx, "", height)
	if err != nil {
		return nil, backend.WrapError(backend.ErrInvalidInput, "unable to get height")
	}

	return &types.AccountBalanceResponse{
		BlockIdentifier: &types.BlockIdentifier{
			Index: int64(height),
			Hash:  block.BlockID.String(),
		},
		Balances: []*types.Amount{
			{
				Value:    strconv.FormatUint(balanceValue, 10),
				Currency: pconstants.AtomicAvaxCurrency,
			},
		},
	}, nil
}

// AccountCoins implements /account/coins endpoint for P-chain
func (b *Backend) AccountCoins(ctx context.Context, req *types.AccountCoinsRequest) (*types.AccountCoinsResponse, *types.Error) {
	if req.AccountIdentifier == nil {
		return nil, backend.WrapError(backend.ErrInvalidInput, "account identifier is not provided")
	}
	addr, err := address.ParseToID(req.AccountIdentifier.Address)
	if err != nil {
		return nil, backend.WrapError(backend.ErrInvalidInput, "unable to convert address")
	}

	assetIDs, wrappedErr := b.buildCurrencyAssetIDs(ctx, req.Currencies)
	if err != nil {
		return nil, wrappedErr
	}

	var subAccountAddress string
	if req.AccountIdentifier.SubAccount != nil {
		subAccountAddress = req.AccountIdentifier.SubAccount.Address
	}
	fetchSharedMemory := subAccountAddress == pmapper.SubAccountTypeSharedMemory

	// utxos from fetchUTXOsAndStakedOutputs are guarateed to:
	// 1. be unique (no duplicates)
	// 2. containt only assetIDs
	// 3. have not multisign utxos
	// by parseAndFilterUTXOs call in fetchUTXOsAndStakedOutputs
	height, utxos, _, typedErr := b.fetchUTXOsAndStakedOutputs(ctx, addr, false, fetchSharedMemory, assetIDs)
	if typedErr != nil {
		return nil, typedErr
	}

	// convert UTXOs to Rosetta Coins
	coins := []*types.Coin{}
	for _, utxo := range utxos {
		amounter, ok := utxo.Out.(avax.Amounter)
		if !ok {
			return nil, backend.WrapError(backend.ErrInternalError, errUnableToGetUTXOOut)
		}
		coin := &types.Coin{
			CoinIdentifier: &types.CoinIdentifier{Identifier: utxo.UTXOID.String()},
			Amount: &types.Amount{
				Value:    strconv.FormatUint(amounter.Amount(), 10),
				Currency: pconstants.AtomicAvaxCurrency,
			},
		}
		coins = append(coins, coin)
	}

	block, err := b.indexerParser.ParseNonGenesisBlock(ctx, "", height)
	if err != nil {
		return nil, backend.WrapError(backend.ErrInvalidInput, "unable to get height")
	}

	// this is needed just for sorting. Uniqueness is guaranteed by utxos uniqueness
	coins = common.SortUnique(coins)
	return &types.AccountCoinsResponse{
		BlockIdentifier: &types.BlockIdentifier{
			Index: int64(height),
			Hash:  block.BlockID.String(),
		},
		Coins: coins,
	}, nil
}

func (b *Backend) fetchBalance(ctx context.Context, addrString string, fetchImportable bool, assetIds ids.Set) (uint64, *AccountBalance, *types.Error) {
	addr, err := address.ParseToID(addrString)
	if err != nil {
		return 0, nil, backend.WrapError(backend.ErrInvalidInput, "unable to convert address")
	}

	// utxos from fetchUTXOsAndStakedOutputs are guarateed to:
	// 1. be unique (no duplicates)
	// 2. containt only assetIDs
	// 3. have not multisign utxos
	// by parseAndFilterUTXOs call in fetchUTXOsAndStakedOutputs
	height, utxos, stakedUTXOBytes, typedErr := b.fetchUTXOsAndStakedOutputs(ctx, addr, !fetchImportable, fetchImportable, assetIds)
	if typedErr != nil {
		return 0, nil, typedErr
	}

	balance, err := b.getBalancesWithoutMultisig(utxos)
	if err != nil {
		return 0, nil, backend.WrapError(backend.ErrInternalError, err)
	}

	// parse staked UTXO bytes to UTXO structs
	stakedAmount, err := b.calculateStakedAmount(stakedUTXOBytes)
	if err != nil {
		return 0, nil, backend.WrapError(backend.ErrInternalError, err)
	}

	balance.Staked = stakedAmount
	balance.Total += stakedAmount

	return height, balance, nil
}

// Copy of the platformvm service's GetBalance implementation.
// This is needed as multisig UTXOs are cleaned in parseUTXOs and its output must be used for the calculations. Ref:
// https://github.com/ava-labs/avalanchego/blob/0950acab667e0c16a55e9a9bb72bcbe25c3b88cf/vms/platformvm/backend.go#L184
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

func (b *Backend) buildCurrencyAssetIDs(ctx context.Context, currencies []*types.Currency) (ids.Set, *types.Error) {
	assetIDs := ids.NewSet(len(currencies))
	for _, reqCurrency := range currencies {
		description, err := b.pClient.GetAssetDescription(ctx, reqCurrency.Symbol)
		if err != nil {
			return nil, backend.WrapError(backend.ErrInternalError, "unable to get asset description")
		}
		if int32(description.Denomination) != reqCurrency.Decimals {
			return nil, backend.WrapError(backend.ErrInvalidInput, "incorrect currency decimals")
		}
		assetIDs.Add(description.AssetID)
	}

	return assetIDs, nil
}

// Fetches UTXOs and staked outputs for the given account.
//
// Since these APIs don't return the corresponding block height or hash,
// which is needed for both /account/balance and /account/coins, chain height is checked before and after
// and if they differ, an error is returned.
func (b *Backend) fetchUTXOsAndStakedOutputs(ctx context.Context, addr ids.ShortID, fetchStaked bool, fetchSharedMemory bool, assetIds ids.Set) (uint64, []avax.UTXO, [][]byte, *types.Error) {
	// fetch preHeight before the balance fetch
	preHeight, err := b.pClient.GetHeight(ctx)
	if err != nil {
		return 0, nil, nil, backend.WrapError(backend.ErrInvalidInput, "unable to get chain height pre-lookup")
	}

	sourceChains := []constants.ChainIDAlias{constants.AnyChain}
	if fetchSharedMemory {
		sourceChains = []constants.ChainIDAlias{
			constants.CChain,
			constants.XChain,
		}
	}

	var utxoBytes [][]byte

	for _, sc := range sourceChains {
		// fetch all UTXOs for addr
		chainUtxoBytes, err := b.getAccountUTXOs(ctx, addr, sc)
		if err != nil {
			return 0, nil, nil, backend.WrapError(backend.ErrInternalError, err)
		}
		utxoBytes = append(utxoBytes, chainUtxoBytes...)
	}

	if err != nil {
		return 0, nil, nil, backend.WrapError(backend.ErrInternalError, err)
	}

	var stakedUTXOBytes [][]byte
	if fetchStaked {
		// fetch staked outputs for addr
		_, stakedUTXOBytes, err = b.pClient.GetStake(ctx, []ids.ShortID{addr})
		if err != nil {
			return 0, nil, nil, backend.WrapError(backend.ErrInvalidInput, "unable to get stake")
		}
	}

	// fetch postHeight after the balance fetch and compare with preHeight
	postHeight, err := b.pClient.GetHeight(ctx)
	if err != nil {
		return 0, nil, nil, backend.WrapError(backend.ErrInvalidInput, "unable to get chain height post-lookup")
	}
	if postHeight != preHeight {
		return 0, nil, nil, backend.WrapError(backend.ErrInternalError, "new block added while fetching utxos")
	}

	// parse UTXO bytes to UTXO structs
	utxos, err := b.parseAndFilterUTXOs(utxoBytes, assetIds)
	if err != nil {
		return 0, nil, nil, backend.WrapError(backend.ErrInternalError, err)
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

func (b *Backend) parseAndFilterUTXOs(utxoBytes [][]byte, assetIDs ids.Set) ([]avax.UTXO, error) {
	utxos := []avax.UTXO{}

	// when results are paginated, duplicate UTXOs may be provided. guarantee uniqueness
	utxoIDs := ids.NewSet(len(utxoBytes))
	for _, bytes := range utxoBytes {
		utxo := avax.UTXO{}
		_, err := b.codec.Unmarshal(bytes, &utxo)
		if err != nil {
			return nil, errUnableToParseUTXO
		}

		// Skip UTXO if req.Currencies is specified, but it doesn't contain the UTXOs asset
		if assetIDs.Len() > 0 && !assetIDs.Contains(utxo.AssetID()) {
			continue
		}

		// remove duplicates
		if utxoIDs.Contains(utxo.UTXOID.InputID()) {
			continue
		}
		utxoIDs.Add(utxo.UTXOID.InputID())

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

func (b *Backend) getAccountUTXOs(ctx context.Context, addr ids.ShortID, sourceChain constants.ChainIDAlias) ([][]byte, error) {
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
			sourceChain.String(),
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
