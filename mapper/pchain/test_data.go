package pchain

import (
	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/avalanchego/utils/formatting/address"
	"github.com/ava-labs/avalanchego/vms/components/avax"
	"github.com/ava-labs/avalanchego/vms/platformvm/blocks"
	"github.com/ava-labs/avalanchego/vms/platformvm/stakeable"
	"github.com/ava-labs/avalanchego/vms/platformvm/txs"
	"github.com/ava-labs/avalanchego/vms/secp256k1fx"
	"github.com/coinbase/rosetta-sdk-go/types"
)

func buildImport() (*txs.Tx, *txs.ImportTx, map[string]*types.AccountIdentifier) {
	avaxAssetID, _ := ids.FromString("U8iRqJoiJm8xZHAacmvYyZVwqQx6uDNtQeP3CQ6fcgQk3JqnK")
	sourceChain, _ := ids.FromString("2JVSBoinj9C2J33VntvzYtVJNZdN2NKiwwKjcumHUWEb5DbBrm")
	outAddr1, _ := address.ParseToID("P-fuji1xm0r37l6gyf2mly4pmzc0tz6wnwqkugedh95fk")
	outAddr2, _ := address.ParseToID("P-fuji1fmragvegm5k26qzlt6vy0ghhdr508u6r4a5rxj")
	outAddr3, _ := address.ParseToID("P-fuji1j3sw805usytrsymfwxxrcwfqguyarumn45cllj")
	importAddr, _ := address.ParseToID("C-fuji1xm0r37l6gyf2mly4pmzc0tz6wnwqkugedh95fk")
	importedTxID, _ := ids.FromString("2DtYhzCvo9LRYMRJ6sCtYJ4aNPRpsibp46ETNyY6H5Cox1VLvX")
	importTx := &txs.ImportTx{
		BaseTx: txs.BaseTx{
			BaseTx: avax.BaseTx{
				NetworkID:    uint32(5),
				BlockchainID: [32]byte{},
				Outs: []*avax.TransferableOutput{
					{
						Asset: avax.Asset{ID: avaxAssetID},
						FxID:  [32]byte{},
						Out: &secp256k1fx.TransferOutput{
							Amt: 8000000,
							OutputOwners: secp256k1fx.OutputOwners{
								Locktime:  0,
								Threshold: 1,
								Addrs:     []ids.ShortID{outAddr1},
							},
						},
					},
					{ //  this will be skipped as it is multisig
						Asset: avax.Asset{ID: avaxAssetID},
						FxID:  [32]byte{},
						Out: &secp256k1fx.TransferOutput{
							Amt: 8000000,
							OutputOwners: secp256k1fx.OutputOwners{
								Locktime:  0,
								Threshold: 2,
								Addrs:     []ids.ShortID{outAddr1, outAddr2, outAddr3},
							},
						},
					},
					{ //  this will be skipped as it does not have any addresses
						Asset: avax.Asset{ID: avaxAssetID},
						FxID:  [32]byte{},
						Out: &secp256k1fx.TransferOutput{
							Amt: 1000000,
							OutputOwners: secp256k1fx.OutputOwners{
								Locktime:  0,
								Threshold: 0,
								Addrs:     []ids.ShortID{},
							},
						},
					},
				},
				Ins:  nil,
				Memo: []byte{},
			},
		},
		SourceChain: sourceChain,
		ImportedInputs: []*avax.TransferableInput{{
			UTXOID: avax.UTXOID{
				TxID:        importedTxID,
				OutputIndex: 0,
				Symbol:      false,
			},
			Asset: avax.Asset{ID: avaxAssetID},
			FxID:  [32]byte{},
			In: &secp256k1fx.TransferInput{
				Amt: 9000000,
				Input: secp256k1fx.Input{
					SigIndices: []uint32{},
				},
			},
		}},
	}

	signedTx, _ := txs.NewSigned(importTx, blocks.Codec, nil)

	inputTxAccounts := map[string]*types.AccountIdentifier{}
	inputTxAccounts[importTx.ImportedInputs[0].String()] = &types.AccountIdentifier{Address: importAddr.String()}

	return signedTx, importTx, inputTxAccounts
}

