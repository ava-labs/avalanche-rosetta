package client

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

type infoPeersResponse struct {
	Peers []Peer `json:"peers"`
}

type Peer struct {
	ID           string `json:"nodeID"`
	IP           string `json:"ip"`
	PublicIP     string `json:"publicIP"`
	Version      string `json:"version"`
	LastSent     string `json:"lastSent"`
	LastReceived string `json:"lastReceived"`
}

func (p Peer) Metadata() map[string]interface{} {
	return map[string]interface{}{
		"ip":            p.IP,
		"public_ip":     p.PublicIP,
		"version":       p.Version,
		"last_sent":     p.LastSent,
		"last_received": p.LastReceived,
	}
}

type Blockchain struct {
	ID       string `json:"id"`
	SubnetID string `json:"subnetID"`
	Name     string `json:"name"`
	VMID     string `json:"vmId"`
}

type TxNonceMap map[string]string
type TxAccountMap map[string]TxNonceMap

type TxPoolStatus struct {
	PendingCount int `json:"pending"`
	QueuedCount  int `json:"queued"`
}

type TxPoolContent struct {
	Pending TxAccountMap `json:"pending"`
	Queued  TxAccountMap `json:"queued"`
}

type Asset struct {
	ID           string `json:"assetId"`
	Name         string `json:"name"`
	Symbol       string `json:"symbol"`
	Denomination string `json:"denomination"`
}

type Trace struct {
	Type    string         `json:"type"`
	From    common.Address `json:"from"`
	To      common.Address `json:"to"`
	Value   *hexutil.Big   `json:"value"`
	GasUsed *hexutil.Big   `json:"gasUsed"`
	Revert  bool           `json:"revert"`
	Error   string         `json:"error,omitempty"`
	Calls   []*Trace       `json:"calls,omitempty"`
}

type FlatTrace struct {
	Type    string         `json:"type"`
	From    common.Address `json:"from"`
	To      common.Address `json:"to"`
	Value   *hexutil.Big   `json:"value"`
	GasUsed *hexutil.Big   `json:"gasUsed"`
	Revert  bool           `json:"revert"`
	Error   string         `json:"error,omitempty"`
}

func (t *Trace) flatten() *FlatTrace {
	return &FlatTrace{
		Type:    t.Type,
		From:    t.From,
		To:      t.To,
		Value:   t.Value,
		GasUsed: t.GasUsed,
		Revert:  t.Revert,
		Error:   t.Error,
	}
}

func (t *Trace) init() []*FlatTrace {
	if t.Value == nil {
		t.Value = new(hexutil.Big)
	}
	if t.GasUsed == nil {
		t.GasUsed = new(hexutil.Big)
	}
	if len(t.Error) > 0 {
		// Any error surfaced by the decoder means that the transaction
		// has reverted.
		t.Revert = true
	}

	results := []*FlatTrace{t.flatten()}
	for _, child := range t.Calls {
		// Ensure all children of a reverted call
		// are also reverted!
		if t.Revert {
			child.Revert = true

			// Copy error message from parent
			// if child does not have one
			if len(child.Error) == 0 {
				child.Error = t.Error
			}
		}

		children := child.init()
		results = append(results, children...)
	}

	return results
}
