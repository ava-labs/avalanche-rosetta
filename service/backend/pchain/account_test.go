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
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	"github.com/ava-labs/avalanche-rosetta/client"
	"github.com/ava-labs/avalanche-rosetta/constants"
	"github.com/ava-labs/avalanche-rosetta/mapper"
	pmapper "github.com/ava-labs/avalanche-rosetta/mapper/pchain"
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
	parserMock.EXPECT().ParseNonGenesisBlock(ctx, "", blockHeight).Return(parsedBlock, nil)
	backend, err := NewBackend(
		service.ModeOnline,
		pChainMock,
		parserMock,
		avaxAssetID,
		pChainNetworkIdentifier,
		avalancheNetworkID,
	)
	assert.Nil(t, err)
	backend.getUTXOsPageSize = 2

	t.Run("Account Balance Test", func(t *testing.T) {
		addr, _ := address.ParseToID(pChainAddr)
		utxo0Bytes := makeUtxoBytes(t, backend, utxos[0].id, utxos[0].amount)
		utxo1Bytes := makeUtxoBytes(t, backend, utxos[1].id, utxos[1].amount)
		utxo1Id, _ := ids.FromString(utxos[1].id)
		stakeUtxoBytes := makeStakeUtxoBytes(t, backend, utxos[1].amount)

		pageSize := uint32(2)
		backend.getUTXOsPageSize = pageSize

		gomock.InOrder(
			pChainMock.EXPECT().GetAssetDescription(ctx, mapper.AtomicAvaxCurrency.Symbol).Return(mockAssetDescription, nil),
			pChainMock.EXPECT().GetHeight(ctx).Return(blockHeight, nil),
			pChainMock.EXPECT().GetHeight(ctx).Return(blockHeight, nil),
			pChainMock.EXPECT().GetAtomicUTXOs(ctx, []ids.ShortID{addr}, "", pageSize, ids.ShortEmpty, ids.Empty).
				Return([][]byte{utxo0Bytes, utxo1Bytes}, addr, utxo1Id, nil),
			pChainMock.EXPECT().GetAtomicUTXOs(ctx, []ids.ShortID{addr}, "", pageSize, addr, utxo1Id).
				Return([][]byte{utxo1Bytes}, addr, utxo1Id, nil),
			pChainMock.EXPECT().GetStake(ctx, []ids.ShortID{addr}, false).Return(map[ids.ID]uint64{}, [][]byte{stakeUtxoBytes}, nil),
		)

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
	})

	t.Run("Account Balance should return total of shared memory balance", func(t *testing.T) {
		utxo0Bytes := makeUtxoBytes(t, backend, utxos[0].id, utxos[0].amount)
		utxo1Bytes := makeUtxoBytes(t, backend, utxos[1].id, utxos[1].amount)
		utxo1Id, _ := ids.FromString(utxos[1].id)
		pChainAddrID, errp := address.ParseToID(pChainAddr)
		assert.Nil(t, errp)

		pageSize := uint32(1024)
		backend.getUTXOsPageSize = pageSize

		gomock.InOrder(
			pChainMock.EXPECT().GetHeight(ctx).Return(blockHeight, nil),
			pChainMock.EXPECT().GetAtomicUTXOs(ctx, []ids.ShortID{pChainAddrID}, constants.CChain.String(), pageSize, ids.ShortEmpty, ids.Empty).
				Return([][]byte{utxo0Bytes, utxo1Bytes}, pChainAddrID, utxo1Id, nil),
			pChainMock.EXPECT().GetAtomicUTXOs(ctx, []ids.ShortID{pChainAddrID}, constants.XChain.String(), pageSize, ids.ShortEmpty, ids.Empty).
				Return([][]byte{}, pChainAddrID, ids.Empty, nil),
		)

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
	})

	t.Run("Account Balance should error if new block was added while fetching UTXOs", func(t *testing.T) {
		addr, _ := address.ParseToID(pChainAddr)

		pageSize := uint32(2)
		backend.getUTXOsPageSize = pageSize
		gomock.InOrder(
			pChainMock.EXPECT().GetHeight(ctx).Return(blockHeight, nil),
			pChainMock.EXPECT().GetAtomicUTXOs(ctx, []ids.ShortID{addr}, "", pageSize, ids.ShortEmpty, ids.Empty).
				Return([][]byte{}, addr, ids.Empty, nil),
			pChainMock.EXPECT().GetStake(ctx, []ids.ShortID{addr}, false).Return(map[ids.ID]uint64{}, [][]byte{}, nil),
			// return blockHeight + 1 to indicate a new block arrival
			pChainMock.EXPECT().GetHeight(ctx).Return(blockHeight+1, nil),
		)

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
	})
}