func buildExport() (*txs.Tx, *txs.ExportTx, map[string]*types.AccountIdentifier) {
	avaxAssetID, _ := ids.FromString("U8iRqJoiJm8xZHAacmvYyZVwqQx6uDNtQeP3CQ6fcgQk3JqnK")
	outAddr, _ := address.ParseToID("P-fuji1wmd9dfrqpud6daq0cde47u0r7pkrr46ep60399")
	exportOutAddr, _ := address.ParseToID("P-fuji1wmd9dfrqpud6daq0cde47u0r7pkrr46ep60399")
	txID, _ := ids.FromString("27LaDkrUrMY1bhVf2i8RARCrRwFjeRw7vEu8ntLQXracgLzL1v")
	destinationID, _ := ids.FromString("yH8D7ThNJkxmtkuv2jgBa4P1Rn3Qpr4pPr7QYNfcdoS6k6HWp")
	exportTx := &txs.ExportTx{
		BaseTx: txs.BaseTx{
			BaseTx: avax.BaseTx{
				NetworkID:    uint32(5),
				BlockchainID: [32]byte{},
				Outs: []*avax.TransferableOutput{{
					Asset: avax.Asset{ID: avaxAssetID},
					FxID:  [32]byte{},
					Out: &secp256k1fx.TransferOutput{
						Amt: 2910137500,
						OutputOwners: secp256k1fx.OutputOwners{
							Locktime:  0,
							Threshold: 1,
							Addrs:     []ids.ShortID{outAddr},
						},
					},
				}},
				Ins: []*avax.TransferableInput{{
					UTXOID: avax.UTXOID{TxID: txID, OutputIndex: 0, Symbol: false},
					Asset:  avax.Asset{ID: avaxAssetID},
					FxID:   [32]byte{},
					In: &secp256k1fx.TransferInput{
						Amt:   2921137500,
						Input: secp256k1fx.Input{SigIndices: []uint32{}},
					},
				}},
				Memo: []byte{},
			},
		},
		DestinationChain: destinationID,
		ExportedOutputs: []*avax.TransferableOutput{{
			Asset: avax.Asset{ID: avaxAssetID},
			FxID:  [32]byte{},
			Out: &secp256k1fx.TransferOutput{
				Amt: 10000000,
				OutputOwners: secp256k1fx.OutputOwners{
					Locktime:  0,
					Threshold: 1,
					Addrs:     []ids.ShortID{exportOutAddr},
				},
			},
		}},
	}

	signedTx, _ := txs.NewSigned(exportTx, blocks.Codec, nil)

	inputTxAccounts := map[string]*types.AccountIdentifier{}
	inputTxAccounts[exportTx.Ins[0].String()] = &types.AccountIdentifier{Address: outAddr.String()}

	return signedTx, exportTx, inputTxAccounts
}

func buildAddDelegator() (*txs.Tx, *txs.AddDelegatorTx, map[string]*types.AccountIdentifier) {
	avaxAssetID, _ := ids.FromString("U8iRqJoiJm8xZHAacmvYyZVwqQx6uDNtQeP3CQ6fcgQk3JqnK")
	txID, _ := ids.FromString("2JQGX1MBdszAaeV6eApCZm7CBpc917qWiyQ2cygFRJ6WteDkre")
	outAddr, _ := address.ParseToID("P-fuji1gdkq8g208e3j4epyjmx65jglsw7vauh86l47ac")
	validatorID, _ := ids.NodeIDFromString("NodeID-BFa1padLXBj7VHa2JYvYGzcTBPQGjPhUy")
	stakeAddr, _ := address.ParseToID("P-fuji1l022sue7g2kzvrcuxughl30xkss2cj0az3e5r2")
	rewardAddr, _ := address.ParseToID("P-fuji1l022sue7g2kzvrcuxughl30xkss2cj0az3e5r2")
	addDelegator := &txs.AddDelegatorTx{
		BaseTx: txs.BaseTx{
			BaseTx: avax.BaseTx{
				NetworkID:    uint32(5),
				BlockchainID: [32]byte{},
				Outs: []*avax.TransferableOutput{{
					Asset: avax.Asset{ID: avaxAssetID},
					FxID:  [32]byte{},
					Out: &secp256k1fx.TransferOutput{
						Amt: 996649063,
						OutputOwners: secp256k1fx.OutputOwners{
							Locktime:  9,
							Threshold: 1,
							Addrs:     []ids.ShortID{outAddr},
						},
					},
				}},
				Ins: []*avax.TransferableInput{{
					UTXOID: avax.UTXOID{TxID: txID, OutputIndex: 0, Symbol: false},
					Asset:  avax.Asset{ID: avaxAssetID},
					FxID:   [32]byte{},
					In: &secp256k1fx.TransferInput{
						Amt:   1996649063,
						Input: secp256k1fx.Input{SigIndices: []uint32{}},
					},
				}},
				Memo: []byte{},
			},
		},
		Validator: txs.Validator{
			NodeID: validatorID,
			Start:  1656058022,
			End:    1657872569,
			Wght:   1000000000,
		},
		StakeOuts: []*avax.TransferableOutput{{
			Asset: avax.Asset{ID: avaxAssetID},
			FxID:  [32]byte{},
			Out: &secp256k1fx.TransferOutput{
				Amt: 1000000000,
				OutputOwners: secp256k1fx.OutputOwners{
					Locktime:  0,
					Threshold: 1,
					Addrs:     []ids.ShortID{stakeAddr},
				},
			},
		}},
		DelegationRewardsOwner: &secp256k1fx.OutputOwners{
			Locktime:  0,
			Threshold: 1,
			Addrs:     []ids.ShortID{rewardAddr},
		},
	}

	signedTx, _ := txs.NewSigned(addDelegator, blocks.Codec, nil)

	inputTxAccounts := map[string]*types.AccountIdentifier{}
	inputTxAccounts[addDelegator.Ins[0].String()] = &types.AccountIdentifier{Address: stakeAddr.String()}

	return signedTx, addDelegator, inputTxAccounts
}

