package pchain

import (
	pconstants "github.com/ava-labs/avalanche-rosetta/constants/pchain"
	"github.com/coinbase/rosetta-sdk-go/types"
)

// txOps collects all balance-changing information within a transaction
type txOps struct {
	isConstruction bool
	Ins            []*types.Operation
	Outs           []*types.Operation
	StakeOuts      []*types.Operation
	ImportIns      []*types.Operation
	ExportOuts     []*types.Operation
}

func newTxOps(isConstruction bool) *txOps {
	return &txOps{isConstruction: isConstruction}
}

func (t *txOps) IncludedOperations() []*types.Operation {
	ops := []*types.Operation{}
	ops = append(ops, t.Ins...)
	ops = append(ops, t.Outs...)
	ops = append(ops, t.StakeOuts...)
	return ops
}

// Used to populate operation identifier
func (t *txOps) Len() int {
	return len(t.Ins) + len(t.Outs) + len(t.StakeOuts)
}

// Used to populate coin identifier
func (t *txOps) OutputLen() int {
	return len(t.Outs) + len(t.StakeOuts)
}

func (t *txOps) Append(op *types.Operation, metaType pconstants.Op) {
	switch metaType {
	case pconstants.Import:
		if t.isConstruction {
			t.Ins = append(t.Ins, op)
		} else {
			// removing operation identifier as these will be skipped in the final operations list
			op.OperationIdentifier = nil
			t.ImportIns = append(t.ImportIns, op)
		}
	case pconstants.Export:
		if t.isConstruction {
			t.Outs = append(t.Outs, op)
		} else {
			// removing operation identifier as these will be skipped in the final operations list
			op.OperationIdentifier = nil
			t.ExportOuts = append(t.ExportOuts, op)
		}
	case pconstants.Stake, pconstants.Reward:
		t.StakeOuts = append(t.StakeOuts, op)
	case pconstants.Output:
		t.Outs = append(t.Outs, op)
	case pconstants.Input:
		t.Ins = append(t.Ins, op)
	}
}
