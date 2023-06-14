package pchain

import (
	"fmt"

	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/avalanchego/utils"
	"github.com/ava-labs/avalanchego/utils/formatting/address"
	"github.com/ava-labs/avalanchego/vms/components/avax"
	"github.com/ava-labs/avalanchego/vms/platformvm/txs"
	"github.com/coinbase/rosetta-sdk-go/types"

	"github.com/ava-labs/avalanche-rosetta/constants"
)

type BlockTxDependencies map[ids.ID]*SingleTxDependency

// GetTxDependenciesIDs generates the list of transaction ids used in the inputs to given unsigned transaction
// this list is then used to fetch the dependency transactions in order to extract source addresses
// as this information is not part of the transaction objects on chain.
func (bd BlockTxDependencies) GetTxDependenciesIDs(tx txs.UnsignedTx) ([]ids.ID, error) {
	// collect tx inputs
	var ins []*avax.TransferableInput
	switch unsignedTx := tx.(type) {
	case *txs.ExportTx:
		ins = unsignedTx.Ins
	case *txs.ImportTx:
		ins = unsignedTx.Ins
	case *txs.AddValidatorTx:
		ins = unsignedTx.Ins
	case *txs.AddPermissionlessValidatorTx:
		ins = unsignedTx.Ins
	case *txs.AddDelegatorTx:
		ins = unsignedTx.Ins
	case *txs.AddPermissionlessDelegatorTx:
		ins = unsignedTx.Ins
	case *txs.CreateSubnetTx:
		ins = unsignedTx.Ins
	case *txs.CreateChainTx:
		ins = unsignedTx.Ins
	case *txs.AddSubnetValidatorTx:
		ins = unsignedTx.Ins
	case *txs.TransformSubnetTx:
		ins = unsignedTx.Ins
	case *txs.RemoveSubnetValidatorTx:
		ins = unsignedTx.Ins

	case *txs.RewardValidatorTx:
		return []ids.ID{unsignedTx.TxID}, nil
	case *txs.AdvanceTimeTx:
		return []ids.ID{}, nil
	default:
		return nil, fmt.Errorf("unknown tx type %T", unsignedTx)
	}

	// extract txIDs and filter out duplicates
	txIDs := make(map[ids.ID]ids.ID)
	for _, in := range ins {
		txIDs[in.UTXOID.TxID] = in.UTXOID.TxID
	}
	uniqueTxIDs := make([]ids.ID, 0, len(txIDs))
	for _, txnID := range txIDs {
		uniqueTxIDs = append(uniqueTxIDs, txnID)
	}
	utils.Sort(uniqueTxIDs)

	return uniqueTxIDs, nil
}

// GetReferencedAccounts extracts destination accounts from given dependency transactions
func (bd BlockTxDependencies) GetReferencedAccounts(hrp string) (map[string]*types.AccountIdentifier, error) {
	addresses := make(map[string]*types.AccountIdentifier)
	for _, dependencyTx := range bd {
		utxoMap := dependencyTx.GetUtxos()

		for _, utxo := range utxoMap {
			addressable, ok := utxo.Out.(avax.Addressable)
			if !ok {
				return nil, errFailedToGetUTXOAddresses
			}

			addrs := addressable.Addresses()

			if len(addrs) != 1 {
				continue
			}

			addr, err := address.Format(constants.PChain.String(), hrp, addrs[0])
			addresses[utxo.UTXOID.String()] = &types.AccountIdentifier{Address: addr}
			if err != nil {
				return nil, err
			}
		}
	}

	return addresses, nil
}

// SingleTxDependency represents a single dependency of a give transaction
type SingleTxDependency struct {
	// [Tx] has some of its outputs spent as
	// input from a tx dependent on it
	Tx *txs.Tx

	// Staker txs are rewarded at the end of staking period
	// with some utxos appended to staker txs' ones.
	// [RewardUTXOs] collects those reward utxos
	RewardUTXOs []*avax.UTXO

	// [utxosMap] caches mapping of Tx utxoID --> Tx utxo
	// for both Tx and RewardUTXOs
	utxosMap map[avax.UTXOID]*avax.UTXO
}

func (d *SingleTxDependency) GetUtxos() map[avax.UTXOID]*avax.UTXO {
	if d.utxosMap != nil {
		return d.utxosMap
	}
	d.utxosMap = make(map[avax.UTXOID]*avax.UTXO)

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