func buildValidatorTx() (*txs.Tx, *txs.AddValidatorTx, map[string]*types.AccountIdentifier) {
	avaxAssetID, _ := ids.FromString("U8iRqJoiJm8xZHAacmvYyZVwqQx6uDNtQeP3CQ6fcgQk3JqnK")

	txID, _ := ids.FromString("88tfp1Pkw9vyKrRtVNiMrghFBrre6Q6CzqPW1t7StDNX9PJEo")
	stakeAddr, _ := address.ParseToID("P-fuji1ljdzyey6vu3hgn3cwg4j5lpy0svd6arlxpj6je")
	rewardAddr, _ := address.ParseToID("P-fuji1ljdzyey6vu3hgn3cwg4j5lpy0svd6arlxpj6je")
	validatorID, _ := ids.NodeIDFromString("NodeID-CCecHmRK3ANe92VyvASxkNav26W4vAVpX")
	addValidator := &txs.AddValidatorTx{
		BaseTx: txs.BaseTx{
			BaseTx: avax.BaseTx{
				NetworkID:    uint32(5),
				BlockchainID: [32]byte{},
				Outs:         nil,
				Ins: []*avax.TransferableInput{ // two inputs, the second locktimed
					{
						UTXOID: avax.UTXOID{TxID: txID, OutputIndex: 0},
						Asset:  avax.Asset{ID: avaxAssetID},
						FxID:   [32]byte{},
						In: &secp256k1fx.TransferInput{
							Amt:   2000000000,
							Input: secp256k1fx.Input{SigIndices: []uint32{1}},
						},
					},
					{
						UTXOID: avax.UTXOID{TxID: txID, OutputIndex: 1},
						Asset:  avax.Asset{ID: avaxAssetID},
						FxID:   [32]byte{},
						In: &stakeable.LockIn{
							Locktime: uint64(1666781236), // a unix time
							TransferableIn: &secp256k1fx.TransferInput{
								Amt:   2000000000,
								Input: secp256k1fx.Input{SigIndices: []uint32{1}},
							},
						},
					},
				},
				Memo: []byte{},
			},
		},
		Validator: txs.Validator{
			NodeID: validatorID,
			Start:  1656084079,
			End:    1687620079,
			Wght:   2000000000,
		},
		StakeOuts: []*avax.TransferableOutput{{
			Asset: avax.Asset{ID: avaxAssetID},
			FxID:  [32]byte{},
			Out: &secp256k1fx.TransferOutput{
				Amt: 2000000000,
				OutputOwners: secp256k1fx.OutputOwners{
					Locktime:  0,
					Threshold: 1,
					Addrs:     []ids.ShortID{stakeAddr},
				},
			},
		}},
		RewardsOwner: &secp256k1fx.OutputOwners{
			Locktime:  0,
			Threshold: 1,
			Addrs:     []ids.ShortID{rewardAddr},
		},
		DelegationShares: 20000,
	}

	signedTx, _ := txs.NewSigned(addValidator, blocks.Codec, nil)

	inputTxAccounts := map[string]*types.AccountIdentifier{
		addValidator.Ins[0].String(): {Address: stakeAddr.String()},
		addValidator.Ins[1].String(): {Address: stakeAddr.String()},
	}

	return signedTx, addValidator, inputTxAccounts
}
