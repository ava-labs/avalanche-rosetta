package pchain

import (
	"context"
	"testing"

	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/avalanchego/utils/formatting/address"
	"github.com/ava-labs/avalanchego/vms/avm"
	"github.com/ava-labs/avalanchego/vms/components/avax"
	"github.com/ava-labs/avalanchego/vms/platformvm"
	"github.com/ava-labs/avalanchego/vms/secp256k1fx"
	"github.com/coinbase/rosetta-sdk-go/types"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/ava-labs/avalanche-rosetta/client"
	"github.com/ava-labs/avalanche-rosetta/constants"
	"github.com/ava-labs/avalanche-rosetta/mapper"
	"github.com/ava-labs/avalanche-rosetta/service/backend/pchain/indexer"

	pmapper "github.com/ava-labs/avalanche-rosetta/mapper/pchain"
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

	mockAssetDescription = &avm.GetAssetDescriptionReply{
		Name:         "Avalanche",
		Symbol:       mapper.AtomicAvaxCurrency.Symbol,
		Denomination: 9,
	}
)

func TestAccountBalance(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	pChainMock := client.NewMockPChainClient(ctrl)
	parserMock := indexer.NewMockParser(ctrl)
	parserMock.EXPECT().GetGenesisBlock(ctx).Return(dummyGenesis, nil)
	parserMock.EXPECT().ParseNonGenesisBlock(ctx, "", blockHeight).Return(parsedBlock, nil).AnyTimes()
	backend, err := NewBackend(
		pChainMock,
		parserMock,
		avaxAssetID,
		pChainNetworkIdentifier,
		avalancheNetworkID,
	)
	require.NoError(t, err)
	backend.getUTXOsPageSize = 2

	t.Run("Account Balance Test", func(t *testing.T) {
		require := require.New(t)

		addr, err := address.ParseToID(pChainAddr)
		require.NoError(err)
		utxo0Bytes := makeUtxoBytes(t, backend, utxos[0].id, utxos[0].amount)
		utxo1Bytes := makeUtxoBytes(t, backend, utxos[1].id, utxos[1].amount)
		utxo1ID, err := mapper.DecodeUTXOID(utxos[1].id)
		require.NoError(err)
		stakeUtxoBytes := makeStakeUtxoBytes(t, backend, utxos[1].amount)

		// Mock on GetAssetDescription
		pChainMock.EXPECT().GetAssetDescription(ctx, mapper.AtomicAvaxCurrency.Symbol).Return(mockAssetDescription, nil).AnyTimes()

		// once before other calls, once after
		pChainMock.EXPECT().GetHeight(ctx).Return(blockHeight, nil).Times(2)
		// Make sure pagination works as well
		pageSize := uint32(2)
		backend.getUTXOsPageSize = pageSize
		pChainMock.EXPECT().GetAtomicUTXOs(ctx, []ids.ShortID{addr}, "", pageSize, ids.ShortEmpty, ids.Empty).
			Return([][]byte{utxo0Bytes, utxo1Bytes}, addr, utxo1ID.InputID(), nil)
		pChainMock.EXPECT().GetAtomicUTXOs(ctx, []ids.ShortID{addr}, "", pageSize, addr, utxo1ID.InputID()).
			Return([][]byte{utxo1Bytes}, addr, utxo1ID.InputID(), nil)
		pChainMock.EXPECT().GetStake(ctx, []ids.ShortID{addr}, false).Return(map[ids.ID]uint64{}, [][]byte{stakeUtxoBytes}, nil)

		resp, terr := backend.AccountBalance(
			ctx,
			&types.AccountBalanceRequest{
				NetworkIdentifier: &types.NetworkIdentifier{
					Network: constants.FujiNetwork,
					SubNetworkIdentifier: &types.SubNetworkIdentifier{
						Network: constants.PChain.String(),
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

		require.Nil(terr)
		require.Equal([]*types.Amount{
			{
				Value:    "5000000000", // 1B + 2B from UTXOs, 1B from staked
				Currency: mapper.AtomicAvaxCurrency,
			},
		}, resp.Balances)
	})

	t.Run("Account Balance should return total of shared memory balance", func(t *testing.T) {
		require := require.New(t)

		// Mock on GetUTXOs
		utxo0Bytes := makeUtxoBytes(t, backend, utxos[0].id, utxos[0].amount)
		utxo1Bytes := makeUtxoBytes(t, backend, utxos[1].id, utxos[1].amount)
		utxo1ID, err := mapper.DecodeUTXOID(utxos[1].id)
		require.NoError(err)
		pChainAddrID, err := address.ParseToID(pChainAddr)
		require.NoError(err)

		// once before other calls, once after
		pChainMock.EXPECT().GetHeight(ctx).Return(blockHeight, nil).Times(2)
		pageSize := uint32(1024)
		backend.getUTXOsPageSize = pageSize
		pChainMock.EXPECT().GetAtomicUTXOs(ctx, []ids.ShortID{pChainAddrID}, constants.CChain.String(), pageSize, ids.ShortEmpty, ids.Empty).
			Return([][]byte{utxo0Bytes, utxo1Bytes}, pChainAddrID, utxo1ID.InputID(), nil)
		pChainMock.EXPECT().GetAtomicUTXOs(ctx, []ids.ShortID{pChainAddrID}, constants.XChain.String(), pageSize, ids.ShortEmpty, ids.Empty).
			Return([][]byte{}, pChainAddrID, ids.Empty, nil)

		resp, terr := backend.AccountBalance(
			ctx,
			&types.AccountBalanceRequest{
				NetworkIdentifier: &types.NetworkIdentifier{
					Network: constants.FujiNetwork,
					SubNetworkIdentifier: &types.SubNetworkIdentifier{
						Network: constants.PChain.String(),
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
		require.Nil(terr)
		require.Equal(&types.AccountBalanceResponse{
			BlockIdentifier: &types.BlockIdentifier{
				Index: int64(blockHeight),
				Hash:  parsedBlock.BlockID.String(),
			},
			Balances: []*types.Amount{{
				Value:    "3000000000",
				Currency: mapper.AtomicAvaxCurrency,
			}},
		}, resp)
	})

	t.Run("Account Balance should error if new block was added while fetching UTXOs", func(t *testing.T) {
		require := require.New(t)
		addr, err := address.ParseToID(pChainAddr)
		require.NoError(err)

		pageSize := uint32(2)
		backend.getUTXOsPageSize = pageSize
		pChainMock.EXPECT().GetHeight(ctx).Return(blockHeight, nil)
		pChainMock.EXPECT().GetAtomicUTXOs(ctx, []ids.ShortID{addr}, "", pageSize, ids.ShortEmpty, ids.Empty).
			Return([][]byte{}, addr, ids.Empty, nil)
		pChainMock.EXPECT().GetStake(ctx, []ids.ShortID{addr}, false).Return(map[ids.ID]uint64{}, [][]byte{}, nil)
		// return blockHeight + 1 to indicate a new block arrival
		pChainMock.EXPECT().GetHeight(ctx).Return(blockHeight+1, nil)

		resp, terr := backend.AccountBalance(
			ctx,
			&types.AccountBalanceRequest{
				NetworkIdentifier: &types.NetworkIdentifier{
					Network: constants.FujiNetwork,
					SubNetworkIdentifier: &types.SubNetworkIdentifier{
						Network: constants.PChain.String(),
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

		require.Nil(resp)
		require.Equal("Internal server error", terr.Message)
		require.Equal("new block added while fetching utxos", terr.Details["error"])
	})
}

func TestAccountPendingRewardsBalance(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	pChainMock := client.NewMockPChainClient(ctrl)
	parserMock := indexer.NewMockParser(ctrl)

	parserMock.EXPECT().GetGenesisBlock(ctx).Return(dummyGenesis, nil)
	parserMock.EXPECT().ParseNonGenesisBlock(ctx, "", blockHeight).Return(parsedBlock, nil).AnyTimes()

	validator1NodeID, err := ids.NodeIDFromString("NodeID-Bvsx89JttQqhqdgwtizAPoVSNW74Xcr2S")
	require.NoError(t, err)
	validator1Reward := uint64(100000)
	validator1AddressStr := "P-fuji1csj0hzu7rtljuhqnzp8m9shawlcefuvyl0m3e9"
	validator1Address, err := address.ParseToID(validator1AddressStr)
	require.NoError(t, err)
	validator1ValidationRewardOwner := &platformvm.ClientOwner{Addresses: []ids.ShortID{validator1Address}}

	delegate1Reward := uint64(20000)
	delegate1AddressStr := "P-fuji1raffss40pyr7hdhyp7p4hs6p049hjlc60xxwks"
	delegate1Address, err := address.ParseToID(delegate1AddressStr)
	require.NoError(t, err)
	delegate1RewardOwner := &platformvm.ClientOwner{Addresses: []ids.ShortID{delegate1Address}}

	delegate2Reward := uint64(30000)
	delegate2AddressStr := "P-fuji1tlt564kc8mqwr575lyg539r8h6xg7hfmgxnkcg"
	delegate2Address, err := address.ParseToID(delegate2AddressStr)
	require.NoError(t, err)
	delegate2RewardOwner := &platformvm.ClientOwner{Addresses: []ids.ShortID{delegate2Address}}

	validators := []platformvm.ClientPermissionlessValidator{
		{
			ClientStaker:          platformvm.ClientStaker{NodeID: validator1NodeID},
			ValidationRewardOwner: validator1ValidationRewardOwner,
			PotentialReward:       &validator1Reward,
			DelegationFee:         10,
			Delegators: []platformvm.ClientDelegator{
				{
					RewardOwner:     delegate1RewardOwner,
					PotentialReward: &delegate1Reward,
				},
				{
					RewardOwner:     delegate2RewardOwner,
					PotentialReward: &delegate2Reward,
				},
			},
		},
	}

	backend, err := NewBackend(
		pChainMock,
		parserMock,
		avaxAssetID,
		pChainNetworkIdentifier,
		avalancheNetworkID,
	)
	require.NoError(t, err)

	t.Run("Pending Rewards Validator By NodeID", func(t *testing.T) {
		pChainMock.EXPECT().GetCurrentValidators(ctx, ids.Empty, []ids.NodeID{validator1NodeID}).Return(validators, nil)
		pChainMock.EXPECT().GetHeight(ctx).Return(blockHeight, nil)

		resp, err := backend.AccountBalance(
			ctx,
			&types.AccountBalanceRequest{
				NetworkIdentifier: &types.NetworkIdentifier{
					Network: constants.FujiNetwork,
					SubNetworkIdentifier: &types.SubNetworkIdentifier{
						Network: constants.PChain.String(),
					},
				},
				AccountIdentifier: &types.AccountIdentifier{
					Address: validator1AddressStr,
					SubAccount: &types.SubAccountIdentifier{
						Address: validator1NodeID.String(),
					},
				},
			},
		)

		expected := &types.AccountBalanceResponse{
			BlockIdentifier: &types.BlockIdentifier{
				Index: int64(blockHeight),
				Hash:  parsedBlock.BlockID.String(),
			},
			Balances: []*types.Amount{
				{
					Value:    "105000",
					Currency: mapper.AtomicAvaxCurrency,
					Metadata: map[string]interface{}{
						pmapper.MetadataValidatorRewards:     "100000", // 100000 from validation
						pmapper.MetadataDelegationFeeRewards: "5000",   // 10% fee of total 50000 delegation
						pmapper.MetadataDelegationRewards:    "0",
					},
				},
			},
		}

		require.Nil(t, err)
		require.Equal(t, expected, resp)
	})

	t.Run("Pending Rewards Delegate by NodeID", func(t *testing.T) {
		pChainMock.EXPECT().GetCurrentValidators(ctx, ids.Empty, []ids.NodeID{validator1NodeID}).Return(validators, nil)
		pChainMock.EXPECT().GetHeight(ctx).Return(blockHeight, nil)

		resp, err := backend.AccountBalance(
			ctx,
			&types.AccountBalanceRequest{
				NetworkIdentifier: &types.NetworkIdentifier{
					Network: constants.FujiNetwork,
					SubNetworkIdentifier: &types.SubNetworkIdentifier{
						Network: constants.PChain.String(),
					},
				},
				AccountIdentifier: &types.AccountIdentifier{
					Address: delegate1AddressStr,
					SubAccount: &types.SubAccountIdentifier{
						Address: validator1NodeID.String(),
					},
				},
			},
		)

		expected := &types.AccountBalanceResponse{
			BlockIdentifier: &types.BlockIdentifier{
				Index: int64(blockHeight),
				Hash:  parsedBlock.BlockID.String(),
			},
			Balances: []*types.Amount{
				{
					Value:    "18000",
					Currency: mapper.AtomicAvaxCurrency,
					Metadata: map[string]interface{}{
						pmapper.MetadataDelegationRewards:    "18000", // 10 percent goes to validator, remaining is here
						pmapper.MetadataValidatorRewards:     "0",
						pmapper.MetadataDelegationFeeRewards: "0",
					},
				},
			},
		}

		require.Nil(t, err)
		require.Equal(t, expected, resp)
	})
}

func TestAccountCoins(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	pChainMock := client.NewMockPChainClient(ctrl)
	parserMock := indexer.NewMockParser(ctrl)
	parserMock.EXPECT().GetGenesisBlock(ctx).Return(dummyGenesis, nil)
	parserMock.EXPECT().ParseNonGenesisBlock(ctx, "", blockHeight).Return(parsedBlock, nil).AnyTimes()
	backend, err := NewBackend(
		pChainMock,
		parserMock,
		avaxAssetID,
		pChainNetworkIdentifier,
		avalancheNetworkID,
	)
	require.NoError(t, err)

	t.Run("Account Coins Test regular coins", func(t *testing.T) {
		require := require.New(t)

		// Mock on GetAssetDescription
		pChainMock.EXPECT().GetAssetDescription(ctx, mapper.AtomicAvaxCurrency.Symbol).Return(mockAssetDescription, nil)

		// Mock on GetUTXOs
		utxo0Bytes := makeUtxoBytes(t, backend, utxos[0].id, utxos[0].amount)
		utxo1Bytes := makeUtxoBytes(t, backend, utxos[1].id, utxos[1].amount)
		utxo1ID, err := mapper.DecodeUTXOID(utxos[1].id)
		require.NoError(err)
		pChainAddrID, err := address.ParseToID(pChainAddr)
		require.NoError(err)

		// once before other calls, once after
		pChainMock.EXPECT().GetHeight(ctx).Return(blockHeight, nil).Times(2)
		// Make sure pagination works as well
		pageSize := uint32(2)
		backend.getUTXOsPageSize = pageSize
		pChainMock.EXPECT().GetAtomicUTXOs(ctx, []ids.ShortID{pChainAddrID}, "", pageSize, ids.ShortEmpty, ids.Empty).
			Return([][]byte{utxo0Bytes, utxo1Bytes}, pChainAddrID, utxo1ID.InputID(), nil)
		pChainMock.EXPECT().GetAtomicUTXOs(ctx, []ids.ShortID{pChainAddrID}, "", pageSize, pChainAddrID, utxo1ID.InputID()).
			Return([][]byte{utxo1Bytes}, pChainAddrID, utxo1ID.InputID(), nil)

		resp, terr := backend.AccountCoins(
			ctx,
			&types.AccountCoinsRequest{
				NetworkIdentifier: &types.NetworkIdentifier{
					Network: constants.FujiNetwork,
					SubNetworkIdentifier: &types.SubNetworkIdentifier{
						Network: constants.PChain.String(),
					},
				},
				AccountIdentifier: &types.AccountIdentifier{
					Address: pChainAddr,
				},
				Currencies: []*types.Currency{
					mapper.AtomicAvaxCurrency,
				},
			})

		require.Nil(terr)
		require.Equal(&types.AccountCoinsResponse{
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
		}, resp)
	})

	t.Run("Account Coins Test shared memory coins", func(t *testing.T) {
		require := require.New(t)
		// Mock on GetAssetDescription
		pChainMock.EXPECT().GetAssetDescription(ctx, mapper.AtomicAvaxCurrency.Symbol).Return(mockAssetDescription, nil)

		// Mock on GetUTXOs
		utxo0Bytes := makeUtxoBytes(t, backend, utxos[0].id, utxos[0].amount)
		utxo0ID, err := mapper.DecodeUTXOID(utxos[0].id)
		require.NoError(err)
		utxo1Bytes := makeUtxoBytes(t, backend, utxos[1].id, utxos[1].amount)
		utxo1ID, err := mapper.DecodeUTXOID(utxos[1].id)
		require.NoError(err)
		pChainAddrID, err := address.ParseToID(pChainAddr)
		require.NoError(err)

		// once before other calls, once after
		pChainMock.EXPECT().GetHeight(ctx).Return(blockHeight, nil).Times(2)
		pageSize := uint32(1024)
		backend.getUTXOsPageSize = pageSize
		pChainMock.EXPECT().GetAtomicUTXOs(ctx, []ids.ShortID{pChainAddrID}, constants.CChain.String(), pageSize, ids.ShortEmpty, ids.Empty).
			Return([][]byte{utxo0Bytes}, pChainAddrID, utxo0ID.InputID(), nil)
		pChainMock.EXPECT().GetAtomicUTXOs(ctx, []ids.ShortID{pChainAddrID}, constants.XChain.String(), pageSize, ids.ShortEmpty, ids.Empty).
			Return([][]byte{utxo1Bytes}, pChainAddrID, utxo1ID.InputID(), nil)

		resp, terr := backend.AccountCoins(
			ctx,
			&types.AccountCoinsRequest{
				NetworkIdentifier: &types.NetworkIdentifier{
					Network: constants.FujiNetwork,
					SubNetworkIdentifier: &types.SubNetworkIdentifier{
						Network: constants.PChain.String(),
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

		require.Nil(terr)
		require.Equal(expected, resp)
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
