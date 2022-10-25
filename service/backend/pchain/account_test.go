package pchain

import (
	"context"
	"testing"

	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/avalanchego/utils/formatting/address"
	"github.com/ava-labs/avalanchego/vms/avm"
	"github.com/ava-labs/avalanchego/vms/components/avax"
	"github.com/ava-labs/avalanchego/vms/secp256k1fx"
	"github.com/coinbase/rosetta-sdk-go/types"
	"github.com/stretchr/testify/assert"

	"github.com/ava-labs/avalanche-rosetta/mapper"
	pmapper "github.com/ava-labs/avalanche-rosetta/mapper/pchain"
	mocks "github.com/ava-labs/avalanche-rosetta/mocks/client"
	idxmocks "github.com/ava-labs/avalanche-rosetta/mocks/service/backend/pchain/indexer"
	"github.com/ava-labs/avalanche-rosetta/service"
	"github.com/ava-labs/avalanche-rosetta/service/backend/pchain/indexer"
)

type utxo struct {
	id     string
	amount uint64
}

var (
	utxos = []utxo{
		{"NGcWaGCzBUtUsD85wDuX1DwbHFkvMHwJ9tDFiN7HCCnVcB9B8:0", 1000000000},
		{"pyQfA1Aq9vLaDETjeQe5DAwVxr2KAYdHg4CHzawmaj9oA6ppn:0", 2000000000},
	}
	blockID, _  = ids.FromString("mq1enPCRAwWyRjFNY8nSmkLde6U5huUcp9PXueF2h7Kjb2csd")
	blockHeight = uint64(42)
	parsedBlock = &indexer.ParsedBlock{BlockID: blockID}

	pChainAddr = "P-avax1yp8v6x7kf7ar2q5g0cs0a9jk4cmt0sgam72zfz"

	dummyGenesis = &indexer.ParsedGenesisBlock{}
)

