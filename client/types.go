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

func (t *Call) Flatten() *FlatCall {
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

func FlattenTraces(data *Call, flattened []*FlatCall) []*FlatCall {
	if data.Value == nil {
		data.Value = new(hexutil.Big)
	}
	if data.GasUsed == nil {
		data.GasUsed = new(hexutil.Big)
	}
	if len(data.ErrorMessage) > 0 {
		data.Revert = true
	}

	results := append(flattened, data.Flatten())
	for _, child := range data.Calls {
		// Ensure all children of a reverted call
		// are also reverted!
		if data.Revert {
			child.Revert = true

			// Copy error message from parent
			// if child does not have one
			if len(child.ErrorMessage) == 0 {
				child.ErrorMessage = data.ErrorMessage
			}
		}

		children := FlattenTraces(child, flattened)
		results = append(results, children...)
	}
	return results
}
