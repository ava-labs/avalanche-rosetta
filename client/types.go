package client

import (
	"encoding/json"
	"math/big"

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

type Call struct {
	Type         string         `json:"type"`
	From         common.Address `json:"from"`
	To           common.Address `json:"to"`
	Value        *big.Int       `json:"value"`
	GasUsed      *big.Int       `json:"gasUsed"`
	Revert       bool
	ErrorMessage string  `json:"error"`
	Calls        []*Call `json:"calls"`
}

type FlatCall struct {
	Type         string         `json:"type"`
	From         common.Address `json:"from"`
	To           common.Address `json:"to"`
	Value        *big.Int       `json:"value"`
	GasUsed      *big.Int       `json:"gasUsed"`
	Revert       bool
	ErrorMessage string `json:"error"`
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

func (t *Call) UnmarshalJSON(input []byte) error {
	type CustomTrace struct {
		Type         string         `json:"type"`
		From         common.Address `json:"from"`
		To           common.Address `json:"to"`
		Value        *hexutil.Big   `json:"value"`
		GasUsed      *hexutil.Big   `json:"gasUsed"`
		Revert       bool
		ErrorMessage string  `json:"error"`
		Calls        []*Call `json:"calls"`
	}
	var dec CustomTrace
	if err := json.Unmarshal(input, &dec); err != nil {
		return err
	}

	t.Type = dec.Type
	t.From = dec.From
	t.To = dec.To
	if dec.Value != nil {
		t.Value = (*big.Int)(dec.Value)
	} else {
		t.Value = new(big.Int)
	}
	if dec.GasUsed != nil {
		t.GasUsed = (*big.Int)(dec.Value)
	} else {
		t.GasUsed = new(big.Int)
	}
	if dec.ErrorMessage != "" {
		// Any error surfaced by the decoder means that the transaction
		// has reverted.
		t.Revert = true
	}
	t.ErrorMessage = dec.ErrorMessage
	t.Calls = dec.Calls
	return nil
}

func FlattenTraces(data *Call, flattened []*FlatCall) []*FlatCall {
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
