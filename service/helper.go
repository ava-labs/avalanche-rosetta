package service

import (
	"context"
	"encoding/json"
	"math/big"

	"github.com/coinbase/rosetta-sdk-go/parser"
	"github.com/coinbase/rosetta-sdk-go/types"
	"github.com/figment-networks/avalanche-rosetta/client"

	ethtypes "github.com/ava-labs/coreth/core/types"
	ethcommon "github.com/ethereum/go-ethereum/common"
)

const (
	transferGasLimit = uint64(21000)
)

type unsignedTx struct {
	Nonce    uint64   `json:"nonce"`
	From     string   `json:"from"`
	To       string   `json:"to"`
	Value    *big.Int `json:"value"`
	GasPrice *big.Int `json:"gas_price"`
	GasLimit uint64   `json:"gas"`
	ChainID  *big.Int `json:"chain_id"`
	Input    []byte   `json:"input"`
}

type txMetadata struct {
	Nonce    uint64   `json:"nonce"`
	GasPrice *big.Int `json:"gas_price"`
}

func makeGenesisBlock(hash string) *types.Block {
	return &types.Block{
		BlockIdentifier: &types.BlockIdentifier{
			Index: 0,
			Hash:  hash,
		},
		ParentBlockIdentifier: &types.BlockIdentifier{
			Index: 0,
			Hash:  hash,
		},
		Timestamp: 946713601000,
	}
}

func blockHeaderFromInput(c client.Client, input *types.PartialBlockIdentifier) (*ethtypes.Header, *types.Error) {
	var (
		header *ethtypes.Header
		err    error
	)

	if input == nil {
		header, err = c.HeaderByNumber(context.Background(), nil)
	} else {
		if input.Hash == nil && input.Index == nil {
			return nil, errInvalidInput
		}

		if input.Index != nil {
			header, err = c.HeaderByNumber(context.Background(), big.NewInt(*input.Index))
		} else {
			header, err = c.HeaderByHash(context.Background(), ethcommon.HexToHash(*input.Hash))
		}
	}

	if err != nil {
		return nil, wrapError(errInternalError, err)
	}

	return header, nil
}

func txFromInput(input string) (*ethtypes.Transaction, error) {
	tx := &ethtypes.Transaction{}
	if err := tx.UnmarshalJSON([]byte(input)); err != nil {
		return nil, err
	}
	return tx, nil
}

func txFromMatches(matches []*parser.Match, kv map[string]interface{}, chainID *big.Int) (*ethtypes.Transaction, *unsignedTx, error) {
	var metadata txMetadata
	if err := unmarshalJSONMap(kv, &metadata); err != nil {
		return nil, nil, err
	}

	fromOp, _ := matches[0].First()
	fromAddress := fromOp.Account.Address
	toOp, amount := matches[1].First()
	toAddress := toOp.Account.Address
	nonce := metadata.Nonce
	gasPrice := metadata.GasPrice
	transferData := []byte{}

	tx := ethtypes.NewTransaction(
		nonce,
		ethcommon.HexToAddress(toAddress),
		amount,
		transferGasLimit,
		gasPrice,
		transferData,
	)

	unTx := &unsignedTx{
		From:     fromAddress,
		To:       toAddress,
		Value:    amount,
		Input:    tx.Data(),
		Nonce:    tx.Nonce(),
		GasPrice: gasPrice,
		GasLimit: tx.Gas(),
		ChainID:  chainID,
	}

	return tx, unTx, nil
}

func unmarshalJSONMap(m map[string]interface{}, i interface{}) error {
	b, err := json.Marshal(m)
	if err != nil {
		return err
	}

	return json.Unmarshal(b, i)
}
