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

type Call struct {
	Type         string         `json:"type"`
	From         common.Address `json:"from"`
	To           common.Address `json:"to"`
	Value        *hexutil.Big   `json:"value"`
	GasUsed      *hexutil.Big   `json:"gasUsed"`
	Revert       bool           `json:"revert"`
	ErrorMessage string         `json:"error,omitempty"`
	Calls        []*Call        `json:"calls,omitempty"`
}

type FlatCall struct {
	Type         string         `json:"type"`
	From         common.Address `json:"from"`
	To           common.Address `json:"to"`
	Value        *hexutil.Big   `json:"value"`
	GasUsed      *hexutil.Big   `json:"gasUsed"`
	Revert       bool           `json:"revert"`
	ErrorMessage string         `json:"error,omitempty"`
}

func (t *Call) flatten() *FlatCall {
	return &FlatCall{
		Type:         t.Type,
		From:         t.From,
		To:           t.To,
		Value:        t.Value,
		GasUsed:      t.GasUsed,
		Revert:       t.Revert,
		ErrorMessage: t.ErrorMessage,
	}
}

func (t *Call) Init() []*FlatCall {
	if t.Value == nil {
		t.Value = new(hexutil.Big)
	}
	if t.GasUsed == nil {
		t.GasUsed = new(hexutil.Big)
	}
	if len(t.ErrorMessage) > 0 {
		t.Revert = true
	}

	results := []*FlatCall{t.flatten()}
	for _, child := range t.Calls {
		// Ensure all children of a reverted call
		// are also reverted!
		if t.Revert {
			child.Revert = true

			// Copy error message from parent
			// if child does not have one
			if len(child.ErrorMessage) == 0 {
				child.ErrorMessage = t.ErrorMessage
			}
		}

		children := child.Init()
		results = append(results, children...)
	}

	return results
}