func TestAccountBalance(t *testing.T) {
	ctx := context.Background()
	pChainMock := &mocks.PChainClient{}
	pChainMock.Mock.On("GetBlockchainID", ctx, mapper.CChainNetworkIdentifier).Return(ids.ID{'C'}, nil)
	pChainMock.Mock.On("GetBlockchainID", ctx, mapper.XChainNetworkIdentifier).Return(ids.ID{'X'}, nil)
	parserMock := &idxmocks.Parser{}
	parserMock.Mock.On("GetGenesisBlock", ctx).Return(dummyGenesis, nil)
	parserMock.Mock.On("ParseNonGenesisBlock", ctx, "", blockHeight).Return(parsedBlock, nil)
	backend, err := NewBackend(service.ModeOnline, pChainMock, parserMock, avaxAssetID, pChainNetworkIdentifier)
	assert.Nil(t, err)
	backend.getUTXOsPageSize = 2

	t.Run("Account Balance Test", func(t *testing.T) {
		addr, _ := address.ParseToID(pChainAddr)
		utxo0Bytes := makeUtxoBytes(t, backend, utxos[0].id, utxos[0].amount)
		utxo1Bytes := makeUtxoBytes(t, backend, utxos[1].id, utxos[1].amount)
		utxo1Id, _ := ids.FromString(utxos[1].id)
		stakeUtxoBytes := makeStakeUtxoBytes(t, backend, utxos[1].amount)

		// Mock on GetAssetDescription
		mockAssetDescription := &avm.GetAssetDescriptionReply{
			Name:         "Avalanche",
			Symbol:       mapper.AtomicAvaxCurrency.Symbol,
			Denomination: 9,
		}
		pChainMock.Mock.On("GetAssetDescription", ctx, mapper.AtomicAvaxCurrency.Symbol).Return(mockAssetDescription, nil)

		// once before other calls, once after
		pChainMock.Mock.On("GetHeight", ctx).Return(blockHeight, nil).Twice()
		// Make sure pagination works as well
		pageSize := uint32(2)
		backend.getUTXOsPageSize = pageSize
		pChainMock.Mock.On("GetAtomicUTXOs", ctx, []ids.ShortID{addr}, "", pageSize, ids.ShortEmpty, ids.Empty).
			Return([][]byte{utxo0Bytes, utxo1Bytes}, addr, utxo1Id, nil).Once()
		pChainMock.Mock.On("GetAtomicUTXOs", ctx, []ids.ShortID{addr}, "", pageSize, addr, utxo1Id).
			Return([][]byte{utxo1Bytes}, addr, utxo1Id, nil).Once()
		pChainMock.Mock.On("GetStake", ctx, []ids.ShortID{addr}).Return(map[ids.ID]uint64{}, [][]byte{stakeUtxoBytes}, nil).Once()

		resp, err := backend.AccountBalance(
			ctx,
			&types.AccountBalanceRequest{
				NetworkIdentifier: &types.NetworkIdentifier{
					Network: mapper.FujiNetwork,
					SubNetworkIdentifier: &types.SubNetworkIdentifier{
						Network: mapper.PChainNetworkIdentifier,
					},
				},
				AccountIdentifier: &types.AccountIdentifier{
					Address: pChainAddr,
				},
				Currencies: []*types.Currency{
					mapper.AtomicAvaxCurrency,
				},
			},
		)

		expected := &types.AccountBalanceResponse{
			Balances: []*types.Amount{
				{
					Value:    "5000000000", // 1B + 2B from UTXOs, 1B from staked
					Currency: mapper.AtomicAvaxCurrency,
				},
			},
		}

		assert.Nil(t, err)
		assert.Equal(t, expected.Balances, resp.Balances)
		pChainMock.AssertExpectations(t)
		parserMock.AssertExpectations(t)
	})

	t.Run("Account Balance should return total of shared memory balance", func(t *testing.T) {
		// Mock on GetUTXOs
		utxo0Bytes := makeUtxoBytes(t, backend, utxos[0].id, utxos[0].amount)
		utxo1Bytes := makeUtxoBytes(t, backend, utxos[1].id, utxos[1].amount)
		utxo1Id, _ := ids.FromString(utxos[1].id)
		pChainAddrID, errp := address.ParseToID(pChainAddr)
		assert.Nil(t, errp)

		// once before other calls, once after
		pChainMock.Mock.On("GetHeight", ctx).Return(blockHeight, nil).Twice()
		pageSize := uint32(1024)
		backend.getUTXOsPageSize = pageSize
		pChainMock.Mock.On("GetAtomicUTXOs", ctx, []ids.ShortID{pChainAddrID}, "C", pageSize, ids.ShortEmpty, ids.Empty).
			Return([][]byte{utxo0Bytes, utxo1Bytes}, pChainAddrID, utxo1Id, nil).Once()
		pChainMock.Mock.On("GetAtomicUTXOs", ctx, []ids.ShortID{pChainAddrID}, "X", pageSize, ids.ShortEmpty, ids.Empty).
			Return([][]byte{}, pChainAddrID, ids.Empty, nil).Once()

		resp, err := backend.AccountBalance(
			ctx,
			&types.AccountBalanceRequest{
				NetworkIdentifier: &types.NetworkIdentifier{
					Network: mapper.FujiNetwork,
					SubNetworkIdentifier: &types.SubNetworkIdentifier{
						Network: mapper.PChainNetworkIdentifier,
					},
				},
				AccountIdentifier: &types.AccountIdentifier{
					Address:    pChainAddr,
					SubAccount: &types.SubAccountIdentifier{Address: pmapper.SubAccountTypeSharedMemory},
				},
				Currencies: []*types.Currency{
					mapper.AtomicAvaxCurrency,
				},
			})

		expected := &types.AccountBalanceResponse{
			BlockIdentifier: &types.BlockIdentifier{
				Index: int64(blockHeight),
				Hash:  parsedBlock.BlockID.String(),
			},
			Balances: []*types.Amount{{
				Value:    "3000000000",
				Currency: mapper.AtomicAvaxCurrency,
			}},
		}

		assert.Nil(t, err)
		assert.Equal(t, expected, resp)
		pChainMock.AssertExpectations(t)
		parserMock.AssertExpectations(t)
	})

	t.Run("Account Balance should error if new block was added while fetching UTXOs", func(t *testing.T) {
		addr, _ := address.ParseToID(pChainAddr)

		pageSize := uint32(2)
		backend.getUTXOsPageSize = pageSize
		pChainMock.Mock.On("GetHeight", ctx).Return(blockHeight, nil).Once()
		pChainMock.Mock.On("GetAtomicUTXOs", ctx, []ids.ShortID{addr}, "", pageSize, ids.ShortEmpty, ids.Empty).
			Return([][]byte{}, addr, ids.Empty, nil).Once()
		pChainMock.Mock.On("GetStake", ctx, []ids.ShortID{addr}).Return(map[ids.ID]uint64{}, [][]byte{}, nil)
		// return blockHeight + 1 to indicate a new block arrival
		pChainMock.Mock.On("GetHeight", ctx).Return(blockHeight+1, nil).Once()

		resp, err := backend.AccountBalance(
			ctx,
			&types.AccountBalanceRequest{
				NetworkIdentifier: &types.NetworkIdentifier{
					Network: mapper.FujiNetwork,
					SubNetworkIdentifier: &types.SubNetworkIdentifier{
						Network: mapper.PChainNetworkIdentifier,
					},
				},
				AccountIdentifier: &types.AccountIdentifier{
					Address: pChainAddr,
				},
				Currencies: []*types.Currency{
					mapper.AtomicAvaxCurrency,
				},
			},
		)

		assert.Nil(t, resp)
		assert.Equal(t, "Internal server error", err.Message)
		assert.Equal(t, "new block added while fetching utxos", err.Details["error"])
		pChainMock.AssertExpectations(t)
		parserMock.AssertExpectations(t)
	})
}

