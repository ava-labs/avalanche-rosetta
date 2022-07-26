package common

import (
	"sort"

	"github.com/coinbase/rosetta-sdk-go/types"
)

// SortUnique deduplicates given slice of coins and sorts them by UTXO id in ascending order for consistency
//
// Per https://docs.avax.network/apis/avalanchego/apis/p-chain#platformgetutxos
// and https://docs.avax.network/apis/avalanchego/apis/c-chain#avaxgetutxos
// paginated getUTXOs calls may have duplicate UTXOs in different pages, this helper eliminates them.
func SortUnique(coins []*types.Coin) []*types.Coin {
	coinsMap := make(map[string]*types.Coin)
	for i, coin := range coins {
		coinsMap[coin.CoinIdentifier.Identifier] = coins[i]
	}

	uniqueCoinIdentifiers := make([]string, 0, len(coinsMap))
	for identifier := range coinsMap {
		uniqueCoinIdentifiers = append(uniqueCoinIdentifiers, identifier)
	}
	sort.Strings(uniqueCoinIdentifiers)

	uniqueCoins := make([]*types.Coin, 0, len(coinsMap))
	for _, identifier := range uniqueCoinIdentifiers {
		uniqueCoins = append(uniqueCoins, coinsMap[identifier])
	}

	return uniqueCoins
}
