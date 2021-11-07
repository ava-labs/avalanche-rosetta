package client

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

const (
	ERC721DefaultSymbol   = "ERC721"
	ERC721DefaultDecimals = 0
	ERC20DefaultSymbol    = "ERC20"
	ERC20DefaultDecimals  = 0
)

type infoPeersResponse struct {
	Peers []Peer `json:"peers"`
}

type ContractInfo struct {
	Symbol   string `json:"symbol"`
	Decimals uint8  `json:"decimals"`
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
	Type    string         `json:"type"`
	From    common.Address `json:"from"`
	To      common.Address `json:"to"`
	Value   *hexutil.Big   `json:"value"`
	GasUsed *hexutil.Big   `json:"gasUsed"`
	Revert  bool           `json:"revert"`
	Error   string         `json:"error,omitempty"`
	Calls   []*Call        `json:"calls,omitempty"`
}

type FlatCall struct {
	Type    string         `json:"type"`
	From    common.Address `json:"from"`
	To      common.Address `json:"to"`
	Value   *big.Int       `json:"value"`
	GasUsed *big.Int       `json:"gasUsed"`
	Revert  bool           `json:"revert"`
	Error   string         `json:"error,omitempty"`
}

func (c *Call) flatten() *FlatCall {
	return &FlatCall{
		Type:    c.Type,
		From:    c.From,
		To:      c.To,
		Value:   c.Value.ToInt(),
		GasUsed: c.GasUsed.ToInt(),
		Revert:  c.Revert,
		Error:   c.Error,
	}
}

func (c *Call) init() []*FlatCall {
	if c.Value == nil {
		c.Value = new(hexutil.Big)
	}
	if c.GasUsed == nil {
		c.GasUsed = new(hexutil.Big)
	}
	if len(c.Error) > 0 {
		// Any error surfaced by the decoder means that the transaction
		// has reverted.
		c.Revert = true
	}

	results := []*FlatCall{c.flatten()}
	for _, child := range c.Calls {
		// Ensure all children of a reverted call
		// are also reverted!
		if c.Revert {
			child.Revert = true

			// Copy error message from parent
			// if child does not have one
			if len(child.Error) == 0 {
				child.Error = c.Error
			}
		}

		children := child.init()
		results = append(results, children...)
	}

	return results
}