func TestAccountCoins(t *testing.T) {
	ctx := context.Background()
	pChainMock := &mocks.PChainClient{}
	pChainMock.Mock.On("GetBlockchainID", ctx, mapper.CChainNetworkIdentifier).Return(ids.ID{'C'}, nil)
	pChainMock.Mock.On("GetBlockchainID", ctx, mapper.XChainNetworkIdentifier).Return(ids.ID{'X'}, nil)
	parserMock := &idxmocks.Parser{}
	parserMock.Mock.On("GetGenesisBlock", ctx).Return(dummyGenesis, nil)
	parserMock.Mock.On("ParseNonGenesisBlock", ctx, "", blockHeight).Return(parsedBlock, nil)
	backend, err := NewBackend(service.ModeOnline, pChainMock, parserMock, avaxAssetID, pChainNetworkIdentifier)
	assert.Nil(t, err)

	t.Run("Account Coins Test regular coins", func(t *testing.T) {
		// Mock on GetAssetDescription
		mockAssetDescription := &avm.GetAssetDescriptionReply{
			Name:         "Avalanche",
			Symbol:       mapper.AtomicAvaxCurrency.Symbol,
			Denomination: 9,
		}
		pChainMock.Mock.On("GetAssetDescription", ctx, mapper.AtomicAvaxCurrency.Symbol).Return(mockAssetDescription, nil)

		// Mock on GetUTXOs
		utxo0Bytes := makeUtxoBytes(t, backend, utxos[0].id, utxos[0].amount)
		utxo1Bytes := makeUtxoBytes(t, backend, utxos[1].id, utxos[1].amount)
		utxo1Id, _ := ids.FromString(utxos[1].id)
		pChainAddrID, errp := address.ParseToID(pChainAddr)
		assert.Nil(t, errp)

		// once before other calls, once after
		pChainMock.Mock.On("GetHeight", ctx).Return(blockHeight, nil).Twice()
		// Make sure pagination works as well
		pageSize := uint32(2)
		backend.getUTXOsPageSize = pageSize
		pChainMock.Mock.On("GetAtomicUTXOs", ctx, []ids.ShortID{pChainAddrID}, "", pageSize, ids.ShortEmpty, ids.Empty).
			Return([][]byte{utxo0Bytes, utxo1Bytes}, pChainAddrID, utxo1Id, nil).Once()
		pChainMock.Mock.On("GetAtomicUTXOs", ctx, []ids.ShortID{pChainAddrID}, "", pageSize, pChainAddrID, utxo1Id).
			Return([][]byte{utxo1Bytes}, pChainAddrID, utxo1Id, nil).Once()

		resp, err := backend.AccountCoins(
			ctx,
			&types.AccountCoinsRequest{
				NetworkIdentifier: &types.NetworkIdentifier{
					Network: mapper.FujiNetwork,
					SubNetworkIdentifier: &types.SubNetworkIdentifier{
						Network: mapper.PChainNetworkIdentifier,
					},
				},
				AccountIdentifier: &types.AccountIdentifier{
					Address: pChainAddr,
				},
				Currencies: []*types.Currency{
					mapper.AtomicAvaxCurrency,
				},
			})

		expected := &types.AccountCoinsResponse{
			BlockIdentifier: &types.BlockIdentifier{
				Index: int64(blockHeight),
				Hash:  parsedBlock.BlockID.String(),
			},
			Coins: []*types.Coin{
				{
					CoinIdentifier: &types.CoinIdentifier{
						Identifier: "NGcWaGCzBUtUsD85wDuX1DwbHFkvMHwJ9tDFiN7HCCnVcB9B8:0",
					},
					Amount: &types.Amount{
						Value:    "1000000000",
						Currency: mapper.AtomicAvaxCurrency,
					},
				},
				{
					CoinIdentifier: &types.CoinIdentifier{
						Identifier: "pyQfA1Aq9vLaDETjeQe5DAwVxr2KAYdHg4CHzawmaj9oA6ppn:0",
					},
					Amount: &types.Amount{
						Value:    "2000000000",
						Currency: mapper.AtomicAvaxCurrency,
					},
				},
			},
		}

		assert.Nil(t, err)
		assert.Equal(t, expected, resp)
		pChainMock.AssertExpectations(t)
		parserMock.AssertExpectations(t)
	})

	t.Run("Account Coins Test shared memory coins", func(t *testing.T) {
		// Mock on GetAssetDescription
		mockAssetDescription := &avm.GetAssetDescriptionReply{
			Name:         "Avalanche",
			Symbol:       mapper.AtomicAvaxCurrency.Symbol,
			Denomination: 9,
		}
		pChainMock.Mock.On("GetAssetDescription", ctx, mapper.AtomicAvaxCurrency.Symbol).Return(mockAssetDescription, nil)

		// Mock on GetUTXOs
		utxo0Bytes := makeUtxoBytes(t, backend, utxos[0].id, utxos[0].amount)
		utxo0Id, _ := ids.FromString(utxos[0].id)
		utxo1Bytes := makeUtxoBytes(t, backend, utxos[1].id, utxos[1].amount)
		utxo1Id, _ := ids.FromString(utxos[1].id)
		pChainAddrID, errp := address.ParseToID(pChainAddr)
		assert.Nil(t, errp)

		// once before other calls, once after
		pChainMock.Mock.On("GetHeight", ctx).Return(blockHeight, nil).Twice()
		pageSize := uint32(1024)
		backend.getUTXOsPageSize = pageSize
		pChainMock.Mock.On("GetAtomicUTXOs", ctx, []ids.ShortID{pChainAddrID}, "C", pageSize, ids.ShortEmpty, ids.Empty).
			Return([][]byte{utxo0Bytes}, pChainAddrID, utxo0Id, nil).Once()
		pChainMock.Mock.On("GetAtomicUTXOs", ctx, []ids.ShortID{pChainAddrID}, "X", pageSize, ids.ShortEmpty, ids.Empty).
			Return([][]byte{utxo1Bytes}, pChainAddrID, utxo1Id, nil).Once()

		resp, err := backend.AccountCoins(
			ctx,
			&types.AccountCoinsRequest{
				NetworkIdentifier: &types.NetworkIdentifier{
					Network: mapper.FujiNetwork,
					SubNetworkIdentifier: &types.SubNetworkIdentifier{
						Network: mapper.PChainNetworkIdentifier,
					},
				},
				AccountIdentifier: &types.AccountIdentifier{
					Address:    pChainAddr,
					SubAccount: &types.SubAccountIdentifier{Address: pmapper.SubAccountTypeSharedMemory},
				},
				Currencies: []*types.Currency{
					mapper.AtomicAvaxCurrency,
				},
			})

		expected := &types.AccountCoinsResponse{
			BlockIdentifier: &types.BlockIdentifier{
				Index: int64(blockHeight),
				Hash:  parsedBlock.BlockID.String(),
			},
			Coins: []*types.Coin{
				{
					CoinIdentifier: &types.CoinIdentifier{
						Identifier: "NGcWaGCzBUtUsD85wDuX1DwbHFkvMHwJ9tDFiN7HCCnVcB9B8:0",
					},
					Amount: &types.Amount{
						Value:    "1000000000",
						Currency: mapper.AtomicAvaxCurrency,
					},
				},
				{
					CoinIdentifier: &types.CoinIdentifier{
						Identifier: "pyQfA1Aq9vLaDETjeQe5DAwVxr2KAYdHg4CHzawmaj9oA6ppn:0",
					},
					Amount: &types.Amount{
						Value:    "2000000000",
						Currency: mapper.AtomicAvaxCurrency,
					},
				},
			},
		}

		assert.Nil(t, err)
		assert.Equal(t, expected, resp)
		pChainMock.AssertExpectations(t)
		parserMock.AssertExpectations(t)
	})
}

func makeUtxoBytes(t *testing.T, backend *Backend, utxoIDStr string, amount uint64) []byte {
	utxoID, err := mapper.DecodeUTXOID(utxoIDStr)
	if err != nil {
		t.Fail()
		return nil
	}

	utxoBytes, err := backend.codec.Marshal(0, &avax.UTXO{
		UTXOID: *utxoID,
		Out:    &secp256k1fx.TransferOutput{Amt: amount},
	})
	if err != nil {
		t.Fail()
	}

	return utxoBytes
}

func makeStakeUtxoBytes(t *testing.T, backend *Backend, amount uint64) []byte {
	utxoBytes, err := backend.codec.Marshal(0, &avax.TransferableOutput{
		Out: &secp256k1fx.TransferOutput{Amt: amount},
	})
	if err != nil {
		t.Fail()
	}

	return utxoBytes
}
