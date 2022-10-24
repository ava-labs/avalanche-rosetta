package pchain

import (
	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/avalanchego/vms/components/avax"
	"github.com/ava-labs/avalanchego/vms/platformvm/txs"
)

// DependencyTx represents a transaction dependency
type DependencyTx struct {
	// [Tx] has some of its outputs spent as
	// input from a tx dependent on it
	Tx *txs.Tx

	// Staker txs are rewarded at the end of staking period
	// [RewardUTXOs] collects those reward utxos
	RewardUTXOs []*avax.UTXO

	// [utxosMap] caches mapping of Tx utxo index --> Tx utxo
	// for both Tx and RewardUTXOs
	utxosMap map[avax.UTXOID]*avax.UTXO
}

func (d *DependencyTx) GetUtxos() map[avax.UTXOID]*avax.UTXO {
	if d.utxosMap != nil {
		return d.utxosMap
	}

	utxos := make(map[avax.UTXOID]*avax.UTXO)
	if d.Tx != nil {
		// Generate UTXOs from outputs
		switch unsignedTx := d.Tx.Unsigned.(type) {
		case *txs.ExportTx:
			mapUTXOs(d.Tx.ID(), unsignedTx.Outputs(), utxos)
		case *txs.ImportTx:
			mapUTXOs(d.Tx.ID(), unsignedTx.Outputs(), utxos)
		case *txs.AddValidatorTx:
			mapUTXOs(d.Tx.ID(), unsignedTx.Outputs(), utxos)
			mapUTXOs(d.Tx.ID(), unsignedTx.Stake(), utxos)
		case *txs.AddPermissionlessValidatorTx:
			mapUTXOs(d.Tx.ID(), unsignedTx.Outputs(), utxos)
			mapUTXOs(d.Tx.ID(), unsignedTx.Stake(), utxos)
		case *txs.AddDelegatorTx:
			mapUTXOs(d.Tx.ID(), unsignedTx.Outputs(), utxos)
			mapUTXOs(d.Tx.ID(), unsignedTx.Stake(), utxos)
		case *txs.AddPermissionlessDelegatorTx:
			mapUTXOs(d.Tx.ID(), unsignedTx.Outputs(), utxos)
			mapUTXOs(d.Tx.ID(), unsignedTx.Stake(), utxos)
		case *txs.CreateSubnetTx:
			mapUTXOs(d.Tx.ID(), unsignedTx.Outputs(), utxos)
		case *txs.AddSubnetValidatorTx:
			mapUTXOs(d.Tx.ID(), unsignedTx.Outputs(), utxos)
		case *txs.TransformSubnetTx:
			mapUTXOs(d.Tx.ID(), unsignedTx.Outputs(), utxos)
		case *txs.RemoveSubnetValidatorTx:
			mapUTXOs(d.Tx.ID(), unsignedTx.Outputs(), utxos)
		case *txs.CreateChainTx:
			mapUTXOs(d.Tx.ID(), unsignedTx.Outputs(), utxos)
		default:
			// no utxos extracted from unsupported transaction types
		}
	}

	// Add reward UTXOs
	for _, utxo := range d.RewardUTXOs {
		utxos[utxo.UTXOID] = utxo
	}

	d.utxosMap = utxos
	return utxos
}

func mapUTXOs(txID ids.ID, outs []*avax.TransferableOutput, utxos map[avax.UTXOID]*avax.UTXO) {
	for i, out := range outs {
		utxoID := avax.UTXOID{
			TxID:        txID,
			OutputIndex: uint32(i),
		}
		utxos[utxoID] = &avax.UTXO{
			UTXOID: utxoID,
			Asset:  out.Asset,
			Out:    out.Out,
		}
	}
}
