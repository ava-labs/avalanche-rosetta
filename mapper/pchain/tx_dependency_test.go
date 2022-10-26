package pchain

import (
	"testing"

	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/avalanchego/utils/crypto"
	"github.com/ava-labs/avalanchego/vms/components/avax"
	"github.com/ava-labs/avalanchego/vms/platformvm/txs"
	"github.com/ava-labs/avalanchego/vms/secp256k1fx"
	"github.com/stretchr/testify/require"
)

var preFundedKeys = crypto.BuildTestKeys()

func TestTxDependencyIsCreateChain(t *testing.T) {
	require := require.New(t)

	in := &avax.TransferableInput{
		UTXOID: avax.UTXOID{
			TxID:        ids.ID{'t', 'x', 'I', 'D'},
			OutputIndex: 2,
		},
		Asset: avax.Asset{ID: ids.ID{'a', 's', 's', 'e', 'r', 't'}},
		In: &secp256k1fx.TransferInput{
			Amt:   uint64(5678),
			Input: secp256k1fx.Input{SigIndices: []uint32{0}},
		},
	}

	// simple output
	out := &avax.TransferableOutput{
		Asset: avax.Asset{ID: ids.ID{'a', 's', 's', 'e', 't', '1'}},
		Out: &secp256k1fx.TransferOutput{
			Amt: uint64(1234),
			OutputOwners: secp256k1fx.OutputOwners{
				Threshold: 1,
				Addrs:     []ids.ShortID{preFundedKeys[0].PublicKey().Address()},
			},
		},
	}

	// multisign output
	multiSignOut := &avax.TransferableOutput{
		Asset: avax.Asset{ID: ids.ID{'a', 's', 's', 'e', 't', '2'}},
		Out: &secp256k1fx.TransferOutput{
			Amt: uint64(5678),
			OutputOwners: secp256k1fx.OutputOwners{
				Threshold: 1,
				Addrs: []ids.ShortID{
					preFundedKeys[1].PublicKey().Address(),
					preFundedKeys[2].PublicKey().Address(),
				},
			},
		},
	}

	// create a non-reward validator tx
	utx := &txs.CreateChainTx{
		BaseTx: txs.BaseTx{BaseTx: avax.BaseTx{
			NetworkID:    10,
			BlockchainID: ids.ID{'c', 'h', 'a', 'i', 'n', 'I', 'D'},
			Ins:          []*avax.TransferableInput{in},
			Outs:         []*avax.TransferableOutput{out, multiSignOut},
			Memo:         []byte{1, 2, 3, 4, 5, 6, 7, 8},
		}},
		SubnetID:    ids.ID{'s', 'u', 'b', 'n', 'e', 't', 'I', 'D'},
		ChainName:   "a chain",
		VMID:        ids.GenerateTestID(),
		FxIDs:       []ids.ID{ids.GenerateTestID()},
		GenesisData: []byte{'g', 'e', 'n', 'D', 'a', 't', 'a'},
		SubnetAuth:  &secp256k1fx.Input{SigIndices: []uint32{1}},
	}
	tx, err := txs.NewSigned(utx, txs.Codec, nil)
	require.NoError(err)

	dep := &SingleTxDependency{Tx: tx}
	res := dep.GetUtxos()
	require.True(len(res) == 2)

	expectedUTXOs := []*avax.UTXO{
		{
			UTXOID: avax.UTXOID{
				TxID:        tx.ID(),
				OutputIndex: 0,
			},
			Asset: out.Asset,
			Out:   out.Out,
		},
		{
			UTXOID: avax.UTXOID{
				TxID:        tx.ID(),
				OutputIndex: 1,
			},
			Asset: multiSignOut.Asset,
			Out:   multiSignOut.Out,
		},
	}

	utxo, found := res[expectedUTXOs[0].UTXOID]
	require.True(found)
	require.Equal(utxo, expectedUTXOs[0])

	utxo, found = res[expectedUTXOs[1].UTXOID]
	require.True(found)
	require.Equal(utxo, expectedUTXOs[1])

	// show idempotency
	res2 := dep.GetUtxos()
	require.Equal(res, res2)
}
