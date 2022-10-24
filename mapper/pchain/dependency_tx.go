package pchain

import (
	"github.com/ava-labs/avalanchego/vms/components/avax"
	"github.com/ava-labs/avalanchego/vms/platformvm/txs"
)

// DependencyTx represents a single dependency of a give transaction
type DependencyTx struct {
	// [Tx] has some of its outputs spent as
	// input from a tx dependent on it
	Tx *txs.Tx

	// Staker txs are rewarded at the end of staking period
	// with some utxos appended to staker txs.
	// [RewardUTXOs] collects those reward utxos
	RewardUTXOs []*avax.UTXO

	// [utxosMap] caches mapping of Tx utxoID --> Tx utxo
	// for both Tx and RewardUTXOs
	utxosMap map[avax.UTXOID]*avax.UTXO
}

func (d *DependencyTx) GetUtxos() map[avax.UTXOID]*avax.UTXO {
	if d.utxosMap != nil {
		return d.utxosMap
	}

	// Add reward UTXOs
	for _, utxo := range d.RewardUTXOs {
		d.utxosMap[utxo.UTXOID] = utxo
	}

	if d.Tx != nil {
		// Generate UTXOs from outputs
		outsToAdd := make([]*avax.TransferableOutput, 0)
		switch unsignedTx := d.Tx.Unsigned.(type) {
		case *txs.ExportTx:
			outsToAdd = append(outsToAdd, unsignedTx.Outputs()...)
		case *txs.ImportTx:
			outsToAdd = append(outsToAdd, unsignedTx.Outputs()...)
		case *txs.AddValidatorTx:
			outsToAdd = append(outsToAdd, unsignedTx.Outputs()...)
			outsToAdd = append(outsToAdd, unsignedTx.Stake()...)
		case *txs.AddPermissionlessValidatorTx:
			outsToAdd = append(outsToAdd, unsignedTx.Outputs()...)
			outsToAdd = append(outsToAdd, unsignedTx.Stake()...)
		case *txs.AddDelegatorTx:
			outsToAdd = append(outsToAdd, unsignedTx.Outputs()...)
			outsToAdd = append(outsToAdd, unsignedTx.Stake()...)
		case *txs.AddPermissionlessDelegatorTx:
			outsToAdd = append(outsToAdd, unsignedTx.Outputs()...)
			outsToAdd = append(outsToAdd, unsignedTx.Stake()...)
		case *txs.CreateSubnetTx:
			outsToAdd = append(outsToAdd, unsignedTx.Outputs()...)
		case *txs.AddSubnetValidatorTx:
			outsToAdd = append(outsToAdd, unsignedTx.Outputs()...)
		case *txs.TransformSubnetTx:
			outsToAdd = append(outsToAdd, unsignedTx.Outputs()...)
		case *txs.RemoveSubnetValidatorTx:
			outsToAdd = append(outsToAdd, unsignedTx.Outputs()...)
		case *txs.CreateChainTx:
			outsToAdd = append(outsToAdd, unsignedTx.Outputs()...)
		default:
			// no utxos extracted from unsupported transaction types
		}

		// add collected utxos
		txID := d.Tx.ID()
		for i, out := range outsToAdd {
			utxoID := avax.UTXOID{
				TxID:        txID,
				OutputIndex: uint32(i),
			}
			d.utxosMap[utxoID] = &avax.UTXO{
				UTXOID: utxoID,
				Asset:  out.Asset,
				Out:    out.Out,
			}
		}
	}

	return d.utxosMap
}
