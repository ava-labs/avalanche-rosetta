package client

import (
	"context"
	"fmt"

	"github.com/ava-labs/avalanchego/api"
	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/avalanchego/utils/formatting"
	"github.com/ava-labs/avalanchego/utils/formatting/address"
	"github.com/ava-labs/avalanchego/utils/json"
	"github.com/ava-labs/avalanchego/utils/rpc"
	"github.com/ava-labs/avalanchego/vms/components/avax"
	"github.com/ava-labs/avalanchego/vms/saevm/cchain/tx"
)

// EvmClient calls the C-Chain's `avax` JSON-RPC API.
//
// It is built on the RPC and API types directly instead of avalanchego's `vms/saevm/cchain` client
// because that package imports the whole C-Chain VM, which links the Firewood cgo library. Go 1.26.11
// and later reject that library's LTO sections, so importing it would break every build of this
// module.
type EvmClient struct {
	requester rpc.EndpointRequester
}

// NewEvmClient returns an [EvmClient] for the C-Chain of the node at `uri`.
func NewEvmClient(uri string) EvmClient {
	return EvmClient{
		requester: rpc.NewEndpointRequester(uri + "/ext/bc/C/avax"),
	}
}

// IssueTx issues the serialized atomic transaction `txBytes` and returns its ID.
func (c EvmClient) IssueTx(ctx context.Context, txBytes []byte, options ...rpc.Option) (ids.ID, error) {
	res := &api.JSONTxID{}
	txStr, err := formatting.Encode(formatting.Hex, txBytes)
	if err != nil {
		return res.TxID, fmt.Errorf("problem hex encoding bytes: %w", err)
	}
	err = c.requester.SendRequest(ctx, "avax.issueTx", &api.FormattedTx{
		Tx:       txStr,
		Encoding: formatting.Hex,
	}, res, options...)
	return res.TxID, err
}

// GetAtomicUTXOs returns the UTXOs controlled by `addrs` that were exported to the C-Chain from the
// chain with alias or ID `sourceChain`, along with the cursor to pass as `startAddress` and
// `startUTXOID` to fetch the next page.
func (c EvmClient) GetAtomicUTXOs(
	ctx context.Context,
	addrs []ids.ShortID,
	sourceChain string,
	limit uint32,
	startAddress ids.ShortID,
	startUTXOID ids.ID,
	options ...rpc.Option,
) ([]*avax.UTXO, ids.ShortID, ids.ID, error) {
	res := &api.GetUTXOsReply{}
	err := c.requester.SendRequest(ctx, "avax.getUTXOs", &api.GetUTXOsArgs{
		Addresses:   ids.ShortIDsToStrings(addrs),
		SourceChain: sourceChain,
		Limit:       json.Uint32(limit),
		StartIndex: api.Index{
			Address: startAddress.String(),
			UTXO:    startUTXOID.String(),
		},
		Encoding: formatting.Hex,
	}, res, options...)
	if err != nil {
		return nil, ids.ShortID{}, ids.Empty, err
	}

	utxos := make([]*avax.UTXO, len(res.UTXOs))
	for i, raw := range res.UTXOs {
		utxoBytes, err := formatting.Decode(res.Encoding, raw)
		if err != nil {
			return nil, ids.ShortID{}, ids.Empty, err
		}
		utxos[i], err = tx.ParseUTXO(utxoBytes)
		if err != nil {
			return nil, ids.ShortID{}, ids.Empty, err
		}
	}
	endAddr, err := address.ParseToID(res.EndIndex.Address)
	if err != nil {
		return nil, ids.ShortID{}, ids.Empty, err
	}
	endUTXOID, err := ids.FromString(res.EndIndex.UTXO)
	return utxos, endAddr, endUTXOID, err
}
