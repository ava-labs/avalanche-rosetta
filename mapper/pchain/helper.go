package pchain

import (
	"fmt"

	"github.com/ava-labs/avalanchego/codec"
	"github.com/ava-labs/avalanchego/vms/platformvm/txs"
	"github.com/coinbase/rosetta-sdk-go/types"
)

func ParseRosettaTxs(
	parserCfg TxParserConfig,
	c codec.Manager,
	txs []*txs.Tx,
	dependencyTxs BlockTxDependencies,
) ([]*types.Transaction, error) {
	inputAddresses, err := dependencyTxs.GetReferencedAccounts(parserCfg.Hrp)
	if err != nil {
		return nil, err
	}

	parser, err := NewTxParser(parserCfg, inputAddresses, dependencyTxs)
	if err != nil {
		return nil, err
	}

	transactions := make([]*types.Transaction, 0, len(txs))
	for _, tx := range txs {
		if err != tx.Sign(c, nil) {
			return nil, fmt.Errorf("failed tx initialization, %w", err)
		}

		t, err := parser.Parse(tx)
		if err != nil {
			return nil, err
		}

		transactions = append(transactions, t)
	}
	return transactions, nil
}