func TestAccountPendingRewardsBalance(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	pChainMock := client.NewMockPChainClient(ctrl)
	parserMock := indexer.NewMockParser(ctrl)

	parserMock.EXPECT().GetGenesisBlock(ctx).Return(dummyGenesis, nil)
	parserMock.EXPECT().ParseNonGenesisBlock(ctx, "", blockHeight).Return(parsedBlock, nil).Times(2)

	validator1NodeID, _ := ids.NodeIDFromString("NodeID-Bvsx89JttQqhqdgwtizAPoVSNW74Xcr2S")
	validator1Reward := uint64(100000)
	validator1AddressStr := "P-fuji1csj0hzu7rtljuhqnzp8m9shawlcefuvyl0m3e9"
	validator1Address, _ := address.ParseToID(validator1AddressStr)
	validator1ValidationRewardOwner := &platformvm.ClientOwner{Addresses: []ids.ShortID{validator1Address}}

	delegate1Reward := uint64(20000)
	delegate1AddressStr := "P-fuji1raffss40pyr7hdhyp7p4hs6p049hjlc60xxwks"
	delegate1Address, _ := address.ParseToID(delegate1AddressStr)
	delegate1RewardOwner := &platformvm.ClientOwner{Addresses: []ids.ShortID{delegate1Address}}

	delegate2Reward := uint64(30000)
	delegate2AddressStr := "P-fuji1tlt564kc8mqwr575lyg539r8h6xg7hfmgxnkcg"
	delegate2Address, _ := address.ParseToID(delegate2AddressStr)
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
		service.ModeOnline,
		pChainMock,
		parserMock,
		avaxAssetID,
		pChainNetworkIdentifier,
		avalancheNetworkID,
	)
	assert.Nil(t, err)

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

		assert.Nil(t, err)
		assert.Equal(t, expected, resp)
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

		assert.Nil(t, err)
		assert.Equal(t, expected, resp)
	})
}

func TestAccountCoins(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	pChainMock := client.NewMockPChainClient(ctrl)
	parserMock := indexer.NewMockParser(ctrl)
	parserMock.EXPECT().GetGenesisBlock(ctx).Return(dummyGenesis, nil)
	parserMock.EXPECT().ParseNonGenesisBlock(ctx, "", blockHeight).Return(parsedBlock, nil).Times(2)
	backend, err := NewBackend(
		service.ModeOnline,
		pChainMock,
		parserMock,
		avaxAssetID,
		pChainNetworkIdentifier,
		avalancheNetworkID,
	)
	assert.Nil(t, err)

	t.Run("Account Coins Test regular coins", func(t *testing.T) {
		pChainMock.EXPECT().GetAssetDescription(ctx, mapper.AtomicAvaxCurrency.Symbol).Return(mockAssetDescription, nil)

		utxo0Bytes := makeUtxoBytes(t, backend, utxos[0].id, utxos[0].amount)
		utxo1Bytes := makeUtxoBytes(t, backend, utxos[1].id, utxos[1].amount)
		utxo1Id, _ := ids.FromString(utxos[1].id)
		pChainAddrID, errp := address.ParseToID(pChainAddr)
		assert.Nil(t, errp)

		pageSize := uint32(2)
		backend.getUTXOsPageSize = pageSize

		gomock.InOrder(
			pChainMock.EXPECT().GetHeight(ctx).Return(blockHeight, nil),
			pChainMock.EXPECT().GetAtomicUTXOs(ctx, []ids.ShortID{pChainAddrID}, "", pageSize, ids.ShortEmpty, ids.Empty).
				Return([][]byte{utxo0Bytes, utxo1Bytes}, pChainAddrID, utxo1Id, nil),
			pChainMock.EXPECT().GetAtomicUTXOs(ctx, []ids.ShortID{pChainAddrID}, "", pageSize, pChainAddrID, utxo1Id).
				Return([][]byte{utxo1Bytes}, pChainAddrID, utxo1Id, nil),
			pChainMock.EXPECT().GetHeight(ctx).Return(blockHeight, nil),
		)

		resp, err := backend.AccountCoins(
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
	})

	t.Run("Account Coins Test shared memory coins", func(t *testing.T) {
		pChainMock.EXPECT().GetAssetDescription(ctx, mapper.AtomicAvaxCurrency.Symbol).Return(mockAssetDescription, nil)

		utxo0Bytes := makeUtxoBytes(t, backend, utxos[0].id, utxos[0].amount)
		utxo0Id, _ := ids.FromString(utxos[0].id)
		utxo1Bytes := makeUtxoBytes(t, backend, utxos[1].id, utxos[1].amount)
		utxo1Id, _ := ids.FromString(utxos[1].id)
		pChainAddrID, errp := address.ParseToID(pChainAddr)
		assert.Nil(t, errp)

		pageSize := uint32(1024)
		backend.getUTXOsPageSize = pageSize

		gomock.InOrder(
			pChainMock.EXPECT().GetHeight(ctx).Return(blockHeight, nil),
			pChainMock.EXPECT().GetAtomicUTXOs(ctx, []ids.ShortID{pChainAddrID}, constants.CChain.String(), pageSize, ids.ShortEmpty, ids.Empty).
				Return([][]byte{utxo0Bytes}, pChainAddrID, utxo0Id, nil),
			pChainMock.EXPECT().GetAtomicUTXOs(ctx, []ids.ShortID{pChainAddrID}, constants.XChain.String(), pageSize, ids.ShortEmpty, ids.Empty).
				Return([][]byte{utxo1Bytes}, pChainAddrID, utxo1Id, nil),
			pChainMock.EXPECT().GetHeight(ctx).Return(blockHeight, nil),
		)

		resp, err := backend.AccountCoins(
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

		assert.Nil(t, err)
		assert.Equal(t, expected, resp)
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
