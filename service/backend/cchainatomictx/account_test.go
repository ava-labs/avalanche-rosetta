package cchainatomictx

import (
	"context"
	"strconv"
	"testing"

	"github.com/ava-labs/avalanchego/api"
	"github.com/ava-labs/avalanchego/vms/components/avax"
	"github.com/ava-labs/avalanchego/vms/secp256k1fx"
	"github.com/coinbase/rosetta-sdk-go/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/ava-labs/avalanche-rosetta/mapper"
	mocks "github.com/ava-labs/avalanche-rosetta/mocks/client"
)

type utxo struct {
	id     string
	amount uint64
}

var utxos = []utxo{
	{"23CLURk1Czf1aLui1VdcuWSiDeFskfp3Sn8TQG7t6NKfeQRYDj:2", 1_000_000},
	{"2QmMXKS6rKQMnEh2XYZ4ZWCJmy8RpD3LyVZWxBG25t4N1JJqxY:1", 1_500_000},
	{"2QmMXKS6rKQMnEh2XYZ4ZWCJmy8RpD3LyVZWxBG25t4N1JJqxY:1", 1_500_000}, // duplicate
	{"23CLURk1Czf1aLui1VdcuWSiDeFskfp3Sn8TQG7t6NKfeQRYDj:4", 2_000_000}, // out of order

}

func TestAccountBalance(t *testing.T) {
	evmMock := &mocks.Client{}
	backend := NewBackend(evmMock)
	accountAddress := "C-fuji15f9g0h5xkr5cp47n6u3qxj6yjtzzzrdr23a3tl"

	t.Run("C-chain atomic tx balance is sum of UTXOs", func(t *testing.T) {
		utxo0Bytes := makeUtxoBytes(t, backend, utxos[0].id, utxos[0].amount)
		utxo1Bytes := makeUtxoBytes(t, backend, utxos[1].id, utxos[1].amount)

		utxos := [][]byte{utxo0Bytes, utxo1Bytes}

		evmMock.
			On("GetAtomicUTXOs", mock.Anything, []string{accountAddress}, "P", backend.getUTXOsPageSize, "", "").
			Return(utxos, api.Index{}, nil)
		evmMock.
			On("GetAtomicUTXOs", mock.Anything, []string{accountAddress}, "X", backend.getUTXOsPageSize, "", "").
			Return([][]byte{}, api.Index{}, nil)

		resp, apiErr := backend.AccountBalance(context.Background(), &types.AccountBalanceRequest{
			NetworkIdentifier: &types.NetworkIdentifier{},
			AccountIdentifier: &types.AccountIdentifier{
				Address: accountAddress,
			},
		})
		assert.Nil(t, apiErr)

		evmMock.AssertExpectations(t)

		assert.Equal(t, 1, len(resp.Balances))
		assert.Equal(t, mapper.AvaxCurrency, resp.Balances[0].Currency)
		assert.Equal(t, "2500000", resp.Balances[0].Value)
	})
}

func TestAccountCoins(t *testing.T) {
	evmMock := &mocks.Client{}
	backend := NewBackend(evmMock)
	// changing page size to 2 to test pagination as well
	backend.getUTXOsPageSize = 2
	accountAddress := "C-fuji15f9g0h5xkr5cp47n6u3qxj6yjtzzzrdr23a3tl"

	t.Run("C-chain atomic tx coins returns UTXOs", func(t *testing.T) {
		utxo0Bytes := makeUtxoBytes(t, backend, utxos[0].id, utxos[0].amount)
		utxo1Bytes := makeUtxoBytes(t, backend, utxos[1].id, utxos[1].amount)
		utxo2Bytes := makeUtxoBytes(t, backend, utxos[2].id, utxos[2].amount)
		utxo3Bytes := makeUtxoBytes(t, backend, utxos[3].id, utxos[3].amount)

		evmMock.
			On("GetAtomicUTXOs", mock.Anything, []string{accountAddress}, "P", backend.getUTXOsPageSize, "", "").
			Return([][]byte{utxo0Bytes, utxo1Bytes}, api.Index{Address: accountAddress, UTXO: utxos[1].id}, nil)
		evmMock.
			On("GetAtomicUTXOs", mock.Anything, []string{accountAddress}, "P", backend.getUTXOsPageSize, accountAddress, utxos[1].id).
			Return([][]byte{utxo2Bytes, utxo3Bytes}, api.Index{Address: accountAddress, UTXO: utxos[3].id}, nil)
		evmMock.
			On("GetAtomicUTXOs", mock.Anything, []string{accountAddress}, "P", backend.getUTXOsPageSize, accountAddress, utxos[3].id).
			Return([][]byte{utxo3Bytes}, api.Index{Address: accountAddress, UTXO: utxos[3].id}, nil)
		evmMock.
			On("GetAtomicUTXOs", mock.Anything, []string{accountAddress}, "X", backend.getUTXOsPageSize, "", "").
			Return([][]byte{}, api.Index{}, nil)

		resp, apiErr := backend.AccountCoins(context.Background(), &types.AccountCoinsRequest{
			NetworkIdentifier: &types.NetworkIdentifier{},
			AccountIdentifier: &types.AccountIdentifier{
				Address: accountAddress,
			},
		})
		assert.Nil(t, apiErr)

		evmMock.AssertExpectations(t)

		assert.Equal(t, 3, len(resp.Coins))

		assert.Equal(t, utxos[0].id, resp.Coins[0].CoinIdentifier.Identifier)
		assert.Equal(t, mapper.AvaxCurrency, resp.Coins[0].Amount.Currency)
		assert.Equal(t, strconv.FormatUint(utxos[0].amount, 10), resp.Coins[0].Amount.Value)

		assert.Equal(t, utxos[3].id, resp.Coins[1].CoinIdentifier.Identifier)
		assert.Equal(t, mapper.AvaxCurrency, resp.Coins[1].Amount.Currency)
		assert.Equal(t, strconv.FormatUint(utxos[3].amount, 10), resp.Coins[1].Amount.Value)

		assert.Equal(t, utxos[1].id, resp.Coins[2].CoinIdentifier.Identifier)
		assert.Equal(t, mapper.AvaxCurrency, resp.Coins[2].Amount.Currency)
		assert.Equal(t, strconv.FormatUint(utxos[1].amount, 10), resp.Coins[2].Amount.Value)
	})
}

func makeUtxoBytes(t *testing.T, backend *Backend, utxoIDStr string, amount uint64) []byte {
	utxoID, err := mapper.DecodeUTXOID(utxoIDStr)
	if err != nil {
		t.Fail()
		return nil
	}

	utxoBytes, err := backend.codec.Marshal(backend.codecVersion, &avax.UTXO{
		UTXOID: *utxoID,
		Out:    &secp256k1fx.TransferOutput{Amt: amount},
	})
	if err != nil {
		t.Fail()
	}

	return utxoBytes
}
